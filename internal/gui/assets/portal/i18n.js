// The user portal's own words. The English string is the key, as i18n.js
// does it for the management page; a missing key falls back to the English,
// so a half-translated language still works. {name} slots are filled by
// t(str, vars). Only the portal's copy lives here: the management page's
// 2.6 MB dictionary stays out of this page.
window.PORTAL_I18N = {
  zh: {
    "AI model relay platform": "AI 模型中转平台",
    "API Keys are separate from sign-in accounts: sign in to the web with an account and password, call models with an API Key; disabling or deleting a key stops its calls at once. History stays after a key is deleted.":
      "API Key 与登录账号分离：网页登录使用账号密码，调用模型使用 API Key；停用或删除 Key 会立即停止对应的调用。历史日志在 Key 删除后仍然保留。",
    "API Keys are separate from web accounts and can be disabled on their own; the Gateway records requests, models, tokens and latency.":
      "API Key 与网页账号分离，可独立停用；请求、模型、Token 与耗时由 Gateway 记录。",
    "Account ID": "账号 ID",
    "Account role": "账号角色",
    "Account": "账号",
    "Actions": "操作",
    "API Keys": "API Key 数量",
    "Key": "密钥",
    "Models": "模型",
    "Model grants": "模型授权",
    "Overview": "概览",
    "Portal navigation": "门户导航",
    "Token / requests · last 30 days": "近 30 天 Token / 请求",
    "{tokens} tokens · {requests} requests": "{tokens} Token · {requests} 请求",
    "Requests made with this account's API Keys will appear here.":
      "当前账号的 API Key 产生真实调用后，会在这里按天显示。",
    "Model usage appears after this account's API Keys make calls.":
      "当前账号的 API Key 产生调用后，会在这里按模型汇总。",
    "Add and enable providers and models in the original admin page first; they appear here automatically.":
      "先在原管理后台添加并启用供应商与模型；它们会自动出现在这里。",
    "Admin console": "管理后台",
    "Administrator": "管理员",
    "Administrator · Model catalogue": "管理员 · 模型目录",
    "All": "全部",
    "All models": "全部模型",
    "All providers": "全部供应商",
    "Availability": "可用状态",
    "All statuses": "全部状态",
    "Allow new user registration": "允许新用户注册",
    "Already have an account?": "已有账号？",
    "At least 8 characters": "至少 8 个字符",
    "Back to the console": "返回控制台",
    "By total tokens": "按总 Token",
    "Call example": "调用示例",
    "Cancel": "取消",
    "Cancelled": "已取消",
    "Choose providers and accounts by quota, order, weight or speed; a failure moves to the next available target.":
      "按额度、顺序、权重或速度选择供应商与账号；一次失败自动换下一个可用目标。",
    "Close": "关闭",
    "Closed": "未开放",
    "On": "开启",
    "Off": "关闭",
    "Routing group": "路由组",
    "Routing groups": "路由组",
    "Plaza shows routing groups only": "模型广场只显示路由组",
    "On: the model plaza and new API-key grants offer open routing groups alone. Provider models stay available to agents and administrators. Off: every open model is offered.":
      "开启后，模型广场和新 API Key 的授权只提供已开放的路由组；供应商模型仍可供 Agent 和管理员使用。关闭后，所有已开放的模型都会提供。",
    "Add and enable providers and models in the original admin page first; they appear here automatically. Routing groups appear too once made.":
      "先在原管理页添加并启用供应商与模型，它们会自动出现在这里；创建路由组后，路由组也会出现在这里。",
    "Confirm password": "确认密码",
    "Console": "控制台",
    "Copy": "复制",
    "Create": "创建",
    "Create API Key": "创建 API Key",
    "Create an API Key": "创建 API Key",
    "Create an account": "创建账号",
    "Daily token usage trend": "每日 Token 用量趋势",
    "Daily usage": "每日用量",
    "Defaults to the model name": "默认使用模型名称",
    "Description": "描述",
    "Display name": "显示名称",
    "Edit display info": "编辑展示信息",
    "Email": "邮箱",
    "Email or username": "邮箱或用户名",
    "Enter the password again": "再次输入密码",
    "Every request through the Gateway; only this account's records.":
      "每一次经 Gateway 的请求，只显示当前账号的记录。",
    "Failed": "失败",
    "Filter by open state": "按开放状态筛选",
    "Filter by protocol": "按协议筛选",
    "Find model IDs in the model plaza; models the platform has not opened do not appear in the call list.":
      "模型 ID 请从模型广场查看；平台尚未开放的模型不会出现在调用列表中。",
    "Go to the console": "去控制台",
    "Home": "首页",
    "Input tokens": "输入 Token",
    "It is not shown again after closing or reloading; copy it now.":
      "关闭或刷新后不再显示，请现在复制保存。",
    "Keys and usage": "密钥与用量",
    "Keys used to call models, belonging to this account. The full key is shown only once, when it is created.":
      "调用模型使用的密钥，只属于当前账号。完整密钥只在创建时显示一次。",
    "Keys, personal data and custom words are recognised and replaced before sending, then restored in the reply, so less sensitive data leaves the platform.":
      "发送前识别并替换密钥、个人信息和自定义词，回复时再还原，减少敏感数据外流。",
    "Language": "语言",
    "Last 30 days": "近 30 天",
    "Last 7 days": "近 7 天",
    "Main navigation": "主导航",
    "MiniRelay home": "MiniRelay 首页",
    "MiniRelay · AI model relay platform": "MiniRelay · AI 模型中转平台",
    "Model": "模型",
    "Model access": "模型开放管理",
    "Model access is on the platform admin page: open or close models and set the names and descriptions users see. Providers, routing, the Gateway, platform-wide usage, masking and settings remain in the original admin page.":
      "模型开放管理在平台管理页：开放或关闭模型、设置用户看到的名称与说明。供应商、路由、Gateway、平台整体用量、脱敏与设置仍在原管理后台。",
    "Model plaza": "模型广场",
    "Model usage": "模型用量",
    "Models available": "可用模型",
    "Models the platform currently opens, from the Gateway's real catalogue. Closed models are not listed and cannot be called with a user key.":
      "平台当前开放的模型，来自 Gateway 的真实目录。关闭的模型不在这里，也不能被用户 Key 调用。",
    "Multi-protocol support": "多协议兼容",
    "Name": "名称",
    "Name users see": "用户看到的名称",
    "Names and descriptions change what users see, not model IDs. A model that is open but temporarily unavailable is still listed and marked unavailable; the Gateway decides whether a call succeeds.":
      "名称与说明只改变用户端展示，不改变模型 ID。已开放但供应商暂时不可用的模型仍会显示，并标注为不可用；其调用是否成功由 Gateway 按实际路由决定。",
    "New user registration": "新用户注册",
    "User registration": "用户注册",
    "Next page": "下一页",
    "No API Key yet. After creating one, paste it into your client's API Key field to call models.":
      "还没有 API Key。创建一个后，把它填进客户端的 API Key 即可调用模型。",
    "No account yet?": "还没有账号？",
    "No model calls in this time range.": "这个时间范围内还没有模型调用。",
    "No models open yet": "还没有开放的模型",
    "No models to manage yet": "平台还没有可管理的模型",
    "No requests under these filters.": "这个条件下没有请求。",
    "One API for OpenAI, Anthropic and Gemini protocols; MiniRelay handles provider routing, fallback, keys and usage, keeping the integration inside the platform.":
      "用一套 API 接入 OpenAI、Anthropic、Gemini 等协议；由 MiniRelay 负责供应商路由、故障回退、密钥与用量，把复杂的接入留在平台内部。",
    "One entry,": "一个入口，",
    "Opened": "已开放",
    "Open in the model plaza and allow user keys to call it": "在模型广场开放，允许用户 Key 调用",
    "Open models": "开放模型",
    "Open the original admin page": "打开原管理后台",
    "OpenAI Chat Completions, Responses, Anthropic Messages and Gemini generateContent, forwarded through one entry as the client sends them.":
      "OpenAI Chat Completions、Responses、Anthropic Messages 与 Gemini generateContent，同一入口按客户端原样转发。",
    "Optional": "可选",
    "Optional, up to 500 characters": "可选，最多 500 字",
    "Output tokens": "输出 Token",
    "Password": "密码",
    "Platform capabilities": "平台能力",
    "Platform models": "平台模型",
    "Point your client at the platform and call models with the key.":
      "把客户端的地址指向平台，使用 Key 调用模型。",
    "Previous page": "上一页",
    "Privacy and masking": "隐私与脱敏",
    "Protocol": "协议",
    "Provider": "供应商",
    "Quick start": "快速接入",
    "Reading the model catalogue…": "正在读取模型目录…",
    "Reading…": "读取中…",
    "Real Gateway calls made by this account's API Keys, for the selected time range.":
      "当前账号的 API Key 在 Gateway 上的真实调用，按所选时间范围统计。",
    "Refresh": "刷新",
    "Register": "注册",
    "Registered": "注册时间",
    "Replace the Base URL": "替换 Base URL",
    "Request log": "使用日志",
    "Routing and fallback": "路由与故障回退",
    "Save": "保存",
    "Save model access": "保存模型授权",
    "Search model name, ID or provider": "搜索模型名称、ID 或供应商",
    "Search models": "搜索模型",
    "Select all": "全选",
    "Select none": "全不选",
    "Sign in": "登录",
    "Sign in to the console": "登录控制台",
    "Skip to main content": "跳到主要内容",
    "Status": "状态",
    "Success": "成功",
    "Team or your name": "团队或你的名字",
    "The console issues, disables and deletes keys per account (wired in Phase 3).":
      "控制台将按账号发放、停用和删除 Key（Phase 3 接入）。",
    "This decides what the model plaza shows, which models user keys may request, and whether the Gateway passes them. Closing takes effect immediately.":
      "这里决定用户模型广场展示什么、用户 Key 能申请哪些模型，以及 Gateway 是否放行。关闭立即生效。",
    "This is the full key, shown only once": "这是完整密钥，只显示这一次",
    "This key may call only the checked models; a model the platform closed cannot be called even if it was checked.":
      "这个 Key 只能调用勾选的模型；平台关闭的模型即使勾选过也不能调用。",
    "This page needs JavaScript to sign in and show the console.":
      "这个页面需要 JavaScript 才能登录和显示控制台。",
    "Three steps to call a model": "三步开始调用",
    "Today": "今天",
    "Total requests": "总请求",
    "Total tokens": "总 Token",
    "Usage overview": "用量概览",
    "Use your platform account to open the console": "使用平台账号进入控制台",
    "Use your email or username to open the console": "使用邮箱或用户名进入控制台",
    "Use your platform account to open your console.": "使用平台账号进入个人控制台。",
    "Regular user": "普通用户",
    "Users sign in to the web with an account and call models with an API Key; the two are managed separately.":
      "普通用户使用账号登录网页，使用 API Key 调用模型，两者分开管理。",
    "Welcome back": "欢迎回来",
    "What this model is good at, and what limits it has": "这个模型适合做什么、有什么限制",
    "When the administrator opens models, their names, providers, protocol support, availability and call examples appear here.":
      "管理员开放模型后，这里会显示模型名称、供应商、协议能力、可用状态与调用示例。",
    "You enter the console after registering; API Keys and accounts are managed separately":
      "注册后进入控制台；API Key 与账号分开管理",
    "e.g. My laptop": "例如：我的笔记本",
    "every AI model": "接入所有 AI 模型",

    "Network error. Try again.": "网络错误，请重试。",
    "Too many sign-in attempts. Try again later.": "登录尝试过于频繁，请稍后再试。",
    "The email or password is not correct.": "邮箱或密码不正确。",
    "The email, username or password is not correct.": "邮箱、用户名或密码不正确。",
    "This account is disabled. Ask the administrator to enable it.": "该账号已被停用，请联系管理员。",
    "Registration is closed. Ask the administrator.": "注册暂未开放，请联系管理员。",
    "Registration is closed: the platform has no administrator yet.": "注册暂未开放：平台还没有管理员账号。",
    "You do not have permission to do that.": "没有权限执行这个操作。",
    "That email is already registered; sign in instead.": "该邮箱已被注册，请直接登录。",
    "That conflicts with existing data. Refresh and try again.": "操作与现有数据冲突，请刷新后重试。",
    "Enter a valid email address.": "请输入有效的邮箱地址。",
    "The password needs at least 8 characters.": "密码至少 8 个字符。",
    "The password is 1024 characters at most.": "密码最多 1024 个字符。",
    "The name is 80 characters at most.": "名称最多 80 个字符。",
    "Invalid request: {why}": "请求无效：{why}",
    "Invalid request. Check what you filled in.": "请求无效，请检查填写内容。",
    "That did not work. Try again.": "操作失败，请重试。",
    "Sign out": "退出登录",
    "Open the console": "进入控制台",
    "Browse models": "浏览模型广场",
    "Enter your email and password.": "请填写邮箱和密码。",
    "Enter your email or username and password.": "请填写邮箱或用户名和密码。",
    "you@example.com or username": "you@example.com 或用户名",
    "Signing in…": "登录中…",
    "The passwords do not match.": "两次输入的密码不一致。",
    "Creating…": "创建中…",
    "Open · Any visitor can register": "已开放 · 任何访客都可以自行注册",
    "Closed · Visitors cannot register themselves; administrators can still create accounts":
      "已关闭 · 访客不能自行注册，管理员仍可创建账号",
    "Needs administrator rights.": "需要管理员权限。",
    "Could not save the registration setting. Try again later.": "注册设置保存失败，请稍后重试。",
    "Could not read": "读取失败",
    "Could not load the registration setting. Refresh and try again.": "注册设置加载失败，请刷新重试。",
    "Last used {when}": "最后使用 {when}",
    "Never used": "从未使用",
    "No models granted": "未授权任何模型",
    "{n} models · {names}": "{n} 个模型 · {names}",
    "{n} models · {names} and more": "{n} 个模型 · {names} 等",
    "The platform has closed this model; saving the grant removes it from the key.":
      "平台已关闭这个模型；保存授权后会从 Key 中移除。",
    "Closed by the platform": "平台已关闭",
    "The platform has no open models yet.": "平台目前没有开放模型。",
    "Reading the platform's open models…": "正在读取平台开放模型…",
    "The platform has no open models yet; the administrator opens them before keys can be created.":
      "平台还没有开放模型，管理员开放后才能创建 Key。",
    "All chosen by default; this key may call only the checked models.":
      "默认全选；这个 Key 只能调用勾选的模型。",
    "Untitled": "未命名",
    "Created {when}": "创建于 {when}",
    "Old key: its grant was fixed to the models open when it was first enabled":
      "旧 Key：授权已按首次启用时开放的模型固定",
    "Disabled": "已禁用",
    "Enabled": "已启用",
    "Enable": "启用",
    "Disable": "禁用",
    "Delete": "删除",
    "Delete the API Key “{name}”? Clients using it stop working at once.":
      "删除 API Key “{name}”？使用它的客户端会立即停止工作。",
    "Model access · {name}": "模型授权 · {name}",
    "This is an old key: its model grant was fixed when it was first enabled. Choose again; closed models are removed.":
      "这是旧 Key：模型授权已按首次启用时开放的模型固定。现在可以重新选择；已关闭的模型会被移除。",
    "Takes effect on save. Closed models cannot stay granted and are removed.":
      "保存后立即生效。已关闭的模型无法继续授权，保存后会从 Key 中移除。",
    "The platform has no open models yet; the administrator opens them before an API Key can be created.":
      "平台还没有开放模型，管理员开放后才能创建 API Key。",
    "Give this key a name.": "请给这个 Key 起一个名字。",
    "Choose at least one model for this key.": "请至少选择一个模型授权给这个 Key。",
    "Choose at least one open model; otherwise this key cannot call any model.":
      "请至少选择一个开放模型；否则这个 Key 不能调用任何模型。",
    "Saving…": "保存中…",
    "Copied": "已复制",
    "Copy failed": "复制失败",
    "Could not load the model catalogue; keys cannot be created until it does. Refresh later.":
      "模型目录加载失败，暂时不能创建 API Key；请稍后刷新。",
    "The platform has no open models yet": "平台还没有开放模型",
    "{n} keys": "{n} 个密钥",
    "{n} models": "{n} 个模型",
    "Loading…": "加载中…",
    "No requests in this time range.": "这个时间范围内还没有请求。",
    "Daily token usage": "每日 Token 用量",
    "{tokens} tokens · {calls} requests": "{tokens} tokens · {calls} 请求",
    "{n} calls": "{n} 次调用",
    "Could not load usage. Try again later.": "用量加载失败，请稍后重试。",
    "Time": "时间",
    "Duration": "耗时",
    "Total {n}": "共 {n} 条",
    "Page {page} / {pages}": "第 {page} / {pages} 页",
    "Could not load the log. Try again later.": "日志加载失败，请稍后重试。",
    "Not declared": "未标注",
    "No matching models": "没有匹配的模型",
    "Try another keyword, provider or protocol filter.": "换个关键词、供应商或协议筛选条件再试。",
    "Try another keyword, provider, protocol or availability filter.":
      "换个关键词、供应商、协议或可用状态筛选条件再试。",
    "{shown} / {total}": "{shown} / {total}",
    "{n} open": "{n} 个已开放",
    "Only models open now are listed here. When the platform closes a model it disappears from the plaza at once, and existing keys cannot call it either.":
      "这里只列出当前开放的模型。平台关闭某个模型后，它会立即从广场消失，已有的 Key 也不能再调用它。",
    "View {name}'s details and call example": "查看 {name} 的详情与调用示例",
    "Unknown provider": "未知供应商",
    "Context {value}": "上下文 {value}",
    "Service healthy": "服务正常",
    "Provider temporarily unavailable": "供应商暂不可用",
    "View details →": "查看详情 →",
    "Context": "上下文",
    "Copy model ID": "复制模型 ID",
    "Model information": "模型信息",
    "Normal": "正常",
    "Providers": "供应商",
    "Temporarily unavailable": "暂不可用",
    "Available": "可用",
    "Model ID": "模型 ID",
    "Input images": "输入图片",
    "Supported": "支持",
    "Open state": "开放状态",
    "Gateway: {url} · Use an API Key created in the console. Model IDs are as shown on this page.":
      "Gateway：{url} · 使用控制台创建的 API Key。模型 ID 以本页显示为准。",
    "Could not load the model catalogue. Try again later.": "模型目录加载失败，请稍后重试。",
    "hello": "你好",
    "Your sign-in expired. Sign in again.": "登录已失效，请重新登录。",
    "The Gateway does not serve this model now; refresh the catalogue.": "Gateway 现在不提供这个模型，请刷新目录。",
    "The name users see is 80 characters at most.": "用户看到的名称最多 80 个字符。",
    "The description is 500 characters at most.": "说明最多 500 个字符。",
    "Save failed: {why}": "保存失败：{why}",
    "Save failed. Try again later.": "保存失败，请稍后重试。",
    "This page is for administrators.": "这个页面只对管理员开放。",
    "Try another keyword or open state.": "换个关键词或开放状态再试。",
    "Service": "服务",
    "Unavailable": "不可用",
    "Close {name}": "关闭 {name}",
    "Open {name}": "开放 {name}",
    "Edit": "编辑",
    "Usage and requests for this account": "当前账号的用量与请求记录",
    "Usage statistics": "用量统计",
    "Request details": "请求明细",
    "Requests are recorded by the Gateway; only this account's records are shown.":
      "请求由 Gateway 记录，只显示当前账号的记录。",
    "Yesterday": "昨天",
    "Auto refresh": "自动刷新",
    "API keys": "API 密钥",
    "Create key": "创建密钥",
    "Keys used to call models, belonging to this account.": "调用模型使用的密钥，只属于当前账号。",
    "Profile": "个人资料",
    "Account details and token activity": "账号信息与 Token 活跃度",
    "Joined {date}": "加入于 {date}",
    "Tokens · last year": "近一年 Token",
    "Peak day": "单日峰值",
    "Current streak": "当前连续",
    "Longest streak": "最长连续",
    "Token activity": "Token 活跃度",
    "Daily view": "每日",
    "Weekly view": "每周",
    "Cumulative view": "累计",
    "Activity view": "活跃度视图",
    "Token activity heatmap": "Token 活跃度热力图",
    "No token activity yet": "暂无用量",
    "Less → More": "少 → 多",
    "Could not load token activity. Try again later.": "Token 活跃度加载失败，请稍后重试。",
    "Login account": "登录账号",
    "Account permission": "账号权限",
    "Last sign-in": "最近登录",
    "Manage keys": "管理密钥",
    "Copy account ID": "复制账号 ID",
  },
};

