# golang-payment-ledger

Fiber + GORM + Postgres ledger: accounts with a materialized balance, an
immutable `transactions` audit log, and an idempotent `/transfer` endpoint
covering deposit, withdraw, and account-to-account transfer.

## Run

```bash
docker-compose up
curl -X POST http://localhost:8080/accounts \
  -H "Content-Type: application/json" \
  -d '{"balance":1000}'
```

The app runs pending migrations on startup, then serves on `:8080`. See
`curl/flow.md` for a full walkthrough.

## Endpoints

- `POST /accounts`
- `GET /accounts/:id/balance`
- `POST /transfer` (header `Idempotency-Key`, required)
- Idempotency is enforced by a unique constraint on `idempotency_key`; a
  crash mid-transfer leaves no partial state either way, since the whole
  transfer runs inside one atomic Postgres transaction.
- Redis could sit in front of this instead of or alongside Postgres. It
  would help under heavy duplicate-request traffic, or when several app
  instances need a shared fast-path lookup before hitting the database.
  Either way, the unique constraint stays the source of truth; a cache
  only speeds up the common case. This project stays Postgres-only.

See `curl/flow.md` for full request/response examples.

## Tests

```bash
go test ./...
go generate ./...   # regenerate repository mocks
```

- `service/*_test.go`: table-driven validation and error-mapping tests.
- `ledger_integration_test.go`: concurrency tests against a real Postgres,
  covering no double-spend under parallel withdrawals and idempotency
  replay applying exactly once.
