# P0 修复与验收报告

> 日期：2026-11-24
> 基线：`/root/magpie-original`，分支 `remove-partners`，HEAD `e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e`
> 范围：只处理 P0-1 / P0-2 / P0-3 / P0-4；未进入 P1，未 commit / push
> 状态：P0-1 PASS，P0-4 PASS，P0-2 行为 PASS（独立探针）/ 仓库测试 NOT VERIFIED（OOM），P0-3 方案 PASS（发布构建未执行）

---

## 0. 状态总表

| 项 | 状态 | 说明 |
|---|---|---|
| P0-1 空授权 Key 跨租户越权 | **PASS** | `heldKey` 纳入 Owner；空授权创建/更新拒绝；跨租户与全局动作测试通过 |
| P0-2 用户 Key 默认拒绝 MCP/工具 | **PASS（独立探针）** | 7 个入口全部 403；管理员 Key 仍 200。仓库内 `TestUserKeyHasNoMCPTools` 因测试包编译 OOM 未重跑，标 **NOT VERIFIED** |
| P0-3 可复现发布基线 | **PASS（方案）** | 文件清单、HEAD/工作树差异、Docker/版本问题、发布流程已成文；未执行 commit/build |
| P0-4 平台状态备份与恢复 | **PASS** | 账号、Session、模型授权、注册开关进入加密备份；恢复有校验、锁与回滚；隔离 HOME 验证通过 |
| 最终发布 | **未发布** | 工作树仍 dirty；报告 §6 列出发布前剩余事项 |

状态只用 PASS / FAIL / NOT VERIFIED；未把未验证项写成通过。

---

## 1. Git 与工作区保护

| 项 | 值 |
|---|---|
| 分支 / HEAD | `remove-partners` / `e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e` |
| `git status --porcelain` | 89 行 |
| 已跟踪修改 | 50 文件 |
| 未跟踪（目录折叠） | 39 项 |
| 未跟踪文件（展开） | 10,309：`.mimosa` 10,101、docs 160、源码 48 |
| `git diff --stat` | `50 files changed, 1380 insertions(+), 692 deletions(-)` |
| 暂存区 | 0 |
| `git describe` | `v0.1.4574-1-ge41adc151-dirty` |

未执行 checkout / reset / clean / stash / cherry-pick / commit / push。未修改生产配置、生产账号或生产数据；P0 复现、备份/恢复与探针均使用临时 HOME。

完整源码清单：`docs/product-v2/P0-RELEASE-MANIFEST.txt`。

---

## 2. P0-1：空模型授权 Key 跨租户越权

### 2.1 原始问题与复现

修复前，普通用户可 `POST /api/user/keys` 传 `{"models":[""]}`：

- GUI 只检查原始数组长度（`len == 1` 通过）；
- `cleanUserKeyModels` 把空字符串 trim 掉后得到空列表；
- Key 以 `ModelsSet=true`、`Models=[]`、`Owner!=""` 保存；
- Gateway 的 `heldKey` 只看 `Limited/Models/Accounts`，不看 `Owner`，把该 Key 当成“未受限”。

在隔离实例上实测修复前行为：该 Key 对 `/v1/magpie/usage`、`/usage/requests`、`/quotas` 返回 200；用只读探针写入另一租户记录后，能读到 `other-user-key`、`other@example.com`、模型与 token；`/quotas/codex-reset` 越过 held 拦截，`/v1/magpie/update` 返回 200。

### 2.2 修改文件

| 文件 | 修改 |
|---|---|
| `internal/gateway/usage_api.go` | `heldKey` 改为 `KeyID != "" && (Restricted() || Limited())`；`Restricted()` 包含 `Owner` |
| `internal/gateway/usage_ingest.go` | 平台用户 Key 一律不能上报用量 |
| `internal/access/userkeys.go` | 新增 `ErrEmptyGrant`；显式空授权创建/更新拒绝；新用户 Key 默认 `MCP=NoMCP` |
| `internal/access/access.go` | 用户 Key 的 `models-key` 空数组拒绝；用户 Key 的 `mcp-key` 只允许 `-`；ownerless 语义不变 |
| `internal/access/issue.go` | 用户 Key 强制 `MCP=NoMCP` |
| `internal/gui/userkeys.go` | PATCH 空授权映射 400（原 500） |
| `internal/access/modelgrant_test.go` | 更新旧测试对“显式空授权”的期望：由接受改为拒绝 |

