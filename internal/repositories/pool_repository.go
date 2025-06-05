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

	tx, err := r.db.Beginx()
	if err != nil {
		log.Error(err)
		return err
	}

	query, args, err := tx.BindNamed(
		`insert into
pool(owner_id, reserve, jetton_wallet, reward, period, is_active, insurance_coating, is_commission_paid, jetton_master, created_at, jetton_name)
values (:owner_id, :reserve, :jetton_wallet, :reward, :period, :is_active, :insurance_coating, :is_commission_paid, :jetton_master, :created_at, :jetton_name)
returning id`,
		pool,
	)

	if err != nil {
		log.Error("Error while creating pool query: ", err)
		return err
	}

	if err := tx.QueryRowContext(ctx, query, args...).Scan(&pool.Id); err != nil {
		log.Error("Error while saving pool: ", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Error("Error while committing transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", er)
			return er
		}
		return err
	}

	return nil
}

func (r *PoolRepository) Update(pool *models.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error while beginning transaction: ", err)
		return err
	}
	if _, err := tx.NamedExecContext(
		ctx,
		"update pool set owner_id = :owner_id, reserve = :reserve, jetton_wallet = :jetton_wallet, reward = :reward, period = :period, is_active = :is_active, is_commission_paid = :is_commission_paid, jetton_master = :jetton_master, created_at = :created_at, jetton_name=:jetton_name where id = :id",
		pool); err != nil {
		log.Error("Error while updating pool: ", err)
	}
	if err := tx.Commit(); err != nil {
		log.Error("Error while committing transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", er)
			return er
		}
		return err
	}
	return nil
}

func (r *PoolRepository) DeleteById(id uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error while beginning transaction: ", err)
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		"delete from pool where id=$1",
		id,
	); err != nil {
		log.Error("Error while deleting pool: ", err)
		return err
	}
	if err := tx.Commit(); err != nil {
		log.Error("Error while committing transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", er)
			return er
		}
		return err
	}

	return nil
}

func (r *PoolRepository) FindById(id uint64) *models.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var pool models.Pool
	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error while beginning transaction: ", err)
		return nil
	}
	if err := tx.GetContext(ctx, &pool, "select * from pool where id=$1", id); err != nil {
		log.Error("Error while getting pool: ", err)
		return nil
	}

	if err := tx.Commit(); err != nil {
		log.Error("Error while committing transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", er)
			return nil
		}
		return nil
	}

	return &pool
}

func (r *PoolRepository) FindAll() *[]models.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var pools []models.Pool
	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error while beginning transaction: ", err)
		return nil
	}
	if err := tx.SelectContext(ctx, &pools, "select * from pool order by created_at desc"); err != nil {
		log.Error("Error while getting pool: ", err)
		return nil
	}

	if err := tx.Commit(); err != nil {
		log.Error("Error while committing transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
		}
		return nil
	}

	return &pools
}

func (r *PoolRepository) FindAllLimit(offset, limit int) *[]models.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var pools []models.Pool

	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error while beginning transaction: ", err)
		return nil
	}
	if err := tx.SelectContext(ctx, &pools, "select * from pool order by created_at desc limit $1 offset $2", limit, offset); err != nil {
		log.Error("Error while getting pool: ", err)
		return nil
	}

	if err := tx.Commit(); err != nil {
		log.Error("Error while committing transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
		}
		return nil
	}

	return &pools
}

func (r *PoolRepository) FindAllByStatus(b bool) *[]models.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var pools []models.Pool
	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error while beginning transaction: ", err)
		return nil
	}
	if err := tx.SelectContext(ctx, &pools, "select * from pool where is_active=$1 order by created_at desc", b); err != nil {
		log.Error("Error while getting pool: ", err)
		return nil
	}

	if err := tx.Commit(); err != nil {
		log.Error("Error while committing transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
		}
		return nil
	}

	return &pools
}

func (r *PoolRepository) FindAllByStatusLimit(b bool, offset, limit int) *[]models.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var pools []models.Pool

	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error while beginning transaction: ", err)
		return nil
	}
	if err := tx.SelectContext(ctx, &pools, "select * from pool where is_active=$1 order by created_at desc limit $2 offset $3", b, limit, offset); err != nil {
		log.Error("Error while getting pool: ", err)
		return nil
	}

	if err := tx.Commit(); err != nil {
		log.Error("Error while committing transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
		}
		return nil
	}

	return &pools
}

func (r *PoolRepository) FindByOwnerId(ownerId uint64) *[]models.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var pools []models.Pool
	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error while beginning transaction: ", err)
		return nil
	}
	if err := tx.SelectContext(ctx, &pools, "select p.* from pool as p join usr as u on p.owner_id = u.id where u.id = $1 order by p.created_at desc", ownerId); err != nil {
		log.Error("Error while getting pool: ", err)
		return nil
	}

	if err := tx.Commit(); err != nil {
		log.Error("Error while committing transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
		}
		return nil
	}

	return &pools
}

func (r *PoolRepository) FindByOwnerIdLimit(ownerId uint64, offset, limit int) *[]models.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var pools []models.Pool
	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error while beginning transaction: ", err)
		return nil
	}
	if err := tx.SelectContext(ctx, &pools, "select p.* from pool as p join usr as u on p.owner_id = u.id where u.id = $1 order by p.created_at desc offset $2 limit $3", ownerId, offset, limit); err != nil {
		log.Error("Error while getting pool: ", err)
		return nil
	}

	if err := tx.Commit(); err != nil {
		log.Error("Error while committing transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return nil
		}
		return nil
	}

	return &pools
}

func (r *PoolRepository) CountAllByStatus(b bool) int {
	var res int
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error while beginning transaction: ", err)
		return 0
	}

	err = tx.QueryRowxContext(ctx, "select count(*) as count from pool where is_active=$1", b).Scan(&res)
	if err != nil {
		log.Error("Error while getting pool: ", err)
		return 0
	}
	if err := tx.Commit(); err != nil {
		log.Error("Error while committing transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return 0
		}
		return 0
	}

	return res
}

func (r *PoolRepository) CountAll() int {
	var res int
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error while beginning transaction: ", err)
		return 0
	}

	err = tx.QueryRowxContext(ctx, "select count(*) as count from pool").Scan(&res)
	if err != nil {
		log.Error("Error while getting pool: ", err)
		return 0
	}
	if err := tx.Commit(); err != nil {
		log.Error("Error while committing transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return 0
		}
		return 0
	}

	return res
}

func (r *PoolRepository) CountUser(userId uint64) int {
	var res int
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error while beginning transaction: ", err)
		return 0
	}
	if err := tx.QueryRowxContext(ctx, "select count(*) from pool p join usr u on u.id=p.owner_id where u.id=$1", userId).
		Scan(&res); err != nil {
		log.Error("Error while getting pool: ", err)
		return 0
	}

	if err := tx.Commit(); err != nil {
		log.Error("Error while committing transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return 0
		}
		return 0
	}

	return res
}
