# Magpie → Web AI 模型中转站：可执行实施方案

> 基线：`/root/magpie-original`，`main` @ `7dc85f26e6d77978a0c43ea3af49cd893370518d`（`v0.1.4574`）
> 本文件为**只读阶段的实施方案**。本轮未修改任何业务源码、前端、配置、Schema、数据库或测试文件；未提交、未推送。
> 关键结论均已在本轮重新核实，未直接沿用上一份审计报告的声明；凡未能复现或未执行的，一律标 **NOT TESTED**。

---

## 0. 本轮重新核实的事实（与上一份报告的差异）

| 项目 | 上一份报告 | 本轮核实 | 结论 |
|---|---|---|---|
| Web API 路由数 | 382 | **381 条生产注册**（54 个文件）；另有 1 条在测试文件 `pluginpage_test.go` 中，grep 合计 382 | 以 381 为生产口径 |
| 路由的文件分布 | 未细分 | **36 个"核心"文件 101 条**（不 import `agent`/`library`/`sessions`）＋ **18 个"耦合"文件 280 条** | 决定解耦顺序 |
| `internal/gui` 文件构成 | 204 个非测试文件 | 204 个：**136 个可 Web 化**（无 `!nogui`、无 Wails）、**66 个桌面专用**、2 个 `nogui` 专用 | 一致 |
| 136 个可 Web 化文件中耦合的 | 未给出 | **26 个** import `agent`/`library`/`sessions` | 解耦真实成本 |
| 测试缺陷根因 | 怀疑 Go 版本差异 | **已在 Go 1.26.8（go.mod 声明版本）与 Go 1.27.2 上双向复现**，与版本无关 | 根因改写，见 §9 |
| Gateway 依赖 | sessions 硬依赖 | 复测 `go list -deps`：gateway 只 import `sessions`；provider 一个都不 import；middleware 经 `usage` 传递 | 一致 |

本轮实际执行的命令与结果见 §13。

---

## 1. 结论摘要

| # | 结论 | 判定 |
|---|---|---|
| 1 | Gateway 已是纯 Go 服务，不 import GUI，可脱离桌面运行 | **PASS**（`go list -deps` + 实跑） |
| 2 | `internal/gui` 同时是桌面外壳和 Web 控制平面，是唯一的结构性障碍 | **PASS**（204 文件拆分实测） |
| 3 | 381 条路由中 101 条可直接迁出，280 条需先做 facade 解耦 | **PASS**（逐文件统计） |
| 4 | 桌面菜单栏/托盘/通知已由 `!nogui` 隔离，`nogui` 依赖图 0 个 Wails 包 | **PASS** |
| 5 | 产品级用户/组/RBAC 完全不存在；`access.Key` 无 owner，`usage.Record` 无 user | **PASS（缺失已证实）** |
| 6 | 控制面没有关系型数据库，只有 JSON/JSONL；"数据库迁移"实为文件格式演进 | **PASS**（SQLite 仅用于会话索引与外部 App 读取） |
| 7 | 测试套件的包级 `time.Timer` + `synctest` 冲突可复现，且与 Go 版本无关 | **FAIL（缺陷已复现）** |
| 8 | 修复设计 E 在独立复现工程中于两个 Go 版本均通过 | **PASS（在复现工程中）**；仓库内 **NOT TESTED** |

---

## 2. 当前架构事实（重新核实）

### 2.1 单一二进制、三个运行面

```
main.go:25 main()
 └─ cli.Main                        internal/cli/main.go:220
     ├─ (无参数/app/tray/panel)      → gui.Run            internal/gui/app.go:334   [!nogui]
     ├─ ("serve")                    → serve()             internal/cli/providers_cli.go:1736
     ├─ ("serve --web[=addr]")       → serveCmd()          internal/cli/web_cli.go:225 → gui.StartWeb :252
     └─ ("web [--addr] [--lan]…")    → webCmd()            internal/cli/web_cli.go:29  → gui.StartWeb :87
```

- `gui.StartWeb` 定义在 `internal/gui/web.go:91`，**文件无 build tag**，`nogui` 下照常编译。
- `startBackend`（`internal/gui/backend.go:48`）设 `gateway.Window = true` 并启动 gateway；纯 `serve` 不设该标记。
- 纯 `serve` 不调用任何 GUI 函数，但 `internal/cli` 这个**包**无条件 import `gui`（`web_cli.go:15`、`daemon_cli.go:14`、`gui_on.go:9`），所以二进制仍链接它。

### 2.2 `internal/gui` 的实际构成（本轮实测）

| 类别 | 文件数 | 说明 |
|---|---:|---|
| 可 Web 化（无 `!nogui`、不 import Wails） | **136** | 含 `api.go`、`web.go`、`usage.go`、`providers.go`、`caller_keys.go`… |
| 桌面专用（`!nogui` 或 import Wails） | **66** | `app.go`、`mainwindow.go`、`panel*`、`tray*`、`dock*`、`glide_*`… |
| 仅 `nogui`（2 个 `_other` 桩） | 2 | `fontdpi_other.go`、`filedrag_other.go` |

136 个可 Web 化文件中，**26 个** import `agent`/`library`/`sessions`：

```
agent_models.go api.go backend.go caller_keys.go context.go gatewaymode.go gatewaypause.go
groups.go https.go library.go memory.go page_memory.go plugins.go port.go providers.go
quitask.go restoreonquit.go sessions.go sessions_codex.go sessions_manage.go sessions_reads.go
statewrite.go trace.go tune.go turn_done.go usage.go
```

### 2.3 路由分布（381 条生产注册）

按"该文件是否 import 产品面包"分类：

| 类别 | 文件数 | 路由数 | 典型文件 |
|---|---:|---:|---|
| 核心（无 agent/library/sessions） | **36** | **101** | `masking.go`(9)、`memory_distill.go`(7)、`pluginpage.go`(6)、`backup.go`(6)、`admin.go`(5)、`update.go`(5)、`modelprice.go`(2)、`speeds.go`(2)… |
| 耦合 | **18** | **280** | `api.go`(102)、`library.go`(39)、`providers.go`(35)、`usage.go`(22)、`plugins.go`(17)、`sessions.go`(11)、`sessions_manage.go`(10)、`memory.go`(8)、`caller_keys.go`(7)、`trace.go`(6)、`tune.go`(6)、`https.go`(5)、`context.go`(3)、`agent_models.go`(3)、`sessions_reads.go`(2)、`groups.go`(2)、`gatewaymode.go`(1)、`gatewaypause.go`(1) |

`api.go` 内的 102 条按前缀：`settings` 66、`agents` 12、其余 24（`state`/`set`/`profile`/`sync`/`quit`/`cli`…）。
66 条 settings 中，按名称可粗分为**约 32 条桌面/Agent 专属**（`codex-*`、`agent-*`、`desktop-*`、`panel-shortcut`…）与**约 32 条中转站相关**（`allowips`、`lan`、`port`、`cors`、`redact-*`、`search-*`、`github-token`、`bark`…）。

### 2.4 认证与授权现状

