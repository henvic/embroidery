package authhandles

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	schema "github.com/gorilla/Schema"
	"github.com/henvic/embroidery/auth"
	"github.com/henvic/embroidery/handles"
	"github.com/henvic/embroidery/server"
	"golang.org/x/crypto/bcrypt"
)

var router = server.Instance.Mux

type loginForm struct {
	Email    string
	Password string
}

func init() {
	router().HandleFunc("/login", loginHandler)
	router().HandleFunc("/logout", logoutHandler)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var session, err = server.SessionStore.Get(r, server.UserSessionName)

	if err != nil {
		log.Printf("Session store error: %v", err)
		handles.ErrorHandler(w, r, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodGet {
		http.Redirect(w, r, "/gui/login", http.StatusFound)
		return
	}

	if r.Method != http.MethodPost {
		handles.ErrorHandler(w, r, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if ok, iok := session.Values["authenticated"].(bool); iok && ok {
		handles.ErrorHandler(w, r, "Conflict: user is already signed in", http.StatusConflict)
	}

	if err := r.ParseForm(); err != nil {
		handles.ErrorHandler(w, r, "Invalid form", http.StatusBadRequest)
		return
	}

	decoder := schema.NewDecoder()
	login := loginForm{}

	if err := decoder.Decode(&login, r.PostForm); err != nil {
		handles.ErrorHandler(w, r, fmt.Sprintf("Error decoding request body: %v", err), http.StatusBadRequest)
		return
	}

	if len(login.Email) == 0 {
		handles.ErrorHandler(w, r, "Wrong credentials.", http.StatusUnauthorized)
		return
	}

	auth, err := auth.GetAuthenticationByEmail(context.Background(), login.Email)

	switch err {
	case nil:
		break
	case sql.ErrNoRows:
		handles.ErrorHandler(w, r, "Wrong credentials.", http.StatusUnauthorized)
		return
	default:
		log.Printf("Error getting authentication data from DB: %v", err)
		handles.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(auth.Password), []byte(login.Password)); err != nil {
		handles.ErrorHandler(w, r, "Wrong credentials.", http.StatusUnauthorized)
		return
	}

	session.Values["user"] = auth.EmployeeID
	session.Values["email"] = auth.Email
	session.Values["authenticated"] = true
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handles.ErrorHandler(w, r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	session, _ := server.SessionStore.Get(r, server.UserSessionName)
	session.Values["authenticated"] = false
	delete(session.Values, "authenticated")
	session.Values = map[interface{}]interface{}{}
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
