package database

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	// Test SQLite connection
	tmpFile := "/tmp/test_secure_proxy.db"
	defer os.Remove(tmpFile)

	db, err := New(Config{
		Driver: DriverSQLite,
		DSN:    tmpFile,
	})
	if err != nil {
		t.Fatalf("failed to create SQLite database: %v", err)
	}
	defer db.Close()

	if db.Driver() != DriverSQLite {
		t.Errorf("expected driver SQLite, got %s", db.Driver())
	}
}

func TestNewWithInvalidDriver(t *testing.T) {
	_, err := New(Config{
		Driver: "invalid",
		DSN:    "test.db",
	})
	if err == nil {
		t.Error("expected error for invalid driver")
	}
}

func TestPlaceholder(t *testing.T) {
	tests := []struct {
		driver   Driver
		index    int
		expected string
	}{
		{DriverSQLite, 1, "?"},
		{DriverMySQL, 1, "?"},
		{DriverPostgres, 1, "$1"},
		{DriverPostgres, 5, "$5"},
	}

	for _, tt := range tests {
		db := &DB{driver: tt.driver}
		result := db.Placeholder(tt.index)
		if result != tt.expected {
			t.Errorf("Placeholder(%d) for %s: expected %s, got %s", tt.index, tt.driver, tt.expected, result)
		}
	}
}

func TestAutoIncrement(t *testing.T) {
	tests := []struct {
		driver   Driver
		expected string
	}{
		{DriverSQLite, "INTEGER PRIMARY KEY AUTOINCREMENT"},
		{DriverMySQL, "INT AUTO_INCREMENT PRIMARY KEY"},
		{DriverPostgres, "SERIAL PRIMARY KEY"},
	}

	for _, tt := range tests {
		db := &DB{driver: tt.driver}
		result := db.AutoIncrement()
		if result != tt.expected {
			t.Errorf("AutoIncrement for %s: expected %s, got %s", tt.driver, tt.expected, result)
		}
	}
}

func TestTimestampType(t *testing.T) {
	tests := []struct {
		driver   Driver
		expected string
	}{
		{DriverSQLite, "DATETIME"},
		{DriverMySQL, "DATETIME"},
		{DriverPostgres, "TIMESTAMP"},
	}

	for _, tt := range tests {
		db := &DB{driver: tt.driver}
		result := db.TimestampType()
		if result != tt.expected {
			t.Errorf("TimestampType for %s: expected %s, got %s", tt.driver, tt.expected, result)
		}
	}
}

func TestBoolType(t *testing.T) {
	tests := []struct {
		driver   Driver
		expected string
	}{
		{DriverSQLite, "INTEGER"},
		{DriverMySQL, "TINYINT(1)"},
		{DriverPostgres, "BOOLEAN"},
	}

	for _, tt := range tests {
		db := &DB{driver: tt.driver}
		result := db.BoolType()
		if result != tt.expected {
			t.Errorf("BoolType for %s: expected %s, got %s", tt.driver, tt.expected, result)
		}
	}
}

func TestBoolValue(t *testing.T) {
	// Test SQLite/MySQL
	db := &DB{driver: DriverSQLite}
	if db.BoolValue(true) != 1 {
		t.Error("BoolValue(true) for SQLite should return 1")
	}
	if db.BoolValue(false) != 0 {
		t.Error("BoolValue(false) for SQLite should return 0")
	}

	// Test PostgreSQL
	db = &DB{driver: DriverPostgres}
	if db.BoolValue(true) != true {
		t.Error("BoolValue(true) for PostgreSQL should return true")
	}
	if db.BoolValue(false) != false {
		t.Error("BoolValue(false) for PostgreSQL should return false")
	}
}

func TestParseBool(t *testing.T) {
	db := &DB{driver: DriverSQLite}

	tests := []struct {
		input    interface{}
		expected bool
	}{
		{true, true},
		{false, false},
		{int64(1), true},
		{int64(0), false},
		{int(1), true},
		{int(0), false},
		{[]uint8{'1'}, true},
		{[]uint8{'0'}, false},
	}

	for _, tt := range tests {
		result := db.ParseBool(tt.input)
		if result != tt.expected {
			t.Errorf("ParseBool(%v): expected %v, got %v", tt.input, tt.expected, result)
		}
	}
}

func TestTransaction(t *testing.T) {
	tmpFile := "/tmp/test_transaction.db"
	defer os.Remove(tmpFile)

	db, err := New(Config{
		Driver: DriverSQLite,
		DSN:    tmpFile,
	})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	// Create test table
	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	// Test successful transaction
	err = db.Transaction(func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO test (name) VALUES (?)", "test1")
		return err
	})
	if err != nil {
		t.Errorf("transaction should succeed: %v", err)
	}

	// Verify insert
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test").Scan(&count)
	if err != nil || count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}

	// Test failed transaction (rollback)
	err = db.Transaction(func(tx *sql.Tx) error {
		_, _ = tx.Exec("INSERT INTO test (name) VALUES (?)", "test2")
		return fmt.Errorf("intentional error")
	})
	if err == nil {
		t.Error("transaction should fail")
	}

	// Verify rollback
	err = db.QueryRow("SELECT COUNT(*) FROM test").Scan(&count)
	if err != nil || count != 1 {
		t.Errorf("expected 1 row after rollback, got %d", count)
	}
}
