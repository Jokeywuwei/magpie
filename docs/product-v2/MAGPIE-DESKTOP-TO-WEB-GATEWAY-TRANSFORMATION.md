# Magpie 桌面功能剥离与 Web 模型中转站转型审计

> 审计对象：`/root/magpie-original`，`main` @ `7dc85f26e6d77978a0c43ea3af49cd893370518d`（`v0.1.4574`）
> 审计阶段：**只读**。未修改任何源码、前端、后端、配置、Schema 或数据库；未提交、未推送。
> 本文所有结论按 PASS / FAIL / PARTIAL / NOT TESTED 标注。**只标记实际执行过的测试为 PASS。**

---

## 0. 结论摘要

| # | 结论 | 判定 |
|---|---|---|
| 1 | Gateway 是纯 Go HTTP 服务，不导入 GUI 包，可脱离桌面独立运行 | **PASS**（源码 + 实跑） |
| 2 | `magpie serve` 可在无 DISPLAY 的 Linux 上长期运行，且不自启任何 GUI 组件 | **PASS**（实跑，独立 XDG 目录） |
| 3 | `magpie serve --web` 同进程同时提供 gateway 与浏览器管理页 | **PASS**（实跑 3498/3499） |
| 4 | 桌面菜单栏/托盘/通知全部由 `!nogui` 或平台标签隔离；`nogui` 依赖图中 0 个 Wails 包 | **PASS**（`go list` 实测） |
| 5 | Docker 官方镜像 = `-tags nogui` + `CMD ["serve"]` | **PASS**（Dockerfile:12-13,85-86） |
| 6 | `internal/agent`、`internal/library` 不被 gateway 依赖 | **PASS**（`go list -deps` 实测） |
| 7 | `internal/sessions` **被 gateway 硬依赖**（5 个源文件 + 经 `usage` 传递） | **PASS**（依赖实测）→ 移除成本高，标 **PARTIAL** 可行性 |
| 8 | 前端 7/11 个页面已是内置插件，可用插件开关整体隐藏 | **PASS** |
| 9 | 页面隐藏只在 CSS/JS 层，服务端 API 路由始终注册 | **PASS**（源码）→ **需改造** |
| 10 | 产品级用户/租户/RBAC 完全不存在 | **PASS（缺失已证实）** |
| 11 | 无 Dashboard、无独立 Models 管理页、无审计日志 | **PASS（缺失已证实）** |
| 12 | 全量测试套件在本机（Go 1.27.2）存在测试隔离缺陷，合并运行 `synctest` fatal | **FAIL**（可复现 3/3） |

---

## 1. Magpie 现有架构：桌面应用与服务端

### 1.1 单一二进制、三个运行面

`main.go:25` 只做 `cli.Main(version)`，所有命令在 `internal/cli/main.go:220` 的 `run()` 分发。三个运行面共用同一个二进制：

| 运行面 | 入口 | 进程内启动的服务 | 桌面依赖 |
|---|---|---|---|
| 桌面应用 | `gui.Run` `internal/gui/app.go:334`（`!nogui`） | Wails 窗口 + 托盘 + panel + gateway + Web API | 强 |
| Web 页面 | `StartWeb` `internal/gui/web.go:91`（**无 build tag**） | HTTP 页面 + Web API + 可选 gateway | 无 |
| 纯 Gateway | `serve()` `internal/cli/providers_cli.go:1736` | gateway only | 无 |

关键事实：**`internal/gui` 不是桌面专用包**。它共有 204 个非测试 `.go` 文件，其中 66 个带 `!nogui`（桌面实现），其余约 138 个（`api.go`、`web.go`、`usage.go`、`caller_keys.go`、`modelprice.go`、`gatewaymode.go` …）在 `nogui` 下照常编译，承载**整个 Web 控制平面**。这正是「Web UI 已可独立运行」的源码基础。

### 1.2 桌面/服务端的唯一抽象缝：`Windows` 接口

```go
// internal/gui/api.go:156
type Windows interface {
	HidePanel(); ShowMain(view string); Quit()
	OpenURL(url string) error
	OpenFolder(path string) error
	ChooseFolder(title string) (string, error)
	Copy(text string) bool
	FitPanel(height int, g Glide)
	TintPanel(rgba [4]uint8, ms int) bool
	TintTitleBar(rgba [4]uint8, dark bool) bool
	SetTextSize(percent int)
}
```

两个实现：
- 桌面：`*host` `internal/gui/app.go:76`
- 浏览器：`webHost` `internal/gui/web.go:36`（`HidePanel`/`ShowMain`/`FitPanel`/`Tint*` 全部空实现；`OpenFolder`/`ChooseFolder` 返回 "in the browser magpie can't …"）

**这是本次转型最重要的既有资产**：服务端 API 已经面向接口编程，桌面能力是可替换的桩。判定 **PASS**（源码）。

### 1.3 服务启动链

```
main.go:25 main()
 └─ cli.Main            internal/cli/main.go:220
     ├─ ("serve")      → serve()            providers_cli.go:1736
     │                    gateway.New()      :1737
     │                    ListenAndServe(ctx) :1804
     ├─ ("serve --web") → serveCmd()         web_cli.go:225
     │                    gui.StartWeb()     :252  → web.go:91
     └─ ("web")         → webCmd()           web_cli.go:29
                          gui.StartWeb()     :87
```

