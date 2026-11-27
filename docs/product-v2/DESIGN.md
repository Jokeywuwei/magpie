# MiniRelay 统一设计规范（DESIGN.md）

> 基准：Magpie 原生管理后台源码（`/root/magpie-original` @ `e41adc151`）
> 依据审计：`docs/product-v2/UI-AUDIT.md`
> 版本：**v1.3（2026-11-24）**，四项设计决策不变；v1.2 按 P1 收尾校正 Token，v1.3 按 UI 对齐收尾同步已实现的状态（§1.5 / §2.1 / §2.5 / §14 / §15）
> 状态：已落地（P0/P1 已验收，2026-11-24 UI 对齐收尾按本规范校正）；后续批次仍以本文件的功能/无障碍/响应式约束为准
> 视觉方向 v2（2026-11-24）：用户门户的视觉 token、导航与组件语言改由 [`DARK-UI-SYSTEM.md`](DARK-UI-SYSTEM.md) 统一；本文件继续约束业务结构、无障碍、响应式、主题注入、CSP、隐私与 i18n。两者冲突时，视觉以 DARK-UI-SYSTEM 为准，功能以本文件为准。

---

## 0. 怎么读这份规范

### 0.1 来源标记

| 标记 | 含义 | 可信度 |
|---|---|---|
| **【源】** | 直接从 Magpie 源码取值，附 `文件:行` | 可核对 |
| **【推】** | 由源码值归一、取整或合并档位 | 有判断成分，取值可商量 |
| **【建】** | 源码里不存在，本次新增的建议值 | 未经验证，需确认 |

不把推断写成原始设计。凡标【建】的，Magpie 源码里没有对应实现。

### 0.2 北极星

> **像一台仪器，不像一张海报。** 白面板直接压在浅灰底上，分组靠留白和发丝线，强调色只出现在当前位置、主操作和状态点上；门户沿用同一套色阶与控件，只把密度放宽一档。

### 0.3 已确认的四项决策（v1.1 起生效）

| # | 决策 | 落地含义 |
|---|---|---|
| D1 | **品牌 = `MiniRelay · AI 模型中转平台`**。Magpie 是底层源码与技术基础，不再作为最终产品主品牌 | 只改用户可见品牌；技术标识（模块路径、二进制、环境变量、配置目录、API 路径）一律不动，见 §1 |
| D2 | **语言接入现有 i18n.js**。门户中文默认、支持英文；管理后台保留现有语言能力与默认行为 | 同一套键约定与 `t()` 语义；门户用独立小词典，不加载 2.6 MB 的 `i18n.js`，见 §4 |
| D3 | **双档位密度**。管理后台 13px 紧凑，用户门户 15px 舒适，共用同一套 token | 只有一个 token 层，密度是"档位"参数而不是第二套系统，见 §3 |
| D4 | **门户接入三态主题与自定义强调色**，且不得破坏色盲辅助、字体设置 | 单一配置源 `settings.Settings`，门户服务端注入，见 §2.7 |

### 0.4 一句话规则

> 颜色一套、字号一套、圆角一套、控件一套、主题一份配置；**密度分两档**，语言分两档（zh/en），其余不允许分叉。

---

## 1. 品牌规范

### 1.1 名称

| 位置 | 规范 | 来源 |
|---|---|---|
| 产品名（显示） | `MiniRelay`（大小写固定，全大写 M、R） | D1【建】 |
| 副标（中文） | `AI 模型中转平台` | 【源】`portal/index.html:19` |
| 副标（英文） | `AI model relay platform` | 【建】 |
| 文档标题（门户） | `MiniRelay · AI 模型中转平台` / `MiniRelay · AI model relay platform` | 【源】`portal/index.html:9` + D1 |
| 文档标题（后台） | `MiniRelay`（标签页）；带实例名时 `{instance} · MiniRelay` | 【源】`index-web.html:10`、`api.go:1392-1396` + D1 |
| PWA / 主屏名称 | `MiniRelay` / `MiniRelay` | 【源】`manifest.json:2-3` + D1 |
| 页脚 | `MiniRelay · AI 模型中转平台` | 【源】`portal/index.html:363` + D1 |

大小写规则：**面向用户一律 `MiniRelay`**；不再出现全小写 `magpie` 作为产品名（后台现用的全小写 wordmark 属于源头风格，改品牌时一并改为 `MiniRelay`，【推】）。

### 1.2 标志

| 项 | 规范 | 来源 |
|---|---|---|
| 主标志 | 沿用后台的飞鸟 SVG（`index-web.html:64`），18×18，`currentColor` | 【源】 |
| 门户标志位 | 30×30 / 圆角 9px / `--accent-soft` 底 / 飞鸟 18px / `--accent` 着色；**停用字母 M 渐变块** | 【推】 |
| 图标资产 | `icons/apple-touch-icon.png`、`icons/icon-512.png` 沿用；是否换图不影响本轮 | 【源】 |
| 禁用 | 不用字母 M 方块、不用渐变填充标志、不给标志加阴影 | 【推】 |

> 备选：若认为飞鸟与 Magpie 关联过强，可改为中性的几何标记；本轮默认保留飞鸟，因为它是既有资产且 PWA 图标一致。

### 1.3 技术标识：**本轮明确不改**

改名会破坏兼容性与升级路径，以下保持 `magpie`：

- Go 模块路径 `github.com/yetone/magpie`
- 二进制 `magpie`、`magpie web` 等子命令
- 环境变量 `MAGPIE_WEB_KEY`、`MAGPIE_HOME`、`MAGPIE_DRAIN`、`MAGPIE_PUBLIC_URL`
- 配置目录 `~/.config/magpie`、缓存 `~/.cache/magpie`
- Cookie `magpie_web_<port>`、页面键参数 `?k=`
- API 路径 `/api/*`、插件页前缀、`/portal/*`
- CSS 变量名、DOM id/class（`--accent`、`.seg`、`#nav` 等）
- 插件与 agent 集成里的 `providers.magpie` 等字段

> 判断标准：**用户看得见的名字改，机器和协议里的名字不改。**

### 1.4 文案中的品牌句

- `portal/index.html:38` 的导语："由 magpie 负责供应商路由…" → "由 **MiniRelay** 负责供应商路由…"（【建】）
- `portal/index.html:16` 的 `aria-label="Magpie 首页"` → `"MiniRelay 首页"`（【建】）
- 后台 `web.go` 的登录/拒绝页标题 `magpie web` / `magpie` → `MiniRelay`（【建】，见实施计划 B1）

### 1.5 品牌审计面（实测）

| 位置 | 出现次数 | 说明 |
|---|---|---|
| `portal/index.html` | 7 | title、aria-label、wordmark、副标、eyebrow、导语、页脚 |
| `portal/portal.css` | 1 | 文件头注释（不影响显示） |
| `index-web.html` | 2 | `<title>`、`apple-mobile-web-app-title` + 1 处 wordmark |
| `index.html`（桌面壳） | 3 | 同上；非 Web 范围，可选同步 |
| `manifest.json` | 2 | `name`、`short_name` |
| `api.go:1392-1396` | 1 | 标签页标题 `{name} · magpie` |
| `web.go:389,391,522` | 3 | 登录页与"仅管理员"页 |

全仓库的**用户可见** `Magpie` 已由 P1 统一为 `MiniRelay`；技术标识按 §1.3 白名单保留 `magpie`。上表为 P0 前基线，现行品牌实现见 P1-REPORT / P1-CLOSURE-REPORT。

---

## 2. 颜色与主题变量

### 2.1 中性色（浅色）

