export PATH := $(HOME)/go/bin:$(PATH)

TEMPL := $(HOME)/go/bin/templ

.PHONY: generate css js build build-lambda run dev dev-server dev-templ dev-css test test-v test-e2e lint fmt clean help \
       docker-build docker-build-direct docker-up docker-down docker-logs docs

setup: ## Install dependencies and required Go tools (templ, air, golangci-lint)
	npm install
	npx playwright install
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/air-verse/air@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

generate: ## Generate templ Go code
	$(TEMPL) generate

css: ## Build Tailwind CSS
	npx @tailwindcss/cli -i static/css/input.css -o static/css/app.css --minify

js: ## Bundle Stimulus controllers
	npx esbuild static/js/app.js --bundle --outfile=static/js/bundle.js --minify

build: generate css js ## Build the server binary
	go build -o bin/markd ./cmd/server

build-lambda: generate ## Build the Lambda binary
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/bootstrap ./cmd/lambda

run: build ## Build and run the server
	./bin/markd

dev: ## Run all watchers (server + templ + css)
	@echo "Starting dev mode — 4 processes..."
	@make -j4 dev-server dev-templ dev-css dev-js

dev-server: ## Live-reload Go server
	air

dev-templ: ## Watch and regenerate templ files
	$(TEMPL) generate --watch

dev-css: ## Watch and rebuild Tailwind CSS
	npx @tailwindcss/cli -i static/css/input.css -o static/css/app.css --watch

dev-js: ## Watch and rebuild JS bundle
	npx esbuild static/js/app.js --bundle --outfile=static/js/bundle.js --watch

test: ## Run Go tests
	go test ./...

test-v: ## Run Go tests (verbose)
	go test -v ./...

test-e2e: build ## Run Playwright E2E tests
	npx playwright test

lint: ## Run Go linter
	golangci-lint run

fmt: ## Format Go code
	go fmt ./...

clean: ## Remove build artifacts
	rm -rf bin/ tmp/ static/css/app.css static/js/bundle.js

docs: ## Generate documentation images from drawio files
	/Applications/draw.io.app/Contents/MacOS/draw.io --export --format png --scale 2 --border 20 --output docs/request-walkthrough.png docs/request-walkthrough.drawio

# Docker
docker-build: ## Build docker images
	docker compose build

docker-build-direct: ## Build docker images (bypass Go proxy)
	docker compose build --build-arg GOPROXY=direct

docker-up: ## Start docker containers
	docker compose up --build -d

docker-down: ## Stop docker containers
	docker compose down

docker-logs: ## Tail docker logs
	docker compose logs -f

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