`StartWeb` 内部（`web.go:105-132`）：监听 → 可选 TLS → 构造 `Handler(webHost{quit}, startBackend())`。`startBackend`（`internal/gui/backend.go:48`）设 `gateway.Window = true`（`backend.go:49`），然后 `serveGateway()`。

`gateway.Window` 是「本进程同时服务页面」的标记，纯 `serve` 不设置它——这解释了 `serve` 启动横幅只打印 API 地址、不打印页面链接。

---

## 2. 桌面菜单栏、托盘、通知：源码位置与依赖

### 2.1 菜单栏（macOS）

| 项 | 位置 |
|---|---|
| 系统菜单（Cmd-Q 等） | `internal/gui/app.go:491` `application.DefaultApplicationMenu()`，`:499` `h.app.Menu.Set` |
| 托盘菜单项构建 | `internal/gui/app.go:534` `trayMenuLabels()` → `:535` `h.app.NewMenu()` → `:536-586` open/pause/version/restart/quit |
| 菜单文案（中英） | `internal/gui/traymenu.go:158` `trayWords`、`:201` `trayMenuLabels`、`:185` `trayText` |

### 2.2 系统托盘

| 项 | 位置 | Build tag | Docker `nogui` |
|---|---|---|---|
| 托盘创建 | `app.go:637` `h.app.SystemTray.New()` | `!nogui` | 排除 |
| 图标/提示 | `app.go:651-661`；`trayglyph.go:28` | `!nogui` | 排除 |
| 单击/双击 | `app.go:700,707`；`trayclick.go:15`、`traydouble.go:15` | app `!nogui`；click/double **无 tag** | 死代码编入 |
| Linux StatusNotifierItem | `traytip_linux.go:23` | `linux && !nogui` | 排除 |
| 托盘用量数字 | `traynumber.go:34`、`trayusage_tray.go:12` | `!nogui` | 排除 |
| Dock | `dock.go:20` `dockStep`；`glide_darwin.go:189` `setDock` | dock.go **无 tag**；实现 `!nogui` | 死代码编入 |

### 2.3 桌面通知

| 项 | 位置 |
|---|---|
| 订阅入口 | `app.go:694` `h.watchAlerts()` |
| 实现 | `internal/gui/quota_alert_notify.go:169` `watchAlerts`（文件头 `//go:build !nogui`） |
| 通知器 | `:27` `type notifier`、`:39` `notifications.New()`、`:103` `send`、`:105` `sendAs` |
| 事件源 | `:189` `newMaskAlerts`、`:193` `gateway.OnMasked`、`:195` `gateway.TurnDone`、`:203` `provider.WatchSignIns`、`:209` `provider.WatchQuotas` |
| 平台投递 | Wails v3：Linux D-Bus `org.freedesktop.Notifications`；macOS `UNUserNotificationCenter`；Windows `go-toast` |
| Bark 推送（跨平台 HTTP） | `internal/bark/bark.go:60` `Push`；定时器 `bark_daily.go:121` `watchBarkDaily` |

**关键结论**：`quota_alert_notify.go` 是 `!nogui`，因此 **Docker/nogui 构建里完全不发任何桌面通知**；`provider.WatchQuotas`（`internal/provider/quota_alert.go:269`）是便携的，负责「算」，`quota_alert_notify.go` 只负责「发」。判定 **PASS**（源码 + `!nogui` 编译边界）。

### 2.4 桌面代码与便携逻辑的纠缠（转型必须处理的点）

| 文件 | 无 tag，编入 Docker | 其中 Web/gateway 真正需要的 |
|---|---|---|
| `trayusage.go` | 是 | `withTrayKinds:237`、`trayWindowKind:135`、`trayChoices:169`（被 `usage.go:295,315` 调用）；`trayTotalSum:481`（被 `bark_daily.go:115`、`trace.go:370` 调用） |
| `traymenu.go` | 是 | `trayLang:166`、`trayText:185`、`notesLang:174`（被 `api.go:3068` Bark 测试、`update.go:558`、`whatsnew.go:415` 调用） |
| `traywords.go` | 是 | `pageWords:20`（解析 `assets/i18n.js`），只被 `trayusage.go` 使用 → 运行期死代码 |
| `quota_alert.go` | 是 | `onAlerts:17`、`notifyProblem:23` 被 `api.go:913,1768` 读取；文案函数运行期死代码 |
| `mask_alert.go` / `signin_alert.go` / `turn_done.go` | 是 | 仅被 `quota_alert_notify.go` 使用 → Docker 中死代码 |
| `panelkey.go` | 是 | `panelShortcut:90`、`panelKeysState:43` 被 `api.go:875,2690` 使用（可复用的纯校验逻辑） |
| `dock.go` / `panelside.go` / `trayclick.go` / `traydouble.go` | 是 | 无 Web 调用 → 纯死代码 |

判定 **PARTIAL**：桌面渲染可整体移除，但 **不能整包删 `internal/gui`**，否则会带走整个 Web 控制平面；需先把上表右列的可复用函数迁出。

---

## 3. Agent、Sessions、Library：源码与依赖

### 3.1 规模

| 包 | 非测试文件 | 非测试 LOC | 测试文件 |
|---|---:|---:|---:|
| `internal/gateway` | 369 | 143,588 | 1,385 |
| `internal/provider` | 324 | 106,164 | 715 |
| `internal/gui` | 204 | 42,524 | 506 |
| `internal/agent` | 201 | 72,405 | 502 |
| `internal/sessions` | 127 | 40,029 | 231 |
| `internal/library` | 115 | 46,634 | 214 |
| `internal/usage` | 61 | 18,183 | 157 |

