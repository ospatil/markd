package handler

import (
	"log"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ospatil/markd/components"
	"github.com/ospatil/markd/internal/model"
	"github.com/ospatil/markd/internal/store"
)

type Handler struct {
	store store.Store
}

func New(s store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) ListBookmarks(w http.ResponseWriter, r *http.Request) {
	folderID := r.URL.Query().Get("folder")
	var bookmarks []model.Bookmark
	var err error
	if folderID != "" {
		bookmarks, err = h.store.GetBookmarksByFolder(folderID)
	} else {
		bookmarks, err = h.store.ListBookmarks()
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if isHTMX(r) {
		render(w, r, components.BookmarkList(bookmarks))
		return
	}
	folders, _ := h.store.ListFolders()
	render(w, r, components.IndexPage(bookmarks, folders, folderID))
}

func (h *Handler) GetBookmark(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	b, err := h.store.GetBookmark(id)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	render(w, r, components.BookmarkItem(*b))
}

func (h *Handler) CreateBookmark(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	url := strings.TrimSpace(r.FormValue("url"))
	tagsRaw := strings.TrimSpace(r.FormValue("tags"))
	folderID := r.FormValue("folder_id")

	errors := map[string]string{}
	if title == "" {
		errors["title"] = "Title is required"
	}
	if url == "" {
		errors["url"] = "URL is required"
	}
	if len(errors) > 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)
		render(w, r, components.FormErrors(errors))
		return
	}

	b := &model.Bookmark{
		ID:       uuid.NewString(),
		Title:    title,
		URL:      url,
		Tags:     parseTags(tagsRaw),
		FolderID: folderID,
	}
	if err := h.store.CreateBookmark(b); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Trigger", "closeDialog")
	render(w, r, components.BookmarkItem(*b))
}

func (h *Handler) EditBookmark(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	b, err := h.store.GetBookmark(id)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	folders, _ := h.store.ListFolders()
	render(w, r, components.BookmarkEditForm(*b, folders, nil))
}

func (h *Handler) UpdateBookmark(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	url := strings.TrimSpace(r.FormValue("url"))
	tagsRaw := strings.TrimSpace(r.FormValue("tags"))
	folderID := r.FormValue("folder_id")

	errors := map[string]string{}
	if title == "" {
		errors["title"] = "Title is required"
	}
	if url == "" {
		errors["url"] = "URL is required"
	}

	b, err := h.store.GetBookmark(id)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	b.Title = title
	b.URL = url
	b.Tags = parseTags(tagsRaw)
	b.FolderID = folderID

	if len(errors) > 0 {
		folders, _ := h.store.ListFolders()
		w.WriteHeader(http.StatusUnprocessableEntity)
		render(w, r, components.BookmarkEditForm(*b, folders, errors))
		return
	}

	if err := h.store.UpdateBookmark(b); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	render(w, r, components.BookmarkItem(*b))
}

func (h *Handler) DeleteBookmark(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.DeleteBookmark(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) SearchBookmarks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	var bookmarks []model.Bookmark
	var err error
	if q == "" {
		bookmarks, err = h.store.ListBookmarks()
	} else {
		bookmarks, err = h.store.SearchBookmarks(q)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	render(w, r, components.BookmarkList(bookmarks))
}

func (h *Handler) FilterByTag(w http.ResponseWriter, r *http.Request) {
	tag := r.URL.Query().Get("tag")
	bookmarks, err := h.store.GetBookmarksByTag(tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	render(w, r, components.BookmarkListWithTag(bookmarks, tag))
}

// Folders

func (h *Handler) CreateFolder(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "Name is required", http.StatusUnprocessableEntity)
		return
	}
	f := &model.Folder{ID: uuid.NewString(), Name: name}
	if err := h.store.CreateFolder(f); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	folders, _ := h.store.ListFolders()
	render(w, r, components.FolderSidebar(folders, ""))
}

func (h *Handler) DeleteFolder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.DeleteFolder(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	folders, _ := h.store.ListFolders()
	render(w, r, components.FolderSidebar(folders, ""))
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func render(w http.ResponseWriter, r *http.Request, c templ.Component) {
	if err := c.Render(r.Context(), w); err != nil {
		log.Printf("render error: %v", err)
	}
}

func parseTags(raw string) []string {
	var tags []string
	for _, t := range strings.Split(raw, ",") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}
