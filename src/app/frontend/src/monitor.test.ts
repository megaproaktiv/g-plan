import { beforeAll, beforeEach, expect, mock, test } from "bun:test";
import { Window } from "happy-dom";

// ---- mocks (installed before monitor.ts is imported) ----
type Listener = (ev: { data: unknown }) => void;
const listeners: Listener[] = [];
mock.module("@wailsio/runtime", () => ({
  Events: { On: (_n: string, cb: Listener) => { listeners.push(cb); return () => { listeners.splice(listeners.indexOf(cb), 1); }; } },
}));

let project: any;
let summary: any = null;
let setMonitorError: string | null = null;
let syncGate: (() => void) | null = null;
let testGate: ((v: "ok" | "fail") => void) | null = null;
const calls: string[] = [];

const backendMock = {
  monitorSystems: async () => [
    { id: "aerion", label: "Aerion", available: true },
    { id: "csv", label: "CSV (not yet implemented)", available: false },
  ],
  getMonitorSummary: async () => summary,
  setMonitor: async (enabled: boolean, system: string, projectId: string) => {
    calls.push(`setMonitor(${enabled},${system},${projectId})`);
    if (setMonitorError) throw new Error(setMonitorError);
    project = { ...project, monitor: { enabled, system, projectId } };
    return project;
  },
  setMonitorDefaultAction: async (id: string) => {
    calls.push(`setDefault(${id})`);
    if (setMonitorError) throw new Error(setMonitorError);
    project = { ...project, monitor: { ...project.monitor, defaultAction: id } };
    return project;
  },
  setMonitorRange: async (before: boolean, after: boolean) => {
    calls.push(`setRange(${before},${after})`);
    if (setMonitorError) throw new Error(setMonitorError);
    const mm = { ...project.monitor, includeBefore: before, includeAfter: after };
    mm.syncFrom = before ? "2025-12-01" : "2026-01-01";
    mm.syncTo = after ? "2026-04-01" : "2026-03-01";
    project = { ...project, monitor: mm };
    return project;
  },
  getSystemConfig: async () => ({ baseUrl: "https://x.example", username: "me", clientId: "cid", passwordSet: true, clientSecretSet: true }),
  saveSystemConfig: async (...a: string[]) => { calls.push("saveSystemConfig:" + a.join("|")); },
  testSystemConnection: async () => {
    calls.push("test");
    const r = await new Promise<"ok" | "fail">((res) => { testGate = res; });
    if (r === "fail") throw new Error("authentication failed (HTTP 401)");
    return { projectName: "Demo", status: 1, start: "2026-01-01", end: "2026-12-31" };
  },
  cancelSync: async () => { calls.push("cancel"); },
  syncMonitor: async () => {
    await new Promise<void>((r) => { syncGate = r; });
    summary = makeSummary();
    return summary;
  },
};
const originalSync = backendMock.syncMonitor; // tests may replace it; restored in beforeEach
mock.module("./backend", () => ({ backend: backendMock }));

function makeProject(over: Record<string, unknown> = {}): any {
  const wp = (id: string, cluster: string, externalId = "") =>
    ({ id, name: id, displayId: id.toUpperCase(), cluster, externalId, est: "10", estHours: "80", estDays: "10" });
  return {
    unit: "hours", sectionNums: { Design: 1, Build: 2, Empty: 3 },
    packages: [wp("a", "Design", "WP-1"), wp("b", "Design"), wp("c", "Build"), wp("d", "Phases", "X")],
    monitor: { enabled: false, system: "", projectId: "", defaultAction: "" }, ...over,
  };
}
function makeSummary(): any {
  return {
    sections: [
      { name: "Design", actions: 2, tracked: 1, est: "30", estTracked: "10", actual: "3,5", entries: 3, remaining: "6,5", remainingPct: "65 %",
        details: [{ id: "a", est: "10", remaining: "6,5", remainingPct: "65 %", externalId: "WP-1", name: "<b>a</b>", actual: "3,5", entries: 3, isDefault: false, defaultActual: "0", defaultEntries: 0,
          actualHours: "3,5", actualDays: "0,44", matchedHours: "3,5", matchedDays: "0,44", matchedEntries: 3, defaultHours: "0", defaultDays: "0" }] },
      { name: "Build", actions: 1, tracked: 0, est: "5", estTracked: "0", actual: "0", entries: 0, details: [] },
    ],
    est: "35", estTracked: "10", actual: "3,5", entries: 3, unmatchedActual: "0,42", unmatchedEntries: 2,
    fetched: 5, from: "2026-01-01", to: "2026-03-01", at: "2026-11-04 12:00",
  };
}

