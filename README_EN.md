# IPTV Tool Management Platform

[![GitHub Release](https://img.shields.io/github/v/release/super321/iptv-tool)](https://github.com/super321/iptv-tool/releases/latest)
[![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/super321/iptv-tool/total)](https://github.com/super321/iptv-tool/releases/latest)
![GitHub License](https://img.shields.io/github/license/super321/iptv-tool)

[简体中文](README.md) | **English**

## Project Introduction
IPTV Tool is a lightweight IPTV management and aggregation distribution platform developed based on Go and Vue 3. The platform is dedicated to providing users with convenient features for live streaming sources (M3U / TXT, etc.) management, EPG (Electronic Program Guide) fetching and synchronization, as well as channel logo management. Through local deployment, you can easily aggregate multiple network live stream sources and EPG data, and generate unified subscription links with one click for use by various players.

The backend is built on a pure local storage structure using Go 1.25+ (Gin + GORM + SQLite), and the frontend uses Vue 3 (Element Plus + Vite), compiled and embedded into Go's single-file binary, achieving one-click distribution and minimalist deployment.

## Core Features
* **Live Source Management**: Supports importing and managing live streaming sources in multiple formats, such as network URLs and local files, and supports parsing and status checking for M3U / TXT formats.
* **EPG Source Management**: Supports configuring multiple XMLTV format EPG interface sources, scheduled refresh fetching, and detailed display of channels and program counts.
* **Logo Management**: Supports channel logo preview, URL path extraction, and custom uploads, providing you with a localized logo file hosting service.
* **Aggregated Publishing**: Supports combining multiple live streaming sources and EPG data to generate unified subscription links (such as m3u, xmltv, etc. formats), which can be directly used by various mainstream IPTV players.
* **Lightweight Pure Local Execution**: Built-in SQLite database, no external dependencies (such as Redis / MySQL), completely keeping the data in your own hands.

## Interface Tour

### Live Source Management
Manage all your live channel sources, supporting multiple types of live source fetching and updating.
![Live Source Management](./docs/images/live.png)

### EPG Source Management
Unified management of Electronic Program Guide data from various sources, supporting scheduled fetching, and intuitively displaying the current number of channels, programs, and last updated time.
![EPG Source Management](./docs/images/epg.png)

### Logo Management
Provides a unified logo file hosting and management page for your channels, allowing direct preview of all currently configured logos.
![Logo Management](./docs/images/logo.png)

### Aggregated Publishing
The organized live streaming sources and EPG sources can be aggregated to an online publishing interface, generating the final subscription links for players.
![Aggregated Publishing](./docs/images/publish.png)

## Deployment Guide

### Manual Deployment (Binary Execution)
This project has packaged the frontend build artifacts and static files into a Go binary file. You only need to download the executable file for your platform to run without setting up an environment.

1. **Obtain the program package**: Go to `Releases` to download the compressed package for the latest version corresponding to your system, and extract it to any directory.
2. **Run the program**:
   Run the executable file in the command line or terminal:
   ```bash
   ./iptv-server
   # In Windows environment, you can simply double-click to run, or execute .\iptv-server.exe
   ```
3. **Access the system**: Open the built-in Web port in the browser (the default address depends on personal configuration, usually `http://127.0.0.1:8023`, please note the terminal startup log).

### Docker Deployment

#### Run Directly

```bash
docker run -d \
  --name iptv-tool \
  -p 8023:8023 \
  -v /your/path/data:/app/data \
  super321/iptv-tool:latest
```

#### Use Docker Compose

Copy the `docker-compose.yml` file under the `docker` directory, and then execute:
```bash
docker compose up -d
```

## Instructions for Use
1. **Add Data Sources**: First, in the "Live Source Management" or "EPG Source Management" menu, add the basic network URLs you need to aggregate.
2. **Confirm Fetch Status**: Wait for the background scheduled fetching or click manual refresh, ensure the data volume (number of channels, number of programs) displays normally and the last updated time is up to date.
3. **Edit Publishing Interface**: Enter the "Aggregated Publishing" menu, click "Add Publishing Interface", select the corresponding live streaming source and EPG source for mixed association configuration.
4. **Get Subscription**: Click the `Address Path` below the corresponding name in the "Aggregated Publishing" list to get the real m3u or xmltv links to feed into your player (such as Tivimate, Kodi, etc.).

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
