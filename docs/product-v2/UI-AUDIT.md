# MiniRelay 全站 UI 审计（第一阶段）

> 审计对象：`/root/magpie-original`（工作树，`remove-partners` @ `e41adc151`）
> 审计日期：2026-11-23
> 阶段：只读审计。未修改任何产品代码、未提交、未推送。
> 配套规范：`docs/product-v2/DESIGN.md`（第二阶段产物）
> 截图证据：`docs/product-v2/screenshots/`（38 张，含 `report.json` / `report-admin.json`）

---

## 0. 先纠正一个前提

任务把本项目称作 **MiniRelay**，但源码里没有这个字符串：

```sh
grep -rn "MiniRelay\|minirelay" docs/ internal/   # 0 条
```

实际身份是 magpie 的一个分支：

| 项 | 实际值 | 位置 |
|---|---|---|
| 上游 | `yetone/magpie` | `.git/config` |
| 分支 | `remove-partners` | 本地 |
| HEAD | `e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e` | 本地 |
| 版本描述 | `v0.1.4574-1-ge41adc151` | `git describe` |
| 管理后台品牌 | 小写 `magpie` + 飞鸟 SVG | `internal/gui/assets/index-web.html:64-66` |
| 门户品牌 | 大写 `Magpie` + 渐变方块字母 `M` | `internal/gui/assets/portal/index.html:17-19` |
| 门户副标题 | `AI 模型中转平台` | 同上 `:19` |

**结论**：品牌名在代码里是 Magpie，不是 MiniRelay；且管理端与用户端用了两个不同的标志和两种大小写。这是本轮唯一需要你拍板的身份问题（见 §9 待确认 1）。本审计与 DESIGN.md 其余部分按"Magpie 为品牌、MiniRelay 为项目代号"处理，不擅自改名。

---

## 1. 结论摘要

| # | 结论 | 判定 |
|---|---|---|
| 1 | 管理端与门户是两套互不加载对方样式的独立设计系统：`app.css` 5703 行 / 86 个 token，`portal.css` 697 行 / 21 个 token | **事实**（源码注释明说） |
| 2 | 唯一贯穿两端的是强调色：`#4f46e5`（浅色）/ `#a5b4fc`（深色），完全一致 | **一致** |
| 3 | 中性色阶、字号基准、圆角尺度、控件高度、阴影、状态色六项全部不一致 | **不一致**（§5） |
| 4 | 门户不能跟随管理端设置的主题、强调色、字体、色盲模式，只跟随操作系统 | **能力缺失**（§5.7） |
| 5 | 门户硬编码文本语言为 `zh-CN`，无 i18n；管理端英文优先并支持 `zh/ja/de` | **能力缺失**（§5.8） |
| 6 | 门户 390px 宽度下"使用日志"表格横向溢出（文档宽 751px）、统计卡挤成 44px | **实测缺陷**（§7 P1） |
| 7 | 门户 23 处硬编码颜色绕过 token；深色模式靠 6 个重复的 media 块补丁 | **技术债**（§7 P2） |
| 8 | 管理端自身也有对比度不达标项（`--muted` 在卡片上 3.66:1），源码内有已知的局部补救 | **基线缺陷**（§7 P2） |
| 9 | 管理端已有现成的移动端规则（输入框 16px、控件 44px、统计卡 2 列），门户没有采用 | **可复用**（§5.9） |

---

## 2. 审计方法与真实证据

### 2.1 无头浏览器能力

| 项 | 结果 |
|---|---|
| 系统 Chromium | `/usr/bin/chromium`（可直接跑，root 需 `--no-sandbox`） |
| Playwright 浏览器缓存 | `~/.cache/ms-playwright/chromium-1254/chrome-linux64/chrome` ✅ |
| Playwright Node 包 | `~/.npm/_npx/86170c4cd1c5da32/node_modules/playwright` ✅ |
| Python Playwright | ❌ 驱动连接失败（版本错配），未使用 |
| 截图结论 | **真实截图已生成**，不是伪造或占位 |

### 2.2 运行环境

工作树带 71 项未提交改动（Phase 1–8），为了不改动仓库，采用隔离运行：

```sh
# 编译到 /tmp，不覆盖仓库内 ./magpie
cd /root/magpie-original
GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn \
CGO_ENABLED=0 go build -tags nogui -trimpath \
  -ldflags "-s -w -X main.version=v0.1.4574-audit" -o /tmp/magpie-audit-bin .
# 配置目录指向 /tmp，避免写入 ~/.config/magpie
XDG_CONFIG_HOME=/tmp/magpie-audit-home MAGPIE_WEB_KEY=<审计用 key> \
  /tmp/magpie-audit-bin web --addr 127.0.0.1:3431 --no-open
# 审计账号
XDG_CONFIG_HOME=/tmp/magpie-audit-home /tmp/magpie-audit-bin user add audit@example.com --name 审计账号 --role admin --password-stdin
```

