# AGENTS.md

Guidance for AI coding agents working in the `iptv-tool-v2` repository.

## Project Overview

IPTV management tool with a Go 1.25+ backend (Gin + GORM + SQLite) and an embedded Vue 3 frontend (Element Plus + Vite). The server aggregates live TV channel sources, fetches EPG data, and publishes subscriptions in M3U/TXT/XMLTV formats. The Vue SPA is embedded into the Go binary at build time via `//go:embed` in `web/embed.go`.

## Repository Structure

```
cmd/iptv-server/main.go    # Entrypoint (CLI flags, init order, server start)
internal/
  api/                      # Gin HTTP handlers, router setup, rate limiting
  iptv/                     # IPTV platform client abstractions (Huawei; ZTE is empty placeholder)
  model/                    # GORM models (models.go) and DB init (db.go); global model.DB
  publish/                  # Aggregation engine and /sub/ endpoint handlers
  service/                  # Business logic (user, live_source, epg_source, detect)
  task/                     # Interval-based task scheduler (native Go)
pkg/
  auth/                     # JWT generation/parsing, RSA key management, Gin middleware
  epg/                      # XMLTV parsing and generation
  logger/                   # slog setup with lumberjack log rotation
  m3u/                      # M3U and TXT/DIYP parsing and generation
  utils/                    # 3DES crypto, brute-force cracker
web/                        # Vue 3 frontend (JS, not TypeScript)
  src/                      # Vue source (views, components, stores, api, router)
  dist/                     # Built frontend output (committed, embedded into binary)
```

## Build Commands

```bash
# Go backend
go build -o bin/iptv-server ./cmd/iptv-server
go run ./cmd/iptv-server
go vet ./...
go fmt ./...          # Run before committing
go mod tidy

# Vue frontend (run from web/ directory)
npm install           # Install dependencies
npm run dev           # Vite dev server (proxies /api, /sub, /logo to localhost:8023)
npm run build         # Production build to web/dist/
```

## Test Commands

```bash
go test ./...                                     # All tests
go test ./pkg/m3u/                                # Single package
go test -run TestParseM3U ./pkg/m3u/              # Single test (regex match)
go test -v -run "TestParseM3U/empty_input" ./pkg/m3u/  # Single sub-test
go test -race ./...                               # With race detection
go test -coverprofile=coverage.out ./...          # With coverage
```

Go tests by package, not by file. Use `-run` with a regex to target specific test functions. Note: no test files currently exist in the codebase -- these commands show the correct syntax for when tests are added.

## Code Style Guidelines

### Formatting

Standard `gofmt` (tabs). No additional linters are configured.

### Import Ordering

Three groups separated by blank lines (omit empty groups):
1. Standard library
2. Third-party packages
3. Internal packages (`iptv-tool-v2/...`)

```go
import (
    "context"
    "fmt"
    "net/http"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "iptv-tool-v2/internal/model"
    "iptv-tool-v2/internal/service"
)
```

Use package aliases to disambiguate: `epgpkg "iptv-tool-v2/pkg/epg"`.

### Naming Conventions

- **Packages:** lowercase, single word (`api`, `model`, `service`, `auth`, `m3u`)
- **Types/Structs:** PascalCase (`LiveSourceController`, `HTTPClient`, `TripleDESCrypto`)
- **Constants:** PascalCase with type prefix (`LiveSourceTypeIPTV`, `PublishFormatM3U`, `RuleTypeAlias`)
- **Errors:** package-level `var` with `Err` prefix (`ErrUserExists`, `ErrInvalidPassword`, `ErrNoToken`)
- **Constructors:** `NewXxx()` (`NewLiveSourceController()`, `NewHTTPClient()`)
- **Request DTOs:** suffixed with `Request` (`CreateLiveSourceRequest`, `LoginRequest`)
- **JSON tags:** snake_case (`json:"source_id"`, `json:"cron_time"`)
- **GORM tags:** `gorm:"primarykey"`, `gorm:"uniqueIndex;not null"`, `gorm:"default:true"`
- **Binding tags:** Gin validation (`binding:"required,min=3"`)

### Types

- Custom string types for enums: `type LiveSourceType string`, `type PublishFormat string`
- `json.RawMessage` for flexible JSON fields in request bodies
- `*string`, `*bool`, `*json.RawMessage` for optional fields in update requests (partial updates)
- `*time.Time` for nullable timestamps, `*uint` for optional foreign keys
- `map[string]interface{}` for dynamic GORM updates
- `gin.H` for JSON responses

### Error Handling

1. **Sentinel errors** at package level: `var ErrUserExists = errors.New("user already exists")`
2. **Wrapped errors** with `%w`: `return nil, fmt.Errorf("failed to fetch channels: %w", err)`
3. **API responses**: `c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})`
4. **Status by error type**: map sentinel errors to HTTP status codes (e.g., `ErrUserExists` -> 409)
5. **Middleware abort**: always `c.Abort()` + `return` after unauthorized responses
6. **Context timeouts**: `context.WithTimeout` for external calls (30s network, 5-10min IPTV/EPG)
7. **Fatal at startup only**: `logger.Fatalf` (custom wrapper: `slog.Error` + `os.Exit(1)`)

### Logging

Uses `log/slog` (Go structured logging) exclusively. Initialized in `pkg/logger` with dual output (stdout + rotated file via lumberjack). Use structured key-value pairs:

```go
slog.Info("fetched channels", "source_id", id, "count", len(channels))
slog.Error("failed to sync", "error", err)
```

### Localization

User-facing error messages and UI strings are in Chinese (Simplified). Maintain this convention (e.g., `"该名称已存在"`, `"用户名或密码错误"`). Internal/system log messages use English.

## Architectural Patterns

- **Global DB**: `model.DB` accessed directly from controllers and services (no DI)
- **Auto-migration**: all models migrated at startup in `model.InitDB()`
- **Strategy pattern**: `EPGFetchStrategy` interface for platform-specific EPG fetching
- **init() registration**: EPG strategies register in `init()`, triggered via blank import in main.go (`_ "iptv-tool-v2/internal/iptv/huawei"`)
- **Factory pattern**: `createIPTVClient()` for IPTV platform instantiation
- **Worker pool**: `sync.WaitGroup` + channels for concurrent operations (crack, detect)
- **Rate limiting**: buffered channel as semaphore (`internal/iptv/http.go`); sliding-window IP rate limiter on login (`internal/api/ratelimit.go`)
- **Batch DB inserts**: `CreateInBatches(records, 100)` for channels, 200 for EPG programs
- **Security**: RSA-OAEP password encryption (frontend encrypts, backend decrypts); CAPTCHA after 3 failed login attempts (`base64Captcha`); per-IP rate limiting on login

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/gin-gonic/gin` | HTTP framework |
| `gorm.io/gorm` + `github.com/glebarez/sqlite` | ORM + pure-Go SQLite |
| `github.com/golang-jwt/jwt/v5` | JWT auth |
| `github.com/mojocn/base64Captcha` | Login CAPTCHA generation |
| `golang.org/x/crypto` | bcrypt password hashing |
| `gopkg.in/natefinch/lumberjack.v2` | Log file rotation |
| `vue` 3 + `element-plus` + `pinia` + `axios` | Frontend stack (JS, hash routing) |
