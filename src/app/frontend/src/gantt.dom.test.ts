import { beforeAll, expect, test } from "bun:test";
import { Window } from "happy-dom";

// Integration test for the render gate (R3.3, R7.1-7.3, R8.7): zero marker/curve
// elements when showProgress is off, exactly one of each per eligible action
// when on, and the marker x follows completionPos.
let Gantt: typeof import("./gantt").Gantt;

beforeAll(async () => {
  const win = new Window();
  const g = globalThis as unknown as Record<string, unknown>;
  g.window = win; g.document = win.document; g.HTMLElement = win.HTMLElement;
  g.localStorage = win.localStorage;
  ({ Gantt } = await import("./gantt"));
});

function wp(id: string, over: Record<string, unknown> = {}) {
  return {
    id, num: 1, displayId: "1.1", name: id, desc: "", start: "2026-01-05", end: "2026-02-05",
    est: "10", realEst: "5", contour: "Flat", progressPercent: 50, completionPos: 0.5,
    estHours: "80", estDays: "10", realHours: "40", realDays: "5", synced: false, syncedAt: "",
    progressPermille: 500, contourPermille: 500,
    mode: "fixed", summary: false, color: "#8a90dd", fontColor: "", textPos: "", noteVisible: false,
    notePos: "", cluster: "s", sameLine: false, hasWarning: false, details: [], ...over,
  };
}
function project(showProgress: boolean, packages: unknown[]) {
  return {
    title: "t", description: "", unit: "days", hoursPerDay: 8, language: "en",
    start: "2026-01-01", end: "2026-03-01", totalEst: "0",
    milestones: [], phases: [], packages, deps: [], roles: [],
    sectionColors: {}, sectionNums: { s: 1 }, arrowStyle: "spline",
    showDetails: false, showRoleActions: false, showRolePercent: false, showRoleDays: false, showPhases: false, showProgress,
    zoom: "months", autosaveEnabled: true, autosaveMinutes: 5, autosaveOnExit: true,
    saveVersions: false, versionCount: 5, reportFormat: "pdf", reportTodayLine: false,
    reportGrid: "month", reportShowNotes: true, savePath: "", warnings: [],
  };
}
function make() {
  const host = document.createElement("div");
  document.body.appendChild(host);
  const noop = () => {};
  return { host, gantt: new Gantt(host as HTMLElement, {
    onMove: noop, onOpen: noop, onLink: noop, onContextMenu: noop, onSelect: noop,
  } as never) };
}

test("no marker/curve when showProgress is off", () => {
  const { host, gantt } = make();
  gantt.setProject(project(false, [wp("a"), wp("b")]) as never);
  expect(host.querySelectorAll(".progress-marker").length).toBe(0);
  expect(host.querySelectorAll(".progress-curve").length).toBe(0);
});

test("one marker and one curve per eligible action when on; summary excluded", () => {
  const { host, gantt } = make();
  gantt.setProject(project(true, [wp("a"), wp("b", { contour: "Double-Peak" }), wp("s", { summary: true })]) as never);
  expect(host.querySelectorAll(".progress-marker").length).toBe(2);
  expect(host.querySelectorAll(".progress-curve").length).toBe(2);
});

test("marker x follows completionPos across the bar", () => {
  const { host, gantt } = make();
  gantt.setProject(project(true, [wp("a", { completionPos: 0 })]) as never);
  const x0 = Number(host.querySelector(".progress-marker")!.getAttribute("x1"));
  gantt.setProject(project(true, [wp("a", { completionPos: 1 })]) as never);
  const x1 = Number(host.querySelector(".progress-marker")!.getAttribute("x1"));
  gantt.setProject(project(true, [wp("a", { completionPos: 0.5 })]) as never);
  const xm = Number(host.querySelector(".progress-marker")!.getAttribute("x1"));
  expect(x1).toBeGreaterThan(x0);
  expect(xm).toBeCloseTo((x0 + x1) / 2, 3);
});

