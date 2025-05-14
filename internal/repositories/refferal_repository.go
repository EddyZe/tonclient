package repositories

import (
	"context"
	"time"
	"tonclient/internal/models"

	"github.com/jmoiron/sqlx"
)

type ReferralRepository struct {
	db *sqlx.DB
}

func NewReferralRepository(db *sqlx.DB) *ReferralRepository {
	return &ReferralRepository{
		db: db,
	}
}

func (r *ReferralRepository) Save(ref *models.Referral) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx := r.db.MustBegin()
	query, args, err := tx.BindNamed(
		"insert into referral(referrer_user_id, referral_user_id, first_stake_id, reward_given, reward_amount) values (:referrer_user_id, :referral_user_id, :first_stake_id, :reward_given, :reward_amount) returning id",
		ref,
	)

	if err != nil {
		log.Error("Error inserting referral:", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback:", er)
			return er
		}
		return err
	}

	if err := tx.QueryRowxContext(ctx, query, args...).Scan(&ref.Id); err != nil {
		log.Error("Error inserting referral:", err)
		if er := tx.Rollback(); er != nil {
			log.Error("Failed to rollback:", er)
			return er
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Error("Error committing referral:", err)
		return err
	}
	return nil
}
