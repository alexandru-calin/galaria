package main

import (
	"fmt"
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

	cfg.PSQL = models.PostgresConfig{
		User:     os.Getenv("PSQL_USER"),
		Password: os.Getenv("PSQL_PASSWORD"),
		Host:     os.Getenv("PSQL_HOST"),
		Port:     os.Getenv("PSQL_PORT"),
		Database: os.Getenv("PSQL_DATABASE"),
		SSLMode:  os.Getenv("PSQL_SSLMODE"),
	}

	cfg.SMTP.Host = os.Getenv("SMTP_HOST")
	cfg.SMTP.Port, err = strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		return cfg, nil
	}
	cfg.SMTP.Username = os.Getenv("SMTP_USERNAME")
	cfg.SMTP.Password = os.Getenv("SMTP_PASSWORD")

	cfg.CSRF.Key = os.Getenv("CSRF_KEY")
	cfg.CSRF.Secure = os.Getenv("CSRF_SECURE") == "true"

	cfg.Server.Address = os.Getenv("SERVER_ADDRESS")

	return cfg, nil
}

func main() {
	cfg, err := loadEnvConfig()
	if err != nil {
		panic(err)
	}

	err = run(cfg)
	if err != nil {
		panic(err)
	}
}

func run(cfg config) error {
	// Setup database
	db, err := models.Open(cfg.PSQL)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		return err
	}

	err = models.MigrateFS(db, migrations.FS, ".")
	if err != nil {
		return err
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
	galleryService := &models.GalleryService{
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
		GalleryService:       galleryService,
		SessionService:       sessionService,
		PasswordResetService: passwordResetService,
		EmailService:         emailService,
	}
	usersC.Templates.Home = views.Must(views.ParseFS(ui.FS, "base.html", "home.html"))
	usersC.Templates.New = views.Must(views.ParseFS(ui.FS, "base.html", "users/register.html"))
	usersC.Templates.Login = views.Must(views.ParseFS(ui.FS, "base.html", "users/login.html"))
	usersC.Templates.ForgotPassword = views.Must(views.ParseFS(ui.FS, "base.html", "users/password-forgot.html"))
	usersC.Templates.CheckYourEmail = views.Must(views.ParseFS(ui.FS, "base.html", "users/check-your-email.html"))
	usersC.Templates.ResetPassword = views.Must(views.ParseFS(ui.FS, "base.html", "users/password-reset.html"))
	usersC.Templates.Me = views.Must(views.ParseFS(ui.FS, "base.html", "users/me.html"))

	galleriesC := controllers.Galleries{
		GalleryService: galleryService,
	}
	galleriesC.Templates.New = views.Must(views.ParseFS(ui.FS, "base.html", "galleries/new.html"))
	galleriesC.Templates.Edit = views.Must(views.ParseFS(ui.FS, "base.html", "galleries/edit.html"))
	galleriesC.Templates.Index = views.Must(views.ParseFS(ui.FS, "base.html", "galleries/index.html"))
	galleriesC.Templates.Show = views.Must(views.ParseFS(ui.FS, "base.html", "galleries/show.html"))

	// Setup router and routes
	r := chi.NewRouter()

	r.Use(csrfMw)
	r.Use(umw.SetTheme)
	r.Use(umw.SetUser)

	assetsHandler := http.FileServer(http.Dir("assets"))

	r.Get("/", usersC.Home)
	r.Get("/assets/*", http.StripPrefix("/assets", assetsHandler).ServeHTTP)
	r.Get("/register", usersC.New)
	r.Get("/login", usersC.Login)
	r.Post("/login", usersC.ProcessLogin)
	r.Post("/logout", usersC.ProcessLogout)
	r.Get("/forgot-password", usersC.ForgotPassword)
	r.Post("/forgot-password", usersC.ProcessForgotPassword)
	r.Get("/reset-password", usersC.ResetPassword)
	r.Post("/reset-password", usersC.ProcessResetPassword)
	r.Post("/change-theme", usersC.ChangeTheme)
	r.Route("/users", func(r chi.Router) {
		r.Post("/", usersC.Create)
		r.Group(func(r chi.Router) {
			r.Use(umw.RequireUser)
			r.Get("/me", usersC.Me)
			r.Post("/me/delete", usersC.Delete)
		})
	})
	r.Route("/galleries", func(r chi.Router) {
		r.Get("/{id}", galleriesC.Show)
		r.Get("/{id}/images/{filename}", galleriesC.Image)
		r.Group(func(r chi.Router) {
			r.Use(umw.RequireUser)
			r.Get("/", galleriesC.Index)
			r.Get("/new", galleriesC.New)
			r.Post("/", galleriesC.Create)
			r.Get("/{id}/edit", galleriesC.Edit)
			r.Post("/{id}", galleriesC.Update)
			r.Post("/{id}/images", galleriesC.UploadImage)
			r.Post("/{id}/delete", galleriesC.Delete)
			r.Post("/{id}/images/{filename}/delete", galleriesC.DeleteImage)
		})
	})

	// Start server
	fmt.Printf("Starting the server on %s\n", cfg.Server.Address)
	return http.ListenAndServe(cfg.Server.Address, r)
}
