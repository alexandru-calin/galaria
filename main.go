package main

import (
	"net/http"
	"os"
	"strconv"

	"github.com/alexandru-calin/galaria/controllers"
	"github.com/alexandru-calin/galaria/migrations"
	"github.com/alexandru-calin/galaria/models"
	"github.com/alexandru-calin/galaria/ui"
	"github.com/alexandru-calin/galaria/views"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"
	"github.com/joho/godotenv"
)

type config struct {
	PSQL models.PostgresConfig
	SMTP models.SMTPConfig
	CSRF struct {
		Key    string
		Secure bool
	}
	Server struct {
		Address string
	}
}

func loadEnvConfig() (config, error) {
	var cfg config

	err := godotenv.Load()
	if err != nil {
		return cfg, nil
	}

	cfg.PSQL = models.DefaultPostgresConfig()
	cfg.SMTP.Host = os.Getenv("SMTP_HOST")
	cfg.SMTP.Port, err = strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		return cfg, nil
	}
	cfg.SMTP.Username = os.Getenv("SMTP_USERNAME")
	cfg.SMTP.Password = os.Getenv("SMTP_PASSWORD")
	cfg.CSRF.Key = "S8XocRepHuI7WOHeWc3RmnxfrrtVVoy0"
	cfg.CSRF.Secure = false
	cfg.Server.Address = ":3000"

	return cfg, nil
}

func main() {
	cfg, err := loadEnvConfig()
	if err != nil {
		panic(err)
	}

	// Setup database
	db, err := models.Open(cfg.PSQL)
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
	userService := &models.UserService{
		DB: db,
	}
	sessionService := &models.SessionService{
		DB: db,
	}
	passwordResetService := &models.PasswordResetService{
		DB: db,
	}
	emailService := models.NewEmailService(cfg.SMTP)

	// Setup middleware
	umw := controllers.UserMiddleware{
		SessionService: sessionService,
	}

	csrfMw := csrf.Protect(
		[]byte(cfg.CSRF.Key),
		csrf.Secure(cfg.CSRF.Secure),
		csrf.Path("/"),
	)

	// Setup controllers
	usersC := controllers.Users{
		UserService:          userService,
		SessionService:       sessionService,
		PasswordResetService: passwordResetService,
		EmailService:         emailService,
	}
	usersC.Templates.New = views.Must(views.ParseFS(ui.FS, "base.html", "register.html"))
	usersC.Templates.Login = views.Must(views.ParseFS(ui.FS, "base.html", "login.html"))
	usersC.Templates.ForgotPassword = views.Must(views.ParseFS(ui.FS, "base.html", "forgot-password.html"))
	usersC.Templates.CheckYourEmail = views.Must(views.ParseFS(ui.FS, "base.html", "check-your-email.html"))
	usersC.Templates.ResetPassword = views.Must(views.ParseFS(ui.FS, "base.html", "reset-password.html"))

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
	r.Get("/forgot-password", usersC.ForgotPassword)
	r.Post("/forgot-password", usersC.ProcessForgotPassword)
	r.Get("/reset-password", usersC.ResetPassword)
	r.Post("/reset-password", usersC.ProcessResetPassword)
	r.Route("/users/me", func(r chi.Router) {
		r.Use(umw.RequireUser)
		r.Get("/", usersC.CurrentUser)
	})

	// Start server
	err = http.ListenAndServe(cfg.Server.Address, r)
	if err != nil {
		panic(err)
	}
}