test("contour change redraws the curve", () => {
  const { host, gantt } = make();
  gantt.setProject(project(true, [wp("a", { contour: "Flat" })]) as never);
  const flat = host.querySelector(".progress-curve")!.getAttribute("points");
  gantt.setProject(project(true, [wp("a", { contour: "Front-Loaded" })]) as never);
  expect(host.querySelector(".progress-curve")!.getAttribute("points")).not.toBe(flat);
});

test("showing progress: shaded area, rating in the label and a tooltip per action", () => {
  const { host, gantt } = make();
  gantt.setProject(project(true, [
    wp("a"),
    wp("b", { contour: "Double-Peak", completionPos: 0.126, progressPermille: 164, contourPermille: 126, realHours: "52,5", realDays: "6,56", estHours: "320", estDays: "40", synced: true }),
    wp("s", { summary: true }),
  ]) as never);
  expect(host.querySelectorAll(".progress-area").length).toBe(2); // summary excluded
  const texts = [...host.querySelectorAll("text")].map((t) => t.textContent);
  expect(texts.some((t) => t!.includes("50.0%"))).toBe(true);
  expect(texts.some((t) => t!.includes("16.4% (Double-Peak 12.6%)"))).toBe(true);
  const titles = [...host.querySelectorAll("title")].map((t) => t.textContent!);
  expect(titles.some((t) => t.includes("Planned 40 days (320 h), real 6,56 days (52,5 h)") && t.includes("synchronized"))).toBe(true);
});

test("no rating text or shading when Show Progress is off", () => {
  const { host, gantt } = make();
  gantt.setProject(project(false, [wp("a")]) as never);
  expect(host.querySelectorAll(".progress-area").length).toBe(0);
  expect([...host.querySelectorAll("text")].some((t) => t.textContent!.includes("%"))).toBe(false);
});

test("the label is drawn above the progress graphics (z-order)", () => {
  const { host, gantt } = make();
  gantt.setProject(project(true, [wp("a")]) as never);
  const g = host.querySelector(".progress-area")!.parentElement!;
  const kids = [...g.children];
  const area = kids.indexOf(host.querySelector(".progress-area")!);
  const marker = kids.indexOf(host.querySelector(".progress-marker")!);
  const label = kids.findIndex((k) => k.tagName.toLowerCase() === "text" && k.textContent!.includes("50.0%"));
  expect(area).toBeGreaterThanOrEqual(0);
  expect(label).toBeGreaterThan(area);   // text after (= above) the shaded area
  expect(label).toBeGreaterThan(marker); // and above the marker
});

test("contour None: no marker, curve, shaded area or rating on the bar; other actions keep theirs", () => {
  const { host, gantt } = make();
  gantt.setProject(project(true, [
    wp("flat"),
    wp("none", { contour: "None", est: "0", estHours: "0", estDays: "0", progressPermille: 0, contourPermille: 0, completionPos: 0 }),
  ]) as never);
  expect(host.querySelectorAll(".progress-marker").length).toBe(1);
  expect(host.querySelectorAll(".progress-curve").length).toBe(1);
  expect(host.querySelectorAll(".progress-area").length).toBe(1);
  const texts = [...host.querySelectorAll("text")].map((t) => t.textContent!);
  expect(texts.filter((t) => t.includes("%")).length).toBe(1); // only the Flat action shows a rating
  expect(texts.some((t) => t.includes("none"))).toBe(true);      // the None action is still labelled
  // even if stale numbers arrive, nothing is drawn for None
  const { host: h2, gantt: g2 } = make();
  g2.setProject(project(true, [wp("none", { contour: "None", progressPermille: 500, contourPermille: 500, completionPos: 0.5 })]) as never);
  expect(h2.querySelectorAll(".progress-marker, .progress-curve, .progress-area").length).toBe(0);
  expect([...h2.querySelectorAll("text")].some((t) => t.textContent!.includes("%"))).toBe(false);
});

