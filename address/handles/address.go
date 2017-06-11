package addresshandles

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
	"github.com/henvic/embroidery/address"
	"github.com/henvic/embroidery/clients"
	"github.com/henvic/embroidery/handles"
	"github.com/henvic/embroidery/server"
	"github.com/henvic/embroidery/sitetemplate"
)

var router = server.Instance.Mux

func init() {
	router().Handle("/clients/{client_id}/address", handles.AuthenticatedHandler(addressesHandler))
	router().Handle("/clients/{client_id}/address/add", handles.AuthenticatedHandler(addressesAddHandler))
	router().Handle("/clients/{client_id}/address/{address_id}", handles.AuthenticatedHandler(addressesEditHandler))
}

type addressAddForm struct {
	Name         string `schema:"name"`
	AddressLine1 string `schema:"address_line1"`
	AddressLine2 string `schema:"address_line2"`
	City         string `schema:"city"`
	State        string `schema:"state"`
	Country      string `schema:"country"`
	ZipCode      string `schema:"zip_code"`
	Phone        string `schema:"phone"`
}

type addressEditForm struct {
	Name         string `schema:"name"`
	AddressLine1 string `schema:"address_line1"`
	AddressLine2 string `schema:"address_line2"`
	City         string `schema:"city"`
	State        string `schema:"state"`
	Country      string `schema:"country"`
	ZipCode      string `schema:"zip_code"`
	Phone        string `schema:"phone"`
	Status       string `schema:"status"`
}

func addressesEditHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	vars := mux.Vars(r)
	clientID, ok1 := vars["client_id"]
	addressID, ok2 := vars["address_id"]

	if !ok1 || !ok2 {
		handles.ErrorHandler(w, r, "Missing client or address ID parameter", http.StatusBadRequest)
		return
	}

	var client, err = clients.Get(r.Context(), clientID)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	address, err := address.Get(r.Context(), client.ClientID, addressID)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		var t = sitetemplate.Template{
			Title:     fmt.Sprintf("Endereço do cliente %v %v", client.FirstName, client.LastName),
			Section:   "clients",
			Filenames: []string{"gui/address/client-address.html"},
			Data: map[string]interface{}{
				"Client":  client,
				"Address": address,
			},
			Request:        r,
			ResponseWriter: w,
		}

		t.Respond()
	case http.MethodPost:
		addressPostEditHandler(client, address, w, r)
		return
	default:
		handles.ErrorHandler(w, r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func addressesAddHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
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
	case http.MethodPost:
		addressPostAddHandler(client, w, r)
	case http.MethodGet:
		var t = sitetemplate.Template{
			Title:     fmt.Sprintf("Adicionando endereço para o cliente %v %v", client.FirstName, client.LastName),
			Section:   "clients",
			Filenames: []string{"gui/address/add-address.html"},
			Data: map[string]interface{}{
				"Client": client,
			},
			Request:        r,
			ResponseWriter: w,
		}

		t.Respond()
	default:
		handles.ErrorHandler(w, r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func addressPostAddHandler(client clients.Client, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		handles.ErrorHandler(w, r, "Invalid form", http.StatusBadRequest)
		return
	}

	decoder := schema.NewDecoder()
	caf := addressAddForm{}

	if err := decoder.Decode(&caf, r.PostForm); err != nil {
		handles.ErrorHandler(w, r, fmt.Sprintf("Error decoding request body: %v", err), http.StatusBadRequest)
		return
	}

	if len(caf.Name) == 0 {
		handles.ErrorHandler(w, r, "No name given", http.StatusBadRequest)
		return
	}

	caf.ZipCode = strings.Replace(caf.ZipCode, "-", "", -1)

	if len(caf.ZipCode) == 0 {
		handles.ErrorHandler(w, r, "Invalid zip code", http.StatusBadRequest)
		return
	}

	c := address.Address{
		ClientID:     client.ClientID,
		Name:         caf.Name,
		AddressLine1: caf.AddressLine1,
		AddressLine2: caf.AddressLine2,
		City:         caf.City,
		State:        caf.State,
		Country:      caf.Country,
		ZipCode:      caf.ZipCode,
		Phone:        caf.Phone,
		Status:       "ACTIVE",
	}

	_, err := address.Insert(context.Background(), c)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/clients/%v/address", url.QueryEscape(c.ClientID)), http.StatusSeeOther)
}

func addressPostEditHandler(client clients.Client, addr address.Address, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		handles.ErrorHandler(w, r, "Invalid form", http.StatusBadRequest)
		return
	}

	decoder := schema.NewDecoder()
	caf := addressEditForm{}

	if err := decoder.Decode(&caf, r.PostForm); err != nil {
		handles.ErrorHandler(w, r, fmt.Sprintf("Error decoding request body: %v", err), http.StatusBadRequest)
		return
	}

	if len(caf.Name) == 0 {
		handles.ErrorHandler(w, r, "No name given", http.StatusBadRequest)
		return
	}

	switch caf.Status {
	case "active", "archived":
	default:
		handles.ErrorHandler(w, r, "Invalid address status", http.StatusBadRequest)
		return
	}

	caf.ZipCode = strings.Replace(caf.ZipCode, "-", "", -1)

	if len(caf.ZipCode) == 0 {
		handles.ErrorHandler(w, r, "Invalid zip code", http.StatusBadRequest)
		return
	}

	addr.Name = caf.Name
	addr.AddressLine1 = caf.AddressLine1
	addr.AddressLine2 = caf.AddressLine2
	addr.City = caf.City
	addr.State = caf.State
	addr.Country = caf.Country
	addr.ZipCode = caf.ZipCode
	addr.Phone = caf.Phone
	addr.Status = caf.Status

	err := address.Update(r.Context(), addr)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/clients/%v/address", url.QueryEscape(client.ClientID)), http.StatusSeeOther)
}

func addressesHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	var showArchived = (r.URL.Query().Get("showArchived") != "")
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

	addresses, err := address.List(r.Context(), address.ListFilter{
		ClientID:     clientID,
		ShowArchived: showArchived,
	})

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	var t = sitetemplate.Template{
		Title:     fmt.Sprintf("Endereços do cliente %v %v", client.FirstName, client.LastName),
		Section:   "clients",
		Filenames: []string{"gui/address/list-client.html"},
		Data: map[string]interface{}{
			"Client":       client,
			"Addresses":    addresses,
			"ShowArchived": showArchived,
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}
