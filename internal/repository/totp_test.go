package repository

import (
	"os"
	"testing"

	"github.com/mylxsw/secure-proxy/internal/database"
)

func setupTestDBForTOTP(t *testing.T) (*database.DB, func()) {
	tmpFile := "/tmp/test_totp_repo.db"

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

func TestTOTPRepository_Create(t *testing.T) {
	db, cleanup := setupTestDBForTOTP(t)
	defer cleanup()

	repo := NewTOTPRepository(db)

	totp := &UserTOTPModel{
		UserType: "local",
		Account:  "testuser",
		Secret:   "TESTSECRET123456",
		Enabled:  false,
		Verified: false,
	}

	err := repo.Create(totp)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if totp.ID == 0 {
		t.Error("TOTP ID should be set after create")
	}

	// Verify by getting
	got, err := repo.GetByUser("local", "testuser")
	if err != nil {
		t.Fatalf("GetByUser failed: %v", err)
	}
	if got == nil {
		t.Fatal("TOTP not found")
	}
	if got.Secret != "TESTSECRET123456" {
		t.Errorf("expected secret TESTSECRET123456, got %s", got.Secret)
	}
	if got.Enabled {
		t.Error("TOTP should not be enabled")
	}
	if got.Verified {
		t.Error("TOTP should not be verified")
	}
}

func TestTOTPRepository_GetByUser(t *testing.T) {
	db, cleanup := setupTestDBForTOTP(t)
	defer cleanup()

	repo := NewTOTPRepository(db)

	// Create TOTP
	totp := &UserTOTPModel{
		UserType: "local",
		Account:  "testuser",
		Secret:   "TESTSECRET123456",
	}
	_ = repo.Create(totp)

	// Get existing user
	got, err := repo.GetByUser("local", "testuser")
	if err != nil {
		t.Fatalf("GetByUser failed: %v", err)
	}
	if got == nil {
		t.Fatal("TOTP should be found")
	}

	// Get non-existent user
	got, err = repo.GetByUser("local", "nonexistent")
	if err != nil {
		t.Fatalf("GetByUser failed: %v", err)
	}
	if got != nil {
		t.Error("should return nil for non-existent user")
	}

	// Get different user type
	got, err = repo.GetByUser("ldap", "testuser")
	if err != nil {
		t.Fatalf("GetByUser failed: %v", err)
	}
	if got != nil {
		t.Error("should return nil for different user type")
	}
}

func TestTOTPRepository_Update(t *testing.T) {
	db, cleanup := setupTestDBForTOTP(t)
	defer cleanup()

	repo := NewTOTPRepository(db)

	totp := &UserTOTPModel{
		UserType: "local",
		Account:  "testuser",
		Secret:   "TESTSECRET123456",
		Enabled:  false,
		Verified: false,
	}
	_ = repo.Create(totp)

	// Update
	totp.Secret = "NEWSECRET789012"
	totp.Enabled = true
	totp.Verified = true

	err := repo.Update(totp)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify
	got, _ := repo.GetByUser("local", "testuser")
	if got.Secret != "NEWSECRET789012" {
		t.Errorf("expected secret NEWSECRET789012, got %s", got.Secret)
	}
	if !got.Enabled {
		t.Error("TOTP should be enabled")
	}
	if !got.Verified {
		t.Error("TOTP should be verified")
	}
}

func TestTOTPRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDBForTOTP(t)
	defer cleanup()

	repo := NewTOTPRepository(db)

	totp := &UserTOTPModel{
		UserType: "local",
		Account:  "testuser",
		Secret:   "TESTSECRET123456",
	}
	_ = repo.Create(totp)

	// Delete
	err := repo.Delete("local", "testuser")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	got, _ := repo.GetByUser("local", "testuser")
	if got != nil {
		t.Error("TOTP should be deleted")
	}
}

