package repositories

import (
	"context"
	"time"
	"tonclient/internal/models"

	"github.com/jmoiron/sqlx"
)

type OperationRepository struct {
	Db *sqlx.DB
}

func NewOperationRepository(db *sqlx.DB) *OperationRepository {
	return &OperationRepository{
		Db: db,
	}
}

func (r *OperationRepository) Save(op *models.Operation) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := r.Db.Beginx()
	if err != nil {
		log.Error("Error starting transaction", "error", err)
		return err
	}

	query, args, err := tx.BindNamed(
		"insert into operation(user_id, num_operation, name, created_at, description) values (:user_id, :num_operation, :name, :created_at, :description) returning id",
		op,
	)
	if err != nil {
		log.Error("Error creating query", "error", err)
		return err
	}
	if err := tx.QueryRowxContext(ctx, query, args...).Scan(&op.Id); err != nil {
		log.Error("Error creating query", "error", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Error("Error committing query", "error", err)
		if err := tx.Rollback(); err != nil {
			log.Error("Error rolling back", "error", err)
			return err
		}
		return err
	}

	return nil
}

func (r *OperationRepository) FindById(id uint64) (*models.Operation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var op models.Operation
	err := r.Db.GetContext(ctx, &op, "select * from operation where id=$1", id)
	if err != nil {
		return nil, err
	}
	return &op, nil
}

func (r *OperationRepository) FindAll() ([]models.Operation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var ops []models.Operation
	err := r.Db.SelectContext(ctx, &ops, "select * from operation")
	if err != nil {
		return nil, err
	}
	return ops, nil
}

func (r *OperationRepository) FindAllLimit(offset, limit int) ([]models.Operation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var ops []models.Operation
	if err := r.Db.SelectContext(ctx, &ops, "select * from operation limit $1 offset $2", limit, offset); err != nil {
		return nil, err
	}

	return ops, nil
}

func (r *OperationRepository) FindByUserId(userId uint64) ([]models.Operation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var ops []models.Operation
	err := r.Db.SelectContext(ctx, &ops, "select * from operation where user_id=$1", userId)
	if err != nil {
		return nil, err
	}
	return ops, nil
}

func (r *OperationRepository) FindByUserIdLimit(userId uint64, offset, limit int) ([]models.Operation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var ops []models.Operation

	if err := r.Db.SelectContext(ctx, &ops, "select * from operation where user_id=$1 limit $2 offset $3", userId, limit, offset); err != nil {
		return nil, err
	}
	return ops, nil
}

func (r *OperationRepository) CountAll() int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var count int
	if err := r.Db.QueryRowxContext(ctx, "select count(*) from operation").Scan(&count); err != nil {
		return 0
	}

	return count
}

func (r *OperationRepository) CountByUserId(userId uint64) int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var count int
	if err := r.Db.QueryRowxContext(ctx, "select count(*) from operation where user_id=$1", userId).Scan(&count); err != nil {
		return 0
	}
	return count
}
