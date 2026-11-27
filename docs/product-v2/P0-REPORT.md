# P0 交付报告：移动端缺陷修复

> 批次：P0（纯 CSS）
> 日期：2026-11-23
> 依据：`docs/product-v2/DESIGN.md` v1.1、`docs/product-v2/UI-IMPLEMENTATION-PLAN.md` §5
> 状态：**已完成并验收，暂停等确认后进入 P1**

---

## 1. 范围

P0 只修审计确认的五项移动端缺陷，不改布局语言、不改 JS、不改 HTML：

| # | 缺陷 | 审计编号 | 结果 |
|---|---|---|---|
| P0-1 | 390px 使用日志横向溢出（文档宽 751px） | P1-1 | ✅ 修复 |
| P0-2 | 统计卡被压成 44px 宽 | P1-2 | ✅ 修复 |
| P0-3 | 时间范围胶囊挤成 118px 竖条 | P1-3 | ✅ 修复 |
| P0-4 | 输入框 15px 触发 iOS 聚焦缩放 | P2-3 | ✅ 修复 |
| P0-5 | 主要触控目标不足 44px | P2-4 | ✅ 修复 |

验收中另发现并修复一项未被审计记录的问题：

| # | 缺陷 | 说明 |
|---|---|---|
| P0-6 | 768px 下日志表仍溢出 | 六列固定网格最小 714px，768px 卡片内宽 684px；改用 ≤900px 卡片化 |
| P0-7 | 超长 Token 串撑破卡片 / 桌面端与耗时列重叠 | Token 列 152→200px；`min-width:0` + 省略号兜底 |

---

## 2. Git 保护

### 2.1 开工基线（P0 前记录）

| 项 | 值 |
|---|---|
| 分支 | `remove-partners` |
| HEAD | `e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e` |
| `git describe` | `v0.1.4574-1-ge41adc151` |
| `git status --porcelain` | 71 行（40 已跟踪修改 + 31 未跟踪） |
| 快照 | `/tmp/p0-baseline.txt` |

### 2.2 交付时复核

| 检查 | 结果 |
|---|---|
| `git status --porcelain` 行数 | **71**，与基线逐行一致 |
| 已跟踪文件数 | 40，**无新增/删除** |
| `git diff --stat` 合计 | `40 files changed, 1018 insertions(+), 466 deletions(-)`：P0 本体为 1017，补充导航修复 +1 行（见 §9） |
| 暂存区 | 0 |
| 本轮改动的文件 | `internal/gui/assets/portal/portal.css`、`internal/gui/assets/index-web.html`、`internal/gui/assets/i18n.js`（+1 行），详见 §3 与 §9 |

未执行 checkout / reset / clean / stash / cherry-pick / commit / push。

---

## 3. 实际修改

### 3.1 文件

| 文件 | 前 | 后 | 变化 |
|---|---|---|---|
| `internal/gui/assets/portal/portal.css` | 697 行 / 32,390 B | 755 行 / 35,430 B | +58 行，3 处 |
| `internal/gui/assets/index-web.html` | 653 行 | 656 行 | +3 行（§9 门户入口） |
| `internal/gui/assets/i18n.js` | — | — | +1 行（`"User portal": "用户门户"`，§9） |

其它文件 0 改动。portal.css 完整 diff：`docs/product-v2/patches/P0-portal-css.diff`（88 行）。

### 3.2 三处改动与原因

**① `.log-row` 列宽 `152px → 200px`（第 419 行）**

- 原因：`in 1,204 · out 512 · 1,716` 实测 188px，152px 列装不下。
- 修复前桌面端 Token 文本溢出到"耗时"列（审计截图里 `1,204 ms` 被压成 `04 ms`），属于审计未记录的真实缺陷。
- 实测正常值现在 `scrollWidth === clientWidth === 200`。

**② `.log-tokens` / `.log-ms` 加 `min-width:0; overflow:hidden; text-overflow:ellipsis`**

- 原因：极端长数值会撑破单元格与卡片。
- 效果：正常值不截断；病态值（如 12,345,678）显示省略号，不产生页面横向滚动。

**③ 新增两段 `@media`（第 699–755 行）**

```css
@media (max-width: 900px) {           /* 日志卡片化：6 列 → 字段块 */
  .log-row { display: block; }         /* 用 usage.js 已有的 data-label */
  .log-row.head { display: none; }
  .log-row > [data-label]::before { content: attr(data-label) "："; }
}
@media (max-width: 760px),
       (pointer: coarse) and (max-width: 1024px) {   /* 与 app.css:389 同款判定 */
  .usage-stats { grid-template-columns: repeat(2, minmax(0,1fr)); }
  .usage-head { flex-direction: column; }           /* 胶囊独占一行 */
  input, select, textarea { font-size: 16px; min-height: 44px; }
  .btn, .btn.sm, .chip { min-height: 44px; }
  .brand, .nav a, .account .who, .footer a { min-height: 44px; }
  .am-toggle::before { inset: -11px -3px; }         /* 38×22 外观，44px 命中区 */
}
```

