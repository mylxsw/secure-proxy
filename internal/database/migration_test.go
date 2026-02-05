package database

import (
	"os"
	"testing"
)

func TestMigration(t *testing.T) {
	tmpFile := "/tmp/test_migration.db"
	defer os.Remove(tmpFile)

	db, err := New(Config{
		Driver: DriverSQLite,
		DSN:    tmpFile,
	})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	migration := NewMigration(db)

	// Run migrations
	err = migration.Migrate()
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Verify tables were created
	tables := []string{"migrations", "backends", "backend_groups", "backend_users", "backend_headers", "user_totp"}
	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil {
			t.Errorf("failed to check table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("table %s should exist", table)
		}
	}

	// Verify migration records
	var migrationCount int
	err = db.QueryRow("SELECT COUNT(*) FROM migrations").Scan(&migrationCount)
	if err != nil {
		t.Fatalf("failed to count migrations: %v", err)
	}
	if migrationCount != 5 {
		t.Errorf("expected 5 migrations, got %d", migrationCount)
	}

	// Run migrations again (should be idempotent)
	err = migration.Migrate()
	if err != nil {
		t.Errorf("second migration should succeed: %v", err)
	}

	// Count should remain the same
	err = db.QueryRow("SELECT COUNT(*) FROM migrations").Scan(&migrationCount)
	if err != nil || migrationCount != 5 {
		t.Errorf("migration count should remain 5, got %d", migrationCount)
	}
}

func TestMigrationTableCreation(t *testing.T) {
	tmpFile := "/tmp/test_migration_table.db"
	defer os.Remove(tmpFile)

	db, err := New(Config{
		Driver: DriverSQLite,
		DSN:    tmpFile,
	})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	migration := NewMigration(db)

	// Create migration table
	err = migration.createMigrationTable()
	if err != nil {
		t.Fatalf("failed to create migration table: %v", err)
	}

	// Verify table exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='migrations'").Scan(&count)
	if err != nil || count != 1 {
		t.Error("migrations table should exist")
	}

	// Create again (should be idempotent)
	err = migration.createMigrationTable()
	if err != nil {
		t.Errorf("second createMigrationTable should succeed: %v", err)
	}
}

func TestIsMigrationApplied(t *testing.T) {
	tmpFile := "/tmp/test_migration_applied.db"
	defer os.Remove(tmpFile)

	db, err := New(Config{
		Driver: DriverSQLite,
		DSN:    tmpFile,
	})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	migration := NewMigration(db)
	_ = migration.createMigrationTable()

	// Test migration not applied
	applied, err := migration.isMigrationApplied("test_migration")
	if err != nil {
		t.Fatalf("isMigrationApplied failed: %v", err)
	}
	if applied {
		t.Error("migration should not be applied")
	}

	// Record migration
	err = migration.recordMigration("test_migration")
	if err != nil {
		t.Fatalf("recordMigration failed: %v", err)
	}

	// Test migration applied
	applied, err = migration.isMigrationApplied("test_migration")
	if err != nil {
		t.Fatalf("isMigrationApplied failed: %v", err)
	}
	if !applied {
		t.Error("migration should be applied")
	}
}

func TestBackendsTableSchema(t *testing.T) {
	tmpFile := "/tmp/test_backends_schema.db"
	defer os.Remove(tmpFile)

	db, err := New(Config{
		Driver: DriverSQLite,
		DSN:    tmpFile,
	})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	migration := NewMigration(db)
	err = migration.createBackendsTable(db)
	if err != nil {
		t.Fatalf("failed to create backends table: %v", err)
	}

	// Test insert
	result, err := db.Exec("INSERT INTO backends (host, upstream, privilege, enabled) VALUES (?, ?, ?, ?)",
		"test.example.com", "http://localhost:8080", "internal", 1)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	id, _ := result.LastInsertId()
	if id != 1 {
		t.Errorf("expected id 1, got %d", id)
	}

	// Test unique constraint
	_, err = db.Exec("INSERT INTO backends (host, upstream, privilege, enabled) VALUES (?, ?, ?, ?)",
		"test.example.com", "http://localhost:8081", "internal", 1)
	if err == nil {
		t.Error("duplicate host should fail")
	}
}

func TestUserTOTPTableSchema(t *testing.T) {
	tmpFile := "/tmp/test_totp_schema.db"
	defer os.Remove(tmpFile)

	db, err := New(Config{
		Driver: DriverSQLite,
		DSN:    tmpFile,
	})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	migration := NewMigration(db)
	err = migration.createUserTOTPTable(db)
	if err != nil {
		t.Fatalf("failed to create user_totp table: %v", err)
	}

	// Test insert
	result, err := db.Exec("INSERT INTO user_totp (user_type, account, secret, enabled, verified) VALUES (?, ?, ?, ?, ?)",
		"local", "testuser", "TESTSECRET", 0, 0)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	id, _ := result.LastInsertId()
	if id != 1 {
		t.Errorf("expected id 1, got %d", id)
	}

	// Test unique constraint
	_, err = db.Exec("INSERT INTO user_totp (user_type, account, secret, enabled, verified) VALUES (?, ?, ?, ?, ?)",
		"local", "testuser", "ANOTHERSECRET", 0, 0)
	if err == nil {
		t.Error("duplicate user_type+account should fail")
	}
}
