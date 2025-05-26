package repositories

import (
	"context"
	"time"
	"tonclient/internal/models"

	"github.com/jmoiron/sqlx"
)

type StakeRepository struct {
	db *sqlx.DB
}

func NewStakeRepository(db *sqlx.DB) *StakeRepository {
	return &StakeRepository{
		db: db,
	}
}

func (r *StakeRepository) Save(stake *models.Stake) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error starting transaction:", err)
		return err
	}

	query, args, err := tx.BindNamed(
		"insert into stake(user_id, pool_id, amount, start_date, is_active, deposit_creation_price, balance, is_insurance_paid, is_reward_paid) values (:user_id, :pool_id, :amount, :start_date, :is_active, :deposit_creation_price, :balance, :is_insurance_paid, :is_reward_paid) returning id",
		stake,
	)

	if err != nil {
		log.Error("Failed to create new query: ", err)
		return err
	}

	if err := tx.QueryRowxContext(ctx, query, args...).Scan(&stake.Id); err != nil {
		log.Error("Failed to save stake: ", err)
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

func (r *StakeRepository) Update(stake *models.Stake) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error starting transaction:", err)
		return err
	}

	if _, err := tx.NamedExecContext(
		ctx,
		"update stake set user_id = :user_id, pool_id = :pool_id, amount = :amount, start_date=:start_date, is_active = :is_active, deposit_creation_price = :deposit_creation_price, balance = :balance, is_insurance_paid = :is_insurance_paid, is_reward_paid = :is_reward_paid where id=:id",
		stake,
	); err != nil {
		log.Error("Failed to update stake: ", err)
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

func (r *StakeRepository) DeleteById(id int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error starting transaction:", err)
		return err
	}
	if _, err := tx.ExecContext(ctx, "delete from stake where id = $1", id); err != nil {
		log.Error("Failed to delete stake: ", err)
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

func (r *StakeRepository) GetById(id int64) *models.Stake {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var stake models.Stake
	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error starting transaction:", err)
		return nil
	}
	if err := tx.GetContext(ctx, &stake, "select * from stake where id = $1", id); err != nil {
		log.Error("Failed to get stake: ", err)
		return nil
	}
	if err := tx.Commit(); err != nil {
		log.Error("Failed to commit transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return nil
		}
		return nil
	}

	return &stake
}

func (r *StakeRepository) FindAll() *[]models.Stake {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var stakes []models.Stake
	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error starting transaction:", err)
		return nil
	}
	if err := tx.SelectContext(ctx, &stakes, "select * from stake"); err != nil {
		log.Error("Failed to get stake: ", err)
		return nil
	}

	if err := tx.Commit(); err != nil {
		log.Error("Failed to commit transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return nil
		}
		return nil
	}
	return &stakes
}

func (r *StakeRepository) FindAllLimit(offset, limit int) *[]models.Stake {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var stakes []models.Stake
	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error starting transaction:", err)
		return nil
	}
	if err := tx.SelectContext(
		ctx,
		&stakes,
		"select * from stake offset $1 limit $2",
		offset,
		limit); err != nil {
		log.Error("Failed to get stake: ", err)
		return nil
	}

	if err := tx.Commit(); err != nil {
		log.Error("Failed to commit transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return nil
		}
		return nil
	}

	return &stakes
}

func (r *StakeRepository) GetUserStakes(userId uint64) *[]models.Stake {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var stakes []models.Stake
	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error starting transaction:", err)
		return nil
	}

	if err := tx.SelectContext(
		ctx,
		&stakes,
		"select s.* from stake as s join usr as u on s.user_id = u.id where u.id=$1", userId); err != nil {
		log.Error("Failed to get stake: ", err)
		return nil
	}

	if err := tx.Commit(); err != nil {
		log.Error("Failed to commit transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return nil
		}
		return nil
	}

	return &stakes
}

