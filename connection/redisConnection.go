package connection

import (
	"context"
	"main/config"
	"main/utility"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Redis struct {
	Conn *redis.Client
}

var RDB Redis

func InitRedis() {
	// Load logger
	logger := utility.AppLogger.Logger

	rdb := redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.Redis.Addr,
		Password: config.AppConfig.Redis.Password, // no password set
		DB:       config.AppConfig.Redis.DB,       // use default DB
	})
	ctx := context.Background()
	RDB.Conn = rdb

	_, err := RDB.Conn.Ping(ctx).Result()
	if err != nil {
		logger.Error("Fatal Error Connection To Redis", zap.Error(err))
	}

	logger.Info("Connection To Redis Established")
}

func CloseRedis() {
	if RDB.Conn != nil {
		err := RDB.Conn.Close()
		if err != nil {
			return
		}
		utility.AppLogger.Logger.Info("Connection To Redis Closed")
	}
}