| token | 值 | 用途 | 对比度（对 `--bg` / `--surface`） | 来源 |
|---|---|---|---|---|
| `--bg` | `#f4f4f6` | 页面底 | — | 【源】`app.css:3` |
| `--surface` | `#ffffff` | 面板、卡片、输入框 | — | 【源】`app.css:4` |
| `--surface-2` | `#f9f9fb` | 展开体、只读区 | — | 【源】`app.css:5` |
| `--surface-3` | `#f1f1f4` | 胶囊底、行悬停、禁用底 | — | 【源】`app.css:6` |
| `--surface-4` | `#e6e6eb` | 分段控件轨道 | — | 【源】`app.css:10` |
| `--line` | `#e3e3e8` | 主描边、行分隔 | — | 【源】`app.css:16` |
| `--line-soft` | `#ececf0` | 发丝线、卡内分隔 | — | 【源】`app.css:17` |
| `--text` | `#1c1c21` | 主文字 | 15.45 / — | 【源】`app.css:18` |
| `--text-2` | `#4b4b53` | 次文字、行内标签 | 7.87 / — | 【源】`app.css:19` |
| `--text-quiet` | `#63636b` | **说明文字、表单标签、占位符、控件的安静文字** | 5.42 / 5.95 ✅ | 【源】`app.css:15`（`--ctl-fg`）；`app.css:20`（`--muted`）同值 |
| `--text-muted` | `#63636b` | 与 `--text-quiet` 同值；后台 `--muted` 已并入此档，不再是更浅的一档 | 5.42 / 5.95 ✅ | 【源】`app.css:20`，见 §2.5 |
| `--text-faint` | `#6b6b74` | **三级弱文字**：占位符、11px 注释、图表轴；P1 收尾后达 AA | 4.80 / 5.28 ✅ | 【源】`app.css:21` |

> 命名映射（v1.3 补记）：本表的 `--surface*` / `--text*` 是门户 `portal.css` 的实际 token 名；后台 `app.css` 用 `--card` / `--card-2` / `--pill` / `--seg-track` 与 `--fg` / `--fg-2`，数值相同。后台没有独立的 `--text-muted`：`--muted` 现与 `--text-quiet` 同值 `#63636b`。门户 `--muted` 指向 `--text-quiet`，`--faint` 指向 `--text-faint` `#6b6b74`（2026-11-24 起不再并入 quiet，恢复三级文字层级）；`--text-muted` 仍保留声明但无引用。

### 2.2 强调色（浅色）

| token | 值 | 用途 | 来源 |
|---|---|---|---|
| `--accent` | `#4f46e5` | 主操作底、当前项、图表主线、链接 | 【源】`app.css:22` |
| `--accent-hover` | `brightness(1.06)` | 主操作悬停（滤镜） | 【源】`portal.css:295` |
| `--accent-soft` | `#eeedfd` | 强调底、图标底 | 【源】`app.css:23` |
| `--accent-on` | `#ffffff` | 强调底上的文字（6.29:1 ✅） | 【源】`app.css:24` |
| `--accent-selected` | `#ecebfb` | 选中行底 | 【源】`app.css:25` |

用户可自选强调色，运行时派生：`accentVars` 的 6 个变量（`--accent/-soft/-fg/--sel/--mark/--mark-fill`）、`tintVars` 的地面染色（`--bg/--card-2/--pill/--pill-hover/--seg-track/--line/--line-2`）与 `popVar` 的 `--pop-bg`（`appearance.go:190-206,54-86`；实测 `Accent=#0d9488` 时输出 `--accent:#0b7c72` 等 19 个变量）。派生规则必须保留：soft/selected 由强调色与底色混合，`--accent-on` 按黑/白对比度自动取（`appearance.go:192-195`）。**门户必须走同一派生，不得写死原色，也不得只取 accent 五项而丢掉表面染色**——否则自定义强调色下两端底色会分叉。

### 2.3 状态色：四种语义（三元组 + 描边）

四种：**成功 / 危险 / 警示 / 信息**。信息不新增色相，复用强调色系（【推】）。

| 组 | fill（点、条、边框） | text（文字、图标） | soft（底） | border | 来源 |
|---|---|---|---|---|---|
| 成功 | `#189a58` | `#067647`（页面 5.18 / 白底 5.69 / 软底 5.07 ✅） | `#e4f6ec` | — | fill/soft【源】`app.css:32-33`；text【源】`app.css:36`、`portal.css:36` |
| 危险 | `#d64545` | `#b42318`（页面 5.98 / 白底 6.57 / 软底 5.61 ✅） | 门户 `#fef3f2` / 后台 `#fbe9e9` | `#fecdca` | fill【源】`app.css:37`；text【源】`app.css:39`、`portal.css:39`；soft/border【源】`app.css:38`、`portal.css:40-41` |
| 警示 | `#b45309` | 门户 `#b54708`（白底 5.43 ✅）/ 后台 `#a54d08`（页面 5.22 / 白底 5.73 / 软底 5.08 ✅） | `#fbf0df` | — | fill/soft【源】`app.css:44-45`；text【源】`app.css:46`、`portal.css:43` |
| 信息 | `--accent` | `--accent` | `--accent-soft` | — | 【推】复用强调色，避免第五个色相 |

深色：成功 fill `#6fd99b` / text `#75e0a7` / soft `#1e3a2b`；危险 fill `#f28b82` / text `#fecdca` / soft 门户 `#2a1717`·后台 `#45272a` / border `#5b2a2a`；警示 fill `#f2b544` / text 门户 `#fec84b`·后台 `#f2b544` / soft `#3f3220`（【源】`app.css:85-93`、`portal.css:126-135,166-175`）。

> 为什么拆 fill 与 text：`#189a58` 当文字只有 3.62:1，`#067647` 有 5.69:1。同一语义的两个槽位，不是两个品牌色。

> P1 收尾后的两处实际分叉（都达 AA，不改变语义）：后台提示文字 `--amber-text #a54d08` 与门户 `--warn-text #b54708`；危险 soft 后台 `#fbe9e9`（深 `#45272a`）、门户 `#fef3f2`（深 `#2a1717`）。

### 2.4 深色

| token | 浅色 | 深色 | 来源 |
|---|---|---|---|
| `--bg` | `#f4f4f6` | `#1a1a1e` | 【源】`app.css:61` |
| `--surface` | `#ffffff` | `#232327` | 【源】`app.css:62` |
| `--surface-2` | `#f9f9fb` | `#1f1f23` | 【源】`app.css:63` |
| `--surface-3` | `#f1f1f4` | `#2d2d33` | 【源】`app.css:64` |
| `--surface-4` | `#e6e6eb` | `rgba(255,255,255,.035)` | 【源】`app.css:66` |
| `--line` | `#e3e3e8` | `#333339` | 【源】`app.css:69` |
| `--line-soft` | `#ececf0` | `#2c2c32` | 【源】`app.css:70` |
| `--text` | `#1c1c21` | `#ededf1` | 【源】`app.css:71` |
| `--text-2` | `#4b4b53` | `#b9b9c1` | 【源】`app.css:72` |
| `--text-quiet` | `#63636b` | `#9c9ca6`（5.76:1 ✅） | 【源】`app.css:78` |
| `--text-muted` | `#63636b` | `#9c9ca6`（5.76:1 ✅，与 quiet 同值） | 【源】`app.css:78` |
| `--text-faint` | `#6b6b74` | `#9c9ca6`（5.03–5.76:1 ✅） | 【源】`app.css:79` |
| `--accent` | `#4f46e5` | `#a5b4fc` | 【源】`app.css:81` |
| `--accent-soft` | `#eeedfd` | `#2b2c47` | 【源】`app.css:82` |
| `--accent-on` | `#ffffff` | `#14142a`（9.06:1 ✅） | 【源】`app.css:83` |
| `--accent-selected` | `#ecebfb` | `#33345a` | 【源】`app.css:84` |
| `--pop-bg` | `rgba(255,255,255,.94)` | `rgba(38,38,43,.96)` | 【源】`app.css:50,96` |

### 2.5 颜色使用规则

