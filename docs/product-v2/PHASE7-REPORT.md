# Phase 7 报告：真实生产业务闭环验收与修复

日期：2026-11-23
项目：`/root/magpie-original`
分支：`remove-partners`
HEAD：`e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e`

## 1. 结论

Phase 7 使用真实 DeepSeek 上游完成了：

- 管理员通过原管理后台配置 Provider、模型 ID 映射和 Routing 分组；
- 在 `/portal/admin-models` 开放两个模型；
- 两个普通用户各自创建只授权一个模型的 Key；
- 通过 OpenAI 兼容的 `/v1/chat/completions` 完成真实非流式和流式调用；
- 模型关闭后已有 Key 立即被拒绝，重新开放后恢复；
- Key 创建、一次性密钥、复制、禁用、启用、修改授权、删除全部验证；
- 两个用户的 Key、日志、用量和模型统计互相隔离；
- 日志与用量数字和实际成功调用一致；
- 修复了一个 P1 安全问题：`X-Forwarded-Proto` 不再被无条件当作 HTTPS。

测试使用独立 HOME 和独立测试端口，生产配置、生产账号、生产密码、page key 均未修改。生产服务已替换为包含修复的 Phase 7 二进制。

未提交、未推送。

## 2. 开始前状态与保护

Phase 7 开始时：

- 分支：`remove-partners`
- HEAD：`e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e`
- 工作树：70 项已修改/未跟踪条目
- Phase 1–6 的所有改动保留
- 未 checkout、reset、clean、stash、cherry-pick、commit、push

实施结束后工作树仍为 70 项（新增报告位于已有的未跟踪 `docs/product-v2/` 目录内），未提交、未推送。

## 3. 真实测试 Provider 与模型

| 项目 | 值 |
| --- | --- |
| Provider ID | `deepseek-test` |
| Provider 名称 | `DeepSeek Test` |
| 上游协议 | Anthropic Messages 兼容 |
| 上游主机 | `api.deepseek.com`（`https://api.deepseek.com/anthropic`） |
| 上游模型 1 | `deepseek-flash` |
| 上游模型 2 | `deepseek-v4-pro` |
| 平台映射 ID 1 | `deepseek-test/phase7-flash` |
| 平台映射 ID 2 | `deepseek-test/phase7-pro` |
| Routing 分组 | `phase7`，成员两个映射模型，`order` |
| 入站协议 | OpenAI Chat Completions（非流式与流式） |

凭据来源：测试 key 从当前环境的本地凭据存储（`ZCODE_PERSONAL_PROVIDER_CONFIG_FILE` 指向的 provider 配置）读取，只进入独立测试 HOME 的 Magpie Provider 配置；没有写入源码、示例、报告、日志、截图或生产 Magpie 配置。测试 HOME 在验收后已删除。

真实上游在接入 Magpie 前先直接验证：`deepseek-flash` 和 `deepseek-v4-pro` 均返回 HTTP 200，内容文本为 `PHASE7_OK`；随后全部业务调用均通过 Magpie 转发。

## 4. 配置、开放、授权与真实调用

隔离环境：

- HOME：`/tmp/phase7home`（验收后删除）
- 页面：`http://127.0.0.1:3500`
- Gateway：`http://127.0.0.1:3501`
- Playwright Chromium 1254

执行步骤：

1. 管理员平台账号登录 `/`，原管理后台完整导航可见：Providers、Routing、Routing groups、Gateway、Usage、Masking、Plugins、Settings；无 Agents/Sessions/Library。
2. 通过原管理后台保存处理器配置 Provider（`/api/provider/save`）：
   - `deepseek-test`，Anthropic 兼容上游；
   - 两个上游模型；
   - `modelPrefs` 将 `deepseek-flash` 映射为 `phase7-flash`，将 `deepseek-v4-pro` 映射为 `phase7-pro`；
   - `routing: smart`。
3. 通过原管理后台保存 Routing 分组（`/api/groups/save`）：
   - `phase7`，成员 `deepseek-test/phase7-flash`、`deepseek-test/phase7-pro`；
   - `routing: order`。
