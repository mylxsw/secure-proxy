package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/str"
	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/backend"
	"github.com/mylxsw/secure-proxy/internal/repository"
	"github.com/mylxsw/secure-proxy/internal/secure"
)

// BackendAPIHandler Backend 管理 API 处理器
type BackendAPIHandler struct {
	conf          *config.Config
	logger        log.Logger
	cookieManager *secure.CookieManager
	backendRepo   *repository.BackendRepository
	backendStore  *backend.Store
	adminGroups   []string // 允许管理 backend 的用户组
}

// NewBackendAPIHandler 创建 Backend API 处理器
func NewBackendAPIHandler(
	conf *config.Config,
	cookieManager *secure.CookieManager,
	backendRepo *repository.BackendRepository,
	backendStore *backend.Store,
	logger log.Logger,
) *BackendAPIHandler {
	return &BackendAPIHandler{
		conf:          conf,
		cookieManager: cookieManager,
		backendRepo:   backendRepo,
		backendStore:  backendStore,
		logger:        logger,
		adminGroups:   []string{"admin", "devops"}, // 默认管理员组
	}
}

// RegisterHandlers 注册路由
func (h *BackendAPIHandler) RegisterHandlers() {
	http.HandleFunc("/secure-proxy/api/backends", h.handleBackends)
	http.HandleFunc("/secure-proxy/api/backends/", h.handleBackendByID)
}

