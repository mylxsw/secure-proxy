package backend

import (
	"os"
	"testing"
	"time"

	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/database"
	"github.com/mylxsw/secure-proxy/internal/repository"
)

func setupTestDB(t *testing.T) (*database.DB, *repository.BackendRepository, func()) {
	tmpFile := "/tmp/test_backend_store.db"

	db, err := database.New(database.Config{
		Driver: database.DriverSQLite,
		DSN:    tmpFile,
	})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	migration := database.NewMigration(db)
	if err := migration.Migrate(); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	repo := repository.NewBackendRepository(db)

	cleanup := func() {
		db.Close()
		os.Remove(tmpFile)
	}

	return db, repo, cleanup
}

func TestNewStore(t *testing.T) {
	conf := &config.Config{
		Backends: []config.Backend{
			{Host: "config.example.com", Upstream: "http://localhost:8001", Privilege: "internal"},
		},
		Database: config.Database{Enabled: false},
	}

	store := NewStore(conf, nil)

	if store == nil {
		t.Fatal("NewStore returned nil")
	}

	// Should load from config
	backend, ok := store.Get("config.example.com")
	if !ok {
		t.Error("backend from config should exist")
	}
	if backend.Upstream != "http://localhost:8001" {
		t.Errorf("expected upstream http://localhost:8001, got %s", backend.Upstream)
	}
}

func TestStore_WithDatabase(t *testing.T) {
	_, repo, cleanup := setupTestDB(t)
	defer cleanup()

	// Add backend to database
	dbBackend := &repository.BackendModel{
		Host:      "db.example.com",
		Upstream:  "http://localhost:8002",
		Privilege: "internal",
		Enabled:   true,
	}
	_ = repo.Create(dbBackend)

	conf := &config.Config{
		Backends: []config.Backend{
			{Host: "config.example.com", Upstream: "http://localhost:8001", Privilege: "internal"},
		},
		Database: config.Database{Enabled: true},
	}

	store := NewStore(conf, repo)

	// Should have both config and database backends
	configBackend, ok := store.Get("config.example.com")
	if !ok {
		t.Error("config backend should exist")
	}
	if configBackend.Upstream != "http://localhost:8001" {
		t.Errorf("expected upstream http://localhost:8001, got %s", configBackend.Upstream)
	}

	dbBackendResult, ok := store.Get("db.example.com")
	if !ok {
		t.Error("database backend should exist")
	}
	if dbBackendResult.Upstream != "http://localhost:8002" {
		t.Errorf("expected upstream http://localhost:8002, got %s", dbBackendResult.Upstream)
	}
}

func TestStore_DatabaseOverridesConfig(t *testing.T) {
	_, repo, cleanup := setupTestDB(t)
	defer cleanup()

	// Add backend to database with same host as config but different upstream
	dbBackend := &repository.BackendModel{
		Host:      "test.example.com",
		Upstream:  "http://localhost:9000", // Different from config
		Privilege: "private",
		Enabled:   true,
	}
	_ = repo.Create(dbBackend)

	conf := &config.Config{
		Backends: []config.Backend{
			{Host: "test.example.com", Upstream: "http://localhost:8001", Privilege: "internal"},
		},
		Database: config.Database{Enabled: true},
	}

	store := NewStore(conf, repo)

	// Database should override config
	backend, ok := store.Get("test.example.com")
	if !ok {
		t.Error("backend should exist")
	}
	if backend.Upstream != "http://localhost:9000" {
		t.Errorf("database should override config, expected upstream http://localhost:9000, got %s", backend.Upstream)
	}
	if backend.Privilege != "private" {
		t.Errorf("expected privilege private, got %s", backend.Privilege)
	}
}

func TestStore_GetAll(t *testing.T) {
	_, repo, cleanup := setupTestDB(t)
	defer cleanup()

	_ = repo.Create(&repository.BackendModel{
		Host:      "db1.example.com",
		Upstream:  "http://localhost:8001",
		Privilege: "internal",
		Enabled:   true,
	})
	_ = repo.Create(&repository.BackendModel{
		Host:      "db2.example.com",
		Upstream:  "http://localhost:8002",
		Privilege: "internal",
		Enabled:   true,
	})

	conf := &config.Config{
		Backends: []config.Backend{
			{Host: "config.example.com", Upstream: "http://localhost:8000", Privilege: "internal"},
		},
		Database: config.Database{Enabled: true},
	}

	store := NewStore(conf, repo)

	all := store.GetAll()
	if len(all) != 3 {
		t.Errorf("expected 3 backends, got %d", len(all))
	}
}

func TestStore_Reload(t *testing.T) {
	_, repo, cleanup := setupTestDB(t)
	defer cleanup()

	conf := &config.Config{
		Database: config.Database{Enabled: true},
	}

	store := NewStore(conf, repo)

	// Initially no backends
	all := store.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 backends, got %d", len(all))
	}

	// Add backend to database
	_ = repo.Create(&repository.BackendModel{
		Host:      "new.example.com",
		Upstream:  "http://localhost:8001",
		Privilege: "internal",
		Enabled:   true,
	})

	// Reload
	store.Reload()

	// Should now have the new backend
	backend, ok := store.Get("new.example.com")
	if !ok {
		t.Error("new backend should exist after reload")
	}
	if backend.Upstream != "http://localhost:8001" {
		t.Errorf("expected upstream http://localhost:8001, got %s", backend.Upstream)
	}
}

func TestStore_ReloadIfNeeded(t *testing.T) {
	_, repo, cleanup := setupTestDB(t)
	defer cleanup()

	conf := &config.Config{
		Database: config.Database{Enabled: true},
	}

	store := NewStore(conf, repo)
	store.SetReloadTTL(100 * time.Millisecond)

	// Add backend
	_ = repo.Create(&repository.BackendModel{
		Host:      "test.example.com",
		Upstream:  "http://localhost:8001",
		Privilege: "internal",
		Enabled:   true,
	})

	// Reload to get initial data
	store.Reload()

	// Get initial data
	backend1, _ := store.Get("test.example.com")
	if backend1.Upstream != "http://localhost:8001" {
		t.Errorf("expected upstream http://localhost:8001")
	}

	// Immediately call ReloadIfNeeded - should not reload yet
	store.ReloadIfNeeded()

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Update database
	b, _ := repo.GetByHost("test.example.com")
	b.Upstream = "http://localhost:9000"
	_ = repo.Update(b)

	// Now ReloadIfNeeded should reload
	store.ReloadIfNeeded()

	backend2, _ := store.Get("test.example.com")
	if backend2.Upstream != "http://localhost:9000" {
		t.Errorf("expected upstream http://localhost:9000 after reload, got %s", backend2.Upstream)
	}
}

func TestStore_SetReloadTTL(t *testing.T) {
	conf := &config.Config{
		Database: config.Database{Enabled: false},
	}

	store := NewStore(conf, nil)

	// Default TTL should be 30 seconds
	if store.reloadTTL != 30*time.Second {
		t.Errorf("expected default TTL 30s, got %v", store.reloadTTL)
	}

	// Set custom TTL
	store.SetReloadTTL(5 * time.Minute)
	if store.reloadTTL != 5*time.Minute {
		t.Errorf("expected TTL 5m, got %v", store.reloadTTL)
	}
}
