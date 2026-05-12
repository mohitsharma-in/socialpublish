# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/migrate ./cmd/migrate

FROM alpine:3.19 AS base
RUN apk add --no-cache ca-certificates
WORKDIR /app

FROM base AS server
COPY --from=builder /out/server /app/server
ENTRYPOINT ["/app/server"]

FROM base AS worker
COPY --from=builder /out/worker /app/worker
ENTRYPOINT ["/app/worker"]

FROM base AS migrate
COPY --from=builder /out/migrate /app/migrate
ENTRYPOINT ["/app/migrate"]
