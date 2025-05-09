package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

const (
	CONFIG_TON_TESTNET_URL string = "https://ton-blockchain.github.io/testnet-global.config.json"
	CONFIG_TON_MAINNET_URL string = "https://ton.org/global.config.json"
)

var WALLET_SEED []string

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
		return err
	}

	WALLET_SEED = strings.Split(os.Getenv("WALLET_SEED"), " ")

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
