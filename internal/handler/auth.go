package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/str"
	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/auth"
	"github.com/mylxsw/secure-proxy/internal/secure"
	"github.com/mylxsw/secure-proxy/internal/store"
)

type AuthHandler struct {
	conf          *config.Config
	logger        log.Logger
	cookieManager *secure.CookieManager
	author        auth.Auth
	store         *store.Manager
}

func NewAuthHandler(conf *config.Config, author auth.Auth, cookieManager *secure.CookieManager, storeManager *store.Manager, logger log.Logger) *AuthHandler {
	return &AuthHandler{conf: conf, author: author, cookieManager: cookieManager, store: storeManager, logger: logger}
}

func (handler *AuthHandler) RegisterHandlers() {
	http.HandleFunc("/secure-proxy", handler.buildStatusHandler())
	http.HandleFunc("/secure-proxy/auth/login", handler.buildLoginHandler())
	http.HandleFunc("/secure-proxy/auth/logout", handler.buildLogoutHandler())
	http.HandleFunc("/secure-proxy/auth", handler.buildIndexPageHandler())
	http.Handle("/secure-proxy/assets/", http.StripPrefix("/secure-proxy/assets/", http.FileServer(FS(false))))
}

func (handler *AuthHandler) buildLogoutHandler() func(rw http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		if err := handler.cookieManager.SetCookie(rw, config.UserAuthInfo{}); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte(fmt.Sprintf("Internal Server Error: %v", err)))
			return
		}

		http.Redirect(rw, r, "/", http.StatusSeeOther)
	}
}

func (handler *AuthHandler) buildLoginHandler() func(rw http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		userType := r.FormValue("k0")
		username := strings.TrimSpace(r.FormValue("k1"))
		password := strings.TrimSpace(r.FormValue("k2"))

		// 要忽略的账号 suffix
		if handler.conf.Users.IgnoreAccountSuffix != "" {
			username = strings.TrimSuffix(username, handler.conf.Users.IgnoreAccountSuffix)
		}

		if username == "" || password == "" {
			http.Redirect(rw, r, fmt.Sprintf("/secure-proxy/auth?k1=%s&k0=%s&error=%s", username, userType, "用户名或密码不能为空"), http.StatusSeeOther)
			return
		}

		// 不同的鉴权类型，对账号的格式要求不同
		// misc：必须提供 k0(userType) 请求参数，可选值为 local|ldap
		// ldap：忽略用户请求的 k0(userType) 参数，强制设置为 ldap
		// 其它：忽略用户请求的 k0(userType) 参数，强制设置为 local
		switch handler.conf.AuthType {
		case "ldap":
			userType = "ldap"
		case "misc":
			if !str.In(userType, []string{"local", "ldap"}) {
				http.Redirect(rw, r, fmt.Sprintf("/secure-proxy/auth?k1=%s&k0=%s&error=%s", username, userType, "请求参数有误"), http.StatusSeeOther)
				return
			}

		default:
			userType = "local"
		}

		// 用户登录请求安全检查，登录次数限制检查
		if err := handler.store.UserCanLogin(userType, username); err != nil {
			log.WithFields(log.Fields{"username": username, "userType": userType, "host": r.Host}).Warningf("user %s login failed: %v", username, err)
			http.Redirect(rw, r, fmt.Sprintf("/secure-proxy/auth?k1=%s&k0=%s&error=%s", username, userType, err), http.StatusSeeOther)
			return
		}

		// misc 类型的鉴权模式下，用户名格式为 account=authType:username，后端鉴权模块会根据 authType 判断当前鉴权方式
		account := username
		if handler.conf.AuthType == "misc" {
			account = fmt.Sprintf("%s:%s", userType, username)
		}

		authedUser, err := handler.author.Login(account, password)
		if err != nil {
			_ = handler.store.UserLoginAttempt(userType, username)
			log.WithFields(log.Fields{"username": username, "userType": userType, "host": r.Host}).Warningf("user %s login failed: %v", username, err)
			http.Redirect(rw, r, fmt.Sprintf("/secure-proxy/auth?k1=%s&k0=%s&error=用户名或密码错误", username, userType), http.StatusSeeOther)
			return
		}

		// 鉴权成功，完成用户登录设置
		userAuthInfo := config.UserAuthInfo{
			UserType:  authedUser.Type,
			Account:   authedUser.Account,
			UUID:      authedUser.UUID,
			Name:      authedUser.Name,
			Groups:    authedUser.Groups,
			LoginHost: r.Host,
			CreatedAt: time.Now(),
		}

		log.With(userAuthInfo).Infof("user %s login succeed", username)
		if err := handler.cookieManager.SetCookie(rw, userAuthInfo); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte(fmt.Sprintf("Internal Server Error: %v", err)))
			return
		}

		http.Redirect(rw, r, "/", http.StatusSeeOther)
	}
}
