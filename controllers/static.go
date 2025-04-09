package controllers

import (
	"net/http"
)

func StaticHandler(t Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.Execute(w, r, nil)
	}
}
