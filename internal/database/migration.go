package database

import (
	"fmt"
)

// Migration 数据库迁移
type Migration struct {
	db *DB
}

// NewMigration 创建迁移实例
func NewMigration(db *DB) *Migration {
	return &Migration{db: db}
}

// Migrate 执行数据库迁移
func (m *Migration) Migrate() error {
	// 创建迁移记录表
	if err := m.createMigrationTable(); err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}

	// 执行所有迁移
	migrations := m.getMigrations()
	for _, migration := range migrations {
		applied, err := m.isMigrationApplied(migration.Name)
		if err != nil {
			return fmt.Errorf("failed to check migration %s: %w", migration.Name, err)
		}
		if applied {
			continue
		}

		if err := migration.Up(m.db); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.Name, err)
		}

		if err := m.recordMigration(migration.Name); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", migration.Name, err)
		}
	}

	return nil
}

func (m *Migration) createMigrationTable() error {
	var query string
	switch m.db.Driver() {
	case DriverSQLite:
		query = `CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`
	case DriverMySQL:
		query = `CREATE TABLE IF NOT EXISTS migrations (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`
	case DriverPostgres:
		query = `CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`
	}

	_, err := m.db.Exec(query)
	return err
}

func (m *Migration) isMigrationApplied(name string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM migrations WHERE name = " + m.db.Placeholder(1)
	err := m.db.QueryRow(query, name).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *Migration) recordMigration(name string) error {
	query := "INSERT INTO migrations (name) VALUES (" + m.db.Placeholder(1) + ")"
	_, err := m.db.Exec(query, name)
	return err
}

type migrationItem struct {
	Name string
	Up   func(*DB) error
}

func (m *Migration) getMigrations() []migrationItem {
	return []migrationItem{
		{
			Name: "001_create_backends_table",
			Up:   m.createBackendsTable,
		},
		{
			Name: "002_create_backend_groups_table",
			Up:   m.createBackendGroupsTable,
		},
		{
			Name: "003_create_backend_users_table",
			Up:   m.createBackendUsersTable,
		},
		{
			Name: "004_create_backend_headers_table",
			Up:   m.createBackendHeadersTable,
		},
		{
			Name: "005_create_user_totp_table",
			Up:   m.createUserTOTPTable,
		},
	}
}

