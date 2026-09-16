# PulsePoll — Production Live Polling Platform

PulsePoll is a production-grade live polling web application built for the **HCL GUVI Full Stack Development Internship Selection Assignment**.

It enables creators to build interactive polls, share public links with audiences, and observe real-time vote updates instantaneously without manual page refreshes.

---

## 1. Technology Stack Overview

| Layer | Technology | Primary Responsibility |
| :--- | :--- | :--- |
| **Frontend** | React (Vite) | Responsive audience voting UI, animated live result charts, WebSocket stream consumer |
| **Backend** | Go (Gin) | High-throughput REST API, payload validation, JWT authentication, WebSocket broadcaster |
| **Database** | MongoDB 7.0 | Document persistence for users, polls, options, and vote audit logs |
| **Realtime** | Redis 7.0 | Atomic live vote counters (`HINCRBY`) and event distribution via Redis Pub/Sub channels |

---

## 2. Realtime Architecture Flow

```
[ Audience Client (React) ]
         │ (HTTP POST /api/polls/:id/vote)
         ▼
[ Go / Gin Backend ]
         ├── 1. Validates poll state & voting criteria
         ├── 2. Persists vote audit document to MongoDB
         ├── 3. Atomically increments counter in Redis (HINCRBY)
         └── 4. Publishes update event to Redis Pub/Sub channel (poll:{id}:updates)
                     │
                     ▼
             [ Redis Broker ]
                     │
                     ▼
         [ Go WebSocket Hub ] ─── (Broadcasts JSON frame) ───► [ Connected Clients ]
```

---

## 3. Project Structure

```
.
├── docker-compose.yml        # Development services: MongoDB (27017) & Redis (6380)
├── .env.example              # High-level configuration pointers
├── .gitignore                # Production ignore rules
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go       # Application entrypoint & graceful shutdown
│   ├── internal/
│   │   ├── config/           # Type-safe configuration loader with validation
│   │   ├── database/
│   │   │   ├── mongodb/      # MongoDB connection manager & lifecycle
│   │   │   └── redis/        # Redis connection manager & lifecycle
│   │   ├── health/           # Dependency health check handler
│   │   ├── middleware/       # Structured slog logger, panic recovery, CORS
│   │   ├── response/         # Standardized JSON response & error envelopes
│   │   ├── auth/             # (Phase 4)
│   │   ├── polls/            # (Phase 5)
│   │   ├── votes/            # (Phase 6)
│   │   └── websocket/        # (Phase 7)
│   ├── .env.example          # Backend configuration reference
│   ├── go.mod                # Go module specification
│   └── go.sum                # Cryptographic checksums for dependencies
└── frontend/                 # React application scaffolding (Vite)
```

---

## 4. Local Development Setup

### Prerequisites
* Go 1.23+
* Node.js 20+ & npm
* Docker & Docker Compose

### Step 1: Start Database & Cache Containers
```bash
docker compose up -d
```
* MongoDB is mapped to `localhost:27017`
* Redis is mapped to `localhost:6380` (avoids colliding with existing Redis instances on 6379)

### Step 2: Configure Environment
```bash
cp backend/.env.example backend/.env
```

### Step 3: Run Backend Tests
```bash
cd backend
go test -v ./...
```

### Step 4: Run Backend Server
```bash
cd backend
go run cmd/server/main.go
```

### Step 5: Verify Health Check
```bash
curl http://localhost:8080/health
```
Response:
```json
{
  "status": "ok",
  "services": {
    "mongodb": "ok",
    "redis": "ok"
  }
}
```

---

## 5. Authentication API & Security

PulsePoll implements secure, state-free authentication using **bcrypt (cost 10)** for password hashing and signed **JWT (HMAC-SHA256)** for request authorization.

### Endpoints

