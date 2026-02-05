# auth - JWT Authentication with Proof-of-Work

A comprehensive authentication package with JWT tokens and Argon2id-based proof-of-work puzzles.

## Packages

- **`jwt/`** - JWT token generation and verification
- **`puzzle/`** - Proof-of-work puzzle system with HMAC protection
- **`config/`** - Flexible configuration management
- **`handlers/`** - HTTP request handlers
- **`metrics/`** - Metrics recording (Prometheus + Noop)
- **`server/`** - Convenient server wrapper

## Command-line Tools

- **`cmd/server/`** - Production-ready auth server
- **`cmd/test-puzzle-auth/`** - End-to-end testing utility

## Quick Start

```go
import "github.com/status-im/proxy-common/auth/server"

srv, _ := server.Quick()
srv.ListenAndServe(":8081")
```

## Building

```bash
# Build server
go build -o auth-server ./cmd/server

# Build test utility
go build -o test-puzzle-auth ./cmd/test-puzzle-auth

# Or use Makefile from root
cd ..
make build-all
```

## Usage Examples

See the main [README.md](../README.md) for usage documentation.

## Features

- JWT tokens with custom claims
- Argon2id proof-of-work puzzles
- HMAC-protected puzzle validation
- Per-token rate limiting
- Optional Prometheus metrics
- Flexible configuration (functional options, env vars, JSON)
