package employeeshandles

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/henvic/embroidery/employees"
	"github.com/henvic/embroidery/handles"
	"github.com/henvic/embroidery/server"
	"github.com/henvic/embroidery/sitetemplate"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
)

var router = server.Instance.Mux

func init() {
	router().Handle("/employees", handles.AuthenticatedHandler(employeesHandler))
	router().Handle("/employees/add", handles.AuthenticatedHandler(createHandler))
	router().Handle("/employees/{employee_id}", handles.AuthenticatedHandler(editHandler))
}

func employeesHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	var showRevoked = (r.URL.Query().Get("showRevoked") != "")

	employees, err := employees.List(r.Context())

	if err != nil {
		handles.ErrorHandler(w, r, "Can't get employees list", http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		return
	}

	var t = &sitetemplate.Template{
		Title:     "Employees",
		Section:   "employees",
		Filenames: []string{"gui/employees/employees.html"},
		Data: map[string]interface{}{
			"Employees":   employees,
			"ShowRevoked": showRevoked,
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}

func createHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	if r.Method == http.MethodGet {
		var t = &sitetemplate.Template{
			Title:          "Add an employee",
			Section:        "employees",
			Filenames:      []string{"gui/employees/create.html"},
			Data:           map[string]interface{}{},
			Request:        r,
			ResponseWriter: w,
		}

		t.Respond()
		return
	}

	var email = r.PostFormValue("email")
	var password = r.PostFormValue("password")

	if email == "" {
		handles.ErrorHandler(w, r, "Missing email parameter", http.StatusBadRequest)
		return
	}

	if password == "" {
		handles.ErrorHandler(w, r, "Missing password parameter", http.StatusBadRequest)
		return
	}

	var accessLevel = r.PostFormValue("access_level")

	switch accessLevel {
	case "owner", "employee", "revoked":
	default:
		handles.ErrorHandler(w, r, "Missing / invalid accessLevel parameter", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		handles.ErrorHandler(w, r, "Error generating password with bcrypt", http.StatusInternalServerError)
		return
	}

	var employee = employees.Employee{
		EmployeeID:  uuid.NewV4().String(),
		Email:       email,
		AccessLevel: accessLevel,
		Password:    string(hash),
	}

	if err := employees.Create(r.Context(), employee); err != nil {
		handles.ErrorHandler(w, r, "Internal Server Error: saving user", http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		return
	}

	http.Redirect(w, r, "/employees", http.StatusSeeOther)
}

func editHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	vars := mux.Vars(r)
	employeeID, ok := vars["employee_id"]

	if !ok {
		handles.ErrorHandler(w, r, "Missing employee ID parameter", http.StatusBadRequest)
		return
	}

	var employee, err = employees.Get(r.Context(), employeeID)

	if err != nil {
		handles.ErrorHandler(w, r, fmt.Sprintf("Internal Server Error: %v", err), http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodGet:
		editHandlerGetHandler(w, r, employee)
	case http.MethodPost:
		editHandlerPostHandler(w, r, employee)
		return
	default:
		handles.ErrorHandler(w, r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func editHandlerGetHandler(w http.ResponseWriter, r *http.Request, employee employees.Employee) {
	var t = sitetemplate.Template{
		Title:     "Edit employee access",
		Section:   "employees",
		Filenames: []string{"gui/employees/edit.html"},
		Data: map[string]interface{}{
			"Employee": employee,
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}

func editHandlerPostHandler(w http.ResponseWriter, r *http.Request, employee employees.Employee) {
	if err := r.ParseForm(); err != nil {
		handles.ErrorHandler(w, r, "Internal Server Error: parsing employee edit form", http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "%+v\n", err)
	}

	var email = r.PostFormValue("email")

	if !strings.Contains(email, "@") {
		handles.ErrorHandler(w, r, "Internal Server Error: email is invalid", http.StatusInternalServerError)
		return
	}

	var accessLevel = r.PostFormValue("access_level")

	switch accessLevel {
	case "owner", "employee", "revoked":
	default:
		handles.ErrorHandler(w, r,
			fmt.Sprintf("Internal Server Error: access_level is not recognized: %v", accessLevel),
			http.StatusInternalServerError)
		return
	}

	employee.Email = email
	employee.AccessLevel = accessLevel

	var password = r.PostFormValue("password")

	if password != "" {
		var hash, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

		if err != nil {
			handles.ErrorHandler(w, r, "Error generating password with bcrypt", http.StatusInternalServerError)
			return
		}

		employee.Password = string(hash)
	}

	if err := employees.Update(r.Context(), employee); err != nil {
		handles.ErrorHandler(w, r, "Internal Server Error: saving user", http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		return
	}

	http.Redirect(w, r, "/employees", http.StatusSeeOther)
}
