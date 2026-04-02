# AGENTS.md

Guidance for AI coding agents working in the `iptv-tool-v2` repository.

## Project Overview

IPTV management tool with a Go 1.25+ backend (Gin + GORM + SQLite) and an embedded Vue 3 frontend (Element Plus + Vite). The server aggregates live TV channel sources, fetches EPG data, and publishes subscriptions in M3U/TXT/XMLTV/DIYP formats. The Vue SPA is embedded into the Go binary at build time via `//go:embed` in `web/embed.go`. Full i18n support (zh, en, zh-Hant) with backend-driven locale loading.

## Repository Structure

```
cmd/iptv-server/main.go    # Entrypoint (CLI flags, init order, server start)
internal/
  api/                      # Gin HTTP handlers, router setup, middleware, rate limiting
    access_control.go        # IP whitelist/blacklist CRUD controller
    acl_middleware.go        # Access control middleware with cached evaluation
    epg_source.go            # EPG source CRUD controller
    geoip.go                 # GeoIP settings controller (status, download, auto-update)
    live_source.go           # Live source CRUD + sync/detect trigger controller
    log.go                   # Runtime & access log ring buffers + API handlers
    log_middleware.go         # Access logging middleware (records to ring buffer + AccessStat)
    logo.go                  # Channel logo upload/CRUD controller
    publish.go               # Publish interface CRUD + preview controller
    ratelimit.go             # Sliding-window IP rate limiter on login
    router.go                # Central route registration (SetupRouter)
    rule.go                  # Aggregation rule CRUD controller
    settings.go              # Detection settings + ffprobe upload controller
    system.go                # Init, login, password change, key crack controller
    update.go                # GitHub release check-update handler
  iptv/                      # IPTV platform client abstractions
    http.go                   # Shared HTTP client with rate-limiting semaphore
    interface.go              # IPTVClient interface definition
    types.go                  # Shared types (Channel, EPGProgram, etc.)
    huawei/                   # Huawei IPTV platform implementation + EPG strategies
    zte/                      # ZTE placeholder (empty)
  model/                     # GORM models (models.go) and DB init (db.go); global model.DB
  publish/                   # Aggregation engine, /sub/ handlers, and in-memory cache
    cache.go                  # Generic TTL cache with per-key mutex (stampede prevention)
    engine.go                 # Channel/EPG aggregation with rule application
    handler.go                # /sub/live/:path and /sub/epg/:path handlers
  service/                   # Business logic
    access_stat.go            # IP access statistics with async batched DB writes
    detect.go                 # Channel availability detection (ffprobe worker pool)
    epg_source.go             # EPG source sync service
    geoip.go                  # GeoIP database download/extract/lookup (GeoLite2 mmdb)
    iptv_lock.go              # Per-LiveSource mutex registry (prevents concurrent IPTV sessions)
    live_source.go            # Live source sync service
    url_selector.go           # Multicast/unicast URL priority selection for detection
    user.go                   # User registration, login, password, credential reset
  task/                      # Interval-based task scheduler (native Go, no cron library)
  version/                   # Build-time version injection + semver comparison
pkg/
  auth/                      # JWT generation/parsing (jwt.go), RSA key management (rsa.go)
  epg/                       # XMLTV parsing and generation
  i18n/                      # Internationalization engine
    i18n.go                   # Locale loading, Accept-Language negotiation (golang.org/x/text/language)
    middleware.go             # Gin middleware: resolves lang from X-Language / Accept-Language header
  logger/                    # slog setup with lumberjack + web UI log tee (LogAppender interface)
  m3u/                       # M3U and TXT/DIYP parsing and generation
  utils/                     # Utilities
    crack.go                  # Brute-force IPTV password cracker
    crypto.go                 # 3DES-ECB crypto for IPTV auth
    password.go               # Cryptographically random password generation
    sort.go                   # Natural sort comparison (e.g. "CCTV-2" < "CCTV-10")
locales/                     # Backend i18n locale files (embedded via //go:embed)
  embed.go                    # Embeds *.json locale files as embed.FS
  zh.json                     # Simplified Chinese (frontend + backend keys)
  en.json                     # English
  zh-Hant.json                # Traditional Chinese
web/                         # Vue 3 frontend (JS, not TypeScript)
  src/
    api/index.js              # Axios instance with auth interceptors
    composables/usePolling.js # Vue composable for periodic polling
    i18n/index.js             # vue-i18n setup with backend-driven locale loading
    layout/Layout.vue         # Main app layout (sidebar + header + content)
    router/index.js           # Hash-based routing with auth guards
    stores/auth.js            # Pinia auth store (JWT token, init status)
    stores/theme.js           # Pinia theme store (dark/light mode)
    utils/crypto.js           # RSA-OAEP encryption for password fields
    utils/promptTemplates.js  # Reusable Element Plus dialog/confirm helpers
    views/                    # Page-level SFC components (13 views)
    style.css                 # Global styles with CSS variables
  dist/                      # Built frontend output (committed, embedded into binary)
docker/
  Dockerfile.ci               # Multi-stage CI build (Node frontend → Go backend → Alpine runtime)
  docker-compose.yml           # Compose file for containerized deployment
.github/workflows/
  release.yml                 # GitHub Actions release workflow
```

