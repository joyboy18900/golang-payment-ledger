package main_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"golang-payment-ledger/repository"
	"golang-payment-ledger/service"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const testPostgresDSN = "postgres://postgres:postgres@localhost:5432/golang_payment_ledger?sslmode=disable"

func connectTestGormDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(postgres.Open(testPostgresDSN), &gorm.Config{
		TranslateError: true,
		Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
		}),
	})
	if err != nil {
		t.Skipf("skipping integration test: open gorm db: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("skipping integration test: gorm db handle: %v", err)
	}
	if err := sqlDB.PingContext(context.Background()); err != nil {
		t.Skipf("skipping integration test: postgres not reachable: %v", err)
	}

	return db
}

func mustCreateAccount(t *testing.T, db *gorm.DB, balance int64) int64 {
	t.Helper()

	accountRepo := repository.NewAccountRepositoryDB(db)
	account, err := accountRepo.Create(context.Background(), balance)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	return account.ID
}

func TestTransfer_NoDoubleSpendUnderConcurrentWithdrawals(t *testing.T) {
	db := connectTestGormDB(t)

	const (
		startingBalance    = 1000
		withdrawAmount     = 100
		concurrentRequests = 20
		expectedSuccesses  = startingBalance / withdrawAmount
	)

	accountID := mustCreateAccount(t, db, startingBalance)
	svc := service.NewTransferService(repository.NewTransactionRepositoryDB(db))

	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := 0
	failures := 0

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			_, err := svc.Transfer(context.Background(), service.TransferRequest{
				IdempotencyKey: fmt.Sprintf("withdraw-race-%d-%d", accountID, i),
				Type:           "withdraw",
				FromAccountID:  &accountID,
				Amount:         withdrawAmount,
			})

			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				successes++
			} else {
				failures++
			}
		}(i)
	}
	wg.Wait()

	if successes != expectedSuccesses {
		t.Fatalf("successes = %d, want %d", successes, expectedSuccesses)
	}
	if failures != concurrentRequests-expectedSuccesses {
		t.Fatalf("failures = %d, want %d", failures, concurrentRequests-expectedSuccesses)
	}

	accountRepo := repository.NewAccountRepositoryDB(db)
	final, err := accountRepo.FindByID(context.Background(), accountID)
	if err != nil {
		t.Fatalf("find account: %v", err)
	}
	if final.Balance != 0 {
		t.Fatalf("final balance = %d, want 0 (no double-spend, no overdraft)", final.Balance)
	}
}

func TestTransfer_IdempotencyKeyReplayUnderConcurrencyAppliesOnce(t *testing.T) {
	db := connectTestGormDB(t)

	const (
		startingBalance    = 1000
		depositAmount      = 100
		concurrentRequests = 20
	)

	accountID := mustCreateAccount(t, db, startingBalance)
	svc := service.NewTransferService(repository.NewTransactionRepositoryDB(db))

	idempotencyKey := fmt.Sprintf("deposit-replay-%d", accountID)

	var wg sync.WaitGroup
	var mu sync.Mutex
	txnIDs := map[int64]bool{}
	var firstErr error

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			resp, err := svc.Transfer(context.Background(), service.TransferRequest{
				IdempotencyKey: idempotencyKey,
				Type:           "deposit",
				ToAccountID:    &accountID,
				Amount:         depositAmount,
			})

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			txnIDs[resp.ID] = true
		}()
	}
	wg.Wait()

	if firstErr != nil {
		t.Fatalf("expected every replay to succeed, got error: %v", firstErr)
	}
	if len(txnIDs) != 1 {
		t.Fatalf("distinct transaction ids returned = %d, want 1 (same key must return the same transaction)", len(txnIDs))
	}

	accountRepo := repository.NewAccountRepositoryDB(db)
	final, err := accountRepo.FindByID(context.Background(), accountID)
	if err != nil {
		t.Fatalf("find account: %v", err)
	}
	if final.Balance != startingBalance+depositAmount {
		t.Fatalf("final balance = %d, want %d (deposit must apply exactly once, not %d times)",
			final.Balance, startingBalance+depositAmount, concurrentRequests)
	}

	var rowCount int64
	if err := db.Table("transactions").Where("idempotency_key = ?", idempotencyKey).Count(&rowCount).Error; err != nil {
		t.Fatalf("count transactions rows: %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("transactions rows for key = %d, want 1", rowCount)
	}
}
