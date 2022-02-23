package ldap

import (
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/go-utils/str"
	"github.com/mylxsw/secure-proxy/config"
)

type Provider struct{}

func (pro Provider) Register(cc infra.Binder) {
	cc.MustSingletonOverride(New)
	log.Debugf("provider internal.auth.ldap loaded")
}

func (p Provider) ShouldLoad(config *config.Config) bool {
	return str.InIgnoreCase(config.AuthType, []string{"ldap"})
}