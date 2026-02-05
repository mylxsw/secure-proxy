package database

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// Driver 数据库驱动类型
type Driver string

const (
	DriverSQLite   Driver = "sqlite"
	DriverMySQL    Driver = "mysql"
	DriverPostgres Driver = "postgres"
)

// Config 数据库配置
type Config struct {
	Driver          Driver `json:"driver" yaml:"driver"`
	DSN             string `json:"dsn" yaml:"dsn"`
	MaxOpenConns    int    `json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns    int    `json:"max_idle_conns" yaml:"max_idle_conns"`
	ConnMaxLifetime int    `json:"conn_max_lifetime" yaml:"conn_max_lifetime"` // 秒
}

// DB 数据库连接封装
type DB struct {
	*sql.DB
	driver Driver
	mu     sync.RWMutex
}

// New 创建新的数据库连接
func New(conf Config) (*DB, error) {
	var driverName string
	switch conf.Driver {
	case DriverSQLite:
		driverName = "sqlite3"
	case DriverMySQL:
		driverName = "mysql"
	case DriverPostgres:
		driverName = "postgres"
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", conf.Driver)
	}

	db, err := sql.Open(driverName, conf.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 设置连接池参数
	if conf.MaxOpenConns > 0 {
		db.SetMaxOpenConns(conf.MaxOpenConns)
	} else {
		db.SetMaxOpenConns(25)
	}

	if conf.MaxIdleConns > 0 {
		db.SetMaxIdleConns(conf.MaxIdleConns)
	} else {
		db.SetMaxIdleConns(5)
	}

	if conf.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(time.Duration(conf.ConnMaxLifetime) * time.Second)
	} else {
		db.SetConnMaxLifetime(5 * time.Minute)
	}

	// 验证连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db, driver: conf.Driver}, nil
}

// Driver 获取数据库驱动类型
func (db *DB) Driver() Driver {
	return db.driver
}

// Placeholder 获取占位符格式
// SQLite 和 MySQL 使用 ?
// PostgreSQL 使用 $1, $2, ...
func (db *DB) Placeholder(index int) string {
	if db.driver == DriverPostgres {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

// AutoIncrement 获取自增语法
func (db *DB) AutoIncrement() string {
	switch db.driver {
	case DriverSQLite:
		return "INTEGER PRIMARY KEY AUTOINCREMENT"
	case DriverMySQL:
		return "INT AUTO_INCREMENT PRIMARY KEY"
	case DriverPostgres:
		return "SERIAL PRIMARY KEY"
	default:
		return "INTEGER PRIMARY KEY AUTOINCREMENT"
	}
}

// TimestampType 获取时间戳类型
func (db *DB) TimestampType() string {
	switch db.driver {
	case DriverMySQL:
		return "DATETIME"
	case DriverPostgres:
		return "TIMESTAMP"
	default:
		return "DATETIME"
	}
}

// TextType 获取文本类型
func (db *DB) TextType() string {
	return "TEXT"
}

// BoolType 获取布尔类型
func (db *DB) BoolType() string {
	switch db.driver {
	case DriverMySQL:
		return "TINYINT(1)"
	case DriverPostgres:
		return "BOOLEAN"
	default:
		return "INTEGER"
	}
}

// BoolValue 转换布尔值为数据库值
func (db *DB) BoolValue(v bool) interface{} {
	if db.driver == DriverPostgres {
		return v
	}
	if v {
		return 1
	}
	return 0
}

// ParseBool 从数据库值解析布尔值
func (db *DB) ParseBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case int64:
		return val != 0
	case int:
		return val != 0
	case []uint8:
		return len(val) > 0 && val[0] != '0'
	default:
		return false
	}
}

// Transaction 执行事务
func (db *DB) Transaction(fn func(tx *sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
