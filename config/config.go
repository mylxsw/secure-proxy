package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/mylxsw/go-utils/file"
	"github.com/mylxsw/go-utils/str"
	"gopkg.in/yaml.v2"
)

// Config 系统配置
type Config struct {
	Verbose            bool   `json:"verbose" yaml:"verbose,omitempty"`
	Listen             string `json:"listen" yaml:"listen,omitempty"`
	AuthType           string `json:"auth_type" yaml:"auth_type,omitempty"`
	ClientRealIPHeader string `json:"client_ip_header" yaml:"client_ip_header,omitempty"`
	LogPath            string `json:"log_path" yaml:"log_path,omitempty"`

	Users    Users     `json:"users,omitempty" yaml:"users,omitempty"`
	Backends []Backend `json:"backends,omitempty" yaml:"backends,omitempty"`

	Session Session `json:"session" yaml:"session,omitempty"`
	LDAP    LDAP    `json:"ldap" yaml:"ldap,omitempty"`
	Redis   Redis   `json:"redis" yaml:"redis,omitempty"`
}

// populateDefault 填充默认值
func (conf Config) populateDefault() Config {
	if conf.AuthType == "" {
		conf.AuthType = "misc"
	}

	if conf.Session.CookieName == "" {
		conf.Session.CookieName = "secure-proxy-auth"
	}

	if conf.Session.MaxAge == 0 {
		// 默认会话有效时间为 24小时
		conf.Session.MaxAge = 24 * 60 * 60
	}

	if conf.Listen == "" {
		conf.Listen = ":8080"
	}

	if conf.LDAP.DisplayName == "" {
		conf.LDAP.DisplayName = "displayName"
	}

	if conf.LDAP.UID == "" {
		conf.LDAP.UID = "sAMAccountName"
	}

	if conf.Redis.Addr == "" {
		conf.Redis.Addr = "127.0.0.1:6379"
	}

	if !strings.Contains(conf.Redis.Addr, ":") {
		conf.Redis.Addr = fmt.Sprintf("%s:6379", conf.Redis.Addr)
	}

	return conf
}

// validate 配置合法性检查
func (conf Config) validate() error {
	if conf.Session.HashKey == "" {
		return fmt.Errorf("session.hash_key is required")
	}
	if conf.Session.BlockKey == "" {
		return fmt.Errorf("session.block_key is required")
	}

	if conf.Session.MaxAge < 0 {
		return fmt.Errorf("session.max_age must be positive")
	}

	if !str.In(conf.AuthType, []string{"misc", "ldap", "local"}) {
		return fmt.Errorf("invalid auth_type: must be one of misc|local|ldap")
	}

	for _, backend := range conf.Backends {
		if err := backend.validate(); err != nil {
			return err
		}
	}

	return nil
}

const (
	// BackendPrivilegeInternal 内部访问权限，登录用户皆可访问
	BackendPrivilegeInternal = "internal"
	// BackendPrivilegePrivate 私有访问权限，授权用户访问
	BackendPrivilegePrivate = "private"
)

// Backend 后端服务配置
type Backend struct {
	Host     string `json:"host" yaml:"host"`
	Upstream string `json:"upstream" yaml:"upstream"`
	// Privilege 访问权限，默认为 private，只允许指定的 group/user 访问
	Privilege string   `json:"privilege,omitempty" yaml:"privilege,omitempty"`
	Groups    []string `json:"groups,omitempty" yaml:"groups,omitempty"`
	Users     []string `json:"users,omitempty" yaml:"users,omitempty"`
}

// validate Backend 配置校验
func (backend Backend) validate() error {
	if backend.Privilege == "" {
		backend.Privilege = BackendPrivilegeInternal
	}

	if backend.Host == "" || backend.Upstream == "" {
		return fmt.Errorf("host and upstream is required for %s:%s", backend.Host, backend.Upstream)
	}

	if !str.In(backend.Privilege, []string{BackendPrivilegeInternal, BackendPrivilegePrivate}) {
		return fmt.Errorf("invalid privilege for %s:%s", backend.Host, backend.Upstream)
	}

	return nil
}

// getPrivilege 后端服务权限模式
func (backend Backend) getPrivilege() string {
	if backend.Privilege == "" {
		return BackendPrivilegeInternal
	}

	return backend.Privilege
}