访问路径：管理端 `/?k=<key>`；门户 `/portal/*`；登录走 `/api/auth/login`。

### 2.3 已实际访问的页面

| 页面 | 路由 | 匿名 | 登录后 | 截图 |
|---|---|---|---|---|
| 门户首页 | `/portal/` | 200 | 200 | light/dark × desktop/mobile |
| 登录 | `/portal/login` | 200 | 302→console | light/dark × desktop/mobile |
| 注册 | `/portal/register` | 200 | 302→console | light/dark × desktop/mobile |
| 控制台 | `/portal/console` | 302→login | 200 | light/dark × desktop/mobile |
| 模型广场 | `/portal/models` | 302→login | 200 | light/dark × desktop/mobile |
| 管理端模型页 | `/portal/admin-models` | 302→login | 200（admin）/403（user） | light/dark × desktop/mobile |
| 管理后台 Providers | `/?k=…` | 401 | 200 | light/dark × desktop + mobile |
| 管理后台 Routing / Gateway / Usage / Plugins / Masking | 同上（页内切换） | 401 | 200 | light/dark × desktop |

全部 38 张截图均无控制台错误、无页面级横向溢出（除 §7 P1-1 的证据图，那是缺陷本身）。

### 2.4 Oil UI Skill 使用证据

| 项 | 证据 |
|---|---|
| 安装位置 | `~/.zcode/skills/oil-ui/`，版本 `0.34.6` |
| 加载方式 | 本会话通过 Skill 工具正式调用 `oil-ui`（返回内容含 SKILL.md 全文） |
| 读取的规范文件 | `SKILL.md`、`references/light-path.md`、`references/task-review.md`、`references/workdir.md` |
| 遵循的流程 | 入口"UI/UX 评审"→ 只读评审协议：只交证据与建议，不改产品文件 |
| 使用其自带工具 | `scripts/shoot.mjs` 对门户登录页、管理端 Providers 页各跑一遍（含它自己的 lint） |
| 自带工具输出 | 门户登录页：输入框对比度 2.58:1 / 2.49:1 不达标；390px 下输入框 15px 会触发 iOS 缩放。管理端 Providers：10 处对比度提示（`3.33:1`，`nav-gl` 分组标签等）。两者均"无控制台错误、无横向溢出、无加载失败图片" |
| 版本检查 | `scripts/check_update.sh` 无输出（无需更新） |
| 自定义截图脚本 | `shoot.mjs` 不支持会话 cookie，门户登录后页面另写 Playwright 脚本 `/tmp/audit-tools/shoot-portal.cjs`（项目外，未入库） |

> 说明：`shoot.mjs` 的 lint 与本文档 §7 的对比度结论互相独立得出、数值一致，可交叉印证。

---

## 3. Git 保护

### 3.1 基线（任务开始时记录）

| 项 | 值 |
|---|---|
| 分支 | `remove-partners` |
| HEAD | `e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e` |
| 状态行数 | 71（40 个已跟踪修改 + 31 个未跟踪项） |
| 基线快照 | `/tmp/git-baseline.txt` |

### 3.2 结束时复核

```text
只有 baseline 有：无
只有 now 有：无
行数：baseline 71 / now 71
```

逐行 `comm` 比对为空差异，说明：

- 没有 checkout / reset / clean / stash / cherry-pick
- 没有 commit / push
- 没有新增、删除或修改任何已跟踪文件
- 本轮新增的只有 `docs/product-v2/screenshots/`（位于审计任务本来就要产出的 `docs/product-v2/` 之内，该目录此前已是未跟踪状态）
- 审计用的编译产物在 `/tmp/magpie-audit-bin`，配置在 `/tmp/magpie-audit-home`，均不在仓库内
- 服务进程：`/tmp/magpie-audit-bin web --addr 127.0.0.1:3431`，可用 `pkill -f magpie-audit-bin` 停止

---

## 4. Magpie 原生管理后台：从源码提取的规范

### 4.1 视觉特征

密集的桌面工具型界面：13px 基准字号、6–8px 控件圆角、白色面板直接压在浅灰底上、分组靠留白与 1px 发丝线而不是卡片阴影；强调色只出现在当前页、主操作和数据图上。

### 4.2 设计 token（浅色，`app.css:2-53`）

