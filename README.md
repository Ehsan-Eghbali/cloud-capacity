# Cloud Capacity Allocator

Backend core for allocating cloud resource capacity: it tells a customer whether the resources for a new service are available, **holds** them for a short time, and ends every reservation as **committed**, **cancelled** or **expired**.

Built with **Go** and **PostgreSQL 16** in a hexagonal (ports and adapters) layout.

- Design document: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) (English) · [docs/ARCHITECTURE_FA.md](docs/ARCHITECTURE_FA.md) (فارسی) · [PDF](docs/Architecture_and_Design_Document.pdf)
- Assumptions made to complete the problem are listed in section 3 of the design document.

## What it guarantees

- **No overselling.** Checking availability and deducting it is one atomic statement, and a database `CHECK` forbids `allocated + reserved > total`.
- **All or nothing.** A request for several resources (CPU, memory, disk, public IP, ...) holds all of them or none.
- **Safe retries.** The same `Idempotency-Key` always returns the same reservation, also under concurrent retries; commit and cancel can be repeated.
- **Contract limits.** Per-customer caps count usage across all regions.
- **Nothing is left hanging.** Holds have a TTL; a sweeper releases abandoned ones, also after a crash or a database outage.
- **Traceable.** Every state change writes an audit row in the same transaction.

## Quick start

Requires Docker with Compose.

```bash
docker compose up --build -d
curl http://localhost:8080/readyz
# {"success":true,"data":{"status":"ready"}}
```

This starts PostgreSQL, applies the schema, loads demo data and runs the API with the background sweeper.

| If... | Run |
|---|---|
| port 5432 or 8080 is taken | `POSTGRES_HOST_PORT=5433 APP_PORT=8081 docker compose up --build -d` |
| the Go module proxy is unreachable | `GOPROXY=https://goproxy.io,direct docker compose up --build -d` |
| you ran an older version of this project | `docker compose down -v` first (the schema changed) |

Stop and remove everything, including the database volume: `docker compose down -v`.

### Demo data

| | Value |
|---|---|
| Regions | `ir-thr-fr1` (Tehran), `ir-tbz-1` (Tabriz) |
| Resources | `cpu_cores`, `ram_mb`, `disk_gb`, `public_ip`, `gpu_t4` |
| Enterprise customer | `c1000000-0000-0000-0000-000000000001` (for example `gpu_t4`: 4 in total, 2 per request) |
| Community customer | `c2000000-0000-0000-0000-000000000002` (smaller caps, `gpu_t4` not in contract) |

## API walkthrough

Every response has the shape `{"success": bool, "data": ..., "error": "..."}`.

### 1. View capacity

```bash
curl -s http://localhost:8080/api/v1/regions
curl -s http://localhost:8080/api/v1/resources
curl -s http://localhost:8080/api/v1/regions/ir-thr-fr1/capacity
```

```json
{
  "success": true,
  "data": {
    "region_id": "ir-thr-fr1",
    "region_name": "Tehran - Forough Data Center",
    "region_status": "ONLINE",
    "capacities": [
      {"resource_type_id": "gpu_t4", "total_capacity": 8, "allocated_capacity": 2, "reserved_capacity": 0, "available_capacity": 6}
    ]
  }
}
```

This is a snapshot. Only a reservation guarantees capacity.

### 2. Request capacity (hold)

```bash
curl -s -X POST http://localhost:8080/api/v1/reservations \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: req-order-1001" \
  -d '{
    "customer_id": "c1000000-0000-0000-0000-000000000001",
    "region_id": "ir-thr-fr1",
    "metadata": {"target_service": "virtual_machine", "flavor": "g1.standard"},
    "items": [
      {"resource_type_id": "cpu_cores", "amount": 4},
      {"resource_type_id": "ram_mb", "amount": 8192},
      {"resource_type_id": "disk_gb", "amount": 50},
      {"resource_type_id": "public_ip", "amount": 1}
    ]
  }'
```

`201 Created`:

