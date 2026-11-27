# P1 收尾报告：品牌扫尾、术语统一与全站对比度

> 批次：P1 收尾
> 日期：2026-11-23
> 依据：`DESIGN.md` v1.1、`P1-REPORT.md`
> 状态：已完成并自验，未提交。**未进入 P2。**

---

## 1. 结果摘要

| 项 | 结果 |
|---|---|
| 品牌 | 用户可见文案经统一品牌层显示为 MiniRelay；剩余 82 处渲染命中全部是第三方插件名/说明与 `magpie-plugin` 技术标签 |
| 术语 | 中文 Provider 统一为「供应商」：zh 311 处、0 处旧译；zh-TW 修正 11 处混入的简体；日/德保持自然对应 |
| 语言切换 | 保留当前页面与查询参数；Cookie 生效；管理员全局 `lang` 未变；重载丢未提交表单已确认并记录 |
| 对比度 | 修复前 123 处/118 唯一失败；修复后 **22 个页面×主题组合全部 0 失败** |
| 布局 | 4 语言 × 2 宽度 × 管理端/门户共 44 组，**0 横向溢出**；移动端门户入口 44×44 图标 |
| 测试 | gui **ok 168.534s**、provider **ok 79.810s**、library **ok 98.356s**；词表四语一致测试通过；Oil UI lint 通过 |

---

## 2. Git 保护

| 项 | 开工基线 | 交付时 |
|---|---|---|
| 分支 / HEAD | `remove-partners` / `e41adc151` | 未变 |
| `git status` | 75 行 | **79 行**（46 已跟踪修改 + 33 未跟踪） |
| `git diff --stat` | 43 files / 1055 insertions / 481 deletions | **46 files / 1316 insertions / 683 deletions** |
| 暂存 | 0 | 0 |

相对收尾基线新增的已跟踪修改：

- `internal/gui/assets/instancename.js`（实例名占位符与注释）
- `internal/gui/assets/plugins.css`（状态文字色换 token）
- `internal/gui/assets/plugins/routing/client.js`（路由图中心标签）
- `internal/provider/presets.go`（Remote 预设显示名/说明，追加在 Phase 既有修改上）

新增未跟踪：`internal/gui/i18n_parity_test.go`。

未执行 checkout / reset / clean / stash / cherry-pick / commit / push。

---

## 3. 管理端品牌扫尾

### 3.1 为什么不是逐条改 2,600 句

`app.js` 有 280 个 `t()` 英文键含 "magpie"；`i18n.js` 每种语言约 1,650 条译文含它。逐键改名意味着四种语言共约 6,600 条同步迁移，任何一条漏改都会让该语言回退成英文。因此采用**显示层统一**：

- `i18n.js` 新增 `rebrandText()`，在 `t()` 与 `tNotice()` 的输出上执行；
- 只改**品牌词**，保护三类技术标识：
  - 程序/命令：`` `magpie` ``、`magpie web/serve/tui/command/CLI/PATH`、`run|runs|invoking magpie`；
  - 文件/环境/头部：`magpie.cmd/.exe/.ps1/.app/.json/.toml/.js`、`MAGPIE_*`、`X-Magpie-*`；
  - 路径/域名/仓库：`/…magpie…`、`~/…magpie…`、`https://…magpie…`、`usemagpie.ai`、`magpie-community`、`magpie-releases`。
- 插值变量（`{path}`、`{agent}`、`{url}` 等）在替换之后才填入，永远不碰。
- 德文所有格 `magpies` → `MiniRelays`；英文/中文 `magpie's` → `MiniRelay's`。

这是一次可逆、可测的显示层变更，源键仍是内部标识（和包名、环境变量一样不随品牌改）。

### 3.2 结构位置（不经 `t()` 的展示）

| 位置 | 前 | 后 |
|---|---|---|
| 路由图中心节点 | `magpie` | `MiniRelay` |
| 设置 › 实例名占位符 | `magpie` | `MiniRelay` |
| Remote 供应商预设名 | `Remote magpie` | `Remote MiniRelay` |
| Remote 预设说明 | `another computer's magpie…` | `another computer's MiniRelay…` |
| Remote 预设端点提示 / 缺失提示 | `…the other computer's magpie…` / `The other magpie's address…` | MiniRelay |
| Azure 端点提示 | `…magpie asks the resource…` | `…MiniRelay asks…` |

### 3.3 渲染扫描（真实页面）

对 5 种语言（zh、zh-TW、ja、de、en）× 管理端 7 页 + 门户 4 页，遍历可见文本与 `placeholder/title/aria-label`，按同一保护规则过滤：

| 项 | 数量 |
|---|---|
| 总命中 | 166（去重后 82） |
| 门户 | **0** |
| 管理端 | 82，全部落在插件市场 |
| 其中第三方插件名/README | 70 |
| 其中 `magpie-plugin` 社区标签说明（会出现在多页底部） | 12 |

处理决定：第三方插件 ID（`magpie-purge`、`magpie-fingerprint`…）、作者自己的说明文字、`magpie-plugin` 标签都**保留原样**——它们不是本产品品牌，且改名会破坏插件索引与作者署名（任务要求保留第三方名称）。

