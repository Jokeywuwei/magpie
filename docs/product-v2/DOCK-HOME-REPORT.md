# DOCK-HOME-REPORT — 概览默认首页 + 配置驱动 Dock + Tooltip

日期：2026-11-25
分支：`remove-partners`
HEAD：`e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e`
版本标记：`v0.1.4574-dock-home`
范围：用户门户 Dock、营销首页入口、登录落点、系统字体
未 commit / push / reset / stash；未改后端 API、路由权限、数据结构。

---

## 1. Dock 配置结构

唯一配置在 `internal/gui/assets/portal/portal.js`：

```js
const DOCK_ITEMS = [
  { id: "overview", href: "/portal/console", label: "Overview", icon: "chart", order: 10 },
  { id: "models",   href: "/portal/models",  label: "Model plaza", icon: "grid", order: 20 },
  { id: "keys",     href: "/portal/keys",    label: "API keys", icon: "key", order: 30 },
  { id: "admin",    href: "/",               label: "Admin console", icon: "admin", order: 40, adminOnly: true },
  { id: "profile",  href: "/portal/profile", label: "Profile", icon: "avatar", order: 50, avatar: true },
];
```

- `label` 是 i18n 键，渲染时调用 `t(item.label)`；中文输出“概览 / 模型广场 / API 密钥 / 管理后台 / 个人资料”，英文输出 “Overview / Model plaza / API keys / Admin console / Profile”。没有硬编码中文。
- `renderDock(user)` 读取 `order` 排序、过滤 `adminOnly`（普通用户不渲染管理后台）、过滤未登录的 `avatar`；每个 item 生成 `<a class="dock-item" data-dock-id aria-label href>`，`avatar: true` 的 item 额外加 `dock-avatar` 并显示姓名首字母。
- `index.html` 的 Dock 是空容器 `<nav class="dock" id="dock"></nav>`；顺序、增删、隐藏只改 `DOCK_ITEMS`，不需要改 HTML。
- 头像前自动插入 1px `.dock-sep`；图标用内联 SVG，`DOCK_ICONS` 映射 `chart/grid/key/admin`，未知 icon 回退为圆形细线图标。
- 当前路由映射：`console → overview`、`models → models`、`keys → keys`、`profile → profile`、`admin-models → admin`；`location.pathname === "/"` 时映射 admin。当前项加 `.on` 和 `aria-current="page"`。

---

## 2. 行为变化

- **概览是登录后的默认首页**：登录成功、注册成功、已登录访问 `/portal/login` 都落到 `/portal/console`；登录页的 `?next=` 不再改变落点（服务端已登录访问登录页仍重定向 console）。
- **Dock 第一项是概览**，不再有营销首页按钮；营销首页 `/portal/` 只通过登录页品牌 Logo、页脚 Home 链接访问。
- 页脚导航在已登录页面重新可见（Home / Model plaza / Console），登录/注册页隐藏页脚导航，保留登录品牌 Logo；这样从概览/密钥/资料都能回到营销首页。
- Dock 悬浮文字：默认只显示图标；hover / focus-visible 显示 tooltip（`--popover` 92%、1px border、8px 圆角、6px 10px 内距、12px/500、无阴影、无箭头、120ms 淡入上移）；移动端禁用 tooltip，保留 aria-label。
- Dock 焦点：keyboard focus-visible 使用 2px `--foreground` 白环并显示 tooltip；Enter 按锚点跳转。当前项为 `--accent` 底 + 1px 白环。
- **字体**：统一为
  `-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", "Noto Sans", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji"`；
  页面标题 32/700，Dock tooltip 12/500，按钮 14/500，表格正文 14，标签 12，数字 tabular-nums；没有新增字体文件、图标库或依赖。

---

## 3. 验收数据（Chromium + Playwright）

### 3.1 落点 / 默认首页

| 项 | 实测 |
|---|---|
| 登录 `?next=/portal/models` | 最终 URL `/portal/console` ✅ |
| 已登录访问 `/portal/login` | 服务端重定向 `/portal/console` ✅ |
| 注册成功 | 代码落点 `/portal/console`（未改动） |
| 原始 HTML Dock | 空容器（`rawDockEmpty = true`），items 由 JS 渲染 ✅ |

### 3.2 1440 Dock

