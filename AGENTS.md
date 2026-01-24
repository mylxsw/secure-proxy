# Repository Guidelines

## Project Structure & Module Organization
- `cmd/server` is the main server entrypoint; `cmd/tool` contains the CLI utilities.
- `internal/` holds core services (auth, handler, secure, cache, store, otp). Most application logic lives here.
- `config/` defines configuration models and loaders.
- `assets/` contains login page CSS/JS. Static assets are embedded into `internal/handler/static.go` via `esc`.
- `secure-proxy.yaml` is the runtime config file used by the server and CLI tool.

## Build, Test, and Development Commands
- `make build`: build debug server binary at `build/debug/secure-proxy` (race detector enabled).
- `make build-tool`: build CLI tool at `build/debug/secure-proxy-tool`.
- `make run`: build and run the server with `./secure-proxy.yaml`.
- `make build-release`: produce static Linux AMD64 binaries under `build/release/`.
- `make build-static`: embed assets into `internal/handler/static.go` (requires `esc`).
- `go test ./...`: run Go unit tests.

## Coding Style & Naming Conventions
- Go code follows standard `gofmt` style (tabs for indentation, canonical formatting).
- Package and file names are lowercase; exported identifiers use `CamelCase`.
- Keep handlers and providers grouped by feature under `internal/` (e.g., `internal/auth`, `internal/handler`).

## Testing Guidelines
- Tests use Go’s standard `testing` package.
- Current tests live in files like `internal/otp/otp_test.go`; name new tests `*_test.go` and keep them near the code.
- Prefer table-driven tests for auth and config behaviors.

## Commit & Pull Request Guidelines
- Git history favors short, direct messages (e.g., “update”, “add …”), sometimes in Chinese. Keep commits concise and scoped.
- PRs should include: purpose summary, config changes (if any), and how to run/reproduce. Add UI screenshots when changing login assets.

## Security & Configuration Tips
- Do not commit real LDAP credentials or session keys. Use `secure-proxy-tool session-key-generate` and keep secrets local.
- When editing `assets/`, rerun `make build-static` to refresh embedded assets.
