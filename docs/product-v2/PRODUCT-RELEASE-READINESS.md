# MiniRelay 产品整体交付评估（Release Readiness）

> 日期：2026-11-24
> 基线：`/root/magpie-original`，分支 `remove-partners`，HEAD `e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e`
> 工作树：46 个已修改文件、33 个未跟踪条目（平台账号/门户/模型授权/文档均未提交）
> 结论：**暂不可发布**。存在 2 个 P0 安全缺陷（其中跨租户读取已在隔离实例实测复现）以及发布基线与灾备两类 P0 阻塞。

---

## 1. 结论摘要

| 维度 | 判定 | 说明 |
|---|---|---|
| 模型中转核心链路 | **实现完整，有 P0** | OpenAI/Anthropic/Gemini 入口、用户 Key 认证、模型授权、路由与故障转移、流式与非流式、用量记录均存在；但空模型授权 Key 被当成“全权限”，可跨租户读取与执行全局动作 |
| 用户端产品闭环 | **基本完整，有 P0/P1** | 注册/登录/会话/Key/模型广场/用量日志可用，跨用户 IDOR 未发现；但 Key 默认拥有全部 MCP，图片/视频可绕过授权，缺少用户配额与找回密码 |
| 管理端产品闭环 | **功能完整，权限模型偏宽** | Provider/Routing/Groups/Gateway/模型开放/用户管理/用量/设置齐全；但 admin 等于完整运维者，页 key 等价 root，缺少审计与再认证 |
| 生产可用性 | **部分可用** | 构建、原子写、权限、CORS/Host/SSRF、drain、并发限制、加密备份都有实现；但发布基线不可复现、灾备缺平台状态、长期运行治理不足 |
| 最终结论 | **暂不可发布** | 见 §9 的发布条件 |

---

## 2. 范围、方法与边界

### 2.1 本轮做了什么

- 只读审计当前工作区源码、测试与现有文档；未修改业务代码、CSS、JS、Schema、迁移或配置。
- 运行了**针对性**测试（不重跑 P1 全量、不重跑 Gateway 全量）。
- 用当前源码构建了临时二进制并做了黑盒边界探测。
- 在**独立临时 HOME**（`/tmp/relaywebrepro`，非生产配置）中复现了 P0；生产配置、端口、账号未被写入。
- 所有 P0/P1 结论都落到当前 `file:line`、现有测试或本轮实测；未沿用历史报告结论。

### 2.2 证据分级

| 级别 | 含义 |
|---|---|
| **本轮实测** | 本轮构建/启动/curl/探针直接观察到的结果 |
| **当前代码** | 当前工作区源码可复核，附 `file:line` |
| **现有测试** | 仓库中已有测试；本轮只跑了一部分，未跑全量 |
| **历史证据** | Phase 7/8 报告的记录（真实上游/mock E2E），仅作背景，不当作当前可用性证明 |
| **未验证** | 当前环境无法执行（真实厂商、真实 TLS、Docker、多 pod、长期运行等） |

### 2.3 Git 保护

| 项 | 开工 | 结束 |
|---|---|---|
| 分支 / HEAD | `remove-partners` / `e41adc151` | 未变 |
| `git status --porcelain` | 79 行（46 M + 33 ??) | 未变 |
| 暂存区 | 0 | 未变 |
| `git diff --stat` | 46 files / 1316 insertions / 683 deletions | 未变 |

未执行 checkout/reset/clean/stash/cherry-pick/commit/push。

---

## 3. 产品功能完成度

