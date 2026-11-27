# P1 最终交付核对与冻结报告

> 日期：2026-11-23
> 范围：只做交付一致性核对；不重新审计、不重构、不进入 P2
> 结论：**满足冻结条件，可冻结并等待人工批准**。核对中发现的问题是文档计数偏差与 P2 级代码清理项，不改变 P1 结论。

---

## 1. 冻结判定

| 条件 | 结果 | 依据 |
|---|---|---|
| P1 实现与报告一致 | ✅ | 代码、CSS、i18n 与两份 P1 报告描述的功能一致；未发现功能缺失 |
| Token 表与当前 CSS 一致 | ✅（本轮已校正） | DESIGN.md 32 行 token 逐一与 `app.css` 比对，0 不符 |
| 证据文件完整 | ✅（1 项不可复现） | 截图与 4 份 JSON 齐全；`brand-scan` 的“总命中 166”无法从 JSON 复算 |
| 无未记录的代码修改 | ✅ | 收尾报告之后无业务代码改动；收尾期间的改动均有报告条目 |
| Git 受保护 | ✅ | 分支/HEAD 未变，暂存 0，无禁止操作 |
| 可以冻结 | **是** | 见 §9 的附带条件（均为文档项，不是代码阻塞） |

---

## 2. Git 状态与保护确认

| 项 | 开工记录（P1 收尾交付时） | 本次核对时 |
|---|---|---|
| 分支 | `remove-partners` | `remove-partners`（未变） |
| HEAD | `e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e` | 同（未变） |
| `git status --porcelain` | 79 行 | **79 行** |
| 已跟踪修改 | 46 | **46** |
| 未跟踪条目（目录折叠） | 33 | **33** |
| 暂存区 | 0 | **0** |
| `git diff --stat` | 46 files / 1316 insertions / 683 deletions | **同** |

未跟踪条目展开后为 **10,288 个文件**，其中：

- `.mimosa/` 相关工具状态 **10,090 个**（hook-state / history / reports 等），非产品代码；
- 其余 **198 个**：`docs/product-v2/`（报告与截图）、`internal/gui/assets/portal/`、`internal/gui/assets/index-web.html`、Phase 1–8 的新增源码与测试。

本轮**未执行** checkout / reset / clean / stash / cherry-pick / commit / push；未修改业务代码、CSS、JS、Go、配置、Schema 或迁移。唯一的文件改动是 `docs/product-v2/DESIGN.md`（Token 数值同步，见 §5）。

---

## 3. 报告与证据完整性

### 3.1 文档清单

| 文件 | 大小 | 行数 | 状态 |
|---|---|---|---|
| `docs/product-v2/UI-AUDIT.md` | 30,104 B | 528 | 在 |
| `docs/product-v2/DESIGN.md` | 48,884 B | 848 | 在（本轮 v1.2） |
| `docs/product-v2/UI-IMPLEMENTATION-PLAN.md` | 28,757 B | 592 | 在 |
| `docs/product-v2/P0-REPORT.md` | 12,291 B | 273 | 在 |
| `docs/product-v2/P1-REPORT.md` | 17,779 B | 338 | 在 |
| `docs/product-v2/P1-CLOSURE-REPORT.md` | 12,949 B | 250 | 在 |

### 3.2 证据目录

| 路径 | 内容 | 实测 |
|---|---|---|
| `screenshots/`（根） | 审计截图 | 38 PNG + `report.json`、`report-admin.json` |
| `screenshots/p0/` | P0 证据 | **52 PNG** + `report.json`（5.6 MB） |
| `screenshots/p1/` | P1 证据 | **27 PNG** + `theme-report.json`、`layout-report.json`（3.5 MB） |
| `screenshots/p1-closure/` | 收尾证据 | **7 PNG** + 4 JSON（684 KB） |
| `screenshots/p1-wip/` | 过程图（未在报告中列为交付物） | 11 PNG |

JSON 内部一致性复核：

| 文件 | 报告说法 | 实测 | 判定 |
|---|---|---|---|
| `contrast-after.json` | 22 组合 0 失败 | 22 行，`failures` 全空 | ✅ |
| `brand-scan.json` | 82 唯一命中、门户 0 | 82 条；plugins 70 + masking/gateway/settings 各 4；portal 0 | ✅ |
| `closure-check.json` | 44 组布局 0 溢出；语言切换保留 URL | 44 条 `overflow:false`；URL `/portal/?x=1&y=2` 前后一致，lang zh-CN→en，`settingsUnchanged:true`，输入框清空 | ✅ |
| `theme-report.json` | 5 个主题状态；隐私检查 | 5 行 + privacy；CSP 错误 0 | ✅ |
| `layout-report.json` | 320/390/768/1440 × 中英 | 41 行，溢出 0 | ✅ |
| `contrast-before.json` | 123 处 / 118 唯一 | **123 处 / 114 唯一**（见下） | ⚠️ 计数口径 |

