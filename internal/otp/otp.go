package otp

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"github.com/xlzd/gotp"
)

const totpInterval = 30 // TOTP 时间间隔（秒）

// Service TOTP 服务
type Service struct {
	issuer string
}

// NewService 创建 TOTP 服务
func NewService(issuer string) *Service {
	return &Service{issuer: issuer}
}

// GenerateSecret 生成 TOTP 密钥
func (s *Service) GenerateSecret() (string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("failed to generate secret: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// GenerateProvisioningURI 生成用于 Google Authenticator 的配置 URI
func (s *Service) GenerateProvisioningURI(account, secret string) string {
	totp := gotp.NewDefaultTOTP(secret)
	return totp.ProvisioningUri(account, s.issuer)
}

// getCurrentTimestamp 获取当前 TOTP 时间步
func getCurrentTimestamp() int {
	return int(time.Now().Unix() / totpInterval)
}

// Verify 验证 TOTP 验证码
func (s *Service) Verify(secret, code string) bool {
	// 去除空格
	code = strings.ReplaceAll(code, " ", "")

	totp := gotp.NewDefaultTOTP(secret)
	return totp.Verify(code, getCurrentTimestamp())
}

// VerifyWithWindow 验证 TOTP 验证码（带时间窗口容差）
// window 表示允许的时间偏移步数，例如 window=1 表示允许前后 30 秒的误差
func (s *Service) VerifyWithWindow(secret, code string, window int) bool {
	// 去除空格
	code = strings.ReplaceAll(code, " ", "")

	totp := gotp.NewDefaultTOTP(secret)
	timestamp := getCurrentTimestamp()

	// 检查当前时间步
	if totp.Verify(code, timestamp) {
		return true
	}

	// 检查时间窗口内的其他时间步
	for i := 1; i <= window; i++ {
		// 检查前面的时间步
		if totp.Verify(code, timestamp-i) {
			return true
		}
		// 检查后面的时间步
		if totp.Verify(code, timestamp+i) {
			return true
		}
	}

	return false
}

// GetCurrentCode 获取当前验证码（仅用于测试）
func (s *Service) GetCurrentCode(secret string) string {
	totp := gotp.NewDefaultTOTP(secret)
	return totp.Now()
}
