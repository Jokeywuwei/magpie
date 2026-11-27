# Phase 8 报告：稳定性修复、生产就绪与最终回归

日期：2026-11-23
项目：`/root/magpie-original`
分支：`remove-partners`
HEAD：`e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e`

## 1. 结论

Phase 8 完成：

- 修复 Phase 7 遗留的 Gateway 全量并发 flake（`TestSearchQueueKeepsCodexAlive/progress`），未跳过测试、未删除断言、未用超长等待掩盖；
- 修复 P2 模型展示不一致：成功与被拒绝的请求现在使用同一个平台模型 ID（`requested`），API 同时保留记录模型 ID、展示 ID 和原始请求 ID，历史记录兼容；
- 复查 Phase 7 的 P1 `X-Forwarded-Proto` 修复，并补充 Host 不匹配、HTTP 公共地址、page key Cookie 等配置错误分支；
- 在独立 HOME、独立端口、独立 mock 上游下完成生产就绪与端到端回归：权限、持久化、优雅/异常重启、usage 索引缓存重建、端口冲突、日志脱敏、注册默认关闭、管理员初始化、健康检查、备份恢复；
- 使用仓库目标 Go 1.26.8 跑完 Gateway 全量（711.738s，零失败）、GUI 全量（167.44s，零失败）、usage/identity/cli 与 `go vet`；
- 将 Phase 8 最终二进制部署到生产端口，保持生产 page key、管理员账号和既有配置不变。

未提交、未推送；未执行 checkout/reset/clean/stash/cherry-pick。

## 2. 开始前状态与保护

| 项目 | 值 |
| --- | --- |
| 分支 | `remove-partners` |
| HEAD | `e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e` |
| 工作树 | 71 项（40 已修改、31 未跟踪） |
| Phase 1–7 改动 | 全部保留 |
| 破坏性 Git 操作 | 未执行 |

Phase 8 只修改了：

- `internal/gateway/search_queue_test.go`（测试内时序余量）；
- `internal/usage/user_usage.go`、`internal/usage/user_usage_test.go`（P2）；
- `internal/gui/assets/portal/usage.js`（P2 展示）；
- `internal/gui/admin_session_test.go`（P1 补充回归分支，Phase 7 已存在修复）；
- 本报告与 Phase 7 报告中的一条修订说明。

生产账号、生产 page key、生产 Provider 配置未被覆盖；所有测试使用独立 HOME 与独立端口。

## 3. 修复

### 3.1 Gateway 队列保活测试的并发 flake

**现象**（Phase 7 全量）：`TestSearchQueueKeepsCodexAlive/progress` 偶发失败：

```text
queue progress or native tool call lost: pings=1 complete=false
```

fake 上游在 43ms 后返回 504。单独运行 20/20 通过，全量并发时偶发。

**根因**：测试把 `translatedQuietLongest` 压到 40ms，fake 队列每 20ms 才发一次进度事件。全量并发下 goroutine 调度抖动超过 40ms，上游被判定为“静默”，返回 504，客户端只看到 1 个 ping。

**修复**（`internal/gateway/search_queue_test.go`）：

```text
translatedQuietLongest: 40ms → 100ms
fake 队列 ticker:       20ms → 5ms
```

保持：

- fake 仍在 3 个进度事件后才返回完整工具调用；
- “silent 模式必须被切断且 pings 有界”的负向断言不变；
- “cancel 模式必须收到 ≥3 pings 且不完成”的断言不变；
- “progress/comments 模式必须完成并包含 `get_value` / `call_1`”的断言不变；
- 没有 skip、没有删除断言、没有把超时当通过。

**验证**：

| 项目 | 结果 |
| --- | --- |
| 单独 `-count=20` | PASS，13.861s |
| 单独 `-count=5`（本轮复跑） | PASS，3.468s |
| 目标 Go 全量 Gateway | PASS，711.738s，零失败 |

### 3.2 P2：成功请求与被拒绝请求的模型展示不一致