let mon: typeof import("./monitor");
beforeAll(async () => {
  const win = new Window();
  const g = globalThis as unknown as Record<string, unknown>;
  g.window = win; g.document = win.document; g.HTMLElement = win.HTMLElement;
  mon = await import("./monitor");
});
beforeEach(() => {
  backendMock.syncMonitor = originalSync;
  document.body.innerHTML = "";
  project = makeProject(); summary = null; setMonitorError = null; syncGate = null; testGate = null; calls.length = 0; listeners.length = 0;
});

const host = () => ({ getProject: () => project, setProject: (p: any) => { project = p; }, refresh: () => calls.push("refresh") });
const $ = <T extends Element>(s: string) => document.querySelector<T>(s)!;
const tick = () => new Promise((r) => setTimeout(r, 0));

test("sectionStates: green only when an action has an External ID; empty and unnumbered sections listed", () => {
  const s = mon.sectionStates(project);
  expect(s.map((x) => x.name)).toEqual(["Design", "Build", "Empty"]);
  expect(s.find((x) => x.name === "Design")).toMatchObject({ tracked: 1, actions: 2, label: "1 Design" });
  expect(s.find((x) => x.name === "Build")!.tracked).toBe(0);
  expect(s.find((x) => x.name === "Empty")).toMatchObject({ tracked: 0, actions: 0 });
  expect(s.find((x) => x.name === "Phases")).toBeUndefined(); // the Phases lane is not monitored
});

test("result table: sections, project sum, unmatched row, and HTML is escaped", () => {
  const html = mon.resultTableHTML(project, makeSummary());
  expect(html).toContain("1 Design");
  expect(html).toContain("2 Build");
  expect(html).toContain("mon-total");
  expect(html).toContain("0,42");
  expect(html).not.toContain("<b>a</b>"); // booking-derived/user text must not become markup
  expect(html).toContain("&lt;b&gt;a&lt;/b&gt;");
  expect(html).toContain("5 bookings fetched");
});

test("page: sections are green/grey, CSV is listed but disabled, Synchronize needs monitoring on", async () => {
  await mon.openMonitor(host());
  const greens = [...document.querySelectorAll(".mon-green")].map((e) => e.textContent!.trim());
  expect(greens.length).toBe(1); // Design (the Phases lane is not monitored)
  expect(document.querySelectorAll(".mon-grey").length).toBe(2); // Build, Empty
  const csv = $<HTMLOptionElement>('#mon_sys option[value="csv"]');
  expect(csv.disabled).toBe(true);
  expect($<HTMLOptionElement>('#mon_sys option[value="aerion"]').disabled).toBe(false);
  expect($<HTMLButtonElement>("#mon_sync").disabled).toBe(true);
});

test("switching monitoring on without system/id shows the error and stays off", async () => {
  setMonitorError = "monitoring needs an external system type";
  await mon.openMonitor(host());
  const on = $<HTMLInputElement>("#mon_on");
  on.checked = true;
  on.dispatchEvent(new window.Event("change", { bubbles: true }) as unknown as Event);
  await tick();
  expect($("#mon_msg").textContent).toContain("needs an external system type");
  expect($<HTMLInputElement>("#mon_on").checked).toBe(false);
  expect($<HTMLButtonElement>("#mon_sync").disabled).toBe(true);
});