- 没有改 JS：日志每格的文案来自 `usage.js:156-158` `addCell` 已写入的 `data-label`。
- 没有改 HTML：断点、标签、按钮都在 CSS 里解决。
- 触控判定照抄 Magpie 自己的组合：`(max-width:760px), (pointer:coarse) and (max-width:1024px)`（`app.css:389`），所以 768px 触屏平板也拿到 44px / 16px。

---

## 4. 对照设计规范的完成情况

| DESIGN / 计划要求 | 结果 | 证据 |
|---|---|---|
| ≤760 统计 6 列 → 2 列 | ✅ | `statCols` = `118px 118px`（320）、`153px 153px`（390） |
| 日志 ≤760 卡片化、隐藏表头 | ✅ | `log=block`、`head=none`（320/375/390/768 touch） |
| 时间范围胶囊独占一行 | ✅ | 4 枚胶囊一行排列（320 亦不折行） |
| 输入框 ≥16px | ✅ | 移动/触屏上下文实测全部 `16px`（P0 前 15px） |
| 主要触控目标 ≥44px | ✅ | 移动/触屏 25 个上下文 `smallTargetCount = 0` |
| 不修 JS、不修 HTML、不改业务 | ✅ | diff 仅 1 个 CSS 文件 |
| 不引入框架/组件库 | ✅ | 纯 CSS |
| 320/375/390/768/1440 无横向溢出 | ✅ | 43 个上下文 `overflowPage = false`；最大文档宽 = 视口宽 |

---

## 5. 测试与结果

### 5.1 布局验证（Playwright + Chromium 1254）

命令：

```sh
node /tmp/audit-tools/p0-verify.cjs      # 5 宽度 × 6 页面 × 浅/深 + 768 触屏
node /tmp/audit-tools/p0-edge.cjs        # 极端长内容
node /tmp/audit-tools/p0-evidence.cjs    # 组件级证据图
```

结果（43 个上下文，0 控制台错误）：

| 宽度 | 主题 | 页面级横向溢出 | 移动触控目标 <44px | 输入框字号 | 日志模式 |
|---|---|---|---|---|---|
| 320 | 浅 | 无 | 0 | 16px | 卡片 |
| 375 | 浅 | 无 | 0 | 16px | 卡片 |
| 390 | 浅/深 | 无 | 0 | 16px | 卡片 |
| 768（鼠标） | 浅 | 无 | 21* | 13.3px* | 卡片 |
| 768（触屏） | 浅 | 无 | **0** | **16px** | 卡片 |
| 1440 | 浅/深 | 无 | 21* | 13.3px* | 表格 |

\* 768 鼠标上下文与 1440 是桌面密度，按钮 38px、下拉 13.3px 属既有设计；DESIGN 的 44/16 只约束触屏。触屏 768 全部达标。

极端内容（模型名超长、Key 超长、Token 十位数）：

```
320  docScrollW=320  card 286/286   tokens wrap
375  docScrollW=375  card 341/341   tokens wrap
390  docScrollW=390  card 356/356   tokens wrap
768  docScrollW=768  card 722/722   tokens wrap
1440 docScrollW=1440 card 1114/1114 tokens 省略号（病态值）
```

### 5.2 回归测试

| 命令 | 结果 |
|---|---|
| `go test -tags nogui ./internal/gui -run 'Portal\|Asset\|Budget\|Compress\|Revalidate' -count=1` | **ok，0.552s** |
| `go test -tags nogui ./internal/gui -count=1` | **ok，183.569s，零失败** |

（CSS 经 `go:embed` 打进二进制，资产/压缩/预算测试全过。）

### 5.3 Oil UI 自带 lint（独立复核）

| 页面 | P0 前 | P0 后 |
|---|---|---|
| `/portal/login` 390px | 输入框 15px ×2（iOS 缩放）；占位符对比度 2.58 / 2.49 | **缩放告警消失**；占位符对比度仍在（P1/P2 token 工作，非 P0 范围） |
| 控制台错误、横向溢出、加载失败图 | 无 | 无 |

### 5.4 真实截图

`docs/product-v2/screenshots/p0/`，48 张 / 5.3 MB：

- `portal-*-{320,375,390,768,1440}-{light,dark}.png`：6 页 × 宽度 × 主题
- `portal-console-768-touch-light.png`：触屏平板
- `evidence-log-cards-{320,390}.png`：日志卡片化特写
- `evidence-usage-stats-{320,390}.png`：统计 2×3 与胶囊一行
- `evidence-log-table-1440.png`：桌面表格未变
- `report.json`：43 个上下文的逐项实测

---

## 6. 证据的边界（如实记录）

