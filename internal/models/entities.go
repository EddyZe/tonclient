package models

import (
	"database/sql"
	"time"
)

type User struct {
	Id        sql.NullInt64 `db:"id" json:"id"`
	Username  string        `db:"username" json:"username"`
	CreatedAt time.Time     `db:"created_at" json:"created_at"`
	RefererId sql.NullInt64 `db:"referer_id" json:"referer_id"`
}

type Pool struct {
	Id                     sql.NullInt64 `db:"id" json:"id"`
	OwnerId                uint64        `db:"owner_id" json:"owner_id"`
	Reserve                float64       `db:"reserve" json:"reserve"`
	JettonWallet           string        `db:"jetton_wallet" json:"jetton_wallet"`
	JettonMaster           string        `db:"jetton_master" json:"jetton_master"`
	Reward                 uint          `db:"reward" json:"reward"`
	Period                 uint          `db:"period" json:"period"`
	InsuranceCoating       uint          `db:"insurance_coating" json:"insurance_coating"`
	MaxCompensationPercent uint          `db:"max_compensation_percent" json:"max_compensation_percent"`
	CreatedAt              time.Time     `db:"created_at" json:"created_at"`
	IsActive               bool          `db:"is_active" json:"is_active"`
	IsCommissionPaid       bool          `db:"is_commission_paid" json:"is_commission_paid"`
}

type Stake struct {
	Id                   sql.NullInt64 `db:"id" json:"id"`
	UserId               uint64        `db:"user_id" json:"user_id"`
	PoolId               uint64        `db:"pool_id" json:"pool_id"`
	Amount               float64       `db:"amount" json:"amount"`
	StartDate            time.Time     `db:"start_date" json:"start_date"`
	IsActive             bool          `db:"is_active" json:"is_active"`
	DepositCreationPrice float64       `db:"deposit_creation_price" json:"deposit_creation_price"`
}

type Telegram struct {
	Id         sql.NullInt64 `db:"id" json:"id"`
	UserId     uint64        `db:"user_id" json:"user_id"`
	TelegramId uint64        `db:"telegram_id" json:"telegram_id"`
	Username   string        `db:"username" json:"username"`
}

type WalletTon struct {
	Id     sql.NullInt64 `db:"id" json:"id"`
	UserId uint64        `db:"user_id" json:"user_id"`
	Name   string        `db:"name" json:"name"`
	Addr   string        `db:"addr" json:"addr"`
}

type Referral struct {
	Id             sql.NullInt64 `db:"id" json:"id"`
	ReferrerUserId sql.NullInt64 `db:"referrer_user_id" json:"referrer_user_id"`
	ReferralUserId sql.NullInt64 `db:"referral_user_id" json:"referral_user_id"`
	FirstStakeId   sql.NullInt64 `db:"first_stake_id" json:"first_stake_id"`
	RewardGiven    bool          `db:"reward_given" json:"reward_given"`
	RewardAmount   float64       `db:"reward_amount" json:"reward_amount"`
}
