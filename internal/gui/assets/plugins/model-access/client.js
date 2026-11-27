// One page for what the platform offers its users: open or close a provider
// model or a routing group, set the name and description the plaza shows,
// and whether the plaza offers routing groups alone. The same server rules
// feed the plaza and every user key, so nothing here is a page-local switch.
// It is loaded by the management page alone, after app.js: the page exists
// only there, and the core's budget keeps a page's own code out of it.
if (document.getElementById("view-model-access")) (() => {
  const list = $("#maList");
  const search = $("#maQ");
  const filter = $("#maFilter");
  const groupsOnlyBtn = $("#maGroupsOnly");
  const refresh = $("#maRefresh");
  const error = $("#maError");
  const state = { models: [], groupsOnly: false, filter: "all", query: "", edit: null };

  const clearError = () => { if (error) { error.hidden = true; error.textContent = ""; } };
  const showError = (why) => { if (error) { error.textContent = why; error.hidden = false; } };
  const kind = (m) => m.kind === "group" ? t("Routing group") : (m.provider || "—");
  const proto = (m) => (Array.isArray(m.protocols) ? m.protocols : []).join(" · ") || "—";
  const hay = (m) => [m.name, m.id, m.provider, m.kind === "group" ? t("Routing group") : ""].filter(Boolean).join(" ").toLowerCase();
  const visible = () => state.models.filter((m) => (state.filter === "all" || (state.filter === "open" ? m.open : !m.open))
    && (!state.query || hay(m).includes(state.query)));

  const paintGroupsOnly = () => {
    if (!groupsOnlyBtn) return;
    groupsOnlyBtn.textContent = t("Plaza: routing groups only") + ": " + t(state.groupsOnly ? "On" : "Off");
    groupsOnlyBtn.setAttribute("aria-pressed", String(state.groupsOnly));
    groupsOnlyBtn.setAttribute("aria-label", groupsOnlyBtn.textContent);
  };

  const replace = (m) => {
    const i = state.models.findIndex((x) => x.id === m.id);
    if (i >= 0) state.models[i] = m; else state.models.push(m);
  };

  const editRow = (m) => {
    const row = el("div", "row ma-row ma-edit");
    const fields = el("div", "ma-fields");
    const name = el("input", "sess-filter");
    name.type = "text";
    name.value = m.name || "";
    name.placeholder = t("Display name");
    const desc = el("input", "sess-filter");
    desc.type = "text";
    desc.value = m.description || "";
    desc.placeholder = t("Description users see");
    fields.append(name, desc);
    const acts = el("div", "ma-actions");
    const save = el("button", "text action", t("Save"));
    save.type = "button";
    const cancel = el("button", "text", t("Cancel"));
    cancel.type = "button";
    const done = async (saveIt) => {
      clearError();
      if (!saveIt) { state.edit = null; render(); return; }
      save.disabled = true;
      try {
        const r = await api("model-access", { id: m.id, name: name.value.trim(), description: desc.value.trim() });
        replace(r.model);
        state.edit = null;
        render();
        status(t("Saved"), "ok");
      } catch (e) { showError(e.message); save.disabled = false; }
    };
    save.onclick = () => done(true);
    cancel.onclick = () => done(false);
    name.onkeydown = (e) => { if (e.key === "Enter") done(true); };
    desc.onkeydown = (e) => { if (e.key === "Enter") done(true); };
    acts.append(save, cancel);
    row.append(fields, acts);
    return row;
  };

  const modelRow = (m) => {
    if (state.edit === m.id) return editRow(m);
    const row = el("div", "row ma-row");
    const who = el("div", "who");
    const callId = m.publicId || m.id;
    who.append(el("div", "name", m.name || m.id),
      el("div", "sub", [callId, kind(m), proto(m), m.available ? t("Available") : t("Unavailable")].join(" · ")));
    const acts = el("div", "ma-actions");
    acts.append(el("span", "status " + (m.open ? "ok" : ""), m.open ? t("Opened") : t("Closed")));
    const flip = el("button", "text action", m.open ? t("Close access") : t("Open access"));
    flip.type = "button";
    flip.onclick = async () => {
      clearError();
      flip.disabled = true;
      try {
        const r = await api("model-access", { id: m.id, open: !m.open });
        replace(r.model);
        render();
      } catch (e) { showError(e.message); flip.disabled = false; }
    };
    const edit = el("button", "text", t("Edit"));
    edit.type = "button";
    edit.onclick = () => { state.edit = m.id; render(); };
    acts.append(flip, edit);
    row.append(who, acts);
    return row;
  };

  const render = () => {
    if (!list) return;
    list.replaceChildren();
    const rows = visible();
    if (!rows.length) {
      const empty = el("div", "empty-state");
      empty.append(el("b", "", state.models.length ? t("No matching models") : t("No models or routing groups yet")),
        el("span", "", state.models.length ? t("Try another search or open state") : t("Add a provider in Providers, or make a routing group in Routing.")));
      list.append(empty);
      return;
    }
    for (const m of rows) list.append(modelRow(m));
  };

  const load = async () => {
    if (!list) return;
    clearError();
    try {
      const data = await api("model-access");
      state.models = Array.isArray(data?.models) ? data.models : [];
      state.groupsOnly = !!data?.groupsOnly;
      paintGroupsOnly();
      render();
    } catch (e) { showError(e.message); }
  };
  window.loadModelAccess = load;

  groupsOnlyBtn?.addEventListener("click", async () => {
    clearError();
    groupsOnlyBtn.disabled = true;
    try {
      const r = await api("admin/model-access/groups-only", { groupsOnly: !state.groupsOnly });
      state.groupsOnly = !!r.groupsOnly;
      paintGroupsOnly();
      status(t("Saved"), "ok");
    } catch (e) { showError(e.message); }
    groupsOnlyBtn.disabled = false;
  });
  refresh?.addEventListener("click", load);
  search?.addEventListener("input", () => { state.query = search.value.trim().toLowerCase(); render(); });
  for (const b of filter?.querySelectorAll(".opt") || []) {
    b.addEventListener("click", () => {
      state.filter = b.dataset.state || "all";
      for (const x of filter.querySelectorAll(".opt")) x.classList.toggle("on", x === b);
      render();
    });
  }
  load();
})();