| 面 | 机制 | 源码 |
|---|---|---|
| Web 页面 | 单一 `MAGPIE_WEB_KEY` → `HttpOnly; SameSite=Lax` cookie；`?k=` 换取 cookie；`POST /_magpie/sign-in` 粘贴 key | `internal/gui/web.go:260 webGuard`、`:409 webServed`、`:429 webHandler`、`:352 webSignInPath` |
| 写请求 CSRF | `net/http.CrossOriginProtection`（`Sec-Fetch-Site`/`Origin`） | `internal/gui/web.go:452 webOwnPage` |
| Gateway 数据面 | `access.Key`（Bearer），`Check` → `access.Identity` | `internal/access/issue.go:245 Check`、`access.go:265 Identity` |
| 管理 API | `/v1/magpie/admin/` 需 `Key.Admin`（`local`/`any`），拒绝带 `Origin`/`Sec-Fetch-*` 的浏览器请求 | `internal/gateway/admin.go:47 admin`、`internal/gui/admin.go:39 adminRoutes` |
| 自助页 | `GET /me` 显示某 key 自己的额度、用量、模型 | `internal/gateway/key_page.go:20 KeyPagePath` |

**不存在**：用户表、密码登录、会话管理、角色、租户、审计日志。`Admin Key` 只是某个共享密钥上的能力位，**不等价于用户体系**。

### 2.5 持久化现状

| 数据 | 位置 | 格式 | 源码 |
|---|---|---|---|
| 设置 | `~/.config/magpie/settings.json`（0600） | JSON，232 个字段 | `internal/settings/settings.go:65`、`:1989 Load`、`:2287 Save` |
| 渠道/凭证 | `providers.json`（0600） | `{providers:[], groups:[], searches:[]}` | `internal/provider/provider.go:769 type file`、`:801 Path` |
| 订阅账号 | `logins.json`（0600） | JSON | `internal/provider/logins.go:208` |
| 网关令牌 | `caller-keys.json`（0600） | `[]access.Key` | `internal/access/access.go:131 Key`、`:360 Path` |
| 用量账本 | `usage.jsonl` | 每行一条 `usage.Record`，追加写 | `internal/usage/usage.go:30 Record`、`:322 Path`、`:332 Append` |
| 会话索引 | `~/.cache/magpie/said-v1.db` | SQLite FTS5 | `internal/sessions/said_index.go:116` |
| 插件 | `plugins.yml` | YAML | `internal/pluginrt/config.go` |

**控制面没有关系型数据库**。SQLite 只用于会话搜索索引，以及读取外部 Agent App 自己的数据库（`internal/agent/newmax.go`、`internal/provider/importapps.go`、`internal/sessions/*`）。

---

## 3. 目标架构

### 3.1 模块图

```
                        ┌──────────────────────────────┐
  浏览器 / API 客户端 →  │  internal/relay (新增)        │
                        │  HTTP 服务、生命周期、TLS      │
                        └──────────┬───────────────────┘
                                   │
                        ┌──────────▼───────────────────┐
                        │  internal/webapi (新增)       │
                        │  Host 接口 / 路由注册 / JSON   │
                        │  auth(会话/RBAC) / assets     │
                        ├──────────────────────────────┤
                        │ modules:                     │
                        │  system · channels · models  │
                        │  routing · tokens · users    │
                        │  usage · plugins · settings  │
                        └──────────┬───────────────────┘
                                   │ 依赖（只向下）
   ┌───────────────┬───────────────┼───────────────┬──────────────┐
   ▼               ▼               ▼               ▼              ▼
gateway        provider        access           usage        pluginrt
(数据面)        (渠道/凭证)      (令牌/限额)       (账本/计费)   (插件)
   │
   ▼
sessions(仅会话存储子集)  ← 当前硬依赖，后续剥离

   internal/gui (桌面外壳，保留但不被 relay 链接)
     ├─ app.go / mainwindow.go / panel* / tray* / dock*   Wails + GTK/WebKit
     └─ deskapi (新增)：agents · sessions · library · tune · memory …
```

依赖规则（由架构测试强制）：

```
relay        → webapi, gateway
webapi       → gateway, provider, access, usage, settings, catalog, pluginrt, middleware
webapi       -/-> gui, agent, library, sessions
gui          → webapi, agent, library, sessions, Wails
gateway      → sessions（现状；Phase 6 收窄）
```

### 3.2 构建形态

| 构建 | 命令 | 包含 | 不包含 |
|---|---|---|---|
| 桌面 | `go build -tags "production"` | 全量 | — |
| Web（现状，过渡） | `CGO_ENABLED=0 go build -tags nogui` | webapi + gui 的 Web 部分 + agent/library/sessions | Wails/GTK |
| 中转站（目标） | `go build -tags "relay"` | relay + webapi + gateway + provider + access + usage + 插件子集 | gui、agent、library |

`relay` 构建的关键约束：`main.go:11` 的 `plugins/all` 会经 `plugins/library → internal/library → internal/agent` 把 agent 拉回来，因此需要 `plugins/all` 的 relay 变体（见 §6.4）。

---

## 4. 控制平面解耦设计

### 4.1 现状边界

- 唯一入口：`gui.Handler(w Windows, gw *gateway.Server)`，`internal/gui/api.go:1202`。
- 唯一"桌面抽象"：`Windows` 接口，`internal/gui/api.go:156`，11 个方法：

```
HidePanel · ShowMain · Quit · OpenURL · OpenFolder · ChooseFolder
Copy · FitPanel · TintPanel · TintTitleBar · SetTextSize
```

- 生产代码中这些方法的调用点只有 11 个文件、30 处：
  `OpenFolder` 13、`ChooseFolder` 5、`OpenURL` 3、`Quit` 2、`Copy` 2，其余各 1。
- 两个实现：`*host`（Wails，`app.go:76`）与 `webHost`（浏览器，`web.go:36`，其中桌面动作全是空实现或返回错误）。

**结论：桌面抽象已经存在且足够窄，解耦不需要发明新机制，只需要移动包边界。**

### 4.2 目标模块

新增 `internal/webapi`，把 136 个可 Web 化文件逐步迁入，按域拆子包：

```
internal/webapi/
  host.go          Host（= 原 Windows，改名后 gui 实现它）
  deps.go          Deps{Host, Gateway, Facades…}  // 显式注入
  server.go        Mux 组装、静态资源、错误/JSON 助手
  auth/            会话、用户、RBAC（Phase 4）
  system/          /api/state · /api/settings/* · /api/quit · /api/cli · /api/update
  channels/        /api/providers · /api/provider/* · /api/keys/* · /api/login/*
  models/          /api/model-price · /api/providers 的模型视图 · 模型映射
  routing/         /api/groups · /api/set 的路由部分
  tokens/          /api/caller-keys · /api/account-groups · /gateway key 自助页
  usage/           /api/usage/* · /api/trace/*
  plugins/         /api/plugins/* · /api/plugin-pages
  settings/        /api/settings/*（中转站子集）
  admin/           /v1/magpie/admin/* 的宿主侧
  assets/          index.html / app.js / app.css / plugins/*
```

桌面侧保留：

```
internal/gui/
  app.go mainwindow.go panel* tray* dock* glide_*   (Wails)
  deskapi/  agents · sessions · library · tune · memory · agent_models · https · gatewaypause
```

`deskapi` 通过 `webapi.Deps` 的扩展点注册路由，relay 构建里为空。

### 4.3 Facade 接口（解耦 280 条耦合路由的关键）

现状中 26 个可 Web 化文件直接调用产品面包级函数。把它们收敛为三个窄接口，由 `gui`/`deskapi` 实现，relay 提供空实现：

