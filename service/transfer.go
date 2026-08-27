package service

import (
	"context"
	"time"
)

type TransferRequest struct {
	IdempotencyKey string
	Type           string `json:"type"`
	FromAccountID  *int64 `json:"from_account_id"`
	ToAccountID    *int64 `json:"to_account_id"`
	Amount         int64  `json:"amount"`
}

type TransactionResponse struct {
	ID            int64     `json:"id"`
	Type          string    `json:"type"`
	FromAccountID *int64    `json:"from_account_id"`
	ToAccountID   *int64    `json:"to_account_id"`
	Amount        int64     `json:"amount"`
	CreatedAt     time.Time `json:"created_at"`
}

type TransferService interface {
	Transfer(ctx context.Context, req TransferRequest) (*TransactionResponse, error)
}