| token | 值 | 用途 |
|---|---|---|
| `--bg` | `#f4f4f6` | 页面底 |
| `--card` | `#ffffff` | 面板/卡片 |
| `--card-2` | `#f9f9fb` | 展开的编辑器、次级面板 |
| `--pill` | `#f1f1f4` | 胶囊按钮底、行悬停 |
| `--pill-hover` | `#e9e9ee` | 胶囊悬停 |
| `--seg-track` | `#e6e6eb` | 分段控件轨道 |
| `--seg-thumb` | `var(--card)` | 分段控件滑块 |
| `--seg-edge` | `0 1px 2px rgba(0,0,0,.14), 0 0 0 .5px rgba(0,0,0,.08)` | 滑块描边+阴影 |
| `--ctl-fg` | `#63636b` | 控件内的安静文字（刻意比 `--muted` 深，保证 4.5:1） |
| `--line` | `#e3e3e8` | 主分隔线/描边 |
| `--line-2` | `#ececf0` | 更浅的发丝线 |
| `--fg` | `#1c1c21` | 主文字 |
| `--fg-2` | `#4b4b53` | 次文字 |
| `--muted` | `#85858d` | 说明文字（源码注释承认真实场景需 4.5:1 时改用 `--ctl-fg`） |
| `--faint` | `#b9b9c2` | 占位/装饰 |
| `--accent` | `#4f46e5` | 强调色 |
| `--accent-soft` | `#eeedfd` | 强调底色 |
| `--accent-fg` | `#ffffff` | 强调色上的文字 |
| `--sel` | `#ecebfb` | 选中行底 |
| `--mark` | `var(--fg)` | 选中项的"墨色"，默认石墨 |
| `--green` / `--green-soft` | `#189a58` / `#e4f6ec` | 成功 |
| `--red` / `--red-soft` | `#d64545` / `#fbe9e9` | 失败/危险 |
| `--amber` / `--amber-soft` | `#b45309` / `#fbf0df` | 警示 |
| `--drift` / `--drift-soft` | `#ce2c31` / `#feebec` | "已偏离 magpie 管理" |
| `--pop-bg` | `rgba(255,255,255,.94)` | 浮层底（带模糊） |
| `--shadow` | `0 16px 44px rgba(20,22,30,.16), 0 2px 8px rgba(20,22,30,.06), 0 0 0 .5px rgba(20,22,30,.1)` | 浮层阴影 |
| `--font` | `-apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", system-ui, sans-serif` | 字族（**无 CJK 回退**） |
| `--code` | `ui-monospace, "SF Mono", Menlo, Consolas, monospace` | 等宽 |
| `--c1…--c7` | `#4f46e5 #0ea5e9 #10b981 #f59e0b #ec4899 #8b5cf6 #64748b` | 图表分类色 |

深色（`app.css:54-94`）：

| token | 值 |
|---|---|
| `--bg` | `#1a1a1e` |
| `--card` / `--card-2` | `#232327` / `#1f1f23` |
| `--pill` / `--pill-hover` | `#2d2d33` / `#36363d` |
| `--line` / `--line-2` | `#333339` / `#2c2c32` |
| `--fg` / `--fg-2` | `#ededf1` / `#b9b9c1` |
| `--muted` / `--faint` / `--ctl-fg` | `#9c9ca6` / `#90909a` / `#a1a1aa` |
| `--accent` / `--accent-soft` / `--accent-fg` / `--sel` | `#a5b4fc` / `#2b2c47` / `#14142a` / `#33345a` |
| `--green` / `--red` / `--amber` | `#6fd99b` / `#f28b82` / `#f2b544` |

### 4.3 排版与密度

| 项 | 值 | 位置 |
|---|---|---|
| 基准字号 | `13px/1.4` | `app.css:149` |
| 字号取值分布 | 11px(122) 11.5px(106) 12px(150) 12.5px(56) 13px(23) 10.5px(56) | `grep font-size` |
| 权重 | 400 / 500 / 600 | 全表 |
| 分区标签 | 11px / 600 / `letter-spacing .05em` / 大写 | `.label` `app.css:2455` |
| KPI 数值 | 20px / 600 / `letter-spacing -.01em` / tabular | `.kpi b` `app.css:1455` |
| KPI 名称 | 11px / 600 / 大写 / `--muted` | `.kpi span` `app.css:1458` |
| 表格 | 11.5px / th 10.5px 大写 / td 上下 6px | `.led` `app.css:1828-1830` |

### 4.4 布局

| 元素 | 尺寸 | 位置 |
|---|---|---|
| 侧边栏 | `--side-w: 212px`（<860px 收窄到 184px） | `app.css:435-436` |
| 顶栏高度 | `--top-h: 46px` | 实测 |
| 侧栏项 | 30px 高 / 圆角 7px / 13px / 左右 8px | `app.css:487-492` |
| 分段控件 | 轨道内 2px / 圆角 8px；按钮 24px 高 / 圆角 6px / 12.5px | `app.css:305-312` |
| 行 | web 40px / 窗口 48px，上下 5–7px | `app.css:767-768` |
| 主内容内边距 | `2px 16px 8px` | `app.css:719` |
| 桌面内容宽 | 1228px（1440 视口 − 212 侧栏） | 实测 |