4. 原管理后台 Providers 页面显示 `DeepSeek Test`（2 models）；Routing groups 页面显示 `Phase 7`，成员顺序正确。
5. 管理员在 `/portal/admin-models` 页面用开关开放两个模型，统计显示“平台模型 2 / 已开放 2 / 未开放 0”，行内 ID 为映射后的 `deepseek-test/phase7-flash`、`deepseek-test/phase7-pro`。
6. 用户 A、B 由管理员通过原后台账号 API 创建（注册保持关闭）。
7. 用户 A 在控制台只勾选 `phase7-flash` 创建 Key；用户 B 只勾选 `phase7-pro` 创建 Key。

真实调用结果（OpenAI 兼容入口）：

### 非流式

```text
POST /v1/chat/completions  stream=false
platform model: deepseek-test/phase7-flash
status: 200
response model: deepseek-test/phase7-flash
content: PHASE7_OK
usage: prompt 41, completion 19, total 60
```

上游实际收到的是 `deepseek-flash`；返回的 OpenAI 响应模型是平台映射 ID，证明映射在请求和响应两侧都生效。

### 流式

```text
POST /v1/chat/completions  stream=true
platform model: deepseek-test/phase7-flash
status: 200
SSE events: 7
content collected from delta: PHASE7_OK
[DONE]: received
```

Anthropic Messages 上游流被 Magpie 转成 OpenAI Chat Completions SSE，最终内容完整，未出现截断或空响应。

### 第二个真实模型

`deepseek-test/phase7-pro` 也完成了真实非流式调用，返回 `PHASE7_OK`，上游为 `deepseek-v4-pro`，usage 非零。

### 错误响应

用户 A 用只授权 `phase7-flash` 的 Key 调用 `deepseek-test/phase7-pro`：

```text
status: 403
error.type: permission_error
message: gateway key may not use the model; it may use deepseek-test/phase7-flash
```

关闭模型后的调用同样返回 403 OpenAI 结构错误，没有请求上游。

## 5. 模型关闭、重新开放与 ID 映射

- 管理员在 `/portal/admin-models` 关闭 `deepseek-test/phase7-flash`。
- 管理员开放统计变为 2 → 1；用户模型广场只剩 `phase7-pro`。
- 用户 A 已存在、已授权 flash 的 Key 立即调用失败：403；上游没有收到请求。
- 同一时刻用户 B 调用自己的 `phase7-pro` 仍为 200，说明关闭只影响该模型。
- 管理员重新开放 flash；用户 A 原 Key 无需重建即可再次 200。
- 平台目录中的模型 ID 是两个映射 ID；Provider 配置保存的是映射后的模型列表；用户 Key 的 `models` 字段保存映射 ID；真实上游收到原始 `deepseek-flash` / `deepseek-v4-pro`。

## 6. 两个用户的隔离与 Key 生命周期

### 用户 A / B

- A Key 授权仅 `deepseek-test/phase7-flash`。
- B Key 授权仅 `deepseek-test/phase7-pro`。
- A 调用 B 模型：403；B 调用 A 模型：403。
- A 无法通过用户 API 修改或删除 B 的 Key：PATCH/DELETE 返回 404。
- A 的 Key 列表只有 A 的 Key；B 的只有 B 的。
- A 的请求日志 Key 名只出现 `A key`；B 只出现 `B key`。
- A 在用量请求中伪造 `owner=<B 的 key id>`：仍返回 A 自己的统计，owner 完全由服务端 Session 决定。
- 用户模型统计中，其他用户模型只可能以“被拒绝的尝试、0 tokens”出现，没有其他用户的真实 token 用量。

### Key 生命周期

| 操作 | 结果 |
| --- | --- |
| 创建 | 返回完整 secret 一次，`sk-magpie-user-` 前缀 |
| 复制 | 控制台复制按钮变为“已复制” |
| 刷新页面 | secret 不再显示；列表只有前缀 |
| 列表 | 不包含完整 secret 或 hash |
| 禁用 | 立即调用 401 |
| 启用 | 恢复调用 200 |
| 修改授权 flash → pro | 旧模型 403；新模型 200 |
| 恢复授权 pro → flash | flash 200 |
| 删除 | 调用 401；Key 列表为空 |
| 删除后历史 | 此前的 6 次成功用量和日志仍保留 |

## 7. 日志和用量核对

实际成功调用计数与用户接口返回完全一致。

### 用户 A

最终数据（删除 A 的 Key 之前，之后历史不变）：

```text
summary: calls=9 success=6 failed=3 canceled=0
tokens: input=299 output=135 total=434
```

