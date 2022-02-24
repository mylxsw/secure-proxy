package store

import (
	"context"
	"fmt"
	"github.com/mylxsw/secure-proxy/internal/cache"
	"strconv"
	"time"
)

type Store interface {
	UserCanLogin(userType string, account string) error
	UserLoginAttempt(userType string, account string) error
}

type defaultStore struct {
	cacheDriver cache.Cache
}

func NewDefaultStore(cacheDriver cache.Cache) Store {
	return &defaultStore{cacheDriver: cacheDriver}
}

const (
	UserLoginRateKey = "secure-proxy:user:%s:%s:login-count"
)

func (sm *defaultStore) UserCanLogin(userType string, account string) error {
	attempt, err := sm.cacheDriver.Get(context.TODO(), fmt.Sprintf(UserLoginRateKey, userType, account))
	if err != nil {
		return err
	}

	if attempt == "" {
		return nil
	}

	attemptN, _ := strconv.Atoi(attempt)
	if attemptN > 5 {
		ttl, err := sm.cacheDriver.TTL(context.TODO(), fmt.Sprintf(UserLoginRateKey, userType, account))
		if err != nil {
			return err
		}

		return fmt.Errorf("尝试次数过多，请 %s 后再试", ttl)
	}

	return nil
}

func (sm *defaultStore) UserLoginAttempt(userType string, account string) error {
	if err := sm.cacheDriver.Incr(context.TODO(), fmt.Sprintf(UserLoginRateKey, userType, account)); err != nil {
		return err
	}

	return sm.cacheDriver.Expire(context.TODO(), fmt.Sprintf(UserLoginRateKey, userType, account), 10*time.Minute)
}
