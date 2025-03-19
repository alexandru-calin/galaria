package main

import (
	"net/http"

	"github.com/alexandru-calin/galaria/ui"
	"github.com/alexandru-calin/galaria/views"
)

func main() {
	mux := http.NewServeMux()

	tpl, err := views.ParseFS(ui.FS, "base.html")
	if err != nil {
		panic(err)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tpl.Execute(w, nil)
	})

	http.ListenAndServe(":4000", mux)
}