```go
// internal/webapi/facades.go
type AgentFacade interface {
    Looks() []AgentLook                       // usage.go:45
    All() []*agent.Agent                      // plugins.go
    DetectedIn([]*agent.Agent) []*agent.Agent // providers.go:1488
    FindIn(string) (*agent.Agent, error)      // providers.go:619
    Reseat(func() error) ([]agent.Move, error)// providers.go:1807, groups.go:596
    ConnectsAgents() bool                     // providers.go:1559
    OpenAppLink(string) error                 // providers.go:1674
    SyncCatalogSoon()
    PluginErrors()
}

type SessionFacade interface {
    Content, ContentOf, FindCall, Part        // usage.go:76-93
    List(days int) []sessions.Session         // trace.go:213, context.go:553
    Titles, CodexTitles, GatewayTitler        // trace.go:98,396,428
}

type LibraryFacade interface {
    KeyMCPServers(...)                        // caller_keys.go
    PluginServers(), PluginServer(...)        // plugins.go
    BorrowedOn/BorrowNote/SetBorrow/…         // providers.go
    MemoryInInstructions/MemorySkill/…        // memory.go
}
```

**relay 的空实现**返回空列表/`false`/`nil`，页面不显示相应列，路由不注册。这样同一套 handler 代码同时服务桌面与中转站，不需要分叉。

### 4.4 生命周期与网关管理接口

现状（`internal/gui/backend.go`）：

- `startBackend`（:48）：`gateway.Window=true` → `serveGateway()` → `watchGateway` → `usage.ReadAhead` → Codex/catalog 维护。
- `serveGatewayLocked`（:145）：端口被占则等待，另一个 magpie 服务时待命。
- `stopServing`（:31）：取消 context，等待在途请求。
- 管理接口：`POST /api/gateway/take-over`、`/api/gateway/restart`（`gateway_takeover.go:51`），暂停 `gatewayPauseRoutes`（`gatewaypause.go:227`）。

迁移方案：

1. 新增 `internal/gatewayhost`，只保留**与网关有关**的部分：
   `Serve() *gateway.Server`、`Watch(ctx)`、`Stop()`、`Restart()`、`TakeOver()`、`Paused()/Pause()/Resume()`。
2. `backend.go` 中与 agent 有关的三段（`agent.KeepCodexDaemonCurrent`、`agent.KeepCodexRolloutsPortable`、`agent.SyncCatalog`）改为 `gatewayhost.Extras` 中的可选钩子，由 `gui`/`cli` 注入；relay 不注入。
3. `webapi` 只依赖 `gatewayhost` 的接口，不依赖 `gui`。

### 4.5 GUI 与 Web 分别构建/运行

| 目标 | 现状 | 目标 |
|---|---|---|
| 分别构建 | `-tags nogui` 已可，但仍链接 agent/library/sessions | `-tags relay`，`go list -deps` 断言不含 `gui`/`agent`/`library` |
| 分别运行 | 已可：`serve`、`serve --web`、`web`、桌面 app | 保持；新增 `magpie relay` 命令作为中转站入口 |
| 共享代码 | 同一包 | 同一 `webapi`，不同 facade 实现 |

---

## 5. 中转站核心能力审计

状态口径：**已有** = 源码已实现且有进入点；**部分实现** = 有基础但缺关键环节；**完全缺失** = 无代码；**未验证** = 本轮未实际运行。

### 5.1 渠道 / Provider / 凭证

| 能力 | 状态 | 现有实现 | 缺口 / 改造 |
|---|---|---|---|
| 渠道 CRUD | 已有 | `provider.Provider`（`provider.go:51`）、`Provider.Save`（`:1270`）、`/api/provider/{action}`（`providers.go`） | 无 |
| 多凭证（key 轮换） | 已有 | `KeyAccount`（`keys.go:24`）、`KeyName/Keys/KeyWeight`（`provider.go:83-90`） | 无 |
| 订阅账号 | 已有 | `provider.Account`（`account.go:45`）、`logins.json` | 无 |
| 渠道排序/权重 | 已有 | `Provider.Order`（`provider.go:786`）、`AccountWeights`（`:238`） | Web 端缺独立 UI |
| 渠道健康/测试 | 已有 | `/api/provider/{action}` 的 `test`、`balance`、`models` | 无 |
| 渠道级中间件 | 已有 | `model-map`、`param-override`（README:237-238） | 无 |
| 渠道级 RPM | 已有 | `Provider.MaxRPM` → `RPMLimit()`（`provider/concurrency.go:269`） | 无 |
| **渠道/模型映射表 UI** | **部分实现** | `ModelWires`（`settings.go:1231`）、`Key.Aliases`（`access.go:152`） | 无可视化编辑页 |
| **渠道分组** | **部分实现** | `provider.Group`（`group.go:197`）是模型路由组，不是渠道分组 | 需新增渠道分组概念 |

数据结构（现状）：

```go
// internal/provider/provider.go:51
type Provider struct {
    ID, Name, Icon, Preset, Key string
    KeyName string; Keys []KeyAccount; KeyWeight int
    Chat, Responses, Anthropic, Gemini string   // base URLs
    Routing, Affinity, Sink string
    MaxRPM, MaxConcurrency int
    Order []string
    // …
}
```

### 5.2 模型管理 / 映射 / 定价

| 能力 | 状态 | 现有实现 | 缺口 |
|---|---|---|---|
| 模型目录 | 已有 | `internal/catalog`（models.dev + 内置回退）、`provider.Catalog()` | 无 |
| 模型列表（对客户端） | 已有 | `GET /v1/models` → `listModels`（`gateway.go:1827`） | 无 |
| 模型价格（用户设定） | 已有 | `settings.ModelPrices`（`:1209`）、`/api/model-price`（`modelprice.go:53,61`） | **无独立页面**，现在在 Usage 行右键 |
| 价格优先级 | 已有 | `provider/models.go:2178 userPrice` 精确→`provider/*`→`*/model`→厂商价 | 无 |
| 分级/峰谷价 | 已有 | `catalog.Price{Tiers, Peak}`（`catalog.go:269`） | 无 |
| 上游模型名改写 | 已有 | `ModelWires`（`settings.go:1231`）、`provider.UpstreamName` | 无可视化 |
| 客户端别名 | 已有 | `access.Key.Aliases`（`access.go:152`） | 仅按 key，无全局别名 |
| 模型禁用 | 已有 | `settings.DisabledModels`（`/api/settings/disabled-models`） | 无 |
| **表达式定价** | **完全缺失** | — | New API 有，需新增 |
| **独立 Models 页面** | **完全缺失** | — | 需新增 |

### 5.3 路由组 / 策略 / 故障切换

| 能力 | 状态 | 现有实现 |
|---|---|---|
| 路由组 | 已有 | `provider.Group`（`group.go:197`）：`Members`/`Match`/`Placed`/`Routing`/`Affinity`/`Sink` |
| 路由策略 | 已有 | `internal/routes.Strategy`（`routes.go:123`）；smart/order/rotate/usage/pace/weight/free/fast/cheap/drain/expiring |
| 故障切换 | 已有 | `gateway/fallback.go:51 type candidate`、`:286 perKey`、`:2048 retryable` |
| 会话亲和 | 已有 | `gateway/affinity.go:104 Affinity`、`:294 sticksPath`、`gateway.go:10477 SessionHeader` |
| 限流退避 | 已有 | `gateway/rest_probe.go`、`routing.go:1491 restAfter` |
| 组模板 | 已有 | `plugins/routing/client.js` 的 6 个模板（Hard to strong…） |
| **命名策略库（可复用、跨组）** | **部分实现** | 策略是字符串，绑定在 provider/group 上，无独立策略对象 |
| **策略可视化编辑器** | **部分实现** | Routing 页有实时图与组编辑，无策略 DSL |