| Method | Endpoint | Auth Required | Description |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/auth/register` | No | Registers new account; enforces unique normalized email & minimum 8-char password |
| `POST` | `/api/auth/login` | No | Validates credentials; returns signed JWT and safe user profile DTO |
| `GET` | `/api/auth/me` | Yes (`Bearer <token>`) | Retrieves safe profile information for the authenticated user |

### Security Measures
1. **Password Hashing:** Passwords are hashed with bcrypt before storage; plaintext passwords and hashes are never returned to clients or logged.
2. **Email Normalization:** Emails are converted to lowercase and trimmed before uniqueness checks, preventing case-manipulation duplicates (`User@Pulse.io` vs `user@pulse.io`).
3. **Timing-Safe Generic Errors:** Login returns a generic `401 Unauthorized` (`Invalid email or password`) to eliminate account enumeration vulnerabilities.
4. **JWT Hardening:** Tokens include standard `iss`, `sub`, `iat`, and `exp` claims with strict signature validation against the HMAC algorithm, preventing `alg: none` exploits.

---

## 6. Poll Management API (Phase 5)

PulsePoll provides a domain model and REST API for poll creation, audience discovery, and ownership-guarded poll administration.

### Poll Data Model

Polls are persisted as documents in MongoDB under the `polls` collection. Each poll choice is generated with a stable, globally unique identifier (`bson.ObjectID.Hex()`) to ensure consistent referencing in downstream voting and realtime phases.

```json
{
  "id": "6aa9eac27795b69d0e40874d",
  "creator_id": "6aa9eac27795b69d0e408747",
  "question": "What is the best primary backend language for microservices?",
  "options": [
    { "id": "6aa9eac27795b69d0e408749", "text": "Go" },
    { "id": "6aa9eac27795b69d0e40874a", "text": "Rust" },
    { "id": "6aa9eac27795b69d0e40874b", "text": "TypeScript" },
    { "id": "6aa9eac27795b69d0e40874c", "text": "Python" }
  ],
  "status": "active",
  "created_at": "2026-09-16T01:02:58.775Z",
  "updated_at": "2026-09-16T01:02:58.775Z"
}
```

> **Note on Future Realtime Integration:** The poll document strictly models poll definitions and lifecycle states. Live vote counters are intentionally excluded from MongoDB poll documents and will be handled via Redis atomic operations in subsequent phases.

### Creator Ownership & Authorization

1. **Identity Extraction:** Creator identity is derived strictly from the verified JWT claim (`sub`), preventing client-side forgery of `creator_id`.
2. **Ownership Enforcement:** All mutating operations (`PATCH`, `DELETE`, `status transitions`) verify that `poll.creator_id == authenticated_user_id`. Unauthorized requests receive `403 Forbidden`.
3. **Scoped Listings:** `GET /api/my/polls` returns only polls belonging to the authenticated user, ordered newest first with clean pagination.

### Public Poll Access

Audiences access polls via `GET /api/polls/:id` without authentication. The returned payload is a sanitized DTO (`PublicPollResponse`) containing the question, options, lifecycle status, and expiration, omitting private creator IDs and internal database flags.

### Endpoints

| Method | Endpoint | Auth Required | Description |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/polls` | Yes (`Bearer <token>`) | Creates a new active poll with stable option IDs |
| `GET` | `/api/polls/:id` | No (Public) | Retrieves sanitized poll details for audience view |
| `GET` | `/api/my/polls` | Yes (`Bearer <token>`) | Returns paginated list of polls owned by the authenticated creator |
| `PATCH` | `/api/polls/:id` | Yes (Owner only) | Updates question or option texts while preserving stable IDs |
| `PATCH` | `/api/polls/:id/status` | Yes (Owner only) | Transitions lifecycle status (e.g. `{"status": "closed"}`) |
| `POST` | `/api/polls/:id/close` | Yes (Owner only) | Dedicated endpoint to idempotently close a poll |
| `DELETE` | `/api/polls/:id` | Yes (Owner only) | Soft-deletes poll; preserves referential integrity for future vote audits |

### Poll Lifecycle Status

* **`active`**: Poll is open and accepting audience interactions.
* **`closed`**: Poll is closed to further modifications and votes. Public audience can still view the question, options, and closed status.
* **Soft Deletion**: Deletion sets `is_deleted: true` and records `deleted_at`. Deleted polls immediately return `404 Not Found` for audience and creator queries, but retain document and option records in MongoDB to ensure future vote audit trails are never orphaned.

---

## 7. Voting & Realtime Counters (Phase 6)