// handleBackends 处理 backends 列表请求
func (h *BackendAPIHandler) handleBackends(w http.ResponseWriter, r *http.Request) {
	if !h.checkAdminAuth(w, r) {
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.listBackends(w, r)
	case http.MethodPost:
		h.createBackend(w, r)
	default:
		h.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleBackendByID 处理单个 backend 请求
func (h *BackendAPIHandler) handleBackendByID(w http.ResponseWriter, r *http.Request) {
	if !h.checkAdminAuth(w, r) {
		return
	}

	// 从 URL 路径获取 ID
	path := strings.TrimPrefix(r.URL.Path, "/secure-proxy/api/backends/")
	if path == "" {
		h.jsonError(w, "backend id is required", http.StatusBadRequest)
		return
	}

	// 支持 /backends/reload 特殊路径
	if path == "reload" && r.Method == http.MethodPost {
		h.reloadBackends(w, r)
		return
	}

	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		h.jsonError(w, "invalid backend id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getBackend(w, r, id)
	case http.MethodPut:
		h.updateBackend(w, r, id)
	case http.MethodDelete:
		h.deleteBackend(w, r, id)
	default:
		h.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// checkAdminAuth 检查管理员权限
func (h *BackendAPIHandler) checkAdminAuth(w http.ResponseWriter, r *http.Request) bool {
	userInfo, err := h.cookieManager.GetAuthUser(r)
	if err != nil || userInfo == nil {
		h.jsonError(w, "unauthorized", http.StatusUnauthorized)
		return false
	}

	// 检查用户是否在管理员组
	for _, group := range userInfo.Groups {
		if str.In(group, h.adminGroups) {
			return true
		}
	}

	h.jsonError(w, "forbidden: admin privileges required", http.StatusForbidden)
	return false
}

// listBackends 获取所有 backends
func (h *BackendAPIHandler) listBackends(w http.ResponseWriter, r *http.Request) {
	backends, err := h.backendRepo.GetAllIncludeDisabled()
	if err != nil {
		h.logger.Errorf("failed to list backends: %v", err)
		h.jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.jsonSuccess(w, backends)
}

// getBackend 获取单个 backend
func (h *BackendAPIHandler) getBackend(w http.ResponseWriter, r *http.Request, id int64) {
	backend, err := h.backendRepo.GetByID(id)
	if err != nil {
		h.logger.Errorf("failed to get backend: %v", err)
		h.jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if backend == nil {
		h.jsonError(w, "backend not found", http.StatusNotFound)
		return
	}

	h.jsonSuccess(w, backend)
}

// createBackend 创建 backend
func (h *BackendAPIHandler) createBackend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Host        string          `json:"host"`
		Upstream    string          `json:"upstream"`
		Privilege   string          `json:"privilege"`
		Enabled     bool            `json:"enabled"`
		Description string          `json:"description"`
		Groups      []string        `json:"groups"`
		Users       []string        `json:"users"`
		Headers     []config.Header `json:"headers"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Host == "" || req.Upstream == "" {
		h.jsonError(w, "host and upstream are required", http.StatusBadRequest)
		return
	}

	if req.Privilege == "" {
		req.Privilege = config.BackendPrivilegeInternal
	}

	backend := &repository.BackendModel{
		Host:        req.Host,
		Upstream:    req.Upstream,
		Privilege:   req.Privilege,
		Enabled:     req.Enabled,
		Description: req.Description,
		Groups:      req.Groups,
		Users:       req.Users,
		Headers:     req.Headers,
	}

	if err := h.backendRepo.Create(backend); err != nil {
		h.logger.Errorf("failed to create backend: %v", err)
		h.jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// 重新加载 backend store
	if h.backendStore != nil {
		h.backendStore.Reload()
	}

	h.logger.Infof("backend created: %s -> %s", req.Host, req.Upstream)
	h.jsonSuccess(w, backend)
}

// updateBackend 更新 backend
func (h *BackendAPIHandler) updateBackend(w http.ResponseWriter, r *http.Request, id int64) {
	var req struct {
		Host        string          `json:"host"`
		Upstream    string          `json:"upstream"`
		Privilege   string          `json:"privilege"`
		Enabled     bool            `json:"enabled"`
		Description string          `json:"description"`
		Groups      []string        `json:"groups"`
		Users       []string        `json:"users"`
		Headers     []config.Header `json:"headers"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Host == "" || req.Upstream == "" {
		h.jsonError(w, "host and upstream are required", http.StatusBadRequest)
		return
	}

	existing, err := h.backendRepo.GetByID(id)
	if err != nil {
		h.logger.Errorf("failed to get backend: %v", err)
		h.jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if existing == nil {
		h.jsonError(w, "backend not found", http.StatusNotFound)
		return
	}

	backend := &repository.BackendModel{
		ID:          id,
		Host:        req.Host,
		Upstream:    req.Upstream,
		Privilege:   req.Privilege,
		Enabled:     req.Enabled,
		Description: req.Description,
		Groups:      req.Groups,
		Users:       req.Users,
		Headers:     req.Headers,
	}

	if err := h.backendRepo.Update(backend); err != nil {
		h.logger.Errorf("failed to update backend: %v", err)
		h.jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// 重新加载 backend store
	if h.backendStore != nil {
		h.backendStore.Reload()
	}

	h.logger.Infof("backend updated: %s -> %s", req.Host, req.Upstream)
	h.jsonSuccess(w, backend)
}

// deleteBackend 删除 backend
func (h *BackendAPIHandler) deleteBackend(w http.ResponseWriter, r *http.Request, id int64) {
	existing, err := h.backendRepo.GetByID(id)
	if err != nil {
		h.logger.Errorf("failed to get backend: %v", err)
		h.jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if existing == nil {
		h.jsonError(w, "backend not found", http.StatusNotFound)
		return
	}

	if err := h.backendRepo.Delete(id); err != nil {
		h.logger.Errorf("failed to delete backend: %v", err)
		h.jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// 重新加载 backend store
	if h.backendStore != nil {
		h.backendStore.Reload()
	}

	h.logger.Infof("backend deleted: %s", existing.Host)
	h.jsonSuccess(w, map[string]interface{}{
		"message": "backend deleted successfully",
	})
}

// reloadBackends 重新加载 backends
func (h *BackendAPIHandler) reloadBackends(w http.ResponseWriter, r *http.Request) {
	if h.backendStore != nil {
		h.backendStore.Reload()
	}

	h.jsonSuccess(w, map[string]interface{}{
		"message": "backends reloaded successfully",
	})
}

func (h *BackendAPIHandler) jsonSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

func (h *BackendAPIHandler) jsonError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}
