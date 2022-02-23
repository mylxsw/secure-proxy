# secure-proxy

secure-proxy 是一个前置代理，为企业内部基于 http 协议的服务提供一个安全的访问入口，集成企业内部 LDAP 账户体系，同时也提供了对用户行为的审计支持。

## Server （secure-proxy）

该项目作为前端代理，为内部 HTTP 服务提供安全的访问入口，基于 LDAP 账号体系。

## Tool （secure-proxy-tool）

提供了一些工具，用于配合 secure-proxy 生成配置文件。

```
NAME:
   secure-proxy 工具库

USAGE:
    [global options] command [command options] [arguments...]

COMMANDS:
   generate-conf         生成默认配置文件
   encrypt-file          加密配置文件，将配置文件中所有的明文密码部分使用 bcrypt 算法加密
   ldap-users            根据配置文件中的配置，输出查询到的 LDAP 用户清单
   session-key-generate  随机生成 session.hash_key 和 session.block_key
   encrypt-password      使用 bcrypt 算法加密密码
   help, h               Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h  show help
```