package views

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
)

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

func (t Template) Execute(w http.ResponseWriter, data any) {
	buf := new(bytes.Buffer)

	err := t.htmlTpl.Execute(buf, data)
	if err != nil {
		http.Error(w, "Error while executing template", http.StatusInternalServerError)
		return
	}

	buf.WriteTo(w)
}
