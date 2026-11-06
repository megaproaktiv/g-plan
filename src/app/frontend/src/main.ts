import "./style.css";
import { backend, whenReady } from "./backend";
import { Gantt } from "./gantt";
import { actualEffortHTML, externalIdState, openMonitor } from "./monitor";
import { glossaryHTML } from "./glossary";
import { helpPageHTML } from "./helppage";
import { syncAfterChat } from "./chatflow";
import { batchSummary } from "./batch";
import { errorText } from "./errortext";
import { contextMenuItems } from "./ctxmenu";
import { effortHTML } from "./effort";
import { workdayCount } from "./workdays";
import { ratingLabel } from "./progress";
import type { Project, Zoom } from "./types";

let project: Project | null = null;
let gantt: Gantt;

const app = document.querySelector<HTMLDivElement>("#app")!;
app.innerHTML = `
  <div class="toolbar">
    <h1>pt-plan</h1>
    <button id="details">Project…</button>
    <button id="new">New</button>
    <button id="open">Open</button>
    <button id="save" class="primary" title="Save to the current file (⌘S)">Save</button>
    <button id="saveas" title="Save to a new location…">Save As…</button>
    <button id="pdf">Report…</button>
    <button id="baselines">Baselines</button>
    <button id="monitor" title="Compare with real effort booked in an external system">Monitor</button>
    <button id="link">Link (L)</button>
    <span class="spacer"></span>
    <label>Unit
      <select id="unit"><option value="days">days</option><option value="hours">hours</option></select>
    </label>
    <label>h/day <input id="hpd" type="number" min="1" max="24" style="width:52px" /></label>
    <label>Zoom
      <select id="zoom">
        <option value="days">days</option><option value="weeks">weeks</option>
        <option value="months" selected>months</option><option value="year">year</option>
        <option value="window">Window (fit)</option>
      </select>
    </label>
    <span class="total" id="total"></span>
  </div>
  <div class="main">
    <div class="canvas-wrap" id="canvas"></div>
    <div class="side" id="side"></div>
  </div>
  <div class="chat">
    <label>Mode
      <select id="mode"><option value="action">action</option><option value="role">role</option><option value="section">section</option><option value="milestone">milestone</option></select>
    </label>
    <input id="chatin" autocomplete="off" autocorrect="off" autocapitalize="off" spellcheck="false" placeholder='command, e.g.  add action Demo   |   move 3d   |   type "help"' />
    <textarea id="batchin" class="hidden" rows="5" spellcheck="false" autocomplete="off" autocorrect="off" autocapitalize="off"
      placeholder="Batch mode: one command per line. Lines starting with # are comments. Stops at the first error or at 'quit'. Run with the button or Cmd+Enter."></textarea>
    <button id="batchbtn" title="Batch mode: paste several commands and run them one after another" aria-pressed="false" aria-label="Batch mode">🧩</button>
    <button id="batchrun" class="hidden" title="Run all commands (Cmd+Enter)">Run batch</button>
    <button id="helpbtn">help</button>
    <span class="log" id="chatlog"></span>
  </div>
  <div class="drill hidden" id="drill"></div>
  <div class="drill hidden" id="helpbox"></div>
  <div class="drill hidden" id="projbox"></div>
  <div class="drill hidden" id="baselinebox"></div>
`;

let savedStatus: string | null = null; // the status line text while a drag shows its own
gantt = new Gantt(document.querySelector("#canvas")!, {
  onMove: async (id, s, e) => {
    project = await backend.moveWorkPackage(id, s, e);
    refresh();
    // keep the right-panel start/end/duration in sync after a drag/resize
    if (selectedActionId === id) renderSide();
  },
  onOpen: (id) => openActionEdit(id),
  onLink: async (from, to, fa, ta) => {
    // Derive the relationship kind from the drawn anchors. l=start edge,
    // r=end edge, t/b=middle. A = source (from), B = target (to). Milestones
    // are zero-duration points but follow the SAME before/after rules via the
    // action's edge anchor (e.g. action end -> milestone = FS, action start ->
    // milestone = SF). Phases-row / project Start-End milestones are rejected
    // by the backend (full-height markers are not link endpoints).
    const mid = (x: string) => x === "t" || x === "b";
    let kind: string;
    if (mid(fa)) kind = mid(ta) ? "MM" : (ta === "r" ? "ME" : "MS");
    else if (fa === "l" && ta === "l") kind = "SS";
    else if (fa === "r" && ta === "r") kind = "FF";
    else if (fa === "l" && ta === "r") kind = "SF";
    else kind = "FS";
    try { project = await backend.addDependency(from, to, kind, fa, ta); refresh(); }
    catch (err) { alert(String(err)); }
  },
  onContextMenu: (x, y, target) => openContextMenu(x, y, target),
  onReassign: async (id, cluster) => {
    try { project = await backend.setActionCluster(id, cluster); refresh(); }
    catch (err) { alert(String(err)); }
  },
  // While something is dragged the status line shows the state it is moving to
  // (the start first); the previous text comes back when the drag ends.
  onDragStatus: (text) => {
    const el = document.querySelector<HTMLElement>("#total")!;
    if (text !== null) {
      if (savedStatus === null) savedStatus = el.textContent ?? "";
      el.textContent = text;
      el.classList.add("dragging");
    } else if (savedStatus !== null) {
      el.textContent = savedStatus;
      savedStatus = null;
      el.classList.remove("dragging");
    }
  },
  onSelect: (kind, id) => {
    if (kind === "action") { gantt.setSelected(id); selectAction(id); }
    else selectSection(id);
  },
  onDrillChange: (wpId) => {
    // keep the command bar in sync: `add ...` in the chat targets a detail of
    // the drilled action while a drill plane is open.
    backend.setDrill(wpId ?? "").catch(() => {});
    const el = document.querySelector("#total") as HTMLElement;
    if (wpId && project) {
      const wp = project.packages.find((w) => w.id === wpId);
      if (wp) el.textContent = `Drill-in: ${wp.displayId} ${wp.name}  (U = up, "add detail <name>" adds a sub-action here)`;
    } else refresh();
  },
  onDetailMove: async (wpId, detId, s, e) => {
    try { project = await backend.moveDetail(wpId, detId, s, e); gantt.setProject(project); updateSaveButtonState(); }
    catch (err) { alert(String(err)); }
  },
  onDetailOpen: (wpId, detId) => openDetailEditor(wpId, detId),
  onDetailSelect: (wpId, detId) => selectDetail(wpId, detId),
  onMilestoneSelect: (id) => selectMilestone(id),
  onMilestoneMove: async (id, date) => {
    try { project = await backend.moveMilestone(id, date); gantt.setProject(project); updateSaveButtonState(); if (selectedMilestone === id) renderSide(); }
    catch (err) { alert(String(err)); }
  },
  onDetailAdd: async (wpId) => {
    const wp = project!.packages.find((w) => w.id === wpId)!;
    try { project = await backend.addDetail(wpId, "New detail", wp.start, wp.end, "0", "", ""); gantt.setProject(project); updateSaveButtonState(); }
    catch (err) { alert(String(err)); }
  },
  onDrillIcon: (wpId) => {
    // Clicking the drilled icon opens ONLY the drill-down plane for that
    // action. It must NOT also open the detail editor/table (regression: it
    // used to call openDrill() as well).
    gantt.drillInto(wpId);
  },
});

function refresh() {
  if (!project) return;
  // restore the persisted zoom (saved with the project) when it differs
  if (project.zoom && project.zoom !== gantt.getZoom()) {
    gantt.setZoom(project.zoom as Zoom);
    const zsel = document.querySelector("#zoom") as HTMLSelectElement | null;
    if (zsel) zsel.value = project.zoom;
  }
  gantt.setProject(project);
  document.title = "pt-plan — " + (project.title || "Untitled");
  (document.querySelector("#total") as HTMLElement).textContent =
    `${project.title || "Untitled"} · ${project.start || "?"}–${project.end || "?"} · Total ${project.totalEst} ${project.unit}`
    + ` · Mode: ${currentMode}` + ` · View: ${gantt.getZoom()}`;
  (document.querySelector("#unit") as HTMLSelectElement).value = project.unit;
  (document.querySelector("#hpd") as HTMLInputElement).value = String(project.hoursPerDay || 8);
  renderSide();
  updateSaveButtonState();
}

// updateSaveButtonState greys out the quick-save button when there are no
// unsaved changes, and enables it when the project is dirty. Save As… stays
// always active. The dirty check is async, so the button updates shortly after
// each refresh.
function updateSaveButtonState() {
  const btn = document.querySelector<HTMLButtonElement>("#save");
  if (!btn) return;
  backend.isDirty().then((dirty) => {
    btn.disabled = !dirty;
    btn.title = dirty ? "Save to the current file (⌘S)" : "No unsaved changes";
  }).catch(() => { btn.disabled = false; });
}

// --- Autosave timer ---------------------------------------------------------
// A single interval ticks every 30s. On each tick, if enough time has elapsed
// for the project's autosave interval (minutes; 0 = every change, debounced to
// the next tick), it calls backend.autosaveTick(), which saves ONLY when the
// project is dirty AND already has a path. The backend owns the dirty/path
// gate; the frontend only decides WHEN to poll based on the configured minutes.
let autosaveLastRun = Date.now();
function setupAutosave() {
  window.setInterval(async () => {
    if (!project) return;
    if (!project.autosaveEnabled) return; // autosave turned off in settings
    const minutes = project.autosaveMinutes ?? 5;
    const intervalMs = minutes <= 0 ? 0 : minutes * 60_000;
    if (Date.now() - autosaveLastRun < intervalMs) return;
    autosaveLastRun = Date.now();
    try {
      const saved = await backend.autosaveTick();
      if (saved) { project = await backend.getProject(); updateSaveButtonState(); }
    } catch { /* autosave is best-effort; ignore errors */ }
  }, 30_000);
}

const roleFilter = new Set<string>(); // selected roles; empty = all
let selectedActionId: string | null = null;
let selectedSectionId: string | null = null; // section shown in the right pane
let selectedDetail: { wp: string; det: string } | null = null; // detail shown in the right pane
let selectedMilestone: string | null = null; // milestone shown in the right pane
let actionEditMode = false; // false = read-only "show" dialog; true = editable form
let selectedRoleName: string | null = null;
let currentMode = "action"; // mirrors the #mode selector; drives the context panel

// selectAction: show the READ-ONLY action dialog on the right (Edit to modify).
// Selecting an action also moves the view to "action" (the object-type icon
// follows the selection) and draws a selection border on the canvas.
function selectAction(id: string) {
  selectedActionId = id;
  selectedSectionId = null;
  selectedDetail = null;
  actionEditMode = false;
  currentMode = "action";
  gantt.setSelected(id);
  backend.select("action", id).catch(() => {});
  const sel = document.querySelector("#mode") as HTMLSelectElement | null;
  if (sel) sel.value = "action";
  renderSide();
}
// selectSection: show the section dialog in the right #side context pane.
function selectSection(id: string) {
  selectedSectionId = id;
  selectedActionId = null;
  selectedDetail = null;
  currentMode = "section";
  gantt.setSelectedSection(id);
  backend.select("section", id).catch(() => {});
  const sel = document.querySelector("#mode") as HTMLSelectElement | null;
  if (sel) sel.value = "section";
  renderSide();
}
// selectDetail: show the detail (sub-action) editor in the right #side pane.
function selectDetail(wpId: string, detId: string) {
  selectedDetail = { wp: wpId, det: detId };
  selectedActionId = null;
  selectedSectionId = null;
  gantt.setSelectedDetail(detId);
  renderSide();
}
// selectMilestone: show the milestone editor; moves the view to milestone mode.
function selectMilestone(id: string) {
  selectedMilestone = id;
  selectedActionId = null; selectedSectionId = null; selectedDetail = null;
  currentMode = "milestone";
  gantt.setSelectedMilestone(id);
  backend.select("milestone", id).catch(() => {});
  const sel = document.querySelector("#mode") as HTMLSelectElement | null;
  if (sel) sel.value = "milestone";
  renderSide();
}

// renderMilestonePanel: name, date, color, delete for the selected milestone.
function renderMilestonePanel(side: HTMLDivElement, id: string) {
  const m = project!.milestones.find((x) => x.id === id);
  if (!m) { selectedMilestone = null; side.innerHTML = ""; return; }
  const locked = m.id === "m_start" || m.id === "m_end";
  side.innerHTML = `<h2>◆ Milestone ${m.displayId ? m.displayId + " — " : ""}${esc(m.name)}</h2>
    <div class="editor">
      <label>Name<br/><input id="ms_name" value="${esc(m.name)}" ${locked ? "disabled" : ""}/></label>
      <label>Date <input id="ms_date" type="date" value="${m.date}" /></label>
      <label>Color <input id="ms_color" type="color" value="${m.color || "#b44"}" /></label>
      <div>Section: ${esc(m.cluster || "(none)")}</div>
      ${m.cluster ? `<div>Section order <button id="ms_secup">▲ up</button> <button id="ms_secdown">▼ down</button></div>` : ""}
      <div class="erow">
        <button id="ms_save" class="primary">Save</button>
        ${locked ? "" : `<button id="ms_del">Delete</button>`}
        <button id="ms_close">Close</button>
      </div>
      ${locked ? `<div class="muted">This is a project Start/End milestone.</div>` : ""}
    </div>` + linksTableHTML(id);
  side.querySelector("#ms_save")!.addEventListener("click", async () => {
    const name = (side.querySelector("#ms_name") as HTMLInputElement).value;
    const date = (side.querySelector("#ms_date") as HTMLInputElement).value;
    const color = (side.querySelector("#ms_color") as HTMLInputElement).value;
    try { project = await backend.setMilestoneFields(id, name, date, color); refresh(); }
    catch (err) { alert(String(err)); }
  });
  side.querySelector("#ms_del")?.addEventListener("click", async () => {
    try { project = await backend.removeMilestone(id); selectedMilestone = null; gantt.clearSelection(); refresh(); }
    catch (err) { alert(String(err)); }
  });
  side.querySelector("#ms_close")!.addEventListener("click", () => { selectedMilestone = null; renderSide(); });
  // Section order up/down (moves the milestone's whole section/swimlane).
  const msCluster = m.cluster || "";
  if (msCluster) {
    side.querySelector("#ms_secup")?.addEventListener("click", async () => {
      try { project = await backend.moveSection(msCluster, -1); refresh(); }
      catch (err) { alert(String(err)); }
    });
    side.querySelector("#ms_secdown")?.addEventListener("click", async () => {
      try { project = await backend.moveSection(msCluster, 1); refresh(); }
      catch (err) { alert(String(err)); }
    });
  }
  wireLinksTable(side);
}
// openActionEdit: show the EDITABLE action form (double-click or Edit button).
function openActionEdit(id: string) {
  selectedActionId = id;
  selectedSectionId = null;
  selectedDetail = null;
  actionEditMode = true;
  renderSide();
  focusEditorName();
}