### 3.3 发现的不一致（均为文档计数，不影响代码与结论）

| # | 位置 | 报告写的 | 证据实际 | 影响 |
|---|---|---|---|---|
| 1 | P1-CLOSURE §1 / §6.2 / §8 | “118 唯一失败” | 123 次出现、**114 条唯一**（按 text+fg+bg+size；按 text 为 108）；**118 是浅色出现次数**，深色 5 | 结论（修复后 0）不变；数字应改为 114 或写成“浅色 118 次” |
| 2 | P1-CLOSURE §6.3 | app.css 替换 “26 green / 57 red / 42 amber、3 处 border-color” | 与收尾快照 diff：**25 green / 57 red / 42 amber、4 处 border-color** | 低；最终 0 失败已由浏览器实测 |
| 3 | P1-CLOSURE §4 | zh-TW “297 处 供應商” | 当前 **308**（297 为修正 11 处前的数量） | 低 |
| 4 | P1-CLOSURE §3.3 | “总命中 166（去重后 82）” | JSON 只有 82 条去重记录，166 无法复算 | 低；82 与门户 0 已验证 |
| 5 | P1-REPORT §2.2 vs P1-CLOSURE §2 | 1054 insertions / 480 deletions | 收尾基线记录 1055 / 481 | 1 行差；不影响 |
| 6 | P1-REPORT §7.1 | “强制浅色在深色系统、强制深色在浅色系统均实测” | `theme-report.json` 只有 5 个常规状态；强制主题截图在 `p1-wip/`，未列入交付清单 | 低；证据存在但未归档到 `p1/` |
| 7 | P1-REPORT §4.4 vs `theme-report.json` | look.css 1,520 B（设置强调色时） | privacy 记录 `portalBytes: 0`（默认强调色） | 非矛盾：两种设置下的不同测量 |
| 8 | P1-REPORT §5.3 / P1-CLOSURE §3.1 | app.js 125 处 / 280 个 t() 键 | 当前 `t("…magpie…")` 唯一字面量 **271**、出现 289 处；源码内 `magpie` 出现 775 次 | 口径不同（渲染处 / 键 / 源码），建议报告注明口径 |
| 9 | P1-CLOSURE §2 | 列出 4 个“新增的已跟踪修改” | 文件数 43→46 只增 3 个；`presets.go` 在收尾基线前已是修改状态（报告括注已说明“追加在既有修改上”） | 归档口径；不影响内容 |

---

## 4. 未记录的代码修改检查

### 4.1 收尾之后（14:17 报告生成 → 现在）

- 业务文件最新 mtime：`assets/i18n.js` **14:09**；其后只有 `P1-CLOSURE-REPORT.md`（14:19）与 `DESIGN.md`（本轮）被写。
- **未发现收尾后新增或改动的代码文件。**

### 4.2 收尾期间（13:24 快照 → 交付）与报告的对应

用 `/tmp/close-before.tar`（13:24，含 14 个 P1 文件）做逐文件比对：

| 文件 | 与快照差异 | 报告中的记录 |
|---|---|---|
| `assets/app.css` | +146 / −134 行；124 处状态文字换 `-text` token、4 处 border | §6.3 ✅（计数偏差见 §3.3） |
| `assets/i18n.js` | +92 / −45 行；`rebrandText`/`BRAND_KEEP`、术语、补键 | §3.1、§3.4、§4 ✅ |
| `assets/index-web.html` | +443 B：门户入口改为图标 + 文字、`aria-label` | §7 记录了行为，但 **§2 文件清单未列出该文件** |
| `assets/portal/i18n.js` | 4 处“提供商→供应商” | §4 ✅ |
| `internal/gui/i18n_parity_test.go`（新） | 四语键集一致测试 | §3.4、§2 ✅ |

`plugins.css`、`presets.go`、`instancename.js`、`routing/client.js` 不在快照内，mtime 均在 13:40–13:46，且都有报告条目（§6.3、§3.2）。

结论：**不存在未记录的代码修改**；唯一清单缺口是 `index-web.html`（行为已在 §7 说明，未进 §2 文件表）。

---

## 5. DESIGN.md Token 同步结果

### 5.1 发现的结构性问题

DESIGN 的 §2 用一套“门户名”（`--text-quiet` / `--surface`）描述共享 token，但后台 `app.css` 实际用 `--muted` / `--faint` / `--ctl-fg` / `--card`。P1 收尾上调后台文字色后，表里的值停留在旧值，且 §2.5 仍写着“`--text-faint` 永不做文字”，与代码（`var(--faint)` 121 处用法，其中 99 处是 `color:`）直接冲突。本轮只改文档。

