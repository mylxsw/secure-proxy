package config

import "time"

type UserAuthInfo struct {
	UserType  string    `json:"user_type"`
	Account   string    `json:"account"`
	UUID      string    `json:"uuid"`
	Name      string    `json:"name"`
	Groups    []string  `json:"groups"`
	LoginHost string    `json:"login_host"`
	CreatedAt time.Time `json:"created_at"`
}
