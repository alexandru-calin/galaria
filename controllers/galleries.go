package controllers

import "net/http"

type Galleries struct {
	Templates struct {
		New Template
	}
}

func (g Galleries) New(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Title string
	}
	data.Title = r.FormValue("title")

	g.Templates.New.Execute(w, r, data)
}
