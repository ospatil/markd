package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/ospatil/markd/internal/model"
)

func renderComponent(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestBookmarkItem(t *testing.T) {
	b := model.Bookmark{ID: "1", Title: "Go Blog", URL: "https://go.dev", Tags: []string{"go", "blog"}}
	html := renderComponent(t, BookmarkItem(b))

	checks := []string{"Go Blog", "https://go.dev", "badge-secondary", "go", "blog", `hx-delete`}
	for _, want := range checks {
		if !strings.Contains(html, want) {
			t.Errorf("BookmarkItem missing %q", want)
		}
	}
}

func TestBookmarkListEmpty(t *testing.T) {
	html := renderComponent(t, BookmarkList(nil))

	if !strings.Contains(html, "No bookmarks yet") {
		t.Error("empty list should show empty state")
	}
	if strings.Contains(html, "card") {
		t.Error("empty list should not contain cards")
	}
}

func TestBookmarkListWithItems(t *testing.T) {
	bookmarks := []model.Bookmark{
		{ID: "1", Title: "First", URL: "https://first.com"},
		{ID: "2", Title: "Second", URL: "https://second.com"},
	}
	html := renderComponent(t, BookmarkList(bookmarks))

	if strings.Contains(html, "No bookmarks yet") {
		t.Error("non-empty list should not show empty state")
	}
	if !strings.Contains(html, "First") || !strings.Contains(html, "Second") {
		t.Error("should contain both bookmarks")
	}
}

func TestBookmarkListWithTag(t *testing.T) {
	bookmarks := []model.Bookmark{{ID: "1", Title: "Go", URL: "https://go.dev"}}
	html := renderComponent(t, BookmarkListWithTag(bookmarks, "go"))

	if !strings.Contains(html, "Filtered by tag") {
		t.Error("should show active tag filter")
	}
	if !strings.Contains(html, "Clear") {
		t.Error("should show clear button")
	}
}
