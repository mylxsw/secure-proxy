package auth

import "errors"

type Auth interface {
	Login(username, password string) (*AuthedUser, error)
	GetUser(username string) (*AuthedUser, error)
	Users() ([]AuthedUser, error)
}

type AuthedUser struct {
	Type    string   `json:"type,omitempty"`
	UUID    string   `json:"uuid,omitempty"`
	Name    string   `json:"name,omitempty"`
	Account string   `json:"account,omitempty"`
	Groups  []string `json:"groups,omitempty"`
	Status  int8     `json:"status,omitempty"`
}

var ErrNoSuchUser = errors.New("user not found")
var ErrInvalidPassword = errors.New("invalid password")
var ErrAccountDisabled = errors.New("用户账户已禁用")
