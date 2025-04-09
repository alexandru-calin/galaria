package main

import (
	"net/http"

	"github.com/alexandru-calin/galaria/controllers"
	"github.com/alexandru-calin/galaria/models"
	"github.com/alexandru-calin/galaria/ui"
	"github.com/alexandru-calin/galaria/views"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"
)

func main() {
	r := chi.NewRouter()

	tpl := views.Must(views.ParseFS(ui.FS, "base.html", "home.html"))
	r.Get("/", controllers.StaticHandler(tpl))

	cfg := models.DefaultPostgresConfig()

	db, err := models.Open(cfg)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	userService := models.UserService{
		DB: db,
	}

	usersC := controllers.Users{
		UserService: &userService,
	}

	usersC.Templates.New = views.Must(views.ParseFS(ui.FS, "base.html", "register.html"))
	usersC.Templates.Login = views.Must(views.ParseFS(ui.FS, "base.html", "login.html"))

	r.Get("/register", usersC.New)
	r.Post("/users", usersC.Create)
	r.Get("/login", usersC.Login)
	r.Post("/login", usersC.ProcessLogin)

	csrfKey := "S8XocRepHuI7WOHeWc3RmnxfrrtVVoy0"
	csrfMw := csrf.Protect([]byte(csrfKey), csrf.Secure(false))

	http.ListenAndServe(":4000", csrfMw(r))
}
