package ldap_local

import (
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/go-utils/str"
	"github.com/mylxsw/secure-proxy/config"
)

type Provider struct{}

func (p Provider) Register(cc infra.Binder) {
	cc.MustSingletonOverride(New)

	log.Debugf("provider internal.auth.ldap+local loaded")
}

func (p Provider) ShouldLoad(config *config.Config) bool {
	return str.InIgnoreCase(config.AuthType, []string{"ldap+local"})
}
