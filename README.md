# SocialPublish

A production-grade Go SaaS backend for scheduling and publishing content to social media platforms. It exposes a RESTful API, background workers for async processing, a Go SDK for programmatic access, and Kubernetes manifests for deployment.

## Table of Contents

- [Key Features](#key-features)
- [Tech Stack](#tech-stack)
- [Architecture](#architecture)
  - [System Overview](#system-overview)
  - [Directory Structure](#directory-structure)
  - [Request Lifecycle](#request-lifecycle)
  - [Publish Pipeline](#publish-pipeline)
  - [Database Schema](#database-schema)
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
  - [Clone the Repository](#1-clone-the-repository)
  - [Install Dependencies](#2-install-dependencies)
  - [Environment Setup](#3-environment-setup)
  - [Database Setup](#4-database-setup)
  - [Run the API Server](#5-run-the-api-server)
  - [Run the Worker Pool](#6-run-the-worker-pool)
- [Configuration](#configuration)
- [API Reference](#api-reference)
- [SDK Usage](#sdk-usage)
- [Background Workers](#background-workers)
- [Testing](#testing)
- [Deployment](#deployment)
  - [Docker](#docker)
  - [Kubernetes](#kubernetes)
- [Development](#development)
- [Implementation Status](#implementation-status)
- [License](#license)

---

## Key Features

- **Multi-tenant** — Workspace-scoped data isolation with per-tenant rate limiting.
- **Multi-platform publishing** — Pluggable adapter architecture supporting Instagram (Reels, Stories, Carousel) and YouTube (Shorts, Videos).
- **Presigned media uploads** — Direct-to-S3 uploads with presigned URLs; no media proxied through the API server.
- **Async processing** — Redis-backed Asynq job queue for transcoding, publishing, analytics polling, token refresh, and webhook delivery.
- **FFmpeg transcoding** — Server-side video processing with platform-specific presets (bitrate, resolution, codec).
- **Encrypted token storage** — AES-256-GCM encrypted OAuth tokens with key versioning support.
- **SDK with fluent builder** — Go SDK with retry, error typing, webhook verification, and a fluent post builder.
- **Kubernetes-native** — Multi-stage Docker, Kustomize manifests, HPA, migration Job.
- **CI/CD** — GitHub Actions pipeline with linting (golangci-lint v2), static analysis, and tests.

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| **Language** | Go 1.23 |
| **HTTP Router** | [chi/v5](https://github.com/go-chi/chi) |
| **Database** | PostgreSQL 16 (pgx/v5 driver, connection pooling) |
| **Job Queue** | Redis + [Asynq](https://github.com/hibiken/asynq) |
| **Object Storage** | S3-compatible (AWS S3, Cloudflare R2, MinIO) |
| **Video Processing** | FFmpeg (exec-based runner) |
| **Migrations** | [golang-migrate/migrate](https://github.com/golang-migrate/migrate) |
| **Testing** | [testify](https://github.com/stretchr/testify) + httptest |
| **Linting** | [golangci-lint v2](https://golangci-lint.run/) |
| **Container** | Docker multi-stage (Alpine 3.19) |
| **Orchestration** | Kubernetes + Kustomize |

---

## Architecture

### System Overview

```
┌─────────────┐      ┌─────────────┐      ┌─────────────────┐
│  SDK Client  │─────▶│  API Server  │─────▶│   PostgreSQL     │
│  (sdk/)      │ HTTP │  (cmd/server)│      │   (9 tables)     │
└─────────────┘      └──────┬──────┘      └─────────────────┘
                            │
                     ┌──────▼──────┐      ┌─────────────────┐
                     │  Redis Queue │─────▶│  Worker Pool     │
                     │  (Asynq)    │      │  (cmd/worker)    │
                     └─────────────┘      └──────┬──────────┘
                                                  │
                                    ┌─────────────┼─────────────┐
                                    ▼             ▼             ▼
                              ┌──────────┐ ┌──────────┐ ┌──────────┐
                              │ Instagram│ │ YouTube  │ │ FFmpeg   │
                              │ Graph API│ │ Data API │ │ Runner   │
                              └──────────┘ └──────────┘ └──────────┘
                                                          │
                                                    ┌─────▼─────┐
                                                    │ S3 / R2   │
                                                    │ Storage   │
                                                    └───────────┘
```

### Directory Structure

```
socialpublish/
├── cmd/
│   ├── server/main.go          # API server entrypoint
│   ├── worker/main.go          # Worker pool entrypoint
│   └── migrate/main.go         # Migration runner (K8s Job)
│
├── sdk/                        # PUBLIC SDK — importable by customers
│   ├── client.go               # Root client with service accessors
│   ├── transport.go            # HTTP transport with auth + retry
│   ├── errors.go               # Structured error types (Code, Error, ValidationError)
│   ├── options.go              # Functional options pattern
│   ├── pagination.go           # Cursor-based pagination types
│   ├── webhook.go              # HMAC-SHA256 webhook verification + router
│   └── services/               # Resource-specific service packages
│       ├── account/            #   Account management
│       ├── analytics/          #   Analytics retrieval
│       ├── media/              #   Media upload + processing
│       ├── post/               #   Post lifecycle + fluent builder
│       └── schedule/           #   Scheduling + calendar
│
├── internal/                   # SERVER-SIDE — not importable by SDK consumers
│   ├── api/
│   │   ├── server.go           # chi router, middleware wiring, http.Server
│   │   ├── middleware/         # auth, tenant, ratelimit, requestid, logger
│   │   └── handler/            # Resource handlers (account, media, post, etc.)
│   ├── config/config.go        # Env-based configuration with validation
│   ├── ffmpeg/                 # FFmpeg runner + platform presets
│   ├── platform/               # PlatformAdapter interface + registry
│   │   ├── instagram/          #   Instagram Graph API adapter
│   │   └── youtube/            #   YouTube Data API adapter
│   ├── queue/                  # Queue interface + Asynq implementation
│   ├── storage/                # ObjectStorage interface + S3 implementation
│   ├── store/                  # Data access layer (interfaces + pgx implementations)
│   ├── tenant/                 # Request-scoped workspace context
│   ├── token/                  # Token store interface
│   └── worker/                 # Background job handlers + supervised pool
│
├── migrations/                 # SQL migrations (golang-migrate format)
│   ├── 000001_create_workspaces.{up,down}.sql
│   ├── 000002_create_api_keys.{up,down}.sql
│   ├── 000003_create_accounts.{up,down}.sql
│   ├── 000004_create_encrypted_tokens.{up,down}.sql
│   ├── 000005_create_media.{up,down}.sql
│   ├── 000006_create_posts.{up,down}.sql
│   ├── 000007_create_post_targets.{up,down}.sql
│   ├── 000008_create_webhook_endpoints.{up,down}.sql
│   └── 000009_create_webhook_deliveries.{up,down}.sql
│
├── deploy/
│   └── k8s/                    # Kubernetes manifests (Kustomize)
│       ├── namespace.yaml
│       ├── configmap.yaml
│       ├── secrets.yaml
│       ├── kustomization.yaml
│       ├── migrate-job.yaml
│       ├── ingress.yaml
│       ├── api/                # API deployment + service + HPA
│       ├── worker/             # Worker deployment
│       ├── postgres/           # Dev-only StatefulSet
│       └── redis/              # Dev-only StatefulSet
│
├── Dockerfile                  # Multi-stage, multi-target (server/worker/migrate)
├── Makefile                    # test, lint, build, docker-build, k8s-apply
├── .golangci.yml               # golangci-lint v2 configuration
└── .github/workflows/ci.yaml  # GitHub Actions CI pipeline
```

### Request Lifecycle

1. **SDK Client** sends HTTP request with `Bearer` API key.
2. **chi Middleware** processes: request ID → real IP → recovery → logging.
3. **Auth middleware** hashes the key with SHA-256 and looks up `api_keys` table.
4. **Tenant middleware** loads the workspace record and injects it into context.
5. **Rate limit middleware** checks per-workspace request budget.
6. **Handler** processes the request using store interfaces.
7. **Response** returned as JSON with `X-Request-ID` header.

### Publish Pipeline

1. Client calls `POST /v1/media/upload` → receives presigned S3 URL.
2. Client uploads media directly to S3.
3. Worker transcodes media via FFmpeg with platform-specific presets.
4. Client calls `POST /v1/posts` with media IDs and platform targets.
5. Worker dequeues `post:publish` job per target.
6. Worker decrypts OAuth token, calls platform adapter.
7. Platform adapter publishes via Graph API / YouTube Data API.
8. Worker records result and enqueues webhook delivery.

### Database Schema

9 tables organized around the multi-tenant model:

| Table | Purpose |
|-------|---------|
| `workspaces` | Tenant/organization with billing plan |
| `api_keys` | SHA-256 hashed API keys per workspace |
| `accounts` | Connected social accounts (Instagram, YouTube) |
| `encrypted_tokens` | AES-GCM encrypted OAuth tokens with key versioning |
| `media` | Uploaded media assets with processing status |
| `posts` | Post resources with media references |
| `post_targets` | Per-platform publish targets with status tracking |
| `webhook_endpoints` | Customer webhook configurations |
| `webhook_deliveries` | Webhook delivery audit log with retry tracking |

---

## Prerequisites

- **Go 1.23** or later
- **PostgreSQL 16** (or Docker)
- **Redis 7** (or Docker)
- **FFmpeg** (for video transcoding)
- **AWS credentials** or S3-compatible storage (MinIO for local dev)

---

## Getting Started

### 1. Clone the Repository

```bash
git clone git@github.com:mohitsharma-in/socialpublish.git
cd insta-poster
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Environment Setup

Create a `.env` file or export these environment variables:

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `DATABASE_URL` | ✅ | PostgreSQL connection string | `postgres://user:pass@localhost:5432/socialpublish?sslmode=disable` |
| `TOKEN_ENCRYPTION_KEY` | ✅ | Exactly 32 bytes for AES-256-GCM | `my-32-byte-secret-encryption-k!` |
| `LISTEN_ADDR` | ❌ | Server listen address | `0.0.0.0:8080` (default) |
| `REDIS_ADDR` | ❌ | Redis address | `127.0.0.1:6379` (default) |
| `FFMPEG_BIN` | ❌ | Path to FFmpeg binary | `ffmpeg` (default) |
| `S3_REGION` | ❌ | AWS S3 region | `us-east-1` |
| `S3_ENDPOINT` | ❌ | S3-compatible endpoint | `http://localhost:9000` (MinIO) |
| `S3_BUCKET` | ❌ | S3 bucket name | `socialpublish-media` |
| `S3_ACCESS_KEY_ID` | ❌ | S3 access key | — |
| `S3_SECRET_ACCESS_KEY` | ❌ | S3 secret key | — |

### 4. Database Setup

Start PostgreSQL (Docker example):

```bash
docker run --name postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=socialpublish -p 5432:5432 -d postgres:16-alpine
```

Start Redis:

```bash
docker run --name redis -p 6379:6379 -d redis:7-alpine
```

Run migrations:

```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/socialpublish?sslmode=disable"
export TOKEN_ENCRYPTION_KEY="my-32-byte-secret-encryption-k!"
go run ./cmd/migrate up
```

### 5. Run the API Server

```bash
go run ./cmd/server
```

The API will be available at `http://localhost:8080`. Verify with:

```bash
curl http://localhost:8080/health
# {"status":"ok"}

curl http://localhost:8080/readyz
# {"status":"ready"}
```

### 6. Run the Worker Pool

In a separate terminal:

```bash
go run ./cmd/worker
```

---

## Configuration

All configuration is loaded from environment variables via `internal/config/config.go`. The `Load()` function validates required fields at startup:

- `DATABASE_URL` — **required**, must be a valid PostgreSQL connection string.
- `TOKEN_ENCRYPTION_KEY` — **required**, must be exactly 32 bytes.

Helper functions `config.Duration(name, fallback)` and `config.Int(name, fallback)` are available for typed env parsing.

---

## API Reference

All `/v1/*` endpoints require a `Bearer` API key in the `Authorization` header.

### Health & Readiness

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Process liveness check |
| `GET` | `/readyz` | Dependency readiness (DB ping) |

### Accounts

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/accounts/connect` | Start OAuth flow for a social account |
| `GET` | `/v1/accounts` | List connected accounts |
| `GET` | `/v1/accounts/{accountID}` | Get account details |
| `DELETE` | `/v1/accounts/{accountID}` | Disconnect account |
| `GET` | `/v1/accounts/{accountID}/status` | Check token health |

### Media

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/media/upload` | Get presigned upload URL |
| `GET` | `/v1/media` | List media assets |
| `GET` | `/v1/media/{mediaID}` | Get media details |
| `DELETE` | `/v1/media/{mediaID}` | Delete media |
| `POST` | `/v1/media/{mediaID}/thumbnail` | Set custom thumbnail |

### Posts

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/posts` | Create a post |
| `GET` | `/v1/posts` | List posts |
| `GET` | `/v1/posts/{postID}` | Get post details |
| `PATCH` | `/v1/posts/{postID}` | Update post |
| `DELETE` | `/v1/posts/{postID}` | Delete post |
| `POST` | `/v1/posts/{postID}/publish` | Trigger immediate publish |
| `POST` | `/v1/posts/{postID}/cancel` | Cancel scheduled post |
| `POST` | `/v1/posts/{postID}/duplicate` | Duplicate post |

### Schedule

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/schedule` | Calendar view of scheduled posts |
| `GET` | `/v1/schedule/queue` | Processing queue status |
| `GET` | `/v1/schedule/next-available` | Next available publishing slot |

### Analytics

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/analytics/posts/{postID}` | Post engagement metrics |
| `GET` | `/v1/analytics/accounts/{accountID}` | Account-level metrics |
| `GET` | `/v1/analytics/summary` | Workspace analytics summary |

### Webhooks

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/webhooks` | Create webhook endpoint |
| `GET` | `/v1/webhooks` | List webhook endpoints |
| `DELETE` | `/v1/webhooks/{webhookID}` | Delete webhook endpoint |
| `POST` | `/v1/webhooks/{webhookID}/test` | Send test delivery |

### Rate Limiting

All `/v1/*` responses include rate limit headers:

| Header | Description |
|--------|-------------|
| `X-RateLimit-Limit` | Requests per minute for your plan |
| `X-RateLimit-Remaining` | Remaining requests in window |
| `X-RateLimit-Reset` | Unix timestamp when limit resets |
| `Retry-After` | Seconds to wait (on 429) |

Plan limits: Free (60/min), Starter (300/min), Pro (1000/min), Enterprise (5000/min).

---

## SDK Usage

The `sdk/` package is a Go client library for the SocialPublish API.

### Installation

```go
import socialpublish "github.com/mohitsharma-in/socialpublish/sdk"
```

### Creating a Client

```go
client, err := socialpublish.New(
    socialpublish.WithAPIKey("sp_live_your_api_key"),
    socialpublish.WithBaseURL("https://api.socialpublish.io"), // optional
)
```

Or set `SOCIALPUBLISH_API_KEY` in your environment:

```go
client, err := socialpublish.New() // reads from env
```

### Building a Post

```go
import "github.com/mohitsharma-in/socialpublish/sdk/services/post"

req, err := post.NewPost().
    WithMedia("med_abc123").
    ForInstagram("acc_ig_001").AsReel("Check this out! #golang").Done().
    ForYouTube("acc_yt_001").AsShort("Go Tips", "Quick Go tips").WithPrivacy(post.PrivacyPublic).Done().
    ScheduleAt(time.Now().Add(24 * time.Hour)).
    Build()
```

### Webhook Verification

```go
router := socialpublish.NewWebhookRouter("whsec_your_secret")
router.Handle("post.published", func(ctx context.Context, event socialpublish.WebhookEvent) error {
    // Process event.Data
    return nil
})
http.Handle("/webhook", router)
```

### Error Handling

```go
var apiErr *socialpublish.Error
if errors.As(err, &apiErr) {
    switch apiErr.Code {
    case socialpublish.CodeRateLimit:
        // Wait and retry
    case socialpublish.CodeNotFound:
        // Resource doesn't exist
    }
}
```

---

## Background Workers

Workers consume jobs from the Asynq Redis queue with priority-based processing:

| Task Type | Queue | Description |
|-----------|-------|-------------|
| `media:transcode` | default | FFmpeg transcoding with platform presets |
| `post:publish` | critical | Publish to platform via adapter → update status → enqueue webhook |
| `account:token_refresh` | default | Refresh expiring OAuth tokens |
| `analytics:poll` | low | Pull engagement metrics from platforms |
| `webhook:deliver` | default | Send webhook events to customer endpoints |

Queue priorities: `critical` (10), `default` (5), `low` (2).

Failed tasks log errors via `slog` and follow Asynq retry semantics. Permanent failures (bad payload, unknown platform) use `asynq.SkipRetry` to avoid retry loops.

---

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run a specific package
go test -v ./internal/api/...
go test -v ./sdk/...
go test -v ./internal/ffmpeg/...
go test -v ./internal/platform/...

# Run with race detection
go test -race ./...
```

### Test Structure

| Package | Tests | What's Covered |
|---------|-------|----------------|
| `internal/api` | 4 tests | Health endpoint, readiness (with/without check, pass/fail) |
| `internal/ffmpeg` | 2 tests | Transcode args construction, preset inventory |
| `internal/platform` | 1 test | Registry lookup, adapter validation (Instagram + YouTube) |
| `sdk` | 6 tests | Transport auth injection, retry on 5xx, API error parsing, webhook sign/verify/dispatch |

### Linting

```bash
# Run golangci-lint
golangci-lint run ./...

# Or via Makefile
make lint
```

---

## Deployment

### Docker

The Dockerfile uses multi-stage builds with separate targets:

```bash
# Build all images
make docker-build

# Or individually:
docker build --target server -t ghcr.io/mohitsharma-in/socialpublish:server-latest .
docker build --target worker -t ghcr.io/mohitsharma-in/socialpublish:worker-latest .
docker build --target migrate -t ghcr.io/mohitsharma-in/socialpublish:migrate-latest .
```

Run with Docker:

```bash
docker run -p 8080:8080 \
  -e DATABASE_URL="postgres://..." \
  -e TOKEN_ENCRYPTION_KEY="..." \
  -e REDIS_ADDR="redis:6379" \
  ghcr.io/mohitsharma-in/socialpublish:server-latest
```

### Kubernetes

Manifests are in `deploy/k8s/` using Kustomize:

```bash
# Apply all resources
kubectl apply -k deploy/k8s

# Or via Makefile
make k8s-apply
```

Included resources:

| Resource | Description |
|----------|-------------|
| `namespace.yaml` | `socialpublish` namespace |
| `configmap.yaml` | Non-sensitive config (listen addr, Redis, S3 settings) |
| `secrets.yaml` | Sensitive values (DB URL, S3 keys) |
| `api/deployment.yaml` | API server (2 replicas) |
| `api/service.yaml` | ClusterIP service on port 8080 |
| `api/hpa.yaml` | HPA: min 3 → max 20, CPU 65% / mem 80% |
| `worker/deployment.yaml` | Worker pool (1 replica) |
| `postgres/statefulset.yaml` | Dev-only PostgreSQL 16 with PVC |
| `redis/statefulset.yaml` | Dev-only Redis |
| `ingress.yaml` | Ingress for external access |
| `migrate-job.yaml` | One-off migration K8s Job |

> **Note:** For production, use managed PostgreSQL and Redis (e.g., RDS, ElastiCache, Upstash). The StatefulSets are for development only.

#### Running Migrations in K8s

```bash
# Set a unique job name per deploy (append git SHA)
kubectl create job socialpublish-migrate-$(git rev-parse --short HEAD) \
  --from=job/socialpublish-migrate \
  -n socialpublish
```

---

## Development

| Command | Description |
|---------|-------------|
| `make test` | Run all tests |
| `make lint` | Run golangci-lint |
| `make build` | Build server, worker, and migrate binaries (Linux AMD64) |
| `make docker-build` | Build all Docker images |
| `make k8s-apply` | Apply Kubernetes manifests |
| `make fmt` | Format all Go files |
| `make clean` | Remove build artifacts |
| `make all` | Run test → lint → build |

### Adding a New Platform Adapter

1. Create `internal/platform/<name>/` with `adapter.go`, `types.go`.
2. Implement the `platform.PlatformAdapter` interface:
   - `Platform() string`
   - `Publish(ctx, *PublishRequest) (*PublishResult, error)`
   - `RefreshToken(ctx, refreshToken) (*OAuthToken, error)`
   - `FetchMetrics(ctx, accessToken, platformPostID) (*Metrics, error)`
   - `ValidateTarget(target PostTarget) error`
3. Register the adapter in `cmd/worker/main.go`:
   ```go
   adapters := platform.NewRegistry(instagram.New(...), youtube.New(...), tiktok.New(...))
   ```
4. Add FFmpeg preset in `internal/ffmpeg/presets.go` if needed.

---

## Implementation Status

| Phase | Status | Description |
|-------|--------|-------------|
| 1. Scaffold | ✅ Done | Module, migrations, tenant, types, SDK errors |
| 2. SDK transport | ✅ Done | HTTP client with auth, retry, error handling |
| 3. Post builder | ✅ Done | Fluent builder with Instagram/YouTube sub-builders |
| 4. SDK services | ✅ Done | Account, media, post, schedule, analytics services |
| 5. Store layer | ✅ Done | pgx-backed implementations for all 8 stores |
| 6. Platform adapters | ⚠️ Partial | Validation + skeleton; live API flows not implemented |
| 7. FFmpeg | ✅ Done | Runner + 4 presets + tests |
| 8. Workers | ✅ Done | 5 job handlers registered and wired |
| 9. API handlers | ⚠️ Partial | Routes wired; handlers return `501 Not Implemented` |
| 10. Entrypoints | ✅ Done | `cmd/server`, `cmd/worker`, `cmd/migrate` |
| 11. Deployment | ⚠️ Partial | Kustomize + Docker; Helm chart not added |
| 12. CI | ⚠️ Partial | Lint + test; no Postgres/Redis service containers |

See `implementation_plan.md` for the detailed phase-by-phase plan.

---

## License

This project is licensed under the MIT License.