test("enable, then synchronize: progress is shown and the result table appears", async () => {
  project = makeProject({ monitor: { enabled: false, system: "aerion", projectId: "42" } });
  await mon.openMonitor(host());
  const on = $<HTMLInputElement>("#mon_on");
  on.checked = true;
  on.dispatchEvent(new window.Event("change", { bubbles: true }) as unknown as Event);
  await tick();
  expect(calls).toContain("setMonitor(true,aerion,42)");
  expect($<HTMLButtonElement>("#mon_sync").disabled).toBe(false);

  $<HTMLButtonElement>("#mon_sync").click();
  await tick();
  expect($<HTMLButtonElement>("#mon_sync").disabled).toBe(true);
  expect($<HTMLButtonElement>("#mon_cancel").disabled).toBe(false);
  listeners.forEach((l) => l({ data: { fetched: 2500, total: 5000 } }));
  expect($("#mon_cnt").textContent).toBe("2500 of 5000 entries fetched");
  expect($<HTMLProgressElement>("#mon_prog").value).toBe(2500);
  expect($<HTMLProgressElement>("#mon_prog").max).toBe(5000);

  syncGate!();
  await tick(); await tick();
  expect(document.querySelector("#mon_result .mon-total")).not.toBeNull();
  expect($("#mon_msg").textContent).toContain("5 entries fetched");
  expect(listeners.length).toBe(0); // progress listener released
});

test("a failed sync shows the error and re-enables the button", async () => {
  project = makeProject({ monitor: { enabled: true, system: "aerion", projectId: "42" } });
  backendMock.syncMonitor = async () => { throw new Error("Aerion is not configured (missing: username) - use Configure System"); };
  await mon.openMonitor(host());
  $<HTMLButtonElement>("#mon_sync").click();
  await tick(); await tick();
  expect($("#mon_msg").textContent).toContain("not configured");
  expect($("#mon_msg").className).toContain("mon-bad");
  expect($<HTMLButtonElement>("#mon_sync").disabled).toBe(false);
});

test("Configure System: secrets are never prefilled; saving sends the typed values", async () => {
  project = makeProject({ monitor: { enabled: false, system: "aerion", projectId: "" } });
  await mon.openMonitor(host());
  $<HTMLButtonElement>("#mon_cfg").click();
  await tick();
  expect($<HTMLInputElement>("#sc_pw").type).toBe("password");
  expect($<HTMLInputElement>("#sc_cs").type).toBe("password");
  expect($<HTMLInputElement>("#sc_pw").value).toBe("");
  expect($<HTMLInputElement>("#sc_cs").value).toBe("");
  expect($<HTMLInputElement>("#sc_pw").placeholder).toContain("leave empty to keep");
  expect($<HTMLInputElement>("#sc_url").value).toBe("https://x.example");
  expect(document.body.innerHTML).not.toContain("secret-value");
  $<HTMLInputElement>("#sc_pw").value = "newpw";
  $<HTMLButtonElement>("#sc_save").click();
  await tick();
  expect(calls.find((c) => c.startsWith("saveSystemConfig:"))).toBe("saveSystemConfig:aerion|https://x.example|me|newpw|cid|");
  expect($("#sysbox").classList.contains("hidden")).toBe(true);
});

test("default action: actions grouped by section (no Phases lane), saved on change", async () => {
  await mon.openMonitor(host());
  const groups = [...document.querySelectorAll<HTMLOptGroupElement>("#mon_def optgroup")].map((g) => g.label);
  expect(groups).toEqual(["1 Design", "2 Build"]); // Empty has no actions; Phases is not a work package
  const opts = [...document.querySelectorAll<HTMLOptionElement>("#mon_def option")].map((o) => o.value);
  expect(opts).toEqual(["", "a", "b", "c"]);
  expect(document.querySelector<HTMLOptionElement>('#mon_def option[value="a"]')!.textContent).toBe("A a");
  const sel = $<HTMLSelectElement>("#mon_def");
  sel.value = "c";
  sel.dispatchEvent(new window.Event("change", { bubbles: true }) as unknown as Event);
  await tick();
  expect(calls).toContain("setDefault(c)");
  // happy-dom ignores the parse-time `selected` attribute in select.value (real
  // browsers honour it), so assert the attribute that the browser will use
  expect(document.querySelector<HTMLOptionElement>("#mon_def option[selected]")!.value).toBe("c");
  // the section that holds the default action is marked in the section list
  expect([...document.querySelectorAll(".mon-secs li")].some((li) => li.textContent!.includes("contains the default action"))).toBe(true);
});

