# P1 交付报告：品牌、Token、主题与语言

> 批次：P1
> 日期：2026-11-23
> 依据：`docs/product-v2/DESIGN.md` v1.1、`UI-IMPLEMENTATION-PLAN.md`、`P0-REPORT.md`
> 状态：**已完成并自验，未提交**。P2 未开始。

---

## 1. 结果摘要

| 项 | 结果 |
|---|---|
| 品牌 | 门户 8 处、管理端外壳 3 处、manifest 2、Go 4 处已改为 MiniRelay；飞鸟 SVG 成为门户标志 |
| Token | 门户采用管理端同一套色阶/圆角阶梯/状态色/阴影/字族，并保留 15px 舒适密度 |
| 主题 | 门户由服务端注入 `data-theme` / `data-colorblind` / `data-motion`，强调色经 `/portal/look.css` 复用同一派生 |
| 隐私 | 壁纸与 UIFont 从不进入门户 CSS；门户也不请求 `/api/appearance/*` |
| i18n | 门户独立词典 243 键；中文默认、英文可切；缺失键回退、插值、Cookie 持久、`?lang=` 临时覆盖 |
| 布局 | 320/390/768/1440 × 中英：无横向溢出；移动/触屏 0 个 <44px 目标；输入 16px |
| 测试 | `go test ./internal/gui` **ok 93.870s**；新增词典防漂移测试通过；Oil UI lint 全清 |

---

## 2. Git 保护

### 2.1 开工基线（P1 前）

| 项 | 值 |
|---|---|
| 分支 | `remove-partners` |
| HEAD | `e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e` |
| `git status` | 71 行 |
| `git diff --stat` | `40 files changed, 1018 insertions(+), 466 deletions(-)` |

### 2.2 交付时

| 项 | 值 |
|---|---|
| 分支 / HEAD | 未变：`remove-partners` / `e41adc151` |
| `git status` | **75 行**（43 已跟踪修改 + 32 未跟踪） |
| `git diff --stat` | `43 files changed, 1054 insertions(+), 480 deletions(-)` |
| 暂存区 | 0 |
| 新增的已跟踪修改（相对基线） | `internal/gui/appearance.go`、`internal/gui/assets/manifest.json`、`internal/gui/instance_name_test.go`、`internal/gui/web_app_test.go` |
| 其余已跟踪修改 | 都是 Phase 1–8 既有的文件（`app.css`、`app.js`、`api.go`、`web.go` 等），本轮在其上追加 |
| 新增未跟踪 | `internal/gui/portal_i18n_test.go`；`assets/portal/bird.svg`、`assets/portal/i18n.js`（`assets/portal/` 本就是未跟踪目录） |

未执行 checkout / reset / clean / stash / cherry-pick / commit / push。

### 2.3 为什么有两个测试被改

品牌改名后，两处断言仍写着旧名，属**预期更新**而不是放宽测试：

- `instance_name_test.go:61`：标签页标题 `Tokyo box · magpie` → `Tokyo box · MiniRelay`
- `web_app_test.go:102`：manifest `name` 断言 `magpie` → `MiniRelay`

---

## 3. 实际修改文件

行数 = 相对 P1 开工快照的增删合计。

| 文件 | 变更 | 原因 |
|---|---|---|
| `assets/portal/portal.css` | ~350 行 | Token 层重写、深色块收敛、状态色提 token、圆角阶梯、CJK 字族、语言开关样式、日志卡片标签 |
| `assets/portal/index.html` | ~338 行 | 全量英文键 + `data-t/-ta/-tp`；品牌；飞鸟标志；页脚语言开关；favicon；i18n.js 引入 |
| `assets/portal/portal.js` | ~165 行 | `t()` 化、日期/数字按语言、语言切换（Cookie + 保留地址） |
| `assets/portal/usage.js` | ~36 行 | 图表/日志/分页文案 `t()` 化，`Intl` 按语言 |
| `assets/portal/models.js` | ~60 行 | 模型卡/详情/示例/空态 `t()` 化，示例问候语本地化 |
| `assets/portal/admin-models.js` | ~49 行 | 管理表/开关/编辑弹窗 `t()` 化，移动卡片标签改 `data-label` |
| `assets/portal/i18n.js`（新） | 326 行 | 门户词典 243 键 + `t()`/`applyPortalI18n()` |
| `assets/portal/bird.svg`（新） | 4,967 B | 管理端飞鸟轮廓，供 CSS mask 着色 |
| `gui/portal.go` | ~75 行 | `look.css` 路由、页面注入、语言决议、i18n.js 静态清单 |
| `gui/appearance.go` | ~27 行 | `lookCSS` 拆为 `accentCSS` + `wallpaperCSS`；新增 `portalLookCSS` |
| `assets/index-web.html` | 6 行 | 标题、主屏名、wordmark 改 MiniRelay |
| `assets/manifest.json` | 4 行 | `name` / `short_name` |
| `assets/app.css` | 6 行 | `--font` 加 CJK；焦点环 offset 1→2；`nav-gl` 提到 `--ctl-fg` |
| `assets/app.js` | 2 行 | 标签页标题 `· magpie` → `· MiniRelay` |
| `gui/api.go` | 2 行 | boot.js 标题后缀 |
| `gui/web.go` | 6 行 | 登录页与"仅管理员"页标题、`<h1>` |
| `gui/portal_i18n_test.go`（新） | 105 行 | 键覆盖 + 与 i18n.js 的译文一致性 |
| 测试断言 | 各 2 行 | 见 §2.3 |

