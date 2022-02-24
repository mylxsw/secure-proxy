package ldap_local

import (
	"strings"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/auth"
	"github.com/mylxsw/secure-proxy/internal/auth/ldap"
	"github.com/mylxsw/secure-proxy/internal/auth/local"
)

type Auth struct {
	logger    log.Logger
	ldapAuth  auth.Auth
	localAuth auth.Auth
}

func New(ldapConf *config.LDAP, localConf *config.Users) auth.Auth {
	return &Auth{logger: log.Module("auth:ldap_local"), ldapAuth: ldap.New(ldapConf, localConf), localAuth: local.New(localConf)}
}

func (provider *Auth) GetUser(username string) (*auth.AuthedUser, error) {
	if strings.HasPrefix(username, "local:") {
		return provider.localAuth.GetUser(strings.TrimPrefix(username, "local:"))
	}

	if strings.HasPrefix(username, "ldap:") {
		return provider.ldapAuth.GetUser(strings.TrimPrefix(username, "ldap:"))
	}

	rs, err := provider.localAuth.GetUser(username)
	if err != nil {
		return provider.ldapAuth.GetUser(username)
	}

	return rs, nil
}

func (provider *Auth) Login(username, password string) (*auth.AuthedUser, error) {
	if strings.HasPrefix(username, "local:") {
		return provider.localAuth.Login(strings.TrimPrefix(username, "local:"), password)
	}

	if strings.HasPrefix(username, "ldap:") {
		return provider.ldapAuth.Login(strings.TrimPrefix(username, "ldap:"), password)
	}

	rs, err := provider.localAuth.Login(username, password)
	if err != nil {
		return provider.ldapAuth.Login(username, password)
	}

	return rs, nil
}

func (provider *Auth) Users() ([]auth.AuthedUser, error) {
	localUsers, _ := provider.localAuth.Users()
	ldapUsers, _ := provider.ldapAuth.Users()

	localUsers = append(localUsers, ldapUsers...)
	return localUsers, nil
}
