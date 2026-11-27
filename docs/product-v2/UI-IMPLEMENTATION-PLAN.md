# MiniRelay 全站 UI 实施计划与技术方案

> 配套规范：`docs/product-v2/DESIGN.md` v1.1
> 依据审计：`docs/product-v2/UI-AUDIT.md`
> 日期：2026-11-23
> 状态：**计划，未落地**。本轮不改任何业务页面，等确认后按 P0→P3 执行。

---

## 0. 状态与保护

### 0.1 执行前实测

| 项 | 值 |
|---|---|
| 目录 | `/root/magpie-original` |
| 分支 | `remove-partners` |
| HEAD | `e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e` |
| `git describe` | `v0.1.4574-1-ge41adc151` |
| `git status --porcelain` 行数 | **71**（40 已跟踪修改 + 31 未跟踪） |
| 与审计开始时相比 | 逐行一致，无新增/删除/修改 |

### 0.2 本轮承诺

- 不 `checkout` / `reset` / `clean` / `stash` / `cherry-pick`，不 `commit` / `push`
- 不修改 `internal/identity`、`internal/access`、`internal/gateway`、`internal/provider`、`internal/usage` 等业务包
- 不删除管理端任何现有功能与入口
- 不引入新前端框架、组件库或 i18n 运行时库
- 本轮只写 `docs/product-v2/` 下的文档

### 0.3 涉及面（按包）

| 包/文件 | P0 | P1 | P2 | P3 | 性质 |
|---|---|---|---|---|---|
| `internal/gui/assets/portal/portal.css` | ● | ● | ● | ● | 样式 |
| `internal/gui/assets/portal/index.html` | | ● | ● | ● | 门户结构 |
| `internal/gui/assets/portal/i18n.js`（新） | | ● | | | 词典 |
| `internal/gui/assets/portal/{portal,usage,models,admin-models}.js` | | ● | | ● | 门户脚本（文案/布局测量） |
| `internal/gui/portal.go` | | ● | | | 页面服务（注入、静态清单） |
| `internal/gui/appearance.go` | | ● | | | 新增 `portalLookCSS`（只读复用） |
| `internal/gui/assets/index-web.html` | | ● | | | 后台品牌 |
| `internal/gui/assets/manifest.json` | | ● | | | 后台品牌 |
| `internal/gui/web.go` | | ● | | | 后台登录/拒绝页标题 |
| `internal/gui/api.go`（boot 标题） | | ● | | | 标签页标题 |
| `internal/gui/assets/app.css` | | ● | ● | ● | 后台三处微调 |
| `internal/gui/assets/i18n.js` | | ○ | | | 只读（防漂移测试引用） |

○ = 仅读取，不修改。

---

## 1. 决策 → 施工映射

| 决策 | 施工批次 | 主要动作 |
|---|---|---|
| D1 品牌 MiniRelay | P1 | 门户 8 处 + 后台 5 处 + Go 4 处（§4） |
| D2 i18n | P1 | `portal/i18n.js` + 226 串迁移 + 服务端 `lang` |
| D3 双档位 | P1（token）/ P2（控件）/ P3（逐页） | 一个 token 层，两档尺寸 |
| D4 主题 | P1 | 服务端注入 + `/portal/look.css` + 深色 token 重构 |

---

## 2. 技术方案 A：主题复用（D4）

### 2.1 现状：配置存在哪、谁在读