---

## 4. Token 层与主题架构

### 4.1 共享 Token（同名同值，两份样式表各自声明）

| 组 | Token |
|---|---|
| 表面 | `--bg #f4f4f6`、`--surface #fff`、`--surface-2 #f9f9fb`、`--surface-3 #f1f1f4`、`--surface-4 #e6e6eb` |
| 线 | `--line #e3e3e8`、`--line-soft #ececf0` |
| 文字 | `--text #1c1c21`、`--text-2 #4b4b53`、`--text-quiet #63636b`、`--text-muted #85858d`、`--text-faint #b9b9c2` |
| 强调 | `--accent #4f46e5`、`--accent-soft #eeedfd`、`--accent-on #fff`、`--accent-selected #ecebfb` |
| 状态 | 成功/危险/警示各三槽（fill/text/soft），危险另加 border |
| 协议 chip | OpenAI / Anthropic / Gemini 各一组前景/背景 |
| 圆角 | `4 / 6 / 8 / 12 / 14 / 16 / 999`（xs/sm/ctl/card/dialog/hero/pill） |
| 字号 | `12 / 12.5 / 13 / 15 / 16 / 17 / 20 / 24 / clamp(...)`（舒适档） |
| 阴影 | `--shadow-1/2/3`；卡片 1、浮层 2、对话框 3 |
| 字族 | 后台栈 + PingFang/Hiragino/YaHei/Noto CJK 回退（后台 app.css 同步） |

做法说明：**没有把两份 CSS 合成一个文件**。管理端与门户各自是被独立加载的页面，合并会强迫两侧加载对方的选择器；共享的是 DESIGN 定义的 token 名与值，两侧同值声明，P2 若需要可再抽公共 `tokens.css`。

### 4.2 深色收敛

门户原有 6 个 `@media (prefers-color-scheme: dark)` 组件补丁，现收敛为：

- 一份深色 token 列表，写在 `@media (prefers-color-scheme: dark) :root:not([data-theme="light"])`
- 同一份值再写 `:root[data-theme="dark"]`（服务端强制时）
- 组件规则全部改用 token，不再有组件级深色补丁
- `:root[data-theme="light"] { color-scheme: light only }`、`dark` 时 `color-scheme: dark`

### 4.3 主题注入链

```text
settings.json（管理员唯一写入）
  └─ portalServed
       ├─ <html lang="zh-CN|en" data-theme="dark?" data-colorblind="1"? data-motion="reduce"?>
       ├─ <link rel="stylesheet" href="/portal/look.css">（在 portal.css 之后）
       ├─ <meta name="theme-color">（强制主题单值；system 用两条 media）
       └─ HTML 本身
GET /portal/look.css
  └─ portalLookCSS(settings.Load()) = accentCSS(s)
       accent 6 变量 + 表面染色 7 变量 + pop-bg，明暗两套，与后台逐字一致
```

- **唯一来源**：`settings.Settings`；门户无写入口，不存任何偏好（语言 Cookie 除外，见 §6）。
- **CSP**：门户 `style-src 'self'`，所以主题是外部同源 CSS，不是内联 `<style>`；实测 0 条 CSP 违规。
- **缓存**：`look.css` `Cache-Control: no-store`，改强调色后刷新即生效（实测由 `#0d9488` → 派生 `#0b7c72`，再次改 `#db2777` → `#e35795`）。
- **隐私边界**：`portalLookCSS` 只返回 `accentCSS`。壁纸在独立的 `wallpaperCSS`，门户从不调用。

