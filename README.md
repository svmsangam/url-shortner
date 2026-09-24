# URL Shortener Service

This project provides a lightweight URL-shortening platform that turns long URLs into compact, shareable short codes while tracking per-device ownership and click analytics. It focuses on fast read performance, durable persistence, and simple browser-based access through a Go backend and React frontend.

## 1. Project Overview

The service solves the common problem of long, error-prone URLs by generating short public links that can be redirected efficiently and traced back to their source device. The system combines Redis for hot-path caching and ID sequencing, Cassandra for durable storage, and a Go REST API for request handling, while the React client provides a simple end-user interface.

## 2. High-Level Architecture & Data Flow

### Request Lifecycle and Data Flow

```text
+---------------------+      +----------------------------+      +---------------------------+
| Browser / Client    | ---> | Go HTTP API (Chi Router)   | ---> | Middleware Layer           |
| React App           |      | main.go + router.go        |      | CORS + Device Token       |
+---------------------+      +----------------------------+      +---------------------------+
                                         |
                                         v
                              +---------------------------+
                              | Request Handler           |
                              | shorten / redirect / info |
                              +---------------------------+
                                         |
                     +-------------------+--------------------+
                     |                                        |
                     v                                        v
      +---------------------------+          +------------------------------+
      | Redis Cache               |          | Cassandra / Persistent Store |
      | - hot URL lookup          |          | - url mappings               |
      | - unique ID generator    |          | - device-owned URLs          |
      | - short code cache TTL   |          | - click counters             |
      +---------------------------+          +------------------------------+
                     |
                     +---------------------> Redirects and analytics responses

``` 

### Architectural Patterns in Use

- REST-first API design: The backend exposes HTTP endpoints using Chi, and the frontend calls them through Axios. There is no gRPC or GraphQL layer in the current implementation.
- Cache-aside pattern: Redirect lookups and URL metadata fetches first read Redis; on a miss, the handler falls back to Cassandra and then refreshes the cache.
- Distributed ID generation: The `redisid.Generator` uses Redis `INCR` and `SETNX` semantics to create globally ordered numeric IDs, which are then encoded into Base62 short codes.
- Background async processing: Redirects trigger an asynchronous click increment, and the device URL secondary record is written in a goroutine to avoid blocking the main request path.
- Middleware guard model: Request middleware enforces CORS and device-token validation before business handlers execute, so device ownership and browser cross-origin behavior are centralized.
- Stateless client session model: The browser stores a device token in `localStorage` and attaches it via the `X-Device-Token` header, allowing a lightweight ownership model without login or account state.
- Concurrency-aware storage access: Cassandra sessions are configured with tuned connection pools and timeouts, while Redis uses a pool and connection timeouts to limit pressure under bursts of traffic.

## 3. Tech Stack & Infrastructure

### Languages and Runtime

- Go 1.26.4
- JavaScript (ES modules)
- React 19
- Vite 8 for the frontend build and dev server

### Backend Frameworks and Libraries

- `github.com/go-chi/chi/v5` for routing and middleware
- `github.com/redis/go-redis/v9` for Redis access and cache operations
- `github.com/gocql/gocql` for Cassandra connectivity
- `github.com/google/uuid` for UUID v4 device token generation
- `github.com/joho/godotenv` for loading environment variables

### Databases and Caching

- Redis: primary cache layer and unique ID counter source
- Cassandra: durable short-link storage, per-device mapping index, and click counter table

### Protocols and API Surface

- REST over HTTP
- Browser-to-API token propagation via custom `X-Device-Token` header
- Public redirect endpoints return HTTP 302/Found responses for short URLs

### Frontend Libraries and UI

- React DOM and React
- `axios` for API requests
- `lucide-react` for UI icons
- Tailwind CSS via Vite plugin for styling

### Infrastructure / Local Deployment

- Docker Compose for Cassandra and Redis containers
- `.env`-style configuration loaded from the environment or defaults
- `go.work` workspace setup for the Go module layout

## 4. Key Engineering & Performance Highlights

