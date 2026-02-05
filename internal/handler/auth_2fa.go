package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/str"
	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/auth"
	"github.com/mylxsw/secure-proxy/internal/otp"
	"github.com/mylxsw/secure-proxy/internal/repository"
	"github.com/mylxsw/secure-proxy/internal/secure"
	"github.com/mylxsw/secure-proxy/internal/store"
)

// AuthHandlerWith2FA 带有两阶段认证支持的认证处理器
type AuthHandlerWith2FA struct {
	conf          *config.Config
	logger        log.Logger
	cookieManager *secure.CookieManager
	author        auth.Auth
	store         store.Store
	otpService    *otp.Service
	totpRepo      *repository.TOTPRepository
}

// NewAuthHandlerWith2FA 创建带 2FA 的认证处理器
func NewAuthHandlerWith2FA(
	conf *config.Config,
	author auth.Auth,
	cookieManager *secure.CookieManager,
	s store.Store,
	otpService *otp.Service,
	totpRepo *repository.TOTPRepository,
	logger log.Logger,
) *AuthHandlerWith2FA {
	return &AuthHandlerWith2FA{
		conf:          conf,
		author:        author,
		cookieManager: cookieManager,
		store:         s,
		otpService:    otpService,
		totpRepo:      totpRepo,
		logger:        logger,
	}
}

// RegisterHandlers 注册路由
func (h *AuthHandlerWith2FA) RegisterHandlers() {
	http.HandleFunc("/secure-proxy", h.buildStatusHandler())
	http.HandleFunc("/secure-proxy/auth/login", h.buildLoginHandler())
	http.HandleFunc("/secure-proxy/auth/logout", h.buildLogoutHandler())
	http.HandleFunc("/secure-proxy/auth", h.buildIndexPageHandler())
	http.HandleFunc("/secure-proxy/auth/2fa", h.build2FAPageHandler())
	http.HandleFunc("/secure-proxy/auth/2fa/verify", h.build2FAVerifyHandler())
	http.Handle("/secure-proxy/assets/", http.StripPrefix("/secure-proxy/assets/", http.FileServer(FS(false))))
}

func (h *AuthHandlerWith2FA) buildLogoutHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		if err := h.cookieManager.SetCookie(rw, config.UserAuthInfo{}); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte(fmt.Sprintf("Internal Server Error: %v", err)))
			return
		}

		http.Redirect(rw, r, "/", http.StatusSeeOther)
	}
}

func (h *AuthHandlerWith2FA) buildLoginHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		userType := r.FormValue("k0")
		username := strings.TrimSpace(r.FormValue("username"))
		password := strings.TrimSpace(r.FormValue("password"))

		// 忽略账号后缀
		if h.conf.Users.IgnoreAccountSuffix != "" {
			username = strings.TrimSuffix(username, h.conf.Users.IgnoreAccountSuffix)
		}

		if username == "" || password == "" {
			http.Redirect(rw, r, fmt.Sprintf("/secure-proxy/auth?username=%s&k0=%s&error=%s", username, userType, "用户名或密码不能为空"), http.StatusSeeOther)
			return
		}

		switch h.conf.AuthType {
		case "ldap":
			userType = "ldap"
		case "ldap+local":
			if !str.In(userType, []string{"local", "ldap"}) {
				http.Redirect(rw, r, fmt.Sprintf("/secure-proxy/auth?username=%s&k0=%s&error=%s", username, userType, "请求参数有误"), http.StatusSeeOther)
				return
			}
		default:
			userType = "local"
		}

		// 登录次数限制检查
		if err := h.store.UserCanLogin(userType, username); err != nil {
			h.logger.WithFields(log.Fields{"username": username, "userType": userType, "host": r.Host}).Warningf("user %s login failed: %v", username, err)
			http.Redirect(rw, r, fmt.Sprintf("/secure-proxy/auth?username=%s&k0=%s&error=%s", username, userType, err), http.StatusSeeOther)
			return
		}

		account := username
		if h.conf.AuthType == "ldap+local" {
			account = fmt.Sprintf("%s:%s", userType, username)
		}

		authedUser, err := h.author.Login(account, password)
		if err != nil {
			_ = h.store.UserLoginAttempt(userType, username)
			h.logger.WithFields(log.Fields{"username": username, "userType": userType, "host": r.Host}).Warningf("user %s login failed: %v", username, err)
			http.Redirect(rw, r, fmt.Sprintf("/secure-proxy/auth?username=%s&k0=%s&error=用户名或密码错误", username, userType), http.StatusSeeOther)
			return
		}

		// 检查是否需要 2FA 验证
		if h.conf.TwoFA.Enabled && h.totpRepo != nil {
			enabled, err := h.totpRepo.IsEnabled(authedUser.Type, authedUser.Account)
			if err != nil {
				h.logger.Errorf("failed to check 2FA status: %v", err)
			}

			if enabled {
				// 需要 2FA 验证，设置临时会话
				tempUserInfo := config.UserAuthInfo{
					UserType:     authedUser.Type,
					Username:     username,
					Account:      authedUser.Account,
					UUID:         authedUser.UUID,
					Name:         authedUser.Name,
					Groups:       authedUser.Groups,
					LoginHost:    r.Host,
					CreatedAt:    time.Now(),
					TwoFAPending: true, // 标记等待 2FA 验证
				}

				if err := h.cookieManager.SetCookie(rw, tempUserInfo); err != nil {
					rw.WriteHeader(http.StatusInternalServerError)
					_, _ = rw.Write([]byte(fmt.Sprintf("Internal Server Error: %v", err)))
					return
				}

				// 重定向到 2FA 验证页面
				http.Redirect(rw, r, "/secure-proxy/auth/2fa", http.StatusSeeOther)
				return
			}
		}

		// 无需 2FA，直接完成登录
		userAuthInfo := config.UserAuthInfo{
			UserType:  authedUser.Type,
			Username:  username,
			Account:   authedUser.Account,
			UUID:      authedUser.UUID,
			Name:      authedUser.Name,
			Groups:    authedUser.Groups,
			LoginHost: r.Host,
			CreatedAt: time.Now(),
		}

		h.logger.With(userAuthInfo).Infof("user %s login succeed", username)
		if err := h.cookieManager.SetCookie(rw, userAuthInfo); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte(fmt.Sprintf("Internal Server Error: %v", err)))
			return
		}

		http.Redirect(rw, r, "/", http.StatusSeeOther)
	}
}

