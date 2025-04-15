package views

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"github.com/alexandru-calin/galaria/context"
	"github.com/alexandru-calin/galaria/errors"
	"github.com/alexandru-calin/galaria/models"
	"github.com/gorilla/csrf"
)

type publicError interface {
	Public() string
}

func Must(t Template, err error) Template {
	if err != nil {
		panic(err)
	}

	return t
}

func ParseFS(fs fs.FS, patterns ...string) (Template, error) {
	tpl := template.New(patterns[0])

	tpl = tpl.Funcs(template.FuncMap{
		"csrfField": func() (template.HTML, error) {
			return "", fmt.Errorf("csrfField not implemented")
		},
		"currentUser": func() (*models.User, error) {
			return nil, fmt.Errorf("currentUser not implemented")
		},
		"errors": func() []string {
			return nil
		},
	})

	tpl, err := tpl.ParseFS(fs, patterns...)
	if err != nil {
		return Template{}, fmt.Errorf("parsing template: %w", err)
	}

	return Template{
		htmlTpl: tpl,
	}, nil
}

func Parse(patterns ...string) (Template, error) {
	tpl, err := template.ParseFiles(patterns...)
	if err != nil {
		return Template{}, fmt.Errorf("parsing template: %w", err)
	}

	return Template{
		htmlTpl: tpl,
	}, nil
}

type Template struct {
	htmlTpl *template.Template
}

func (t Template) Execute(w http.ResponseWriter, r *http.Request, data any, errs ...error) {
	tpl, err := t.htmlTpl.Clone()
	if err != nil {
		log.Printf("cloning template: %v", err)
		http.Error(w, "Error while rendering the page", http.StatusInternalServerError)
		return
	}

	tpl = tpl.Funcs(template.FuncMap{
		"csrfField": func() template.HTML {
			return csrf.TemplateField(r)
		},
		"currentUser": func() *models.User {
			return context.User(r.Context())
		},
		"errors": func() []string {
			var errMessages []string
			for _, err := range errs {
				var pubErr publicError
				if errors.As(err, &pubErr) {
					errMessages = append(errMessages, pubErr.Public())
				} else {
					errMessages = append(errMessages, "Something went wrong")
				}
			}

			return errMessages
		},
	})

	buf := new(bytes.Buffer)

	err = tpl.Execute(buf, data)
	if err != nil {
		log.Printf("executing template: %v", err)
		http.Error(w, "Error while rendering the page", http.StatusInternalServerError)
		return
	}

	buf.WriteTo(w)
}