### 3.2 依赖方向（`go list -deps` 实测）

```
gateway  → sessions   （是）
gateway  → agent      （否）
gateway  → library    （否）
provider → 三者       （否）
middleware → sessions （仅经 usage 传递）
```

`internal/agent` 与 `internal/library` 是 gateway 的**消费者**，不是依赖方：

- `agent → gateway`：51 个非测试文件（如 `agent/applied.go:18`、`agent/agents.go:26`）
- `library → gateway`：14 个非测试文件（`library/mcp_hub.go:52`、`library/mcp_run.go:42`、`library/borrow.go:19`）
- gateway 通过**可空钩子变量**反向接入：`gateway.MCPHub`（`gateway/hub_mcp.go:294`）、`gateway.RunMCP`（`gateway/mcp.go:45`）、`gateway.SharedLibrary`（`gateway/gateway.go:1336`），三者 nil 时对应路由 404/空列表，不影响其他功能。

### 3.3 `internal/sessions` 为什么不能直接删

gateway 有 5 个非测试文件直接导入它：

| 文件 | 导入行 | 用途 |
|---|---|---|
| `internal/gateway/conversations.go` | `:14` | Gateway 会话记录（opt-in） |
| `internal/gateway/hub_mcp.go` | `:17` | Codex thread cwd 回退 |
| `internal/gateway/qoder_session.go` | `:17` | Qoder 会话归属推断 |
| `internal/gateway/sessions_mcp.go` | `:10` | `/mcp/magpie/sessions` 工具 |
| `internal/gateway/title_link.go` | `:11` | Codex 标题指纹 |

此外 gateway 有 47 个非测试文件导入 `internal/usage`，而 `internal/usage` **导入 sessions**（19 个文件）。所以即使删掉上面 5 个直接引用，`sessions` 仍留在 gateway 的构建闭包中。

**重要的是**：真正的路由级会话机制（会话亲和性、请求关联 ID）**已经在 gateway 内部原生实现**，不依赖 `sessions` 包：

- `internal/gateway/affinity.go:104` `type Affinity`、`:569` `affine`、`:294` `sticksPath`（持久化 `affinity.json`）
- `internal/gateway/gateway.go:10477` `SessionHeader = "X-Magpie-Session"`、`:10487` `sessionOf`
- `internal/gateway/gateway.go:10454-10458` 原生会话头（`x-opencode-session`、`x-session-affinity`、`session_id`、`x-claude-code-session-id`、`x-conversation-id`、`x-deepseek-harness-session-id`）

判定：**「会话亲和性保留」PASS（已原生）；「sessions 包整体移除」PARTIAL（需先拆分）**。

### 3.4 Web 侧现状

- Sessions 页：`index.html:76` `data-view="sessions"`；路由 23 条（`internal/gui/sessions.go:116…471`、`sessions_manage.go:103…428`、`sessions_reads.go:78,84`）
- Library 页：`index.html:77`；路由 39 条（`internal/gui/library.go:195…721`）
- Agents 页：`index.html:75`（无 `data-feature`，只能由 gateway mode 隐藏）；路由 13 条（`api.go:1430…2734` + `agent_models.go:248`）

---

## 4. 可以移除的 Web 页面与导航入口

### 4.1 现有导航（`index.html:74-87`，`#prefs` 在 `:113`）

| 页 | view id | 渲染方式 | `data-feature` | 能否隐藏 |
|---|---|---|---|---|
| Agents | `agents` | 静态 `index.html:137` | 无 | 仅 gateway mode（`app.js:45 GATEWAY_HIDES=["agents"]`） |
| Sessions | `sessions` | 插件 `plugins/sessions` | `sessions` | **可**（插件开关） |
| Library | `library` | 插件 `plugins/library` | `library` | **可** |
| Providers | `providers` | 静态 `:150` | 无 | 不可（核心） |
| Routing | `routing` | 插件 `plugins/routing` | `routing` | **可** |
| Routing groups | `groups` | 插件脚本内注册 | `routing groups` | **可** |
| Gateway | `gateway` | 插件 `plugins/gateway` | `gateway` | **可** |
| Usage | `usage` | 插件 `plugins/usage` | `usage` | **可** |
| Masking | `masking` | 插件 `plugins/redact` | `redact` | **可** |
| Plugins | `plugins` | 静态 `:176` | 无 | 不可 |
| Settings | `settings` | 静态 `:178` | — | — |

7 个页面已经是内置插件，`plugins.yml` 里把 `enabled: false` 即可让导航按钮与页面整体消失（`internal/gui/builtins.go:124 builtinsOff` → `api.go:1294 boot["featuresOff"]` → `app.js:51-65 featureStyle()` 注入 `display:none!important`）。

### 4.2 三层隐藏机制

| 机制 | 位置 | 语义 |
|---|---|---|
| 内置插件关闭 | `pluginrt.Manifest.Enabled` + `builtinsOff` | `data-feature="<id>"` 的元素全部隐藏，含导航按钮 |
| Gateway mode | `internal/gui/gatewaymode.go:43 gatewayMode()`；`app.js:45` | 强制隐藏 Agents；Library 收窄到 Instructions/MCP/Skills；Sessions 只读 gateway 记录 |
| 用户自选 | `app.js:23451 navMine()`、`localStorage["magpie.navHidden"]` | 每浏览器 |

