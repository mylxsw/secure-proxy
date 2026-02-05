package database

import (
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/secure-proxy/config"
)

type Provider struct{}

func (Provider) Register(binder infra.Binder) {
	binder.MustSingleton(func(conf *config.Config) (*DB, error) {
		if !conf.Database.Enabled {
			return nil, nil
		}

		db, err := New(Config{
			Driver:          Driver(conf.Database.Driver),
			DSN:             conf.Database.DSN,
			MaxOpenConns:    conf.Database.MaxOpenConns,
			MaxIdleConns:    conf.Database.MaxIdleConns,
			ConnMaxLifetime: conf.Database.ConnMaxLifetime,
		})
		if err != nil {
			return nil, err
		}

		// 执行数据库迁移
		migration := NewMigration(db)
		if err := migration.Migrate(); err != nil {
			return nil, err
		}

		log.Debugf("database connected and migrated successfully, driver: %s", conf.Database.Driver)
		return db, nil
	})
}

func (Provider) ShouldLoad(conf *config.Config) bool {
	return conf.Database.Enabled
}
