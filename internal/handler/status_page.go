package handler

import (
	"fmt"
	"net/http"

	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/template"
)

func (handler *AuthHandler) buildStatusHandler() func(rw http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("Content-Type", "text/html")

		userAuthInfo, err := handler.cookieManager.GetCookie(r)
		if err != nil {
			http.Redirect(rw, r, "/secure-proxy/auth", http.StatusSeeOther)
			return
		}

		renderStatusPage(rw, userAuthInfo, false)
	}
}

// renderStatusPage 渲染状态页面
func renderStatusPage(rw http.ResponseWriter, userAuthInfo *config.UserAuthInfo, twoFAEnabled bool) {
	renderStatusPageFull(rw, userAuthInfo, twoFAEnabled, false)
}

// renderStatusPageFull 渲染状态页面（完整参数）
func renderStatusPageFull(rw http.ResponseWriter, userAuthInfo *config.UserAuthInfo, twoFAEnabled bool, dbEnabled bool) {
	rw.Header().Add("Content-Type", "text/html")

	data := map[string]interface{}{
		"Account":      userAuthInfo.Account,
		"Name":         userAuthInfo.Name,
		"CreatedAt":    userAuthInfo.CreatedAt,
		"Groups":       userAuthInfo.Groups,
		"TwoFAEnabled": twoFAEnabled,
		"DBEnabled":    dbEnabled,
	}

	templateStr := statusPage
	if twoFAEnabled {
		templateStr = statusPageWith2FA
	}

	if err := template.Render(templateStr, rw, data); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte(fmt.Sprintf("Internal Server Error: %v", err)))
		return
	}
}

var statusPage = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Unified Authentication - Status</title>

    <link href="/secure-proxy/assets/css/bootstrap.min.css" rel="stylesheet">

    <style>
      .bd-placeholder-img {
        font-size: 1.125rem;
        text-anchor: middle;
        -webkit-user-select: none;
        -moz-user-select: none;
        user-select: none;
      }

      @media (min-width: 768px) {
        .bd-placeholder-img-lg {
          font-size: 3.5rem;
        }
      }
    </style>

    <link href="/secure-proxy/assets/style.css?v=20220118" rel="stylesheet">
  </head>
  <body class="text-center">

  <main class="form-signin">
	<div class="card card-default">
		<div class="card-header">Status</div>
		<div class="card-body text-left">
			<p>Current Login Information:</p>
			<li>User: <b>{{ .Account }}({{ .Name }})</b></li>
			<li>Login Time: <b>{{ .CreatedAt }}</b></li>
			<li>User Groups:
			{{- range $i, $group := .Groups }}
			<span class="badge rounded-pill bg-success">{{ $group }}</span>
			{{- end }}
			</li>
			<p class="mt-2"><a href="/secure-proxy/auth/logout" class="btn btn-primary">Logout</a></p>
		</div>
		<p class="mt-1 mb-3 text-muted">© 2022</p>
	</div>
  </main>

<script>

</script>

  </body>
