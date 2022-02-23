package store

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mylxsw/secure-proxy/config"
)

type Manager struct {
	conf *config.Redis
	rdb  *redis.Client
}

func New(conf *config.Redis) *Manager {
	return &Manager{conf: conf, rdb: redis.NewClient(&redis.Options{
		Addr:     conf.Addr,
		Password: conf.Password,
		DB:       conf.DB,
	})}
}

const (
	UserLoginRateKey = "secure-proxy:user:%s:%s:login-count"
)

func (sm *Manager) UserCanLogin(userType string, account string) error {
	attempt, err := sm.rdb.Get(context.TODO(), fmt.Sprintf(UserLoginRateKey, userType, account)).Int()
	if err != nil && err != redis.Nil {
		return err
	}

	if err == redis.Nil {
		return nil
	}

	if attempt > 5 {
		ttl := sm.rdb.TTL(context.TODO(), fmt.Sprintf(UserLoginRateKey, userType, account)).Val()
		return fmt.Errorf("尝试次数过多，请 %s 后再试", ttl)
	}

	return nil
}

func (sm *Manager) UserLoginAttempt(userType string, account string) error {
	if err := sm.rdb.Incr(context.TODO(), fmt.Sprintf(UserLoginRateKey, userType, account)).Err(); err != nil {
		return err
	}

	return sm.rdb.Expire(context.TODO(), fmt.Sprintf(UserLoginRateKey, userType, account), 10*time.Minute).Err()
}