## Build Commands

```bash
# Go backend
go build -o bin/iptv-server ./cmd/iptv-server
go run ./cmd/iptv-server

# Production build with version injection
go build -ldflags "-s -w -X iptv-tool-v2/internal/version.Version=v1.0.0" -o bin/iptv-server ./cmd/iptv-server

go vet ./...
go fmt ./...          # Run before committing
go mod tidy

# Vue frontend (run from web/ directory)
npm install           # Install dependencies
npm run dev           # Vite dev server (proxies /api, /sub, /logo to localhost:8023)
npm run build         # Production build to web/dist/
```

### CLI Flags

```
--addr       HTTP listen address (default ":8023")
--data       Data directory for db, logos, detect, geoip (default "data", relative to executable)
--log-dir    Log directory (default "logs", relative to executable)
--jwt-secret JWT secret (auto-generated if empty)
--reset-user Reset admin credentials with a new username (generates random password, then exits)
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

Go tests by package, not by file. Use `-run` with a regex to target specific test functions.

### Existing Test Files

| Package | Test File | Coverage Area |
|---------|-----------|---------------|
| `internal/api` | `acl_middleware_test.go` | ACL IP matching logic |
| `internal/publish` | `engine_test.go`, `engine_epg_test.go`, `engine_logo_test.go` | Aggregation engine |
| `internal/version` | `version_test.go` | Semver comparison |
| `internal/service` | `iptv_lock_test.go`, `url_selector_test.go` | Mutex registry, URL selection |
| `pkg/epg` | `xmltv_test.go` | XMLTV parsing |
| `pkg/utils` | `password_test.go` | Random password generation |

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

- **Packages:** lowercase, single word (`api`, `model`, `service`, `auth`, `m3u`, `i18n`)
- **Types/Structs:** PascalCase (`LiveSourceController`, `HTTPClient`, `TripleDESCrypto`, `GeoIPService`)
- **Constants:** PascalCase with type prefix (`LiveSourceTypeIPTV`, `PublishFormatM3U`, `RuleTypeAlias`, `MatchModeRegex`)
- **Errors:** package-level `var` with `Err` prefix (`ErrUserExists`, `ErrInvalidPassword`, `ErrNoToken`)
- **Constructors:** `NewXxx()` (`NewLiveSourceController()`, `NewGeoIPService()`, `NewAccessStatService()`)
- **Request DTOs:** suffixed with `Request` (`CreateLiveSourceRequest`, `LoginRequest`, `UpdateAccessControlRequest`)
- **Response DTOs:** suffixed with `Response` (`AccessControlResponse`)
- **JSON tags:** snake_case (`json:"source_id"`, `json:"cron_time"`, `json:"last_accessed_at"`)
- **GORM tags:** `gorm:"primarykey"`, `gorm:"uniqueIndex;not null"`, `gorm:"default:true"`, `gorm:"column:iptv_config"`
- **Binding tags:** Gin validation (`binding:"required,min=3"`, `binding:"required,oneof=disabled whitelist blacklist"`)
- **i18n keys:** dot-separated with section prefix (`error.invalid_id`, `message.acl_updated`)

### Types

- Custom string types for enums: `type LiveSourceType string`, `type PublishFormat string`, `type RuleType string`, `type MatchMode string`
- `json.RawMessage` for flexible JSON fields in request bodies and frontend locale data
- `*string`, `*bool`, `*json.RawMessage` for optional fields in update requests (partial updates)
- `*int`, `*time.Time` for nullable fields, `*uint` for optional foreign keys
- `map[string]interface{}` for dynamic GORM updates
- `gin.H` for JSON responses
- Generics: used in `publish/cache.go` (`cacheEntry[T any]`)

### Error Handling

1. **Sentinel errors** at package level: `var ErrUserExists = errors.New("user already exists")`
2. **Wrapped errors** with `%w`: `return nil, fmt.Errorf("failed to fetch channels: %w", err)`
3. **API responses**: `c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})`
4. **i18n error messages**: `c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error.acl_invalid_entry")})`
5. **Status by error type**: map sentinel errors to HTTP status codes (e.g., `ErrUserExists` -> 409)
6. **Middleware abort**: always `c.Abort()` + `return` after unauthorized responses
7. **Context timeouts**: `context.WithTimeout` for external calls (30s network, 5-10min IPTV/EPG)
8. **Fatal at startup only**: `logger.Fatalf` (custom wrapper: `slog.Error` + `os.Exit(1)`)