func (r *StakeRepository) GetUserStakesLimit(offset, limit int, userId int64) *[]models.Stake {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var stakes []models.Stake
	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error starting transaction:", err)
		return nil
	}

	if err := tx.SelectContext(
		ctx,
		&stakes,
		"select s.* from stake as s join usr as u on s.user_id = u.id where u.id=$1 offset $2 limit $3",
		userId,
		offset,
		limit); err != nil {
		log.Error("Failed to get stake: ", err)
		return nil
	}

	if err := tx.Commit(); err != nil {
		log.Error("Failed to commit transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return nil
		}
		return nil
	}

	return &stakes
}

func (r *StakeRepository) FindStakesByPoolId(poolId uint64) *[]models.Stake {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var stakes []models.Stake

	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error starting transaction:", err)
		return nil
	}
	if err := tx.SelectContext(
		ctx,
		&stakes,
		"select s.* from stake as s join pool as p on s.pool_id = p.id where p.id=$1",
		poolId); err != nil {
		log.Error("Failed to get stakes: ", err)
		return nil
	}

	if err := tx.Commit(); err != nil {
		log.Error("Failed to commit transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return nil
		}
		return nil
	}

	return &stakes
}

func (r *StakeRepository) CountAll() int {
	var res int
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error starting transaction:", err)
		return 0
	}
	if err := tx.QueryRowContext(ctx, "select count(*) from stake").Scan(&res); err != nil {
		log.Error("Failed to get stake: ", err)
		return 0
	}

	if err := tx.Commit(); err != nil {
		log.Error("Failed to commit transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return 0
		}
		return 0
	}

	return res
}

func (r *StakeRepository) CountUser(userId uint64) int {
	var res int
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error starting transaction:", err)
		return 0
	}
	if err := tx.QueryRowxContext(
		ctx,
		"select count(*) from stake s join usr u on s.user_id = u.id where u.id=$1",
		userId,
	).Scan(&res); err != nil {
		log.Error("Failed to get stake: ", err)
		return 0
	}

	if err := tx.Commit(); err != nil {
		log.Error("Failed to commit transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return 0
		}
		return 0
	}

	return res
}

func (r *StakeRepository) CountUserAndStatusStake(userId uint64, b bool) int {
	var res int
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error starting transaction:", err)
		return 0
	}
	if err := tx.QueryRowxContext(
		ctx,
		"select count(*) from stake s join usr u on s.user_id = u.id where u.id=$1 and s.is_active=$2",
		userId,
		b,
	).Scan(&res); err != nil {
		log.Error("Failed to get stake: ", err)
		return 0
	}

	if err := tx.Commit(); err != nil {
		log.Error("Failed to commit transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return 0
		}
		return 0
	}

	return res
}

func (r *StakeRepository) CountPoolStakes(poolId uint64) int {
	var res int
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error starting transaction:", err)
		return 0
	}
	if err := tx.QueryRowxContext(
		ctx,
		"select count(*) from stake s join pool p on p.id=s.pool_id where p.id=$1",
		poolId).Scan(&res); err != nil {
		log.Error("Failed to get stake: ", err)
		return 0
	}

	if err := tx.Commit(); err != nil {
		log.Error("Failed to commit transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return 0
		}
		return 0
	}

	return res
}

func (r *StakeRepository) GetStakeStatusUser(userId uint64, b bool) *[]models.Stake {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var stakes []models.Stake

	tx, err := r.db.Beginx()
	if err != nil {
		log.Error("Error starting transaction:", err)
		return nil
	}
	if err := tx.SelectContext(
		ctx,
		&stakes,
		"select s.* from stake s join usr u on s.user_id = u.id where u.id=$1 and s.is_active=$2",
		userId,
		b); err != nil {
		log.Error("Failed to get stake: ", err)
	}

	if err := tx.Commit(); err != nil {
		log.Error("Failed to commit transaction: ", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback transaction: ", err)
			return nil
		}
		return nil
	}

	return &stakes
}