func TestTOTPRepository_Enable(t *testing.T) {
	db, cleanup := setupTestDBForTOTP(t)
	defer cleanup()

	repo := NewTOTPRepository(db)

	totp := &UserTOTPModel{
		UserType: "local",
		Account:  "testuser",
		Secret:   "TESTSECRET123456",
		Enabled:  false,
	}
	_ = repo.Create(totp)

	// Enable
	err := repo.Enable("local", "testuser")
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	// Verify
	got, _ := repo.GetByUser("local", "testuser")
	if !got.Enabled {
		t.Error("TOTP should be enabled")
	}
}

func TestTOTPRepository_Disable(t *testing.T) {
	db, cleanup := setupTestDBForTOTP(t)
	defer cleanup()

	repo := NewTOTPRepository(db)

	totp := &UserTOTPModel{
		UserType: "local",
		Account:  "testuser",
		Secret:   "TESTSECRET123456",
		Enabled:  true,
	}
	_ = repo.Create(totp)

	// Disable
	err := repo.Disable("local", "testuser")
	if err != nil {
		t.Fatalf("Disable failed: %v", err)
	}

	// Verify
	got, _ := repo.GetByUser("local", "testuser")
	if got.Enabled {
		t.Error("TOTP should be disabled")
	}
}

func TestTOTPRepository_SetVerified(t *testing.T) {
	db, cleanup := setupTestDBForTOTP(t)
	defer cleanup()

	repo := NewTOTPRepository(db)

	totp := &UserTOTPModel{
		UserType: "local",
		Account:  "testuser",
		Secret:   "TESTSECRET123456",
		Verified: false,
	}
	_ = repo.Create(totp)

	// Set verified
	err := repo.SetVerified("local", "testuser", true)
	if err != nil {
		t.Fatalf("SetVerified failed: %v", err)
	}

	// Verify
	got, _ := repo.GetByUser("local", "testuser")
	if !got.Verified {
		t.Error("TOTP should be verified")
	}

	// Set not verified
	err = repo.SetVerified("local", "testuser", false)
	if err != nil {
		t.Fatalf("SetVerified failed: %v", err)
	}

	got, _ = repo.GetByUser("local", "testuser")
	if got.Verified {
		t.Error("TOTP should not be verified")
	}
}

func TestTOTPRepository_IsEnabled(t *testing.T) {
	db, cleanup := setupTestDBForTOTP(t)
	defer cleanup()

	repo := NewTOTPRepository(db)

	// Non-existent user
	enabled, err := repo.IsEnabled("local", "nonexistent")
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if enabled {
		t.Error("should return false for non-existent user")
	}

	// Create disabled TOTP
	totp := &UserTOTPModel{
		UserType: "local",
		Account:  "testuser",
		Secret:   "TESTSECRET123456",
		Enabled:  false,
		Verified: false,
	}
	_ = repo.Create(totp)

	enabled, err = repo.IsEnabled("local", "testuser")
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if enabled {
		t.Error("should return false when enabled=false")
	}

	// Enable but not verified
	totp.Enabled = true
	_ = repo.Update(totp)

	enabled, _ = repo.IsEnabled("local", "testuser")
	if enabled {
		t.Error("should return false when not verified")
	}

	// Enable and verified
	totp.Verified = true
	_ = repo.Update(totp)

	enabled, _ = repo.IsEnabled("local", "testuser")
	if !enabled {
		t.Error("should return true when enabled and verified")
	}
}

func TestTOTPRepository_GetSecret(t *testing.T) {
	db, cleanup := setupTestDBForTOTP(t)
	defer cleanup()

	repo := NewTOTPRepository(db)

	// Non-existent user
	secret, err := repo.GetSecret("local", "nonexistent")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if secret != "" {
		t.Error("should return empty string for non-existent user")
	}

	// Create TOTP
	totp := &UserTOTPModel{
		UserType: "local",
		Account:  "testuser",
		Secret:   "TESTSECRET123456",
	}
	_ = repo.Create(totp)

	secret, err = repo.GetSecret("local", "testuser")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if secret != "TESTSECRET123456" {
		t.Errorf("expected secret TESTSECRET123456, got %s", secret)
	}
}
