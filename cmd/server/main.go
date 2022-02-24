package main

import (
	"context"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/mylxsw/asteria/formatter"
	"github.com/mylxsw/asteria/level"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/asteria/writer"
	"github.com/mylxsw/glacier/infra"
	"github.com/mylxsw/glacier/starter/application"
	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/auth/ldap"
	"github.com/mylxsw/secure-proxy/internal/auth/ldap_local"
	"github.com/mylxsw/secure-proxy/internal/auth/local"
	"github.com/mylxsw/secure-proxy/internal/handler"
	"github.com/mylxsw/secure-proxy/internal/secure"
	"github.com/mylxsw/secure-proxy/internal/store"
	"time"
)

func main() {
	log.All().LogFormatter(formatter.NewJSONFormatter())

	app := application.Create("v2").WithShutdownTimeoutFlagSupport(5 * time.Second)
	app.AddStringFlag("conf", "secure-proxy.yaml", "配置文件路径")
	app.Singleton(func(c infra.FlagContext) (*config.Config, error) {
		return config.LoadConfFromFile(c.String("conf"))
	})

	app.AfterInitialized(func(resolver infra.Resolver) error {
		return resolver.Resolve(func(conf *config.Config) {
			if conf.LogPath != "" {
				log.All().LogWriter(writer.NewDefaultRotatingFileWriter(context.TODO(), func(le level.Level, module string) string {
					return fmt.Sprintf(conf.LogPath, fmt.Sprintf("%s-%s", le.GetLevelName(), time.Now().Format("20060102")))
				}))
			}
		})
	})

	app.Provider(
		config.Provider{},
		secure.Provider{},
		store.Provider{},
		ldap.Provider{},
		local.Provider{},
		ldap_local.Provider{},
		handler.Provider{},
	)

	application.MustRun(app)
}