未发现本产品文案残留。`screenshots/p1-closure/brand-scan.json` 保存了逐条证据（含语言、页面、文本）。

### 3.4 词表一致性测试

新增 `TestI18nLanguagesAgree`：

- 四种语言（zh、zh-TW、ja、de）键集必须**完全相等**；
- 每条译文非空；
- 每个语言块内键不得重复。

收尾时补齐了此前缺失的 23 个中文键（`Computer`、`File`、`Password`、`Secret`、`WebDAV or S3 sync` 等）与 ja/zh-TW/de 的 `User portal`，现在四语各 **5,489 键**，测试通过。

---

## 4. 术语统一

| 语言 | 处理 | 结果 |
|---|---|---|
| zh | `提供商` 34 处、`提供方` 5 处 → `供应商` | 311 处「供应商」，0 处旧译 |
| zh-TW | 批量替换误将 11 处简体「供应商」带入繁体块 | 已逐条改回「供應商」；0 处简体残留 |
| ja | 保持「プロバイダ / プロバイダー」 | 297 / 93 处，自然用法 |
| de | 保持「Anbieter」 | 409 处 |
| 英文 | 键仍是 `Provider` | 未改 |

同时修正门户词典 4 处「提供商」→「供应商」。渲染扫描（zh/zh-TW/ja/de/en）中 `提供商|提供方` 命中 **0**。

说明：中文界面统一用「供应商」；英文 API/代码仍用 Provider；日、德各用其自然词，不直译英文。

---

## 5. 语言切换验证

| 检查 | 结果 |
|---|---|
| 切换前 | `/portal/?x=1&y=2`，`lang=zh-CN`，`minirelay_lang` Cookie 未设置 |
| 点「English」后 | 同一路径与查询参数 `/portal/?x=1&y=2`，`lang=en`，语言 Cookie 设为 en（仅记键值，不记录会话凭据） |
| 管理员全局设置 | `settings.lang` 仍为 `system`（未写） |
| `?lang=` | 仍是临时覆盖，不落盘 |
| 未提交表单 | 切换会整页重载，已输入的邮箱被清空（`typedFieldAfterSwitch: ""`）——这是当前方案的可预期行为，已在交付说明记录 |
| 页面位置 | 保留页面与查询参数，不回到首页 |

---

## 6. 全站文字对比度（WCAG 2.2 AA）

### 6.1 测量方法

Playwright 遍历每个可见文本节点：

- 前景 = `getComputedStyle().color`（含 alpha 时与其背景合成）；
- 背景 = 最近的非透明 `background-color` 祖先逐层 alpha 合成（无则页面底）；
- 阈值：普通文字 **4.5:1**，大字（≥24px，或 ≥18.66px 且 ≥700）**3:1**；
- 跳过：不可见、`aria-hidden`、`sr-only`、SVG 文本；
- 禁用控件单独记录，按 WCAG「非活动控件」豁免。

覆盖 11 个页面 × 浅/深 = **22 个组合**：管理端 Providers/Routing/Usage/Plugins/Masking/Gateway/Settings，门户首页/控制台/模型广场/模型管理。

### 6.2 修复前后

| 项 | 修复前 | 修复后 |
|---|---|---|
| 失败出现次数 | 123 | **0** |
| 唯一失败 | 118 | **0** |
| 浅色 / 深色 | 118 / 5 | 0 / 0 |
| 失败中的大字 | 0（无大字失败） | — |
| 禁用控件失败 | 1（按豁免不计） | 0 |

失败集中在管理端浅色：`--muted` 3.33–3.66:1、`--faint` 1.77–1.95:1、绿色状态文字 3.62:1。门户此前已达标。

### 6.3 改动（不把所有文字变黑）

**浅色 token（app.css）**

| token | 前 | 后 | 对比度（页面/卡片/胶囊） |
|---|---|---|---|
| `--muted` | `#85858d` | `#63636b` | 5.42 / 5.95 / 5.28 |
| `--faint` | `#b9b9c2` | `#6b6b74` | 4.80 / 5.28 / 4.68 |
| `--green-text`（新） | — | `#067647` | 5.69 / 5.07（green-soft） |
| `--red-text`（新） | — | `#b42318` | 6.57 / 5.61（red-soft） |
| `--amber-text`（新） | — | `#a54d08` | 5.73 / 5.08（amber-soft） |
| `--tk-c`（代码注释色） | `#85858d` | `#63636b` | 5.95 |

**深色 token**

| token | 前 | 后 | 对比度 |
|---|---|---|---|
| `--faint` | `#90909a` | `#9c9ca6` | 5.76（卡片）/ 5.03（胶囊） |
| `--green-text` / `--red-text` / `--amber-text` | — | `#75e0a7` / `#fecdca` / `#f2b544` | 9.68 / 11.03 / 8.55 以上 |

**用法替换**