| 项 | 实测 |
|---|---|
| 管理员顺序 | `overview / models / keys / admin / profile` ✅ |
| 普通用户顺序 | `overview / models / keys / profile`；管理后台 DOM 不存在 ✅ |
| 未登录 | `overview / models / keys`；头像和管理后台不渲染 ✅ |
| 营销首页入口 | Dock 无 `home` / `/portal/` 项；页脚 Home 可见 ✅ |
| 当前项 | console=overview；profile=avatar `.on`；admin-models=admin `.on` ✅ |
| Tooltip 标签 | 概览 / 模型广场 / API 密钥 / 管理后台 / 个人资料，全部 12px/500 ✅ |
| Tooltip 样式 | background popover 92%、1px border、8px 圆角、无阴影 ✅ |
| 字体 | tooltip 与 body 均为要求的系统栈；按钮 14/500；标题 32/700；日志正文 14；标签 12 ✅ |
| 键盘 | Tab 聚焦 Dock 后 outline `oklch(0.95 0 0) solid 2px` offset 2px + tooltip；Enter 跳到 `/portal/keys` 且 keys `.on` ✅ |
| 语言 | 英文 Dock labels/tooltips/aria：Overview / Model plaza / API keys / Admin console / Profile ✅ |

### 3.3 390 / 320

| 项 | 实测 |
|---|---|
| Dock 宽度 | 390 = 299px；320 = 241px；均居中、无横向溢出 ✅ |
| 可点目标 | 390 三页 0 个 <44px；页脚 Home/Console 链接也补到 44px ✅ |
| Tooltip | 390 下 `display: none`（触屏不依赖 hover），aria-label 保留 ✅ |
| 页面宽度 | 390 = 390；320 = 320 ✅ |
| Dock 底部 | 首页/概览滚动到底 Footer 与 Dock 间距 22px ✅ |
| 未登录头像 | 390 未登录首页 Dock 只有 3 项，无 avatar/admin ✅ |

### 3.4 构建 / 测试

```sh
GOMAXPROCS=2 CGO_ENABLED=0 go build -tags nogui -trimpath \
  -ldflags "-s -w -X main.version=v0.1.4574-dock-home" -o /tmp/mr-dark .
# BUILD OK

GOMAXPROCS=2 CGO_ENABLED=0 go test -tags nogui ./internal/gui \
  -run 'TestPortal|TestPortalI18nKeys|TestAuth|TestUserUsage' -count=1
# ok
```

未跑全量 gateway；未改 gateway。

---

## 4. 截图

目录：`docs/product-v2/screenshots/dock-home/`（18 张）

| 截图 | 说明 |
|---|---|
| `home-1440-zh-dark.png` / `home-1440-en-dark.png` | 未登录营销首页，Dock 仅概览/模型广场/API 密钥，无头像/管理后台 |
| `home-390-zh-dark.png` / `home-390-en-dark.png` | 390 营销首页，Dock 3 项，无溢出 |
| `home-tooltip-1440-zh-dark.png` | 首页 Dock 第一项 hover tooltip“概览” |
| `home-signedin-1440-zh-dark.png` | 登录后营销首页，Dock 5 项、无当前项 |
| `home-footer-1440-zh-dark.png` | 营销首页页脚 Home 链接 |
| `overview-1440-zh-dark.png` / `overview-1440-en-dark.png` | 登录后概览默认首页，Dock 5 项，概览当前项 |
| `overview-390-zh-dark.png` / `overview-390-en-dark.png` | 390 概览，Dock 5 项，无溢出 |
| `overview-regular-1440-zh-dark.png` | 普通用户，Dock 4 项，无管理后台 |
| `overview-footer-1440-zh-dark.png` | 概览页脚 Home 链接（营销首页入口） |
| `dock-tooltip-models-1440-zh-dark.png` | hover“模型广场”tooltip |
| `dock-tooltip-models-1440-en-dark.png` | hover “Model plaza” tooltip |
| `dock-tooltip-profile-1440-zh-dark.png` | 头像 hover“个人资料”tooltip |
| `dock-tooltip-focus-1440-zh-dark.png` | 键盘 focus-visible 白环 + “概览”tooltip |
| `profile-dock-1440-zh-dark.png` | 个人资料页头像当前项白环 |

---

## 5. 未做项 / 偏差

1. 主管理后台 `/` 有自己的侧栏和页面壳，不加载门户 Dock；`/` → 管理后台的当前项映射已写进 `markNav`，但主管理后台页面看不到门户 Dock。`/portal/admin-models` 会正确高亮管理后台项。未改管理后台。
2. 营销首页 `/portal/` 的 hero 标题仍是营销 display 尺寸（52px），应用页标题为 32/700；字体栈、字重、数字均统一。若要求营销 hero 也收敛到 28–32，会牺牲营销页层级，本轮未改。
3. 移动端 tooltip 按需求禁用，保留 aria-label；没有做长按/点击浮层。
4. `DOCK_ITEMS` 的 order/删除/新增通过数组和 `renderDock()` 生效；页面没有提供运行时编辑 Dock 的入口（不需要，也不应暴露配置编辑）。
5. 登录 `?next=` 现在被忽略，统一到概览；这是本轮明确要求，不是回归。
6. 页脚导航只在已登录页面和首页可见；登录/注册页用品牌 Logo 回营销首页。
7. 未 commit / push。
