package repositories

import (
	"context"
	"time"
	"tonclient/internal/config"
	"tonclient/internal/models"

	"github.com/jmoiron/sqlx"
)

var log = config.InitLogger()

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (u *UserRepository) Save(user *models.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx := u.db.MustBegin()
	query, args, err := tx.BindNamed(
		"INSERT into usr (username, created_at, referer_id) values (:username, :created_at, :referer_id) returning id",
		user,
	)
	if err != nil {
		log.Error("Failed insert user ", err)
		er := tx.Rollback()
		if er != nil {
			return er
		}
		return err
	}

	err = tx.QueryRowContext(ctx, query, args...).Scan(&user.Id)
	if err != nil {
		log.Error("Failed save user ", err)
		er := tx.Rollback()
		if er != nil {
			return er
		}
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Error("Failed to commit transaction")
		er := tx.Rollback()
		if er != nil {
			log.Error("Failed to rollback transaction")
			return er
		}
		return err
	}

	return nil
}

func (u *UserRepository) Update(user *models.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx := u.db.MustBegin()
	_, err := tx.NamedExecContext(
		ctx,
		"update usr set username = :username where id=:id",
		user,
	)
	if err != nil {
		log.Error("failed update user: ", err)
		er := tx.Rollback()
		if er != nil {
			log.Error("Failed to rollback transaction")
			return err
		}
		return err
	}
	err = tx.Commit()
	if err != nil {
		log.Error("Failed to commit transaction")
		er := tx.Rollback()
		if er != nil {
			log.Error("Failed to rollback transaction")
			return er
		}
		return err
	}

	return nil
}

func (u *UserRepository) FindUserReferal(refererId uint64) *[]models.User {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var users []models.User

	if err := u.db.SelectContext(ctx, &users, "select * from usr where referer_id = $1", refererId); err != nil {
		log.Error("Failed find user ", err)
		return nil
	}

	return &users
}

func (u *UserRepository) FindById(id uint64) *models.User {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var user models.User

	err := u.db.GetContext(
		ctx,
		&user,
		"select * from usr where id=$1",
		id,
	)

	if err != nil {
		return nil
	}

	return &user
}

func (u *UserRepository) DeleteById(id uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx := u.db.MustBegin()
	tx.MustExecContext(
		ctx,
		"delete from usr where id=$1",
		id,
	)
	err := tx.Commit()
	if err != nil {
		log.Error("Failed to commit transaction")
		err := tx.Rollback()
		if err != nil {
			log.Error("Failed to rollback transaction")
			return err
		}
		return err
	}

	return nil
}

func (u *UserRepository) FindByUsername(username string) *models.User {
	var user models.User

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := u.db.GetContext(
		ctx,
		&user,
		"select * from usr where username=$1",
		username,
	)
	if err != nil {
		log.Error("Failed find user by username ", err)
		return nil
	}

	return &user
}

func (u *UserRepository) FindByTelegramChatId(telegramId uint64) *models.User {
	var user models.User

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := u.db.GetContext(
		ctx,
		&user,
		"select u.* from usr as u join telegram as t on t.user_id=u.id where t.telegram_id=$1",
		telegramId,
	)

	if err != nil {
		log.Error("Failed find user by telegramId ", err)
		return nil
	}

	return &user
}

func (u *UserRepository) FindAll() *[]models.User {
	users := make([]models.User, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := u.db.SelectContext(ctx, &users, "select * from usr"); err != nil {
		log.Error("Failed find all users", err)
		return nil
	}

	return &users
}

func (u *UserRepository) FindAllLimit(offset int, limit int) *[]models.User {
	users := make([]models.User, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := u.db.SelectContext(
		ctx,
		&users,
		"select * from usr offset $1 limit $2",
		offset,
		limit,
	); err != nil {
		log.Error("Failed find all users", err)
		return nil
	}

	return &users
}

func (u *UserRepository) CountAll() int {
	var res int
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx := u.db.MustBegin()
	if err := tx.QueryRowxContext(ctx, "select count(*) from usr").Scan(&res); err != nil {
		log.Error("Failed find all users", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction", er)
		}
		return 0
	}
	if err := tx.Commit(); err != nil {
		log.Error("Failed to commit transaction")
		return 0
	}

	return res
}
