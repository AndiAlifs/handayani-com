# Angular SPA: build with Node, serve the static bundle with nginx.
FROM node:20-alpine AS build

WORKDIR /app
COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM nginx:alpine
# Angular 18 application builder emits to dist/<name>/browser.
COPY --from=build /app/dist/handayani-com/browser /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