test("default action: error is shown and selection reverts", async () => {
  setMonitorError = "not an action";
  await mon.openMonitor(host());
  const sel = $<HTMLSelectElement>("#mon_def");
  sel.value = "a";
  sel.dispatchEvent(new window.Event("change", { bubbles: true }) as unknown as Event);
  await tick();
  expect($("#mon_msg").textContent).toContain("not an action");
  expect(document.querySelector<HTMLOptionElement>("#mon_def option[selected]")!.value).toBe(""); // reverted to none
});

test("default action: a deleted action is shown, not silently treated as none", async () => {
  project = makeProject({ monitor: { enabled: false, system: "", projectId: "", defaultAction: "gone" } });
  await mon.openMonitor(host());
  const sel = document.querySelector<HTMLOptionElement>("#mon_def option[selected]")!;
  expect(sel.value).toBe("gone");
  expect(sel.textContent).toContain("no longer exists");
});

test("result table: the default action gets a marked row with its unassigned share, unmatched row disappears", () => {
  const s = makeSummary();
  s.defaultAction = "b"; s.defaultActual = "0,42"; s.defaultEntries = 2;
  s.sections[0].details.push({ id: "b", externalId: "", name: "b", actual: "0,42", entries: 2, isDefault: true, defaultActual: "0,42", defaultEntries: 2,
    actualHours: "0,42", actualDays: "0,05", matchedHours: "0", matchedDays: "0", matchedEntries: 0, defaultHours: "0,42", defaultDays: "0,05" });
  s.unmatchedEntries = 0; s.unmatchedActual = "0";
  const html = mon.resultTableHTML(project, s);
  expect(html).toContain("(default action)");
  expect(html).toContain("<td>b</td>");
  expect(html).toContain("mon-defact");
  expect(html).toContain("of which unassigned bookings (default action)");
  expect(html).not.toContain("mon-unmatched");
  // without a default the separate line stays and no default rows appear
  const plain = mon.resultTableHTML(project, makeSummary());
  expect(plain).toContain("mon-unmatched");
  expect(plain).not.toContain("default action");
  expect(plain).not.toContain("(default)");
});

test("Test: button is disabled and 'Testing connection' is shown under the project line while it runs", async () => {
  project = makeProject({ monitor: { enabled: false, system: "aerion", projectId: "42", defaultAction: "" } });
  await mon.openMonitor(host());
  expect($<HTMLButtonElement>("#mon_test").disabled).toBe(false);
  $<HTMLButtonElement>("#mon_test").click();
  await tick();
  expect($<HTMLButtonElement>("#mon_test").disabled).toBe(true);
  const line = $("#mon_msg");
  expect(line.textContent).toContain("Testing connection");
  // the status line sits directly under the paragraph with the project ID
  expect(line.previousElementSibling!.querySelector("#mon_pid")).not.toBeNull();
  // a second click while running does nothing
  $<HTMLButtonElement>("#mon_test").click();
  await tick();
  expect(calls.filter((c) => c === "test").length).toBe(1);

  testGate!("ok");
  await tick(); await tick();
  expect($<HTMLButtonElement>("#mon_test").disabled).toBe(false);
  expect($("#mon_msg").textContent).toContain('Connected: project "Demo"');
  expect($("#mon_msg").textContent).not.toContain("Testing connection");
});

test("Test: a failed test re-enables the button and shows the error", async () => {
  project = makeProject({ monitor: { enabled: false, system: "aerion", projectId: "42", defaultAction: "" } });
  await mon.openMonitor(host());
  $<HTMLButtonElement>("#mon_test").click();
  await tick();
  expect($("#mon_msg").textContent).toContain("Testing connection");
  testGate!("fail");
  await tick(); await tick();
  expect($<HTMLButtonElement>("#mon_test").disabled).toBe(false);
  expect($("#mon_msg").textContent).toContain("authentication failed");
  expect($("#mon_msg").className).toContain("mon-bad");
});

const change = (el: Element) => el.dispatchEvent(new window.Event("change", { bubbles: true }) as unknown as Event);

