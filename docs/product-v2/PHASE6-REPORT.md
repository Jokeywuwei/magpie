# Phase 6 报告：管理员身份统一与管理后台接入

日期：2026-11-23
项目：`/root/magpie-original`

## 1. 结论

管理员现在可以用平台账号登录门户，并从门户直接进入原有完整管理后台；访问 `/` 不再需要 page key。普通用户会话不能打开管理页面或管理 API；未登录访问 `/` 会跳转 `/portal/login?next=%2F`。旧 page key（Cookie、`?k=` 链接、粘贴表单）继续按原权限工作，没有被 Session 认证扩大。

Phase 6 已部署到 3430，Gateway 在 3425；未提交、未推送。

## 2. 开始前保护与 Git 状态

实际记录（开始实施前）：

- 分支：`remove-partners`
- HEAD：`e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e`
- 工作树：64 项已修改/未跟踪条目，包含 Phase 1–5 的现有改动和既有无关改动
- 未执行 checkout、reset、clean、stash、cherry-pick、commit 或 push
- Phase 5 的 modelaccess、用户 Key、用量、门户和管理页改动均保留

实施完成后的状态：

- 分支：`remove-partners`
- HEAD：`e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e`
- 工作树：70 项（本报告写入后为 71 项），其中 Phase 6 新增了 `internal/identity/registration.go`、`internal/gui/loginlimit.go`、`internal/gui/admin_session_test.go` 等
- 未提交、未推送
- 未修改生产账号密码、page key 或既有供应商配置

## 3. 修改文件及实现内容

| 文件 | 内容 |
| --- | --- |
| `internal/gui/web.go` | 增加管理员 Session 进入管理页和全部页内管理 API 的分支；未登录 `/` 跳转门户登录；普通用户 403；page key 优先保持兼容；page key Cookie 在 HTTPS 下 Secure |
| `internal/gui/auth.go` | 登录失败限速；注册开关检查；`/api/admin/registration` 读写；Session Cookie 的 Secure 判断（TLS 或 `X-Forwarded-Proto: https`）；登出清理 Cookie |
| `internal/gui/loginlimit.go` | 内存登录限速：同一账号+来源 15 分钟 5 次失败，同一来源 15 分钟 20 次；成功清除该账号+来源计数；429 + `Retry-After` |
| `internal/gui/portal.go` | 公开门户路径注释更新；管理员模型页继续只允许管理员 Session |
| `internal/gui/assets/portal/index.html` | 控制台管理员入口增加“新用户注册”开关 |
| `internal/gui/assets/portal/portal.js` | 管理员注册开关的读取/切换；登录后的 `next` 支持 `/` 管理页；普通用户从 `/` 登录后回落控制台；429/注册关闭错误文案 |
| `internal/gui/assets/portal/portal.css` | 管理员入口卡片内边距和注册开关行的样式，复用现有视觉变量 |
| `internal/identity/registration.go` | `registration.json` 注册开关，默认关闭，0600；`RegistrationOpen` / `SetRegistrationOpen` / `ErrRegistrationClosed` |
| `internal/identity/identity.go` | 角色变更先撤销该账号全部 Session 再写入新角色；密码修改先撤销 Session 再写入新密码（失败也不会留下旧权限 Session） |
| `internal/cli/user_cli.go` | 新增 `magpie user registration [on|off]`，无参数显示当前状态 |
| `internal/cli/settings_parity_test.go` | 登记管理员注册 API 与 CLI 的对应关系 |
| `internal/gui/web_test.go`、`web_app_test.go`、`web_lazycat_test.go`、`portal_test.go`、`auth_test.go` | 适配未登录 `/` 的登录跳转、普通用户 403、page key 兼容、注册开关与 Session 撤销 |
| `internal/gui/admin_session_test.go` | 新增管理 Session 边界、登出/禁用/改密/降级、page key 兼容、CSRF、Secure Cookie、登录限速测试 |
| `internal/gui/loginlimit_test.go` | 登录限速的账号+来源与来源维度、窗口重置、成功清除测试 |
| `internal/identity/registration_test.go` | 注册开关默认关闭、持久化、0600、损坏文件失败关闭、角色变更撤销 Session |
| `internal/cli/user_cli_test.go` | CLI 注册开关测试 |

Phase 1–5 的其余工作树改动未触碰。

## 4. 认证边界

