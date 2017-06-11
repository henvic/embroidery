package jobshandles

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
	"github.com/henvic/embroidery/asset"
	"github.com/henvic/embroidery/clients"
	"github.com/henvic/embroidery/handles"
	"github.com/henvic/embroidery/jobs"
	"github.com/henvic/embroidery/orders"
	"github.com/henvic/embroidery/server"
	"github.com/henvic/embroidery/sitetemplate"
)

var router = server.Instance.Mux

func init() {
	router().Handle("/jobs", handles.AuthenticatedHandler(jobsHandler))
	router().Handle("/orders/{order_id}/add-job", handles.AuthenticatedHandler(jobAddHandler))
	router().Handle("/jobs/{job_id}", handles.AuthenticatedHandler(jobEditHandler))
}

type jobAddForm struct {
	AssetID    string `schema:"asset_id"`
	Type       string `schema:"type"`
	Amount     int    `schema:"amount"`
	Price      int64  `schema:"price"`
	Complexity int64  `schema:"complexity"`
}

func jobEditHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	vars := mux.Vars(r)
	jobID, ok := vars["job_id"]

	if !ok {
		handles.ErrorHandler(w, r, "Missing job ID parameter", http.StatusBadRequest)
		return
	}

	job, err := jobs.Get(r.Context(), jobID)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	client, err := clients.Get(r.Context(), job.ClientID)

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
			Section:   "jobs",
			Filenames: []string{"gui/jobs/client-job.html"},
			Data: map[string]interface{}{
				"Client":    client,
				"Job":       job,
				"Addresses": addresses,
				"AllStatus": jobs.GetStatusFilter(),
			},
			Request:        r,
			ResponseWriter: w,
		}

		t.Respond()
	case http.MethodPost:
		jobPostEditHandler(client, job, w, r)
		return
	default:
		handles.ErrorHandler(w, r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func jobAddHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
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

	as, err := asset.List(r.Context(), asset.ListFilter{
		ClientID: client.ClientID,
	})

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	switch r.Method {
	case http.MethodPost:
		jobPostAddHandler(order, client, as, w, r)
		return
	case http.MethodGet:
		var t = sitetemplate.Template{
			Title:     fmt.Sprintf("Criando ordem de serviço"),
			Section:   "orders",
			Filenames: []string{"gui/jobs/add-job.html"},
			Data: map[string]interface{}{
				"Client":        client,
				"Order":         order,
				"Assets":        as,
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

func jobPostAddHandler(order orders.Order, client clients.Client, as []asset.Asset, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		handles.ErrorHandler(w, r, "Invalid form", http.StatusBadRequest)
		return
	}

	decoder := schema.NewDecoder()
	caf := jobAddForm{}

	if err := decoder.Decode(&caf, r.PostForm); err != nil {
		handles.ErrorHandler(w, r, fmt.Sprintf("Error decoding request body: %v", err), http.StatusBadRequest)
		return
	}

	var foundAsset bool

	for _, a := range as {
		if a.AssetID == caf.AssetID {
			foundAsset = true
		}
	}

	if len(caf.AssetID) == 0 || !foundAsset {
		handles.ErrorHandler(w, r, "Missing asset ID parameter", http.StatusBadRequest)
		return
	}

	if !foundAsset {
		handles.ErrorHandler(w, r, "Asset not found", http.StatusNotFound)
		return
	}

	o := jobs.Job{
		ClientID:   client.ClientID,
		OrderID:    order.OrderID,
		AssetID:    caf.AssetID,
		Status:     "CREATED",
		Type:       caf.Type,
		Amount:     caf.Amount,
		Price:      caf.Price,
		Complexity: caf.Complexity,
	}

	added, err := jobs.Insert(context.Background(), o)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/jobs/%v", url.QueryEscape(added)), http.StatusSeeOther)
}

func jobPostEditHandler(client clients.Client, job jobs.Job, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		handles.ErrorHandler(w, r, "Invalid form", http.StatusBadRequest)
		return
	}

	var status = r.FormValue("status")

	switch status {
	case "created", "queue", "in_progress", "canceled", "done":
		job.Status = status
	default:
		handles.ErrorHandler(w, r, "Invalid job status", http.StatusBadRequest)
		return
	}

	err := jobs.UpdateStatus(r.Context(), job)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/jobs?order_id=%v", url.QueryEscape(job.OrderID)), http.StatusSeeOther)
}

func jobsHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	var clientID = (r.URL.Query().Get("client_id"))
	var orderID = (r.URL.Query().Get("order_id"))
	var currentStatus = (r.URL.Query().Get("status"))

	if _, ok := jobs.GetStatusFilter()[currentStatus]; !ok {
		handles.ErrorHandler(w, r, "Job status doesn't exists", http.StatusBadRequest)
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

	jobsList, err := jobs.List(r.Context(), jobs.ListFilter{
		ClientID: clientID,
		OrderID:  orderID,
		Status:   currentStatus,
	})

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	var clientsMap = clients.GetClientsMapFromSlice(cList)

	var t = sitetemplate.Template{
		Title:     "Jobs",
		Section:   "jobs",
		Filenames: []string{"gui/jobs/list-client.html"},
		Data: map[string]interface{}{
			"ClientsMap":    clientsMap,
			"Client":        c,
			"Order":         o,
			"Jobs":          jobsList,
			"AllStatus":     jobs.GetStatusFilter(),
			"CurrentStatus": currentStatus,
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}
