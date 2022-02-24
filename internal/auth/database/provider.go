package database

import (
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/go-utils/str"
	"github.com/mylxsw/secure-proxy/config"
)

type Provider struct{}

func (Provider) Register(cc infra.Binder) {
	cc.MustSingleton(New)
	log.Debugf("provider internal.auth.database loaded")
}

func (Provider) ShouldLoad(config *config.Config) bool {
	return str.InIgnoreCase(config.AuthType, []string{"database"})
}
