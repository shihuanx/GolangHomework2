package cache

import "github.com/redis/go-redis/v9"

var RedisClient *redis.Client

// InitRedis 初始化redis
func InitRedis(addr, password string, db int) {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}
