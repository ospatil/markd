package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ospatil/markd/internal/model"
	"github.com/ospatil/markd/internal/store"
)

type Handler struct {
	store store.Store
}

func New(s store.Store) *Handler {
	return &Handler{store: s}
}

type createBookmarkRequest struct {
	Title    string   `json:"title"`
	URL      string   `json:"url"`
	Tags     []string `json:"tags,omitempty"`
	FolderID string   `json:"folder_id,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// ListBookmarks godoc
//
//	@Summary	List all bookmarks
//	@Tags		bookmarks
//	@Produce	json
//	@Param		q		query	string	false	"Search query"
//	@Param		tag		query	string	false	"Filter by tag"
//	@Param		folder	query	string	false	"Filter by folder ID"
//	@Success	200		{array}	model.Bookmark
//	@Router		/api/bookmarks [get]
func (h *Handler) ListBookmarks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	tag := r.URL.Query().Get("tag")
	folder := r.URL.Query().Get("folder")

	var bookmarks []model.Bookmark
	var err error
	switch {
	case q != "":
		bookmarks, err = h.store.SearchBookmarks(q)
	case tag != "":
		bookmarks, err = h.store.GetBookmarksByTag(tag)
	case folder != "":
		bookmarks, err = h.store.GetBookmarksByFolder(folder)
	default:
		bookmarks, err = h.store.ListBookmarks()
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if bookmarks == nil {
		bookmarks = []model.Bookmark{}
	}
	writeJSON(w, http.StatusOK, bookmarks)
}

// GetBookmark godoc
//
//	@Summary	Get a bookmark by ID
//	@Tags		bookmarks
//	@Produce	json
//	@Param		id	path		string	true	"Bookmark ID"
//	@Success	200	{object}	model.Bookmark
//	@Failure	404	{object}	errorResponse
//	@Router		/api/bookmarks/{id} [get]
func (h *Handler) GetBookmark(w http.ResponseWriter, r *http.Request) {
	b, err := h.store.GetBookmark(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "bookmark not found")
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// CreateBookmark godoc
//
//	@Summary	Create a bookmark
//	@Tags		bookmarks
//	@Accept		json
//	@Produce	json
//	@Param		body	body		createBookmarkRequest	true	"Bookmark data"
//	@Success	201		{object}	model.Bookmark
//	@Failure	400		{object}	errorResponse
//	@Failure	422		{object}	errorResponse
//	@Router		/api/bookmarks [post]
func (h *Handler) CreateBookmark(w http.ResponseWriter, r *http.Request) {
	var req createBookmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Title == "" || req.URL == "" {
		writeError(w, http.StatusUnprocessableEntity, "title and url are required")
		return
	}
	b := &model.Bookmark{
		ID:       uuid.NewString(),
		Title:    strings.TrimSpace(req.Title),
		URL:      strings.TrimSpace(req.URL),
		Tags:     req.Tags,
		FolderID: req.FolderID,
	}
	if err := h.store.CreateBookmark(b); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

// UpdateBookmark godoc
//
//	@Summary	Update a bookmark
//	@Tags		bookmarks
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"Bookmark ID"
//	@Param		body	body		createBookmarkRequest	true	"Bookmark data"
//	@Success	200		{object}	model.Bookmark
//	@Failure	404		{object}	errorResponse
//	@Failure	422		{object}	errorResponse
//	@Router		/api/bookmarks/{id} [put]
func (h *Handler) UpdateBookmark(w http.ResponseWriter, r *http.Request) {
	b, err := h.store.GetBookmark(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "bookmark not found")
		return
	}
	var req createBookmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Title == "" || req.URL == "" {
		writeError(w, http.StatusUnprocessableEntity, "title and url are required")
		return
	}
	b.Title = strings.TrimSpace(req.Title)
	b.URL = strings.TrimSpace(req.URL)
	b.Tags = req.Tags
	b.FolderID = req.FolderID
	if err := h.store.UpdateBookmark(b); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// DeleteBookmark godoc
//
//	@Summary	Delete a bookmark
//	@Tags		bookmarks
//	@Param		id	path	string	true	"Bookmark ID"
//	@Success	204
//	@Failure	404	{object}	errorResponse
//	@Router		/api/bookmarks/{id} [delete]
func (h *Handler) DeleteBookmark(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteBookmark(chi.URLParam(r, "id")); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListFolders godoc
//
//	@Summary	List all folders
//	@Tags		folders
//	@Produce	json
//	@Success	200	{array}	model.Folder
//	@Router		/api/folders [get]
func (h *Handler) ListFolders(w http.ResponseWriter, r *http.Request) {
	folders, err := h.store.ListFolders()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if folders == nil {
		folders = []model.Folder{}
	}
	writeJSON(w, http.StatusOK, folders)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
