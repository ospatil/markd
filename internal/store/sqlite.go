package store

import (
	"database/sql"
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/ospatil/markd/internal/model"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type SQLiteStore struct {
	db *sql.DB
}

func New(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

func (s *SQLiteStore) migrate() error {
	data, err := migrations.ReadFile("migrations/001_init.sql")
	if err != nil {
		return err
	}
	_, err = s.db.Exec(string(data))
	return err
}

func (s *SQLiteStore) CreateBookmark(b *model.Bookmark) error {
	now := time.Now()
	b.CreatedAt = now
	b.UpdatedAt = now
	_, err := s.db.Exec(
		`INSERT INTO bookmarks (id, title, url, folder_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		b.ID, b.Title, b.URL, nullIfEmpty(b.FolderID), b.CreatedAt.Unix(), b.UpdatedAt.Unix(),
	)
	if err != nil {
		return err
	}
	return s.setTags(b.ID, b.Tags)
}

func (s *SQLiteStore) GetBookmark(id string) (*model.Bookmark, error) {
	b := &model.Bookmark{}
	var createdAt, updatedAt int64
	var folderID sql.NullString
	err := s.db.QueryRow(
		`SELECT id, title, url, folder_id, created_at, updated_at FROM bookmarks WHERE id = ?`, id,
	).Scan(&b.ID, &b.Title, &b.URL, &folderID, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	b.FolderID = folderID.String
	b.CreatedAt = time.Unix(createdAt, 0)
	b.UpdatedAt = time.Unix(updatedAt, 0)
	b.Tags, err = s.getTags(id)
	return b, err
}

func (s *SQLiteStore) ListBookmarks() ([]model.Bookmark, error) {
	rows, err := s.db.Query(`SELECT id, title, url, folder_id, created_at, updated_at FROM bookmarks ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return s.scanBookmarks(rows)
}

func (s *SQLiteStore) UpdateBookmark(b *model.Bookmark) error {
	b.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		`UPDATE bookmarks SET title = ?, url = ?, folder_id = ?, updated_at = ? WHERE id = ?`,
		b.Title, b.URL, nullIfEmpty(b.FolderID), b.UpdatedAt.Unix(), b.ID,
	)
	if err != nil {
		return err
	}
	return s.setTags(b.ID, b.Tags)
}

func (s *SQLiteStore) DeleteBookmark(id string) error {
	_, err := s.db.Exec(`DELETE FROM bookmarks WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) GetBookmarksByTag(tag string) ([]model.Bookmark, error) {
	rows, err := s.db.Query(
		`SELECT b.id, b.title, b.url, b.folder_id, b.created_at, b.updated_at
     FROM bookmarks b
     JOIN bookmark_tags bt ON b.id = bt.bookmark_id
     JOIN tags t ON bt.tag_id = t.id
     WHERE t.name = ?
     ORDER BY b.created_at DESC`, tag,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return s.scanBookmarks(rows)
}

func (s *SQLiteStore) GetBookmarksByFolder(folderID string) ([]model.Bookmark, error) {
	var rows *sql.Rows
	var err error
	if folderID == "" {
		rows, err = s.db.Query(
			`SELECT id, title, url, folder_id, created_at, updated_at FROM bookmarks WHERE folder_id IS NULL ORDER BY created_at DESC`)
	} else {
		rows, err = s.db.Query(
			`SELECT id, title, url, folder_id, created_at, updated_at FROM bookmarks WHERE folder_id = ? ORDER BY created_at DESC`, folderID)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return s.scanBookmarks(rows)
}

func (s *SQLiteStore) SearchBookmarks(query string) ([]model.Bookmark, error) {
	like := "%" + query + "%"
	rows, err := s.db.Query(
		`SELECT id, title, url, folder_id, created_at, updated_at FROM bookmarks
     WHERE title LIKE ? OR url LIKE ?
     ORDER BY created_at DESC`, like, like,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return s.scanBookmarks(rows)
}

func (s *SQLiteStore) CreateFolder(f *model.Folder) error {
	f.CreatedAt = time.Now()
	_, err := s.db.Exec(
		`INSERT INTO folders (id, name, parent_id, created_at) VALUES (?, ?, ?, ?)`,
		f.ID, f.Name, nullIfEmpty(f.ParentID), f.CreatedAt.Unix(),
	)
	return err
}

func (s *SQLiteStore) ListFolders() ([]model.Folder, error) {
	rows, err := s.db.Query(`SELECT id, name, parent_id, created_at FROM folders ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var folders []model.Folder
	for rows.Next() {
		var f model.Folder
		var createdAt int64
		var parentID sql.NullString
		if err := rows.Scan(&f.ID, &f.Name, &parentID, &createdAt); err != nil {
			return nil, err
		}
		f.ParentID = parentID.String
		f.CreatedAt = time.Unix(createdAt, 0)
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

func (s *SQLiteStore) DeleteFolder(id string) error {
	_, err := s.db.Exec(`DELETE FROM folders WHERE id = ?`, id)
	return err
}

// helpers

func (s *SQLiteStore) scanBookmarks(rows *sql.Rows) ([]model.Bookmark, error) {
	var bookmarks []model.Bookmark
	for rows.Next() {
		var b model.Bookmark
		var createdAt, updatedAt int64
		var folderID sql.NullString
		if err := rows.Scan(&b.ID, &b.Title, &b.URL, &folderID, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		b.FolderID = folderID.String
		b.CreatedAt = time.Unix(createdAt, 0)
		b.UpdatedAt = time.Unix(updatedAt, 0)
		bookmarks = append(bookmarks, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Fetch tags after closing the rows cursor to avoid nested query deadlock
	for i := range bookmarks {
		var err error
		bookmarks[i].Tags, err = s.getTags(bookmarks[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return bookmarks, nil
}

func (s *SQLiteStore) getTags(bookmarkID string) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT t.name FROM tags t JOIN bookmark_tags bt ON t.id = bt.tag_id WHERE bt.bookmark_id = ?`, bookmarkID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (s *SQLiteStore) setTags(bookmarkID string, tags []string) error {
	_, err := s.db.Exec(`DELETE FROM bookmark_tags WHERE bookmark_id = ?`, bookmarkID)
	if err != nil {
		return err
	}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		_, err := s.db.Exec(`INSERT OR IGNORE INTO tags (name) VALUES (?)`, tag)
		if err != nil {
			return err
		}
		_, err = s.db.Exec(
			`INSERT INTO bookmark_tags (bookmark_id, tag_id) SELECT ?, id FROM tags WHERE name = ?`,
			bookmarkID, tag,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
