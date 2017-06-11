package handles

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/henvic/embroidery/server"
	"github.com/henvic/embroidery/sitetemplate"
)

// AuthenticatedHandler is a handler for an authenticated-only request
type AuthenticatedHandler func(w http.ResponseWriter, r *http.Request, s *sessions.Session)

func (h AuthenticatedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	session, err := server.SessionStore.Get(r, server.UserSessionName)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error with session: %v\n", err)
		ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if _, ok := session.Values["authenticated"]; !ok {
		ErrorHandler(w, r, "Access restricted. Please log in.", http.StatusUnauthorized)
		return
	}

	h(w, r, session)
}

// Load routes
func init() {
	var mux = server.Instance.Mux()
	mux.HandleFunc("/", homeHandler)
	mux.PathPrefix("/static").HandlerFunc(staticHandler)
	mux.NotFoundHandler = &notFoundHandler{}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	var t = &sitetemplate.Template{
		Title:          "Dashboard",
		Filenames:      []string{"gui/home/home.html"},
		Data:           map[string]interface{}{},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, fmt.Sprintf("gui/%v", r.URL.Path))
}

type notFoundHandler struct{}

func (n *notFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ErrorHandler(w, r, "Page not found", http.StatusNotFound)
}

// ErrorHandler for errors
func ErrorHandler(w http.ResponseWriter, r *http.Request, error string, code int) {
	w.WriteHeader(code)

	var t = &sitetemplate.Template{
		Title:     http.StatusText(code),
		Section:   "error",
		Filenames: []string{"gui/errors/error.html"},
		Data: map[string]interface{}{
			"ErrorStatusText": http.StatusText(code),
			"Error":           error,
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}
