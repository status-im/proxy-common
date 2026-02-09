# proxy-common

[![Tests](https://github.com/status-im/proxy-common/actions/workflows/test.yml/badge.svg)](https://github.com/status-im/proxy-common/actions/workflows/test.yml)

Go library providing common components for proxy services: authentication, caching, HTTP client utilities, API key management, rate limiting, and task scheduling.

## Installation

```bash
go get github.com/status-im/proxy-common
```

## Packages

| Package | Description | Documentation |
|---------|-------------|---------------|
| [auth](auth/) | Proof-of-work auth service (Argon2 puzzles + JWT) | [README](auth/README.md) |
| [cache](cache/) | Multi-level cache (L1 BigCache + L2 KeyDB/Redis) | [README](cache/README.md) |
| [httpclient](httpclient/) | HTTP client with retries, backoff, rate limiting | [README](httpclient/README.md) |
| [apikeys](apikeys/) | API key rotation with failure tracking and backoff | [README](apikeys/README.md) |
| [ratelimit](ratelimit/) | Per-key rate limiting (golang.org/x/time/rate) | [README](ratelimit/README.md) |
| [scheduler](scheduler/) | Background task scheduling at intervals | [README](scheduler/README.md) |
| [batch](batch/) | Generic chunk processing for large datasets | [README](batch/README.md) |
| [models](models/) | Shared cache data models and types | [README](models/README.md) |

## License

MIT License. See [LICENSE](LICENSE) file for details.
