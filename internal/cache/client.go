package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/beastixq/marketplace/internal/config"
)

func NewRedisClient(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:        cfg.Addr,
		Password:    cfg.Password,
		DB:          cfg.DB,
		DialTimeout: cfg.DialTimeout.Std(),
	})

	pingCtx, cancel := context.WithTimeout(ctx, cfg.DialTimeout.Std())
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return client, nil
}