### 2.3 语义核对

| 输入 | GUI POST | GUI PATCH | 内部 `CreateUserKey` | 说明 |
|---|---|---|---|---|
| `models` 缺失 | 400 | 400（且无 `off`） | `nil` = 当前所有已开放模型 | GUI 不接受缺失 |
| `models: null` | 400 | 400 | `nil` | 解码为 nil |
| `models: []` | 400 | 400 | 拒绝（`ErrEmptyGrant`） | 显式空授权 fail-closed |
| `models: [""]` | 400 | 400 | 拒绝 | 修复前的利用入口 |
| `models: [" "]` | 400 | 400 | 拒绝 | trim 后为空 |
| `models: [null]` | 400 | 400 | 拒绝 | 解码为空字符串 |
| 合法开放模型 | 201 | 200 | 成功 | 仍受平台 open 策略约束 |
| 管理员 `models-key` 空数组（用户 Key） | — | — | **拒绝** | 不再把用户 Key 变成全权 |
| 管理员 `models-key` 空数组（ownerless） | — | — | 成功 | 保留“空=全部模型”的历史语义 |
| 历史 Key 无 `ModelsSet`、当前无开放模型 | — | — | 回填空授权，`ModelsSet=true` | Key 仍被 held，不能读平台数据 |
| 历史 Key 无 `ModelsSet`、有开放模型 | — | — | 回填当时开放模型 | 不自动获得未来新开放的模型 |

**`nil` 是否会导致非预期权限**：`nil` 只在内部/测试路径可达，GUI 明确要求非 nil 且非空。其语义是“创建时所有已开放模型”的快照，不是“未来所有模型”；即使开放集合为空，得到的也是 held 的空授权，不会变成不受限 Key。

**Owner 隔离**：`/api/user/*` 的 owner 来自会话，不接受请求参数；`/v1/magpie/usage*` 等租户数据接口统一经 `heldKey`；`usageIngest` 显式拒绝用户 Key；ownerless 管理员 Key 保留跨租户读取。

### 2.4 新增测试与结果

| 测试 | 结果 |
|---|---|
| `TestEmptyGrantUserKeyIsHeld` | PASS：空授权不能读全局 usage/quotas/trace，不能改 B 数据，不能触发 reset/update/ingest，不能调用模型 |
| `TestLegacyEmptyUserKeyIsHeld` | PASS：历史 Key 在没有开放模型时回填空授权，仍被 held；读不到 B 的记录 |
| `TestGrantedUserKeyCannotReadPlatformData` | PASS：有模型授权的用户 Key 同样不能读全局数据 |
| `TestAdminGatewayKeyKeepsPlatformAccess` | PASS：ownerless 管理员 Key 的 usage/quotas 仍 200 |
| `TestCreateUserKeyRejectsEmptyGrant`、`TestUpdateUserKeyRejectsEmptyGrant` | PASS |
| `TestAdminModelsKeyEmptyRefusedForUserKey` | PASS：用户 Key 拒绝；ownerless Key 仍允许 |
| `TestUserKeyAPIRejectsEmptyGrant`（GUI） | PASS：POST/PATCH 的 `""`、`" "`、`null`、`[]` 全部 400，合法授权 201 且不被拒绝操作改动 |
| `internal/access` 全量 | PASS（0.187s） |
| `internal/gui` 全量 | PASS（90.052s） |
| Gateway 鉴权/Key/配额/用量子集 | PASS（72.817s） |

---

## 3. P0-2：用户 Key 默认拒绝 MCP/插件工具

### 3.1 修改

| 文件 | 修改 |
|---|---|
| `internal/access/issue.go` | `Owner != ""` 的 Key 强制 `MCP = []string{NoMCP}`，无论存储里写了什么 |
| `internal/access/userkeys.go` | 新建用户 Key 直接存 `MCP=NoMCP` |
| `internal/access/access.go` | 管理员对用户 Key 的 `mcp-key` 只接受 `-`（或空，归一为 `-`）；其他服务器/工具拒绝 |

管理员 ownerless Key、显式 MCP 授权的内部 Key 不受影响。