### Logging

Uses `log/slog` (Go structured logging) exclusively. Initialized in `pkg/logger` with triple output: stdout + rotated file via lumberjack + web UI ring buffer (via `LogAppender` interface). Use structured key-value pairs:

```go
slog.Info("fetched channels", "source_id", id, "count", len(channels))
slog.Error("failed to sync", "error", err)
```

### Localization / i18n

Full i18n support with three languages: Simplified Chinese (`zh`), English (`en`), Traditional Chinese (`zh-Hant`).

**Backend i18n** (`pkg/i18n`):
- Locale JSON files in `locales/` embedded via `//go:embed` at build time
- Each JSON has `frontend` (raw JSON served to Vue) and `backend` (dot-key messages) sections
- `i18n.T(lang, key, args...)` for backend message translation with English fallback
- `i18n.Middleware()` resolves language from `X-Language` header (frontend override) or `Accept-Language` header via `golang.org/x/text/language` matcher
- `i18n.Lang(c)` retrieves resolved language from Gin context

**Frontend i18n** (`web/src/i18n`):
- Uses `vue-i18n` with lazy-loaded locales from `/api/locales/:lang`
- Browser language detection with locale mapping (e.g., `zh-TW` → `zh-Hant`, `zh-CN` → `zh`)
- Saved locale preference in `localStorage`

**Convention**: All user-facing messages (errors, confirmations, UI labels) must use i18n keys. Internal/system log messages remain in English.

## Architectural Patterns