1. 文字只用 `--text` / `--text-2` / `--text-quiet` / `--text-faint`。P1 收尾后 `--faint` 达 AA（浅 `#6b6b74` 4.80:1、深 `#9c9ca6` 5.76:1），是三级文字（占位符、11px 注释、图表轴），不再是装饰专用；`--text-muted` 在后台已与 `--text-quiet` 同值，不再单独使用。
2. 最小 11px；11px 文字必须使用达 AA 的 token（`--text-quiet` 或 `--text-faint`）。
3. 卡片内说明文字继承 `--text-quiet`——Magpie 已有做法：`.subscription-card { --muted: var(--ctl-fg) }`（`app.css:1141`）。
4. 强调色不用于大面积填充；一屏一个主操作、一个当前项。
5. 状态色不混用槽位：点用 fill，成功文字用 text，底用 soft。
6. 图表分类色沿用后台 7 色（`app.css:57` / 深色 `:100,114`）：
   `#4f46e5 #0ea5e9 #10b981 #f59e0b #ec4899 #8b5cf6 #64748b`
   深色：`#a5b4fc #7dd3fc #6ee7b7 #fcd34d #f9a8d4 #c4b5fd #94a3b8`
7. 色盲模式必须把分类色切到 Okabe–Ito 调色板，并给曲线加不同虚实（【源】`app.css:127-138`、`--c1…--c7`）——门户有分类色后同样适用。

### 2.6 主题机制（源码事实）

| 能力 | 实现 | 来源 |
|---|---|---|
| 三态 | `data-theme="light"｜"dark"｜(缺省=跟随系统)` | 【源】`app.css:54,95,115` |
| 首屏防闪 | `boot.js` 写入 `window.bootPrefs`，`theme.js` 在样式生效前设 `data-theme` | 【源】`api.go:1276`、`theme.js:3-11` |
| 强调色 | `boot.js` 注入 `<style id="lookCSS">`，内容来自 `lookCSS(s)` | 【源】`api.go:1397-1399`、`appearance.go:54-86` |
| 原生控件 | `color-scheme` 跟随主题 | 【源】`app.css:114-116` |
| 减少动效 | `data-motion="reduce"` + `prefers-reduced-motion` | 【源】`theme.js:7,15-26`、`app.css:4798` |
| 色盲 | `data-colorblind="1"` | 【源】`theme.js:6`、`app.css:123-130` |
| 字体 | `--ui-font` / `--ui-style` / `--ui-weight` / `--ui-stretch` | 【源】`app.css:149` |
| 优先级 | `?theme=` > 保存值 > 系统 | 【源】`theme.js:3-6` |

### 2.7 主题配置：单一来源与跨页面复用（D4）

**单一来源**：`settings.Settings`（`internal/settings/settings.go:65-95`）。相关字段：

| 字段 | 取值 | 用途 |
|---|---|---|
| `Theme` | `system｜light｜dark` | 三态主题 |
| `Accent` | `#rrggbb` / 空 | 自定义强调色 |
| `Lang` | `system｜en｜zh` | 语言（见 §4） |
| `Colorblind` | bool | 色盲辅助 |
| `ReduceMotion` | bool | 减少动效 |
| `NoBird` | bool | 骨架屏上的鸟（仅后台） |
| `UIFont` / `CodeFont` | `fonts.Face`（Family/Weight/Style/Stretch） | 字体 |
| `TextSize` | 百分比 | 后台缩放 |
| `Wallpaper` / `WallpaperPath()` | 时间戳/文件 | **仅后台**，见下 |

写入全部在管理员侧（设置页 `POST /api/settings`、外观 `POST /api/appearance/wallpaper`、备份恢复等，统一走 `settings.Save`）。**门户不新增任何写入口**，因此不会出现第二份配置。

**门户读取方式（服务端注入，【建】）**：

| 注入项 | 方式 | 原因 |
|---|---|---|
| `<html lang>` | `portalServed` 替换 | 首屏语言正确 |
| `data-theme` | `portalServed` 在 `<html>` 上写入（非 `system` 时） | 首屏主题正确、无闪烁 |
| `data-colorblind` / `data-motion` | 同上 | 与后台同一套属性 |
| 强调色（含表面染色） | 新增 `GET /portal/look.css`，复用 `lookCSS` 的强调色块（accent 6 变量 + tint 表面 + pop-bg），`<link>` 排在 `portal.css` 之后 | 门户 CSP 是 `style-src 'self'`（`portal.go:38`），**内联 `<style>` 会被拦**；外部同源 CSS 合规范。颜色派生必须与后台逐字一致 |
| JS 需要的小开关 | 服务端注入 `<script type="application/json" id="portal-prefs">` | 不写内联脚本（CSP `script-src 'self'`） |

**三条硬约束**：

1. **门户不输出壁纸**。`lookCSS` 会带用户私有背景图（`/api/appearance/wallpaper`），门户是公开页，必须剥离，只保留强调色与表面染色。这是隐私边界，不是裁剪。
2. **门户不复用 `/api/appearance/css`**。该路由不在 `publicAPI`（`web.go:464`）；实测匿名 401、普通账号 403、管理员 200（`web.go:485-500`）。
3. **不改写 settings 的写入协议**。门户只读，后台仍独占写。

**已定义的近似**：`?theme=` / `?lang=` 在门户上作为只读覆盖（与后台 `theme.js` 的 `?theme=` 一致，【推】），用于验收截图与排障，不持久化。

**多用户限制（必须记录）**：MiniRelay 有多个平台账号，而 `settings.Settings` 是本机管理员的偏好。本轮方案 = 门户复用这份设置，即"平台外观由管理员决定"；不做按账号外观。若将来要按账号，需在 `identity` 下加偏好存储，届时门户优先读会话用户、回退到全局（【建】，不在本轮）。

**不得破坏的既有能力**（D4 验收项）：`Colorblind` 切换图表调色板与曲线虚实；`UIFont/CodeFont` 改变界面与等宽字体；`ReduceMotion` 停掉过渡与骨架动画；三者门户必须表现一致。

---

## 3. 字体、字号与层级（D3 双档位）

### 3.1 字族

| 角色 | 规范 | 来源 |
|---|---|---|
| 无衬线（界面） | `-apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", system-ui, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Noto Sans SC", sans-serif` | 【推】后台 `app.css:47` + 门户 CJK 回退 `portal.css:19` 合并 |
| 等宽 | `ui-monospace, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace` | 【源】`app.css:48` + `portal.css:20` |

> **CJK 回退是合并，不是新字体**：后台栈缺 CJK，中文会落到浏览器默认；补上 `PingFang SC / Hiragino Sans GB / Microsoft YaHei / Noto Sans SC` 后 macOS 仍先命中 SF Pro Text，拉丁字符渲染不变。

### 3.2 档位

**一个 token 层，两个密度档**：`compact`（管理后台）/ `comfortable`（用户门户）。档位由**页面**决定，不由用户决定。

| 角色 token | compact | comfortable | 字重 | 行高 | 用途 | 来源 |
|---|---|---|---|---|---|---|
| `--fs-display` | — | `clamp(2rem, 4.6vw, 3.35rem)` | 700 | 1.2 | 仅首页首屏 | 【源】`portal.css:63` |
| `--fs-h1` | 22 | `clamp(1.8rem, 3vw, 2.2rem)` | 700 | 1.2 | 页面标题 | 后台 22【推】（`.led-strip .v` `app.css:1570`）；门户【源】`portal.css:253` |
| `--fs-h2` | 20 | 24 | 650/700 | 1.2 | 区块标题 | 后台 20【推】；门户【源】`portal.css:64` |
| `--fs-h3` | 16 | 20 | 600 | 1.3 | 卡片标题 | 后台【源】`app.css:4693`；门户 16【源】`portal.css:65`，20【推】 |
| `--fs-lead` | 15 | 17 | 400 | 1.6 | 导语 | 【源】`portal.css:179` |
| `--fs-body-lg` | 14 | 16 | 400 | 1.5/1.6 | 长段落 | 【源】`app.css:4805`、`portal.css` |
| `--fs-body` | 13 | 15 | 400 | 1.4 / 1.6 | 正文基准 | 【源】`app.css:149`、`portal.css:57` |
| `--fs-meta` | 12 | 13 | 400/500 | 1.4/1.5 | 次要说明 | 【源】 |
| `--fs-caption` | 11.5 | 12.5 | 500 | 1.4 | 卡片脚注 | 【源】`app.css` 11.5 |
| `--fs-micro` | 11 | 12 | 600 | 1.35 | 分区标签、KPI 名、表头 | 【源】`app.css:2455,1458` |

