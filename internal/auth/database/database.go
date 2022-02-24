package database

import (
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/auth"
)

type Auth struct {
	conf   *config.Config
	logger log.Logger
}

func New(conf *config.Config, logger log.Logger) auth.Auth {
	return &Auth{
		conf:   conf,
		logger: logger,
	}
}

func (auth *Auth) Login(username, password string) (*auth.AuthedUser, error) {
	//TODO implement me
	panic("implement me")
}

func (auth *Auth) GetUser(username string) (*auth.AuthedUser, error) {
	//TODO implement me
	panic("implement me")
}

func (auth *Auth) Users() ([]auth.AuthedUser, error) {
	//TODO implement me
	panic("implement me")
}
