package controllers

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/alexandru-calin/galaria/context"
	"github.com/alexandru-calin/galaria/errors"
	"github.com/alexandru-calin/galaria/models"
	"github.com/go-chi/chi/v5"
)

type Galleries struct {
	Templates struct {
		New   Template
		Edit  Template
		Index Template
		Show  Template
	}
	GalleryService *models.GalleryService
}

func (g Galleries) New(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Title string
	}
	data.Title = r.FormValue("title")

	g.Templates.New.Execute(w, r, data)
}

func (g Galleries) Create(w http.ResponseWriter, r *http.Request) {
	var data struct {
		UserID int
		Title  string
	}
	data.UserID = context.User(r.Context()).ID
	data.Title = r.FormValue("title")

	gallery, err := g.GalleryService.Create(data.UserID, data.Title)
	if err != nil {
		g.Templates.New.Execute(w, r, data, err)
		return
	}

	editPath := fmt.Sprintf("/galleries/%d/edit", gallery.ID)
	http.Redirect(w, r, editPath, http.StatusFound)
}

func (g Galleries) Edit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	gallery, err := g.GalleryService.ByID(id)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			http.NotFound(w, r)
			return
		}

		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	user := context.User(r.Context())
	if user.ID != gallery.UserID {
		http.Error(w, "You are not authorized to edit this gallery", http.StatusForbidden)
		return
	}

	var data struct {
		ID    int
		Title string
	}
	data.ID = gallery.ID
	data.Title = gallery.Title

	g.Templates.Edit.Execute(w, r, data)
}

func (g Galleries) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	gallery, err := g.GalleryService.ByID(id)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			http.NotFound(w, r)
			return
		}

		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	user := context.User(r.Context())
	if user.ID != gallery.UserID {
		http.Error(w, "You are not authorized to edit this gallery", http.StatusForbidden)
		return
	}

	gallery.Title = r.FormValue("title")

	err = g.GalleryService.Update(gallery)
	if err != nil {
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	editPath := fmt.Sprintf("/galleries/%d/edit", gallery.ID)
	http.Redirect(w, r, editPath, http.StatusFound)
}

func (g Galleries) Show(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	gallery, err := g.GalleryService.ByID(id)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			http.NotFound(w, r)
			return
		}

		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	var data struct {
		Title  string
		Images []string
	}
	data.Title = gallery.Title

	for i := 0; i < 20; i++ {
		x, y := rand.Intn(500)+200, rand.Intn(500)+200

		imageURL := fmt.Sprintf("https://placecats.com/%d/%d", x, y)
		data.Images = append(data.Images, imageURL)
	}

	g.Templates.Show.Execute(w, r, data)
}

func (g Galleries) Index(w http.ResponseWriter, r *http.Request) {
	type Gallery struct {
		ID    int
		Title string
	}
	var data struct {
		Galleries []Gallery
	}

	user := context.User(r.Context())

	galleries, err := g.GalleryService.ByUserID(user.ID)
	if err != nil {
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	for _, gallery := range galleries {
		data.Galleries = append(data.Galleries, Gallery{
			ID:    gallery.ID,
			Title: gallery.Title,
		})
	}

	g.Templates.Index.Execute(w, r, data)
}
