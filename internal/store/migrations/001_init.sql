CREATE TABLE IF NOT EXISTS folders (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  parent_id TEXT REFERENCES folders(id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS bookmarks (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  url TEXT NOT NULL,
  folder_id TEXT REFERENCES folders(id) ON DELETE SET NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS tags (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS bookmark_tags (
  bookmark_id TEXT NOT NULL REFERENCES bookmarks(id) ON DELETE CASCADE,
  tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (bookmark_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_bookmarks_folder ON bookmarks(folder_id);
CREATE INDEX IF NOT EXISTS idx_bookmarks_created ON bookmarks(created_at);
CREATE INDEX IF NOT EXISTS idx_bookmark_tags_bookmark ON bookmark_tags(bookmark_id);
CREATE INDEX IF NOT EXISTS idx_bookmark_tags_tag ON bookmark_tags(tag_id);