// focus the inline action/role name input after a re-render
function focusEditorName() {
  setTimeout(() => {
    const el = document.querySelector<HTMLInputElement>("#e_name") || document.querySelector<HTMLInputElement>("#re_name");
    if (el) { el.focus(); el.select(); }
  }, 0);
}

async function renderSide() {
  const side = document.querySelector<HTMLDivElement>("#side")!;
  if (!project) return;
  renderSideBody(side);
  // Prepend the view/mode switcher (icons: action, role, section — same order
  // as the toolbar/chat mode) so it sits at the top of the right pane. It is
  // DISABLED while an action is in edit mode (you must leave edit mode first).
  const editing = !!(selectedActionId && actionEditMode);
  side.insertAdjacentHTML("afterbegin", modeSwitcherHTML(editing));
  wireModeSwitcher(side, editing);
}

// modeSwitcherHTML renders three icon buttons for the view mode: action, role,
// section (in that order). The current mode is highlighted. When `locked`
// (edit mode) the buttons are disabled.
function modeSwitcherHTML(locked: boolean): string {
  const btn = (mode: string, icon: string, label: string) => {
    const active = currentMode === mode ? " active" : "";
    const dis = locked ? " disabled" : "";
    return `<button class="modeico${active}" data-mode="${mode}" title="${label}"${dis}>${icon}</button>`;
  };
  return `<div class="modeswitch">
    ${btn("action", "▭", "Action view")}
    ${btn("role", "☺", "Role view")}
    ${btn("section", "≣", "Section view")}
    ${btn("milestone", "◆", "Milestone view")}
    ${locked ? `<span class="muted" style="font-size:11px;margin-left:6px">leave edit to switch</span>` : ""}
  </div>`;
}

function wireModeSwitcher(side: HTMLElement, locked: boolean) {
  if (locked) return; // disabled in edit mode
  side.querySelectorAll<HTMLButtonElement>(".modeico").forEach((b) =>
    b.addEventListener("click", () => {
      const m = b.dataset.mode!;
      if (m === currentMode) return;
      // Switching the view does NOT clear the current selection: if an action
      // (or section) is selected it stays selected (and its border remains).
      // Only leave edit mode.
      actionEditMode = false;
      setModeUI(m);
    }));
}

// renderSideBody renders the context panel body (without the mode switcher).
// The body follows the current VIEW mode:
//   role    -> the roles list + load
//   section -> the selected section's panel (or a hint)
//   action  -> the selected action (or detail) panel (or a hint)
// A selected action/section keeps its canvas border across view switches; the
// panel shown here reflects the chosen view, not the selection.
function renderSideBody(side: HTMLDivElement) {
  if (!project) return;
  if (currentMode === "role") {
    renderRolesPanel(side);
    return;
  }
  if (currentMode === "milestone") {
    if (selectedMilestone && project.milestones.some((m) => m.id === selectedMilestone)) {
      renderMilestonePanel(side, selectedMilestone);
    } else {
      side.innerHTML = `<div class="muted" style="padding:12px">Click a milestone ◆ on the canvas to edit it, or type <b>milestone &lt;name&gt;</b> in the command bar.</div>`;
    }
    return;
  }
  if (currentMode === "section") {
    if (selectedSectionId && (project.sectionColors[selectedSectionId] !== undefined ||
        project.packages.some((w) => (w.cluster || "") === selectedSectionId))) {
      renderSectionPanel(side, selectedSectionId);
    } else {
      side.innerHTML = `<div class="muted" style="padding:12px">Click a section band to edit it.</div>` + linksTableHTML();
      wireLinksTable(side);
    }
    return;
  }
  // action view (default)
  if (selectedDetail) {
    renderDetailPanel(side, selectedDetail.wp, selectedDetail.det);
    return;
  }
  if (selectedActionId && project.packages.some((w) => w.id === selectedActionId)) {
    const wp = project.packages.find((w) => w.id === selectedActionId)!;
    side.innerHTML = (actionEditMode ? actionEditorHTML(wp) : actionShowHTML(wp)) + linksTableHTML();
    if (actionEditMode) wireActionEditor(selectedActionId);
    else wireActionShow(selectedActionId);
    wireLinksTable(side);
    return;
  }
  // a section is selected -> show the section dialog in the right pane
  if (selectedSectionId) {
    renderSectionPanel(side, selectedSectionId);
    return;
  }

  // nothing selected: show a hint plus the dependency/link table.
  side.innerHTML = `<div class="muted" style="padding:12px">
    Click an action or a section to see its details here.<br/>
    Switch Mode to <b>role</b> to manage roles.</div>` + linksTableHTML();
  wireLinksTable(side);
}

// linksTableHTML renders the right-side dependency table: one row per link
// showing source → target (by display id + name), the link kind (FS/SS/FF/SF),
// a red flag when the timing rule is violated, and a × delete control. When
// `onlyId` is given, only links that touch that endpoint are shown (used by the
// milestone panel to show just that milestone's links).
function linksTableHTML(onlyId?: string): string {
  let deps = project?.deps ?? [];
  if (onlyId) deps = deps.filter((d) => d.from === onlyId || d.to === onlyId);
  const label = (id: string): string => {
    const wp = project!.packages.find((w) => w.id === id);
    if (wp) return `${wp.displayId} ${wp.name}`;
    for (const w of project!.packages) {
      const d = w.details?.find((x) => x.id === id);
      if (d) return `${d.displayId} ${d.name}`;
    }
    const ms = project!.milestones.find((x) => x.id === id);
    if (ms) return `◆ ${ms.displayId ? ms.displayId + " " : ""}${ms.name}`;
    return id;
  };
  if (!deps.length) return `<h2>Links</h2><div class="muted" style="padding:0 12px">No dependencies yet.</div>`;
  const rows = deps.map((d) => `
    <tr${d.violated ? ' style="color:#e03131"' : ""}>
      <td>${esc(label(d.from))}</td>
      <td style="text-align:center"><b>${esc(kindAbbr(d.kind || "FS"))}</b></td>
      <td>${esc(label(d.to))}</td>
      <td><button class="deldep" data-from="${esc(d.from)}" data-to="${esc(d.to)}" title="delete link">×</button></td>
    </tr>`).join("");
  return `<h2>Links</h2>
    <table class="links"><thead><tr><th>From</th><th>Kind</th><th>To</th><th></th></tr></thead>
    <tbody>${rows}</tbody></table>`;
}

// kindAbbr maps a canonical dependency kind to the abbreviation shown in the
// current GUI language (English FS.. / German EA..). Mirrors the backend.
function kindAbbr(kind: string): string {
  const de = (project?.language || "en") === "de";
  if (de) {
    const m: Record<string, string> = { FS: "EA", SS: "AA", FF: "EE", SF: "AE", MS: "MA", ME: "ME", MM: "MM", OV: "MM" };
    return m[kind] || kind;
  }
  return kind === "OV" ? "MM" : kind;
}

// wireLinksTable attaches delete handlers to the × buttons in the link table.
function wireLinksTable(root: HTMLElement) {
  root.querySelectorAll<HTMLButtonElement>(".deldep").forEach((b) =>
    b.addEventListener("click", async () => {
      try { project = await backend.removeDependency(b.dataset.from!, b.dataset.to!); refresh(); }
      catch (err) { alert(String(err)); }
    }));
}

// renderRolesPanel renders the roles-only right panel (role mode).
async function renderRolesPanel(side: HTMLDivElement) {
  const allOn = roleFilter.size === 0;
  let html = `<h2>Roles</h2>
    <label class="rolerow"><input type="checkbox" id="showroleactions" ${project!.showRoleActions ? "checked" : ""}/> <b>Show role actions</b></label>
    <label class="rolerow" style="margin-left:18px"><input type="checkbox" id="showrolepercent" ${project!.showRolePercent ? "checked" : ""} ${project!.showRoleActions ? "" : "disabled"}/> Show percent</label>
    <label class="rolerow" style="margin-left:18px"><input type="checkbox" id="showroledays" ${project!.showRoleDays ? "checked" : ""} ${project!.showRoleActions ? "" : "disabled"}/> Show days</label>
    <label class="rolerow"><input type="checkbox" id="allroles" ${allOn ? "checked" : ""}/> <b>All roles</b></label>`;
  for (const r of project!.roles) {
    const checked = allOn || roleFilter.has(r.name);
    const editing = selectedRoleName === r.name;
    html += `<div class="rolerow">
      <input type="checkbox" class="rolechk" data-role="${esc(r.name)}" ${checked ? "checked" : ""}/>
      <span class="rswatch" style="display:inline-block;width:12px;height:12px;border-radius:2px;background:${r.color || "#999"};margin-right:4px"></span>
      <span class="rname rolename" data-role="${esc(r.name)}" title="click to edit">${esc(r.name)}</span>
      <span class="rload">${r.totalEst} ${project!.unit}</span>
      <button class="rolefilter" data-role="${esc(r.name)}" title="show only this role">filter</button>
    </div>`;
    if (editing) {
      html += `<div class="editor roleedit">
        <div id="re_id" class="muted">Role ID: <b>R${r.id}</b> <span title="stable number; use it in commands such as: assign R${r.id} to ...">(stays the same when you rename the role)</span></div>
        <label>Name<br/><input id="re_name" value="${esc(r.name)}" /></label>
        <label>Description<br/><textarea id="re_desc" rows="2">${esc(r.description)}</textarea></label>
        <label>Color <input id="re_color" type="color" value="${r.color || "#1f77b4"}" /></label>
        <div class="erow"><button id="re_save" class="primary">Save</button>
          <button id="re_del" ${r.inUse ? 'disabled aria-describedby="re_inuse"' : ""}>Delete</button><button id="re_close">Close</button></div>
        ${r.inUse ? `<div id="re_inuse" class="warn">Cannot be deleted while it is in use: ${esc(r.usedBy)}. Remove these references first.</div>` : ""}
      </div>`;
    }
  }
  html += `<button id="addrole">+ role</button>`;
  html += `<h2>Role load (filtered)</h2><div id="loadout"></div>`;
  if (project!.warnings.length) {
    html += `<h2>Warnings</h2>` + project!.warnings.map((w) => `<div class="warn">${esc(w)}</div>`).join("");
  }
  side.innerHTML = html;

  side.querySelector("#showroleactions")!.addEventListener("change", async (e) => {
    const on = (e.target as HTMLInputElement).checked;
    project = await backend.setViewOptions(project!.showDetails, on, project!.showPhases, project!.showProgress);
    renderSide(); // re-render so the Show percent/days checkboxes enable/disable
    refresh();
  });
  side.querySelector("#showrolepercent")!.addEventListener("change", async (e) => {
    const on = (e.target as HTMLInputElement).checked;
    project = await backend.setRoleLabelOptions(on, project!.showRoleDays);
    refresh();
  });
  side.querySelector("#showroledays")!.addEventListener("change", async (e) => {
    const on = (e.target as HTMLInputElement).checked;
    project = await backend.setRoleLabelOptions(project!.showRolePercent, on);
    refresh();
  });

  side.querySelector("#allroles")!.addEventListener("change", (e) => {
    // "All roles" => clear the filter (empty set means show every role).
    if ((e.target as HTMLInputElement).checked) roleFilter.clear();
    renderSide();
  });
  // Per-role checkbox: toggles whether that role is INCLUDED in the filter.
  // roleFilter holds the set of roles to SHOW; empty = all. Checking several
  // boxes shows several roles at once (multi-select).
  side.querySelectorAll<HTMLInputElement>(".rolechk").forEach((c) =>
    c.addEventListener("change", () => {
      const name = c.dataset.role!;
      const allOnNow = roleFilter.size === 0;
      if (allOnNow) {
        // transition from "all" to an explicit selection: seed with every role
        // then remove/keep based on this toggle so the others stay visible.
        project!.roles.forEach((r) => roleFilter.add(r.name));
      }
      if (c.checked) roleFilter.add(name); else roleFilter.delete(name);
      // if the user unchecked everything, fall back to "all"
      if (roleFilter.size === project!.roles.length) roleFilter.clear();
      renderSide();
    }));
  // The per-role "filter" button narrows to ONLY that role (quick single
  // filter). Use the checkboxes to build a multi-role selection.
  side.querySelectorAll<HTMLButtonElement>(".rolefilter").forEach((b) =>
    b.addEventListener("click", () => { roleFilter.clear(); roleFilter.add(b.dataset.role!); renderSide(); }));
  side.querySelectorAll<HTMLElement>(".rolename").forEach((el) =>
    el.addEventListener("click", () => { selectedRoleName = el.dataset.role!; renderSide(); focusEditorName(); }));
  const re = (s: string) => side.querySelector(s);
  re("#re_save")?.addEventListener("click", async () => {
    const nn = (side.querySelector("#re_name") as HTMLInputElement).value;
    const dd = (side.querySelector("#re_desc") as HTMLTextAreaElement).value;
    const col = (side.querySelector("#re_color") as HTMLInputElement | null)?.value || "";
    try {
      const old = selectedRoleName!;
      project = await backend.setRoleFields(old, nn, dd);
      const finalName = nn.trim() || old;
      // the role filter is kept by name: carry it over to the new name
      if (finalName !== old && roleFilter.delete(old)) roleFilter.add(finalName);
      if (col) project = await backend.setRoleColor(finalName, col);
      selectedRoleName = finalName; refresh();
    } catch (err) { alert(String(err)); }
  });
  re("#re_del")?.addEventListener("click", async () => {
    try {
      project = await backend.removeRole(selectedRoleName!);
      roleFilter.delete(selectedRoleName!);
      selectedRoleName = null; refresh();
    } catch (err) { alert(String(err)); }
  });
  re("#re_close")?.addEventListener("click", () => { selectedRoleName = null; renderSide(); });
  side.querySelector("#addrole")?.addEventListener("click", async () => {
    // add a role with a default name, then open its editor focused on the name
    const n = project!.roles.length + 1;
    const name = "Role " + n;
    project = await backend.addRole(name, "");
    selectedRoleName = name;
    refresh();
    focusEditorName();
  });
  applyRoleFilter();
  renderRoleLoad();
}