// GetUpstream 当前后端服务配置的后端 upstream 地址
func (backend Backend) GetUpstream() string {
	if str.HasPrefixes(backend.Upstream, []string{"http://", "https://"}) {
		return backend.Upstream
	}

	return fmt.Sprintf("http://%s", backend.Upstream)
}

// HasPrivilege 检查用户是否有权限访问该后端
func (backend Backend) HasPrivilege(userAuthInfo *UserAuthInfo) bool {
	if backend.getPrivilege() == BackendPrivilegeInternal {
		return true
	}

	if str.In(userAuthInfo.Account, backend.Users) {
		return true
	}

	for _, grp := range userAuthInfo.Groups {
		if grp == "" {
			continue
		}

		if str.In(grp, backend.Groups) {
			return true
		}
	}

	return false
}

// Session 用户会话管理配置
type Session struct {
	HashKey      string `json:"hash_key" yaml:"hash_key,omitempty"`
	BlockKey     string `json:"block_key" yaml:"block_key,omitempty"`
	CookieName   string `json:"cookie_name" yaml:"cookie_name,omitempty"`
	CookieDomain string `json:"cookie_domain" yaml:"cookie_domain,omitempty"`
	MaxAge       int    `json:"max_age" yaml:"max_age,omitempty"`
}

// LDAP 域账号登录配置
type LDAP struct {
	URL         string `json:"url" yaml:"url,omitempty"`
	BaseDN      string `json:"base_dn" yaml:"base_dn,omitempty"`
	Username    string `json:"username" yaml:"username,omitempty"`
	Password    string `json:"-" yaml:"password,omitempty"`
	DisplayName string `json:"display_name" yaml:"display_name,omitempty"`
	UID         string `json:"uid" yaml:"uid,omitempty"`
	UserFilter  string `json:"user_filter" yaml:"user_filter,omitempty"`
}

// Redis 连接配置
type Redis struct {
	Addr     string `json:"addr" yaml:"addr,omitempty"`
	Password string `json:"-" yaml:"password"`
	DB       int    `json:"db" yaml:"db,omitempty"`
}

// Users 用户配置
type Users struct {
	IgnoreAccountSuffix string      `json:"ignore_account_suffix" yaml:"ignore_account_suffix,omitempty"`
	Local               []LocalUser `json:"local,omitempty" yaml:"local,omitempty"`
	LDAP                []LDAPUser  `json:"ldap,omitempty" yaml:"ldap,omitempty"`
}

// LDAPUser ldap 用户配置
type LDAPUser struct {
	Account string   `json:"account" yaml:"account"`
	Group   string   `json:"group,omitempty" yaml:"group,omitempty"`
	Groups  []string `json:"groups,omitempty" yaml:"groups,omitempty"`
}

// GetUserGroups 获取用户所属 groups
func (user LDAPUser) GetUserGroups() []string {
	if user.Groups == nil {
		user.Groups = make([]string, 0)
	}
	if user.Group != "" {
		user.Groups = append(user.Groups, user.Group)
	}

	return str.Distinct(user.Groups)
}

// LocalUser 本地用户配置
type LocalUser struct {
	Name     string   `json:"name" yaml:"name"`
	Account  string   `json:"account" yaml:"account"`
	Password string   `json:"-" yaml:"password"`
	Group    string   `json:"group,omitempty" yaml:"group,omitempty"`
	Groups   []string `json:"groups,omitempty" yaml:"groups,omitempty"`
	Algo     string   `json:"algo" yaml:"algo"`
}

// GetUserGroups 获取用户所属的 groups
func (user LocalUser) GetUserGroups() []string {
	if user.Groups == nil {
		user.Groups = make([]string, 0)
	}
	if user.Group != "" {
		user.Groups = append(user.Groups, user.Group)
	}

	return str.Distinct(user.Groups)
}

// LoadConfFromFile 从配置文件加载配置
func LoadConfFromFile(configPath string) (*Config, error) {
	if configPath == "" {
		return nil, errors.New("proxy config is required")
	}

	if !file.Exist(configPath) {
		return nil, fmt.Errorf("proxy config file %s not exist", configPath)
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var conf Config
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return nil, err
	}

	conf = conf.populateDefault()
	if err := conf.validate(); err != nil {
		return nil, err
	}

	return &conf, nil
}

// BuildDefaultConfig 创建默认配置
func BuildDefaultConfig() Config {
	return Config{}.populateDefault()
}
