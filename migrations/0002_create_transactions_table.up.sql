CREATE TABLE transactions (
    id              BIGSERIAL PRIMARY KEY,
    type            TEXT NOT NULL CHECK (type IN ('deposit', 'withdraw', 'transfer')),
    from_account_id BIGINT NULL REFERENCES accounts(id),
    to_account_id   BIGINT NULL REFERENCES accounts(id),
    amount          BIGINT NOT NULL CHECK (amount > 0),
    idempotency_key TEXT NOT NULL UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
