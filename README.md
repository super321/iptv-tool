# IPTV Tool 管理平台

[![GitHub Release](https://img.shields.io/github/v/release/super321/iptv-tool)](https://github.com/super321/iptv-tool/releases/latest)
[![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/super321/iptv-tool/total)](https://github.com/super321/iptv-tool/releases/latest)
![GitHub License](https://img.shields.io/github/license/super321/iptv-tool)

**简体中文** | [English](README_EN.md)

## 项目简介
IPTV Tool 是一个基于 Go 和 Vue 3 开发的轻量级 IPTV 管理和聚合分发平台。该平台致力于为用户提供便捷的直播源（M3U / TXT 等）管理、EPG（电子节目单）抓取与同步、以及台标管理等功能。通过本地部署，您可以轻松地将多个网络直播源和 EPG 数据聚合，并一键发布聚合后的订阅链接供各端播放器使用。

后端基于 Go 1.25+ (Gin + GORM + SQLite) 纯本地存储结构，前端采用 Vue 3 (Element Plus + Vite) 并编译内嵌至 Go 的单文件二进制中，实现了一键下发，极简部署。

## 核心功能
* **直播源管理**：支持网络URL、本地文件等多种形式的直播源导入与管理，支持 M3U / TXT 格式解析与状态检查。
* **EPG 源管理**：支持配置多个 XMLTV 格式的 EPG 接口源，支持定时刷新拉取以及详细的频道和节目数展示。
* **台标管理**：支持频道台标预览、URL 路径提取与自定义上传，为您提供本地化的台标文件托管服务。
* **聚合发布**：支持将多个直播源和 EPG 数据进行组合，生成统一的订阅链接（如 m3u, xmltv 等格式），可直接用于各类主流 IPTV 播放器。
* **轻量级纯本地运行**：内置 SQLite 数据库，无外部依赖（如 Redis / MySQL），数据完全掌握在自己手中。

## 界面导览

### 直播源管理
管理您的所有直播频道来源，支持多种类型的直播源拉取与更新。
![直播源管理](./docs/images/live.png)

### EPG源管理
统一管理各个来源的电子节目单数据，支持定时抓取，直观展示当前频道数、节目数及最后更新时间。
![EPG源管理](./docs/images/epg.png)

### 台标管理
为您的频道提供统一的台标文件托管与管理页面，可直观预览当前所有配置的台标。
![台标管理](./docs/images/logo.png)

### 聚合发布
可以将整理好的直播源和EPG源进行聚合发布在线接口，生成最终的订阅链接用于播放器。
![聚合发布](./docs/images/publish.png)

## 部署指导

### 手动部署 (二进制运行)
本项目已将前端构建产物及静态文件打包进 Go 二进制文件，您只需下载对应平台的可执行文件即可免配置环境运行。

1. **获取程序包**：前往 `Releases` 下载最新版本的对应系统压缩包，解压压缩包至任意目录。
2. **运行程序**：
   在命令行或终端中运行可执行文件：
   ```bash
   ./iptv-server
   # Windows 环境下如果是直接双击运行即可，或者运行 .\iptv-server.exe
   ```
3. **访问系统**：在浏览器中打开内置的 Web 端端口（默认地址因个人配置而定，通常为 `http://127.0.0.1:8023`，请留意终端启动日志）。

### Docker 部署

#### 直接运行

```bash
docker run -d \
  --name iptv-tool \
  -p 8023:8023 \
  -v /你的路径/data:/app/data \
  super321/iptv-tool:latest
```

#### 使用 Docker Compose

拷贝 `docker` 目录下的`docker-compose.yml`文件，然后执行：
```bash
docker compose up -d
```

## 使用说明
1. **添加数据源**：首先在“直播源管理”或“EPG源管理”菜单中，添加你需要聚合的基础网络URL。
2. **确认拉取状态**：等待后台定时抓取或点击手动刷新，确保数据量（频道数、节目数）显示正常并且最后更新时间也是最新的。
3. **编辑发布接口**：进入“聚合发布”菜单，点击“新增发布接口”，选择对应的直播源及 EPG 源进行混合关联配置。
4. **获取订阅**：在“聚合发布”列表点击所对应名称下方的 `地址路径`，即可获取真实的 m3u 或 xmltv 链接到您的播放器（如 Tivimate, Kodi 等）中食用。

## 更新日志

详细的版本更新记录请查阅 [CHANGELOG](CHANGELOG.md)。

## Star History

<a href="https://www.star-history.com/?repos=super321%2Fiptv-tool&type=date&legend=top-left">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/image?repos=super321/iptv-tool&type=date&theme=dark&legend=top-left" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/image?repos=super321/iptv-tool&type=date&legend=top-left" />
   <img alt="Star History Chart" src="https://api.star-history.com/image?repos=super321/iptv-tool&type=date&legend=top-left" />
 </picture>
</a>

## 免责声明

本项目的初衷是为研究、学习和相关技术交流提供帮助与可行性实践。本项目自身不包含、不托管也不提供任何直播源数据或电视节目内容版权。

> **⚠️ 注意：** 任何人不得将本项目及其源代码用于任何违法或不正当的目的，包括但不限于**商业用途**、**侵权行为**或**任何破坏性操作**。使用者因过度抓取、私自滥用本项目功能或传播侵权内容而导致的任何法律纠纷及不良后果，均由使用者自行承担，与本项目及开发者无关。
