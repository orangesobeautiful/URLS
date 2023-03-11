package models

import (
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const RedisIndex = 2

var redisDB *redis.Client

var logger *zap.Logger

func InitModels(inRedis *redis.Client, inLogger *zap.Logger) {
	redisDB = inRedis
	logger = inLogger
}
