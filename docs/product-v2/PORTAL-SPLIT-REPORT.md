# PORTAL-SPLIT-REPORT — 概览 / API 密钥 / 个人资料拆分

日期：2026-11-25
分支：`remove-partners`
HEAD：`e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e`
版本标记：`v0.1.4574-portal-split`
范围：用户门户 `/portal/console`、新增 `/portal/keys`、新增 `/portal/profile`、Dock 与门户路由
未 commit / push / reset / stash。

---

## 1. 信息架构

| 页面 | 内容 |
|---|---|
| `/portal/console` | 概览：4 格真实用量摘要、用量统计（5 个时间范围 + 6 统计）、2 张独立图表卡、请求明细 |
| `/portal/keys` | API 密钥：页面头、创建密钥、密钥表格、模型授权、启用/禁用/删除、完整 Key 只显示一次 |
| `/portal/profile` | 个人资料：头像/UID/加入日期、四统计、53 周 Token 热力图、6 格账号信息、管理员卡片（仅管理员） |

路由：`portalPages` 与需登录页面列表加入 `keys`、`profile`；未新增 API、权限或数据结构。
Dock：概览 `/portal/console`、模型广场 `/portal/models`、API 密钥 `/portal/keys`、管理后台 `/`（仅管理员）、头像 `/portal/profile`。营销首页不占 Dock，经品牌 Logo / 页脚 Home 链接访问。当前项高亮沿用 Dock；个人资料页用头像白环高亮。

---

## 2. 概览实现

- 页头：“概览” + 13px muted 说明“当前账号的用量与请求记录” + 右侧账号/退出；不显示 email、角色、注册时间。
- 摘要条：4 格，仅真实用量数据，跟随当前时间范围：
  - 总请求、成功、失败、总 Token；
  - 数值 28/700，标签 12px muted。
- 用量统计卡：标题 20/600；时间范围 5 档（今天/昨天/近 7 天/近 30 天/全部）；现有 6 个统计 3×2（移动 2×3）。
- 图表：每日用量、模型用量拆成 2 张独立卡（≥1100 两列 646px + 646px；≤980 单列）；空态 48px 圆形图标 + 标题 + 说明，空图表高度 181px，不是 166px 空白。
- 请求明细：标题 20/600；全部/成功/失败分段、全部模型 select、自动刷新开关（15s）、刷新图标；表格行高 60、表头 12px muted、数字右对齐、移动端转卡片。
- 删除：API Key 表格、账号角色/注册时间/Key 数量、管理后台说明卡；底部只留一句“请求由 Gateway 记录，只显示当前账号的记录。”

**时间范围数据**：`/api/user/usage/summary|trend|models|requests`。今天/近 7 天/近 30 天/全部用现有 `period=`；昨天用现有 `from=YYYY-MM-DD&to=YYYY-MM-DD` 单日查询，没有新增后端。

**筛选范围**：请求明细保留现有模型 select 与全部/成功/失败状态分段。`/api/user/usage/requests` 仅支持 `model` 与 `state` 查询，没有 key 查询参数，因此没有添加“全部密钥”假筛选；用量 summary/trend/models 也不接受模型筛选，模型 select 只作用于请求明细，不伪装成统计筛选。

---

## 3. API 密钥页

- 页头：标题“API 密钥”20–32/700 + 副标题；右上角真实 Key 数量徽标 + 白底“创建密钥”主按钮。
- 表格：名称、掩码 Key（mono）、模型授权、状态、操作；行内主操作“模型开放管理”品牌蓝描边，启用/禁用/删除为文字按钮。
- 创建表单、完整 Key 只显示一次、复制、模型授权弹窗、启用/禁用/删除、空态/错误态全部沿现有 `consoleKeys()` 逻辑；页面只在 `data-page=keys` 初始化。
- 不显示分组、倍率、额度、并发。

---

## 4. 个人资料页

