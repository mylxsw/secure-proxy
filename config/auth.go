package config

import (
	"crypto/md5"
	"fmt"
	"time"
)

type UserAuthInfo struct {
	UserType string `json:"user_type"`
	// Username 用户登录标识
	Username string `json:"username"`
	// Account 用户实际的账号唯一标识，非登录标识
	Account   string    `json:"account"`
	UUID      string    `json:"uuid"`
	Name      string    `json:"name"`
	Groups    []string  `json:"groups"`
	LoginHost string    `json:"login_host"`
	CreatedAt time.Time `json:"created_at"`
}

func (info UserAuthInfo) ID() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s,%s,%s,%s", info.UserType, info.Account, info.UUID, info.CreatedAt))))
}
