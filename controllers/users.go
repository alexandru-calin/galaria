package controllers

import (
	"fmt"
	"net/http"

	"github.com/alexandru-calin/galaria/models"
)

type Users struct {
	Templates struct {
		New   Template
		Login Template
	}
	UserService *models.UserService
}

func (u Users) New(w http.ResponseWriter, r *http.Request) {
	u.Templates.New.Execute(w, nil)
}

func (u Users) Create(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	user, err := u.UserService.Create(email, password)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "User created: %+v", user)
}

func (u Users) Login(w http.ResponseWriter, r *http.Request) {
	u.Templates.Login.Execute(w, nil)
}

func (u Users) ProcessLogin(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	user, err := u.UserService.Authenticate(email, password)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "User authenticated: %+v", user)
}
