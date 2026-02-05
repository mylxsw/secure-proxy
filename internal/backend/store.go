package backend

import (
	"sync"
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/repository"
)

// Store 后端服务存储
// 支持从配置文件和数据库双重来源加载 backends
type Store struct {
	conf       *config.Config
	repo       *repository.BackendRepository
	mu         sync.RWMutex
	backends   map[string]config.Backend
	lastReload time.Time
	reloadTTL  time.Duration
}

// NewStore 创建后端服务存储
func NewStore(conf *config.Config, repo *repository.BackendRepository) *Store {
	s := &Store{
		conf:      conf,
		repo:      repo,
		backends:  make(map[string]config.Backend),
		reloadTTL: 30 * time.Second, // 默认 30 秒缓存
	}

	// 初始化加载
	s.reload()

	return s
}

// Get 获取指定 host 的 backend
func (s *Store) Get(host string) (config.Backend, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	backend, ok := s.backends[host]
	return backend, ok
}

// GetAll 获取所有 backends
func (s *Store) GetAll() map[string]config.Backend {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]config.Backend, len(s.backends))
	for k, v := range s.backends {
		result[k] = v
	}
	return result
}

// Reload 重新加载 backends
func (s *Store) Reload() {
	s.reload()
}

// ReloadIfNeeded 根据缓存 TTL 决定是否重新加载
func (s *Store) ReloadIfNeeded() {
	s.mu.RLock()
	needReload := time.Since(s.lastReload) > s.reloadTTL
	s.mu.RUnlock()

	if needReload {
		s.reload()
	}
}

func (s *Store) reload() {
	s.mu.Lock()
	defer s.mu.Unlock()

	backends := make(map[string]config.Backend)

	// 首先从配置文件加载
	for _, b := range s.conf.Backends {
		backends[b.Host] = b
	}

	// 如果数据库启用，从数据库加载并覆盖
	if s.conf.Database.Enabled && s.repo != nil {
		dbBackends, err := s.repo.GetAll()
		if err != nil {
			log.Errorf("failed to load backends from database: %v", err)
		} else {
			for _, b := range dbBackends {
				backends[b.Host] = b.ToConfigBackend()
			}
			log.Debugf("loaded %d backends from database", len(dbBackends))
		}
	}

	s.backends = backends
	s.lastReload = time.Now()

	log.Debugf("backends store reloaded, total: %d", len(s.backends))
}

// SetReloadTTL 设置缓存重载间隔
func (s *Store) SetReloadTTL(ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reloadTTL = ttl
}
