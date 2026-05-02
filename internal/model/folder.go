package model

import "time"

type Folder struct {
  ID        string
  Name      string
  ParentID  string
  CreatedAt time.Time
}
