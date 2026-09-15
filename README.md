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