- Peak load handling: Redis-backed ID generation prevents contention around unique short-code creation and keeps the write path fast for hot traffic.
- Cache-first read path: Fresh redirect lookups are served from Redis before Cassandra, reducing latency for frequently accessed URLs.
- Durable persistence model: Cassandra stores the canonical `urls` mapping, the per-device index, and click counts, so user-visible data remains available even if cache entries expire.
- Async critical-path optimization: The secondary `device_urls` write is intentionally executed in a goroutine, so the request does not wait on a second storage write before returning a successful response.
- HTTPS-safe validation: Input validation rejects malformed URLs, local-only hosts, unsupported schemes, and forbidden IP-based targets before the mapping is stored.
- Device ownership semantics: Each browser client receives a UUID v4 token that identifies its created links without requiring authentication.
- Timeout and pool tuning: Redis and Cassandra both use bounded timeouts and tuned connection pools for predictable behavior under concurrency.
- Rate-limit and resilience posture: The system does not implement a central broker or queue; instead, it relies on lightweight middleware guards, safe failure handling, and application-level TTLs to keep the API responsive under stress.

## 5. Project Layout

```text
.
├── README.md
├── docker-compose.yml
├── go.work
├── backend/
│   ├── Makefile
│   ├── go.mod
│   ├── main.go                     # application bootstrap and service startup
│   ├── router.go                   # Chi route registration and middleware wiring
│   ├── schema.cql                  # Cassandra schema and table definitions
│   ├── base62/
│   │   └── base62.go               # Base62 encoder/decoder for short codes
│   ├── handler/
│   │   ├── shorten.go              # create short-link endpoint
│   │   ├── redirect.go             # public redirect handler
│   │   ├── get_url.go              # device-scoped and cache-backed URL lookup
│   │   ├── click_count.go          # analytics endpoint for click counts
│   │   └── *_test.go               # contract and security-focused tests
│   ├── middleware/
│   │   ├── device_token.go         # UUID token generation and request context
│   │   └── cors_setup.go           # browser cross-origin policy
│   ├── redisid/
│   │   ├── redisid.go              # Redis-backed ID generator
│   │   └── cache.go                # key/value URL cache helpers
│   ├── store/
│   │   └── cassandra.go            # repository layer for Cassandra writes/reads
│   └── utils/
│       └── uuid.go                 # UUID helper and shared generation logic
├── frontend/
│   ├── package.json
│   ├── vite.config.js
│   ├── index.html
│   ├── eslint.config.js
│   ├── public/
│   └── src/
│       ├── App.jsx                 # main browser UI workflow and state management
│       ├── main.jsx                # React entry point
│       ├── index.css               # styling layer
│       └── api/
│           └── client.js           # Axios client with device-token interceptors
└── schema.cql                      # repository-level schema reference / local init artifact
```

## 6. Getting Started

### Prerequisites

- Go 1.26.4 or compatible Go toolchain
- Node.js 18+ and npm
- Docker Desktop or Docker Engine with Compose support
- Access to localhost ports `8080`, `5173`, `9042`, and `6379`

### 1) Start supporting infrastructure

```bash
docker compose up -d
```

This launches:

- Cassandra on `localhost:9042`
- Redis on `localhost:6379`

### 2) Configure environment variables (optional)

The backend accepts defaults for common settings, but you can export or create an environment file before running:

```bash
export PORT=8080
export CASSANDRA_HOSTS=127.0.0.1
export CASSANDRA_PORT=9042
export CASSANDRA_KEYSPACE=urlshortener
export REDIS_ADDR=127.0.0.1:6379
```

### 3) Run the Go API

```bash
cd backend
go mod tidy
make run
```

or directly:

```bash
cd backend
go run .
```

The service will start on `http://localhost:8080` and expose routes under `/api`.

### 4) Run the frontend

```bash
cd frontend
npm install
npm run dev -- --host 0.0.0.0
```

Then open:

```text
http://localhost:5173
```

### 5) Verify local behavior

- Create a URL using the frontend form or direct API call to `POST /api/shorten`
- Open the generated short code in the browser to confirm redirect behavior
- Use the history and click count endpoints to validate device-scoped lookup and analytics

### Example API Call

```bash
curl -X POST http://localhost:8080/api/shorten \
  -H 'Content-Type: application/json' \
  -d '{"long_url":"https://example.com/very/long/path"}'
```

### Local Notes

- The API is intentionally small and service-oriented, not a distributed event bus system.
- Device tokens are generated per browser session and stored client-side to preserve a simple stateless API model.
- Redis cache entries are best-effort and expire with TTL; Cassandra remains the source of truth.

## Summary

This project is a compact, production-minded URL-shortener service built around a Go REST API, Redis caching, and Cassandra persistence. It demonstrates the practical combination of cache-aside reads, atomic ID generation, middleware-based access control, and a React front end in a single deployable application.
