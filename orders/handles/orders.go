package ordershandles

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
	"github.com/henvic/embroidery/address"
	"github.com/henvic/embroidery/clients"
	"github.com/henvic/embroidery/handles"
	"github.com/henvic/embroidery/orders"
	"github.com/henvic/embroidery/server"
	"github.com/henvic/embroidery/sitetemplate"
)

var router = server.Instance.Mux

func init() {
	router().Handle("/orders", handles.AuthenticatedHandler(ordersHandler))
	router().Handle("/orders/add", handles.AuthenticatedHandler(orderAddHandler))
	router().Handle("/orders/{order_id}", handles.AuthenticatedHandler(orderEditHandler))
}

type orderEditForm struct {
	ClientAddressID string `schema:"client_address_id"`
	Status          string `schema:"status"`
}

func orderEditHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	vars := mux.Vars(r)
	orderID, ok := vars["order_id"]

	if !ok {
		handles.ErrorHandler(w, r, "Missing order ID parameter", http.StatusBadRequest)
		return
	}

	order, err := orders.Get(r.Context(), orderID)

	if err != nil {
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

	addresses, err := address.List(r.Context(), address.ListFilter{
		ClientID: client.ClientID,
	})

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		var t = sitetemplate.Template{
			Title:     fmt.Sprintf("Endereço do cliente %v %v", client.FirstName, client.LastName),
			Section:   "orders",
			Filenames: []string{"gui/order/client-order.html"},
			Data: map[string]interface{}{
				"Client":    client,
				"Order":     order,
				"Addresses": addresses,
				"AllStatus": orders.GetStatusFilter(),
			},
			Request:        r,
			ResponseWriter: w,
		}

		t.Respond()
	case http.MethodPost:
		orderPostEditHandler(client, order, w, r)
		return
	default:
		handles.ErrorHandler(w, r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func orderAddHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	var cs, err = clients.List(r.Context(), clients.ListFilter{})

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	switch r.Method {
	case http.MethodPost:
		orderPostAddHandler(w, r, s)
		return
	case http.MethodGet:
		var t = sitetemplate.Template{
			Title:     fmt.Sprintf("Criando ordem de serviço"),
			Section:   "orders",
			Filenames: []string{"gui/order/add-order.html"},
			Data: map[string]interface{}{
				"Clients":       cs,
				"MaybeClientID": r.URL.Query().Get("maybe_client_id"),
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

func orderPostAddHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	if err := r.ParseForm(); err != nil {
		handles.ErrorHandler(w, r, "Invalid form", http.StatusBadRequest)
		return
	}

	var clientID = r.PostFormValue("client_id")

	if len(clientID) == 0 {
		handles.ErrorHandler(w, r, "Missing client ID parameter", http.StatusBadRequest)
		return
	}

	var client, err = clients.Get(r.Context(), clientID)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	as, err := address.List(r.Context(), address.ListFilter{
		ClientID: client.ClientID,
	})

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	if len(as) == 0 {
		handles.ErrorHandler(w, r, "Client has no address. Please add one before creating an order.", http.StatusPreconditionFailed)
		return
	}

	o := orders.Order{
		ClientID:        client.ClientID,
		ClientAddressID: as[0].AddressID,
	}

	added, err := orders.Insert(context.Background(), o)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/orders/%v", url.QueryEscape(added)), http.StatusSeeOther)
}

func orderPostEditHandler(client clients.Client, order orders.Order, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		handles.ErrorHandler(w, r, "Invalid form", http.StatusBadRequest)
		return
	}

	decoder := schema.NewDecoder()
	caf := orderEditForm{}

	if err := decoder.Decode(&caf, r.PostForm); err != nil {
		handles.ErrorHandler(w, r, fmt.Sprintf("Error decoding request body: %v", err), http.StatusBadRequest)
		return
	}

	switch caf.Status {
	case "open", "waiting_for_payment", "stand_by", "queue", "in_progress", "canceled", "done":
	default:
		handles.ErrorHandler(w, r, "Invalid order status", http.StatusBadRequest)
		return
	}

	err := orders.Update(r.Context(), order, caf.Status, caf.ClientAddressID)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/orders/%v", url.QueryEscape(order.OrderID)), http.StatusSeeOther)
}

func ordersHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	var clientID = (r.URL.Query().Get("client_id"))
	var currentStatus = (r.URL.Query().Get("status"))

	if _, ok := orders.GetStatusFilter()[currentStatus]; !ok {
		handles.ErrorHandler(w, r, "Order status doesn't exists", http.StatusBadRequest)
	}

	var c clients.Client

	var cList, err = clients.List(r.Context(), clients.ListFilter{})

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	if clientID != "" {
		c, err = clients.Get(r.Context(), clientID)

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

	order, err := orders.List(r.Context(), orders.ListFilter{
		ClientID: c.ClientID,
		Status:   currentStatus,
	})

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	var clientsMap = clients.GetClientsMapFromSlice(cList)

	var t = sitetemplate.Template{
		Title:     "Orders",
		Section:   "orders",
		Filenames: []string{"gui/order/list-client.html"},
		Data: map[string]interface{}{
			"ClientsMap":    clientsMap,
			"Client":        c,
			"Orders":        order,
			"AllStatus":     orders.GetStatusFilter(),
			"CurrentStatus": currentStatus,
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}
