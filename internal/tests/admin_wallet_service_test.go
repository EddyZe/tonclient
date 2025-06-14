package tests

import (
	"context"
	"log"
	"strings"
	"testing"
	"time"
	"tonclient/internal/config"
	"tonclient/internal/database"
	"tonclient/internal/models"
	"tonclient/internal/repositories"
	"tonclient/internal/services"
	"tonclient/internal/tonbot"

	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

func TestAdminWalletService_StartSubscribeTransaction(t *testing.T) {
	s := InitAdminService()
	s.StartSubscribeTransaction(make(chan models.SubmitTransaction))

}

func TestGetData_GetDataJetton(t *testing.T) {

	s := InitAdminService()

	info, err := s.DataJetton("EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs")
	if err != nil {
		log.Println("Error getting jetton data", err)
		return
	}
	log.Println(info)
}

func TestSendJetton(t *testing.T) {
	s := InitAdminService()
	_, err := s.SendJetton(
		"EQAJKTfw3qP0OFUba-1l7rtA7_TzXd9Cbm4DjNCaioCdofF_",
		"UQAdpNJR-hZ72cPb70eFuQU3VDx8EcLsOEgm7K0Puh9cHA1d",
		"test",
		50,
		9,
	)
	if err != nil {
		log.Fatalln("Error getting jetton data", err)
		return
	}
}

func TestWallet(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	client := liteclient.NewConnectionPool()
	cfg, err := liteclient.GetConfigFromUrl(ctx, config.CONFIG_TON_MAINNET_URL)
	if err != nil {
		log.Fatalln("get config err: ", err.Error())
	}
	if err := client.AddConnectionsFromConfig(ctx, cfg); err != nil {
		log.Fatal("Failed to add connections to config server:", err)
	}
	api := ton.NewAPIClient(client)
	api.SetTrustedBlockFromConfig(cfg)
	w := wallet.NewSeedWithPassword("qwe123")
	log.Println(w)

	wall, err := wallet.FromSeedWithPassword(api, w, "qwe123", wallet.V4R1)
	if err != nil {
		log.Fatalln("Error getting wallet", err)
	}
	log.Println(wall.WalletAddress().String())
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

func InitAdminService() *services.AdminWalletService {
	seeds := strings.Split("coin often elevator dust photo welcome assault trim bar when fit usage danger candy doctor cage input general start concert vocal dove smile brush", " ")

	db, err := InitDBDefault()
	if err != nil {
		log.Fatal("Failed connect to database: ", err)
	}

	ur := repositories.NewUserRepository(db.Db)
	pr := repositories.NewPoolRepository(db.Db)
	us := services.NewUserService(ur)
	ps := services.NewPoolService(pr, us)
	tr := repositories.NewTelegramRepository(db.Db)
	ts := services.NewTelegramService(tr, us)
	rr := repositories.NewReferralRepository(db.Db)
	rs := services.NewReferalService(rr)
	stS := repositories.NewStakeRepository(db.Db)
	ss := services.NewStakeService(stS, us, ps)
	wr := repositories.NewWalletRepository(db.Db)
	ws := services.NewWalletTonService(us, wr)
	ops := services.NewOperationService(repositories.NewOperationRepository(db.Db))
	s, err := services.NewAdminWalletService(&config.TonClientConfig{
		Seed:                seeds,
		WalletAddr:          "UQD6A01mB8tAKJVekRrMjoA3l188LSCF2zrIHoH94tWhZGAO",
		JettonAdminContract: "UQD6A01mB8tAKJVekRrMjoA3l188LSCF2zrIHoH94tWhZGAO",
	},
		ps,
		ts,
		ss,
		ws,
	)

	tcs := services.NewTonConnectService(redisInit(), s)
	if err != nil {
		log.Fatal("Failed connect to database: ", err)
	}
	bot := tonbot.NewTgBot("8112143412:AAE1EZ3rEmqNx4O41UYch1MtD7NLIxb6-i0", us, ts, ps, s, ss, ws, tcs, ops, rs)
	go func() {
		err := bot.StartBot(make(chan models.SubmitTransaction))
		if err != nil {
			return
		}
	}()

	return s
}