PulsePoll features a concurrency-safe, dual-layer voting system combining durable audit records in MongoDB with high-throughput atomic live counters in Redis.

### Vote Data Model (MongoDB)

Audience votes are recorded as compact documents in the `votes` collection, avoiding denormalization of entire poll documents:

```json
{
  "_id": ObjectId("6aa9ed250cf658f70a932a88"),
  "poll_id": "6aa9ed250cf658f70a932a87",
  "option_id": "6aa9ed250cf658f70a932a84",
  "voter_id": "9be105f2ec74d293a2a62ba8f97706c6",
  "created_at": ISODate("2026-09-16T01:13:09.181Z")
}
```

### Redis Live Counters

Live vote counting is executed entirely in-memory using Redis Hashes for immediate reads and atomic increments:

* **Key Schema**: `poll:{pollID}:votes`
* **Fields**: `{optionID} -> {count}`
* **Atomic Mutation**: Increments are performed strictly via `HINCRBY poll:{pollID}:votes {optionID} 1` (preventing read-modify-write lost updates).

```
redis-cli HGETALL poll:6aa9ed250cf658f70a932a87:votes
1) "6aa9ed250cf658f70a932a84"
2) "1"
3) "6aa9ed250cf658f70a932a85"
4) "1"
5) "6aa9ed250cf658f70a932a86"
6) "0"
```

### Anonymous Voter Identification & Duplicate Prevention

1. **Session Resolution**: Audience participants are tracked using a cryptographically random 128-bit session token generated server-side.
2. **Persistence Mechanisms**:
   * Stored in an `HttpOnly`, `SameSite=Lax` cookie (`pulsepoll_voter_id`).
   * Supported via `X-Voter-ID` request/response headers for programmatic clients and tests.
3. **Database Concurrency Barrier**:
   * MongoDB enforces a unique compound index on `(poll_id, voter_id)`.
   * Concurrent requests from the same session race at the database layer; only one insert succeeds, while subsequent attempts are rejected with `409 Conflict` (`DUPLICATE_VOTE`).

### MongoDB & Redis Consistency Strategy

* **Persistence Order**: MongoDB durable persistence occurs **first**. If MongoDB rejects the vote (due to duplicate key, database validation, or failure), Redis is never touched.
* **Atomic Increment**: Redis `HINCRBY` is invoked only upon verified MongoDB insertion. If an intermittent network error occurs, exponential backoff retries are attempted.
* **Crash Recovery & Reconciliation**: Redis is an ephemeral live-count cache, while MongoDB remains the durable source of truth. If Redis restarts or drops keys, the service detects the missing hash and idempotently reconstructs the counters directly from MongoDB aggregation pipelines using `HSETNX`.

### Endpoints

| Method | Endpoint | Auth Required | Description |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/polls/:id/vote` | No (Audience) | Submits a vote; generates/validates voter session; persists to MongoDB; increments Redis live count |
| `GET` | `/api/polls/:id/results` | No (Audience) | Returns real-time vote tallies and computed percentages from Redis live counters |

### Live Results Envelope

```json
{
  "poll_id": "6aa9ed250cf658f70a932a87",
  "total_votes": 2,
  "results": [
    {
      "option_id": "6aa9ed250cf658f70a932a84",
      "text": "Redis",
      "votes": 1,
      "percentage": 50.0
    },
    {
      "option_id": "6aa9ed250cf658f70a932a85",
      "text": "Memcached",
      "votes": 1,
      "percentage": 50.0
    },
    {
      "option_id": "6aa9ed250cf658f70a932a86",
      "text": "Dragonfly",
      "votes": 0,
      "percentage": 0.0
    }
  ]
}
```

---

## 7. Realtime Broadcasting Architecture (Phase 7)

### Component Responsibilities

| Component | Responsibility |
| :--- | :--- |
| **MongoDB** | Durable source of truth for users, polls, options, and immutable vote audit logs. Poll documents store **zero** mutable vote counter fields. |
| **Redis Hash (`poll:{id}:votes`)** | Authoritative live counter aggregation. Modified strictly via atomic `HINCRBY`. Read by results queries and snapshot generators. |
| **Redis Pub/Sub (`poll:{id}:updates`)** | Realtime event transport. Distributes lightweight JSON events across Go server instances. Transient, memory-efficient broadcast channel. |
| **Go WebSocket Gateway (`PollHub`)** | Concurrency-safe client registry. Pre-validates polls before upgrade. Dynamically provisions Redis subscriptions on first client join; cancels them on last client exit. Dispatches ping/pong keepalives and isolates slow clients via buffered channels. |
| **React Frontend (Phase 8)** | Consumer of WebSocket streams. Merges initial snapshot and incremental live vote updates into UI state without page reload. |

### End-to-End Realtime Flow

```
Audience Member Votes (POST /api/polls/:id/vote)
  │
  ├── 1. Validate poll (active, not expired, valid option)
  ├── 2. Persistent audit write to MongoDB (unique compound index prevents duplicates)
  ├── 3. Atomic counter increment in Redis (HINCRBY poll:{id}:votes {opt} 1)
  └── 4. Publish JSON event to Redis Pub/Sub (PUBLISH poll:{id}:updates)
             │
             ▼
     Redis Pub/Sub Subscriber (Go backend)
             │
             ▼
     Go PollHub (poll-scoped broadcast channel)
             │
             ▼
     All Connected WebSockets (GET /api/polls/:id/ws)
             │
             ▼
     Browser / React Client updates live bars & counts instantaneously