// applyRoleFilter: when roles are selected, the canvas shows ONLY actions that
// have at least one of the selected roles attached. "All roles" => show all.
function applyRoleFilter() {
  if (!project) return;
  if (roleFilter.size === 0) { gantt.setVisible(null); return; }
  const ids = new Set<string>();
  for (const r of project.roles) {
    if (!roleFilter.has(r.name)) continue;
    for (const a of r.assignments) ids.add(a.wp);
  }
  gantt.setVisible(ids);
}

async function renderRoleLoad() {
  const out = document.querySelector<HTMLDivElement>("#loadout");
  if (!out || !project) return;
  const roles = roleFilter.size === 0 ? project.roles.map((r) => r.name) : [...roleFilter];
  const months: Record<string, number> = {};
  const perRole: Record<string, Record<string, string>> = {};
  for (const name of roles) {
    try {
      const load = await backend.roleLoad(name);
      perRole[name] = load;
      Object.keys(load).forEach((m) => { months[m] = 1; });
    } catch { /* ignore */ }
  }
  const cols = Object.keys(months).sort();
  if (cols.length === 0) { out.innerHTML = `<div class="muted">no load</div>`; return; }
  let t = `<table class="loadtbl"><thead><tr><th>Role</th>${cols.map((c) => `<th>${c}</th>`).join("")}</tr></thead><tbody>`;
  for (const name of roles) {
    t += `<tr><td>${esc(name)}</td>${cols.map((c) => `<td>${perRole[name]?.[c] ?? ""}</td>`).join("")}</tr>`;
  }
  t += `</tbody></table>`;
  out.innerHTML = t;
}

// Section dialog rendered INLINE in the right #side pane (BUG fix: it used to
// be a separate floating window). Shows name, action count, order up/down,
// color, rename. The top "Phases" section is locked (it is the Phases view,
// not a work package) and carries no WP number.
function renderSectionPanel(side: HTMLDivElement, id: string) {
  const acts = project!.packages.filter((w) => (w.cluster || "") === id);
  const cur = project!.sectionColors?.[id] || "#eef0fa";
  const locked = id.toLowerCase() === "phases";
  const wpn = project!.sectionNums?.[id];
  const wpLabel = wpn && !locked ? ` · WP-${wpn}` : (locked ? " · Phases view (no number)" : "");
  // Determine the section's position among the REORDERABLE work-package lanes
  // so the up/down buttons disable correctly at the edges (P4.10). The "Phases"
  // view lane carries no WP number and is NOT reorderable, so it is excluded —
  // this must match the backend MoveSection order (which is built from the
  // numbered work packages only). Order mirrors the canvas: by WP number asc,
  // unnumbered last.
  const sectionNames = (() => {
    const set = new Set<string>();
    (project!.packages ?? []).forEach((w) => set.add(w.cluster || ""));
    Object.keys(project!.sectionColors ?? {}).forEach((c) => { if (c) set.add(c); });
    const isPhases = (c: string) => c.toLowerCase() === "phases";
    const rank = (c: string) => { const n = project!.sectionNums?.[c]; return n && n > 0 ? n : 1e9; };
    // exclude the Phases view lane — it is not a reorderable work package
    const arr = [...set].filter((c) => !isPhases(c));
    return arr.sort((a, b) => rank(a) - rank(b));
  })();
  const pos = sectionNames.indexOf(id);
  // Phases is a fixed view lane (not reorderable) — both buttons disabled.
  const atTop = locked || pos <= 0;
  const atBottom = locked || pos === sectionNames.length - 1;
  side.innerHTML = `
    <h2>Section</h2>
    <div class="editor">
      <label>Name <input id="sec_name" value="${esc(id)}" ${locked ? "disabled" : ""}/></label>
      <div>${acts.length} action(s)${wpLabel}</div>
      <div>Order <button id="sec_up" ${atTop ? "disabled" : ""}>▲ up</button> <button id="sec_down" ${atBottom ? "disabled" : ""}>▼ down</button></div>
      <div>Color <input type="color" id="sec_color" value="${cur}" /> <button id="sec_clear">default</button></div>
      <div class="erow">
        ${locked ? `<span class="muted">The top "Phases" section is fixed and cannot be renamed.</span>`
          : `<button id="sec_rename" class="primary">Rename</button>
             <button id="sec_delete">Delete section</button>`}
        <button id="sec_close">Close</button>
      </div>
    </div>`;
  const q = (s: string) => side.querySelector(s);
  if (!locked) {
    q("#sec_rename")!.addEventListener("click", async () => {
      const nn = (side.querySelector("#sec_name") as HTMLInputElement).value;
      try { project = await backend.renameSection(id, nn); selectedSectionId = nn; refresh(); }
      catch (err) { alert(String(err)); }
    });
    q("#sec_delete")!.addEventListener("click", async () => {
      // no confirm(): the desktop web view does not reliably show it; Cmd-Z undoes the delete
      try { project = await backend.removeSection(id); selectedSectionId = null; refresh(); }
      catch (err) { alert(String(err)); }
    });
  }
  q("#sec_up")!.addEventListener("click", async () => {
    try { project = await backend.moveSection(id, -1); refresh(); }
    catch (err) { alert(String(err)); }
  });
  q("#sec_down")!.addEventListener("click", async () => {
    try { project = await backend.moveSection(id, 1); refresh(); }
    catch (err) { alert(String(err)); }
  });
  q("#sec_color")!.addEventListener("change", async (e) => {
    project = await backend.setSectionColor(id, (e.target as HTMLInputElement).value); refresh();
  });
  q("#sec_clear")!.addEventListener("click", async () => {
    project = await backend.setSectionColor(id, ""); refresh();
  });
  q("#sec_close")!.addEventListener("click", () => { selectedSectionId = null; renderSide(); });
}

// --- add action: + key, context menu, helper ---
// Adds an action immediately (no prompt, which is unreliable in the WebView),
// selects it, and opens the inline editor so the name can be changed at once.
async function addActionInteractive(name = "New action", cluster?: string) {
  // Adding an action first DESELECTS any current selection (fixes a bug where a
  // stale selection made the newly added action "not found"/not focused).
  await backend.deselect().catch(() => {});
  selectedActionId = null; selectedSectionId = null; selectedDetail = null; selectedMilestone = null;
  gantt.clearSelection();
  const start = project?.start || new Date().toISOString().slice(0, 10);
  const end = project?.end || start;
  try {
    project = await backend.addWorkPackage(name, start, end);
    const last = project.packages[project.packages.length - 1];
    const id = last?.id ?? null;
    // place it into the requested section (context-menu "New action here")
    if (id && cluster !== undefined && (last?.cluster || "") !== cluster) {
      project = await backend.setActionCluster(id, cluster);
    }
    selectedActionId = id;
    currentMode = "action";
    actionEditMode = true; // new actions open straight into the editable form
    if (id) gantt.setSelected(id);
    refresh();
    focusEditorName();
  } catch (err) { alert(String(err)); }
}

// addMilestoneInteractive adds a milestone in the given section at the given
// date (defaulting to the project start) and selects it.
async function addMilestoneInteractive(cluster = "", date = "") {
  await backend.deselect().catch(() => {});
  selectedActionId = null; selectedSectionId = null; selectedDetail = null; selectedMilestone = null;
  gantt.clearSelection();
  const when = date || project?.start || new Date().toISOString().slice(0, 10);
  try {
    project = await backend.addMilestone("Milestone", when, cluster);
    const sel = await backend.selection();
    if (sel.kind === "milestone") selectMilestone(sel.id);
    else refresh();
  } catch (err) { alert(String(err)); }
}

window.addEventListener("keydown", (e) => {
  const tag = (e.target as HTMLElement)?.tagName;
  // Cmd-Z / Ctrl-Z: undo the last destructive edit (delete link/action/
  // section). Skip when editing a text field so native text undo still works.
  if ((e.metaKey || e.ctrlKey) && (e.key === "z" || e.key === "Z") && !e.shiftKey) {
    if (tag === "INPUT" || tag === "TEXTAREA") return;
    e.preventDefault();
    backend.undo().then((p) => { project = p; refresh(); }).catch((err) => alert(String(err)));
    return;
  }
  if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;
  // Escape (or Ctrl pressed alone as a quick gesture) clears the selection.
  if (e.key === "Escape" || (e.key === "Control")) {
    deselectAll();
    return;
  }
  if (e.key === "+" || (e.key === "=" && e.shiftKey)) { e.preventDefault(); addActionInteractive(); }
});

// deselectAll clears the current action/section/detail selection (Escape / Ctrl
// / click on empty canvas) and removes the canvas selection border.
function deselectAll() {
  selectedActionId = null; selectedSectionId = null; selectedDetail = null; selectedMilestone = null;
  gantt.clearSelection();
  backend.deselect().catch(() => {});
  renderSide();
}

function openContextMenu(x: number, y: number, target: import("./gantt").CtxTarget) {
  closeContextMenu();
  const menu = document.createElement("div");
  menu.id = "ctxmenu";
  menu.className = "ctxmenu";
  menu.style.left = x + "px";
  menu.style.top = y + "px";

  // Items depend on what was right-clicked (see ctxmenu.ts). Deletes are
  // undoable with Cmd-Z and are not confirmed with confirm(), which the
  // desktop web view does not reliably show.
  const fail = (err: unknown) => alert(String(err));
  const items = contextMenuItems(target, {
    editAction: (id) => openActionEdit(id),
    deleteAction: async (id) => {
      try {
        project = await backend.removeWorkPackage(id);
        if (selectedActionId === id) { selectedActionId = null; gantt.clearSelection(); }
        refresh();
      } catch (err) { fail(err); }
    },
    moveAction: async (id, dir) => { try { project = await backend.moveAction(id, dir); refresh(); } catch (err) { fail(err); } },
    editMilestone: (id) => selectMilestone(id),
    deleteMilestone: async (id) => {
      try { project = await backend.removeMilestone(id); selectedMilestone = null; gantt.clearSelection(); refresh(); }
      catch (err) { fail(err); }
    },
    editSection: (c) => selectSection(c),
    deleteSection: async (c) => {
      try { project = await backend.removeSection(c); if (selectedSectionId === c) selectedSectionId = null; refresh(); }
      catch (err) { fail(err); }
    },
    moveSection: async (c, dir) => { try { project = await backend.moveSection(c, dir); refresh(); } catch (err) { fail(err); } },
    newAction: (cluster) => addActionInteractive("New action", cluster),
    newMilestone: (cluster, date) => addMilestoneInteractive(cluster, date),
    newSection: async () => {
      const n = Object.keys(project?.sectionColors ?? {}).length + 1;
      try { project = await backend.addSection("Section " + n); refresh(); } catch (err) { fail(err); }
    },
    addRole: async () => {
      const n = (project?.roles.length ?? 0) + 1;
      try { project = await backend.addRole("Role " + n, ""); refresh(); } catch (err) { fail(err); }
    },
  });

  menu.innerHTML = items.map((it, i) => `<button data-i="${i}">${esc(it.label)}</button>`).join("");
  document.body.appendChild(menu);
  menu.querySelectorAll<HTMLButtonElement>("button").forEach((b) =>
    b.addEventListener("click", () => { closeContextMenu(); items[Number(b.dataset.i)].run(); }));

  // close on an outside click (ignore clicks inside the menu itself)
  setTimeout(() => {
    const onDoc = (ev: MouseEvent) => {
      if (!(ev.target as HTMLElement)?.closest("#ctxmenu")) { closeContextMenu(); window.removeEventListener("mousedown", onDoc); }
    };
    window.addEventListener("mousedown", onDoc);
  }, 0);
}
function closeContextMenu() { document.querySelector("#ctxmenu")?.remove(); }

// roleBg returns the bar background a detail uses (its role's color, or the
// default detail fill) so the Text-color picker can preview the auto-contrast.
function roleBg(roleName: string): string {
  if (roleName) {
    const r = project!.roles.find((x) => x.name === roleName);
    if (r && r.color) return r.color;
  }
  return "#8a90dd";
}

// openDetailEditor is kept for the drill plane's "edit a detail" gesture: it
// selects the detail so its editor shows in the right #side pane.
function openDetailEditor(wpId: string, detId: string) {
  selectDetail(wpId, detId);
}

