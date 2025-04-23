package controllers

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/alexandru-calin/galaria/context"
	"github.com/alexandru-calin/galaria/errors"
	"github.com/alexandru-calin/galaria/models"
)

type Users struct {
	Templates struct {
		New            Template
		Login          Template
		ForgotPassword Template
		CheckYourEmail Template
		ResetPassword  Template
	}
	UserService          *models.UserService
	SessionService       *models.SessionService
	PasswordResetService *models.PasswordResetService
	EmailService         *models.EmailService
}

func (u Users) New(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string
	}
	data.Email = r.FormValue("email")

	u.Templates.New.Execute(w, r, data)
}

func (u Users) Create(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email    string
		Password string
	}

	data.Email = r.FormValue("email")
	data.Password = r.FormValue("password")

	user, err := u.UserService.Create(data.Email, data.Password)
	if err != nil {
		if errors.Is(err, models.ErrEmailTaken) {
			msg := "This email address is already taken. Please try another."
			err = errors.Public(err, msg)
		}
		u.Templates.New.Execute(w, r, data, err)
		return
	}

	session, err := u.SessionService.Create(user.ID)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	flash := fmt.Sprintf("Successfully registered and logged in as %s", data.Email)

	setCookie(w, CookieSession, session.Token)
	setCookie(w, CookieFlash, flash)

	http.Redirect(w, r, "/galleries", http.StatusFound)
}

func (u Users) Login(w http.ResponseWriter, r *http.Request) {
	flash, err := readCookie(r, CookieFlash)
	if err != nil {
		if !errors.Is(err, http.ErrNoCookie) {
			fmt.Println(err)
			http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
			return
		}
		u.Templates.Login.Execute(w, r, nil)
		return
	}

	var data struct {
		Flash string
	}

	data.Flash = flash
	deleteCookie(w, CookieFlash)

	u.Templates.Login.Execute(w, r, data)
}

func (u Users) ProcessLogin(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	user, err := u.UserService.Authenticate(email, password)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			w.WriteHeader(http.StatusBadRequest)
			msg := "Incorrect email or password."
			err = errors.Public(err, msg)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Println(err)
		u.Templates.Login.Execute(w, r, nil, err)
		return
	}

	session, err := u.SessionService.Create(user.ID)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	flash := fmt.Sprintf("Successfully logged in as %s", email)

	setCookie(w, CookieSession, session.Token)
	setCookie(w, CookieFlash, flash)
	http.Redirect(w, r, "/galleries", http.StatusFound)
}

func (u Users) ProcessLogout(w http.ResponseWriter, r *http.Request) {
	token, err := readCookie(r, CookieSession)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	err = u.SessionService.Delete(token)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	setCookie(w, CookieFlash, "Logged out successfully")
	deleteCookie(w, CookieSession)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (u Users) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	u.Templates.ForgotPassword.Execute(w, r, nil)
}

func (u Users) ProcessForgotPassword(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")

	pwReset, err := u.PasswordResetService.Create(email)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	vals := url.Values{
		"token": {pwReset.Token},
	}
	resetURL := "https://www.galaria.com/reset-password?" + vals.Encode()

	err = u.EmailService.ForgotPassword(email, resetURL)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	u.Templates.CheckYourEmail.Execute(w, r, nil)
}

func (u Users) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Token string
	}

	data.Token = r.FormValue("token")
	u.Templates.ResetPassword.Execute(w, r, data)
}

func (u Users) ProcessResetPassword(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	password := r.FormValue("password")

	user, err := u.PasswordResetService.Consume(token)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	err = u.UserService.UpdatePassword(user.ID, password)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	session, err := u.SessionService.Create(user.ID)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	setCookie(w, CookieSession, session.Token)
	http.Redirect(w, r, "/users/me", http.StatusFound)
}

func (u Users) CurrentUser(w http.ResponseWriter, r *http.Request) {
	user := context.User(r.Context())
	fmt.Fprintf(w, "Current user: %s\n", user.Email)
}

func (u Users) ChangeTheme(w http.ResponseWriter, r *http.Request) {
	theme := r.FormValue("theme")
	setCookie(w, CookieTheme, theme)

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusFound)
}

type UserMiddleware struct {
	SessionService *models.SessionService
}

func (umw UserMiddleware) SetUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := readCookie(r, CookieSession)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		user, err := umw.SessionService.User(token)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		ctx = context.WithUser(ctx, user)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func (umw UserMiddleware) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := context.User(r.Context())
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (umw UserMiddleware) SetTheme(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		theme, err := readCookie(r, CookieTheme)
		if err != nil {
			theme = "dark"
			setCookie(w, CookieTheme, theme)
		}

		ctx := r.Context()
		ctx = context.WithTheme(ctx, theme)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}
