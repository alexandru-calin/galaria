package main

import (
	"net/http"

	"github.com/alexandru-calin/galaria/ui"
	"github.com/alexandru-calin/galaria/views"
	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	tpl := views.Must(views.ParseFS(ui.FS, "base.html"))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		tpl.Execute(w, nil)
	})

	http.ListenAndServe(":4000", r)
}