// A real drag on the rendered canvas: pointer down on the bar, move 3 days, release.
test("dragging an action reports the state it moves to (start first) and clears it on release", () => {
  const host = document.createElement("div"); document.body.appendChild(host);
  const noop = () => {};
  const status: Array<string | null> = [];
  const moves: string[][] = [];
  const gantt = new Gantt(host as HTMLElement, {
    onMove: (...a: string[]) => moves.push(a), onOpen: noop, onLink: noop, onContextMenu: noop, onSelect: noop,
    onReassign: noop, onDragStatus: (t: string | null) => status.push(t),
  } as never);
  const p = project(false, [wp("a", { name: "AWS Landing Zone", start: "2026-01-05", end: "2026-01-16" })]) as Record<string, unknown>;
  gantt.setZoom("days"); // 28 px per day, snapping to whole days
  gantt.setProject(p as never);
  const bar = [...host.querySelectorAll("rect")].find((r) => (r as unknown as HTMLElement).style.cursor === "grab")!;
  const ev = (type: string, x: number) => {
    const e = new window.Event(type, { bubbles: true, cancelable: true });
    Object.defineProperty(e, "clientX", { value: x }); Object.defineProperty(e, "clientY", { value: 0 });
    return e;
  };
  bar.dispatchEvent(ev("pointerdown", 100) as unknown as Event);
  window.dispatchEvent(ev("pointermove", 100 + 3 * 28) as unknown as Event);
  const during = status.filter((s) => s !== null) as string[];
  expect(during.length).toBeGreaterThan(0);
  expect(during.at(-1)).toContain('Moving "AWS Landing Zone": start 2026-01-08 (+3 days)');
  expect(during.at(-1)).toContain("end 2026-01-19 (+3 days)");
  expect(status.at(-1)).not.toBeNull(); // still dragging
  window.dispatchEvent(ev("pointerup", 100 + 3 * 28) as unknown as Event);
  expect(status.at(-1)).toBeNull();      // released: the status line can be restored
  expect(moves).toEqual([["a", "2026-01-08", "2026-01-19"]]);
});

// An empty section that was ordered up is drawn where its WP number says.
test("lanes follow the section numbers, including an empty section in the middle", () => {
  const { host, gantt } = make();
  const p = project(false, [wp("a", { cluster: "Alpha" }), wp("b", { cluster: "Beta" })]) as Record<string, any>;
  p.sectionColors = { Alpha: "", Beta: "", Empty: "" };
  p.sectionNums = { Alpha: 1, Empty: 2, Beta: 3 }; // "Empty" was moved up from the end
  gantt.setProject(p as never);
  const labels = [...host.querySelectorAll("text")].map((t) => t.textContent!).filter((t) => t.startsWith("WP-"));
  expect(labels).toEqual(["WP-1 Alpha", "WP-2 Empty", "WP-3 Beta"]);
  // and the other way round
  p.sectionNums = { Empty: 1, Alpha: 2, Beta: 3 };
  gantt.setProject(p as never);
  const labels2 = [...host.querySelectorAll("text")].map((t) => t.textContent!).filter((t) => t.startsWith("WP-"));
  expect(labels2).toEqual(["WP-1 Empty", "WP-2 Alpha", "WP-3 Beta"]);
});

