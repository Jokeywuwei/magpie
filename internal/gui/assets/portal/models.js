// The user's model plaza: only the models the platform opens now, from
// /api/user/models. The server decides what is open; this page filters and
// draws, and never invents a model or a price. A card's detail shows how to
// call it through this deployment's own Gateway address.
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
  const protocolLabel = (p) => ({
    chat: "Chat Completions",
    responses: "Responses",
    anthropic: "Anthropic Messages",
    gemini: "Gemini",
  })[p] || p;
  const protocolShort = (p) => ({
    chat: "Chat",
    responses: "Responses",
    anthropic: "Anthropic",
    gemini: "Gemini",
  })[p] || p;
  const protocolClass = (p) => p === "chat" || p === "responses" ? "openai" : p;
  const matchesProtocol = (m, filter) => {
    if (filter === "all") return true;
    const ps = Array.isArray(m.protocols) ? m.protocols : [];
    if (filter === "openai") return ps.includes("chat") || ps.includes("responses");
    return ps.includes(filter);
  };
  const matchesAvailability = (m, want) => {
    if (want === "all") return true;
    return want === "available" ? !!m.available : !m.available;
  };
  const q = (m) => [m.name, m.id, m.publicId, m.provider, m.description].filter(Boolean).join(" ").toLowerCase();
  // the id users call: a routing group's own id, without "group/"
  const callId = (m) => m.publicId || m.id;
  const initialOf = (s) => {
    const v = String(s || "").trim();
    return v ? v.charAt(0).toUpperCase() : "?";
  };
  // a model whose public id starts with deepseek wears the DeepSeek mark
  const isDeepSeek = (m) => /^deepseek/i.test(String(m.publicId || m.id || "")) ||
    /^deepseek/i.test(String(m.name || ""));
  const brandMark = (m) => {
    if (!isDeepSeek(m)) return null;
    const img = el("img");
    img.src = "/portal/deepseek.svg";
    img.alt = "";
    img.setAttribute("aria-hidden", "true");
    return img;
  };

  async function api(path) {
    let res;
    try { res = await fetch(path, { credentials: "same-origin", cache: "no-store" }); }
    catch { return { ok: false, status: 0, data: null }; }
    let data = null;
    try { data = await res.json(); } catch { /* no body */ }
    return { ok: res.ok, status: res.status, data };
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

  function exampleFor(model, proto, base) {
    const origin = (base || location.origin).replace(/\/+$/, "");
    const key = "<YOUR_API_KEY>";
    const id = model.publicId || model.id;
    if (proto === "responses") {
      return `curl ${origin}/v1/responses \\
  -H "Authorization: Bearer ${key}" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"${id}","input":"${t("hello")}"}'`;
    }
    if (proto === "anthropic") {
      return `curl ${origin}/v1/messages \\
  -H "x-api-key: ${key}" \\
  -H "anthropic-version: 2023-06-01" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"${id}","max_tokens":512,"messages":[{"role":"user","content":"${t("hello")}"}]}'`;
    }
    if (proto === "gemini") {
      return `curl "${origin}/v1beta/models/${id}:generateContent" \\
  -H "x-goog-api-key: ${key}" \\
  -H "Content-Type: application/json" \\
  -d '{"contents":[{"parts":[{"text":"${t("hello")}"}]}]}'`;
    }
    return `curl ${origin}/v1/chat/completions \\
  -H "Authorization: Bearer ${key}" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"${id}","messages":[{"role":"user","content":"${t("hello")}"}],"stream":false}'`;
  }

  // contextLabel shows the catalog's real context size; 0 is not declared.
  function contextLabel(n) {
    n = Number(n) || 0;
    if (!n) return "—";
    if (n >= 1000000) return (n / 1000000).toFixed(n % 1000000 ? 1 : 0) + "M tokens";
    if (n >= 1000) return Math.round(n / 1000) + "K tokens";
    return n + " tokens";
  }

  const protocolsText = (m) => {
    const ps = Array.isArray(m.protocols) ? m.protocols : [];
    return ps.length ? ps.map(protocolShort).join(" · ") : "—";
  };
  const protocolBadge = (m) => {
    const ps = Array.isArray(m.protocols) ? m.protocols : [];
    return ps.length ? protocolShort(ps[0]) : t("Opened");
  };

  function copyIcon() {
    const ns = "http://www.w3.org/2000/svg";
    const svg = document.createElementNS(ns, "svg");
    svg.setAttribute("viewBox", "0 0 24 24");
    svg.setAttribute("fill", "none");
    svg.setAttribute("stroke", "currentColor");
    svg.setAttribute("stroke-width", "1.5");
    svg.setAttribute("stroke-linecap", "round");
    svg.setAttribute("stroke-linejoin", "round");
    svg.setAttribute("aria-hidden", "true");
    const a = document.createElementNS(ns, "rect");
    a.setAttribute("x", "9"); a.setAttribute("y", "9");
    a.setAttribute("width", "11"); a.setAttribute("height", "11");
    a.setAttribute("rx", "2");
    const b = document.createElementNS(ns, "path");
    b.setAttribute("d", "M15 5H5a2 2 0 0 0-2 2v10");
    svg.append(a, b);
    return svg;
  }

  function gridIcon() {
    const ns = "http://www.w3.org/2000/svg";
    const svg = document.createElementNS(ns, "svg");
    svg.setAttribute("viewBox", "0 0 12 12");
    svg.setAttribute("fill", "none");
    svg.setAttribute("stroke", "currentColor");
    svg.setAttribute("stroke-width", "1.3");
    svg.setAttribute("aria-hidden", "true");
    for (const [x, y] of [[2, 2], [7, 2], [2, 7], [7, 7]]) {
      const r = document.createElementNS(ns, "rect");
      r.setAttribute("x", String(x));
      r.setAttribute("y", String(y));
      r.setAttribute("width", "3");
      r.setAttribute("height", "3");
      r.setAttribute("rx", ".8");
      svg.append(r);
    }
    return svg;
  }

  window.modelPlaza = function modelPlaza(user) {
    if (!user) { location.replace("/portal/login?next=/portal/models"); return; }
    const list = $("#modelList");
    const loading = $("#modelLoading");
    if (!list) return;

    const providerBox = $("#providerList");
    const plazaProvider = $("#plazaProvider");
    const plazaCount = $("#plazaCount");
    const error = $("#modelError");
    const empty = $("#modelEmpty");
    const emptyTitle = $("#modelEmptyTitle");
    const emptyText = $("#modelEmptyText");
    const emptyAction = $("#modelEmptyAction");
    const note = $("#modelNote");
    const search = $("#modelSearch");
    const stateFilters = [...document.querySelectorAll("#modelStateFilters .chip")];
    const filters = [...document.querySelectorAll("#modelFilters .chip")];
    const dialog = $("#modelDetail");
    const detailName = $("#detailName");
    const detailProvider = $("#detailProvider");
    const detailId = $("#detailId");
    const detailBadges = $("#detailBadges");
    const detailDesc = $("#detailDesc");
    const detailGrid = $("#detailGrid");
    const detailTabs = $("#detailTabs");
    const detailCode = $("#detailCode");
    const detailNote = $("#detailNote");
    const state = {
      models: [], filter: "all", avail: "all", provider: "", query: "",
      current: null, proto: "", gateway: location.origin,
    };

    const providerValue = (m) => m.kind === "group" ? "__groups" : (m.provider || "");
    const providerLabel = (m) => m.kind === "group" ? t("Routing groups") : (m.provider || t("Unknown provider"));
    const providerName = (key) => key === "__groups" ? t("Routing groups") : key;
    const visible = () => state.models.filter((m) => matchesProtocol(m, state.filter)
      && (!state.provider || providerValue(m) === state.provider)
      && matchesAvailability(m, state.avail)
      && (!state.query || q(m).includes(state.query)));

    const providerIcon = (name, all) => {
      const ico = el("span", "provider-ico");
      ico.setAttribute("aria-hidden", "true");
      if (all) ico.append(gridIcon());
      else ico.textContent = initialOf(name);
      return ico;
    };

    const syncProviderRows = () => {
      if (!providerBox) return;
      for (const row of providerBox.querySelectorAll(".provider-row")) {
        const on = (row.dataset.provider || "") === state.provider;
        row.classList.toggle("on", on);
        row.setAttribute("aria-pressed", on ? "true" : "false");
      }
    };

    const selectProvider = (name) => {
      state.provider = name;
      syncProviderRows();
      render();
    };

    // buildProviders reads the real provider names and counts once, after
    // the catalogue arrives. Filters do not change a provider's model count.
    const buildProviders = () => {
      if (!providerBox) return;
      providerBox.replaceChildren();
      const counts = new Map();
      for (const m of state.models) {
        const key = providerValue(m);
        if (!key) continue;
        const c = counts.get(key) || { total: 0, available: false };
        c.total += 1;
        c.available = c.available || !!m.available;
        counts.set(key, c);
      }
      if (state.provider && !counts.has(state.provider)) state.provider = "";

      const all = el("button", "provider-row");
      all.type = "button";
      all.dataset.provider = "";
      all.setAttribute("aria-pressed", "false");
      all.append(
        providerIcon("", true),
        el("span", "provider-name", t("All providers")),
        el("span", "provider-count", String(state.models.length)),
        el("span", "provider-dot" + (state.models.some((m) => m.available) ? " on" : " off"), ""),
      );
      all.addEventListener("click", () => selectProvider(""));
      providerBox.append(all);

      const names = [...counts.keys()].sort((a, b) => providerName(a).localeCompare(providerName(b), window.portalLangTag));
      for (const name of names) {
        const c = counts.get(name);
        const row = el("button", "provider-row");
        row.type = "button";
        row.dataset.provider = name;
        row.setAttribute("aria-pressed", "false");
        row.append(
          providerIcon(providerName(name), false),
          el("span", "provider-name", providerName(name)),
          el("span", "provider-count", String(c.total)),
          el("span", "provider-dot" + (c.available ? " on" : " off"), ""),
        );
        row.addEventListener("click", () => selectProvider(name));
        providerBox.append(row);
      }
      syncProviderRows();
    };

    const cardOf = (m) => {
      const card = el("article", "model-card");

      const top = el("div", "model-card-top");
      const icon = el("span", "model-icon");
      icon.setAttribute("aria-hidden", "true");
      const mark = brandMark(m);
      if (mark) { icon.classList.add("brand"); icon.append(mark); }
      else icon.textContent = initialOf(providerLabel(m));
      const title = el("div", "model-card-title");
      title.append(el("b", "", m.name || m.id));
      const copy = el("button", "model-copy");
      copy.type = "button";
      copy.setAttribute("aria-label", t("Copy model ID"));
      copy.title = t("Copy model ID");
      copy.append(copyIcon());
      copy.addEventListener("click", async (ev) => {
        ev.stopPropagation();
        const ok = await copyText(callId(m));
        copy.classList.toggle("copied", ok);
        copy.setAttribute("aria-label", ok ? t("Copied") : t("Copy failed"));
        setTimeout(() => {
          copy.classList.remove("copied");
          copy.setAttribute("aria-label", t("Copy model ID"));
        }, 1400);
      });
      top.append(icon, title, copy);
      card.append(top);

      const line = el("div", "model-state");
      line.append(
        el("span", "status-dot" + (m.available ? "" : " off"), ""),
        el("span", "", m.available ? t("Normal") : t("Temporarily unavailable")),
      );
      card.append(line);

      const spec = el("div", "model-spec");
      const head = el("div", "model-spec-head");
      head.append(el("b", "", t("Model information")), el("span", "pchip", protocolBadge(m)));
      spec.append(head);
      for (const [key, value] of [
        [t("Context"), contextLabel(m.context)],
        [t("Input images"), m.images ? t("Supported") : "—"],
        [t("Protocol"), protocolsText(m)],
        [t("Open state"), m.open ? t("Opened") : "—"],
      ]) {
        const row = el("div", "model-spec-row");
        row.append(el("span", "", key), el("span", "", value));
        spec.append(row);
      }
      card.append(spec);

      const bar = el("div", "model-availability");
      bar.setAttribute("aria-label", t("Availability") + ": " + (m.available ? t("Available") : t("Temporarily unavailable")));
      for (let i = 0; i < 18; i++) {
        bar.append(el("span", "model-avail-sq" + (m.available ? " on" : "")));
      }
      card.append(bar);

      const foot = el("div", "model-foot");
      foot.append(el("span", "model-avail-text", m.available ? t("Available") : t("Temporarily unavailable")));
      const more = el("button", "model-more", t("View details →"));
      more.type = "button";
      more.addEventListener("click", (ev) => { ev.stopPropagation(); openDetail(m); });
      foot.append(more);
      card.append(foot);

      card.addEventListener("click", (ev) => {
        if (ev.target.closest("button, a")) return;
        openDetail(m);
      });
      return card;
    };

    const render = () => {
      const all = state.models;
      const filtered = visible();
      list.replaceChildren();
      const any = filtered.length > 0;
      list.hidden = !any;
      empty.hidden = any;
      if (!any) {
        const noOpen = all.length === 0;
        emptyTitle.textContent = noOpen ? t("No models open yet") : t("No matching models");
        emptyText.textContent = noOpen
          ? t("When the administrator opens models, their names, providers, protocol support, availability and call examples appear here.")
          : t("Try another keyword, provider, protocol or availability filter.");
        emptyAction.hidden = !noOpen;
      }
      if (plazaProvider) plazaProvider.textContent = state.provider || t("All models");
      if (plazaCount) plazaCount.textContent = t("{n} models", { n: filtered.length });
      note.hidden = all.length === 0;
      note.textContent = all.length
        ? t("Only models open now are listed here. When the platform closes a model it disappears from the plaza at once, and existing keys cannot call it either.")
        : "";
      for (const m of filtered) list.append(cardOf(m));
      syncProviderRows();
    };

    const setProto = (m, proto) => {
      state.proto = proto;
      for (const b of detailTabs.querySelectorAll("button")) b.classList.toggle("on", b.dataset.proto === proto);
      detailCode.textContent = exampleFor(m, proto || "chat", state.gateway);
    };

    const openDetail = (m) => {
      if (!dialog) return;
      state.current = m;
      detailName.textContent = m.name || m.id;
      detailProvider.textContent = providerLabel(m);
      detailId.textContent = callId(m);
      detailBadges.replaceChildren();
      for (const p of (Array.isArray(m.protocols) ? m.protocols : [])) {
        detailBadges.append(el("span", "pchip " + protocolClass(p), protocolLabel(p)));
      }
      detailBadges.append(el("span", "tag " + (m.available ? "on" : "off"), m.available ? t("Available") : t("Temporarily unavailable")));
      detailDesc.hidden = !m.description;
      detailDesc.textContent = m.description || "";
      detailGrid.replaceChildren();
      const rows = [
        [t("Model ID"), callId(m)],
        [t("Provider"), m.provider || "—"],
        [t("Context"), contextLabel(m.context)],
        [t("Input images"), m.images ? t("Supported") : "—"],
        [t("Open state"), m.open ? t("Opened") : "—"],
      ];
      for (const [k, v] of rows) {
        const d = el("div");
        d.append(el("dt", "", k), el("dd", "", v));
        detailGrid.append(d);
      }
      const protos = Array.isArray(m.protocols) && m.protocols.length ? m.protocols : ["chat"];
      detailTabs.replaceChildren();
      for (const p of protos) {
        const b = el("button", "", protocolShort(p));
        b.type = "button";
        b.dataset.proto = p;
        b.addEventListener("click", () => setProto(m, p));
        detailTabs.append(b);
      }
      detailNote.textContent = t("Gateway: {url} · Use an API Key created in the console. Model IDs are as shown on this page.", { url: state.gateway });
      setProto(m, protos[0]);
      dialog.showModal();
    };

    const closeDetail = () => { if (dialog?.open) dialog.close(); };
    $("#detailClose")?.addEventListener("click", closeDetail);
    dialog?.addEventListener("click", (e) => { if (e.target === dialog) closeDetail(); });
    $("#detailCopy")?.addEventListener("click", async () => {
      const text = detailCode.textContent || "";
      if (!text) return;
      const ok = await copyText(text);
      const b = $("#detailCopy");
      b.textContent = ok ? t("Copied") : t("Copy failed");
      setTimeout(() => { b.textContent = t("Copy"); }, 1600);
    });

    for (const b of filters) {
      b.addEventListener("click", () => {
        state.filter = b.dataset.filter || "all";
        for (const x of filters) x.classList.toggle("on", x === b);
        render();
      });
    }
    for (const b of stateFilters) {
      b.addEventListener("click", () => {
        state.avail = b.dataset.avail || "all";
        for (const x of stateFilters) x.classList.toggle("on", x === b);
        render();
      });
    }
    search?.addEventListener("input", () => { state.query = search.value.trim().toLowerCase(); render(); });

    const load = async () => {
      clearError(error);
      loading.hidden = false;
      list.hidden = true;
      empty.hidden = true;
      const r = await api("/api/user/models");
      loading.hidden = true;
      if (r.status === 401) { location.replace("/portal/login?next=/portal/models"); return; }
      if (!r.ok) {
        showError(error, r.status === 0 ? t("Network error. Try again.") : t("Could not load the model catalogue. Try again later."));
        list.hidden = true;
        empty.hidden = true;
        if (plazaCount) plazaCount.textContent = "—";
        return;
      }
      state.models = Array.isArray(r.data?.models) ? r.data.models : [];
      if (typeof r.data?.gateway === "string" && /^https?:\/\//.test(r.data.gateway)) state.gateway = r.data.gateway;
      buildProviders();
      render();
    };
    load();
  };
})();