| 入口 | 认证 |
| --- | --- |
| `/portal/`、登录、注册、公开门户文件 | 公开；登录/注册页面自己检查 Session |
| `/portal/console`、`/portal/models` | 平台 Session |
| `/portal/admin-models` | 平台 Session 且 `role=admin` |
| `/api/auth/*` | 公开账号端点；写操作有 Origin/Fetch Metadata 防护；登录有限速；注册受开关控制 |
| `/api/user/*` | 平台 Session；owner 从服务端 Session 取得，忽略 query/header 中的 owner |
| `/api/admin/*` | 平台 Session 且 `role=admin`；page key 不是账号，单独 page key 返回 401 |
| 管理页 `/` 及其静态文件 | **有效 page key Cookie / `?k=` / 粘贴表单** 或 **管理员平台 Session** |
| 管理页 API：`/api/state`、`/api/providers`、`/api/settings`、`/api/usage`、`/api/groups` 等 | 同上；普通用户 Session 不通过 |
| Gateway：`/v1/*`、`/v1beta/*`、`/mcp/*` | Gateway Key；平台 Session 不转化为 Gateway 权限 |
| 远端管理 API：`/v1/magpie/admin/*` | Gateway 管理 Key；本阶段未改为平台 Session |
| 插件页面及其服务 | 插件自己的页面 token；不属于管理员 API |
| 桌面应用自身窗口 | 本机进程内访问，保持原有本地信任模型；本阶段针对 Web 浏览器入口 |

管理页的 `/api/*` 仍由 `webOwnPage` 检查 Origin 与 Fetch Metadata；管理员 Session 走的是同一页面处理器，没有绕过 CSRF 检查。

页面没有新增管理 WebSocket。GUI 的长流式入口是 `/api/uninstall` 的 SSE，它在请求开始时检查管理权限；权限在流进行中被撤销时，已开始的流不会自动中断（见风险）。插件页面的 WebSocket/SSE 走插件 token，不是管理员 API。

## 5. 旧 page key 兼容策略

- `magpie_web_<port>` Cookie、启动链接的 `?k=`、`/_magpie/sign-in` 表单均保留。
- 有效 page key Cookie 的优先级高于 Session 角色：page-key 浏览器看到的行为与 Phase 5 一致。
- 单独 page key **不能**打开 `/api/admin/users` 或 `/api/admin/registration`；这两个仍是平台管理员 Session 的 API（401）。
- 管理员 Session 不会写入或读取 page key，不会把 page key 放进 URL、页面 JavaScript、localStorage 或 sessionStorage。
- page key 仍是兼容用的完整运维凭据：普通用户如果另外持有 page key，仍可按旧规则访问管理页/旧管理 API；这不是普通用户 Session 提权，而是 page key 的既有权限。建议后续迁移完成后再退役 page key。
- page key Cookie 在直接 TLS 或反向代理 `X-Forwarded-Proto: https` 下带 `Secure`；HttpOnly、SameSite=Lax 保持不变。

## 6. 管理员账号安全

### 登录限速
- 同一账号 + 来源：15 分钟 5 次失败。
- 同一来源：15 分钟 20 次失败。
- 第 6 次即使密码正确也返回 429 和 `Retry-After`；成功登录清除该账号+来源的失败计数。
- 限速在内存中，重启清空；不记录密码、邮箱或 Session Token。
- 按账号+来源计数，避免攻击者用 5 次失败永久锁死某个已知管理员账号；其他来源仍可登录。

### 注册开关
- 默认关闭。没有 `registration.json` 时注册关闭。
- 首管理员仍通过 `magpie user add <email> --role admin` 创建，不依赖注册。
- 管理员在门户控制台、`POST /api/admin/registration` 或 `magpie user registration on|off` 控制。
- `registration.json` 0600；损坏/不可读按关闭处理。
- 注册请求提交 `role` 字段会被 JSON 解码拒绝（400），不能伪造管理员。

### Session 生命周期
- 登出：删除服务端 Session。
- 禁用：删除全部 Session，之后 SessionUser 也拒绝。
- 删除账号：删除 Session。
- 修改密码：先撤销全部 Session，再写新密码；旧 Cookie 立即无效。
- 角色变更：先撤销全部 Session，再写新角色；降级和升级都不会让旧浏览器继续使用旧角色。
- Session 过期：沿用 7 天 TTL。
- 最后一名启用管理员：删除、禁用、降级仍由既有 `ErrLastAdmin` 保护。

### Cookie / CSRF
- Session Cookie：HttpOnly、SameSite=Lax、Path=/、HTTPS 下 Secure。
- page key Cookie：HttpOnly、SameSite=Lax、HTTPS 下 Secure。
- 登录、注册、登出、用户写操作、管理员写操作均检查 Origin/Fetch Metadata；管理页 `/api` 另有 `webOwnPage`。
- 未发现通过 query、请求头或 JSON 角色字段伪造管理员权限的入口。

## 7. 验证矩阵与真实结果

### Go 测试