### 5.2 修改前后（逐项）

| 位置 | 修改前 | 修改后 | 代码依据 |
|---|---|---|---|
| 头部版本 | v1.1，未提实现 | v1.2，注明按 P1 收尾校正 Token | — |
| §2.1 `--text-muted` | `#85858d`，3.33/3.66 ❌ | `#63636b`，5.42/5.95 ✅，与 quiet 同值 | `app.css:20` |
| §2.1 `--text-faint` | `#b9b9c2`，1.77/1.95 ❌，“只用于装饰” | `#6b6b74`，4.80/5.28 ✅，三级弱文字 | `app.css:21` |
| §2.1 `--text-quiet` 来源 | 仅 `app.css:15`（`--ctl-fg`） | 补 `app.css:20`（`--muted`）同值 | `app.css:15,20` |
| §2.1 新增说明 | — | 命名映射（门户名 ↔ 后台名；门户 `--muted/--faint` 是别名；`--text-muted/--text-faint` 0 引用） | `portal.css:24-26,94-95` |
| §2.2 `--accent-hover` 来源 | `portal.css:143` | `portal.css:295` | 实测 |
| §2.3 成功 text | 5.69:1；来源 `portal.css:21` | 页面 5.18 / 白底 5.69 / 软底 5.07；来源 `app.css:36` + `portal.css:36` | 实测 |
| §2.3 危险 soft | `#fef3f2`（单值） | 门户 `#fef3f2` / 后台 `#fbe9e9`；fill 来源 34→37 | `app.css:37-39` |
| §2.3 警示 text | `#b54708`（单值） | 门户 `#b54708` / 后台 `#a54d08` | `app.css:46` |
| §2.3 深色行 | 单一 soft/text + 旧行号 | 标明后台/门户分叉 + 实际行号 | `app.css:85-93` |
| §2.3 新增说明 | — | 记录两处分叉（均达 AA） | — |
| §2.4 深色表 | `--text-muted #90909a`、`--text-faint #90909a`；行号 56–79,45,102 | 两者均 `#9c9ca6`；行号 61–84,50,96 | `app.css:61-96` |
| §2.5 规则 1 | “`--text-faint` 永不做文字” | `--faint` 为达 AA 的三级文字；`--text-muted` 已并入 quiet | 121 处用法（99 处 `color:`）+ 实测 |
| §2.5 规则 2 | 11px 必须 `--text-quiet` 或更深 | 11px 必须使用达 AA 的 token（quiet 或 faint） | `.field .k` 等 |
| §2.5 规则 3 行号 | `app.css:1133` | `app.css:1141` | 实测 |
| §2.5 规则 6 行号 | `app.css:52` / `:106` | `app.css:57` / `:100,114` | 实测 |
| §2.5 规则 7 行号 | `app.css:123-130` | `app.css:127-138` | 实测 |
| §7 输入框占位符 | `--text-quiet`（不用 faint） | `--faint`（4.80 / 5.76 ✅） | `app.css:2597…`、`portal.css:387` |
| §14 portal `--muted` 行 | 目标 `#63636b` | 注明实现为别名 | `portal.css:94` |
| §14 portal `--faint` 行 | 目标 `#b9b9c2`，只留装饰 | 实现为 `var(--text-quiet)` `#63636b`；`#b9b9c2` 未采用 | `portal.css:95` |
| §14 后台说明 | “只改三处…其余不动” | 注明收尾又上调了文字与状态 token | — |
| §15 验收 1 | “当前 23 处” | 实测 4 处（代码块 3 色 + 开关滑块 `#fff`），未达 0 | `portal.css:366-367,767` |

### 5.3 同步后的校验

- 用脚本把 DESIGN §2 的所有 token 行与 `app.css` 的浅色/深色值逐一比对：**32 行，0 处不符**。
- 状态色三元组与 `app.css` / `portal.css` 逐值核对，全部一致；分叉项已在表中标注。
- 对比度用 WCAG 相对亮度公式重算，表内数字（5.18/5.69/5.07 等）均可复现。

`DESIGN.md` 仍是**未跟踪文档**，本轮修改不进入任何提交。

---

## 6. 代码侧的遗留（不在本轮允许范围，未改动）