func (m *Migration) createBackendsTable(db *DB) error {
	var query string
	switch db.Driver() {
	case DriverSQLite:
		query = `CREATE TABLE IF NOT EXISTS backends (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			host TEXT NOT NULL UNIQUE,
			upstream TEXT NOT NULL,
			privilege TEXT NOT NULL DEFAULT 'internal',
			enabled INTEGER NOT NULL DEFAULT 1,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`
	case DriverMySQL:
		query = `CREATE TABLE IF NOT EXISTS backends (
			id INT AUTO_INCREMENT PRIMARY KEY,
			host VARCHAR(255) NOT NULL UNIQUE,
			upstream VARCHAR(500) NOT NULL,
			privilege VARCHAR(50) NOT NULL DEFAULT 'internal',
			enabled TINYINT(1) NOT NULL DEFAULT 1,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
	case DriverPostgres:
		query = `CREATE TABLE IF NOT EXISTS backends (
			id SERIAL PRIMARY KEY,
			host VARCHAR(255) NOT NULL UNIQUE,
			upstream VARCHAR(500) NOT NULL,
			privilege VARCHAR(50) NOT NULL DEFAULT 'internal',
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`
	}

	_, err := db.Exec(query)
	return err
}

func (m *Migration) createBackendGroupsTable(db *DB) error {
	var query string
	switch db.Driver() {
	case DriverSQLite:
		query = `CREATE TABLE IF NOT EXISTS backend_groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			backend_id INTEGER NOT NULL,
			group_name TEXT NOT NULL,
			FOREIGN KEY (backend_id) REFERENCES backends(id) ON DELETE CASCADE,
			UNIQUE(backend_id, group_name)
		)`
	case DriverMySQL:
		query = `CREATE TABLE IF NOT EXISTS backend_groups (
			id INT AUTO_INCREMENT PRIMARY KEY,
			backend_id INT NOT NULL,
			group_name VARCHAR(255) NOT NULL,
			FOREIGN KEY (backend_id) REFERENCES backends(id) ON DELETE CASCADE,
			UNIQUE KEY unique_backend_group (backend_id, group_name)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
	case DriverPostgres:
		query = `CREATE TABLE IF NOT EXISTS backend_groups (
			id SERIAL PRIMARY KEY,
			backend_id INTEGER NOT NULL REFERENCES backends(id) ON DELETE CASCADE,
			group_name VARCHAR(255) NOT NULL,
			UNIQUE(backend_id, group_name)
		)`
	}

	_, err := db.Exec(query)
	return err
}

func (m *Migration) createBackendUsersTable(db *DB) error {
	var query string
	switch db.Driver() {
	case DriverSQLite:
		query = `CREATE TABLE IF NOT EXISTS backend_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			backend_id INTEGER NOT NULL,
			username TEXT NOT NULL,
			FOREIGN KEY (backend_id) REFERENCES backends(id) ON DELETE CASCADE,
			UNIQUE(backend_id, username)
		)`
	case DriverMySQL:
		query = `CREATE TABLE IF NOT EXISTS backend_users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			backend_id INT NOT NULL,
			username VARCHAR(255) NOT NULL,
			FOREIGN KEY (backend_id) REFERENCES backends(id) ON DELETE CASCADE,
			UNIQUE KEY unique_backend_user (backend_id, username)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
	case DriverPostgres:
		query = `CREATE TABLE IF NOT EXISTS backend_users (
			id SERIAL PRIMARY KEY,
			backend_id INTEGER NOT NULL REFERENCES backends(id) ON DELETE CASCADE,
			username VARCHAR(255) NOT NULL,
			UNIQUE(backend_id, username)
		)`
	}

	_, err := db.Exec(query)
	return err
}

func (m *Migration) createBackendHeadersTable(db *DB) error {
	var query string
	switch db.Driver() {
	case DriverSQLite:
		query = `CREATE TABLE IF NOT EXISTS backend_headers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			backend_id INTEGER NOT NULL,
			header_key TEXT NOT NULL,
			header_value TEXT NOT NULL,
			FOREIGN KEY (backend_id) REFERENCES backends(id) ON DELETE CASCADE
		)`
	case DriverMySQL:
		query = `CREATE TABLE IF NOT EXISTS backend_headers (
			id INT AUTO_INCREMENT PRIMARY KEY,
			backend_id INT NOT NULL,
			header_key VARCHAR(255) NOT NULL,
			header_value TEXT NOT NULL,
			FOREIGN KEY (backend_id) REFERENCES backends(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
	case DriverPostgres:
		query = `CREATE TABLE IF NOT EXISTS backend_headers (
			id SERIAL PRIMARY KEY,
			backend_id INTEGER NOT NULL REFERENCES backends(id) ON DELETE CASCADE,
			header_key VARCHAR(255) NOT NULL,
			header_value TEXT NOT NULL
		)`
	}

	_, err := db.Exec(query)
	return err
}

func (m *Migration) createUserTOTPTable(db *DB) error {
	var query string
	switch db.Driver() {
	case DriverSQLite:
		query = `CREATE TABLE IF NOT EXISTS user_totp (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_type TEXT NOT NULL,
			account TEXT NOT NULL,
			secret TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 0,
			verified INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_type, account)
		)`
	case DriverMySQL:
		query = `CREATE TABLE IF NOT EXISTS user_totp (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_type VARCHAR(50) NOT NULL,
			account VARCHAR(255) NOT NULL,
			secret VARCHAR(255) NOT NULL,
			enabled TINYINT(1) NOT NULL DEFAULT 0,
			verified TINYINT(1) NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY unique_user_totp (user_type, account)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
	case DriverPostgres:
		query = `CREATE TABLE IF NOT EXISTS user_totp (
			id SERIAL PRIMARY KEY,
			user_type VARCHAR(50) NOT NULL,
			account VARCHAR(255) NOT NULL,
			secret VARCHAR(255) NOT NULL,
			enabled BOOLEAN NOT NULL DEFAULT FALSE,
			verified BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_type, account)
		)`
	}

	_, err := db.Exec(query)
	return err
}