| 命令 | 结果 |
| --- | --- |
| `go test -tags nogui ./internal/identity -count=1` | PASS |
| `go test -tags nogui ./internal/gui -count=1` | PASS（最终完整运行；一次中间运行出现无关的 `TestPrivacyModelFolder` 时序失败，单测通过，随后完整重跑 PASS） |
| `go test -tags nogui ./internal/cli -count=1` | PASS，42.0s（在 3425 未被运行中服务占用时） |
| `go test -tags nogui ./internal/cli -run 'TestEveryRouteIsNamedForTheCLI|TestUserCLI' -count=1` | PASS |
| `go vet -tags nogui ./...` | PASS（exit 0） |
| Phase 5 全量 Gateway 套件 | 本阶段未重跑；Phase 5 已知在 600s 工具时限内未跑完，Phase 6 未修改 Gateway 代码 |

CLI 全量测试第一次在 3425 被已部署服务占用时出现 3 个 healthcheck/update 发现测试失败；停止该服务后完整重跑通过。这不是 Phase 6 代码失败。

### 浏览器 E2E（独立 HOME）

环境：
- HOME：`/tmp/phase6home`
- 页面：`http://127.0.0.1:3498`
- Gateway：`http://127.0.0.1:3499`
- 模拟供应商：`http://127.0.0.1:39002`
- 独立测试 page key 与测试账号；未修改生产 HOME
- Playwright Chromium 1254

结果：PASS，约 4.9s。

| 场景 | 实际结果 |
| --- | --- |
| 未登录访问 `/` | 302 → `/portal/login?next=%2F`，未返回管理页 |
| 未登录注册 | 默认关闭，注册页显示“注册暂未开放” |
| 普通用户 Session 访问 `/` | 403，页面不含管理导航 |
| 普通用户访问 `/api/state`、`/api/providers`、`/api/settings`、`/api/cli`、`/api/admin/*` | 403 |
| query/header 伪造角色 | 仍为 403 |
| 普通用户自己的 `/api/auth/me`、`/api/user/keys`、`/api/user/models`、`/api/user/usage` | 200，未回退 |
| 管理员从 `?next=/` 登录 | 进入 `/` 完整管理页，无需 page key |
| 管理导航 | Providers、Routing、Routing groups、Gateway、Usage、Masking、Plugins、Settings 均在；无 Agents/Sessions/Library 入口 |
| 管理员 Cookie | 只有 `magpie_session`，无 `magpie_web_3498` |
| 管理页 URL / localStorage / sessionStorage | 无 page key |
| 管理员 `/api/state`、`/api/providers`、`/api/settings`、`/api/admin/users`、`/api/admin/registration` | 200 |
| 管理员开放真实模型 → 用户模型广场显示 | 通过 |
| 用户创建 Key → Gateway 调用已开放模型 | 200，上游收到 `m1` |
| 用户用量接口 | 200，调用数 ≥ 1 |
| 管理员打开注册 → 新用户注册 | 成功进入控制台 |
| 注册时提交 `role:admin` | 400，未创建账号 |
| 管理员关闭注册 → 游客注册 | 403 |
| 管理员登出后访问 `/` | 跳转登录，`/api/state` 401 |
| 禁用管理员后旧 Session | 跳转登录，401 |
| 修改密码后旧 Session / 旧密码 | Cookie 失效；旧密码 401 |
| 降级管理员后旧 Session | 失效；重新登录按普通用户进入控制台 |
| 最后一名管理员降级/禁用/删除 | 409 |
| 旧 page key 链接 `?k=` | 303 → `/`，管理页 200 |
| 仅 page key 的 `/api/state`、`/api/providers` | 200（旧权限保留） |
| 仅 page key 的 `/api/admin/users`、`/api/admin/registration` | 401（未增权） |
| 跨站 POST `/api/settings`（管理员 Cookie） | 403 |
| 跨站 GET `/api/state` | 403 |
| 跨站登录 | 403 |
| HTTPS 代理下登录 | Cookie 含 Secure、HttpOnly、SameSite=Lax |
| HTTPS 代理下 page key 链接 | Cookie 含 Secure，303 |
| 连续 5 次错误密码后正确密码 | 429，`Retry-After` > 0 |

截图：
- `/tmp/phase6-anonymous-login.png`
- `/tmp/phase6-user-denied.png`
- `/tmp/phase6-admin-no-key.png`
- `/tmp/phase6-second-admin.png`
- `/tmp/phase6-page-key.png`
- `/tmp/phase6-admin-console.png`
- `/tmp/phase6-prod-admin.png`

## 8. 管理端体验