```

### WebSocket Endpoint: `GET /api/polls/:id/ws`

* **Access**: Public audience endpoint (no creator auth required).
* **Pre-Validation**: Checks hex ObjectID format, queries MongoDB for poll presence, and rejects soft-deleted polls with HTTP 404 before upgrading the HTTP connection.
* **Closed Polls**: Audience members can connect to closed polls to view final results. When an active poll is closed by its creator, a terminal `poll_closed` event is broadcast to all connected viewers while preserving the connection.

### WebSocket Protocol Message Contracts

#### 1. Initial State: `results_snapshot`

Sent immediately upon WebSocket connection to eliminate race conditions between connection establishment and in-flight votes:

```json
{
  "type": "results_snapshot",
  "poll_id": "6aa9ed250cf658f70a932a87",
  "options": [
    {
      "option_id": "6aa9ed250cf658f70a932a84",
      "text": "Redis",
      "count": 12,
      "votes": 12,
      "percentage": 60.0
    },
    {
      "option_id": "6aa9ed250cf658f70a932a85",
      "text": "Memcached",
      "count": 8,
      "votes": 8,
      "percentage": 40.0
    }
  ],
  "total_votes": 20
}
```

#### 2. Live Vote Update: `vote_update`

Broadcast whenever an audience member casts a valid vote:

```json
{
  "type": "vote_update",
  "poll_id": "6aa9ed250cf658f70a932a87",
  "option_id": "6aa9ed250cf658f70a932a84",
  "count": 13,
  "total_votes": 21,
  "timestamp": "2026-09-16T01:38:25.619Z"
}
```

*Strict Privacy Enforcement*: Realtime update events never include voter identifiers, session cookies, JWTs, or creator credentials.

#### 3. Poll Lifecycle: `poll_closed`

Broadcast when the creator closes the poll:

```json
{
  "type": "poll_closed",
  "poll_id": "6aa9ed250cf658f70a932a87",
  "timestamp": "2026-09-16T01:38:38.572Z"
}
```

### Initial State & Race Elimination Strategy

To prevent the classic race condition where an audience member connects and misses in-flight votes that occurred before subscription:
1. When a client connects, the Go backend registers the client with the `PollHub` and **ensures the Redis Pub/Sub subscription is active first** (awaiting subscription acknowledgement from Redis).
2. The server queries the Redis live counter hash for the current tallies (`voteService.GetResults`).
3. The server immediately enqueues the `results_snapshot` into the client's outbound buffer.
4. Any vote in flight during connection either reflects in the snapshot read or arrives immediately as a subsequent `vote_update` event in the client's FIFO channel.

### Dynamic Subscription Lifecycle

To prevent unbounded resource consumption from thousands of idle polls:
* **First Client Connects**: The `HubManager` dynamically allocates a `PollHub` and initiates the Redis Pub/Sub subscription for `poll:{pollID}:updates`.
* **Last Client Disconnects**: The `PollHub` terminates its event loop, closes its Redis Pub/Sub subscription cleanly, and unregisters itself from `HubManager`.

### Failure Semantics & Resilience

* **Execution Order**: MongoDB persistence **always** precedes Redis mutation, which **always** precedes Redis Pub/Sub publishing.
* **Failed Votes Never Broadcast**: If a vote is rejected (e.g. duplicate voter session, closed poll, database error), Redis `HINCRBY` and Pub/Sub are never invoked.
* **Best-Effort Delivery**: If Redis Pub/Sub publishing encounters a network glitch after `HINCRBY` succeeds, the vote is **not** rolled back. The MongoDB vote record and Redis count remain durable and authoritative. The error is logged, and clients reconcile upon reconnection or results refresh.

---

## 9. Phase 8 — React Frontend & Complete User Experience

PulsePoll provides a modern, responsive single-page web interface built with **React 19**, **Vite**, and vanilla CSS design tokens. It integrates directly with the Go/Gin backend and Redis Pub/Sub WebSocket streaming infrastructure without any mock data or polling intervals.

### Frontend Architecture & Structure

```
frontend/
├── src/
│   ├── api/
│   │   ├── client.js               # Centralized fetch wrapper with JWT bearer injection and error normalization
│   │   ├── auth.js                 # Authentication API calls (register, login, getMe)
│   │   ├── polls.js                # Poll management API calls (create, getPublic, getMy, update, close, delete)
│   │   └── votes.js                # Voting and results API calls (castVote, getResults)
│   ├── components/
│   │   ├── Alert.jsx               # Accessible dismissible notification banner
│   │   ├── Footer.jsx              # Application branding and technology stack footer
│   │   ├── LiveBadge.jsx           # Pulsing connection status badge (Connected, Connecting, Reconnecting, Offline)
│   │   ├── Navbar.jsx              # Brand header with dynamic session state and navigation
│   │   ├── PollCard.jsx            # Creator poll dashboard card with one-click copy link & actions
│   │   ├── PollResults.jsx         # Live results display with animated progress bars & leading badges
│   │   └── ProtectedRoute.jsx      # Authentication route guard with session restoration
│   ├── context/
│   │   └── AuthContext.jsx         # React context provider with /api/auth/me startup verification
│   ├── hooks/
│   │   └── usePollWebSocket.js     # Robust WebSocket hook with bounded exponential backoff & cleanup
│   ├── pages/
│   │   ├── HomePage.jsx            # Architectural landing page with feature breakdowns
│   │   ├── LoginPage.jsx           # Creator sign-in form with client validation and redirect
│   │   ├── RegisterPage.jsx        # Creator account registration with auto-login
│   │   ├── DashboardPage.jsx       # Authenticated poll dashboard with metrics, filters, and cards
│   │   ├── CreatePollPage.jsx      # Dynamic poll builder (2–10 options) with copyable shareable link
│   │   ├── PublicPollPage.jsx      # Public audience voting screen & real-time WebSocket live results
│   │   └── ManagePollPage.jsx      # Creator poll management console (close, delete, update question)
│   ├── test/
│   │   ├── setup.js                # Vitest / testing-library jest-dom configuration
│   │   ├── auth.test.jsx           # Unit tests for login, registration, and route guarding
│   │   ├── poll_form.test.jsx      # Unit tests for poll form validation & dynamic options
│   │   ├── poll_results.test.jsx   # Unit tests for results rendering, percentages, and progress bars
│   │   └── public_poll_websocket.test.jsx # Integration tests for voting, duplicates, and WebSocket events
│   ├── App.jsx                     # Route definitions (public and protected)
│   ├── index.css                   # Polished dark-mode design system with smooth animations
│   └── main.jsx                    # React entrypoint
├── .env.example                    # Frontend environment variable reference
├── package.json                    # Dependencies and scripts
└── vite.config.js                  # Vite bundler and Vitest test runner configuration
```

### Route Table

| Route | Access | Purpose |
| :--- | :--- | :--- |
| `/` | Public | Landing page with architecture diagram and get-started CTAs |
| `/login` | Public | Creator sign-in page with validation and session restore |
| `/register` | Public | Creator account registration with auto-login into dashboard |
| `/polls/:id` | **Public** | **Public poll participation**: radio voting + live animated results (No auth required) |
| `/dashboard` | Protected | Creator dashboard listing all created polls, metrics, and filters |
| `/polls/create` | Protected | Poll creation form (2 to 10 options, optional deadline) |
| `/polls/:id/manage` | Protected | Creator management console to close, delete, or edit poll |

### Environment Variables

Defined in `frontend/.env`:
```bash
VITE_API_BASE_URL=http://localhost:8080
VITE_API_URL=http://localhost:8080/api
VITE_WS_URL=ws://localhost:8080/api
```

### How to Run Frontend

```bash
# Navigate to frontend directory
cd frontend

