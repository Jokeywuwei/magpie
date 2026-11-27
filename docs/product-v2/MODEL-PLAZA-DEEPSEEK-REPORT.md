# MODEL-PLAZA-DEEPSEEK — 模型广场 DeepSeek 风格改造

日期：2026-11-25
分支：`remove-partners`
HEAD：`e41adc1513bb1e9fe953e7dd8c35dc40e23bf60e`
版本标记：`v0.1.4574-model-plaza`
范围：用户门户 `/portal/models`（模型广场）
未 commit / push / reset / stash。

---

## 1. 改了什么

### 1.1 布局

- 桌面 ≥1100：左侧 220px Provider 列表 + 右侧模型区，间距 24px；provider 面板不伪造图标，用首字母方块，全部供应商用 4 格网格图标。
- Provider 行：20px 图标方块 + 名称 13px + 模型数 12px muted + 6px 状态点；行高 40px、圆角 8px、padding 0 10px；当前行 `--accent` 底 + 1px `--border`，文字 `--foreground`，状态点按真实可用状态取 `--success`/`--danger`。
- 组头：Provider 名称 20/600 + “N 个模型” 13px muted；右侧“全部/可用/不可用”分段 + 搜索框；协议筛选放在组头下方一行，并加“协议”标签避免与可用状态分段混淆。
- 卡片网格：`repeat(auto-fit, minmax(340px, 1fr))`，最多 3 列，gap 20px；2 个模型时两张卡占两列（见 §5 偏差说明）。
- ≤760：Provider 列表改为横向滚动胶囊，行高 44px、999px 圆角；组头纵向堆叠，可用状态分段和搜索各自成行；模型卡单列；协议筛选保留。
- 全部入口仍是现有搜索、Provider 过滤、协议过滤、可用状态过滤、详情弹窗、复制模型 ID、语言切换、主题切换、底部 Dock。

### 1.2 模型卡

- `--card` 底、1px `--border`、12px 圆角、20px 内距、无阴影；hover 只把边框提亮到 `--muted-foreground`。
- 顶部：32px Provider 首字母方块（`--accent` 底、8px 圆角）+ 模型名 15/600 + 右侧 16px 复制图标（复制真实模型 ID，aria-label 为“复制模型 ID”）。
- 第二行：6px 状态点 + “正常”/“暂不可用” 12px muted。
- 信息块：API 没有价格/倍率字段，因此使用“模型信息”分支：
  - 标题“模型信息”，右侧 1px 描边协议徽标（如 Chat）；
  - 四行真实字段：上下文、输入图片、协议、开放状态；
  - 缺失字段显示“—”，不编造价格、倍率、折扣、延迟或可用率。
- 可用性条：18 个 8×8px、3px 圆角、gap 4px 的小方块；API 只有 `available` 布尔值，因此可用时 18 格全 `--success`，不可用时 18 格全 `--muted`；没有随机颜色或百分比。
- 底部：左侧“可用”/“暂不可用”12px muted，右侧“查看详情 →”品牌蓝按钮；整卡鼠标/触控可点，详情按钮和复制按钮是键盘/无障碍入口。
- 卡片不放长描述；描述只在现有详情弹窗中显示。

### 1.3 文件

| 文件 | 改动 |
|---|---|
| `internal/gui/assets/portal/index.html` | 模型广场改为 Provider 列表 + 组头 + 可用状态分段 + 协议行 + 新卡片网格 |
| `internal/gui/assets/portal/models.js` | Provider 列表/计数、可用状态过滤、新卡片 DOM、复制模型 ID、详情弹窗接线 |
| `internal/gui/assets/portal/portal.css` | `.plaza-*` / `.provider-*` / 新模型卡样式与移动端规则 |
| `internal/gui/assets/portal/i18n.js` | Providers、{n} models、Copy model ID、Model information、Normal、Temporarily unavailable、Availability、空态文案 |
| `docs/product-v2/DARK-UI-SYSTEM.md` | §7.3 更新为 DeepSeek 风格卡片规范；v1.2 修订记录 |
| `docs/product-v2/screenshots/model-plaza-deepseek/` | 验收截图 |

未改：其他门户页面、管理后台、Dock、API/权限/数据结构、`app.css`。

---

## 2. 真实数据映射

`GET /api/user/models` 实际返回字段：

```json
{
  "id": "uialign/model-alpha",
  "name": "model-alpha",
  "description": "UI alignment sample model for layout checks.",
  "provider": "UI Test Provider",
  "providerId": "uialign",
  "protocols": ["chat"],
  "available": true,
  "open": true
}
```

- `context`、`images` 在本次样本中未出现（omitempty），因此卡片对应行显示“—”。
- `protocols` → 协议徽标和“协议”行；`available` → 顶部状态点、可用性条、底部状态；`open` → “开放状态：已开放”。
- Provider 列表的模型数来自当前返回的开放模型计数；状态点来自该 Provider 是否存在可用模型。
- 组头“N 个模型”是当前搜索/Provider/协议/可用状态过滤后的真实数量。
- 没有价格、倍率、折扣、延迟、可用率百分比字段；页面不显示这些数据。

---

## 3. 验收数据

### 3.1 1440（深色）

| 项 | 实测 |
|---|---|
| 布局 | Provider 面板 220px，主区 1068px，间距 24px |
| 卡片列 | `524px 524px 0px`（2 张卡占 2 列；auto-fit 无空轨道） |
| 卡片 | 524×378px；背景 `oklch(0.195 0 0)`；边框 `oklch(0.29 0 0)`；12px 圆角；`box-shadow: none` |
| 像素 | 页面背景 `rgb(10,10,10)`、卡片 `rgb(21,21,21)`、边框 `rgb(43,43,43)` |
| Provider 行 | 40px 高；当前行 `--accent` + 1px 边框 + 成功点 |
| 组头 | “全部模型”20/600；“2 个模型”13px muted |
| 卡片信息块 | 173px 高；四行：上下文—、输入图片—、协议 Chat、开放状态已开放 |
| 可用性条 | 18 个方块，18 个 `--success` |
| 横向溢出 | 1440 = 1440 |
| 控制台错误 | 0 |