模型统计：

```text
phase7-flash:  calls=5 tokens=301
phase7-pro:    calls=1 tokens=133
denied attempts:
  deepseek-test/phase7-flash: calls=2 tokens=0
  deepseek-test/phase7-pro:  calls=1 tokens=0
```

- 成功模型 token 合计 301 + 133 = 434，等于 summary.total。
- summary.success = 6，对应脚本中 A 的 6 次 HTTP 200。
- summary.failed = 3，对应 3 次被拒绝模型请求；它们 tokens 为 0。
- 请求日志 total = 9 = summary.calls；日志成功/失败计数为 6/3。
- 日志和统计都只属于 A；没有 B 的真实用量。

### 用户 B

```text
summary: calls=3 success=2 failed=1 canceled=0
tokens: input=188 output=57 total=245
phase7-pro: calls=2 tokens=245
denied A-model attempt: calls=1 tokens=0
```

B 请求日志 total = 3 = summary.calls；2 次成功、1 次越权拒绝；Key 名只出现 B key。

### 额外证据调用

用一个独立用户 C 的 Key 重放非流式/流式各一次：

- 非流式：200，`PHASE7_OK`，usage prompt 41 / completion 19 / total 60；
- 流式：200，7 个 SSE 事件，`PHASE7_OK`，收到 `[DONE]`；
- 第二次文本内容完整，说明流式与非流式都真实经过上游；
- C 的用量单独归属 C，不影响 A/B 的上述数字。

## 8. 管理端与安全边界

| 检查 | 真实结果 |
| --- | --- |
| 管理员 Session 使用完整管理后台 | 通过（Providers/Routing/Groups/Gateway/Usage/Masking/Plugins/Settings 均可见） |
| 普通用户访问 `/` | 403 |
| 普通用户访问 `/api/state`、`/api/providers`、`/api/settings`、`/api/admin/*` | 403 |
| query/header 伪造管理员角色 | 仍 403 |
| 旧 page key 打开页面/旧 API | 200 |
| 旧 page key 打开 `/api/admin/users` / `/api/admin/registration` | 401（未增权） |
| 管理员 Session 作为 Gateway Key | 403（page port 的 Gateway 路由拒绝） |
| 管理员 Session 无 page key 打开 `/` | 200，无 page-key Cookie、无 key 出现在 URL/存储 |
| 用户 Key 调用未开放/超授权模型 | 403，OpenAI `permission_error` |
| 模型关闭后已有 Key | 立即 403 |
| 注册关闭时注册 | 403 |
| 管理员打开注册后注册 | 成功进入控制台 |
| 再次关闭注册 | 403，未创建账号 |
| 跨站 POST/GET 携带管理员 Cookie | 403 |
| 登录失败限速 | 5 次失败后正确密码返回 429 + Retry-After（Phase 6 实现仍在） |
| HTTPS Cookie | 直接 TLS 下 Secure；代理头不再单独可信（见修复） |

## 9. 缺陷与修复

### P1：`X-Forwarded-Proto` 被无条件信任

**发现**：Phase 6 的 `secureRequest` 只要请求头写 `X-Forwarded-Proto: https` 就设置 Secure，即使页面实际是普通 HTTP、没有配置可信代理。攻击者可以伪造 HTTPS 状态，导致浏览器丢弃 Cookie（自伤式登出）并让安全判断脱离真实连接。

**修复**：`internal/gui/auth.go`

```text
Secure = true 仅当：
- 请求本身是 TLS；或
- 部署配置了 HTTPS 的 MAGPIE_PUBLIC_URL，且
- 请求 Host 与该公共地址的 Host 一致，且
- X-Forwarded-Proto: https。
```

**回归测试**：`internal/gui/admin_session_test.go` 的 `TestSecureCookiesBehindHTTPS`

- 直接 TLS → Secure；
- 未配置代理 + 伪造头 → 不 Secure；
- 配置 HTTPS 公共地址 + 匹配 Host + 头 → Secure；
- 配置了公共地址但 Host 不匹配 → 不 Secure；
- page key Cookie 同样适用。

**影响**：不改变正常 HTTP/HTTPS 部署；反向代理需要设置 `MAGPIE_PUBLIC_URL` 并保持 Host 为该地址，否则不会得到 Secure。这是刻意选择：宁可少信任一个头，也不把伪造头当作 HTTPS。