### 4.5 组件规格

| 组件 | 规格 | 位置 |
|---|---|---|
| 图标按钮 | 26×26 / 圆角 7px / `--muted` | `app.css:318-324` |
| 列表磁贴 | 34px 高 / 圆角 8px / 悬停 `--pill` / 选中 `--sel` | `app.css:3652-3657` |
| 文字按钮 | 12px / 上下 3px 左右 7px / 圆角 6px | `app.css:2464` |
| 次级动作按钮 | 27px 高 / 1px `--line` / 圆角 6px / `0 1px 1px rgba(0,0,0,.04)` | `.text.action` `app.css:2466` |
| 主操作按钮 | `--accent` 底 + `--accent-fg` 字 / 500 | `.text.primary` `app.css:2468` |
| 输入框 | 27px 高 / 圆角 7px / 1px `--line` / 12.5px | `app.css:3825-3828` |
| 搜索框 | 24px 高 / 圆角 7px / 12px | `.find` `app.css:3633` |
| 卡片 | 1px `--line` / 圆角 10px / 内边距 10px 12px 11px / **无阴影** | `.subscription-card` `app.css:1128` |
| 面板 | 圆角 12px / 1px `--line` | `.list` `app.css:761` |
| 浮层 | 320px 宽 / 340px 高 / 圆角 11px / 24px 模糊 / `--shadow` | `.pop` `app.css:2565-2573` |
| 对话框 | `min(600px,100%)` / 圆角 14px / 1px ring + `0 24px 60px` | `app.css:4730-4732` |
| 遮罩 | `rgba(0,0,0,.28)` + `blur(8px) saturate(1.1)` | `app.css:4702-4703` |
| 胶囊标签 | 25px 高 / 999px / 12px / 1px `--line` | `.chip` `app.css:2472-2478` |
| 空状态 | 上下 36px / 左右 20px / 居中 / `--muted` | `.empty-state` `app.css:778` |
| 骨架屏 | `--pill`→`--pill-hover` 渐变 240% / 1.4s | `app.css:1396-1397` |

### 4.6 主题能力（管理端真实支持，门户缺失）

| 能力 | 实现 | 位置 |
|---|---|---|
| 浅/深/跟随系统 | `data-theme` + `prefers-color-scheme` | `app.css:54,95,115-116`；`app.js:19588-19600` |
| 强调色自选 | 7 个预设 + 系统取色器 | `appearance.js:10,34-76` |
| 强调色派生 | 运行时计算 `--accent/-soft/-fg/--sel/--mark/--mark-fill` | `appearance.go:190-206` |
| 色盲模式 | `data-colorblind` 与红绿点换成叉形 | `app.css:123-130`、`--x-mark` |
| 字体自选 | `--ui-font/-style/-weight/-stretch` | `app.css:149,1657-1658` |
| 减少动效 | `prefers-reduced-motion` 多段 | `app.css:483,526,596,644,718` |

### 4.7 国际化

- 默认英文（`index-web.html:2` `<html lang="en">`），142 处 `data-t` 由 `i18n.js` 翻译
- 词典覆盖 `zh` / `ja` / `de`（`i18n.js` 顶层键），未命中回退英文

### 4.8 响应式（真实存在，可直接复用）

| 断点 | 行为 | 位置 |
|---|---|---|
| ≥860px | 左侧栏 212px；<860px 收窄 184px | `app.css:435-436` |
| ≤760px（或粗指针 ≤1024px） | 管理端转底部标签栏（≤5 项，图标在上文字在下），`--tabbar: 50.5px + safe-area` | `app.css:389-398` |
| ≤760px | `body.web` 字号 14px；图标按钮 44×44 | `app.css:4805-4806` |
| ≤760px | **所有 input/textarea/select 16px、44px 高** | `app.css:4901-4903` |
| ≤760px | `.stats` / `.stats.six` 改 2 列 | `app.css:4886` |
| ≤760px | 对话框边距收到安全区 8px | `app.css:4905` |

---

## 5. MiniRelay 门户：现状

### 5.1 视觉特征

营销/消费型界面：15px 基准、10–16px 圆角、卡片带阴影、顶部导航 + 1160px 居中栏；深色只跟随系统。

### 5.2 token（浅色，`portal.css:5-27`）

