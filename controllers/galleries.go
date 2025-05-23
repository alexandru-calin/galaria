package controllers

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
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
		All   Template
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

	setCookie(w, CookieFlash, "Gallery created successfully")

	editPath := fmt.Sprintf("/galleries/%d/edit", gallery.ID)
	http.Redirect(w, r, editPath, http.StatusFound)
}

func (g Galleries) Edit(w http.ResponseWriter, r *http.Request) {
	gallery, err := g.galleryByID(w, r, userMustOwnGallery)
	if err != nil {
		return
	}

	type Image struct {
		GalleryID       int
		Filename        string
		FilenameEscaped string
	}

	var data struct {
		ID        int
		Title     string
		Images    []Image
		UpdatedAt string
		Flash     string
	}
	data.ID = gallery.ID
	data.Title = gallery.Title
	data.UpdatedAt = gallery.UpdatedAt.Format("January 02, 2006 15:04")

	images, err := g.GalleryService.Images(gallery.ID)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	for _, image := range images {
		data.Images = append(data.Images, Image{
			GalleryID:       gallery.ID,
			Filename:        image.Filename,
			FilenameEscaped: url.PathEscape(image.Filename),
		})
	}

	flash, err := readCookie(r, CookieFlash)
	if err != nil {
		if !errors.Is(err, http.ErrNoCookie) {
			fmt.Println(err)
			http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
			return
		}
		g.Templates.Edit.Execute(w, r, data)
		return
	}

	data.Flash = flash
	deleteCookie(w, CookieFlash)

	g.Templates.Edit.Execute(w, r, data)
}

func (g Galleries) Update(w http.ResponseWriter, r *http.Request) {
	gallery, err := g.galleryByID(w, r, userMustOwnGallery)
	if err != nil {
		return
	}

	gallery.Title = r.FormValue("title")

	err = g.GalleryService.Update(gallery)
	if err != nil {
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	setCookie(w, CookieFlash, "Gallery updated successfully")

	editPath := fmt.Sprintf("/galleries/%d/edit", gallery.ID)
	http.Redirect(w, r, editPath, http.StatusFound)
}

func (g Galleries) Show(w http.ResponseWriter, r *http.Request) {
	gallery, err := g.galleryByID(w, r)
	if err != nil {
		return
	}

	type Image struct {
		GalleryID       int
		Filename        string
		FilenameEscaped string
		CreatedAt       string
	}

	var data struct {
		Title     string
		Images    []Image
		UpdatedAt string
	}
	data.Title = gallery.Title
	data.UpdatedAt = gallery.UpdatedAt.Format("January 02, 2006 15:04")

	images, err := g.GalleryService.Images(gallery.ID)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	for _, image := range images {
		data.Images = append(data.Images, Image{
			GalleryID:       image.GalleryID,
			Filename:        image.Filename,
			FilenameEscaped: url.PathEscape(image.Filename),
			CreatedAt:       image.CreatedAt.Format("January 02, 2006 15:04"),
		})
	}

	g.Templates.Show.Execute(w, r, data)
}

func (g Galleries) Image(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Base(chi.URLParam(r, "filename"))

	galleryID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	image, err := g.GalleryService.Image(galleryID, filename)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			http.NotFound(w, r)
			return
		}

		fmt.Println(err)
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	http.ServeFile(w, r, image.Path)
}

func (g Galleries) UploadImage(w http.ResponseWriter, r *http.Request) {
	gallery, err := g.galleryByID(w, r, userMustOwnGallery)
	if err != nil {
		return
	}

	err = r.ParseMultipartForm(5 << 20)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	fileHeaders := r.MultipartForm.File["images"]
	for _, fileHeader := range fileHeaders {
		file, err := fileHeader.Open()
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		err = g.GalleryService.CreateImage(gallery.ID, fileHeader.Filename, file)
		if err != nil {
			var fileErr models.FileError
			if errors.As(err, &fileErr) {
				msg := fmt.Sprintf("%v has an invalid content type or extension", fileHeader.Filename)
				http.Error(w, msg, http.StatusBadRequest)
				return
			}

			fmt.Println(err)
			http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
			return
		}
	}

	err = g.GalleryService.Update(gallery)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	setCookie(w, CookieFlash, "Gallery updated successfully")

	editPath := fmt.Sprintf("/galleries/%d/edit", gallery.ID)
	http.Redirect(w, r, editPath, http.StatusFound)
}

func (g Galleries) DeleteImage(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Base(chi.URLParam(r, "filename"))

	gallery, err := g.galleryByID(w, r, userMustOwnGallery)
	if err != nil {
		return
	}

	err = g.GalleryService.DeleteImage(gallery.ID, filename)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	err = g.GalleryService.Update(gallery)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	setCookie(w, CookieFlash, "Gallery updated successfully")

	editPath := fmt.Sprintf("/galleries/%d/edit", gallery.ID)
	http.Redirect(w, r, editPath, http.StatusFound)
}

func (g Galleries) Delete(w http.ResponseWriter, r *http.Request) {
	gallery, err := g.galleryByID(w, r, userMustOwnGallery)
	if err != nil {
		return
	}

	err = g.GalleryService.Delete(gallery.ID)
	if err != nil {
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	setCookie(w, CookieFlash, "Gallery deleted successfully")

	http.Redirect(w, r, "/galleries", http.StatusFound)
}

func (g Galleries) Index(w http.ResponseWriter, r *http.Request) {
	type Gallery struct {
		ID        int
		Title     string
		CreatedAt string
	}
	var data struct {
		Galleries []Gallery
		Flash     string
		Sort      string
		Order     string
	}

	sort := r.FormValue("s")
	if sort == "" {
		sort = "created_at"
	}

	order := r.FormValue("o")
	if order == "" {
		order = "desc"
	}

	user := context.User(r.Context())

	galleries, err := g.GalleryService.ByUserID(user.ID, sort, order)
	if err != nil {
		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return
	}

	for _, gallery := range galleries {
		data.Galleries = append(data.Galleries, Gallery{
			ID:        gallery.ID,
			Title:     gallery.Title,
			CreatedAt: gallery.CreatedAt.Format("01-02-2006 15:04"),
		})
	}

	flash, err := readCookie(r, CookieFlash)
	if err != nil {
		if !errors.Is(err, http.ErrNoCookie) {
			fmt.Println(err)
			http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
			return
		}
	}

	data.Flash = flash
	deleteCookie(w, CookieFlash)

	data.Order = order
	data.Sort = sort

	g.Templates.Index.Execute(w, r, data)
}

type galleryOpt func(w http.ResponseWriter, r *http.Request, gallery *models.Gallery) error

func (g Galleries) galleryByID(w http.ResponseWriter, r *http.Request, opts ...galleryOpt) (*models.Gallery, error) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return nil, err
	}

	gallery, err := g.GalleryService.ByID(id)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			http.NotFound(w, r)
			return nil, err
		}

		http.Error(w, "Oops, something went wrong...", http.StatusInternalServerError)
		return nil, err
	}

	for _, opt := range opts {
		err := opt(w, r, gallery)
		if err != nil {
			return nil, err
		}
	}

	return gallery, nil
}

func userMustOwnGallery(w http.ResponseWriter, r *http.Request, gallery *models.Gallery) error {
	user := context.User(r.Context())
	if user.ID != gallery.UserID {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return fmt.Errorf("user does not own the gallery")
	}

	return nil
}