### 5.4 令牌 / 额度 / 限额

| 能力 | 状态 | 现有实现 |
|---|---|---|
| 令牌 CRUD | 已有 | `access.Key`（`access.go:131`）、`access.Update`（`:539`）、`/api/caller-keys`（`caller_keys.go:70`） |
| 额度（时间窗） | 已有 | `access.Limit`（`limit.go:26`）：day/week/month/days/hours/account |
| 成本/Token 上限 | 已有 | `Limit.Tokens`/`Cost`/`CacheReads`（`limit.go:32-34`） |
| 模型/账号/MCP 作用域 | 已有 | `Key.Models`（`:147`）/`Accounts`（`:156`）/`MCP`（`:160`） |
| 并发上限 | 已有 | `Key.MaxConcurrency`（`:208`） |
| 只读/浏览器/Admin | 已有 | `ReadOnly`/`Browser`/`Admin`（`:171-186`） |
| 短期签发 | 已有 | `access.IssueKey`、`POST /v1/magpie/keys`、OIDC 兑换 |
| 自助页 | 已有 | `GET /me`（`key_page.go:20`） |
| 预算执行 | 已有（插件） | `plugins/budget` → `gwbudget.Limit`；`budget.Limiter` 从 `usage.jsonl` 汇总 |
| **按用户额度** | **完全缺失** | 限额只挂在 key 上 |
| **按 key 的 RPM** | **部分实现** | 只有 `MaxConcurrency`；RPM 在 provider 级 |
| **令牌归属用户** | **完全缺失** | `Key` 无 owner 字段 |

### 5.5 用量 / 请求日志 / 费用

| 能力 | 状态 | 现有实现 |
|---|---|---|
| 调用账本 | 已有 | `usage.Record`（`usage.go:30`）、`Append`（`:332`）、`usage.jsonl` |
| 汇总维度 | 已有 | `Summary`（`:1119`）：Agents/Models/ProviderKeys/Accounts/CallerKeys/Sessions |
| 费用计算 | 已有 | `Record.CostAt`（`:560`）→ `catalog.Price.CostSplit` |
| 请求日志页 | 已有 | `/api/usage/requests`、`/api/usage/traffic`、`/api/usage/heatmap` |
| 配额历史 | 已有 | `/api/usage/quotas/history` |
| OTLP 导出 | 已有 | `plugins/otel` → `internal/otlp` |
| **按用户维度** | **完全缺失** | `Record` 无 `User` 字段 |
| **审计日志** | **完全缺失** | 只有请求账本，没有管理操作审计 |
| **Dashboard** | **完全缺失** | 无聚合首页 |

### 5.6 插件 / Middleware

| 能力 | 状态 | 现有实现 |
|---|---|---|
| 插件运行时 | 已有 | `pluginrt.Manifest`（`pluginrt.go:35`）、`Reconcile` 生命周期、`plugins.yml` |
| 内置插件 | 已有 | 14 个：budget/gateway/groups/library/middleware/otel/redact/rotation/routing/search/sessions/tools/usage/warmups |
| JS 中间件 | 已有 | `internal/middleware`：`onRequest/onAccount/onProvider/onEvent/onResponse`，moejs 进程内执行 |
| 扩展点 | 已有 | 路由策略、观察者、预算、工具、中间件（`docs/subsystems/plugin-runtime.md` P3） |
| 社区插件页 | 已有 | `plugin.PageSpec`、`pluginpages.js`、`Replaceable` |
| **插件权限/隔离** | **部分实现** | 插件以 Go 包形式编译进主程序；社区插件是 JS 页面，但可访问 `/api` |
| **插件审计** | **完全缺失** | — |

---

## 6. 用户与权限体系设计

### 6.1 原则

1. **不假设 Admin Key 等于用户**。`Admin` 是共享密钥上的能力位（`access.go:186`），没有身份、没有密码、没有会话、不能审计到人。
2. 用户是新的第一类实体；API Key 归用户所有；用量按用户归集。
3. 兼容存量：现有 key 无 owner、现有 usage 无 user，都按"遗留/管理员"归集，**不重写历史账本**。
4. 不引入关系型数据库也能落地；如需扩展再评估 SQLite（`internal/sqlitedb` 已是依赖）。

### 6.2 数据结构

```go
// internal/identity/identity.go (新增)
type User struct {
    ID           string    `json:"id"`
    Name         string    `json:"name"`
    Email        string    `json:"email,omitempty"`
    Role         string    `json:"role"`        // owner | admin | user
    Groups       []string  `json:"groups,omitempty"`
    Off          bool      `json:"off,omitempty"`
    PasswordHash string    `json:"passwordHash,omitempty"` // argon2id
    Notes        string    `json:"notes,omitempty"`
    CreatedAt    time.Time `json:"createdAt"`
}

type Group struct {
    ID     string         `json:"id"`
    Name   string         `json:"name"`
    Models []string       `json:"models,omitempty"`   // "<provider>/<model>"、"<provider>/*"
    Limits []access.Limit `json:"limits,omitempty"`
    Off    bool           `json:"off,omitempty"`
}

type Session struct {
    ID        string    `json:"id"`        // 随机 32 字节，cookie 里只放它
    UserID    string    `json:"userId"`
    CreatedAt time.Time `json:"createdAt"`
    LastSeen  time.Time `json:"lastSeen"`
    Expires   time.Time `json:"expires"`
}
```

存储：`identity.json`（0600，`{version:1, users:[], groups:[], sessions:[]}`），写入沿用 `filelock.Mutex` + `lastgood.Keep(..., 0o600)` + `steady` 的模式（与 `access`、`provider` 一致）。

对现有结构的增量（**全部是可选字段，向后兼容**）：

```go
// internal/access/access.go:131 Key +=
Owner string `json:"owner,omitempty"`   // user id；"" = 遗留/管理员

// internal/usage/usage.go:30 Record +=
User string `json:"user,omitempty"`     // user id；"" = 按 CallerKeyID 归属
```

`json.Unmarshal` 会忽略未知字段，旧版本读新文件不会坏；新版本读旧文件得到空 owner。

### 6.3 权限矩阵

| 动作 | owner | admin | user |
|---|---|---|---|
| 渠道 CRUD / 凭证 | ✔ | ✔ | ✖ |
| 查看渠道列表/模型 | ✔ | ✔ | ✔（只读） |
| 模型价格/映射 | ✔ | ✔ | ✖ |
| 路由组/策略 | ✔ | ✔ | ✖ |
| 令牌：管理自己的 | ✔ | ✔ | ✔ |
| 令牌：管理他人的 | ✔ | ✔ | ✖ |
| 用户/组管理 | ✔ | ✔（不可提权） | ✖ |
| 用量：全部 | ✔ | ✔ | ✖ |
| 用量：自己的 | ✔ | ✔ | ✔ |
| 系统设置 | ✔ | ✔（安全项除外） | ✖ |
| 审计日志 | ✔ | ✔ | ✖ |

`owner` 是首个引导用户，不可删除；`MAGPIE_WEB_KEY` 保留为 break-glass（等价 owner），用于首次登录/找回。

### 6.4 认证流程

```
浏览器 → GET /login
       → POST /api/auth/login {name,password}  （argon2id 校验）
       → Set-Cookie: magpie_session=<32B>; HttpOnly; SameSite=Lax; Secure(TLS)
       → 服务端 identity.json 里的 Session 记录（可撤销）

API 客户端 → Authorization: Bearer <gateway key>
          → access.Check → Identity{KeyID, Owner}
          → usage.Record.User = Key.Owner
```

