package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type transactionRow struct {
	ID             int64
	Type           string
	FromAccountID  *int64
	ToAccountID    *int64
	Amount         int64
	IdempotencyKey string
	CreatedAt      time.Time
}

func (transactionRow) TableName() string {
	return "transactions"
}

type transactionRepositoryDB struct {
	db *gorm.DB
}

func NewTransactionRepositoryDB(db *gorm.DB) TransactionRepository {
	return transactionRepositoryDB{db: db}
}

func (r transactionRepositoryDB) ApplyTransfer(ctx context.Context, params ApplyTransferParams) (*Transaction, error) {
	existing, err := r.findByIdempotencyKey(ctx, r.db, params.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return matchOrConflict(*existing, params)
	}

	var result *Transaction
	txErr := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		applied, err := r.apply(tx, params)
		if err != nil {
			return err
		}
		result = applied
		return nil
	})
	if txErr == nil {
		return result, nil
	}

	if isUniqueViolation(txErr) {
		racedWith, findErr := r.findByIdempotencyKey(ctx, r.db, params.IdempotencyKey)
		if findErr != nil {
			return nil, findErr
		}
		if racedWith != nil {
			return matchOrConflict(*racedWith, params)
		}
	}

	if isSentinel(txErr) {
		return nil, txErr
	}
	return nil, fmt.Errorf("apply transfer: %w", txErr)
}

func (r transactionRepositoryDB) apply(tx *gorm.DB, params ApplyTransferParams) (*Transaction, error) {
	ids := lockOrder(params.FromAccountID, params.ToAccountID)

	locked := make(map[int64]*accountRow, len(ids))
	for _, id := range ids {
		var row accountRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrAccountNotFound
			}
			return nil, fmt.Errorf("lock account %d: %w", id, err)
		}
		locked[id] = &row
	}

	if params.FromAccountID != nil {
		from := locked[*params.FromAccountID]
		if from.Balance < params.Amount {
			return nil, ErrInsufficientBalance
		}
		from.Balance -= params.Amount
		if err := tx.Model(&accountRow{}).Where("id = ?", from.ID).Update("balance", from.Balance).Error; err != nil {
			return nil, fmt.Errorf("debit account %d: %w", from.ID, err)
		}
	}
	if params.ToAccountID != nil {
		to := locked[*params.ToAccountID]
		to.Balance += params.Amount
		if err := tx.Model(&accountRow{}).Where("id = ?", to.ID).Update("balance", to.Balance).Error; err != nil {
			return nil, fmt.Errorf("credit account %d: %w", to.ID, err)
		}
	}

	row := transactionRow{
		Type:           string(params.Type),
		FromAccountID:  params.FromAccountID,
		ToAccountID:    params.ToAccountID,
		Amount:         params.Amount,
		IdempotencyKey: params.IdempotencyKey,
	}
	if err := tx.Create(&row).Error; err != nil {
		return nil, err
	}

	txn := toTransaction(row)
	return &txn, nil
}

func (r transactionRepositoryDB) findByIdempotencyKey(ctx context.Context, db *gorm.DB, key string) (*Transaction, error) {
	var row transactionRow
	err := db.WithContext(ctx).Where("idempotency_key = ?", key).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find transaction by idempotency key: %w", err)
	}

	txn := toTransaction(row)
	return &txn, nil
}

func matchOrConflict(existing Transaction, params ApplyTransferParams) (*Transaction, error) {
	if existing.Type == params.Type &&
		equalPtr(existing.FromAccountID, params.FromAccountID) &&
		equalPtr(existing.ToAccountID, params.ToAccountID) &&
		existing.Amount == params.Amount {
		return &existing, nil
	}
	return nil, ErrIdempotencyKeyConflict
}

func equalPtr(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func lockOrder(from, to *int64) []int64 {
	switch {
	case from != nil && to != nil:
		if *from < *to {
			return []int64{*from, *to}
		}
		return []int64{*to, *from}
	case from != nil:
		return []int64{*from}
	default:
		return []int64{*to}
	}
}

func isSentinel(err error) bool {
	return errors.Is(err, ErrAccountNotFound) || errors.Is(err, ErrInsufficientBalance) || errors.Is(err, ErrIdempotencyKeyConflict)
}

func isUniqueViolation(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey)
}

func toTransaction(row transactionRow) Transaction {
	return Transaction{
		ID:             row.ID,
		Type:           TransactionType(row.Type),
		FromAccountID:  row.FromAccountID,
		ToAccountID:    row.ToAccountID,
		Amount:         row.Amount,
		IdempotencyKey: row.IdempotencyKey,
		CreatedAt:      row.CreatedAt,
	}
}