| 环节 | 事实 | 位置 |
|---|---|---|
| 存储 | `settings.Settings` JSON，路径 `~/.config/magpie/settings.json`（`appdir`） | `internal/settings/settings.go` |
| 字段 | `Theme`(system/light/dark)、`Accent`(#rrggbb)、`Lang`(system/en/zh)、`Colorblind`、`ReduceMotion`、`NoBird`、`UIFont`/`CodeFont`、`TextSize`、`Wallpaper` | `settings.go:65-95` |
| 读取 | `settings.Load()`，经 `filememo` 缓存，逐请求调用成本可忽略 | `settings.go:1989-1993` |
| 写入 | 均在管理员侧：设置页 `POST /api/settings` → `settings.Save`；壁纸 `POST /api/appearance/wallpaper`；备份恢复等 | `app.js:22800`、`appearance.go:399-415`、`api.go` 多处 |
| 后台加载顺序 | ① `boot.js`（生成）写 `window.bootPrefs` + `document.title` + `<style id="lookCSS">` → ② `theme.js` 设 `data-theme`/`data-colorblind`/`data-motion`/`--zoom` → ③ CSS → ④ 内容 → ⑤ `i18n.js`/`app.js` | `api.go:1276,1392-1399`、`theme.js:3-11`、`index-web.html:26-28,626` |
| 首屏防闪 | theme 属性在样式应用前写好；强调色以 CSS 变量随第一个 `<style>` 到达 | 同上 |
| 后台 CSP | 未设 `style-src`，故内联 `<style id="lookCSS">` 可用 | 实测（仅插件页/库 SVG 有 CSP） |

### 2.2 关键约束：门户不能直接读后台接口（已实测）

| 检查 | 结果 | 依据 |
|---|---|---|
| `/api/appearance/css` 是否公开 | **否** | 不在 `publicAPI`；`web.go:464` 只放行 `publicAPI`/`publicPage` |
| 匿名请求 | **401** | 实测 2026-11-23（页面键守卫，`web.go:485-500`） |
| 普通账号会话 | **403** | 实测（`webAdminOnly`，`web.go:486-495`）；`/api/state` 同为 403 |
| 管理员会话 | **200** | 实测 |
| 输出内容 | `{"css":"…","key":"…"}`；`Accent=#0d9488` 时 css 含 19 个变量：`--accent:#0b7c72`、`--accent-soft`、`--accent-fg`、`--sel`、`--mark`、`--mark-fill` + `--bg`/`--card-2`/`--pill`/`--pill-hover`/`--seg-track`/`--line`/`--line-2` + 深色一套 | 实测 |
| 门户 CSP | `style-src 'self'; script-src 'self'` | `portal.go:38` |
| 推论 | **内联 `<style>` 会被 CSP 拦**；必须用同源外部 CSS；且必须复用**完整强调色块**（含表面染色），不能只取 accent 五项 | — |

### 2.3 设计

```text
settings.json (唯一写入者: 后台 POST /api/settings)
        │
        ├─ 后台页面：boot.js → bootPrefs + <style id="lookCSS">（现状，不动）
        │
        └─ 门户页面：portalServed 读 settings.Load()
                 ├─ 注入 <html lang data-theme data-colorblind data-motion>
                 ├─ 注入 <link rel="stylesheet" href="/portal/look.css">（在 portal.css 之后）
                 └─ 注入 <script type="application/json" id="portal-prefs">
```

**新增 `/portal/look.css`**

- 在 `portalServed` 里以 `name == "look.css"` 单独分支生成（不是磁盘文件，所以不放进 `portalFiles` 的静态清单）。
- 内容 = `lookCSS(s)` **去掉壁纸段**：`accentVars`（6 变量）+ `tintVars`（表面染色）+ `popVar` + 明暗两套规则，与后台逐字一致（实测 `Accent=#0d9488` 时共 19 个变量）。
- **不含壁纸**：`lookCSS` 里 `s.Wallpaper` 段落必须显式跳过。门户是公开页，壁纸是用户私有图片。
- **不含 `--ui-*` 字体变量**（除非 §2.6 决策改为"门户也吃字体"）。
- 响应头：`Content-Type: text/css; charset=utf-8`、`Cache-Control: no-store`（随设置变化）、`X-Content-Type-Options: nosniff`。
- 生成规则与后台逐字一致，避免"后台绿、门户紫"。

**注入属性**（`portalServed` 里替换，现成注入点是 `data-page="__PAGE__"`）

```html
<html lang="zh-CN" data-theme="dark" data-colorblind="1" data-motion="reduce">
```

- `data-theme` 仅在 `Theme != "system"` 时写；`system` 交给 `@media (prefers-color-scheme)`。
- `lang` 由 §3.3 的语言决议决定。
- 这三个属性都是 HTML 属性，CSP 不拦。

**`theme-color` 一并修正**：`portal/index.html:7` 现为固定 `#4f46e5`（强调色），在深色主题下手机状态栏会与页面底色脱节。改为按解析后的主题写 `#f4f4f6` / `#1a1a1e`（与后台 `manifest.json` 的 `theme_color` 同源，【推】）。

**portal.css 的改造**（P1）

| 现状 | 目标 |
|---|---|
| 6 个分散的 `@media (prefers-color-scheme: dark)` 块逐条覆盖组件 | ① 一个深色 token 块（`@media` + `[data-theme=dark]` 两路）② 组件层不再写死深色值 |
| 无 `[data-theme]` 选择器 | 增加 `:root[data-theme="dark"]` 与 `:root:not([data-theme="light"])` 规则 |
| 无 `color-scheme` | `:root { color-scheme: light dark }` + 强制态覆盖（照 `app.css:114-116`） |
| 无 `data-motion`/`data-colorblind` | 增加对应选择器（照 `app.css:4798,123-130`） |

深色 token 采用 DESIGN §2.4 的值（`--bg #1a1a1e`、`--surface #232327` …），替换门户自有的 `#0f1016/#181a23` 系列。

### 2.4 为什么不做"门户自己存一份偏好"

- 会出现两套配置源，正是需求要避免的。
- 门户是公开页，没有可信的写入通道；新增写接口等于扩大攻击面。
- 结论：**门户只读，后台独占写**。

### 2.5 已知限制（必须写进交付说明）

1. **全局而非按账号**：`settings` 是本机管理员的偏好。MiniRelay 有多个平台账号，本轮所有访问者共享管理员设定的外观。
2. **壁纸不跨页**：门户不显示后台背景图（隐私边界）。
3. **缓存**：`look.css` 必须 `no-store`，否则改强调色不生效。
4. **首屏**：属性 + 外部 CSS 都在 `<head>`，无闪烁；门户 HTML 已是 `no-store`。

后续若要按账号：在 `identity` 用户记录里加偏好字段，门户优先读会话用户、回退全局（【建】，不在本轮）。

### 2.6 字体设置的边界（建议）

`settings.UIFont` 是"本机装了什么字体"的选择。把它输出到公开门户会泄露本机字体名（轻微的机器信息）。建议：

- **门户不复用 `UIFont`**，固定用规范里的 CJK 合并栈。
- 后台能力完全保留（`--ui-*` 不动）。
- 若坚持门户也跟随，改为只输出家族名并在 DESIGN 记一条"公开页可见本机字体名"的说明。

> 这一条是建议，需你确认（§13-4）。

---

## 3. 技术方案 B：i18n 接入（D2）

### 3.1 现状：`i18n.js` 怎么组织

| 项 | 事实 | 位置 |
|---|---|---|
| 形态 | 单个静态 JS：`const I18N = { zh:{…}, ja:{…}, de:{…} }`，键是**英文源串** | `i18n.js:1-4` |
| 体量 | 2.6 MB（后台全部页面 + 插件 + 提示） | 实测 |
| 取词 | `t(s, vars)`：`I18N[locale]?.[s] ?? s`，`{name}` 插值 | `i18n.js:22212-22216` |
| 语言决议 | `?locale=` > 保存值 > 系统；`system` 映射 `zh-Hant→zh-TW`、`zh→zh`、`ja`、`de`、其余 `en` | `i18n.js:22165-22179` |
| 静态文本 | `data-t`（文本）/`data-tt`（title）/`data-tp`（placeholder），首次把英文缓存到 `data-en*` | `i18n.js:22291-22307` |
| `<html lang>` | 随语言写 `LANG_TAGS` | `i18n.js:22172-22173` |
| 加载位置 | 后台 body 末尾（`index-web.html:626`），之后才是 `app.js` | 【源】 |
| 插件词典 | `plugins/routing/i18n.js` 用 `magpie.strings({zh:{…}})` 合并进主词典 | 【源】 |

### 3.2 方案：同约定，小词典

**新增 `internal/gui/assets/portal/i18n.js`**

```js
// 门户的词条：英文是键，中文是值；缺键回退英文（与 i18n.js 同约定）。
window.PORTAL_I18N = {
  zh: {
    "Sign in": "登录",
    "Register": "注册",
    "Model plaza": "模型广场",
    ...
  },
};
```

- 加入 `portal.go` 的 `portalFiles`：`"i18n.js": "text/javascript; charset=utf-8"`。
- 在 `index.html` 里排在 `usage.js/models.js/admin-models.js/portal.js` **之前**（`<script src="/portal/i18n.js">`）。
- 不加载 2.6 MB 的 `i18n.js`：门户只用其中极小一部分，整包引入会让首屏多背 20 倍体积。

**门户侧 `t()`（`portal.js` 顶部，约 10 行）**

```js
const LANGS = { zh: "zh-CN", en: "en" };
let locale = "zh";
function t(s, vars) {
  const d = window.PORTAL_I18N?.[locale];
  let out = (d && d[s]) ?? s;
  if (vars) for (const k in vars) out = out.split("{" + k + "}").join(vars[k]);
  return out;
}
```

**静态文本应用**（照 `translateStatic` 的 `data-en` 缓存策略）

```js
function applyI18n(root = document) {
  for (const e of root.querySelectorAll("[data-t]")) {
    if (e.dataset.en === undefined) e.dataset.en = e.textContent;
    e.textContent = t(e.dataset.en);
  }
  for (const e of root.querySelectorAll("[data-tp]")) { … }
  for (const e of root.querySelectorAll("[data-tt]")) { … }
  document.documentElement.lang = LANGS[locale] || "zh-CN";
}
```

**动态渲染**：`usage.js`/`models.js`/`admin-models.js`/`portal.js` 里 126 个中文字面量改 `t("English key")`；日期与数字改用 `Intl` + `LANGS[locale]`（现在 `when()` 写死 `zh-CN`，`portal.js:72-76`）。

### 3.3 语言决议

| 来源 | 结果 |
|---|---|
| 服务端 `settings.Lang` = `zh` | 中文 |
| = `en` | 英文 |
| = `system` | **中文**（D2：门户中文默认） |
| `?lang=zh｜en` | 覆盖（只读，验收/排障用） |
| 后台 | 完全不动（`system` 仍按系统语言） |

服务端在 `<html lang>` 里写 `zh-CN`/`en`；`portal.js` 若发现 `?lang=` 再校正一次。

### 3.4 覆盖清单（226 个唯一串）

> 下表的"出现处"是各文件里中文字符串的粗略计数（含重复），用于估工；唯一串总计 226 来自对文本节点与字符串字面量的去重统计。

| 文件 | 中文出现处（粗略） | 处理方式 |
|---|---|---|
| `portal/index.html` | 159 个文本节点 → 120 唯一 | 中文 → 英文键 + `data-t/-tp/-tt`；`<title>` 服务端注入 |
| `portal.js` | 123 | `t()`；`authError()` 的 8 条正则映射改为"正则 → 英文键 → t()" |
| `usage.js` | 26 | 状态名、空态、单位、图表 aria |
| `models.js` | 50 | 筛选、空态、详情标签 |
| `admin-models.js` | 36 | 表格列、开关文案、确认框 |

> 字符串字面量去重后 126 条，与 HTML 的 120 条无重叠，合计 226 条。

### 3.5 服务端错误文案

`authError()` 已是"英文原因 → 中文句子"的正则映射（`portal.js:87-103`），天然适合 i18n：

- 正则匹配继续保持，命中的是**英文键**，再 `t()`。
- 未命中的 `400` 会把服务端英文原因嵌进句子（`{why}` 占位），中文模式下会出现夹英——这是可接受的降级，记入交付说明。
- 服务端 `authFail` 的文案（`auth.go:108-135`）**不改**，属于 API 行为。

### 3.6 防漂移

新增 Go 测试 `internal/gui/portal_i18n_test.go`（【建】，P1 一并提交）：

1. 解析 `portal/i18n.js` 与 `i18n.js`，取各自 `zh` 字典。
2. 对门户词典的每个键：若主词典存在同键且值不同 → 失败。
3. 扫描 `portal/*.js` 与 `index.html` 的 `data-t` 键：词典缺键只警告不失败（回退英文是合法状态）。

> 测试只在 `internal/gui` 内，不碰业务包。

### 3.7 CSP 与体积

| 项 | 约束 |
|---|---|
| CSP | `script-src 'self'` → 词典必须是外部文件，不能内联 |
| 体积 | 预计 30–40 KB（gzip 后 ~10 KB），可接受 |
| 缓存 | 与其它 portal 文件一致 `Cache-Control: no-cache` |
| 回退 | 词典文件缺失 → `t()` 返回英文键，页面仍可用 |

---

## 4. 技术方案 C：品牌替换清单（D1）

### 4.1 门户

| 文件:行 | 现值 | 目标 |
|---|---|---|
| `portal/index.html:9` | `<title>Magpie · AI 模型中转平台</title>` | `<title>MiniRelay · AI 模型中转平台</title>`（英文模式下由服务端/JS 切为 `MiniRelay · AI model relay platform`） |
| `:16` | `aria-label="Magpie 首页"` | `aria-label="MiniRelay"` + `data-ta` 本地化 |
| `:18` | `<span class="brand-name">Magpie</span>` | `MiniRelay` |
| `:19` | `brand-sub` 副标 | 保留文案，`data-t` 化 |
| `:36` | eyebrow `AI 模型中转平台` | 保留，`data-t` 化 |
| `:38` | 导语含 `magpie` | `MiniRelay` |
| `:363` | 页脚 `Magpie · AI 模型中转平台` | `MiniRelay · AI 模型中转平台` |
| `portal.css:2` | 注释 `shares magpie's accent` | 注释改为 `MiniRelay`（不影响渲染） |

### 4.2 后台（Web）

| 文件:行 | 现值 | 目标 |
|---|---|---|
| `index-web.html:10` | `<title>magpie</title>` | `MiniRelay` |
| `index-web.html:22` | `apple-mobile-web-app-title content="magpie"` | `MiniRelay` |
| `index-web.html:65` | `<span>magpie</span>` | `MiniRelay` |
| `manifest.json:2-3` | `name`/`short_name` = `magpie` | `MiniRelay` |
| `api.go:1392-1396` | 标签页 `n + " · magpie"` | `n + " · MiniRelay"` |

### 4.3 Go 的可见页面

| 文件:行 | 现值 | 目标 |
|---|---|---|
| `web.go:389,391` | `<title>magpie web</title>` / `<h1>magpie web</h1>` | `MiniRelay` / `MiniRelay` |
| `web.go:522` | `<title>magpie</title>`（仅管理员页） | `MiniRelay` |

### 4.4 不在本轮改

- `index.html`（Wails 桌面壳）——非 Web 范围；若要一致，随 P3 顺带
- 二进制名、`MAGPIE_*`、模块路径、配置目录、Cookie、API 路径、CSS 变量、DOM id
- README / docs / site（对外文档，另行授权）

### 4.5 验证

```sh
grep -rn 'Magpie\|magpie' internal/gui/assets/portal/ internal/gui/assets/manifest.json
grep -n '<title>' internal/gui/assets/index-web.html
```
白名单（技术标识）之外应为 0 命中。

---

## 5. P0：移动端缺陷（先做，独立可交付）

### 5.1 范围

只改 `portal.css`，**不改 JS、不改 HTML**。审计 P1-1/P1-2/P1-3/P2-3/P2-4 全部在这一批。

### 5.2 具体改动

| # | 问题 | 改法 | 依据 |
|---|---|---|---|
| P0-1 | 日志表 390px 溢出 751px | ≤760 把 `.log-row` 从 6 列网格改为块式卡片；隐藏 `.log-row.head`；用已有 `data-label` 生成字段标签 | 审计 P1-1；`usage.js:156-158` 已给每个单元格写 `data-label`，**无需改 JS** |
| P0-2 | 统计卡 44px 宽 | `.usage-stats` ≤760 改 `repeat(2, minmax(0,1fr))` | 审计 P1-2；后台 `app.css:4886` 同款 |
| P0-3 | 时间胶囊 118px 竖条 | `.usage-head` ≤760 纵向堆叠；`.segs` 独占一行、`flex-wrap:nowrap; overflow-x:auto` | 审计 P1-3 |
| P0-4 | 输入框 15px 触发 iOS 缩放 | ≤760 `input, select, textarea { font-size:16px; min-height:44px }` | 审计 P2-3；后台 `app.css:4901-4903` |
| P0-5 | 刷新/分页 30px 高 | ≤760 `#logRefresh/#logPrev/#logNext/.log-filters select { min-height:44px }` | 审计 P2-4；后台 `app.css:4806` |

### 5.3 建议实现（示意，未落盘）

```css
@media (max-width: 760px) {
  .usage-stats { grid-template-columns: repeat(2, minmax(0, 1fr)); }

  .usage-head, .logs-head { flex-direction: column; align-items: stretch; gap: 10px; }
  .usage-head .segs { flex-wrap: nowrap; overflow-x: auto; padding-bottom: 2px; }
  .usage-head .segs .chip { flex: none; }

  .log-row { display: block; padding: 12px 0; }
  .log-row.head { display: none; }
  .log-row > * { display: block; }
  .log-row > [data-label]::before {
    content: attr(data-label) "：";
    color: var(--text-quiet); font-size: 11.5px; font-weight: 600;
    margin-right: 6px;
  }
  .log-ms { text-align: left; }
  .log-status { justify-self: start; margin-top: 4px; }

  input, select, textarea { font-size: 16px; min-height: 44px; }
  .log-filters select, #logRefresh, #logPrev, #logNext { min-height: 44px; }
}
```

> 用 `data-label` 而不是新加 DOM，是因为 `usage.js:156-158` 已经为每个单元格写了标签（时间/模型/API Key/状态/Token/耗时），P0 保持零 JS 改动，回滚只需删掉这一段。

### 5.4 验收

| 项 | 方法 | 通过标准 |
|---|---|---|
| 无横向溢出 | Playwright 390×844，注入 2 行日志（同 `usage.js` DOM），读 `documentElement.scrollWidth` | `=== 390` |
| 统计卡 | 读 `.usage-stats .stat` 宽 | ≥140px，2 列 |
| 时间胶囊 | 读 `#usagePeriod` 宽高 | 宽 > 300px，高 ≤ 40px |
| 输入框 | 读 `input` 计算字号 | 16px |
| 点按目标 | 枚举可见按钮高度 | 全部 ≥44px |
| Oil UI lint | `node shoot.mjs <url> --size mobile` | 无"输入框字号 <16px"提示 |

### 5.5 风险

- 390px 卡片化后行高增加，页面变长——接受，优于横向滚动。
- `data-label` 只在 `usage.js` 渲染的行上存在；空态不受影响。
- 回滚：单个 `@media` 块，删除即恢复。

---

## 6. P1：品牌、Token、主题、语言

> P1 是"地基"。做完后门户与后台共享颜色/字号/圆角/主题/语言；页面结构不变。

### 6.1 P1-a 全局 Token 归一

| 动作 | 文件 | 说明 |
|---|---|---|
| 重写 `:root` 为 DESIGN §2/§6 的 token 集 | `portal.css:1-47` | 语义 token 名（`--surface/--text-quiet/--r-ctl/--r-card`），保留旧名为别名一版 |
| 深色从 6 个块收敛为 1 个 token 块 + 属性选择器 | `portal.css` | 见 §2.3 |
| 字号改用 `--fs-*` 档位 | `portal.css` 全文 | 映射表见 DESIGN §3.2 |
| 圆角改用 `--r-*` 阶梯 | `portal.css` 全文 | 11 种 → 7 种 |
| 硬编码状态色提 token | `portal.css:240,243,273,275,348-352,376,432-437,440,555-560` | 23 处 → 0 |
| 焦点环补全 | `portal.css:237,416,462,530,654` | `outline:none` 改为保留环 |
| 后台三处微调 | `app.css` | `--font` 加 CJK、`outline-offset` 1→2、11px 标签 `--muted`→`--ctl-fg` |

### 6.2 P1-b 主题注入

| 步骤 | 文件 | 内容 |
|---|---|---|
| 1 | `appearance.go` | 新增 `portalLookCSS(s)`：只取强调色/字体部分，显式跳过壁纸 |
| 2 | `portal.go` | `portalFiles` 加 `i18n.js`；`portalServed` 分支服务 `/portal/look.css`；`<html>` 注入 `lang/data-theme/data-colorblind/data-motion`；注入 `#portal-prefs` JSON |
| 3 | `index.html` | `<link rel="stylesheet" href="/portal/look.css">` 放在 `portal.css` 之后 |
| 4 | `portal.js` | 读 `#portal-prefs`（可选，属性已够用） |

**不改**：`settings` 结构、`POST /api/settings`、页面键守卫、`webHandler` 公开路由表。

### 6.3 P1-c 语言

| 步骤 | 文件 | 内容 |
|---|---|---|
| 1 | `portal/i18n.js`（新） | 约 200 条 zh 词条，英文为键 |
| 2 | `portal/index.html` | 120 处静态中文 → 英文键 + `data-t/-tp/-tt` |
| 3 | `portal.js` | `t()`/`applyI18n()`；`authError()` 改为键 |
| 4 | `usage/models/admin-models.js` | 126 处字面量 → `t()`；`Intl` 用当前语言 |
| 5 | `portal.go` | `<html lang>` 按 `settings.Lang` 注入 |
| 6 | `internal/gui/portal_i18n_test.go`（新） | 共享键防漂移 |

### 6.4 P1-d 品牌

按 §4 清单逐条替换；`portal/index.html` 的 8 处与 i18n 改造同一次提交（同一批文案改动，避免两次翻工）。

### 6.5 P1 验收

| 项 | 标准 |
|---|---|
| 品牌 | 白名单外无 `Magpie/magpie` |
| Token | 门户 token 块外无十六进制颜色 |
| 主题 | `Theme=dark` 首屏深色；`Accent=#0d9488` 两端强调色一致；`Colorblind` 图表换色；`ReduceMotion` 停动效 |
| 语言 | `Lang=en` 门户全英文无中文残留；`system/zh` 中文 |
| 后台 | 主题/字体/色盲/语言行为与改造前逐项一致（回归） |
| CSP | 门户无内联 `<style>`/`<script>`；控制台无 CSP 报错 |

### 6.6 P1 风险

| 风险 | 缓解 |
|---|---|
| 深色块重构漏改组件 | 用 `Theme=dark` 逐页截图对比（门户 6 页 × 2 主题） |
| `look.css` 被缓存 | 强制 `no-store`；验收时改一次强调色再刷新 |
| 字典漏键 | `t()` 回退英文不崩；防漂移测试 + 全页英文模式扫描中文 |
| 品牌替换误伤技术标识 | 白名单 grep（§4.5）+ code review |
| `portal.js` 改动引入回归 | 保持"只换文案来源，不改流程控制"；登录/注册/Key 创建路径脚本回归 |

---

## 7. P2：基础控件统一

### 7.1 范围

| 组件 | 目标规格 | 文件 |
|---|---|---|
| 按钮 | 6 变体 × 尺寸；圆角 `--r-ctl`；光晕仅首屏 | `portal.css`、`app.css`（仅焦点/对比度） |
| 输入/下拉 | 34/42/44 三档；底 `--surface`；占位符 `--text-quiet` | `portal.css` |
| 开关/复选 | 38×22 / 16×16 | `portal.css` |
| 卡片 | `--r-card` + `--shadow-1`（舒适）/ `--shadow-0`（密集） | `portal.css` |
| 表格 | `.led` 规格 + ≤760 卡片化 | `portal.css`、`usage.js`（表格语义，可选） |
| 弹窗 | 600/14/遮罩 blur 8/头脚 48 | `portal.css` |
| 导航 | 项 30–36/`--r-ctl`/500 | `portal.css` |
| 状态 | 5 态（含信息）+ 骨架 240%/1.4s | `portal.css` |

### 7.2 验收

- 组件级截图对照：门户 `console` 与后台 `Usage` 并排，按钮/输入/标签/表头四组对齐。
- DESIGN §15 的 7/8/9 项（圆角集合、字号档位、无内联样式）脚本检查。

### 7.3 风险

- 按钮高度从 38 → 42 会改变首屏节奏 → 只在 P3 逐页确认留白。
- 表格规格变化影响日志页信息密度 → P0 已保证移动端可用，桌面端在 P3 复核。

---

## 8. P3：逐页视觉迁移

### 8.1 顺序（先低风险，后高流量）

| 批次 | 页面 | 原因 |
|---|---|---|
| P3-1 | 门户登录、注册 | 结构简单，只 1 张卡 + 表单，先验证表单/错误态 |
| P3-2 | 门户首页 | 首屏光晕、标题层级、卡片组 |
| P3-3 | 门户控制台 | 组件最多（统计、图表、日志、开关、弹窗） |
| P3-4 | 模型广场 + 管理端模型页 | 卡片网格、筛选、表格 |
| P3-5 | 后台 Providers / Routing / Gateway / Usage / Plugins / Masking | 只吃 token 与三处微调，逐页截图回归 |
| P3-6 | 后台设置、对话框、浮层 | 密集控件、弹窗规格复核 |

### 8.2 每页交付物

1. 1440 / 390 × 浅 / 深 四张截图
2. Oil UI `shoot.mjs` lint 清零
3. 该页交互路径（登录、创建 Key、筛选、分页、开关）手测通过
4. 与 DESIGN §15 清单逐项核对

### 8.3 禁止

- 不在 P3 顺手改业务逻辑、接口、数据结构
- 不做"再设计"，只按规范对齐
- 不删除功能；入口和按钮数量不变

---

## 9. 顺序与依赖

```text
P0（纯 CSS，可独立上线）
  └→ P1-a Token（依赖 P0 的媒体查询位置）
       └→ P1-b 主题注入（依赖 Token 的深色块重构）
       └→ P1-c 语言（依赖 index.html 文案改造，与品牌同批）
            └→ P2 控件（依赖 Token 名）
                 └→ P3 逐页（依赖 P1+P2）
```

关键依赖：

- **P1-b 依赖 P1-a**：没有统一的深色 token，注入 `data-theme` 只会让 6 个补丁块互相打架。
- **P1-c 与 P1-d 同批**：同一批文案改动，避免先改品牌再改键位两次翻工。
- **P2 依赖 P1-a**：控件要用 `--r-ctl`/`--fs-*`/语义色。
- **P0 不依赖任何批次**，可以先合、先验收、先上线。

---

## 10. 验收与测试策略

| 层 | 工具 | 频率 |
|---|---|---|
| 页面截图 | Playwright + Chromium 1254（`/tmp/audit-tools/shoot-portal.cjs` 可复用） | 每批 |
| Oil UI lint | `~/.zcode/skills/oil-ui/scripts/shoot.mjs` | 每批 |
| 布局断言 | 注入合成日志行后读 `scrollWidth`、元素尺寸 | P0/P2/P3 |
| 主题断言 | 切换 `Theme/Accent/Colorblind/ReduceMotion` 各截一张 | P1/P3 |
| 语言断言 | `Lang=en/zh` 各跑一遍，正则扫中文残留 | P1/P3 |
| 品牌断言 | `grep` 白名单 | P1 |
| Go 测试 | `go test -tags nogui ./internal/gui -run Portal` | P1（新增防漂移测试） |
| 回归 | 现有 `internal/gui` 全量测试（P1/P2 后各一次） | 每批 |
| 业务不动 | `git diff --stat` 检查 P0–P3 只触及上表文件 | 每批 |

---

## 11. 风险与回滚

| 风险 | 影响 | 缓解 / 回滚 |
|---|---|---|
| 深色重构漏组件 | 局部浅色残留 | 逐页双主题截图；回滚到 6 个补丁块 |
| 主题注入导致首屏闪 | 体验 | 属性在 HTML 里、CSS 在 head；不接受 JS 后置改主题 |
| `look.css` 泄露隐私 | 安全 | 只输出颜色/字体；绝不输出壁纸；代码审查 + 测试断言 `lookCSS` 的壁纸段未进入 `portalLookCSS` |
| i18n 漏词 | 中英混杂 | 回退英文 + 英文模式扫残留；词典测试 |
| 品牌替换破坏技术标识 | 兼容 | 白名单 grep + 不改协议字段 |
| 改动面过大 | 回归难 | 严格按 P0→P1→P2→P3 分批，每批独立可回滚 |
| 双档位被"平均"成一套 | 失去门户可读性 | DESIGN §3.4 明确两档尺寸，review 时核对 |

回滚单位：每个批次独立，`git diff` 可整批撤销；P0 是单段媒体查询，P1-b 是新增函数 + 一处 HTML 注入。

---

## 12. 本轮不做

- 不改任何 `.css`/`.js`/`.html`/`.go`（本轮只写文档）
- 不改 `settings` 结构、写入协议、认证、API Key、Gateway、Provider、Routing、用量
- 不引入框架/组件库/i18n 库
- 不做按账号外观、不做门户壁纸、不做语言切换 UI（除非 §13 确认）
- 不扩展审计范围

---

## 13. 待确认（开工前）

| # | 事项 | 建议 | 影响批次 |
|---|---|---|---|
| 1 | 门户标志 | 用后台飞鸟 SVG，停用字母 M 块 | P1-d |
| 2 | 后台 wordmark 大小写 | `MiniRelay`（不再全小写） | P1-d |
| 3 | 门户语言开关 | 本轮不加，`?lang=` 仅供验收 | P1-c |
| 4 | 门户是否吃 `UIFont` | **建议不吃**（公开页泄露本机字体名） | P1-b |
| 5 | 多账号外观 | 本轮全局（管理员决定），后续再按账号 | P1-b |
| 6 | P0 是否先单独上线 | 建议先合 P0（纯 CSS、无依赖、修三个实际缺陷） | P0 |

---

## 14. 交付检查（每批结束时）

1. `git status --porcelain` 的已跟踪项只包含本批声明的文件
2. `git diff` 中无业务包（`identity/access/gateway/provider/usage`）改动
3. 端口/进程/临时文件已清理（截图脚本在 `/tmp`，不写入仓库）
4. 截图与 lint 报告已归档到 `docs/product-v2/screenshots/<批次>/`
5. 若新增测试，`go test -tags nogui ./internal/gui -run Portal` 通过
6. 交付说明列出：改了什么、没做什么、如何验证、下一批前置条件