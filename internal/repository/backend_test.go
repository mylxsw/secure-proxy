package repository

import (
	"os"
	"testing"

	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/database"
)

func setupTestDB(t *testing.T) (*database.DB, func()) {
	tmpFile := "/tmp/test_backend_repo.db"

	db, err := database.New(database.Config{
		Driver: database.DriverSQLite,
		DSN:    tmpFile,
	})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	// Run migrations
	migration := database.NewMigration(db)
	if err := migration.Migrate(); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(tmpFile)
	}

	return db, cleanup
}

func TestBackendRepository_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBackendRepository(db)

	backend := &BackendModel{
		Host:        "test.example.com",
		Upstream:    "http://localhost:8080",
		Privilege:   "internal",
		Enabled:     true,
		Description: "Test backend",
		Groups:      []string{"admin", "devops"},
		Users:       []string{"user1", "user2"},
		Headers:     []config.Header{{Key: "X-Test", Value: "test"}},
	}

	err := repo.Create(backend)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if backend.ID == 0 {
		t.Error("backend ID should be set after create")
	}

	// Verify by getting
	got, err := repo.GetByID(backend.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Host != backend.Host {
		t.Errorf("expected host %s, got %s", backend.Host, got.Host)
	}
	if got.Upstream != backend.Upstream {
		t.Errorf("expected upstream %s, got %s", backend.Upstream, got.Upstream)
	}
	if len(got.Groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(got.Groups))
	}
	if len(got.Users) != 2 {
		t.Errorf("expected 2 users, got %d", len(got.Users))
	}
	if len(got.Headers) != 1 {
		t.Errorf("expected 1 header, got %d", len(got.Headers))
	}
}

func TestBackendRepository_GetByHost(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBackendRepository(db)

	backend := &BackendModel{
		Host:      "test.example.com",
		Upstream:  "http://localhost:8080",
		Privilege: "internal",
		Enabled:   true,
	}
	_ = repo.Create(backend)

	// Get by host
	got, err := repo.GetByHost("test.example.com")
	if err != nil {
		t.Fatalf("GetByHost failed: %v", err)
	}
	if got == nil {
		t.Fatal("backend not found")
	}
	if got.Host != "test.example.com" {
		t.Errorf("expected host test.example.com, got %s", got.Host)
	}

	// Get non-existent host
	got, err = repo.GetByHost("nonexistent.example.com")
	if err != nil {
		t.Fatalf("GetByHost failed: %v", err)
	}
	if got != nil {
		t.Error("should return nil for non-existent host")
	}
}

func TestBackendRepository_GetAll(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBackendRepository(db)

	// Create backends (one enabled, one disabled)
	_ = repo.Create(&BackendModel{
		Host:      "enabled.example.com",
		Upstream:  "http://localhost:8080",
		Privilege: "internal",
		Enabled:   true,
	})
	_ = repo.Create(&BackendModel{
		Host:      "disabled.example.com",
		Upstream:  "http://localhost:8081",
		Privilege: "internal",
		Enabled:   false,
	})

	// GetAll should only return enabled backends
	backends, err := repo.GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(backends) != 1 {
		t.Errorf("expected 1 enabled backend, got %d", len(backends))
	}

	// GetAllIncludeDisabled should return all backends
	allBackends, err := repo.GetAllIncludeDisabled()
	if err != nil {
		t.Fatalf("GetAllIncludeDisabled failed: %v", err)
	}
	if len(allBackends) != 2 {
		t.Errorf("expected 2 backends, got %d", len(allBackends))
	}
}

func TestBackendRepository_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBackendRepository(db)

	backend := &BackendModel{
		Host:      "test.example.com",
		Upstream:  "http://localhost:8080",
		Privilege: "internal",
		Enabled:   true,
		Groups:    []string{"admin"},
	}
	_ = repo.Create(backend)

	// Update
	backend.Upstream = "http://localhost:9090"
	backend.Privilege = "private"
	backend.Groups = []string{"admin", "devops"}
	backend.Users = []string{"user1"}

	err := repo.Update(backend)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify
	got, _ := repo.GetByID(backend.ID)
	if got.Upstream != "http://localhost:9090" {
		t.Errorf("expected upstream http://localhost:9090, got %s", got.Upstream)
	}
	if got.Privilege != "private" {
		t.Errorf("expected privilege private, got %s", got.Privilege)
	}
	if len(got.Groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(got.Groups))
	}
	if len(got.Users) != 1 {
		t.Errorf("expected 1 user, got %d", len(got.Users))
	}
}

func TestBackendRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBackendRepository(db)

	backend := &BackendModel{
		Host:      "test.example.com",
		Upstream:  "http://localhost:8080",
		Privilege: "internal",
		Enabled:   true,
		Groups:    []string{"admin"},
	}
	_ = repo.Create(backend)

	// Delete
	err := repo.Delete(backend.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	got, _ := repo.GetByID(backend.ID)
	if got != nil {
		t.Error("backend should be deleted")
	}

	// Verify related data deleted
	var groupCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM backend_groups WHERE backend_id = ?", backend.ID).Scan(&groupCount)
	if groupCount != 0 {
		t.Errorf("expected 0 groups, got %d", groupCount)
	}
}

func TestBackendModel_ToConfigBackend(t *testing.T) {
	model := &BackendModel{
		Host:      "test.example.com",
		Upstream:  "http://localhost:8080",
		Privilege: "private",
		Groups:    []string{"admin", "devops"},
		Users:     []string{"user1"},
		Headers:   []config.Header{{Key: "X-Test", Value: "test"}},
	}

	cfg := model.ToConfigBackend()

	if cfg.Host != model.Host {
		t.Errorf("Host mismatch: expected %s, got %s", model.Host, cfg.Host)
	}
	if cfg.Upstream != model.Upstream {
		t.Errorf("Upstream mismatch: expected %s, got %s", model.Upstream, cfg.Upstream)
	}
	if cfg.Privilege != model.Privilege {
		t.Errorf("Privilege mismatch: expected %s, got %s", model.Privilege, cfg.Privilege)
	}
	if len(cfg.Groups) != len(model.Groups) {
		t.Errorf("Groups length mismatch: expected %d, got %d", len(model.Groups), len(cfg.Groups))
	}
	if len(cfg.Users) != len(model.Users) {
		t.Errorf("Users length mismatch: expected %d, got %d", len(model.Users), len(cfg.Users))
	}
	if len(cfg.RewriteHeaders) != len(model.Headers) {
		t.Errorf("Headers length mismatch: expected %d, got %d", len(model.Headers), len(cfg.RewriteHeaders))
	}
}