| token | 值 |
|---|---|
| `--bg` / `--bg-2` | `#f6f7fb` / `#eef1f8` |
| `--card` / `--card-2` | `#ffffff` / `#fafbff` |
| `--fg` / `--fg-2` | `#191a2b` / `#34364c` |
| `--muted` / `--faint` | `#667085` / `#98a2b3` |
| `--line` / `--line-2` | `#e4e7ec` / `#d0d5dd`（注意：`--line-2` 比 `--line` **深**，与后端相反） |
| `--accent` / `--accent-2` | `#4f46e5` / `#6366f1` |
| `--accent-fg` / `--accent-soft` | `#ffffff` / `#eef2ff` |
| `--ok` / `--warn` | `#067647` / `#b54708` |
| `--shadow` | `0 1px 2px rgba(16,24,40,.04), 0 10px 30px rgba(16,24,40,.06)` |
| `--radius` / `--radius-sm` | `16px` / `10px` |
| `--font` | `-apple-system, …, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Noto Sans SC", sans-serif`（**含 CJK**） |
| `--mono` | `ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace` |

深色（`portal.css:29-47`）：`--bg #0f1016`、`--card #181a23`、`--fg #f2f4f7`、`--muted #98a2b3`、`--faint #667085`、`--accent #a5b4fc`、`--accent-2 #818cf8`、`--ok #75e0a7`、`--warn #fec84b`。

### 5.3 排版与密度

| 项 | 值 | 位置 |
|---|---|---|
| 基准字号 | `15px/1.6` | `portal.css:57` |
| 字号取值 | 11 / 11.5 / 12 / 12.5 / 13 / 13.5 / 14 / 15 / 16 / 17 / 19 / 20 / 24 | `grep` |
| 字重 | 550 / 600 / 650 / 700（非标准档位） | 全表 |
| 页面标题 | `clamp(1.8rem, 3vw, 2.2rem)` ≈ 35px | `.page-head h1` `portal.css:253` |
| 首屏标题 | `clamp(2rem, 4.6vw, 3.35rem)` ≈ 53.6px | `portal.css:63` |
| 小标签 | 12px / 700 / `--accent` / **非大写** | `.eyebrow` `portal.css:162` |

### 5.4 布局

| 元素 | 尺寸 | 位置 |
|---|---|---|
| 顶栏 | 内容 1160px / 内边距 12px 22px / 高 63px / 模糊 14px | `portal.css:88-97`；实测 63px |
| 内容栏 | `max-width:1160px` / `padding:0 22px 72px` | `portal.css:154` |
| 页面顶部间距 | 34px | `portal.css:155` |
| 首页特性 | 4 列 / 间距 14px | `portal.css:183-186` |
| 控制台网格 | 2 列 / 间距 14px | `portal.css:261-263` |
| 账号卡 | 3 列 / 间距 14px / 内边距 18px 20px | `portal.css:255-258` |
| 模型卡网格 | 3 列 → 980px 两列 → 760px 一列 | `portal.css:471-473,497-500,680` |

### 5.5 组件规格

| 组件 | 规格 | 位置 |
|---|---|---|
| 按钮 | 38px 高 / 圆角 10px / 14px 600 / 左右 15px | `portal.css:129-136` |
| 大按钮 | 46px 高 / 圆角 12px / 15px / 左右 20px | `portal.css:146` |
| 主按钮 | 强调底 + `0 6px 18px accent 28%` 光晕 | `portal.css:139-142` |
| 幽灵按钮 | `--card` 底 / 1px `--line-2` / 圆角 10px | `portal.css:144` |
| 输入框 | 42px 高 / 圆角 10px / `--card-2` 底 / 1px `--line-2` | `portal.css:229-234` |
| 下拉 | 34px（日志筛选）/ 42px（工具栏）/ 圆角 9px / 11px | `portal.css:412-415,687-690` |
| 卡片 | 1px `--line` / 圆角 16px / 阴影 `--shadow` | `portal.css:187-190` |
| 统计块 | 12px 13px / 圆角 12px / `--card-2` 底 | `portal.css:369-372` |
| 标签 | 上 2px 下 8px / 999px / 11px 650 | `portal.css:267-271` |
| 筛选胶囊 | 34px 高 / 999px / 13px 550 | `portal.css:464-468` |
| 日志行 | 6 列固定网格 `136px minmax(130,1.2fr) minmax(110,1fr) 74px 152px 62px` / 间距 10px / 13px | `portal.css:418-422` |
| 开关 | 38×22 / 滑块 16px | `portal.css:615-625` |
| 对话框 | `min(660px,100vw-32)` / 圆角 16px / 遮罩 `blur(2px)` | `portal.css:631-641` |
| 空状态 | 上下 54px / 左右 24px / 46px 图标块 | `portal.css:474-483` |
| 骨架屏 | 200% / 1.2s | `portal.css:575-584` |

### 5.6 主题

只有 `prefers-color-scheme`（6 处），无 `data-theme`，无强调色自选、无色盲模式、无字体设置。用户在管理端选了深色或自定义强调色，门户不会跟。

### 5.7 国际化

