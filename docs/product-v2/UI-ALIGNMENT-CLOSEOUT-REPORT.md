# UI 对齐收尾报告（2026-11-24）

> 范围：只做实际页面与 `docs/product-v2/DESIGN.md` 的视觉/交互对齐收尾与真实浏览器验收
> 基线：`/root/magpie-original`，分支 `remove-partners`，HEAD `e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e`
> 结论：P0/P1 已验收项未重做；本轮修复 24 项剩余偏差，浏览器验收全部通过；未 commit / push

---

## 1. 结果摘要

| 项 | 结果 |
|---|---|
| 改动范围 | 只改 `portal.css`、`app.css` 与 `DESIGN.md`；未改 JS / HTML / Go / 路由 / API / 数据 |
| 门户视觉 | 弹窗 600px、遮罩 28%+blur8、主按钮光晕只留首屏、页脚固定底部、空用量图收高 |
| 门户交互 | 输入/下拉/文本域焦点环、全站链接 2px 焦点环、移动端 Key 行换行、`linkish` 44px |
| Token | 门户死 token 修正、`--faint` 恢复三级文字、代码块/开关提 token；token 块外 hex = 0 |
| 管理后台 | 控件圆角统一 8px、卡片 12px、两路深色 `--tk-c` 统一、链接/下拉/文本域焦点环 |
| 布局 | 门户 320/390/768 触屏/1440 × 中英 **0 溢出、0 个 <44px 触控目标、输入 16px** |
| 对比度 | 22 个页面×主题组合 **0 失败** |
| 主题 | 浅/深/跟随系统/强制浅/强制深/自定义强调色/色盲/减少动效全部符合 |
| 语言 | 切换保留路径与查询参数，Cookie 生效，管理员全局设置未变 |
| 证据 | `screenshots/p2-ui-align/`，49 个文件（44 图 + 5 JSON，8.0 MB） |

---

## 2. Git 保护

| 项 | 基线 | 交付时 |
|---|---|---|
| 分支 / HEAD | `remove-partners` / `e41adc151` | 未变 |
| `git status --porcelain` | 89 行 | **89 行** |
| 已跟踪修改 | 50 文件 | 50 文件（仅 `app.css` 追加 83/77） |
| `git diff --stat` | `50 files, 1380 insertions(+), 692 deletions(-)` | `50 files, 1463 insertions(+), 769 deletions(-)` |
| 暂存区 | 0 | 0 |
| 未跟踪 | 39 项（含 docs / portal / 新源码） | 39 项（docs 内新增本报告与证据） |

未执行 checkout / reset / clean / stash / cherry-pick / commit / push。仅使用 `/tmp` 隔离 HOME 与临时编译产物；临时数据未写入仓库。

---

## 3. 真实浏览器验收环境

| 项 | 值 |
|---|---|
| 浏览器 | Playwright + Chromium（首批 Playwright 1254；补充验收使用系统 Chromium 150） |
| 服务 | 当前工作树编译的 `magpie web`，`127.0.0.1:3440`，隔离 `XDG_CONFIG_HOME=/tmp/minirelay-ui-home` |
| 账号 | 临时管理员 `admin@ui.test`；临时普通用户；临时供应商与 3 个开放模型；临时 API Key |
| 宽度 | 320 / 390 / 768（鼠标 + 触屏）/ 1440 |
| 语言 | 中文 / 英文 |
| 主题 | 浅色 / 深色 / 跟随系统 / 强制 / 自定义强调色 / 色盲 / 减少动效 |
| 页面 | 门户首页、登录、控制台、模型广场、模型管理；后台 Providers / Routing / Usage / Plugins / Masking / Gateway / Settings |

说明：模型、Key、请求数据均为隔离实例的临时夹具；未调用外部厂商，未修改产品数据。

---

## 4. 修复清单

### 4.1 门户（`internal/gui/assets/portal/portal.css`）