**现象**：成功调用记录的是 Provider 的短模型 ID（如 `phase7-flash`），被拒绝的调用只记录客户端请求的完整平台 ID（如 `deepseek-test/phase7-pro`）。用户统计和日志因此出现两套命名，虽然 token 总数一致。

**修复**（`internal/usage/user_usage.go`）：

- `Row.DisplayModel`：展示用平台 ID，取 `Requested`，缺失时回退 `Model`，再回退 `unknown`；
- 模型排行按 `displayModel` 聚合；过滤同时接受展示 ID、记录 ID、原始请求 ID 和路由组别名；
- 用户日志 API 每条记录输出：
  - `model`：历史记录中的模型 ID（保持不变）；
  - `displayModel`：统一展示 ID；
  - `requested`：原始请求 ID（保持不变）；
- 历史记录没有 `requested` 时回退到 `model`，旧数据语义不变；
- 控制台日志列显示 `displayModel || model || "unknown"`。

**新增回归**（`internal/usage/user_usage_test.go`）：

- 成功调用（短 `model` + 完整 `requested`）与拒绝调用（只有完整 `requested`）在排行中合并为同一个平台 ID；
- 成功请求同时保留 `model` 与 `displayModel`；
- 旧短 ID 过滤仍能选中它命名的成功调用；
- 完整 ID 过滤同时选中成功与拒绝；
- 历史无 `requested` 的记录仍按 `model` 展示。

**验证**：目标 Go 下该测试 `-count=3` 全通过；`internal/usage` 全量 PASS（98.595s）；浏览器中的排行与日志都显示 `phase8-mock/phase8-m1`、`phase8-mock/phase8-m2`。

### 3.3 P1 受信代理 HTTPS 复查

Phase 7 已把 `secureRequest` 改为：

```text
Secure = 直接 TLS
      或 MAGPIE_PUBLIC_URL 为 HTTPS
         且请求 Host 等于该公共地址的 Host
         且 X-Forwarded-Proto: https
```

Phase 8 复查并补充 `TestSecureCookiesBehindHTTPS` 分支：

| 场景 | 结果 |
| --- | --- |
| 直接 TLS | Secure + HttpOnly + SameSite=Lax |
| 未配置公共地址 + 伪造 `X-Forwarded-Proto: https` | 不 Secure |
| HTTPS 公共地址 + Host 匹配 + 头 | Secure |
| 已配置公共地址但 Host 不匹配 | 不 Secure |
| `MAGPIE_PUBLIC_URL=http://...` + 头 | 不 Secure |
| page key 链接在配置代理下 | 303 + Secure Cookie |

目标 Go 下测试通过。判断没有放宽到“任意代理头”，配置错误一律按不安全处理。

## 4. 最终隔离环境与回归

### 4.1 环境

| 项目 | 值 |
| --- | --- |
| 隔离 HOME | `/tmp/phase8home`（运行前重建） |
| 页面端口 | `127.0.0.1:3520` |
| Gateway 端口 | `127.0.0.1:3521` |
| 恢复验证端口 | 页面 3524 / Gateway 3525 |
| 上游 | 本机 mock OpenAI 兼容服务 `127.0.0.1:39003` |
| 管理员 | `admin@phase8.test`，由 `magpie user add ... --role admin --password-stdin` 初始化，无默认密码 |
| 普通用户 | `a@phase8.test`、`b@phase8.test`，由管理员通过后台创建 |
| 浏览器 | Playwright Chromium 1254 |
| 凭据 | 每次运行内存中随机生成；不写源码、报告、证据或日志 |

脚本：`/tmp/phase8verify/verify.cjs`；运行器 `/tmp/phase8final/run.cjs`（生成随机凭据、清理 HOME、CLI 初始化管理员）；结果：**PASS，7.065s**。

### 4.2 覆盖与结果