### 3.3 规则

1. 同一角色只允许上表两个值，不得临时插值（【推】）。
2. 分区标签：11px / 600 / 字距 `.05em` / 大写（【源】`app.css:2455`）。中文无大小写 → 11px / 600 / 字距 `.02em` / 不强转（【建】）。
3. 数字 `font-variant-numeric: tabular-nums`（【源】）。
4. 行高：紧凑 1.4、舒适 1.6；**中文正文 ≥15px 时行高 ≥1.5**（【建】）。
5. 字重只保留 `400 / 500 / 600 / 700`（【推】）。门户现用 550/650 归到 500/600。
6. 标题字距 `-.02em`，数值 `-.01em`（【源】）。
7. 门户字体若由 `settings.UIFont` 指定，只覆盖 `--font` 家族，不改变上表字号。

### 3.4 密度（控件与间距的档位）

| 项 | compact | comfortable | 来源 |
|---|---|---|---|
| 控件高（按钮/输入） | 27–30 | 38–42（首屏 CTA 46） | 【源】`app.css:2466,3826` / `portal.css:131,230,146` |
| 图标按钮 | 26 | 44（移动端一律 44） | 【源】`app.css:318,4806` |
| 列表行 | 40 | 44+ | 【源】`app.css:767` |
| 卡片内距 | 10–12 | 20 | 【源】`app.css:1128` / `portal.css:189` |
| 页面底留白 | 8 | 56–72 | 【源】`app.css:719` / `portal.css:154,506` |
| 栅格间距 | 6–8 | 12–14 | 【源】 |

> 两档只在"高度与内距"上分叉；颜色、圆角、边框、阴影、字号层级、状态语义完全共用。

---

## 4. 语言与翻译资源（D2）

### 4.1 键约定（沿用 `i18n.js`）

| 约定 | 规则 | 来源 |
|---|---|---|
| 键 | **英文源串本身**（不是 `login.title` 这类符号键） | 【源】`i18n.js:2-4` |
| 缺键 | 回退英文 | 【源】`i18n.js:22212-22216` |
| 插值 | `{name}` 槽位，`t(s, vars)` 替换 | 【源】`i18n.js:22213-22215` |
| 静态文本 | `data-t`（文本）/ `data-tp`（placeholder）/ `data-tt`（title）；首次翻译把英文缓存进 `data-en` / `data-en-placeholder` | 【源】`i18n.js:22291-22307` |
| 语言选择 | `?locale=` > 保存值 > 系统 | 【源】`i18n.js:22165-22167` |
| 系统映射 | `zh-Hant → zh-TW`，`zh → zh`，`ja`，`de`，其余 `en` | 【源】`i18n.js:22176-22179` |
| `<html lang>` | 随语言更新（`LANG_TAGS`） | 【源】`i18n.js:22172-22173` |

### 4.2 资源分层

| 层 | 文件 | 语言 | 体量 | 谁加载 |
|---|---|---|---|---|
| 后台主词典 | `internal/gui/assets/i18n.js` | `zh` / `ja` / `de`（en 为键） | 2.6 MB | 后台页面（`index-web.html:626`） |
| 插件词典 | `plugins/<name>/i18n.js`，如 `plugins/routing/i18n.js` | 同左 | 小 | 插件页 |
| **门户词典（新增）** | `internal/gui/assets/portal/i18n.js` | `zh`（en 为键） | 预计 30–40 KB | 门户页面，排在 `portal.js` 之前 |

**门户不加载 2.6 MB 的 `i18n.js`**：那是后台全部页面、插件、工具提示的词条，门户用不到 5%。用独立小词典 + 相同键约定，才是"接入现有体系"而不是"复制一份体系"。

### 4.3 语言决议

| 场景 | 结果 | 依据 |
|---|---|---|
| 门户 `Lang=system` | **中文**（`zh`） | D2 |
| 门户 `Lang=en` | 英文 | D2 |
| 门户 `Lang=zh` | 中文 | D2 |
| 后台 `Lang=system` | 现有行为不变（系统语言 → `en/zh/ja/de`） | D2 |
| 门户 `?lang=en｜zh` | 只读覆盖，用于验收/排障 | 【推】 |

> 差异是刻意的：门户面向中文用户，默认中文；后台是工具，保持系统语言。两者共用同一字段，不引入第二套语言配置。

### 4.4 覆盖范围

门户共 **226 个唯一可翻译串**（实测：`portal/index.html` 159 个文本节点去重后 120 个，四个脚本的中文字符串字面量去重后 126 个，两者无重叠）。

| 文件 | 处理 |
|---|---|
| `portal/index.html` | 中文文案 → 英文键 + `data-t` / `data-tp` / `data-tt`；`<title>` 与 `lang` 服务端注入 |
| `portal.js` | 生成文案改调 `t()`；错误文案、状态名、时间格式走 `t()` 与 `Intl` |
| `usage.js` / `models.js` / `admin-models.js` | 同上（状态名、空态、单位） |
| 新增 `portal/i18n.js` | `zh` 全量；`en` 空对象（键即英文） |

### 4.5 防漂移

- 共享键（如 `Refresh`、`Cancel`、`Save`、`Loading…`）**英文必须与 `i18n.js` 完全一致**，zh 译文一致。
- 加一条 Go 测试：遍历门户词典的每个键，若 `i18n.js` 里存在同键且语言为 `zh`，则译文必须相同；不一致即失败（【建】）。
- 门户新增词条只在 `portal/i18n.js`；后台词条不动。

### 4.6 不做的

- 门户本轮不做 `zh-TW` / `ja` / `de`（字典结构留位，加语言只是补 key）。
- 不引入 i18n 框架或运行时库。
- 不改 `i18n.js` 的生成方式（当前是静态资源，无构建步骤）。

---

## 5. 间距与布局

### 5.1 间距刻度

基数 **4px**，2px 只用于行内微调（【推】）。可用值：`2 4 6 8 10 12 14 16 20 24 32 40 48 56 72`。

| 语义 | 值 | 用途 | 来源 |
|---|---|---|---|
| `--sp-1` | 2 | 文字基线微调、图标与文字 | 【推】 |
| `--sp-2` | 4 | 图标内边距、紧凑列表 | 【源】gap 4（83 处） |
| `--sp-3` | 6 | 行内元素间距、标签内距 | 【源】gap 6（117 处） |
| `--sp-4` | 8 | **控件间距、列表项内距** | 【源】gap 8（117 处） |
| `--sp-5` | 10 | 卡片内块间距 | 【源】 |
| `--sp-6` | 12 | 卡片内距（紧凑）、组间距 | 【源】 |
| `--sp-7` | 16 | 区块间距（紧凑） | 【源】 |
| `--sp-8` | 20 | **卡片内边距（舒适）** | 【源】`portal.css:362` |
| `--sp-9` | 24 | 区块间距（舒适） | 【源】 |
| `--sp-10` | 32 | 大区块 | 【推】 |
| `--sp-11` | 40 | 首屏纵向留白下限 | 【推】 |
| `--sp-12` | 56 | 页面底部留白（移动） | 【源】`portal.css:506` |
| `--sp-13` | 72 | 页面底部留白（桌面） | 【源】`portal.css:154` |

