package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/database"
)

// BackendModel 后端服务数据模型
type BackendModel struct {
	ID          int64     `json:"id"`
	Host        string    `json:"host"`
	Upstream    string    `json:"upstream"`
	Privilege   string    `json:"privilege"`
	Enabled     bool      `json:"enabled"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 关联数据
	Groups  []string        `json:"groups,omitempty"`
	Users   []string        `json:"users,omitempty"`
	Headers []config.Header `json:"headers,omitempty"`
}

// ToConfigBackend 转换为配置 Backend
func (m *BackendModel) ToConfigBackend() config.Backend {
	return config.Backend{
		Host:           m.Host,
		Upstream:       m.Upstream,
		Privilege:      m.Privilege,
		Groups:         m.Groups,
		Users:          m.Users,
		RewriteHeaders: m.Headers,
	}
}

// BackendRepository Backend 数据仓库
type BackendRepository struct {
	db *database.DB
}

// NewBackendRepository 创建 Backend 仓库
func NewBackendRepository(db *database.DB) *BackendRepository {
	return &BackendRepository{db: db}
}

// GetAll 获取所有启用的 backends
func (r *BackendRepository) GetAll() ([]BackendModel, error) {
	query := "SELECT id, host, upstream, privilege, enabled, COALESCE(description, ''), created_at, updated_at FROM backends WHERE enabled = " + r.db.Placeholder(1)
	rows, err := r.db.Query(query, r.db.BoolValue(true))
	if err != nil {
		return nil, fmt.Errorf("failed to query backends: %w", err)
	}
	defer rows.Close()

	var backends []BackendModel
	for rows.Next() {
		var b BackendModel
		var enabled interface{}
		if err := rows.Scan(&b.ID, &b.Host, &b.Upstream, &b.Privilege, &enabled, &b.Description, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan backend: %w", err)
		}
		b.Enabled = r.db.ParseBool(enabled)
		backends = append(backends, b)
	}

	// 加载关联数据
	for i := range backends {
		groups, err := r.getBackendGroups(backends[i].ID)
		if err != nil {
			return nil, err
		}
		backends[i].Groups = groups

		users, err := r.getBackendUsers(backends[i].ID)
		if err != nil {
			return nil, err
		}
		backends[i].Users = users

		headers, err := r.getBackendHeaders(backends[i].ID)
		if err != nil {
			return nil, err
		}
		backends[i].Headers = headers
	}

	return backends, nil
}

// GetAllIncludeDisabled 获取所有 backends（包括禁用的）
func (r *BackendRepository) GetAllIncludeDisabled() ([]BackendModel, error) {
	query := "SELECT id, host, upstream, privilege, enabled, COALESCE(description, ''), created_at, updated_at FROM backends ORDER BY id"
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query backends: %w", err)
	}
	defer rows.Close()

	var backends []BackendModel
	for rows.Next() {
		var b BackendModel
		var enabled interface{}
		if err := rows.Scan(&b.ID, &b.Host, &b.Upstream, &b.Privilege, &enabled, &b.Description, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan backend: %w", err)
		}
		b.Enabled = r.db.ParseBool(enabled)
		backends = append(backends, b)
	}

	// 加载关联数据
	for i := range backends {
		groups, err := r.getBackendGroups(backends[i].ID)
		if err != nil {
			return nil, err
		}
		backends[i].Groups = groups

		users, err := r.getBackendUsers(backends[i].ID)
		if err != nil {
			return nil, err
		}
		backends[i].Users = users

		headers, err := r.getBackendHeaders(backends[i].ID)
		if err != nil {
			return nil, err
		}
		backends[i].Headers = headers
	}

	return backends, nil
}

// GetByHost 根据 host 获取 backend
func (r *BackendRepository) GetByHost(host string) (*BackendModel, error) {
	query := "SELECT id, host, upstream, privilege, enabled, COALESCE(description, ''), created_at, updated_at FROM backends WHERE host = " + r.db.Placeholder(1)
	var b BackendModel
	var enabled interface{}
	err := r.db.QueryRow(query, host).Scan(&b.ID, &b.Host, &b.Upstream, &b.Privilege, &enabled, &b.Description, &b.CreatedAt, &b.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query backend: %w", err)
	}
	b.Enabled = r.db.ParseBool(enabled)

	// 加载关联数据
	groups, err := r.getBackendGroups(b.ID)
	if err != nil {
		return nil, err
	}
	b.Groups = groups

	users, err := r.getBackendUsers(b.ID)
	if err != nil {
		return nil, err
	}
	b.Users = users

	headers, err := r.getBackendHeaders(b.ID)
	if err != nil {
		return nil, err
	}
	b.Headers = headers

	return &b, nil
}

// GetByID 根据 ID 获取 backend
func (r *BackendRepository) GetByID(id int64) (*BackendModel, error) {
	query := "SELECT id, host, upstream, privilege, enabled, COALESCE(description, ''), created_at, updated_at FROM backends WHERE id = " + r.db.Placeholder(1)
	var b BackendModel
	var enabled interface{}
	err := r.db.QueryRow(query, id).Scan(&b.ID, &b.Host, &b.Upstream, &b.Privilege, &enabled, &b.Description, &b.CreatedAt, &b.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query backend: %w", err)
	}
	b.Enabled = r.db.ParseBool(enabled)

	// 加载关联数据
	groups, err := r.getBackendGroups(b.ID)
	if err != nil {
		return nil, err
	}
	b.Groups = groups

	users, err := r.getBackendUsers(b.ID)
	if err != nil {
		return nil, err
	}
	b.Users = users

	headers, err := r.getBackendHeaders(b.ID)
	if err != nil {
		return nil, err
	}
	b.Headers = headers

	return &b, nil
}

// Create 创建 backend
func (r *BackendRepository) Create(b *BackendModel) error {
	return r.db.Transaction(func(tx *sql.Tx) error {
		var query string
		var result sql.Result
		var err error

		switch r.db.Driver() {
		case database.DriverPostgres:
			query = `INSERT INTO backends (host, upstream, privilege, enabled, description) VALUES ($1, $2, $3, $4, $5) RETURNING id`
			err = tx.QueryRow(query, b.Host, b.Upstream, b.Privilege, r.db.BoolValue(b.Enabled), b.Description).Scan(&b.ID)
		default:
			query = `INSERT INTO backends (host, upstream, privilege, enabled, description) VALUES (?, ?, ?, ?, ?)`
			result, err = tx.Exec(query, b.Host, b.Upstream, b.Privilege, r.db.BoolValue(b.Enabled), b.Description)
			if err == nil {
				b.ID, err = result.LastInsertId()
			}
		}
		if err != nil {
			return fmt.Errorf("failed to insert backend: %w", err)
		}

		// 插入 groups
		if err := r.insertGroups(tx, b.ID, b.Groups); err != nil {
			return err
		}

		// 插入 users
		if err := r.insertUsers(tx, b.ID, b.Users); err != nil {
			return err
		}

		// 插入 headers
		if err := r.insertHeaders(tx, b.ID, b.Headers); err != nil {
			return err
		}

		return nil
	})
}

// Update 更新 backend
func (r *BackendRepository) Update(b *BackendModel) error {
	return r.db.Transaction(func(tx *sql.Tx) error {
		var query string
		switch r.db.Driver() {
		case database.DriverPostgres:
			query = `UPDATE backends SET host = $1, upstream = $2, privilege = $3, enabled = $4, description = $5, updated_at = $6 WHERE id = $7`
		default:
			query = `UPDATE backends SET host = ?, upstream = ?, privilege = ?, enabled = ?, description = ?, updated_at = ? WHERE id = ?`
		}

		_, err := tx.Exec(query, b.Host, b.Upstream, b.Privilege, r.db.BoolValue(b.Enabled), b.Description, time.Now(), b.ID)
		if err != nil {
			return fmt.Errorf("failed to update backend: %w", err)
		}

		// 删除旧的关联数据
		if err := r.deleteRelatedData(tx, b.ID); err != nil {
			return err
		}

		// 重新插入关联数据
		if err := r.insertGroups(tx, b.ID, b.Groups); err != nil {
			return err
		}
		if err := r.insertUsers(tx, b.ID, b.Users); err != nil {
			return err
		}
		if err := r.insertHeaders(tx, b.ID, b.Headers); err != nil {
			return err
		}

		return nil
	})
}

// Delete 删除 backend
func (r *BackendRepository) Delete(id int64) error {
	return r.db.Transaction(func(tx *sql.Tx) error {
		// 删除关联数据
		if err := r.deleteRelatedData(tx, id); err != nil {
			return err
		}

		query := "DELETE FROM backends WHERE id = " + r.db.Placeholder(1)
		_, err := tx.Exec(query, id)
		if err != nil {
			return fmt.Errorf("failed to delete backend: %w", err)
		}

		return nil
	})
}

func (r *BackendRepository) getBackendGroups(backendID int64) ([]string, error) {
	query := "SELECT group_name FROM backend_groups WHERE backend_id = " + r.db.Placeholder(1)
	rows, err := r.db.Query(query, backendID)
	if err != nil {
		return nil, fmt.Errorf("failed to query backend groups: %w", err)
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var group string
		if err := rows.Scan(&group); err != nil {
			return nil, fmt.Errorf("failed to scan backend group: %w", err)
		}
		groups = append(groups, group)
	}

	return groups, nil
}

func (r *BackendRepository) getBackendUsers(backendID int64) ([]string, error) {
	query := "SELECT username FROM backend_users WHERE backend_id = " + r.db.Placeholder(1)
	rows, err := r.db.Query(query, backendID)
	if err != nil {
		return nil, fmt.Errorf("failed to query backend users: %w", err)
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var user string
		if err := rows.Scan(&user); err != nil {
			return nil, fmt.Errorf("failed to scan backend user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *BackendRepository) getBackendHeaders(backendID int64) ([]config.Header, error) {
	query := "SELECT header_key, header_value FROM backend_headers WHERE backend_id = " + r.db.Placeholder(1)
	rows, err := r.db.Query(query, backendID)
	if err != nil {
		return nil, fmt.Errorf("failed to query backend headers: %w", err)
	}
	defer rows.Close()

	var headers []config.Header
	for rows.Next() {
		var h config.Header
		if err := rows.Scan(&h.Key, &h.Value); err != nil {
			return nil, fmt.Errorf("failed to scan backend header: %w", err)
		}
		headers = append(headers, h)
	}

	return headers, nil
}

func (r *BackendRepository) insertGroups(tx *sql.Tx, backendID int64, groups []string) error {
	if len(groups) == 0 {
		return nil
	}

	var query string
	switch r.db.Driver() {
	case database.DriverPostgres:
		query = `INSERT INTO backend_groups (backend_id, group_name) VALUES ($1, $2)`
	default:
		query = `INSERT INTO backend_groups (backend_id, group_name) VALUES (?, ?)`
	}

	for _, group := range groups {
		if _, err := tx.Exec(query, backendID, group); err != nil {
			return fmt.Errorf("failed to insert backend group: %w", err)
		}
	}

	return nil
}

func (r *BackendRepository) insertUsers(tx *sql.Tx, backendID int64, users []string) error {
	if len(users) == 0 {
		return nil
	}

	var query string
	switch r.db.Driver() {
	case database.DriverPostgres:
		query = `INSERT INTO backend_users (backend_id, username) VALUES ($1, $2)`
	default:
		query = `INSERT INTO backend_users (backend_id, username) VALUES (?, ?)`
	}

	for _, user := range users {
		if _, err := tx.Exec(query, backendID, user); err != nil {
			return fmt.Errorf("failed to insert backend user: %w", err)
		}
	}

	return nil
}

func (r *BackendRepository) insertHeaders(tx *sql.Tx, backendID int64, headers []config.Header) error {
	if len(headers) == 0 {
		return nil
	}

	var query string
	switch r.db.Driver() {
	case database.DriverPostgres:
		query = `INSERT INTO backend_headers (backend_id, header_key, header_value) VALUES ($1, $2, $3)`
	default:
		query = `INSERT INTO backend_headers (backend_id, header_key, header_value) VALUES (?, ?, ?)`
	}

	for _, h := range headers {
		if _, err := tx.Exec(query, backendID, h.Key, h.Value); err != nil {
			return fmt.Errorf("failed to insert backend header: %w", err)
		}
	}

	return nil
}

func (r *BackendRepository) deleteRelatedData(tx *sql.Tx, backendID int64) error {
	tables := []string{"backend_groups", "backend_users", "backend_headers"}
	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s WHERE backend_id = %s", table, r.db.Placeholder(1))
		if _, err := tx.Exec(query, backendID); err != nil {
			return fmt.Errorf("failed to delete from %s: %w", table, err)
		}
	}
	return nil
}
