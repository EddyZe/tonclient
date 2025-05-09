package database

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

var cts = context.Background()

var Client *redis.Client

type RedisDb struct {
	cli *redis.Client
}

func InitRedisCli() (*redis.Client, error) {
	if Client != nil {
		return Client, nil
	}

	url := os.Getenv("REDIS_URL")
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	cli := redis.NewClient(opts)

	Client = cli

	return cli, nil
}