### 4.4 look.css 隐私证据（实测）

| 检查 | 结果 |
|---|---|
| 管理端 `/api/appearance/css` 含壁纸 | ✅ `wallpaper?v=1795451569158` |
| 门户 `/portal/look.css` 含壁纸 | ❌ 0 次 |
| 门户 CSS 含 `Comic Sans` / `uiFont` / `ui-font` | ❌ 0 次 |
| 门户 CSS 体积（强调色已设） | 1,520 B（仅颜色变量） |
| 门户页面请求 `/api/appearance/*` | ❌ 0 次 |
| 门户引用 `wallpaper` 字样 | ❌ 0 次 |

### 4.5 管理端三处微调

| 项 | 前 | 后 | 原因 |
|---|---|---|---|
| `--font` | 无 CJK 回退 | 加 PingFang / Hiragino / YaHei / Noto Sans SC | 中文渲染与门户一致 |
| 焦点环 offset | 1px | 2px | DESIGN §4.2 |
| 侧栏分组标签 | `--muted`（3.33:1） | `--ctl-fg`（5.42:1） | 11px 文字达 AA |

---

## 5. 品牌

### 5.1 已改

| 位置 | 现值 |
|---|---|
| 门户 `<title>` | `MiniRelay · AI 模型中转平台`（英：`MiniRelay · AI model relay platform`） |
| 门户品牌区 | 飞鸟 SVG（CSS mask 着色）+ `MiniRelay` + 副标 |
| 门户 aria-label / 页脚 / 导语 | `MiniRelay` |
| 门户 favicon | `/portal/bird.svg` |
| 管理端 `<title>` | `MiniRelay`；有实例名时 `{name} · MiniRelay` |
| 管理端 wordmark | `MiniRelay` |
| 主屏名 / manifest | `MiniRelay` |
| 登录页、仅管理员页 | `<title>MiniRelay</title>`，`<h1>MiniRelay</h1>` |
| 门户 ← 管理端入口（P0 补） | 保留 |

### 5.2 技术标识未动（白名单）

模块路径、二进制 `magpie`、`MAGPIE_*`、`~/.config/magpie`、Cookie `magpie_web_<port>`、API 路径、DOM id/CSS 变量、`providers.magpie` 等。

### 5.3 未做：管理端内部文案中的 "magpie"

`app.js` 仍有 125 处、`index-web.html` 仍有 14 处 "magpie"，主要是 Agents 连接、托盘/设置行、远程实例等**桌面与引擎集成文案**（如 "Add magpie"、"magpie's models"）。这些是 `data-t` / `t()` 的**英文键**，改名必须同步迁移 zh/ja/de 三条词典键，属于一次独立的键迁移批次。

本轮按 DESIGN §4 的品牌清单处理了 Web 门面；这部分留到 P3（或单列一个品牌扫尾批次），已在 §10 标注。

---

## 6. 国际化

### 6.1 架构

- `assets/portal/i18n.js`：`window.PORTAL_I18N = { zh: {英文键: 中文} }`，243 键。
- 同文件定义 `t()`、`applyPortalI18n()`；键约定与后台一致：英文源串为键、`{name}` 插值、缺键回退英文。
- 静态文案：`data-t`（文本）/ `data-ta`（aria-label）/ `data-tp`（placeholder），首次翻译把英文缓存在 `data-en*`，重绘不丢源。
- 动态文案：所有生成文本改为 `t()`；日期/数字用 `Intl` + `window.portalLangTag`。
- **不加载** 2.6 MB 的 `assets/i18n.js`。

### 6.2 语言决议

| 来源 | 结果 |
|---|---|
| `?lang=en｜zh` | 临时覆盖，不落盘 |
| Cookie `minirelay_lang` | 页脚开关写入，服务端据此写 `<html lang>` |
| `settings.Lang` | `en` → 英文；`zh` / `system` → 中文（门户默认） |

切换实现：写 Cookie → 删除 `?lang` → 以当前 `pathname + search + hash` 重新加载，**保留当前页面与查询参数**，不写管理员全局语言设置。

### 6.3 覆盖与防漂移

