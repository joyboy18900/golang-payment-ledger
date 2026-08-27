package service

import (
	"context"
	"errors"

	"golang-payment-ledger/errs"
	"golang-payment-ledger/logs"
	"golang-payment-ledger/repository"
)

type accountService struct {
	repo repository.AccountRepository
}

func NewAccountService(repo repository.AccountRepository) AccountService {
	return accountService{repo: repo}
}

func (s accountService) Create(ctx context.Context, req CreateAccountRequest) (*AccountResponse, error) {
	if req.Balance < 0 {
		return nil, errs.NewValidationError("balance must not be negative")
	}

	created, err := s.repo.Create(ctx, req.Balance)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	return &AccountResponse{
		ID:        created.ID,
		Balance:   created.Balance,
		CreatedAt: created.CreatedAt,
	}, nil
}

func (s accountService) GetBalance(ctx context.Context, id int64) (*BalanceResponse, error) {
	account, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrAccountNotFound) {
			return nil, errs.NewNotFoundError("account not found")
		}
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	return &BalanceResponse{ID: account.ID, Balance: account.Balance}, nil
}
