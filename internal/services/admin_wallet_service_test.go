package services

import (
	"strings"
	"testing"
	"tonclient/internal/config"
	"tonclient/internal/database"
	"tonclient/internal/repositories"
)

func TestAdminWalletService_StartSubscribeTransaction(t *testing.T) {
	seeds := strings.Split("coin often elevator dust photo welcome assault trim bar when fit usage danger candy doctor cage input general start concert vocal dove smile brush", " ")

	db, err := InitDBDefault()
	if err != nil {
		log.Fatal("Failed connect to database: ", err)
	}

	ur := repositories.NewUserRepository(db.Db)
	pr := repositories.NewPoolRepository(db.Db)
	us := NewUserService(ur)
	ps := NewPoolService(pr, us)
	tr := repositories.NewTelegramRepository(db.Db)
	ts := NewTelegramService(tr, us)

	s := NewAdminWalletService(&config.TonClientConfig{
		Seed:                seeds,
		WalletAddr:          "UQD6A01mB8tAKJVekRrMjoA3l188LSCF2zrIHoH94tWhZGAO",
		JettonAddr:          "EQDpOMdV41bpJn6UItHl8P-TGo6bjNLRjUF4lMA6WzQc66ol",
		JettonAdminContract: "EQAJKTfw3qP0OFUba-1l7rtA7_TzXd9Cbm4DjNCaioCdofF_",
	},
		ps,
		ts,
	)
	s.StartSubscribeTransaction()

}
func InitDBDefault() (*database.Postgres, error) {
	return database.NewPostgres(&config.PostgresConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "admin",
		DBName:   "toninsurancebot",
	})
}
