// The administrator's model-open page: every model the Gateway serves,
// from /api/admin/model-access. Opening or closing one is the same server
// rule the user plaza and the Gateway read, so the answer here is the
// answer there. No model is invented and no change is local to the page.
(() => {
  "use strict";

  const $ = (sel, root = document) => root.querySelector(sel);
  const el = (tag, cls, text) => {
    const n = document.createElement(tag);
    if (cls) n.className = cls;
    if (text != null) n.textContent = text;
    return n;
  };
  const protocolShort = (p) => ({
    chat: "Chat",
    responses: "Responses",
    anthropic: "Anthropic",
    gemini: "Gemini",
  })[p] || p;
  const haystack = (m) => [m.name, m.id, m.provider, m.description, m.kind === "group" ? t("Routing group") : ""].filter(Boolean).join(" ").toLowerCase();
  // the card layout's field names (portal.css's ::before) read in the page's
  // language, from the label each cell carries
  const labelOf = (s) => t(s) + (window.portalLang === "en" ? ": " : "：");

  async function api(path, body) {
    let res;
    try {
      res = await fetch(path, {
        method: body === undefined ? "GET" : "POST",
        credentials: "same-origin",
        cache: "no-store",
        headers: body === undefined ? {} : { "Content-Type": "application/json" },
        body: body === undefined ? undefined : JSON.stringify(body),
      });
    } catch { return { ok: false, status: 0, data: null }; }
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
  function errorText(r) {
    if (r.status === 0) return t("Network error. Try again.");
    if (r.status === 401) return t("Your sign-in expired. Sign in again.");
    if (r.status === 403) return t("Needs administrator rights.");
    const why = String(r.data?.error || "");
    if (/doesn't serve that model/i.test(why)) return t("The Gateway does not serve this model now; refresh the catalogue.");
    if (/display name is 80/i.test(why)) return t("The name users see is 80 characters at most.");
    if (/description is 500/i.test(why)) return t("The description is 500 characters at most.");
    return why ? t("Save failed: {why}", { why }) : t("Save failed. Try again later.");
  }

  window.adminModels = function adminModels(user) {
    if (!user) { location.replace("/portal/login?next=/portal/admin-models"); return; }
    const list = $("#adminModelList");
    if (!list) return;
    if (user.role !== "admin") {
      showError($("#adminModelsError"), t("This page is for administrators."));
      return;
    }
    const loading = $("#adminModelsLoading");
    const empty = $("#adminModelsEmpty");
    const emptyTitle = $("#adminModelsEmptyTitle");
    const emptyText = $("#adminModelsEmptyText");
    const error = $("#adminModelsError");
    const total = $("#adminModelTotal");
    const openCount = $("#adminModelOpen");
    const closedCount = $("#adminModelClosed");
    const search = $("#adminModelSearch");
    const filters = [...document.querySelectorAll("#adminModelFilters .chip")];
    const groupsOnlyBtn = $("#adminGroupsOnly");
    const groupsOnlyLabel = $("#adminGroupsOnlyLabel");
    const editDialog = $("#adminModelEdit");
    const editForm = $("#adminModelEditForm");
    const editError = $("#adminEditError");
    const state = { models: [], filter: "all", query: "", edit: null, groupsOnly: false };

    const paintGroupsOnly = () => {
      if (!groupsOnlyBtn) return;
      groupsOnlyBtn.setAttribute("aria-checked", state.groupsOnly ? "true" : "false");
      if (groupsOnlyLabel) groupsOnlyLabel.textContent = state.groupsOnly ? t("On") : t("Off");
    };

    const filtered = () => state.models.filter((m) => {
      if (state.filter === "open" && !m.open) return false;
      if (state.filter === "closed" && m.open) return false;
      return !state.query || haystack(m).includes(state.query);
    });

    const renderSummary = () => {
      const open = state.models.filter((m) => m.open).length;
      total.textContent = String(state.models.length);
      openCount.textContent = String(open);
      closedCount.textContent = String(state.models.length - open);
    };

    const render = () => {
      const rows = filtered();
      list.replaceChildren();
      const any = rows.length > 0;
      list.hidden = !any;
      empty.hidden = any;
      if (!any) {
        const none = state.models.length === 0;
        emptyTitle.textContent = none ? t("No models to manage yet") : t("No matching models");
        emptyText.textContent = none
          ? t("Add and enable providers and models in the original admin page first; they appear here automatically.")
          : t("Try another keyword or open state.");
        const action = empty.querySelector("a.btn");
        if (action) action.hidden = !none;
      }
      if (any) {
        const head = el("div", "am-row head");
        for (const title of [t("Model"), t("Provider"), t("Protocol"), t("Service"), t("Opened")]) head.append(el("span", "", title));
        list.append(head);
        for (const m of rows) list.append(rowOf(m));
      }
      renderSummary();
    };

    const rowOf = (m) => {
      const row = el("div", "am-row");
      const main = el("div", "am-main");
      main.append(el("b", "", m.name || m.id), el("code", "", m.id));
      row.append(main);
      const provider = el("div", "am-provider", m.kind === "group" ? t("Routing group") : (m.provider || "—"));
      provider.dataset.label = labelOf("Provider");
      row.append(provider);
      const protos = el("div", "am-protos");
      protos.dataset.label = labelOf("Protocol");
      for (const p of (Array.isArray(m.protocols) ? m.protocols : [])) {
        if (protos.children.length) protos.append(el("span", "am-sep", "·"));
        protos.append(el("span", "am-proto", protocolShort(p)));
      }
      if (!protos.children.length) protos.append(el("span", "am-proto muted", t("Not declared")));
      row.append(protos);
      const service = el("div", "am-state");
      service.dataset.label = labelOf("Service");
      service.append(el("span", "am-avail " + (m.available ? "ok" : "off"), m.available ? t("Available") : t("Unavailable")));
      row.append(service);
      const side = el("div", "am-side");
      const toggle = el("button", "am-toggle");
      toggle.type = "button";
      toggle.setAttribute("role", "switch");
      toggle.setAttribute("aria-checked", m.open ? "true" : "false");
      toggle.setAttribute("aria-label", t(m.open ? "Close {name}" : "Open {name}", { name: m.name || m.id }));
      const label = el("span", "am-toggle-label", m.open ? t("Opened") : t("Closed"));
      toggle.addEventListener("click", () => flip(m, toggle, label));
      const edit = el("button", "linkish am-edit-link", t("Edit"));
      edit.type = "button";
      edit.addEventListener("click", () => openEdit(m));
      side.append(toggle, label, edit);
      row.append(side);
      return row;
    };

    const replace = (m) => {
      const i = state.models.findIndex((x) => x.id === m.id);
      if (i >= 0) state.models[i] = m;
      else state.models.push(m);
    };

    const flip = async (m, button, label) => {
      clearError(error);
      const want = !m.open;
      button.disabled = true;
      const r = await api("/api/admin/model-access", { id: m.id, open: want });
      button.disabled = false;
      if (r.status === 401) { location.replace("/portal/login?next=/portal/admin-models"); return; }
      if (!r.ok) { showError(error, errorText(r)); return; }
      replace(r.data.model);
      render();
    };

    const openEdit = (m) => {
      if (!editDialog) return;
      state.edit = m;
      clearError(editError);
      $("#adminEditId").textContent = m.id;
      editForm.elements.name.value = m.name || "";
      editForm.elements.description.value = m.description || "";
      editForm.elements.open.checked = !!m.open;
      editDialog.showModal();
      editForm.elements.name.focus();
      editForm.elements.name.select();
    };

    const closeEdit = () => { if (editDialog?.open) editDialog.close(); };
    $("#adminEditClose")?.addEventListener("click", closeEdit);
    $("#adminEditCancel")?.addEventListener("click", closeEdit);
    editDialog?.addEventListener("click", (e) => { if (e.target === editDialog) closeEdit(); });
    editForm?.addEventListener("submit", async (ev) => {
      ev.preventDefault();
      const m = state.edit;
      if (!m) return;
      clearError(editError);
      const name = editForm.elements.name.value.trim();
      const description = editForm.elements.description.value.trim();
      const open = editForm.elements.open.checked;
      const save = editForm.querySelector("button[type=submit]");
      save.disabled = true;
      save.textContent = t("Saving…");
      const r = await api("/api/admin/model-access", { id: m.id, open, name, description });
      save.disabled = false;
      save.textContent = t("Save");
      if (r.status === 401) { location.replace("/portal/login?next=/portal/admin-models"); return; }
      if (!r.ok) { showError(editError, errorText(r)); return; }
      replace(r.data.model);
      closeEdit();
      render();
    });

    for (const b of filters) {
      b.addEventListener("click", () => {
        state.filter = b.dataset.state || "all";
        for (const x of filters) x.classList.toggle("on", x === b);
        render();
      });
    }
    search?.addEventListener("input", () => { state.query = search.value.trim().toLowerCase(); render(); });
    $("#adminModelsRefresh")?.addEventListener("click", () => load());

    const load = async () => {
      clearError(error);
      loading.hidden = false;
      list.hidden = true;
      empty.hidden = true;
      const r = await api("/api/admin/model-access");
      loading.hidden = true;
      if (r.status === 401) { location.replace("/portal/login?next=/portal/admin-models"); return; }
      if (!r.ok) {
        showError(error, r.status === 403 ? t("Needs administrator rights.") : errorText(r));
        total.textContent = openCount.textContent = closedCount.textContent = "—";
        return;
      }
      state.models = Array.isArray(r.data?.models) ? r.data.models : [];
      state.groupsOnly = !!r.data?.groupsOnly;
      paintGroupsOnly();
      render();
    };
    groupsOnlyBtn?.addEventListener("click", async () => {
      clearError(error);
      groupsOnlyBtn.disabled = true;
      const want = !state.groupsOnly;
      const r = await api("/api/admin/model-access/groups-only", { groupsOnly: want });
      groupsOnlyBtn.disabled = false;
      if (r.status === 401) { location.replace("/portal/login?next=/portal/admin-models"); return; }
      if (!r.ok) { showError(error, errorText(r)); return; }
      state.groupsOnly = !!r.data?.groupsOnly;
      paintGroupsOnly();
    });
    load();
  };
})();