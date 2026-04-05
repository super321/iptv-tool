# IPTV Tool 管理平台

[![GitHub Release](https://img.shields.io/github/v/release/super321/iptv-tool)](https://github.com/super321/iptv-tool/releases/latest)
![GitHub License](https://img.shields.io/github/license/super321/iptv-tool)

**简体中文** | [English](README_EN.md)

## 项目简介
IPTV Tool 是一个基于 Go 和 Vue 3 开发的轻量级 IPTV 管理和聚合分发平台。该平台致力于为用户提供便捷的直播源（M3U / TXT 等）管理、EPG（电子节目单）抓取与同步、以及台标管理等功能。通过本地部署，您可以轻松地将多个网络直播源和 EPG 数据聚合，并一键发布聚合后的订阅链接供各端播放器使用。

后端基于 Go 1.25+ (Gin + GORM + SQLite) 纯本地存储结构，前端采用 Vue 3 (Element Plus + Vite) 并编译内嵌至 Go 的单文件二进制中，实现了一键下发，极简部署。

## 核心功能

### 📺 直播源管理
支持多种类型的直播源导入与管理：
- **网络URL直播源**：支持 M3U / TXT 格式的在线订阅地址，支持自定义 HTTP 请求头
- **手动编辑直播源**：支持直接粘贴/编辑频道列表内容
- **定时同步**：支持配置定时刷新间隔（1h ~ 24h），以及指定每天的执行时间
- **频道检测**：基于 ffprobe 的频道有效性探测，支持组播/单播优先策略选择，可自动过滤超时频道
- **频道列表**：直观展示频道名称、分组、直播地址、回看地址、检测延迟及视频编码/分辨率信息

### 📅 EPG 源管理
- 支持配置多个 XMLTV 格式的 EPG 接口源
- 支持网络 XMLTV 源自定义 HTTP 请求头
- 定时刷新拉取，直观展示频道数、节目数及最后更新时间
- 添加 M3U 直播源时可自动创建对应的 XMLTV 格式 EPG 源

### 🖼️ 台标管理
- 支持频道台标文件的上传（单个/批量）、预览和管理
- 支持批量删除
- 提供本地化的台标文件托管，可直接在聚合发布中引用

### 📐 聚合规则
支持创建可复用的聚合规则，在发布接口中灵活组合使用：
- **别名规则**：通过正则或字符串匹配，批量替换频道名称（如统一 "CCTV-1" 为 "CCTV1"）
- **过滤规则**：按频道名称或别名进行正则/字符串匹配，过滤掉不需要的频道（如购物频道）
- **分组规则**：按条件将频道重新归类到自定义的分组中
- **AI 辅助生成**：支持通过 AI 对话辅助生成以上三种类型的规则，预设快捷标签简化操作

### 🔗 聚合发布
将整理好的直播源和 EPG 源进行聚合发布，生成最终的订阅链接：
- **发布格式**：支持 M3U、TXT、XMLTV、DIYP 等多种格式
- **协议转换**：支持组播地址类型选择（UDPxy / RTP / IGMP），支持 FCC 快速换台（rtp2httpd），支持单播协议代理转换
- **按源独立配置**：每个直播源可独立配置输出参数（地址类型、组播协议等）
- **自定义参数**：支持为组播代理地址配置自定义参数
- **Catchup 回看**：M3U 格式支持配置回看模板
- **过滤无效频道**：可指定哪些源的超时频道在发布时自动过滤
- **UA 校验**：支持配置 User-Agent 校验白名单以保护订阅链接
- **一键下载**：支持一键下载生成的订阅文件
- **发布预览**：保存前可即时预览聚合内容

### 🔐 安全与访问控制
- **访问控制**：支持全局 IP 白名单/黑名单模式，支持单 IP、CIDR 网段和 IP 范围
- **登录安全**：RSA-OAEP 密码加密传输，多次失败后触发验证码（CAPTCHA），IP 级别登录限流
- **HTTPS 支持**：支持上传 TLS 证书启用 HTTPS，支持自定义端口，支持双向 TLS（mTLS）认证
- **密码管理**：Web UI 修改密码 + 命令行一键重置管理员凭据

### 📊 日志与监控
- **运行日志**：实时查看系统运行日志（5000 条环形缓冲），支持搜索和下载
- **访问日志**：实时查看 API 访问日志（5000 条环形缓冲），支持搜索和下载
- **访问统计**：记录近 7 天的 IP 访问数据，统计总请求数和订阅请求数
- **GeoIP 地理定位**：自动下载 GeoLite2 数据库，为访问统计中的 IP 提供地理位置信息，支持自动更新

### 🌍 国际化与主题
- 支持简体中文、繁体中文、英文三种语言
- 语言自动检测，支持手动切换和记忆
- 支持深色/浅色主题切换

### 📦 其他功能
- **数据导入/导出**：支持按模块选择性导出和导入配置数据（包括直播源、EPG源、台标、规则、发布接口、检测设置、访问控制），导出为 ZIP 格式
- **版本更新检查**：自动检查 GitHub 最新版本，展示更新日志，一键跳转下载
- **轻量级纯本地运行**：内置 SQLite 数据库，无需 Redis / MySQL 等外部依赖，数据完全掌握在自己手中

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

## 启动参数说明

IPTV Tool 支持通过命令行参数自定义运行配置。所有参数均为可选，有合理的默认值。

### 参数列表

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--addr` | `:8023` | HTTP 监听地址，如 `:8023` 或 `0.0.0.0:9090` |
| `--data` | `data` | 数据存储目录，包含数据库、台标、检测文件和 GeoIP 库 |
| `--log-dir` | `logs` | 日志文件目录 |
| `--jwt-secret` | *(自动生成)* | JWT 密钥，留空则每次启动自动生成 |
| `--reset-user` | *(无)* | 重置管理员凭据，指定新用户名（生成随机密码后退出） |

> **路径规则**：`--data` 和 `--log-dir` 如果传入相对路径，将自动解析为**相对于可执行文件所在目录**的路径；也可以传入绝对路径。

### 数据目录结构

运行时，`--data` 指定的目录下会自动创建以下子目录和文件：

```
data/
├── db/
│   └── iptv.db          # SQLite 数据库文件
├── logos/                # 上传的频道台标文件
├── detect/              # ffprobe 二进制及检测临时文件
└── geoip/               # GeoLite2-City.mmdb（自动下载）
```

## 部署指导

### 手动部署（二进制运行）

本项目已将前端构建产物及静态文件打包进 Go 二进制文件，您只需下载对应平台的可执行文件即可免配置环境运行。

1. **获取程序包**：前往 [Releases](https://github.com/super321/iptv-tool/releases/latest) 下载最新版本的对应系统压缩包，解压压缩包至任意目录。
2. **运行程序**：
   在命令行或终端中运行可执行文件：
   ```bash
   # 使用默认配置启动
   ./iptv-server

   # Windows 环境下直接双击运行即可，或者运行 .\iptv-server.exe
   ```
3. **访问系统**：在浏览器中打开 `http://127.0.0.1:8023`（请留意终端启动日志中的实际地址）。

**更多启动示例**：

```bash
# 自定义监听端口
./iptv-server --addr :9090

# 自定义数据目录和日志目录（相对路径，相对于可执行文件位置）
./iptv-server --data mydata --log-dir mylogs

# 自定义数据目录和日志目录（绝对路径）
./iptv-server --data /opt/iptv/data --log-dir /var/log/iptv

# 重置管理员账号（执行后会打印新密码，然后退出）
./iptv-server --reset-user admin
```

### Docker 部署

在 Docker 中，可执行文件位于 `/app/iptv-server`，默认数据目录为 `/app/data`。通过 `-v` 挂载数据卷即可持久化数据。

> **提示**：Docker 镜像的运行时环境已预装 `ffmpeg`（包含 ffprobe），可直接用于频道检测功能，无需额外上传 ffprobe 文件。

#### 直接运行

```bash
# 基本运行（数据持久化到宿主机目录）
docker run -d \
  --name iptv-tool \
  -p 8023:8023 \
  -v /你的路径/data:/app/data \
  super321/iptv-tool:latest

# 自定义端口 + 持久化数据和日志
docker run -d \
  --name iptv-tool \
  -p 9090:9090 \
  -v /你的路径/data:/app/data \
  -v /你的路径/logs:/app/logs \
  super321/iptv-tool:latest \
  --addr :9090

# 同时启用 HTTPS（需要映射 HTTPS 端口）
docker run -d \
  --name iptv-tool \
  -p 8023:8023 \
  -p 8024:8024 \
  -v /你的路径/data:/app/data \
  super321/iptv-tool:latest
```

#### 使用 Docker Compose

拷贝 `docker` 目录下的 `docker-compose.yml` 文件，然后执行：

```bash
docker compose up -d
```

如需自定义配置，可参考以下示例：

```yaml
services:
  iptv-tool:
    image: super321/iptv-tool:latest
    container_name: iptv-tool
    volumes:
      - /你的路径/data:/app/data
      - /你的路径/logs:/app/logs
    ports:
      - "8023:8023"
      - "8024:8024"     # HTTPS 端口（如需启用）
    command: ["--addr", ":8023"]
    restart: unless-stopped
```

## 使用说明
1. **添加数据源**：首先在"直播源管理"或"EPG源管理"菜单中，添加你需要聚合的基础网络URL。
2. **确认拉取状态**：等待后台定时抓取或点击手动刷新，确保数据量（频道数、节目数）显示正常并且最后更新时间也是最新的。
3. **配置聚合规则**（可选）：在"聚合规则"菜单中，根据需要创建别名、过滤或分组规则来规范化频道信息。可使用 AI 辅助生成功能加速规则编写。
4. **编辑发布接口**：进入"聚合发布"菜单，点击"新增发布接口"，选择对应的直播源及 EPG 源进行关联配置，并关联已创建的聚合规则。
5. **获取订阅**：在"聚合发布"列表点击所对应名称下方的 `地址路径`，即可获取真实的 m3u 或 xmltv 链接到您的播放器（如 Tivimate, Kodi 等）中食用。

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
