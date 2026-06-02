package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	svc "github.com/beastixq/marketplace/internal/service"
)

type TokenBlocklist struct {
	rdb *redis.Client
}

var _ svc.TokenBlocklist = (*TokenBlocklist)(nil)

func NewTokenBlocklist(rdb *redis.Client) *TokenBlocklist {
	return &TokenBlocklist{rdb: rdb}
}

func (b *TokenBlocklist) Add(ctx context.Context, jti string, exp time.Duration) error {
	return b.rdb.Set(ctx, SessionKey(jti), "1", exp).Err()
}

func (b *TokenBlocklist) Contains(ctx context.Context, jti string) (bool, error) {
	err := b.rdb.Get(ctx, SessionKey(jti)).Err()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