### 3.2 390 / 320（深色）

| 项 | 实测 |
|---|---|
| Provider 列表 | `display:flex` + `overflow-x:auto`；行高 44px、999px 圆角；320 下可横向滚动 |
| 组头 | 纵向堆叠；搜索框全宽 44px 高、16px 字号 |
| 模型卡 | 单列 358px 宽；卡片 358×402px |
| 可点目标 | 0 个 <44px；复制按钮 44×44，详情按钮 66×44 |
| 横向溢出 | 390 = 390；320 = 320 |
| Dock | 底部固定；滚动到底 Footer 与 Dock 间距 22px |
| 控制台错误 | 0 |

### 3.3 对比度 / 焦点

| 组合 | 对比度 |
|---|---|
| foreground / card | 15.74 |
| muted-foreground / card | 7.33 |
| brand / card | 9.33 |
| foreground / background | 17.06 |
| muted-foreground / background | 7.94 |
| success / card | 9.42 |

- 焦点环：复制按钮实测 `outline: oklch(0.78 .13 232) solid 2px; outline-offset: 2px`。
- Esc 关闭详情弹窗后焦点回到“查看详情”按钮。

### 3.4 功能回归

| 功能 | 结果 |
|---|---|
| Provider 过滤 | 点击 UI Test Provider：组头变该 Provider，2 张卡；返回全部正常 |
| 搜索 | 输入 gamma：1 张卡；清空恢复 2 张 |
| 协议过滤 | Anthropic：0 张卡 + “没有匹配的模型”空态；全部恢复 |
| 可用状态过滤 | 不可用：0 张卡 + 空态；可用：2 张卡；全部恢复 |
| 复制模型 ID | 剪贴板得到 `uialign/model-alpha`，按钮 aria-label 变“已复制” |
| 详情 | 打开、标题/ID 正确；Esc 关闭并归还焦点 |
| 语言切换 | 中文/English 全部标签、组头、卡片、空态正确；390 无溢出 |
| 主题切换 | 隔离 HOME 切 light 后卡片白底、浅色边框、无阴影；切回 dark 正常 |

### 3.5 构建与测试

```sh
GOMAXPROCS=2 CGO_ENABLED=0 go build -tags nogui -trimpath \
  -ldflags "-s -w -X main.version=v0.1.4574-model-plaza" -o /tmp/mr-dark .
# BUILD OK

GOMAXPROCS=2 CGO_ENABLED=0 go test -tags nogui ./internal/gui \
  -run 'TestPortal|TestAuth|TestUserUsage|TestCrossOriginLoginRefused|TestAdminRoutesRequireAnAdmin|TestWebAuthBypassesThePageKey' -count=1
# ok

GOMAXPROCS=2 CGO_ENABLED=0 go test -tags nogui ./internal/identity \
  -run 'TestLoginByEmailOrUsername' -count=1
# ok
```

---

## 4. 截图

目录：`docs/product-v2/screenshots/model-plaza-deepseek/`

| 截图 | 说明 |
|---|---|
| `plaza-1440-zh-dark.png` | 1440 模型广场，中文，深色 |
| `plaza-1440-en-dark.png` | 1440 模型广场，英文，深色 |
| `plaza-390-zh-dark.png` | 390 模型广场，中文，深色，首屏 |
| `plaza-390-en-dark.png` | 390 模型广场，英文，深色，首屏 |
| `plaza-390-zh-dark-cards.png` | 390 中文，滚动到完整模型卡 |
| `plaza-390-zh-dark-bottom.png` | 390 中文，滚动到底（Dock 与 Footer 间距） |
| `plaza-1440-zh-filter-empty.png` | 可用状态“不可用”筛选空态 |
| `plaza-1440-zh-light.png` | 浅色主题回退（额外回归证据） |

截图中的模型来自隔离 HOME 的真实 `model-access.json`：`uialign/model-alpha`、`uialign/model-gamma` 均为平台已有模型，通过管理员 API 打开 `model-alpha` 以获得两张卡；没有编造模型或业务数据。

---

## 5. 未做项 / 偏差

1. API 没有价格/倍率/折扣字段，因此卡片使用“模型信息”分支；没有显示任何价格。
2. API 只有 `available` 布尔值，没有真实可用率；可用性条全绿或全 muted，不生成随机比例或百分比。
3. 本次隔离目录只有一个 Provider（UI Test Provider）和两个开放模型；Provider 列表只展示真实数据，未添加虚构 Provider。
4. 本次样本没有不可用模型，因此“不可用”卡片外观未用真实数据截图；不可用筛选空态已验收。卡片代码按 `available=false` 显示“暂不可用”、红色状态点和全 muted 可用性条。
5. 卡片网格使用 `auto-fit` 而不是字面的 `auto-fill`：`auto-fill` 在 2 张卡时会留下第三列空轨道，与“只有 2 个模型时就是 2 列、不要大块空白”的验收冲突；`minmax(340px,1fr)`、最大 3 列、gap 20px 保持不变。若要严格改回 `auto-fill`，只需改 `portal.css` 的 `.model-grid` 一行。
6. 上下文/输入图片在 API 样本中缺失，卡片显示“—”；没有为了填满卡片而编造数值。
7. “查看详情”沿用现有 i18n 的 `查看详情 →`，未改成字面 `>`。
8. 未 commit / push。
