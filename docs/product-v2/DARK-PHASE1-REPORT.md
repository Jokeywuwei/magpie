# DARK-PHASE1-REPORT — MiniRelay 用户门户深色改造（第一批）

日期：2026-11-24
分支：`remove-partners`
HEAD：`e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e`（`web: the add sheet's Partners block is removed, not hidden`）
版本标记：`v0.1.4574-dark-phase1`
范围：用户门户 token 层、门户外壳、底部 Dock、登录/注册、概览 `/portal/console`
未提交、未推送、未 reset、未 stash。工作区原有的其他改动未动。

---

## 1. 改了什么

### 1.1 Token 层（`internal/gui/assets/portal/portal.css`）

- `:root` 写入 `DARK-UI-SYSTEM §2.1` 的深色 token，保留 oklch 值与 hex 注释：`--background #0a0a0a`、`--card #151515`、`--popover #1e1e1e`、`--border #2b2b2b`、`--foreground #eeeeee`、`--primary #fafafa`、`--ring/--brand #4fc5fc`、四种状态色、五档 chart 色。
- 浅色派生模式使用 §2.2 的 token；浅色 `--ring/--brand` 按无障碍要求调到 `oklch(52% .15 237)`（`#006cad`），在 `#f7f7f7` 上 4.83:1，已同步写回 `DARK-UI-SYSTEM.md` v1.1 修订记录。
- 圆角 token：`--radius 12px`、`--radius-sm 8px`、`--radius-lg 16px`、`--radius-pill 999px`。Dock 图标按钮按 §5.1 的明确值使用 14px；其余容器只出现 8/12/16/999。
- 字体改为系统栈 + PingFang/Noto CJK 回退；正文 14px，页面标题 32/700，KPI 数字 28/700，标签 12px；数字统一 `tabular-nums`。
- 卡片统一 `--card` 底、1px `--border`、12px 圆角、`box-shadow: none`；只有 Dock 和弹窗使用 `--elevation-float`。

### 1.2 门户外壳与底部 Dock（`index.html` / `portal.css` / `portal.js`）

- 删除门户顶部品牌 + 导航 + 账号横条；页面内容从页头开始。
- 各页页头：左侧 28–32/700 标题 + 13px 说明，右侧账号胶囊（账号名 + 退出）与页面动作。
- 新增固定底部 Dock：底部 16px 居中、`--popover` 92% + blur(16px)、1px 边框、16px 圆角、8px 内距；48×48 图标按钮、22px 细线图标；当前项 `--accent` 底 + 1px `--foreground` 环；右侧 44px 圆形头像。
- Dock 只复用现有入口：概览 `/portal/console`、模型广场 `/portal/models`、首页 `/portal/`、管理后台 `/`（仅管理员，非管理员隐藏）。
- 移动端 Dock 保留底部；`body` 底部预留 104px + safe-area，390 滚动到底时 Footer 与 Dock 间距 22px。
- 品牌只保留在登录页卡片上方、首页页脚和页面标题区域；登录后各页顶部不再有品牌横条。

### 1.3 登录 / 注册

- 深色认证卡：max-width 440px、`--card`、1px 边框、12px 圆角、24–28px 内距、无阴影；品牌位于卡片上方。
- 标题 26/700；说明 13px `--muted-foreground`。
- 输入 42px（移动 44px / 16px 字号）、`--background` 底、1px 边框、8px 圆角；聚焦边框 `--ring` + 3px 20% 光晕，并保留 2px 外焦点环。
- 主按钮白底黑字、全宽；错误条为 `--danger` 12% 暗红底 + 30% 边框 + 13px 文字。
- 原有登录/注册逻辑、错误文案、密码输入显示、语言切换全部保留。

### 1.4 概览 `/portal/console`

- 页头“概览” + 账号/退出胶囊。
- 4 格摘要条：账号角色、注册时间、API Key 数量、近 30 天 Token / 请求；格间为 1px 竖细线。只读取现有真实字段，不展示余额、套餐、充值、购买。
- 用量区：时间范围分段（今天 / 近 7 天 / 近 30 天 / 全部）+ 现有 6 个统计，桌面 3×2、移动 2×3 紧凑网格；数字 28/700，标签 12px muted。
- 图表区：保留现有每日用量与模型用量；无数据时用 48px 圆形图标 + 标题 + 说明的居中空态，图表高度从 166px 收为内容高度（1440 实测 181px，390 实测 200px），不再留空白。
- 请求日志：深色数据表，表头 12px muted，行高 60px（失败行含错误文案自动增高），1px 行分隔，筛选/刷新/分页右对齐；≤900px 每行一卡，字段带本地化标签。
- API Key：名称、掩码 Key、模型授权、状态、操作五列表格；行内主操作（模型开放管理）为品牌蓝描边，启用/禁用/删除为文字按钮；不展示分组、倍率、额度。
- 管理后台入口降级为细线卡片 + 文字按钮，不再使用大色块或 primary。
- 创建 Key、模型授权弹窗、注册开关、语言切换、主题切换读取全部保留。

