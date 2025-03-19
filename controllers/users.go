package controllers

import (
	"net/http"

	"github.com/alexandru-calin/galaria/views"
)

type User struct {
	Templates struct {
		New views.Template
	}
}

func (u User) New(w http.ResponseWriter, r *http.Request) {
	u.Templates.New.Execute(w, nil)
}
