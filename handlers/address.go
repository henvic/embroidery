package handlers

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func init() {
	Register(AddressListRoute)
}

// AddressListRoute is /address/:client_id
func AddressListRoute() {
	Router.HandleFunc("/address/{client_id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		fmt.Fprintf(w, "%+v", vars)
	}).Name("address_by_client")
}
