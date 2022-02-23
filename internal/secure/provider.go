package secure

import (
	"encoding/base64"
	"fmt"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/secure-proxy/config"
)

type Provider struct{}

func (pro Provider) Register(binder infra.Binder) {
	binder.MustSingletonOverride(func(conf *config.Config) (*CookieManager, error) {
		hashKey, err := base64.StdEncoding.DecodeString(conf.Session.HashKey)
		if err != nil {
			return nil, fmt.Errorf("invalid session.hash_key, must be base64 encoded string")
		}

		blockKey, err := base64.StdEncoding.DecodeString(conf.Session.BlockKey)
		if err != nil {
			return nil, fmt.Errorf("invalid session.block_key, must be base64 encoded string")
		}

		return NewCookieManager(conf.Session.CookieName, conf.Session.CookieDomain, hashKey, blockKey, conf.Session.MaxAge), nil
	})

	log.Debugf("provider internal.secure loaded")
}
