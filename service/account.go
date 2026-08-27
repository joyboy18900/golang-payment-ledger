package service

import (
	"context"
	"time"
)

type CreateAccountRequest struct {
	Balance int64 `json:"balance"`
}

type AccountResponse struct {
	ID        int64     `json:"id"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
}

type BalanceResponse struct {
	ID      int64 `json:"id"`
	Balance int64 `json:"balance"`
}

type AccountService interface {
	Create(ctx context.Context, req CreateAccountRequest) (*AccountResponse, error)
	GetBalance(ctx context.Context, id int64) (*BalanceResponse, error)
}
