# Insta Poster

A Go based service for scheduling and publishing posts to social platforms (Instagram, YouTube, etc.). It provides a REST API, background workers, and a CLI for managing accounts, media, analytics and webhook integrations.

## Table of Contents
- [Features](#features)
- [Architecture](#architecture)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Setup](#setup)
  - [Running Locally](#running-locally)
  - [Running Tests](#running-tests)
- [Configuration](#configuration)
- [API Overview](#api-overview)
- [Background Workers](#background-workers)
- [Deployment](#deployment)
- [Development](#development)
- [License](#license)

## Features
- **Multi‑tenant** support with per‑tenant context.
- **Account management** for Instagram and YouTube.
- **Media handling** (upload, storage on S3).
- **Scheduling** of posts with cron‑like semantics.
- **Analytics** collection and reporting.
- **Webhook** endpoints for delivery status and callbacks.
- **Asynchronous processing** using Asynq queues.
- **Dockerised** for easy deployment.

## Architecture
```
+-------------------+      +-------------------+      +-------------------+
|   API Server      | ---> |   Queue (Asynq)  | ---> |   Workers (pool) |
+-------------------+      +-------------------+      +-------------------+
        |                         |                         |
        v                         v                         v
   PostgreSQL                Redis (broker)            S3 (media storage)
```
- **cmd/** – entry points (`server/main.go`, `worker/main.go`, `migrate/main.go`).
- **internal/api** – HTTP handlers for accounts, media, posts, schedule, analytics and webhook.
- **internal/ffmpeg** – utilities for video processing.
- **internal/platform** – adapters for Instagram and YouTube APIs.
- **internal/queue** – Asynq queue wrapper.
- **internal/storage** – S3 abstraction.
- **internal/store** – PostgreSQL data layer.
- **internal/tenant** – request‑scoped tenant handling.
- **sdk/** – Go client library for external services to interact with the API.
- **migrations/** – SQL migration files.
- **deploy/** – Kubernetes manifests (ConfigMap, Ingress, Deployments, Services, StatefulSets).

## Getting Started
### Prerequisites
- Go 1.22 or later
- Docker & Docker‑Compose (optional for local dev)
- PostgreSQL 15
- Redis 7
- AWS credentials for S3 (or compatible storage)

### Setup
```bash
# Clone the repository
git clone git@github.com:mohitsharma-in/socialpublish.git
cd insta-poster

# Install Go dependencies
go mod tidy

# Run migrations (requires a running Postgres instance)
make migrate
```

### Running Locally
```bash
# Start dependencies with Docker Compose (Postgres, Redis, MinIO for S3)
docker compose up -d

# Run the API server
go run ./cmd/server

# In another terminal, start the worker pool
go run ./cmd/worker
```
The API will be available at `http://localhost:8080`.

### Running Tests
```bash
go test ./...   # runs unit and integration tests
```
The CI workflow (`.github/workflows/ci.yaml`) runs `gofmt`, `go vet`, `golangci-lint` and the test suite on every push.

## Configuration
Configuration is loaded from environment variables (see `config/config.go`). Common variables:
```
DB_DSN=postgres://user:pass@localhost:5432/insta_poster?sslmode=disable
REDIS_ADDR=localhost:6379
S3_ENDPOINT=https://s3.amazonaws.com
S3_BUCKET=insta-poster-media
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...
PORT=8080
```
You can also provide a `.env` file and load it with `github.com/joho/godotenv` (included in the project).

## API Overview
The API follows a RESTful design. Main endpoints (see `internal/api/handler`):
- `POST /v1/accounts` – create a new social account.
- `GET /v1/accounts/{id}` – fetch account details.
- `POST /v1/media` – upload media files.
- `POST /v1/posts` – schedule a post.
- `GET /v1/posts/{id}` – retrieve post status.
- `GET /v1/analytics` – fetch analytics data.
- `POST /v1/webhook` – webhook delivery endpoint.

OpenAPI/Swagger documentation can be generated with `swag init` (the project includes a `docs/` folder).

## Background Workers
Workers consume jobs from the Asynq Redis queue. Types of jobs:
- **PublishJob** – sends the post to the appropriate platform via the platform adapters.
- **AnalyticsJob** – pulls analytics data after a post is published.
- **TranscodeJob** – runs ffmpeg to prepare video assets.

Worker pool size and concurrency can be configured via `WORKER_POOL_SIZE`.

## Deployment
Kubernetes manifests are located under `deploy/k8s/`. Typical deployment steps:
```bash
# Apply namespace and config
kubectl apply -k deploy/k8s
```
The manifests include:
- ConfigMap for environment variables.
- Ingress for external access.
- Deployments for API server and worker pool.
- StatefulSets for PostgreSQL and Redis.
- Secrets for DB credentials and AWS keys.

Docker images are built with the `Dockerfile` and pushed to your registry:
```bash
docker build -t ghcr.io/yourorg/insta-poster:latest .
```

## Development
- **Linting**: `golangci-lint run ./...`
- **Formatting**: `gofmt -w .`
- **Running migrations**: `go run ./cmd/migrate`
- **Generating SDK**: The `sdk/` package provides a client; extend it as needed.
- **Adding new platform adapters**: Implement the `platform.Adapter` interface in `internal/platform/<platform>/adapter.go`.

## License
This project is licensed under the MIT License – see the `LICENSE` file for details.
