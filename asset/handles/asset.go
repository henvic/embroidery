package assethandles

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	schema "github.com/gorilla/Schema"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/henvic/embroidery/asset"
	"github.com/henvic/embroidery/clients"
	"github.com/henvic/embroidery/handles"
	"github.com/henvic/embroidery/server"
	"github.com/henvic/embroidery/sitetemplate"
)

var router = server.Instance.Mux

func init() {
	router().Handle("/assets", handles.AuthenticatedHandler(assetsFinderHandler))
	router().Handle("/clients/{client_id}/assets", handles.AuthenticatedHandler(assetsHandler))
	router().Handle("/clients/{client_id}/assets/add", handles.AuthenticatedHandler(assetsAddHandler))
	router().Handle("/clients/{client_id}/assets/{asset_id}", handles.AuthenticatedHandler(assetsEditHandler))
}

type assetAddForm struct {
	Filepath         string `schema:"filepath"`
	OriginalFilepath string `schema:"original_filepath"`
}

type assetEditForm struct {
	Filepath string `schema:"filepath"`
	Status   string `schema:"status"`
}

func assetsFinderHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	var cs, err = clients.List(r.Context(), clients.ListFilter{
		ShowArchived: true,
	})

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	var t = sitetemplate.Template{
		Title:     fmt.Sprintf("Assets of clients"),
		Section:   "assets",
		Filenames: []string{"gui/assets/list-clients.html"},
		Data: map[string]interface{}{
			"Clients": cs,
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}

func assetsEditHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	vars := mux.Vars(r)
	clientID, ok1 := vars["client_id"]
	assetID, ok2 := vars["asset_id"]

	if !ok1 || !ok2 {
		handles.ErrorHandler(w, r, "Missing client or asset ID parameter", http.StatusBadRequest)
		return
	}

	var client, err = clients.Get(r.Context(), clientID)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	asset, err := asset.Get(r.Context(), client.ClientID, assetID)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		var t = sitetemplate.Template{
			Title:     fmt.Sprintf("Asset do cliente %v %v", client.FirstName, client.LastName),
			Section:   "assets",
			Filenames: []string{"gui/assets/client-asset.html"},
			Data: map[string]interface{}{
				"Client": client,
				"Asset":  asset,
			},
			Request:        r,
			ResponseWriter: w,
		}

		t.Respond()
	case http.MethodPost:
		assetPostEditHandler(client, asset, w, r)
		return
	default:
		handles.ErrorHandler(w, r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func assetsAddHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
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
		assetPostAddHandler(client, w, r)
	case http.MethodGet:
		var t = sitetemplate.Template{
			Title:     fmt.Sprintf("Adicionando endereço para o cliente %v %v", client.FirstName, client.LastName),
			Section:   "assets",
			Filenames: []string{"gui/assets/add-asset.html"},
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

func assetPostAddHandler(client clients.Client, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		handles.ErrorHandler(w, r, "Invalid form", http.StatusBadRequest)
		return
	}

	decoder := schema.NewDecoder()
	caf := assetAddForm{}

	if err := decoder.Decode(&caf, r.PostForm); err != nil {
		handles.ErrorHandler(w, r, fmt.Sprintf("Error decoding request body: %v", err), http.StatusBadRequest)
		return
	}

	if len(caf.OriginalFilepath) == 0 {
		handles.ErrorHandler(w, r, "Original filepath can't be empty", http.StatusBadRequest)
		return
	}

	if len(caf.Filepath) == 0 {
		handles.ErrorHandler(w, r, "Filepath can't be empty", http.StatusBadRequest)
		return
	}

	c := asset.Asset{
		ClientID:         client.ClientID,
		Filepath:         caf.Filepath,
		OriginalFilepath: caf.OriginalFilepath,
		Status:           "ACTIVE",
	}

	_, err := asset.Insert(context.Background(), c)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/clients/%v/assets", url.QueryEscape(c.ClientID)), http.StatusSeeOther)
}

func assetPostEditHandler(client clients.Client, a asset.Asset, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		handles.ErrorHandler(w, r, "Invalid form", http.StatusBadRequest)
		return
	}

	decoder := schema.NewDecoder()
	caf := assetEditForm{}

	if err := decoder.Decode(&caf, r.PostForm); err != nil {
		handles.ErrorHandler(w, r, fmt.Sprintf("Error decoding request body: %v", err), http.StatusBadRequest)
		return
	}

	switch caf.Status {
	case "active", "archived":
	default:
		handles.ErrorHandler(w, r, "Invalid asset status", http.StatusBadRequest)
		return
	}

	if len(caf.Filepath) == 0 {
		handles.ErrorHandler(w, r, "Filepath can't be empty", http.StatusBadRequest)
		return
	}

	a.Filepath = caf.Filepath
	a.Status = caf.Status

	err := asset.Update(r.Context(), a)

	if err != nil {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "Internal Server Error: %v\n", err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/clients/%v/assets", url.QueryEscape(client.ClientID)), http.StatusSeeOther)
}

func assetsHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
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

	assets, err := asset.List(r.Context(), asset.ListFilter{
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
		Section:   "assets",
		Filenames: []string{"gui/assets/list-client.html"},
		Data: map[string]interface{}{
			"Client":       client,
			"Assets":       assets,
			"ShowArchived": showArchived,
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}
