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
	defer func() { _ = db.Close() }()

	r := app.NewRouter(db)

	fmt.Printf("\nmarkd listening on http://localhost:%s\n", port)
	fmt.Printf("  App:     http://localhost:%s/\n", port)
	fmt.Printf("  API:     http://localhost:%s/api/bookmarks\n", port)
	if os.Getenv("ENABLE_API_DOCS") == "true" {
		fmt.Printf("  Swagger: http://localhost:%s/api/docs/\n", port)
	}
	fmt.Println()
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