- 管理员在门户控制台看到“管理后台”卡片，可进入“模型开放管理”或“打开原管理后台”。
- 原管理后台保留 Providers、Routing、Routing groups、Gateway、Usage、Masking、Plugins、Settings；`index-web.html` 未加入 Agents、Sessions、Library。
- 门户控制台管理员入口新增“新用户注册”开关，沿用门户卡片、按钮、开关的现有视觉变量，没有做无关视觉重构。
- 管理员登录返回原目标 `/`；普通用户即使从 `/` 的登录链接登录，也会回到控制台，避免落到 403。

## 9. 服务部署与 smoke test（实际执行）

- 旧 Phase 5 进程已停止；3430/3425 端口释放
- 新二进制：`/tmp/magpie-phase6`，SHA-256 `947668a71855ca9ae3c1a90f3374c01cd18c30d35eb13763ea4cd869287acc08`
- 进程：`magpie-phase6 web --lan --no-open --gateway`，PID 229512
- 页面：`http://127.0.0.1:3430`
- Gateway：`http://127.0.0.1:3425`
- 原 `MAGPIE_WEB_KEY` 与管理员账号保持不变；未改生产密码
- 生产 smoke：
  - `/portal/` 200
  - 未登录 `/` 302 → `/portal/login?next=%2F`
  - Gateway `/v1/models` 200
  - 管理员登录 200
  - 管理员 Session 访问 `/` 200，`/api/state` 200，`/api/admin/users` 200
  - `GET /api/admin/registration` → `{"open":false}`（默认关闭）
  - 旧 page key 访问 `/` 200
  - Playwright 管理员登录后加载完整管理导航，无 page key Cookie，无页面错误
- 生产当前没有供应商，模型管理页显示 0 个模型；注册默认关闭，管理员可在门户控制台或 CLI 打开。

## 10. 未解决问题与风险

1. **page key 仍是完整运维凭据**：兼容保留意味着持有 page key 的普通用户仍能按旧权限访问管理页和旧 API。它不再是唯一入口，但无法按管理员账号逐人撤销。建议后续在确认旧客户端迁移完成后退役。
2. **登录限速是进程内内存状态**：重启清零；按账号+来源和来源计数，不能完全阻止大量不同 IP 的分布式尝试。若部署多实例，各实例计数独立。
3. **反向代理后来源是代理地址**：限速按 `RemoteAddr` 计数，不信任可伪造的转发头；同一代理后的用户共享 20 次/来源额度。
4. **注册开关是本机文件**：`registration.json` 未纳入现有备份/同步；迁移机器后会回到默认关闭（安全侧默认），管理员需重新打开。
5. **权限变更与已打开页面**：服务端下一次请求即拒绝，但已经渲染的页面可能暂时显示旧数据；没有管理 WebSocket。`/api/uninstall` 的 SSE 在开始时鉴权，流进行中撤销角色不会中途终止。
6. **桌面窗口**：应用自身窗口仍按本机进程内信任运行；本阶段只改变 Web 页面入口。
7. **远端管理 API**：`/v1/magpie/admin/*` 仍用 Gateway 管理 Key，不是平台 Session；保持原有客户端兼容。
8. **测试环境说明**：完整 GUI 套件的一次中间运行出现无关 `TestPrivacyModelFolder` 时序失败，单独通过并完整重跑通过；完整 CLI 第一次运行受 3425 已占用服务影响，释放端口后通过。Phase 5 全量 Gateway 套件在本阶段未重跑。

## 11. 完成标准对照

| 标准 | 结果 |
| --- | --- |
| 管理员仅用平台账号登录即可使用完整管理后台 | 通过（E2E + 生产 smoke） |
| 普通用户无法访问管理页/管理 API | 通过（403；page key 另论） |
| 旧 page key 兼容不回退、不增权 | 通过（页面/旧 API 200；账号 API 401） |
| 未登录 `/` 跳登录并可返回原目标 | 通过（302 + `next=/`） |
| Session 失效/禁用/降级/改密后立即失败 | 通过（登出、禁用、改密、降级均 E2E） |
| CSRF/Origin/Cookie/登录限速/注册开关 | 通过（Go + E2E） |
| 用户 Key、模型授权、用量、Gateway 不回退 | 通过（独立 HOME E2E：创建 Key、Gateway 200、用量 200） |
| 未提交、未推送 | 已遵守 |

等待验收。
## Phase 7 修订说明

Phase 7 发现并修复了 HTTPS 状态判断过宽的问题：`X-Forwarded-Proto: https` 不再单独被信任。只有部署配置了 HTTPS 的 `MAGPIE_PUBLIC_URL`，且请求 Host 与该地址一致时，才把代理的转发头当作 HTTPS；直接 TLS 仍始终 Secure。Phase 6 报告里“仅凭 X-Forwarded-Proto 得到 Secure”的验证条目已被 Phase 7 的 `TestSecureCookiesBehindHTTPS` 取代，详见 `PHASE7-REPORT.md`。
