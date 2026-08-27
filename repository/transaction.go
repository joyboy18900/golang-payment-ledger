package repository

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInsufficientBalance    = errors.New("insufficient balance")
	ErrIdempotencyKeyConflict = errors.New("idempotency key already used with a different request")
)

type TransactionType string

const (
	TransactionTypeDeposit  TransactionType = "deposit"
	TransactionTypeWithdraw TransactionType = "withdraw"
	TransactionTypeTransfer TransactionType = "transfer"
)

type Transaction struct {
	ID             int64
	Type           TransactionType
	FromAccountID  *int64
	ToAccountID    *int64
	Amount         int64
	IdempotencyKey string
	CreatedAt      time.Time
}

type ApplyTransferParams struct {
	IdempotencyKey string
	Type           TransactionType
	FromAccountID  *int64
	ToAccountID    *int64
	Amount         int64
}

//go:generate go tool mockgen -destination=../mock/mock_repository/transaction.go golang-payment-ledger/repository TransactionRepository
type TransactionRepository interface {
	ApplyTransfer(ctx context.Context, params ApplyTransferParams) (*Transaction, error)
}
