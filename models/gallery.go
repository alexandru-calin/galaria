package models

import (
	"database/sql"
	"io"
	"io/fs"
	"slices"
	"sort"
	"syscall"
	"time"

	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexandru-calin/galaria/errors"
)

const (
	DefaultImagesDir = "images"
)

type Image struct {
	GalleryID int
	Path      string
	Filename  string
	CreatedAt time.Time
}

type Gallery struct {
	ID        int
	UserID    int
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type GalleryService struct {
	DB        *sql.DB
	ImagesDir string
}

func (gs *GalleryService) Create(userID int, title string) (*Gallery, error) {
	gallery := Gallery{
		UserID: userID,
		Title:  title,
	}

	row := gs.DB.QueryRow(`
		INSERT INTO galleries (user_id, title)
		VALUES ($1, $2)
		RETURNING id, created_at`, gallery.UserID, gallery.Title)

	err := row.Scan(&gallery.ID, &gallery.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating gallery: %w", err)
	}

	return &gallery, nil
}

func (gs *GalleryService) All() ([]Gallery, error) {
	rows, err := gs.DB.Query(`
		SELECT id, title, created_at, updated_at FROM galleries ORDER BY created_at DESC`)

	if err != nil {
		return nil, fmt.Errorf("retrieving all galleries: %w", err)
	}

	var galleries []Gallery

	for rows.Next() {
		var gallery Gallery

		err = rows.Scan(&gallery.ID, &gallery.Title, &gallery.CreatedAt, &gallery.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("retrieving all galleries: %w", err)
		}

		galleries = append(galleries, gallery)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("retrieving all galleries: %w", err)
	}

	return galleries, nil
}

func (gs *GalleryService) ByID(id int) (*Gallery, error) {
	gallery := Gallery{
		ID: id,
	}

	row := gs.DB.QueryRow(`
		SELECT user_id, title, created_at, updated_at FROM galleries WHERE id=$1`, gallery.ID)

	err := row.Scan(&gallery.UserID, &gallery.Title, &gallery.CreatedAt, &gallery.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("query gallery by id: %w", err)
	}

	return &gallery, nil
}

func (gs *GalleryService) ByUserID(userID int, sort, order string) ([]Gallery, error) {
	sort = strings.ToLower(sort)
	order = strings.ToUpper(order)

	if !slices.Contains(gs.sortableColumns(), sort) {
		sort = "created_at"
	}

	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}

	query := fmt.Sprintf("SELECT id, title, created_at FROM galleries WHERE user_id=$1 ORDER BY %s %s", sort, order)
	rows, err := gs.DB.Query(query, userID)

	if err != nil {
		return nil, fmt.Errorf("query galleries by user: %w", err)
	}

	var galleries []Gallery

	for rows.Next() {
		gallery := Gallery{
			UserID: userID,
		}

		err := rows.Scan(&gallery.ID, &gallery.Title, &gallery.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("query galleries by user: %w", err)
		}

		galleries = append(galleries, gallery)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("query galleries by user: %w", err)
	}

	return galleries, nil
}

func (gs *GalleryService) Update(gallery *Gallery) error {
	_, err := gs.DB.Exec(`
		UPDATE galleries
		SET title=$2, updated_at=$3
		WHERE id=$1`, gallery.ID, gallery.Title, time.Now())

	if err != nil {
		return fmt.Errorf("updating gallery: %w", err)
	}

	return nil
}

func (gs *GalleryService) Delete(id int) error {
	_, err := gs.DB.Exec(`
		DELETE FROM galleries WHERE id=$1`, id)

	if err != nil {
		return fmt.Errorf("deleting gallery: %w", err)
	}

	err = os.RemoveAll(gs.galleryDir(id))
	if err != nil {
		return fmt.Errorf("deleting gallery images: %w", err)
	}

	return nil
}

