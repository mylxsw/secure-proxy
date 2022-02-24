package memory

import (
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/go-utils/str"
	"github.com/mylxsw/secure-proxy/config"
)

type Provider struct{}

func (Provider) Register(binder infra.Binder) {
	binder.MustSingletonOverride(New)

	log.Debugf("provider internal.store.memory loaded")
}

func (Provider) ShouldLoad(config *config.Config) bool {
	return str.InIgnoreCase(config.Cache.Driver, []string{"memory"})
}
