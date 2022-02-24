package memory

import (
	"context"
	"github.com/mylxsw/secure-proxy/internal/cache"
	"strconv"
	"sync"
	"time"
)

type data struct {
	data      string
	expiredAt time.Time
}

type memoryCache struct {
	data map[string]data
	lock sync.RWMutex
}

func New(ctx context.Context) cache.Cache {
	m := &memoryCache{data: make(map[string]data)}
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.lock.Lock()

				deleteKeys := make([]string, 0)
				for k, v := range m.data {
					if v.expiredAt.IsZero() {
						continue
					}

					if time.Now().After(v.expiredAt) {
						deleteKeys = append(deleteKeys, k)
					}
				}

				for _, k := range deleteKeys {
					delete(m.data, k)
				}

				m.lock.Unlock()
			}
		}
	}()

	return m
}

func (m *memoryCache) Get(ctx context.Context, key string) (string, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if res, ok := m.data[key]; ok {
		if !res.expiredAt.IsZero() {
			if time.Now().After(res.expiredAt) {
				return "", nil
			}
		}
		return res.data, nil
	}

	return "", nil
}

func (m *memoryCache) Set(ctx context.Context, key string, value string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.data[key] = data{data: value}
	return nil
}

func (m *memoryCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if res, ok := m.data[key]; ok {
		res.expiredAt = time.Now().Add(ttl)
		m.data[key] = res
	}

	return nil
}

func (m *memoryCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if res, ok := m.data[key]; ok {
		if res.expiredAt.IsZero() {
			return 0, nil
		}

		return res.expiredAt.Sub(time.Now()), nil
	}

	return 0, nil
}

func (m *memoryCache) Incr(ctx context.Context, key string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if res, ok := m.data[key]; ok {
		resN, err := strconv.Atoi(res.data)
		if err != nil {
			return err
		}

		res.data = strconv.Itoa(resN + 1)

		m.data[key] = res
	} else {
		m.data[key] = data{data: "1"}
	}

	return nil
}
