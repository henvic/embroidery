package clientshandles

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	schema "github.com/gorilla/Schema"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/henvic/embroidery/clients"
	"github.com/henvic/embroidery/handles"
	"github.com/henvic/embroidery/server"
	"github.com/henvic/embroidery/sitetemplate"
)

var router = server.Instance.Mux

func init() {
	router().Handle("/clients", handles.AuthenticatedHandler(clientsHandler))
	router().Handle("/clients/add", handles.AuthenticatedHandler(createHandler))
	router().Handle("/clients/{client_id}", handles.AuthenticatedHandler(editHandler))
}

type clientAddForm struct {
	FirstName string `schema:"first_name"`
	LastName  string `schema:"last_name"`
	Email     string `schema:"email"`
	Password  string `schema:"password"`
}

func clientsHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	var showArchived = (r.URL.Query().Get("showArchived") != "")
	var filter = clients.ListFilter{
		ShowArchived: showArchived,
	}

	clients, err := clients.List(r.Context(), filter)

	if err != nil {
		handles.ErrorHandler(w, r, "Can't get clients list", http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		return
	}

	var t = &sitetemplate.Template{
		Title:     "Clients",
		Section:   "clients",
		Filenames: []string{"gui/clients/clients.html"},
		Data: map[string]interface{}{
			"Clients":      clients,
			"ShowArchived": showArchived,
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}

func createHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	if r.Method == http.MethodGet {
		var t = &sitetemplate.Template{
			Title:          "Add a client",
			Section:        "clients",
			Filenames:      []string{"gui/clients/create.html"},
			Data:           map[string]interface{}{},
			Request:        r,
			ResponseWriter: w,
		}

		t.Respond()
		return
	}

	clientPostHandler(w, r, s)
}

func clientPostHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	if r.Method != http.MethodPost {
		handles.ErrorHandler(w, r, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		handles.ErrorHandler(w, r, "Invalid form", http.StatusBadRequest)
		return
	}

	decoder := schema.NewDecoder()
	caf := clientAddForm{}

	if err := decoder.Decode(&caf, r.PostForm); err != nil {
		handles.ErrorHandler(w, r, fmt.Sprintf("Error decoding request body: %v", err), http.StatusBadRequest)
		return
	}

	if len(caf.FirstName) == 0 {
		handles.ErrorHandler(w, r, "No first name given", http.StatusBadRequest)
		return
	}

	if len(caf.LastName) == 0 {
		handles.ErrorHandler(w, r, "No last name given", http.StatusBadRequest)
		return
	}

	if len(caf.Email) == 0 {
		handles.ErrorHandler(w, r, "No email given", http.StatusBadRequest)
		return
	}

	c := clients.Client{
		FirstName: caf.FirstName,
		LastName:  caf.LastName,
		Email:     caf.Email,
		Status:    "ACTIVE",
	}

	uid, err := clients.Insert(context.Background(), c)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/clients/%v", url.QueryEscape(uid)), http.StatusSeeOther)
	fmt.Fprintf(w, `Added client %v`, uid)
}

func editHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	vars := mux.Vars(r)
	clientID, ok := vars["client_id"]

	if !ok {
		handles.ErrorHandler(w, r, "Missing client ID parameter", http.StatusBadRequest)
		return
	}

	var client, err = clients.Get(r.Context(), clientID)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		editHandlerGetHandler(w, r, client)
	case http.MethodPost:
		editHandlerPostHandler(w, r, client)
		return
	default:
		handles.ErrorHandler(w, r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func editHandlerGetHandler(w http.ResponseWriter, r *http.Request, client clients.Client) {
	var t = sitetemplate.Template{
		Title:     "Edit client",
		Section:   "clients",
		Filenames: []string{"gui/clients/edit.html"},
		Data: map[string]interface{}{
			"Client": client,
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}

func editHandlerPostHandler(w http.ResponseWriter, r *http.Request, client clients.Client) {
	if err := r.ParseForm(); err != nil {
		handles.ErrorHandler(w, r, "Internal Server Error: parsing client edit form", http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "%+v\n", err)
	}

	var firstName = r.PostFormValue("first_name")
	var lastName = r.PostFormValue("last_name")

	if len(firstName) == 0 {
		handles.ErrorHandler(w, r, "Internal Server Error: first_name is empty", http.StatusInternalServerError)
		return
	}

	var status = r.PostFormValue("status")
	switch status {
	case "active", "archived":
	default:
		handles.ErrorHandler(w, r, "Internal Server Error: invalid status value", http.StatusInternalServerError)
		return
	}

	var email = r.PostFormValue("email")
	if !strings.Contains(email, "@") {
		handles.ErrorHandler(w, r, "Internal Server Error: email is invalid", http.StatusInternalServerError)
		return
	}

	client.FirstName = firstName
	client.LastName = lastName
	client.Email = email
	client.Status = status

	if err := clients.Update(r.Context(), client); err != nil {
		handles.ErrorHandler(w, r, "Internal Server Error: saving user", http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		return
	}

	http.Redirect(w, r, "/clients", http.StatusSeeOther)
}
