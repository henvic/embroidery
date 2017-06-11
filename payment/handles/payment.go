package paymenthandles

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"os"

	schema "github.com/gorilla/Schema"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/henvic/embroidery/clients"
	"github.com/henvic/embroidery/handles"
	"github.com/henvic/embroidery/orders"
	"github.com/henvic/embroidery/payment"
	"github.com/henvic/embroidery/server"
	"github.com/henvic/embroidery/sitetemplate"
)

var router = server.Instance.Mux

func init() {
	router().Handle("/payments", handles.AuthenticatedHandler(paymentsHandler))
	router().Handle("/orders/{order_id}/pay", handles.AuthenticatedHandler(paymentAddHandler))
}

type paymentAddForm struct {
	PriceTotal int64  `schema:"price_Total"`
	Provider   string `schema:"provider"`
}

func paymentAddHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	vars := mux.Vars(r)
	orderID, ok := vars["order_id"]

	if !ok {
		handles.ErrorHandler(w, r, "Missing order ID parameter", http.StatusBadRequest)
		return
	}

	order, err := orders.Get(r.Context(), orderID)

	switch err {
	case nil:
	case sql.ErrNoRows:
		handles.ErrorHandler(w, r, "Order not found", http.StatusNotFound)
		return
	default:
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	client, err := clients.Get(r.Context(), order.ClientID)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	switch r.Method {
	case http.MethodPost:
		paymentPostAddHandler(order, client, w, r)
		return
	case http.MethodGet:
		var t = sitetemplate.Template{
			Title:     fmt.Sprintf("Registro de pagamento"),
			Section:   "orders",
			Filenames: []string{"gui/payments/add-payment.html"},
			Data: map[string]interface{}{
				"Client":       client,
				"Order":        order,
				"AllProviders": payment.GetProvidersFilter(),
			},
			Request:        r,
			ResponseWriter: w,
		}

		t.Respond()
		return
	default:
		handles.ErrorHandler(w, r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func paymentPostAddHandler(order orders.Order, client clients.Client, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		handles.ErrorHandler(w, r, "Invalid form", http.StatusBadRequest)
		return
	}

	decoder := schema.NewDecoder()
	caf := paymentAddForm{}

	if err := decoder.Decode(&caf, r.PostForm); err != nil {
		handles.ErrorHandler(w, r, fmt.Sprintf("Error decoding request body: %v", err), http.StatusBadRequest)
		return
	}

	if caf.PriceTotal == 0 {
		handles.ErrorHandler(w, r, "Don't be so cheap!", http.StatusBadRequest)
		return
	}

	if _, ok := payment.GetProvidersFilter()[caf.Provider]; !ok || caf.Provider == "" {
		handles.ErrorHandler(w, r, "Payment provider not recognized", http.StatusBadRequest)
		return
	}

	o := payment.Payment{
		ClientID:   client.ClientID,
		OrderID:    order.OrderID,
		PriceTotal: caf.PriceTotal,
		Provider:   caf.Provider,
	}

	_, err := payment.Insert(context.Background(), o)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/orders/%v", url.QueryEscape(order.OrderID)), http.StatusSeeOther)
}

func paymentsHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	var clientID = (r.URL.Query().Get("client_id"))
	var orderID = (r.URL.Query().Get("order_id"))
	var currentProvider = (r.URL.Query().Get("provider"))

	if _, ok := payment.GetProvidersFilter()[currentProvider]; !ok {
		handles.ErrorHandler(w, r, "Payment provider doesn't exists", http.StatusBadRequest)
	}

	var c *clients.Client
	var o *orders.Order

	var cList, err = clients.List(r.Context(), clients.ListFilter{})

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	if orderID != "" {
		op, err := orders.Get(r.Context(), orderID)
		o = &op

		switch err {
		case nil:
		case sql.ErrNoRows:
			handles.ErrorHandler(w, r, "Order not found", http.StatusNotFound)
			return
		default:
			handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
			return
		}

		switch clientID {
		case o.ClientID:
		case "":
			clientID = o.ClientID
		default:
			handles.ErrorHandler(w, r, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
	}

	if clientID != "" {
		cp, err := clients.Get(r.Context(), clientID)
		c = &cp

		switch err {
		case nil:
		case sql.ErrNoRows:
			handles.ErrorHandler(w, r, "Client not found", http.StatusNotFound)
			return
		default:
			handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
			return
		}
	}

	paymentsList, err := payment.List(r.Context(), payment.ListFilter{
		ClientID: clientID,
		OrderID:  orderID,
	})

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	var clientsMap = clients.GetClientsMapFromSlice(cList)

	var t = sitetemplate.Template{
		Title:     "Payments",
		Section:   "payment",
		Filenames: []string{"gui/payments/list-client.html"},
		Data: map[string]interface{}{
			"ClientsMap":      clientsMap,
			"Client":          c,
			"Order":           o,
			"Payments":        paymentsList,
			"AllProviders":    payment.GetProvidersFilter(),
			"CurrentProvider": currentProvider,
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}