- `app.css`：26 处 `color: var(--green)`、57 处 `color: var(--red)`、42 处 `color: var(--amber)` 改为对应 `-text` token；填充点/条/边框仍用原 fill token。
- `plugins.css`：5 / 7 / 2 处同上。
- 3 处 `border-color` 随批量替换改用 `-text`（更深的描边，语义仍成立）；`outline-color` 同样保留可视性。
- 占位符统一走 `--faint`，随 token 达标。

层级没有被压平：`--fg #1c1c21`（15.4:1）> `--fg-2 #4b4b53`（7.9:1）> `--muted #63636b`（5.4:1）> `--faint #6b6b74`（4.8:1）；差别靠字号与字重继续表达，颜色只保证可读。

### 6.4 禁用与装饰

| 类别 | 处理 |
|---|---|
| 禁用按钮/输入 | 保留 `opacity .55`；WCAG 豁免，不计入失败；修复前仅 1 处落在禁用态，修复后 0 |
| 装饰线、图标底 | `--line`/`--line-soft`/`--surface-*` 未改；不承担正文信息 |
| 头像/图表色 | 非文字，未按文字阈值调整 |

---

## 7. 布局与横向溢出

44 组（管理端 4 语言 × 390/1440 × 可见页；门户 zh/en × 390）全部 **0 溢出**。

过程中发现并修复一处**本轮引入的回归**：P0 加的管理端「用户门户」文字按钮，在 390px 的德语/日语/英语下把顶栏撑出 115px。现改为窄屏只显示箭头图标（44×44，`aria-label` 随语言），文字在 ≥761px 或鼠标环境显示。

| 视口 | 入口形态 |
|---|---|
| 390 / 触屏 768 | 44×44 图标，`aria-label="用户门户"` |
| 768 鼠标 / 1440 | 图标 + 「用户门户」文字 |

---

## 8. 验收证据

| 项 | 结果 |
|---|---|
| 对比度 | `screenshots/p1-closure/contrast-before.json`（118 唯一失败）、`contrast-after.json`（0） |
| 品牌扫描 | `brand-scan.json`：82 唯一命中，全部第三方/标签；门户 0 |
| 语言/布局 | `closure-check.json`：切换前后、设置未变、44 组布局 0 溢出 |
| 截图 | `admin-providers-1440-{light,dark}.png`、`admin-usage-1440-*.png`、`admin-settings-1440-*.png`、`admin-providers-390-light.png` |
| Oil UI lint | 登录页 desktop+mobile：通过，无控制台错误/横向溢出/加载失败 |
| 控制台错误 | 门户仅匿名页 `/api/auth/me` 401（既有探测行为）；管理端 0 新增 |
| Go 测试 | gui 168.534s、provider 79.810s、library 98.356s，全部 ok |

---

## 9. 未完成与风险

1. **显示层而非源键迁移**：`app.js` 的 280 个英文键与 i18n.js 的译文源文件仍写 `magpie`，显示时统一。若未来有代码**绕过 `t()`** 直接读 `I18N` 或拼接译文，品牌会漏出。目前已核查：`I18N` 的直接读取只有 `t()`、`tNotice()` 与插件词典合并（`kernel.js` 合并后仍经 `t()` 显示）。
2. **Go 侧用户可见诊断**仍有 3,500+ 字符串含 magpie（认证/代理/Agent 集成错误等）。这些多为技术诊断，且部分字面引用命令 `magpie`；本任务未逐条处理。若它们通过 API 进入页面，会绕过显示层（渲染扫描中未出现，但覆盖不到全部状态）。
3. **第三方内容**：插件市场名称/README/`magpie-plugin` 标签保留旧词，属预期。
4. **层级变化**：`--muted` 与 `--faint` 都压到 AA 后视觉差异变小；如需更强层级，只能靠字号/字重，不能再调浅颜色。
5. **禁用态**：仍以 opacity 表达，未改颜色；WCAG 豁免，但低视力用户可读性有限。
6. **zh-TW**：原有译文混入少量简体（不止术语）；本轮只修了本次误入的 11 处，全面繁体清理不在范围。
7. **德/日**：`Provider API` 等专有用法保留英文；`プロバイダ` 与 `プロバイダー` 混用为原文风格，未强行统一。

---

## 10. P2 前置条件

1. 确认「显示层统一」可作为品牌最终方案；若要求源键也改名，需要单列一次 4 语言键迁移（约 6,600 条）并有完整回退测试。
2. 确认中文统一词「供应商」。
3. 确认管理端 390px 的门户入口只留图标。
4. 对比度 token 新值进入 DESIGN.md 的 token 表（当前 DESIGN 仍记旧值），建议 P2 首项同步文档。

---

## 11. 附件

| 路径 | 内容 |
|---|---|
| `docs/product-v2/screenshots/p1-closure/` | 7 张截图 + 4 份 JSON 实测 |
| `internal/gui/i18n_parity_test.go` | 四语键集一致、非空、无重复 |
| `internal/gui/assets/i18n.js` | 品牌显示层、术语、23 条补译、`data-ta` 支持 |
| `internal/gui/assets/app.css` | 对比度 token 与状态文字 token |
| `/tmp/close-before.tar` | 收尾开工快照（临时，未入库） |
| `/tmp/magpie-p1c-bin` | 收尾编译产物（临时，未入库） |