| 能力 | 实现 | 状态 | 主要证据 |
|---|---|---|---|
| 用户 Key → Gateway 身份认证 | `access.Check`（SHA-256/常量时间）、`lanGuard`/`identifyCaller` | 可用 | `internal/access/issue.go:246`、`internal/gateway/lan.go:556,798` |
| 模型授权（Key 白名单 + 平台开放策略） | `modelAllowed`/`entryAllowed`/`modelaccess.AllowsModel` | 可用（具名模型） | `internal/gateway/key_models.go:237,428`、`internal/modelaccess/modelaccess.go:138` |
| Routing / 分组 / 权重 / 故障转移 | provider 组、候选排序、fallback、亲和 | 可用 | `internal/provider/group.go`、`internal/gateway/fallback.go`；本轮跑过 4 个故障转移测试 |
| OpenAI 非流式 / 流式 | `/v1/chat/completions`、SSE 转换 | 可用（mock/单测） | `internal/gateway/gateway.go:1065-1130`；本轮跑过流式与用量测试 |
| 多协议 | Responses、Anthropic Messages、Gemini、embeddings、images、videos、audio | 可用 | `internal/gateway/gateway.go:1248` 列出全部入口 |
| `/v1/models` 与授权隔离 | 按 Key 过滤；匿名本机可见空目录 | 可用（Key 维度） | `internal/gateway/key_models.go:428`；`TestGatewayKeyModelsHoldAKey` |
| 错误 / 超时 / 取消 / 重试 | OpenAI 错误结构、上游响应头超时、上传停滞、drain | 可用（有测试） | `internal/gateway/upstream_transport.go:73`、`internal/gateway/handoff*`；本轮跑过取消/流式测试 |
| 用量与日志 | `usage.Record`、按 Key/Owner 过滤、用户页 | 可用（有隔离测试） | `internal/usage/usage.go:30`、`internal/usage/user_usage.go`；`TestUserUsageIsTheOwnersOnly` |
| 注册 / 登录 / 会话 | 注册默认关闭、Argon2id、服务端会话、撤销 | 可用 | `internal/identity/registration.go:36`、`internal/identity/session.go:21` |
| 用户 API Key 生命周期 | 创建/列表/禁用/删除、仅显示一次、哈希存储 | 可用（有 P0 语义缺陷） | `internal/access/userkeys.go:167`、`internal/gui/userkeys.go:15-110` |
| 模型广场 / 授权模型展示 | 只列开放模型 | 可用 | `internal/gui/modelaccess.go:51` |
| 管理端 Provider / Routing / Groups | 增删改查、保存生效、故障转移 | 可用 | `internal/gui/providers.go`、`internal/gui/groups.go`；现有测试 |
| 模型开放 / 命名 / 描述 | `model-access.json` | 可用 | `internal/modelaccess/modelaccess.go:270` |
| 用户管理 | 列表/禁用/删除/改密/改角色/最后管理员保护 | 后端可用；**无管理 UI** | `internal/gui/auth.go:282-382`、`internal/identity/identity.go:460-580` |
| 管理端用量 | 全局 summary/requests/clear | 可用；**无平台用户维度** | `internal/gui/usage.go:141`、`internal/usageapi/usageapi.go:230` |
| 备份 / 恢复 | 加密备份、usage 合并 | 部分；**缺平台账号/模型授权/注册开关** | `internal/backup/backup.go:74-124` |
| 首次部署 / 初始化 | CLI 建首管理员；注册默认关闭 | 可用；**文档缺失** | `internal/cli/user_cli.go:134` |
| 健康检查 / 观测 | CLI `healthcheck`；无 `/healthz`、无指标 | 部分 | `internal/cli/healthcheck_cli.go:24` |

---

## 4. 核心中转链路图与真实代码入口

```text
客户端（OpenAI / Anthropic / Gemini / Responses）
  │
  ├─ 页面端口 3430 /v1/*（oneport.go；一律按“另一台机器”处理）
  │     └─ 需要共享开启 + 有效 Key（gateway/user key）
  │
  └─ Gateway 端口 3425（默认 loopback；MAGPIE_ADDR 可改）
        │
        ├─ http.Server                                    gateway.go:911
        ├─ lanGuard：允许列表 → 本机/远程判定 → 共享判定   lan.go:556
        ├─ identifyCaller → access.Check                  lan.go:798 / access/issue.go:246
        │     ├─ 用户 Key 使用访问前缀（`access/access.go:33`）+ 常量时间比较
        │     ├─ Gateway Key：secret 比较（管理员可再复制）
        │     ├─ 已停用/过期/issuer 失效 → 401
        │     └─ Owner 账号被禁用/删除 → 401
        ├─ 路由表（/v1/chat/completions 等）               gateway.go:1065-1130
        ├─ 模型解析与授权
        │     ├─ Key 的 Models/Accounts/Limit              access/models.go, issue.go
        │     ├─ 平台开放策略（用户 Key 必须同时开放）     key_models.go:237
        │     └─ 分组/成员/账号可用性                       key_models.go:267-298
        ├─ 候选、排序、故障转移、亲和                       fallback.go, provider/group*.go
        ├─ 上游 HTTP 传输（超时/HTTP2/上传停滞）            upstream_transport.go:73
        ├─ 协议转换与 SSE 流式（flush/断流/续写）           gateway 各协议文件
        ├─ 用量记录 usage.Record                           usage/usage.go:30；gateway.go:716
        └─ 用户/管理端读取
              ├─ /api/user/usage/*（owner 来自会话）        gui/userusage.go:51
              └─ /api/usage、/v1/magpie/usage（全局）        gui/usage.go, gateway/usage_api.go
```

关键安全判定点：

- **本机信任**：`local(r)` 为真时无需 Key，直接以空 Identity 访问全部模型（`internal/gateway/lan.go:983-990`）。
- **远程信任**：`remote` 请求必须有有效 Key；无 Key 401（`lan.go:591-665`）。
- **模型授权**：用户 Key 同时受 Key grant 与平台开放策略约束；管理员/本机 Key 保留原有全量能力（`key_models.go:237-249`）。
- **MCP 授权**：空 MCP 列表 = 全部服务器/工具；用户 Key 创建时不设置 MCP（见 P0-2）。