func (gs *GalleryService) DeleteByUserID(id int) error {
	rows, err := gs.DB.Query(`
		DELETE FROM galleries WHERE user_id=$1 RETURNING id`, id)

	if err != nil {
		return fmt.Errorf("deleting galleries by user: %w", err)
	}

	for rows.Next() {
		var id int
		rows.Scan(&id)

		err = os.RemoveAll(gs.galleryDir(id))
		if err != nil {
			return fmt.Errorf("deleting galleries by user:%w", err)
		}
	}

	err = rows.Err()
	if err != nil {
		return fmt.Errorf("deleting galleries by user:%w", err)
	}

	return nil
}

func (gs *GalleryService) Images(galleryID int) ([]Image, error) {
	globPattern := filepath.Join(gs.galleryDir(galleryID), "*")

	files, err := filepath.Glob(globPattern)
	if err != nil {
		return nil, fmt.Errorf("getting images: %w", err)
	}

	var images []Image
	for _, file := range files {
		if hasExtension(file, gs.extensions()) {
			fileInfo, err := os.Stat(file)
			if err != nil {
				return nil, fmt.Errorf("getting images info: %w", err)
			}

			stat, ok := fileInfo.Sys().(*syscall.Stat_t)
			if !ok {
				return nil, fmt.Errorf("getting image info: %w", err)
			}

			images = append(images, Image{
				GalleryID: galleryID,
				Path:      file,
				Filename:  filepath.Base(file),
				CreatedAt: time.Unix(stat.Ctim.Sec, stat.Ctim.Nsec),
			})
		}
	}

	sort.SliceStable(images, func(i, j int) bool {
		return images[i].CreatedAt.After(images[j].CreatedAt)
	})

	return images, nil
}

func (gs *GalleryService) Image(galleryID int, filename string) (Image, error) {
	imagePath := filepath.Join(gs.galleryDir(galleryID), filename)

	_, err := os.Stat(imagePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Image{}, ErrNotFound
		}

		return Image{}, fmt.Errorf("getting image: %w", err)
	}

	return Image{
		GalleryID: galleryID,
		Path:      imagePath,
		Filename:  filename,
	}, nil
}

func (gs *GalleryService) CreateImage(galleryID int, filename string, contents io.ReadSeeker) error {
	err := checkContentType(contents, gs.imageContentTypes())
	if err != nil {
		return fmt.Errorf("creating image %v: %w", filename, err)
	}

	err = checkExtension(filename, gs.extensions())
	if err != nil {
		return fmt.Errorf("creating image %v: %w", filename, err)
	}

	galleryDir := gs.galleryDir(galleryID)

	err = os.MkdirAll(galleryDir, 0755)
	if err != nil {
		return fmt.Errorf("creating gallery-%d images directory: %w", galleryID, err)
	}

	imagePath := filepath.Join(galleryDir, filename)

	dst, err := os.Create(imagePath)
	if err != nil {
		return fmt.Errorf("creating image file: %w", err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, contents)
	if err != nil {
		return fmt.Errorf("copying contents to file: %w", err)
	}

	return nil
}

func (gs *GalleryService) DeleteImage(galleryID int, filename string) error {
	image, err := gs.Image(galleryID, filename)
	if err != nil {
		return fmt.Errorf("deleting image: %w", err)
	}

	err = os.Remove(image.Path)
	if err != nil {
		return fmt.Errorf("deleting image: %w", err)
	}

	return nil
}

func (gs *GalleryService) galleryDir(id int) string {
	imagesDir := gs.ImagesDir
	if imagesDir == "" {
		imagesDir = DefaultImagesDir
	}

	return filepath.Join(imagesDir, fmt.Sprintf("gallery-%d", id))
}

func (gs *GalleryService) extensions() []string {
	return []string{".jpg", ".jpeg", ".png", ".gif"}
}

func (gs *GalleryService) imageContentTypes() []string {
	return []string{"image/jpeg", "image/png", "image/gif"}
}

func (gs *GalleryService) sortableColumns() []string {
	return []string{"title", "created_at"}
}

func hasExtension(file string, extensions []string) bool {
	for _, ext := range extensions {
		file = strings.ToLower(file)
		ext = strings.ToLower(ext)

		if filepath.Ext(file) == ext {
			return true
		}
	}

	return false
}