### 5.2 布局尺寸

| 元素 | 值 | 来源 |
|---|---|---|
| 后台侧栏宽 | `212px`；`<860px` 时 `184px` | 【源】`app.css:435-436` |
| 后台顶栏高 | `46px` | 【源】`--top-h` |
| 后台内容内边距 | `2px 16px 8px` | 【源】`app.css:719` |
| 门户内容宽 | `max-width 1160px`，左右 `22px` | 【源】`portal.css:154` |
| 门户页面顶部 | `34px` | 【源】`portal.css:155` |
| 门户底栏内距 | `22px`，字号 13px | 【源】`portal.css:488-490` |
| 栅格间距 | 卡片 14 / 步骤 12 / 统计 10 | 【源】`portal.css:185,204,367` |
| 后台列表行高 | web `40` / 窗口 `48` | 【源】`app.css:767-768` |

### 5.3 栅格

| 场景 | 桌面 | ≤980 | ≤760 |
|---|---|---|---|
| 门户首页特性 | 4 列 | 2 列 | 1 列 |
| 门户步骤 | 3 列 | 3 列 | 1 列 |
| 模型卡 | 3 列 | 2 列 | 1 列 |
| 控制台账号卡 | 3 列 | 3 列 | 1 列 |
| 统计块 | 6 列 | 6 列 | **2 列**（照 `app.css:4886`） |
| 后台供应商磁贴 | 5 列 | 5 列 | 1 列 |

前三行与末行【源】`portal.css:497-517,680-696`；统计块 2 列【推】。

---

## 6. 圆角、边框、阴影

### 6.1 圆角阶梯（5 档 + 胶囊）

| token | 值 | 用途 | 来源 |
|---|---|---|---|
| `--r-xs` | `4px` | 微型标签、内嵌代码块 | 【源】4px（23 处） |
| `--r-sm` | `6px` | 分段控件、文字按钮、小徽标 | 【源】`app.css:312,2464` |
| `--r-ctl` | `8px` | **按钮、输入框、下拉、导航项、磁贴** | 【推】后台 7/8、门户 10 归一 |
| `--r-card` | `12px` | **卡片、面板、浮层** | 【源】`app.css:761,1564` |
| `--r-dialog` | `14px` | 对话框 | 【源】`app.css:4730` |
| `--r-hero` | `16px` | 首屏/大尺度品牌面 | 【推】门户原 24（`portal.css:171`），收一档 |
| `--r-pill` | `999px` | 状态标签、筛选胶囊、开关 | 【源】`app.css:2475`、`portal.css:268` |

阶梯规则：`r-ctl < r-card < r-dialog < r-hero`，单调递增；`r-xs`/`r-sm` 只用于行内与微控件。**同一容器内的子元素圆角必须 ≤ 父级**（【建】）。

### 6.2 边框

| 规格 | 值 | 来源 |
|---|---|---|
| 主描边 | `1px solid var(--line)` | 【源】 |
| 发丝线 | `1px solid var(--line-soft)` | 【源】 |
| 选中描边 | `1px solid var(--accent)` 或只换底 | 【源】`portal.css:470` |
| 焦点环 | `outline: 2px solid var(--accent); outline-offset: 2px` | 【推】后台原 1px |
| 输入聚焦 | `border-color: var(--accent); box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 22%, transparent)` | 【源】`app.css:3831` |
| 错误描边 | `1px solid #fecdca`（深色 `#5b2a2a`） | 【源】`portal.css:240,243` |

> 分组先用空间，最后才用线；留白能分开的不补线；同一处分隔不画两次。

### 6.3 阴影（4 档）

| token | 值 | 用途 | 来源 |
|---|---|---|---|
| `--shadow-0` | `none` | 密集列表里的卡片 | 【源】`app.css:1128` |
| `--shadow-1` | `0 1px 2px rgba(16,24,40,.04), 0 10px 30px rgba(16,24,40,.06)` | 门户卡片 | 【源】`portal.css:24` |
| `--shadow-2` | `0 16px 44px rgba(20,22,30,.16), 0 2px 8px rgba(20,22,30,.06), 0 0 0 .5px rgba(20,22,30,.1)` | 浮层、picker | 【源】`app.css:46` |
| `--shadow-3` | `0 0 0 1px var(--line), 0 24px 60px rgba(0,0,0,.32), 0 2px 8px rgba(0,0,0,.12)` | 对话框 | 【源】`app.css:4732` |
| 遮罩 | `rgba(0,0,0,.28)` + `blur(8px) saturate(1.1)` | 对话框背后 | 【源】`app.css:4702-4703` |

规则：同屏只允许一档阴影；列表卡片不投影，只有浮起来的才投影。

### 6.4 克制的首屏光晕

| 规则 | 值 | 来源 |
|---|---|---|
| 允许位置 | **仅首页首屏的主 CTA**，每屏最多 1 个 | 【推】，源自 `portal.css:141` |
| 光晕 | `0 6px 18px color-mix(in srgb, var(--accent) 28%, transparent)` | 【源】 |
| 禁止 | 次级按钮、行内按钮、表格行、弹窗按钮、列表卡片 | 【推】 |
| 悬停 | 提亮 6%，不加第二层光晕 | 【源】 |
| 深色 | 同参数（accent 变 `#a5b4fc`，光晕随之变浅） | 【推】 |
| 减少动效 | 光晕不变（不是动效） | 【源】 |

> 光晕是整份规范里唯一"非工具感"的元素，只给"用户第一次点进来的那一下"。超过一处就退化成海报。

---

## 7. 按钮

### 7.1 变体

| 变体 | 底 | 字 | 描边 | 阴影 | 来源 |
|---|---|---|---|---|---|
| `primary` | `--accent` | `--accent-on` | 无 | 无（首屏 CTA 可有 §6.4 光晕） | 【源】`app.css:2468`、`portal.css:139-142` |
| `secondary` | `--surface` | `--text-2` | `1px --line` | `0 1px 1px rgba(0,0,0,.04)` | 【源】`app.css:2466` |
| `ghost` | 透明 | `--text-2` | `1px --line` | 无 | 【推】 |
| `text` | 透明 | `--text-quiet` | 无；悬停 `--surface-3` | 无 | 【源】`app.css:2464` |
| `danger` | 透明 | `--danger-text` | 无；悬停 `--danger-soft` | 无 | 【源】`app.css:2470` |
| `icon` | 透明 | `--text-muted` | 无；悬停 `--surface-3` + `--text` | 无 | 【源】`app.css:318-324` |

### 7.2 尺寸

| 尺寸 | compact | comfortable | 左右内距 | 字号 | 圆角 | 用途 |
|---|---|---|---|---|---|---|
| `sm` | 26–27 | — | 7–10 | 12 | `--r-sm` | 行内动作、图标按钮 |
| `md` | 30–34 | 34 | 10–12 | 13 | `--r-ctl` | 后台主按钮、门户筛选 |
| `lg` | — | 38 | 15 | 14 / 600 | `--r-ctl` | 门户主按钮 |
| `xl` | — | 46 | 20 | 15 / 600 | 12 | 仅首屏 CTA |
| `icon-lg` | — | 44×44 | — | — | `--r-ctl` | 移动端图标按钮 |

来源【源】`app.css:2464-2466,318,487,4806`、`portal.css:131-132,146`；圆角归一为【推】。

### 7.3 状态

| 状态 | 规则 | 来源 |
|---|---|---|
| hover | primary 提亮 6%；secondary 描边转 `--text-muted`、底转 `--surface-2`；ghost 描边转 `--accent` | 【源】 |
| active | `translateY(1px)`（门户）或底色加深（后台） | 【源】`portal.css:138` |
| focus-visible | `2px solid --accent`，offset 2px | 【推】 |
| disabled | `opacity .55` + `not-allowed`，不投影 | 【源】`portal.css:148` |
| loading | 文字不动，前置 14px 旋转图标，禁点 | 【建】 |