`<html lang="zh-CN">`（`portal/index.html:2`），全部文案硬编码中文，`portal.js` / `usage.js` / `models.js` / `admin-models.js` 内无任何 i18n 引用。

### 5.8 响应式

| 断点 | 行为 | 位置 |
|---|---|---|
| ≤980px | 特性 2 列、模型卡 2 列 | `portal.css:497-500` |
| ≤760px | 顶栏换行、导航独占一行可横向滚动、单列、标题缩小 | `portal.css:501-517` |
| ≤760px | 管理端模型表改卡片式（`am-row`） | `portal.css:680-696` |
| ≤760px | **没有**：统计 6 列改列、日志表改卡片、输入框 16px、触控 44px | 缺失 |

---

## 6. 差异矩阵

| 维度 | 管理端（基准） | 门户（现状） | 差距 | 处置 |
|---|---|---|---|---|
| 强调色 | `#4f46e5` / `#a5b4fc` | 同 | 0 | 保留 |
| 页面底（浅） | `#f4f4f6` 中性 | `#f6f7fb` 偏蓝 | ΔE 可见 | 统一到基准 |
| 主文字 | `#1c1c21` | `#191a2b` | 偏蓝 | 统一 |
| 说明文字 | `#85858d`（3.33:1） | `#667085`（4.65:1） | 门户更合规但不同源 | 统一到 `--ctl-fg #63636b`（5.42:1） |
| 最浅文字 | `#b9b9c2`（1.77:1） | `#98a2b3`（2.34:1） | 两者都不合规 | 限定为装饰；文字改用 quiet |
| 描边 | `#e3e3e8` / `#ececf0`（浅→更浅） | `#e4e7ec` / `#d0d5dd`（**浅→更深**） | 语义相反 | 统一为"line 深、line-soft 更浅" |
| 字族 | 无 CJK 回退 | 含 CJK | 中文渲染可能不同 | 统一为含 CJK 的栈 |
| 基准字号 | 13/1.4 | 15/1.6 | 密度差 15% | 双密度档位，见 DESIGN §3 |
| 层级 | 11/12/13/20 为主 | 12/13/15/17/24 | 两套 | 统一 10 档 |
| 圆角 | 6/7/8/10/11/12/14 | 8/9/10/11/12/16/18/24 | 门户更圆 | 语义化 5 档 |
| 控件高 | 24–30 | 34–46 | 门户更松 | 双密度档位 |
| 卡片 | 1px 线 + 圆角 10，无阴影 | 1px 线 + 圆角 16 + 阴影 | 两种卡片语言 | 统一：圆角 12 + 轻阴影 |
| 浮层 | 圆角 11 / 24px 模糊 / 重阴影 | 对话框圆角 16 / 2px 模糊 / 80px 阴影 | 不同 | 统一对话框规格 |
| 状态色 | green/red/amber + soft | ok/warn + 4 处硬编码红绿 | 两套语义名 | 统一为 ok/danger/warn |
| 图表色 | `--c1…--c7` | 只有 accent | 门户缺分类色 | 补 |
| 主题 | data-theme + 自选强调 + 色盲 + 字体 | 仅系统 | 能力缺失 | 门户对齐 |
| 语言 | en 默认 + zh/ja/de | zh-CN 硬编码 | 策略冲突 | 待确认（§9） |
| 导航 | 侧栏 212 / 移动底栏 | 顶栏 1160 / 移动换行 | IA 可不同 | 视觉规格统一 |
| 移动控件 | 16px / 44px / 统计 2 列 | 15px / 30–34px / 统计 6 列 | 门户缺移动规则 | 直接复用管理端 |
| 焦点环 | 2px accent / offset 1px | 2px / offset 2px，且 select/textarea 只改边框色 | 不完整 | 统一完整焦点环 |
| 动效 | `--ease` + 4 档时长 | `.12/.15s ease` | 曲线不同 | 统一曲线 |

---

## 7. 问题清单（按影响排序）

### P1 — 移动端布局失效（已实测）

**P1-1 门户"使用日志"在 390px 横向溢出**

- 现象：`.log-row` 是固定 6 列 `136 + 130 + 110 + 74 + 152 + 62` + 5×10px 间距，最小 714px；容器仅 316px。文档宽度被撑到 **751px**。
- 证据：`screenshots/evidence-log-table-overflow-390.png`（用与 `usage.js:169-190` 相同的 DOM 结构注入两行渲染后实拍），实测 `docScrollW 751 / winW 390`。
- 位置：`portal.css:418-422`；`usage.js:161-190`
- 说明：当前环境无真实请求，空态不触发；一旦有日志就会横向滚动整页。
- 修法（来自管理端已有做法）：≤760px 时把 `.log-row` 改成卡片式（照 `am-row` 的 `portal.css:680-696`），或给 `.log-list` 加容器内横向滚动。**建议卡片式**。

