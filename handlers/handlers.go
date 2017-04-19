package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gorilla/mux"
)

var (
	// DB is a handle for the DB
	DB *sql.DB

	// Router (mux)
	Router *mux.Router
)

var routes = []func(){}

// Load routes
func Load(r *mux.Router) {
	Router = r

	r.HandleFunc("/gui/", staticHandler)

	for _, route := range routes {
		route()
	}
}

// Register routes
func Register(r func()) {
	routes = append(routes, r)
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, r.URL.Path[1:])
}
