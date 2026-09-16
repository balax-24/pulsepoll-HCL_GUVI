# PulsePoll — Real-Time Live Polling Platform

PulsePoll is a production live polling web application built for the **HCL GUVI Full Stack Development Internship Selection Assignment**.

It enables creators to build interactive polls, share public links with audiences, and observe real-time vote updates over persistent WebSocket connections without manual page refreshes.

---

## 1. Live Public Deployment & Submission Links

PulsePoll is fully deployed and accessible on the public internet:

| Component | URL / Endpoint | Infrastructure |
| :--- | :--- | :--- |
| **Frontend Application** | [https://pulsepoll-frontend.onrender.com](https://pulsepoll-frontend.onrender.com) | Render (Static Site with SPA rewrite) |
| **Backend REST API** | [https://pulsepoll-backend.onrender.com/api](https://pulsepoll-backend.onrender.com/api) | Render (Docker Web Service) |
| **Public Health Check** | [https://pulsepoll-backend.onrender.com/api/health](https://pulsepoll-backend.onrender.com/api/health) | Live dependency probe (`mongodb: ok`, `redis: ok`) |
| **Realtime WebSocket** | `wss://pulsepoll-backend.onrender.com/api/polls/:id/ws` | Go / Gorilla WebSocket over WSS |
| **Database** | MongoDB Atlas (Cluster0 M0) | Managed Cloud Database |
| **Cache & Realtime Broker** | Render KeyValue (Redis 7) | Managed In-Memory Data Store |
| **GitHub Repository** | [https://github.com/balax-24/pulsepoll-HCL_GUVI](https://github.com/balax-24/pulsepoll-HCL_GUVI) | Source Code & CI/CD Spec |

---

## 2. Technology Stack Overview

| Layer | Technology | Primary Architectural Responsibility |
| :--- | :--- | :--- |
| **Frontend** | React 19, Vite, Vanilla CSS | Responsive audience voting UI, animated live result bars, custom SVG icon system, WebSocket stream consumer |
| **Backend** | Go 1.23, Gin Web Framework | Concurrency-safe REST API, payload validation, JWT authentication, WebSocket connection hub |
| **Database** | MongoDB 7.0 | Authoritative persistence for users, polls, options, and immutable vote audit logs |
| **Cache & Realtime** | Redis 7.0 | Atomic live vote tallying (`HINCRBY`) and cross-instance event distribution via Redis Pub/Sub |

### Architectural Design Principles
- **Dual-Layer Storage**: Separates durable disk persistence (MongoDB) from in-memory live counter aggregation (Redis).
- **Atomic Concurrency**: Vote counting uses atomic Redis `HINCRBY` operations, eliminating lost updates and race conditions under concurrent submissions.
- **Push-Based Updates**: Connected audience clients receive vote events via persistent WebSocket connections, avoiding periodic polling loops.
- **Strict Data Integrity**: MongoDB unique compound indexes `(poll_id, voter_id)` enforce single-vote rules at the database engine level.

---

## 3. Realtime Architecture Flow

```
[ Audience Client (React) ]
         │ (HTTP POST /api/polls/:id/vote)
         ▼
[ Go / Gin Backend ]
         ├── 1. Validates poll lifecycle (active, not expired, valid option)
         ├── 2. Persists immutable vote audit document to MongoDB (unique index prevents duplicates)
         ├── 3. Atomically increments option tally in Redis (HINCRBY poll:{id}:votes {opt} 1)
         └── 4. Publishes lightweight JSON event to Redis Pub/Sub (poll:{id}:updates)
                     │
                     ▼
             [ Redis Broker ]
                     │
                     ▼
         [ Go WebSocket Hub ] ─── (Broadcasts JSON frame over WSS) ───► [ Connected Audience Clients ]
```

---

## 4. Repository Structure

```
.
├── docker-compose.yml          # Local dev services: MongoDB (27017) & Redis (6380)
├── docker-compose.prod.yml     # Self-hosted production stack with Nginx & persistent volumes
├── render.yaml                 # Render Blueprint for automated cloud deployment
├── .env.example                # Unified configuration reference
├── .gitignore                  # Production-safe exclusions (secrets, builds, temp files)
├── README.md                   # System documentation & assignment alignment
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go         # Application entrypoint, route wiring & graceful shutdown
│   ├── internal/
│   │   ├── config/             # Environment loader with type validation
│   │   ├── database/
│   │   │   ├── mongodb/        # MongoDB connection lifecycle & health ping
│   │   │   └── redis/          # Redis connection lifecycle & health ping
│   │   ├── health/             # Dependency health check endpoint (/health & /api/health)
│   │   ├── middleware/         # Structured slog logger, panic recovery, CORS handler
│   │   ├── response/           # Standardized JSON envelopes & structured error codes
│   │   ├── auth/               # User registration, bcrypt hashing, JWT issuance & auth middleware
│   │   ├── polls/              # Poll domain model, CRUD, creator ownership & lifecycle
│   │   ├── votes/              # Dual-layer voting engine (MongoDB audit log + Redis hash counters)
│   │   └── websocket/          # Dynamic PollHub registry, Redis Pub/Sub listener & WSS broadcaster
│   ├── Dockerfile              # Multi-stage Alpine container build
│   ├── go.mod                  # Go module dependencies
│   └── go.sum                  # Cryptographic checksums
└── frontend/
    ├── src/
    │   ├── api/
    │   │   ├── client.js       # Centralized fetch wrapper with JWT bearer injection & error normalization
    │   │   ├── auth.js         # Authentication endpoints (register, login, getMe)
    │   │   ├── polls.js        # Poll management endpoints (create, getPublic, getMy, update, close, delete)
    │   │   └── votes.js        # Voting and results endpoints (castVote, getResults)
    │   ├── components/
    │   │   ├── Alert.jsx       # Dismissible notification banner
    │   │   ├── Footer.jsx      # Application branding and navigation footer
    │   │   ├── Icons.jsx       # Handcrafted SVG icon library (Zero external font dependencies)
    │   │   ├── LiveBadge.jsx   # Real-time connection indicator (Connected, Connecting, Reconnecting, Offline)
    │   │   ├── Navbar.jsx      # Navigation bar with dynamic authentication state
    │   │   ├── PollCard.jsx    # Creator dashboard card with one-click link sharing & actions
    │   │   ├── PollResults.jsx # Animated result bars with W3C ARIA progress semantics
    │   │   └── ProtectedRoute.jsx # Client-side route guard with session restoration
    │   ├── context/
    │   │   └── AuthContext.jsx # Authentication state provider with automatic session verification
    │   ├── hooks/
    │   │   └── usePollWebSocket.js # WebSocket hook with bounded exponential backoff & cleanup
    │   ├── pages/
    │   │   ├── HomePage.jsx    # Landing page with architectural overview & interactive product preview
    │   │   ├── LoginPage.jsx   # Creator sign-in with password toggle & validation
    │   │   ├── RegisterPage.jsx # Creator registration with 8-char validation & auto-login
    │   │   ├── DashboardPage.jsx # Creator poll management dashboard with search, metrics & filters
    │   │   ├── CreatePollPage.jsx # Dynamic poll builder (2–10 options) with share modal
    │   │   ├── PublicPollPage.jsx # Public voting interface & live animated results
    │   │   └── ManagePollPage.jsx # Creator poll administration console (close, delete, edit)
    │   ├── test/               # Vitest component & integration test suite
    │   ├── App.jsx             # React Router route hierarchy
    │   ├── index.css           # Linear-inspired design system with CSS custom properties
    │   └── main.jsx            # Application mount
    ├── Dockerfile              # Production Nginx container build
    ├── package.json            # Node.js dependencies and scripts
    └── vite.config.js          # Vite build & test configuration
```

---

## 5. Local Development vs. Production Deployment

| Setting | Local Development | Production Cloud Deployment |
| :--- | :--- | :--- |
| **Frontend Host** | Vite Dev Server (`http://localhost:5173`) | Render Static Site (`https://pulsepoll-frontend.onrender.com`) |
| **Backend Host** | Local Go Process (`http://localhost:8080`) | Render Docker Web Service (`https://pulsepoll-backend.onrender.com`) |
| **WebSocket Scheme** | Unencrypted WebSocket (`ws://localhost:8080/api/...`) | TLS Encrypted WebSocket (`wss://pulsepoll-backend.onrender.com/api/...`) |
| **MongoDB** | Local Docker Container (`mongodb://localhost:27017`) | MongoDB Atlas Managed Replica Set (`mongodb+srv://...`) |
| **Redis** | Local Docker Container (`redis://localhost:6380`) | Render KeyValue Managed Redis (`rediss://...`) |
| **Session Cookies** | `SameSite=Lax`, `Secure=false` | `SameSite=None`, `Secure=true`, `HttpOnly=true` |
| **CORS Origins** | `http://localhost:5173,http://127.0.0.1:5173` | `https://pulsepoll-frontend.onrender.com` |

### Step-by-Step Local Setup

#### Prerequisites
* Go 1.23+
* Node.js 20+ and npm
* Docker and Docker Compose

#### 1. Start Local Infrastructure (MongoDB & Redis)
```bash
docker compose up -d
```
* MongoDB is mapped to `localhost:27017`
* Redis is mapped to `localhost:6380` (avoids colliding with default port 6379)

#### 2. Configure Backend Environment
```bash
cp backend/.env.example backend/.env
```

#### 3. Run Backend Server
```bash
cd backend
go run cmd/server/main.go
```
Verify the server is running:
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

#### 4. Configure & Start Frontend
```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```
Open `http://localhost:5173` in your browser.

---

## 6. Authentication API & Security

PulsePoll implements secure, state-free authentication for poll creators using **bcrypt (cost 10)** for password hashing and signed **JWT (HMAC-SHA256)** for request authorization.

### Endpoints

| Method | Endpoint | Auth Required | Description |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/auth/register` | No | Creates a creator account; enforces unique normalized email & minimum 8-char password |
| `POST` | `/api/auth/login` | No | Validates credentials; returns signed JWT and safe user profile DTO |
| `GET` | `/api/auth/me` | Yes (`Bearer <token>`) | Returns profile information for the authenticated creator |

### Security Measures
1. **Password Hashing**: Passwords are saved only as bcrypt hashes. Plaintext passwords are never logged or stored.
2. **Email Normalization**: Emails are trimmed and converted to lowercase before uniqueness verification, preventing case-manipulation duplicates (`Creator@Pulse.io` vs `creator@pulse.io`).
3. **Timing-Safe Generic Errors**: Login returns a generic `401 Unauthorized` (`Invalid email or password`) to eliminate account enumeration attacks.
4. **JWT Verification**: Tokens include standard `iss`, `sub`, `iat`, and `exp` claims with strict signature validation against HMAC-SHA256.

---

## 7. Poll Management API

PulsePoll provides a structured domain model and REST API for poll creation, audience discovery, and ownership-guarded poll administration.

### Poll Data Model (MongoDB `polls` collection)
```json
{
  "id": "6aa9eac27795b69d0e40874d",
  "creator_id": "6aa9eac27795b69d0e408747",
  "question": "Which architecture pattern fits your team best?",
  "options": [
    { "id": "6aa9eac27795b69d0e408749", "text": "Modular Monolith" },
    { "id": "6aa9eac27795b69d0e40874a", "text": "Microservices" },
    { "id": "6aa9eac27795b69d0e40874b", "text": "Event-Driven Serverless" }
  ],
  "status": "active",
  "created_at": "2026-09-16T01:02:58.775Z",
  "updated_at": "2026-09-16T01:02:58.775Z"
}
```

*Option Stability*: Each poll option is assigned a stable, globally unique ObjectID at creation time. This ID remains consistent across all voting submissions, Redis counters, and WebSocket broadcast payloads.

### Creator Ownership & Administration
* **JWT Identity**: The creator identity is extracted strictly from the verified JWT claim (`sub`), preventing unauthorized manipulation of `creator_id`.
* **Ownership Verification**: Mutating operations (`PATCH`, `DELETE`, status transitions) verify that `poll.creator_id == authenticated_user_id`. Unauthorized requests receive `403 Forbidden`.
* **Scoped Dashboard**: `GET /api/my/polls` returns only polls belonging to the authenticated user, ordered newest first.

### Endpoints

| Method | Endpoint | Auth Required | Description |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/polls` | Yes (`Bearer <token>`) | Creates a new active poll with 2–10 options |
| `GET` | `/api/polls/:id` | No (Public) | Retrieves sanitized poll details for audience view |
| `GET` | `/api/my/polls` | Yes (`Bearer <token>`) | Returns paginated list of polls owned by the creator |
| `PATCH` | `/api/polls/:id` | Yes (Owner only) | Updates question or option texts while preserving stable IDs |
| `PATCH` | `/api/polls/:id/status` | Yes (Owner only) | Transitions lifecycle status (e.g. `{"status": "closed"}`) |
| `POST` | `/api/polls/:id/close` | Yes (Owner only) | Dedicated endpoint to idempotently close a poll |
| `DELETE` | `/api/polls/:id` | Yes (Owner only) | Soft-deletes poll; preserves audit integrity for vote records |

---

## 8. Voting & Dual-Layer Realtime Counters

PulsePoll features a concurrency-safe, dual-layer voting system combining durable audit records in MongoDB with in-memory atomic counters in Redis.

### Vote Data Model (MongoDB `votes` collection)
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
Live vote counting is executed in-memory using Redis Hashes:
* **Key Schema**: `poll:{pollID}:votes`
* **Fields**: `{optionID} -> {count}`
* **Atomic Mutation**: Increments are performed strictly via `HINCRBY poll:{pollID}:votes {optionID} 1` (preventing read-modify-write lost updates).

```bash
redis-cli HGETALL poll:6aa9ed250cf658f70a932a87:votes
1) "6aa9ed250cf658f70a932a84"
2) "42"
3) "6aa9ed250cf658f70a932a85"
4) "18"
```

### Anonymous Voter Identification & Duplicate Prevention
1. **Session Resolution**: Audience participants receive a cryptographically random 128-bit session token generated server-side.
2. **Dual Transmission**: Stored in an `HttpOnly` cookie (`pulsepoll_voter_id`) and mirrored in `X-Voter-ID` response headers for programmatic clients.
3. **Database Concurrency Barrier**: MongoDB enforces a unique compound index on `(poll_id, voter_id)`. Concurrent requests from the same session race at the database layer; exactly one insert succeeds, while duplicate submissions return `409 Conflict` (`DUPLICATE_VOTE`).

### Storage Consistency & Reconciliation
* **Write Order**: MongoDB durable persistence occurs **first**. If MongoDB rejects the vote (due to duplicate key or validation error), Redis is never touched.
* **Crash Reconciliation**: If Redis restarts or drops in-memory keys, the service detects the missing hash and reconstructs counters directly from MongoDB aggregation pipelines using `HSETNX`.

### Endpoints

| Method | Endpoint | Auth Required | Description |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/polls/:id/vote` | No (Public) | Casts a vote; validates voter session; persists to MongoDB; increments Redis live count |
| `GET` | `/api/polls/:id/results` | No (Public) | Returns vote tallies and calculated percentages from Redis live counters |

---

## 9. Realtime WebSocket Broadcasting Architecture

### Component Roles

| Component | Responsibility |
| :--- | :--- |
| **MongoDB** | Durable source of truth for users, polls, options, and immutable vote records. |
| **Redis Hash (`poll:{id}:votes`)** | Authoritative live counter aggregation. Modified strictly via atomic `HINCRBY`. |
| **Redis Pub/Sub (`poll:{id}:updates`)** | Realtime event transport distributing vote notifications across server instances. |
| **Go WebSocket Gateway (`PollHub`)** | Concurrency-safe client registry. Pre-validates polls before upgrade. Dynamically provisions Redis subscriptions on first client join; cancels them on last client exit. Dispatches ping/pong keepalives and isolates slow clients via buffered channels. |
| **React Frontend Client** | Consumer of WebSocket streams. Merges initial snapshot and incremental live vote updates into UI state without page reload. |

### WebSocket Endpoint: `GET /api/polls/:id/ws`
* **Access**: Public audience endpoint (no creator auth required).
* **Pre-Validation**: Checks hex ObjectID format and verifies poll existence in MongoDB before completing the HTTP-to-WebSocket upgrade.
* **Closed Poll Support**: Audience members can connect to closed polls to view final results. When an active poll is closed by its creator, a terminal `poll_closed` event is broadcast to all connected viewers while preserving the connection.

### WebSocket Protocol Message Contracts

#### 1. Initial State: `results_snapshot`
Sent immediately upon WebSocket connection to establish the baseline:
```json
{
  "type": "results_snapshot",
  "poll_id": "6aa9ed250cf658f70a932a87",
  "options": [
    {
      "option_id": "6aa9ed250cf658f70a932a84",
      "text": "Go",
      "count": 12,
      "votes": 12,
      "percentage": 60.0
    },
    {
      "option_id": "6aa9ed250cf658f70a932a85",
      "text": "Rust",
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
*Privacy Note*: Realtime update events never include voter identifiers, session cookies, JWTs, or IP addresses.

#### 3. Poll Lifecycle: `poll_closed`
Broadcast when the creator closes the poll:
```json
{
  "type": "poll_closed",
  "poll_id": "6aa9ed250cf658f70a932a87",
  "timestamp": "2026-09-16T01:38:38.572Z"
}
```

---

## 10. Frontend Architecture & User Experience

PulsePoll features a modern, responsive single-page web interface built with **React 19**, **Vite**, and a cohesive dark-mode design system inspired by tools like Linear and GitHub.

### Critical Distinction: Marketing Preview vs. Production Polls

> [!IMPORTANT]
> **Homepage Product Preview vs. Production Polls:**
> The homepage features an interactive **Product Preview card** designed strictly for product demonstration and onboarding. It simulates the voting interaction locally in the browser with sample options to illustrate the user experience.
>
> In contrast, actual polls accessed via `/polls/:id` and `/polls/:id/manage` connect directly to the production Go/Gin backend and Redis Pub/Sub WebSocket streaming infrastructure (`wss://pulsepoll-backend.onrender.com/api/polls/:id/ws`).

### Key Frontend Features
* **Handcrafted SVG Icon System (`Icons.jsx`)**: Comprehensive set of vector icons crafted specifically for PulsePoll, eliminating external font dependencies or icon bloat.
* **Centralized API Client (`client.js`)**: Handles URL configuration, JWT bearer injection, cookie credential transmission, and structured error code normalization.
* **Resilient WebSocket Hook (`usePollWebSocket.js`)**: Implements bounded exponential backoff (1s, 2s, 4s, up to 10s), ping/pong resilience, and React 19 StrictMode cleanup.
* **Smooth Animations**: Real-time result progress bars update smoothly using CSS transitions (`transition: width 0.45s cubic-bezier(0.4, 0, 0.2, 1)`).
* **Accessibility**: W3C ARIA progressbar semantics, visible keyboard focus rings (`:focus-visible`), labeled inputs, and `@media (prefers-reduced-motion: reduce)` support.

### Application Routes

| Route | Access | Purpose |
| :--- | :--- | :--- |
| `/` | Public | Marketing landing page with interactive preview & architecture walkthrough |
| `/login` | Public | Creator sign-in with password toggle & validation |
| `/register` | Public | Creator registration with auto-login into dashboard |
| `/polls/:id` | **Public** | **Public audience voting & live real-time results** (No authentication required) |
| `/dashboard` | Protected | Creator dashboard listing all created polls, metrics, and filters |
| `/polls/create` | Protected | Dynamic poll builder (2 to 10 options) with share modal |
| `/polls/:id/manage` | Protected | Creator poll administration console (close, delete, update question) |

---

## 11. Production Cloud Deployment (Render + Atlas)

PulsePoll is deployed using Render's Infrastructure as Code blueprint (`render.yaml`):

```
                    [ React SPA (Render Static Site) ]
                                   │
                              HTTPS / WSS
                                   │
                                   ▼
                      [ Go / Gin API Server ]
                      (Render Docker Service)
                        /                 \
                       /                   \
                      ▼                     ▼
           [ MongoDB Atlas Cluster0 ]   [ Render KeyValue Redis ]
           (Durable Persistence)        ├── Live Counters (Hash)
                                        └── Realtime Broker (Pub/Sub)
                                                 │
                                                 ▼
                                     [ Go WebSocket Hub ]
                                                 │ (wss://)
                                                 ▼
                                     [ Active Browser Clients ]
```

### Protocol Resolution & Security in Production
* **REST Calls**: The frontend automatically routes requests to `https://pulsepoll-backend.onrender.com/api`.
* **WebSocket Connections**: Dynamically resolves to `wss://pulsepoll-backend.onrender.com/api/polls/:id/ws`.
* **Cross-Site Cookies**: The anonymous voter session cookie (`pulsepoll_voter_id`) uses `SameSite=None; Secure=true; HttpOnly=true; Path=/` when served over HTTPS, allowing session persistence across distinct Render domains.

### Self-Hosted Production Stack (Docker Compose)
For self-hosted environments, PulsePoll includes `docker-compose.prod.yml`:
```bash
# Launch self-hosted production stack
docker compose -f docker-compose.prod.yml up -d --build
```
This deploys containerized MongoDB with persistent volumes, Redis with append-only file (AOF) persistence, the Go API server, and Nginx serving the production React bundle with SPA routing.

---

## 12. Automated Verification & Testing Commands

### Backend Verification
From the `backend/` directory:
```bash
# Run all unit and integration tests with Go race detector enabled
go test -race -count=1 ./...

# Run static analysis
go vet ./...
```
*Result*: 100% of internal packages (`auth`, `config`, `health`, `middleware`, `polls`, `response`, `votes`, `websocket`) pass with zero race conditions or vet warnings.

### Frontend Verification
From the `frontend/` directory:
```bash
# Run Vitest test suite
npm test -- --run

# Run Oxlint static analysis
npm run lint

# Build production bundle
npm run build
```
*Result*: 16/16 Vitest tests passing, 0 linter errors, and clean production asset bundling.

### Live Production End-to-End Test Suite
PulsePoll includes a 19-step automated production test suite verifying the live Render deployment:
```bash
go run verify_public_deployment.go
```
*Test Coverage*:
1. Health check returns MongoDB and Redis status as `ok`.
2. Creator registration and login via JWT.
3. Authenticated session validation (`/api/auth/me`).
4. Poll creation with multiple options.
5. Creator dashboard retrieval (`/api/my/polls`).
6. Unauthenticated public poll access (`/api/polls/:id`).
7. Multi-client WebSocket connection over WSS (`Client A` and `Client B`).
8. Initial state snapshot delivery upon connection.
9. Bidirectional live vote updates propagated over WSS without page refresh.
10. Duplicate vote rejection (`HTTP 409 DUPLICATE_VOTE`).
11. Creator poll closure and broadcast of `poll_closed` event over WSS.
12. Vote rejection on closed polls (`HTTP 409 POLL_CLOSED`).
13. Direct SPA route navigation without 404s.

---

## 13. HCL GUVI Assignment Requirement Checklist

| # | Assignment Requirement | Implementation Detail | Verified Status |
| :--- | :--- | :--- | :--- |
| 1 | **React Frontend** | Built with React 19, Vite, responsive Linear-inspired design system | Verified |
| 2 | **Go / Gin Backend** | REST API and WebSocket gateway built with Go 1.23 & Gin | Verified |
| 3 | **MongoDB Persistence** | MongoDB 7.0 (Atlas M0 in prod) stores users, polls, options & vote audit logs | Verified |
| 4 | **Redis Realtime Engine** | Redis 7.0 handles atomic counters (`HINCRBY`) and Pub/Sub event distribution | Verified |
| 5 | **Poll Creation** | Creator form supports 2–10 options, dynamic option management, and instant link generation | Verified |
| 6 | **Shareable Public Link** | Public route `/polls/:id` accessible by anyone without login or signup | Verified |
| 7 | **Audience Voting Flow** | Public audience selects option, submits vote, and receives immediate confirmation | Verified |
| 8 | **Realtime Updates Without Refresh** | Connected clients receive live vote updates via WebSockets and animated progress bars | Verified |
| 9 | **Creator Authentication** | JWT-protected routes for poll creation, dashboard, editing, closing, and deletion | Verified |
| 10 | **Input & Concurrency Validation** | Backend validates question length, option limits, and enforces single-vote rules | Verified |
| 11 | **Public Cloud Deployment** | Frontend & Backend deployed on Render; MongoDB on Atlas; Redis on Render KeyValue | Verified |
| 12 | **Public GitHub Repository** | Complete source code, Docker configs, documentation, and tests on GitHub | Verified |

---

## 14. Recommended 3–5 Minute Demo Walkthrough Sequence

For evaluator demonstrations and video recordings, follow this sequence:

1. **Open Public Application**: Navigate to `https://pulsepoll-frontend.onrender.com`. Point out the system architecture and the interactive demo preview.
2. **Register/Login Creator**: Click **Sign In** or **Get Started**. Log in with creator credentials (or register a new account).
3. **Explore Dashboard**: View existing polls, status badges (Active/Closed), and total vote metrics.
4. **Create a New Poll**: Click **Create Poll**. Enter a question (e.g. *"Which distributed caching architecture do you prefer?"*) with 3 options:
   - Option 1: `Redis Cluster`
   - Option 2: `Memcached Distributed`
   - Option 3: `Dragonfly In-Memory`
5. **Copy Shareable Link**: Click **Create Poll & Get Link**, then use the **Copy Link** button from the share modal.
6. **Open Split-Screen Audience View**: Open two browser windows side by side (e.g., standard browser as Client A, incognito/private window as Client B) navigating to the public poll URL `/polls/:id`.
7. **Inspect Real-Time Indicator**: Show the green **Live** badge indicating active WebSocket connection over WSS.
8. **Vote from Client A**: Cast a vote for *Option 1* in Client A.
9. **Observe Realtime Propagation**: Point out that Client B's progress bar and vote tally update immediately without any page refresh.
10. **Vote from Client B**: Cast a vote for *Option 2* in Client B. Observe Client A updating immediately.
11. **Demonstrate Duplicate Protection**: Attempt to vote again in Client A. Show the friendly duplicate prevention alert: *"You have already voted in this poll."*
12. **Close Poll from Creator Console**: Switch to the Creator Dashboard/Manage screen and click **Close Poll**.
13. **Observe Closed Broadcast**: Both audience screens receive the `poll_closed` event, transitioning the status badge to **Voting Closed** and disabling the vote button.
14. **Architecture Summary**: Conclude by explaining the technical separation: durable MongoDB vote audit logs + in-memory Redis atomic counters + Redis Pub/Sub WebSocket event distribution.

---

## 15. Known Constraints & Architectural Trade-offs

1. **Render Free-Tier Cold Starts**: On Render's free tier, the backend web service spins down after 15 minutes of inactivity. The first request after a sleep cycle may experience an initial spin-up latency (~30–50 seconds). Once warm, requests respond normally.
2. **Cross-Site Cookie Privacy Settings**: In strict privacy browsers (e.g. Brave shields or Safari with cross-site tracking blocked), third-party cookies across differing domains (`onrender.com`) may be restricted. PulsePoll gracefully handles this by falling back to `localStorage` and `X-Voter-ID` headers.
3. **In-Memory Cache Reconciliation**: Redis is treated as an ephemeral live counter cache, while MongoDB remains the durable source of truth. In the event of a Redis restart, the application automatically reconstructs counters directly from MongoDB vote aggregation.