**P1-2 门户统计卡在 390px 挤成 44px 宽**

- 现象：`.usage-stats` 固定 6 列，390px 视口下每格 **44×117px**，标签「输入 Token」逐字换行。
- 位置：`portal.css:365-372`（无任何媒体查询覆盖）
- 管理端已有对应规则：`.stats / .stats.six → repeat(2, minmax(0,1fr))`（`app.css:4886`）
- 修法：≤760px 改 2 列（管理端同款），或 3 列 × 2 行。

**P1-3 门户时间范围胶囊在 390px 挤成 118×151 的窄条**

- 现象：`.usage-head` 是 `flex` 两端对齐，`.segs` 只剩 118px，4 个胶囊折成 3 行、共 151px 高。
- 位置：`portal.css:265`、`portal.css:362-364`
- 修法：≤760px 时 `.panel-head` 改纵向堆叠，`.segs` 占满整行。

### P2 — 一致性 / 可访问性 / 技术债

**P2-1 门户 23 处硬编码颜色绕过 token**

- 例：`#b42318`（5 次）、`#fef3f2`（3）、`#ecfdf3`（3）、`#12241a`（4）、`#fecdca`（4）、`#2a1717`（3）。
- 位置：`portal.css:214-215,240,243,273,275,348-352,376,432-437,440,555-560`
- 后果：状态色有 4 套不同的红/绿；改主题时改不全。

**P2-2 对比度不达标（两边都有，门户更严重）**

| 组合 | 实测 | AA(4.5:1) | 使用场景 |
|---|---|---|---|
| 门户 `--faint` on bg | **2.41:1** | ✗ | 统计标签、日志表头、图表轴、账号卡标签（11–13px） |
| 门户 `--faint` on card | **2.58:1** | ✗ | 同上 |
| 门户输入框占位符 | **2.58 / 2.49:1** | ✗ | 登录/注册/API Key（Oil UI lint 独立复现） |
| 管理端 `--muted` on bg | **3.33:1** | ✗ | 侧栏分组标签 `nav-gl`（11px，Oil UI lint 复现 3 处，共 10 处） |
| 管理端 `--muted` on card | **3.66:1** | ✗ | 卡片说明文字（源码 `app.css:1129-1135` 已承认并局部改用 `--ctl-fg`） |
| 管理端 `--faint` on card | **1.95:1** | ✗ | 占位/装饰 |
| 门户 `--muted` on card | 4.97:1 | ✓ | 说明文字 |
| 管理端 `--ctl-fg` on card | 5.95:1 | ✓ | 控件文字 |

- 结论：两端都**不能**把最浅两档用于正文级文字。管理端已有正确做法（卡片内把 `--muted` 重映射为 `--ctl-fg`），门户没有。

**P2-3 移动端输入框 15px → iOS 聚焦自动放大**

- 位置：`portal.css:229-234`（`font-size` 继承 body 15px）
- 管理端已解决：`body.web input/textarea/select { font-size:16px; min-height:44px }`（`app.css:4901-4903`）
- Oil UI lint 在 390px 下明确报出 2 处。

**P2-4 触控目标不足 44px**

- 实测：日志「刷新」46×30、「上一页」58×30、「下一页」58×30；管理端移动端图标按钮是 44×44（`app.css:4806`）。

**P2-5 门户不跟随管理端主题与强调色**

- 管理端支持 `data-theme` / 自选 accent / 色盲 / 字体；门户只有 `prefers-color-scheme`，且 accent 写死。
- 后果：同一产品在同一台机器上出现"后台深色、前台浅色""后台绿色强调、前台紫色强调"。

**P2-6 焦点可见性不完整（门户）**

- `.field input:focus`、`.log-filters select:focus`、`.search input:focus`、`.toolbar-select:focus`、`.modal textarea:focus` 都是 `outline:none` + 仅改边框色；键盘用户看不到 2px 焦点环（`portal.css:237,416,462,530,654`）。
- 管理端：`button:focus-visible, input:focus-visible { outline:2px solid var(--accent) }`（`app.css:191`）。

**P2-7 品牌与语言策略冲突**

- 两个 logo（飞鸟 SVG / 字母 M 渐变块）、两种大小写（`magpie` / `Magpie`）、两套语言（en 默认 / zh 硬编码）。
- 位置：`index-web.html:64-66`、`portal/index.html:17-19`。

### P3 — 细节

