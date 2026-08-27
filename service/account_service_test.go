package service_test

import (
	"context"
	"testing"
	"time"

	"golang-payment-ledger/errs"
	"golang-payment-ledger/mock/mock_repository"
	"golang-payment-ledger/repository"
	"golang-payment-ledger/service"

	"go.uber.org/mock/gomock"
)

func TestAccountService_Create_RejectsNegativeBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_repository.NewMockAccountRepository(ctrl)
	svc := service.NewAccountService(repo)

	_, err := svc.Create(context.Background(), service.CreateAccountRequest{Balance: -1})

	appErr, ok := err.(errs.AppError)
	if !ok || appErr.Code != 422 {
		t.Fatalf("error = %v, want a 422 validation error", err)
	}
}

func TestAccountService_Create_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_repository.NewMockAccountRepository(ctrl)
	created := &repository.Account{ID: 1, Balance: 500, CreatedAt: time.Now()}
	repo.EXPECT().Create(gomock.Any(), int64(500)).Return(created, nil)

	svc := service.NewAccountService(repo)
	resp, err := svc.Create(context.Background(), service.CreateAccountRequest{Balance: 500})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if resp.ID != 1 || resp.Balance != 500 {
		t.Fatalf("resp = %+v, want id=1 balance=500", resp)
	}
}

func TestAccountService_GetBalance_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_repository.NewMockAccountRepository(ctrl)
	repo.EXPECT().FindByID(gomock.Any(), int64(99)).Return(nil, repository.ErrAccountNotFound)

	svc := service.NewAccountService(repo)
	_, err := svc.GetBalance(context.Background(), 99)

	appErr, ok := err.(errs.AppError)
	if !ok || appErr.Code != 404 {
		t.Fatalf("error = %v, want a 404 not-found error", err)
	}
}
