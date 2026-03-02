# AGENTS.md

This file provides guidance for AI coding agents working in the `iptv-tool-v2` repository.

## Project Overview

IPTV management tool with a Go backend (Gin + GORM + SQLite) and an embedded Vue 3 frontend (Element Plus + Vite). The server aggregates live TV channel sources, fetches EPG data, and publishes subscriptions in M3U/TXT/XMLTV formats. The Vue SPA is embedded into the Go binary at build time.

## Repository Structure

```
cmd/iptv-server/main.go    # Application entrypoint (CLI flags, init, server start)
internal/
  api/                      # Gin HTTP handlers (controllers), router setup
  iptv/                     # IPTV platform client abstractions (Huawei, ZTE)
  model/                    # GORM models and DB initialization (global model.DB)
  publish/                  # Aggregation engine and /sub/ endpoint handlers
  service/                  # Business logic layer
  task/                     # Cron scheduler (robfig/cron)
pkg/
  auth/                     # JWT generation, parsing, Gin middleware
  epg/                      # XMLTV parsing and generation
  m3u/                      # M3U and TXT/DIYP parsing and generation
  utils/                    # 3DES crypto, brute-force cracker
web/                        # Vue 3 frontend (embedded via Go embed)
  src/                      # Vue source (views, components, stores, api, router)
  dist/                     # Built frontend output (committed, embedded into binary)
```

## Build Commands

### Go Backend

```bash
# Build
go build -o bin/iptv-server ./cmd/iptv-server

# Run directly
go run ./cmd/iptv-server

# Vet / static analysis
go vet ./...

# Format
go fmt ./...

# Tidy dependencies
go mod tidy
```

### Vue Frontend (run from web/ directory)

```bash
npm install           # Install dependencies
npm run dev           # Vite dev server (proxies /api, /sub, /logo to localhost:8080)
npm run build         # Production build to web/dist/
npm run preview       # Preview production build
```

## Test Commands

```bash
# Run all tests
go test ./...

# Run tests in a specific package
go test ./pkg/m3u/
go test ./internal/service/

# Run a single test by name (regex match)
go test -run TestParseM3U ./pkg/m3u/

# Run a specific sub-test
go test -v -run "TestParseM3U/empty_input" ./pkg/m3u/

# Verbose output
go test -v ./...

# With race detection
go test -race ./...

# With coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
```

Note: Go tests by package, not by file. Use `-run` with a regex to target specific test functions.

## Code Style Guidelines

### Formatting

Use standard `gofmt` (tabs for indentation). No additional linters or formatters are configured. Run `go fmt ./...` before committing.

### Import Ordering

Three groups separated by blank lines:
1. Standard library
2. Third-party packages
3. Internal project packages (`iptv-tool-v2/...`)

```go
import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "iptv-tool-v2/internal/model"
    "iptv-tool-v2/internal/service"
)
```

When there are no third-party imports, use two groups (stdlib + internal).

### Naming Conventions

- **Packages:** lowercase, single word (`api`, `model`, `service`, `auth`, `m3u`)
- **Types/Structs:** PascalCase (`LiveSourceController`, `HTTPClient`, `TripleDESCrypto`)
- **Constants:** PascalCase with type prefix (`LiveSourceTypeIPTV`, `PublishFormatM3U`, `RuleTypeAlias`, `MatchModeRegex`)
- **Errors:** package-level `var` with `Err` prefix (`ErrUserExists`, `ErrInvalidPassword`, `ErrNoToken`)
- **Constructors:** `NewXxx()` pattern (`NewLiveSourceController()`, `NewHTTPClient()`)
- **Methods:** PascalCase exported, camelCase unexported (`FetchAndUpdate`, `fetchIPTV`)
- **Request DTOs:** suffixed with `Request` (`CreateLiveSourceRequest`, `LoginRequest`)
- **JSON tags:** snake_case (`json:"source_id"`, `json:"cron_time"`)
- **GORM tags:** include column/index directives (`gorm:"primarykey"`, `gorm:"uniqueIndex;not null"`)
- **Binding tags:** Gin validation (`binding:"required,min=3"`)

### Types

- Custom string types for enums: `type LiveSourceType string`, `type PublishFormat string`
- `json.RawMessage` for flexible JSON fields in request bodies
- `*string`, `*bool`, `*json.RawMessage` for optional fields in update requests (partial updates)
- `*time.Time` for nullable timestamps, `*uint` for optional foreign keys
- `map[string]interface{}` for dynamic GORM updates
- `gin.H` for JSON responses

### Error Handling

1. **Sentinel errors** at package level:
   ```go
   var ErrUserExists = errors.New("user already exists")
   ```

2. **Wrapped errors** with `fmt.Errorf` and `%w`:
   ```go
   return nil, fmt.Errorf("failed to fetch channels: %w", err)
   ```

3. **API error responses** via Gin:
   ```go
   c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
   ```

4. **Status selection by error type**:
   ```go
   status := http.StatusInternalServerError
   if err == service.ErrUserExists {
       status = http.StatusConflict
   }
   c.JSON(status, gin.H{"error": err.Error()})
   ```

5. **Middleware abort pattern**: always call `c.Abort()` + `return` after unauthorized responses.

6. **Context timeouts**: use `context.WithTimeout` for external calls (30s-5min).

7. **Fatal at startup only**: `log.Fatalf` for unrecoverable init errors (DB, scheduler).

### Architectural Patterns

- **Global DB**: `model.DB` (GORM instance), accessed directly from controllers and services
- **Auto-migration**: all models auto-migrated at startup in `model.InitDB()`
- **Strategy pattern**: `EPGFetchStrategy` interface for platform-specific EPG fetching
- **Factory pattern**: `createIPTVClient()` for IPTV platform instantiation
- **init() registration**: platform implementations register via blank imports (`_ "iptv-tool-v2/internal/iptv/huawei"`)
- **Worker pool concurrency**: `sync.WaitGroup` + channels (see `pkg/utils/crack.go`)
- **Rate limiting**: buffered channel as semaphore (`internal/iptv/http.go`)
- **Batch DB inserts**: `CreateInBatches(records, 100)`

### Localization

Some user-facing error messages and comments are in Chinese (Simplified). Maintain this convention for UI-facing strings (e.g., `"该名称已存在"`). System/API messages use English.

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/gin-gonic/gin` | HTTP framework |
| `gorm.io/gorm` + `github.com/glebarez/sqlite` | ORM + pure-Go SQLite |
| `github.com/golang-jwt/jwt/v5` | JWT auth |
| `github.com/robfig/cron/v3` | Cron scheduler |
| `golang.org/x/crypto` | bcrypt password hashing |
| `vue` 3 + `element-plus` + `pinia` + `axios` | Frontend stack |