- 保留 `?k=MAGPIE_WEB_KEY` 与 `/_magpie/sign-in` 作为 break-glass（映射到 owner 会话）。
- CSRF 继续用 `CrossOriginProtection`。
- 登录限速：内存令牌桶（每 IP 每分钟 N 次），失败计数。

### 6.5 历史数据兼容与迁移

| 数据 | 迁移策略 | 风险 | 回滚 |
|---|---|---|---|
| `caller-keys.json` | 读取时 `Owner==""` → 视为 `legacy` 用户；首次保存时才写入字段 | 旧版本读到新字段会忽略，安全 | 删除 `identity.json`，key 文件不变 |
| `usage.jsonl` | **不重写**。读取时 `User==""` → 用 `CallerKeyID` 反查当前 key 的 owner；查不到归 `legacy` | 归属会随 key 删除而变化；需在 UI 标注"历史归属" | 无（只读逻辑） |
| `settings.json` | 不改；新增 `auth` 段由 `identity.json` 承载 | 无 | 无 |
| 新用户/组 | 首次启动创建 `owner` 用户（密码由 break-glass 登录后设置） | 若 `MAGPIE_WEB_KEY` 未设置，需一次性生成并打印 | 删除 `identity.json` |
| 预算 | 用户额度 = 其所有 key 的额度之和；`access.Identity` 增加 `Owner`，`budget.reserve` 同时结算 owner 维度 | 在途请求可能双计；需与 issuer 逻辑同样加锁 | 关闭用户额度开关 |

**没有 SQL schema 迁移**：控制面是文件。若 Phase 4 选择 SQLite 存 users/sessions，则新建 `relay.db`，用 `schema_migrations(version, applied_at)` 表；与现有文件**并存**，不迁移历史账本。

### 6.6 与现有 Admin Key 的关系（明确）

| 概念 | 现状 | 目标 |
|---|---|---|
| `access.Key.Admin` | 某个 key 可调 `/v1/magpie/admin/` | 保留，作为**机器自动化**凭据；映射到某个 owner 用户，审计记该 key |
| 用户 | 不存在 | 新增，用于**人**的登录与归属 |
| 两者关系 | — | 一个用户可以拥有多个 key；一个 key 必须属于一个用户（遗留为空） |

---

## 7. 前端信息架构

### 7.1 现有导航（`index.html:74-87`，`#prefs` 在 `:113`）

10 个 `data-view` 按钮：agents、sessions、library、providers、routing、groups、gateway、usage、masking、plugins；Settings 为 `#prefs`。
11 个视图：4 个静态（agents/providers/plugins/settings），7 个由插件脚本创建（sessions/library/routing/groups/gateway/usage/masking）。

隐藏机制：`data-feature` + `builtinsOff`（`builtins.go:124`）→ `boot["featuresOff"]`（`api.go:1294`）→ `app.js:51-65`；gateway mode `GATEWAY_HIDES=["agents"]`（`app.js:45`）。

### 7.2 目标导航与页面处置

| 目标页 | 现有页 | 处置 | 说明 |
|---|---|---|---|
| **Dashboard** | 无 | **新增** | 聚合：gateway 状态、今日调用/成本、渠道健康、错误率、top 模型 |
| **渠道管理** | Providers | **复用 + 重构** | 改名；拆出 Tokens；保留凭证/账号/排序/权重 |
| **模型管理** | 无（散在 Providers + Settings›Models + Usage 右键） | **新增** | 模型目录、价格表、映射（ModelWires/Aliases）、禁用 |
| **路由管理** | Routing + Routing groups | **复用** | 保留实时图与组编辑；新增策略对象 |
| **令牌管理** | Gateway 页的 keys.js | **拆分** | 独立页；增加归属用户、RPM |
| **用户管理** | 无 | **新增** | 用户/组/角色/额度 |
| **数据统计** | Usage | **复用** | 去掉 Sessions 标签；增加按用户维度、审计 |
| **插件管理** | Plugins + Middleware | **复用** | relay 下过滤 library/sessions 插件 |
| **系统设置** | Settings 9 个子页 | **复用 + 裁剪** | 去掉托盘/Dock/Agent 行；保留 Network/Models/Privacy/OTEL/Sync/About |
| **Agents** | Agents | **移除导航** | relay 不注册 |
| **Sessions** | Sessions | **移除导航** | 保留 gateway 会话亲和机制 |
| **Library** | Library | **移除导航** | 保留 MCP hub 服务端能力 |
| **Masking** | Masking | **降级** | 并入 Settings›Privacy 或 Plugins |

### 7.3 实施方式

- 新页面用现有插件页机制（`pluginrt.Manifest{ Parts: []Part{{Slot:"nav.page", Key:"dashboard"}} }`），不改内核。
- 导航分组重写只改 `index.html` 的 `nav` 段与 `app.js` 的 `show()`/`pageName()`。
- 删除入口优先用 `plugins.yml` 的 `enabled:false`，不改代码；确认后再删源。

---

## 8. 桌面能力处理

分三级，**不得跳级**：L1 产品入口隐藏 → L2 依赖解除 → L3 源码删除。L3 必须在 L2 之后至少一个发布周期。

| 功能 | L1 入口隐藏 | L2 依赖解除 | L3 源码删除 | 依赖风险 | 验证要求 |
|---|---|---|---|---|---|
| macOS 菜单栏 | relay 构建不链接 gui（已由 `!nogui` 保证）；网关模式不显示 | 无（仅 gui 内部） | `app.go:491-499` 的 `DefaultApplicationMenu` | 无 | `go list -deps -tags relay` 无 wails |
| 系统托盘 | 同上 | 无 | `app.go:637`、`tray*` 无 tag 文件 | `traymenu.go`/`trayusage.go` 被 Web 复用（`trayLang`/`withTrayKinds`），**不能整文件删** | 先迁 `trayLang`/`trayText`/`withTrayKinds` |
| 桌面通知 | `quota_alert_notify.go` 是 `!nogui`，relay 天然没有 | 无 | `quota_alert_notify.go`、`mask_alert.go`、`signin_alert.go`、`turn_done.go` | `quota_alert.go:17,23` 被 `api.go:913,1768` 读取；`bark_daily.go` 被 `/api/bark/test` 使用 | 保留 `quota_alert.go` 与 `bark_*`；Web 侧补通知中心 |
| 本地 Agent 管理 | relay 不注册 `/api/agents/*`、不显示 Agents 页 | `internal/agent` 移出 relay 依赖闭包（需 agent facade + `plugins/all` relay 变体） | `internal/agent`（201 文件） | `usage.AgentOf`、`gateway.StandIn` 等 8 个钩子会退化；`provider` 不依赖它 | `go list -deps -tags relay` 断言；用量按 UA 回退 |
| Sessions 页面 | 移除导航 | 拆出只读 `sessionsstore`（`gateway.go`/`content.go`/`access.go`/`codex_*`）供 gateway | `internal/sessions` 其余（357 文件） | **gateway 硬依赖**：`conversations.go:14`、`hub_mcp.go:17`、`qoder_session.go:17`、`sessions_mcp.go:10`、`title_link.go:11` + `usage` 19 文件 | 先做只读子集，`go list -deps ./internal/gateway` 验证 |
| Library 页面 | 移除导航 | relay 不链接 `plugins/library` | `internal/library`（115 文件） | gateway 通过**可空钩子**`MCPHub`/`RunMCP`/`SharedLibrary` 接入，nil 时 404 | 注册 `plugins/allrelay`，不含 library/sessions |
| Skills / MCP / Subagents | 随 Library 页移除 | 同上 | 随 `internal/library` | MCP hub 的**类型定义在 gateway**（`hub_mcp.go:53-250`），可保留 | broker 测试：nil 钩子时 `/mcp/run` 404 |
| Dock / Panel / 快捷键 | relay 不链接 | 无 | `dock*.go`、`panel*.go`、`mainwindow.go` | `panelkey.go` 的 `panelShortcut` 被 `/api/settings` 使用 | 先迁纯函数 |
| 文件选择器 / 剪贴板 | `webHost` 已返回错误/`false` | 无 | 桌面实现 | 无 | 已有 |