// renderDetailPanel renders a detail sub-action's editor in the right #side
// pane. A detail action has the SAME edit features as a level-1 action: Title,
// Description, Effort, Role, editable Start/End dates, detail-order ▲/▼ reorder,
// Save/Delete/Close, and the shared dependency (links) table — so links made in
// the drill plane appear here and can be deleted.
function renderDetailPanel(side: HTMLDivElement, wpId: string, detId: string) {
  const wp = project!.packages.find((w) => w.id === wpId);
  const d = wp?.details.find((x) => x.id === detId);
  if (!d) { selectedDetail = null; side.innerHTML = ""; return; }
  let ddTextPos = d.textPos || "auto";
  const roleOpts = (sel: string) =>
    `<option value="">(none)</option>` +
    project!.roles.map((r) => `<option value="${esc(r.name)}"${r.name === sel ? " selected" : ""}>${esc(r.name)}</option>`).join("");
  side.innerHTML = `<h2>Detail ${d.displayId} — ${esc(d.name)}</h2>
    <div class="editor">
      <label>Title<br/><input id="dd_name" value="${esc(d.name)}" /></label>
      <label>Description<br/><textarea id="dd_desc" rows="2">${esc(d.desc)}</textarea></label>
      <label>Effort <input id="dd_est" value="${esc(d.est)}" size="6" /> ${project!.unit}</label>
      <label>Role <select id="dd_role">${roleOpts(d.role)}</select></label>
      <label><input type="checkbox" id="dd_summary" ${d.summary ? "checked" : ""}/> Summary bar (Sammelvorgang)</label>
      <label>Text color <input id="dd_fontcolor" type="color" value="${fontColorValue(d.fontColor, roleBg(d.role))}" />
        <button id="dd_fontauto" type="button" title="auto-contrast (black/white picked from the bar color)">auto</button></label>
      ${textPosPickerHTML(d.textPos, "dd_tp")}
      <label>Start <input id="dd_start" type="date" value="${d.start}" /></label>
      <label>End <input id="dd_end" type="date" value="${d.end}" /></label>
      <div>Detail order <button id="dd_up">▲ up</button> <button id="dd_down">▼ down</button></div>
      <div class="erow">
        <button id="dd_save" class="primary">Save</button>
        <button id="dd_del">Delete</button>
        <button id="dd_close">Close</button>
      </div>
    </div>` + linksTableHTML();
  side.querySelector("#dd_save")!.addEventListener("click", async () => {
    const name = (side.querySelector("#dd_name") as HTMLInputElement).value;
    const desc = (side.querySelector("#dd_desc") as HTMLTextAreaElement).value;
    const est = (side.querySelector("#dd_est") as HTMLInputElement).value;
    const role = (side.querySelector("#dd_role") as HTMLSelectElement).value;
    const start = (side.querySelector("#dd_start") as HTMLInputElement).value;
    const end = (side.querySelector("#dd_end") as HTMLInputElement).value;
    const fontEl = side.querySelector("#dd_fontcolor") as HTMLInputElement;
    const fontColor = fontEl.dataset.auto === "1" ? "" : fontEl.value;
    const summary = (side.querySelector("#dd_summary") as HTMLInputElement).checked;
    try {
      project = await backend.setDetailFields(wpId, detId, name, desc, est, role);
      project = await backend.setDetailSummary(wpId, detId, summary);
      if (start && end) project = await backend.moveDetail(wpId, detId, start, end);
      project = await backend.setDetailFontColor(wpId, detId, fontColor);
      project = await backend.setDetailTextPos(wpId, detId, ddTextPos);
      refresh();
    } catch (err) { alert(String(err)); }
  });
  side.querySelector("#dd_del")!.addEventListener("click", async () => {
    project = await backend.removeDetail(wpId, detId); selectedDetail = null; refresh();
  });
  side.querySelector("#dd_close")!.addEventListener("click", () => { selectedDetail = null; renderSide(); });
  side.querySelector("#dd_up")!.addEventListener("click", async () => {
    try { project = await backend.moveDetailOrder(wpId, detId, -1); refresh(); }
    catch (err) { alert(String(err)); }
  });
  side.querySelector("#dd_down")!.addEventListener("click", async () => {
    try { project = await backend.moveDetailOrder(wpId, detId, 1); refresh(); }
    catch (err) { alert(String(err)); }
  });
  // Text color: picking a color turns OFF auto; "auto" clears the explicit
  // color so the renderer auto-contrasts against the detail's bar color.
  const ddFont = side.querySelector("#dd_fontcolor") as HTMLInputElement | null;
  if (ddFont) {
    ddFont.dataset.auto = d.fontColor ? "0" : "1";
    ddFont.addEventListener("input", () => { ddFont.dataset.auto = "0"; });
  }
  side.querySelector("#dd_fontauto")?.addEventListener("click", () => {
    if (!ddFont) return;
    ddFont.dataset.auto = "1";
    ddFont.value = autoContrast(roleBg((side.querySelector("#dd_role") as HTMLSelectElement).value));
  });
  const ddTpWrap = side.querySelector<HTMLElement>("#dd_tp_wrap");
  if (ddTpWrap) wireTextPosPicker(ddTpWrap, "dd_tp", (pos) => { ddTextPos = pos; });
  wireLinksTable(side);
}

// Action editor rendered INLINE in the side area (not a modal). Lets you edit
// title, description, estimation OR duration (summary), up to 3 roles with %,
// start/end, color, and summary flag.
// Read-only "show action" dialog: displays Effort, Duration, dates, section,
// description and roles. Apply/Delete are present but DISABLED (grayed out);
// only Edit, Details… and Close act. Click Edit to switch to the editable form.
function actionShowHTML(wp: import("./types").WorkPack): string {
  const assigned: string[] = [];
  for (const r of project!.roles)
    for (const a of r.assignments)
      if (a.wp === wp.id) assigned.push(`${esc(r.name)} ${a.percent}%`);
  const effortLine = wp.summary
    ? `<div>Effort: <b>(summary — rolled up)</b></div>`
    : wp.contour === "None"
    ? `<div class="muted">Contour <b>None</b>: no estimation and no real effort, no progress bar.</div>`
    : `<div>Planned effort: ${effortHTML(project!.unit, wp.estHours, wp.estDays)} · mode: ${wp.mode}</div>
       <div>Real effort: ${effortHTML(project!.unit, wp.realHours, wp.realDays)}${wp.synced ? ` <span class="muted">synchronized${wp.syncedAt ? " " + esc(wp.syncedAt) : ""}</span>` : ""}
         · consumed <b>${esc(ratingLabel(wp.est !== "0", wp.progressPermille, wp.contourPermille, wp.contour, project!.language === "de")) || "-"}</b></div>
       <div id="s_actual" data-action="${esc(wp.id)}">${project!.monitor?.enabled ? `<span class="muted">Actual effort: loading…</span>` : ""}</div>`;
  const warnHTML = actionWarnings(wp).map((w) => `<div class="warn">${esc(w)}</div>`).join("");
  return `<h2>Action ${wp.displayId} — ${esc(wp.name)}</h2>
    <div class="editor">
      ${effortLine}
      <div>Duration: <b>${dayCount(wp.start, wp.end)}</b> days (${wp.start} → ${wp.end})</div>
      <div id="s_workdays">Working days (Mon–Fri, without weekends): <b>${wp.workdays}</b></div>
      <div>Section: ${esc(wp.cluster || "(none)")}</div>
      ${assigned.length ? `<div>Roles: ${assigned.join(", ")}</div>` : ""}
      ${roleFitHTML(wp)}
      ${wp.desc ? `<div>${esc(wp.desc)}</div>` : ""}
      <div>Action order <button id="s_up">▲ up</button> <button id="s_down">▼ down</button></div>
      ${wp.cluster ? `<div>Section order <button id="s_secup">▲ up</button> <button id="s_secdown">▼ down</button></div>` : ""}
      <div id="e_warn">${warnHTML}</div>
      <div class="erow">
        <button id="s_edit" class="primary">Edit</button>
        <button id="s_details">Details…</button>
        ${project!.showDetails && (wp.details?.length ?? 0) > 0
          ? `<button id="s_toggledetails">${wp.detailsHidden ? "Show detail bars" : "Hide detail bars"}</button>`
          : ""}
      </div>
    </div>`;
}

function wireActionShow(id: string) {
  const q = <T extends Element>(s: string) => document.querySelector<T>(s)!;
  // actual effort of the last synchronization (only while monitoring is on)
  const slot = document.querySelector<HTMLElement>("#s_actual");
  const wpShown = project!.packages.find((w) => w.id === id);
  if (slot && wpShown && project!.monitor?.enabled) {
    backend.getMonitorSummary().then((s) => {
      const cur = document.querySelector<HTMLElement>("#s_actual");
      if (cur && cur.dataset.action === id) cur.innerHTML = actualEffortHTML(project!, wpShown, s);
    }).catch(() => { slot.innerHTML = ""; });
  }
  q("#s_edit").addEventListener("click", () => openActionEdit(id));
  q("#s_details").addEventListener("click", () => openDrill(id));
  // Per-action show/hide of this action's detail bars (only present when the
  // project-wide Show Details is on and the action has details).
  document.querySelector<HTMLButtonElement>("#s_toggledetails")?.addEventListener("click", async () => {
    const wpNow = project!.packages.find((w) => w.id === id);
    try { project = await backend.setActionDetailsHidden(id, !wpNow?.detailsHidden); refresh(); selectAction(id); }
    catch (err) { alert(String(err)); }
  });
  q("#s_up").addEventListener("click", async () => {
    try { project = await backend.moveAction(id, -1); refresh(); }
    catch (err) { alert(String(err)); }
  });
  q("#s_down").addEventListener("click", async () => {
    try { project = await backend.moveAction(id, 1); refresh(); }
    catch (err) { alert(String(err)); }
  });
  // Section order up/down (moves the action's whole section/swimlane).
  const wp = project!.packages.find((w) => w.id === id);
  const cluster = wp?.cluster || "";
  const secUp = document.querySelector<HTMLButtonElement>("#s_secup");
  const secDown = document.querySelector<HTMLButtonElement>("#s_secdown");
  if (cluster && secUp && secDown) {
    secUp.addEventListener("click", async () => {
      try { project = await backend.moveSection(cluster, -1); refresh(); }
      catch (err) { alert(String(err)); }
    });
    secDown.addEventListener("click", async () => {
      try { project = await backend.moveSection(cluster, 1); refresh(); }
      catch (err) { alert(String(err)); }
    });
  }
}

// actionWarnings returns per-action warnings shown directly under the edit
// form. It uses the authoritative backend warnings (which already include the
// per-WP role-percentage check) plus a direct effort-coverage check so the
// message appears immediately after Apply.
function actionWarnings(wp: import("./types").WorkPack): string[] {
  const out: string[] = [];
  if (wp.summary) return out; // summaries roll up; no direct effort assignment
  // Role percentages are checked PER LINE only (R-6.8): a single assignment
  // may not exceed 100%. Lines are NOT summed, so multiple roles each at 100%
  // on one action is valid and raises no warning.
  for (const r of project!.roles)
    for (const a of r.assignments)
      if (a.wp === wp.id && a.percent > 100)
        out.push(`Role ${r.name} is assigned ${a.percent}% (a single assignment may not exceed 100%).`);
  // FS dependency violations involving this action (as predecessor or successor)
  for (const dep of project!.deps ?? []) {
    if (!dep.violated) continue;
    if (dep.from === wp.id || dep.to === wp.id) {
      out.push(`Dependency violated: ${dep.to} starts before ${dep.from} ends (end-start). Move it later or relax the link.`);
    }
  }
  // Over-allocation: surface the backend warning that mentions this action
  // ("WP <id>: effort Nd exceeds M working day(s) ...").
  for (const warn of project!.warnings ?? []) {
    if (warn.includes(`WP ${wp.id}:`) && warn.includes("over-allocated")) {
      out.push(`Effort exceeds the working days available in the window (over-allocated). Extend the dates or reduce the effort.`);
    }
  }
  return out;
}

// fontColorValue picks the value shown in the "Text" color picker: the explicit
// font color if set, else the auto-contrast color computed from the bar color
// (so the swatch previews what will actually be drawn). A picker cannot show
// "auto", so the "auto" button clears the explicit color to restore it.
function fontColorValue(fontColor: string, bg: string): string {
  if (fontColor) return fontColor;
  return autoContrast(bg || "#8a90dd");
}

// autoContrast mirrors gantt.ts contrastOn: #111 on light backgrounds, #fff on
// dark ones, via W3C relative luminance.
function autoContrast(hex: string): string {
  const m = /^#?([0-9a-f]{3}|[0-9a-f]{6})$/i.exec((hex || "").trim());
  if (!m) return "#ffffff";
  let h = m[1];
  if (h.length === 3) h = h[0] + h[0] + h[1] + h[1] + h[2] + h[2];
  const toLin = (c: number) => { const s = c / 255; return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4; };
  const r = parseInt(h.slice(0, 2), 16), g = parseInt(h.slice(2, 4), 16), b = parseInt(h.slice(4, 6), 16);
  const L = 0.2126 * toLin(r) + 0.7152 * toLin(g) + 0.0722 * toLin(b);
  return L > 0.179 ? "#111111" : "#ffffff";
}

