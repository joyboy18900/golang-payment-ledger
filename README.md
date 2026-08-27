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
- A Redis cache in front would only speed up replay lookups, never replace
  this constraint (Stripe layers theirs the same way). This project stays
  Postgres-only.

See `curl/flow.md` for full request/response examples.

## Tests

```bash
go test ./...
go generate ./...   # regenerate repository mocks
```

- `service/*_test.go` - table-driven validation and error-mapping tests.
- `ledger_integration_test.go` - concurrency tests against a real Postgres:
  no double-spend under parallel withdrawals, idempotency replay applies
  once.

Integration tests own the database. Use a scratch Postgres.
