# v2.0.10 (2026-04-10)

## ✨ Features

- feat: 聚合规则新增支持测试功能 (35a6ef7)
- feat: 优化增强频道分组和过滤规则 (#9) (1f0b295)

## 🐛 Bug Fixes

- fix: 修改部分前端页面报错问题 (3280cdc)

## ⚡ Performance

- perf: 优化台标管理页面的性能 (331a19c)
- perf: 优化运行和访问日志的前端页面性能 (652e357)

# v2.0.9 (2026-04-04)

## ✨ Features

- feat: 台标管理新增批量删除功能 (4dcf4e7)
- feat: 直播类型发布接口新增支持单播协议的转换配置 (08783c8)
- feat: 优化支持所有聚合规则的AI生成功能 (cdbc57b)
- feat: 新增支持HTTPS配置和启用 (a2e09ce)

## 🐛 Bug Fixes

- fix: 修复UA校验问题（如有配置，需重新编辑保存） (730dc69)
- fix: 部分功能兼容支持IPv6 (80cf894)

# v2.0.8 (2026-04-02)

## ✨ Features

- feat: 新增一款IPTV类型EPG源的抓取策略实现 (b2e8a0c)
- feat: 优化一键破解功能，并新增十六进制密钥模式破解 (311448d)
- feat: 聚合发布接口新增一键下载功能 (2866a9a)
- feat: 定时任务新增支持指定每天的执行时间 (#10) (432e928)

## 🐛 Bug Fixes

- fix: 将运行日志和访问日志的缓冲区容量缩减到5k (9b84f6a)
- fix: 修复联通IPTV模拟认证失败的问题 (#11) (78aff5d)
- fix: 修复直播源或EPG源同步期间被删除导致脏数据的问题 (c0885cc)
- fix: 增加一键破解的超时时间 (11840f1)

# v2.0.7 (2026-03-27)

## ✨ Features

- feat: 新增支持数据的导出/导入功能 (be1f727)
- feat: 直播类型聚合接口，新增支持按直播源独立配置输出参数 (5f37ba1)
- feat: 新增命令行参数可一键重置用户信息 (ca1377b)
- feat: 组播代理支持配置自定义参数 (50b649d)
- feat: 各功能列表增加名称搜索功能 (09ebca2)
- feat: 新增深色主题以及主题切换功能。 (15a9b99)

## 🐛 Bug Fixes

- fix: 直播源接口部分错误提示未正常展示 (83cf683)
- fix: 修改直播类型聚合接口，组播协议的选项名称和默认值 (5e1763a)

# v2.0.6 (2026-03-22)

## ✨ Features

- feat: 新增支持繁體中文语言切换和展示 (dfcd91f)
- feat: 访问控制页面新增访问统计 (c5d5fc4)
- feat: 新增系统运行日志与访问日志的查询和下载功能 (e8fd269)
- feat: 网络XMLTV类型EPG源支持配置自定义请求头 (2bbf039)
- feat: 新增支持系统全局访问控制，支持黑/白名单模式配置 (af39dc2)
- feat: 聚合发布接口支持配置UA（User-Agent）校验 (32f2d80)

## 🐛 Bug Fixes

- fix: 优化批量上传台标的提示信息 (1e31940)
- fix: 访问控制页面的前端样式问题 (d7e053a)
- fix: 修改和清理国际化资源文件 (fcb413c)
- fix: 优化AI生成频道分组的提示词和校验逻辑 (1843aeb)
- fix: 修改AI生成频道分组的提示信息 (657ada7)
- fix: 修复频道别名类型的替换问题 (1c59ae7)

## ⚡ Performance

- perf: 优化后端服务的性能 (f0b753c)
- perf: 将聚合发布接口的缓存时间增大到15分钟 (b65a2bf)
- perf: 优化SQLite数据库性能参数 (72d7111)

# v2.0.5 (2026-03-18)

## ✨ Features

- feat: 支持AI生成频道分组类型的聚合规则 (9a8ecc8)

## 🐛 Bug Fixes

- fix: 修复IPTV类型直播源和EPG源并发刷新可能导致失败的问题。 (f9ede52)
- fix: 未正确保存和展示自动检测到的可用EPG策略 (f9eaed8)
- fix: 修复最新版本信息的展示样式问题 (8656ecf)

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
