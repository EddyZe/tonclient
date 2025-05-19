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

type RedisConfig struct {
	Addr     string
	Password string
	Db       int
}

type TonClientConfig struct {
	Seed                []string
	WalletAddr          string
	JettonAddr          string
	JettonAdminContract string
}

func InitConfig() error {
	err := godotenv.Load()
	if err != nil {
		log.Error("Error loading .env file")
	}

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

func LoadTonConfig() *TonClientConfig {
	seed := strings.Split(os.Getenv("WALLET_SEED"), " ")
	walletAddr := os.Getenv("WALLET_ADDR")
	jettonAddr := os.Getenv("JETTON_WALLET_MAIN_COIN")
	contract := os.Getenv("JETTON_CONTRACT_ADMIN_JETTON")

	return &TonClientConfig{
		Seed:                seed,
		WalletAddr:          walletAddr,
		JettonAddr:          jettonAddr,
		JettonAdminContract: contract,
	}
}

func LoadRedisConfig() *RedisConfig {
	addr := os.Getenv("REDIS_ADDR")
	password := os.Getenv("REDIS_PASSWORD")
	b := os.Getenv("REDIS_DB")
	convDb, err := strconv.Atoi(b)
	if err != nil {
		log.Error("Error parsing REDIS_DB")
	}
	return &RedisConfig{
		Addr:     addr,
		Password: password,
		Db:       convDb,
	}
}
