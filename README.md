# AQI Health Monitor

## Overview

**AQI Health Monitor** is a personalized air quality monitoring and health alert platform designed for urban residents in Hanoi and Ho Chi Minh City. The system combines public air quality index (AQI) telemetry from the World Air Quality Index (WAQI) API with individual respiratory health profiles to calculate personalized safety thresholds and deliver proactive health advisories.

The repository is structured as a fullstack monolith containing a Go backend (split into an HTTP REST API server and an asynchronous background worker), a Next.js App Router frontend, PostgreSQL with PostGIS and TimescaleDB extensions, Redis, and Keycloak for identity management.

## Problem Statement

Public air quality applications generally report raw AQI figures using a single generic scale for all users. However, individuals with respiratory conditions—such as asthma or allergic rhinitis—as well as children, the elderly, and people with heightened sensitivity experience adverse health effects at significantly lower pollutant concentrations than healthy adults. Relying on passive lookups and one-size-fits-all thresholds often leaves vulnerable groups exposed before they realize air quality has degraded beyond their personal safe limit.

AQI Health Monitor addresses this gap by computing a personalized AQI alert threshold from a 18-combination health profile matrix (or a user-defined custom threshold) and tracking monitoring stations near the user's saved locations.

## Key Features and Implementation Status

To reflect the current state of the repository accurately, features are categorized below by their implementation status in source code:

| Feature Area | Status | Details |
| :--- | :--- | :--- |
| **OIDC Authentication & User Sync** | Implemented (Backend) | Stateless Bearer JWT verification via Keycloak JWKS (`GET /api/v1/me`) with automatic idempotent user provisioning in PostgreSQL. |
| **Health Profile & Threshold Matrix** | Implemented (Backend) | Upsert endpoint (`POST /api/v1/health-profile`) supporting 18 condition/age/sensitivity combinations and custom threshold overrides (`50`–`500`) in a single ACID transaction. |
| **Geospatial User Locations** | Implemented (Backend) | Upsert endpoint (`POST /api/v1/locations`) storing WGS84 coordinates (`GEOGRAPHY(Point, 4326)`) for `current` and `home` labels using PostGIS. |
| **WAQI Station & Reading Ingestion** | Implemented (Backend) | Worker CLI and Asynq task handlers (`sync_stations`, `fetch_readings`) that discover stations in Hanoi and Ho Chi Minh City, store readings in a TimescaleDB hypertable, and populate Redis cache (`aqi:current:{station_id}`). |
| **Current AQI Lookup (`GET /api/v1/aqi/current`)** | Planned / In Progress | Handler and service stubs exist; PostGIS nearest-station query (`<= 20 km`), Redis cache-aside read, and 3-hour staleness check are specified in `./docs/09-API_CONTRACT.md`. |
| **Automated Alert Evaluation & Notifications** | Planned / Stubbed | Database migrations exist for `push_subscriptions` and `alerts_log`; Go service/handler files (`alert_service.go`, `notification_service.go`, `push_handler.go`) are currently package stubs. |
| **Frontend Application UI** | Scaffolded | Next.js 16 App Router project initialized with Tailwind CSS v4 and 17 `shadcn/ui` primitives. Route pages, domain components, API client, and auth hooks exist as `0-byte` placeholder files awaiting integration. |

## Technology Stack

| Layer | Technology | Version / Details |
| :--- | :--- | :--- |
| **Backend Language & Runtime** | Go | `1.26.4` (`net/http` standard library router) |
| **Background Jobs & Queue** | Asynq + Redis | `github.com/hibiken/asynq v0.26.0`, `go-redis/v9 v9.22.0` |
| **Database** | PostgreSQL + PostGIS + TimescaleDB | PostgreSQL 16 (`timescale/timescaledb-ha:pg16`), `pgx/v5`, `sqlx` |
| **Identity & Access Management** | Keycloak (OAuth2 / OIDC) | Keycloak `26.0`, JWT verification via `lestrrat-go/jwx/v2` |
| **External Telemetry Provider** | WAQI API (`aqicn.org`) | Station discovery (`/map/bounds/`) and station feed (`/feed/@id/`) |
| **Frontend Framework** | Next.js (App Router) + React | Next.js `16.3.3`, React `19.2.8`, TypeScript `5` |
| **Frontend Styling & UI** | Tailwind CSS + shadcn/ui | Tailwind CSS v4, Radix UI (`radix-ui`), Lucide icons |
| **Infrastructure & Containers** | Docker & Docker Compose | Multi-stage Dockerfiles for Backend (`api`, `worker`) and Frontend |

## System Architecture

The backend follows a **Modular Monolith** architecture. Both the synchronous HTTP API (`cmd/api`) and the asynchronous background processor (`cmd/worker`) are compiled from the same Go module and share domain models, repositories, and services, while running as independent OS processes.

