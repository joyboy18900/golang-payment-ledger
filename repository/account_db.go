package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type accountRow struct {
	ID        int64
	Balance   int64
	CreatedAt time.Time
}

func (accountRow) TableName() string {
	return "accounts"
}

type accountRepositoryDB struct {
	db *gorm.DB
}

func NewAccountRepositoryDB(db *gorm.DB) AccountRepository {
	return accountRepositoryDB{db: db}
}

func (r accountRepositoryDB) Create(ctx context.Context, balance int64) (*Account, error) {
	row := accountRow{Balance: balance}

	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}

	account := toAccount(row)
	return &account, nil
}

func (r accountRepositoryDB) FindByID(ctx context.Context, id int64) (*Account, error) {
	var row accountRow
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("find account: %w", err)
	}

	account := toAccount(row)
	return &account, nil
}

func toAccount(row accountRow) Account {
	return Account{
		ID:        row.ID,
		Balance:   row.Balance,
		CreatedAt: row.CreatedAt,
	}
}