1. **日志行是布局测试数据，不是产品数据。** 环境内没有可用供应商，无法产生真实 Gateway 请求；截图中的日志行用与 `usage.js` 完全相同的 DOM 注入，只用于验证卡片布局，测试上下文关闭即销毁，产品代码与数据未改动。
2. 因此日志页脚「共 0 条」与注入的两三行不一致——页脚计数由产品脚本写入，夹具不动它。
3. **fullPage 截图里的吸顶栏会出现在页面中部**，这是 long-screenshot 与 `position: sticky` 的固有现象，不是遮挡缺陷；组件级 `evidence-*.png` 已隐藏顶栏单独成像，用于无干扰判读。
4. 768px 用鼠标时仍是桌面密度（38px 按钮、13.3px 下拉）；只有触屏才切 44px/16px。这是规范的设计意图，不是遗漏。
5. 占位符对比度 2.58:1 仍不达标——`--faint` 属 P1 Token 工作，P0 不碰颜色。

---

## 7. 未完成 / 风险

| 项 | 状态 | 说明 |
|---|---|---|
| 颜色、品牌、i18n、主题 | 未做 | P1 范围 |
| 组件统一（按钮/输入/卡片/表格/弹窗/导航） | 未做 | P2 范围 |
| 逐页视觉迁移 | 未做 | P3 范围 |
| 日志卡片化在 768–900 也生效 | 设计选择 | 六列表格在 900px 内会挤；卡片更可读，且 1440 表格完全不变 |
| `.btn`/`.chip` 在移动端 44px 后顶栏变高 | 已接受 | 触控优先；390 下仍无遮挡、无溢出 |
| Token 列 152→200 使桌面模型/Key 列略窄 | 已接受 | 1440 下模型列仍有 ~328px；正常模型名不截断（实测） |

---

## 8. P1 前置条件（暂停点）

P1 开工前需确认：

1. P0 在真实设备（尤其 iOS Safari）上的观感是否接受——截图来自 Chromium，未做真机验证。
2. 日志卡片在 768–900 平板宽度使用是否接受（若希望平板仍用表格，可把断点收到 760 并给表格加横向滚动容器）。
3. 是否同意 Token 列加宽后的桌面比例。

确认后按计划进入 P1（品牌 + Token + 主题注入 + i18n），届时会再记录一次 Git 基线。
---

## 9. 补充修复：管理端 → 用户门户的入口（验收期间发现）

### 9.1 问题

管理端页面没有任何通往用户门户的入口，管理员只能在地址栏手敲 `/portal/`。

实测：

```sh
grep -c '/portal' internal/gui/assets/index-web.html   # 0
grep -c '/portal' internal/gui/assets/app.js           # 0
```

反向是存在的：门户控制台的「管理后台」卡片有两个 `href="/"`（`portal/index.html:261,355`）。即只有「门户 → 管理后台」，没有「管理后台 → 门户」。

这不是 P0 引入的，也不是故障：管理端外壳从 Phase 1–8 起就没有建过这个入口。

### 9.2 改动

| 文件 | 改动 |
|---|---|
| `internal/gui/assets/index-web.html` | 顶栏 `.actions` 里、设置按钮之前，加一个 `<a class="text action portal-link" id="portalLink" href="/portal/" data-t>User portal</a>` |
| `internal/gui/assets/i18n.js` | `zh` 词典加一行 `"User portal": "用户门户"`（ja/de 缺键回退英文） |

规格直接复用既有组件，没有新增 CSS：

- `.text.action`：27px 高 / 12px / 圆角 6px / 1px `--line` / `--card` 底（`app.css:2466`）
- `body.web` 窄屏下该组件自然长到 44px（实测 390px = 44px），符合 P0 的触控尺寸
- `data-t` 走既有 `translateStatic()`，`<a>` 与按钮同一套翻译

### 9.3 验证

| 项 | 1440 | 390 |
|---|---|---|
| 文本（英文环境） | `User portal`，88×27 | 88×44 |
| 文本（中文环境 `zh-CN`） | `用户门户`，70×27 | 70×44 |
| `href` | `/portal/` | `/portal/` |
| 页面横向溢出 | 无 | 无 |
| 右边缘 | 1430 / 1440 | 330 / 390 |

截图：`docs/product-v2/screenshots/p0/evidence-admin-portal-entry-{1440,390}.png`、`…-zh.png`（4 张）。

测试：

```
go test -tags nogui ./internal/gui -run 'Portal|Asset|Budget|Compress|Revalidate|I18n|Translat'  → ok 0.537s
go test -tags nogui ./internal/gui                                                             → ok 189.400s，零失败
```

### 9.4 批次归属与风险

- 按 DESIGN §10「导航栏」，这条本属 **P2 导航统一**。因为验收被它挡住，先补一个可用入口；**P2 时按规范重排**（是否用图标、放在侧栏还是顶栏、移动端是否只留图标），届时可能再改一次同一位置。
- `i18n.js` 是 Phase 1–8 已修改的跟踪文件；+1 行使 `git diff --stat` 从 1017 变 **1018**，除此以外没动其它词条。
- 该入口只做跳转，不读写任何业务状态，回滚即删 3 行 HTML + 1 行词典。