### 1.5 文件清单

| 文件 | 改动 |
|---|---|
| `internal/gui/assets/portal/portal.css` | 重写 token 层、外壳、Dock、认证、概览、表格、空态与响应式样式 |
| `internal/gui/assets/portal/index.html` | 移除顶部横条；新增 Dock；概览摘要条/Key 表头/空态；登录品牌；页脚语言切换保留 |
| `internal/gui/assets/portal/portal.js` | 账号动作填多个页头槽位；头像/管理员 Dock 显隐；Dock 当前项；Key 行五列与摘要 Key 数 |
| `internal/gui/assets/portal/usage.js` | 摘要条 30 天真实汇总；图表/模型空态；加载与错误态保持 |
| `internal/gui/assets/portal/i18n.js` | 新增 Overview / API Keys / Model grants / Actions / Dock 等词条 |
| `internal/gui/portal.go` | 仅更新 `theme-color` 为新的深色/浅色地面；无路由、无 API、无权限改动 |
| `internal/identity/identity.go` | 登录接受邮箱或用户名（邮箱 @ 前的部分）；用户名大小写不敏感、同名歧义拒绝 |
| `internal/identity/identity_test.go` | 邮箱/用户名登录、大小写、歧义用户名测试 |
| `internal/gui/auth.go` | 登录处理器把输入同时视为邮箱或用户名；JSON 字段仍为 `email` |
| `internal/gui/auth_test.go` | 用户名登录的处理器集成测试 |
| `docs/product-v2/DARK-UI-SYSTEM.md` | 浅色 `--ring/--brand` AA 修正；v1.1 修订记录 |
| `docs/product-v2/screenshots/dark-phase1/` | 验收截图 |
| `docs/product-v2/DARK-PHASE1-REPORT.md` | 本报告 |

### 1.6 2026-11-25 补充：邮箱或用户名登录

- 登录输入现在同时接受完整邮箱和用户名；用户名定义为邮箱 `@` 前的部分（如 `admin@local` → `admin`），大小写不敏感，首尾空格忽略。
- 输入含 `@` 时按邮箱匹配；不含 `@` 时按用户名匹配。同一用户名对应多个账号时不猜测，返回通用 401，使用完整邮箱即可登录。
- 登录页标签改为“邮箱或用户名”，说明改为“使用邮箱或用户名进入控制台”，占位符为 `you@example.com 或用户名`；401 文案改为“邮箱、用户名或密码不正确。”
- 为兼容已有客户端，`POST /api/auth/login` 的 JSON 字段名仍是 `email`，只是值可以是邮箱或用户名。
- 显示名不参与登录；账号名始终是邮箱 `@` 前的部分，避免显示名重复导致登录歧义。

明确未改：`internal/gui/assets/app.css`、后端 API/权限/数据结构、管理后台侧栏与紧凑密度、路由与依赖。

---

## 2. 对应参考图 / 规范

| 规范 | 落地 |
|---|---|
| `DARK-UI-SYSTEM §2.1/§2.2` | 深色 token 全量；浅色派生模式与 AA 修正 |
| `§4.1/§4.2` | 32px 页面留白、1376px 内容宽、8/12/16/999 圆角阶梯（Dock 按钮 14px 按 §5.1） |
| `§5.1` | 底部悬浮 Dock、当前项白环、圆形头像、移动端安全区 |
| `§5.2` | 页头标题 28–32/700 + 说明 + 右侧账号动作 |
| `§6.1–§6.9` | 按钮变体、输入/下拉、分段控件、卡片、表格、徽标、空态、弹窗、图表 |
| `§7.1/§7.2` | 登录卡、概览摘要条、6 统计、图表、请求日志、API Key、管理入口 |
| `§8` | 1100/980/900/760/390/360 响应式；44px 触控；16px 输入；焦点环；Esc 归还焦点 |
| `§10` | 逐项验收见下节 |

---

## 3. 验收数据

### 3.1 1440 概览（深色）

