# NYAMPE attendance backend (Go). Multi-stage: build a static binary, run on alpine.
FROM golang:1.24-alpine AS build

WORKDIR /src

# Modules resolve via the Go proxy (HTTPS), so no git/apk packages are needed.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/server .

FROM alpine:3.20
WORKDIR /app
# CA certs copied from the build image (no network apk fetch needed); the static
# binary needs no other runtime packages.
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/server .

EXPOSE 8090
CMD ["./server"]