- 页头：“个人资料” + 13px muted 说明；右侧“退出登录”，管理员额外显示“管理后台”。
- 顶部大卡：80px 圆形头像（姓名首字母）+ 右上角真实角色徽标；姓名 24/600；UID 与“加入于 YYYY-MM-DD”；UID 复制按钮；四列统计（近一年 Token、单日峰值、当前连续、最长连续），移动端 2×2。
- Token 活跃度：全宽卡片；每日/每周/累计三个真实视图；7 行 × 53 周热力格（10×10、3px 圆角、gap 3px、`--chart-4` 五档）；月份标签 11px muted；图例“少 → 多”；无数据全灰 + “暂无用量”。移动端横向滚动，首次显示最近月份。
- 账号信息：3×2 网格，32px 图标 + 值 14/600 + 说明 12 muted + 右侧描边小按钮；6 个真实字段：
  1. 邮箱 + 登录账号
  2. 角色 + 账号权限
  3. 注册时间
  4. API Key 数量 + 管理密钥
  5. 最近登录（无则“—”）
  6. 账号 ID + 复制
- 管理员卡片（模型开放、原管理后台、注册开关）保留在账号信息之后，仅管理员可见；普通用户没有管理后台按钮和卡片。
- 没有改名接口，因此没有做编辑图标/假编辑；没有 OAuth、邀请返利、密码修改、设备管理按钮。

**统计口径（全部来自 `/api/user/usage/trend?from=…&to=…` 的 365 天）**：

- 近一年 Token = 365 天 `total` 之和；
- 单日峰值 = `max(单日 total)`；
- 当前连续 = 从今天往前连续 `total > 0` 的天数；
- 最长连续 = 全年最长连续 `total > 0` 的天数；
- 没有数据时为 0；接口失败时统计为“—”，热力图不伪造数据。

---

## 5. 浏览器验收（Chromium + Playwright）

### 5.1 1440 概览

| 项 | 实测 |
|---|---|
| 摘要条 | 4 格：5 / 4 / 1 / 10,200；无角色/注册时间/Key 数量 |
| 概览 key 表 / 管理卡 | 均不存在 |
| 时间范围 | 今天、昨天、近 7 天、近 30 天、全部 5 个 |
| 昨天 | 摘要 2 / 2 / 0 / 3,500；日志 2 条 |
| 今天 | 摘要 0 / 0 / 0 / 0；图表空态；空图表 181px；模型空态 |
| 全部 | 摘要 5 / 4 / 1 / 10,200 |
| 统计 | 6 个；图表卡 2 张，646px + 646px；无阴影 |
| 请求明细 | 失败 1 条、成功 4 条、全部 5 条；自动刷新开关 true→false；刷新图标可用 |
| 横向溢出 | 1440 = 1440 |
| 控制台错误 | 登录后 0；退出登录后 `/api/auth/me` 的 401 为预期鉴权探测 |

### 5.2 API 密钥

| 项 | 实测 |
|---|---|
| 表格 | 4 行；列名：名称/密钥/模型授权/状态/操作 |
| 创建 | 完整 Key 63 字符；创建后 5 行；关闭后只显示一次；截图未包含完整 Key |
| 模型授权 | 弹窗打开，2 个真实开放模型、2 个勾选；Esc 关闭，焦点回到触发按钮 |
| 删除 | 确认后回到 4 行，目标 Key 不再存在 |
| Dock | API 密钥项高亮 |

### 5.3 个人资料

| 项 | 实测 |
|---|---|
| 管理员 | 姓名 admin；角色徽标“管理员”；加入于 2026-11-24；统计 10,200 / 5,100 / 0 / 3 |
| 热力图 | 371 格 = 7×53；月份标签 53；日视图有 3 个真实着色日；每日/每周/累计切换正常 |
| 账号信息 | 6 格；Key 数量 4；最近登录真实；账号 ID 复制到剪贴板匹配 |
| 管理入口 | 管理员 header“管理后台”+ 管理卡；注册开关仍 false→true→false |
| 普通用户 | 无管理后台按钮/卡；统计 0/0/0/0；热力图全灰 + “暂无用量”；Key 数量 0 |
| 禁用文案扫描 | 无余额/套餐/充值/购买/折扣/返利/邀请/Google/GitHub/密码/设备/OAuth |

### 5.4 390 / 320

| 项 | 实测 |
|---|---|
| `scrollWidth` | 390 = 390；320 = 320（概览/密钥/资料） |
| 可点目标 <44px | 0（三页，390 与 320） |
| 输入 | 请求明细 select 16px/44px；密钥创建名称 16px/44px |
| 概览 | 摘要 2×2、统计 2×3、图表单列、请求明细卡片 |
| 密钥 | 表格每行一卡；创建按钮 44px；表单输入 16px |
| 资料 | 统计 2×2、账号卡单列、热力图滚动 689/324（390）与 689/254（320） |
| Dock | 390 宽 299，320 宽 241；管理后台仅管理员；退出底部 Footer 与 Dock 间距 22px |