### 3.2 测试超时/OOM 定位

用户运行的组合命令在编译阶段被系统杀死：

```text
github.com/yetone/magpie/internal/gateway: /usr/lib/go-1.27/pkg/tool/linux_amd64/compile: signal: killed
```

`dmesg`：

```text
Out of memory: Killed process 429647 (compile)
total-vm:4731108kB, anon-rss:3165100kB
```

当时主机 4.8 GiB RAM、4.9 GiB swap 已用 4.7 GiB；`compile` 峰值约 3.1 GiB 匿名 RSS。结论：

- 不是 handler 死锁，也不是测试断言失败；测试尚未开始；
- 不是 `-test.timeout 60s` 触发；
- 是 Gateway 测试包（1,387 个 `_test.go`）的编译内存把宿主机压到 OOM；
- 之前 `/tmp/gateway-p0.test` 编译成功过，改断言后重新编译时 OOM。

### 3.3 独立探针（小体量）

为避免再次编译 1,387 个测试文件，使用 `/tmp/mcpprobe` 独立测试，只编译 Gateway 包，调用导出的 `gateway.New().Front()`（与真实 HTTP 入口相同的 `lanGuard`/身份链），以普通用户 Key 请求 7 个 MCP 入口：

```text
/mcp/magpie/tools       -> 403
/mcp/magpie/web-search  -> 403
/mcp/magpie/vision      -> 403
/mcp/magpie/image       -> 403
/mcp/magpie/browser     -> 403
/mcp/magpie/memory      -> 403
/mcp/magpie/sessions    -> 403
ownerless admin key tools/list -> 200
PASS
ok github.com/yetone/magpie/mcpprobe 0.056s
```

其中 memory/sessions 由各自的前置守卫返回 403（不同话术），tools/search/vision/image/browser 由 `mcpKeyServe` 返回 403。产品权限结果一致。

仓库内 `TestUserKeyHasNoMCPTools` 已改为按每个入口的实际拒绝话术断言（七入口），但**没有重编译执行**，状态 **NOT VERIFIED**。此前旧断言的测试二进制运行结果同样是七个入口全部 403，只是断言只接受一种话术而 FAIL。

### 3.4 P0-2 结论

- 行为：**PASS**（独立探针覆盖全部 7 个入口，管理员 Key 200）。
- 仓库测试：**NOT VERIFIED**（OOM；不是产品失败）。
- 未通过降低断言或跳过入口获得 PASS；memory/sessions 使用各自真实的 403 话术。

---

## 4. P0-3：可复现发布基线

状态：**PASS（方案完成）**；发布构建未执行、未 commit。

### 4.1 文件清单

完整清单见 `docs/product-v2/P0-RELEASE-MANIFEST.txt`。摘要：

- 已跟踪修改 **50** 个；
- 未跟踪源码 **48** 个（不含 `docs/` 与 `.mimosa`）；
- 未跟踪 docs **160** 个（报告/截图/证据，不含 `.mimosa`）；
- `.mimosa` **10,101** 个工具状态文件；
- 平台关键组：
  - 账号/会话：`internal/identity/*`、`gui/auth.go`、`loginlimit.go`、`web.go`、`cli/user_cli.go`
  - 用户 Key/Owner：`access/userkeys.go`、`access/access.go`、`issue.go`、`models.go`、`gui/userkeys.go`、`usage/user_usage.go`
  - 模型授权：`modelaccess/*`、`gui/modelaccess.go`、`gateway/key_models.go`、`lan.go`、`usage_api.go`
  - 门户：`gui/portal.go`、`gui/assets/portal/*`、`assets/index-web.html`
  - 备份：`backup/backup.go`、`backup/platform.go`、`cli/backup_cmd.go`

### 4.2 HEAD 与当前工作树

| 项 | HEAD `git archive` | 当前工作树 |
|---|---|---|
| `internal/identity` | 不存在 | 存在 |
| `internal/modelaccess` | 不存在 | 存在 |
| `gui/portal.go` / `assets/portal` | 不存在 | 存在 |
| `magpie user` 命令 | 无 | 有（list/add/passwd/role/disable/enable/remove） |
| 构建 | 成功 | 成功 |
| 版本 | `head`（测试注入） | `p0-remediation`（测试注入） |