test("period: both options can be checked and the fetched dates are shown", async () => {
  project = makeProject({ monitor: { enabled: true, system: "aerion", projectId: "42", defaultAction: "",
    includeBefore: false, includeAfter: false, syncFrom: "2026-01-01", syncTo: "2026-03-01" } });
  await mon.openMonitor(host());
  expect($<HTMLInputElement>("#mon_before").checked).toBe(false);
  expect($<HTMLInputElement>("#mon_after").checked).toBe(false);
  expect($("#mon_period").textContent).toContain("2026-01-01 to 2026-03-01");

  $<HTMLInputElement>("#mon_before").checked = true;
  change($("#mon_before"));
  await tick();
  expect(calls).toContain("setRange(true,false)");
  expect($<HTMLInputElement>("#mon_before").checked).toBe(true);
  expect($("#mon_period").textContent).toContain("2025-12-01 to 2026-03-01");

  $<HTMLInputElement>("#mon_after").checked = true;
  change($("#mon_after"));
  await tick();
  expect(calls).toContain("setRange(true,true)");
  expect($<HTMLInputElement>("#mon_before").checked).toBe(true); // first choice is kept
  expect($<HTMLInputElement>("#mon_after").checked).toBe(true);
  expect($("#mon_period").textContent).toContain("2025-12-01 to 2026-04-01");
});

test("period: options are locked while a sync runs, errors are shown", async () => {
  project = makeProject({ monitor: { enabled: true, system: "aerion", projectId: "42", defaultAction: "",
    includeBefore: false, includeAfter: false, syncFrom: "2026-01-01", syncTo: "2026-03-01" } });
  await mon.openMonitor(host());
  $<HTMLButtonElement>("#mon_sync").click();
  await tick();
  expect($<HTMLInputElement>("#mon_before").disabled).toBe(true);
  expect($<HTMLInputElement>("#mon_after").disabled).toBe(true);
  syncGate!();
  await tick(); await tick();
  expect($<HTMLInputElement>("#mon_before").disabled).toBe(false);

  setMonitorError = "boom";
  $<HTMLInputElement>("#mon_after").checked = true;
  change($("#mon_after"));
  await tick();
  expect($("#mon_msg").textContent).toContain("boom");
});

test("actualEffortHTML: not enabled, not synced yet, not tracked, and the real numbers", () => {
  const wp = project.packages[0]; // "a", externalId "WP-1"
  expect(mon.actualEffortHTML(project, wp, null)).toBe("");
  const onProj = { ...project, monitor: { ...project.monitor, enabled: true } };
  expect(mon.actualEffortHTML(onProj, wp, null)).toContain("not synchronized yet");
  const s = makeSummary();
  expect(mon.actualEffortHTML(onProj, project.packages[2], s)).toContain("not tracked");

  const html = mon.actualEffortHTML(onProj, wp, s);
  expect(html).toContain("3,5 h");
  expect(html).toContain("0,44 days");
  expect(html).toContain('assigned to "WP-1"');
  expect(html).not.toContain("unassigned (default action)"); // not the default action

  // the default-action scenario (test-assignment.ptplan): matched 4h/0,5d,
  // unassigned 48,5h/6,06d, total 52,5h/6,56d
  const defWp = { ...wp, id: "b", externalId: "" };
  const s2 = makeSummary();
  s2.sections[0].details[0] = { id: "b", externalId: "", name: "b", actual: "6,56", entries: 14, isDefault: true,
    defaultActual: "6,06", defaultEntries: 13, actualHours: "52,5", actualDays: "6,56",
    matchedHours: "4", matchedDays: "0,5", matchedEntries: 1, defaultHours: "48,5", defaultDays: "6,06" };
  const html2 = mon.actualEffortHTML(onProj, defWp, s2);
  expect(html2).toContain("52,5 h");
  expect(html2).toContain("6,56 days");
  expect(html2).toContain("unassigned (default action)");
  expect(html2).toContain("48,5 h");
  expect(html2).not.toContain('assigned to "WP-1"'); // no External ID on this action
});

