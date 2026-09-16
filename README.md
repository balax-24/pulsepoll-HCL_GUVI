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
│   │   └── websocket/        # (Phase 8)
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
