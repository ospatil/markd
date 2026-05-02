# ---- Build stage ----
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache nodejs npm git

# Override GOPROXY for restricted networks: --build-arg GOPROXY=direct
ARG GOPROXY
RUN if [ -n "$GOPROXY" ]; then go env -w GOPROXY=$GOPROXY GONOSUMDB=*; fi
ENV GOTOOLCHAIN=auto

WORKDIR /app

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Cache npm modules
COPY package.json package-lock.json ./
RUN npm ci --ignore-scripts

# Copy source
COPY . .

# Generate templ
RUN go run github.com/a-h/templ/cmd/templ generate

# Build CSS and JS
RUN npx @tailwindcss/cli -i static/css/input.css -o static/css/app.css --minify
RUN npx esbuild static/js/app.js --bundle --outfile=static/js/bundle.js --minify

# Build static Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/markd ./cmd/server

# ---- Runtime stage: distroless (non-root, CA certs, ~2MB base) ----
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/markd /bin/markd
COPY --from=builder /app/static/css/app.css /static/css/app.css
COPY --from=builder /app/static/js/bundle.js /static/js/bundle.js
COPY --from=builder /app/static/vendor/ /static/vendor/

EXPOSE 3000
ENTRYPOINT ["/bin/markd"]