Gateway mode 的触发条件（`gatewaymode.go:43-58`）：Settings 的 `gatewayMode` 显式 on/off 优先；否则 `--gateway` 标志；否则「本机无 agent」自动开启。

### 4.3 关键限制：隐藏 ≠ 下线

`sessionRoutes`、`libraryRoutes`、`agentModelsAPI` 在 `api.go:1706,1716,3325` **无条件注册**，没有任何服务端按插件状态/页面可见性拦截的代码。判定 **PASS（源码）**，同时这是转型必须补的洞：一旦作为公网中转站，`/api/sessions/*`、`/api/library/*` 仍可被直接调用（需登录 cookie）。

### 4.4 插件替换页

`internal/plugin/page.go:108` `Replaceable = ["sessions","library","routing","usage"]`，社区插件可用 `replaces` 顶替这四个页面（`pluginpages.js:296-303` + `app.css:4373`）。这是未来「用自研页面替换 Agent 页」的现成机制。

---

## 5. 必须保留的底层业务能力

| 能力 | 源码 | 为什么必须保留 | 判定 |
|---|---|---|---|
| 协议翻译（Chat/Responses/Anthropic/Gemini） | `gateway/chat.go:112`、`responses.go`、`anthropic.go`、`gemini.go` | 中转站核心 | PASS |
| 多协议端点全覆盖 | `gateway/gateway.go:1157-1190`（chat、responses、messages、images、videos、embeddings、rerank、audio、realtime、v1beta） | 与 New API 对齐 | PASS |
| 路由与故障转移 | `gateway/routing.go:851 Rest`、`fallback.go:51 candidate`、`:286 perKey` | 核心 | PASS |
| 亲和性/粘性 | `gateway/affinity.go:104` | 保留会话语义 | PASS |
| 会话标识与关联 ID | `gateway.go:10477 SessionHeader`、`:10487 sessionOf` | 见 §3.3 | PASS |
| Provider / Key / Account 模型 | `provider/provider.go:51 Provider`、`:67 Key`、`provider/account.go:45 Account` | 渠道与凭证 | PASS |
| 模型目录与定价 | `catalog/catalog.go:269 Price`、`provider/models.go:2177` 价格优先级 | 计费核心 | PASS |
| 用量账本 | `usage/usage.go:30 Record`、`:1119 Summary`、`:322 usage.jsonl` | 统计核心 | PASS |
| 网关密钥/限额 | `access/access.go:131 Key`、`access/limit.go:26 Limit` | 令牌管理 | PASS |
| 中间件扩展点 | `internal/middleware/`、`gateway/middleware.go` | 渠道级改写 | PASS |
| 插件运行时 | `internal/pluginrt/pluginrt.go:35 Manifest` | 扩展机制 | PASS |
| 管理 API | `gateway/admin.go:30 AdminPath`、`internal/gui/admin.go:39 adminRoutes` | 远程控制面 | PASS |
| 健康检查 | `cli/healthcheck_cli.go:24` | 运维 | PASS（实跑） |
| 服务化 | `internal/daemon/daemon.go` | 长期运行 | PASS（测试） |

---

## 6. 需要改造为 Web 能力的功能

| 功能 | 桌面现状 | Web 现状 | 改造方向 | 判定 |
|---|---|---|---|---|
| 服务状态/启停 | 托盘菜单 Pause/Resume/Quit（`app.go:540-586`） | Gateway 页有 Restart/Start；`POST /api/gateway/restart` `gateway_takeover.go:55` | 增加 Start/Stop/Drain 与 uptime | PARTIAL |
| 异常告警 | 系统通知（`quota_alert_notify.go`，`!nogui`） | **Docker 下无任何通知**；`notifyProblem` 恒 nil | 新增站内通知中心 + Webhook | FAIL（Web 侧缺失） |
| 托盘用量卡片 | `trayusage.go:802 trayCardsView` | Usage 页已有数据 | 复用数据，重建卡片组件 | PARTIAL |
| Agent 管理 | Agents 页 + 配置写入 | 同页面，gateway mode 隐藏 | 移出主产品，降级为「本地客户端」可选模块 | PASS（可隐藏） |
| Sessions 页 | 读本地 agent 会话文件 | 同页面 | 移除页面，保留 gateway 会话记录 | PARTIAL（需拆分包） |
| Library 页 | 写入各 agent 配置文件 | 同页面 | 移除入口；保留 MCP hub 供 gateway | PARTIAL |
| 桌面快捷键 | `panelkey_app.go:22` | 无（也不需要） | 移除 | PASS |
| 文件夹选择器 | Wails dialog | `webHost.ChooseFolder` 报错 | 改为路径文本框（已实现） | PASS |

---

## 7. Gateway 脱离桌面应用独立运行：可行性

### 7.1 结论：完全可行，无需改造

| 验证项 | 方法 | 结果 |
|---|---|---|
| gateway 不导入 gui | `go list -deps ./internal/gateway` | 0 个 `internal/gui`；**PASS** |
| `serve` 不调用任何 GUI 函数 | 读 `providers_cli.go:1736-1815` | 无 `gui.*` 调用；**PASS** |
| `nogui` 构建无 Wails | `go list -tags nogui -deps ./... \| grep -c wails` | **0**；PASS |
| 默认构建有 Wails | 同上（无 tag） | 27；PASS |
| `nogui` 可构建 | `CGO_ENABLED=0 go build -tags nogui` | exit 0；**PASS（实跑）** |
| 独立运行 `serve` | 隔离 XDG，端口 3499 | 启动成功；**PASS（实跑）** |
| 独立运行 `serve --web` | 端口 3498/3499 | 页面 200、`<title>magpie</title>`；**PASS（实跑）** |
| 健康检查 | `magpie healthcheck --addr 127.0.0.1:3499` | exit 0；**PASS（实跑）** |
| 无桌面库加载 | `lsof -p <pid> \| grep -ci "gtk\|webkit\|X11"` | 0；**PASS（实跑）** |
| 未知模型返回结构化错误 | `POST /v1/chat/completions` | 404 + `not_found_error`；**PASS（实跑）** |

