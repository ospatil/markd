package model

import "time"

type Bookmark struct {
	ID        string
	Title     string
	URL       string
	Tags      []string
	FolderID  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
