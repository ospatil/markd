package api

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/ospatil/markd/internal/model"
	"github.com/ospatil/markd/internal/store"
)

func setupTestAPI(t *testing.T) (*Handler, store.Store, *chi.Mux) {
	t.Helper()
	dbPath := t.TempDir() + "/test.db"
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	h := New(s)
	r := chi.NewRouter()
	r.Get("/api/bookmarks", h.ListBookmarks)
	r.Post("/api/bookmarks", h.CreateBookmark)
	r.Get("/api/bookmarks/{id}", h.GetBookmark)
	r.Put("/api/bookmarks/{id}", h.UpdateBookmark)
	r.Delete("/api/bookmarks/{id}", h.DeleteBookmark)
	r.Get("/api/folders", h.ListFolders)
	return h, s, r
}

func TestAPIListBookmarks(t *testing.T) {
	_, s, r := setupTestAPI(t)
	_ = s.CreateBookmark(&model.Bookmark{ID: "1", Title: "Test", URL: "https://test.com", Tags: []string{"go"}})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/bookmarks", nil))

	if w.Code != 200 {
		t.Fatalf("got status %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("got content-type %q, want application/json", ct)
	}
	var bookmarks []model.Bookmark
	if err := json.NewDecoder(w.Body).Decode(&bookmarks); err != nil {
		t.Fatal(err)
	}
	if len(bookmarks) != 1 || bookmarks[0].Title != "Test" {
		t.Errorf("got %v, want [Test]", bookmarks)
	}
}

func TestAPICreateBookmark(t *testing.T) {
	_, _, r := setupTestAPI(t)

	body, _ := json.Marshal(createBookmarkRequest{Title: "New", URL: "https://new.com", Tags: []string{"test"}})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/api/bookmarks", bytes.NewReader(body)))

	if w.Code != 201 {
		t.Fatalf("got status %d, want 201", w.Code)
	}
	var b model.Bookmark
	if err := json.NewDecoder(w.Body).Decode(&b); err != nil {
		t.Fatal(err)
	}
	if b.Title != "New" || b.ID == "" {
		t.Errorf("got %+v", b)
	}
}

func TestAPICreateBookmarkValidation(t *testing.T) {
	_, _, r := setupTestAPI(t)

	body, _ := json.Marshal(createBookmarkRequest{Title: "", URL: ""})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/api/bookmarks", bytes.NewReader(body)))

	if w.Code != 422 {
		t.Fatalf("got status %d, want 422", w.Code)
	}
}

func TestAPIGetBookmark(t *testing.T) {
	_, s, r := setupTestAPI(t)
	_ = s.CreateBookmark(&model.Bookmark{ID: "42", Title: "Found", URL: "https://found.com"})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/bookmarks/42", nil))

	if w.Code != 200 {
		t.Fatalf("got status %d, want 200", w.Code)
	}
	var b model.Bookmark
	if err := json.NewDecoder(w.Body).Decode(&b); err != nil {
		t.Fatal(err)
	}
	if b.Title != "Found" {
		t.Errorf("got title %q, want Found", b.Title)
	}
}

func TestAPIGetBookmarkNotFound(t *testing.T) {
	_, _, r := setupTestAPI(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/bookmarks/nonexistent", nil))

	if w.Code != 404 {
		t.Fatalf("got status %d, want 404", w.Code)
	}
}

func TestAPIDeleteBookmark(t *testing.T) {
	_, s, r := setupTestAPI(t)
	_ = s.CreateBookmark(&model.Bookmark{ID: "42", Title: "Delete Me", URL: "https://del.com"})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/api/bookmarks/42", nil))

	if w.Code != 204 {
		t.Fatalf("got status %d, want 204", w.Code)
	}
}

func TestAPIListFolders(t *testing.T) {
	_, s, r := setupTestAPI(t)
	_ = s.CreateFolder(&model.Folder{ID: "f1", Name: "Work"})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/folders", nil))

	if w.Code != 200 {
		t.Fatalf("got status %d, want 200", w.Code)
	}
	var folders []model.Folder
	if err := json.NewDecoder(w.Body).Decode(&folders); err != nil {
		t.Fatal(err)
	}
	if len(folders) != 1 || folders[0].Name != "Work" {
		t.Errorf("got %v, want [Work]", folders)
	}
}