// notePosPickerHTML is the position picker for the sticky NOTE. Like
// textPosPickerHTML but WITHOUT the Auto option and WITH four diagonal corner
// dots (ul/ur/ll/lr) at the bar corners, so the connector can run at ~45°.
// Reuses the tp_dot class + #<idp>_wrap/#<idp>_cur so wireTextPosPicker works.
function notePosPickerHTML(current: string, idp: string): string {
  const cur = current || "up";
  const dot = (pos: string, cx: number, cy: number) => {
    const on = cur === pos;
    return `<circle class="tp_dot" data-pos="${pos}" cx="${cx}" cy="${cy}" r="4"
      fill="${on ? "#534fbc" : "#fff"}" stroke="#534fbc" stroke-width="1.5"
      style="cursor:pointer"><title>${pos}</title></circle>`;
  };
  const bx = 30, by = 14, bw = 60, bh = 20;
  const mx = bx + bw / 2, my = by + bh / 2;
  const topY = 4, botY = by + bh + 10, leftX = bx - 14, rightX = bx + bw + 14;
  return `<div class="tppickrow" style="margin:6px 0">
    <div style="font-size:12px;margin-bottom:2px">Note position</div>
    <span class="tppick" id="${idp}_wrap" style="display:inline-flex;align-items:center;gap:8px">
      <svg width="120" height="52" viewBox="0 0 120 52" style="border:1px solid #ddd;border-radius:4px;background:#fafafb">
        <rect x="${bx}" y="${by}" width="${bw}" height="${bh}" rx="2" fill="#8a90dd" stroke="#534fbc"/>
        ${dot("ul", leftX, topY)}
        ${dot("up", mx, topY)}
        ${dot("ur", rightX, topY)}
        ${dot("left", leftX, my)}
        ${dot("center", mx, my)}
        ${dot("right", rightX, my)}
        ${dot("ll", leftX, botY)}
        ${dot("down", mx, botY)}
        ${dot("lr", rightX, botY)}
      </svg>
      <span class="muted" id="${idp}_cur">${esc(cur)}</span>
    </span>
  </div>`;
}

// wireNotePosPicker attaches click handlers to the note position dots. Unlike
// wireTextPosPicker it has no Auto control and no right->outright normalization
// (the note picker uses plain up/down/left/right/center + ul/ur/ll/lr corners).
function wireNotePosPicker(root: HTMLElement, onPick: (pos: string) => void) {
  const curLbl = root.querySelector<HTMLElement>('[id$="_cur"]');
  root.querySelectorAll<SVGCircleElement>(".tp_dot").forEach((c) =>
    c.addEventListener("click", (e) => {
      e.preventDefault(); e.stopPropagation();
      const v = c.dataset.pos!;
      if (curLbl) curLbl.textContent = v;
      root.querySelectorAll<SVGCircleElement>(".tp_dot").forEach((d) =>
        d.setAttribute("fill", d.dataset.pos === v ? "#534fbc" : "#fff"));
      onPick(v);
    }));
}

// textPosPickerHTML renders a small graphic position picker: a mini action bar
// with 5 clickable dots (up / left / center / right / down) plus an Auto
// toggle. `current` is "" / "auto" | center | left | right | up | down. The
// active choice is highlighted. `idp` is an id prefix so the action editor and
// the detail panel can host their own picker on screen at once.
function textPosPickerHTML(current: string, idp: string): string {
  const cur = current || "auto";
  const dot = (pos: string, cx: number, cy: number) => {
    const on = cur === pos;
    return `<circle class="tp_dot" data-pos="${pos}" cx="${cx}" cy="${cy}" r="4"
      fill="${on ? "#534fbc" : "#fff"}" stroke="#534fbc" stroke-width="1.5"
      style="cursor:pointer"><title>${pos}</title></circle>`;
  };
  // mini bar 60x20 in a 120x44 canvas, leaving room for the OUTSIDE dots to the
  // left (outleft) and right (outright) of the bar, plus up/down above/below.
  const bx = 30, by = 12, bw = 60, bh = 20;
  const mx = bx + bw / 2, my = by + bh / 2;
  // NOTE: do NOT wrap this in a <label>. A <label> forwards any click inside it
  // to its single form control (the Auto checkbox), so clicking an SVG dot would
  // also toggle Auto and snap the choice back to "auto". Use a plain div.
  return `<div class="tppickrow" style="margin:6px 0">
    <div style="font-size:12px;margin-bottom:2px">Text position</div>
    <span class="tppick" id="${idp}_wrap" style="display:inline-flex;align-items:center;gap:8px">
      <svg width="120" height="44" viewBox="0 0 120 44" style="border:1px solid #ddd;border-radius:4px;background:#fafafb">
        <rect x="${bx}" y="${by}" width="${bw}" height="${bh}" rx="2" fill="#8a90dd" stroke="#534fbc"/>
        ${dot("up", mx, 4)}
        ${dot("outleft", bx - 14, my)}
        ${dot("left", bx + 8, my)}
        ${dot("center", mx, my)}
        ${dot("outright", bx + bw + 14, my)}
        ${dot("down", mx, 40)}
      </svg>
      <label style="font-weight:normal"><input type="checkbox" id="${idp}_auto" ${cur === "auto" ? "checked" : ""}/> Auto</label>
      <span class="muted" id="${idp}_cur">${esc(cur === "right" ? "outright" : cur)}</span>
    </span>
  </div>`;
}

// wireTextPosPicker attaches click handlers to the picker dots and the Auto
// radio; onPick receives the chosen value ("auto" | center | left | right |
// up | down). It also refreshes the picker's own highlight + label so the user
// sees the choice before Save.
function wireTextPosPicker(root: HTMLElement, idp: string, onPick: (pos: string) => void) {
  const curLbl = root.querySelector<HTMLElement>(`#${idp}_cur`);
  const setCur = (v: string) => {
    // normalize the legacy "right" alias to "outright" for display + highlight
    const nv = v === "right" ? "outright" : v;
    if (curLbl) curLbl.textContent = nv;
    root.querySelectorAll<SVGCircleElement>(".tp_dot").forEach((c) => {
      const on = c.dataset.pos === nv;
      c.setAttribute("fill", on ? "#534fbc" : "#fff");
    });
    const auto = root.querySelector<HTMLInputElement>(`#${idp}_auto`);
    if (auto) auto.checked = nv === "auto";
  };
  // Clicking a dot selects an explicit position (and turns Auto off).
  root.querySelectorAll<SVGCircleElement>(".tp_dot").forEach((c) =>
    c.addEventListener("click", (e) => {
      e.preventDefault(); e.stopPropagation();
      const v = c.dataset.pos!; setCur(v); onPick(v);
    }));
  // The Auto checkbox: checked -> auto; unchecked -> fall back to an explicit
  // "left" so the choice is no longer auto.
  root.querySelector<HTMLInputElement>(`#${idp}_auto`)?.addEventListener("change", (e) => {
    const on = (e.target as HTMLInputElement).checked;
    const v = on ? "auto" : "left";
    setCur(v); onPick(v);
  });
}

// realNoteHTML explains where an action's real effort comes from.
function realNoteHTML(wp: import("./types").WorkPack): string {
  if (wp.contour === "None") return `Contour <b>None</b>: this action has no estimation and no real effort. The fields are greyed out and the action has no progress bar.`;
  const hd = effortHTML(project!.unit, wp.realHours, wp.realDays);
  if (!wp.synced) return `Real effort ${hd} (entered manually; editable)`;
  const sys = project!.monitor.system ? project!.monitor.system[0].toUpperCase() + project!.monitor.system.slice(1) : "the external system";
  const why = wp.externalId ? `External ID ${esc(wp.externalId)}` : "default action";
  return `Real effort ${hd}: <b>synchronized</b> from ${esc(sys)} (${why})${wp.syncedAt ? `, ${esc(wp.syncedAt)}` : ""} - read-only. `
    + `Switch monitoring off on the Monitor page to edit it by hand.`;
}

function actionEditorHTML(wp: import("./types").WorkPack): string {
  // seed the editable role lines from the action's current assignments; keep at
  // least 3 visible lines. More lines can be added via "Add Role".
  editRoleLines = [];
  for (const r of project!.roles)
    for (const a of r.assignments)
      if (a.wp === wp.id) editRoleLines.push({ role: r.name, pct: a.percent });
  while (editRoleLines.length < 3) editRoleLines.push({ role: "", pct: 0 });
  const extState = externalIdState(project!, wp);
  const estOrDur = wp.summary
    ? `<label>Duration: ${wp.start} → ${wp.end} (set via dates below)</label>`
    : `<div class="headline-effort">
        <label>Estimated <input id="e_est" value="${esc(wp.est)}" size="6" ${wp.contour === "None" ? 'disabled aria-disabled="true" title="contour None: no estimation"' : ""} /> ${project!.unit}</label>
        <label>Real <input id="e_realest" value="${esc(wp.realEst)}" size="6"
          ${wp.contour === "None" ? 'disabled aria-disabled="true" title="contour None: no real effort"' : wp.synced ? `readonly aria-readonly="true" title="synchronized from the external system - cannot be edited"` : ""} /> ${project!.unit}</label>
        <span class="muted" id="e_ratings">${wp.contour === "None" ? "" : "Consumed " + esc(ratingLabel(wp.est !== "0", wp.progressPermille, wp.contourPermille, wp.contour, project!.language === "de"))}</span>
        <div class="muted" id="e_realnote" style="flex-basis:100%">${realNoteHTML(wp)}</div>
        <label>Contour
          <select id="e_contour">
            ${["Flat", "Front-Loaded", "Back-Loaded", "Double-Peak", "None"].map((k) =>
              `<option value="${k}"${wp.contour === k ? " selected" : ""}>${k}</option>`).join("")}
          </select>
        </label>
        <label>External ID
          <input id="e_extid" maxlength="16" size="16" value="${esc(wp.externalId)}" ${extState.readOnly ? 'disabled aria-disabled="true"' : ""}
            title="${extState.readOnly ? "an action with the contour None cannot carry an External ID" : "id used to find this action's bookings in the external system (max. 16 characters)"}" />
        </label>
        ${extState.noteHTML ? `<div class="muted" id="e_extnote" style="flex-basis:100%">${extState.noteHTML}</div>` : ""}
        <div class="warn" id="e_fielderr" role="alert" style="flex-basis:100%"></div>
      </div>`;
  const warnHTML = actionWarnings(wp).map((w) => `<div class="warn">${esc(w)}</div>`).join("");
  return `<h2>Edit action ${wp.displayId} — ${esc(wp.name)}</h2>
    <div class="editor">
      <label>Title<br/><input id="e_name" value="${esc(wp.name)}" /></label>
      <label>Description<br/><textarea id="e_desc" rows="2">${esc(wp.desc)}</textarea></label>
      ${estOrDur}
      <div><b>Roles (each line max 100% of effort)</b>
        <div id="e_roles">${roleLinesHTML()}</div>
        <button id="e_addrole" type="button">Add Role</button>
        ${roleFitHTML(wp)}
      </div>
      <label>Start <input id="e_start" type="date" value="${wp.start}" /></label>
      <label>End <input id="e_end" type="date" value="${wp.end}" /></label>
      <div class="muted" id="e_workdays" role="status">Working days (without weekends): <b>${workdayCount(wp.start, wp.end)}</b></div>
      <label>Color <input id="e_color" type="color" value="${wp.color || "#8a90dd"}" />
        &nbsp; Text <input id="e_fontcolor" type="color" value="${fontColorValue(wp.fontColor, wp.color)}" />
        <button id="e_fontauto" type="button" title="auto-contrast (black/white picked from the bar color)">auto</button>
        <label><input id="e_summary" type="checkbox" ${wp.summary ? "checked" : ""}/> summary</label>
        <label><input id="e_sameline" type="checkbox" ${wp.sameLine ? "checked" : ""}/> same line</label></label>
      ${textPosPickerHTML(wp.textPos, "e_tp")}
      <div style="margin:6px 0;border-top:1px solid #eee;padding-top:6px">
        <label><input id="e_note_on" type="checkbox" ${wp.noteVisible ? "checked" : ""}/> <b>Note</b> (show Description as a yellow note)</label>
        <div id="e_note_pos" style="${wp.noteVisible ? "" : "display:none"}">${notePosPickerHTML(wp.notePos || "up", "e_np")}</div>
      </div>
      <div class="erow">
        <button id="e_apply" class="primary">Apply</button>
        <button id="e_details">Details…</button>
        <button id="e_del">Delete</button>
        <button id="e_closeedit">Close</button>
      </div>
      <div id="e_warn">${warnHTML}</div>
    </div>`;
}

// editRoleLines holds the role-assignment lines for the open action editor.
// The same role may appear more than once (e.g. two Developers). Each line's
// percent is capped at 100 independently (R-6.7); lines are NOT summed.
let editRoleLines: { role: string; pct: number }[] = [];

// editTextPos holds the text-position choice for the open action editor, read
// by the Apply handler. Seeded in wireActionEditor from the action.
let editTextPos = "auto";

// roleLinesHTML renders the current editRoleLines. Lines beyond the first
// three carry a × remove control (R-6.9).
function roleLinesHTML(): string {
  const roleOpts = (sel: string) =>
    `<option value="">(none)</option>` +
    project!.roles.map((r) => `<option value="${esc(r.name)}"${r.name === sel ? " selected" : ""}>${esc(r.name)}</option>`).join("");
  return editRoleLines.map((x, i) => `
    <div class="erole"><select class="e_role" data-i="${i}">${roleOpts(x.role)}</select>
      <input class="e_pct" data-i="${i}" type="number" min="0" max="100" value="${x.pct}" size="3" />%${
      i >= 3 ? ` <button type="button" class="e_delrole" data-i="${i}" title="remove role line">×</button>` : ""
    }</div>`).join("");
}

