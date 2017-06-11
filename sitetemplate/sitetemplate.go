package sitetemplate

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/henvic/embroidery/server"
)

// this is not optimal for production
// there is no cache, etc
// but who cares? this is only for a db-related class project

// Template to use
type Template struct {
	Title          string
	Section        string
	Filenames      []string
	Data           interface{}
	Request        *http.Request
	ResponseWriter http.ResponseWriter
}

const base = "gui/template.html"

var basicFunctions = template.FuncMap{
	"json": func(v interface{}) string {
		a, _ := json.Marshal(v)
		return string(a)
	},
	"split": strings.Split,
	"join":  strings.Join,
	"title": strings.Title,
	"lower": strings.ToLower,
	"upper": strings.ToUpper,
}

func (t *Template) isSectionActiveFunc(v interface{}) bool {
	return v.(string) == t.Section
}

func (t *Template) printSectionActiveFunc(section string) string {
	return t.printValueIfSectionIsActiveFunc(section, " active ")
}

func (t *Template) printValueIfSectionIsActiveFunc(section string, value string) string {
	if t.isSectionActiveFunc(section) {
		return " active "
	}

	return ""
}

// Execute template
func (t *Template) Execute() error {
	var files = []string{base}
	files = append(files, t.Filenames...)

	var to = template.New("").Funcs(basicFunctions).Funcs(template.FuncMap{
		"isSectionActive":           t.isSectionActiveFunc,
		"printSectionActive":        t.printSectionActiveFunc,
		"printValueIfSectionActive": t.printValueIfSectionIsActiveFunc,
	})

	to, err := to.ParseFiles(files...)

	if err != nil {
		return err
	}

	var maps = map[string]interface{}{
		"title":   t.Title,
		"Data":    t.Data,
		"Session": map[interface{}]interface{}{},
	}

	if t.Request != nil {
		session, err := server.SessionStore.Get(t.Request, server.UserSessionName)

		if err != nil {
			return err
		}

		maps["Session"] = session.Values
	}

	return to.ExecuteTemplate(t.ResponseWriter, "base", maps)
}

// Respond request
func (t *Template) Respond() {
	if err := t.Execute(); err != nil {
		http.Error(t.ResponseWriter, "Internal Server Error: template parsing", http.StatusInternalServerError)
		fmt.Fprintf(os.Stderr, "%+v\n", err)
	}
}
