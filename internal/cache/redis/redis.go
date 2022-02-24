package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/cache"
	"time"
)

type redisStore struct {
	conf *config.Redis
	rdb  *redis.Client
}

func New(conf *config.Redis) cache.Cache {
	return &redisStore{conf: conf, rdb: redis.NewClient(&redis.Options{
		Addr:     conf.Addr,
		Password: conf.Password,
		DB:       conf.DB,
	})}
}

func (r *redisStore) Get(ctx context.Context, key string) (string, error) {
	return r.rdb.Get(ctx, key).Result()
}

func (r *redisStore) Set(ctx context.Context, key string, value string) error {
	return r.rdb.Set(ctx, key, value, 0).Err()
}

func (r *redisStore) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.rdb.Expire(ctx, key, ttl).Err()
}

func (r *redisStore) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.rdb.TTL(ctx, key).Result()
}

func (r *redisStore) Incr(ctx context.Context, key string) error {
	return r.rdb.Incr(ctx, key).Err()
}