// roleFitHTML renders the READ-ONLY capacity line below the roles: whether the
// assigned role capacity (percent = fraction of the working day each role
// spends) covers the effort, and whether the effort fits the start..end
// window. Empty when there is no capacity summary (no effort / no roles).
function roleFitHTML(wp: import("./types").WorkPack): string {
  const rf = wp.roleFit;
  if (!rf) return "";
  const cls = rf.fits && rf.fitsWindow ? "rolefit-ok" : "rolefit-bad";
  return `<div class="rolefit ${cls}" id="e_rolefit">${esc(rf.message)}</div>`;
}
function syncEditRoleLines() {
  document.querySelectorAll<HTMLSelectElement>(".e_role").forEach((s) => {
    const i = Number(s.dataset.i);
    const pctEl = document.querySelector(`.e_pct[data-i="${i}"]`) as HTMLInputElement | null;
    if (editRoleLines[i]) { editRoleLines[i].role = s.value; editRoleLines[i].pct = Number(pctEl?.value || 0); }
  });
}

// rerenderRoleLines redraws the roles block and re-wires its controls.
function rerenderRoleLines() {
  const host = document.querySelector<HTMLDivElement>("#e_roles");
  if (!host) return;
  host.innerHTML = roleLinesHTML();
  wireRoleLineControls();
}

// wireRoleLineControls attaches the per-line × remove handlers and clamps each
// percent input to 0..100 (per line only; R-6.7/R-6.8).
function wireRoleLineControls() {
  document.querySelectorAll<HTMLButtonElement>(".e_delrole").forEach((b) =>
    b.addEventListener("click", () => {
      syncEditRoleLines();
      editRoleLines.splice(Number(b.dataset.i), 1);
      rerenderRoleLines();
    }));
  document.querySelectorAll<HTMLInputElement>(".e_pct").forEach((inp) =>
    inp.addEventListener("change", () => {
      let v = Number(inp.value);
      if (v > 100) v = 100; if (v < 0) v = 0; // cap a single line at 100%
      inp.value = String(v);
    }));
}

// setInvalid marks (or clears) the validation indication on a form control:
// aria-invalid for assistive tech plus a red outline for sighted users.
function setInvalid(el: HTMLElement, bad: boolean) {
  if (bad) { el.setAttribute("aria-invalid", "true"); el.style.outline = "2px solid #e03131"; }
  else { el.removeAttribute("aria-invalid"); el.style.outline = ""; }
}

// wireProgressControls hooks up the Headline_Effort lines (estimated + real
// effort, each committed on Enter or blur) and the Work_Contour picker of the
// open action editor. A rejected commit marks the control invalid and puts the
// stored value back; the stored value is never changed by a rejected commit.
function wireProgressControls(id: string) {
  const stored = (f: "est" | "realEst" | "contour" | "externalId") =>
    project!.packages.find((w) => w.id === id)?.[f] ?? "";

  // a rejected value is explained in words (not only by a red outline)
  const fieldError = (msg: string) => {
    const el = document.querySelector<HTMLElement>("#e_fielderr");
    if (el) el.textContent = msg;
  };
  const wireEffort = (sel: string, f: "est" | "realEst" | "externalId",
                      commit: (id: string, v: string) => Promise<import("./types").Project>) => {
    const inp = document.querySelector<HTMLInputElement>(sel);
    if (!inp || inp.disabled || inp.readOnly) return; // locked fields are not wired
    let busy = false;
    const go = async () => {
      if (busy || inp.value === stored(f)) return; // nothing to commit
      busy = true;
      try {
        project = await commit(id, inp.value);
        refresh();
      } catch (err) {
        inp.value = String(stored(f));
        setInvalid(inp, true);
        fieldError(errorText(err));
      } finally { busy = false; }
    };
    inp.addEventListener("keydown", (e) => { if (e.key === "Enter") { e.preventDefault(); void go(); } });
    inp.addEventListener("blur", () => { void go(); });
    inp.addEventListener("input", () => { setInvalid(inp, false); fieldError(""); });
  };
  wireEffort("#e_est", "est", backend.setWorkPackageEstimation);
  // locked inputs (synchronized real effort, contour None) are skipped by wireEffort
  wireEffort("#e_realest", "realEst", backend.setWorkPackageRealEffort);
  // the External ID is editable at any time, also while monitoring is on
  wireEffort("#e_extid", "externalId", backend.setActionExternalId);

  const sel = document.querySelector<HTMLSelectElement>("#e_contour");
  sel?.addEventListener("change", async () => {
    try {
      project = await backend.setWorkPackageContour(id, sel.value);
      refresh();
    } catch (err) {
      sel.value = String(stored("contour"));
      setInvalid(sel, true);
      fieldError(errorText(err)); // e.g. "remove the External ID first"
    }
  });
}

function wireActionEditor(id: string) {
  const q = <T extends Element>(s: string) => document.querySelector<T>(s)!;
  const apply = async () => {
    const name = (q("#e_name") as HTMLInputElement).value;
    const desc = (q("#e_desc") as HTMLTextAreaElement).value;
    const estEl = document.querySelector("#e_est") as HTMLInputElement | null;
    const est = estEl && !estEl.disabled ? estEl.value : ""; // contour None: the greyed-out field is not sent
    const start = (q("#e_start") as HTMLInputElement).value;
    const end = (q("#e_end") as HTMLInputElement).value;
    const color = (q("#e_color") as HTMLInputElement).value;
    const fontEl = q("#e_fontcolor") as HTMLInputElement;
    // an explicit font color unless the user pressed "auto" (data-auto="1"),
    // which clears it so the renderer auto-contrasts against the bar color.
    const fontColor = fontEl.dataset.auto === "1" ? "" : fontEl.value;
    const summary = (q("#e_summary") as HTMLInputElement).checked;
    const sameLine = (q("#e_sameline") as HTMLInputElement).checked;
    const roles: string[] = []; const pcts: number[] = [];
    syncEditRoleLines();
    for (const x of editRoleLines) {
      if (!x.role) continue;
      let p = x.pct; if (p > 100) p = 100; if (p < 0) p = 0; // per-line cap
      roles.push(x.role); pcts.push(p);
    }
    try {
      project = await backend.setActionFields(id, name, desc, est, start, end);
      project = await backend.setWorkPackageSummary(id, summary);
      project = await backend.setWorkPackageSameLine(id, sameLine);
      project = await backend.setActionColor(id, color);
      project = await backend.setActionFontColor(id, fontColor);
      project = await backend.setActionTextPos(id, editTextPos);
      project = await backend.setActionRoles(id, roles, pcts);
      refresh();
    } catch (err) { alert(String(err)); }
  };
  q("#e_apply").addEventListener("click", apply);
  wireProgressControls(id);
  // live working-day count while the dates are edited
  const updWd = () => {
    const el = document.querySelector<HTMLElement>("#e_workdays");
    const s = document.querySelector<HTMLInputElement>("#e_start")?.value ?? "";
    const e = document.querySelector<HTMLInputElement>("#e_end")?.value ?? "";
    if (el) el.innerHTML = `Working days (without weekends): <b>${workdayCount(s, e)}</b>`;
  };
  document.querySelector("#e_start")?.addEventListener("input", updWd);
  document.querySelector("#e_end")?.addEventListener("input", updWd);
  // Font color: picking a color turns OFF auto; the "auto" button turns it back
  // ON (clears the explicit color) and previews the auto-contrast swatch.
  const fontEl = document.querySelector<HTMLInputElement>("#e_fontcolor");
  const wpForFont = project!.packages.find((w) => w.id === id);
  if (fontEl) {
    fontEl.dataset.auto = wpForFont && wpForFont.fontColor ? "0" : "1";
    fontEl.addEventListener("input", () => { fontEl.dataset.auto = "0"; });
  }
  document.querySelector("#e_fontauto")?.addEventListener("click", () => {
    if (!fontEl) return;
    fontEl.dataset.auto = "1";
    const bg = (document.querySelector("#e_color") as HTMLInputElement | null)?.value || "#8a90dd";
    fontEl.value = autoContrast(bg);
  });
  // Text position picker.
  editTextPos = wpForFont?.textPos || "auto";
  const tpWrap = document.querySelector<HTMLElement>("#e_tp_wrap");
  if (tpWrap) wireTextPosPicker(tpWrap, "e_tp", (pos) => { editTextPos = pos; });
  // Note controls: Visible toggle + position picker. These persist immediately
  // via setActionNote (they don't touch the description, which saves on Apply).
  const noteOn = document.querySelector<HTMLInputElement>("#e_note_on");
  const notePosWrap = document.querySelector<HTMLElement>("#e_note_pos");
  let editNotePos = wpForFont?.notePos || "up";
  const applyNote = async () => {
    const vis = !!noteOn?.checked;
    if (notePosWrap) notePosWrap.style.display = vis ? "" : "none";
    try { project = await backend.setActionNote(id, vis, editNotePos); refresh(); }
    catch (err) { alert(String(err)); }
  };
  noteOn?.addEventListener("change", applyNote);
  const npWrap = document.querySelector<HTMLElement>("#e_np_wrap");
  if (npWrap) wireNotePosPicker(npWrap, (pos) => { editNotePos = pos; applyNote(); });
  q("#e_addrole")?.addEventListener("click", () => {
    syncEditRoleLines();
    editRoleLines.push({ role: "", pct: 0 });
    rerenderRoleLines();
  });
  wireRoleLineControls();
  q("#e_details").addEventListener("click", () => openDrill(id));
  q("#e_del").addEventListener("click", async () => {
    project = await backend.removeWorkPackage(id); selectedActionId = null; actionEditMode = false; refresh();
  });
  // Close returns to the read-only show dialog (does not deselect).
  q("#e_closeedit").addEventListener("click", () => { actionEditMode = false; renderSide(); });
}

function openDrill(id: string) {
  const wp = project!.packages.find((w) => w.id === id);
  if (!wp) return;
  const drill = document.querySelector<HTMLDivElement>("#drill")!;
  drill.classList.remove("hidden");
  const roleOpts = (sel: string) =>
    `<option value="">(no role)</option>` +
    (project!.roles).map((r) => `<option value="${esc(r.name)}"${r.name === sel ? " selected" : ""}>${esc(r.name)}</option>`).join("");
  const rows = wp.details.map((d) => `
    <tr data-det="${d.id}">
      <td><input class="d_name" value="${esc(d.name)}" /></td>
      <td><input class="d_desc" value="${esc(d.desc)}" placeholder="description" /></td>
      <td><input class="d_est" value="${esc(d.est)}" size="5" /></td>
      <td><select class="d_role">${roleOpts(d.role)}</select></td>
      <td><label><input type="checkbox" class="d_sum" ${d.summary ? "checked" : ""}/> ∑</label></td>
      <td>${d.start}→${d.end}</td>
      <td><button class="d_apply">save</button><button class="deldet">×</button></td>
    </tr>`).join("");
  drill.innerHTML = `
    <header>
      <strong>${esc(wp.name)}</strong>
      <span>mode:</span>
      <select id="mode"><option value="fixed">fixed</option><option value="derived">derived</option></select>
      <label><input type="checkbox" id="summary" ${wp.summary ? "checked" : ""}/> summary (Summiervorgang)</label>
      <span style="flex:1"></span>
      <button id="adddet">+ detail</button>
      <button id="closedrill">Close</button>
    </header>
    <div class="body">
      <table><thead><tr><th>Title</th><th>Description</th><th>Effort</th><th>Role</th><th title="Sammelvorgang (rollup) bar">∑</th><th>Dates</th><th></th></tr></thead>
      <tbody>${rows || `<tr><td colspan="7">no details yet</td></tr>`}</tbody></table>
    </div>`;
  (drill.querySelector("#mode") as HTMLSelectElement).value = wp.mode;
  drill.querySelector("#mode")!.addEventListener("change", async (e) => {
    project = await backend.setWorkPackageMode(id, (e.target as HTMLSelectElement).value); refresh();
  });
  drill.querySelector("#summary")!.addEventListener("change", async (e) => {
    project = await backend.setWorkPackageSummary(id, (e.target as HTMLInputElement).checked); refresh(); openDrill(id);
  });
  drill.querySelector("#closedrill")!.addEventListener("click", () => drill.classList.add("hidden"));
  drill.querySelector("#adddet")!.addEventListener("click", async () => {
    // add a blank detail row directly (no prompt, which is unreliable in the
    // webview); the user edits it inline and clicks save.
    try {
      project = await backend.addDetail(id, "New detail", wp.start, wp.end, "0", "", "");
      refresh(); openDrill(id);
    } catch (err) { alert(String(err)); }
  });
  drill.querySelectorAll<HTMLButtonElement>(".d_apply").forEach((b) =>
    b.addEventListener("click", async () => {
      const tr = b.closest("tr") as HTMLElement;
      const det = tr.dataset.det!;
      const name = (tr.querySelector(".d_name") as HTMLInputElement).value;
      const desc = (tr.querySelector(".d_desc") as HTMLInputElement).value;
      const est = (tr.querySelector(".d_est") as HTMLInputElement).value;
      const role = (tr.querySelector(".d_role") as HTMLSelectElement).value;
      const summary = (tr.querySelector(".d_sum") as HTMLInputElement).checked;
      try {
        project = await backend.setDetailFields(id, det, name, desc, est, role);
        project = await backend.setDetailSummary(id, det, summary);
        refresh(); openDrill(id);
      } catch (err) { alert(String(err)); }
    }));
  drill.querySelectorAll<HTMLButtonElement>(".deldet").forEach((b) =>
    b.addEventListener("click", async () => {
      const det = (b.closest("tr") as HTMLElement).dataset.det!;
      project = await backend.removeDetail(id, det); refresh(); openDrill(id);
    }));
}

