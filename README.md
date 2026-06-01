# FlowGate

**Usage-based API monetization gateway** — metering, prepaid billing, and Stellar-powered settlement for developers.

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql)](https://postgresql.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue)](LICENSE)
![Status](https://img.shields.io/badge/Status-MVP-yellow)
[![codecov](https://codecov.io/gh/mustapha-aminu-abubakar/flowgate/branch/main/graph/badge.svg)](https://codecov.io/gh/mustapha-aminu-abubakar/flowgate)

---

## What is FlowGate?

FlowGate is a Go-based reverse proxy that sits in front of existing APIs and enables per-request monetization. Instead of forcing API providers into subscription models or Stripe billing, FlowGate handles authentication, request metering, prepaid balance validation, and batched Stellar settlement — all transparently to the end user.

Each request is priced, validated against a prepaid balance, recorded in an internal ledger, and aggregated for batched blockchain settlement. No per-request blockchain transactions. No subscription overhead. No payment infrastructure to build.

---

## Architecture

```
Client
  ↓
FlowGate Gateway
  ├── Auth Layer          → API key validation, consumer resolution
  ├── Metering Engine     → Per-request accounting
  ├── Pricing Engine      → Route-based cost resolution
  ├── Ledger Service      → Balance mgmt, reservations, deductions
  ├── Usage Logger        → Structured JSON logging
  └── Proxy Engine        → httputil.ReverseProxy forwarding
  ↓
Provider API

Background Workers
  ├── Settlement Worker   → Aggregate earnings, execute Stellar payouts
  ├── Deposit Watcher     → Monitor Stellar for incoming payments
  └── Usage Aggregator    → Prepare provider settlement data

Databases
  ├── PostgreSQL          → Users, providers, pricing, ledger, usage events
  └── Redis               → Rate limiting, temporary reservations, cache

Blockchain Layer
  └── Stellar Network     → Deposit routing, batched settlement payouts
```

Requests do **not** trigger blockchain transactions. Usage is aggregated in the internal ledger and settled to Stellar in batches.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Gateway | Go (`net/http`, `httputil.ReverseProxy`) |
| Database | PostgreSQL 16 + Redis 7 |
| Settlement | Stellar Network (XLM, future USDC) |
| Dashboard | Next.js 15, Tailwind CSS, shadcn/ui, recharts, QRCode.react |
| Workers | Go background services (polling, timers) |
| Containerization | Docker Compose |
| Observability | Prometheus, Grafana, OpenTelemetry |
| Migrations | goose |
| Query Layer | sqlc (type-safe Go from SQL) |

---

## Project Structure

```
flowgate/
├── cmd/
│   └── api/              # Gateway HTTP server entrypoint
├── internal/
│   ├── repository/       # sqlc queries + generated Go code
│   │   └── query/        # 10 .sql files, 48 queries
│   └── server/           # HTTP server, routes, handlers
├── migrations/           # 11 goose migration files (sequential)
├── dashboard/            # Next.js 15 dashboard (12 pages, dark theme, shadcn/ui)
├── docs/                 # PRDs, schema docs, design analysis
│   ├── flowgate_MVP_PRD.md
│   ├── mvp_schema.md
│   ├── mvp_erd.md
│   ├── ui_spec.md
│   └── db_design_analysis.md
├── docker-compose.yml    # Postgres, Redis, app services
├── Dockerfile            # Multi-stage Go build
└── Makefile              # Build, test, watch targets
```

> **Note:** The architecture above describes the target design. Business-logic packages (`auth/`, `ledger/`, `metering/`, `pricing/`, `settlement/`, `wallet/`, `worker/`) are actively being extracted from `internal/server/` as the codebase evolves from MVP toward production.

---

## Database

10 PostgreSQL tables:

```mermaid
erDiagram
    users ||--o{ api_keys : "has"
    users ||--o{ providers : "owns"
    users ||--o| accounts : "has"
    users ||--o{ usage_events : "consumes"
    providers ||--o{ api_endpoints : "exposes"
    providers ||--o{ usage_events : "metered"
    providers ||--o{ settlement_entries : "payouts"
    accounts ||--o{ ledger_entries : "audit"
    accounts ||--o{ deposits : "funds"
    api_endpoints ||--o{ usage_events : "billed"
    settlement_batches ||--o{ settlement_entries : "contains"
```

| Table | Purpose |
|---|---|
| `users` | Core identity, role (`provider`, `consumer`, or `both`), deposit memo, payout address |
| `api_keys` | Hashed bearer tokens for auth |
| `providers` | Upstream API configuration |
| `api_endpoints` | Routes with fixed per-request pricing |
| `accounts` | Internal prepaid credit (fast auth, not on-chain) |
| `ledger_entries` | Immutable audit trail for all financial ops |
| `usage_events` | Per-request metering records (idempotent) |
| `deposits` | Incoming Stellar payment tracking |
| `settlement_batches` | Grouped payout transactions |
| `settlement_entries` | Per-provider payout line items |

See [`docs/mvp_schema.md`](docs/mvp_schema.md) for full DDL and [`docs/mvp_erd.md`](docs/mvp_erd.md) for the ERD.

---

## Screenshots

<p align="center">
  <img src="docs/screenshots/overview.png" alt="Provider Overview" width="700">
  <br><em>Provider dashboard — earnings overview, 7-day chart, recent API calls</em>
</p>

<p align="center">
  <img src="docs/screenshots/analytics.png" alt="Analytics" width="700">
  <br><em>Usage analytics — stacked request timeline and endpoint revenue breakdown</em>
</p>

<p align="center">
  <img src="docs/screenshots/deposits.png" alt="Deposit" width="700">
  <br><em>Consumer deposit screen — SEP-7 QR code, address/memo copy, deposit history</em>
</p>

---

## Quick Start

### Prerequisites

- Go 1.26+
- Docker & Docker Compose
- PostgreSQL 16 (or via Docker)
- Redis 7 (or via Docker)
- Stellar testnet account (for development)

### Setup

```bash
# Clone the repo
git clone https://github.com/mustapha-aminu-abubakar/flowgate.git
cd flowgate

# Start infrastructure
docker compose up -d postgres redis

# Run migrations
goose -s -dir migrations postgres "postgres://postgres:postgres@localhost:5432/flowgate?sslmode=disable" up

# Build and run gateway
go run ./cmd/api
```

### Dashboard

```bash
cd dashboard
npm install
npm run dev      # → http://localhost:3000
```

The dashboard runs independently — mock data is used until the Go API is running.  
Role toggle supports all three options: **Provider**, **Consumer**, or **Both** — the sidebar adapts nav items per role.

### Configuration

Environment variables (or `.env` file):

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/flowgate` | PostgreSQL connection |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection |
| `STELLAR_HORIZON` | `https://horizon-testnet.stellar.org` | Stellar network endpoint |
| `GATEWAY_PORT` | `8080` | Gateway HTTP port |
| `WALLET_SECRET_KEY` | — | Stellar secret key for deposit wallet |

---

## API Overview

Management endpoints (dashboard consumes these):

```
POST   /providers                     # Register a provider
GET    /providers/:id                 # Get provider details
POST   /providers/:id/endpoints       # Add an endpoint with pricing
GET    /endpoints/:id                 # Get endpoint details
POST   /wallet/deposit                # Request deposit address + SEP-7 URI
GET    /wallet/balance                # Get prepaid balance
GET    /usage                         # List usage events
```

Proxy endpoint (consumer-facing):

```
GET /proxy/{provider}/{route}
Authorization: Bearer fg_xxx
```

Full specifications in [`docs/flowgate_MVP_PRD.md`](docs/flowgate_MVP_PRD.md).

---

## Key Design Decisions

| Decision | Rationale |
|---|---|
| **accounts vs on-chain balance** | Gateway reads internal `accounts.balance` per request — never queries Stellar. Stellar wallet refs inlined on `users` table. |
| **balance_after on ledger_entries** | O(1) balance lookups and built-in consistency checking without summing full history. |
| **request_id on usage_events** | Idempotency key prevents double-billing on retry (Stripe-style). |
| **Batched settlement** | One Stellar transaction per batch, not per request. Keeps latency low and costs minimal. |
| **SEP-7 QR deposits** | Stellar URI scheme auto-fills destination + memo in compatible wallets, eliminating the #1 deposit failure mode. |
| **Polymorphic ledger refs** | `reference_id` + `reference_type` avoids three nullable FK columns per entry. |
| **sqlc over ORM** | Type-safe Go code generated from SQL. No runtime reflection, no N+1 queries, full control over join patterns. |

---

## Contributing

PRs are welcome. This is an early-stage MVP — code churn is expected.

```bash
# Run tests
make test

# Regenerate sqlc code after query changes
cd internal/repository && sqlc generate && cd ../..

# Create a new migration
goose -s -dir migrations create add_some_table sql
```

See [`docs/db_design_analysis.md`](docs/db_design_analysis.md) for known risks and design tradeoffs before making schema changes.

---

## License

Apache 2.0
