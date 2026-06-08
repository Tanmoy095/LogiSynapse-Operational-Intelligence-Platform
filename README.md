# LogiSynapse

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)](docker-compose.yml)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)](#quick-start)
[![Code Quality](https://img.shields.io/badge/code%20quality-production%20patterns-brightgreen)](#)

> **AI-native distributed logistics intelligence platform** combining production-grade backend engineering with practical AI systems engineering.

---

## 🎯 What is LogiSynapse?

LogiSynapse is a **modern, event-driven shipment and billing platform** with distributed workflow orchestration. It's designed as a learning project and production reference implementation that demonstrates:

- **Distributed Systems**: microservices, event-driven architecture, saga patterns, eventual consistency
- **Production Engineering**: idempotency, retries, observability, auditability, clean architecture
- **AI Systems**: retrieval-augmented generation (RAG), typed tools, workflow orchestration, AI evals
- **Backend Mastery**: Go service design, PostgreSQL/Redis/Kafka patterns, gRPC/GraphQL, authentication, authorization

LogiSynapse manages the **complete shipment lifecycle**: order creation → carrier selection → tracking updates → exception handling → billing → AI-assisted operational decisions.

---

## 📑 Quick Navigation

- [Architecture Overview](#architecture-overview)
- [Key Features](#key-features)
- [Tech Stack](#tech-stack)
- [Quick Start](#quick-start)
- [Service Domains](#service-domains)
- [Project Structure](#project-structure)
- [Development](#development)
- [Learning Resources](#learning-resources)

---

## 🏗️ Architecture Overview

LogiSynapse is organized into **four loose-coupled planes** that separate concerns and scale independently:

```mermaid
graph TB
    subgraph "Product Plane"
        GW["GraphQL Gateway"]
        OrdSvc["Order Service"]
        TrackSvc["Tracking Service"]
        AuthSvc["Authentication Service"]
    end
    
    subgraph "Data Plane"
        PG[("PostgreSQL")]
        REDIS[("Redis<br/>Cache")]
        KAFKA["Kafka"]
        PGVEC[("pgVector<br/>Search")]
    end
    
    subgraph "Workflow Plane"
        TEMPORAL["Temporal<br/>Orchestrator"]
        WF["Workflow<br/>Service"]
    end
    
    subgraph "Intelligence Plane"
        AIGTW["AI Gateway"]
        RETR["Retrieval<br/>Service"]
        MODEL["Model<br/>Gateway"]
        EVAL["Eval<br/>Service"]
    end
    
    subgraph "External"
        SHIPPO["Shippo API"]
        STRIPE["Stripe"]
        SMTP["Email/SMS"]
    end
    
    GW -->|validate token| AuthSvc
    GW -->|GraphQL| OrdSvc
    GW -->|GraphQL| TrackSvc
    
    OrdSvc -->|read/write| PG
    OrdSvc -->|publish events| KAFKA
    
    TrackSvc -->|cache reads| REDIS
    TrackSvc -->|fallback| PG
    
    KAFKA -->|consume| WF
    KAFKA -->|consume| EVAL
    
    WF -->|orchestrate| TEMPORAL
    WF -->|integrate| SHIPPO
    
    AuthSvc -->|persist| PG
    
    AIGTW -->|retrieve context| RETR
    RETR -->|semantic search| PGVEC
    RETR -->|read facts| PG
    
    AIGTW -->|call model| MODEL
    MODEL -->|tokens| STRIPE
    
    EVAL -->|quality checks| KAFKA
```

### Architecture Planes Explained

| Plane | Purpose | Components | Why It Matters |
|-------|---------|-----------|---|
| **Product Plane** | User-facing workflows | GraphQL Gateway, Order, Tracking, Auth | Low latency, clear APIs, user experience |
| **Data Plane** | Durable facts & read models | PostgreSQL, Redis, Kafka, pgVector | Correctness, replay, consistency, search |
| **Workflow Plane** | Long-running execution | Temporal, workflow workers | Durability, retry safety, compensation |
| **Intelligence Plane** | AI retrieval, reasoning, tools | AI Gateway, Retrieval, Model Gateway, Evals | Grounded context, tool safety, quality control |

---

## ✨ Key Features

- **Event-Driven Architecture**
  - Transactional outbox pattern for reliable event publishing
  - Kafka topic-per-aggregate-type for replay and consumer groups
  - Idempotent event handlers prevent duplicate processing

- **Durable Workflow Orchestration**
  - Temporal-backed long-running business processes
  - Automatic retry, compensation, and saga support
  - Transparent replay on failures without data corruption

- **Multi-Tenant Authentication & Authorization**
  - JWT-based access tokens with refresh-token rotation
  - Tenant membership with role-based access (RBAC)
  - Audit trail for all security events

- **AI-Native Capabilities**
  - Semantic search over shipment data and policies
  - Retrieval-augmented generation (RAG) for operator assistance
  - Typed tools prevent AI hallucination and enforce data validation
  - Quality evaluation pipeline tracks AI reliability

- **Production-Grade Observability**
  - OpenTelemetry traces across all services
  - Structured logging with correlation IDs
  - Prometheus metrics and health checks
  - Audit logs for compliance and debugging

- **Clean Architecture**
  - Domain → Application → Port → Adapter separation
  - No business logic leaks into database or HTTP layers
  - Testable, swappable infrastructure
  - Clear bounded contexts per service

---

## 🛠️ Tech Stack

| Layer | Technology | Purpose | Why Chosen |
|-------|-----------|---------|-----------|
| **Services** | Go 1.24+ | Backend services, gRPC, CLI tools | Fast, concurrent, simple, strong typing, production reliability |
| **API Gateway** | GraphQL + gRPC | Public and internal service boundaries | Type safety, strong contracts, code generation, federation |
| **Database** | PostgreSQL 15+ | Transactional data, JSONB, full-text search | ACID compliance, rich querying, proven at scale, pgVector for embeddings |
| **Caching** | Redis | Hot read cache, session storage | Sub-millisecond latency, atomic operations, Pub/Sub |
| **Events** | Apache Kafka | Domain events, event replay, consumer groups | Distributed durability, topic replay, ordering guarantees |
| **Workflows** | Temporal | Long-running orchestration, retries | Durable execution, automatic compensation, visibility |
| **Search** | pgVector + PostgreSQL | Semantic and hybrid search | Embedding storage, HNSW indexing, single database |
| **Message Queue** | RabbitMQ | Async task distribution, notifications | Dead-letter queues, manual ack, familiar for operations |
| **Observability** | OpenTelemetry + Prometheus | Tracing, metrics, logs | Vendor-neutral, standard instrumentation, ecosystem |
| **Container** | Docker & Docker Compose | Local development, deployment units | Reproducibility, isolation, multi-service orchestration |
| **Auth** | JWT (HMAC SHA-256) | Stateless access tokens, session management | Fast verification, no central lookup, refresh-token rotation |

---

## 🚀 Quick Start

**Get LogiSynapse running in ~5 minutes.**

### Prerequisites

```bash
# Verify versions
go version                      # Go 1.24+
docker --version                # Docker 24.0+
docker compose version          # Docker Compose 2.0+
```

### 1. Clone & Setup

```bash
git clone https://github.com/Tanmoy095/LogiSynapse.git
cd LogiSynapse
```

### 2. Configure Environment

The `.env` file is already configured with sensible defaults for local development:

```bash
# Review/edit if needed (most defaults are fine for local dev)
cat .env

# For production, create a separate config and update:
# - AUTH_JWT_SECRET (change from default)
# - STRIPE_SECRET_KEY (add your actual key)
# - SHIPPO_API_KEY (add your actual key)
```

If you need the template for reference:
```bash
cp .env.example .env.local  # Keep original, create a variant
```

### 3. Start the Full Stack

```bash
# Build all service images and start containers
docker compose up --build

# First run: images build and migrations run (~30-60s)
# Subsequent runs: just start containers (~5-10s)
# Logs stream in real-time; press Ctrl+C to stop
```

### 4. Verify All Services are Healthy

```bash
# In another terminal, verify containers are running:
docker compose ps

# Expected output: all services showing "Up"
# - postgres (port 5432)
# - authentication-service (port 50052)
# - shipment-service (port 50051)
# - graphql-gateway (port 8080)
# - kafka (port 9092)
# - rabbitmq (ports 5672, 15672)
# - temporal, temporal-db, temporal-ui
```

### 5. Check Service Health

```bash
# GraphQL Gateway health
curl http://localhost:8080/health

# Expected: {"status":"healthy","service":"graphql-gateway"}

# GraphQL playground (no auth required for UI)
open http://localhost:8080/

# RabbitMQ Management UI
open http://localhost:15672/  # guest:guest

# Temporal Workflow UI
open http://localhost:8088/
```

### 6. Try the GraphQL API

```bash
# 1. Register a user (no auth required)
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { registerUser(input: {email: \"user@example.com\" password: \"SecurePass123!\" firstName: \"John\" lastName: \"Doe\"}) { userId } }"
  }'

# 2. Login to get tokens
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { loginUser(input: {email: \"user@example.com\" password: \"SecurePass123!\"}) { accessToken refreshToken } }"
  }'

# 3. Use the accessToken in protected queries
# Replace TOKEN with the actual token from login response
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TOKEN" \
  -d '{
    "query": "query { shipments { id status carrier } }"
  }'
```

Or use the **GraphQL Playground** UI at http://localhost:8080/ with the same queries.

### 7. Run Tests

```bash
# Test all services
go test ./...

# Test with verbose output
go test -v ./...

# Test a specific service
cd services/shipment-service
go test ./...

# Run with race detector (recommended)
go test -race ./...
```

### 8. Stop the Stack

```bash
# Stop all containers
docker compose down

# Also remove volumes (WARNING: deletes database data)
docker compose down -v
```

---

## 📦 Service Domains

| Service | Responsibility | Primary API | Integrations | Language |
|---------|---|---|---|---|
| **GraphQL Gateway** | Public GraphQL API, bearer-token auth, tenant routing | GraphQL over HTTP | Auth Service, Shipment Service | Go |
| **Authentication Service** | Users, tenants, memberships, JWT tokens, auth audit | gRPC (JSON codec) | PostgreSQL, Kafka | Go |
| **Shipment Service** | Shipment creation, tracking, carrier integration | gRPC | Shippo API, PostgreSQL, Kafka | Go |
| **Billing Service** | Usage tracking, invoices, ledger, payments | gRPC | PostgreSQL, Stripe API, Kafka | Go |
| **Workflow Orchestrator** | Long-running shipment workflows, retries, compensation | Temporal SDK | Temporal Server, Shippo API, RabbitMQ | Go |
| **Communications Service** | Email, SMS, webhook notifications | gRPC | Email provider, SMS provider, RabbitMQ | Go |

---

## 📁 Project Structure

```
LogiSynapse/
├── services/                          # Microservices (each is independent, deployable)
│   ├── authentication-service/        # User, tenant, role, token management
│   │   ├── cmd/main.go               # Service entrypoint
│   │   ├── internal/
│   │   │   ├── domain/               # Business logic (no dependencies)
│   │   │   ├── app/                  # Use cases (coordinates domain)
│   │   │   ├── ports/                # Interfaces (contracts)
│   │   │   ├── infra/                # Adapters (SQL, gRPC, HTTP)
│   │   │   ├── transport/            # API handlers
│   │   │   └── config/               # Configuration
│   │   ├── db/migrations/            # SQL migrations (Goose)
│   │   ├── Dockerfile
│   │   └── go.mod
│   │
│   ├── graphql-gateway/              # Public GraphQL entry point
│   │   ├── cmd/main.go
│   │   ├── graph/                    # GraphQL resolvers
│   │   ├── client/                   # gRPC clients to services
│   │   └── Dockerfile
│   │
│   ├── shipment-service/             # Shipment lifecycle, carrier integration
│   │   ├── cmd/main.go
│   │   ├── service/
│   │   ├── handler/grpc/
│   │   ├── store/
│   │   ├── db/migrations/
│   │   └── Dockerfile
│   │
│   ├── billing-service/              # Usage, invoices, payments
│   │   ├── internal/
│   │   │   ├── accounts/
│   │   │   ├── billing/
│   │   │   ├── invoice/
│   │   │   ├── ledger/
│   │   │   ├── payment/
│   │   │   └── usage/
│   │   ├── db/migrations/
│   │   └── go.mod
│   │
│   ├── workflow-orchestrator/        # Temporal-based durable workflows
│   │   ├── cmd/main.go
│   │   ├── workflows/
│   │   ├── activities/
│   │   └── go.mod
│   │
│   └── communications-service/       # Email, SMS, webhooks
│       ├── cmd/main.go
│       └── go.mod
│
├── shared/                            # Shared libraries (contracts, Kafka, config)
│   ├── contracts/                    # Protobuf models
│   ├── kafka/                        # Kafka producer/consumer
│   ├── proto/                        # .proto files and generated code
│   ├── rabbitmq/                     # RabbitMQ client
│   ├── config/                       # Configuration loading
│   └── go.mod
│
├── docs/                              # Architecture, decisions, learning
│   ├── mainreadme.md                 # High-level overview
│   ├── 01-business/                  # Business context
│   ├── 02-system-design/             # HLD & LLD documents
│   │   ├── 00-implementation-status.md
│   │   ├── 01-system-overview-hld.md
│   │   ├── 02-architecture-design-hld.md
│   │   ├── 03-database-design-hld.md
│   │   ├── 04-caching-and-async-hld.md
│   │   ├── 05-networking-concepts-hld.md
│   │   ├── 06-low-level-design-lld.md
│   │   └── 07-project-structure-lld.md
│   ├── 03-microservices/
│   ├── 05-infrastructure/
│   ├── 06-observability/
│   ├── 07-roadmaps/
│   ├── 08-deployment/
│   ├── 09-decisions/                 # ADRs (Architecture Decision Records)
│   └── 10-learning/
│
├── docker-compose.yml                # Local development stack
├── .env.example                      # Environment template
├── go.mod                            # Root module (if used)
├── LICENSE                           # GPLv3
└── README.md                         # This file
```

### Detailed Navigation

- **Learn System Design**: Start with [docs/mainreadme.md](docs/mainreadme.md)
- **Architecture Decisions**: See [docs/09-decisions/](docs/09-decisions/)
- **Implementation Status**: Check [docs/02-system-design/00-implementation-status.md](docs/02-system-design/00-implementation-status.md)
- **Service Structure**: Review [docs/02-system-design/07-project-structure-lld.md](docs/02-system-design/07-project-structure-lld.md)
- **Database Schema**: See [docs/02-system-design/03-database-design-hld.md](docs/02-system-design/03-database-design-hld.md)

---

## 🔑 Core Workflows

### Order-to-Shipment Data Flow

```
Client Request (GraphQL)
  ↓
GraphQL Gateway
  ├─ Validate bearer token (auth-service)
  ├─ Parse tenant from X-Tenant-ID header
  └─ Route to shipment-service
    ↓
Shipment Service
  ├─ Validate order data
  ├─ Insert shipment + outbox rows (single transaction)
  └─ Return shipment ID
    ↓
Outbox Relay (background job)
  ├─ Poll outbox table
  ├─ Publish to Kafka: shipment.created.v1
  └─ Mark as published
    ↓
Kafka Consumers
  ├─ Workflow Service → starts fulfillment
  ├─ Tracking Service → creates read model
  ├─ Billing Service → records usage event
  └─ Notification Service → schedules customer email
```

### Tracking Read Flow (Cache Optimized)

```
Client Query (GraphQL)
  ↓
Tracking Service
  ├─ Try Redis cache lookup (key: tracking:{shipment_id})
  ├─ Hit? → Return cached result (99% case)
  └─ Miss?
    ├─ Query PostgreSQL
    ├─ Update Redis cache (60s TTL)
    └─ Return result
```

### Authentication Flow

```
Client Login Request
  ↓
Auth Service
  ├─ Normalize email
  ├─ Hash password with Argon2id
  ├─ Verify against stored hash
  ├─ Generate JWT access token (15 min TTL)
  ├─ Generate refresh token (opaque, 7 days)
  ├─ Hash and store refresh token in DB
  └─ Return both tokens
    ↓
GraphQL Gateway (subsequent requests)
  ├─ Extract bearer token from Authorization header
  ├─ Call auth-service: ValidateAccessToken
  ├─ Extract claims (user_id, tenant_id, role)
  ├─ Store in request context
  └─ Resolve GraphQL query with identity context
```

---

## � Troubleshooting

### Container Issues

**Containers fail to start or crash immediately**

```bash
# Check logs for the failing service
docker compose logs graphql-gateway
docker compose logs authentication-service
docker compose logs shipment-service

# Rebuild from scratch
docker compose down
docker system prune -a --volumes
docker compose up --build
```

**Port already in use**

```bash
# Check which process is using the port
lsof -i :8080   # GraphQL gateway
lsof -i :50052  # Auth service
lsof -i :50051  # Shipment service

# Kill the process
kill -9 <PID>

# Or update docker-compose.yml ports
```

### Database Issues

**Migrations fail on startup**

```bash
# Check auth service logs
docker compose logs authentication-service

# Manually run migrations
docker compose exec postgres psql -U postgres -d logisynapse -f /migrations/001_create_users.sql
```

**Database connection refused**

```bash
# Ensure PostgreSQL is healthy
docker compose ps postgres

# Check health status
docker compose logs postgres | grep "ready to accept"

# Wait a bit longer and retry
docker compose down
docker compose up --build --wait
```

### Service Communication Issues

**Cannot connect to authentication-service from gateway**

```bash
# Verify services are on the same network
docker network inspect loginet

# Check gateway logs for connection errors
docker compose logs graphql-gateway | grep "failed to connect"

# Ping from gateway container
docker compose exec graphql-gateway ping authentication-service

# Verify DNS resolution
docker compose exec graphql-gateway getent hosts authentication-service
```

**gRPC connection timeouts**

```bash
# Ensure AUTH_SERVICE_ADDR in .env matches docker-compose
# Should be: authentication-service:50052 (not localhost:50052)

# Check if service is actually listening
docker compose exec authentication-service netstat -tulnp | grep 50052
```

### Development

**Tests fail with "dial tcp: lookup localhost: no such host"**

```bash
# When testing in Docker, use service names, not localhost
# In docker-compose: authentication-service:50052
# In localhost dev: localhost:50052

# Run tests locally (not in container)
go test ./services/authentication-service/...
```

---

### Running Tests

```bash
# Test all services
go test ./...

# Test with coverage
go test -cover ./...

# Test a specific service
cd services/shipment-service && go test ./...

# Run tests with race detector
go test -race ./...
```

### Database Migrations

Migrations are managed per-service using [Goose](https://github.com/pressly/goose):

```bash
# Apply migrations (happens automatically on service startup with AUTO_MIGRATE=true)
cd services/authentication-service
goose postgres "postgres://user:pass@localhost/db" up

# Rollback
goose postgres "postgres://user:pass@localhost/db" down
```

### Code Generation

Regenerate artifacts after schema changes:

```bash
# GraphQL (from schema.graphql)
cd services/graphql-gateway
go generate ./...

# Protocol Buffers (from .proto files)
cd shared/proto
protoc --go_out=. --go-grpc_out=. *.proto
```

### Adding a New Service

1. Create `services/my-service/` with structure matching existing services:
   ```
   services/my-service/
   ├── cmd/main.go              # Service entrypoint
   ├── internal/
   │   ├── domain/              # Business logic
   │   ├── app/                 # Use cases
   │   ├── ports/               # Interfaces
   │   ├── infra/               # Adapters (SQL, gRPC)
   │   ├── transport/           # API handlers
   │   └── config/              # Configuration
   ├── db/migrations/           # SQL migrations
   ├── Dockerfile
   ├── go.mod
   ├── go.sum
   └── README.md                # Service-specific docs
   ```

2. Create a service `README.md` documenting:
   - Service purpose and responsibilities
   - API/gRPC contracts
   - Database schema
   - Configuration options
   - Example requests
   - Known limitations

3. Add to `docker-compose.yml` with:
   - Build context and Dockerfile
   - Port mappings
   - Environment variables
   - Health check
   - Dependencies
   - Network and volumes

4. Wire into GraphQL gateway if needed:
   ```go
   // Add client in services/graphql-gateway/client/my-service.client.go
   // Update resolver.go to inject the client
   ```

5. Document in [docs/02-system-design/07-project-structure-lld.md](docs/02-system-design/07-project-structure-lld.md)

### Local Development Best Practices

- **Use `docker compose up`** for the full stack (simulates production)
- **Run `go mod tidy`** after changing dependencies
- **Run `go fmt ./...`** before committing
- **Add tests** for new behavior (unit + integration)
- **Document decisions** in `docs/09-decisions/`

---

## 📚 Learning Resources

### What You'll Learn

1. **Distributed Systems**
   - Event-driven architecture patterns
   - Eventual consistency and CAP theorem tradeoffs
   - Saga patterns for distributed transactions
   - Idempotency, retries, and circuit breakers

2. **Backend Engineering**
   - Clean architecture and hexagonal design
   - Database transactions and outbox pattern
   - gRPC, GraphQL, and API design
   - Authentication, authorization, multi-tenancy

3. **Production Patterns**
   - Observability: tracing, logging, metrics
   - Audit trails and compliance
   - Configuration management
   - Health checks and graceful shutdown

4. **AI Systems Engineering**
   - Retrieval-augmented generation (RAG)
   - Semantic search with embeddings
   - Agent workflows and tool calling
   - AI quality evaluation

### Suggested Reading Order

1. [docs/mainreadme.md](docs/mainreadme.md) — High-level vision
2. [docs/02-system-design/01-system-overview-hld.md](docs/02-system-design/01-system-overview-hld.md) — System architecture
3. [docs/02-system-design/02-architecture-design-hld.md](docs/02-system-design/02-architecture-design-hld.md) — Design deep-dive
4. [docs/09-decisions/](docs/09-decisions/) — Architecture decision records
5. Service READMEs — Start with [services/authentication-service/](services/authentication-service/)

### External References

- [Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html) by Martin Fowler
- [Microservices Patterns](https://microservices.io/patterns/index.html) by Chris Richardson
- [Temporal Workflow Orchestration](https://temporal.io)
- [PostgreSQL Best Practices](https://wiki.postgresql.org/wiki/Performance_Optimization)

---

## 🔐 Security & Compliance

- **Authentication**: JWT access tokens with SHA-256 HMAC, refresh-token rotation
- **Authorization**: Role-based access control (RBAC), tenant isolation
- **Audit**: Complete audit trail of user actions and state changes
- **Data**: PostgreSQL transactions, no unencrypted secrets in code
- **API**: Bearer token validation on every protected endpoint

---

## 📊 Observability

All services are instrumented with:

- **Traces**: OpenTelemetry context propagation across service boundaries
- **Metrics**: Prometheus-compatible metrics (request latency, error rates, queue depth)
- **Logs**: Structured JSON logging with correlation IDs
- **Health**: `/health` endpoints with service readiness checks

View observability data:

```bash
# (Add observability stack to docker-compose.yml)
# Jaeger: http://localhost:16686 (traces)
# Prometheus: http://localhost:9090 (metrics)
# Grafana: http://localhost:3000 (dashboards)
```

---

## 🗺️ Roadmap

### Phase 1: Foundation (Current)
- [x] Multi-service Go architecture with clean boundaries
- [x] PostgreSQL with migrations and transactional consistency
- [x] GraphQL gateway with authentication middleware
- [x] JWT-based multi-tenant authentication and authorization
- [x] gRPC service-to-service communication
- [x] Docker Compose local development stack
- [x] Basic service documentation and architecture decisions

### Phase 2: Event-Driven Architecture
- [ ] Kafka topic-per-service event publishing
- [ ] Transactional outbox pattern in shipment-service
- [ ] Event replay and consumer group support
- [ ] Idempotent event handlers across services

### Phase 3: Workflow Orchestration
- [ ] Temporal workflow definitions for shipment lifecycle
- [ ] Activity implementations for carrier integration
- [ ] Compensation and saga patterns
- [ ] Temporal Web UI integration with logs

### Phase 4: AI Integration
- [ ] Retrieval service with pgVector semantic search
- [ ] Embedding generation for shipments and policies
- [ ] AI gateway for quota and token management
- [ ] Orchestrated AI agents for investigation and decisions
- [ ] LangGraph integration for multi-step reasoning

### Phase 5: Observability & Reliability
- [ ] Distributed tracing with Jaeger
- [ ] Prometheus metrics and custom dashboards
- [ ] Structured logging with correlation IDs
- [ ] Health checks and graceful shutdown
- [ ] Service resilience patterns (circuit breaker, etc.)

### Phase 6: Advanced Features
- [ ] Order service with outbox pattern
- [ ] Billing service ledger and invoice generation
- [ ] Notification service with email/SMS/webhook
- [ ] Rate limiting and quota enforcement
- [ ] API versioning and backward compatibility

### Phase 7: Production Readiness
- [ ] Kubernetes deployment manifests
- [ ] CI/CD pipeline (GitHub Actions)
- [ ] Automated testing suite (unit, integration, E2E)
- [ ] Security audit and penetration testing
- [ ] Performance benchmarks and load testing

---

## 🤝 Contributing

We welcome contributions! Please:

1. **Follow conventional commits**: `feat(service): description`
2. **Keep changes focused**: One feature or fix per PR
3. **Add tests**: New behavior must have unit + integration tests
4. **Document decisions**: Architecture changes deserve a brief ADR
5. **Run the full stack**: `docker compose up --build && go test ./...`

---

## 📄 License

GPLv3 — see the [LICENSE](LICENSE) file for details.

---

## 📞 Contact & Community

- **Issues**: Report bugs or request features via GitHub Issues
- **Discussions**: Ask questions in GitHub Discussions
- **Documentation**: See [docs/](docs/) for architecture and design decisions

---

**Built with ❤️ for learning and production engineering excellence.**
- [Getting Started](#getting-started)
- [API Documentation](#api-documentation)
- [Development and Testing](#development-and-testing)
- [Roadmap Highlights](#roadmap-highlights)
- [Contributing](#contributing)
- [License](#license)

## Architecture Overview

LogiSynapse is designed around clear, loosely coupled planes that separate product concerns, data ownership, orchestration, and intelligence.

### Key Features

- Event-driven architecture with transactional outbox and Kafka for replayable domain events
- Durable workflow orchestration using Temporal for long-running, retryable processes
- AI-native capabilities including retrieval, RAG, typed tools, and audited AI workflows
- Microservices design with clear bounded contexts for order, tracking, billing, notifications, and AI
- Observability and auditability with OpenTelemetry, metrics, traces, and a full audit trail
- Production patterns such as idempotency, retries, compensation, and strong testing boundaries

### Architecture Planes

- Product plane: API gateway, order, tracking, support, dispatch
- Data plane: PostgreSQL, Redis, Kafka, vector DB for durable facts and read models
- Workflow plane: Temporal workers and orchestrators for durable business processes
- Intelligence plane: AI gateway, model gateway, retrieval, tool-service, and eval pipelines

### Core Flow Examples

- Order write: API -> order-service -> Postgres + outbox -> outbox relay -> Kafka -> downstream consumers
- Tracking read: API -> tracking-service -> Redis cache -> Postgres fallback
- AI assistant: ai-gateway -> retrieval -> model -> typed tools -> validated, cited response

## Service Domains

| Service | Responsibility | Primary Integrations |
|---|---|---|
| api-gateway | Public API surface, auth, rate limits | GraphQL / REST |
| order-service | Accept orders, outbox, idempotency | Postgres, Kafka |
| tracking-service | Shipment timeline and read models | Redis, Postgres |
| workflow-service | Temporal workflows and retries | Temporal, Shippo |
| billing-service | Usage aggregation, ledger, invoices | Stripe, Postgres |
| notification-service | Email, SMS, webhooks | RabbitMQ, SQS |
| ai-gateway | Tenant AI requests, quotas, streaming | model-gateway, retrieval |
| retrieval-service | Embeddings and hybrid search | pgvector / Qdrant |

## Project Structure

```text
LogiSynapse/
├── services/
│   ├── shipment-service/
│   ├── workflow-orchestrator/
│   ├── graphql-gateway/
│   ├── communications-service/
│   └── billing-service/
├── shared/
├── docs/
├── graphify-out/
├── docker-compose.yml
└── README.md
```

## Getting Started

### Prerequisites

- Go 1.24+
- Docker and Docker Compose
- Shippo API key for shipment workflows
- Stripe API key for billing workflows

### Quick Start

1. Clone the repository.

	```bash
	git clone https://github.com/Tanmoy095/LogiSynapse.git
	cd LogiSynapse
	```

2. Create a `.env` file in the project root with the required database and integration settings.

3. Start the local stack.

	```bash
	docker compose up --build
	```

4. Open the local tools.

	- GraphQL Playground: http://localhost:8080/
	- Temporal Web UI: http://localhost:8088/
	- RabbitMQ Management: http://localhost:15672/

## API Documentation

### GraphQL API

- Mutations: `createShipment`, `updateShipment`, and related workflow actions
- Queries: `shipments`, `usageSummary`, `invoiceHistory`

### gRPC API

- ShipmentService: `CreateShipment`, `GetShipments`
- BillingService: `GetInvoices`, `CreateInvoice`, `FinalizeInvoice`

### Billing API

- Usage summary by tenant, period, and type
- Invoice history and invoice details
- Ledger views for transaction-level auditability

## Development and Testing

- Run tests:

  ```bash
  go test ./...
  ```

- Database migrations are managed with Goose under service migration folders.
- Regenerate proto and GraphQL artifacts after schema changes.
- Keep health checks and observability hooks in place for each service.
- Use OpenTelemetry for traces and metrics across the stack.

## Roadmap Highlights

- Add Kafka to local compose for end-to-end event testing
- Expand unit, integration, and E2E coverage
- Strengthen logging, tracing, and AI evaluation pipelines
- Harden the gateway with authentication and authorization
- Add CI/CD, Makefile tasks, and deployment manifests

## Contributing

- Follow conventional commits.
- Keep changes small and focused on one responsibility.
- Add tests and documentation for new behavior.
- Document tradeoffs and failure modes for architecture changes.

## License

GPLv3 - see the [LICENSE](LICENSE) file for details.