| # | 问题 | 修复 | 依据 |
|---|---|---|---|
| P-1 | 主按钮光晕出现在控制台、管理入口、登录/注册等所有 primary 上 | 光晕只保留 `.hero .btn.primary`，其余 primary 平面 | DESIGN §6.4 |
| P-2 | 输入框/下拉/文本域聚焦只换边框色，无可见焦点环 | 统一 `border + 0 0 0 3px accent 22%`；链接/按钮/`[tabindex]` 补 2px accent 焦点环 | §6.2 / §8.1 / §13.3 |
| P-3 | 弹窗 660px、遮罩 55% + blur 2px，与管理端弹窗语言不同 | 改 `min(600px, 100vw-32)`、14px 圆角、遮罩 28% + blur 8px saturate 1.1；头/脚 48px + 发丝分隔 | §9.3 |
| P-4 | 短页面页脚悬在内容后，底部留一条空底色 | `body` 列向 flex，`main` 撑满，页脚固定底部 | 布局一致性 |
| P-5 | 字号/字重越出档位（550/650、13.5/19、11.5 等） | 字重归 400/500/600/700；字号归 12/12.5/13/15/16/17/20/24 等档位 | §3.2 / §3.3 |
| P-6 | 日志与模型管理表头 11px 常规字重 | 10.5px / 600 / `.04em` / 大写 | §9.2 |
| P-7 | 门户 `--faint` 并入 `--text-quiet`，三级文字层级消失；`--text-muted` 仍是旧值 | `--text-faint #6b6b74`（深 `#9c9ca6`）并让 `--faint` 指向它；`--text-muted` 同步 `#63636b` | §2.1 / §2.5 |
| P-8 | 代码块 3 个 hex、开关滑块 `#fff` 在 token 块外 | 新增 `--code-bg/-fg/-line`、`--switch-knob`；token 块外 hex = 0 | §15.1 |
| P-9 | 门户 `<select>` 不继承界面字体，桌面下拉比输入框小 | `button, input, select, textarea { font: inherit }` | §8.2 |
| P-10 | 空用量图保留 166px 空白与分隔线 | `.usage-chart:has(.key-empty)` 收高并去掉分隔线 | §11.2 |
| P-11 | 320px 英文下 API Key 行把页面撑到 373px | 移动端 Key 行 `minmax(0,1fr)`，操作区允许换行 | §12.2 / §12.3 |
| P-12 | 移动端「编辑」「全选/全不选」等 `linkish` 仅 19px | 移动/触屏 `linkish` 最小 44px | §12.2 |
| P-13 | `key-model-pick:empty::before` 硬编码中文，英文页会漏出 | 删除 CSS 伪内容，空态由 `usage.js/portal.js` 的 `t()` 文案承担 | §4.1 / D2 |
| P-14 | eyebrow 12/700/.12em，偏离分区标签规则 | 12/600/.05em；中文 `.02em` | §3.3 |
| P-15 | 门户内容/卡片/弹窗控件圆角与档位不完全一致 | 小按钮回 `--r-sm`，大按钮/弹窗/卡片按 `--r-ctl/-card/-dialog` | §6.1 / §7.2 |

### 4.2 管理后台（`internal/gui/assets/app.css`）

| # | 问题 | 修复 | 依据 |
|---|---|---|---|
| A-1 | 同一深色主题，系统深色 `--tk-c #9c9ca6`、强制深色 `#8b8b94` | 两路统一 `#9c9ca6` | P1 冻结 C1 / §2.5 |
| A-2 | 控件圆角出现 7px（输入、下拉、菜单、图标按钮等 43 处） | 统一 8px（r-ctl） | §6.1 |
| A-3 | 卡片级圆角 10/11px（订阅卡、popover、用量卡、账号组等） | 统一 12px（r-card） | §6.1 |
| A-4 | 侧栏选中底 7px，与 8px 侧栏项不齐 | 侧栏项与选中底均 8px | §10.1 |
| A-5 | 链接、下拉、文本域无统一键盘焦点环 | 增 `:where(a, select, textarea, [tabindex]):focus-visible` 2px accent / offset 2 | §6.2 / §13.3 |
| A-6 | 深色面板的 `--faint #90909a` 与主深色 `#9c9ca6` 分叉 | 面板深色同步 `#9c9ca6` | §2.4 |
| A-7 | 390px 设置页 8 个分区标签仅 22px 高 | 移动/触屏下 `body.web .warm-tabs button` 44px | §12.2 |
| A-8 | 供应商/Agent 拖拽把手视觉 28px、命中区不足 | 保持 28px 外观，`::after` 扩到 48px 命中区；供应商行最小 44px | §12.2 |
| A-9 | 内嵌卡片仍用 10/11px 圆角（订阅卡、prof-detail、提示、用量卡、设置向导等） | 统一到 12px；tab 轨道用 8px；仅保留滚动条、圆形徽标和条形端头的特殊值 | §6.1 |

