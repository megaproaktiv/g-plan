import { effortText } from "./effort";
import { dragStatusText, milestoneDragStatusText } from "./dragstatus";
import { contourAreaPoints, contourCurvePoints, curveColor, markerColor, ratingLabel } from "./progress";
const DAY = 86400000;
const SVGNS = "http://www.w3.org/2000/svg";
function parseDate(s) {
    const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(s || "");
    if (!m)
        return NaN;
    return Date.UTC(+m[1], +m[2] - 1, +m[3]);
}
function fmtDate(ms) {
    if (!isFinite(ms))
        return "";
    const d = new Date(ms);
    return `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, "0")}-${String(d.getUTCDate()).padStart(2, "0")}`;
}
const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
// wrapText greedily wraps a string into lines of at most `width` characters,
// capped at `maxLines` (the last line gets an ellipsis if text remains). Used
// by the sticky-note renderer to keep the pad small.
function wrapText(s, width, maxLines) {
    const words = s.split(/\s+/).filter(Boolean);
    const lines = [];
    let cur = "";
    for (const w of words) {
        const cand = cur ? cur + " " + w : w;
        if (cand.length > width && cur) {
            lines.push(cur);
            cur = w;
        }
        else
            cur = cand;
        if (lines.length >= maxLines)
            break;
    }
    if (cur && lines.length < maxLines)
        lines.push(cur);
    if (lines.length === maxLines) {
        // if text remains beyond the cap, ellipsize the last line
        const used = lines.join(" ").length;
        if (used < s.replace(/\s+/g, " ").length) {
            let last = lines[maxLines - 1];
            if (last.length > width - 1)
                last = last.slice(0, width - 1);
            lines[maxLines - 1] = last.replace(/\s*\S*$/, "") + "…";
            if (lines[maxLines - 1] === "…")
                lines[maxLines - 1] = last + "…";
        }
    }
    return lines.length ? lines : [s.slice(0, width)];
}
// maxLineLen returns the longest line length (characters) in a list.
function maxLineLen(lines) {
    return lines.reduce((m, l) => Math.max(m, l.length), 0);
}
// defaultFromAnchor / defaultToAnchor pick the natural anchor for a dependency
// kind when the stored link carries no explicit anchor (e.g. a chat `link`).
// l = start edge, r = end edge, t = middle (top). A = from, B = to.
//   FS: A.end(r)   -> B.start(l)
//   SS: A.start(l) -> B.start(l)
//   FF: A.end(r)   -> B.end(r)
//   SF: A.start(l) -> B.end(r)
//   MS: A.mid(t)   -> B.start(l)
//   ME: A.mid(t)   -> B.end(r)
//   MM: A.mid(t)   -> B.mid(t)
function defaultFromAnchor(kind) {
    switch (kind) {
        case "SS":
        case "SF": return "l";
        case "MS":
        case "ME":
        case "MM":
        case "OV": return "t";
        default: return "r"; // FS, FF
    }
}
function defaultToAnchor(kind) {
    switch (kind) {
        case "FF":
        case "SF":
        case "ME": return "r";
        case "MM":
        case "OV": return "t";
        default: return "l"; // FS, SS, MS
    }
}
// dedupeColinear removes points that lie on the same horizontal/vertical line
// as their neighbours (and exact duplicates), minimizing the number of corners
// in an orthogonal path.
function dedupeColinear(pts) {
    const out = [];
    for (const p of pts) {
        const last = out[out.length - 1];
        if (last && last.x === p.x && last.y === p.y)
            continue; // dup
        out.push(p);
    }
    let i = 1;
    while (i < out.length - 1) {
        const a = out[i - 1], b = out[i], c = out[i + 1];
        const colinearH = a.y === b.y && b.y === c.y;
        const colinearV = a.x === b.x && b.x === c.x;
        if (colinearH || colinearV)
            out.splice(i, 1); // drop the middle corner
        else
            i++;
    }
    return out;
}
function orthoPath(pts) {
    return pts.map((p, i) => `${i === 0 ? "M" : "L"} ${p.x} ${p.y}`).join(" ");
}
/** isoWeek returns the ISO 8601 calendar week number of a UTC day (ms). */
export function isoWeek(t) {
    const d = new Date(t);
    const day = d.getUTCDay() || 7; // Monday = 1 ... Sunday = 7
    const thursday = Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate() + 4 - day);
    const jan1 = Date.UTC(new Date(thursday).getUTCFullYear(), 0, 1);
    return Math.floor((thursday - jan1) / (7 * DAY)) + 1;
}
const ZOOMS = ["days", "weeks", "months", "year"];
const PPD = { days: 28, weeks: 10, months: 3.2, year: 0.9 };
const SNAP = { days: 1, weeks: 1, months: 7, year: 30, window: 1 };
const ROW_H = 36;
const BAR_H = 20;
const LEFT = 220;
const TOP = 56;
const LANE_PAD = 6; // vertical padding inside a swimlane band
export class Gantt {
    constructor(host, cb) {
        // The date row (month/day/week labels) lives in this group. It is appended
        // LAST and moved down by the scroll offset, so it stays at the top of the
        // visible area while the chart is scrolled. Exports drop the offset.
        this.head = null;
        this.project = null;
        this.zoom = "months";
        this.t0 = 0;
        this.linkMode = false;
        this.linkFrom = null;
        this.selected = null; // last selected action id (for L-link flow)
        this.selectedSection = null; // selected section/cluster (border)
        this.selectedMilestone = null; // selected milestone (diamond highlight)
        this.selectedDetail = null; // last selected detail id (drill-plane L-link)
        this.lanes = [];
        this.rowOf = {}; // action id -> absolute y (its bar row top)
        this.detailRowOf = {}; // detail id -> absolute y
        this.detailRowY = 0; // running y while laying out the drill plane rows
        this.detailParent = {}; // detail id -> wp id
        this.roleRowOf = {}; // wp id -> role sub-rows
        this.laneOf = {};
        this.visible = null; // null = show all; else only these ids
        this.showDetails = false; // draw level-2 detail bars as sub-rows (project.showDetails)
        this.mode = "action"; // current chat/view mode; "role" enables role sub-rows
        this.showRoleActions = false; // in role mode, expand each action into per-role sub-rows
        this.showRolePercent = false; // role sub-row label: append the percent
        this.showRoleDays = false; // role sub-row label: append the effort share ("= N d")
        this.showProgress = false; // draw progress markers + contour curves (project.showProgress)
        this.drilledOnce = new Set(); // action ids whose drill-down has been opened (session)
        this.drillWp = null; // when set, render that action's detail plane
        this.cb = cb;
        this.host = host;
        this.svg = document.createElementNS(SVGNS, "svg");
        this.svg.setAttribute("class", "gantt");
        host.appendChild(this.svg);
        host.addEventListener("scroll", () => this.applyStick());
        this.svg.addEventListener("contextmenu", (e) => {
            e.preventDefault();
            const me = e;
            const rect = this.svg.getBoundingClientRect();
            const pt = this.svgPoint(me.clientX, me.clientY, rect);
            this.cb.onContextMenu(me.clientX, me.clientY, this.hitTest(pt.x, pt.y));
        });
        // pinch / ctrl+wheel zoom
        this.svg.addEventListener("wheel", (e) => {
            if (!(e.ctrlKey || e.metaKey))
                return; // trackpad pinch sends ctrlKey
            e.preventDefault();
            if (e.deltaY < 0)
                this.zoomIn();
            else
                this.zoomOut();
        }, { passive: false });
        window.addEventListener("keydown", (e) => {
            const tag = e.target?.tagName;
            if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT")
                return;
            if (e.key === "l" || e.key === "L") {
                // Link flow: select an action, press L (source locks in), then click
                // the target action. If nothing is selected, L toggles link mode and
                // the first click chooses the source, the second the target.
                if (this.drillWp) {
                    // in the drill plane L links DETAILS the same way
                    if (this.selectedDetail) {
                        this.linkFrom = this.selectedDetail;
                        this.setLinkMode(true, true);
                    }
                    else
                        this.setLinkMode(!this.linkMode);
                }
                else if ((this.selected || this.selectedMilestone) && !this.linkMode) {
                    // a selected action OR milestone becomes the link source
                    this.setLinkMode(true, true); // keep source
                    this.linkFrom = this.selected || this.selectedMilestone;
                }
                else {
                    this.setLinkMode(!this.linkMode);
                }
            }
            else if (e.key === "d" || e.key === "D") {
                // Drill DOWN into the selected action's third-level detail plane.
                if (this.drillWp)
                    return;
                if (this.selected)
                    this.drillInto(this.selected);
            }
            else if (e.key === "u" || e.key === "U") {
                if (this.drillWp)
                    this.drillUp();
            }
            else if (e.key === "Escape") {
                // Escape only cancels link mode here. It must NOT drill up (U does
                // that) because Escape is also the "deselect objects" key elsewhere.
                this.setLinkMode(false);
            }
            else if (e.key === "ArrowUp" || e.key === "ArrowDown" || e.key === "ArrowLeft" || e.key === "ArrowRight") {
                // Arrow keys navigate between actions from the current selection.
                if (this.drillWp || !this.selected)
                    return;
                const dir = e.key.slice(5).toLowerCase();
                const next = this.neighborAction(this.selected, dir);
                if (next) {
                    e.preventDefault();
                    this.cb.onSelect("action", next);
                }
            }
        });
    }
    // neighborAction returns the id of the nearest action to the given one in a
    // direction. up/down compare row centers; left/right compare bar-start x.
    neighborAction(fromId, dir) {
        const pkgs = this.project?.packages ?? [];
        const cur = pkgs.find((w) => w.id === fromId);
        if (!cur)
            return null;
        const cy = (id) => (this.rowOf[id] ?? 0) + ROW_H / 2;
        const cx = (w) => this.x(parseDate(w.start));
        const fromY = cy(fromId), fromX = cx(cur);
        let best = null, bestD = Infinity;
        for (const w of pkgs) {
            if (w.id === fromId || this.rowOf[w.id] == null)
                continue;
            const wy = cy(w.id), wx = cx(w);
            let ok = false, d = 0;
            if (dir === "up") {
                ok = wy < fromY - 1;
                d = (fromY - wy) + Math.abs(wx - fromX) * 0.25;
            }
            else if (dir === "down") {
                ok = wy > fromY + 1;
                d = (wy - fromY) + Math.abs(wx - fromX) * 0.25;
            }
            else if (dir === "left") {
                ok = wx < fromX - 1;
                d = (fromX - wx) + Math.abs(wy - fromY) * 0.25;
            }
            else {
                ok = wx > fromX + 1;
                d = (wx - fromX) + Math.abs(wy - fromY) * 0.25;
            }
            if (ok && d < bestD) {
                bestD = d;
                best = w.id;
            }
        }
        return best;
    }
    // setSelected records the externally-selected action (from onSelect) so the
    // L key can use it as the link source AND so the bar is drawn with a
    // selection border. Passing null clears the action selection.
    setSelected(id) { this.selected = id; if (id)
        this.selectedSection = null; this.render(); }
    // setSelectedSection records the selected section/cluster so its lane band is
    // drawn with a selection border.
    setSelectedSection(name) { this.selectedSection = name; if (name) {
        this.selected = null;
        this.selectedMilestone = null;
    } this.render(); }
    // setSelectedMilestone highlights the selected milestone diamond.
    setSelectedMilestone(id) { this.selectedMilestone = id; if (id) {
        this.selected = null;
        this.selectedSection = null;
    } this.render(); }
    // setSelectedDetail records the selected detail (drill-plane border + L-link
    // source) so selecting a detail from outside (the right-pane editor) stays in
    // sync with the canvas.
    setSelectedDetail(id) { this.selectedDetail = id; this.render(); }
    // clearSelection removes any action/section/milestone selection border.
    clearSelection() { this.selected = null; this.selectedSection = null; this.selectedMilestone = null; this.render(); }
    setZoom(z) { this.zoom = z; this.render(); }
    getZoom() { return this.zoom; }
    // exportPNG rasterizes the CURRENT on-screen Gantt SVG to a PNG and returns
    // the base64 payload (no data-URL prefix). This is what the report export
    // sends to the backend, so the PDF/PNG matches the live canvas exactly
    // (notes, spline/square arrows, section headers, today line, grid). Editor
    // chrome is NOT exported: the link anchors (.anchor) and the selection
    // borders (.editchrome) are stripped, and the chart is rendered with no
    // selection so no selected-object highlight (e.g. a milestone diamond's
    // thickened stroke) bleeds into the report.
    async exportPNG(scale = 2) {
        return this.exportPNGWithDetails(this.showDetails, scale);
    }
    // rasterizeCurrent serializes the CURRENT on-screen SVG (as last rendered) to
    // a PNG and returns the base64 payload (no data-URL prefix). It strips
    // editor-only chrome and inlines a white background. Callers are responsible
    // for having rendered the frame they want and for save/restore of view state.
    async rasterizeCurrent(scale = 2) {
        const vb = this.svg.viewBox.baseVal;
        const w = vb.width || this.svg.clientWidth || 1000;
        const h = vb.height || this.svg.clientHeight || 600;
        const clone = this.svg.cloneNode(true);
        clone.setAttribute("width", String(w));
        clone.setAttribute("height", String(h));
        // strip editor-only chrome: link anchors + any selection borders.
        clone.querySelectorAll(".anchor, .editchrome").forEach((el) => el.remove());
        // the sticky date row: the export always shows it at its home position
        clone.querySelectorAll(".stickyhead").forEach((el) => el.removeAttribute("transform"));
        clone.querySelectorAll(".stickybg, .dragmarks").forEach((el) => el.remove());
        // inline a white background so the PNG isn't transparent
        const bg = document.createElementNS(SVGNS, "rect");
        bg.setAttribute("x", "0");
        bg.setAttribute("y", "0");
        bg.setAttribute("width", String(w));
        bg.setAttribute("height", String(h));
        bg.setAttribute("fill", "#ffffff");
        clone.insertBefore(bg, clone.firstChild);
        const xml = new XMLSerializer().serializeToString(clone);
        const svg64 = "data:image/svg+xml;base64," + btoa(unescape(encodeURIComponent(xml)));
        const img = new Image();
        await new Promise((resolve, reject) => {
            img.onload = () => resolve();
            img.onerror = () => reject(new Error("svg image load failed"));
            img.src = svg64;
        });
        const canvas = document.createElement("canvas");
        canvas.width = Math.ceil(w * scale);
        canvas.height = Math.ceil(h * scale);
        const ctx = canvas.getContext("2d");
        ctx.imageSmoothingEnabled = true;
        ctx.imageSmoothingQuality = "high";
        ctx.fillStyle = "#ffffff";
        ctx.fillRect(0, 0, canvas.width, canvas.height);
        ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
        const dataUrl = canvas.toDataURL("image/png");
        return dataUrl.split(",", 2)[1] || "";
    }
    // exportPNGWithDetails rasterizes the on-screen Gantt with the detail bars
    // forced ON or OFF (independent of the live toggle). The live showDetails /
    // selection state is saved and restored afterwards.
    async exportPNGWithDetails(showDetails, scale = 2) {
        const savedSel = this.selected, savedSec = this.selectedSection, savedMs = this.selectedMilestone, savedDet = this.selectedDetail, savedShow = this.showDetails, savedDrill = this.drillWp;
        this.selected = this.selectedSection = this.selectedMilestone = this.selectedDetail = null;
        this.showDetails = showDetails;
        this.drillWp = null; // overview, never the drill plane
        this.render();
        try {
            return await this.rasterizeCurrent(scale);
        }
        finally {
            this.selected = savedSel;
            this.selectedSection = savedSec;
            this.selectedMilestone = savedMs;
            this.selectedDetail = savedDet;
            this.showDetails = savedShow;
            this.drillWp = savedDrill;
            this.render();
        }
    }
    // exportDrillPNG rasterizes the DETAIL (drill) plane of a single work package
    // — the same view the user sees after drilling into it — regardless of the
    // Show Details toggle. Live drill/selection state is saved and restored.
    async exportDrillPNG(wpId, scale = 2) {
        const savedSel = this.selected, savedSec = this.selectedSection, savedMs = this.selectedMilestone, savedDet = this.selectedDetail, savedDrill = this.drillWp;
        this.selected = this.selectedSection = this.selectedMilestone = this.selectedDetail = null;
        this.drillWp = wpId;
        this.render();
        try {
            return await this.rasterizeCurrent(scale);
        }
        finally {
            this.selected = savedSel;
            this.selectedSection = savedSec;
            this.selectedMilestone = savedMs;
            this.selectedDetail = savedDet;
            this.drillWp = savedDrill;
            this.render();
        }
    }
    // exportReportImages produces the set of images the report embeds: the whole
    // project overview (details compacted), plus one detail-plane image for EACH
    // work package that has detail packages — independent of the Show Details
    // toggle. Returns the overview and an ordered list of per-WP detail views.
    async exportReportImages(scale = 2) {
        const overview = await this.exportPNGWithDetails(false, scale);
        const details = [];
        for (const w of this.project?.packages ?? []) {
            if ((w.details?.length ?? 0) === 0)
                continue;
            const png = await this.exportDrillPNG(w.id, scale);
            const label = (w.displayId ? w.displayId + " " : "") + w.name;
            details.push({ id: w.id, name: label, png });
        }
        return { overview, details };
    }
    zoomIn() { const i = ZOOMS.indexOf(this.curLevel()); if (i > 0) {
        this.zoom = ZOOMS[i - 1];
        this.emitZoom();
        this.render();
    } }
    zoomOut() { const i = ZOOMS.indexOf(this.curLevel()); if (i < ZOOMS.length - 1) {
        this.zoom = ZOOMS[i + 1];
        this.emitZoom();
        this.render();
    } }
    // curLevel maps the current zoom to one of the fixed stepping levels; when
    // in "window" mode it resolves to the nearest fixed level by scale so
    // pinch/step zoom continues sensibly.
    curLevel() {
        if (this.zoom !== "window")
            return this.zoom;
        const p = this.ppd();
        let best = "months", bd = Infinity;
        for (const z of ZOOMS) {
            const d = Math.abs(PPD[z] - p);
            if (d < bd) {
                bd = d;
                best = z;
            }
        }
        return best;
    }
    emitZoom() {
        const sel = document.querySelector("#zoom");
        if (sel)
            sel.value = this.zoom;
    }
    setLinkMode(on, keepSource = false) {
        this.linkMode = on;
        if (!keepSource)
            this.linkFrom = null;
        this.svg.classList.toggle("linkmode", on);
        this.render();
    }
    isLinkMode() { return this.linkMode; }
    setProject(p) {
        this.project = p;
        this.showDetails = !!p.showDetails;
        this.showRoleActions = !!p.showRoleActions;
        this.showRolePercent = !!p.showRolePercent;
        this.showRoleDays = !!p.showRoleDays;
        this.showProgress = !!p.showProgress;
        this.render();
    }
    // setMode records the current app mode ("action"|"role"|...). Role sub-rows
    // only appear in role mode (and when showRoleActions is on).
    setMode(m) { this.mode = m; this.render(); }
    // setVisible restricts which actions are drawn; null shows all.
    setVisible(ids) { this.visible = ids; this.render(); }
    // --- drill-down third level ---
    drillInto(wpId) {
        const wp = this.project?.packages.find((w) => w.id === wpId);
        if (!wp)
            return;
        this.drillWp = wpId;
        this.drilledOnce.add(wpId); // remember so the main view shows the drilled icon
        this.cb.onDrillChange(wpId);
        this.render();
    }
    drillUp() {
        this.drillWp = null;
        this.cb.onDrillChange(null);
        this.render();
    }
    isDrilled() { return this.drillWp !== null; }
    x(ms) { return LEFT + ((ms - this.t0) / DAY) * this.ppd(); }
    // dateAt is the inverse of x(): the date (ms, snapped to whole days) at an
    // SVG x coordinate. Used to create a milestone at the right-clicked position.
    dateAt(svgX) {
        const days = Math.round((svgX - LEFT) / this.ppd());
        return this.t0 + days * DAY;
    }
    snapDays(d) { const s = SNAP[this.zoom] ?? 1; return Math.round(d / s) * s; }
    // ppd returns the pixels-per-day scale for the current zoom. For the fixed
    // levels it is a constant; for "window" it is computed so the whole plan
    // (bounds lo..hi) fits inside the visible canvas width (fit-to-view / zoom
    // window). Falls back to the "days" scale if the width is not yet known.
    ppd() {
        if (this.zoom !== "window")
            return PPD[this.zoom] ?? PPD.months;
        const [lo, hi] = this.bounds();
        const days = Math.max(1, (hi - lo) / DAY);
        const host = this.svg.parentElement;
        const avail = (host?.clientWidth || this.svg.clientWidth || 1000) - LEFT - 48;
        const scale = avail / days;
        // clamp to sane bounds: never finer than the days level, never below the
        // year level, so "window" stays within the allowed zoom range (R-7.4).
        return Math.max(PPD.year, Math.min(PPD.days, scale));
    }
    // roleRowsFor returns the per-role sub-rows to draw under an action in role
    // mode: one row per role assigned to the action, with that role's share of
    // the action's estimation. Empty unless role mode + showRoleActions.
    roleRowsFor(wpId) {
        if (!(this.mode === "role" && this.showRoleActions) || !this.project)
            return [];
        const out = [];
        for (const r of this.project.roles ?? [])
            for (const a of r.assignments ?? [])
                if (a.wp === wpId)
                    out.push({ role: r.name, percent: a.percent });
        return out;
    }
    buildLanes() {
        const p = this.project;
        // group actions by cluster, preserving first-seen order; "" => default lane
        const order = [];
        const byCluster = {};
        const ensure = (c) => { if (!(c in byCluster)) {
            byCluster[c] = [];
            order.push(c);
        } };
        // declared sections (even empty ones) show as lanes
        Object.keys(p.sectionColors ?? {}).forEach((c) => { if (c)
            ensure(c); });
        (p.packages ?? []).forEach((w) => {
            if (this.visible && !this.visible.has(w.id))
                return;
            const c = w.cluster || "";
            ensure(c);
            byCluster[c].push(w);
        });
        if (order.length === 0) {
            order.push("");
            byCluster[""] = [];
        }
        const isPhases = (c) => c.toLowerCase() === "phases";
        // ADD: when "Show Phases" is on, the Phases view must always have a lane at
        // the top — synthesize an (empty) "Phases" lane if none exists yet, so the
        // toggle actually produces a visible section.
        if (p.showPhases && !order.some(isPhases)) {
            ensure("Phases");
        }
        // BUG 3: sort lanes numerically by WP number (first work package on top).
        // The "Phases" view (no number) always sorts FIRST; unnumbered/default
        // lanes sort last. ADD: when showPhases is off, drop the Phases lane.
        const rank = (c) => {
            if (isPhases(c))
                return -1; // Phases always first
            const n = p.sectionNums?.[c];
            return n && n > 0 ? n : 1e9; // unnumbered last
        };
        let ordered = order.slice().sort((a, b) => rank(a) - rank(b));
        if (!p.showPhases)
            ordered = ordered.filter((c) => !isPhases(c));
        this.lanes = [];
        this.rowOf = {};
        this.laneOf = {};
        this.detailRowOf = {};
        this.detailParent = {};
        this.roleRowOf = {};
        let y = TOP;
        for (const c of ordered) {
            const acts = byCluster[c];
            // Row accounting. A "same line" action (and the first action never is)
            // shares the previous action's row instead of taking its own. Details and
            // role sub-rows still add rows, attributed to the action that owns them.
            let rows = 0;
            acts.forEach((w, idx) => {
                const shares = idx > 0 && w.sameLine;
                rows += (shares ? 0 : 1)
                    + (this.showDetails && !w.detailsHidden ? (w.details?.length ?? 0) : 0)
                    + this.roleRowsFor(w.id).length;
            });
            rows = Math.max(1, rows);
            const h = rows * ROW_H + LANE_PAD * 2;
            const lane = { cluster: c, actions: acts, y, h };
            this.lanes.push(lane);
            let ry = y + LANE_PAD;
            let prevRow = ry;
            acts.forEach((w, idx) => {
                if (idx > 0 && w.sameLine) {
                    // share the previous action's row; do NOT advance ry
                    this.rowOf[w.id] = prevRow;
                    this.laneOf[w.id] = lane;
                }
                else {
                    this.rowOf[w.id] = ry;
                    this.laneOf[w.id] = lane;
                    prevRow = ry;
                    ry += ROW_H;
                }
                if (this.showDetails && !w.detailsHidden) {
                    (w.details ?? []).forEach((d) => { this.detailRowOf[d.id] = ry; this.detailParent[d.id] = w.id; ry += ROW_H; });
                }
                this.roleRowsFor(w.id).forEach((rr) => {
                    var _a, _b;
                    ((_a = this.roleRowOf)[_b = w.id] ?? (_a[_b] = [])).push({ role: rr.role, percent: rr.percent, y: ry });
                    ry += ROW_H;
                });
            });
            y += h;
        }
    }
    bounds() {
        const p = this.project;
        let lo = Infinity, hi = -Infinity;
        const take = (s) => { const a = parseDate(s); if (isFinite(a)) {
            lo = Math.min(lo, a);
            hi = Math.max(hi, a);
        } };
        (p.milestones ?? []).forEach((m) => take(m.date));
        (p.phases ?? []).forEach((ph) => { take(ph.start); take(ph.end); });
        (p.packages ?? []).forEach((w) => { take(w.start); take(w.end); });
        if (!isFinite(lo) || !isFinite(hi)) {
            const now = Date.UTC(new Date().getUTCFullYear(), new Date().getUTCMonth(), 1);
            lo = now;
            hi = now + 30 * DAY;
        }
        return [lo - 7 * DAY, hi + 7 * DAY];
    }
    render() {
        this.head = null;
        this.renderInner();
        this.finishHead();
    }
    // finishHead puts the date row on top of everything and sticks it to the top.
    finishHead() {
        if (!this.head)
            return;
        this.svg.appendChild(this.head);
        this.applyStick();
    }
    // applyStick moves the date row down by the scroll offset (the white
    // background is only visible while scrolled, so the unscrolled chart looks
    // exactly as before).
    applyStick() {
        const h = this.head;
        if (!h)
            return;
        const st = Math.max(0, this.host.scrollTop || 0);
        h.setAttribute("transform", st > 0 ? `translate(0 ${st})` : "");
        const bg = h.querySelector(".stickybg");
        if (bg)
            bg.setAttribute("visibility", st > 0 ? "visible" : "hidden");
    }
    renderInner() {
        const p = this.project;
        this.svg.innerHTML = "";
        if (!p)
            return;
        if (this.drillWp) {
            this.renderDrill();
            return;
        }
        this.buildLanes();
        const [lo, hi] = this.bounds();
        this.t0 = lo;
        const width = this.x(hi) + 40;
        const lastLane = this.lanes[this.lanes.length - 1];
        const height = (lastLane ? lastLane.y + lastLane.h : TOP) + 40;
        this.svg.setAttribute("width", String(width));
        this.svg.setAttribute("height", String(height));
        this.svg.setAttribute("viewBox", `0 0 ${width} ${height}`);
        this.drawAxis(lo, hi, height);
        // swimlane bands + left labels
        // swimlane bands + left labels (work packages: "WP-n Title")
        this.lanes.forEach((lane, i) => {
            const sc = p.sectionColors?.[lane.cluster];
            const color = sc && sc !== "" ? sc : (i % 2 ? "#f6f6fb" : "#eef0fa");
            const band = this.rect(0, lane.y, width, lane.h, color, "", 0);
            band.style.cursor = "pointer";
            band.addEventListener("click", () => this.cb.onSelect("section", lane.cluster));
            // selection border around the selected section band
            if (this.selectedSection != null && this.selectedSection === lane.cluster) {
                const sb = this.rect(1, lane.y + 1, width - 2, lane.h - 2, "transparent", "#111", 2);
                sb.setAttribute("stroke-dasharray", "5 3");
                sb.setAttribute("rx", "3");
                sb.setAttribute("class", "editchrome"); // editor-only; stripped from exports
            }
            this.line(0, lane.y, width, lane.y, "#d0d0dc", 1);
            const wpn = p.sectionNums?.[lane.cluster];
            const prefix = wpn ? `WP-${wpn} ` : "";
            const label = lane.cluster ? `${prefix}${lane.cluster}` : (wpn ? `WP-${wpn}` : "(no section)");
            this.text(8, lane.y + 16, label, "#333", 12);
        });
        this.line(LEFT, TOP, LEFT, height - 30, "#c4c4d4", 1);
        (p.milestones ?? []).forEach((m) => {
            const t = parseDate(m.date);
            if (!isFinite(t))
                return;
            const mx = this.x(t);
            if (this.isFullHeightMilestone(m)) {
                // project Start/End, an unclustered milestone, OR a Phases-row
                // milestone: a full top-to-bottom dashed line. Draggable by the mouse
                // (horizontal = date), but NOT a link endpoint.
                const col = m.color || "#b44";
                const g = document.createElementNS(SVGNS, "g");
                this.line(mx, TOP, mx, height - 30, col, 1, "3 3");
                this.text(mx + 3, TOP + 10, m.name, col, 10, g);
                // a transparent hit strip over the line for selecting + dragging
                const hit = this.rect(mx - 4, TOP, 8, height - 30 - TOP, "transparent", "", 0, g);
                hit.style.cursor = this.linkMode ? "not-allowed" : "ew-resize";
                if (!this.linkMode) {
                    hit.addEventListener("click", (e) => { e.stopPropagation(); this.cb.onMilestoneSelect(m.id); });
                    this.enableMilestoneDrag(hit, m.id, t);
                }
                this.svg.appendChild(g);
                return;
            }
            // user milestone: a selectable, draggable, linkable diamond in its lane
            const lane = this.lanes.find((l) => l.cluster === m.cluster);
            const cy = lane.y + Math.min(lane.h - 10, ROW_H) / 2 + 4;
            const col = m.color || "#b44";
            const g = document.createElementNS(SVGNS, "g");
            const d = 7;
            const dia = document.createElementNS(SVGNS, "path");
            dia.setAttribute("d", `M ${mx} ${cy - d} L ${mx + d} ${cy} L ${mx} ${cy + d} L ${mx - d} ${cy} Z`);
            dia.setAttribute("fill", col);
            dia.setAttribute("stroke", this.selectedMilestone === m.id ? "#111" : "#7a2");
            dia.setAttribute("stroke-width", this.selectedMilestone === m.id ? "2.5" : "1");
            dia.style.cursor = this.linkMode ? "crosshair" : "move";
            dia.addEventListener("click", (e) => {
                e.stopPropagation();
                if (this.linkMode)
                    this.handleLinkClick(m.id); // link to/from this milestone
                else
                    this.cb.onMilestoneSelect(m.id);
            });
            if (!this.linkMode)
                this.enableMilestoneDrag(dia, m.id, t);
            g.appendChild(dia);
            const lbl = m.displayId ? `${m.displayId} ${m.name}` : m.name;
            this.text(mx + d + 3, cy + 4, lbl, "#333", 10, g);
            // link anchors (l/r) so a dependency can be drawn FROM the milestone by
            // dragging (milestone is a point, so only the two horizontal anchors).
            this.addMilestoneAnchors(g, m.id, mx, cy);
            this.svg.appendChild(g);
        });
        (p.packages ?? []).forEach((w) => {
            // action n.k id in the left column, aligned to its row
            const ry = this.rowOf[w.id];
            if (ry != null)
                this.text(96, ry + ROW_H / 2 + 4, w.displayId, "#777", 11);
            this.drawBar(w, ry);
            if (w.noteVisible && w.desc && this.project?.reportShowNotes !== false)
                this.drawNote(w, ry);
            if (this.showDetails && !w.detailsHidden)
                (w.details ?? []).forEach((d) => this.drawDetail(w, d));
            (this.roleRowOf[w.id] ?? []).forEach((rr) => this.drawRoleRow(w, rr));
        });
        (p.deps ?? []).forEach((d) => this.drawDep(d));
        // today marker on top (report/settings option, default on)
        this.drawTodayLine(height);
        // Section click targets: the left label column (x < LEFT) for each lane,
        // drawn LAST so nothing overlays them. Clicking a section label opens the
        // section dialog in the right pane. (The band itself is also clickable but
        // gridlines/bars paint over it, so this transparent strip guarantees the
        // hit.)
        this.lanes.forEach((lane) => {
            const hit = this.rect(0, lane.y, LEFT, lane.h, "transparent", "", 0);
            hit.style.cursor = "pointer";
            hit.addEventListener("click", (e) => { e.stopPropagation(); this.cb.onSelect("section", lane.cluster); });
        });
        if (this.linkMode) {
            this.text(LEFT + 6, TOP - 40, this.linkFrom
                ? `LINK: click target action (from ${this.linkFrom})`
                : "LINK MODE: click source action (Esc to cancel)", "#534fbc", 12);
        }
    }
    // Render the third-level plane: the drilled action's detail packages as
    // first-class draggable bars on a time axis bounded by the action window.
    renderDrill() {
        const p = this.project;
        const wp = p.packages.find((w) => w.id === this.drillWp);
        if (!wp) {
            this.drillUp();
            return;
        }
        const lo = parseDate(wp.start), hi = parseDate(wp.end);
        if (!isFinite(lo) || !isFinite(hi)) {
            this.drillUp();
            return;
        }
        this.t0 = lo - 2 * DAY;
        const details = wp.details ?? [];
        // Each detail takes one row; when "Show role actions" is on, a detail that
        // has a role gets an extra sub-row beneath it (a detail carries a single
        // role, i.e. 100% of its own effort).
        const roleRowFor = (d) => this.showRoleActions && !!d.role;
        const width = this.x(hi) + 60;
        let rowCount = 0;
        details.forEach((d) => { rowCount += 1 + (roleRowFor(d) ? 1 : 0); });
        const rows = Math.max(1, rowCount);
        const height = TOP + rows * ROW_H + 60;
        this.svg.setAttribute("width", String(width));
        this.svg.setAttribute("height", String(height));
        this.svg.setAttribute("viewBox", `0 0 ${width} ${height}`);
        this.detailRowOf = {};
        this.drawAxis(lo - 2 * DAY, hi + 2 * DAY, height);
        // header: breadcrumb WP-n.k + Back (U)
        this.text(8, 18, `▸ ${wp.displayId}  ${wp.name}   (press U to go up)`, "#534fbc", 13);
        // Back button: group both rect AND label into one <g> that carries the
        // click handler, so clicking the glyph (which paints over the rect) still
        // triggers it. The label is pointer-transparent for good measure.
        const backG = document.createElementNS(SVGNS, "g");
        backG.style.cursor = "pointer";
        const back = this.rect(8, 26, 70, 20, "#eef0fa", "#534fbc", 1, backG);
        const backLbl = this.text(18, 40, "▲ up", "#534fbc", 12, backG);
        backLbl.style.pointerEvents = "none";
        backG.addEventListener("click", () => this.drillUp());
        void back;
        this.svg.appendChild(backG);
        const addG = document.createElementNS(SVGNS, "g");
        addG.style.cursor = "pointer";
        this.rect(86, 26, 90, 20, "#534fbc", "#534fbc", 1, addG);
        const addLbl = this.text(94, 40, "+ detail", "#fff", 12, addG);
        addLbl.style.pointerEvents = "none";
        addG.addEventListener("click", () => this.cb.onDetailAdd(wp.id));
        this.svg.appendChild(addG);
        if (this.linkMode) {
            this.text(184, 40, this.linkFrom ? "LINK: click target detail" : "LINK: click source detail", "#534fbc", 12);
        }
        // parent window shading
        this.rect(this.x(lo), TOP, this.x(hi) - this.x(lo), rows * ROW_H, "#f6f6fb", "#d0d0dc", 1);
        this.detailRowY = TOP;
        details.forEach((d) => {
            const y = this.detailRowY;
            this.detailRowOf[d.id] = y;
            this.text(8, y + ROW_H / 2 + 4, d.displayId, "#777", 11);
            this.drawDrillBar(wp.id, d, y, lo, hi);
            this.detailRowY += ROW_H;
            if (roleRowFor(d)) {
                this.drawDetailRoleRow(d, this.detailRowY, lo, hi);
                this.detailRowY += ROW_H;
            }
        });
        if (details.length === 0) {
            this.text(this.x(lo) + 10, TOP + ROW_H / 2, "no detail packages yet — click + detail", "#999", 12);
        }
        // detail-level dependencies: draw arrows for deps whose BOTH endpoints are
        // details of this action.
        const detIds = new Set(details.map((d) => d.id));
        (p.deps ?? []).forEach((dep) => {
            if (detIds.has(dep.from) && detIds.has(dep.to))
                this.drawDetailDep(details, dep);
        });
    }
    // drawDetailDep draws an arrow between two detail bars in the drill plane,
    // honouring the link's anchors (so SS/FF/OV links and graphical anchor links
    // route from the chosen edges, not always right->left).
    drawDetailDep(details, dep) {
        const a = details.find((d) => d.id === dep.from), b = details.find((d) => d.id === dep.to);
        if (!a || !b)
            return;
        const ay = this.detailRowOf[a.id], byy = this.detailRowOf[b.id];
        if (ay == null || byy == null)
            return;
        const fa = (dep.fromAnchor || defaultFromAnchor(dep.kind || "FS"));
        const ta = (dep.toAnchor || defaultToAnchor(dep.kind || "FS"));
        const detAnchor = (d, row, an) => {
            const x0 = this.x(parseDate(d.start)), x1 = this.x(parseDate(d.end));
            const midY = row + ROW_H / 2;
            switch (an) {
                case "l": return { x: x0, y: midY };
                case "r": return { x: x1, y: midY };
                case "t": return { x: (x0 + x1) / 2, y: midY - BAR_H / 2 };
                case "b": return { x: (x0 + x1) / 2, y: midY + BAR_H / 2 };
            }
        };
        const ap = detAnchor(a, ay, fa), bp = detAnchor(b, byy, ta);
        const color = dep.violated ? "#e03131" : (dep.color || "#555");
        const out = this.stub(fa, 24), inn = this.stub(ta, 24);
        const c1 = { x: ap.x + out.x, y: ap.y + out.y }, c2 = { x: bp.x + inn.x, y: bp.y + inn.y };
        const path = document.createElementNS(SVGNS, "path");
        path.setAttribute("d", `M ${ap.x} ${ap.y} C ${c1.x} ${c1.y} ${c2.x} ${c2.y} ${bp.x} ${bp.y}`);
        path.setAttribute("fill", "none");
        path.setAttribute("stroke", color);
        path.setAttribute("stroke-width", "1.5");
        this.svg.appendChild(path);
        const ah = document.createElementNS(SVGNS, "path");
        ah.setAttribute("d", this.arrowHead(bp, ta));
        ah.setAttribute("fill", color);
        this.svg.appendChild(ah);
    }
    drawDrillBar(wpId, d, y, lo, hi) {
        const ps = parseDate(d.start), pe = parseDate(d.end);
        if (!isFinite(ps) || !isFinite(pe))
            return;
        const x0 = this.x(ps), x1 = this.x(pe);
        const by = y + (ROW_H - BAR_H) / 2;
        const g = document.createElementNS(SVGNS, "g");
        if (d.summary) {
            // Sammelvorgang: the same solid thick rollup bar with downward end-cap
            // wedges that a summary ACTION uses (drawBar), so a summary detail reads
            // the same on the detail plane. Label sits to the right of the right end.
            const col = "#333", barH = 7;
            const xL = Math.round(x0), xR = Math.max(xL + 4, Math.round(x1));
            const sy = Math.round(by), capW = 9, capH = 8, yb = sy + barH;
            const path = [
                `M ${xL} ${sy}`, `L ${xR} ${sy}`, `L ${xR} ${yb + capH}`,
                `L ${xR - capW} ${yb}`, `L ${xL + capW} ${yb}`, `L ${xL} ${yb + capH}`, `Z`,
            ].join(" ");
            const bar = document.createElementNS(SVGNS, "path");
            bar.setAttribute("d", path);
            bar.setAttribute("fill", col);
            bar.setAttribute("shape-rendering", "crispEdges");
            if (this.selectedDetail === d.id) {
                bar.setAttribute("stroke", "#534fbc");
                bar.setAttribute("stroke-width", "2");
            }
            g.appendChild(bar);
            const sl = d.est && d.est !== "0" ? `${d.name}  ${d.est}` : d.name;
            const [hx0, hx1] = this.placeSummaryLabel(g, xL, xR, sy, barH, capH, sl, d.textPos, d.fontColor);
            // a transparent hit area over the whole block (thin spine is hard to hit)
            const hit = this.rect(hx0, y, hx1 - hx0, ROW_H, "transparent", "", 0, g);
            if (this.linkMode) {
                hit.style.cursor = "crosshair";
                hit.addEventListener("click", (e) => { e.stopPropagation(); this.handleDetailLinkClick(d.id); });
            }
            else {
                hit.style.cursor = "pointer";
                hit.addEventListener("click", (e) => { e.stopPropagation(); this.selectedDetail = d.id; this.cb.onDetailSelect(wpId, d.id); });
                hit.addEventListener("dblclick", () => this.cb.onDetailOpen(wpId, d.id));
                this.addDetailAnchors(g, d.id, xL, xR, by);
            }
            this.svg.appendChild(g);
            return;
        }
        const fill = d.role ? this.roleColor(d.role) : "#8a90dd";
        const bar = this.rect(x0, by, Math.max(4, x1 - x0), BAR_H, fill, "#534fbc", 1, g);
        const label = d.est && d.est !== "0" ? `${d.name}  ${d.est}` : d.name;
        this.placeLabel(g, x0, x1, by + BAR_H / 2, BAR_H, label, 11, d.textPos, this.labelColor(fill, d.fontColor), d.fontColor);
        if (this.selectedDetail === d.id)
            bar.setAttribute("stroke-width", "2.5");
        if (this.linkMode) {
            // link DETAILS: first click = source, next = target (or source preset
            // via select+L).
            bar.style.cursor = "crosshair";
            bar.addEventListener("click", (e) => { e.stopPropagation(); this.handleDetailLinkClick(d.id); });
        }
        else {
            bar.style.cursor = "grab";
            const hL = this.rect(x0 - 3, by, 6, BAR_H, "transparent", "", 0, g);
            const hR = this.rect(x1 - 3, by, 6, BAR_H, "transparent", "", 0, g);
            hL.style.cursor = "ew-resize";
            hR.style.cursor = "ew-resize";
            this.enableDetailDrag(bar, d, "move", lo, hi);
            this.enableDetailDrag(hL, d, "l", lo, hi);
            this.enableDetailDrag(hR, d, "r", lo, hi);
            bar.addEventListener("click", (e) => { e.stopPropagation(); this.selectedDetail = d.id; this.cb.onDetailSelect(wpId, d.id); });
            bar.addEventListener("dblclick", () => this.cb.onDetailOpen(wpId, d.id));
            // hover anchors (l/r/t/b) so a detail can be linked graphically by
            // dragging a red routing line to another detail — same affordance as the
            // main-view action bars.
            this.addDetailAnchors(g, d.id, x0, x1, by);
        }
        this.svg.appendChild(g);
    }
    // addDetailAnchors draws the four l/r/t/b anchor dots on a detail bar and
    // wires each to startDetailAnchorLink (drill-plane analogue of addAnchors).
    addDetailAnchors(g, detId, x0, x1, by) {
        const midY = by + BAR_H / 2;
        const pts = {
            l: [x0, midY], r: [x1, midY], t: [(x0 + x1) / 2, by], b: [(x0 + x1) / 2, by + BAR_H],
        };
        Object.keys(pts).forEach((a) => {
            const [cx, cy] = pts[a];
            const dot = document.createElementNS(SVGNS, "circle");
            dot.setAttribute("cx", String(cx));
            dot.setAttribute("cy", String(cy));
            dot.setAttribute("r", "4");
            dot.setAttribute("class", "anchor");
            dot.setAttribute("fill", "#534fbc");
            dot.addEventListener("pointerdown", (ev) => { ev.stopPropagation(); this.startDetailAnchorLink(detId, a, ev); });
            g.appendChild(dot);
        });
    }
    // startDetailAnchorLink drags a red routing line from a detail anchor and, on
    // release over another detail of the SAME drilled action, fires onLink with
    // the source anchor and the nearest anchor on the target detail.
    startDetailAnchorLink(fromId, fromAnchor, ev) {
        ev.preventDefault();
        const line = this.line(0, 0, 0, 0, "#e03131", 2, "4 3"); // red routing line
        const rect = this.svg.getBoundingClientRect();
        const sp = this.svgPoint(ev.clientX, ev.clientY, rect);
        const onMove = (e) => {
            const pt = this.svgPoint(e.clientX, e.clientY, rect);
            line.setAttribute("x1", String(sp.x));
            line.setAttribute("y1", String(sp.y));
            line.setAttribute("x2", String(pt.x));
            line.setAttribute("y2", String(pt.y));
        };
        const onUp = (e) => {
            window.removeEventListener("pointermove", onMove);
            window.removeEventListener("pointerup", onUp);
            line.remove();
            const pt = this.svgPoint(e.clientX, e.clientY, rect);
            const target = this.detailAt(pt.y);
            if (target && target !== fromId)
                this.cb.onLink(fromId, target, fromAnchor, this.nearestDetailAnchor(target, pt));
        };
        window.addEventListener("pointermove", onMove);
        window.addEventListener("pointerup", onUp);
    }
    // detailAt returns the detail id whose drill row contains the y coordinate.
    detailAt(y) {
        for (const id in this.detailRowOf) {
            const top = this.detailRowOf[id];
            if (y >= top && y < top + ROW_H)
                return id;
        }
        return null;
    }
    // nearestDetailAnchor picks the closest l/r/t/b anchor of a detail bar to a
    // point (used to infer the target anchor of a dragged drill link).
    nearestDetailAnchor(detId, p) {
        const d = this.detailById(detId);
        if (!d)
            return "l";
        const x0 = this.x(parseDate(d.start)), x1 = this.x(parseDate(d.end));
        const y = (this.detailRowOf[detId] ?? 0) + ROW_H / 2;
        const cand = {
            l: [x0, y], r: [x1, y], t: [(x0 + x1) / 2, y - BAR_H / 2], b: [(x0 + x1) / 2, y + BAR_H / 2],
        };
        let best = "l", bd = Infinity;
        Object.keys(cand).forEach((a) => { const [cx, cy] = cand[a]; const dd = (cx - p.x) ** 2 + (cy - p.y) ** 2; if (dd < bd) {
            bd = dd;
            best = a;
        } });
        return best;
    }
    // detailById finds a detail across all packages (the drilled action's details
    // share the global dependency space).
    detailById(detId) {
        for (const w of this.project?.packages ?? []) {
            const d = (w.details ?? []).find((x) => x.id === detId);
            if (d)
                return d;
        }
        return null;
    }
    // handleDetailLinkClick completes a detail→detail link inside the drill plane.
    handleDetailLinkClick(id) {
        if (!this.linkFrom) {
            this.linkFrom = id;
            this.render();
            return;
        }
        if (this.linkFrom !== id)
            this.cb.onLink(this.linkFrom, id, "r", "l");
        this.linkFrom = null;
        this.setLinkMode(false);
    }
    enableDetailDrag(el, d, mode, lo, hi) {
        el.addEventListener("pointerdown", (ev) => {
            ev.preventDefault();
            const startX = ev.clientX;
            const origStart = parseDate(d.start), origEnd = parseDate(d.end);
            const clamp = (v) => Math.max(lo, Math.min(hi, v));
            const compute = (clientX) => {
                const dDays = this.snapDays((clientX - startX) / this.ppd());
                let ns = origStart, ne = origEnd;
                if (mode === "move") {
                    ns = clamp(origStart + dDays * DAY);
                    ne = clamp(origEnd + dDays * DAY);
                }
                else if (mode === "l") {
                    ns = clamp(Math.min(origStart + dDays * DAY, origEnd - DAY));
                }
                else {
                    ne = clamp(Math.max(origEnd + dDays * DAY, origStart + DAY));
                }
                return [ns, ne];
            };
            const onMove = (e) => {
                const [ns, ne] = compute(e.clientX);
                this.cb.onDragStatus?.(dragStatusText({ name: d.name, mode, origStart, origEnd, start: ns, end: ne }));
            };
            const onUp = (e) => {
                window.removeEventListener("pointermove", onMove);
                window.removeEventListener("pointerup", onUp);
                this.cb.onDragStatus?.(null);
                const [ns, ne] = compute(e.clientX);
                this.cb.onDetailMove(this.drillWp, d.id, fmtDate(ns), fmtDate(ne));
            };
            window.addEventListener("pointermove", onMove);
            window.addEventListener("pointerup", onUp);
        });
    }
    // drawProgress adds the Contour_Curve (a polyline of the work contour shape,
    // omitted for bars narrower than 8px) and the Progress_Marker (a vertical tick
    // at the contour-adjusted completion position) to an action bar's group. Both
    // ignore pointer events so bar selection/drag keep working.
    drawProgress(g, w, x0, x1, by, fill) {
        const pos0 = Math.min(1, Math.max(0, w.completionPos || 0));
        // the effort done = the area under the curve up to the marker, shaded
        const area = contourAreaPoints({ x0, x1, by, h: BAR_H }, w.contour, pos0);
        if (area.length > 0) {
            const poly = document.createElementNS(SVGNS, "polygon");
            poly.setAttribute("points", area.map(([px, py]) => `${px},${py}`).join(" "));
            poly.setAttribute("fill", curveColor(fill));
            poly.setAttribute("fill-opacity", "0.3");
            poly.setAttribute("class", "progress-area");
            poly.setAttribute("pointer-events", "none");
            g.appendChild(poly);
        }
        const tip = document.createElementNS(SVGNS, "title");
        const dec = this.project?.language === "de";
        const unit = this.project?.unit ?? "days";
        tip.textContent = `Planned ${effortText(unit, w.estHours, w.estDays)}, real ${effortText(unit, w.realHours, w.realDays)}: ${ratingLabel(true, w.progressPermille, w.contourPermille, w.contour, dec)} consumed`
            + (w.synced ? " (real effort synchronized)" : "");
        g.appendChild(tip);
        const pts = contourCurvePoints({ x0, x1, by, h: BAR_H }, w.contour);
        if (pts.length > 0) {
            const pl = document.createElementNS(SVGNS, "polyline");
            pl.setAttribute("points", pts.map(([px, py]) => `${px},${py}`).join(" "));
            pl.setAttribute("fill", "none");
            pl.setAttribute("stroke", curveColor(fill));
            pl.setAttribute("stroke-width", "1");
            pl.setAttribute("class", "progress-curve");
            pl.setAttribute("pointer-events", "none");
            g.appendChild(pl);
        }
        const pos = Math.min(1, Math.max(0, w.completionPos || 0));
        const mx = x0 + pos * (x1 - x0);
        const mk = this.line(mx, by - 2, mx, by + BAR_H + 2, markerColor(fill), 2);
        mk.setAttribute("class", "progress-marker");
        mk.setAttribute("pointer-events", "none");
        g.appendChild(mk);
    }
    drawBar(w, y) {
        if (y == null)
            return;
        const ps = parseDate(w.start), pe = parseDate(w.end);
        if (!isFinite(ps) || !isFinite(pe))
            return;
        const x0 = this.x(ps), x1 = this.x(pe);
        const by = y + (ROW_H - BAR_H) / 2;
        const g = document.createElementNS(SVGNS, "g");
        if (w.summary) {
            // Summary (Summiervorgang): a SOLID thick bar (MS-Project rollup style),
            // with downward end-cap wedges at both ends — the left wedge drops from
            // the bar's bottom-left corner, the right wedge from the bottom-right
            // corner. The name label sits to the RIGHT of the right end.
            // Summary bars use the MS-Project black rollup look regardless of the
            // action's bar color (the color applies to normal bars only).
            const col = "#333";
            const barH = 7; // thick solid bar
            // Snap the span ends to whole device pixels so the bar and the end-cap
            // wedges share EXACT edges — fractional x from this.x() previously left a
            // 1px seam/gap at the left and right ends of the Sammelvorgang.
            const xL = Math.round(x0);
            const xR = Math.max(xL + 4, Math.round(x1));
            const sy = Math.round(by); // top of the bar sits at the row bar-top
            // Draw the whole rollup as a SINGLE filled polygon (bar body + both
            // downward end-cap wedges). One path = no internal seams between the bar
            // and the wedges, and no sub-pixel gaps at the ends.
            const capW = 9, capH = 8; // end-cap wedge size
            const yb = sy + barH; // bar bottom = wedge top
            const d = [
                `M ${xL} ${sy}`, // top-left
                `L ${xR} ${sy}`, // top-right
                `L ${xR} ${yb + capH}`, // down the right edge to the right wedge tip
                `L ${xR - capW} ${yb}`, // right wedge inner corner (back up to bar bottom)
                `L ${xL + capW} ${yb}`, // across to the left wedge inner corner
                `L ${xL} ${yb + capH}`, // down-left to the left wedge tip
                `Z`, // close up the left edge back to top-left
            ].join(" ");
            const bar = document.createElementNS(SVGNS, "path");
            bar.setAttribute("d", d);
            bar.setAttribute("fill", col);
            bar.setAttribute("shape-rendering", "crispEdges");
            bar.style.cursor = this.linkMode ? "crosshair" : "grab";
            g.appendChild(bar);
            const w0 = xR - xL;
            if (w.hasWarning) {
                // red warning outline around the summary span
                this.rect(xL - 1, sy - 2, w0 + 2, barH + capH + 4, "transparent", "#e03131", 2, g);
            }
            // label (name + optional effort) at the chosen text position; default:
            // to the right of the end
            const sl = w.est && w.est !== "0" ? `${w.name}  ${w.est}` : w.name;
            const [hx0, hx1] = this.placeSummaryLabel(g, xL, xR, sy, barH, capH, sl, w.textPos, w.fontColor);
            // A summary's thin spine is hard to click, so add a transparent hit area
            // covering the WHOLE block (the full row height across the span and the
            // label). This hit rect is the click/dblclick/link target.
            const hit = this.rect(hx0, y, hx1 - hx0, ROW_H, "transparent", "", 0, g);
            hit.style.cursor = this.linkMode ? "crosshair" : "pointer";
            if (this.linkMode) {
                hit.addEventListener("click", () => this.handleLinkClick(w.id));
            }
            else {
                hit.addEventListener("click", (e) => { e.stopPropagation(); this.cb.onSelect("action", w.id); });
                hit.addEventListener("dblclick", (e) => { e.stopPropagation(); this.cb.onOpen(w.id); });
                // hover anchors so a summary can also be linked from its edges
                this.addAnchors(g, w, xL, xR, by);
                // right-edge drill icon (smaller on the thin summary bar) when it has
                // details — single click drills in; cursor shows a plus (zoom-in).
                if ((w.details?.length ?? 0) > 0 || this.drilledOnce.has(w.id))
                    this.drawDrillIcon(w, y, g);
            }
        }
        else {
            const stroke = w.hasWarning ? "#e03131" : "#534fbc";
            const sw = w.hasWarning ? 2.5 : 1;
            const bar = this.rect(x0, by, Math.max(4, x1 - x0), BAR_H, w.color || "#8a90dd", stroke, sw, g);
            bar.style.cursor = this.linkMode ? "crosshair" : "grab";
            let label = w.est && w.est !== "0" ? `${w.name}  ${w.est}` : w.name;
            // with Show Progress the consumed rating(s) are part of the label
            if (this.showProgress && w.contour !== "None" && w.est && w.est !== "0") {
                label += "  " + ratingLabel(true, w.progressPermille, w.contourPermille, w.contour, this.project?.language === "de");
            }
            // progress graphics first, so the label text is drawn on top of them
            if (this.showProgress && w.contour !== "None")
                this.drawProgress(g, w, x0, x1, by, w.color || "#8a90dd");
            this.placeLabel(g, x0, x1, by + BAR_H / 2, BAR_H, label, 11, w.textPos, this.labelColor(w.color, w.fontColor), w.fontColor);
            if (this.linkMode) {
                bar.addEventListener("click", () => this.handleLinkClick(w.id));
            }
            else {
                const hL = this.rect(x0 - 3, by, 6, BAR_H, "transparent", "", 0, g);
                const hR = this.rect(x1 - 3, by, 6, BAR_H, "transparent", "", 0, g);
                hL.style.cursor = "ew-resize";
                hR.style.cursor = "ew-resize";
                this.enableDrag(bar, w, "move");
                this.enableDrag(hL, w, "l");
                this.enableDrag(hR, w, "r");
                bar.addEventListener("click", (e) => { e.stopPropagation(); this.cb.onSelect("action", w.id); });
                // Double-click opens the editor (same for normal + summary actions).
                // Drilling into details is done via the right-edge drill icon (single
                // click), not double-click.
                bar.addEventListener("dblclick", (e) => { e.stopPropagation(); this.cb.onOpen(w.id); });
                this.addAnchors(g, w, x0, x1, by);
                // right-edge drill icon when the action has details — single click
                // drills in; cursor shows a plus (zoom-in).
                if ((w.details?.length ?? 0) > 0 || this.drilledOnce.has(w.id))
                    this.drawDrillIcon(w, y, g);
            }
        }
        // selection border: a dashed outline around the selected action's bar
        if (this.selected === w.id) {
            const sy = w.summary ? by - 3 : by - 3;
            const sh = w.summary ? 13 : BAR_H + 6;
            const sel = this.rect(x0 - 3, sy, Math.max(4, x1 - x0) + 6, sh, "transparent", "#111", 2, g);
            sel.setAttribute("stroke-dasharray", "4 3");
            sel.setAttribute("rx", "3");
            sel.setAttribute("class", "editchrome"); // editor-only; stripped from exports
        }
        this.svg.appendChild(g);
    }
    // drawNote draws the action's Description as a small YELLOW sticky note near
    // the bar, connected to the nearest bar edge by a plain (arrowless) line.
    // The note is placed relative to the bar per w.notePos (up/down/left/outleft/
    // right/outright/center; default up), at a short distance. Small letters.
    drawNote(w, y) {
        if (y == null)
            return;
        const ps = parseDate(w.start), pe = parseDate(w.end);
        if (!isFinite(ps) || !isFinite(pe))
            return;
        const x0 = this.x(ps), x1 = this.x(pe);
        const barMidY = y + ROW_H / 2;
        const text = w.desc.trim();
        if (!text)
            return;
        // wrap the note text into short lines (~22 chars) so the pad stays small.
        const lines = wrapText(text, 22, 4);
        const fontSize = 9;
        const lineH = 12;
        const padX = 6, padY = 5;
        const noteW = Math.max(40, Math.min(180, maxLineLen(lines) * fontSize * 0.56 + padX * 2));
        const noteH = lines.length * lineH + padY * 2;
        // place the note at a short distance from the bar per notePos.
        const DIST = 16;
        const barCx = (x0 + x1) / 2;
        const pos = w.notePos || "up";
        let nx, ny; // note top-left
        switch (pos) {
            case "down":
                nx = barCx - noteW / 2;
                ny = y + ROW_H + DIST;
                break;
            case "left":
            case "outleft":
                nx = x0 - noteW - DIST;
                ny = barMidY - noteH / 2;
                break;
            case "right":
            case "outright":
                nx = x1 + DIST;
                ny = barMidY - noteH / 2;
                break;
            case "center":
                nx = barCx - noteW / 2;
                ny = barMidY - noteH / 2;
                break;
            // diagonal corners: the note sits beyond a bar corner so the connector
            // runs at ~45 degrees.
            case "ul":
                nx = x0 - noteW - DIST;
                ny = y - DIST - noteH;
                break;
            case "ur":
                nx = x1 + DIST;
                ny = y - DIST - noteH;
                break;
            case "ll":
                nx = x0 - noteW - DIST;
                ny = y + ROW_H + DIST;
                break;
            case "lr":
                nx = x1 + DIST;
                ny = y + ROW_H + DIST;
                break;
            default: /* up */
                nx = barCx - noteW / 2;
                ny = y - DIST - noteH;
                break;
        }
        const g = document.createElementNS(SVGNS, "g");
        // connector line (no arrowhead). For diagonal positions it runs corner to
        // corner at ~45 degrees; otherwise from the note center to the nearest bar
        // point.
        let fromX, fromY, toX, toY;
        const noteCx = nx + noteW / 2, noteCy = ny + noteH / 2;
        const barTop = y + (ROW_H - BAR_H) / 2, barBot = barTop + BAR_H;
        switch (pos) {
            case "ul":
                fromX = nx + noteW;
                fromY = ny + noteH;
                toX = x0;
                toY = barTop;
                break;
            case "ur":
                fromX = nx;
                fromY = ny + noteH;
                toX = x1;
                toY = barTop;
                break;
            case "ll":
                fromX = nx + noteW;
                fromY = ny;
                toX = x0;
                toY = barBot;
                break;
            case "lr":
                fromX = nx;
                fromY = ny;
                toX = x1;
                toY = barBot;
                break;
            default: {
                const anchorX = Math.max(x0, Math.min(x1, noteCx));
                fromX = noteCx;
                fromY = noteCy;
                toX = anchorX;
                toY = barMidY;
                break;
            }
        }
        this.line(fromX, fromY, toX, toY, "#c9a227", 1, "");
        // the yellow sticky pad
        const pad = this.rect(nx, ny, noteW, noteH, "#fff7cc", "#e3c44d", 1, g);
        pad.setAttribute("rx", "2");
        pad.style.cursor = "pointer";
        pad.addEventListener("click", (e) => { e.stopPropagation(); this.cb.onSelect("action", w.id); });
        lines.forEach((ln, i) => {
            this.text(nx + padX, ny + padY + (i + 1) * lineH - 3, ln, "#5b4e12", fontSize, g);
        });
        this.svg.appendChild(g);
    }
    // a thin bar spanning the action window, colored by role, labeled with the
    // role name and its share "percent% = <amount>" of the action estimation.
    drawRoleRow(parent, rr) {
        const ps = parseDate(parent.start), pe = parseDate(parent.end);
        if (!isFinite(ps) || !isFinite(pe))
            return;
        const x0 = this.x(ps), x1 = this.x(pe);
        const bh = BAR_H - 4;
        const by = rr.y + (ROW_H - bh) / 2;
        this.text(96, rr.y + ROW_H / 2 + 4, "↳", "#aaa", 11);
        const g = document.createElementNS(SVGNS, "g");
        this.rect(x0, by, Math.max(3, x1 - x0), bh, this.roleColor(rr.role), "#8a90dd", 1, g);
        const lbl = this.roleRowLabel(rr.role, rr.percent, this.roleShare(parent.est, rr.percent));
        this.text(x0 + 6, by + bh / 2 + 3, lbl, this.labelColor(this.roleColor(rr.role)), 10, g);
        this.svg.appendChild(g);
    }
    // roleRowLabel builds a role sub-row label honouring the two display
    // modifiers: by default (both off) it is the role NAME only; showRolePercent
    // appends " <p>%"; showRoleDays appends " = <share>". share may be "" (no
    // amount available), in which case the days part is omitted.
    roleRowLabel(role, percent, share) {
        let lbl = role;
        if (this.showRolePercent)
            lbl += `  ${percent}%`;
        if (this.showRoleDays && share)
            lbl += ` = ${share}`;
        return lbl;
    }
    // roleShare formats percent% of an effort DISPLAY string (e.g. "1,5"),
    // keeping the project's comma decimal, and appends the unit (d/h). Used for
    // both an action's role sub-rows and a detail's role sub-row.
    roleShare(estDisplay, percent) {
        const raw = (estDisplay || "0").replace(",", ".");
        const n = parseFloat(raw);
        if (!isFinite(n) || n === 0)
            return "";
        const v = (n * percent) / 100;
        const unit = this.project?.unit === "hours" ? "h" : "d";
        return `${v.toFixed(2).replace(/\.00$/, "").replace(".", ",")} ${unit}`;
    }
    // drawDetailRoleRow draws a role sub-row under a DETAIL bar in the drill
    // plane, mirroring drawRoleRow on the main view. A detail carries a single
    // role (no percentage), i.e. 100% of its own effort, so the percent shown is
    // 100% and the share is the detail's full estimation. The bar spans the
    // detail's own window.
    drawDetailRoleRow(d, y, lo, hi) {
        if (!d.role)
            return;
        const ps = parseDate(d.start), pe = parseDate(d.end);
        if (!isFinite(ps) || !isFinite(pe))
            return;
        void lo;
        void hi;
        const x0 = this.x(ps), x1 = this.x(pe);
        const bh = BAR_H - 4;
        const by = y + (ROW_H - bh) / 2;
        this.text(96, y + ROW_H / 2 + 4, "↳", "#aaa", 11);
        const g = document.createElementNS(SVGNS, "g");
        this.rect(x0, by, Math.max(3, x1 - x0), bh, this.roleColor(d.role), "#8a90dd", 1, g);
        const lbl = this.roleRowLabel(d.role, 100, this.roleShare(d.est, 100));
        this.text(x0 + 6, by + bh / 2 + 3, lbl, this.labelColor(this.roleColor(d.role)), 10, g);
        this.svg.appendChild(g);
    }
    // drawDrillIcon paints a drill-down affordance (two stacked chevrons) at the
    // RIGHT edge of an action bar, for BOTH normal and summary actions. It is the
    // single-click target that drills into the detail plane; the cursor shows
    // "zoom-in" (a plus) on hover. On a thin summary (Sammelvorgang) the icon is
    // drawn SMALLER. Appended to the action's own group (`parent`) AFTER the bar
    // hit areas so it wins the click.
    drawDrillIcon(w, y, parent) {
        if (y == null)
            return;
        const pe = parseDate(w.end);
        if (!isFinite(pe))
            return;
        const small = !!w.summary; // summary bar is thin -> smaller icon
        const cx = this.x(pe) - (small ? 7 : 10);
        // summary graphic sits at the row bar-top; a normal bar is centered in the row
        const midY = small ? Math.round(y) + 4 : y + ROW_H / 2;
        const half = small ? 2 : 4; // chevron half-width
        const step = small ? 1.2 : 2; // chevron vertical amplitude
        const gap = small ? 1.5 : 3; // vertical gap between the two chevrons
        const sw = small ? "1.2" : "1.4";
        const chevron = (yy) => {
            const path = document.createElementNS(SVGNS, "path");
            path.setAttribute("d", `M ${cx - half} ${yy - step} L ${cx} ${yy + step} L ${cx + half} ${yy - step}`);
            path.setAttribute("fill", "none");
            path.setAttribute("stroke", "#fff");
            path.setAttribute("stroke-width", sw);
            path.setAttribute("stroke-linecap", "round");
            path.setAttribute("stroke-linejoin", "round");
            parent.appendChild(path);
        };
        chevron(midY - gap);
        chevron(midY + gap);
        // click target over the icon: drill in. cursor = zoom-in ("plus"). The
        // summary's thin icon has a DOUBLED hit area (hs 6 -> 12) so the small
        // glyph is easy to click.
        const hs = small ? 12 : 8;
        const hit = this.rect(cx - hs, midY - hs, hs * 2, hs * 2, "transparent", "", 0, parent);
        hit.style.cursor = "zoom-in";
        hit.classList.add("drilldown");
        hit.addEventListener("click", (e) => { e.stopPropagation(); this.cb.onDrillIcon(w.id); });
    }
    // draw a level-2 detail bar as an indented sub-row under its work package
    drawDetail(parent, d) {
        const y = this.detailRowOf[d.id];
        if (y == null)
            return;
        const ps = parseDate(d.start), pe = parseDate(d.end);
        if (!isFinite(ps) || !isFinite(pe))
            return;
        const x0 = this.x(ps), x1 = this.x(pe);
        const bh = BAR_H - 6;
        const by = y + (ROW_H - bh) / 2;
        const g = document.createElementNS(SVGNS, "g");
        const fill = d.role ? this.roleColor(d.role) : "#b9bdec";
        const bar = this.rect(x0, by, Math.max(3, x1 - x0), bh, fill, "#8a90dd", 1, g);
        bar.style.cursor = "pointer";
        const lbl = d.est && d.est !== "0" ? `${d.name}  ${d.est}` : d.name;
        this.placeLabel(g, x0, x1, by + bh / 2, bh, lbl, 10, d.textPos, this.labelColor(fill, d.fontColor), d.fontColor);
        // single click selects the detail and shows its editor in the right pane;
        // double click opens the parent action's drill plane.
        bar.addEventListener("click", (e) => { e.stopPropagation(); this.cb.onDetailSelect(parent.id, d.id); });
        bar.addEventListener("dblclick", () => this.cb.onOpen(parent.id));
        this.svg.appendChild(g);
    }
    // deterministic color per role name so detail bars are visually grouped
    // color for a role: use the role's assigned color from the project when set,
    // otherwise a deterministic hash fallback so bars are visually grouped.
    roleColor(name) {
        const r = this.project?.roles?.find((x) => x.name === name);
        if (r && r.color)
            return r.color;
        const palette = ["#d62728", "#2ca02c", "#1f77b4", "#ff7f0e", "#9467bd", "#8c564b", "#e377c2", "#17becf"];
        let h = 0;
        for (let i = 0; i < name.length; i++)
            h = (h * 31 + name.charCodeAt(i)) >>> 0;
        return palette[h % palette.length];
    }
    addAnchors(g, w, x0, x1, by) {
        const midY = by + BAR_H / 2;
        const pts = {
            l: [x0, midY], r: [x1, midY], t: [(x0 + x1) / 2, by], b: [(x0 + x1) / 2, by + BAR_H],
        };
        Object.keys(pts).forEach((a) => {
            const [cx, cy] = pts[a];
            const dot = document.createElementNS(SVGNS, "circle");
            dot.setAttribute("cx", String(cx));
            dot.setAttribute("cy", String(cy));
            dot.setAttribute("r", "4");
            dot.setAttribute("class", "anchor");
            dot.setAttribute("fill", "#534fbc");
            dot.addEventListener("pointerdown", (ev) => { ev.stopPropagation(); this.startAnchorLink(w.id, a, ev); });
            g.appendChild(dot);
        });
    }
    // isFullHeightMilestone reports whether a milestone renders as a full
    // top-to-bottom line (project Start/End, an unclustered milestone, or any
    // milestone in the top "Phases" row). Such milestones are draggable but NOT
    // link endpoints.
    isFullHeightMilestone(m) {
        if (m.id === "m_start" || m.id === "m_end")
            return true;
        if (!m.cluster)
            return true;
        return m.cluster.toLowerCase() === "phases";
    }
    // enableMilestoneDrag lets a milestone be dragged horizontally to a new date
    // (both the diamond and the full-height line). On release it fires
    // onMilestoneMove with the snapped date.
    enableMilestoneDrag(el, id, origDate) {
        el.addEventListener("pointerdown", (ev) => {
            if (this.linkMode)
                return;
            ev.preventDefault();
            ev.stopPropagation();
            const startX = ev.clientX;
            let moved = false;
            const compute = (clientX) => {
                const dDays = this.snapDays((clientX - startX) / this.ppd());
                return origDate + dDays * DAY;
            };
            const onMove = (e) => {
                if (Math.abs(e.clientX - startX) > 2)
                    moved = true;
                this.previewMilestone(compute(e.clientX));
                this.setDragMarks(compute(e.clientX));
                if (moved) {
                    const name = this.project?.milestones.find((m) => m.id === id)?.name ?? id;
                    this.cb.onDragStatus?.(milestoneDragStatusText(name, origDate, compute(e.clientX)));
                }
            };
            const onUp = (e) => {
                window.removeEventListener("pointermove", onMove);
                window.removeEventListener("pointerup", onUp);
                this.cb.onDragStatus?.(null);
                this.svg.querySelector("#msghost")?.remove();
                this.setDragMarks();
                if (moved)
                    this.cb.onMilestoneMove(id, fmtDate(compute(e.clientX)));
            };
            window.addEventListener("pointermove", onMove);
            window.addEventListener("pointerup", onUp);
        });
    }
    // previewMilestone draws a faint vertical guide at the dragged date.
    previewMilestone(ns) {
        let g = this.svg.querySelector("#msghost");
        const mx = this.x(ns);
        const h = this.svg.viewBox.baseVal.height;
        if (!g) {
            g = this.line(mx, TOP, mx, h - 30, "#534fbc", 2, "4 3");
            g.id = "msghost";
        }
        g.setAttribute("x1", String(mx));
        g.setAttribute("x2", String(mx));
    }
    // addMilestoneAnchors draws l/r anchor dots on a milestone diamond so a
    // dependency can be drawn FROM it by dragging (a milestone is a point, so
    // only the two horizontal anchors).
    addMilestoneAnchors(g, id, mx, cy) {
        ["l", "r"].forEach((a) => {
            const cx = a === "l" ? mx - 7 : mx + 7;
            const dot = document.createElementNS(SVGNS, "circle");
            dot.setAttribute("cx", String(cx));
            dot.setAttribute("cy", String(cy));
            dot.setAttribute("r", "4");
            dot.setAttribute("class", "anchor");
            dot.setAttribute("fill", "#534fbc");
            dot.addEventListener("pointerdown", (ev) => { ev.stopPropagation(); this.startMilestoneAnchorLink(id, a, ev); });
            g.appendChild(dot);
        });
    }
    // startMilestoneAnchorLink drags a red routing line from a milestone anchor
    // to a target action, then fires onLink (milestone -> action).
    startMilestoneAnchorLink(fromId, fromAnchor, ev) {
        ev.preventDefault();
        const line = this.line(0, 0, 0, 0, "#e03131", 2, "4 3");
        const rect = this.svg.getBoundingClientRect();
        const sp = this.svgPoint(ev.clientX, ev.clientY, rect);
        const onMove = (e) => {
            const pt = this.svgPoint(e.clientX, e.clientY, rect);
            line.setAttribute("x1", String(sp.x));
            line.setAttribute("y1", String(sp.y));
            line.setAttribute("x2", String(pt.x));
            line.setAttribute("y2", String(pt.y));
        };
        const onUp = (e) => {
            window.removeEventListener("pointermove", onMove);
            window.removeEventListener("pointerup", onUp);
            line.remove();
            const pt = this.svgPoint(e.clientX, e.clientY, rect);
            const target = this.actionAt(pt.y);
            if (target)
                this.cb.onLink(fromId, target.id, fromAnchor, this.nearestAnchor(target, pt));
        };
        window.addEventListener("pointermove", onMove);
        window.addEventListener("pointerup", onUp);
    }
    startAnchorLink(fromId, fromAnchor, ev) {
        ev.preventDefault();
        const line = this.line(0, 0, 0, 0, "#e03131", 2, "4 3"); // red routing line
        const rect = this.svg.getBoundingClientRect();
        const sp = this.svgPoint(ev.clientX, ev.clientY, rect);
        const onMove = (e) => {
            const pt = this.svgPoint(e.clientX, e.clientY, rect);
            line.setAttribute("x1", String(sp.x));
            line.setAttribute("y1", String(sp.y));
            line.setAttribute("x2", String(pt.x));
            line.setAttribute("y2", String(pt.y));
        };
        const onUp = (e) => {
            window.removeEventListener("pointermove", onMove);
            window.removeEventListener("pointerup", onUp);
            line.remove();
            const pt = this.svgPoint(e.clientX, e.clientY, rect);
            // drop target may be another action OR a (diamond) milestone
            const ms = this.milestoneAt(pt.x, pt.y);
            if (ms && ms !== fromId) {
                this.cb.onLink(fromId, ms, fromAnchor, "l");
                return;
            }
            const target = this.actionAt(pt.y);
            if (target && target.id !== fromId)
                this.cb.onLink(fromId, target.id, fromAnchor, this.nearestAnchor(target, pt));
        };
        window.addEventListener("pointermove", onMove);
        window.addEventListener("pointerup", onUp);
    }
    // milestoneAt returns the id of a linkable (diamond) milestone at an SVG
    // point, or null. Full-height (Phases/range) milestones are not linkable.
    milestoneAt(x, y) {
        for (const m of this.project?.milestones ?? []) {
            if (this.isFullHeightMilestone(m))
                continue;
            const lane = this.lanes.find((l) => l.cluster === m.cluster);
            if (!lane || y < lane.y || y >= lane.y + lane.h)
                continue;
            const mx = this.x(parseDate(m.date));
            if (Math.abs(x - mx) <= 10)
                return m.id;
        }
        return null;
    }
    actionAt(y) {
        for (const id in this.rowOf) {
            const top = this.rowOf[id];
            if (y >= top && y < top + ROW_H)
                return (this.project.packages).find((w) => w.id === id) || null;
        }
        return null;
    }
    laneAt(y) {
        for (const lane of this.lanes)
            if (y >= lane.y && y < lane.y + lane.h)
                return lane;
        return null;
    }
    // hitTest classifies an SVG point for the context menu:
    //   { kind:"action", id }                        -> over an action bar
    //   { kind:"milestone", id }                     -> over a milestone diamond
    //   { kind:"section-label", cluster }            -> over the left label column
    //   { kind:"empty", cluster }                    -> free space inside a lane
    //   { kind:"empty" }                             -> outside any lane
    hitTest(x, y) {
        // left label column (x < LEFT) -> the lane's section label area
        const lane = this.laneAt(y);
        if (x < LEFT) {
            if (lane)
                return { kind: "section-label", cluster: lane.cluster };
            return { kind: "empty" };
        }
        // over an action bar? (row + x within the bar span)
        const w = this.actionAt(y);
        if (w) {
            const x0 = this.x(parseDate(w.start)), x1 = this.x(parseDate(w.end));
            if (x >= x0 - 4 && x <= x1 + 4)
                return { kind: "action", id: w.id };
        }
        // over a user milestone diamond? (Phases-row / range milestones are
        // full-height lines, not diamonds, and are handled elsewhere.)
        for (const m of this.project?.milestones ?? []) {
            if (this.isFullHeightMilestone(m))
                continue;
            const ml = this.lanes.find((l) => l.cluster === m.cluster);
            if (!ml || y < ml.y || y >= ml.y + ml.h)
                continue;
            const mx = this.x(parseDate(m.date));
            if (Math.abs(x - mx) <= 9)
                return { kind: "milestone", id: m.id };
        }
        // free space inside a lane -> new action / milestone here (at the clicked date)
        const date = fmtDate(this.dateAt(x));
        if (lane)
            return { kind: "empty", cluster: lane.cluster, date };
        return { kind: "empty", date };
    }
    nearestAnchor(w, p) {
        const x0 = this.x(parseDate(w.start)), x1 = this.x(parseDate(w.end));
        const y = this.rowOf[w.id] + ROW_H / 2;
        const cand = {
            l: [x0, y], r: [x1, y], t: [(x0 + x1) / 2, y - BAR_H / 2], b: [(x0 + x1) / 2, y + BAR_H / 2],
        };
        let best = "l", bd = Infinity;
        Object.keys(cand).forEach((a) => { const [cx, cy] = cand[a]; const d = (cx - p.x) ** 2 + (cy - p.y) ** 2; if (d < bd) {
            bd = d;
            best = a;
        } });
        return best;
    }
    svgPoint(cx, cy, rect) {
        const vb = this.svg.viewBox.baseVal;
        return { x: (cx - rect.left) * (vb.width / rect.width), y: (cy - rect.top) * (vb.height / rect.height) };
    }
    handleLinkClick(id) {
        if (!this.linkFrom) {
            this.linkFrom = id;
            this.render();
            return;
        }
        if (this.linkFrom !== id)
            this.cb.onLink(this.linkFrom, id, "r", "l");
        this.linkFrom = null;
        this.setLinkMode(false);
    }
    drawDep(d) {
        const fa = (d.fromAnchor || defaultFromAnchor(d.kind));
        const ta = (d.toAnchor || defaultToAnchor(d.kind));
        // Endpoints may be work packages OR detail sub-actions. Detail endpoints
        // only have a drawn position when details are visible (Show Details on, or
        // in the drill plane); when an endpoint cannot be located we skip the arrow.
        const ap = this.taskAnchorPoint(d.from, fa), bp = this.taskAnchorPoint(d.to, ta);
        if (!ap || !bp)
            return;
        // A violated FS dependency is drawn RED to flag the timing breach.
        const color = d.violated ? "#e03131" : (d.color || "#555");
        const STUB = 12;
        const out = this.stub(fa, STUB), inn = this.stub(ta, STUB);
        const s1 = { x: ap.x + out.x, y: ap.y + out.y };
        const t1 = { x: bp.x + inn.x, y: bp.y + inn.y };
        const path = document.createElementNS(SVGNS, "path");
        if (this.arrowStyle() === "square") {
            // orthogonal route with the FEWEST corners: build candidate points then
            // drop any that are colinear with their neighbours.
            const raw = [ap, s1, { x: s1.x, y: t1.y }, t1, bp];
            path.setAttribute("d", orthoPath(dedupeColinear(raw)));
        }
        else {
            // spline (default): a single smooth cubic Bézier, zero visible corners.
            // control points pushed along the anchor normals so it leaves/enters
            // each bar cleanly.
            const c1 = { x: ap.x + out.x * 3, y: ap.y + out.y * 3 };
            const c2 = { x: bp.x + inn.x * 3, y: bp.y + inn.y * 3 };
            path.setAttribute("d", `M ${ap.x} ${ap.y} C ${c1.x} ${c1.y} ${c2.x} ${c2.y} ${bp.x} ${bp.y}`);
        }
        path.setAttribute("fill", "none");
        path.setAttribute("stroke", color);
        path.setAttribute("stroke-width", "1.5");
        this.svg.appendChild(path);
        const ah = document.createElementNS(SVGNS, "path");
        ah.setAttribute("d", this.arrowHead(bp, ta));
        ah.setAttribute("fill", color);
        this.svg.appendChild(ah);
    }
    arrowStyle() {
        return this.project?.arrowStyle === "square" ? "square" : "spline";
    }
    stub(a, d) {
        switch (a) {
            case "l": return { x: -d, y: 0 };
            case "r": return { x: d, y: 0 };
            case "t": return { x: 0, y: -d };
            case "b": return { x: 0, y: d };
        }
    }
    arrowHead(p, a) {
        switch (a) {
            case "l": return `M ${p.x} ${p.y} l -7 -4 l 0 8 z`;
            case "r": return `M ${p.x} ${p.y} l 7 -4 l 0 8 z`;
            case "t": return `M ${p.x} ${p.y} l -4 -7 l 8 0 z`;
            case "b": return `M ${p.x} ${p.y} l -4 7 l 8 0 z`;
        }
    }
    anchorPoint(w, a) {
        const x0 = this.x(parseDate(w.start)), x1 = this.x(parseDate(w.end));
        const midY = this.rowOf[w.id] + ROW_H / 2;
        if (![x0, x1, midY].every(isFinite))
            return null;
        switch (a) {
            case "l": return { x: x0, y: midY };
            case "r": return { x: x1, y: midY };
            case "t": return { x: (x0 + x1) / 2, y: midY - BAR_H / 2 };
            case "b": return { x: (x0 + x1) / 2, y: midY + BAR_H / 2 };
        }
    }
    // taskAnchorPoint resolves an anchor point for a dependency endpoint that may
    // be a work package OR a detail sub-action. Work packages use rowOf; details
    // use detailRowOf (only present when details are drawn, i.e. Show Details on
    // or in the drill plane). Returns null when the endpoint has no drawn row.
    taskAnchorPoint(id, a) {
        const wp = (this.project?.packages ?? []).find((w) => w.id === id);
        if (wp && this.rowOf[id] != null)
            return this.anchorPoint(wp, a);
        // milestone endpoint: a point at the diamond's center in its section lane
        const ms = (this.project?.milestones ?? []).find((m) => m.id === id);
        if (ms) {
            const mx = this.x(parseDate(ms.date));
            const lane = ms.cluster ? this.lanes.find((l) => l.cluster === ms.cluster) : null;
            const cy = lane ? lane.y + Math.min(lane.h - 10, ROW_H) / 2 + 4 : TOP + 10;
            if (!isFinite(mx))
                return null;
            return { x: mx, y: cy };
        }
        // detail endpoint
        const y = this.detailRowOf[id];
        if (y == null)
            return null;
        // find the detail's span
        let det;
        for (const w of this.project?.packages ?? []) {
            const dd = (w.details ?? []).find((x) => x.id === id);
            if (dd) {
                det = dd;
                break;
            }
        }
        if (!det)
            return null;
        const x0 = this.x(parseDate(det.start)), x1 = this.x(parseDate(det.end));
        const midY = y + ROW_H / 2;
        if (![x0, x1, midY].every(isFinite))
            return null;
        switch (a) {
            case "l": return { x: x0, y: midY };
            case "r": return { x: x1, y: midY };
            case "t": return { x: (x0 + x1) / 2, y: midY - (BAR_H - 6) / 2 };
            case "b": return { x: (x0 + x1) / 2, y: midY + (BAR_H - 6) / 2 };
        }
        return null;
    }
    enableDrag(el, w, mode) {
        el.addEventListener("pointerdown", (ev) => {
            ev.preventDefault();
            const startX = ev.clientX, startY = ev.clientY;
            const origStart = parseDate(w.start), origEnd = parseDate(w.end);
            const rect = this.svg.getBoundingClientRect();
            const compute = (clientX) => {
                const dDays = this.snapDays((clientX - startX) / this.ppd());
                let ns = origStart, ne = origEnd;
                if (mode === "move") {
                    ns = origStart + dDays * DAY;
                    ne = origEnd + dDays * DAY;
                }
                else if (mode === "l") {
                    ns = Math.min(origStart + dDays * DAY, origEnd - DAY);
                }
                else {
                    ne = Math.max(origEnd + dDays * DAY, origStart + DAY);
                }
                return [ns, ne];
            };
            const onMove = (e) => {
                const [ns, ne] = compute(e.clientX);
                this.previewBar(w.id, ns, ne);
                this.setDragMarks(ns, ne);
                // highlight the swimlane under the pointer so the user sees the target
                const pt = this.svgPoint(e.clientX, e.clientY, rect);
                const lane = this.laneAt(pt.y);
                this.highlightLane(lane);
                // the status line shows the state being moved to: the start first
                const curLane = this.laneOf[w.id];
                const otherLane = mode === "move" && lane && curLane && lane.cluster !== curLane.cluster
                    && Math.abs(e.clientY - startY) > ROW_H / 2;
                this.cb.onDragStatus?.(dragStatusText({
                    name: w.name, mode, origStart, origEnd, start: ns, end: ne,
                    intoSection: otherLane ? lane.cluster : undefined,
                }));
            };
            const onUp = (e) => {
                window.removeEventListener("pointermove", onMove);
                window.removeEventListener("pointerup", onUp);
                this.cb.onDragStatus?.(null);
                this.setDragMarks();
                this.highlightLane(null);
                const [ns, ne] = compute(e.clientX);
                // vertical drag in move mode: if dropped on a different swimlane, reassign
                if (mode === "move") {
                    const pt = this.svgPoint(e.clientX, e.clientY, rect);
                    const lane = this.laneAt(pt.y);
                    const curLane = this.laneOf[w.id];
                    const movedVert = Math.abs(e.clientY - startY) > ROW_H / 2;
                    if (lane && curLane && lane.cluster !== curLane.cluster && movedVert) {
                        this.cb.onReassign(w.id, lane.cluster);
                        return;
                    }
                }
                this.cb.onMove(w.id, fmtDate(ns), fmtDate(ne));
            };
            window.addEventListener("pointermove", onMove);
            window.addEventListener("pointerup", onUp);
        });
    }
    // setDragMarks shows, while an action or milestone is moved, one tick in the
    // date row for its start and one for its end (a milestone has one date: one
    // tick). Called with no dates it removes them. They are part of the sticky
    // date row, so they stay visible when the chart is scrolled down.
    setDragMarks(...dates) {
        this.head?.querySelector(".dragmarks")?.remove();
        if (!this.head || dates.length === 0)
            return;
        const g = document.createElementNS(SVGNS, "g");
        g.setAttribute("class", "dragmarks");
        for (const t of dates) {
            const x = this.x(t);
            const l = document.createElementNS(SVGNS, "line");
            l.setAttribute("x1", String(x));
            l.setAttribute("x2", String(x));
            l.setAttribute("y1", "6");
            l.setAttribute("y2", String(TOP - 4));
            l.setAttribute("stroke", "#534fbc");
            l.setAttribute("stroke-width", "2");
            g.appendChild(l);
        }
        this.head.appendChild(g);
    }
    highlightLane(lane) {
        let hl = this.svg.querySelector("#lanehl");
        if (!lane) {
            hl?.remove();
            return;
        }
        const width = this.svg.viewBox.baseVal.width;
        if (!hl) {
            hl = this.rect(0, 0, 0, 0, "rgba(83,79,188,.12)", "#534fbc", 1);
            hl.id = "lanehl";
            hl.setAttribute("stroke-dasharray", "4 3");
        }
        hl.setAttribute("x", "0");
        hl.setAttribute("y", String(lane.y));
        hl.setAttribute("width", String(width));
        hl.setAttribute("height", String(lane.h));
    }
    previewBar(id, ns, ne) {
        const y = this.rowOf[id] + (ROW_H - BAR_H) / 2;
        let ghost = this.svg.querySelector("#ghost");
        if (!ghost) {
            ghost = this.rect(0, 0, 0, 0, "rgba(83,79,188,.35)", "#534fbc", 1);
            ghost.id = "ghost";
        }
        ghost.setAttribute("x", String(this.x(ns)));
        ghost.setAttribute("y", String(y));
        ghost.setAttribute("width", String(Math.max(4, this.x(ne) - this.x(ns))));
        ghost.setAttribute("height", String(BAR_H));
    }
    // drawConfigGrid draws vertical gridlines at the configured interval
    // (day/week/month) across the chart. Behind the bars (called from drawAxis).
    drawConfigGrid(lo, hi, height, grid) {
        if (grid === "month") {
            const s = new Date(lo);
            let yy = s.getUTCFullYear(), mo = s.getUTCMonth();
            for (let guard = 0; guard < 400; guard++) {
                const t = Date.UTC(yy, mo, 1);
                if (t > hi)
                    break;
                if (t >= lo)
                    this.line(this.x(t), TOP - 8, this.x(t), height - 30, "#e6e6ef", 1);
                mo++;
                if (mo > 11) {
                    mo = 0;
                    yy++;
                }
            }
            return;
        }
        const step = grid === "day" ? DAY : 7 * DAY;
        const s0 = new Date(lo);
        let t = Date.UTC(s0.getUTCFullYear(), s0.getUTCMonth(), s0.getUTCDate());
        if (grid === "week") {
            while (new Date(t).getUTCDay() !== 1)
                t -= DAY;
        }
        const dayW = this.ppd();
        for (let guard = 0; guard < 4000 && t <= hi; guard++, t += step) {
            if (t < lo)
                continue;
            if (grid === "day" && dayW < 4)
                break; // too dense to be useful
            this.line(this.x(t), TOP - 8, this.x(t), height - 30, "#eef0f4", 1);
        }
    }
    // drawTodayLine draws a vertical marker at today's date (report/settings
    // option, default on). Drawn on top so it is visible over the bars.
    drawTodayLine(height) {
        if (this.project && this.project.reportTodayLine === false)
            return;
        const now = new Date();
        const t = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate());
        const mx = this.x(t);
        if (!isFinite(mx))
            return;
        this.line(mx, TOP - 8, mx, height - 30, "#e03131", 1.5, "2 2");
        this.text(mx + 3, TOP - 2, "today", "#e03131", 9);
    }
    drawAxis(lo, hi, height) {
        const z = this.zoom;
        // Configurable gridlines (report/settings): draw vertical lines at the
        // chosen interval (day / week / month) across the whole chart, independent
        // of the zoom label density, so the live canvas and the export share the
        // same grid. Default "month".
        const grid = this.project?.reportGrid || "month";
        this.drawConfigGrid(lo, hi, height, grid);
        const head = document.createElementNS(SVGNS, "g");
        head.setAttribute("class", "stickyhead");
        const bg = document.createElementNS(SVGNS, "rect");
        bg.setAttribute("class", "stickybg");
        bg.setAttribute("x", "0");
        bg.setAttribute("y", "0");
        bg.setAttribute("width", this.svg.getAttribute("width") || "100000");
        bg.setAttribute("height", String(TOP - 4));
        bg.setAttribute("fill", "#ffffff");
        bg.setAttribute("visibility", "hidden");
        head.appendChild(bg);
        const rule = document.createElementNS(SVGNS, "line");
        rule.setAttribute("class", "stickybg");
        rule.setAttribute("x1", "0");
        rule.setAttribute("x2", this.svg.getAttribute("width") || "100000");
        rule.setAttribute("y1", String(TOP - 4));
        rule.setAttribute("y2", String(TOP - 4));
        rule.setAttribute("stroke", "#c4c4d4");
        rule.setAttribute("visibility", "hidden");
        head.appendChild(rule);
        this.head = head;
        // Weekend shading: Saturdays and Sundays get a light band so the
        // non-working days are visually distinct. Only drawn when a day is wide
        // enough to be visible (fine zoom or a fitted window), to avoid clutter at
        // coarse zoom. Weekends also carry ZERO work in the capacity calculation
        // (WorkdaysBetween excludes them).
        const dayW = this.ppd();
        if (dayW >= 3) {
            const s0 = new Date(lo);
            let t = Date.UTC(s0.getUTCFullYear(), s0.getUTCMonth(), s0.getUTCDate());
            for (let guard = 0; guard < 2000 && t <= hi; guard++, t += DAY) {
                if (t < lo)
                    continue;
                const dow = new Date(t).getUTCDay();
                if (dow === 0 || dow === 6) { // Sunday or Saturday
                    const bx = this.x(t);
                    const band = this.rect(bx, TOP - 4, Math.max(1, dayW), height - 30 - (TOP - 4), "#eceef4", "", 0);
                    band.setAttribute("rx", "0");
                    band.setAttribute("opacity", "0.7");
                    band.classList.add("weekend");
                }
            }
        }
        if (z === "days" || z === "weeks") {
            const stepDays = z === "days" ? 1 : 7;
            const s = new Date(lo);
            let t = Date.UTC(s.getUTCFullYear(), s.getUTCMonth(), s.getUTCDate());
            for (let guard = 0; guard < 2000 && t <= hi; guard++, t += stepDays * DAY) {
                if (t < lo)
                    continue;
                const mx = this.x(t), d = new Date(t), isWeek = d.getUTCDay() === 1;
                this.line(mx, TOP - 4, mx, height - 30, z === "days" && !isWeek ? "#eef0f4" : "#dcdce6", 1);
                if (z === "days")
                    this.text(mx + 2, TOP - 16, String(d.getUTCDate()), "#999", 9, head);
                else if (isWeek)
                    this.text(mx + 2, TOP - 16, `${d.getUTCDate()}.${d.getUTCMonth() + 1}`, "#999", 9, head);
            }
        }
        const start = new Date(lo);
        let yy = start.getUTCFullYear(), mo = start.getUTCMonth();
        for (let guard = 0; guard < 400; guard++) {
            const t = Date.UTC(yy, mo, 1);
            if (t > hi)
                break;
            if (t >= lo) {
                const mx = this.x(t);
                this.line(mx, TOP - 8, mx, height - 30, "#c4c4d4", 1);
                this.text(mx + 2, TOP - 30, `${MONTHS[mo]} ${String(yy).slice(2)}`, "#555", 11, head);
            }
            mo++;
            if (mo > 11) {
                mo = 0;
                yy++;
            }
        }
        this.drawWeekNumbers(lo, hi, head);
    }
    // drawWeekNumbers writes the ISO calendar week at every Monday into the date
    // row (project setting "show week numbers"), but only when a week is wide
    // enough for the label.
    drawWeekNumbers(lo, hi, head) {
        if (!this.project?.showWeekNumbers)
            return;
        const prefix = this.project.language === "de" ? "KW" : "W";
        const weekW = 7 * this.ppd();
        if (weekW < (prefix.length + 2) * 6 + 8)
            return; // no room: skip
        let t = Date.UTC(new Date(lo).getUTCFullYear(), new Date(lo).getUTCMonth(), new Date(lo).getUTCDate());
        while (new Date(t).getUTCDay() !== 1)
            t += DAY;
        for (let guard = 0; guard < 2000 && t <= hi; guard++, t += 7 * DAY) {
            this.text(this.x(t) + 2, TOP - 6, `${prefix}${isoWeek(t)}`, "#7a7ab8", 9, head);
        }
    }
    rect(x, y, w, h, fill, stroke, sw, parent) {
        const r = document.createElementNS(SVGNS, "rect");
        r.setAttribute("x", String(x));
        r.setAttribute("y", String(y));
        r.setAttribute("width", String(w));
        r.setAttribute("height", String(h));
        r.setAttribute("rx", "3");
        r.setAttribute("fill", fill);
        if (stroke) {
            r.setAttribute("stroke", stroke);
            r.setAttribute("stroke-width", String(sw));
        }
        (parent ?? this.svg).appendChild(r);
        return r;
    }
    line(x1, y1, x2, y2, stroke, sw, dash) {
        const l = document.createElementNS(SVGNS, "line");
        l.setAttribute("x1", String(x1));
        l.setAttribute("y1", String(y1));
        l.setAttribute("x2", String(x2));
        l.setAttribute("y2", String(y2));
        l.setAttribute("stroke", stroke);
        l.setAttribute("stroke-width", String(sw));
        if (dash)
            l.setAttribute("stroke-dasharray", dash);
        this.svg.appendChild(l);
        return l;
    }
    text(x, y, s, fill, size, parent) {
        const t = document.createElementNS(SVGNS, "text");
        t.setAttribute("x", String(x));
        t.setAttribute("y", String(y));
        t.setAttribute("fill", fill);
        t.setAttribute("font-size", String(size));
        t.setAttribute("font-family", "system-ui, sans-serif");
        t.textContent = s;
        (parent ?? this.svg).appendChild(t);
        return t;
    }
    // labelColor resolves the readable text color for a bar: if the user set an
    // explicit font color it wins; otherwise it is auto-contrast — black on a
    // light background, white on a dark one — so a label stays readable no matter
    // how dark the bar color is (fixes white-on-dark in the detail view).
    labelColor(bg, explicit) {
        if (explicit)
            return explicit;
        return contrastOn(bg || "#8a90dd");
    }
    // placeLabel draws a bar's label honouring its text position. `pos` is "" /
    // "auto" | center | left | right | up | down.
    //   auto   -> inside (left-aligned) if the label fits within the bar width,
    //             otherwise just to the RIGHT of the bar.
    //   center -> centered horizontally inside the bar.
    //   left   -> inside, flush to the left edge.
    //   right  -> just to the right of the bar's right edge (outside).
    //   up     -> above the bar, left-aligned to its start.
    //   down   -> below the bar, left-aligned to its start.
    // insideColor is used when the label sits ON the bar (auto-contrast); a label
    // drawn OUTSIDE the bar uses a neutral dark color so it stays readable on the
    // canvas background. `midY` is the bar's vertical center; `h` its height.
    placeLabel(g, x0, x1, midY, h, text, size, pos, insideColor, explicit) {
        const barW = x1 - x0;
        const estW = this.measureText(text, size);
        const outsideColor = explicit || "#333";
        const place = (pos || "auto") === "auto"
            ? (estW + 10 <= barW ? "left" : "right")
            : pos;
        let tx = x0 + 6, ty = midY + size / 2 - 1, anchor = "start", fill = insideColor;
        switch (place) {
            case "center":
                tx = (x0 + x1) / 2;
                anchor = "middle";
                break;
            case "left":
                tx = x0 + 6;
                anchor = "start";
                break;
            // outleft: the label sits OUTSIDE to the left of the bar and ENDS at the
            // bar's left edge (right-aligned, a small gap before the edge).
            case "outleft":
                tx = x0 - 8;
                anchor = "end";
                fill = outsideColor;
                break;
            // right / outright: the label sits OUTSIDE just past the bar's right edge
            // (left-aligned). "right" is kept as a legacy alias of "outright".
            case "right":
            case "outright":
                tx = x1 + 8;
                anchor = "start";
                fill = outsideColor;
                break;
            case "up":
                tx = x0;
                ty = midY - h / 2 - 4;
                anchor = "start";
                fill = outsideColor;
                break;
            case "down":
                tx = x0;
                ty = midY + h / 2 + size;
                anchor = "start";
                fill = outsideColor;
                break;
        }
        const t = this.text(tx, ty, text, fill, size, g);
        if (anchor !== "start")
            t.setAttribute("text-anchor", anchor);
    }
    // placeSummaryLabel writes the label of a summary bar (Sammelvorgang) at the
    // chosen text position and returns the horizontal extent [x0, x1] of bar plus
    // label, for the hit area. The rollup bar is only a few pixels high, so there
    // is no "inside": left and center put the label ABOVE the bar (left-aligned /
    // centered), like "up". Auto (and the legacy "right") keeps it to the right.
    placeSummaryLabel(g, xL, xR, sy, barH, capH, text, pos, explicit) {
        const size = 11, color = explicit || "#333";
        const tw = 8 + text.length * 7 + 6; // generous width estimate for the hit area
        let tx = xR + 8, ty = sy + barH, anchor = "start";
        switch (pos || "auto") {
            case "outleft":
                tx = xL - 8;
                anchor = "end";
                break;
            case "up":
            case "left":
                tx = xL;
                ty = sy - 4;
                break;
            case "center":
                tx = (xL + xR) / 2;
                ty = sy - 4;
                anchor = "middle";
                break;
            case "down":
                tx = xL;
                ty = sy + barH + capH + size;
                break;
        }
        const t = this.text(tx, ty, text, color, size, g);
        if (anchor !== "start")
            t.setAttribute("text-anchor", anchor);
        let lx0 = tx, lx1 = tx + tw;
        if (anchor === "end") {
            lx0 = tx - tw;
            lx1 = tx;
        }
        if (anchor === "middle") {
            lx0 = tx - tw / 2;
            lx1 = tx + tw / 2;
        }
        return [Math.min(xL - 4, lx0), Math.max(xR + 4, lx1)];
    }
    // measureText estimates a label's rendered width in px. SVG getComputedText-
    // Length would be exact but requires the node to be laid out; this average-
    // glyph estimate (0.58em) is good enough to decide inside-vs-right.
    measureText(s, size) {
        return s.length * size * 0.58;
    }
}
// contrastOn returns "#111" (near-black) or "#fff" depending on which has the
// better contrast against the given hex background, using the W3C relative
// luminance of the background. Accepts #rgb or #rrggbb; falls back to white for
// anything it cannot parse.
function contrastOn(hex) {
    const rgb = parseHex(hex);
    if (!rgb)
        return "#fff";
    const lin = (c) => {
        const s = c / 255;
        return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
    };
    const L = 0.2126 * lin(rgb[0]) + 0.7152 * lin(rgb[1]) + 0.0722 * lin(rgb[2]);
    // contrast vs white = 1.05/(L+0.05); vs black = (L+0.05)/0.05.
    // pick whichever is larger -> threshold at L ~ 0.179.
    return L > 0.179 ? "#111" : "#fff";
}
// parseHex parses a #rgb or #rrggbb color to [r,g,b] 0..255, or null.
function parseHex(hex) {
    const m = /^#?([0-9a-f]{3}|[0-9a-f]{6})$/i.exec(hex.trim());
    if (!m)
        return null;
    let h = m[1];
    if (h.length === 3)
        h = h[0] + h[0] + h[1] + h[1] + h[2] + h[2];
    return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)];
}
