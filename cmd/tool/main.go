package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/gorilla/securecookie"
	"github.com/mylxsw/secure-proxy/internal/auth/ldap"
	"github.com/urfave/cli"
	"io/ioutil"
	"os"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/secure-proxy/config"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v2"
)

func main() {
	app := &cli.App{
		Name: "secure-proxy 工具库",
		Commands: []cli.Command{
			buildGenDefaultConfigCommand("generate-conf"),
			buildEncryptFileCommand("encrypt-file"),
			buildListLDAPUsersCommand("ldap-users"),
			buildSessionKeyGenerateCommand("session-key-generate"),
			buildEncryptPasswordCommand("encrypt-password"),
		},
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func buildGenDefaultConfigCommand(name string) cli.Command {
	return cli.Command{
		Name:  name,
		Usage: "生成默认配置文件",
		Action: func(c *cli.Context) error {
			conf := config.BuildDefaultConfig()
			conf.LogPath = "/var/log/secure-proxy-%s.log"
			conf.Session.HashKey = base64.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32))
			conf.Session.BlockKey = base64.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32))
			conf.Session.CookieDomain = ".example.com"

			conf.Backends = append(conf.Backends, config.Backend{
				Host:      "kibana.example.com",
				Upstream:  "http://127.0.0.1:5601",
				Privilege: "private",
				Groups:    []string{"admin"},
			})
			conf.Backends = append(conf.Backends, config.Backend{
				Host:      "consul.example.com",
				Upstream:  "http://127.0.0.1:8500",
				Privilege: "internal",
			})

			conf.LDAP.URL = "ldap://127.0.0.1:389"
			conf.LDAP.UserFilter = "CN=Users,DC=example,DC=com"
			conf.LDAP.UID = "sAMAccountName"
			conf.LDAP.DisplayName = "displayName"
			conf.LDAP.Username = "admin"
			conf.LDAP.Password = "admin"
			conf.LDAP.BaseDN = "dc=example,dc=com"

			conf.Redis.Addr = "127.0.0.1:6379"
			conf.Redis.Addr = "redisPassword"

			conf.ClientRealIPHeader = "X-Real-IP"
			conf.Listen = "127.0.0.1:8080"
			conf.AuthType = "local"

			conf.Users.IgnoreAccountSuffix = "@example.com"
			password, _ := bcrypt.GenerateFromPassword([]byte("lixiaoyao"), bcrypt.DefaultCost)
			conf.Users.Local = append(conf.Users.Local, config.LocalUser{
				Name:     "李逍遥",
				Account:  "lixiaoyao",
				Password: string(password),
				Groups:   []string{"admin"},
				Algo:     "bcrypt",
			})
			conf.Users.Local = append(conf.Users.Local, config.LocalUser{
				Name:     "重楼",
				Account:  "chonglou",
				Password: "chonglou",
				Groups:   []string{"devops", "admin"},
				Algo:     "plain",
			})
			conf.Users.Local = append(conf.Users.Local, config.LocalUser{
				Name:     "景天",
				Account:  "jingtian",
				Password: "amluZ3RpYW4=",
				Groups:   []string{"devops"},
				Algo:     "base64",
			})

			generated, err := yaml.Marshal(conf)
			if err != nil {
				return err
			}

			fmt.Printf("%s\n", string(generated))
			return nil
		},
	}
}

func buildEncryptPasswordCommand(name string) cli.Command {
	return cli.Command{
		Name:  name,
		Usage: "使用 bcrypt 算法加密密码",
		Action: func(c *cli.Context) error {
			password := c.String("password")
			if password == "" {
				return errors.New("password is required")
			}

			encrypted, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("encrypt password failed: %v", err)
			}

			fmt.Printf("密码原文：%s\nBcrypt密文：%s\n", password, string(encrypted))
			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "password",
				Usage: "要加密的密码",
			},
		},
	}
}

func buildSessionKeyGenerateCommand(name string) cli.Command {
	return cli.Command{
		Name:  name,
		Usage: "随机生成 session.hash_key 和 session.block_key",
		Action: func(c *cli.Context) error {
			fmt.Printf("HASH_KEY: %s\n", base64.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32)))
			fmt.Printf("BLOCK_KEY: %s\n", base64.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32)))
			return nil
		},
	}
}

func buildEncryptFileCommand(name string) cli.Command {
	return cli.Command{
		Name:   name,
		Usage:  "加密配置文件，将配置文件中所有的明文密码部分使用 bcrypt 算法加密",
		Action: encryptFileCommand,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "conf",
				Value: "./secure-proxy.yaml",
				Usage: "配置文件路径",
			},
			cli.BoolFlag{
				Name:  "overwrite",
				Usage: "是否直接覆盖源文件，不使用该选项时将直接输出到控制台",
			},
		},
	}
}

func encryptFileCommand(c *cli.Context) error {
	configPath := c.String("conf")
	overwrite := c.Bool("overwrite")

	conf, err := config.LoadConfFromFile(configPath)
	if err != nil {
		panic(err)
	}

	for i, user := range conf.Users.Local {
		if user.Algo == "" {
			encrypted, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
			if err != nil {
				log.Errorf("encrypt password for %s failed: %v", user.Account, err)
				continue
			}

			log.With(user).Debugf("password encrypted for %s", user.Account)

			conf.Users.Local[i].Algo = "bcrypt"
			conf.Users.Local[i].Password = string(encrypted)
		}
	}

	generated, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}

	if !overwrite {
		fmt.Printf("%s\n", string(generated))
		return nil
	}

	if err := ioutil.WriteFile(configPath, generated, os.ModePerm); err != nil {
		return err
	}

	return nil
}

func buildListLDAPUsersCommand(name string) cli.Command {
	return cli.Command{
		Name:   name,
		Usage:  "根据配置文件中的配置，输出查询到的 LDAP 用户清单",
		Action: listLDAPUsersCommand,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "conf",
				Value: "./secure-proxy.yaml",
				Usage: "配置文件路径",
			},
		},
	}
}

func listLDAPUsersCommand(c *cli.Context) error {
	configPath := c.String("conf")

	conf, err := config.LoadConfFromFile(configPath)
	if err != nil {
		return err
	}

	ldapAuth := ldap.New(&conf.LDAP, &conf.Users)
	users, err := ldapAuth.Users()
	if err != nil {
		return err
	}

	for _, user := range users {
		log.With(user).Infof("user found")
	}

	return nil
}
