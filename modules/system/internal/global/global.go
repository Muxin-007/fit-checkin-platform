package global

import (
	"github.com/redis/go-redis/v9"

	"platform/modules/system/internal/config"
)

var (
	Cfg   *config.Config
	Redis redis.UniversalClient
)