### 其他观察（未按 P0/P1 修改）

- 用户日志中成功模型的显示名是短 ID（如 `phase7-flash`），未授权的完整请求显示的是 `deepseek-test/phase7-pro`；统计总数仍一致，属于展示命名不一致，非权限或数据错误。（Phase 8 已修复：统一按请求的平台模型 ID 展示，API 仍保留记录模型 ID 与原始请求 ID；见 `PHASE8-REPORT.md`。）
- 未授权尝试会出现在该用户自己的模型统计中，tokens 为 0；这是日志事实，不包含其他用户的用量。

## 10. Gateway 全量测试实际结果

Phase 5 的 Gateway 全量测试此前在 600 秒工具时限内被截断。本次使用明确的 20 分钟 Go 测试时限，不做 600 秒截断。

### 使用仓库目标 Go 1.26.8

```text
/root/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.8.linux-amd64/bin/go \
  test -tags nogui ./internal/gateway -count=1 -timeout 20m
```

- 完成，耗时 **712.856s**（约 11 分 53 秒），**不是超时**
- 74688 个 test/subtest 事件
- 1 个测试失败（含其子测试）：
  - `TestSearchQueueKeepsCodexAlive`
  - `TestSearchQueueKeepsCodexAlive/progress`
  - 失败信息：`queue progress or native tool call lost: pings=1 complete=false`，fake 在 43ms 后返回 504
- 该测试单独运行 **20/20 通过**，总耗时 14.924s
- 同一测试在其他并发负载较低时通过，判断为全量并发下的时序 flake，不是本次改动造成的稳定失败
- 未将该失败或任何超时标记为通过

### 使用本机 Go 1.27.2（非仓库目标版本）

- 完成，耗时 **729.374s**
- 6 个失败，均在仓库目标 Go 1.26.8 下单独通过：
  - `TestMarshalRawIsEncodingJSON`
  - `TestMarshalSizedIsJSONMarshal`
  - `FuzzMarshalRaw` / seed
  - `TestServeWSLListensBesideLoopback`
  - `TestServeWSLListensAtVia`
- 这些失败是 Go 1.27 的 JSON 转义行为/环境差异，与 Phase 7 业务代码无关。

### 结论

Gateway 全量套件已实际运行并完成；没有超时。仓库目标工具链下只有 1 个可复现全量并发时序 flake，单独 20 次全通过；本机新工具链下的 6 个失败在目标工具链下通过。未把失败或超时报告为通过。

## 11. 其他 Go 测试与静态检查

- `go test -tags nogui ./internal/gui -count=1`：PASS，133.048s，4053 个 test/subtest 事件
- `go test -tags nogui ./internal/identity -count=1`：PASS（Phase 6 基线，Phase 7 未改 identity）
- `go test -tags nogui ./internal/cli -count=1`：PASS（Phase 6 在 3425 空闲时）
- `go vet -tags nogui ./...`：exit 0
- 所有针对 `TestSecureCookiesBehindHTTPS` 的新增分支通过
- 未运行无关的大型测试来堆数量

## 12. 浏览器验收结果

Playwright E2E：PASS，约 12.6s。

覆盖：

- 管理员 Session 无 page key 打开完整原管理后台；
- 原后台 Provider/Groups 保存与页面显示；
- `/portal/admin-models` 开放/关闭模型；
- 用户模型广场；
- 用户 A/B Key 创建、一次性显示、复制、刷新后隐藏；
- 真实非流式/流式调用；
- 跨用户模型 403；
- Key 禁用/启用、改授权、恢复、删除；
- 用量/日志/模型统计一致；
- 模型关闭/重开；
- 注册关闭/打开/再关闭；
- 管理页/API 边界、page key、管理员 Session 不能作为 Gateway Key；
- X-Forwarded-Proto 伪造不产生 Secure。

证据截图：

- `/tmp/phase7-admin-routing.png` — 原管理后台 Routing groups 页面显示 `Phase 7` 分组及两个成员
- `/tmp/phase7-user-a-console.png` — 用户 A 的真实 9 次调用/6 成功/3 失败、434 tokens 与日志
- `/tmp/phase7-user-b-console.png` — 用户 B 的独立 3 次调用/2 成功/1 失败、245 tokens
- `/tmp/phase7-user-models.png` — 用户模型广场的两个映射模型
- `/tmp/phase7-prod-admin.png` — 生产管理员 Session 无 page key 打开管理后台