```json
{
  "success": true,
  "data": {
    "id": "e36eedd4-c106-40df-973b-1ed4a4967f9f",
    "customer_id": "c1000000-0000-0000-0000-000000000001",
    "region_id": "ir-thr-fr1",
    "idempotency_key": "req-order-1001",
    "status": "RESERVED",
    "ttl_seconds": 600,
    "expires_at": "2026-11-10T09:45:01.191236Z",
    "metadata": {"flavor": "g1.standard", "target_service": "virtual_machine"},
    "items": [
      {"resource_type_id": "cpu_cores", "amount": 4},
      {"resource_type_id": "disk_gb", "amount": 50},
      {"resource_type_id": "public_ip", "amount": 1},
      {"resource_type_id": "ram_mb", "amount": 8192}
    ],
    "created_at": "2026-11-10T09:35:01.191236Z",
    "updated_at": "2026-11-10T09:35:01.191236Z"
  }
}
```

Sending the **same request with the same key** again returns `200 OK` and the same reservation; nothing is held twice. The same key with a **different** request is refused with `422`.

A request that cannot be satisfied creates nothing and is answered with the reason:

```json
{"success": false, "error": "insufficient capacity: gpu_t4 available 0, requested 1"}
```

| Reason | Status |
|---|---|
| Not enough capacity in the region | `409` |
| Contract total cap exceeded, resource not in contract, above the per-request maximum, below the minimum | `422` |
| Unknown resource type, or not offered in the region | `422` |

### 3. Track a reservation

```bash
curl -s http://localhost:8080/api/v1/reservations/<RESERVATION_ID>
```

### 4. Complete or cancel

```bash
curl -s -X POST http://localhost:8080/api/v1/reservations/<RESERVATION_ID>/commit
curl -s -X POST http://localhost:8080/api/v1/reservations/<RESERVATION_ID>/cancel
```

Both are safe to retry: repeating a commit (or a cancel) that already succeeded returns `200` with the same body. A commit after the hold's deadline returns `409`.

### 5. A customer's previous requests

```bash
curl -s "http://localhost:8080/api/v1/reservations?customer_id=c1000000-0000-0000-0000-000000000001&status=COMMITTED&page=1&page_size=10"
```

Returns the customer's reservations, newest first. `customer_id` is required; `status`, `page` and `page_size` are optional.

### Status codes

| Code | Meaning |
|---|---|
| `200` | Success, or a repeat of an operation already done |
| `201` | A new reservation was created |
| `400` | Invalid input (body, id, key, amount, repeated resource type) |
| `403` | The customer is not active |
| `404` | Customer, region or reservation not found |
| `409` | Capacity shortage; region unavailable; wrong reservation state; deadline passed |
| `422` | Refused by contract or catalog; key reused for a different request |
| `500` | Internal error or database unreachable |

`GET /healthz` is liveness (the process is up). `GET /readyz` is readiness and answers `503` while the database is unreachable.

## Local development

Requires Go 1.27.3 and PostgreSQL 16.

```bash
docker compose up -d postgres            # or point config.yaml at your own PostgreSQL
go run main.go migrate up -c config.yaml
go run main.go seed -c config.yaml
go run main.go server -c config.yaml     # API + sweeper
go run main.go worker -c config.yaml     # sweeper only (for a separate deployment)
```

`make` wraps the common commands: `build`, `run`, `fmt`, `vet`, `up`, `down`, `logs`.

### Configuration

Settings live in `config.yaml`: server address and timeouts, database connection and pool size, hold TTL, sweeper interval and batch size. These environment variables override the file: `SERVER_HOST`, `SERVER_PORT`, `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`.

## Project structure

```text
├── main.go                     # Entrypoint (Cobra CLI)
├── cmd/                        # server, worker, migrate, seed + shared wiring (bootstrap.go)
├── internal/
│   ├── core/                   # No HTTP, no SQL
│   │   ├── entity/             # Reservation, capacity, customer limit rule, domain errors
│   │   ├── port/               # Service and repository interfaces
│   │   └── service/            # Input validation, hold TTL, paging
│   ├── app/
│   │   ├── http/               # Handlers, router, error-to-status mapping
│   │   └── worker/             # Expiry sweeper
│   ├── infra/postgres/         # Repositories: one transaction per operation
│   └── config/                 # YAML configuration with environment overrides
├── migrations/                 # Schema (up / down)
├── seeds/                      # Demo data
├── docs/                       # Design document (EN, FA, PDF) and diagrams
├── Dockerfile
├── docker-compose.yml
└── Makefile
```