```mermaid
flowchart LR
    subgraph Client["Client Layer"]
        FE["Next.js Frontend (Port 3000)"]
    end

    subgraph Identity["Identity Provider"]
        KC["Keycloak 26.0 (Port 8080)"]
        KCDB[("Keycloak Postgres")]
        KC --> KCDB
    end

    subgraph Backend["Go Modular Monolith (/backend)"]
        API["cmd/api (HTTP Server :8000)"]
        WORKER["cmd/worker (Asynq / CLI)"]
    end

    subgraph Data["Data & Queue Layer"]
        PG[("PostgreSQL 16\nPostGIS + TimescaleDB (:5432)")]
        REDIS[("Redis 7\nCache + Asynq Queue (:6379)")]
    end

    subgraph External["External Services"]
        WAQI["WAQI REST API\n(api.waqi.info)"]
    end

    FE -. "Planned REST Calls (/api/v1/*)" .-> API
    API -- "Fetch JWKS (/certs)" --> KC
    API -- "Read / Write User & Profile Data" --> PG
    WORKER -- "Poll / Enqueue Tasks & Cache Readings" --> REDIS
    WORKER -- "Upsert Stations & Hypertable Readings" --> PG
    WORKER -- "GET /map/bounds/ & /feed/@id/" --> WAQI
```

## Project Directory Structure

```text
aqi-health-monitor/
├── .github/workflows/          # CI workflow placeholders (currently 0-byte files)
├── backend/                    # Go modular monolith (REST API + Asynq Worker)
│   ├── cmd/api/                # HTTP API server entrypoint
│   ├── cmd/worker/             # Background worker & manual ingestion CLI entrypoint
│   ├── internal/               # Private application packages (auth, config, database, handler, middleware, models, repository, service, worker)
│   ├── migrations/             # 12 SQL schema migration pairs (000001 to 000012, up/down)
│   ├── pkg/waqi/               # WAQI HTTP client package
│   ├── Dockerfile              # Multi-stage build for api and worker targets
│   └── go.mod                  # Go module definition
├── frontend/                   # Next.js 16 (App Router) web application
│   ├── app/                    # App Router pages, route groups, and layouts
│   ├── components/             # shadcn/ui primitives (ui/) and domain component placeholders
│   ├── hooks/                  # React UI hooks (use-mobile.ts)
│   ├── lib/                    # Utility functions (utils.ts) and API/auth/hook placeholders
│   ├── store/                  # State management placeholder
│   ├── types/                  # TypeScript type definition placeholders
│   ├── Dockerfile              # Multi-stage standalone Next.js build
│   └── package.json            # Node.js dependencies and scripts
├── docs/                       # System design, business rules, DB schema, and API contract docs
├── task/                       # Project tracking logs and internal audit artifacts
├── .env.example                # Template for root environment variables
└── docker-compose.yml          # Local infrastructure stack (Postgres, Redis, Keycloak)
```

## Prerequisites

Ensure the following tools are installed on your machine before setting up the project:

- **Docker & Docker Compose**: Required to run PostgreSQL (with PostGIS and TimescaleDB), Redis, and Keycloak.
- **Go**: Version `1.26.4` (as specified in `backend/go.mod`).
- **Node.js & npm**: Node.js `20.x` or newer (Dockerfile targets `node:20-alpine`), with `npm`.
- **SQL Migration Client**: `golang-migrate` CLI (`migrate`) or `psql` to apply the SQL migration files in `backend/migrations/`.

## Getting Started

### 1. Configure Environment Variables

Open a terminal at the **repository root** (`aqi-health-monitor/`) and create a `.env` file from `.env.example`:

```bash
cp .env.example .env
```

Update `.env` with safe local values. Below is the complete list of variables referenced across `docker-compose.yml`, `.env.example`, and `backend/internal/config/config.go`:

```dotenv
# PostgreSQL (Application Database)
DB_HOST=localhost
DB_PORT=5432
DB_USER=admin
DB_PASSWORD=replace_with_local_db_password
DB_NAME=aqi

# Keycloak Database (Dedicated PostgreSQL instance in Docker)
KEYCLOAK_DB_USER=keycloak
KEYCLOAK_DB_PASSWORD=replace_with_local_kc_db_password
KEYCLOAK_DB_NAME=keycloak

# Keycloak Administration & OIDC Realm
KEYCLOAK_ADMIN_USER=admin
KEYCLOAK_ADMIN_PASSWORD=replace_with_local_kc_admin_password
KEYCLOAK_REALM=aqi-monitor
KEYCLOAK_ISSUER=http://localhost:8080/realms/aqi-monitor

# Backend API Server
APP_PORT=8000

# Redis (Cache & Asynq Queue)
REDIS_ADDR=localhost:6379

# External APIs
WAQI_API_TOKEN=replace_with_your_waqi_token
WAQI_BASE_URL=https://api.waqi.info
RESEND_API_KEY=replace_with_your_resend_key
```

> **Note on Configuration**: `backend/internal/config/config.go` reads `KEYCLOAK_ISSUER` directly (defaulting to `http://localhost:8080/realms/aqi-monitor`), whereas `docker-compose.yml` uses `KEYCLOAK_REALM`. Defining both in `.env` keeps local execution and Docker Compose aligned.

### 2. Start Infrastructure Services

From the **repository root**, start PostgreSQL (TimescaleDB + PostGIS), Redis, and Keycloak via Docker Compose:

```bash
docker compose up -d
```