实测启动日志（隔离目录 `/tmp/magpie-audit`）：

```
● magpie gateway on http://127.0.0.1:3499
  OpenAI   http://127.0.0.1:3499/v1/chat/completions · http://127.0.0.1:3499/v1/responses
  Anthropic http://127.0.0.1:3499/v1/messages
! no models yet · magpie provider add deepseek sk-…
```

进程内**没有**窗口、托盘、通知、面板、Dock 代码路径。

### 7.2 进程模型：无强耦合

| 问题 | 事实 |
|---|---|
| 桌面进程死了 gateway 会停吗 | 不会。`serve` 是独立 OS 进程（`serve_wait_test.go:13-68` 覆盖「app 服务中 → serve 等待 → app 退出 → serve 接管」） |
| 两个 magpie 能同时跑吗 | 同地址不能（端口绑定，`gateway/listen.go:18`）；不同端口可以；`gateway.serving.lock`（`serving.go:37`）保证后台任务只跑一份 |
| 需要单实例锁吗 | `instanceLock`（`gui/instance_unix.go:30`）只在 macOS app 路径使用 |
| 页面与 gateway 必须同进程吗 | 不必。`magpie web` 发现已有 gateway 时只显示其状态并待命（`web.go:96-109`、`backend.go:103-120`） |

判定 **PASS**。

### 7.3 唯一的真实耦合：Web 控制平面寄生在 `internal/gui`

`gateway.SetAdminHandler` 只在 `internal/gui/admin.go:97` 的 `init()` 中调用；`/api/*` 的 382 条路由全部注册在 `internal/gui/*.go`。因此：

- **能**：一个只跑 `serve` 的进程提供完整模型中转 API。
- **不能**：一个不链接 `internal/gui` 的进程提供 Web 管理 API。

`internal/cli` 无条件导入 `internal/gui`（`web_cli.go:15`、`daemon_cli.go:14`、`gui_on.go:9`），所以「不链接 gui」在当前 CLI 结构下做不到。判定 **PARTIAL**：需要把控制平面从 `internal/gui` 拆出（见 §11 P0）。

---

## 8. Linux 服务器部署可行性

### 8.1 已具备

| 能力 | 证据 | 判定 |
|---|---|---|
| headless 运行 | 实跑（无 DISPLAY） | PASS |
| systemd 用户服务 | `daemon/systemd.go:19`（`$XDG_CONFIG_HOME/systemd/user/magpie.service`）、`Restart=on-failure` | PASS（测试 `TestSystemdInstallAndUninstall` 通过） |
| 服务 + 页面一体 | `daemon install --web` → `serve --web=addr`（`daemon.go:142-147`；`TestInstallWithWeb` 通过） | PASS |
| 开机免登录 | `loginctl enable-linger` 提示（`systemd.go:170-183`） | PASS（源码） |
| 端口/地址 | `MAGPIE_ADDR`（`settings.go:2264`、`gateway.go:346`） | PASS |
| 数据目录 | XDG：`~/.config/magpie`、`~/.cache/magpie`、`~/.local/state/magpie`（`appdir/appdir.go:262,294,310`） | PASS（`magpie data` 实跑） |
| 凭证 | `providers.json`、`logins.json`、`caller-keys.json` 均 0600 明文；Linux 不走 keychain | PARTIAL（安全，非阻塞） |
| 健康检查 | `healthcheck` exit 0/1，drain 时 503 | PASS（实跑 + 源码） |
| 日志 | serve 走 stdout/stderr；systemd → journald；launchd → `serve.log`；**无内建轮转** | PARTIAL |
| 升级 | `serve_follow.go:68` 检测二进制被替换 → 空闲后 re-exec；容器拒绝自更新（`update_cli.go:278`） | PARTIAL |
| Windows 服务 | `daemon.go:71-73` 明确不支持 | N/A |
| 桌面库依赖 | `nogui` 下无 GTK/WebKit/D-Bus 会话 | PASS（`go list`） |

### 8.2 部署形态建议（现状即可）

```bash
# 方式 A：systemd 用户服务 + 页面
magpie daemon install --web --lan
# 方式 B：直接前台
MAGPIE_ADDR=0.0.0.0:3425 MAGPIE_WEB_KEY=… magpie serve --web=0.0.0.0:3430 --tls-cert … --tls-key …
# 方式 C：Docker
docker run -p 3425:3425 -p 3430:3430 -v magpie:/config ghcr.io/yetone/magpie \
  magpie web --addr 0.0.0.0:3430 --no-open
```

注意（文档 `docs/reference.md:3820`）：官方镜像默认 `CMD ["serve"]` **不带 Web UI**，要页面必须覆盖命令为 `magpie web`。

### 8.3 现存的服务器风险

