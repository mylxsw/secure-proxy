package backend

import (
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/repository"
)

type Provider struct{}

func (Provider) Register(binder infra.Binder) {
	binder.MustSingleton(func(conf *config.Config, repo *repository.BackendRepository) *Store {
		return NewStore(conf, repo)
	})
}
