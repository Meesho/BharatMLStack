package infra

import (
	"context"
	"sync"

	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
	"github.com/redis/go-redis/v9"
)

var (
	redisClient *redis.Client
	redisOnce   sync.Once
)

// InitRedis initializes the single Redis client from app config.
func InitRedis() {
	redisOnce.Do(func() {
		cfg := structs.GetAppConfig().Configs
		addr := cfg.RedisAddr
		if addr == "" || cfg.RedisDB == 0 || cfg.RedisPassword == "" {
			panic("redis addr, db, or password is not set")
		}
		redisClient = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			panic("redis ping failed: " + err.Error())
		}
	})
}

// Redis returns the shared Redis client. InitRedis must be called first.
func GetRedisClient() *redis.Client {
	return redisClient
}