### 4.3 文档

`DESIGN.md` 升到 v1.3：状态改为“已落地”；§1.5 过时的“全仓库无 MiniRelay”改为 P1 已落地；§2.1 / §14 同步门户 `--faint`；§15 第 1/3/4/5 项改为 2026-11-24 实测值；新增 v1.3 修订记录。

---

## 5. 验收矩阵

### 5.1 布局（`layout-report.json`）

| 范围 | 溢出 | <44px 触控目标 | 输入字号 |
|---|---|---|---|
| 门户 320 / 390，中/英，5 页 | **0** | **0** | 16px |
| 门户 768 触屏控制台 | **0** | **0** | 16px |
| 门户 1440，中/英 | **0** | 桌面密度（不适用 44px） | 15px |
| 后台 390 / 1440 | **0** | 设置分区标签 44px；分段控件 28px 外观 + 44px 命中区；颜色圆点 18px 见 §6 | 16px（移动）/ 12–13px（桌面） |

短页面（模型广场 1440）实测 `footer.bottom = 900 = viewport`，页脚贴底；空用量图实测 21px 高，不再占 166px。

### 5.2 对比度（`contrast-after.json`）

11 页 × 浅/深 = **22 个组合，失败 0，禁用态失败 0**。覆盖后台 Providers/Routing/Usage/Plugins/Masking/Gateway/Settings 与门户首页/控制台/模型广场/模型管理。

### 5.3 主题与隐私（`theme-i18n-report.json`）

| 状态 | 结果 |
|---|---|
| 浅色默认 / 强制浅色（深色系统） | 浅底 `rgb(244,244,246)`，accent `#4f46e5` |
| 深色 / 强制深色（浅色系统） | 深底 `rgb(26,26,30)`，accent `#a5b4fc` |
| `system` + OS 浅/深 | 自动浅/深，无 `data-theme` |
| 自定义 accent `#0d9488` | 派生 `#0b7c72`，表面染色生效 |
| 自定义 accent `#db2777` + 深色 | 派生 `#e35795`，底色 `rgb(33,19,29)` |
| 色盲 | `--c1 #0072b2`（Okabe–Ito） |
| 减少动效 | `data-motion="reduce"`，过渡 0s |
| 隐私 | `look.css` 1520 B，无壁纸/字体；门户不请求 `/api/appearance/*` |
| 控制台错误 | 0（仅匿名 `/api/auth/me` 401 属既有探测） |

### 5.4 语言（`theme-i18n-report.json`）

- `/portal/?x=1&y=2` 中文 → 点 English：URL **不变**，`lang=en`，Cookie `minirelay_lang=en`，`settings.lang=system` 未变。
- 英文首页/控制台/模型广场/模型管理可见文本中仅剩「中文」——语言开关自身的语言名，属预期。
- 下拉/输入字体一致；`key-model-pick` 空态不再有中文伪元素。

### 5.5 管理后台移动端与 iOS 视口（`admin-mobile-ios-report.json`）

| 检查 | 390 鼠标 | 390 触屏 | 768 触屏 |
|---|---|---|---|
| 页面横向溢出 | 0（docW 390） | 0 | 0 |
| 设置分区标签高 | 44px | 44px | 44px |
| 供应商行高 | 79px | 79px | 79px |
| 拖拽把手 | 28px 外观 / `inset:-10px` / 中心外 20px 命中 | 同左 | 同左 |
| 设置搜索框 | 354×44 / 8px 圆角 / 16px | 同左 | 732×44 / 8px / 16px |
| 设置列表圆角 | 12px | 12px | 12px |

iOS 视口模拟（iPhone 390×844、DPR 3、触屏、Safari UA，Chromium 引擎）：首页/登录/控制台/模型广场/模型管理 **docW = 390**、输入 16px；统计到的 22px 开关均有既有 44px 命中区，剩余小尺寸仅为屏幕外 skip link 与 18px 颜色圆点。**未做真机 iOS Safari**：宿主没有 iOS 设备；WebKit 引擎因系统缺少 Ubuntu 版 ICU/JPEG 依赖无法启动，已在 §6 记录。

### 5.6 交互（`interaction-report.json`）