| 项目 | 结果 |
| --- | --- |
| 管理员 Session 打开完整原后台（Providers/Routing/Groups/Gateway/Usage/Masking/Plugins/Settings） | 通过，无 page key Cookie |
| 原后台保存 Provider `phase8-mock`、两个模型映射 `phase8-m1/m2`、Routing 分组 `phase8` | 通过 |
| `/portal/admin-models` 开放两个模型 | 通过（开放计数 2） |
| 管理员创建用户 A/B（注册关闭） | 通过 |
| 用户 A/B 各建一个只授权自己模型的 Key | 通过 |
| 非流式调用 | 200，`PHASE8_OK`，usage 5/2/7 |
| 流式调用 | 200，SSE 完整，收到 `[DONE]` |
| mock 上游校验 Bearer key | 通过（错误 key 不进入转发） |
| A 调 B 模型 / B 调 A 模型 | 均 403，不请求上游 |
| 模型关闭 | 已有 Key 立即 403；另一模型仍 200 |
| 模型重开 | 原 Key 恢复 200 |
| 用户 A/B 用量与日志隔离 | 通过：A 5 次/3 成功/2 失败/21 tokens；B 3 次/2 成功/1 失败/14 tokens |
| P2 展示 | 排行与日志均为 `phase8-mock/phase8-m1`、`phase8-mock/phase8-m2` |
| 注册默认关闭 | 通过；打开后注册成功，再关闭后 403 |
| 管理/用户/Gateway 边界 | 通过 |
| 优雅重启（SIGTERM） | 退出码 0；会话、Key、Provider、模型规则、用量一致 |
| 异常退出（SIGKILL）后重启 | 管理员 Session、Key、历史仍可用 |
| 健康检查 | 启动时 exit 0；停止后 exit 1 |
| 端口冲突 | 第二个实例被拒绝启动 |
| 配置/数据/Key/会话/规则/用量权限 | 目录 0700；文件 0600 |
| 扫描所有服务日志 | 不含用户 Key、Provider Key、账号密码 |
| 备份 | 加密、0600、无明文凭据 |
| 恢复 | Provider、分组、Gateway Key、用户 Key、用量恢复；平台账号/模型开放/注册开关手动拷贝后登录与调用成功 |

**说明**：本次隔离回归使用 mock 上游，不是真实厂商。真实 DeepSeek 上游的完整业务闭环见 Phase 7 报告；Phase 8 没有再次调用真实厂商。

## 5. 生产就绪检查

### 5.1 文件权限

隔离环境（E2E）：

```text
~/.config/magpie                 0700
settings.json                    0600
users.json                       0600
sessions.json                    0600
caller-keys.json                 0600
providers.json                   0600
model-access.json                0600
registration.json                0600
usage.jsonl                      0600
~/.cache/magpie/log-v2.bin       0600（写入时）
```

实际生产（只读检查）：

```text
/root/.config/magpie             0700
users/sessions/caller-keys/
settings/gateway/stash/auto-picks 0600
plugin-icons.json                0644（非敏感图标缓存）
last-version                     0644（版本标记，非敏感）
```

生产未创建 `usage.jsonl`/`providers.json`/`model-access.json`/`registration.json`（尚无调用、供应商与开放模型；注册按“文件不存在”默认关闭）。

### 5.2 Provider、Key、用户、会话、模型规则持久化

E2E 在优雅重启、SIGKILL 异常重启后检查：

- 管理员 Session 仍有效；
- 用户 A/B API Key 仍可调用；
- Provider 与模型映射仍存在；
- 模型开放规则仍为 2；
- 用量与日志数字一致；
- `caller-keys.json`、`users.json`、`sessions.json`、`model-access.json`、`providers.json` 均在重启后保留。

生产本次由 Phase 7 换成 Phase 8 时也经历了一次重启：`sessions.json` 中 8 条会话仍在，page key 仍有效，管理员账号仍存在（`admin@magpie.local`，role admin）。

### 5.3 usage.jsonl 与索引缓存

E2E 的短历史（8 次调用）没有填满一个 512 行的缓存块，所以重启时没有 `log-v2.bin` 可删；删除清单为空，这是事实，不能声称“删除了已生成的缓存”。

因此另跑了针对性验证（`/tmp/phase8final/cache.cjs`，独立 HOME `/tmp/phase8cache`，600 次调用）：

```text
calls=600
cache_written=true
cache_mode=600
cache_bytes=13159
pages_identical_without_cache=true
cache_rewritten_after_rebuild=true
```

