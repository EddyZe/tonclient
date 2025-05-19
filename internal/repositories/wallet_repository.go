package repositories

import (
	"context"
	"fmt"
	"time"
	"tonclient/internal/models"

	"github.com/jmoiron/sqlx"
)

type WalletTonRepository struct {
	db *sqlx.DB
}

func NewWalletRepository(db *sqlx.DB) *WalletTonRepository {
	return &WalletTonRepository{
		db: db,
	}
}

func (r *WalletTonRepository) Save(ton *models.WalletTon) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
        INSERT INTO wallet_ton (user_id, name, addr)
        VALUES (:user_id, :name, :addr)
        RETURNING id
    `

	// Используем NamedQuery вместо BindNamed
	rows, err := r.db.NamedQueryContext(ctx, query, ton)
	if err != nil {
		log.Error("Failed to execute query: ", err)
		return fmt.Errorf("insert error: %w", err)
	}
	defer func(rows *sqlx.Rows) {
		err := rows.Close()
		if err != nil {
			log.Error("Failed to close rows: ", err)
		}
	}(rows)

	// Сканируем возвращённый ID
	if rows.Next() {
		if err := rows.Scan(&ton.Id); err != nil {
			log.Error("Failed to scan ID: ", err)
			return fmt.Errorf("scan error: %w", err)
		}
	}

	return nil
}

func (r *WalletTonRepository) Update(ton *models.WalletTon) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Failed to begin transaction: ", err)
		return fmt.Errorf("begin transaction error: %w", err)
	}
	if _, err := tx.NamedExecContext(ctx, "update wallet_ton set name = :name, addr = :addr, user_id = :user_id where id = :id", ton); err != nil {
		log.Error("Failed to update wallet: ", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Error("Failed to update wallet: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return er
		}
		return err
	}

	return nil
}

func (r *WalletTonRepository) DeleteById(id uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Failed to begin transaction: ", err)
		return fmt.Errorf("begin transaction error: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "delete from wallet_ton where id = :id", id); err != nil {
		log.Error("Failed to delete wallet: ", err)
		return err
	}
	if err := tx.Commit(); err != nil {
		log.Error("Failed to delete wallet: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return er
		}
		return err
	}

	return nil
}

func (r *WalletTonRepository) FindAll() *[]models.WalletTon {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wallets []models.WalletTon
	if err := r.db.SelectContext(ctx, &wallets, "select * from wallet_ton"); err != nil {
		log.Error("Failed to find all wallets: ", err)
		return nil
	}

	return &wallets
}

func (r *WalletTonRepository) FindAllLimit(offset, limit int) *[]models.WalletTon {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var wallets []models.WalletTon

	if err := r.db.SelectContext(ctx, &wallets, "select * from wallet_ton offset $1 limit $2", offset, limit); err != nil {
		log.Error("Failed to find all wallets: ", err)
		return nil
	}
	return &wallets
}

func (r *WalletTonRepository) FindById(id uint64) *models.WalletTon {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wallet models.WalletTon

	if err := r.db.GetContext(ctx, &wallet, "select * from wallet_ton where id = :id", id); err != nil {
		log.Error("Failed to find wallet: ", err)
		return nil
	}

	return &wallet
}

func (r *WalletTonRepository) FindByUserId(userId uint64) *models.WalletTon {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wallet models.WalletTon
	if err := r.db.GetContext(
		ctx,
		&wallet,
		"select w.* from wallet_ton as w join usr as u on w.user_id = u.id where u.id=$1",
		userId,
	); err != nil {
		log.Error("Failed to find wallet: ", err)
		return nil
	}

	return &wallet
}

func (r *WalletTonRepository) FindByAddr(addr string) *models.WalletTon {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var wallet models.WalletTon

	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Failed to begin transaction: ", err)
		return nil
	}
	if err := tx.GetContext(ctx, &wallet, "select * from wallet_ton where addr = $1", addr); err != nil {
		log.Error("Failed to find wallet: ", err)
		return nil
	}
	if err := tx.Commit(); err != nil {
		log.Error("Failed to find wallet: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return nil
		}
		return nil
	}

	return &wallet
}

func (r *WalletTonRepository) CountAll() int {
	var res int
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Failed to begin transaction: ", err)
		return 0
	}
	if err := tx.QueryRowContext(ctx, "select count(*) from wallet_ton").Scan(&res); err != nil {
		log.Error("Failed to count all wallets: ", err)
		return 0
	}

	if err := tx.Commit(); err != nil {
		log.Error("Failed to count all wallets: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return 0
		}
		return 0
	}

	return res
}
