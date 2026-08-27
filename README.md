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

See `curl/flow.md` for full request/response examples.

## Idempotency and crash safety

- Idempotency is enforced by a unique constraint on
  `transactions.idempotency_key`: the same key with the same request body
  returns the original transaction instead of reapplying it; the same key
  with a different body is rejected with `409`.
- Each transfer runs inside one Postgres transaction that locks the
  involved account row(s) before mutating balances. A crash mid-transfer
  leaves no partial state - the transaction either commits in full or
  never applies at all, so the client can safely retry with the same
  `Idempotency-Key`.
- The Postgres unique constraint is the correctness guarantee, not a cache.
  A Redis (or similar) cache in front of `/transfer` would speed up replay
  lookups, but only as a fast path on top of this constraint, never as a
  replacement for it - Stripe describes their own idempotency handling the
  same way. Not added here: this project stays Postgres-only per its own
  scope, and `golang-redis` already covers Redis patterns separately.

## Tests

```bash
go test ./...
go generate ./...   # regenerate repository mocks
```

- `service/account_service_test.go`, `service/transfer_service_test.go` -
  table-driven validation and repository-error-mapping tests (gomock).
- `ledger_integration_test.go` - against a real Postgres:
  `TestTransfer_NoDoubleSpendUnderConcurrentWithdrawals` fires 20 concurrent
  withdrawals against a balance that can only cover 10 of them, and asserts
  the balance never goes negative;
  `TestTransfer_IdempotencyKeyReplayUnderConcurrencyAppliesOnce` fires 20
  concurrent requests with the same `Idempotency-Key` and asserts the
  deposit applies exactly once.

The integration tests own the database. Run them against a scratch
Postgres, not one holding data you care about.