---

## 5. 已验证能力及证据

### 5.1 构建与静态检查（本轮实测）

| 项 | 命令 | 结果 |
|---|---|---|
| 当前源码构建 | `go build -tags nogui -o /tmp/minirelay-audit-bin .` | **成功，6.8s**，129 MB |
| 版本 | `/tmp/minirelay-audit-bin version` | `magpie dev`（未注入发布版本） |
| 核心包 vet | `go vet -tags nogui ./internal/access ./internal/modelaccess ./internal/identity ./internal/gateway ./internal/gui` | **exit 0** |
| Go 版本 | `go version` | 本机 **1.27.2**；`go.mod` 声明 **1.26.8**；Docker 用 `golang:1.26-alpine` |

### 5.2 针对性测试（本轮实测）

| 测试 | 结果 |
|---|---|
| `TestUserKeyModelAccessThroughTheGateway`、`TestGatewayKeyModelsHoldAKey`、`TestAUserKeyNeverNamesARoutingGroup`、`TestKeyModelsNarrowCandidates`、`TestCallerIdentitySurvivesStreamingFailover` | PASS |
| `TestLiveCountsARealStream`、`TestClaudeStreamOutputCounted`、`TestKeyConcurrencyKeepsStreamAlive`、`TestStreamCutUsageTellsLastTry`、`TestDrainedGatewayFinishesItsStream` | PASS |
| `TestFallbackGivesBackAcrossRoutings`、`TestWeighedRankIsDraggedOrder`、`TestOnAccountSkipMovesToNextAccount`、`TestAtOnceFullLaneWaitsForTheKept` | PASS |
| `TestAdminAPIGuards` | PASS（6 个子用例） |
| `TestUserUsageIsTheOwnersOnly`、`TestUserKeysAreNotTheAdmins`、`TestUserKeysAPI`、`TestUserKeyModelGrantsAPI`、`TestModelAccessAdminAndUserAPIs` | PASS |
| `TestUserReportIsTheOwnersOnly`、`TestUserRequestHidesSecretsAndLimitsError` | PASS |
| `TestI18nLanguagesAgree`、`TestPortalI18nKeys` | PASS（0.776s） |
| `go test -tags nogui ./internal/access ./internal/modelaccess ./internal/identity` | PASS（0.347s / 0.047s / 0.725s） |

> 这些测试证明的是“当前代码在 mock/httptest 下的行为”，**不是真实厂商上游**的证明。Gateway 测试尤其如此：`internal/gateway/main_test.go:120-123` 调 `testenv.Offline()`，把非 loopback 拨号全部拒绝（`internal/testenv/offline.go:25-48`），所有上游都是 `httptest.NewServer` 假服务。唯一的 live 测试默认 `t.Skip`，且网关上游仍是假服务。

### 5.3 黑盒边界探测（当前二进制，P1 隔离实例）

| 请求 | 结果 | 判定 |
|---|---|---|
| `GET /portal/`、`/portal/login`、`/portal/register` | 200 | 公开页可用 |
| 匿名 `GET /portal/console`、`/portal/models`、`/portal/admin-models` | 302 → 登录 | 边界正确 |
| 匿名 `GET /api/settings` | 401 | 管理 API 受保护 |
| 匿名 `GET /api/auth/me` | 401 JSON | 正确 |
| 页面端口匿名 `GET /v1/models` | 403（需共享 + 网关 Key） | 正确 |
| Gateway 端口匿名 `GET /v1/models` | 200，空目录 | 本机信任（设计如此） |
| Gateway 端口伪造用户 Key | 401 JSON | 正确 |
| 恶意 Origin 的 CORS 预检 | 403 | 正确 |
| `POST /v1/chat/completions` 空 body / 坏 JSON | 400 + OpenAI 错误结构 | 正确 |
| 不存在的模型 | 404 + `not_found_error` | 正确 |
| `magpie healthcheck --addr 127.0.0.1:3425` | exit 0；停止后 exit 1 | 可用 |
| 33 个门户页面/文件、管理端 390/1440（P1 证据） | 0 横向溢出；控制台无新增错误 | 沿用 P1 验收 |

### 5.4 历史真实上游与 mock E2E（仅背景）

| 证据 | 内容 | 边界 |
|---|---|---|
| Phase 7 报告 | 真实 DeepSeek（Anthropic Messages 兼容）非流式/流式、模型映射、两用户隔离、403 拒绝、用量核对 | **2026-11-23 的历史记录**；凭据与测试 HOME 已删除，本轮未复现 |
| Phase 8 报告 | mock 上游 + 独立 HOME 的 7.065s E2E：Provider/分组/模型开放/用户/Key/流式/拒绝/重启/备份恢复 | **mock 上游**；不是真实厂商证明 |