**必须保留的桌面能力（转为 Web）**：

| 能力 | 现状 | Web 替代 |
|---|---|---|
| 服务状态/启停 | Gateway 页 Restart、`POST /api/gateway/restart` | 保留，补 Start/Stop/Drain/uptime |
| 异常告警 | 系统通知（relay 下完全没有） | **新增通知中心** + Bark（已有 `internal/bark`）+ Webhook |
| 服务化 | `internal/daemon`（systemd/launchd） | 保留；relay 增加容器健康检查 |
| 升级 | `serve_follow.go`、`update.ReplaceBinary` | 保留；容器用镜像替换 |

---

## 9. 测试缺陷：最小修复设计

### 9.1 复现与根因（本轮重新核实，非版本差异）

**现象**：

```
go test -tags nogui -run '^TestGatewayModeStatsNameTheGatewaySessions$|^TestGatewayModeListsNoOwnSessions$' ./internal/gui/
fatal error: reset of synctest timer from outside bubble
  time.(*Timer).Reset → internal/usage.lingerScratch (request_page.go:2024)
  ← chunkRows.release (request_page.go:720) ← keepRows (:663)
  ← countGateway (gateway_sessions.go:457) ← listedExternal (gui/sessions.go:96)
  ← TestGatewayModeListsNoOwnSessions
```

- 单独运行两个测试：**都 PASS**。
- 同一包内按序运行：**3/3 次 FAIL**。
- 非 synctest 的 `TestGatewayMode` + `TestGatewayModeListsNoOwnSessions`：**PASS**（证明只与 synctest 气泡有关）。

**根因（已隔离验证）**：

1. `internal/usage/request_page.go:2005` 的 `scratchSweep.t` 是**包级** `*time.Timer`。
2. `TestGatewayModeStatsNameTheGatewaySessions` 在 `synctest.Test` 气泡内调用 usage 路径 → `lingerScratch` 首次执行 → `time.AfterFunc` **在气泡的假时钟世界内**创建该 timer。
3. 气泡结束，timer 归属的世界已销毁。
4. 下一个测试（普通测试）调用 `lingerScratch` → 对**另一个世界的 timer** 执行 `Reset` → Go 运行时 `fatal`（不可 recover）。

**独立复现工程**（`/root/magpie-obs/repro`，15 行）：

| 场景 | Go 1.27.2 | Go 1.26.8（go.mod 声明版本） |
|---|---|---|
| 气泡内创建 → 气泡外 `Reset` | **FAIL（同 fatal）** | **FAIL（同 fatal）** |
| 气泡内创建 → 气泡外 `Stop` | **FAIL**（`stop of synctest timer from outside bubble`） | 未单独跑 |
| 真实 timer → 气泡内 `Reset` | **PASS** | 未单独跑 |
| 气泡内创建后遗弃、不 Reset/Stop | **PASS**（无害） | 未单独跑 |

**结论：与 Go 版本无关**。上一份报告"Go 版本差异"的推测被否定。

### 9.2 候选修复

| 方案 | 做法 | 优点 | 缺点 | 判定 |
|---|---|---|---|---|
| A 每次新建 + 世代号 | 不保存 timer，每次 `AfterFunc`，回调检查世代 | 绝对安全 | 每次 linger 一个 timer，高频下浪费 | 可用，非最优 |
| B 禁用测试中的 linger | `if testing.Testing() { return }`（仓库已有先例 `sessions.go:846`） | 1 行 | 测试不再覆盖 linger 逻辑；掩盖问题 | 仅作兜底 |
| **E 单 armed + 到期重算 + 陈旧检测** | 见下 | 保留语义、1 个 timer、跨世界安全 | 需 4 个字段 + 1 个 `ReleaseScratch` 钩子 | **推荐** |

### 9.3 推荐设计 E

```go
// internal/usage/request_page.go（替换 :2005-2025）
var scratchSweep struct {
    sync.Mutex
    armed bool
    at    time.Time   // 本次 armed 的时刻
    last  time.Time   // 最后一次有工作的时刻
}

func scratchArm(d time.Duration) {
    scratchSweep.at = time.Now()
    time.AfterFunc(d, scratchSweepFire)
}

func lingerScratch() {
    scratchSweep.Lock()
    scratchSweep.last = time.Now()
    // 另一个世界留下的 armed：at 远在过去，视为陈旧，重新 arm
    stale := scratchSweep.armed && time.Since(scratchSweep.at) > 2*scratchLinger
    if scratchSweep.armed && !stale {
        scratchSweep.Unlock()
        return
    }
    scratchSweep.armed = true
    scratchSweep.Unlock()
    scratchArm(scratchLinger)
}

func scratchSweepFire() {
    scratchSweep.Lock()
    last := scratchSweep.last
    scratchSweep.Unlock()
    if idle := time.Since(last); idle < scratchLinger {
        scratchSweep.Lock()
        scratchSweep.armed = true
        scratchSweep.Unlock()
        scratchArm(scratchLinger - idle)   // 精确补齐剩余时间
        return
    }
    scratchSweep.Lock()
    scratchSweep.armed = false
    scratchSweep.Unlock()
    keptMatch.Store(nil)
    keptPicks.Store(nil)
    keptSeen.Store(nil)
    releaseRows()
}

// ReleaseScratch 末尾追加：
func ReleaseScratch() {
    // …existing clears…
    scratchSweep.Lock()
    scratchSweep.armed = false
    scratchSweep.at, scratchSweep.last = time.Time{}, time.Time{}
    scratchSweep.Unlock()
}
```

**为什么安全**：全程不调用 `Reset`/`Stop`。目标是"最后一次工作后 30s 释放"，用 `last` 时间戳在回调里判断，语义与原来等价（误差为一个调度周期）。跨世界的 timer 废弃后不再被触碰。

**生命周期状态机**：

```
IDLE --linger--> ARMED(timer T1)
ARMED --linger--> ARMED （不新建、不 Reset，仅更新 last）
ARMED --T1 到期, idle>=30s--> IDLE （清理 scratch）
ARMED --T1 到期, idle<30s--> ARMED(timer T2, 时长=30s-idle)
ARMED --ReleaseScratch--> IDLE （armed=false；T1 若存在则自然到期后走清理，幂等）
ARMED(来自已结束的气泡) --linger, since(at)>60s--> 判定陈旧 → 重新 ARMED(timer T2)
```

**验证证据（独立复现工程，非仓库）**：

| 用例 | Go 1.27.2 | Go 1.26.8 |
|---|---|---|
| 气泡内 arm → 气泡外 linger（无 Release） | **PASS** | **PASS** |
| 气泡内 arm → Release → 气泡外 linger | **PASS** | **PASS** |
| 连续 5 次 linger 只 arm 一次 | **PASS** | **PASS** |

