package repository

import (
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/database"
)

type Provider struct{}

func (Provider) Register(binder infra.Binder) {
	binder.MustSingleton(func(db *database.DB) *BackendRepository {
		if db == nil {
			return nil
		}
		return NewBackendRepository(db)
	})

	binder.MustSingleton(func(db *database.DB) *TOTPRepository {
		if db == nil {
			return nil
		}
		return NewTOTPRepository(db)
	})
}

func (Provider) ShouldLoad(conf *config.Config) bool {
	return conf.Database.Enabled
}
