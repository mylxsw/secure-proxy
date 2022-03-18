package handler

import (
	"context"
	"fmt"
	"github.com/mylxsw/secure-proxy/internal/store"
	"net"
	"net/http"
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/graceful"
	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/auth"
	"github.com/mylxsw/secure-proxy/internal/secure"
)

type Provider struct{}

func (pro Provider) Register(binder infra.Binder) {
	binder.MustSingletonOverride(func(conf *config.Config) (net.Listener, error) {
		return net.Listen("tcp", conf.Listen)
	})

	log.Debugf("provider internal.handler loaded")
}

func (pro Provider) Daemon(ctx context.Context, resolver infra.Resolver) {
	err := resolver.Resolve(func(conf *config.Config, listener net.Listener, gf graceful.Graceful, cookieManager *secure.CookieManager, s store.Store, author auth.Auth) {
		// 创建 HTTP server
		options := DefaultOptions(conf)
		options.AuthHandler = cookieManager.BuildAuthHandler(func(user *config.UserAuthInfo) (bool, error) {
			return s.UserSessionValidate(user.ID(), func() (bool, error) {
				// ldap_local 类型的鉴权模式下，用户名格式为 account=authType:username，后端鉴权模块会根据 authType 判断当前鉴权方式
				account := user.Username
				if conf.AuthType == "ldap+local" {
					account = fmt.Sprintf("%s:%s", user.UserType, user.Username)
				}

				if _, err := author.GetUser(account); err != nil {
					if err == auth.ErrAccountDisabled {
						return false, nil
					}

					return false, err
				}

				return true, nil
			})
		})

		NewAuthHandler(conf, author, cookieManager, s, log.Module("auth")).RegisterHandlers()
		NewProxyHandler(options, author, log.Module("proxy")).RegisterHandlers()

		srv := &http.Server{}
		gf.AddShutdownHandler(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := srv.Shutdown(ctx); err != nil {
				log.Errorf("shutdown http server failed: %s", err)
			}
		})

		log.Debugf("http server started, listen on %s", listener.Addr().String())

		if err := srv.Serve(listener); err != nil {
			log.Warningf("http server stopped: %v", err)
		}
	})
	if err != nil {
		log.Errorf("handler.daemon exited with error: %v", err)
	}
}
