package handler

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/graceful"
	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/auth"
	"github.com/mylxsw/secure-proxy/internal/backend"
	"github.com/mylxsw/secure-proxy/internal/otp"
	"github.com/mylxsw/secure-proxy/internal/repository"
	"github.com/mylxsw/secure-proxy/internal/secure"
	"github.com/mylxsw/secure-proxy/internal/store"
)

type Provider struct{}

func (pro Provider) Register(binder infra.Binder) {
	binder.MustSingletonOverride(func(conf *config.Config) (net.Listener, error) {
		return net.Listen("tcp", conf.Listen)
	})

	log.Debugf("provider internal.handler loaded")
}

func (pro Provider) Daemon(ctx context.Context, resolver infra.Resolver) {
	err := resolver.Resolve(func(
		conf *config.Config,
		listener net.Listener,
		gf graceful.Graceful,
		cookieManager *secure.CookieManager,
		s store.Store,
		author auth.Auth,
	) {
		// 解析可选依赖
		var backendStore *backend.Store
		var totpRepo *repository.TOTPRepository
		var backendRepo *repository.BackendRepository
		var otpService *otp.Service

		// 尝试解析数据库相关依赖
		_ = resolver.Resolve(func(bs *backend.Store) {
			backendStore = bs
		})
		_ = resolver.Resolve(func(tr *repository.TOTPRepository) {
			totpRepo = tr
		})
		_ = resolver.Resolve(func(br *repository.BackendRepository) {
			backendRepo = br
		})
		_ = resolver.Resolve(func(os *otp.Service) {
			otpService = os
		})

		// 创建 HTTP server
		options := DefaultOptions(conf)

		// 如果有 backendStore，设置 BackendProvider
		if backendStore != nil {
			options.SetBackendProvider(backendStore)
		}

		options.AuthHandler = cookieManager.BuildAuthHandler(func(user *config.UserAuthInfo) (bool, error) {
			// 如果用户待验证 2FA，拒绝访问
			if user.TwoFAPending {
				return false, fmt.Errorf("2FA verification pending")
			}

			return s.UserSessionValidate(user.ID(), func() (bool, error) {
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

		// 根据配置注册不同的 AuthHandler
		if conf.TwoFA.Enabled && otpService != nil && totpRepo != nil {
			NewAuthHandlerWith2FA(conf, author, cookieManager, s, otpService, totpRepo, log.Module("auth")).RegisterHandlers()
			NewTOTPHandler(conf, cookieManager, otpService, totpRepo, log.Module("totp")).RegisterHandlers()
		} else {
			NewAuthHandler(conf, author, cookieManager, s, log.Module("auth")).RegisterHandlers()
		}

		// 注册 Backend API（如果数据库启用）
		if conf.Database.Enabled && backendRepo != nil {
			NewBackendAPIHandler(conf, cookieManager, backendRepo, backendStore, log.Module("backend-api")).RegisterHandlers()
		}

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