// toolbar wiring
// setModeUI switches the central-object mode everywhere (selector, backend,
// canvas, side panel). Shared by the toolbar selector and the chat shortcuts.
async function setModeUI(m: string) {
  currentMode = m;
  const sel = document.querySelector("#mode") as HTMLSelectElement | null;
  if (sel) sel.value = m;
  await backend.setMode(m);
  gantt.setMode(m);
  if (project) refresh(); else renderSide(); // refresh updates the status line's Mode
  (document.querySelector("#chatlog") as HTMLElement).textContent = "mode: " + m;
}
document.querySelector("#mode")!.addEventListener("change", (e) => {
  setModeUI((e.target as HTMLSelectElement).value);
});

// applyMode reflects a central-object mode in the UI (the #mode selector, the
// canvas, currentMode) WITHOUT calling backend.setMode (which would reset the
// current object). Used to mirror a mode the backend already switched to via a
// chat command (`mode section`, or a bare `add` that keeps section mode), so
// the selector and the status line are never stale.
function applyMode(m: string) {
  if (!m || m === currentMode) return;
  currentMode = m;
  const sel = document.querySelector("#mode") as HTMLSelectElement | null;
  if (sel) sel.value = m;
  gantt.setMode(m);
}
document.querySelector("#helpbtn")!.addEventListener("click", () => openHelp("commands"));

// openHelp renders the multi-page help overlay. Pages are chosen by icon
// buttons in the header (same icon-button style as the view switcher):
//   ⌨  command help     — the CLI command reference (always English)
//   ⇄  relationship help — the dependency-rule table in the GUI language,
//                          using the language's abbreviations (FS.. / EA..)
// `page` selects the initially-shown page.
type HelpPageId = "commands" | "relationships" | "glossary" | "monitoring" | "objects";
async function openHelp(page: HelpPageId) {
  const box = document.querySelector<HTMLDivElement>("#helpbox")!;
  box.classList.remove("hidden");
  const lang = project?.language || "en";
  let t: Record<string, string> = {};
  try { t = await backend.guiStrings(lang); } catch { /* fall back to raw keys */ }
  const tr = (k: string, d: string) => t[k] || d;

  const icon = (p: string, glyph: string, label: string) =>
    `<button class="modeico${page === p ? " active" : ""}" data-help="${p}" title="${esc(label)}">${glyph}</button>`;
  const header = `<header>
      <div class="modeswitch" style="border:0;margin:0;padding:0">
        ${icon("commands", "⌨", tr("help.tab.commands", "Command help"))}
        ${icon("relationships", "⇄", tr("help.tab.relationships", "Relationship help"))}
        ${icon("glossary", "Aa", tr("help.tab.glossary", "Glossary"))}
        ${icon("monitoring", "⏱", tr("help.tab.monitoring", "Monitoring help"))}
        ${icon("objects", "▤", tr("help.tab.objects", "Help on sections and actions"))}
      </div>
      <strong style="margin-left:8px">${esc(page === "commands" ? tr("help.commands", "Commands") : page === "glossary" ? tr("help.glossary", "Glossary") : page === "monitoring" ? tr("help.monitoring", "Monitoring") : page === "objects" ? tr("help.objects", "Sections and actions") : tr("help.relationships", "Relationships"))}</strong>
      <span style="flex:1"></span>
      <button id="closehelp">${esc(tr("help.close", "Close"))}</button>
    </header>`;

  let body: string;
  if (page === "commands") {
    const help = await backend.commandHelp(lang);
    body = `<div class="body">
      <p class="muted">${esc(tr("help.commands.note", "Commands are entered in the command bar at the bottom. The command language is always English."))}</p>
      <pre style="white-space:pre-wrap;font:13px ui-monospace,monospace">${esc(help)}</pre>
    </div>`;
  } else if (page === "glossary") {
    body = glossaryHTML(await backend.glossary(lang), tr);
  } else if (page === "monitoring") {
    body = helpPageHTML(await backend.monitoringHelp(lang));
  } else if (page === "objects") {
    body = helpPageHTML(await backend.objectHelp(lang));
  } else {
    const rules = await backend.relationshipRules(lang);
    const rows = rules.map((r) => `<tr>
        <td style="text-align:center"><b>${esc(r.abbr)}</b></td>
        <td>${esc(r.name)}</td>
        <td style="white-space:nowrap">${esc(r.rule)}</td>
        <td>${esc(r.example)}</td>
      </tr>`).join("");
    body = `<div class="body">
      <table class="help-rel">
        <thead><tr>
          <th>${esc(tr("help.col.abbr", "Abbr."))}</th>
          <th>${esc(tr("help.col.rel", "Relationship"))}</th>
          <th>${esc(tr("help.col.rule", "Rule"))}</th>
          <th>${esc(tr("help.col.example", "Example"))}</th>
        </tr></thead>
        <tbody>${rows}</tbody>
      </table>
    </div>`;
  }
  box.innerHTML = header + body;
  box.querySelector("#closehelp")!.addEventListener("click", () => box.classList.add("hidden"));
  box.querySelectorAll<HTMLButtonElement>(".modeico").forEach((b) =>
    b.addEventListener("click", () => openHelp(b.dataset.help as HelpPageId)));
}
document.querySelector("#link")!.addEventListener("click", () => gantt.setLinkMode(!gantt.isLinkMode()));
document.querySelector("#baselines")!.addEventListener("click", () => openBaselines());
function openMonitorPage() {
  if (!project) return;
  void openMonitor({ getProject: () => project, setProject: (p) => { project = p; }, refresh });
}
document.querySelector("#monitor")!.addEventListener("click", openMonitorPage);
async function openBaselines() {
  const box = document.querySelector<HTMLDivElement>("#baselinebox")!;
  box.classList.remove("hidden");
  const list = await backend.listBaselines();
  const rows = list.map((b: any) =>
    `<tr><td>${esc(b.name)}</td><td>${b.createdAt}</td><td>${b.totalEst}</td><td>${b.tasks}</td>
      <td><button class="bl_cmp" data-n="${esc(b.name)}">compare</button>
          <button class="bl_del" data-n="${esc(b.name)}">×</button></td></tr>`).join("");
  box.innerHTML = `
    <header><strong>Baselines</strong><span style="flex:1"></span>
      <button id="bl_new" class="primary">Capture now</button><button id="bl_close">Close</button></header>
    <div class="body">
      <table><thead><tr><th>Name</th><th>Created</th><th>Total</th><th>Tasks</th><th></th></tr></thead>
      <tbody>${rows || `<tr><td colspan="5">no baselines yet — click "Capture now"</td></tr>`}</tbody></table>
      <div id="bl_cmpout"></div>
    </div>`;
  box.querySelector("#bl_close")!.addEventListener("click", () => box.classList.add("hidden"));
  box.querySelector("#bl_new")!.addEventListener("click", async () => {
    const n = "Baseline " + (list.length + 1);
    await backend.captureBaseline(n); openBaselines();
  });
  box.querySelectorAll<HTMLButtonElement>(".bl_del").forEach((b) =>
    b.addEventListener("click", async () => { await backend.deleteBaseline(b.dataset.n!); openBaselines(); }));
  box.querySelectorAll<HTMLButtonElement>(".bl_cmp").forEach((b) =>
    b.addEventListener("click", async () => {
      const v = await backend.compareBaseline(b.dataset.n!);
      const changed = v.filter((r: any) => r.status || r.startDays || r.endDays || (r.estDelta && r.estDelta !== "0"));
      const out = document.querySelector<HTMLDivElement>("#bl_cmpout")!;
      if (changed.length === 0) { out.innerHTML = `<p class="muted">No changes vs ${esc(b.dataset.n!)}.</p>`; return; }
      out.innerHTML = `<h3>Variance vs ${esc(b.dataset.n!)}</h3>
        <table><thead><tr><th>WBS</th><th>Name</th><th>Δ start (d)</th><th>Δ end (d)</th><th>Δ est</th><th></th></tr></thead>
        <tbody>${changed.map((r: any) =>
          `<tr><td>${esc(r.wbs || "")}</td><td>${esc(r.name || "")}</td><td>${r.startDays}</td><td>${r.endDays}</td><td>${esc(r.estDelta)}</td><td>${r.status}</td></tr>`).join("")}</tbody></table>`;
    }));
}
document.querySelector("#details")!.addEventListener("click", () => openProjectDialog());
function openProjectDialog() {
  if (!project) return;
  const box = document.querySelector<HTMLDivElement>("#projbox")!;
  box.classList.remove("hidden");
  box.innerHTML = `
    <header><strong>Project details</strong><span style="flex:1"></span>
      <button id="projclose">Close</button></header>
    <div class="body">
      <p><label>Name (max 16)<br/><input id="p_name" maxlength="16" size="18" value="${esc(project.title)}" /></label></p>
      <p><label>Description<br/><textarea id="p_desc" rows="4" cols="50">${esc(project.description)}</textarea></label></p>
      <p><label>Language
        <select id="p_language">
          <option value="en"${project.language !== "de" ? " selected" : ""}>English</option>
          <option value="de"${project.language === "de" ? " selected" : ""}>Deutsch</option>
        </select></label>
        <span class="muted">GUI language &amp; number/date format (the command bar stays English)</span></p>
      <p><label>Start <input id="p_start" type="date" value="${project.start}" /></label>
         &nbsp;<label>End <input id="p_end" type="date" value="${project.end}" /></label></p>
      <p><label>Arrows
        <select id="p_arrow">
          <option value="spline"${project.arrowStyle !== "square" ? " selected" : ""}>rounded (splines)</option>
          <option value="square"${project.arrowStyle === "square" ? " selected" : ""}>square</option>
        </select></label></p>
      <fieldset style="border:1px solid var(--line);border-radius:6px;padding:6px 10px;margin:6px 0">
        <legend style="font-size:12px;color:#666">Report / export</legend>
        <p><label>Format
          <select id="p_reportformat">
            <option value="pdf"${project.reportFormat !== "png" ? " selected" : ""}>PDF report</option>
            <option value="png"${project.reportFormat === "png" ? " selected" : ""}>PNG image</option>
          </select></label></p>
        <p><label>Grid
          <select id="p_reportgrid">
            <option value="day"${project.reportGrid === "day" ? " selected" : ""}>days</option>
            <option value="week"${project.reportGrid === "week" ? " selected" : ""}>weeks</option>
            <option value="month"${project.reportGrid !== "day" && project.reportGrid !== "week" ? " selected" : ""}>months</option>
          </select></label></p>
        <p><label><input type="checkbox" id="p_reporttoday" ${project.reportTodayLine !== false ? "checked" : ""}/> Today line</label></p>
        <p><label><input type="checkbox" id="p_reportnotes" ${project.reportShowNotes !== false ? "checked" : ""}/> Show notes</label></p>
      </fieldset>
      <p><label><input type="checkbox" id="p_showweeks" ${project.showWeekNumbers ? "checked" : ""}/> Show week numbers (calendar weeks in the date row, when there is room)</label></p>
      <p><label><input type="checkbox" id="p_showdetails" ${project.showDetails ? "checked" : ""}/> Show Details (draw level-3 detail bars in the main view)</label></p>
      <p><label><input type="checkbox" id="p_showphases" ${project.showPhases ? "checked" : ""}/> Show Phases (the top Phases view lane)</label></p>
      <p><label><input type="checkbox" id="p_showprogress" ${project.showProgress ? "checked" : ""}/> Show Progress (progress marker and contour curve in each action bar)</label></p>
      <p><label><input type="checkbox" id="p_autosave_on" ${project.autosaveEnabled ? "checked" : ""}/> <b>Autosave</b> (periodic)</label></p>
      <p><label>Autosave (minutes) <input id="p_autosave" type="number" min="0" max="99" style="width:56px" value="${project.autosaveMinutes ?? 5}" ${project.autosaveEnabled ? "" : "disabled"} /></label>
         <span class="muted">0 = after every change; otherwise 5–99 (1–4 round up to 5)</span></p>
      <p><label><input type="checkbox" id="p_autosaveexit" ${project.autosaveOnExit ? "checked" : ""}/> Autosave on exit</label></p>
      <p><label><input type="checkbox" id="p_saveversions" ${project.saveVersions ? "checked" : ""}/> Save versions</label>
         &nbsp;<label>keep <input id="p_versioncount" type="number" min="5" max="10" style="width:52px" value="${project.versionCount ?? 5}" ${project.saveVersions ? "" : "disabled"} /></label>
         <span class="muted">rotated numbered copies on autosave (5–10)</span></p>
      <p class="muted">Saved to: <span id="p_savepath">(not saved yet)</span></p>
      <p><button id="p_save" class="primary">Apply</button></p>
    </div>`;
  // show the current save directory/file (single .ptplan.json)
  backend.savePath().then((sp) => {
    const el = box.querySelector("#p_savepath");
    if (el) el.textContent = sp || "(not saved yet — use Save)";
  }).catch(() => {});
  box.querySelector("#projclose")!.addEventListener("click", () => box.classList.add("hidden"));
  box.querySelector("#p_language")!.addEventListener("change", async (e) => {
    project = await backend.setLanguage((e.target as HTMLSelectElement).value);
    refresh();
  });
  box.querySelector("#p_arrow")!.addEventListener("change", async (e) => {
    project = await backend.setArrowStyle((e.target as HTMLSelectElement).value); refresh();
  });
  // Report / export config: format, grid interval, today line, show notes.
  const applyReportConfig = async () => {
    const format = (box.querySelector("#p_reportformat") as HTMLSelectElement).value;
    const grid = (box.querySelector("#p_reportgrid") as HTMLSelectElement).value;
    const today = (box.querySelector("#p_reporttoday") as HTMLInputElement).checked;
    const notes = (box.querySelector("#p_reportnotes") as HTMLInputElement).checked;
    project = await backend.setReportConfig(format, today, grid, notes);
    refresh(); // grid/today/notes also drive the live canvas
  };
  box.querySelector("#p_reportformat")!.addEventListener("change", applyReportConfig);
  box.querySelector("#p_reportgrid")!.addEventListener("change", applyReportConfig);
  box.querySelector("#p_reporttoday")!.addEventListener("change", applyReportConfig);
  box.querySelector("#p_reportnotes")!.addEventListener("change", applyReportConfig);
  box.querySelector("#p_showweeks")!.addEventListener("change", async (e) => {
    project = await backend.setShowWeekNumbers((e.target as HTMLInputElement).checked); refresh();
  });
  box.querySelector("#p_showdetails")!.addEventListener("change", async (e) => {
    const on = (e.target as HTMLInputElement).checked;
    project = await backend.setViewOptions(on, project!.showRoleActions, project!.showPhases, project!.showProgress); refresh();
  });
  box.querySelector("#p_showphases")!.addEventListener("change", async (e) => {
    const on = (e.target as HTMLInputElement).checked;
    project = await backend.setViewOptions(project!.showDetails, project!.showRoleActions, on, project!.showProgress); refresh();
  });
  box.querySelector("#p_showprogress")!.addEventListener("change", async (e) => {
    const on = (e.target as HTMLInputElement).checked;
    project = await backend.setViewOptions(project!.showDetails, project!.showRoleActions, project!.showPhases, on); refresh();
  });
  // Autosave config: on/off + minutes (0 or 5..99) + on-exit + versioning.
  const applyAutosave = async () => {
    const enabled = (box.querySelector("#p_autosave_on") as HTMLInputElement).checked;
    const mins = Number((box.querySelector("#p_autosave") as HTMLInputElement).value) || 0;
    const onExit = (box.querySelector("#p_autosaveexit") as HTMLInputElement).checked;
    const saveVersions = (box.querySelector("#p_saveversions") as HTMLInputElement).checked;
    const versionCount = Number((box.querySelector("#p_versioncount") as HTMLInputElement).value) || 5;
    project = await backend.setAutosaveConfig(enabled, mins, onExit, saveVersions, versionCount);
    // reflect clamped values (minutes 1..4 -> 5, versions -> 5..10) + enabled state
    (box.querySelector("#p_autosave") as HTMLInputElement).value = String(project!.autosaveMinutes ?? 5);
    (box.querySelector("#p_autosave") as HTMLInputElement).disabled = !enabled;
    (box.querySelector("#p_versioncount") as HTMLInputElement).value = String(project!.versionCount ?? 5);
    (box.querySelector("#p_versioncount") as HTMLInputElement).disabled = !saveVersions;
    refresh();
  };
  box.querySelector("#p_autosave_on")!.addEventListener("change", applyAutosave);
  box.querySelector("#p_autosave")!.addEventListener("change", applyAutosave);
  box.querySelector("#p_autosaveexit")!.addEventListener("change", applyAutosave);
  box.querySelector("#p_saveversions")!.addEventListener("change", applyAutosave);
  box.querySelector("#p_versioncount")!.addEventListener("change", applyAutosave);
  box.querySelector("#p_save")!.addEventListener("click", async () => {
    const name = (box.querySelector("#p_name") as HTMLInputElement).value;
    const desc = (box.querySelector("#p_desc") as HTMLTextAreaElement).value;
    const start = (box.querySelector("#p_start") as HTMLInputElement).value;
    const end = (box.querySelector("#p_end") as HTMLInputElement).value;
    try { project = await backend.setProjectDetails(name, desc, start, end); refresh(); box.classList.add("hidden"); }
    catch (err) { alert(String(err)); }
  });
}
document.querySelector("#zoom")!.addEventListener("change", async (e) => {
  const z = (e.target as HTMLSelectElement).value as Zoom;
  gantt.setZoom(z);
  // persist the zoom into the model so it is saved with the project
  try { project = await backend.setZoom(z); } catch { /* ignore */ }
  if (project) refresh(); // keep the status line's "View:" in sync
});
document.querySelector("#unit")!.addEventListener("change", async (e) => {
  project = await backend.setConfig((e.target as HTMLSelectElement).value,
    Number((document.querySelector("#hpd") as HTMLInputElement).value)); refresh();
});
document.querySelector("#hpd")!.addEventListener("change", async () => {
  project = await backend.setConfig((document.querySelector("#unit") as HTMLSelectElement).value,
    Number((document.querySelector("#hpd") as HTMLInputElement).value)); refresh();
});
document.querySelector("#open")!.addEventListener("click", async () => {
  try {
    // Open auto-saves a dirty project first (must save, no discard), handled by
    // the backend NewProject/Open rule. We save here before opening so nothing
    // is lost, then open.
    await saveIfDirty();
    project = await backend.openDialog(); refresh();
  }
  catch (err) { alert("Open failed: " + String(err)); }
});

