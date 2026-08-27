package service

import (
	"context"
	"errors"

	"golang-payment-ledger/errs"
	"golang-payment-ledger/logs"
	"golang-payment-ledger/repository"
)

type transferService struct {
	repo repository.TransactionRepository
}

func NewTransferService(repo repository.TransactionRepository) TransferService {
	return transferService{repo: repo}
}

func (s transferService) Transfer(ctx context.Context, req TransferRequest) (*TransactionResponse, error) {
	txnType, err := validateTransferRequest(req)
	if err != nil {
		return nil, err
	}

	txn, err := s.repo.ApplyTransfer(ctx, repository.ApplyTransferParams{
		IdempotencyKey: req.IdempotencyKey,
		Type:           txnType,
		FromAccountID:  req.FromAccountID,
		ToAccountID:    req.ToAccountID,
		Amount:         req.Amount,
	})
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrAccountNotFound):
			return nil, errs.NewNotFoundError("account not found")
		case errors.Is(err, repository.ErrInsufficientBalance):
			return nil, errs.NewValidationError("insufficient balance")
		case errors.Is(err, repository.ErrIdempotencyKeyConflict):
			return nil, errs.NewConflictError("idempotency key already used with a different request")
		default:
			logs.Error(err)
			return nil, errs.NewUnexpectedError()
		}
	}

	return &TransactionResponse{
		ID:            txn.ID,
		Type:          string(txn.Type),
		FromAccountID: txn.FromAccountID,
		ToAccountID:   txn.ToAccountID,
		Amount:        txn.Amount,
		CreatedAt:     txn.CreatedAt,
	}, nil
}

func validateTransferRequest(req TransferRequest) (repository.TransactionType, error) {
	if req.IdempotencyKey == "" {
		return "", errs.NewBadRequestError("Idempotency-Key header is required")
	}
	if req.Amount <= 0 {
		return "", errs.NewBadRequestError("amount must be greater than zero")
	}

	switch repository.TransactionType(req.Type) {
	case repository.TransactionTypeDeposit:
		if req.ToAccountID == nil || req.FromAccountID != nil {
			return "", errs.NewBadRequestError("deposit requires to_account_id and no from_account_id")
		}
	case repository.TransactionTypeWithdraw:
		if req.FromAccountID == nil || req.ToAccountID != nil {
			return "", errs.NewBadRequestError("withdraw requires from_account_id and no to_account_id")
		}
	case repository.TransactionTypeTransfer:
		if req.FromAccountID == nil || req.ToAccountID == nil {
			return "", errs.NewBadRequestError("transfer requires both from_account_id and to_account_id")
		}
		if *req.FromAccountID == *req.ToAccountID {
			return "", errs.NewBadRequestError("transfer requires from_account_id and to_account_id to differ")
		}
	default:
		return "", errs.NewBadRequestError("type must be one of deposit, withdraw, transfer")
	}

	return repository.TransactionType(req.Type), nil
}
