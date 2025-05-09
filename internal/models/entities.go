package models

import (
	"database/sql"
	"time"
)

type User struct {
	Id        sql.NullInt64 `db:"id" json:"id"`
	Username  string        `db:"username" json:"username"`
	CreatedAt time.Time     `db:"created_at" json:"created_at"`
}

type Pool struct {
	Id           sql.NullInt64 `db:"id" json:"id"`
	OwnerId      uint64        `db:"owner_id" json:"owner_id"`
	Reserve      float64       `db:"reserve" json:"reserve"`
	JettonWallet string        `db:"jetton_wallet" json:"jetton_wallet"`
	Reward       uint          `db:"reward" json:"reward"`
	Period       uint          `db:"period" json:"period"`
	IsActive     bool          `db:"is_active" json:"is_active"`
}

type Stake struct {
	Id     sql.NullInt64 `db:"id" json:"id"`
	UserId uint64        `db:"user_id" json:"user_id"`
	PoolId uint64        `db:"pool_id" json:"pool_id"`
	Amount float64       `db:"amount" json:"amount"`
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