// saveIfDirty saves the current project when it has unsaved changes, WITHOUT a
// dialog: to the known path, or to a Downloads fallback when never saved. This
// is the "must save, no discard" rule used before New/Open. No-op when clean.
async function saveIfDirty(): Promise<void> {
  let dirty = false;
  try { dirty = await backend.isDirty(); } catch { dirty = false; }
  if (!dirty) return;
  await backend.saveSmart();
}

// #save = quick-save (disk icon). Saves to the current file; if the project was
// never saved it falls back to Save As (native dialog). Greyed when not dirty.
document.querySelector("#save")!.addEventListener("click", async () => {
  try {
    const path = await backend.savePath();
    if (path) { await backend.saveSmart(); project = await backend.getProject(); }
    else { project = await backend.saveDialog(); }
    refresh();
  } catch (err) { alert("Save failed: " + String(err)); }
});

// #saveas = always prompt a native dialog to choose a (new) location.
document.querySelector("#saveas")!.addEventListener("click", async () => {
  try { project = await backend.saveDialog(); refresh(); }
  catch (err) { alert("Save failed: " + String(err)); }
});

document.querySelector("#new")!.addEventListener("click", async () => {
  try {
    // New blanks the slate. A dirty project is saved first by the backend
    // (must save, no discard) — to its path or a Downloads fallback.
    project = await backend.newProject(true);
    selectedActionId = selectedSectionId = null; selectedDetail = null;
    refresh();
  } catch (err) { alert("New failed: " + String(err)); }
});
document.querySelector("#pdf")!.addEventListener("click", async () => {
  try {
    // Export the report using the LIVE canvas image so the output matches the
    // screen exactly (notes, spline/square arrows, WP-n headers, today line,
    // grid). The report ALWAYS prints the details, regardless of the Show
    // Details toggle: one whole-project overview image, then a separate image
    // for each work package's detail (drill) view. The format (pdf|png) comes
    // from the project's report config. Export at a high raster scale so the
    // Gantt is sharp when the PDF scales it up to the full A4 page width.
    const imgs = await gantt.exportReportImages(3);
    const detailsJSON = JSON.stringify(imgs.details.map((d) => ({ name: d.name, png: d.png })));
    const out = await backend.exportReport(imgs.overview, detailsJSON);
    if (out) alert("Report written to " + out);
  }
  catch (err) { alert("Export failed: " + String(err)); }
});
// Global ⌘S / Ctrl-S = quick-save (unless typing in a field, where ⌘S is the
// chat "section mode" shortcut). Triggers the #save button handler.
window.addEventListener("keydown", (e) => {
  if ((e.metaKey || e.ctrlKey) && !e.shiftKey && !e.altKey && e.key.toLowerCase() === "s") {
    const el = document.activeElement;
    const typing = el && (el.tagName === "INPUT" || el.tagName === "TEXTAREA" || (el as HTMLElement).isContentEditable);
    if (typing) return; // let the chat-box ⌘S handler run
    e.preventDefault();
    (document.querySelector("#save") as HTMLButtonElement | null)?.click();
  }
});

const chatin = document.querySelector<HTMLInputElement>("#chatin")!;
const chatHistory: string[] = [];
let histIdx = -1;
chatin.addEventListener("keydown", async (e) => {
  // Command (⌘) mode shortcuts while editing in the chat box:
  //   ⌘A -> action mode, ⌘S -> section mode, ⌘R -> role ("row") mode.
  if (e.metaKey && !e.shiftKey && !e.altKey && !e.ctrlKey) {
    const k = e.key.toLowerCase();
    if (k === "a" || k === "s" || k === "r") {
      e.preventDefault();
      const m = k === "a" ? "action" : k === "s" ? "section" : "role";
      setModeUI(m);
      return;
    }
  }
  if (e.key === "ArrowUp") {
    e.preventDefault();
    if (chatHistory.length === 0) return;
    histIdx = histIdx < 0 ? chatHistory.length - 1 : Math.max(0, histIdx - 1);
    chatin.value = chatHistory[histIdx];
    return;
  }
  if (e.key === "ArrowDown") {
    e.preventDefault();
    if (histIdx < 0) return;
    histIdx++;
    if (histIdx >= chatHistory.length) { histIdx = -1; chatin.value = ""; }
    else chatin.value = chatHistory[histIdx];
    return;
  }
  if (e.key !== "Enter") return;
  const line = chatin.value.trim(); if (!line) return;
  chatHistory.push(line); histIdx = -1;
  try {
    const res = await backend.chat(line);
    (document.querySelector("#chatlog") as HTMLElement).textContent = res;
    // syncAfterChat also re-syncs the mode (the command may have changed it,
    // e.g. `mode section`, or a bare `add` that keeps section mode), so the
    // selector + status line are never stale, THEN draws + follows selection.
    await syncAfterChat({
      getProject: () => backend.getProject(), setProject: (p) => { project = p; }, refresh,
      selection: () => backend.selection(), selectAction, selectSection, selectMilestone,
      mode: () => backend.mode(), applyMode,
    });
    chatin.value = "";
  } catch (err) { (document.querySelector("#chatlog") as HTMLElement).textContent = String(err); }
});

// --- batch mode: a 5-line box; every line is a command, run one after another ---
const batchin = document.querySelector<HTMLTextAreaElement>("#batchin")!;
const batchbtn = document.querySelector<HTMLButtonElement>("#batchbtn")!;
const batchrun = document.querySelector<HTMLButtonElement>("#batchrun")!;
function setBatchMode(on: boolean) {
  batchin.classList.toggle("hidden", !on);
  batchrun.classList.toggle("hidden", !on);
  chatin.classList.toggle("hidden", on);
  batchbtn.classList.toggle("active", on);
  batchbtn.setAttribute("aria-pressed", String(on));
  (on ? batchin : chatin).focus();
}
async function runBatch() {
  const script = batchin.value;
  const log = document.querySelector("#chatlog") as HTMLElement;
  batchrun.disabled = true;
  try {
    const res = await backend.chatBatch(script);
    log.textContent = batchSummary(res);
    log.title = res.log.join("\n"); // hover for the line-by-line log
    await syncAfterChat({
      getProject: () => backend.getProject(), setProject: (p) => { project = p; }, refresh,
      selection: () => backend.selection(), selectAction, selectSection, selectMilestone,
      mode: () => backend.mode(), applyMode,
    });
    if (res.failed === 0 && res.stopped === 0) batchin.value = ""; // all done; on a stop the script stays for fixing
  } catch (err) { log.textContent = String(err); }
  finally { batchrun.disabled = false; }
}
batchbtn.addEventListener("click", () => setBatchMode(batchin.classList.contains("hidden")));
batchrun.addEventListener("click", runBatch);
batchin.addEventListener("keydown", (e) => {
  if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) { e.preventDefault(); runBatch(); }
  else if (e.key === "Escape") setBatchMode(false);
});

function esc(s: string): string {
  return s.replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]!));
}

// inclusive day count between two YYYY-MM-DD dates
function dayCount(start: string, end: string): number {
  const s = Date.parse(start + "T00:00:00Z"), e = Date.parse(end + "T00:00:00Z");
  if (!isFinite(s) || !isFinite(e) || e < s) return 0;
  return Math.round((e - s) / 86400000) + 1;
}

function showStatus(msg: string, detail = "") {
  document.querySelector("#canvas")!.innerHTML =
    `<div style="padding:20px;color:#666;font:13px system-ui">
       <p><strong>${esc(msg)}</strong></p>
       ${detail ? `<pre style="white-space:pre-wrap;color:#a40">${esc(detail)}</pre>` : ""}
       <p>If this persists, reload the window (Cmd+R).</p>
     </div>`;
}

// Surface any uncaught error/rejection WITH stack + location so the real
// source line is visible. Ignore errors originating in Vite's HMR client
// (@vite/client) — those are dev-server WebSocket hiccups, not app failures,
// and must not paint over a working UI.
function fromVite(s: string): boolean { return /@vite\/client|vite.*hmr/i.test(s); }

window.addEventListener("error", (e) => {
  const loc = `${e.filename}:${e.lineno}:${e.colno}`;
  if (fromVite(loc) || fromVite(e.message)) return;
  showStatus("Frontend error", `${e.message}\n${loc}\n${e.error?.stack ?? ""}`);
});
window.addEventListener("unhandledrejection", (e) => {
  const r: any = (e as PromiseRejectionEvent).reason;
  const stack = r?.stack ? "\n" + r.stack : "";
  if (fromVite(stack) || fromVite(String(r?.message ?? r))) return;
  showStatus("Unhandled rejection", `${r?.message ?? String(r)}${stack}`);
});

// initial load, step-by-step so we know exactly which call fails
(async () => {
  try {
    await whenReady();
  } catch (err) {
    showStatus("Step 1: whenReady failed", errStr(err));
    return;
  }
  try {
    project = await backend.getProject();
  } catch (err) {
    showStatus("Step 2: getProject failed", errStr(err));
    return;
  }
  try {
    refresh();
    setupAutosave();
  } catch (err) {
    showStatus("Step 3: refresh/render failed", errStr(err));
  }
})();

function errStr(err: unknown): string {
  const e: any = err;
  return `${e?.message ?? String(e)}${e?.stack ? "\n" + e.stack : ""}`;
}
