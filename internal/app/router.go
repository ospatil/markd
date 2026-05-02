package app

import (
  "net/http"

  "github.com/go-chi/chi/v5"
  "github.com/go-chi/chi/v5/middleware"
  "github.com/ospatil/markd/internal/handler"
  "github.com/ospatil/markd/internal/store"
)

func NewRouter(s store.Store) *chi.Mux {
  h := handler.New(s)

  r := chi.NewRouter()
  r.Use(middleware.Logger)
  r.Use(middleware.Recoverer)

  r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

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

  return r
}