结论：

- 历史超过一个 512 行块时，`~/.cache/magpie/log-v2.bin` 会生成，权限 0600；
- 停止服务并删除该文件后重启，用户日志第 1 页与第 30 页内容与删前完全一致；
- 重建后缓存会再次写入。

短历史的 E2E 同样证明：没有缓存文件时，重启后用 `usage.jsonl` 重建出与重启前相同的用量和日志。

### 5.4 异常退出、重启与端口冲突

| 检查 | 结果 |
| --- | --- |
| SIGTERM | 退出码 0 |
| SIGKILL 后重启 | 通过；无需手工修复 |
| 同端口启动第二个实例 | 拒绝，现有实例不受影响 |
| 健康检查（运行中） | exit 0 |
| 健康检查（已停止） | exit 1，明确连接失败 |
| 启动后页面/Gateway 就绪 | `/portal/` 200 后 Gateway 仍可能稍慢；测试在每次启动/重启后都等待 Gateway 就绪 |

### 5.5 敏感日志

E2E 收集每次服务运行的 stdout/stderr，扫描：

- 用户 API Key 完整值；
- Provider Key；
- 管理员密码；
- 普通用户密码。

均未出现。错误日志中的上游错误仍会脱敏 `sk-…`、`gsk_…` 等模式，并限制长度（Phase 4 已有行为）。

### 5.6 反向代理与 `MAGPIE_PUBLIC_URL`

现有文档已覆盖（`docs/reference.md`）：

- Gateway 默认 `127.0.0.1:3425`，`MAGPIE_ADDR` 可改；
- 反向代理/容器部署设置 `MAGPIE_PUBLIC_URL=https://magpie.example.com`，页面与 CLI 显示该地址；
- 一个入口即可：代理把全部请求送到页面端口（3430），`/v1/…`、`/mcp/…` 等由页面端口转给 Gateway；从页面端口进来的 Gateway 路由按“另一台机器”处理，需要启用的 Gateway Key；
- 代理直接访问 Gateway 端口时，带转发头的 loopback 请求也需要 Gateway Key；`MAGPIE_TRUST_PROXY=1` 只适用于“代理自己认证客户端”的部署，不能替代 Key。

HTTPS Cookie 的额外要求（Phase 7/8 的判断）：

```text
MAGPIE_PUBLIC_URL 必须是 https://…
请求 Host 必须与该公共地址的 Host 完全一致
X-Forwarded-Proto 必须为 https
```

代理改写 Host、使用 HTTP 公共地址或没有配置公共地址时，Cookie 不会是 Secure。这是刻意保守：不把任意转发头当作 HTTPS。部署时需保持 Host 或配置正确的公共地址。

### 5.7 备份与恢复

`magpie backup` 覆盖：

- Providers、模型映射、Routing 分组；
- Settings；
- Gateway Keys 与平台用户 Key（同一 `caller-keys.json` 通过 `GatewayKeys` 导出）；
- 用量历史。

**不覆盖**（平台自身文件）：

- `users.json`（平台账号与密码哈希）；
- `model-access.json`（模型开放与用户授权）；
- `registration.json`（注册开关）；
- `sessions.json`（会话；恢复后可重新登录，不是必须备份）。

E2E 恢复流程：

1. `magpie backup <file>`，输入备份密码；文件 0600、加密、无明文 key；
2. 在新 HOME `magpie restore --no-agents --no-library`；
3. 手动拷贝 `users.json`、`model-access.json`、`registration.json` 并设为 0600；
4. 启动后管理员与用户 A/B 均可登录，两个用户 Key 均可调用，开放模型 2 个，分组与用量恢复。

这是当前运维缺口：**仅用 `magpie backup` 恢复到新机器会缺少平台账号、模型开放/授权和注册开关。** 生产备份流程必须把这三个文件与备份文件一起保存。不要备份 `sessions.json` 到不受控位置；如需保留登录态再单独加密保存。

### 5.8 注册默认与管理员初始化

