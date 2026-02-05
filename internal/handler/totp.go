package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/otp"
	"github.com/mylxsw/secure-proxy/internal/repository"
	"github.com/mylxsw/secure-proxy/internal/secure"
)

// TOTPHandler 处理 TOTP 相关的请求
type TOTPHandler struct {
	conf          *config.Config
	logger        log.Logger
	cookieManager *secure.CookieManager
	otpService    *otp.Service
	totpRepo      *repository.TOTPRepository
}

// NewTOTPHandler 创建 TOTP 处理器
func NewTOTPHandler(
	conf *config.Config,
	cookieManager *secure.CookieManager,
	otpService *otp.Service,
	totpRepo *repository.TOTPRepository,
	logger log.Logger,
) *TOTPHandler {
	return &TOTPHandler{
		conf:          conf,
		cookieManager: cookieManager,
		otpService:    otpService,
		totpRepo:      totpRepo,
		logger:        logger,
	}
}

// RegisterHandlers 注册 TOTP 相关的路由
func (h *TOTPHandler) RegisterHandlers() {
	http.HandleFunc("/secure-proxy/2fa/setup", h.buildSetupHandler())
	http.HandleFunc("/secure-proxy/2fa/verify", h.buildVerifyHandler())
	http.HandleFunc("/secure-proxy/2fa/enable", h.buildEnableHandler())
	http.HandleFunc("/secure-proxy/2fa/disable", h.buildDisableHandler())
	http.HandleFunc("/secure-proxy/2fa/status", h.buildStatusHandler())
}

// buildSetupHandler 生成 TOTP 密钥和 QR Code URI
func (h *TOTPHandler) buildSetupHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			h.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userInfo, err := h.cookieManager.GetAuthUser(r)
		if err != nil {
			h.jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// 检查是否已有配置
		existing, err := h.totpRepo.GetByUser(userInfo.UserType, userInfo.Account)
		if err != nil {
			h.logger.Errorf("failed to get totp config: %v", err)
			h.jsonError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		var secret string
		if existing != nil && existing.Secret != "" {
			// 如果已有未验证的配置，使用现有的 secret
			if !existing.Verified {
				secret = existing.Secret
			} else {
				// 如果已验证，需要先禁用才能重新设置
				h.jsonError(w, "2FA is already enabled. Please disable it first.", http.StatusBadRequest)
				return
			}
		} else {
			// 生成新的密钥
			secret, err = h.otpService.GenerateSecret()
			if err != nil {
				h.logger.Errorf("failed to generate totp secret: %v", err)
				h.jsonError(w, "internal server error", http.StatusInternalServerError)
				return
			}

			// 保存到数据库
			totp := &repository.UserTOTPModel{
				UserType: userInfo.UserType,
				Account:  userInfo.Account,
				Secret:   secret,
				Enabled:  false,
				Verified: false,
			}
			if err := h.totpRepo.Create(totp); err != nil {
				h.logger.Errorf("failed to save totp config: %v", err)
				h.jsonError(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}

		// 生成配置 URI
		account := fmt.Sprintf("%s@%s", userInfo.Account, userInfo.UserType)
		uri := h.otpService.GenerateProvisioningURI(account, secret)

		h.jsonSuccess(w, map[string]interface{}{
			"secret": secret,
			"uri":    uri,
		})
	}
}

// buildVerifyHandler 验证 TOTP 码并启用 2FA
func (h *TOTPHandler) buildVerifyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			h.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userInfo, err := h.cookieManager.GetAuthUser(r)
		if err != nil {
			h.jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		code := strings.TrimSpace(r.FormValue("code"))
		if code == "" {
			h.jsonError(w, "verification code is required", http.StatusBadRequest)
			return
		}

		// 获取用户的 TOTP 配置
		totp, err := h.totpRepo.GetByUser(userInfo.UserType, userInfo.Account)
		if err != nil {
			h.logger.Errorf("failed to get totp config: %v", err)
			h.jsonError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if totp == nil {
			h.jsonError(w, "please setup 2FA first", http.StatusBadRequest)
			return
		}

		// 验证 TOTP 码
		if !h.otpService.VerifyWithWindow(totp.Secret, code, 1) {
			h.jsonError(w, "invalid verification code", http.StatusBadRequest)
			return
		}

		// 标记为已验证并启用
		totp.Verified = true
		totp.Enabled = true
		if err := h.totpRepo.Update(totp); err != nil {
			h.logger.Errorf("failed to update totp config: %v", err)
			h.jsonError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		h.logger.Infof("user %s enabled 2FA", userInfo.Account)
		h.jsonSuccess(w, map[string]interface{}{
			"message": "2FA has been enabled successfully",
		})
	}
}

// buildEnableHandler 启用已验证的 2FA
func (h *TOTPHandler) buildEnableHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			h.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userInfo, err := h.cookieManager.GetAuthUser(r)
		if err != nil {
			h.jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		totp, err := h.totpRepo.GetByUser(userInfo.UserType, userInfo.Account)
		if err != nil {
			h.logger.Errorf("failed to get totp config: %v", err)
			h.jsonError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if totp == nil || !totp.Verified {
			h.jsonError(w, "please complete 2FA setup first", http.StatusBadRequest)
			return
		}

		if err := h.totpRepo.Enable(userInfo.UserType, userInfo.Account); err != nil {
			h.logger.Errorf("failed to enable 2FA: %v", err)
			h.jsonError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		h.logger.Infof("user %s enabled 2FA", userInfo.Account)
		h.jsonSuccess(w, map[string]interface{}{
			"message": "2FA has been enabled",
		})
	}
}

// buildDisableHandler 禁用 2FA
func (h *TOTPHandler) buildDisableHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			h.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userInfo, err := h.cookieManager.GetAuthUser(r)
		if err != nil {
			h.jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// 需要验证当前的 TOTP 码才能禁用
		code := strings.TrimSpace(r.FormValue("code"))
		if code == "" {
			h.jsonError(w, "verification code is required", http.StatusBadRequest)
			return
		}

		totp, err := h.totpRepo.GetByUser(userInfo.UserType, userInfo.Account)
		if err != nil {
			h.logger.Errorf("failed to get totp config: %v", err)
			h.jsonError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if totp == nil {
			h.jsonError(w, "2FA is not enabled", http.StatusBadRequest)
			return
		}

		// 验证 TOTP 码
		if !h.otpService.VerifyWithWindow(totp.Secret, code, 1) {
			h.jsonError(w, "invalid verification code", http.StatusBadRequest)
			return
		}

		// 删除 TOTP 配置
		if err := h.totpRepo.Delete(userInfo.UserType, userInfo.Account); err != nil {
			h.logger.Errorf("failed to disable 2FA: %v", err)
			h.jsonError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		h.logger.Infof("user %s disabled 2FA", userInfo.Account)
		h.jsonSuccess(w, map[string]interface{}{
			"message": "2FA has been disabled",
		})
	}
}

// buildStatusHandler 获取 2FA 状态
func (h *TOTPHandler) buildStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			h.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userInfo, err := h.cookieManager.GetAuthUser(r)
		if err != nil {
			h.jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		totp, err := h.totpRepo.GetByUser(userInfo.UserType, userInfo.Account)
		if err != nil {
			h.logger.Errorf("failed to get totp config: %v", err)
			h.jsonError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		enabled := totp != nil && totp.Enabled && totp.Verified

		h.jsonSuccess(w, map[string]interface{}{
			"enabled":       enabled,
			"setup_pending": totp != nil && !totp.Verified,
		})
	}
}

func (h *TOTPHandler) jsonSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

func (h *TOTPHandler) jsonError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}