### 9.4 同类隐患（必须一并处理）

仓库里还有两个**包级 timer + `Reset`**，目前没有 synctest 测试触发，但结构相同：

| 位置 | 现状 | 建议 |
|---|---|---|
| `internal/gui/page_memory.go:101-113` `pageAway.t.Reset` | `awayPage()` 在 `page_memory_test.go:136,143` 被普通测试调用 | 同步改为设计 E，或至少不要在测试中 arm |
| `internal/gui/sessions.go:838-859` `statsIdleTimer` | 已有 `off = testing.Testing()` 兜底 | 保留；长期统一为同一实现 |

### 9.5 测试方案

1. **单元**（`internal/usage`）：
   - `TestLingerArmsOnce`、`TestLingerExtends`、`TestSweepAfterIdle`、`TestReleaseDisarms`
   - `TestLingerAfterBubble`：`synctest.Test` 内 linger → 气泡外 linger，断言不 fatal 且重新 armed
2. **回归**：`go test -tags nogui -run '^TestGatewayModeStatsNameTheGatewaySessions$|^TestGatewayModeListsNoOwnSessions$' ./internal/gui/` 必须 PASS。
3. **全包**：`go test -tags nogui ./internal/gui/`（本轮 **NOT TESTED**，因未改源码且耗时）。
4. **竞态**：`go test -race -tags nogui ./internal/usage/`。
5. **内存回归**：沿用 `request_page_test.go` 的 `ReleaseScratch` 断言。

---

## 10. API / 数据结构 / 迁移

### 10.1 保留的 API（不改语义）

`/api/state`、`/api/settings`、`/api/providers`、`/api/provider/{action}`、`/api/caller-keys`、`/api/usage/*`、`/api/model-price`、`/api/groups`、`/api/plugins/*`、`/api/gateway/restart`、`/v1/magpie/admin/*`。

