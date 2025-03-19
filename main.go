package main

import (
	"net/http"

	"github.com/alexandru-calin/galaria/controllers"
	"github.com/alexandru-calin/galaria/ui"
	"github.com/alexandru-calin/galaria/views"
	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	tpl := views.Must(views.ParseFS(ui.FS, "base.html", "home.html"))
	r.Get("/", controllers.StaticHandler(tpl))

	usersC := controllers.Users{}
	usersC.Templates.New = views.Must(views.ParseFS(ui.FS, "base.html", "register.html"))

	r.Get("/register", usersC.New)

	http.ListenAndServe(":4000", r)
}
