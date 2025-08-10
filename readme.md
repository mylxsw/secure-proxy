# Secure Proxy

Secure Proxy is a front-end proxy server designed to provide a secure access point for internal enterprise services that use the HTTP protocol. It integrates with the enterprise's LDAP account system and supports auditing of user activities.

## Key Features

*   **Unified Identity Authentication**: Supports authentication via LDAP and local users, with flexible configuration options.
*   **Access Control**: Granular access control to backend services based on user groups and permissions.
*   **Security Auditing**: Logs user access for security reviews and behavioral analysis.
*   **Reverse Proxy**: Functions as a reverse proxy, forwarding user requests to backend services.
*   **Session Management**: Secure session handling to prevent session hijacking.
*   **Flexible Configuration**: Easily configure proxy rules, authentication methods, and user permissions using a YAML configuration file.

## Project Structure

*   `cmd/server`: Main application for the Secure Proxy server.
*   `cmd/tool`: A set of tools for Secure Proxy, including utilities for generating configuration files and encrypting passwords.
*   `config`: Definitions and loading logic for configuration files.
*   `internal`: Core business logic modules, including authentication, authorization, proxying, and caching.
*   `assets`: Static assets like CSS and JavaScript for the login page.

## Installation & Deployment

### Clone the Repository

```bash
git clone https://github.com/mylxsw/secure-proxy.git
cd secure-proxy
```

### Build the Project

```bash
# Build the server binary
go build -o secure-proxy ./cmd/server/main.go

# Build the toolset binary
go build -o secure-proxy-tool ./cmd/tool/main.go
```

Alternatively, you can use the `make` command:

```bash
make build
```

### Run the Server

```bash
./secure-proxy --conf ./secure-proxy.yaml
```

### Run with Docker

A Dockerfile is included to build a Docker image:

```bash
# Build the Docker image
./docker-build.sh

# Run the container
docker run -d -p 8080:8080 -v $(pwd)/secure-proxy.yaml:/app/secure-proxy.yaml --name secure-proxy mylxsw/secure-proxy:latest
```

## Configuration

Secure Proxy uses a YAML configuration file. Below is a detailed explanation of the configuration options:

```yaml
# Listening address and port
listen: 127.0.0.1:8080

# Authentication type: ldap, local, or ldap+local
auth_type: local

# HTTP header for the client's real IP address
client_ip_header: X-Real-IP

# Log file path (supports date-based splitting); leave empty for console output
# log_path: /var/log/secure-proxy-%s.log

# User configurations
users:
  # Suffix to ignore during login (e.g., @example.com)
  ignore_account_suffix: "@example.com"
  
  # LDAP user configurations for specifying additional groups
  ldap:
    - account: CN=guanyiyao,CN=Users,DC=example,DC=com
      groups:
        - admin
        - devops
    - account: CN=liupeng,CN=Users,DC=example,DC=com
      groups:
        - devops
  
  # Local user configurations
  local:
    - name: Zhang San # Username
      account: "zhangsan" # Login account
      # Password (plaintext, bcrypt, or base64 encoded based on algo)
      password: $2a$10$L4tAaN9jLZZV4VCYtHbuKO8gTK8HRPAZU3sGXESWguiOobvMOvuNW
      # Password algorithm: plain, bcrypt, base64
      algo: bcrypt
    - name: Li Si
      account: "lisi"
      password: "111111"
      algo: plain

# Backend service configurations
backends:
  - host: prometheus.example.com # Backend domain
    upstream: http://127.0.0.1:9090 # Backend address
    privilege: private # Access privilege: internal or private
    groups: # Allowed groups
      - devops
  - host: consul.example.com
    upstream: http://127.0.0.1:8500
    privilege: private
    groups:
      - CN=devops,DC=example,DC=com
  - host: hdfs.example.com
    upstream: http://127.0.0.1:50070
    privilege: private
    groups:
      - CN=dev,DC=example,DC=com
  - host: spark.example.com
    upstream: http://127.0.0.1:8080
    privilege: private
    groups:
      - CN=dev,DC=example,DC=com
  - host: pinpoint.example.com
    upstream: http://127.0.0.1:8079
    privilege: private
    groups:
      - CN=devops,DC=example,DC=com
  - host: kibana.example.com
    upstream: http://127.0.0.1:5601
    privilege: private
    groups:
      - CN=devops,DC=example,DC=com
      - CN=dev,DC=example,DC=com

# Session configurations
session:
  # Hash key for session encryption (generated with secure-proxy-tool)
  hash_key: oxjsWLzBv8l4jU4RuwEbC/zw+UjqgPHg8aqY6+RPYbc=
  # Block key for session encryption (generated with secure-proxy-tool)
  block_key: H+3qpeM/vn1FN1R05Vy3HvLDYOaJatKqNWp7/8bXvyg=
  # Session cookie name
  cookie_name: secure-proxy-auth
  # Session cookie domain (defaults to current domain if empty)
  cookie_domain: .example.com
  # Maximum session lifetime (in seconds)
  max_age: 86400 # 24 hours

# LDAP configurations
ldap:
  # LDAP server URL
  url: ldap://127.0.0.1:389
  # LDAP base DN
  base_dn: dc=example,dc=com
  # LDAP admin username
  username: admin
  # LDAP admin password
  password: admin
  # Display name attribute
  display_name: displayName
  # UID attribute
  uid: sAMAccountName
  # User filter for LDAP queries
  user_filter: CN=Users,DC=example,DC=com

# Cache configurations
cache:
  # Cache driver: redis or memory (default: memory)
  driver: memory
  # Redis configurations
  redis:
    # Redis server address
    addr: 127.0.0.1:6379
    # Redis password (optional)
    # password: "111111"
```

## Toolset (secure-proxy-tool)

Secure Proxy includes a command-line utility `secure-proxy-tool` to help with configuration and management tasks.

### Usage

```bash
secure-proxy-tool [global options] command [command options] [arguments...]
```

### Available Commands

*   `generate-conf`: Creates a default configuration file.
*   `encrypt-file`: Encrypts all plaintext passwords in a configuration file using bcrypt.
*   `ldap-users`: Lists LDAP users based on the configuration.
*   `session-key-generate`: Generates random session keys (`hash_key` and `block_key`).
*   `encrypt-password`: Encrypts a password using bcrypt.

### Examples

Generate a default configuration file:

```bash
secure-proxy-tool generate-conf > secure-proxy.yaml
```

Encrypt a password:

```bash
secure-proxy-tool encrypt-password --password your_password
```

Generate session keys:

```bash
secure-proxy-tool session-key-generate
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more information.