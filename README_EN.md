# IPTV Tool Management Platform

[![GitHub Release](https://img.shields.io/github/v/release/super321/iptv-tool)](https://github.com/super321/iptv-tool/releases/latest)
![GitHub License](https://img.shields.io/github/license/super321/iptv-tool)

[简体中文](README.md) | **English**

## Project Introduction
IPTV Tool is a lightweight IPTV management and aggregation distribution platform developed with Go and Vue 3. The platform provides convenient features for live streaming source (M3U / TXT, etc.) management, EPG (Electronic Program Guide) fetching and synchronization, as well as channel logo management. Through local deployment, you can easily aggregate multiple network live stream sources and EPG data, and generate unified subscription links with one click for use by various players.

The backend is built on a pure local storage structure using Go 1.25+ (Gin + GORM + SQLite), and the frontend uses Vue 3 (Element Plus + Vite), compiled and embedded into Go's single-file binary, achieving one-click distribution and minimalist deployment.

## Core Features

### 📺 Live Source Management
Supports importing and managing live sources in multiple formats:
- **Network URL Sources**: Supports M3U / TXT format online subscription URLs with custom HTTP headers
- **Manual Editing**: Supports directly pasting/editing channel list content
- **Scheduled Sync**: Configurable refresh intervals (1h ~ 24h) with daily execution time specification
- **Channel Detection**: FFprobe-based channel availability detection with multicast/unicast priority strategy selection, automatic timeout channel filtering
- **Channel List**: Displays channel name, group, stream URL, catchup URL, detection latency, and video codec/resolution information

### 📅 EPG Source Management
- Supports configuring multiple XMLTV format EPG interface sources
- Supports custom HTTP headers for network XMLTV sources
- Scheduled refresh fetching with intuitive display of channel count, program count, and last update time
- Automatically creates corresponding XMLTV format EPG source when adding M3U live sources

### 🖼️ Logo Management
- Supports channel logo file upload (single/batch), preview, and management
- Supports batch deletion
- Provides localized logo file hosting, directly referenceable in aggregated publishing

### 📐 Aggregation Rules
Supports creating reusable aggregation rules for flexible combining in publishing interfaces:
- **Alias Rules**: Batch rename channels via regex or string matching (e.g., standardize "CCTV-1" to "CCTV1")
- **Filter Rules**: Filter out unwanted channels by name or alias using regex/string matching (e.g., shopping channels)
- **Group Rules**: Reclassify channels into custom groups based on conditions
- **AI-Assisted Generation**: Supports AI-assisted generation for all three rule types, with preset quick tags to simplify operations

### 🔗 Aggregated Publishing
Aggregate organized live sources and EPG sources to generate final subscription links:
- **Publishing Formats**: Supports M3U, TXT, XMLTV, DIYP and other formats
- **Protocol Conversion**: Supports multicast address type selection (UDPxy / RTP / IGMP), FCC fast channel change (rtp2httpd), and unicast protocol proxy conversion
- **Per-Source Configuration**: Each live source can independently configure output parameters (address type, multicast protocol, etc.)
- **Custom Parameters**: Supports custom parameters for multicast proxy addresses
- **Catchup/Timeshift**: M3U format supports catchup template configuration
- **Invalid Channel Filtering**: Specify which sources' timeout channels should be automatically filtered during publishing
- **UA Validation**: Supports User-Agent whitelist validation to protect subscription links
- **One-Click Download**: Supports one-click download of generated subscription files
- **Publish Preview**: Instant preview of aggregated content before saving

### 🔐 Security & Access Control
- **Access Control**: Supports global IP whitelist/blacklist mode with single IP, CIDR, and IP range support
- **Login Security**: RSA-OAEP password encryption in transit, CAPTCHA after multiple failed attempts, IP-level login rate limiting
- **HTTPS Support**: Upload TLS certificates to enable HTTPS, custom port support, mutual TLS (mTLS) authentication
- **Password Management**: Web UI password change + CLI one-click admin credential reset

### 📊 Logging & Monitoring
- **Runtime Logs**: Real-time system runtime logs (5,000-entry ring buffer) with search and download
- **Access Logs**: Real-time API access logs (5,000-entry ring buffer) with search and download
- **Access Statistics**: Records 7-day IP access data with total request and subscription request counts
- **GeoIP Geolocation**: Auto-downloads GeoLite2 database for IP geolocation in access statistics, supports auto-update

### 🌍 Internationalization & Themes
- Supports Simplified Chinese, Traditional Chinese, and English
- Automatic language detection with manual switching and persistence
- Dark/Light theme toggle

### 📦 Other Features
- **Config Import/Export**: Selective module-based export and import of configuration data (including live sources, EPG sources, logos, rules, publish interfaces, detection settings, access control), exported as ZIP format
- **Version Update Check**: Automatic GitHub latest version checking with changelog display and one-click download redirect
- **Lightweight Pure Local Execution**: Built-in SQLite database, no external dependencies (Redis / MySQL), data stays completely in your own hands

## Interface Tour

### Live Source Management
Manage all your live channel sources, supporting multiple types of live source fetching and updating.
![Live Source Management](./docs/images/live.png)

### EPG Source Management
Unified management of EPG data from various sources, supporting scheduled fetching, intuitively displaying channel count, program count, and last update time.
![EPG Source Management](./docs/images/epg.png)

### Logo Management
Provides a unified logo file hosting and management page for your channels, allowing direct preview of all configured logos.
![Logo Management](./docs/images/logo.png)

### Aggregated Publishing
Aggregate organized live sources and EPG sources to publish online interfaces, generating final subscription links for players.
![Aggregated Publishing](./docs/images/publish.png)

## Startup Parameters

IPTV Tool supports customizing runtime configuration through command-line parameters. All parameters are optional with sensible defaults.

### Parameter List

| Parameter | Default | Description |
|-----------|---------|-------------|
| `--addr` | `:8023` | HTTP listen address, e.g., `:8023` or `0.0.0.0:9090` |
| `--data` | `data` | Data storage directory, including database, logos, detection files, and GeoIP database |
| `--log-dir` | `logs` | Log file directory |
| `--jwt-secret` | *(auto-generated)* | JWT secret key; auto-generated on each startup if empty |
| `--reset-user` | *(none)* | Reset admin credentials with specified username (generates random password, then exits) |

> **Path rules**: If `--data` and `--log-dir` receive relative paths, they are resolved **relative to the executable's directory**; absolute paths are also accepted.

### Data Directory Structure

At runtime, the directory specified by `--data` automatically creates the following subdirectories and files:

```
data/
├── db/
│   └── iptv.db          # SQLite database file
├── logos/                # Uploaded channel logo files
├── detect/              # ffprobe binary and detection temp files
└── geoip/               # GeoLite2-City.mmdb (auto-downloaded)
```

## Deployment Guide

### Manual Deployment (Binary Execution)

This project has packaged the frontend build artifacts and static files into a Go binary. You only need to download the executable for your platform to run without setting up an environment.

1. **Obtain the package**: Go to [Releases](https://github.com/super321/iptv-tool/releases/latest) to download the latest version for your system, and extract it to any directory.
2. **Run the program**:
   Run the executable in the command line or terminal:
   ```bash
   # Start with default configuration
   ./iptv-server

   # On Windows, double-click to run or execute .\iptv-server.exe
   ```
3. **Access the system**: Open `http://127.0.0.1:8023` in your browser (check the terminal startup log for the actual address).

**More startup examples**:

```bash
# Custom listen port
./iptv-server --addr :9090

# Custom data and log directories (relative paths, relative to executable location)
./iptv-server --data mydata --log-dir mylogs

# Custom data and log directories (absolute paths)
./iptv-server --data /opt/iptv/data --log-dir /var/log/iptv

# Reset admin account (prints new password, then exits)
./iptv-server --reset-user admin
```

### Docker Deployment

In Docker, the executable is located at `/app/iptv-server` with the default data directory at `/app/data`. Use `-v` to mount volumes for data persistence.

> **Tip**: The Docker runtime image comes pre-installed with `ffmpeg` (including ffprobe), enabling channel detection out of the box without manually uploading an ffprobe binary.

#### Run Directly

```bash
# Basic run (persist data to host directory)
docker run -d \
  --name iptv-tool \
  -p 8023:8023 \
  -v /your/path/data:/app/data \
  super321/iptv-tool:latest

# Custom port + persist data and logs
docker run -d \
  --name iptv-tool \
  -p 9090:9090 \
  -v /your/path/data:/app/data \
  -v /your/path/logs:/app/logs \
  super321/iptv-tool:latest \
  --addr :9090

# Enable HTTPS (map HTTPS port as well)
docker run -d \
  --name iptv-tool \
  -p 8023:8023 \
  -p 8024:8024 \
  -v /your/path/data:/app/data \
  super321/iptv-tool:latest
```

#### Use Docker Compose

Copy the `docker-compose.yml` file from the `docker` directory, then execute:

```bash
docker compose up -d
```

For custom configuration, refer to the following example:

```yaml
services:
  iptv-tool:
    image: super321/iptv-tool:latest
    container_name: iptv-tool
    volumes:
      - /your/path/data:/app/data
      - /your/path/logs:/app/logs
    ports:
      - "8023:8023"
      - "8024:8024"     # HTTPS port (if enabled)
    command: ["--addr", ":8023"]
    restart: unless-stopped
```

## Usage Instructions
1. **Add Data Sources**: First, in the "Live Source Management" or "EPG Source Management" menu, add the network URLs you want to aggregate.
2. **Confirm Fetch Status**: Wait for the scheduled background fetch or click manual refresh. Ensure the data counts (channels, programs) display correctly and the last update time is current.
3. **Configure Aggregation Rules** (Optional): In the "Aggregation Rules" menu, create alias, filter, or group rules to normalize channel information. Use the AI-assisted generation feature to speed up rule creation.
4. **Edit Publishing Interface**: Go to the "Aggregated Publishing" menu, click "Add Publishing Interface", select the corresponding live sources and EPG sources for association, and link your created aggregation rules.
5. **Get Subscription**: In the "Aggregated Publishing" list, click the `Address Path` below the corresponding name to get the actual m3u or xmltv link for your player (e.g., Tivimate, Kodi, etc.).

## Changelog

For detailed version update records, please refer to the [CHANGELOG](CHANGELOG.md).

## Star History

<a href="https://www.star-history.com/?repos=super321%2Fiptv-tool&type=date&legend=top-left">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/image?repos=super321/iptv-tool&type=date&theme=dark&legend=top-left" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/image?repos=super321/iptv-tool&type=date&legend=top-left" />
   <img alt="Star History Chart" src="https://api.star-history.com/image?repos=super321/iptv-tool&type=date&legend=top-left" />
 </picture>
</a>

## Disclaimer

The original intention of this project is to provide help and feasibility practice for research, learning, and related technical exchanges. The project itself does not contain, host, or provide any live stream source data or TV program content copyright.

> **⚠️ Note:** No one may use this project and its source code for any illegal or improper purpose, including but not limited to **commercial use**, **infringement**, or **any destructive operation**. Any legal disputes and adverse consequences caused by excessive scraping, private abuse of this project's functions, or distribution of infringing content shall be borne by the user, and have nothing to do with this project and the developers.