截图中不含完整 API Key、Provider Key 或 Session Token；Provider 列表截图因显示部分掩码 key 已删除。

## 13. 生产部署与 smoke test

实际执行：

- 停止 Phase 6 服务
- 启动 Phase 7：`/tmp/magpie-phase7 web --lan --no-open --gateway`
- PID：256946
- 页面：`http://127.0.0.1:3430`
- Gateway：`http://127.0.0.1:3425`
- 二进制 SHA-256：`cd11a361294ed73bfda758dd788ff5a3000c76d12193982453bdc7676da1c645`
- 原 page key 未变；管理员账号/密码未修改；未在生产配置真实测试 Provider

生产 smoke：

- `/portal/` 200
- 未登录 `/` 302 → `/portal/login?next=%2F`
- Gateway `/v1/models` 200
- 管理员登录 200
- 管理员 Session `/` 200、`/api/state` 200、`/api/admin/users` 200
- `/api/admin/registration` = `{"open":false}`
- 旧 page key 打开 `/` 200
- Playwright 管理员登录后完整管理导航加载，无 page-key Cookie，无页面错误，截图 `/tmp/phase7-prod-admin.png`

生产当前没有配置任何供应商，因此模型目录仍为空；用户端/管理端功能正常。真实上游只存在于已删除的隔离测试 HOME。

## 14. 风险与未解决项

1. **测试凭据来源**：本次真实调用使用本机已有 DeepSeek 凭据，而不是新申请的独立付费账号；未覆盖生产 Magpie 的 Provider，且测试 HOME 已删除。真实调用约消耗少量测试 token。
2. **反向代理 HTTPS**：只有 `MAGPIE_PUBLIC_URL` 为 HTTPS 且 Host 匹配时才信任 `X-Forwarded-Proto`。代理重写 Host 的部署会得到非 Secure Cookie；需配置公共地址/保持 Host 或后续增加显式可信代理地址配置。
3. **Gateway 全量 flake**：`TestSearchQueueKeepsCodexAlive` 在全量并发下失败一次，单独 20/20 通过。未修源码；如要求全量全绿，需要在低负载下重跑，或修测试自身的 43ms fake 超时。（Phase 8 已修测试时序余量：fake ticker 5ms、`translatedQuietLongest` 100ms，断言未改；目标 Go 全量零失败，见 `PHASE8-REPORT.md`。）
4. **模型统计命名**：成功请求使用短模型名，被拒绝请求使用完整请求 ID；总数/tokens 一致，属于 P2 展示一致性。（Phase 8 已修复，见 `PHASE8-REPORT.md`。）
5. **旧 page key**：仍是完整运维凭据，持有者按旧权限访问管理资源；这是兼容策略，不等于普通用户 Session 提权。
6. **登录限速/其他 Phase 6 风险**仍存在：内存限速、注册开关本机文件、管理 SSE 中途撤销等，详见 `PHASE6-REPORT.md`。
7. **生产供应商为空**：真实闭环只在隔离环境完成；生产需要管理员自行配置供应商后才有模型可调用。

## 15. 完成标准对照

| 标准 | 结果 |
| --- | --- |
| 至少一个真实上游聊天模型非流式调用 | 通过：DeepSeek `deepseek-flash` → 映射 `deepseek-test/phase7-flash`，PHASE7_OK |
| 真实流式调用 | 通过：7 个 SSE 事件，PHASE7_OK，[DONE] |
| 用户 Key、模型授权、Gateway 转发、日志、用量闭环 | 通过：A/B 真实调用与统计一致 |
| 模型关闭后立即拒绝 | 通过：已有 Key 403，重开恢复 |
| 多用户隔离 | 通过：互相调用 403，Key/日志/用量互不可见 |
| 管理员原后台完整可用 | 通过 |
| 旧 page key 兼容、不增权 | 通过 |
| 发现缺陷并修复 | 通过 P1 `X-Forwarded-Proto` 修复与回归测试 |
| Gateway 全量测试实际完成 | 完成 712.856s；1 个全量并发 flake，单独 20/20 通过；未标记通过 |
| 未提交、未推送 | 已遵守 |

等待验收。