本轮没有调用任何真实厂商；没有把上述历史结果当作当前生产可用证明。

### 5.5 链路审计的正面结论（当前代码 + 现有测试）

| 结论 | 证据 |
|---|---|
| 远端匿名不能访问任何注册路由 | `TestNetworkGatewayGuardsEveryRoute` 枚举全部路由；`lanGuard` 对非 loopback 强制 Key |
| 受限 Key 的故障转移不会落到无权模型/账号 | 候选在 `allowedCandidates`/`groupKeeps` 二次过滤；`TestGatewayKeyModelsHoldAKey` |
| 未发现伪造响应或伪造 usage | usage 只来自上游事件/JSON；上游不报时记 0，不用估算填充 |
| 调用方 Key 不会到达上游 | `identifyCaller` 把入站 Authorization 改写为内部 token；上游只收到 `p.Sign` 的凭据 |
| 上游重定向只允许同源 | `provider/redirect.go:27-53` + `CheckRedirect` |
| 错误模型/未知模型不会降级到其他模型 | 404/403；候选列表不含未授权模型 |
| 中途断流以协议内 error 事件结束并记为失败 | `stream_cut_*`、`stream_error_*` 测试 |

> 这些是“没有发现某类漏洞”的正面证据，不等于穷尽证明；P0-1/P0-2 仍是已复现的缺陷。

---

## 6. 未验证能力

1. **当前版本的真实厂商上游**：本环境没有可用凭据，也未获授权发起外部付费调用；流式/非流式、429/5xx、真实配额与模型列表均未在真实厂商上复跑。
2. **生产 Docker 镜像**：本机无 Docker/podman；多阶段构建、nonroot 写 `/config`、HEALTHCHECK、镜像 digest 未验证。
3. **真实 TLS / mTLS / 反向代理**：没有真实证书、Caddy/nginx/Cloudflare；Secure Cookie、Host 改写、客户端证书、HTTP/2 未验证。
4. **多机 / 多 pod / 共享卷**：单机 JSON/JSONL + 文件锁；K8s rolling update、两个 pod 同卷并发未验证。
5. **故障注入**：磁盘满、只读卷、kill -9、断电、NFS 锁、时钟跳变未验证。
6. **长期运行**：数天/数周的 heap、goroutine、`unfit` map、usage/日志增长、FD 泄漏未验证。
7. **真实 `magpie update`**：下载、attestation、替换二进制、服务重启/回滚未联网验证。
8. **macOS / Windows / iOS Safari**：launchd、Keychain、WebView2、WSL、移动端 Cookie 限制未验证。
9. **桌面 GUI 构建**：本机缺 `fontconfig`/cgo 依赖，未验证非 `nogui` 构建。
10. **负载与配额**：没有吞吐/并发压测，也没有真实用户量下的成本与配额验证。
11. **Gateway 测试的真实性**：`internal/gateway/main_test.go:120-123` 主动断网，所有上游都是 `httptest` 假服务；因此“测试通过”只证明网关行为，不证明真实 vendor 兼容性、流式时序或限流重试策略。

---

## 7. 阻塞发布的问题

### P0-1（安全，已复现）：空模型授权的用户 Key 被当成“全权限 Key”，可跨租户读取与执行全局动作

**事实链（当前代码）**

1. 门户创建 Key 只检查原始数组长度：`internal/gui/userkeys.go:37`。
2. 清理函数跳过 trim 后为空的项且不报错：`internal/access/userkeys.go:124-134`。
3. `CreateUserKey`/`UpdateUserKey` 接受清理后的空列表并写入 `ModelsSet=true`：`internal/access/userkeys.go:167-243`。
4. 用户可发送 `{"name":"x","models":[""]}` 创建或 PATCH 出这种 Key（`[""]`、`[" "]`、`[null]` 均可）。
   另有两条不依赖手工构造的路径：
   - 管理员在 Gateway keys 页对用户 Key 执行 `models-key` 空数组：管理端语义是“放开全部模型”（`internal/access/access.go:735-741`），但用户 Key 的空数组实际是“无任何模型”（`internal/access/models.go:88-93`）。
   - 历史用户 Key 的 `fillLegacyModels` 在平台无开放模型时写入空授权并 `ModelsSet=true`（`internal/access/userkeys.go:137-162`）。
