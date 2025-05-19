package database

import (
	"context"
	"time"
	"tonclient/internal/config"

	"github.com/redis/go-redis/v9"
)

var cts = context.Background()

var Client *redis.Client

type RedisDb struct {
	Cli *redis.Client
}

func NewRedisDb(cfg *config.RedisConfig) (*RedisDb, error) {
	cli := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.Db,
	})

	ctx, cancel := context.WithTimeout(cts, time.Second*10)
	defer cancel()
	if _, err := cli.Ping(ctx).Result(); err != nil {
		log.Error("Error connecting to redis")
		return nil, err
	}
	return &RedisDb{
		Cli: cli,
	}, nil
}