| # | 项 | 事实 | 影响 | 建议 |
|---|---|---|---|---|
| C1 | 深色代码注释色不一致 | `app.css:99`（系统深色）`--tk-c: #9c9ca6`，`:113`（`data-theme=dark`）`--tk-c: #8b8b94` | 同一深色主题，经系统偏好进入与经设置强制进入时注释色不同；两者均 ≥4.5:1（5.76 / 4.64） | P2 统一 |
| C2 | 门户死 token | `portal.css:25-26,120-121,160-161` 仍声明 `--text-muted #85858d` / `--text-faint #b9b9c2`（深色 `#90909a`），0 处 `var()` 引用 | 无渲染影响；但会让读者以为存在第三档灰色 | P2 删除或改为别名 |
| C3 | 门户 token 块外仍有 4 处 hex | `portal.css:366-367` 代码块 3 色；`:767` 开关滑块 `#fff` | DESIGN §15 目标“0 处”未达成 | P2 提 token |
| C4 | 后台/门户状态色分叉 | 警示文字 `#a54d08` vs `#b54708`；危险 soft `#fbe9e9`/`#45272a` vs `#fef3f2`/`#2a1717` | 均达 AA；两端语义一致、色值略异 | 已在 DESIGN v1.2 记录，P2 决定是否统一 |
| C5 | `--faint` 两端不同值 | 后台 `#6b6b74`（4.80:1）；门户别名到 `--text-quiet` `#63636b`（5.42:1） | 门户的三级文字略深；都达 AA | P2 决定是否统一 |

---

## 7. 未解决事项（文档，影响冻结但不阻塞）

1. **DESIGN §1.5 “全仓库无 `MiniRelay` 字符串” 已为假**（P1 已落品牌）。超出“只改 Token”的授权，未改；建议 P2 首项修。
2. **DESIGN 头部“状态：草案，待确认；本轮不落地任何代码” 已过时**（P1/P0 已实施）。未改（非 Token 内容）。
3. **DESIGN §15 第 3–5 条的“当前 751 / 30 / 15”** 是 P0 前基线，现行值为 0 溢出、44px、16px；属验收历史，未改。
4. **行号引用仍大面积漂移**（P1 在 `app.css` 前部加了注释与 token）：§2.6 的 `app.css:54,95,115`、`114-116`、`123-130`、`4798`、`149`，§3.1 的 `app.css:47`/`portal.css:19`，§6/§7 的组件行号等。本轮只修了 Token 表与相关规则中的行号，其余未动。
5. **DESIGN §14 的“现值”列**描述的是 P1 之前的 `portal.css`（文件已重写），作为施工单历史保留。
6. **P1-REPORT 的历史结论已被收尾覆盖**：§9.2 “管理端对比度未修”、§5.3 “管理端 magpie 未扫”、§4.1 旧文字 token —— 保留为当时状态，不再回改。
7. **P0-REPORT §9.2 说 ja/de 缺 “User portal”** —— 收尾已补齐三语，历史记录。
8. **强制主题证据未归档**：`p1-wip/theme-dark-os.png`、`theme-forced-dark-on-light-os.png` 存在，但 `p1/theme-report.json` 无对应行；如需作为冻结证据，应移入 `p1/` 并附图注（本轮按“不动证据”要求未移动）。

---

## 8. 针对性检查（未重跑全量）

| 检查 | 命令 | 结果 |
|---|---|---|
| 四语键集一致 + 门户键覆盖 | `go test -tags nogui ./internal/gui -run 'TestI18nLanguagesAgree\|TestPortalI18nKeys' -count=1` | **ok 0.776s** |
| Token 表 vs CSS | 只读脚本逐行比对 | 32 行，0 不符 |
| 证据 JSON | 只读解析 + 计数 | 见 §3.2、§3.3 |
| 文件时效 | mtime + 快照 diff | 见 §4 |

全量 `go test ./internal/gui`（168.534s）等结果**未重跑**，按任务要求沿用 P1 报告记录，本次不做复核。

---

## 9. 冻结结论与附带条件

**P1 可以冻结。** 业务实现、Token 层、主题复用、品牌显示层、i18n 与对比度修复均已落地，且当前代码与 DESIGN v1.2 的 Token 表完全一致；Git 未被改动，证据链完整。

冻结的附带条件（不阻塞冻结，进入待办）：

1. 修正 P1-CLOSURE 的四组计数：唯一对比度失败 118→114（或改口径）、green 替换 26→25、border 3→4、zh-TW 297→308；补 `index-web.html` 到 §2 文件清单。
2. 修正 P1-REPORT 的 diffstat 1 行差与 `theme-report` 强制主题证据的归档。
3. P2 首项：修 DESIGN §1.5 / 头部的过时状态句，并把 §15 的“当前”基线标注为历史。
4. P2 代码清理：`--tk-c` 深色统一、门户死 token、4 处 token 块外 hex、状态色/`--faint` 两端取舍。

未进入 P2；未 commit / push；等待人工批准。