- **P3-1** 门户按钮主色带 `0 6px 18px` 光晕，管理端主按钮是平的（`portal.css:141` vs `app.css:2468`）。
- **P3-2** 圆角档位过多：门户 11 种、管理端 13 种；缺少语义化 token。
- **P3-3** 门户 `--line-2` 比 `--line` 深，管理端相反，同名不同义。
- **P3-4** 管理端 `--font` 不含 CJK，中文界面会落到系统默认字体，和门户不一致。
- **P3-5** 管理端用 4 种"分区标题"写法（11px 大写 muted、13px 600、`nav-gl`、`sheet .kind`），门户用 12px accent eyebrow，共 5 种。
- **P3-6** 深色模式实现方式：管理端靠 token 换值，门户靠 6 个 `@media` 块逐条覆盖，维护成本高且易漏。

---

## 8. 验证范围与限制（必须如实记录）

### 8.1 已真实访问并截图

- 门户 6 个页面 × 浅/深 × 1440/390 = 24 张
- 管理端 6 个视图 × 浅/深 + 1 张移动端 = 13 张
- 1 张日志溢出证据图
- 全部页面 HTTP 状态、`body[data-page]`、可见视图、token 计算值、横向溢出、文档高度写入 `screenshots/report.json` 与 `report-admin.json`

### 8.2 仅源码分析、未做视觉验证

| 内容 | 原因 |
|---|---|
| 模型卡（有数据态）、模型详情对话框 | 环境没有可用供应商，模型目录为空 |
| API Key 创建/一次性密钥展示 | 平台未开放模型，创建按钮禁用 |
| 用量图表（有数据）、模型用量条 | 无真实请求 |
| 管理端供应商编辑器、注册登录对话框、pop 浮层 | 无供应商可编辑 |
| 管理端 Usage 有数据态、会话页 | 无数据 |
| 门户 `admin-models` 有数据行 | 平台模型目录为空 |
| 管理端中文界面渲染 | 需切换 locale，未做 |

### 8.3 方法学限制

- 日志表溢出的证据图使用了**注入的合成行**（DOM 结构与 `usage.js` 产出的完全一致），因为环境内没有真实请求。这不是产品数据，仅用于测量布局，测试完即销毁（浏览器上下文关闭）。
- 对比度由 token 值计算（WCAG 相对亮度），并与 Oil UI 自带 lint 的输出交叉核对；未对每个实际像素取色。
- 圆角/字号分布用 `grep` 统计字符串，可能包含注释内的示例，量级可信、精确值以 token 表为准。

---

## 9. 待确认（进入改造前必须定）

1. **品牌名**：继续用 `Magpie / AI 模型中转平台`，还是改成 MiniRelay？两者影响 `<title>`、logo、`brand-sub`、页脚、i18n 词条与门户脚本内的品牌字符串。本审计未做任何改名。
2. **语言策略**：门户是否要接 i18n（跟随 `i18n.js` 的 en/zh/ja/de），还是明确声明为纯中文门面？当前是后者（硬编码 zh-CN），但管理端默认英文，用户前后台会看到两种语言。
3. **密度**：门户是否接受把基准字号从 15px 收到 14px、控件从 38–42px 收到 34–38px（更接近管理端），以换取更强的"同一产品"感？或保留双档位（后台紧凑 / 前台舒适）？DESIGN.md 采用**双档位**方案，这是建议值，不是源码事实。
4. **主题跟随**：门户是否跟随管理端的 `data-theme` 与自选强调色？技术上需要门户读取同一存储的偏好（当前门户连 admin 的 JS 都不加载），属于新增能力，需要你决定是否在下一轮实现。

---

## 10. 本轮没有做的事

- 没有修改任何 `.css` / `.js` / `.html` / `.go`
- 没有重构、没有引入组件库、没有动认证/API Key/Gateway/Provider/Routing/用量逻辑
- 没有提交、推送、切分支、stash
- 没有为"好看"而重新设计任何页面
- 没有把推断写成"原始设计规范"（DESIGN.md 逐项标注来源）

---

## 11. 证据索引

| 路径 | 内容 |
|---|---|
| `docs/product-v2/screenshots/portal-*-light/dark.png` | 门户 6 页桌面端 |
| `docs/product-v2/screenshots/portal-*-mobile-*.png` | 门户 6 页移动端 |
| `docs/product-v2/screenshots/admin-*.png` | 管理后台 6 视图 + 移动端 |
| `docs/product-v2/screenshots/evidence-log-table-overflow-390.png` | P1-1 缺陷证据 |
| `docs/product-v2/screenshots/report.json` | 门户逐页实测（token、组件、溢出、错误） |
| `docs/product-v2/screenshots/report-admin.json` | 管理后台逐视图实测 |
| `/tmp/git-baseline.txt` | Git 基线快照（临时，未入库） |
| `/tmp/oilui-evidence/` | Oil UI 自带截图工具的输出与 lint 报告（临时） |
| `/tmp/audit-tools/` | 自定义 Playwright 脚本（临时，项目外） |