| 风险 | 说明 | 判定 |
|---|---|---|
| 单密钥门禁 | 整个页面一个 `MAGPIE_WEB_KEY`，无用户概念；key 进 URL | PARTIAL |
| 明文密钥 | providers.json 0600，但备份默认带 key | PARTIAL |
| 无审计日志 | 管理操作无独立审计（只有 usage 请求日志） | FAIL |
| 无速率限制（页面层） | 网关层有 key 并发限制；登录页无 | PARTIAL |
| 无反向代理文档化头 | `MAGPIE_TRUST_PROXY`、`MAGPIE_PUBLIC_URL` 存在 | PASS |

---

## 9. 新版 Web 信息架构（目标）

以 Magpie 现有能力为底座，参考 New API 的产品形态，九大模块落位如下：

| 模块 | 复用（源码） | 扩展 | 新增 | 对应现有页 |
|---|---|---|---|---|
| **Dashboard** | `usage` 汇总（`usage.go:1119 Summary`） | 聚合健康、今日成本、渠道状态 | 页面 | 无（新） |
| **渠道/Providers** | `provider.Provider`、`provider/` 324 文件、Providers 页（`index.html:150`） | 渠道优先级/权重 UI（`provider.go:783 Order`、`AccountWeights:234` 已有数据） | 批量导入/导出 UI | Providers |
| **模型/Models** | `catalog.Price`、`provider/models.go` 价格优先级、`/api/model-price` | 独立模型列表页（当前价格在 Usage 行右键） | 模型映射表 UI（`ModelWires` 已有） | 无（新）＋ Settings › Models |
| **路由/Routing** | `routing.go`、`fallback.go`、`routes.Strategy`、Routing/Routing groups 页 | 策略编辑器、可视化（Routing 页已有实时图） | — | Routing / Routing groups |
| **令牌/Tokens** | `access.Key` + `Limit`、Gateway 页 `keys.js`、`POST /v1/magpie/keys` | RPM 限制、密钥自助页（`/me` 已有） | 批量签发/OIDC | Gateway |
| **用户/Users** | 无 | — | 全新（用户、组、RBAC） | 无 |
| **统计/日志** | `usage.Record`、Usage 页、`/v1/magpie/usage*` | 审计日志（管理操作） | Analytics 页 | Usage |
| **插件/Plugins** | `pluginrt`、`plugins/` 15 个、Plugins 页 | 中间件可视化配置 | — | Plugins / Middleware |
| **系统设置** | Settings 8 个子页、`settingsapi.Keys` | 安全（登录、会话、2FA） | 运行时（日志级别、轮转） | Settings |

导航建议（图标沿用现有 SVG）：

```
▸ Dashboard
▸ 渠道      Providers · 凭证/账号 · 导入
▸ 模型      Models · 映射 · 定价
▸ 路由      Routing · 路由组 · 回退策略
▸ 令牌      API Keys · 限额 · 配额
▸ 用户      Users · Groups · 权限
▸ 数据      Usage · Logs · Costs · Analytics
▸ 扩展      Plugins · Middleware
▸ 系统      Settings（安全/网络/运行时）
```

---

## 10. Magpie 与 New API 的能力差距

New API 能力取自其官方 README（`QuantumNous/new-api`，52k stars，Go）：

| 维度 | New API | Magpie 现状 | 差距 |
|---|---|---|---|
| OpenAI Chat / Responses / Anthropic / Gemini | 有 | 有（`gateway.go:1157-1190`） | 持平 |
| 图像/音频/Embeddings/Rerank/Realtime | 有 | 有（:1170-1190） | 持平 |
| 异步任务 API（图片/视频插件） | 有（`/v1/tasks/*` JS 插件） | 部分（`/v1/videos` 原生） | **PARTIAL** |
| 模型映射 | `model_mapping` | `ModelWires` + `model-map` 中间件（README:238 明确对齐） | 持平 |
| 渠道优先级/权重/重试/亲和 | 有 | 有（`provider.Order`、`KeyWeight`、`routes` 策略、`affinity`） | 持平 |
| 配额/订阅/用量/缓存计费 | 有 | 有（`usage` + `catalog.Price` 分级/峰值） | 持平 |
| 表达式定价 | 有 | 分级 + 峰值，无任意表达式 | **PARTIAL** |
| **用户/组/细粒度权限** | 有 | **无** | **FAIL** |
| **OAuth/OIDC 登录、Passkey、2FA** | 有 | 无（`internal/oidc` 只做 CI token → 短期 key 兑换） | **FAIL** |
| 登录会话管理 | 有 | 仅单页面 key + cookie | **FAIL** |
| Playground | 有 | 无（`/deepseek-chat` 聊天页已移除） | **FAIL** |
| 审计日志 | 有 | 无（只有 usage 请求日志） | **FAIL** |
| 多语言控制台 | 7 种 | 5 种（en/zh/zh-TW/ja/de，`app.js:19491 LOCALES`） | PARTIAL |
| 多渠道批量管理 | 有 | 单实例、无批量导入（有 `Import…`） | PARTIAL |
| 单密钥 vs 多用户 | 多用户 | 单用户 | **FAIL** |

**结论**：Magpie 在**协议覆盖、路由引擎、亲和性、计费粒度、插件扩展性**上已达到或超过 New API 的网关层；差距集中在**产品层**——用户体系、认证、审计、独立模型管理页、Dashboard。

---

## 11. 分阶段改造计划

### P0 · 控制平面解耦（1–2 周）

**目标**：把 Web 管理 API 从 `internal/gui` 拆到 `internal/webapi`，使 `serve` 可以不带渲染层独立提供控制面。

