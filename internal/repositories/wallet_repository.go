package repositories

import (
	"context"
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

	tx := r.db.MustBegin()
	query, args, err := tx.BindNamed(
		"insert into wallet_ton(name, addr, user_id) values (:name, :addr, :user_id) returning id",
		ton,
	)

	if err != nil {
		if er := tx.Rollback(); er != nil {
			log.Error("Transaction rollback failed: ", er)
			return er
		}
		log.Error("Failed to create new transaction: ", err)
		return err
	}

	if err := tx.QueryRowxContext(ctx, query, args).Scan(ton.Id); err != nil {
		if er := tx.Rollback(); er != nil {
			log.Error("Transaction rollback failed: ", er)
			return er
		}
		log.Error("Failed to save wallet: ", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		if er := tx.Rollback(); er != nil {
			log.Error("Transaction rollback failed: ", er)
			return er
		}
		log.Error("Failed to save wallet: ", err)
		return err
	}

	return nil
}

func (r *WalletTonRepository) Update(ton *models.WalletTon) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx := r.db.MustBegin()
	if _, err := tx.NamedExecContext(ctx, "update wallet_ton set name = :name, addr = :addr, user_id = :user_id where id = :id", ton); err != nil {
		if er := tx.Rollback(); er != nil {
			log.Error("Transaction rollback failed: ", er)
			return er
		}
		log.Error("Failed to update wallet: ", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		if er := tx.Rollback(); er != nil {
			log.Error("Transaction rollback failed: ", er)
			return er
		}
		log.Error("Failed to update wallet: ", err)
		return err
	}

	return nil
}

func (r *WalletTonRepository) DeleteById(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx := r.db.MustBegin()
	tx.MustExecContext(ctx, "delete from wallet_ton where id = :id", id)
	if err := tx.Commit(); err != nil {
		if er := tx.Rollback(); er != nil {
			log.Error("Transaction rollback failed: ", er)
			return er
		}
		log.Error("Failed to delete wallet: ", err)
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

func (r *WalletTonRepository) FindById(id int) *models.WalletTon {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wallet models.WalletTon

	if err := r.db.GetContext(ctx, &wallet, "select * from wallet_ton where id = :id", id); err != nil {
		log.Error("Failed to find wallet: ", err)
		return nil
	}

	return &wallet
}

func (r *WalletTonRepository) FindByUserId(userId int) *models.WalletTon {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wallet models.WalletTon
	if err := r.db.GetContext(
		ctx,
		&wallet,
		"select * from wallet_ton as w join usr as u on w.user_id = u.id where u.id=$1",
		userId,
	); err != nil {
		log.Error("Failed to find wallet: ", err)
		return nil
	}

	return &wallet
}