5. Gateway 的 `heldKey` 只看 `Limited()/Models/Accounts`，**不看 `Owner`**：`internal/gateway/usage_api.go:31-33`。
6. 于是该 Key 在以下接口被视为“未受限”：
   - `/v1/magpie/usage`、`/v1/magpie/usage/requests`（全局用量与请求明细）
   - `/v1/magpie/quotas`、`/quotas/history`、`/quotas/cards`（账号、邮箱、余额）
   - `/v1/magpie/library`、`/v1/magpie/agent-models`
   - `/v1/magpie/trace`、`/v1/magpie/route`、`/route-events`、`statusline`
   - `POST /v1/magpie/quotas/codex-reset`（不可逆消耗 Codex reset）
   - `POST /v1/magpie/update`（触发 magpie 自更新）
   - `POST /v1/magpie/usage/requests`（写入无 owner 的伪造用量）

**本轮复现（隔离实例，当前二进制）**

```text
POST /api/user/keys  {"name":"empty grant via API","models":[""]}
  -> 201, key with no models

GET /v1/magpie/usage            (with that key) -> 200 全局 summary
GET /v1/magpie/usage/requests   (with that key) -> 200 全局 ledger
GET /v1/magpie/quotas           (with that key) -> 200 全局 quotas
POST /v1/magpie/quotas/codex-reset?user=... -> 502 "no Codex account"（已越过 held 拦截）
POST /v1/magpie/update          (with that key) -> 200 {"state":"unsupported","version":"dev"}
```

另用只读 handler 探针写入一条属于 `other-owner` 的记录后，该 Key 的 `/v1/magpie/usage/requests` 返回：

```json
{"callerKeyName":"other-user-key","providerAccount":"other@example.com",
 "provider":"other-provider","agent":"other-tenant-agent","input":7,"output":3}
```

**影响**

- 任意已登录普通用户无需管理员动作即可构造该 Key。
- 在网关共享/远程部署（多租户平台的前提）下，可读取跨用户用量、Key 名/ID、模型、供应商账号/邮箱/余额、Library/路由信息。
- 可消耗不可逆的上游 Codex reset、触发自更新、注入用量记录。
- 不占用模型授权，但可组合 P0-2/P1 的 MCP 与媒体默认模型。

**修复建议（仅建议）**

- `heldKey` 改为 `who.KeyID != "" && who.Restricted()`（`Restricted()` 已包含 `Owner != ""`）。
- `CreateUserKey`/`UpdateUserKey` 拒绝清理后为空的模型列表（或显式标记为 held）。
- `codexReset`/`remoteUpdate` 这类全局/不可逆动作改为显式管理员能力，不以“未受限 Key”作为授权。
- 为所有 `heldKey`/`refuseHeld` 端点补“空 grant 用户 Key”回归测试。

### P0-2（安全）：平台用户 Key 默认拥有全部 MCP 服务器与工具

**事实链**

- 用户 Key 创建不设置 MCP：`internal/access/userkeys.go:167-210`。
- 空 MCP 列表 = 允许全部服务器/工具：`internal/access/mcp.go:11-17,63-80`。
- `identityOf` 对用户 Key 清空 Admin/Memory/Issues/Browser/ReadOnly，但**没有清空或限制 MCP/Quotas**：`internal/access/issue.go:399-405`。
- MCP 入口只检查 `who.MCPServer/MCPAllows`：`internal/gateway/mcp_key.go:61-72`、`hub_mcp.go:347-364`、`tools_mcp.go:51`、`search_mcp.go:103`、`vision_mcp.go:101`、`image_mcp.go:223`。
- 用户 Key 创建 UI 只选模型，不选 MCP：`internal/gui/userkeys.go:30-42`。

**影响**

- 一个“只调用文本模型”的用户 Key 实际可调用：
  - `magpie-tools`（已启用插件的工具，可能在本机执行插件代码）
  - `magpie` hub（Library 中的 MCP server，可能是文件系统/shell 类）
  - `magpie-web-search` / `web_fetch`（消耗管理员搜索 API、出网抓取）
  - `magpie-vision` / 图片/视频生成（消耗管理员上游模型）
  - 各 `magpie-sessions` / `magpie-browser` 等按当前能力
- 这不是“模型调用”的授权边界，属于把普通租户提升为“工具使用者/主机能力使用者”。

**修复建议**

- 对 `Owner != ""` 的 Key 默认 `MCP = NoMCP`，或在门户提供按 Key 的 MCP 授权并与管理端 `mcp-key` 一致。
- 在 `mcpKeyServe`/`mcpKeyCall`/hub/tools/search/vision/image 入口显式处理用户 Key。
- 增加用户 Key 调用 own MCP、hub、插件工具的拒绝测试。

### P0-3（发布治理）：发布基线不干净，镜像/二进制不可复现