---

## 8. 输入框、下拉框、开关

### 8.1 输入框

| 尺寸 | compact | comfortable | 圆角 | 用途 |
|---|---|---|---|---|
| `sm` | 27 | — | `--r-ctl` | 后台行内编辑 |
| `md` | — | 34 | `--r-ctl` | 门户筛选、搜索 |
| `lg` | — | 42 | `--r-ctl` | 门户表单 |
| 移动 | — | ≥44，字 **16** | `--r-ctl` | 所有输入 |

来源【源】`app.css:3826,4901-4903`、`portal.css:230,413`；圆角 7/10→8【推】。

| 属性 | 规范 | 来源 |
|---|---|---|
| 底 | `--surface` | 【源】后台；门户原 `--card-2`，改为一致【推】 |
| 描边 | `1px solid --line` | 【源】`app.css:195,3827` |
| 占位符 | `--faint`（浅 `#6b6b74` 4.80:1 / 深 `#9c9ca6` 5.76:1） | 【源】`app.css:2597,3643,3840,4400`、`portal.css:387`；P1 收尾前为 `--text-quiet` |
| 悬停 | `color-mix(in srgb, var(--accent) 35%, var(--line))` | 【源】`portal.css:236` |
| 聚焦 | 描边 `--accent` + `0 0 0 3px accent 22%` | 【源】`app.css:3831` |
| 错误 | 描边 `--danger-text` + 12px 错误文字 | 【建】 |
| 只读/禁用 | 底 `--surface-2`，字 `--text-muted` | 【推】 |

### 8.2 下拉框

- 与同尺寸输入框同高、同圆角、同描边；右侧 16px chevron，右内距 32px（【推】）。
- 整块可点，不只箭头（【源】`app.css:4885` 用 `::after` 撑高）。
- 门户沿用原生 `<select>` + 自绘箭头（【源】`portal.css:412`）。

### 8.3 开关

| 规格 | 值 | 来源 |
|---|---|---|
| 尺寸 | 38×22，滑块 16px，左右 3px | 【源】`portal.css:616-621` |
| 关 / 开 | 轨道 `--line-2` / `--accent`；滑块右移 16px | 【源】 |
| 禁用 | `opacity .5` + `wait` | 【源】`portal.css:626` |
| 语义 | `role="switch"` + `aria-checked` | 【源】`portal/index.html:268` |

### 8.4 复选框 / 单选

- 16×16，圆角 `--r-xs`，`accent-color: var(--accent)`（【源】`portal.css:656`）。
- 标签与控件间距 8px，整行可点（【源】`.checkline`）。

---

## 9. 卡片、表格、弹窗

### 9.1 卡片

| 属性 | 密集（后台） | 舒适（门户） | 来源 |
|---|---|---|---|
| 底 | `--surface` | `--surface` | 【源】 |
| 描边 | `1px --line` | `1px --line` | 【源】 |
| 圆角 | `--r-card` | `--r-card`（门户原 16，收一档【推】） | 【源】`app.css:761` |
| 内距 | `10px 12px 11px` | `20px 18px` | 【源】`app.css:1128` / `portal.css:189` |
| 阴影 | `--shadow-0` | `--shadow-1` | 【源】 |
| 卡内分块 | `1px --line-soft` | 同 | 【源】`app.css:769` |

一屏同类卡片不超过两组；靠大图与连续排版建立气氛的区块不强塞卡片。

### 9.2 表格

统一采用后台 `.led`（【源】`app.css:1828-1834`）：

| 属性 | 值 |
|---|---|
| 字号 | 11.5（compact）/ 13（comfortable） |
| 表头 | 10.5px / 600 / 大写 / 字距 `.04em` / `--text-quiet` / 左对齐 / 不换行 |
| 单元格 | 上下 6px、左右 9px；行间 `1px --line-soft` |
| 行悬停 | `--surface-2` |
| 数字列 | 右对齐 + `tabular-nums` |
| 首行 | 无上边框 |

门户"使用日志"改为上表规格；**≤760px 必须卡片化**（照 `portal.css:680-696` 的 `am-row` 做法，每行一卡、字段带中文/英文标签）。

### 9.3 弹窗

| 属性 | 值 | 来源 |
|---|---|---|
| 宽度 | `min(600px, 100%)`；编辑器 `min(max(600px,56vw),880px,100%)` | 【源】`app.css:4730-4734` |
| 圆角 | `--r-dialog` | 【源】 |
| 阴影 | `--shadow-3` | 【源】 |
| 遮罩 | `rgba(0,0,0,.28)` + `blur(8px) saturate(1.1)` | 【源】`app.css:4702-4703` |
| 头/脚 | 各 48px，`1px --line-soft`，内容垂直居中 | 【源】`app.css:4744-4746` |
| 体 | 内距 `14px 18px`，只体滚动 | 【源】`app.css:4740-4742` |
| Esc / 归还焦点 | Esc 关闭并归还焦点；破坏性对话框点遮罩不关闭 | 【建】 |
| 门户 | 现用原生 `<dialog>`，660/16/模糊 2 → 改为上表（【推】） | `portal.css:631-641` |

---

## 10. 导航栏与侧边栏

### 10.1 后台（侧边栏）

| 属性 | 值 | 来源 |
|---|---|---|
| 宽 | 212 / <860px 184 | 【源】`app.css:435-436` |
| 顶栏 | 46px；品牌在左，动作图标 26px | 【源】 |
| 分组标签 | 11px / 500 / 字距 `.04em` / `--text-quiet` | 【源】`app.css:485` + 对比度修正【推】 |
| 项 | 30px / `--r-ctl` / 13px / 左内距 8 / 图标 14 | 【源】`app.css:487-492`（圆角 7→8【推】） |
| 选中 | `--surface` 底 + `--shadow-1` + 字重 500 + `--text` | 【源】`app.css:494` |
| 悬停 | `--surface-3` + `--text` | 【源】 |
| 底栏 | 设置 + Gateway 状态点（7px + `--ok-soft` 光环） | 【源】`app.css:627` |

### 10.2 门户（顶部导航）

| 属性 | 值 | 来源 |
|---|---|---|
| 栏高 | 63px（12px 上下 + 38px 控件） | 【源】实测 |
| 底 | `color-mix(bg 82%, transparent)` + `blur(14px)` + `1px --line` | 【源】`portal.css:89-94` |
| 项 | 30–36px / `--r-ctl` / 14px / 500 | 【源】`portal.css:116-119`，字重归 500【推】 |
| 选中 | `--accent-soft` 底 + `--accent` 字 | 【源】`portal.css:464-470` 同型 |
| 悬停 | `--surface-2` + `--text` | 【源】 |
| 品牌 | 30px 标志块 + `MiniRelay` + 副标（≤760 隐藏副标） | 【源】`portal.css:103-114` + D1 |

> 两处选中态不同（后台白卡浮起 / 门户浅强调底）是有意的容器差异：侧栏在灰底上要浮起，顶栏在模糊底上要染色。同一容器内不得混用。

### 10.3 移动端导航

| 断点 | 后台 | 门户 |
|---|---|---|
| ≤760 | **底部标签栏** 50.5px + 安全区，≤5 项，图标在上 | 【源】`app.css:389-398`；门户建议同构（【建】） |
| 触控 | 图标按钮 44×44【源】`app.css:4806` | 所有可点 ≥44px（【建】，现状 30px） |

---

## 11. 状态：加载、空、错误、成功、信息

### 11.1 加载

| 场景 | 规范 | 来源 |
|---|---|---|
| 骨架块 | `--surface-3`→`--surface-4`，240%，1.4s 往返 | 【源】`app.css:1396-1397`（门户 200%/1.2s 统一到此） |
| 局部刷新 | 容器 `opacity .55` | 【源】`portal.css:375,449` |
| 图标旋转 | 0.9s linear infinite | 【源】`app.css:331` |
| 减少动效 | 停骨架与过渡 | 【源】`app.css:4798` |
| 禁止 | 整页白屏 spinner；用"加载中…"文字替代骨架 | 【建】 |