### 4.3 Docker / 版本问题

- `Dockerfile` 使用 `COPY . .`：会把未跟踪源码一起编入；`COPY` 不会自动只取 Git 跟踪文件。
- `.dockerignore` 未排除 `.mimosa/`、`**/.mimosa/`、`docs/`、仓库根 `magpie` 二进制；构建上下文会携带审计截图和 10,101 个工具状态文件。
- 最终运行镜像只 `COPY --from=build /out/magpie` 与 `/config`，所以截图/工具状态不会进入最终层；但构建上下文和“编入哪些源码”仍不可复现。
- `Dockerfile` 的 `ARG VERSION=dev`；`Makefile` 用 `git describe --tags --always --dirty`，当前为 `v0.1.4574-1-ge41adc151-dirty`。不能作为正式版本。
- 基础镜像 `golang:1.26-alpine` 是浮动 tag；`go.mod` 声明 1.26.8，应固定为 `golang:1.26.8-alpine`。
- `CMD ["serve"]` 只启动 Gateway，不启动门户；门户部署需显式 `web --addr 0.0.0.0:3430 --no-open`。

### 4.4 发布流程（待人工执行）

1. 人工审查后，把清单 [1] + [2] 提交到一个 release commit；排除 `.mimosa`、截图、临时文件。
2. 更新 `.dockerignore`：`.mimosa`、`**/.mimosa`、`docs`、`magpie`、`dist`、`build`、`site`、`.git`、`.github`、`*.md`；保留 `internal/`（`go:embed` 需要）。
3. 只在干净 tag 上构建，`git status --porcelain` 必须为空；用 `make cli VERSION=<tag>` 或 `docker build --build-arg VERSION=<tag>`。
4. 固定 `golang:1.26.8-alpine`；记录 commit/tag/tree、Go 版本、二进制 SHA-256、镜像 digest、SBOM/attestation。
5. 镜像 smoke：`/portal/` 200、注册关闭、`magpie user add` 首管理员、用户 Key 调用 `/v1/models` 与一个模型。
6. 发布镜像不得以 `-dirty` 版本标识出现；不得用当前 dirty 构建冒充正式版本。

---

## 5. P0-4：平台状态备份与恢复

状态：**PASS**。

### 5.1 实现

| 文件 | 内容 |
|---|---|
| `internal/backup/backup.go` | `Bundle.Platform`；`BundleVersion=3`；`Collect` 读取平台状态；`Restore` 校验版本并恢复；`All` 包含 Platform |
| `internal/backup/platform.go`（新） | `Platform` 四文件；JSON 形状校验；固定目标路径；users/model-access/registration 三把文件锁；快照、原子写 0600、失败回滚 |
| `internal/cli/backup_cmd.go` | 恢复输出“platform accounts and policy” |

恢复顺序：平台状态最后写入；校验在任何写之前；写入失败回滚已写文件。备份文件仍是 AES-256-GCM + PBKDF2 的加密 envelope；Bundle 版本 3 检查未来版本。

### 5.2 覆盖范围

- `users.json`：平台账号、密码哈希、角色、禁用状态；
- `sessions.json`：会话 token 哈希；
- `model-access.json`：模型开放/名称/描述；
- `registration.json`：注册开关；
- `caller-keys.json` 中的用户 Key（hash/Owner）沿用既有 GatewayKeys 备份；
- 恢复后账号、登录、Session、Key owner、模型开放、注册开关一致。

### 5.3 测试与结果

| 测试 | 结果 |
|---|---|
| `TestPlatformStateTravelsWithABackup` | PASS：创建账号/用户/会话/用户 Key/模型开放/注册开关 → 加密备份 → 全新 HOME 恢复 → 登录、Session、Key owner、模型开放、注册开关、0600 全部验证 |
| `TestPlatformRestoreRejectsBrokenJSON` | PASS：损坏 JSON 在写之前拒绝，原文件不变 |
| `TestPlatformRestoreRefusesANewerBundle` | PASS：Bundle 版本更高时拒绝 |
| `TestPlatformRestoreRollsBackOnPartialWrite` | PASS：第三次写入失败，users/sessions/model-access 回滚为原内容 |
| `TestPlatformRestoreSkipsAnOlderBundle` | PASS：旧备份不改平台文件 |
| `TestPlatformIsLeftOutOfAKeylessBackup` | PASS：`--no-keys` 不带平台账号 |
| `TestCorruptPlatformFileStopsTheBackup` | PASS：本地损坏 users 文件阻止备份，不生成“无账号”备份 |
| `internal/backup` 全量 | PASS（3.9s） |

