// Package app wires up the markd application router.
//
//	@title			markd API
//	@version		1.0
//	@description	Bookmark manager API
//	@BasePath		/
package app

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ospatil/markd/internal/api"
	"github.com/ospatil/markd/internal/handler"
	"github.com/ospatil/markd/internal/store"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/ospatil/markd/docs/swagger"
)

func NewRouter(s store.Store) *chi.Mux {
	h := handler.New(s)
	a := api.New(s)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// HTML routes
	r.Get("/", h.ListBookmarks)
	r.Post("/bookmarks", h.CreateBookmark)
	r.Get("/bookmarks/search", h.SearchBookmarks)
	r.Get("/bookmarks/filter", h.FilterByTag)
	r.Get("/bookmarks/{id}", h.GetBookmark)
	r.Get("/bookmarks/{id}/edit", h.EditBookmark)
	r.Put("/bookmarks/{id}", h.UpdateBookmark)
	r.Delete("/bookmarks/{id}", h.DeleteBookmark)

	r.Post("/folders", h.CreateFolder)
	r.Delete("/folders/{id}", h.DeleteFolder)

	// JSON API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/bookmarks", a.ListBookmarks)
		r.Post("/bookmarks", a.CreateBookmark)
		r.Get("/bookmarks/{id}", a.GetBookmark)
		r.Put("/bookmarks/{id}", a.UpdateBookmark)
		r.Delete("/bookmarks/{id}", a.DeleteBookmark)
		r.Get("/folders", a.ListFolders)
	})

	// Swagger UI (enable with ENABLE_API_DOCS=true)
	if os.Getenv("ENABLE_API_DOCS") == "true" {
		r.Get("/api/docs/*", httpSwagger.WrapHandler)
	}

	return r
}
