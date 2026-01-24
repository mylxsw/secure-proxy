# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Secure Proxy is a Go-based reverse proxy server that provides authenticated access to internal HTTP services. It integrates with LDAP for enterprise authentication and supports role-based access control.

## Build Commands

```bash
# Build server (debug with race detector)
make build

# Build production release (static Linux AMD64 binary)
make build-release

# Build CLI tool
make build-tool

# Run server with default config
make run

# Clean build artifacts
make clean
```

Direct Go commands:
```bash
go build -o secure-proxy ./cmd/server/main.go
go build -o secure-proxy-tool ./cmd/tool/main.go
```

## Running the Server

```bash
./secure-proxy --conf ./secure-proxy.yaml
# or
./build/debug/secure-proxy --conf ./secure-proxy.yaml
```

## CLI Tool Commands

```bash
# Generate default config
secure-proxy-tool generate-conf > secure-proxy.yaml

# Encrypt password with bcrypt
secure-proxy-tool encrypt-password --password <password>

# Generate session keys (hash_key and block_key)
secure-proxy-tool session-key-generate

# List LDAP users
secure-proxy-tool ldap-users --conf ./secure-proxy.yaml

# Encrypt all plaintext passwords in config
secure-proxy-tool encrypt-file --conf ./secure-proxy.yaml --overwrite
```

## Architecture

**Framework**: Uses `mylxsw/glacier` for dependency injection with a provider pattern.

**Key Components**:

- `cmd/server/main.go` - Application entry point, registers all providers
- `cmd/tool/main.go` - CLI utilities for config management
- `config/` - YAML configuration loading and validation
- `internal/auth/` - Authentication implementations (ldap, local, ldap_local)
- `internal/handler/` - HTTP handlers (ProxyHandler for reverse proxy, AuthHandler for login)
- `internal/secure/` - Cookie-based session management using gorilla/securecookie
- `internal/store/` - Session validation and login rate limiting
- `internal/cache/` - Cache interface with memory and redis implementations

**Request Flow**:
1. Request arrives at `ProxyHandler.ServeHTTP()`
2. Session cookie validated via `AuthHandler`
3. If no valid session → redirect to `/secure-proxy/auth`
4. If valid → check backend permissions against user groups
5. If authorized → forward to upstream via reverse proxy

**Auth Interface** (`internal/auth/auth.go`):
All auth implementations (ldap, local, ldap_local) implement:
- `Login(username, password)` - Returns AuthedUser or error
- `GetUser(account)` - Get user by account
- `Users()` - List all users

**Provider Pattern**:
Each module has a `Provider` struct with `Register(binder)` to register dependencies. Providers are composed in `cmd/server/main.go`.

## Static Assets

Static assets (CSS/JS for login page) are embedded using `esc`:
```bash
make build-static
# Generates internal/handler/static.go from assets/
```

## Configuration

Main config file: `secure-proxy.yaml`

Key sections:
- `auth_type`: ldap, local, or ldap+local
- `users.local`: Local user definitions with bcrypt/plain/base64 passwords
- `users.ldap`: LDAP user group mappings
- `backends`: Host-to-upstream mappings with privilege levels (internal/private)
- `session`: Cookie encryption keys (hash_key, block_key must be base64-encoded 32-byte values)
- `cache.driver`: memory or redis