- 当前工作树 46 个已修改文件 + 33 个未跟踪条目；`internal/identity/`、`internal/modelaccess/`、`internal/gui/portal.go`、`internal/cli/user_cli.go`、整个 `docs/product-v2/` 都未提交。
- `Dockerfile:1-12` 的 `COPY . .` 会把这些未提交代码编入镜像；HEAD 构建则完全没有平台账号/门户功能。
- 预置的 `/root/magpie-original/magpie` 无法证明与当前源码一致；本轮构建版本仍是 `dev`。
- **影响**：无法审计“发布的到底是哪份代码”，HEAD 与目录构建行为不同，回滚困难。
- **建议**：先在干净 commit/tag 上合并、跑 CI，再构建并记录镜像 digest 与 commit；本轮不提交。

### P0-4（灾备）：备份不含平台账号、模型授权、注册开关，恢复后用户 Key 失效

- `backup.Bundle` 只有 providers/settings/gatewayKeys/usage 等，没有 `identity`/`modelaccess`：`internal/backup/backup.go:74-124`。
- `users.json`、`model-access.json`、`registration.json` 不在收集范围；恢复后：
  - 平台管理员/普通用户不存在；
  - 用户 Key 带 `Owner` 但账号不存在 → 网关 401（`internal/gateway/lan.go:847-855`）；
  - 模型开放策略为空 → 模型广场为空。
- **影响**：整机丢失/迁移/勒索恢复后，平台身份与授权永久丢失，用户 Key 全部失效。
- **建议**：把三个文件纳入加密备份/恢复，或把“必须手工拷贝”写成强制步骤并做恢复演练；恢复后增加管理员登录与用户 Key 调用 smoke。

### P1（发布前应处理）

| # | 问题 | 证据 | 影响 |
|---|---|---|---|
| P1-1 | 用户可撤销/重建管理员对其 Key 的限制 | `userkeys.go:56-77,93-110`、`access/userkeys.go:215-243`；`limit-key` 在 `access.go:666` | 管理员设置的模型/预算/关闭/ MCP 限制可被 owner PATCH 或删除重建绕过；没有账号级 entitlement |
| P1-2 | 图片/视频省略 `model` 时绕过开放策略与 Key 授权 | `gateway/draw.go:732-766`、`video.go:1104-1141` | 任意用户 Key 可用管理员默认图片/视频模型，产生真实费用；门户“关闭模型不可调用”不成立 |
| P1-3 | 没有用户组 / 账号级模型授权 / 默认配额 | `modelaccess` 为全局策略；`identity.User` 无组字段；用户 Key 无默认 `Limit` | 无法表达“团队 A 只能用模型 X”；开放注册 + 无配额 = 成本滥用 |
| P1-4 | 注册开放时无限流/验证码/邮箱验证；Argon2id 可被放大量 | `gui/auth.go:181-217`、`identity/identity.go:336-340` | 批量注册消耗 CPU/内存并获得全部开放模型调用能力 |
| P1-5 | 登录限流按代理 IP 计数且只在内存 | `gui/loginlimit.go:120-129` | 反代后 20 次失败可锁住所有用户；重启清零；多实例不共享 |
| P1-6 | Session 不轮换、登出只清一个 cookie、绝对 7 天 | `identity/session.go:21-31,129-140`、`gui/auth.go:67-100,257-268` | 旧 token 在重新登录后仍有效；多同名 cookie/同主机其它端口可干扰；缺少“退出所有设备” |
| P1-7 | 管理员=完整运维者；页 key 等价 root；缺审计/MFA/再认证 | `gui/web.go:471-483`、`gui/caller_keys.go:253`、`gui/auth.go:349` | 页 key 泄露可读全部用量/provider key、创建 Admin 网关 Key；管理操作无 actor 审计 |
| P1-8 | 本机 loopback 无 Key 即全模型；`MAGPIE_TRUST_PROXY=1` 全局信任 | `gateway/lan.go:983-1013`、`key_models.go:242` | 多用户主机上任何本地进程可花掉上游额度；代理配置错误可变远程无鉴权 |
| P1-9 | 用量日志永不轮转；写入失败静默 | `usage/usage.go:326-360`；“清除”只隐藏 `usage/cleared.go:14` | 磁盘无限增长；磁盘满后计量/计费静默丢失 |
| P1-10 | settings/providers 无跨进程锁，文档却声称有 | `settings/settings.go:1970`（仅进程内）、`provider/provider.go:928`（无锁）；`docs/reference.md:4603` | App/服务/CLI/多 pod 并发保存会丢失更新 |
| P1-11 | users/sessions/caller-keys 损坏无 lastgood | `identity/identity.go:114-127`、`session.go:40-66`、`access/issue.go:246` | 一次损坏导致登录失败或所有远程 Key 401 |
| P1-12 | 首次管理员只能 CLI；Docker 文档未写 | `cli/user_cli.go:134`；`site/public/docs/docker.html` 无 `user add` | 托管环境按文档部署后无法创建管理员/开通门户 |
| P1-13 | 升级无 schema 版本、无回滚命令 | `backup/backup.go:411-433`、settings/providers 整结构重写 | 降级可能丢新字段；旧版打开新备份部分恢复 |
| P1-14 | 管理端用量没有平台用户维度 | `usageapi/usageapi.go:230`；`usage.Record` 有 Owner 但 API 行不含 | 无法按用户做成本/滥用审计；用户重建 Key 后更难追踪 |
| P1-15 | 无标准健康/就绪端点、无请求 ID、无指标/集中错误上报 | `cli/healthcheck_cli.go:24`；无 `/healthz`；日志无结构 | K8s/监控集成需 exec 探针；故障定位依赖文本日志 |
| P1-16 | 当前版本未做真实上游验收 | 本环境无凭据/未授权外部调用 | 发布前必须用发布候选版本跑真实厂商非流式/流式/错误/超时 |
| P1-17 | Join 口令签发的是无模型/预算限制的全权 Key | `gateway/join.go:160-173`、`access/access.go:607-620` | 任何拿到共享 join 口令的人绕过平台用户 Key 的模型授权；口令实现本身良好（PBKDF2、限速、可撤销），但权限过大 |
| P1-18 | Provider/Gateway Key、sign-in 明文落盘（0600），无 OS keyring/KMS | `provider/provider.go:68-79`、`provider/logins.go:208`、`access/access.go:1097-1115` | 配置目录/备份/卷泄露即上游凭据泄露；0600 是唯一保护 |