func (r *StakeRepository) GetStakesPoolIdAndStatus(poolId uint64, b bool) *[]models.Stake {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stakes := make([]models.Stake, 0)

	if err := r.db.SelectContext(
		ctx,
		&stakes,
		"select s.* from stake s join pool p on p.id=s.pool_id where p.id=$1 and s.is_active=$2",
		poolId,
		b,
	); err != nil {
		return &stakes
	}

	return &stakes
}

func (r *StakeRepository) CountStakesPoolIdAndStatus(poolId uint64, b bool) int {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var count int

	if err := r.db.QueryRowxContext(
		ctx,
		"select count(*) from stake s join pool p on s.pool_id = p.id where p.id = $1 and s.is_active= $2",
		poolId,
		b,
	).Scan(&count); err != nil {
		log.Error("Failed to get stake: ", err)
		return 0
	}

	return count
}

func (r *StakeRepository) FindAllByStatus(b bool) *[]models.Stake {
	res := make([]models.Stake, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := r.db.SelectContext(ctx, &res, "select * from stake where is_active=$1", b); err != nil {
		log.Error("Failed to get stake: ", err)
		return &res
	}

	return &res
}

func (r *StakeRepository) GroupFromPoolNameByUserId(userId uint64) *[]models.GroupElements {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res := make([]models.GroupElements, 0)

	if err := r.db.SelectContext(
		ctx,
		&res,
		"select p.jetton_name as name, count(*) as count from stake s join pool p on s.pool_id = p.id where s.user_id = $1 group by p.jetton_name order by max(p.created_at) desc ", userId); err != nil {
		log.Error("Failed froup stakes: ", err)
	}
	return &res
}

func (r *StakeRepository) GroupFromPoolNameByUserIdLimit(userId uint64, offset, limit int) *[]models.GroupElements {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res := make([]models.GroupElements, 0)

	if err := r.db.SelectContext(
		ctx,
		&res,
		"select p.jetton_name as name, count(*) as count from stake s join pool p on s.pool_id = p.id where s.user_id = $1 group by p.jetton_name order by max(p.created_at) desc limit $2 offset $3", userId, limit, offset); err != nil {
		log.Error("Failed froup stakes: ", err)
	}
	return &res
}

func (r *StakeRepository) FindByJettonNameAndUserId(userId uint64, jettonName string) *[]models.Stake {
	ctx, cacel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cacel()

	res := make([]models.Stake, 0)

	if err := r.db.SelectContext(ctx, &res, "select s.* from stake s join pool p on s.pool_id = p.id where s.user_id = $1 and p.jetton_name = $2", userId, jettonName); err != nil {
		log.Error("Failed to get stake: ", err)
	}

	return &res
}

func (r *StakeRepository) FindByJettonNameAndUserIdLimit(userId uint64, jettonName string, offset, limit int) *[]models.Stake {
	ctx, cacel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cacel()

	res := make([]models.Stake, 0)

	if err := r.db.SelectContext(ctx, &res, "select s.* from stake s join pool p on s.pool_id = p.id where s.user_id = $1 and p.jetton_name = $2 offset $3 limit $4", userId, jettonName, offset, limit); err != nil {
		log.Error("Failed to get stake: ", err)
	}

	return &res
}

func (r *StakeRepository) CountGroupsStakesUserId(userId uint64) int {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res := 0
	if err := r.db.QueryRowxContext(
		ctx,
		"select count(distinct p.jetton_name) from stake s join pool p on s.pool_id = p.id where s.user_id=$1", userId).Scan(&res); err != nil {
		log.Error("Failed to get stake: ", err)
	}

	return res
}

func (r *StakeRepository) CountGroupsStakesByUserIdAndJettonName(userId uint64, jettonName string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res := 0

	if err := r.db.QueryRowxContext(
		ctx,
		"select count(*) from stake s join pool p on s.pool_id = p.id where s.user_id=$1 and jetton_name = $2", userId, jettonName).Scan(&res); err != nil {
		log.Error("Failed to get stake: ", err)
	}

	return res
}