| 项 | 值 |
|---|---|
| 词典键 | 243 |
| HTML 键 | 135（`data-t` 文本 + `data-ta` + `data-tp`） |
| JS 键 | 133（四个脚本的 `t("…")` 字面量） |
| 并集 | 243，缺失 0，冗余 0 |
| 新测试 | `TestPortalI18nKeys`：HTML/JS 每个键必须在词典；词典键若后台也有，译文必须一致 |

测试曾抓出 10 处真实漂移（如 `Provider` 后台译"供应商"、门户译"提供商"；`{n} keys`、`Next` 语义不同），已按"同义对齐、异义改名"处理：

- 同义对齐：`Provider`→供应商、`All providers`→全部供应商、`Description`→描述、`Sign out`→退出登录、`{n} keys`→{n} 个密钥、`{n} calls`→{n} 次调用、`Reading…`→读取中…
- 异义改名：`Open`（状态）→`Opened`、`Next/Previous`（翻页）→`Next page/Previous page`、`User`（角色）→`Regular user`

### 6.4 实测

| 场景 | 结果 |
|---|---|
| 服务端 `en` 默认 | 登录页英文，`title=MiniRelay · AI model relay platform`，占位符英文 |
| 切到中文 | `lang=zh-CN`，标题/标题字/占位符/按钮全部中文；Cookie `minirelay_lang=zh` |
| `?lang=en`（Cookie=zh） | 页面英文，Cookie 仍 zh；去掉参数后回到中文 → 临时覆盖未落盘 |
| 缺失键 | `t("No such key here")` → 原样返回 |
| 插值 | `t("{n} keys", {n:3})` → `3 keys` / `3 个密钥` |
| 动态内容 | 角色、注册日期（`2026年11月23日` / `Nov 23, 2026`）、键数（`0 个密钥` / `0 keys`）、时间范围、分页、空态均随语言 |
| 切换后 | 仍留在 `/portal/console`，查询参数保留 |

---

## 7. 验收矩阵

### 7.1 主题与强调色（`screenshots/p1/theme-*`）

| 状态 | 设置 | `<html data-theme>` | 计算底色 | 强调色 | CSP 违规 |
|---|---|---|---|---|---|
| 浅色默认 | light / 默认 | light | `rgb(244,244,246)` | `#4f46e5` | 0 |
| 浅色青 | light / `#0d9488` | light | `rgb(240,246,248)` | `#0b7c72`（派生） | 0 |
| 深色玫红 | dark / `#db2777` | dark | `rgb(33,19,29)` | `#e35795`（派生） | 0 |
| 系统（OS 浅） | system / 默认 | 无 | `rgb(244,244,246)` | `#4f46e5` | 0 |
| 系统（OS 深） | system / 默认 | 无 | `rgb(26,26,30)` | `#a5b4fc` | 0 |

强制浅色在深色系统、强制深色在浅色系统均实测（前者 `color-scheme: light only`，后者深底）。

### 7.2 访问边界

| 请求 | 匿名 | 普通账号 | 管理员账号 |
|---|---|---|---|
| `/portal/look.css` | 200 | 200 | 200 |
| `/portal/console` | 302 → 登录 | 200 | 200 |
| `/portal/admin-models` | 302 → 登录 | **403** | 200 |
| `/api/appearance/css`（对照） | 401 | 403 | 200 |

### 7.3 布局（320/390/768/1440；中/英）

| 宽度 | 页面 | 横向溢出 | <44px 目标 | 输入字号 |
|---|---|---|---|---|
| 320 / 390 | 首页/登录/控制台/模型广场/管理页 | 0 | 0 | 16px |
| 768（鼠标） | 同上 | 0 | 21–23* | 13.3px* |
| 768（触屏） | 控制台 | 0 | **0** | **16px** |
| 1440 | 同上 | 0 | 21–23* | 13.3px* |

\* 桌面鼠标密度是 DESIGN 的既有选择；44/16 只约束触屏。

英文比中文长，验收中抓到两处真实回归并修复：

- **320 英文**：日志分页 `Previous page / Page 1 / 1 / Next page` 撑出 9px → `.pager` 允许换行。
- **390 英文**：四个时间范围胶囊一行放不下，`All` 被横向裁掉 → 改为 `flex-wrap: wrap`（头部已是纵向堆叠，换行不挤）。

### 7.4 Oil UI lint

`node ~/.zcode/skills/oil-ui/scripts/shoot.mjs http://127.0.0.1:3430/portal/login --size desktop,mobile`

