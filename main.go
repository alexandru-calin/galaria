package main

import (
	"net/http"

	"github.com/alexandru-calin/galaria/controllers"
	"github.com/alexandru-calin/galaria/migrations"
	"github.com/alexandru-calin/galaria/models"
	"github.com/alexandru-calin/galaria/ui"
	"github.com/alexandru-calin/galaria/views"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"
)

func main() {
	// Setup database
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

	err = models.MigrateFS(db, migrations.FS, ".")
	if err != nil {
		panic(err)
	}

	// Setup services
	userService := models.UserService{
		DB: db,
	}
	sessionService := models.SessionService{
		DB: db,
	}

	// Setup middleware
	umw := controllers.UserMiddleware{
		SessionService: &sessionService,
	}

	csrfKey := "S8XocRepHuI7WOHeWc3RmnxfrrtVVoy0"
	csrfMw := csrf.Protect([]byte(csrfKey), csrf.Secure(false))

	// Setup controllers
	usersC := controllers.Users{
		UserService:    &userService,
		SessionService: &sessionService,
	}
	usersC.Templates.New = views.Must(views.ParseFS(ui.FS, "base.html", "register.html"))
	usersC.Templates.Login = views.Must(views.ParseFS(ui.FS, "base.html", "login.html"))

	// Setup router and routes
	r := chi.NewRouter()

	r.Use(csrfMw)
	r.Use(umw.SetUser)

	r.Get("/", controllers.StaticHandler(views.Must(views.ParseFS(ui.FS, "base.html", "home.html"))))
	r.Get("/register", usersC.New)
	r.Post("/users", usersC.Create)
	r.Get("/login", usersC.Login)
	r.Post("/login", usersC.ProcessLogin)
	r.Post("/logout", usersC.ProcessLogout)
	r.Route("/users/me", func(r chi.Router) {
		r.Use(umw.RequireUser)
		r.Get("/", usersC.CurrentUser)
	})

	// Start server
	http.ListenAndServe(":4000", r)
}