// On the detail plane, a detail marked summary draws the rollup (Sammelvorgang)
// polygon; a normal detail draws a plain rect. Mirrors the summary ACTION bar.
test("a summary detail is drawn with the rollup polygon on the detail plane", () => {
  const host = document.createElement("div");
  document.body.appendChild(host);
  const noop = () => {};
  const gantt = new Gantt(host as HTMLElement, {
    onMove: noop, onOpen: noop, onLink: noop, onContextMenu: noop, onSelect: noop,
    onDetailAdd: noop, onDetailMove: noop, onDetailSelect: noop, onDetailOpen: noop,
    onDrillChange: noop,
  } as never);
  const det = (id: string, over: Record<string, unknown> = {}) => ({
    id, displayId: "1.1." + id, name: id, desc: "", role: "", start: "2026-01-10",
    end: "2026-01-25", est: "0", summary: false, fontColor: "", textPos: "", ...over,
  });
  const p = project(false, [wp("a", { details: [det("x"), det("s", { summary: true })] })]);
  gantt.setProject(p as never);
  gantt.drillInto("a");
  // the summary detail contributes a <path> with crispEdges (the rollup body);
  // the normal detail contributes a <rect>.
  const paths = [...host.querySelectorAll("path")].filter((n) => n.getAttribute("shape-rendering") === "crispEdges");
  expect(paths.length).toBe(1);
  expect(host.querySelectorAll("text")).not.toBeNull();
});

// With Show Details on, an action whose detailsHidden=true draws no detail bars,
// while a normal action in the same project still does.
test("per-action detailsHidden suppresses that action's detail bars", () => {
  const { host, gantt } = make();
  const det = (id: string) => ({
    id, displayId: "1.1." + id, name: id, desc: "", role: "", start: "2026-01-10",
    end: "2026-01-25", est: "0", summary: false, fontColor: "", textPos: "",
  });
  const p = project(false, [
    wp("a", { cluster: "s", details: [det("x"), det("y")] }),
    wp("b", { cluster: "s", details: [det("z")], detailsHidden: true }),
  ]) as Record<string, any>;
  p.showDetails = true;
  gantt.setProject(p as never);
  // detail bars are rects with height BAR_H-6 (=8) and stroke #8a90dd
  const detBars = [...host.querySelectorAll("rect")].filter(
    (r) => r.getAttribute("height") === "14" && r.getAttribute("stroke") === "#8a90dd");
  expect(detBars.length).toBe(2); // only action "a"'s two details; "b"'s one is hidden
});

// --- sticky date row, week numbers, drag ticks ---

test("the date row is one group that follows the scroll offset; export drops it", async () => {
  const { host, gantt } = make();
  gantt.setProject(project(false, [wp("a")]) as never);
  const head = host.querySelector(".stickyhead")!;
  expect(head).not.toBeNull();
  expect(head.querySelectorAll("text").length).toBeGreaterThan(0); // the month labels
  expect(head.getAttribute("transform") || "").toBe("");
  expect(head.querySelector(".stickybg")!.getAttribute("visibility")).toBe("hidden"); // unscrolled: looks as before
  expect(host.querySelector("svg")!.lastElementChild).toBe(head); // drawn on top of everything
  host.scrollTop = 120;
  host.dispatchEvent(new window.Event("scroll") as unknown as Event);
  expect(head.getAttribute("transform")).toBe("translate(0 120)");
  expect(head.querySelector(".stickybg")!.getAttribute("visibility")).toBe("visible");
  gantt.setProject(project(false, [wp("a")]) as never); // a re-render keeps it stuck
  expect(host.querySelector(".stickyhead")!.getAttribute("transform")).toBe("translate(0 120)");
});

test("week numbers: off by default, ISO weeks when on and there is room", () => {
  const { host, gantt } = make();
  const p = project(false, [wp("a")]) as Record<string, any>;
  gantt.setZoom("days");
  gantt.setProject(p as never);
  const weeks = () => [...host.querySelectorAll(".stickyhead text")].map((t) => t.textContent!).filter((t) => /^(W|KW)\d+$/.test(t));
  expect(weeks()).toEqual([]);
  p.showWeekNumbers = true;
  gantt.setProject(p as never);
  expect(weeks()[0]).toBe("W1"); // the chart starts a week before 2026-01-01: Monday 2025-12-29 is ISO week 1
}, 5000);