| 项 | 实测 |
|---|---|
| 页面背景 | 计算值 `oklch(0.145 0 0)`；截图像素 `rgb(10,10,10)` ✅ |
| 卡片背景 | 计算值 `oklch(0.195 0 0)`；截图像素 `rgb(21,21,21)` ✅ |
| 边框 | 计算值 `oklch(0.29 0 0)`；截图像素 `rgb(43,43,43)` ✅ |
| 卡片阴影 | `box-shadow: none`；`.card/.model-card/.admin-model-list/.summary-strip/.usage-stats` 全为 0 处阴影 ✅ |
| 顶部导航 | `.topbar` 数量 0 ✅ |
| Dock | `position: fixed`，宽 299、高 66，中心 x=720，bottom=16px；当前项 console 有 1px 白环 ✅ |
| 页面标题 | 32px / 700 ✅ |
| 统计数字 / 标签 | 28px/700 / 12px `--muted-foreground` ✅ |
| 正文 | 14px ✅ |
| 圆角集合 | 8 / 12 / 14（Dock 按钮）/ 16 / 999 ✅ |
| 分段控件 / 表格布局 | `.segs` 与 `.log-filters` 均为 flex；时间范围横向排列 ✅ |
| 控制台错误 | 登录后 console 页面 0 条错误（登出后的 `/api/auth/me` 401 为预期鉴权探测，不计入页面错误） ✅ |

### 3.2 390 / 320

| 项 | 实测 |
|---|---|
| `scrollWidth` | 390 = 390；320 = 320 ✅ |
| 可点目标 <44px | 0 个 ✅ |
| 输入/下拉字号 | 16px；select 高 44px ✅ |
| 摘要条 / 统计网格 | 2 列 / 2 列 ✅ |
| Dock | 390 宽 299，中心 x=195，bottom=16；320 宽 251，仍居中 ✅ |
| 内容底部 | 滚动到底 Footer 与 Dock 间距 390 为 22.1px、1440 为 21.7px ✅ |
| 表格 | 日志/API Key 每行一卡，字段带本地化标签 ✅ |

### 3.3 对比度与焦点

深色（token 对）：

| 组合 | 对比度 |
|---|---|
| foreground / background | 17.06 |
| muted-foreground / card | 7.33 |
| muted-foreground / background | 7.94 |
| brand / background | 10.11 |
| primary-foreground / primary | 17.18 |
| danger / card | 6.32 |
| success / card | 9.42 |
| warning / card | 10.60 |
| foreground / accent | 13.04 |
| foreground / popover | 14.37 |

浅色（回退截图）：

| 组合 | 对比度 |
|---|---|
| foreground / background | 18.48 |
| muted-foreground / background | 5.61 |
| brand / background | 4.83 ✅（修正后） |
| danger / card | 5.35 |
| success / card | 5.19 |
| warning / card | 5.01 |

焦点环：`outline: oklch(0.78 .13 232) solid 2px; outline-offset: 2px`，键盘 Tab 实测可见。输入聚焦另有 3px 20% 光晕。Esc 关闭模型授权弹窗后 `document.activeElement` 回到触发按钮。

### 3.4 登录 / 注册

| 项 | 实测 |
|---|---|
| 认证卡 | `--card` 底、12px 圆角、`box-shadow: none`；桌面内距 28px、移动 22px ✅ |
| 输入 | 桌面高 42px / 14px；390 高 44px / 16px ✅ |
| 主按钮 | `oklch(0.985 0 0)` 白底 + `oklch(0.205 0 0)` 深字 ✅ |
| 错误态 | 暗红底 `danger 12%`、30% 边框、13px 文案，截图可见 ✅ |
| 语言切换 | 中文/English 均可用；`?lang=en` 与 footer 切换生效 ✅ |
| 品牌 | 登录卡上方 `MiniRelay`；未在登录后的每页顶部重复 ✅ |

### 3.5 功能回归

| 功能 | 结果 |
|---|---|
| 登录 | 成功跳转 `/portal/console`；邮箱 `admin@local` 与用户名 `admin` 均可登录 ✅ |
| 用户名登录 | `admin` / `ADMIN` / `test` 均成功；`admin@local`、`test@local` 同样成功；错误密码与未知用户名均 401 ✅ |
| 退出 | 回到 `/portal/`，Dock 头像变回未登录图标 ✅ |
| 创建 API Key | 表单打开、创建成功、完整 Key 只显示一次、复制按钮可用 ✅ |
| 模型授权弹窗 | 打开、勾选状态正确、Esc 关闭并归还焦点 ✅ |
| 删除 Key | 确认后删除，列表刷新 ✅ |
| 注册开关 | `false → true → false`，写入与回读一致 ✅ |
| 语言切换 | 中 → 英 → 中；Console/Model plaza/Home/Admin console 全部英文 ✅ |
| 主题切换 | 隔离 HOME 写 `theme=dark/light` 后刷新：深色/浅色首屏均正确，无白底闪烁 ✅ |
| 模型广场 | 深色/移动端可打开、0 横向溢出；只跟随 token 与外壳，未整页重设计 ✅ |