// The review case: a project in DAYS shows days first (bold) and hours in brackets,
// in the action's actual-effort block and in each of its parts.
test("actualEffortHTML: a days project puts days first, hours in brackets", () => {
  const daysProj = { ...project, unit: "days", monitor: { ...project.monitor, enabled: true } };
  const defWp = { ...project.packages[0], id: "b", externalId: "WP1" };
  const s = makeSummary();
  s.sections[0].details[0] = { id: "b", externalId: "WP1", name: "b", actual: "6,56", entries: 14, isDefault: true,
    defaultActual: "6,06", defaultEntries: 13, actualHours: "52,5", actualDays: "6,56",
    matchedHours: "4", matchedDays: "0,5", matchedEntries: 1, defaultHours: "48,5", defaultDays: "6,06" };
  const html = mon.actualEffortHTML(daysProj, defWp, s);
  expect(html).toContain("<b>6,56 days</b> (52,5 h) · 14 bookings");
  expect(html).toContain('assigned to "WP1": <b>0,5 days</b> (4 h) · 1 booking');
  expect(html).toContain("unassigned (default action): <b>6,06 days</b> (48,5 h) · 13 bookings");
  expect(html).not.toContain("<b>52,5 h</b>");
  // and an hours project the other way round
  const hoursProj = { ...daysProj, unit: "hours" };
  const h2 = mon.actualEffortHTML(hoursProj, defWp, s);
  expect(h2).toContain("<b>52,5 h</b> (6,56 days)");
  expect(h2).toContain('assigned to "WP1": <b>4 h</b> (0,5 days) · 1 booking');
});

test("contour None: not counted as tracked, not offered as the default action, no actual effort", async () => {
  project = makeProject();
  project.packages[0] = { ...project.packages[0], contour: "None" }; // "a" has External ID WP-1
  const st = mon.sectionStates(project);
  expect(st.find((x) => x.name === "Design")!.tracked).toBe(0); // its External ID no longer counts
  await mon.openMonitor(host());
  const opts = [...document.querySelectorAll<HTMLOptionElement>("#mon_def option")].map((o) => o.value);
  expect(opts).not.toContain("a");
  expect(opts).toContain("b");
  const onProj = { ...project, monitor: { ...project.monitor, enabled: true } };
  const html = mon.actualEffortHTML(onProj, project.packages[0], makeSummary());
  expect(html).toContain("contour None");
  expect(html).not.toContain("not synchronized yet");
});

// External IDs are editable at any time, also while monitoring is on. An action
// with the contour None is not a real action and cannot carry one.
test("externalIdState: editable with monitoring off, no note", () => {
  expect(mon.externalIdState(project, { contour: "Flat" })).toEqual({ readOnly: false, noteHTML: "" });
});

test("externalIdState: editable while monitoring is on, with a hint to synchronize again", () => {
  const on = { ...project, monitor: { ...project.monitor, enabled: true } };
  for (const contour of ["Flat", "Front-Loaded", "Back-Loaded", "Double-Peak"]) {
    const s = mon.externalIdState(on, { contour });
    expect(s.readOnly).toBe(false);
    expect(s.noteHTML).toContain("synchronize again");
    expect(s.noteHTML).not.toContain("Read-only");
    expect(s.noteHTML).not.toContain("e_openmon"); // no detour through the Monitor page any more
  }
});

test("externalIdState: contour None cannot carry an External ID, with or without monitoring", () => {
  for (const enabled of [false, true]) {
    const p = { ...project, monitor: { ...project.monitor, enabled } };
    const s = mon.externalIdState(p, { contour: "None" });
    expect(s.readOnly).toBe(true);
    expect(s.noteHTML).toContain("contour <b>None</b>");
    expect(s.noteHTML).toContain("cannot carry an External ID");
  }
});

test("sync result: every action row shows its External ID in its own cell", () => {
  const html = mon.resultTableHTML(makeProject(), makeSummary());
  expect(html).toContain('<td class="mon-extid"><code>WP-1</code></td>');
  expect(html).toContain("External ID</th>");
});

test("result table: Remaining and Remaining % columns for section and action", () => {
  const html = mon.resultTableHTML(project, makeSummary());
  expect(html).toContain("<th>Remaining</th><th>Remaining %</th>");
  expect(html.match(/<td class="num">6,5<\/td>\s*<td class="num">65 %<\/td>/g)?.length).toBe(2);
});

test("Export button is disabled without a result and enabled with one", async () => {
  await mon.openMonitor(host());
  expect($<HTMLButtonElement>("#mon_export").disabled).toBe(true);
  summary = makeSummary();
  await mon.openMonitor(host());
  expect($<HTMLButtonElement>("#mon_export").disabled).toBe(false);
});