### 5.5 主题与对比度

- 深色 token：foreground/card 15.74、muted/card 7.33、brand/card 9.33；均 ≥4.5。
- 浅色：页面背景 `oklch(0.975 0 0)`、卡片白色、边框 `oklch(0.895 0 0)`、无阴影；概览与资料页截图均正常。
- 焦点环：`#profileUidCopy`、`#keyNew`、`#logRefresh` 实测 `oklch(0.78 .13 232) solid 2px` + offset 2px；其余控件复用同一规则。
- 卡片/图表卡：`--card`、1px `--border`、12px 圆角、无阴影。

### 5.6 构建与测试

```sh
GOMAXPROCS=2 CGO_ENABLED=0 go build -tags nogui -trimpath \
  -ldflags "-s -w -X main.version=v0.1.4574-portal-split" -o /tmp/mr-dark .
# BUILD OK

GOMAXPROCS=2 CGO_ENABLED=0 go test -tags nogui ./internal/gui \
  -run 'TestPortal|TestPortalI18nKeys|TestAuth|TestUserUsage|TestCrossOriginLoginRefused|TestAdminRoutesRequireAnAdmin|TestWebAuthBypassesThePageKey' -count=1
# ok

GOMAXPROCS=2 CGO_ENABLED=0 go test -tags nogui ./internal/identity \
  -run 'TestLoginByEmailOrUsername' -count=1
# ok
```

`TestPortalConsoleNeedsASignedInAccount` 已扩展到 `/portal/keys`、`/portal/profile`（未登录重定向、已登录 200）。

---

## 6. 截图

目录：`docs/product-v2/screenshots/portal-split/`

| 页面 | 截图 |
|---|---|
| 概览 | `console-1440-zh-dark.png`、`console-1440-en-dark.png`、`console-390-zh-dark.png`、`console-390-en-dark.png` |
| 概览补充 | `console-1440-zh-dark-logs.png`、`console-1440-zh-today-empty.png`、`console-1440-zh-light.png` |
| 密钥 | `keys-1440-zh-dark.png`、`keys-1440-en-dark.png`、`keys-390-zh-dark.png`、`keys-390-en-dark.png`、`keys-390-zh-create.png` |
| 资料 | `profile-1440-zh-dark.png`、`profile-1440-en-dark.png`、`profile-390-zh-dark.png`、`profile-390-en-dark.png`、`profile-390-zh-heatmap.png`、`profile-1440-zh-dark-info.png` |
| 资料补充 | `profile-regular-1440-zh-dark.png`、`profile-1440-zh-light.png` |

截图数据来自隔离 HOME `/tmp/mr-dark-home` 的真实测试数据（管理员 4 个 Key、5 条用量记录；普通测试账号 0/0），不写入仓库、不代表生产数据。完整 Key 未进入截图。

---

## 7. 未做项 / 偏差

1. `/api/user/usage/requests` 只支持 `model` 与 `state`，没有 key 查询参数；因此请求明细保留“全部模型”select，没有添加“全部密钥”假筛选。summary/trend/models 也不支持模型筛选，模型 select 只作用于请求明细。
2. 请求明细状态分段按需求只做“全部/成功/失败”；`canceled` 仍可出现在“全部”的表格里，但没有单独的取消筛选。
3. 没有改名接口，个人资料页没有做可点的编辑图标或假编辑。
4. 原概览的管理后台卡（模型开放、注册开关、原管理后台）迁移到个人资料的 **管理员专属卡片**，同时页头给管理员“管理后台”按钮；普通用户看不到。该卡不属于参考图的常规用户资料，但保留了原有管理能力。
5. 个人资料页的“每日/每周/累计”三个视图均使用真实 trend 数据：每周为周合计、累计为区间累计，不是假 tab。
6. 热力图首次加载在移动端自动滚到最近月份；之后切换视图保留用户滚动位置。
7. 浅色模式只做了回归验证和截图，不作为本轮主视觉。
8. 未 commit / push。