- 没有默认管理员、没有默认密码；首个管理员只能通过 `magpie user add <email> --role admin` 创建（或等价 CLI），密码从 stdin/交互读取；
- `registration.json` 不存在时注册默认关闭；
- E2E 用 CLI 初始化 `admin@phase8.test`，随后管理员创建用户，注册开关默认关闭、打开、再关闭均符合预期；
- 生产 `registration.json` 不存在，`POST /api/auth/register` 返回 `403 {"error":"registration is closed"}`；
- 生产存在且仅存在一个管理员账号 `admin@magpie.local`（role admin，启用）。

### 5.9 健康检查、启动与故障排查

- `magpie healthcheck --addr 127.0.0.1:3425`：运行中 exit 0；停止时 exit 1 并输出连接失败原因；
- 端口被占用时新实例拒绝启动，不抢占、不静默换端口；
- 服务启动日志列出页面地址、网络地址、Gateway 地址；生产日志不含 page key 以外的凭据（page key 只出现在访问链接中，属于运维凭据）；
- 启动时页面端口可能先于 Gateway 就绪；自动化/探活应分别检查 `/portal/` 与 Gateway `/v1/models`，不要只等页面。

## 6. 测试结果（仓库目标 Go 1.26.8）

工具链：`/root/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.8.linux-amd64/bin/go`（`go version go1.26.8 linux/amd64`）。

| 命令 | 结果 |
| --- | --- |
| `go test -tags nogui ./internal/gateway -count=1 -timeout 20m -json` | **PASS，711.738s**，74684 条 test/subtest 事件，零 fail |
| `go test -tags nogui ./internal/gui -count=1 -json` | **PASS，167.44s**，4053 条事件，零 fail |
| `go test -tags nogui ./internal/usage -count=1` | **PASS，98.595s** |
| `go test -tags nogui ./internal/identity -count=1` | **PASS，0.558s** |
| `go test -tags nogui ./internal/cli -count=1`（生产停止、3425 空闲） | **PASS，39.891s** |
| `go vet -tags nogui ./...` | **exit 0** |
| `TestSearchQueueKeepsCodexAlive` 单独 `-count=5` | PASS，3.468s |
| `TestUserModelDisplayUsesTheRequestedID -count=3` | PASS，0.041s |
| `TestSecureCookiesBehindHTTPS` | PASS，0.09s |

### 关于 CLI 测试的环境依赖（如实记录）

第一次同时运行 `usage/identity/cli` 时，`internal/cli` 有 3 个失败：

```text
TestHealthcheckFindsTheRunningGatewayWithoutMagpieAddr
TestUpdateRestartsTheGatewayItFindsWithoutMagpieAddr
TestUpdateFindsAMagpieThatWroteNoRecord
```

原因不是代码回归，而是当时 Phase 7 生产实例正在 `127.0.0.1:3425` 上运行；这些 updater/healthcheck 测试会探测默认 3425，发现并选中了真实生产实例，而不是测试 fake：

```text
"the address Settings has is not where the fake serves: http://127.0.0.1:3425"
```

停止生产以部署 Phase 8 后，3425 空闲，同一命令 **PASS（39.891s）**。这说明：

- CLI 的 updater/healthcheck 测试假定默认 3425 没有其他 magpie；
- 在有生产实例的机器上跑这套测试，需要先停服务或隔离 `MAGPIE_ADDR`/HOME；
- 本次没有跳过或修改这些测试来“变绿”。

Gateway 全量耗时约 12 分钟，CI 超时必须 ≥20 分钟；本次没有使用工具层 600s 截断。

## 7. 生产部署与 smoke

| 项目 | 值 |
| --- | --- |
| 二进制 | `/tmp/magpie-phase8` |
| SHA-256 | `eac19d6aa9adbda4e420a9f7e2d8c4449dcf52d1dddd27d2aff230268aa486dd` |
| 源码一致性 | 用同一目标 Go 重新构建，SHA 完全相同 |
| 进程 | PID 296201，`magpie-phase8 web --lan --no-open --gateway` |
| 页面 | `http://127.0.0.1:3430` |
| Gateway | `http://127.0.0.1:3425` |
| page key | 未变（沿用既有环境变量） |
| 管理员账号/密码 | 未修改 |
| Provider/模型 | 未新增、未修改（生产仍无 Provider） |

