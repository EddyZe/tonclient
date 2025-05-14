package repositories

import (
	"context"
	"time"
	"tonclient/internal/models"

	"github.com/jmoiron/sqlx"
)

type TelegramRepository struct {
	db *sqlx.DB
}

func NewTelegramRepository(db *sqlx.DB) *TelegramRepository {
	return &TelegramRepository{
		db: db,
	}
}

func (r *TelegramRepository) Save(telegram *models.Telegram) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx := r.db.MustBegin()
	query, args, err := tx.BindNamed(
		"insert into telegram(username, telegram_id, user_id) values(:username, :telegram_id, :user_id) returning id",
		telegram,
	)

	if err != nil {
		log.Error("Failed to create new query: ", err)
		er := tx.Rollback()
		if er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return er
		}
		return err
	}

	err = tx.QueryRowxContext(
		ctx,
		query,
		args...,
	).Scan(telegram.Id)
	if err != nil {
		log.Error("Failed to get result: ", err)
		er := tx.Rollback()
		if er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return er
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Error("Failed to commit transaction: ", err)
		return err
	}

	return nil
}

func (r *TelegramRepository) Update(telegram *models.Telegram) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx := r.db.MustBegin()
	if _, err := tx.NamedExecContext(
		ctx,
		"update telegram set username = :username, user_id = :user_id, telegram_id = :telegram_id where id=:id",
		telegram); err != nil {
		log.Error("Failed to update telegram: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return er
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Error("Failed to commit transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return er
		}
		return err
	}
	return nil
}

func (r *TelegramRepository) DeleteById(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx := r.db.MustBegin()
	tx.MustExecContext(ctx, "delete from telegram where id=$1", id)
	if err := tx.Commit(); err != nil {
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return er
		}
		log.Error("Failed to commit transaction: ", err)
		return err
	}

	return nil
}

func (r *TelegramRepository) FindById(id uint64) *models.Telegram {
	var telegram models.Telegram

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := r.db.GetContext(ctx, &telegram, "select * from telegram where id=$1", id); err != nil {
		log.Error("Failed to get result: ", err)
		return nil
	}

	return &telegram
}

func (r *TelegramRepository) FindByTelegramId(telegramId uint64) *models.Telegram {
	var telegram models.Telegram

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := r.db.GetContext(ctx, &telegram, "select * from telegram where id=$1", telegramId); err != nil {
		log.Error("Failed to get result: ", err)
		return nil
	}

	return &telegram
}

func (r *TelegramRepository) FindAll() *[]models.Telegram {
	var telegrams []models.Telegram
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := r.db.SelectContext(ctx, &telegrams, "select * from telegram"); err != nil {
		return nil
	}

	return &telegrams
}

func (r *TelegramRepository) FindAllLimit(offset, limit int) *[]models.Telegram {
	var telegrams []models.Telegram
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := r.db.SelectContext(
		ctx,
		&telegrams,
		"select * from telegram offset $1 limit $2",
		offset,
		limit,
	); err != nil {
		return nil
	}

	return &telegrams
}

func (r *TelegramRepository) FindByUserId(userId uint64) *models.Telegram {
	var telegram models.Telegram
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx := r.db.MustBegin()
	if err := tx.GetContext(
		ctx,
		&telegram,
		"select t.* from telegram as t join usr as u on t.user_id = u.id where u.id = $1",
		userId); err != nil {
		if er := tx.Rollback(); er != nil {
			log.Error("Failed RollBack transaction", er)
			return nil
		}
		log.Error("Failed find telegram", err)
		return nil
	}

	if err := tx.Commit(); err != nil {
		if er := tx.Rollback(); er != nil {
			log.Error("Failed rollback transaction", er)
			return nil
		}
		log.Error("Failed commiting transaction", err)
		return nil
	}

	return &telegram
}
