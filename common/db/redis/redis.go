package redis

import (
	"github.com/redis/go-redis/v9"

	"platform/common/config"
)

func InitRedis(cfg config.RedisConfig) redis.UniversalClient {
	options := &redis.UniversalOptions{
		Addrs:    cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: 100,
	}
	return redis.NewUniversalClient(options)
}
