package store

import (
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier/infra"
)

type Provider struct{}

func (Provider) Register(binder infra.Binder) {
	binder.MustSingletonOverride(NewDefaultStore)

	log.Debugf("provider internal.store loaded")
}