生产 smoke：

| 检查 | 结果 |
| --- | --- |
| `/portal/` | 200 |
| 匿名 `/` | 302 → `/portal/login?next=%2F` |
| `/portal/login` | 200，登录表单存在 |
| 匿名 `/portal/console`、`/portal/models` | 302 → 登录 |
| page key 链接 | 303 设置 `magpie_web_3430` Cookie，跟随跳转 200 |
| 带 page key Cookie `/api/state` | 200 |
| 带 page key Cookie `/api/admin/users`、`/api/admin/registration` | 401（page key 未增权） |
| 匿名 `/api/admin/*`、`/api/user/keys` | 401 |
| 匿名 `POST /api/auth/register` | 403，注册关闭 |
| 匿名 `POST /api/auth/login` 空 body | 401 |
| Gateway `/v1/models`（本机 loopback） | 200，空目录 |
| Gateway `/v1/models`（LAN 地址） | 401（Share on LAN 需要 Gateway Key，符合预期） |
| `healthcheck --addr 127.0.0.1:3425` | exit 0 |
| `healthcheck --addr 127.0.0.1:3456` | exit 1，连接失败 |
| 浏览器 | 门户加载无 JS 错误，匿名 `/` 跳登录 |

**生产管理员登录/smoke 的限制**：本轮没有重新执行生产管理员登录。生产管理员密码不在本会话可用范围内，且按约束不能重置或改写生产账号；因此生产侧只验证了匿名边界、page key 兼容与管理员账号存在。同一最终二进制的管理员登录、完整后台、Session 重启持久化已在隔离 E2E 与 GUI 测试中验证；Phase 7 也已在生产用该管理员账号完成登录 smoke。此项是验证覆盖限制，不是功能失败。

截图：

- `/tmp/phase8-prod-portal.png` — 生产门户
- `/tmp/phase8-prod-login.png` — 生产登录页
- `/tmp/phase8-user-a-console.png` — 用户 A 控制台：3 次调用、2 成功、1 失败、14 tokens；排行与日志显示完整平台模型 ID
- `/tmp/phase8-user-b-console.png` — 用户 B 的独立日志与用量

截图中不含完整 API Key、Provider Key、Session Token 或管理员密码。

## 8. 真实生产 / 隔离环境 / mock 的区分

| 类别 | 内容 |
| --- | --- |
| 真实生产 | 端口 3430/3425，既有 page key 与管理员账号；只做部署与只读 smoke。生产无 Provider、无开放模型、无用户 Key。 |
| 隔离环境（真实二进制、mock 上游） | `/tmp/phase8home`、3520/3521/3524/3525，mock 上游 `127.0.0.1:39003`；覆盖 Provider 配置、开放模型、用户、Key、流式/非流式、拒绝、关闭/重开、重启、备份恢复、权限、日志。 |
| 真实上游（Phase 7，非本轮） | DeepSeek `api.deepseek.com`，`deepseek-flash`/`deepseek-v4-pro`，Anthropic Messages 上游，OpenAI 兼容入口；真实非流式/流式调用、模型映射、关闭/重开、两名用户隔离。Phase 8 未重复调用真实厂商。 |
| 代码/单元/全量测试 | 本机仓库源码，目标 Go 1.26.8；不依赖生产数据。 |

## 9. 风险与遗留

