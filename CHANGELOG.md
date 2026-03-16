# v2.0.4 (2026-03-16)

## ✨ Features

- feat: M3U格式直播接口支持直接使用直播源中频道自带的台标URL (9c2c527)
- feat: 增加检查最新版本的功能。 (73e05cf)

## 🐛 Bug Fixes

- fix: 修改ETH赞助二维码过小的问题 (3784c49)

## ⚡ Performance

- perf: 解决查询EPG发布接口时资源占用高，接口响应慢的问题 (8a5e3ba)
- perf: 优化EPG类型JSON格式的发布接口性能 (0cd855b)
- perf: 聚合发布接口增加缓存机制 (42a0109)

# v2.0.3 (2026-03-14)

## ✨ Features

- feat: 直播类型发布接口支持rtp2httpd的FCC快速换台功能 (b429dc8)
- feat: 频道检测支持选择检测策略（组播或单播优先） (4734e36)
- feat: 直播源频道列表增加展示回看地址。 (f4abb96)

## 🐛 Bug Fixes

- fix: 某些情况下直播类型接口中的频道直播地址展示不正确。 (526eb06)
- fix: 对必要的输入参数添加trim处理 (a392d5e)
- fix: 自动创建EPG源时若已存在则增加提示信息。 (5c49f53)
- fix: 频道检测rtsp协议地址时，强制使用 TCP传输 (5343ff4)
- fix: Changing IPTV source clears existing EPG policy. (ef16124)
- fix: Modify front-end styles (86fc81f)

## ⚡ Performance

- perf: Optimize XMLTV parsing performance (ab9ce8c)
- perf: Optimize database read and write performance (7cc6c9a)

# v2.0.2 (2026-03-11)

## ✨ Features

- feat: Added internationalization (i18n) support. (e7f4c50)

## 🐛 Bug Fixes

- fix: 选择单播优先时部分频道地址没有正确显示 (refs #8) (1e21cc8)
- fix: i18n display issues. (8f6909c)
- fix: 关于系统中补充本项目的仓库链接。 (382f0a6)
- fix: 优化EPG源的节目单展示。 (9baec15)
- fix: 优化直播源频道列表展示和检测逻辑 #7 (d93490b)
- fix: 修改IPTV直播源参数配置时，未同步更新关联的EPG源配置。 (a05719d)
- fix: 优化查看直播源和EPG源的频道列表的排序顺序。 (3054718)
- fix: update web favicon icon (c1cb622)

# v2.0.1 (2026-03-08)

## 🐛 Bug Fixes

- fix: M3U格式直播接口的tvg-name属性优先展示频道别名。 (05b0a9c)
- fix: 优化修改定时检测逻辑。 (c95b9ce)
- fix: 修改优化定时任务执行逻辑。 (f26493f)
- fix: 修改初始化、登录接口问题 (326e413)

# v2.0.0 (2026-03-08)

## ✨ Features

- feat: 补充关于信息 (e1c3505)
- feat: 增加M3U直播源时可自动创建对应的XMLTV格式EPG源 (74cc240)
- feat: 增加直播源有效性探测和过滤功能。 (50e316e)
- feat: 新增/修改网络订阅URL直播源，支持自定义HTTP请求头。 (6e52066)
- feat: 优化直播源、EPG源列表展示，支持数据同步时自动刷新 (94d4dad)
- feat: 增强登录接口的安全性。 (8789863)

## 🐛 Bug Fixes

- fix: 增加版本信息展示。 (871b2f7)
- fix: 优化和修改直播频道检测。 (98bc01e)
- fix: 修改直播源参数展示问题 (9f697d0)
- fix: 增强初始化、登录和修改密码接口的安全性 (0690cd0)
- fix: 修改EPG源管理查看节目单的时间展示问题。 (0f07234)
- fix: 优化日志打印 (b26390e)
- fix: 修改直播接口M3U格式的catchup参数问题。 (8c5d094)
- fix: 将接口返回的错误提示信息统一修改为中文。 (f6dc686)