func (h *AuthHandlerWith2FA) build2FAPageHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		userInfo, err := h.cookieManager.GetCookie(r)
		if err != nil || userInfo == nil || !userInfo.TwoFAPending {
			http.Redirect(rw, r, "/secure-proxy/auth", http.StatusSeeOther)
			return
		}

		errorMsg := r.URL.Query().Get("error")
		h.render2FAPage(rw, userInfo.Name, errorMsg)
	}
}

func (h *AuthHandlerWith2FA) build2FAVerifyHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Redirect(rw, r, "/secure-proxy/auth/2fa", http.StatusSeeOther)
			return
		}

		userInfo, err := h.cookieManager.GetCookie(r)
		if err != nil || userInfo == nil || !userInfo.TwoFAPending {
			http.Redirect(rw, r, "/secure-proxy/auth", http.StatusSeeOther)
			return
		}

		code := strings.TrimSpace(r.FormValue("code"))
		if code == "" {
			http.Redirect(rw, r, "/secure-proxy/auth/2fa?error=请输入验证码", http.StatusSeeOther)
			return
		}

		// 获取用户的 TOTP secret
		secret, err := h.totpRepo.GetSecret(userInfo.UserType, userInfo.Account)
		if err != nil || secret == "" {
			h.logger.Errorf("failed to get 2FA secret: %v", err)
			http.Redirect(rw, r, "/secure-proxy/auth/2fa?error=2FA配置异常", http.StatusSeeOther)
			return
		}

		// 验证 TOTP 码
		if !h.otpService.VerifyWithWindow(secret, code, 1) {
			http.Redirect(rw, r, "/secure-proxy/auth/2fa?error=验证码错误", http.StatusSeeOther)
			return
		}

		// 2FA 验证成功，清除待验证标记
		userInfo.TwoFAPending = false

		h.logger.With(userInfo).Infof("user %s 2FA verification succeed", userInfo.Username)
		if err := h.cookieManager.SetCookie(rw, *userInfo); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte(fmt.Sprintf("Internal Server Error: %v", err)))
			return
		}

		http.Redirect(rw, r, "/", http.StatusSeeOther)
	}
}

func (h *AuthHandlerWith2FA) render2FAPage(rw http.ResponseWriter, username, errorMsg string) {
	const twoFAPageTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>两阶段认证 - Secure Proxy</title>
    <link rel="stylesheet" href="/secure-proxy/assets/css/login.css">
    <style>
        .totp-input {
            text-align: center;
            font-size: 24px;
            letter-spacing: 8px;
            font-family: monospace;
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="login-box">
            <h2>两阶段认证</h2>
            <p>欢迎回来，<strong>{{.Username}}</strong></p>
            <p>请输入 Google Authenticator 中的验证码</p>
            {{if .Error}}
            <div class="error-message">{{.Error}}</div>
            {{end}}
            <form action="/secure-proxy/auth/2fa/verify" method="POST">
                <div class="form-group">
                    <input type="text" name="code" class="form-control totp-input" placeholder="000000" maxlength="6" pattern="[0-9]{6}" required autofocus>
                </div>
                <button type="submit" class="btn btn-primary">验证</button>
            </form>
            <div class="form-footer">
                <a href="/secure-proxy/auth/logout">返回登录</a>
            </div>
        </div>
    </div>
</body>
</html>`

	tmpl, err := template.New("2fa").Parse(twoFAPageTemplate)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(rw, map[string]interface{}{
		"Username": username,
		"Error":    errorMsg,
	})
}

func (h *AuthHandlerWith2FA) buildStatusHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		userAuthInfo, err := h.cookieManager.GetCookie(r)
		if err != nil || userAuthInfo == nil || userAuthInfo.Account == "" || userAuthInfo.TwoFAPending {
			http.Redirect(rw, r, "/secure-proxy/auth", http.StatusSeeOther)
			return
		}

		h.renderStatusPage(rw, userAuthInfo)
	}
}

func (h *AuthHandlerWith2FA) buildIndexPageHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		userAuthInfo, err := h.cookieManager.GetCookie(r)
		if err == nil && userAuthInfo != nil && userAuthInfo.Account != "" && !userAuthInfo.TwoFAPending {
			http.Redirect(rw, r, "/secure-proxy", http.StatusSeeOther)
			return
		}

		h.renderLoginPage(rw, r)
	}
}

func (h *AuthHandlerWith2FA) renderLoginPage(rw http.ResponseWriter, r *http.Request) {
	renderLoginPage(rw, r, h.conf)
}

func (h *AuthHandlerWith2FA) renderStatusPage(rw http.ResponseWriter, userAuthInfo *config.UserAuthInfo) {
	renderStatusPageFull(rw, userAuthInfo, h.conf.TwoFA.Enabled, h.conf.Database.Enabled)
}