### P2（发布后治理）

1. 门户文案与实际不一致：`index.html` 仍有 “wired in Phase 3”；注册关闭时仍展示 Register；删除 Key “立即停止”不保证中断进行中的流。
2. 用户用量时区固定为服务器 `time.Local`（`gui/userusage.go:47`），远程用户“今天”可能错误。
3. 用户日志错误文本只掩码 `sk-/gsk_/xai-/Bearer`，可能带 URL/账号/其它密钥（`usage/user_usage.go:299-327`）。
4. `GET /api/user/keys` 会因 legacy 填充产生写副作用（`access/userkeys.go:81-96`）；GET 不应持久化。
5. 没有用户自助改密/找回密码/邮箱验证/账号删除/“退出所有设备”；没有管理员用户管理 UI（只有 API）。
6. 页面 API 多个 POST 直接无界读 body（`gui/api.go:135,1784`）；网关队列默认无界（`key_slots.go`、`concurrency.go:19`）；`s.unfit` 无上限。
7. 页 key 出现在启动日志/URL/服务文件；logout 不清页 key cookie。
8. 备份还缺 `plugins.yml`、`web-page.json`、`redact.key`；自动备份默认同盘。
9. 目录权限混用 0755/0700；`usage.jsonl` 已存在时不再强制 0600。
10. 文档与当前工作区不一致：未带 key 的 `/` 已是 302 登录，Docker 文档仍写 401；`docs/reference.md` 的 Files 章节缺 `users.json`/`sessions.json`/`registration.json`/`model-access.json`。
11. 多凭证请求的“用户 Key 已停用”判断只看第一个凭证（`gateway/lan.go:824-827`）：loopback 上 `Authorization: Bearer magpie` + `x-api-key: <已停用用户 Key>` 会按匿名放行；仅本机、不构成新越权，但撤销语义不完整。
12. 重试耗尽后的最后一次中继可能把“上游 200 + 错误体”以 200 发给客户端（`fallback.go:2329`、`sniff.go:306`）：网关账本记为失败，不会伪造成功，但客户端可能解析失败。
13. 模型别名/Auto 解析发生在白名单检查之后（`gateway.go:6108`、`provider/copilot_auto.go:351`）：只影响 `auto` 类路由模型的授权粒度，不跨供应商。
14. 同一裸模型名被多家 provider 提供时，可能解析到 Key 无权的那家而返回 403（`provider/models.go:4027`）；是可用性问题，不是越权。
15. 上游错误文本写入 `usage.jsonl` 前未过 `redact`（`gateway/reqlog.go:64`），可能含 Key 片段/账号信息。
16. 测试缺口：空 grant/空 MCP/省略 model 的媒体请求/Codex reset 越权/会话固定/多 cookie/代理限流/跨进程写/备份覆盖/usage 轮转/多 pod 均无测试。

---

## 8. 生产部署风险