### 3.6 构建与测试

```sh
GOMAXPROCS=2 CGO_ENABLED=0 go build -tags nogui -trimpath \
  -ldflags "-s -w -X main.version=v0.1.4574-dark-phase1" -o /tmp/mr-dark .
# BUILD OK

GOMAXPROCS=2 CGO_ENABLED=0 go test -tags nogui ./internal/gui \
  -run 'TestPortal|TestAuth|TestUserUsage|TestCrossOriginLoginRefused|TestAdminRoutesRequireAnAdmin|TestWebAuthBypassesThePageKey' -count=1
# ok github.com/yetone/magpie/internal/gui

GOMAXPROCS=2 CGO_ENABLED=0 go test -tags nogui ./internal/identity \
  -run 'TestLoginByEmailOrUsername' -count=1
# ok github.com/yetone/magpie/internal/identity
```

未跑全量 gateway 测试；未改 gateway。

---

## 4. 未做项 / 已知边界

1. 模型广场、首页营销区、模型管理（`/portal/admin-models`）只跟随全局 token 与新外壳，未做整页重设计；管理后台 `app.css` 与侧栏未改。
2. 摘要条第四格使用现有 `/api/user/usage/summary?period=30d` 的真实汇总，标签写“近 30 天 Token / 请求”。现有 API 只有 today/7d/30d/all，没有日历月口径，因此没有伪造“本月”。
3. 门户没有新增 `/billing`、`/profile` 等路由；Dock 只含现有入口。
4. 主题切换仍由管理员设置驱动，门户只读；未新增门户内主题开关（保持原行为）。
5. 未新增密码显示切换；原有的密码输入（圆点）与错误文案保持不变。
6. Dock 图标按钮按 `DARK-UI-SYSTEM §5.1` 使用 14px 圆角；验收圆角集合因此为 8/12/14/16/999，其余容器没有 20/24px 大圆角。
7. 登录页与注册页的 footer 只保留语言切换，品牌位于卡片上方；首页 footer 保留品牌与站内链接。
8. 用户名是邮箱 `@` 前的部分，不是显示名；不同域名使用同一前缀时，用户名会被视为歧义，需改用完整邮箱登录。

---

## 5. 截图

目录：`docs/product-v2/screenshots/dark-phase1/`

| 截图 | 说明 |
|---|---|
| `console-1440-zh-dark.png` | 1440 概览，中文，深色，首屏 |
| `console-1440-zh-dark-bottom.png` | 1440 概览，中文，深色，滚动到底（请求日志/管理入口/Dock） |
| `console-1440-en-dark.png` | 1440 概览，英文，深色 |
| `console-390-zh-dark.png` | 390 概览，中文，深色，首屏 |
| `console-390-zh-dark-bottom.png` | 390 概览，中文，深色，滚动到底（日志卡片/管理入口/Dock） |
| `console-390-en-dark.png` | 390 概览，英文，深色 |
| `console-1440-zh-light.png` | 1440 概览，中文，浅色回退（唯一浅色验收图） |
| `console-1440-zh-dark-empty.png` | 1440 概览空态：0 Key、0 用量、图表 48px 圆形空态 |
| `console-390-zh-dark-empty.png` | 390 概览空态 |
| `login-1440-zh-dark.png` / `login-1440-en-dark.png` | 登录，桌面中/英 |
| `login-390-zh-dark.png` / `login-390-en-dark.png` | 登录，移动中/英 |
| `login-error-390-zh-dark.png` | 登录错误态（暗红条） |
| `register-1440-zh-dark.png` / `register-1440-en-dark.png` | 注册，桌面中/英 |
| `register-390-zh-dark.png` / `register-390-en-dark.png` | 注册，移动中/英 |
| `models-1440-zh-dark.png` / `models-390-en-dark.png` | 模型广场跟随新 token/外壳，未整页重设计 |

### 截图数据说明

请求日志、每日用量与模型用量的截图数据来自隔离 HOME `/tmp/mr-dark-home` 中追加的 5 条测试记录（3 成功、1 失败、1 隔日成功），用于验证表格行高、状态点、错误行与图表密度；这些记录不写入仓库、不进入产品，也不代表真实业务数据。空态截图使用 `test@local`（0 Key、0 用量）拍摄。Dock 为 `position: fixed`，整页长截图会在视口底部重复出现，因此验收图以视口截图为准。
