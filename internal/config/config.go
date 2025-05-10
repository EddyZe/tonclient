package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	CONFIG_TON_TESTNET_URL string = "https://ton-blockchain.github.io/testnet-global.config.json"
	CONFIG_TON_MAINNET_URL string = "https://ton.org/global.config.json"
)

var WALLET_SEED []string
var JETTON_WALLET_MAIN_COIN string
var COMMISSION_AMOUNT float64

var log = InitLogger()

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func InitConfig() error {
	err := godotenv.Load()
	if err != nil {
		log.Error("Error loading .env file")
	}

	WALLET_SEED = strings.Split(os.Getenv("WALLET_SEED"), " ")
	COMMISSION_AMOUNT, err = strconv.ParseFloat(os.Getenv("COMMISSION_AMOUNT"), 64)
	if err != nil {
		log.Error("Error parsing COMMISSION_AMOUNT")
		COMMISSION_AMOUNT = 5
	}

	JETTON_WALLET_MAIN_COIN = os.Getenv("JETTON_WALLET_MAIN_COIN")

	return nil
}

func LoadPostgresConfig() *PostgresConfig {
	return &PostgresConfig{
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		DBName:   os.Getenv("DB_NAME"),
	}
}
