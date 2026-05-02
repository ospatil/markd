package handler

import (
  "net/http/httptest"
  "net/url"
  "strings"
  "testing"

  "github.com/go-chi/chi/v5"
  "github.com/ospatil/markd/internal/model"
  "github.com/ospatil/markd/internal/store"
)

func setupTestHandler(t *testing.T) (*Handler, store.Store) {
  t.Helper()
  dbPath := t.TempDir() + "/test.db"
  s, err := store.New(dbPath)
  if err != nil {
    t.Fatal(err)
  }
  t.Cleanup(func() { s.Close() })
  return New(s), s
}

func TestListBookmarks_FullPage(t *testing.T) {
  h, s := setupTestHandler(t)
  s.CreateBookmark(&model.Bookmark{ID: "1", Title: "Test", URL: "https://test.com"})

  req := httptest.NewRequest("GET", "/", nil)
  w := httptest.NewRecorder()
  h.ListBookmarks(w, req)

  if w.Code != 200 {
    t.Errorf("got status %d, want 200", w.Code)
  }
  body := w.Body.String()
  if !strings.Contains(body, "<html") {
    t.Error("full page request should contain <html>")
  }
  if !strings.Contains(body, "Test") {
    t.Error("should contain bookmark title")
  }
}

func TestListBookmarks_HTMXPartial(t *testing.T) {
  h, s := setupTestHandler(t)
  s.CreateBookmark(&model.Bookmark{ID: "1", Title: "Test", URL: "https://test.com"})

  req := httptest.NewRequest("GET", "/", nil)
  req.Header.Set("HX-Request", "true")
  w := httptest.NewRecorder()
  h.ListBookmarks(w, req)

  body := w.Body.String()
  if strings.Contains(body, "<html") {
    t.Error("HTMX request should NOT contain <html>")
  }
  if !strings.Contains(body, "Test") {
    t.Error("should contain bookmark title")
  }
}

func TestCreateBookmark_Success(t *testing.T) {
  h, _ := setupTestHandler(t)

  form := url.Values{"title": {"Go Blog"}, "url": {"https://go.dev"}, "tags": {"go, blog"}}
  req := httptest.NewRequest("POST", "/bookmarks", strings.NewReader(form.Encode()))
  req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
  req.Header.Set("HX-Request", "true")
  w := httptest.NewRecorder()
  h.CreateBookmark(w, req)

  if w.Code != 200 {
    t.Errorf("got status %d, want 200", w.Code)
  }
  if w.Header().Get("HX-Trigger") != "closeDialog" {
    t.Error("should set HX-Trigger: closeDialog")
  }
  body := w.Body.String()
  if !strings.Contains(body, "Go Blog") {
    t.Error("response should contain bookmark title")
  }
  if !strings.Contains(body, "go") {
    t.Error("response should contain tag")
  }
}

func TestCreateBookmark_ValidationError(t *testing.T) {
  h, _ := setupTestHandler(t)

  form := url.Values{"title": {""}, "url": {""}}
  req := httptest.NewRequest("POST", "/bookmarks", strings.NewReader(form.Encode()))
  req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
  w := httptest.NewRecorder()
  h.CreateBookmark(w, req)

  if w.Code != 422 {
    t.Errorf("got status %d, want 422", w.Code)
  }
  body := w.Body.String()
  if !strings.Contains(body, "Title is required") {
    t.Error("should contain title error")
  }
}

func TestDeleteBookmark(t *testing.T) {
  h, s := setupTestHandler(t)
  s.CreateBookmark(&model.Bookmark{ID: "42", Title: "Delete Me", URL: "https://del.com"})

  r := chi.NewRouter()
  r.Delete("/bookmarks/{id}", h.DeleteBookmark)

  w := httptest.NewRecorder()
  r.ServeHTTP(w, httptest.NewRequest("DELETE", "/bookmarks/42", nil))

  if w.Code != 200 {
    t.Errorf("got status %d, want 200", w.Code)
  }
  if w.Body.Len() != 0 {
    t.Error("delete should return empty body")
  }
}

func TestSearchBookmarks(t *testing.T) {
  h, s := setupTestHandler(t)
  s.CreateBookmark(&model.Bookmark{ID: "1", Title: "Go Blog", URL: "https://go.dev"})
  s.CreateBookmark(&model.Bookmark{ID: "2", Title: "Rust News", URL: "https://rust-lang.org"})

  req := httptest.NewRequest("GET", "/bookmarks/search?q=go", nil)
  req.Header.Set("HX-Request", "true")
  w := httptest.NewRecorder()
  h.SearchBookmarks(w, req)

  body := w.Body.String()
  if !strings.Contains(body, "Go Blog") {
    t.Error("should contain matching bookmark")
  }
  if strings.Contains(body, "Rust News") {
    t.Error("should NOT contain non-matching bookmark")
  }
}

func TestFilterByTag(t *testing.T) {
  h, s := setupTestHandler(t)
  s.CreateBookmark(&model.Bookmark{ID: "1", Title: "Go", URL: "https://go.dev", Tags: []string{"go"}})
  s.CreateBookmark(&model.Bookmark{ID: "2", Title: "Rust", URL: "https://rust-lang.org", Tags: []string{"rust"}})

  req := httptest.NewRequest("GET", "/bookmarks/filter?tag=go", nil)
  w := httptest.NewRecorder()
  h.FilterByTag(w, req)

  body := w.Body.String()
  if !strings.Contains(body, "Go") {
    t.Error("should contain filtered bookmark")
  }
  if strings.Contains(body, "Rust") {
    t.Error("should NOT contain non-matching bookmark")
  }
  if !strings.Contains(body, "Filtered by tag") {
    t.Error("should show active tag filter")
  }
}