```
检查通过：没有控制台错误、横向溢出或加载失败的图片
```

P0 时的"输入框 15px 触发 iOS 缩放 ×2"与"占位符对比度 2.58/2.49"均消失（输入 16px、占位符使用 `--text-quiet`）。

### 7.5 截图

`docs/product-v2/screenshots/p1/`，**27 张**，3.5 MB：

- `theme-{light-default,light-teal,dark-pink,system}-*.png`（5）
- `layout-{zh,en}-{390,1440}-{home,login,console,models,admin-models}.png`（20）
- `layout-zh-768-touch-console.png`
- `brand-admin-1440.png`
- `theme-report.json`、`layout-report.json`

### 7.6 控制台错误

门户页面仅剩匿名页的 `/api/auth/me` 401（既有行为：页面用它判断是否已登录）。favicon 401 已通过给门户加 `/portal/bird.svg` 图标消除。

---

## 8. 测试命令与结果

| 命令 | 结果 |
|---|---|
| `go test -tags nogui ./internal/gui -run 'Portal\|Look\|Appearance\|Asset\|Budget\|Compress' -count=1` | ok 0.927s |
| `go test -tags nogui ./internal/gui -run 'CoreBudget\|TestCore\|InstanceName\|WebApp' -count=1` | ok 0.183s |
| `go test -tags nogui ./internal/gui -count=1` | **ok 93.870s**（P1 期间另有一次 ok 167.200s） |
| `TestPortalI18nKeys` | ok（含 243 键覆盖 + 共享键一致性） |
| Playwright 布局/主题/i18n 脚本 | 见 §7；`theme-report.json`、`layout-report.json` |
| Oil UI `shoot.mjs` | 通过，无提示 |

---

## 9. 未完成 / 风险

1. **管理端内部 "magpie" 文案**（125 + 14 处，§5.3）：需要同步迁移 zh/ja/de 词典键；本轮未做，建议单列品牌扫尾或并入 P3。
2. **管理端对比度**：`--muted` 在浅色卡片上仍为 3.66:1（大量 11–12px 说明文字）。本轮只修了侧栏分组标签；其余需在 P3 逐页替换为 `--ctl-fg`，否则不能通过全站 AA。
3. **多账号外观**：仍为"平台外观由管理员决定"（本机 `settings`）；门户按账号偏好不在本轮。
4. **语言持久化**：`minirelay_lang` Cookie 是门户自己的浏览器偏好，不是账号级设置；换设备/浏览器回到管理员语言。切换会重新加载页面（表单里未提交的输入会丢），这是为保留地址与参数的取舍。
5. **强制主题与 `theme-color`**：`system` 时用两条 `media` meta，强制时用单值；Safari 旧版本对 `media` meta 支持有限，属渐进增强。
6. **未做真机**：iOS Safari 的 44px/16px 仅用 Chromium 触屏上下文验证，真机行为待 P3。
7. **`look.css` 空文件**：默认强调色时返回 0 字节（200），浏览器仍会请求一次；成本可忽略，但属"空响应"，P2 可改为 304 或省略 link。
8. **品牌术语**：`Provider` 的中文已按后台统一为"供应商"，门户原为"提供商"；若产品更倾向后者，需要反向统一后台词典（一次键迁移）。

---

## 10. P2 前置条件

1. 确认 §9.1 的品牌扫尾是否在本轮之后立刻做（涉及词典键迁移，建议独立批次）。
2. 确认"供应商"作为统一中文术语。
3. 确认门户语言切换的重新加载可接受（替代方案是原地重绘，需每个动态区块响应语言事件）。
4. 管理端 `--muted` 的对比度修复范围（全部说明文字 vs 仅关键标签）。

---

## 11. 附件

| 路径 | 内容 |
|---|---|
| `docs/product-v2/screenshots/p1/` | 27 张截图 + 2 份 JSON 实测 |
| `internal/gui/portal_i18n_test.go` | 键覆盖与共享译文一致性测试 |
| `internal/gui/assets/portal/i18n.js` | 门户词典与 `t()` / `applyPortalI18n()` |
| `internal/gui/assets/portal/bird.svg` | 门户标志（CSS mask 着色） |
| `/tmp/p1-before.tar` | P1 开工文件快照（临时，未入库） |
| `/tmp/magpie-p1-bin` | P1 编译产物（临时，未入库） |