test("isoWeek follows ISO 8601", async () => {
  const { isoWeek } = await import("./gantt");
  const d = (s: string) => Date.parse(s + "T00:00:00Z");
  expect(isoWeek(d("2026-01-01"))).toBe(1);   // Thursday
  expect(isoWeek(d("2026-01-05"))).toBe(2);
  expect(isoWeek(d("2021-01-03"))).toBe(53);  // Sunday still in 2020's last week
  expect(isoWeek(d("2024-12-30"))).toBe(1);   // Monday belongs to week 1 of 2025
  expect(isoWeek(d("2026-12-31"))).toBe(53);
});

test("while an action is dragged the date row shows a tick for its start and end", () => {
  const host = document.createElement("div"); document.body.appendChild(host);
  const noop = () => {};
  const gantt = new Gantt(host as HTMLElement, {
    onMove: noop, onOpen: noop, onLink: noop, onContextMenu: noop, onSelect: noop, onReassign: noop,
  } as never);
  gantt.setZoom("days");
  gantt.setProject(project(false, [wp("a", { start: "2026-01-05", end: "2026-01-16" })]) as never);
  const bar = [...host.querySelectorAll("rect")].find((r) => (r as unknown as HTMLElement).style.cursor === "grab")!;
  const ev = (type: string, x: number) => {
    const e = new window.Event(type, { bubbles: true, cancelable: true });
    Object.defineProperty(e, "clientX", { value: x }); Object.defineProperty(e, "clientY", { value: 0 });
    return e as unknown as Event;
  };
  bar.dispatchEvent(ev("pointerdown", 100));
  window.dispatchEvent(ev("pointermove", 100 + 2 * 28));
  const ticks = () => [...host.querySelectorAll(".stickyhead .dragmarks line")];
  expect(ticks().length).toBe(2);
  const [x1, x2] = ticks().map((l) => Number(l.getAttribute("x1")));
  expect(x2).toBeGreaterThan(x1);
  window.dispatchEvent(ev("pointerup", 100 + 2 * 28));
  expect(host.querySelectorAll(".dragmarks").length).toBe(0);
});

// A summary action (Sammelvorgang) honours its text position: "up" puts the
// label above the bar at its left end, auto keeps it to the right of the end.
test("summary action: the label follows the text position", () => {
  const { host, gantt } = make();
  const labelOf = () => [...host.querySelectorAll("text")].find((t) => t.textContent === "Rollup")!;
  gantt.setProject(project(false, [wp("a", { name: "Rollup", est: "0", summary: true })]) as never);
  const bar = [...host.querySelectorAll("path")].find((n) => n.getAttribute("shape-rendering") === "crispEdges")!;
  const xs = bar.getAttribute("d")!.match(/M (\S+) (\S+) L (\S+)/)!;
  const xL = Number(xs[1]), sy = Number(xs[2]), xR = Number(xs[3]);
  expect(Number(labelOf().getAttribute("x"))).toBe(xR + 8); // auto: right of the end
  for (const [pos, check] of [
    ["up", (t: Element) => Number(t.getAttribute("x")) === xL && Number(t.getAttribute("y")) < sy],
    ["outleft", (t: Element) => Number(t.getAttribute("x")) === xL - 8 && t.getAttribute("text-anchor") === "end"],
    ["down", (t: Element) => Number(t.getAttribute("x")) === xL && Number(t.getAttribute("y")) > sy + 7],
    ["center", (t: Element) => t.getAttribute("text-anchor") === "middle" && Number(t.getAttribute("y")) < sy],
    ["outright", (t: Element) => Number(t.getAttribute("x")) === xR + 8],
  ] as const) {
    gantt.setProject(project(false, [wp("a", { name: "Rollup", est: "0", summary: true, textPos: pos })]) as never);
    expect([pos, check(labelOf())]).toEqual([pos, true]);
  }
});
