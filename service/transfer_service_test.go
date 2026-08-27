package service_test

import (
	"context"
	"testing"

	"golang-payment-ledger/errs"
	"golang-payment-ledger/mock/mock_repository"
	"golang-payment-ledger/repository"
	"golang-payment-ledger/service"

	"go.uber.org/mock/gomock"
)

func ptr(v int64) *int64 { return &v }

func TestTransferService_Transfer_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		req  service.TransferRequest
	}{
		{
			name: "missing idempotency key",
			req:  service.TransferRequest{Type: "deposit", ToAccountID: ptr(1), Amount: 100},
		},
		{
			name: "amount zero",
			req:  service.TransferRequest{IdempotencyKey: "k", Type: "deposit", ToAccountID: ptr(1), Amount: 0},
		},
		{
			name: "amount negative",
			req:  service.TransferRequest{IdempotencyKey: "k", Type: "deposit", ToAccountID: ptr(1), Amount: -1},
		},
		{
			name: "deposit missing to_account_id",
			req:  service.TransferRequest{IdempotencyKey: "k", Type: "deposit", Amount: 100},
		},
		{
			name: "deposit with from_account_id set",
			req:  service.TransferRequest{IdempotencyKey: "k", Type: "deposit", FromAccountID: ptr(1), ToAccountID: ptr(2), Amount: 100},
		},
		{
			name: "withdraw missing from_account_id",
			req:  service.TransferRequest{IdempotencyKey: "k", Type: "withdraw", Amount: 100},
		},
		{
			name: "withdraw with to_account_id set",
			req:  service.TransferRequest{IdempotencyKey: "k", Type: "withdraw", FromAccountID: ptr(1), ToAccountID: ptr(2), Amount: 100},
		},
		{
			name: "transfer missing from_account_id",
			req:  service.TransferRequest{IdempotencyKey: "k", Type: "transfer", ToAccountID: ptr(2), Amount: 100},
		},
		{
			name: "transfer missing to_account_id",
			req:  service.TransferRequest{IdempotencyKey: "k", Type: "transfer", FromAccountID: ptr(1), Amount: 100},
		},
		{
			name: "transfer same account both sides",
			req:  service.TransferRequest{IdempotencyKey: "k", Type: "transfer", FromAccountID: ptr(1), ToAccountID: ptr(1), Amount: 100},
		},
		{
			name: "unknown type",
			req:  service.TransferRequest{IdempotencyKey: "k", Type: "bogus", ToAccountID: ptr(1), Amount: 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mock_repository.NewMockTransactionRepository(ctrl)
			svc := service.NewTransferService(repo)

			_, err := svc.Transfer(context.Background(), tt.req)

			appErr, ok := err.(errs.AppError)
			if !ok {
				t.Fatalf("error = %v (%T), want errs.AppError", err, err)
			}
			if appErr.Code != 400 {
				t.Fatalf("error code = %d, want 400", appErr.Code)
			}
		})
	}
}

func TestTransferService_Transfer_MapsRepositoryErrors(t *testing.T) {
	tests := []struct {
		name     string
		repoErr  error
		wantCode int
	}{
		{"account not found", repository.ErrAccountNotFound, 404},
		{"insufficient balance", repository.ErrInsufficientBalance, 422},
		{"idempotency key conflict", repository.ErrIdempotencyKeyConflict, 409},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mock_repository.NewMockTransactionRepository(ctrl)
			repo.EXPECT().ApplyTransfer(gomock.Any(), gomock.Any()).Return(nil, tt.repoErr)

			svc := service.NewTransferService(repo)
			_, err := svc.Transfer(context.Background(), service.TransferRequest{
				IdempotencyKey: "k", Type: "deposit", ToAccountID: ptr(1), Amount: 100,
			})

			appErr, ok := err.(errs.AppError)
			if !ok {
				t.Fatalf("error = %v (%T), want errs.AppError", err, err)
			}
			if appErr.Code != tt.wantCode {
				t.Fatalf("error code = %d, want %d", appErr.Code, tt.wantCode)
			}
		})
	}
}

func TestTransferService_Transfer_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_repository.NewMockTransactionRepository(ctrl)

	txn := &repository.Transaction{
		ID: 1, Type: repository.TransactionTypeDeposit, ToAccountID: ptr(1), Amount: 100, IdempotencyKey: "k",
	}
	repo.EXPECT().ApplyTransfer(gomock.Any(), repository.ApplyTransferParams{
		IdempotencyKey: "k",
		Type:           repository.TransactionTypeDeposit,
		ToAccountID:    ptr(1),
		Amount:         100,
	}).Return(txn, nil)

	svc := service.NewTransferService(repo)
	resp, err := svc.Transfer(context.Background(), service.TransferRequest{
		IdempotencyKey: "k", Type: "deposit", ToAccountID: ptr(1), Amount: 100,
	})
	if err != nil {
		t.Fatalf("Transfer() error = %v", err)
	}
	if resp.ID != 1 || resp.Amount != 100 {
		t.Fatalf("resp = %+v, want id=1 amount=100", resp)
	}
}