# Install dependencies
npm install

# Start local development server (http://localhost:5173)
npm run dev

# Run unit and component test suite
npm run test

# Run linter
npm run lint

# Build production bundle
npm run build
```

### Public Poll Flow & Realtime Results

1. **Unauthenticated Public Access**: Audience members navigate to `/polls/:id`. No signup or login is required.
2. **Initial State Reconciliation**: On component mount, the frontend fetches `GET /api/polls/:id` and `GET /api/polls/:id/results` via REST to display initial question and baseline tallies.
3. **Realtime WebSocket Connection**: The custom hook `usePollWebSocket` opens `ws://localhost:8080/api/polls/:id/ws`:
   * Automatically replaces state upon receiving `results_snapshot`.
   * Directly updates individual option counts and total votes upon receiving `vote_update` frames.
   * Smoothly animates progress bars via CSS transitions (`transition: width 0.45s cubic-bezier(0.4, 0, 0.2, 1)`).
   * **Zero Polling Loops**: No `setInterval` or repeated REST polling is ever used.
4. **Resilient Reconnection**:
   * If the WebSocket connection drops, status transitions to `reconnecting` and an exponential backoff reconnect is attempted (1s, 2s, 4s, bounded at 10s).
   * Upon reconnection, the fresh `results_snapshot` reconciles any votes cast while offline.
   * Safe for React 19 / StrictMode dual mounting (aborts stale sockets and cleans up event handlers).
