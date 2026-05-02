package store

import (
  "testing"

  "github.com/ospatil/markd/internal/model"
)

func setupTestDB(t *testing.T) *SQLiteStore {
  t.Helper()
  dbPath := t.TempDir() + "/test.db"
  s, err := New(dbPath)
  if err != nil {
    t.Fatal(err)
  }
  t.Cleanup(func() { s.Close() })
  return s
}

func TestCreateAndGetBookmark(t *testing.T) {
  s := setupTestDB(t)

  b := &model.Bookmark{ID: "1", Title: "Go Blog", URL: "https://go.dev", Tags: []string{"go", "blog"}}
  if err := s.CreateBookmark(b); err != nil {
    t.Fatal(err)
  }

  got, err := s.GetBookmark("1")
  if err != nil {
    t.Fatal(err)
  }
  if got.Title != "Go Blog" {
    t.Errorf("got title %q, want %q", got.Title, "Go Blog")
  }
  if len(got.Tags) != 2 || got.Tags[0] != "go" {
    t.Errorf("got tags %v, want [go blog]", got.Tags)
  }
}

func TestListBookmarks(t *testing.T) {
  s := setupTestDB(t)

  if err := s.CreateBookmark(&model.Bookmark{ID: "1", Title: "First", URL: "https://first.com"}); err != nil {
    t.Fatal("create 1:", err)
  }
  if err := s.CreateBookmark(&model.Bookmark{ID: "2", Title: "Second", URL: "https://second.com"}); err != nil {
    t.Fatal("create 2:", err)
  }

  bookmarks, err := s.ListBookmarks()
  if err != nil {
    t.Fatal(err)
  }
  if len(bookmarks) != 2 {
    t.Errorf("got %d bookmarks, want 2", len(bookmarks))
  }
}

func TestUpdateBookmark(t *testing.T) {
  s := setupTestDB(t)

  s.CreateBookmark(&model.Bookmark{ID: "1", Title: "Old", URL: "https://old.com", Tags: []string{"old"}})

  b, _ := s.GetBookmark("1")
  b.Title = "New"
  b.Tags = []string{"new", "updated"}
  if err := s.UpdateBookmark(b); err != nil {
    t.Fatal(err)
  }

  got, _ := s.GetBookmark("1")
  if got.Title != "New" {
    t.Errorf("got title %q, want %q", got.Title, "New")
  }
  if len(got.Tags) != 2 {
    t.Errorf("got %d tags, want 2", len(got.Tags))
  }
}

func TestDeleteBookmark(t *testing.T) {
  s := setupTestDB(t)

  s.CreateBookmark(&model.Bookmark{ID: "1", Title: "Delete Me", URL: "https://del.com"})
  if err := s.DeleteBookmark("1"); err != nil {
    t.Fatal(err)
  }

  _, err := s.GetBookmark("1")
  if err == nil {
    t.Error("expected error getting deleted bookmark")
  }
}

func TestGetBookmarksByTag(t *testing.T) {
  s := setupTestDB(t)

  s.CreateBookmark(&model.Bookmark{ID: "1", Title: "Go", URL: "https://go.dev", Tags: []string{"go"}})
  s.CreateBookmark(&model.Bookmark{ID: "2", Title: "Rust", URL: "https://rust-lang.org", Tags: []string{"rust"}})
  s.CreateBookmark(&model.Bookmark{ID: "3", Title: "Go HTMX", URL: "https://htmx.org", Tags: []string{"go", "htmx"}})

  bookmarks, err := s.GetBookmarksByTag("go")
  if err != nil {
    t.Fatal(err)
  }
  if len(bookmarks) != 2 {
    t.Errorf("got %d bookmarks for tag 'go', want 2", len(bookmarks))
  }
}

func TestSearchBookmarks(t *testing.T) {
  s := setupTestDB(t)

  s.CreateBookmark(&model.Bookmark{ID: "1", Title: "Go Blog", URL: "https://go.dev/blog"})
  s.CreateBookmark(&model.Bookmark{ID: "2", Title: "Rust News", URL: "https://rust-lang.org"})

  bookmarks, err := s.SearchBookmarks("go")
  if err != nil {
    t.Fatal(err)
  }
  if len(bookmarks) != 1 {
    t.Errorf("got %d results for 'go', want 1", len(bookmarks))
  }
}

func TestFolderCRUD(t *testing.T) {
  s := setupTestDB(t)

  f := &model.Folder{ID: "f1", Name: "Work"}
  if err := s.CreateFolder(f); err != nil {
    t.Fatal(err)
  }

  folders, err := s.ListFolders()
  if err != nil {
    t.Fatal(err)
  }
  if len(folders) != 1 || folders[0].Name != "Work" {
    t.Errorf("got folders %v, want [Work]", folders)
  }

  if err := s.DeleteFolder("f1"); err != nil {
    t.Fatal(err)
  }
  folders, _ = s.ListFolders()
  if len(folders) != 0 {
    t.Errorf("got %d folders after delete, want 0", len(folders))
  }
}

func TestGetBookmarksByFolder(t *testing.T) {
  s := setupTestDB(t)

  s.CreateFolder(&model.Folder{ID: "f1", Name: "Work"})
  s.CreateBookmark(&model.Bookmark{ID: "1", Title: "Work Link", URL: "https://work.com", FolderID: "f1"})
  s.CreateBookmark(&model.Bookmark{ID: "2", Title: "Personal", URL: "https://personal.com"})

  bookmarks, err := s.GetBookmarksByFolder("f1")
  if err != nil {
    t.Fatal(err)
  }
  if len(bookmarks) != 1 || bookmarks[0].Title != "Work Link" {
    t.Errorf("got %v, want [Work Link]", bookmarks)
  }
}