| 风险 | 事实 | 影响 | 建议 |
|---|---|---|---|
| 两个监听面、两套信任 | 页面端口 3430 的 `/v1` 一律按远程处理，需要共享 + Key；Gateway 端口 3425 本机无 Key 即全权 | 代理把 3425 暴露/转发错误会变成无鉴权或全员可访问 | 只把页面端口放公网；3425 保持 loopback 或 mTLS + Key；明确威胁模型 |
| Docker 默认只跑 `serve` | `Dockerfile:85-86`，无页面；文档要求显式 `web --addr 0.0.0.0:3430` | 按镜像默认启动看不到门户 | 生产 compose/K8s 显式 `web`；补首管理员文档 |
| 首次管理员 CLI-only | 注册默认关闭；页 key 不能调 `/api/admin` | 无 `docker exec` 的托管环境无法初始化 | 初始化 job/一次性 bootstrap；文档写 `--password-stdin` |
| 单机 JSON/JSONL + 文件锁 | 无业务数据库；settings/providers 无跨进程锁；usage 无轮转 | 多 pod/共享卷会丢更新；磁盘增长 | 单副本部署；加锁/迁移到 DB；logrotate/归档 |
| 凭据静态明文 | provider key、gateway key、sign-in 明文 0600；用户 key 哈希 | 文件/备份/卷泄露即上游凭据泄露 | OS keyring/KMS、加密卷、最小权限、轮换 |
| 页 key 等价 root | 启动日志/URL/服务文件；400 天 cookie | 泄露即完整管理面 | 绑定 loopback/VPN；短 TTL/轮换；审计；MFA |
| 反向代理 Cookie 条件严格 | Host 必须等于 `MAGPIE_PUBLIC_URL` 且 `X-Forwarded-Proto=https` | 配置错误导致登录态不保持或不 Secure | 明确部署模板与验收脚本 |
| 无标准探活/监控 | 只有 CLI healthcheck；无 `/healthz`、指标 | K8s/监控集成困难 | 增加 `/healthz`/`/readyz`、请求 ID、指标 |
| 开放注册 | 无限流/验证/配额 | 批量注册与成本滥用 | 保持关闭或加邀请/审批/配额/CAPTCHA |
| 当前未验证真实上游 | 历史 Phase 7 有真实调用，但本轮无凭据 | 发布候选版本可能在上游协议/流式上回归 | 发布前真实厂商验收矩阵 |

---

## 9. 最终结论与发布条件

### 9.1 结论

**暂不可发布。**

理由不是“核心链路没有实现”，而是：

1. **P0-1**：任意普通用户可制造空授权 Key，跨租户读取用量/配额/账号信息，并可消耗不可逆 Codex reset、触发自更新、注入用量。
2. **P0-2**：普通用户 Key 默认拥有全部 MCP/插件工具，越过“模型调用”边界，可能触达主机能力。
3. **P0-3**：发布基线是未提交的 dirty 工作区，镜像/二进制不可复现。
4. **P0-4**：备份/恢复缺平台账号、模型授权、注册开关，灾备会丢身份与授权。

### 9.2 转为“有条件可发布”的最低条件

1. 修复 P0-1：`heldKey` 纳入 `Owner/Restricted`；创建/更新拒绝空 grant；`codex-reset`/`update` 改为显式管理员能力；补全端点回归测试。
2. 修复 P0-2：用户 Key 默认 `NoMCP`，或按 Key 显式授权；补 own MCP/hub/tools 拒绝测试。
3. 修复 P1-2：媒体请求解析默认模型后仍执行 `modelaccess` + Key 授权。
4. 修复 P0-3：在干净 commit/tag 上构建，记录镜像 digest/commit，跑完整 CI。
5. 修复 P0-4：备份纳入平台状态并完成一次真实恢复演练。
6. 用发布候选版本跑真实厂商验收（非 mock）：非流式、流式、错误、超时、429、模型列表、用量。
7. 公开注册前补齐：账号级配额/entitlement、注册限流/邀请、代理感知的登录限流、MFA/管理员审计、usage 轮转与容量告警。
8. 对外部署前确认：Join 口令签发的全权 Key 是否需要收窄、Provider/Gateway Key 是否改用 keyring/加密卷、页 key 的轮换与审计。

### 9.3 若只做内部单租户试点

在**注册保持关闭、用户全部由管理员创建、Gateway 不共享到公网、MCP/插件工具关闭**的前提下，可做受控试点；但 P0-1/P0-2 仍建议先修，因为它们是产品内部的租户边界，不依赖公网暴露。

---

## 10. 证据索引

| 证据 | 位置 |
|---|---|
| 当前源码构建 | `/tmp/minirelay-audit-bin`（129 MB，`magpie dev`） |
| 跨租户 P0 的 handler 复现 | `/tmp/relayprobe/main.go` + 本轮输出 |
| 门户 API 创建空 grant Key 的复现 | `/tmp/relaywebrepro`（独立 HOME；已停止） |
| 针对性测试 | 见 §5.2 的命令与结果 |
| 历史真实上游/mock E2E | `docs/product-v2/PHASE7-REPORT.md`、`PHASE8-REPORT.md`（仅背景） |
| P1 证据 | `docs/product-v2/screenshots/p1-closure/`（对比度/布局/语言） |

**本轮未修改任何仓库文件**；`git status`、HEAD、暂存区与开工时一致。
