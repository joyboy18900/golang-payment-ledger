# Manual test flow

Walkthrough for exercising the ledger API by hand.

## Start

```bash
docker compose up -d --build
docker compose ps
docker compose logs app --tail 20   # should show "server started on port 8080"
```

## 1. Create two accounts

```bash
curl -X POST http://localhost:8080/accounts \
  -H "Content-Type: application/json" \
  -d '{"balance":1000}'
```

```json
{"code":201,"message":"account created","data":{"id":1,"balance":1000,"created_at":"2026-08-27T09:26:27.705899046Z"}}
```

```bash
curl -X POST http://localhost:8080/accounts \
  -H "Content-Type: application/json" \
  -d '{"balance":0}'
```

```json
{"code":201,"message":"account created","data":{"id":2,"balance":0,"created_at":"2026-08-27T09:26:27.720853088Z"}}
```

## 2. Transfer between accounts

```bash
curl -X POST http://localhost:8080/transfer \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: txn-001" \
  -d '{"type":"transfer","from_account_id":1,"to_account_id":2,"amount":300}'
```

```json
{"code":200,"message":"transfer applied","data":{"id":1,"type":"transfer","from_account_id":1,"to_account_id":2,"amount":300,"created_at":"2026-08-27T09:26:27.73549588Z"}}
```

## 3. Replay the same key - not reapplied

```bash
curl -X POST http://localhost:8080/transfer \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: txn-001" \
  -d '{"type":"transfer","from_account_id":1,"to_account_id":2,"amount":300}'
```

Same `id`, same response - the transfer did not apply a second time.

```json
{"code":200,"message":"transfer applied","data":{"id":1,"type":"transfer","from_account_id":1,"to_account_id":2,"amount":300,"created_at":"2026-08-27T09:26:27.735495Z"}}
```

## 4. Reuse the same key with a different body - rejected

```bash
curl -X POST http://localhost:8080/transfer \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: txn-001" \
  -d '{"type":"transfer","from_account_id":1,"to_account_id":2,"amount":999}'
```

```json
{"code":409,"message":"idempotency key already used with a different request","data":null}
```

## 5. Deposit and withdraw

```bash
curl -X POST http://localhost:8080/transfer \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: txn-002" \
  -d '{"type":"deposit","to_account_id":2,"amount":150}'

curl -X POST http://localhost:8080/transfer \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: txn-003" \
  -d '{"type":"withdraw","from_account_id":1,"amount":200}'
```

```json
{"code":200,"message":"transfer applied","data":{"id":2,"type":"deposit","from_account_id":null,"to_account_id":2,"amount":150,"created_at":"2026-08-27T09:26:27.767016838Z"}}
{"code":200,"message":"transfer applied","data":{"id":3,"type":"withdraw","from_account_id":1,"to_account_id":null,"amount":200,"created_at":"2026-08-27T09:26:27.77937088Z"}}
```

## 6. Withdraw past the available balance - rejected

```bash
curl -X POST http://localhost:8080/transfer \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: txn-004" \
  -d '{"type":"withdraw","from_account_id":1,"amount":99999}'
```

```json
{"code":422,"message":"insufficient balance","data":null}
```

## 7. Missing Idempotency-Key header - rejected

```bash
curl -X POST http://localhost:8080/transfer \
  -H "Content-Type: application/json" \
  -d '{"type":"deposit","to_account_id":1,"amount":100}'
```

```json
{"code":400,"message":"Idempotency-Key header is required","data":null}
```

## 8. Read balances

```bash
curl http://localhost:8080/accounts/1/balance
curl http://localhost:8080/accounts/2/balance
```

```json
{"code":200,"message":"balance retrieved","data":{"id":1,"balance":500}}
{"code":200,"message":"balance retrieved","data":{"id":2,"balance":450}}
```

`1000 - 300 - 200 = 500`, `0 + 300 + 150 = 450` - matches the three
transfers above.

```bash
curl http://localhost:8080/accounts/999/balance
```

```json
{"code":404,"message":"account not found","data":null}
```