Verify that `aqi-postgres` (port `5432`), `aqi-redis` (port `6379`), and `aqi-keycloak` (port `8080`) are running:

```bash
docker compose ps
```

> **Note**: In `docker-compose.yml`, the `backend-api`, `backend-worker`, and `frontend` service definitions are currently commented out so you can run and debug the Backend and Frontend processes directly on the host machine.

### 3. Apply Database Migrations

The database schema is defined in 12 ordered migration pairs (`000001` through `000012`) inside `backend/migrations/`. Apply all `.up.sql` migrations in numerical order before starting the backend.

Using `golang-migrate` CLI from the **repository root**:

```bash
migrate -path ./backend/migrations -database "postgres://${DB_USER}:${DB_PASSWORD}@localhost:5432/${DB_NAME}?sslmode=disable" up
```

Alternatively, if you do not have `migrate` installed, you can pipe the `.up.sql` files sequentially into the running `aqi-postgres` container using `psql` (see [Backend README](./backend/README.md#database-setup-and-migrations) for details).

### 4. Install Dependencies and Start the Backend

Open a terminal in the **`backend/`** directory:

```bash
cd backend
go mod download
```

Start the HTTP API server:

```bash
go run ./cmd/api
```

The API server listens on `http://localhost:8000` by default.

To run the background worker or trigger station/reading ingestion manually, open a second terminal in **`backend/`**:

```bash
# Discover and upsert monitoring stations for Hanoi and Ho Chi Minh City
go run ./cmd/worker -task=sync_stations

# Fetch latest AQI readings for active stations and cache them in Redis
go run ./cmd/worker -task=fetch_readings

# Or start the persistent Asynq worker queue server
go run ./cmd/worker -task=serve
```

### 5. Install Dependencies and Start the Frontend

Open a new terminal in the **`frontend/`** directory:

```bash
cd frontend
npm install
npm run dev
```

Access the web application in your browser at `http://localhost:3000`.

### 6. Local Execution Notes

- **Keycloak Realm Setup**: The Keycloak container starts in development mode (`start-dev`) at `http://localhost:8080`. Because no realm export file is bundled in the repository, you must create the realm matching `KEYCLOAK_ISSUER` (e.g., `aqi-monitor`) and an OIDC client in the Keycloak Admin Console before testing live Bearer JWT requests against `/api/v1/*`.
- **Environment File Discovery**: Both `cmd/api/main.go` and `cmd/worker/main.go` automatically look for `.env`, `../.env`, and `../../.env`, so running `go run` from inside `backend/` automatically loads the root `.env` file.
- **WAQI API Token**: Running `sync_stations` or `fetch_readings` against `https://api.waqi.info` requires a valid `WAQI_API_TOKEN` registered at `aqicn.org/data-platform/token/`.

## Testing

### Backend Unit Tests

Run the Go test suite from the **`backend/`** directory:

```bash
cd backend
go test -v ./...
```

This executes unit tests for the 18-combination health profile threshold matrix (`internal/service/health_profile_service_test.go`).

### Frontend Linting

From the **`frontend/`** directory, run ESLint:

```bash
cd frontend
npm run lint
```

*(No unit or end-to-end test runner is currently configured in `frontend/package.json`.)*

## Documentation Links

### Component Documentation

- [Backend Documentation (`./backend/README.md`)](./backend/README.md)
- [Frontend Documentation (`./frontend/README.md`)](./frontend/README.md)

### System & Domain Specifications (`./docs`)

- [01 — Domain Research & WAQI/Health Reference](./docs/01-RESEARCH.md)
- [02 — Business Rules & Threshold Calculation Matrix](./docs/02-BUSINESS-RULES.md)
- [03 — User Flows](./docs/03-USER-FLOWS.md)
- [04 — Low-Fidelity Wireframes](./docs/04-WIREFRAMES.md)
- [05 — UI/UX Design Decisions](./docs/05-DESIGN-DECISIONS.md)
- [06 — System Flows (Ingestion & Alert Pipeline)](./docs/06-SYSTEM-FLOW.md)
- [07 — Database Design & Schema History](./docs/07-DATABASE_DESIGN.md)
- [08 — System Architecture Specification](./docs/08-SA.md)
- [09 — REST API Contract (`v1`)](./docs/09-API_CONTRACT.md)
- [Sequence Diagram — `GET /api/v1/me` Flow](./docs/architecture/sequence-diagrams/get-me-flow.md)

## Visual Assets & Diagrams

Architecture diagrams, wireframes, and flowcharts are available in [`./docs/assets/`](./docs/assets/), including:

- System Context Diagram: [`./docs/assets/hinh-1-1-context-diagram.png`](./docs/assets/hinh-1-1-context-diagram.png)
- High-Fidelity Dashboard Mockup: [`./docs/assets/hifi-dashboard.png`](./docs/assets/hifi-dashboard.png)
- Wireframes (`01`–`07`): [`./docs/assets/wireframe-01.png`](./docs/assets/wireframe-01.png) to [`./docs/assets/wireframe-07.png`](./docs/assets/wireframe-07.png)

## License

No license file is currently present in the repository (`TODO`: specify project license).
