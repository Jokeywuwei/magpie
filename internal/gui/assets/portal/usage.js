// The console's usage and request log, read from /api/user/usage/*. The
// server takes the owner from the signed-in account, so nothing here names
// an account; a record with no owner is never served.
(() => {
  "use strict";
  const $ = (sel, root = document) => root.querySelector(sel);
  const el = (tag, cls, text) => {
    const n = document.createElement(tag);
    if (cls) n.className = cls;
    if (text != null) n.textContent = text;
    return n;
  };
  const num = (n) => new Intl.NumberFormat(window.portalLangTag).format(Number(n) || 0);
  const whenTime = (value) => {
    const d = value ? new Date(value) : null;
    if (!d || Number.isNaN(d.getTime())) return "—";
    return new Intl.DateTimeFormat(window.portalLangTag, { dateStyle: "medium", timeStyle: "short" }).format(d);
  };
  const showError = (node, why) => { if (node) { node.textContent = why; node.hidden = false; } };
  const clearError = (node) => { if (node) { node.hidden = true; node.textContent = ""; } };
  const emptyState = (title, note) => {
    const box = el("div", "empty-state");
    const ico = el("span", "empty-ico", "◌");
    ico.setAttribute("aria-hidden", "true");
    box.append(ico, el("b", "", title));
    if (note) box.append(el("p", "", note));
    return box;
  };
  const isoDate = (d) => d.getFullYear() + "-" + String(d.getMonth() + 1).padStart(2, "0") + "-" + String(d.getDate()).padStart(2, "0");
  async function api(path) {
    let res;
    try { res = await fetch(path, { credentials: "same-origin", cache: "no-store" }); }
    catch { return { ok: false, status: 0, data: null }; }
    let data = null;
    try { data = await res.json(); } catch { /* no body */ }
    return { ok: res.ok, status: res.status, data };
  }

  window.consoleUsage = function consoleUsage() {
    const periodBox = $("#usagePeriod");
    if (!periodBox) return;
    const view = { period: "7d", page: 1, model: "", state: "", models: [] };
    let reportSeq = 0, logSeq = 0, autoTimer = null;
    const stats = {};
    for (const b of document.querySelectorAll("#usageStats [data-stat]")) stats[b.dataset.stat] = b;
    const summaryEls = {};
    for (const b of document.querySelectorAll("#consoleSummary [data-summary]")) summaryEls[b.dataset.summary] = b;
    const usageError = $("#usageError"), logError = $("#logError");
    const chart = $("#usageChart"), axis = $("#usageAxis");
    const modelsBox = $("#usageModels"), modelsEmpty = $("#usageModelsEmpty");
    const logList = $("#logList"), logEmpty = $("#logEmpty"), logTotal = $("#logTotal"), logPage = $("#logPage"), logPrev = $("#logPrev"), logNext = $("#logNext");
    const stateChips = [...document.querySelectorAll("#logStateFilters [data-log-state]")];
    const auto = $("#logAuto");

    const needSignIn = (r) => {
      if (r.status !== 401) return false;
      location.replace("/portal/login?next=/portal/console");
      return true;
    };

    // yesterday is a one-day from/to range; the API's presets stop at 7d.
    const rangeParams = () => {
      const q = new URLSearchParams();
      if (view.period === "yesterday") {
        const d = new Date();
        d.setHours(0, 0, 0, 0);
        d.setDate(d.getDate() - 1);
        const y = isoDate(d);
        q.set("from", y);
        q.set("to", y);
      } else {
        q.set("period", view.period);
      }
      return q;
    };
    const rangeQuery = () => "?" + rangeParams().toString();

    const setSummary = (s) => {
      for (const [k, b] of Object.entries(summaryEls)) b.textContent = s ? num(s[k]) : "—";
    };
    const setStats = (s) => {
      for (const [k, b] of Object.entries(stats)) b.textContent = s ? num(s[k]) : "—";
      if (stats.failed) stats.failed.classList.toggle("failed", !!s);
      if (stats.success) stats.success.classList.toggle("success", !!s);
    };

    const renderChart = (points) => {
      chart.replaceChildren();
      axis.replaceChildren();
      if (!points.length || points.every((p) => !p.calls && !p.total)) {
        chart.append(emptyState(
          t("No requests in this time range."),
          t("Requests made with this account's API Keys will appear here."),
        ));
        return;
      }
      const n = points.length;
      const max = Math.max(1, ...points.map((p) => p.total || 0));
      const width = Math.max(n * 18, 320);
      const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
      svg.setAttribute("viewBox", "0 0 " + width + " 100");
      svg.setAttribute("width", String(width));
      svg.setAttribute("preserveAspectRatio", "none");
      svg.setAttribute("role", "img");
      svg.setAttribute("aria-label", t("Daily token usage"));
      const bw = width / n;
      points.forEach((p, i) => {
        const h = p.total ? Math.max(2, (p.total / max) * 92) : 1;
        const rect = document.createElementNS("http://www.w3.org/2000/svg", "rect");
        rect.setAttribute("x", String(i * bw + bw * 0.18));
        rect.setAttribute("y", String(100 - h));
        rect.setAttribute("width", String(bw * 0.64));
        rect.setAttribute("height", String(h));
        rect.setAttribute("rx", "2");
        if (p.total) rect.setAttribute("fill", "currentColor");
        else rect.setAttribute("class", "zero");
        const title = document.createElementNS("http://www.w3.org/2000/svg", "title");
        title.textContent = p.date + " · " + t("{tokens} tokens · {calls} requests", { tokens: num(p.total), calls: num(p.calls) });
        rect.append(title);
        svg.append(rect);
      });
      chart.append(svg);
      const step = Math.max(1, Math.ceil(n / 8));
      points.forEach((p, i) => axis.append(el("span", "", i % step === 0 ? p.date.slice(5) : "")));
    };

    const renderModels = (models) => {
      modelsBox.replaceChildren();
      modelsEmpty.hidden = models.length > 0;
      const max = Math.max(1, ...models.map((m) => m.total || 0));
      for (const m of models) {
        const row = el("div", "umodel");
        row.append(el("b", "", m.model), el("em", "", t("{n} calls", { n: num(m.calls) })), el("em", "", num(m.total) + " tokens"));
        const bar = document.createElement("progress");
        bar.max = max;
        bar.value = m.total || 0;
        bar.setAttribute("aria-label", m.model + " " + num(m.total) + " tokens");
        row.append(bar);
        modelsBox.append(row);
      }
    };

    const fillModelSelect = () => {
      const sel = $("#logModel");
      if (!sel) return;
      const cur = sel.value;
      const all = el("option", "", t("All models"));
      all.value = "";
      sel.replaceChildren(all);
      for (const m of view.models) {
        const o = el("option", "", m.model);
        o.value = m.model;
        sel.append(o);
      }
      const keep = view.models.some((m) => m.model === cur);
      sel.value = keep ? cur : "";
      view.model = sel.value;
    };

    const loadReport = async () => {
      const seq = ++reportSeq;
      clearError(usageError);
      setSummary(null);
      $("#usageStats").classList.add("is-loading");
      setStats(null);
      chart.replaceChildren(el("p", "key-empty", t("Loading…")));
      axis.replaceChildren();
      modelsBox.replaceChildren();
      modelsEmpty.hidden = true;
      const q = rangeQuery();
      const [sum, trend, models] = await Promise.all([
        api("/api/user/usage/summary" + q),
        api("/api/user/usage/trend" + q),
        api("/api/user/usage/models" + q),
      ]);
      if (seq !== reportSeq) return;
      if (needSignIn(sum) || needSignIn(trend) || needSignIn(models)) return;
      $("#usageStats").classList.remove("is-loading");
      if (!sum.ok || !trend.ok || !models.ok) {
        showError(usageError, t("Could not load usage. Try again later."));
        setStats(null);
        setSummary(null);
        chart.replaceChildren();
        axis.replaceChildren();
        modelsBox.replaceChildren();
        modelsEmpty.hidden = true;
        return;
      }
      const summary = (sum.data && sum.data.summary) || {};
      view.models = Array.isArray(models.data && models.data.models) ? models.data.models : [];
      setStats(summary);
      setSummary(summary);
      renderChart(Array.isArray(trend.data && trend.data.trend) ? trend.data.trend : []);
      renderModels(view.models);
      fillModelSelect();
      await loadLogs(1);
    };

    const addCell = (row, cls, text, label) => {
      const c = el("span", cls, text);
      c.dataset.label = label;
      row.append(c);
    };

    const renderLogs = (data) => {
      logList.replaceChildren();
      const rows = Array.isArray(data.requests) ? data.requests : [];
      logEmpty.hidden = rows.length > 0;
      if (rows.length) {
        const head = el("div", "log-row head");
        for (const title of [t("Time"), t("Model"), "API Key", t("Status"), "Token", t("Duration")]) head.append(el("span", "", title));
        logList.append(head);
      }
      for (const r of rows) {
        const row = el("div", "log-row");
        addCell(row, "log-time", whenTime(r.time), t("Time"));
        const model = el("span", "log-model");
        model.dataset.label = t("Model");
        model.append(el("b", "", r.displayModel || r.model || "unknown"));
        if (r.provider) model.append(el("span", "", r.provider));
        row.append(model);
        addCell(row, "log-key", r.keyName || "—", "API Key");
        const state = r.state === "success" ? "ok" : r.state === "failed" ? "fail" : "cancel";
        const label = r.state === "success" ? t("Success") : r.state === "failed" ? t("Failed") : t("Cancelled");
        const tag = el("span", "log-status tag " + state, label);
        tag.dataset.label = t("Status");
        row.append(tag);
        addCell(row, "log-tokens", "in " + num(r.input) + " · out " + num(r.output) + " · " + num(r.total), "Token");
        addCell(row, "log-ms", num(r.millis) + " ms", t("Duration"));
        if (r.error) row.append(el("div", "log-err", r.error));
        logList.append(row);
      }
      logTotal.textContent = t("Total {n}", { n: num(data.total) });
      logPage.textContent = t("Page {page} / {pages}", { page: data.page, pages: data.pages });
      logPrev.disabled = data.page <= 1;
      logNext.disabled = data.page >= data.pages;
      view.page = data.page;
    };

    const loadLogs = async (page) => {
      const seq = ++logSeq;
      view.page = page || view.page || 1;
      clearError(logError);
      logList.classList.add("is-loading");
      logList.setAttribute("aria-busy", "true");
      const q = rangeParams();
      q.set("page", String(view.page));
      q.set("page_size", "20");
      if (view.model) q.set("model", view.model);
      if (view.state) q.set("state", view.state);
      const r = await api("/api/user/usage/requests?" + q.toString());
      if (seq !== logSeq) return;
      logList.classList.remove("is-loading");
      logList.removeAttribute("aria-busy");
      if (needSignIn(r)) return;
      if (!r.ok) {
        showError(logError, t("Could not load the log. Try again later."));
        logList.replaceChildren();
        logEmpty.hidden = true;
        logTotal.textContent = "";
        logPage.textContent = "";
        logPrev.disabled = true;
        logNext.disabled = true;
        return;
      }
      renderLogs(r.data || {});
    };

    const setAuto = (on) => {
      if (auto) auto.setAttribute("aria-checked", on ? "true" : "false");
      if (autoTimer) {
        clearInterval(autoTimer);
        autoTimer = null;
      }
      if (on) {
        autoTimer = setInterval(() => {
          if (!document.hidden) loadLogs(view.page);
        }, 15000);
      }
    };

    for (const b of periodBox.querySelectorAll("[data-period]")) {
      b.addEventListener("click", () => {
        if (b.dataset.period === view.period) return;
        view.period = b.dataset.period;
        for (const x of periodBox.querySelectorAll("[data-period]")) x.classList.toggle("on", x === b);
        view.page = 1;
        view.model = "";
        const select = $("#logModel");
        if (select) select.value = "";
        loadReport();
      });
    }
    $("#logModel")?.addEventListener("change", (e) => {
      view.model = e.target.value;
      view.page = 1;
      loadLogs(1);
    });
    for (const b of stateChips) {
      b.addEventListener("click", () => {
        const value = b.dataset.logState || "all";
        view.state = value === "all" ? "" : value;
        for (const x of stateChips) x.classList.toggle("on", x === b);
        view.page = 1;
        loadLogs(1);
      });
    }
    $("#logRefresh")?.addEventListener("click", () => loadLogs(view.page));
    auto?.addEventListener("click", () => setAuto(auto.getAttribute("aria-checked") !== "true"));
    logPrev?.addEventListener("click", () => { if (view.page > 1) loadLogs(view.page - 1); });
    logNext?.addEventListener("click", () => loadLogs(view.page + 1));

    loadReport();
  };
})();