</html>`

var statusPageWith2FA = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Unified Authentication - Status</title>

    <link href="/secure-proxy/assets/css/bootstrap.min.css" rel="stylesheet">

    <style>
      .bd-placeholder-img {
        font-size: 1.125rem;
        text-anchor: middle;
        -webkit-user-select: none;
        -moz-user-select: none;
        user-select: none;
      }

      @media (min-width: 768px) {
        .bd-placeholder-img-lg {
          font-size: 3.5rem;
        }
      }
      .two-fa-section {
        margin-top: 20px;
        padding-top: 20px;
        border-top: 1px solid #dee2e6;
      }
    </style>

    <link href="/secure-proxy/assets/style.css?v=20220118" rel="stylesheet">
  </head>
  <body class="text-center">

  <main class="form-signin">
	<div class="card card-default">
		<div class="card-header">Status</div>
		<div class="card-body text-left">
			<p>Current Login Information:</p>
			<li>User: <b>{{ .Account }}({{ .Name }})</b></li>
			<li>Login Time: <b>{{ .CreatedAt }}</b></li>
			<li>User Groups:
			{{- range $i, $group := .Groups }}
			<span class="badge rounded-pill bg-success">{{ $group }}</span>
			{{- end }}
			</li>

			<div class="two-fa-section">
				<p><strong>Two-Factor Authentication (2FA)</strong></p>
				<div id="2fa-status">Loading...</div>
				<div id="2fa-setup" style="display:none;">
					<p>Scan the QR code with Google Authenticator:</p>
					<div id="qr-code"></div>
					<p>Or enter this secret manually: <code id="2fa-secret"></code></p>
					<div class="mt-3">
						<input type="text" id="verify-code" class="form-control mb-2" placeholder="Enter 6-digit code" maxlength="6">
						<button onclick="verify2FA()" class="btn btn-success btn-sm">Verify & Enable</button>
					</div>
				</div>
				<div id="2fa-actions"></div>
			</div>

			<div class="mt-3">
				<a href="/secure-proxy/auth/logout" class="btn btn-primary">Logout</a>
				{{ if .DBEnabled }}
				<a href="/secure-proxy/admin/backends" class="btn btn-outline-secondary ms-2">Backend Management</a>
				{{ end }}
			</div>
		</div>
		<p class="mt-1 mb-3 text-muted">© 2022</p>
	</div>
  </main>

<script src="https://cdn.jsdelivr.net/npm/qrcode@1.5.3/build/qrcode.min.js"></script>
<script>
async function check2FAStatus() {
	try {
		const resp = await fetch('/secure-proxy/2fa/status');
		const data = await resp.json();
		if (data.success) {
			const statusDiv = document.getElementById('2fa-status');
			const actionsDiv = document.getElementById('2fa-actions');
			if (data.data.enabled) {
				statusDiv.innerHTML = '<span class="badge bg-success">Enabled</span>';
				actionsDiv.innerHTML = '<button onclick="disable2FA()" class="btn btn-warning btn-sm mt-2">Disable 2FA</button>';
			} else if (data.data.setup_pending) {
				statusDiv.innerHTML = '<span class="badge bg-warning">Setup Pending</span>';
				setup2FA();
			} else {
				statusDiv.innerHTML = '<span class="badge bg-secondary">Not Enabled</span>';
				actionsDiv.innerHTML = '<button onclick="setup2FA()" class="btn btn-primary btn-sm mt-2">Enable 2FA</button>';
			}
		}
	} catch (e) {
		document.getElementById('2fa-status').innerHTML = '<span class="badge bg-danger">Error</span>';
	}
}

async function setup2FA() {
	try {
		const resp = await fetch('/secure-proxy/2fa/setup', {method: 'POST'});
		const data = await resp.json();
		if (data.success) {
			document.getElementById('2fa-setup').style.display = 'block';
			document.getElementById('2fa-secret').textContent = data.data.secret;
			document.getElementById('2fa-actions').innerHTML = '';
			QRCode.toCanvas(document.getElementById('qr-code'), data.data.uri, {width: 200});
		} else {
			alert(data.error || 'Failed to setup 2FA');
		}
	} catch (e) {
		alert('Failed to setup 2FA');
	}
}

async function verify2FA() {
	const code = document.getElementById('verify-code').value;
	if (!code || code.length !== 6) {
		alert('Please enter a 6-digit code');
		return;
	}
	try {
		const resp = await fetch('/secure-proxy/2fa/verify', {
			method: 'POST',
			headers: {'Content-Type': 'application/x-www-form-urlencoded'},
			body: 'code=' + code
		});
		const data = await resp.json();
		if (data.success) {
			alert('2FA enabled successfully!');
			location.reload();
		} else {
			alert(data.error || 'Verification failed');
		}
	} catch (e) {
		alert('Verification failed');
	}
}

async function disable2FA() {
	const code = prompt('Enter your current 2FA code to disable:');
	if (!code) return;
	try {
		const resp = await fetch('/secure-proxy/2fa/disable', {
			method: 'POST',
			headers: {'Content-Type': 'application/x-www-form-urlencoded'},
			body: 'code=' + code
		});
		const data = await resp.json();
		if (data.success) {
			alert('2FA disabled successfully!');
			location.reload();
		} else {
			alert(data.error || 'Failed to disable 2FA');
		}
	} catch (e) {
		alert('Failed to disable 2FA');
	}
}

check2FAStatus();
</script>

  </body>
</html>`
