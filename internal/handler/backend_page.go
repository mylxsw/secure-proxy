package handler

import (
	"net/http"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/str"
	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/secure"
)

// BackendPageHandler Backend 管理页面处理器
type BackendPageHandler struct {
	conf          *config.Config
	logger        log.Logger
	cookieManager *secure.CookieManager
	adminGroups   []string
}

// NewBackendPageHandler 创建 Backend 管理页面处理器
func NewBackendPageHandler(
	conf *config.Config,
	cookieManager *secure.CookieManager,
	logger log.Logger,
) *BackendPageHandler {
	return &BackendPageHandler{
		conf:          conf,
		cookieManager: cookieManager,
		logger:        logger,
		adminGroups:   []string{"admin", "devops"},
	}
}

// RegisterHandlers 注册路由
func (h *BackendPageHandler) RegisterHandlers() {
	http.HandleFunc("/secure-proxy/admin/backends", h.handleBackendsPage)
}

// handleBackendsPage 处理后端管理页面
func (h *BackendPageHandler) handleBackendsPage(w http.ResponseWriter, r *http.Request) {
	userInfo, err := h.cookieManager.GetAuthUser(r)
	if err != nil || userInfo == nil {
		http.Redirect(w, r, "/secure-proxy/auth", http.StatusSeeOther)
		return
	}

	// 检查管理员权限
	isAdmin := false
	for _, group := range userInfo.Groups {
		if str.In(group, h.adminGroups) {
			isAdmin = true
			break
		}
	}

	if !isAdmin {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Access denied: admin privileges required"))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(backendManagementPage))
}

var backendManagementPage = `<!doctype html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Backend Management - Secure Proxy</title>
    <link href="/secure-proxy/assets/css/bootstrap.min.css" rel="stylesheet">
    <style>
        body { padding: 20px; }
        .container { max-width: 1200px; }
        .btn-group-sm > .btn { padding: 0.25rem 0.5rem; font-size: 0.875rem; }
        .table th, .table td { vertical-align: middle; }
        .badge { margin-right: 2px; }
        .modal-lg { max-width: 700px; }
        #toast-container { position: fixed; top: 20px; right: 20px; z-index: 9999; }
        .toast { min-width: 300px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="d-flex justify-content-between align-items-center mb-4">
            <h2>Backend Management</h2>
            <div>
                <button class="btn btn-outline-secondary me-2" onclick="reloadBackends()">
                    <span id="reload-icon">↻</span> Reload Cache
                </button>
                <button class="btn btn-primary" onclick="showCreateModal()">+ Add Backend</button>
                <a href="/secure-proxy" class="btn btn-outline-secondary ms-2">← Back</a>
            </div>
        </div>

        <div id="toast-container"></div>

        <table class="table table-striped table-hover">
            <thead class="table-dark">
                <tr>
                    <th>ID</th>
                    <th>Host</th>
                    <th>Upstream</th>
                    <th>Privilege</th>
                    <th>Groups</th>
                    <th>Users</th>
                    <th>Status</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody id="backends-table">
                <tr><td colspan="8" class="text-center">Loading...</td></tr>
            </tbody>
        </table>
    </div>

    <!-- Create/Edit Modal -->
    <div class="modal fade" id="backendModal" tabindex="-1">
        <div class="modal-dialog modal-lg">
            <div class="modal-content">
                <div class="modal-header">
                    <h5 class="modal-title" id="modalTitle">Add Backend</h5>
                    <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                </div>
                <div class="modal-body">
                    <form id="backendForm">
                        <input type="hidden" id="backendId">
                        <div class="row mb-3">
                            <div class="col-md-6">
                                <label class="form-label">Host *</label>
                                <input type="text" class="form-control" id="host" placeholder="example.domain.com" required>
                            </div>
                            <div class="col-md-6">
                                <label class="form-label">Upstream URL *</label>
                                <input type="text" class="form-control" id="upstream" placeholder="http://127.0.0.1:8080" required>
                            </div>
                        </div>
                        <div class="row mb-3">
                            <div class="col-md-6">
                                <label class="form-label">Privilege</label>
                                <select class="form-select" id="privilege">
                                    <option value="internal">Internal (all logged-in users)</option>
                                    <option value="private">Private (specific groups/users)</option>
                                </select>
                            </div>
                            <div class="col-md-6">
                                <label class="form-label">Status</label>
                                <select class="form-select" id="enabled">
                                    <option value="true">Enabled</option>
                                    <option value="false">Disabled</option>
                                </select>
                            </div>
                        </div>
                        <div class="mb-3">
                            <label class="form-label">Description</label>
                            <input type="text" class="form-control" id="description" placeholder="Optional description">
                        </div>
                        <div class="row mb-3">
                            <div class="col-md-6">
                                <label class="form-label">Allowed Groups (one per line)</label>
                                <textarea class="form-control" id="groups" rows="3" placeholder="admin&#10;devops"></textarea>
                            </div>
                            <div class="col-md-6">
                                <label class="form-label">Allowed Users (one per line)</label>
                                <textarea class="form-control" id="users" rows="3" placeholder="user1&#10;user2"></textarea>
                            </div>
                        </div>
                        <div class="mb-3">
                            <label class="form-label">Rewrite Headers (JSON format)</label>
                            <textarea class="form-control" id="headers" rows="2" placeholder='[{"key":"X-Custom-Header","value":"custom-value"}]'></textarea>
                            <small class="text-muted">Format: [{"key":"header-name","value":"header-value"}]</small>
                        </div>
                    </form>
                </div>
                <div class="modal-footer">
                    <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
                    <button type="button" class="btn btn-primary" onclick="saveBackend()">Save</button>
                </div>
            </div>
        </div>
    </div>

    <!-- Delete Confirmation Modal -->
    <div class="modal fade" id="deleteModal" tabindex="-1">
        <div class="modal-dialog">
            <div class="modal-content">
                <div class="modal-header">
                    <h5 class="modal-title">Confirm Delete</h5>
                    <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                </div>
                <div class="modal-body">
                    <p>Are you sure you want to delete backend <strong id="deleteHostName"></strong>?</p>
                    <p class="text-danger">This action cannot be undone.</p>
                </div>
                <div class="modal-footer">
                    <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
                    <button type="button" class="btn btn-danger" onclick="confirmDelete()">Delete</button>
                </div>
            </div>
        </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/js/bootstrap.bundle.min.js"></script>
    <script>
        let backendModal, deleteModal;
        let deleteBackendId = null;

        document.addEventListener('DOMContentLoaded', function() {
            backendModal = new bootstrap.Modal(document.getElementById('backendModal'));
            deleteModal = new bootstrap.Modal(document.getElementById('deleteModal'));
            loadBackends();
        });

        function showToast(message, type = 'success') {
            const container = document.getElementById('toast-container');
            const toast = document.createElement('div');
            toast.className = 'toast show';
            toast.innerHTML = '<div class="toast-header bg-' + type + ' text-white">' +
                '<strong class="me-auto">' + (type === 'success' ? 'Success' : 'Error') + '</strong>' +
                '<button type="button" class="btn-close btn-close-white" onclick="this.parentElement.parentElement.remove()"></button>' +
                '</div><div class="toast-body">' + message + '</div>';
            container.appendChild(toast);
            setTimeout(() => toast.remove(), 5000);
        }

        async function loadBackends() {
            try {
                const resp = await fetch('/secure-proxy/api/backends');
                const data = await resp.json();
                if (data.success) {
                    renderBackends(data.data || []);
                } else {
                    showToast(data.error || 'Failed to load backends', 'danger');
                }
            } catch (e) {
                showToast('Failed to load backends: ' + e.message, 'danger');
            }
        }

        function renderBackends(backends) {
            const tbody = document.getElementById('backends-table');
            if (!backends || backends.length === 0) {
                tbody.innerHTML = '<tr><td colspan="8" class="text-center text-muted">No backends configured</td></tr>';
                return;
            }

            tbody.innerHTML = backends.map(b => {
                const groups = (b.groups || []).map(g => '<span class="badge bg-info">' + escapeHtml(g) + '</span>').join(' ');
                const users = (b.users || []).map(u => '<span class="badge bg-secondary">' + escapeHtml(u) + '</span>').join(' ');
                const status = b.enabled ? '<span class="badge bg-success">Enabled</span>' : '<span class="badge bg-danger">Disabled</span>';
                return '<tr>' +
                    '<td>' + b.id + '</td>' +
                    '<td><strong>' + escapeHtml(b.host) + '</strong></td>' +
                    '<td><code>' + escapeHtml(b.upstream) + '</code></td>' +
                    '<td><span class="badge bg-' + (b.privilege === 'private' ? 'warning' : 'primary') + '">' + b.privilege + '</span></td>' +
                    '<td>' + (groups || '-') + '</td>' +
                    '<td>' + (users || '-') + '</td>' +
                    '<td>' + status + '</td>' +
                    '<td>' +
                        '<div class="btn-group btn-group-sm">' +
                            '<button class="btn btn-outline-primary" onclick="editBackend(' + b.id + ')">Edit</button>' +
                            '<button class="btn btn-outline-danger" onclick="showDeleteModal(' + b.id + ', \'' + escapeHtml(b.host) + '\')">Delete</button>' +
                        '</div>' +
                    '</td>' +
                '</tr>';
            }).join('');
        }

        function escapeHtml(text) {
            if (!text) return '';
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        function showCreateModal() {
            document.getElementById('modalTitle').textContent = 'Add Backend';
            document.getElementById('backendForm').reset();
            document.getElementById('backendId').value = '';
            document.getElementById('enabled').value = 'true';
            document.getElementById('privilege').value = 'internal';
            backendModal.show();
        }

        async function editBackend(id) {
            try {
                const resp = await fetch('/secure-proxy/api/backends/' + id);
                const data = await resp.json();
                if (data.success && data.data) {
                    const b = data.data;
                    document.getElementById('modalTitle').textContent = 'Edit Backend';
                    document.getElementById('backendId').value = b.id;
                    document.getElementById('host').value = b.host;
                    document.getElementById('upstream').value = b.upstream;
                    document.getElementById('privilege').value = b.privilege;
                    document.getElementById('enabled').value = b.enabled ? 'true' : 'false';
                    document.getElementById('description').value = b.description || '';
                    document.getElementById('groups').value = (b.groups || []).join('\n');
                    document.getElementById('users').value = (b.users || []).join('\n');
                    document.getElementById('headers').value = (b.headers && b.headers.length > 0) ? JSON.stringify(b.headers) : '';
                    backendModal.show();
                } else {
                    showToast(data.error || 'Failed to load backend', 'danger');
                }
            } catch (e) {
                showToast('Failed to load backend: ' + e.message, 'danger');
            }
        }

        async function saveBackend() {
            const id = document.getElementById('backendId').value;
            const host = document.getElementById('host').value.trim();
            const upstream = document.getElementById('upstream').value.trim();

            if (!host || !upstream) {
                showToast('Host and Upstream are required', 'danger');
                return;
            }

            const groups = document.getElementById('groups').value.split('\n').map(s => s.trim()).filter(s => s);
            const users = document.getElementById('users').value.split('\n').map(s => s.trim()).filter(s => s);

            let headers = [];
            const headersStr = document.getElementById('headers').value.trim();
            if (headersStr) {
                try {
                    headers = JSON.parse(headersStr);
                } catch (e) {
                    showToast('Invalid headers JSON format', 'danger');
                    return;
                }
            }

            const payload = {
                host: host,
                upstream: upstream,
                privilege: document.getElementById('privilege').value,
                enabled: document.getElementById('enabled').value === 'true',
                description: document.getElementById('description').value.trim(),
                groups: groups,
                users: users,
                headers: headers
            };

            try {
                const url = id ? '/secure-proxy/api/backends/' + id : '/secure-proxy/api/backends';
                const method = id ? 'PUT' : 'POST';
                const resp = await fetch(url, {
                    method: method,
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(payload)
                });
                const data = await resp.json();
                if (data.success) {
                    showToast(id ? 'Backend updated successfully' : 'Backend created successfully');
                    backendModal.hide();
                    loadBackends();
                } else {
                    showToast(data.error || 'Failed to save backend', 'danger');
                }
            } catch (e) {
                showToast('Failed to save backend: ' + e.message, 'danger');
            }
        }

        function showDeleteModal(id, host) {
            deleteBackendId = id;
            document.getElementById('deleteHostName').textContent = host;
            deleteModal.show();
        }

        async function confirmDelete() {
            if (!deleteBackendId) return;
            try {
                const resp = await fetch('/secure-proxy/api/backends/' + deleteBackendId, {method: 'DELETE'});
                const data = await resp.json();
                if (data.success) {
                    showToast('Backend deleted successfully');
                    deleteModal.hide();
                    loadBackends();
                } else {
                    showToast(data.error || 'Failed to delete backend', 'danger');
                }
            } catch (e) {
                showToast('Failed to delete backend: ' + e.message, 'danger');
            }
            deleteBackendId = null;
        }

        async function reloadBackends() {
            const icon = document.getElementById('reload-icon');
            icon.style.animation = 'spin 1s linear infinite';
            try {
                const resp = await fetch('/secure-proxy/api/backends/reload', {method: 'POST'});
                const data = await resp.json();
                if (data.success) {
                    showToast('Backend cache reloaded');
                } else {
                    showToast(data.error || 'Failed to reload', 'danger');
                }
            } catch (e) {
                showToast('Failed to reload: ' + e.message, 'danger');
            }
            setTimeout(() => icon.style.animation = '', 1000);
        }
    </script>
    <style>
        @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
    </style>
</body>
</html>`
