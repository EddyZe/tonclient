package main

import (
	"errors"
	"log"
	"os"
	"tonclient/internal/config"
	"tonclient/internal/database"
	"tonclient/internal/models"
	"tonclient/internal/repositories"
	"tonclient/internal/services"
	"tonclient/internal/tonbot"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	run()
}

func run() {
	logger := config.InitLogger()
	if err := config.InitConfig(); err != nil {
		logger.Fatalf("Failed to init config: %v", err)
	}

	logger.Infoln("Config initialized")
	psqlConfig := config.LoadPostgresConfig()

	db := connectPostgres(psqlConfig)
	defer func(db *database.Postgres) {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(db)
	logger.Infoln("Database initialized")

	m, err := migrate.New(
		"file://migrations",
		database.GetConUrl(psqlConfig),
	)

	if err != nil {
		log.Fatalf("Ошибка создания миграции: %v", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("Ошибка применения миграции: %v", err)
	}
	log.Println("Миграции успешно применены.")

	redis, err := database.NewRedisDb(config.LoadRedisConfig())
	if err != nil {
		logger.Fatalf("Failed to connect to redis: %v", err)
	}

	log.Println("Init repositories:")
	ur := repositories.NewUserRepository(db.Db)
	log.Println("User repository initialized")
	tr := repositories.NewTelegramRepository(db.Db)
	log.Println("Telegram repository initialized")
	pr := repositories.NewPoolRepository(db.Db)
	log.Println("Pool repository initialized")
	wr := repositories.NewWalletRepository(db.Db)
	log.Println("Wallet repository initialized")
	sr := repositories.NewStakeRepository(db.Db)
	log.Println("Stake repository initialized")
	or := repositories.NewOperationRepository(db.Db)
	log.Println("Operation repository initialized")
	refr := repositories.NewReferralRepository(db.Db)
	log.Println("Referral repository initialized")

	log.Println("Repository initialized")

	log.Println("Init services: ")
	us := services.NewUserService(ur)
	log.Println("User service initialized")
	ts := services.NewTelegramService(tr, us)
	log.Println("Telegram service initialized")
	ps := services.NewPoolService(pr, us)
	log.Println("Pool service initialized")
	ss := services.NewStakeService(sr, us, ps)
	log.Println("Stake service initialized")
	ws := services.NewWalletTonService(us, wr)
	log.Println("WalletTon service initialized")
	opS := services.NewOperationService(or)
	log.Println("Operation service initialized")
	rs := services.NewReferalService(refr)

	aws, err := services.NewAdminWalletService(
		config.LoadTonConfig(),
		ps,
		ts,
		ss,
		ws,
	)
	if err != nil {
		logger.Fatal(err)
	}
	log.Println("AdminWallet service initialized")
	tcs := services.NewTonConnectService(redis.Cli, aws)
	log.Println("Ton connect service initialized")

	log.Println("Service initialized")

	tokenBot := os.Getenv("TELEGRAM_BOT_TOKEN")

	logger.Infoln("Telegram bot starting:", tokenBot)
	tgbot := tonbot.NewTgBot(tokenBot, us, ts, ps, aws, ss, ws, tcs, opS, rs)

	transaction := make(chan models.SubmitTransaction)

	go aws.StartSubscribeTransaction(transaction)

	if err := tgbot.StartBot(transaction); err != nil {
		logger.Fatalf("Failed to start bot: %v", err)
	}

}

func connectPostgres(psqlConfig *config.PostgresConfig) *database.Postgres {
	psql, err := database.NewPostgres(psqlConfig)
	if err != nil {
		log.Fatal("Failed to connect to database")
	}
	if err := psql.Ping(); err != nil {
		log.Fatal("Failed to ping database")
	}

	return psql
}
