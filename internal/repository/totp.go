package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/mylxsw/secure-proxy/internal/database"
)

// UserTOTPModel 用户 TOTP 数据模型
type UserTOTPModel struct {
	ID        int64     `json:"id"`
	UserType  string    `json:"user_type"`
	Account   string    `json:"account"`
	Secret    string    `json:"-"` // 不对外暴露
	Enabled   bool      `json:"enabled"`
	Verified  bool      `json:"verified"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TOTPRepository TOTP 数据仓库
type TOTPRepository struct {
	db *database.DB
}

// NewTOTPRepository 创建 TOTP 仓库
func NewTOTPRepository(db *database.DB) *TOTPRepository {
	return &TOTPRepository{db: db}
}

// GetByUser 根据用户类型和账号获取 TOTP 配置
func (r *TOTPRepository) GetByUser(userType, account string) (*UserTOTPModel, error) {
	var query string
	switch r.db.Driver() {
	case database.DriverPostgres:
		query = `SELECT id, user_type, account, secret, enabled, verified, created_at, updated_at FROM user_totp WHERE user_type = $1 AND account = $2`
	default:
		query = `SELECT id, user_type, account, secret, enabled, verified, created_at, updated_at FROM user_totp WHERE user_type = ? AND account = ?`
	}

	var t UserTOTPModel
	var enabled, verified interface{}
	err := r.db.QueryRow(query, userType, account).Scan(
		&t.ID, &t.UserType, &t.Account, &t.Secret, &enabled, &verified, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user totp: %w", err)
	}

	t.Enabled = r.db.ParseBool(enabled)
	t.Verified = r.db.ParseBool(verified)

	return &t, nil
}

// Create 创建 TOTP 配置
func (r *TOTPRepository) Create(t *UserTOTPModel) error {
	var query string
	var err error

	switch r.db.Driver() {
	case database.DriverPostgres:
		query = `INSERT INTO user_totp (user_type, account, secret, enabled, verified) VALUES ($1, $2, $3, $4, $5) RETURNING id`
		err = r.db.QueryRow(query, t.UserType, t.Account, t.Secret, r.db.BoolValue(t.Enabled), r.db.BoolValue(t.Verified)).Scan(&t.ID)
	default:
		query = `INSERT INTO user_totp (user_type, account, secret, enabled, verified) VALUES (?, ?, ?, ?, ?)`
		result, execErr := r.db.Exec(query, t.UserType, t.Account, t.Secret, r.db.BoolValue(t.Enabled), r.db.BoolValue(t.Verified))
		if execErr != nil {
			err = execErr
		} else {
			t.ID, err = result.LastInsertId()
		}
	}

	if err != nil {
		return fmt.Errorf("failed to insert user totp: %w", err)
	}

	return nil
}

// Update 更新 TOTP 配置
func (r *TOTPRepository) Update(t *UserTOTPModel) error {
	var query string
	switch r.db.Driver() {
	case database.DriverPostgres:
		query = `UPDATE user_totp SET secret = $1, enabled = $2, verified = $3, updated_at = $4 WHERE id = $5`
	default:
		query = `UPDATE user_totp SET secret = ?, enabled = ?, verified = ?, updated_at = ? WHERE id = ?`
	}

	_, err := r.db.Exec(query, t.Secret, r.db.BoolValue(t.Enabled), r.db.BoolValue(t.Verified), time.Now(), t.ID)
	if err != nil {
		return fmt.Errorf("failed to update user totp: %w", err)
	}

	return nil
}

// Delete 删除 TOTP 配置
func (r *TOTPRepository) Delete(userType, account string) error {
	var query string
	switch r.db.Driver() {
	case database.DriverPostgres:
		query = `DELETE FROM user_totp WHERE user_type = $1 AND account = $2`
	default:
		query = `DELETE FROM user_totp WHERE user_type = ? AND account = ?`
	}

	_, err := r.db.Exec(query, userType, account)
	if err != nil {
		return fmt.Errorf("failed to delete user totp: %w", err)
	}

	return nil
}

// Enable 启用 TOTP
func (r *TOTPRepository) Enable(userType, account string) error {
	var query string
	switch r.db.Driver() {
	case database.DriverPostgres:
		query = `UPDATE user_totp SET enabled = $1, updated_at = $2 WHERE user_type = $3 AND account = $4`
	default:
		query = `UPDATE user_totp SET enabled = ?, updated_at = ? WHERE user_type = ? AND account = ?`
	}

	_, err := r.db.Exec(query, r.db.BoolValue(true), time.Now(), userType, account)
	if err != nil {
		return fmt.Errorf("failed to enable user totp: %w", err)
	}

	return nil
}

// Disable 禁用 TOTP
func (r *TOTPRepository) Disable(userType, account string) error {
	var query string
	switch r.db.Driver() {
	case database.DriverPostgres:
		query = `UPDATE user_totp SET enabled = $1, updated_at = $2 WHERE user_type = $3 AND account = $4`
	default:
		query = `UPDATE user_totp SET enabled = ?, updated_at = ? WHERE user_type = ? AND account = ?`
	}

	_, err := r.db.Exec(query, r.db.BoolValue(false), time.Now(), userType, account)
	if err != nil {
		return fmt.Errorf("failed to disable user totp: %w", err)
	}

	return nil
}

// SetVerified 设置 TOTP 已验证状态
func (r *TOTPRepository) SetVerified(userType, account string, verified bool) error {
	var query string
	switch r.db.Driver() {
	case database.DriverPostgres:
		query = `UPDATE user_totp SET verified = $1, updated_at = $2 WHERE user_type = $3 AND account = $4`
	default:
		query = `UPDATE user_totp SET verified = ?, updated_at = ? WHERE user_type = ? AND account = ?`
	}

	_, err := r.db.Exec(query, r.db.BoolValue(verified), time.Now(), userType, account)
	if err != nil {
		return fmt.Errorf("failed to set user totp verified: %w", err)
	}

	return nil
}

// IsEnabled 检查用户是否启用了 TOTP
func (r *TOTPRepository) IsEnabled(userType, account string) (bool, error) {
	t, err := r.GetByUser(userType, account)
	if err != nil {
		return false, err
	}
	if t == nil {
		return false, nil
	}
	return t.Enabled && t.Verified, nil
}

// GetSecret 获取用户的 TOTP Secret
func (r *TOTPRepository) GetSecret(userType, account string) (string, error) {
	t, err := r.GetByUser(userType, account)
	if err != nil {
		return "", err
	}
	if t == nil {
		return "", nil
	}
	return t.Secret, nil
}