5. **Voting UX & Duplicate Protection**:
   * Audience members select a radio button and submit their vote.
   * Browser stores anonymous voter session tokens (`localStorage` and HTTP cookie).
   * Duplicate submissions are rejected with HTTP 409 `DUPLICATE_VOTE` and display a friendly message: *"You've already voted in this poll."*
   * When a creator closes a poll, a `poll_closed` frame is broadcast over WebSockets, instantly locking further voting across all screens.

---

## 11. Production Deployment Architecture

PulsePoll provides complete production-grade deployment configurations supporting continuous integration, containerized packaging, and managed cloud environments.

```
                    [ React SPA (Vite / Nginx) ]
                                 │
                            HTTPS / WSS
                                 │
                                 ▼
                     [ Go / Gin API Server ]
                     (Docker: alpine:3.20)
                       /               \
                      /                 \
                     ▼                   ▼
           [ MongoDB Database ]   [ Redis In-Memory ]
           (Durable Persistence)  ├── Live Counters (Hash)
                                  └── Realtime Broker (Pub/Sub)
                                           │
                                           ▼
                                [ Go WebSocket Broadcaster ]
                                           │ (wss://)
                                           ▼
                                [ Active Browser Clients ]
```

### Component Roles in Production

* **React Frontend**: Built as an optimized static SPA bundle and served via Nginx (or cloud CDN/static host). Configured with single-page application fallback (`try_files $uri $uri/ /index.html;` / rewrite `/*` to `/index.html`) ensuring direct navigation and browser refreshes on routes like `/polls/:id` or `/dashboard` never produce 404s.
* **Go/Gin Backend**: Packaged in a minimal, multi-stage Docker container (`alpine:3.20`) running as an unprivileged user (`pulsepoll`). Provides REST API endpoints, payload validation, JWT authentication, and WebSocket connection upgrades.
* **MongoDB**: Serves as the authoritative, durable database storing users, polls, options, and vote audit records. Configured with unique compound indexes (`poll_id` + `voter_id`) to enforce idempotency at the database level.
* **Redis**: Decoupled into two distinct roles:
  1. **Atomic Counters (`HINCRBY`)**: Authoritative in-memory cache holding real-time vote totals for instant retrieval without querying MongoDB.
  2. **Pub/Sub Message Bus**: Distributes vote events and lifecycle notifications across server instances to connected WebSocket hubs.