### 10.2 新增 API

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/dashboard` | 聚合：`{gateway:{running,uptime,models}, today:{calls,cost,errors}, channels:[…], topModels:[…]}` |
| GET/POST | `/api/auth/login`、`/api/auth/logout`、`/api/auth/me` | 会话 |
| GET/POST/PUT/DELETE | `/api/users`、`/api/users/{id}` | 用户 CRUD |
| GET/POST/PUT/DELETE | `/api/user-groups`、`/api/user-groups/{id}` | 组 CRUD |
| GET/POST | `/api/models`、`/api/models/{provider}/{model}` | 模型注册视图 + 价格/映射/禁用 |
| POST | `/api/model-wires` | 上游名映射（复用 `settings.ModelWires`） |
| GET | `/api/audit` | 审计日志（分页） |
| GET | `/api/channels/{id}/health` | 渠道健康探测 |
| POST | `/api/gateway/start`、`/api/gateway/stop` | 服务启停 |

命名原则：不破坏现有 `/api/*`；新资源用复数名词；`/api/v2/*` 仅在结构不兼容时引入（本方案不需要）。

### 10.3 数据结构变更清单

| 文件 | 字段 | 可选 | 默认行为 |
|---|---|---|---|
| `access.Key` | `Owner string` | 是 | `""` → legacy 用户 |
| `usage.Record` | `User string` | 是 | `""` → 按 `CallerKeyID` 反查 |
| `identity.json`（新） | `{version, users, groups, sessions}` | — | 不存在 → 创建 owner |
| `settings.Settings` | 不改 | — | 安全设置放 identity |
| `provider.Provider` | 不改 | — | — |

### 10.4 迁移

```
启动时：
  identity.json 不存在？
    ├─ 是 → 读 MAGPIE_WEB_KEY（或生成）→ 建 owner 用户（无密码，用 break-glass 登录后设置）
    └─ 否 → 读 version
            ├─ version<1 → 升级（当前无旧版）
            └─ 正常
  caller-keys.json：读取时不改写 Owner；首次保存该 key 时写入
  usage.jsonl：只读兼容；新记录写 User
```

**不重写历史账本**（可能数 GB）；归属在读取时计算，并在 UI 标"历史归属"。

### 10.5 若选择 SQLite（可选，Phase 4 决策点）

- 用 `internal/sqlitedb`（已封装 modernc.org/sqlite 的环境问题）。
- 新库 `~/.config/magpie/relay.db`，表：`users`、`groups`、`sessions`、`audit`、`schema_migrations`。
- 与 JSON 并存：users/sessions 迁入 DB；providers/keys/usage 保持文件。
- 迁移：一次性导入 + `migrations` 表；失败回滚 = 恢复文件 + 删 db。
- 收益：并发写、审计查询、分页。成本：新依赖路径、备份/同步要覆盖 db（现有 `backup`/`davsync` 需扩展）。

---

## 11. 分阶段实施顺序与依赖

```
P0 基线护栏 ──▶ P1 webapi 缝 ──▶ P2 核心路由迁移 ──▶ P3 relay 构建
                                                        │
                                                        ├─▶ P4 用户/RBAC ──▶ P5 前端 IA
                                                        │                      │
                                                        └──────────────────────┴─▶ P6 桌面下线
```

### P0 · 基线与护栏（3–5 天）

- 修 §9 的 timer 缺陷（唯一允许的代码改动，属测试基础设施）。
- 新增 `internal/arch` 架构测试：
  - 路由清单快照（381 条，方法+路径）。
  - 依赖断言：`webapi` 不得 import `gui`/`agent`/`library`/`sessions`。
- 新增构建矩阵脚本：`desktop` / `nogui` / `relay` × `go build` + `go vet`。
- **验收**：清单基线提交；`go test -tags nogui ./internal/usage ./internal/gui -run 'Linger|GatewayMode'` PASS。
- **回滚**：纯新增测试 + 一个文件的小改。

### P1 · webapi 缝（5–8 天）

- 新建 `internal/webapi`：`Host`、`Deps`、`Server`、JSON/错误助手、assets 嵌入。
- `gui.Handler` 改为调用 `webapi.Handler`；`Windows` → `webapi.Host`（type alias 过渡）。
- 路由一条不迁，行为零变化。
- **验收**：路由快照 diff 为空；`go test -tags nogui ./internal/gui` 与基线一致（含已知 fatal，需 P0 已修）。
- **风险**：类型别名造成循环 import → 用 `type Host = webapi.Host` 单向。
- **回滚**：删除新包，`Handler` 还原。

### P2 · 核心路由迁移（10–15 天）

- 迁移 101 条核心路由到 `webapi/{system,settings,models,tokens,usage,plugins,admin}`。
- 为 `providers.go`/`usage.go`/`plugins.go`/`caller_keys.go`/`api.go` 引入 §4.3 facade。
- `deskapi` 承接 agents/sessions/library/tune/memory/trace/context/agent_models/https/gatewaypause 的 280 条。
- **验收**：`webapi` 不 import 三个包；桌面全量测试通过；relay 模式下 facade 为空，核心页可用。
- **回滚**：保留 `gui` 中的原文件一个版本周期，双注册开关。

### P3 · relay 构建（5–8 天）

- 新增 `internal/relay`（config/serve/health/drain/TLS）与 `magpie relay` 命令。
- `plugins/allrelay`（不含 `plugins/library`、`plugins/sessions`）。
- 构建标签 `relay`。
- **验收**：`go build -tags relay`；`go list -deps -tags relay` 不含 `gui`/`agent`/`library`；`hellcheck` 等价。
  `healthcheck`、`/v1/models`、`/api/state` 实跑。
- **风险**：`internal/cli` 的 `serve()` 直接调用 `agent.*`，需 `serve_extras_on/off.go`。
- **回滚**：删标签，保留 `serve --web`。

### P4 · 用户 / RBAC（15–20 天）

- `internal/identity` + `api/auth` + `Owner`/`User` 字段 + 预算 owner 维度。
- 迁移与 break-glass。
- **验收**：两用户隔离（各自的 key/用量）；历史用量仍可读；`access`/`budget` 单测通过；登录限速生效。
- **风险**：`usage.Record` 结构变更影响约 4539 个 gateway Test 函数（数量为 grep 计数，非执行结果）（只增可选字段，预期无损）。
- **回滚**：`identity` 关闭开关 → 退回单密钥模式。

### P5 · 前端 IA（15–20 天）

- 新导航、Dashboard、Models、Users、Tokens 拆页；移除 Agents/Sessions/Library；Settings 裁剪。
- **验收**：9 个目标页可达；无桌面专属控件；i18n 键完整。
- **风险**：`app.js` 1.34 MB 单文件，改动面大。
- **回滚**：保留旧导航常量，开关切换。

### P6 · 桌面下线（5–10 天，独立发布）

- L1 隐藏 → L2 解除 → L3 删除，按 §8 表逐项。
- **验收**：`go list -deps -tags relay` 稳定；桌面构建仍可（若保留）。
- **回滚**：L3 删除前打 tag，可 revert。

---

## 12. 风险清单（按优先级）

| # | 风险 | 影响 | 概率 | 优先级 | 缓解 |
|---|---|---|---|---|---|
| R1 | `internal/gui` 拆分引发大范围回归 | 高 | 高 | **P0** | 路由快照 + 分阶段迁移 + 双注册开关 |
| R2 | 381 条路由中 280 条耦合，facade 遗漏 | 高 | 中 | **P0** | 架构测试强制；逐文件清单 |
| R3 | gateway 对 `internal/sessions` 硬依赖 | 高 | 高 | **P1** | 先拆只读子集；否则 relay 暂时保留 sessions |
| R4 | `plugins/all` 把 agent 拉回 relay | 中 | 高 | **P1** | `plugins/allrelay` |
| R5 | 用户体系与 `access.Key` 兼容 | 高 | 中 | **P1** | 只增可选字段；不重写账本 |
| R6 | 历史 usage 归属漂移 | 中 | 中 | **P2** | 读时归属 + UI 标注 |
| R7 | 测试套件 timer 缺陷阻塞回归 | 中 | 高（已现） | **P0** | 设计 E；同类两处一并处理 |
| R8 | `serve` 直接调用 `agent.*` | 中 | 高 | **P2** | `serve_extras_*` 分文件 |
| R9 | 单密钥 break-glass 泄露 | 高 | 低 | **P2** | 仅 owner、可轮换、审计 |
| R10 | 明文凭证（providers.json 0600） | 中 | 中 | **P3** | 评估信封加密/KMS |
| R11 | 前端 1.34 MB 单文件 | 中 | 中 | **P3** | 按页拆 chunk；不动内核 |
| R12 | 无审计日志 | 中 | 高 | **P2** | `audit.jsonl` 先行 |

---

## 13. 本轮实际运行结果

### 13.1 已执行（可复核）

| 命令 | 结果 |
|---|---|
| `git log -1` / `git status --porcelain` | `7dc85f26…`；仅 `?? .mimosa/`、`?? docs/product-v2/` |
| `go list -deps ./internal/gateway \| grep -E 'agent\|library\|sessions'` | 仅 `internal/sessions` |
| `go list -deps ./internal/provider \| grep …` | 空 |
| `go list -deps ./internal/middleware \| grep …` | 仅 `internal/sessions`（经 usage） |
| `go list -deps -tags nogui ./internal/cli` | 含 agent/library/sessions |
| `CGO_ENABLED=0 go build -tags nogui -o /tmp/magpie-plan-check .` | **BUILD OK** |
| `go test -tags nogui -count=1 ./internal/access/ ./internal/daemon/` | **ok / ok** |
| `go test -tags nogui -run 'TestWebGuard\|TestWebKey\|TestWebGuardBookmark' ./internal/gui/` | **ok** |
| `go test -tags nogui -run 'TestAffinity\|TestStick' ./internal/gateway/` | **ok** |
| 两条 gateway-mode 测试单独运行 | **各自 PASS** |
| 两条合并运行（anchored） | **FAIL**：`fatal error: reset of synctest timer from outside bubble` |
| 复现工程（Go 1.27.2）：气泡内 `AfterFunc` → 气泡外 `Reset` | **FAIL**（同 fatal） |
| 复现工程（Go 1.26.8，已下载 toolchain）：同上 | **FAIL**（同 fatal） |
| 复现工程：气泡内 `AfterFunc` → 气泡外 `Stop` | **FAIL**（`stop of synctest timer…`） |
| 复现工程：真实 timer → 气泡内 `Reset` | **PASS** |
| 复现工程：设计 E 两用例 × 两个 Go 版本 | **全 PASS** |

### 13.2 未执行（不得视为 PASS）

| 项 | 标记 |
|---|---|
| `go test -tags nogui ./internal/gui` 全包 | **NOT TESTED**（耗时，且本轮禁止改源码） |
| `go test ./internal/gateway`（约 4539 个 Test 函数，未执行） | **NOT TESTED** |
| `go test ./internal/sessions` / `./internal/usage` 全包 | **NOT TESTED** |
| `go build -tags relay` | **NOT TESTED**（尚未实现） |
| Docker 构建/运行 | **NOT TESTED**（环境无 docker） |
| 真实 Provider 端到端调用 | **NOT TESTED**（无真实 Key） |
| 桌面端托盘/通知（macOS/Windows） | **NOT TESTED**（无环境） |
| 设计 E 应用进仓库 | **NOT TESTED**（只读阶段） |

---

## 14. 能力状态总表

| 模块 | 已有 | 部分实现 | 完全缺失 | 未验证 |
|---|---|---|---|---|
| 协议（Chat/Responses/Anthropic/Gemini/图像/音频/Embedding/Rerank/Realtime/视频） | ✔ | | | 端到端 |
| 渠道/凭证/账号 | ✔ | | | 订阅登录 |
| 模型目录/价格/映射 | ✔ | ✔（可视化） | | |
| 路由组/策略/回退/亲和 | ✔ | ✔（策略对象） | | |
| 令牌/额度/限额 | ✔ | ✔（RPM） | | |
| 用量/成本/日志 | ✔ | | ✔（按用户） | |
| 插件/Middleware | ✔ | ✔（隔离/审计） | | |
| 用户/组/RBAC | | | ✔ | |
| Dashboard | | | ✔ | |
| 审计日志 | | | ✔ | |
| Web 通知中心 | | ✔（Bark） | ✔（站内） | |
| 服务启停/健康 | ✔ | ✔（启停 UI） | | |
| 桌面剥离 | | ✔（已隔离） | | relay 构建 |
| 测试基础设施 | | ✔ | | 全量 |

---

## 15. 停止点

本方案为设计与审计产物，**未开始实施**。
- 未修改任何业务源码、前端、配置、Schema、数据库或测试文件。
- 未提交、未推送。
- 等待审核与明确授权后再进入 P0。

**交付物**：本文件 `docs/product-v2/MAGPIE-WEB-RELAY-IMPLEMENTATION-PLAN.md`。