// The language this page speaks: ?lang= wins (a temporary override, never
// saved), then the cookie the footer's switch sets, then the server's saved
// setting written on <html lang>. One dictionary, one rule, both surfaces.
(function () {
  const asked = new URLSearchParams(location.search).get("lang");
  const saved = (document.cookie.match(/(?:^|;\s*)minirelay_lang=(zh|en)\b/) || [])[1];
  const lang = asked === "zh" || asked === "en" ? asked : saved || (document.documentElement.lang === "en" ? "en" : "zh");
  const tags = { zh: "zh-CN", en: "en" };
  window.portalLang = lang;
  window.portalLangTag = tags[lang] || "zh-CN";
  window.t = function (s, vars) {
    let out = (window.PORTAL_I18N[lang] && window.PORTAL_I18N[lang][s]) ?? s;
    if (vars) for (const k in vars) out = out.split("{" + k + "}").join(vars[k]);
    return out;
  };
  // translateStatic's rules, for this page: the English is remembered on the
  // element, so a second pass (a redraw) translates from the same source.
  window.applyPortalI18n = function (root = document) {
    for (const e of root.querySelectorAll("[data-t]")) {
      if (e.dataset.en === undefined) e.dataset.en = e.textContent;
      e.textContent = window.t(e.dataset.en);
    }
    for (const e of root.querySelectorAll("[data-ta]")) {
      if (e.dataset.enAria === undefined) e.dataset.enAria = e.dataset.ta;
      e.setAttribute("aria-label", window.t(e.dataset.enAria));
    }
    for (const e of root.querySelectorAll("[data-tp]")) {
      if (e.dataset.enPlaceholder === undefined) e.dataset.enPlaceholder = e.dataset.tp;
      e.placeholder = window.t(e.dataset.enPlaceholder);
    }
    document.documentElement.lang = window.portalLangTag;
    for (const b of document.querySelectorAll("[data-lang]")) {
      b.setAttribute("aria-pressed", b.dataset.lang === lang ? "true" : "false");
    }
  };
})();