### 11.2 空状态

| 属性 | 值 | 来源 |
|---|---|---|
| 内距 | 36/20（舒适可 54/24） | 【源】`app.css:778` / `portal.css:476` |
| 对齐 | 居中，最大宽 460 | 【源】`portal.css:483` |
| 图标 | 46×46 / `--r-dialog` / `--accent-soft` 底 / `--accent` 图标 | 【源】`portal.css:478-481` |
| 标题 | 16px / 700 / `--text` | 【源】`portal.css:482` |
| 说明 | 13–13.5px / `--text-quiet` | 【源】+ §2.5 |
| 动作 | 最多一个次级按钮 | 【源】 |

### 11.3 错误

| 层级 | 规范 | 来源 |
|---|---|---|
| 表单级 | `--danger-text` 字 / `--danger-soft` 底 / `1px #fecdca` / `--r-ctl` / 9px 11px / 13px | 【源】`portal.css:238-241` |
| 字段级 | 描边 `--danger-text` + 12px 同色文字 | 【建】 |
| 页面级 | 同表单级，置于标题下 | 【建】 |
| 行内 | `--danger-text` 12px，不套盒 | 【源】`portal.css:440` |
| 无障碍 | `role="alert"` | 【源】`portal/index.html:96` |

### 11.4 成功 / 状态标签

| 标签 | 底 | 字 | 来源 |
|---|---|---|---|
| 成功 | `#e4f6ec`（深 `#1e3a2b`） | `#067647`（深 `#75e0a7`） | 【源】 |
| 失败 | `#fef3f2`（深 `#2a1717`） | `#b42318`（深 `#fecdca`） | 【源】`portal.css:432-437` |
| 取消 | `--surface-2` | `--text-muted` | 【源】`portal.css:434` |
| 形状 | 2px 8px / `--r-pill` / 11px 650 / 不换行 | 同左 | 【源】`portal.css:267-271` |

### 11.5 信息

- 用 `--accent` 字 + `--accent-soft` 底，形状同上（【推】）。
- 状态点必须带形状（成功圆点 / 失败叉形），不能只靠红绿——后台为色盲做的既有决定（`app.css:36-39`）。

---

## 12. 响应式规则

### 12.1 断点

| 断点 | 名称 | 行为 |
|---|---|---|
| ≥1200 | 桌面宽 | 后台侧栏 212 + 满宽；门户 1160 居中 |
| 981–1199 | 桌面窄 | 同上门户特性 4 列 |
| 861–980 | 平板横 | 后台仍侧栏；门户特性/模型卡 2 列 |
| 761–860 | 平板竖 | 后台侧栏收 184；门户导航仍一行 |
| ≤760 | 手机 | 后台转底部标签栏；门户单列 |

来源：`app.css:435-436,389`、`portal.css:497-517`；`861–980`/`761–860` 为合并后的【推】。

### 12.2 ≤760px 的强制规则

| 组件 | 规则 | 来源 |
|---|---|---|
| 输入/下拉/文本域 | 字号 **16px**、高 ≥44px | 【源】`app.css:4901-4903` |
| 图标按钮 | 44×44 | 【源】`app.css:4806` |
| 统计块 | 6 列 → **2 列** | 【推】`app.css:4886` |
| 表格 | 行 → 卡片，字段带标签 | 【源】`portal.css:680-696` |
| 时间范围胶囊 | 独占一行、可横滚，不折竖条 | 【推】修审计 P1-3 |
| 对话框 | `100vw - 16px`，边距 8px | 【源】`app.css:4905` |
| 页头 | 标题与动作纵向堆叠 | 【源】`portal.css:514` |
| 底部留白 | 门户 56；后台 `--tabbar` + 安全区 | 【源】 |

### 12.3 禁止项

- 整页横向滚动（审计 P1-1）。
- 可点区域 <44px（P2-4）。
- 输入框字号 <16px（P2-3）。

---

## 13. 交互状态与动效

### 13.1 必须覆盖的状态

交互组件：`default / hover / active / focus-visible / disabled`；数据区域：`loading / empty / error / success / info`（【推】，源自后台覆盖范围）。

### 13.2 动效

| 项 | 值 | 来源 |
|---|---|---|
| 标准曲线 | `cubic-bezier(.2, .8, .2, 1)` | 【源】`app.css:4667` |
| 出场曲线 | `cubic-bezier(.23, 1, .32, 1)` | 【源】 |
| 快 / 中 / 慢 | 120 / 180–220 / 260–300ms | 【源】 |
| 页面切换 | 240ms 淡入 | 【源】`app.css:705` |
| 门户现值 | `.12/.15s ease` → 统一到上表 | 【推】 |
| 减少动效 | 全部过渡 0，骨架停 | 【源】 |

### 13.3 键盘与输入

- Tab 顺序跟随视觉顺序；焦点环必须可见（2px accent）。
- 对话框打开后焦点进入首个可交互元素，Esc 关闭并归还（【建】）。
- 开关 `role="switch"` + 空格切换（【源】）。

---

## 14. 落地映射表（施工单）

| 位置 | 现值 | 目标 | 类型 |
|---|---|---|---|
| `portal.css:6` `--bg` | `#f6f7fb` | `#f4f4f6` | 换值 |
| `portal.css:7` `--bg-2` | `#eef1f8` | `--surface-3 #f1f1f4` | 换值 |
| `portal.css:11` `--fg` | `#191a2b` | `#1c1c21` | 换值 |
| `portal.css:12` `--fg-2` | `#34364c` | `#4b4b53` | 换值 |
| `portal.css:13` `--muted` | `#667085` | `#63636b`（实现为 `--muted: var(--text-quiet)`） | 换值 + 改名 |
| `portal.css:14` `--faint` | `#98a2b3` | `#6b6b74`（实现为 `--faint: var(--text-faint)`；2026-11-24 起恢复三级文字，不再并入 quiet） | 换值 + 改用法 |
| `portal.css:15-16` `--line/-2` | `#e4e7ec` / `#d0d5dd` | `#e3e3e8` / `#ececf0` | 换值 + 语义反转 |
| `portal.css:25-26` `--radius` | 16 / 10 | 12 / 8 | 换值 |
| `portal.css:19` `--font` | 含 CJK | 合并后的同一栈 | 换值 |
| `portal.css:57` body | `15px/1.6` | 保留（comfortable） | 保留 |
| `portal.css:118` 导航字重 | 550 | 500 | 换值 |
| `portal.css:467` 胶囊字重 | 550 | 500 | 换值 |
| `portal.css:189` 卡片圆角 | 16（`--radius`） | 12 | 换值 |
| `portal.css:639` 对话框圆角 | 16 | 14 | 换值 |
| `portal.css:240-243` 错误色 | 硬编码 4 色 | 语义 token | 提 token |
| `portal.css:432-437` 状态标签 | 硬编码 | 语义 token | 提 token |
| `portal.css:555-560` 协议 chip | 硬编码 | 语义 token + 深色成对 | 提 token |
| `portal.css:365-368` 统计 6 列 | 无移动规则 | ≤760 转 2 列 | 加规则 |
| `portal.css:418-422` 日志 6 列 | 固定网格 | 表格规格 + ≤760 卡片化 | 重写 |
| `portal.css:229-234` 输入框 | 42/10/card-2 | 42/8/surface/16@mobile | 换值 |
| `portal.css:237,416,462,530,654` | `outline:none` | 补焦点环 | 加规则 |
| `portal.css` 主题 | 仅系统 | 三态 + 强调色 + 色盲 + 减少动效（服务端注入） | 新增能力（§2.7） |
| `portal/index.html` 品牌 | `Magpie` ×3 + `magpie` ×1 | `MiniRelay` | D1 |
| `portal/index.html` 文案 | 中文硬编码 | 英文键 + `data-t` + 词典 | D2 |
| `manifest.json` | `magpie` ×2 | `MiniRelay` | D1 |

