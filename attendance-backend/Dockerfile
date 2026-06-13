# NYAMPE attendance backend (Go). Multi-stage: build a static binary, run on alpine.
FROM golang:1.24-alpine AS build

RUN apk add --no-cache git
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/server .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/server .

EXPOSE 8090
CMD ["./server"]
