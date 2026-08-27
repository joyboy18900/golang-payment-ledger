package repository

import (
	"context"
	"errors"
	"time"
)

var ErrAccountNotFound = errors.New("account not found")

type Account struct {
	ID        int64
	Balance   int64
	CreatedAt time.Time
}

//go:generate go tool mockgen -destination=../mock/mock_repository/account.go golang-payment-ledger/repository AccountRepository
type AccountRepository interface {
	Create(ctx context.Context, balance int64) (*Account, error)
	FindByID(ctx context.Context, id int64) (*Account, error)
}