| 检查 | 结果 |
|---|---|
| 首页 primary 光晕 | 仅首屏 CTA 1 个 |
| 控制台 primary | 2 个均无光晕 |
| 焦点 | 品牌链接 `2px accent`；搜索/下拉 `3px accent 22%`；下拉字号 15px |
| 模型弹窗 | 600×516，14px 圆角，遮罩 28% + blur 8 + saturate 1.1；Esc 关闭，焦点回到模型卡 |
| 创建 Key | 表单展开 → 填名 → 选模型 → 提交，完整密钥区出现，列表含新 Key |
| 登录失败 | 表单错误显示「邮箱或密码不正确。」，仍停留登录页 |
| 320 英文控制台 | 文档宽 320 = 视口；全部 12 个 Key 操作按钮均 44px |
| 后台焦点 | 门户入口链接 2px accent；浅/深/强制深 `--tk-c` 分别为 `#63636b` / `#9c9ca6` / `#9c9ca6` |

---

## 6. 未做 / 剩余

1. **颜色圆点**：设置页 7 个 accent 色板仍为 18px 外观，未扩命中区；同排间距小，扩大命中区会互相重叠，留待单独设计。
2. **分段选项**：设置页的 28px 外观分段控件依靠既有 `::after` 获得 44px 命中区，视觉仍为 28px；这是后台紧凑密度的有意选择。
3. **状态色分叉**：门户警示文字 `#b54708` 与后台 `#a54d08`、危险 soft `#fef3f2` 与 `#fbe9e9` 仍按 DESIGN §2.3 记录保留，两者均达 AA。
4. **门户模型卡 hover** 使用 accent 12% 投影，不在 §6.3 四档阴影表内；属既有交互反馈，本轮未改。
5. **真机**：未做 iOS Safari 真机。宿主没有 iOS 设备；Playwright WebKit 已下载但系统缺少 Ubuntu 版 ICU/JPEG 依赖，无法启动。已完成 iPhone 390×844 触屏视口模拟，结果见 §5.5。
6. **测试范围**：`go build -tags nogui` 通过；`go test -tags nogui ./internal/gui -run 'Portal|Asset|Budget|Compress|Revalidate|I18n|WebApp|InstanceName'` **ok 1.038s**；未重跑全量 gui / gateway（CSS 改动不触及后端逻辑）。
7. **提交**：未 commit / push；工作树仍有 P0/P1/P2 全部改动与未跟踪文档。

---

## 7. 证据索引

| 路径 | 内容 |
|---|---|
| `docs/product-v2/screenshots/p2-ui-align/layout-report.json` | 45 组布局实测（320/390/768/1440 × 中英 × 门户 + 后台 390/1440） |
| `.../layout-{zh,en}-{390,1440}-*.png` | 门户 5 页与后台 2 页截图 |
| `.../layout-zh-768-touch-console.png` | 768 触屏控制台 |
| `.../contrast-after.json` | 22 个页面×主题组合的对比度失败清单（0） |
| `.../theme-i18n-report.json` | 8 个主题状态、隐私、语言切换与英文残留 |
| `.../interaction-report.json` | 光晕、焦点、弹窗、创建 Key、登录错误、320 英文 |
| `.../theme-*.png` | 浅/深/系统/强制/色盲/减少动效截图 |
| `.../dialog-model-1440-light.png`、`dialog-admin-edit-1440-light.png` | 门户两类弹窗 |
| `.../admin-{providers,settings}-1440-dark.png` | 后台深色 |
| `.../verify-console-en-320.png` | 320 英文 Key 行换行 |
| `.../admin-mobile-ios-report.json` | 后台 390 鼠标/390 触屏/768 触屏 + iPhone 视口模拟 |
| `.../verify-admin-settings-{390,768}-*.png` | 后台设置 44px 分区标签 |
| `.../ios-emulation-{home,console,models}-390.png` | iPhone 390 视口截图 |
| `/tmp/ui-align/*.cjs` | 本次验收脚本（临时，未入库） |
| `/tmp/minirelay-ui-bin` | 本次编译产物（临时，未入库） |

---

## 8. 结论

门户与后台在 token、圆角、状态色、焦点、弹窗与移动端行为上已按 DESIGN v1.3 对齐；真实浏览器 22 个对比度组合、45 组布局、8 个主题状态、语言切换与主要交互均通过；后台移动端分区标签与拖拽把手已补 44px 命中区，内嵌卡片圆角已统一。剩余项仅为色板/分段控件外观与真机 iOS，不影响本轮“UI 对齐收尾”的交付结论。

**未提交、未推送，等待人工验收。**