1. **备份缺口（运维必须知道）**：`magpie backup` 不含 `users.json`、`model-access.json`、`registration.json`。只恢复备份会丢平台账号、模型开放/授权和注册开关。恢复步骤必须同时拷贝这三个文件（0600）。`sessions.json` 可不备份，恢复后重新登录。
2. **反向代理 Secure Cookie 条件严格**：只信任与 `MAGPIE_PUBLIC_URL` Host 完全一致且 `X-Forwarded-Proto: https` 的请求。代理改写 Host、使用 HTTP 或未配置公共地址时 Cookie 不 Secure。当前没有“可信代理地址/CIDR”配置项；若代理必须改写 Host，需要后续设计显式的可信代理配置。
3. **CLI updater/healthcheck 测试依赖 3425 空闲**：有生产实例时这套测试会误选生产实例并失败。CI 应在隔离环境跑，或先停本机服务。不是产品缺陷，但会影响“所有测试一起跑”。
4. **缓存验证边界**：E2E 短历史没有生成 `log-v2.bin`；针对性的 600 次调用测试证明了生成、0600、删除后重建、页面一致。更长历史/更大文件的性能仍未在本轮压测。
5. **生产管理员登录未复跑**：密码不在本会话，且未重置生产账号。管理员 Session 功能已由隔离 E2E、GUI 测试和 Phase 7 生产 smoke 覆盖。
6. **生产仍无 Provider/模型**：真实模型广场与调用需要管理员自行配置 Provider 并开放模型；Phase 8 没有向生产写入测试 Provider。
7. **注册开关是文件状态**：`registration.json` 缺失即关闭；备份/迁移时容易遗漏，已列入备份缺口。
8. **Gateway 全量 ~12 分钟**：常规 CI 的 10 分钟超时会截断，需要单独的任务或更长超时。
9. **日志脱敏是模式扫描**：已验证常见 Key/密码模式与完整凭据不出现在服务日志；不代表对所有自定义敏感文本的完备检测。
10. **模型统计数据语义**：被拒绝的调用仍会以 0 token 出现在该用户自己的模型统计中（Phase 7 已记录）；这是该用户的请求事实，不是跨用户泄漏。

## 10. 完成标准对照

| 标准 | 结果 |
| --- | --- |
| P1 受信代理 HTTPS 判断正确 | 通过：TLS/公共 URL + Host + 头；配置错误不 Secure |
| P2 成功与被拒绝模型展示一致 | 通过：统一 `displayModel`，保留 `model`/`requested`，历史兼容 |
| Gateway 队列 flake 修复且不弱化测试 | 通过：仅调时序余量，断言不变；单独与全量均通过 |
| 目标 Go 1.26.8 相关测试 | 通过：usage/identity/cli |
| Gateway 全量实际结果 | PASS，711.738s，零失败，非超时 |
| GUI 测试 | PASS，167.44s，零失败 |
| go vet | exit 0 |
| 权限/持久化/重启/端口冲突/日志脱敏 | 通过（隔离 E2E + 生产只读检查） |
| usage.jsonl 与索引缓存恢复 | 通过：JSONL 重建；600 次调用下缓存生成、删除、重建一致 |
| 备份/恢复 | 通过（含手动拷贝平台文件的已测试流程）；缺口已记录 |
| 注册默认关闭/管理员初始化 | 通过：无默认密码，CLI 首管理员，默认关闭 |
| 最终隔离回归（管理员/Provider/模型/用户/Key/流式/拒绝/用量/重启） | 通过，7.065s |
| 生产部署与 smoke | 通过；page key/管理员/Provider 未改 |
| 未提交、未推送 | 已遵守 |

## 11. Git 状态

```text
branch: remove-partners
HEAD:   e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e
工作树: 71 项（40 modified、31 untracked）
```

未执行 checkout/reset/clean/stash/cherry-pick/commit/push。Phase 1–7 的改动全部保留；Phase 8 的源码改动仅列于第 2 节。

## 12. 证据位置

- `/tmp/phase8verify/evidence.json` — 最终隔离 E2E 证据
- `/tmp/phase8verify/verify.cjs` — E2E 脚本
- `/tmp/phase8final/run.cjs`、`/tmp/phase8final/run.log` — 最终运行器与结果
- `/tmp/phase8final/cache.cjs` — 600 次调用的索引缓存验证
- `/tmp/p8gateway.json` — 目标 Go Gateway 全量 JSON
- `/tmp/p8guifull.json` — 目标 Go GUI 全量 JSON
- `/tmp/phase8-user-a-console.png`、`/tmp/phase8-user-b-console.png`
- `/tmp/phase8-prod-portal.png`、`/tmp/phase8-prod-login.png`

等待验收。