后台侧最初只改三处：`--font` 加 CJK、焦点偏移 1→2、11px 标签从 `--text-muted` 提到 `--text-quiet`；P1 收尾又上调了 `--muted` / `--faint` 与状态文字 token（见 §2.1、§2.3、§2.4），其余不动。

---

## 15. 验收清单（可自动检查）

1. 门户样式表 token 块外的十六进制颜色为 0（2026-11-24 实测：0；代码块与开关滑块已走 `--code-*` / `--switch-knob`）。
2. 所有文字对比度 ≥4.5:1（≥18.66px 粗体 ≥3:1）。
3. 390px 下 `document.documentElement.scrollWidth === 390`（2026-11-24 实测：中英 320/390 均 0 溢出）。
4. 390px 下所有可点元素高 ≥44px（2026-11-24 实测：门户 390 与触屏 768 均 0 个小于 44px 的目标）。
5. 390px 下所有 `input/select/textarea` 字号 ≥16px（2026-11-24 实测：16px）。
6. 每页只有一个 `primary` 按钮；光晕只在首屏 CTA（≤1 处）。
7. 圆角只出现在 `{4,6,8,12,14,16,999}`。
8. 字号只出现在 §3.2 档位内（compact 或 comfortable 各自集合）。
9. 门户无内联 `<style>` / `<script>`（CSP `style-src/script-src 'self'`）。
10. 主题：`Theme=dark` 时门户首屏即深色（服务端注入，无闪烁）；`Accent=#0d9488` 时门户强调色一致；`Colorblind` 影响图表；`ReduceMotion` 停动效。
11. 语言：`Lang=en` 时门户全英文、无中文残留；`Lang=zh` / `system` 时中文。
12. 品牌：全站无 `Magpie` / `magpie` 作为产品名（技术标识除外，见 §1.3 白名单）。
13. Oil UI `shoot.mjs` 的 lint 无对比度/缩放提示。
14. 后台与门户 1440px 并排截图，主色、中性色、圆角、控件高度一致。

---

## 16. 待确认

| # | 事项 | 本规范选择 | 需确认 |
|---|---|---|---|
| 1 | 标志 | 保留飞鸟 SVG，门户停用字母 M 块 | 是否接受飞鸟作为 MiniRelay 标志，或换中性几何标记 |
| 2 | 后台 wordmark 大小写 | 改 `MiniRelay` | 是否保留后台全小写风格（`minirelay`） |
| 3 | 门户语言 | `system` → 中文 | 是否给门户加语言切换开关（footer，【建】） |
| 4 | 多账号外观 | 复用管理员设置（全局） | 是否接受"平台外观由管理员决定"；若否，需按账号偏好存储 |
| 5 | 门户卡片圆角 | 16 → 12 | 视觉上是否接受 |
| 6 | 状态色四槽位 | fill/text 分列 | 是否同意保留两套色值 |
| 7 | 技术标识 | 全部保留 `magpie` | 确认二进制/环境变量/配置目录不改名 |

---

## 17. 本规范明确不做

- 不引入 UI 组件库、CSS 框架或 i18n 运行时库
- 不改业务逻辑、接口、认证、API Key、Gateway、Provider、Routing、用量统计
- 不改 `settings` 的写入协议，不改 `MAGPIE_*`、二进制名、模块路径
- 不删除门户既有优点（空态图标、骨架屏、跳过链接、CSP）
- 不在本轮落地代码

---

## 18. 引用索引

| 文件 | 引用处 |
|---|---|
| `internal/gui/assets/app.css` | §2–§13（主基准） |
| `internal/gui/assets/portal/portal.css` | §2 状态色、§3/§5 门户档位 |
| `internal/gui/assets/index-web.html` | §1 品牌、§10 侧栏 |
| `internal/gui/assets/portal/index.html` | §1 品牌、§8 开关、§11.3 无障碍 |
| `internal/gui/assets/manifest.json` | §1 品牌 |
| `internal/gui/appearance.go` | §2.2 强调色派生、§2.7 注入 |
| `internal/gui/api.go:1276-1399` | §2.6 boot.js 与 lookCSS 注入 |
| `internal/gui/web.go:455-530` | §2.7 守卫与公开路由 |
| `internal/gui/assets/theme.js` | §2.6 首屏应用顺序 |
| `internal/gui/assets/i18n.js` | §4 全部键约定与 `setLocale`/`t`/`translateStatic` |
| `internal/settings/settings.go:65-95` | §2.7 配置字段 |
| `docs/product-v2/UI-AUDIT.md` | 全部结论的来源与实测数据 |

---

## 19. 修订记录

### v1.3 — 2026-11-24（UI 对齐收尾）

按 2026-11-24 的真实浏览器验收同步实现状态，不改变四项设计决策：

1. **状态**：文档从"草案、不落地代码"改为"已落地"；§1.5 的"全仓库无 MiniRelay"改为 P1 已落地、技术标识按 §1.3 保留。
2. **Token**：§2.1 记录门户 `--faint` 恢复为 `--text-faint #6b6b74`（不再并入 quiet）；§14 同步该实现。
3. **验收值**：§15 第 1/3/4/5 项改为 2026-11-24 实测（0 处 token 块外 hex、320/390 中英 0 溢出、0 个 <44px 触控目标、输入 16px）。
4. **组件**：门户弹窗改 600px / 14px 圆角 / 28% 遮罩 + 8px 模糊；主按钮光晕只留首页首屏；焦点环覆盖链接、按钮、输入、下拉、文本域；后台控件圆角统一到 8px、卡片 12px，深浅两路 `--tk-c` 统一为 `#9c9ca6`。
5. **移动端**：后台设置分区标签 44px；供应商/Agent 拖拽把手 28px 外观 + 48px 命中区；门户 320px 英文 Key 行换行、移动端 `linkish` 44px；空用量图收高。

### v1.1 — 2026-11-23（本轮）

按四项正式设计决策修订：

1. **D1 品牌**：新增 §1，明确 `MiniRelay · AI 模型中转平台`；列出全部品牌出现点（门户 8、后台 5、Go 4）；新增"技术标识不改"白名单。
2. **D2 语言**：新增 §4，明确键约定（英文键、`{slot}`、缺键回退英文）、`data-t/-tp/-tt`、语言决议（门户 `system→zh`，后台不变）、226 串覆盖范围、门户独立小词典（不加载 2.6 MB `i18n.js`）、共享键防漂移测试。
3. **D3 密度**：§3 明确双档位 token 表与"档位由页面决定"；新增 §3.4 密度对照；圆角收为阶梯（§6.1）。
4. **D4 主题**：新增 §2.7，写清单一来源 `settings.Settings`、写入全在管理员侧（门户不新增写入口）、门户服务端注入（`data-theme`/`lang`/`look.css`）、CSP 约束（门户 `style-src 'self'` 禁止内联）、隐私边界（门户不输出壁纸）、不复用管理员专属 `/api/appearance/css`（实测 401/403/200）、多账号限制。
5. **规范细化**：状态色收为四种语义（新增"信息"，复用强调色）；首屏光晕限制为每屏 1 处；CJK 回退写进字族；圆角阶梯加"子元素 ≤ 父级"规则。
6. **验收**：§15 从 12 项扩到 14 项，新增主题/语言/品牌三类可脚本检查项。

### v1.0 — 2026-11-23

首版，基于 `UI-AUDIT.md` 的源码提取与实测。

**与 v1.0 的差异要点**：品牌从"待定"变为已定；语言从"未定"变为已定并新增完整章节；主题从"建议能力"变为已定并补齐注入方案；新增圆角阶梯与光晕约束；状态色由三元组扩为四语义。