**动作**
1. 新建 `internal/webapi`，迁入 `api.go` 的路由装配、`handler` 与 `Windows` 接口。
2. `internal/gui` 只保留 `web.go` 的监听与静态资源嵌入，调用 `webapi.Handler`。
3. 把 `trayusage.go` 中 `withTrayKinds`/`trayTotalSum`/`trayWindowKind` 迁到 `internal/quota`；`traymenu.go` 的 `trayLang`/`trayText` 迁到 `internal/uilang`。
4. 给 `trayclick.go`、`traydouble.go`、`dock.go`、`panelside.go`、`trayimage_other.go`、`dockflap_other.go` 补 `!nogui` 标签。

**验收**：`go build -tags nogui ./...` 通过；`go list -tags nogui -deps ./internal/webapi` 不含 `internal/gui`；`magpie serve` 起 3425，`magpie serve --web` 起 3430；`TestWebGuard`、`TestGatewayMode*` 单跑通过。
**风险**：`api.go` 3,984 行、102 条路由，搬迁冲突面大。
**回滚**：`internal/webapi` 改为薄封装直接转调 `gui.Handler`，保持二进制行为不变。

### P1 · 导航与页面收编（1 周）

**目标**：默认隐藏 Agents/Sessions/Library，建立 Dashboard 骨架。

**动作**
1. `plugins.yml` 默认 `sessions/library` 关闭；`plugins/routing` 保留（中转站需要路由）。
2. 新增 `web` 构建预设：把 `builtinsOff` 的默认集合改为中转站版本。
3. 服务端对 `/api/sessions/*`、`/api/library/*` 加 feature gate（当前只在 JS 层）。

**验收**：浏览器导航只剩 Dashboard/Providers/Models/Routing/Gateway(Tokens)/Usage/Settings；直接 `curl` 已关闭页面的 API 返回 404/403；`go test -tags nogui ./internal/gui` 通过（注意 §12 的隔离缺陷）。
**风险**：gateway mode 与插件开关语义重叠，可能双重隐藏。
**回滚**：恢复 `plugins.yml` 与 `builtinsOff` 默认值。

### P2 · Dashboard + Models 页（2 周）

**目标**：补齐 New API 的信息架构缺口。

**动作**
1. Dashboard：复用 `usage.Summary` + gateway `/` 状态 + provider 健康。
2. Models 页：把 `provider/models.go` 的模型目录、`/api/model-price`、`ModelWires` 映射聚合为独立页。
3. 审计日志：新增 `audit.jsonl`，在 `gateway/admin.go` 与 `gui/admin.go` 的写操作处记录。

**验收**：Dashboard 首屏 < 1s；Models 页能对单个模型设价并生效（`Record.CostAt` 变化）；审计条目含操作者 key、动作、目标、时间。
**风险**：`usage.Summary` 为全量扫描，大账本下慢。
**回滚**：新页面走独立路由前缀 `/api/v2/*`，出问题直接摘除导航。

### P3 · 用户与令牌体系（3–4 周）

**目标**：从单密钥升级为多用户。

**动作**
1. 新增 `internal/identity`（User/Group/Role/Session），存储沿用 `steady.WriteFile` 0600 模式。
2. 登录：用户名密码 + TOTP；OAuth/OIDC 后置。
3. `access.Key` 增加 `OwnerID`，`usage.Record` 增加 `UserID`（当前只有 `CallerKeyID`）。
4. 页面会话从单 cookie 改为 token + session 表。

**验收**：两个用户各自登录、各自只见自己的 key 与用量；现有 `MAGPIE_WEB_KEY` 兼容为 admin 用户；`go test -tags nogui ./internal/access` 通过。
**风险**：`usage.Record` 结构变更影响 1,385 个 gateway 测试。
**回滚**：`OwnerID`/`UserID` 设为可选字段，默认空表示「单用户模式」。

### P4 · 运维面（1 周）

**动作**：服务启停/重建、日志轮转、告警中心（替代桌面通知）、Webhook。
**验收**：Docker 下能收到配额告警（当前完全收不到，见 §6）；日志按大小轮转。
**风险**：通知去重与限流。
**回滚**：告警中心独立开关，默认关。

### P5 · 插件与中间件可视化（2 周）

**动作**：中间件参数表单（`pluginrt.Schema` 已能生成 JSON Schema）、任务插件 API（对齐 New API `/v1/tasks/*`）。
**验收**：在页面上启用 `model-map` 并立即生效；`/v1/tasks/*` 走通一个视频任务。
**风险**：任务 API 与现有 `/v1/videos` 语义冲突。
**回滚**：任务路由挂在独立前缀。

---

## 12. 实际运行结果

### 12.1 已执行（结论可复核）

