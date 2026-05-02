package main

import (
  "fmt"
  "log"
  "net/http"
  "os"

  "github.com/ospatil/markd/internal/app"
  "github.com/ospatil/markd/internal/store"
)

func main() {
  dbPath := envOr("DB_PATH", "markd.db")
  port := envOr("PORT", "3000")

  db, err := store.New(dbPath)
  if err != nil {
    log.Fatal(err)
  }
  defer db.Close()

  r := app.NewRouter(db)

  fmt.Printf("markd listening on :%s\n", port)
  log.Fatal(http.ListenAndServe(":"+port, r))
}

func envOr(key, fallback string) string {
  if v := os.Getenv(key); v != "" {
    return v
  }
  return fallback
}
