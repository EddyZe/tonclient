package repositories

import (
	"context"
	"time"
	"tonclient/internal/models"

	"github.com/jmoiron/sqlx"
)

type PoolRepository struct {
	db *sqlx.DB
}

func NewPoolRepository(db *sqlx.DB) *PoolRepository {
	return &PoolRepository{
		db: db,
	}
}

func (r *PoolRepository) Save(pool *models.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx := r.db.MustBegin()

	query, args, err := tx.BindNamed(
		`insert into
pool(owner_id, reserve, jetton_wallet, reward, period, is_active, insurance_coating, is_commission_paid, jetton_master, max_compensation_percent, created_at)
values (:owner_id, :reserve, :jetton_wallet, :reward, :period, :is_active, :insurance_coating, :is_commission_paid, :jetton_master, :max_compensation_percent, :created_at)
returning id`,
		pool,
	)

	if err != nil {
		log.Error("Error while creating pool query: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Error while rolling back transaction: ", er)
			return er
		}
		return err
	}

	if err := tx.QueryRowContext(ctx, query, args...).Scan(&pool.Id); err != nil {
		log.Error("Error while saving pool: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Error while rolling back transaction: ", er)
			return er
		}
	}

	if err := tx.Commit(); err != nil {
		log.Error("Error while committing transaction: ", err)
		return err
	}

	return nil
}

func (r *PoolRepository) Update(pool *models.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx := r.db.MustBegin()
	if _, err := tx.NamedExecContext(
		ctx,
		"update pool set owner_id = :owner_id, reserve = :reserve, jetton_wallet = :jetton_wallet, reward = :reward, period = :period, is_active = :is_active, is_commission_paid = :is_commission_paid, jetton_master = :jetton_master, max_compensation_percent = :max_compensation_percent, created_at = :created_at where id = :id",
		pool); err != nil {
		log.Error("Error while updating pool: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Error while rolling back transaction: ", er)
			return er
		}
	}
	if err := tx.Commit(); err != nil {
		log.Error("Error while committing transaction: ", err)
		return err
	}
	return nil
}

func (r *PoolRepository) DeleteById(id uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx := r.db.MustBegin()
	tx.MustExecContext(
		ctx,
		"delete from pool where id=$1",
		id,
	)
	if err := tx.Commit(); err != nil {
		log.Error("Error while committing transaction: ", err)
		return err
	}

	return nil
}

func (r *PoolRepository) FindById(id uint64) *models.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var pool models.Pool
	tx := r.db.MustBegin()
	if err := tx.GetContext(ctx, &pool, "select * from pool where id=$1", id); err != nil {
		log.Error("Error while getting pool: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Error while rolling back transaction: ", er)
		}
		return nil
	}

	return &pool
}

func (r *PoolRepository) FindAll() *[]models.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var pools []models.Pool
	tx := r.db.MustBegin()
	if err := tx.SelectContext(ctx, &pools, "select * from pool"); err != nil {
		log.Error("Error while getting pool: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Error while rolling back transaction: ", er)
		}
		return nil
	}

	return &pools
}

func (r *PoolRepository) FindAllLimit(offset, limit int) *[]models.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var pools []models.Pool

	tx := r.db.MustBegin()
	if err := tx.SelectContext(ctx, &pools, "select * from pool limit $1 offset $2", limit, offset); err != nil {
		log.Error("Error while getting pool: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Error while rolling back transaction: ", er)
		}
		return nil
	}

	return &pools
}

func (r *PoolRepository) FindByOwnerId(ownerId int) *[]models.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var pools []models.Pool
	tx := r.db.MustBegin()
	if err := tx.SelectContext(ctx, &pools, "select p.* from pool as p join usr as u on p.owner_id = u.id where u.id = $1", ownerId); err != nil {
		log.Error("Error while getting pool: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Error while rolling back transaction: ", er)
		}
		return nil
	}

	return &pools
}

func (r *PoolRepository) FindByOwnerIdLimit(ownerId, offset, limit int) *[]models.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var pools []models.Pool
	tx := r.db.MustBegin()
	if err := tx.SelectContext(ctx, &pools, "select p.* from pool as p join usr as u on p.owner_id = u.id where u.id = $1 offset $2 limit $3", ownerId, offset, limit); err != nil {
		log.Error("Error while getting pool: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Error while rolling back transaction: ", er)
		}
		return nil
	}

	return &pools
}