| 命令 | 结果 |
|---|---|
| `go list -deps ./internal/gateway \| grep -E 'agent\|library\|sessions'` | 仅 `internal/sessions` |
| `go list -deps ./internal/provider \| grep …` | 空 |
| `go list -deps ./internal/middleware \| grep …` | 仅 `internal/sessions`（经 usage） |
| `go list -tags nogui -deps ./... \| grep -ci wails` | `0` |
| `go list -deps ./... \| grep -ci wails` | `27` |
| `CGO_ENABLED=0 go build -tags nogui -o /tmp/magpie-audit-check .` | **exit 0** |
| `go test -tags nogui -count=1 ./internal/daemon/ ./internal/access/` | **ok** |
| `go test -tags nogui -count=1 -run 'TestWeb|TestStartWeb|TestServedGateway' ./internal/gui/` | **ok** |
| `go test -tags nogui -run 'TestSystemdInstallAndUninstall\|TestInstallWithWeb\|TestWebNotInstalled' ./internal/daemon/` | **全 PASS** |
| `go test -tags nogui -run 'TestWebGuard\|TestWebKey\|TestWebGuardBookmark' ./internal/gui/` | **全 PASS** |
| `go test -tags nogui -run 'TestGatewayMode$'` 等三条单独运行 | **各自 PASS** |
| 上述三条合并运行 | **FAIL**（见 12.2） |
| 隔离 XDG 启动 `magpie serve`（3499） | 启动成功，`/v1/models` 200，未知模型 404 |
| `magpie healthcheck --addr 127.0.0.1:3499` | exit 0 |
| `lsof` 检查该进程 GTK/WebKit/X11 | 0 |
| `magpie serve --web=127.0.0.1:3498` | 页面 200、`<title>magpie</title>` |
| `magpie web --gateway`（3497）浏览器实测 | `body.gateway-mode=true`，`agents` 导航 `hidden=true`，其余 9 个可见 |
| `/v1/magpie/admin/providers` 无 admin key | 401 `admin_key_required` |
| `/api/state` 在 gateway 端口 | 404（控制面只在页面端口） |
| `/api/state` 在页面端口无 cookie | 401 |

截图：`/root/magpie-obs/audit-shots/`（gateway mode 前后）。

### 12.2 失败项（真实缺陷，必须报告）

```
fatal error: reset of synctest timer from outside bubble
  time.(*Timer).Reset
  internal/usage.lingerScratch        request_page.go:2024
  internal/usage.(*chunkRows).release request_page.go:720
  internal/usage.countGateway         gateway_sessions.go:457
  internal/gui.listedExternal         sessions.go:96
  internal/gui.TestGatewayModeListsNoOwnSessions
```

- 复现率：**3/3**（每次都复现）。
- 单独运行 `TestGatewayModeListsNoOwnSessions`：**PASS**（0.13s）。
- 原因：`scratchSweep.t` 是包级 `*time.Timer`，在 `synctest` 气泡外用 `Reset` 触发 Go 运行时 fatal。
- 环境：`go.mod` 声明 `go 1.26.8`，本机 `go1.27.2`；CI 用 `go-version-file: go.mod`（`.github/workflows/test.yml:44`），所以这是**本机工具链差异**暴露的测试隔离缺陷，不是产品逻辑缺陷。
- 影响：**全量 `go test ./internal/gui` 在本机不可作为放行依据**；CI 上是否复现 **NOT TESTED**。
- 结论：**FAIL（测试隔离）**，建议改用 `t.Cleanup` 在气泡内停止 timer，或在测试内改为局部 timer。

### 12.3 未执行（不得视为 PASS）

| 项 | 原因 | 标记 |
|---|---|---|
| `go test ./internal/gateway`（1,385 个测试） | 用户中断，耗时过长 | **NOT TESTED** |
| `go test ./internal/sessions` / `./internal/usage` | 同上 | **NOT TESTED** |
| 真实 Provider 调用（无 API Key） | 任务禁止填真实 Key | **NOT TESTED** |
| Docker 镜像构建/运行 | 环境无 docker | **NOT TESTED** |
| systemd 服务真实安装 | 会写入系统目录，超出只读审计范围 | **NOT TESTED** |
| 桌面 app 的托盘/通知（macOS/Windows） | Linux headless 环境无法验证 | **NOT TESTED** |
| 反向代理 + TLS 真实链路 | 未配置证书 | **NOT TESTED** |

---

## 13. 尚未解决的问题

1. **控制平面位置**：`internal/gui` 同时是桌面渲染层和 Web API 层，是本次转型最大的结构性障碍。P0 必须先解决。
2. **服务端页面权限**：页面隐藏纯 JS，API 始终开放；公网部署前必须补服务端 gate。
3. **多用户模型缺失**：`access.Key` 无 owner，`usage.Record` 无 user 维度，这是 P3 的硬前提。
4. **`internal/sessions` 与 gateway 的近耦合**：移除 Sessions 页面容易，移除包很难；建议保留包、只摘页面。
5. **通知在 Web 下完全缺席**：`!nogui` 边界正确，但 Web 侧没有替代品，属于产品缺口而非 bug。
6. **测试隔离缺陷**：`internal/usage/request_page.go:2024` 的包级 timer 需要修，否则 Web 改造期间无法靠全量测试回归。
7. **明文凭证**：providers.json 0600 明文，作为多用户公网服务需要评估 KMS/信封加密。

---

## 14. 交付清单

| 项 | 状态 |
|---|---|
| 报告路径 | `docs/product-v2/MAGPIE-DESKTOP-TO-WEB-GATEWAY-TRANSFORMATION.md`（本文件） |
| 源码修改 | **无**（`git status` 仅 `?? .mimosa/` 与本次新增的 `docs/product-v2/`） |
| Git 提交/推送 | **无** |
| 生产实例 | 保持运行：`./magpie web --lan --no-open`，3425/3430 |
| 审计实例 | 已停止（3497/3498/3499） |
| 审计产物 | `/root/magpie-obs/audit-shots/`、`/tmp/magpie-audit/`（隔离配置，未污染生产配置） |

**本阶段到此为止，等待审核与明确授权后再进入改造。**