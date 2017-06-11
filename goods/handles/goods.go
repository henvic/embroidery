package goodshandles

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
	"github.com/henvic/embroidery/employees"
	"github.com/henvic/embroidery/goods"
	"github.com/henvic/embroidery/handles"
	"github.com/henvic/embroidery/jobs"
	"github.com/henvic/embroidery/orders"
	"github.com/henvic/embroidery/server"
	"github.com/henvic/embroidery/sitetemplate"
)

var router = server.Instance.Mux

func init() {
	router().Handle("/goods", handles.AuthenticatedHandler(goodsHandler))
	router().Handle("/jobs/{job_id}/add-good", handles.AuthenticatedHandler(goodAddHandler))
	router().Handle("/goods/{good_id}", handles.AuthenticatedHandler(goodEditHandler))
}

type goodAddForm struct {
	EmployeeID string `schema:"employee_id"`
	Type       string `schema:"type"`
	Amount     int    `schema:"amount"`
	Unit       string `schema:"unit"`
	Notes      string `schema:"notes"`
	Status     string `schema:"status"`
}

func goodEditHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	vars := mux.Vars(r)
	goodID, ok := vars["good_id"]

	if !ok {
		handles.ErrorHandler(w, r, "Missing good ID parameter", http.StatusBadRequest)
		return
	}

	good, err := goods.Get(r.Context(), goodID)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	client, err := clients.Get(r.Context(), good.OwnerID)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	job, err := jobs.Get(r.Context(), good.JobID)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	order, err := orders.Get(r.Context(), job.OrderID)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	es, err := employees.List(r.Context())

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		var t = sitetemplate.Template{
			Title:     fmt.Sprintf("Consumíveis do cliente %v %v", client.FirstName, client.LastName),
			Section:   "goods",
			Filenames: []string{"gui/goods/client-good.html"},
			Data: map[string]interface{}{
				"Client":         client,
				"Good":           good,
				"Order":          order,
				"Job":            job,
				"Employees":      es,
				"AvailableTypes": goods.GetAvailableTypes(),
				"AvailableUnits": goods.GetAvailableUnits(),
				"AllStatus":      goods.GetStatusFilter(),
			},
			Request:        r,
			ResponseWriter: w,
		}

		t.Respond()
	case http.MethodPost:
		goodPostEditHandler(client, good, w, r)
		return
	default:
		handles.ErrorHandler(w, r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func goodAddHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	vars := mux.Vars(r)
	jobID, ok := vars["job_id"]

	if !ok {
		handles.ErrorHandler(w, r, "Missing job ID parameter", http.StatusBadRequest)
		return
	}

	job, err := jobs.Get(r.Context(), jobID)

	switch err {
	case nil:
	case sql.ErrNoRows:
		handles.ErrorHandler(w, r, "Job not found", http.StatusNotFound)
		return
	default:
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

	es, err := employees.List(r.Context())

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	switch r.Method {
	case http.MethodPost:
		goodPostAddHandler(job, client, w, r)
		return
	case http.MethodGet:
		var t = sitetemplate.Template{
			Title:     fmt.Sprintf("Adicionando consumível"),
			Section:   "goods",
			Filenames: []string{"gui/goods/add-good.html"},
			Data: map[string]interface{}{
				"Client":         client,
				"Job":            job,
				"Employees":      es,
				"MaybeOwnerID":   r.URL.Query().Get("maybe_client_id"),
				"AvailableTypes": goods.GetAvailableTypes(),
				"AvailableUnits": goods.GetAvailableUnits(),
				"AllStatus":      goods.GetStatusFilter(),
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

func goodPostAddHandler(job jobs.Job, client clients.Client, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		handles.ErrorHandler(w, r, "Invalid form", http.StatusBadRequest)
		return
	}

	decoder := schema.NewDecoder()
	caf := goodAddForm{}

	if err := decoder.Decode(&caf, r.PostForm); err != nil {
		handles.ErrorHandler(w, r, fmt.Sprintf("Error decoding request body: %v", err), http.StatusBadRequest)
		return
	}

	o := goods.Good{
		OwnerID:    client.ClientID,
		JobID:      job.JobID,
		EmployeeID: caf.EmployeeID,
		Type:       caf.Type,
		Amount:     caf.Amount,
		Unit:       caf.Unit,
		Notes:      caf.Notes,
		Status:     caf.Status,
	}

	added, err := goods.Insert(context.Background(), o)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/goods/%v", url.QueryEscape(added)), http.StatusSeeOther)
}

func goodPostEditHandler(client clients.Client, good goods.Good, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		handles.ErrorHandler(w, r, "Invalid form", http.StatusBadRequest)
		return
	}

	var status = r.FormValue("status")

	switch status {
	case "acquired", "in_stock", "in_use", "missing", "decomissioned":
		good.Status = status
	default:
		handles.ErrorHandler(w, r, "Invalid good status", http.StatusBadRequest)
		return
	}

	err := goods.Update(r.Context(), good)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/goods?job_id=%v", url.QueryEscape(good.JobID)), http.StatusSeeOther)
}

func goodsHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	var clientID = (r.URL.Query().Get("client_id"))
	var jobID = (r.URL.Query().Get("job_id"))
	var currentStatus = (r.URL.Query().Get("status"))

	if _, ok := goods.GetStatusFilter()[currentStatus]; !ok {
		handles.ErrorHandler(w, r, "Good status doesn't exists", http.StatusBadRequest)
	}

	var c *clients.Client
	var j *jobs.Job

	var cList, err = clients.List(r.Context(), clients.ListFilter{})

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	if jobID != "" {
		jp, err := jobs.Get(r.Context(), jobID)
		j = &jp

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
		case j.ClientID:
		case "":
			clientID = j.ClientID
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

	goodsList, err := goods.List(r.Context(), goods.ListFilter{
		OwnerID: clientID,
		JobID:   jobID,
		Status:  currentStatus,
	})

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	var clientsMap = clients.GetClientsMapFromSlice(cList)

	var t = sitetemplate.Template{
		Title:     "Goods",
		Section:   "goods",
		Filenames: []string{"gui/goods/list-client.html"},
		Data: map[string]interface{}{
			"ClientsMap":    clientsMap,
			"Client":        c,
			"Job":           j,
			"Goods":         goodsList,
			"AllStatus":     goods.GetStatusFilter(),
			"CurrentStatus": currentStatus,
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}
