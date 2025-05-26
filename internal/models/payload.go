package models

type NotificationStake struct {
	Stake *Stake
	Msg   string
}

type GroupElements struct {
	Name  string `db:"name" json:"name"`
	Count int    `db:"count" json:"count"`
}
