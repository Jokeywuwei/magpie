// The user portal's page script. It signs in with /api/auth, draws the
// account the server says is signed in, and keeps every page honest about
// what is not connected yet: no keys, usage or models are invented here.
(() => {
  "use strict";

  const page = document.body.dataset.page || "home";
  const $ = (sel, root = document) => root.querySelector(sel);
  const el = (tag, cls, text) => {
    const n = document.createElement(tag);
    if (cls) n.className = cls;
    if (text != null) n.textContent = text;
    return n;
  };

  async function api(path, { method = "GET", body } = {}) {
    let res;
    try {
      res = await fetch(path, {
        method,
        credentials: "same-origin",
        cache: "no-store",
        headers: body === undefined ? {} : { "Content-Type": "application/json" },
        body: body === undefined ? undefined : JSON.stringify(body),
      });
    } catch {
      return { ok: false, status: 0, data: null };
    }
    let data = null;
    try { data = await res.json(); } catch { /* no body */ }
    return { ok: res.ok, status: res.status, data };
  }

  async function me() {
    const r = await api("/api/auth/me");
    return r.ok && r.data && r.data.user ? r.data.user : null;
  }

  function showError(node, why) {
    if (!node) return;
    node.textContent = why;
    node.hidden = false;
  }

  function clearError(node) {
    if (!node) return;
    node.hidden = true;
    node.textContent = "";
  }

  function roleName(role) {
    return role === "admin" ? t("Administrator") : t("Regular user");
  }

  // The Dock is the portal's only global navigation. Its items live here and
  // nowhere else: order, visibility, labels and icons are all read from this
  // array, and the empty #dock in index.html is filled by renderDock().
  const DOCK_ITEMS = [
    { id: "overview", href: "/portal/console", label: "Overview", icon: "chart", order: 10 },
    { id: "models", href: "/portal/models", label: "Model plaza", icon: "grid", order: 20 },
    { id: "keys", href: "/portal/keys", label: "API keys", icon: "key", order: 30 },
    { id: "admin", href: "/", label: "Admin console", icon: "admin", order: 40, adminOnly: true },
    { id: "profile", href: "/portal/profile", label: "Profile", icon: "avatar", order: 50, avatar: true },
  ];

  const DOCK_ICONS = {
    chart: '<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M7 15v-4M12 15V9M17 15v-2"/>',
    grid: '<rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/>',
    key: '<circle cx="8" cy="15" r="3.5"/><path d="m10.5 12.5 8-8m-3 0h3v3"/>',
    admin: '<path d="M4 7h9M18.5 7H20M4 17h1.5M10.5 17H20"/><circle cx="15.5" cy="7" r="2.5"/><circle cx="8" cy="17" r="2.5"/>',
  };

  function dockIcon(name) {
    const ns = "http://www.w3.org/2000/svg";
    const svg = document.createElementNS(ns, "svg");
    svg.setAttribute("viewBox", "0 0 24 24");
    svg.setAttribute("fill", "none");
    svg.setAttribute("stroke", "currentColor");
    svg.setAttribute("stroke-width", "1.5");
    svg.setAttribute("stroke-linecap", "round");
    svg.setAttribute("stroke-linejoin", "round");
    svg.setAttribute("aria-hidden", "true");
    svg.innerHTML = DOCK_ICONS[name] || '<circle cx="12" cy="12" r="8"/>';
    return svg;
  }

  function renderDock(user) {
    const dock = $("#dock");
    if (!dock) return;
    dock.replaceChildren();
    const items = DOCK_ITEMS
      .filter((item) => {
        if (item.adminOnly && (!user || user.role !== "admin")) return false;
        if (item.avatar && !user) return false;
        return true;
      })
      .sort((a, b) => a.order - b.order);
    for (const item of items) {
      if (item.avatar && dock.children.length) dock.append(el("span", "dock-sep"));
      const label = t(item.label);
      const a = el("a", item.avatar ? "dock-item dock-avatar" : "dock-item");
      a.href = item.href;
      a.dataset.dockId = item.id;
      a.setAttribute("aria-label", label);
      if (item.avatar) {
        a.append(el("span", "dock-avatar-text", (user.name || user.email || "?").trim().charAt(0).toUpperCase()));
      } else {
        a.append(dockIcon(item.icon));
      }
      const tip = el("span", "dock-tip", label);
      tip.setAttribute("aria-hidden", "true");
      a.append(tip);
      dock.append(a);
    }
  }

  function when(value) {
    const d = value ? new Date(value) : null;
    if (!d || Number.isNaN(d.getTime())) return "—";
    return new Intl.DateTimeFormat(window.portalLangTag, { dateStyle: "medium" }).format(d);
  }

  function whenTime(value) {
    const d = value ? new Date(value) : null;
    if (!d || Number.isNaN(d.getTime())) return "—";
    return new Intl.DateTimeFormat(window.portalLangTag, { dateStyle: "medium", timeStyle: "short" }).format(d);
  }

  const num = (n) => new Intl.NumberFormat(window.portalLangTag).format(Number(n) || 0);

  // isoDate is the local YYYY-MM-DD, as the usage range query takes it.
  function isoDate(value) {
    const d = value ? new Date(value) : null;
    if (!d || Number.isNaN(d.getTime())) return "—";
    return d.getFullYear() + "-" + String(d.getMonth() + 1).padStart(2, "0") + "-" + String(d.getDate()).padStart(2, "0");
  }

  async function copyText(text) {
    let ok = false;
    try { await navigator.clipboard.writeText(text); ok = true; } catch { ok = false; }
    if (!ok) {
      const area = el("textarea");
      area.value = text;
      area.setAttribute("readonly", "");
      area.className = "copy-fallback";
      document.body.append(area);
      area.select();
      try { ok = document.execCommand("copy"); } catch { ok = false; }
      area.remove();
    }
    return ok;
  }

  // authError says the API's answer in the page's words: the server's own
  // messages are English and written for an API caller.
  function authError(status, data, action) {
    const why = String(data?.error || "");
    if (status === 0) return t("Network error. Try again.");
    if (status === 429) return t("Too many sign-in attempts. Try again later.");
    if (status === 401) return t("The email, username or password is not correct.");
    if (status === 403) {
      if (/disabled/i.test(why)) return t("This account is disabled. Ask the administrator to enable it.");
      if (/registration is closed/i.test(why)) return t("Registration is closed. Ask the administrator.");
      if (/no admin/i.test(why)) return t("Registration is closed: the platform has no administrator yet.");
      return t("You do not have permission to do that.");
    }
    if (status === 409) return action === "register"
      ? t("That email is already registered; sign in instead.")
      : t("That conflicts with existing data. Refresh and try again.");
    if (/not an email address|enter an email address/i.test(why)) return t("Enter a valid email address.");
    if (/password is 8/i.test(why)) return t("The password needs at least 8 characters.");
    if (/password is 1024/i.test(why)) return t("The password is 1024 characters at most.");
    if (/name is 80/i.test(why)) return t("The name is 80 characters at most.");
    if (status === 400) return why ? t("Invalid request: {why}", { why }) : t("Invalid request. Check what you filled in.");
    return why || t("That did not work. Try again.");
  }

  async function logout() {
    await api("/api/auth/logout", { method: "POST" });
    location.assign("/portal/");
  }

  function accountChrome(user) {
    for (const box of document.querySelectorAll("[data-account]")) {
      box.replaceChildren();
      if (!user) {
        const login = el("a", "btn ghost", t("Sign in"));
        login.href = "/portal/login";
        const register = el("a", "btn primary", t("Register"));
        register.href = "/portal/register";
        box.append(login, register);
        continue;
      }
      const who = el("a", "who", user.name || user.email);
      who.href = "/portal/console";
      const out = el("button", "btn ghost", t("Sign out"));
      out.type = "button";
      out.addEventListener("click", logout);
      box.append(who, out);
    }
  }

  function markNav() {
    const byPage = {
      console: "overview", models: "models", keys: "keys",
      profile: "profile", "admin-models": "admin",
    };
    const active = byPage[page] || (location.pathname === "/" ? "admin" : "");
    for (const a of document.querySelectorAll("[data-dock-id]")) {
      const on = a.dataset.dockId === active;
      a.classList.toggle("on", on);
      if (on) a.setAttribute("aria-current", "page");
      else a.removeAttribute("aria-current");
    }
  }

  function homeCta(user) {
    const box = $("#homeCta");
    if (!box || !user) return;
    box.replaceChildren();
    const console_ = el("a", "btn primary lg", t("Open the console"));
    console_.href = "/portal/console";
    const models = el("a", "btn ghost lg", t("Browse models"));
    models.href = "/portal/models";
    box.append(console_, models);
  }

  function loginPage(user) {
    if (user) { location.replace("/portal/console"); return; }
    const form = $("#loginForm");
    if (!form) return;
    const error = $("#loginError");
    const button = form.querySelector("button[type=submit]");
    form.addEventListener("submit", async (ev) => {
      ev.preventDefault();
      clearError(error);
      const account = form.elements.email.value.trim();
      const password = form.elements.password.value;
      if (!account || !password) { showError(error, t("Enter your email or username and password.")); return; }
      button.disabled = true;
      button.textContent = t("Signing in…");
      const r = await api("/api/auth/login", { method: "POST", body: { email: account, password } });
      if (r.ok) { location.assign("/portal/console"); return; }
      button.disabled = false;
      button.textContent = t("Sign in");
      showError(error, authError(r.status, r.data, "login"));
    });
    form.elements.email.focus();
  }

  function registerPage(user) {
    if (user) { location.replace("/portal/console"); return; }
    const form = $("#registerForm");
    if (!form) return;
    const error = $("#registerError");
    const button = form.querySelector("button[type=submit]");
    form.addEventListener("submit", async (ev) => {
      ev.preventDefault();
      clearError(error);
      const email = form.elements.email.value.trim();
      const name = form.elements.name.value.trim();
      const password = form.elements.password.value;
      const confirm = form.elements.confirm.value;
      if (!email || !password) { showError(error, t("Enter your email and password.")); return; }
      if (password.length < 8) { showError(error, t("The password needs at least 8 characters.")); return; }
      if (password !== confirm) { showError(error, t("The passwords do not match.")); return; }
      button.disabled = true;
      button.textContent = t("Creating…");
      const r = await api("/api/auth/register", { method: "POST", body: { email, name, password } });
      if (r.ok) { location.assign("/portal/console"); return; }
      button.disabled = false;
      button.textContent = t("Create an account");
      showError(error, authError(r.status, r.data, "register"));
    });
    form.elements.email.focus();
  }

  function consolePage(user) {
    if (!user) { location.replace("/portal/login?next=/portal/console"); return; }
    window.consoleUsage ? window.consoleUsage() : null;
  }

  function keysPage(user) {
    if (!user) { location.replace("/portal/login?next=/portal/keys"); return; }
    consoleKeys();
  }

  // profilePage draws the account itself, its real API-key count and the
  // Gateway's daily trend for the last year. The heatmap and its streaks
  // come from that trend; nothing is invented for a day with no calls.
  function profilePage(user) {
    if (!user) { location.replace("/portal/login?next=/portal/profile"); return; }

    const initials = (user.name || user.email || "?").trim().charAt(0).toUpperCase();
    const avatar = $("#profileAvatar");
    if (avatar) avatar.textContent = initials;
    const name = $("#profileName");
    if (name) name.textContent = user.name || user.email;
    const badge = $("#profileBadge");
    if (badge) badge.textContent = roleName(user.role);
    const uid = $("#profileUid");
    if (uid) uid.textContent = user.id;
    const joined = $("#profileJoined");
    if (joined) joined.textContent = t("Joined {date}", { date: isoDate(user.createdAt) });

    const adminCard = $("#adminEntry");
    if (adminCard) adminCard.hidden = user.role !== "admin";
    $("#profileSignOut")?.addEventListener("click", logout);

    const copyAccountId = async (button) => {
      if (!button) return;
      const ok = await copyText(user.id);
      button.classList.toggle("copied", ok);
      button.setAttribute("aria-label", ok ? t("Copied") : t("Copy failed"));
      setTimeout(() => {
        button.classList.remove("copied");
        button.setAttribute("aria-label", t("Copy account ID"));
      }, 1400);
    };
    $("#profileUidCopy")?.addEventListener("click", (e) => copyAccountId(e.currentTarget));
    $("#infoUidCopy")?.addEventListener("click", (e) => copyAccountId(e.currentTarget));

    const setText = (sel, value) => { const n = $(sel); if (n) n.textContent = value; };
    setText("#infoEmail", user.email);
    setText("#infoRole", roleName(user.role));
    setText("#infoJoined", when(user.createdAt));
    setText("#infoLastLogin", user.lastLoginAt ? whenTime(user.lastLoginAt) : "—");
    setText("#infoUid", user.id);

    if (user.role === "admin") adminRegistration();

    const loadKeys = async () => {
      const count = $("#infoKeyCount");
      if (!count) return;
      const r = await api("/api/user/keys");
      if (r.status === 401) { location.replace("/portal/login?next=/portal/profile"); return; }
      if (!r.ok) { count.textContent = "—"; return; }
      const keys = Array.isArray(r.data?.keys) ? r.data.keys : [];
      count.textContent = num(keys.length);
    };
    loadKeys();

    const error = $("#profileError");
    const heatmapEmpty = $("#heatmapEmpty");
    const grid = $("#heatmapGrid");
    const months = $("#heatmapMonths");
    const tabs = [...document.querySelectorAll("#activityTabs [data-activity]")];
    let daily = [];
    let mode = "daily";

    const addDays = (d, n) => {
      const out = new Date(d);
      out.setDate(out.getDate() + n);
      return out;
    };
    const inRange = (d, start, end) => d >= start && d <= end;

    const setStats = (days) => {
      if (!days) {
        for (const id of ["#profileYear", "#profilePeak", "#profileCurrent", "#profileLongest"]) setText(id, "—");
        return;
      }
      let year = 0, peak = 0, current = 0, longest = 0, run = 0;
      for (const p of days) {
        year += p.total;
        peak = Math.max(peak, p.total);
        if (p.total > 0) { run += 1; longest = Math.max(longest, run); } else run = 0;
      }
      for (let i = days.length - 1; i >= 0; i--) {
        if (days[i].total > 0) current += 1; else break;
      }
      setText("#profileYear", num(year));
      setText("#profilePeak", num(peak));
      setText("#profileCurrent", num(current));
      setText("#profileLongest", num(longest));
    };

    const renderActivity = () => {
      if (!grid || !months) return;
      grid.replaceChildren();
      months.replaceChildren();
      if (!daily.length) {
        if (heatmapEmpty) heatmapEmpty.hidden = false;
        return;
      }
      const byDay = new Map(daily.map((p) => [p.date, p.total]));
      const start = daily[0].dateObj;
      const end = daily[daily.length - 1].dateObj;
      const gridStart = new Date(start);
      gridStart.setDate(gridStart.getDate() - gridStart.getDay());
      const weeks = 53;

      const monthFmt = new Intl.DateTimeFormat(window.portalLangTag, { month: "short" });
      let lastMonth = -1;
      for (let w = 0; w < weeks; w++) {
        const col = addDays(gridStart, w * 7);
        const label = el("span");
        if (col.getMonth() !== lastMonth) {
          label.textContent = monthFmt.format(col);
          lastMonth = col.getMonth();
        }
        months.append(label);
      }

      const weekKey = (d) => {
        const s = new Date(d);
        s.setDate(s.getDate() - s.getDay());
        return isoDate(s);
      };
      const weekly = new Map();
      const cumulative = new Map();
      let running = 0;
      for (const p of daily) {
        const w = weekKey(p.dateObj);
        weekly.set(w, (weekly.get(w) || 0) + p.total);
        running += p.total;
        cumulative.set(p.date, running);
      }
      const maxDaily = Math.max(0, ...daily.map((p) => p.total));
      const maxWeekly = Math.max(0, ...weekly.values());
      const maxCumulative = running;
      const max = mode === "daily" ? maxDaily : mode === "weekly" ? maxWeekly : maxCumulative;

      for (let w = 0; w < weeks; w++) {
        for (let row = 0; row < 7; row++) {
          const d = addDays(gridStart, w * 7 + row);
          const key = isoDate(d);
          let value = 0;
          if (inRange(d, start, end)) {
            if (mode === "daily") value = byDay.get(key) || 0;
            else if (mode === "weekly") value = weekly.get(weekKey(d)) || 0;
            else value = cumulative.get(key) || 0;
          }
          const level = value > 0 && max > 0 ? Math.min(5, Math.ceil(value / max * 5)) : 0;
          const cell = el("span", "heatmap-cell" + (level ? " l" + level : ""));
          cell.title = key + " · " + num(value) + " tokens";
          cell.setAttribute("aria-hidden", "true");
          grid.append(cell);
        }
      }
      if (heatmapEmpty) heatmapEmpty.hidden = daily.some((p) => p.total > 0);
      const scroller = $(".heatmap-scroll");
      if (scroller && !scroller.dataset.ready) {
        scroller.dataset.ready = "1";
        scroller.scrollLeft = scroller.scrollWidth;
      }
    };

    const loadActivity = async () => {
      if (!grid) return;
      clearError(error);
      const today = new Date();
      today.setHours(0, 0, 0, 0);
      const from = addDays(today, -364);
      const q = new URLSearchParams({ from: isoDate(from), to: isoDate(today) });
      const r = await api("/api/user/usage/trend?" + q.toString());
      if (r.status === 401) { location.replace("/portal/login?next=/portal/profile"); return; }
      if (!r.ok) {
        setStats(null);
        daily = [];
        renderActivity();
        showError(error, t("Could not load token activity. Try again later."));
        if (heatmapEmpty) heatmapEmpty.hidden = true;
        return;
      }
      const points = Array.isArray(r.data?.trend) ? r.data.trend : [];
      const byDay = new Map(points.map((p) => [p.date, Number(p.total) || 0]));
      daily = [];
      for (let d = new Date(from); d <= today; d = addDays(d, 1)) {
        const key = isoDate(d);
        daily.push({ date: key, dateObj: new Date(d), total: byDay.get(key) || 0 });
      }
      setStats(daily);
      renderActivity();
    };

    for (const b of tabs) {
      b.addEventListener("click", () => {
        mode = b.dataset.activity || "daily";
        for (const x of tabs) x.classList.toggle("on", x === b);
        renderActivity();
      });
    }
    loadActivity();
  }

  // adminRegistration is the administrator's switch for sign-up in the
  // user portal, from /api/admin/registration: closed until they open it.
  function adminRegistration() {
    const toggle = $("#registrationToggle");
    const sub = $("#registrationSub");
    const error = $("#registrationError");
    if (!toggle) return;
    let open = false;
    const render = () => {
      toggle.disabled = false;
      toggle.setAttribute("aria-checked", open ? "true" : "false");
      sub.textContent = open
        ? t("Open · Any visitor can register")
        : t("Closed · Visitors cannot register themselves; administrators can still create accounts");
    };
    const fail = (r) => {
      if (r.status === 401) { location.replace("/portal/login?next=/portal/profile"); return; }
      showError(error, r.status === 403 ? t("Needs administrator rights.") : t("Could not save the registration setting. Try again later."));
    };
    const load = async () => {
      clearError(error);
      const r = await api("/api/admin/registration");
      if (r.status === 401) { location.replace("/portal/login?next=/portal/profile"); return; }
      if (!r.ok) { sub.textContent = t("Could not read"); showError(error, t("Could not load the registration setting. Refresh and try again.")); return; }
      open = !!r.data?.open;
      render();
    };
    toggle.addEventListener("click", async () => {
      if (toggle.disabled) return;
      clearError(error);
      toggle.disabled = true;
      const want = !open;
      const r = await api("/api/admin/registration", { method: "POST", body: { open: want } });
      if (!r.ok) { toggle.disabled = false; fail(r); return; }
      open = !!r.data?.open;
      render();
    });
    load();
  }

  // consoleKeys is the account's own API keys, from /api/user/keys: the
  // owner is the signed-in account on the server, never a field here. A
  // key's model grant is chosen from /api/user/models, the open catalog.
  function consoleKeys() {
    const list = $("#keyList");
    if (!list) return;
    const empty = $("#keyEmpty");
    const count = $("#keyCount");
    const sumKeys = $("#sumKeys");
    const error = $("#keyError");
    const newBtn = $("#keyNew");
    const form = $("#keyCreate");
    const secretBox = $("#keySecret");
    const secretValue = $("#keySecretValue");
    const pick = $("#keyModelPick");
    const pickNote = $("#keyModelNote");
    const dialog = $("#keyModelsDialog");
    const dialogForm = $("#keyModelsForm");
    const dialogPick = $("#keyModelsPick");
    const dialogNote = $("#keyModelsNote");
    const dialogError = $("#keyModelsError");
    let keys = [];
    let openModels = [];
    let modelsReady = false;
    let editing = null;

    const setError = (why) => { if (why) showError(error, why); else clearError(error); };
    const fail = (r, action) => setError(r.status === 0 ? t("Network error. Try again.") : authError(r.status, r.data, action));
    const keyWhen = (k) => k.lastUsed ? t("Last used {when}", { when: whenTime(k.lastUsed) }) : t("Never used");
    const modelName = (id) => (openModels.find((m) => m.id === id)?.name || id);
    const openIDs = () => new Set(openModels.map((m) => m.id));

    const modelsText = (k) => {
      const ids = Array.isArray(k.models) ? k.models : [];
      if (!ids.length) return t("No models granted");
      const names = ids.slice(0, 2).map(modelName);
      const sep = window.portalLang === "en" ? ", " : "、";
      return ids.length > 2
        ? t("{n} models · {names} and more", { n: ids.length, names: names.join(sep) })
        : t("{n} models · {names}", { n: ids.length, names: names.join(sep) });
    };

    const picker = (box, selected, includeClosed) => {
      if (!box) return;
      box.replaceChildren();
      const chosen = new Set(selected || []);
      const known = openIDs();
      for (const m of openModels) {
        const label = el("label", "key-model-opt");
        const input = el("input");
        input.type = "checkbox";
        input.value = m.id;
        input.checked = chosen.has(m.id);
        label.append(input, el("span", "", m.name || m.id), el("small", "", m.publicId || m.id));
        box.append(label);
      }
      if (includeClosed) {
        for (const id of [...chosen].filter((x) => !known.has(x))) {
          const label = el("label", "key-model-opt closed");
          label.title = t("The platform has closed this model; saving the grant removes it from the key.");
          const input = el("input");
          input.type = "checkbox";
          input.checked = true;
          input.disabled = true;
          label.append(input, el("span", "", id), el("small", "", t("Closed by the platform")));
          box.append(label);
        }
      }
      if (!box.children.length) box.append(el("p", "key-model-note", t("The platform has no open models yet.")));
    };

    const checked = (box) => box
      ? [...box.querySelectorAll("input[type=checkbox]:checked:not(:disabled)")].map((i) => i.value)
      : [];

    const renderCreatePicker = () => {
      if (!pick) return;
      const selected = pick.children.length ? checked(pick) : openModels.map((m) => m.id);
      picker(pick, selected, false);
      if (!modelsReady) pickNote.textContent = t("Reading the platform's open models…");
      else if (!openModels.length) pickNote.textContent = t("The platform has no open models yet; the administrator opens them before keys can be created.");
      else pickNote.textContent = t("All chosen by default; this key may call only the checked models.");
    };

    const rowOf = (k) => {
      const row = el("div", "key-row");
      const main = el("div", "key-main");
      main.dataset.label = t("Name");
      main.append(el("b", "", k.name || t("Untitled")));
      main.append(el("span", "key-subline", t("Created {when}", { when: when(k.createdAt) }) + " · " + keyWhen(k)));
      const masked = el("div", "key-masked");
      masked.dataset.label = t("Key");
      masked.append(el("code", "", (k.prefix || "") + "…"));
      const models = el("div", "key-models");
      models.dataset.label = t("Model grants");
      models.append(el("span", "key-models-line", modelsText(k)));
      if (k.legacyModels) models.append(el("span", "key-legacy", t("Old key: its grant was fixed to the models open when it was first enabled")));
      const status = el("div", "key-status");
      status.dataset.label = t("Status");
      status.append(el("span", "status-dot" + (k.off ? " off" : ""), ""), el("span", "", k.off ? t("Disabled") : t("Enabled")));
      const actions = el("div", "key-actions");
      const modelsBtn = el("button", "btn row-action sm key-models-btn", t("Model access"));
      modelsBtn.type = "button";
      modelsBtn.addEventListener("click", () => openModelsDialog(k));
      const flip = el("button", "btn text sm", k.off ? t("Enable") : t("Disable"));
      flip.type = "button";
      flip.addEventListener("click", async () => {
        setError("");
        flip.disabled = true;
        const r = await api("/api/user/keys/" + encodeURIComponent(k.id), { method: "PATCH", body: { off: !k.off } });
        if (!r.ok) { flip.disabled = false; fail(r, "key"); return; }
        await load();
      });
      const del = el("button", "btn text sm danger", t("Delete"));
      del.type = "button";
      del.addEventListener("click", async () => {
        if (!window.confirm(t("Delete the API Key “{name}”? Clients using it stop working at once.", { name: k.name || t("Untitled") }))) return;
        setError("");
        del.disabled = true;
        const r = await api("/api/user/keys/" + encodeURIComponent(k.id), { method: "DELETE" });
        if (!r.ok && r.status !== 204) { del.disabled = false; fail(r, "key"); return; }
        await load();
      });
      actions.append(modelsBtn, flip, del);
      row.append(main, masked, models, status, actions);
      return row;
    };

    const render = () => {
      list.replaceChildren();
      count.textContent = t("{n} keys", { n: keys.length });
      if (sumKeys) sumKeys.textContent = String(keys.length);
      empty.hidden = keys.length > 0;
      for (const k of keys) list.append(rowOf(k));
    };

    const load = async () => {
      const r = await api("/api/user/keys");
      if (r.status === 401) { location.replace("/portal/login?next=/portal/keys"); return; }
      if (!r.ok) { fail(r, "key"); return; }
      keys = Array.isArray(r.data?.keys) ? r.data.keys : [];
      render();
    };

    const hideSecret = () => { secretBox.hidden = true; secretValue.textContent = ""; };
    hideSecret();

    const openModelsDialog = (k) => {
      if (!dialog) return;
      editing = k;
      clearError(dialogError);
      $("#keyModelsTitle").textContent = t("Model access · {name}", { name: k.name || t("Untitled") });
      $("#keyModelsSub").textContent = t("This key may call only the checked models; a model the platform closed cannot be called even if it was checked.");
      const known = openIDs();
      const granted = Array.isArray(k.models) ? k.models : [];
      const selected = granted.filter((id) => known.has(id));
      // a model closed after the key was granted it stays visible, disabled:
      // saving removes it from the grant rather than hiding what happened
      const shown = [...selected, ...granted.filter((id) => !known.has(id))];
      picker(dialogPick, shown, true);
      dialogNote.textContent = k.legacyModels
        ? t("This is an old key: its model grant was fixed when it was first enabled. Choose again; closed models are removed.")
        : t("Takes effect on save. Closed models cannot stay granted and are removed.");
      dialog.showModal();
    };
    const closeModelsDialog = () => { if (dialog?.open) dialog.close(); };

    newBtn.disabled = true;
    newBtn.addEventListener("click", () => {
      if (!openModels.length) {
        setError(t("The platform has no open models yet; the administrator opens them before an API Key can be created."));
        return;
      }
      form.hidden = false;
      newBtn.hidden = true;
      renderCreatePicker();
      form.elements.name.focus();
    });
    const closeCreate = () => {
      form.hidden = true;
      newBtn.hidden = false;
      form.elements.name.value = "";
    };
    $("#keyCancel")?.addEventListener("click", closeCreate);
    $("#keyModelsAll")?.addEventListener("click", () => {
      for (const i of pick.querySelectorAll("input[type=checkbox]")) i.checked = true;
    });
    $("#keyModelsNone")?.addEventListener("click", () => {
      for (const i of pick.querySelectorAll("input[type=checkbox]")) i.checked = false;
    });
    $("#keyDialogAll")?.addEventListener("click", () => {
      for (const i of dialogPick.querySelectorAll("input[type=checkbox]:not(:disabled)")) i.checked = true;
    });
    $("#keyDialogNone")?.addEventListener("click", () => {
      for (const i of dialogPick.querySelectorAll("input[type=checkbox]:not(:disabled)")) i.checked = false;
    });
    form?.addEventListener("submit", async (ev) => {
      ev.preventDefault();
      setError("");
      const name = form.elements.name.value.trim();
      if (!name) { setError(t("Give this key a name.")); return; }
      const models = checked(pick);
      if (!models.length) { setError(t("Choose at least one model for this key.")); return; }
      const button = form.querySelector("button[type=submit]");
      button.disabled = true;
      button.textContent = t("Creating…");
      const r = await api("/api/user/keys", { method: "POST", body: { name, models } });
      button.disabled = false;
      button.textContent = t("Create");
      if (!r.ok) { fail(r, "key"); return; }
      closeCreate();
      secretValue.textContent = r.data.secret;
      secretBox.hidden = false;
      secretBox.scrollIntoView({ block: "nearest" });
      await load();
    });
    $("#keyModelsClose")?.addEventListener("click", closeModelsDialog);
    $("#keyModelsCancel")?.addEventListener("click", closeModelsDialog);
    dialog?.addEventListener("click", (e) => { if (e.target === dialog) closeModelsDialog(); });
    dialogForm?.addEventListener("submit", async (ev) => {
      ev.preventDefault();
      clearError(dialogError);
      const models = checked(dialogPick);
      if (!models.length) { showError(dialogError, t("Choose at least one open model; otherwise this key cannot call any model.")); return; }
      const button = dialogForm.querySelector("button[type=submit]");
      button.disabled = true;
      button.textContent = t("Saving…");
      const r = await api("/api/user/keys/" + encodeURIComponent(editing.id), { method: "PATCH", body: { models } });
      button.disabled = false;
      button.textContent = t("Save model access");
      if (!r.ok) { showError(dialogError, r.status === 0 ? t("Network error. Try again.") : authError(r.status, r.data, "key")); return; }
      closeModelsDialog();
      await load();
    });
    $("#keySecretClose")?.addEventListener("click", hideSecret);
    $("#keyCopy")?.addEventListener("click", async () => {
      const text = secretValue.textContent;
      if (!text) return;
      let ok = false;
      try { await navigator.clipboard.writeText(text); ok = true; } catch { ok = false; }
      if (!ok) {
        const area = el("textarea");
        area.value = text;
        area.setAttribute("readonly", "");
        area.className = "copy-fallback";
        document.body.append(area);
        area.select();
        try { ok = document.execCommand("copy"); } catch { ok = false; }
        area.remove();
      }
      const button = $("#keyCopy");
      button.textContent = ok ? t("Copied") : t("Copy failed");
      setTimeout(() => { button.textContent = t("Copy"); }, 1600);
    });

    const loadModels = async () => {
      const r = await api("/api/user/models");
      if (r.status === 401) { location.replace("/portal/login?next=/portal/keys"); return; }
      if (!r.ok) {
        modelsReady = false;
        newBtn.disabled = true;
        setError(t("Could not load the model catalogue; keys cannot be created until it does. Refresh later."));
        return;
      }
      openModels = Array.isArray(r.data?.models) ? r.data.models : [];
      modelsReady = true;
      newBtn.disabled = openModels.length === 0;
      newBtn.title = openModels.length ? "" : t("The platform has no open models yet");
      if (!openModels.length) setError(t("The platform has no open models yet; the administrator opens them before an API Key can be created."));
      else setError("");
      renderCreatePicker();
      if (dialog?.open && editing) {
        const known = openIDs();
        const granted = Array.isArray(editing.models) ? editing.models : [];
        const selected = granted.filter((id) => known.has(id));
        const shown = [...selected, ...granted.filter((id) => !known.has(id))];
        picker(dialogPick, shown, true);
      }
    };

    loadModels();
    load();
  }

  function modelsPage(user) {
    if (!user) { location.replace("/portal/login?next=/portal/models"); return; }
    if (window.modelPlaza) window.modelPlaza(user);
  }

  function adminModelsPage(user) {
    if (!user) { location.replace("/portal/login?next=/portal/admin-models"); return; }
    if (user.role !== "admin") { location.replace("/portal/console"); return; }
    if (window.adminModels) window.adminModels(user);
  }

  // languageSwitch is the footer's choice: the cookie keeps it for this browser
// (the server reads it for <html lang>), and ?lang= is dropped so a temporary
// override never turns into a saved one. The page reloads on the same address,
// keeping its query, so what is open stays open.
  function languageSwitch() {
    for (const b of document.querySelectorAll("[data-lang]")) {
      b.addEventListener("click", () => {
        const lang = b.dataset.lang;
        if (lang !== "zh" && lang !== "en") return;
        document.cookie = "minirelay_lang=" + lang + "; Path=/; Max-Age=31536000; SameSite=Lax";
        const url = new URL(location.href);
        url.searchParams.delete("lang");
        location.assign(url.pathname + url.search + url.hash);
      });
    }
  }

  async function main() {
    applyPortalI18n();
    languageSwitch();
    // sign-in and sign-up are served only to a signed-out browser (the
    // server redirects a session to the console), so their first look
    // needs no /api/auth/me
    const formPage = page === "login" || page === "register";
    const user = formPage ? null : await me();
    accountChrome(user);
    renderDock(user);
    markNav();
    homeCta(user);
    if (page === "login") loginPage(user);
    else if (page === "register") registerPage(user);
    else if (page === "console") consolePage(user);
    else if (page === "keys") keysPage(user);
    else if (page === "profile") profilePage(user);
    else if (page === "models") modelsPage(user);
    else if (page === "admin-models") adminModelsPage(user);
    document.body.classList.add("ready");
  }

  main();
})();