- **Global DB**: `model.DB` accessed directly from controllers and services (no DI)
- **SQLite tuning**: WAL mode, `synchronous=NORMAL`, `busy_timeout=5000`, single connection pool
- **Auto-migration**: all 11 models migrated at startup in `model.InitDB()` with legacy index cleanup
- **Defensive startup reset**: `is_syncing` / `is_detecting` flags reset to false on boot
- **Strategy pattern**: `EPGFetchStrategy` interface for platform-specific EPG fetching
- **init() registration**: EPG strategies register in `init()`, triggered via blank import in main.go (`_ "iptv-tool-v2/internal/iptv/huawei"`)
- **Factory pattern**: `createIPTVClient()` for IPTV platform instantiation
- **Worker pool**: `sync.WaitGroup` + channels for concurrent operations (crack, detect)
- **Rate limiting**: buffered channel as semaphore (`internal/iptv/http.go`); sliding-window IP rate limiter on login (`internal/api/ratelimit.go`)
- **Batch DB inserts**: `CreateInBatches(records, 100)` for channels, 200 for EPG programs
- **In-memory publish cache**: TTL-based cache (15 min) with per-key mutex and double-checked locking to prevent stampede; invalidated on any data mutation (`publish.InvalidateAll()`)
- **Async batched writes**: `AccessStatService` uses goroutine + buffered channel + periodic flush (every 5s or 50 entries) for non-blocking IP access recording
- **Per-source IPTV lock**: `iptv_lock.go` provides per-LiveSourceID mutex registry preventing concurrent authenticated sessions to the same IPTV server
- **Ring buffer logs**: `RuntimeLogBuffer` and `AccessLogBuffer` are fixed-size ring buffers (5k entries) for the web UI log center, decoupled from slog via `LogAppender` interface
- **Access control**: IP whitelist/blacklist with cached ACL evaluation, self-lockout prevention, supports single IP, CIDR, and IP range entries
- **GeoIP integration**: Downloads GeoLite2 mmdb from GitHub, supports auto-update (configurable interval 1-7 days), retry with progress tracking, used for access stats IP geolocation
- **Security**: RSA-OAEP password encryption (frontend encrypts, backend decrypts); CAPTCHA after 3 failed login attempts (`base64Captcha`); per-IP rate limiting on login; access control (whitelist/blacklist)
- **Version management**: `internal/version.Version` set via `-ldflags` at build time; `CompareVersions()` for GitHub release update checks

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/gin-gonic/gin` | HTTP framework |
| `gorm.io/gorm` + `github.com/glebarez/sqlite` | ORM + pure-Go SQLite |
| `github.com/golang-jwt/jwt/v5` | JWT auth |
| `github.com/mojocn/base64Captcha` | Login CAPTCHA generation |
| `github.com/oschwald/geoip2-golang/v2` | GeoLite2 mmdb reader for IP geolocation |
| `golang.org/x/crypto` | bcrypt password hashing |
| `golang.org/x/text` | Language tag matching for i18n (Accept-Language negotiation) |
| `gopkg.in/natefinsh/lumberjack.v2` | Log file rotation |
| `vue` 3 + `element-plus` + `pinia` + `axios` + `vue-i18n` | Frontend stack (JS, hash routing, i18n) |

## Data Directory Layout

At runtime, the `--data` flag (default `data/`) contains:

```
data/
  db/iptv.db          # SQLite database
  logos/               # Uploaded channel logo files
  detect/              # ffprobe binary and detection temp files
  geoip/               # GeoLite2-City.mmdb (auto-downloaded)
```

## GORM Models

| Model | Table | Description |
|-------|-------|-------------|
| `User` | `users` | Admin account (single user) |
| `LiveSource` | `live_sources` | Live TV channel sources (IPTV/URL/manual) |
| `EPGSource` | `epg_sources` | EPG data sources (IPTV/XMLTV URL) |
| `ChannelLogo` | `channel_logos` | Uploaded channel logos |
| `PublishInterface` | `publish_interfaces` | Aggregated subscription endpoint configs |
| `AggregationRule` | `aggregation_rules` | Reusable alias/filter/group rules |
| `ParsedChannel` | `parsed_channels` | Channels parsed from live sources |
| `ParsedEPG` | `parsed_epgs` | EPG programs parsed from EPG sources |
| `SystemSetting` | `system_settings` | Key-value system config (ACL mode, GeoIP auto-update) |
| `AccessControlEntry` | `access_control_entries` | IP whitelist/blacklist entries |
| `AccessStat` | `access_stats` | IP access statistics (last 7 days) |