另做隔离 CLI 验证：

```text
magpie backup    -> 1,736 bytes
magpie restore   -> platform accounts and policy
user list        -> admin@cli.test / user@cli.test 恢复
user registration-> open
files            -> users.json 600, registration.json 600
```

### 5.4 已知边界

- `--no-keys` 不携带平台状态（含密码哈希），这是刻意的；运维必须用默认全量备份迁移平台。
- 自动备份（`Collect(false)`）不携带平台状态；依赖自动备份恢复时需要额外的手工步骤。
- 备份完整性由 GCM 认证标签保证；恢复的内容形状另做 JSON 校验。
- 恢复不覆盖正在运行的生产目录进行测试；所有测试使用临时 HOME。

---

## 6. 测试证据汇总

| 命令 | 结果 |
|---|---|
| `go test -tags nogui ./internal/access` | PASS 0.187s |
| `go test -tags nogui ./internal/modelaccess` | PASS 0.033s |
| `go test -tags nogui ./internal/identity` | PASS 0.601s |
| `go test -tags nogui ./internal/backup` | PASS 3.9s |
| `go test -tags nogui ./internal/gui` | PASS 90.052s |
| `go test -tags nogui ./internal/gateway -run 'Test.*(Key|MCP|Held|Quota|Usage|ModelAccess|AdminAPI|ReadOnly|Caller|User)'` | PASS 72.817s |
| P0 gateway 回归（empty/legacy/admin/MCP/usage） | PASS |
| `/tmp/mcpprobe` 独立 7 入口探针 | PASS 0.056s |
| `go vet` 修改包 | PASS（exit 0） |
| 当前源码构建 | PASS |
| 仓库内 `TestUserKeyHasNoMCPTools`（改断言后） | **NOT VERIFIED**：Gateway 测试包编译 OOM，未执行 |

---

## 7. 未解决事项与发布影响

1. **仓库 MCP 测试未重编译**：宿主机 4.8 GiB RAM + 4.7/4.9 GiB swap，Gateway 测试包编译 OOM；独立探针 PASS，仓库测试 NOT VERIFIED。发布前应在内存充足的机器上跑一次 `go test -tags nogui ./internal/gateway -run '^TestUserKeyHasNoMCPTools$'`。
2. **P0-3 未执行**：当前工作树仍 dirty，未 commit/tag；发布流程只是方案，未实际构建正式镜像。发布影响：现在还不能发布。
3. **P0-1 的 `nil` 语义**：内部 `CreateUserKey(nil)` 快照当前开放模型；GUI 不可达。若未来新增内部调用方，需要保持“快照 + Owner held”。
4. **备份边界**：`--no-keys` 与自动备份不含平台状态；运维必须使用默认全量 `magpie backup` 迁移账号/模型授权/注册开关。
5. **未验证真实厂商上游**：本轮未调用外部厂商；历史 Phase 7 的真实调用只作背景。
6. **未执行全量 Gateway 测试**：只跑鉴权/Key/MCP/配额/用量子集；全量约 12 分钟，未在本轮复跑。

---

## 8. 最终结论

- **P0-1：PASS**。空授权不再被当作全权；Owner 隔离覆盖租户数据与管理动作；管理员 ownerless Key 保持兼容。
- **P0-2：PASS（独立探针）**。7 个 MCP 入口全部 403，管理员 Key 200；仓库测试因 OOM 未重跑，标 NOT VERIFIED。
- **P0-3：PASS（方案）**。文件清单、HEAD/工作树差异、Docker/版本问题、发布流程已成文；发布构建与 Git 操作待人工批准。
- **P0-4：PASS**。平台状态进入加密备份，恢复有版本校验、固定路径、锁与回滚；隔离 HOME 与 CLI 验证通过。

**P0 修复阶段完成，等待人工验收。** 未进入 P1，未发布，未 commit / push。