### HTTPS / WSS Protocol Resolution

In production, frontend clients communicate exclusively over secure protocols:
* **HTTP REST**: Automatically uses `https://<backend-domain>/api/...`
* **WebSockets**: Automatically uses `wss://<backend-domain>/api/polls/:id/ws`

The frontend client dynamically derives the WebSocket scheme from the base API URL:
```javascript
// Automatically derives wss:// under https://, or respects explicit VITE_WS_URL
export const WS_BASE_URL =
  (import.meta.env.VITE_WS_URL && import.meta.env.VITE_WS_URL.trim()) ||
  API_BASE_URL.replace(/^http(s)?:\/\//i, (match, s) => (s ? 'wss://' : 'ws://'));
```

### Cross-Origin Security & Anonymous Voter Session Cookies

When the frontend and backend are hosted on separate domains (e.g. `pulsepoll.onrender.com` and `pulsepoll-backend.onrender.com`):
* **CORS**: The backend configures `cors.Config` with explicit origins from `ALLOWED_ORIGINS`, `AllowCredentials: true`, and exposes the `X-Voter-ID` header.
* **Cookie Flags**: The anonymous voter session cookie (`pulsepoll_voter_id`) dynamically detects HTTPS (via direct TLS or `X-Forwarded-Proto: https`):
  * **Production (HTTPS)**: `SameSite=None`, `Secure=true`, `HttpOnly=true`, `Path=/`
  * **Development (HTTP)**: `SameSite=Lax`, `Secure=false`, `HttpOnly=true`, `Path=/`

### Deployment Options

#### Option A: Managed Cloud (Render + MongoDB Atlas + Managed Redis)

PulsePoll includes a complete blueprint specification in `render.yaml`:
1. Push repository to your GitHub account.
2. In [MongoDB Atlas](https://www.mongodb.com/atlas), create a free M0 cluster and acquire the connection string:
   ```
   mongodb+srv://<username>:<password>@cluster0.mongodb.net/?retryWrites=true&w=majority
   ```
3. In [Render](https://render.com), create a new **Blueprint** instance pointing to your repository.
4. Supply `MONGODB_URI` and `ALLOWED_ORIGINS` in the prompt; Render automatically provisions:
   * **Backend**: Docker Web Service running the Go application with health checks on `/health`.
   * **Frontend**: Static Site with automatic SPA rewrite (`/*` -> `/index.html`) and injected `VITE_API_BASE_URL`.
   * **Redis**: Managed Redis instance with connection string linked directly to backend `REDIS_URL`.

#### Option B: Self-Hosted Production Stack (Docker Compose)

PulsePoll provides a fully isolated, production-tuned Docker Compose stack in `docker-compose.prod.yml`:
```bash
# Set your production JWT secret and origins
export JWT_SECRET=$(openssl rand -base64 32)
export ALLOWED_ORIGINS=https://your-domain.com

# Launch production stack
docker compose -f docker-compose.prod.yml up -d --build
```
This deploys:
* `pulsepoll-mongo-prod`: MongoDB 7.0 container with persistent data volume `pulsepoll_mongodb_prod_data`.
* `pulsepoll-redis-prod`: Redis 7.0 Alpine container with persistent AOF storage `pulsepoll_redis_prod_data`.
* `pulsepoll-backend-prod`: Optimized static Go binary in Alpine image.
* `pulsepoll-frontend-prod`: Nginx Alpine serving SPA with WebSocket reverse proxy on port 80/443.

### Production Health Verification

The backend exposes a health endpoint verifying database and cache connectivity:
```bash
curl -f https://<backend-domain>/api/health
```
**Expected Response (HTTP 200)**:
```json
{
  "status": "healthy",
  "services": {
    "mongodb": "ok",
    "redis": "ok"
  },
  "timestamp": "2026-09-16T06:00:00Z"
}
```
