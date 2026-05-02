FROM golang:1.24-alpine AS assets

RUN apk add --no-cache nodejs npm

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --ignore-scripts

COPY static/ static/
COPY components/ components/
COPY internal/ internal/

# Build CSS and JS
RUN npx @tailwindcss/cli -i static/css/input.css -o static/css/app.css --minify
RUN npx esbuild static/js/app.js --bundle --outfile=static/js/bundle.js --minify

FROM nginx:mainline-alpine

RUN rm /etc/nginx/conf.d/default.conf
COPY docker/nginx/nginx.conf /etc/nginx/nginx.conf

COPY static/vendor/ /usr/share/nginx/html/static/vendor/
COPY --from=assets /app/static/css/app.css /usr/share/nginx/html/static/css/app.css
COPY --from=assets /app/static/js/bundle.js /usr/share/nginx/html/static/js/bundle.js
