package otp

import (
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/secure-proxy/config"
)

type Provider struct{}

func (Provider) Register(binder infra.Binder) {
	binder.MustSingleton(func(conf *config.Config) *Service {
		return NewService(conf.TwoFA.Issuer)
	})
}

func (Provider) ShouldLoad(conf *config.Config) bool {
	return conf.TwoFA.Enabled
}
