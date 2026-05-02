package store

import "github.com/ospatil/markd/internal/model"

type Store interface {
	// Bookmarks
	CreateBookmark(b *model.Bookmark) error
	GetBookmark(id string) (*model.Bookmark, error)
	ListBookmarks() ([]model.Bookmark, error)
	UpdateBookmark(b *model.Bookmark) error
	DeleteBookmark(id string) error
	GetBookmarksByTag(tag string) ([]model.Bookmark, error)
	GetBookmarksByFolder(folderID string) ([]model.Bookmark, error)
	SearchBookmarks(query string) ([]model.Bookmark, error)

	// Folders
	CreateFolder(f *model.Folder) error
	ListFolders() ([]model.Folder, error)
	DeleteFolder(id string) error

	Close() error
}
