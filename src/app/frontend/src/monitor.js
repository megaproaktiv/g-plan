// The Monitor page: switch monitoring of real effort on, pick the external
// system and project, configure the system, run a synchronization and show the
// per-section result table.
import { Events } from "@wailsio/runtime";
import { backend } from "./backend";
import { errorText } from "./errortext";
import { effortHTML } from "./effort";
const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
/**
 * sectionStates lists the project's sections (work packages) in display order
 * with how many of their actions carry an External ID. A section is "green"
 * (tracked > 0) when at least one action has one.
 */
export function sectionStates(p) {
    const nums = p.sectionNums ?? {};
    const names = Object.keys(nums).sort((a, b) => nums[a] - nums[b]);
    for (const w of p.packages)
        if (!names.includes(w.cluster))
            names.push(w.cluster);
    // the Phases lane is a view, not a work package: it is ignored by monitoring
    return names.filter((name) => name.trim().toLowerCase() !== "phases").map((name) => {
        const acts = p.packages.filter((w) => w.cluster === name);
        const n = nums[name];
        const base = name === "" ? "(no section)" : name;
        return {
            name, label: n ? `${n} ${base}` : base,
            tracked: acts.filter((w) => !!w.externalId && w.contour !== "None").length, actions: acts.length,
            hasDefault: !!p.monitor?.defaultAction && acts.some((w) => w.id === p.monitor.defaultAction),
        };
    });
}
const bookings = (n) => `${n} booking${n === 1 ? "" : "s"}`;
/**
 * actualEffortHTML renders the "Actual effort" block of the small action view
 * from the last synchronization: the total in hours and days, split into the
 * bookings assigned through the action's External ID and the unassigned
 * bookings credited to it as the default action. Empty when monitoring
 * is off.
 */
export function actualEffortHTML(p, wp, s) {
    if (!p.monitor?.enabled)
        return "";
    if (wp.contour === "None")
        return `<span class="muted">Actual effort: none (contour None: no estimation, no real effort)</span>`;
    if (wp.cluster.trim().toLowerCase() === "phases")
        return ""; // Phases lane: a view, not monitored
    if (!s)
        return `<span class="muted">Actual effort: not synchronized yet (Monitor, then Synchronize)</span>`;
    let a;
    for (const sec of s.sections)
        a = a ?? sec.details.find((d) => d.id === wp.id);
    if (!a)
        return `<span class="muted">Actual effort: not tracked (no External ID, not the default action)</span>`;
    const part = (what, h, d, n) => `<li>${what}: ${effortHTML(p.unit, h, d)} · ${bookings(n)}</li>`;
    const lines = [];
    if (a.externalId)
        lines.push(part(`assigned to "${esc(a.externalId)}"`, a.matchedHours, a.matchedDays, a.matchedEntries));
    if (a.isDefault)
        lines.push(part("unassigned (default action)", a.defaultHours, a.defaultDays, a.defaultEntries));
    return `<div>Actual effort: ${effortHTML(p.unit, a.actualHours, a.actualDays)} · ${bookings(a.entries)}
    <span class="muted">synchronized ${esc(s.at)}</span></div><ul class="mon-parts">${lines.join("")}</ul>`;
}
/**
 * externalIdState decides whether an action's External ID can be edited and what
 * to tell the user. The External ID can be edited at any time, also while
 * monitoring is on (the next synchronization uses it). An action with the
 * contour None is not a real action and cannot carry one: the field is locked
 * and says why.
 */
export function externalIdState(p, wp) {
    if (wp.contour === "None") {
        return {
            readOnly: true,
            noteHTML: `This action has the contour <b>None</b>: it is not a real action and cannot carry an External ID. Choose another contour to give it one.`,
        };
    }
    if (p.monitor?.enabled) {
        return {
            readOnly: false,
            noteHTML: `Monitoring is on: after changing the External ID, synchronize again (the last result is cleared and the real effort follows at the next synchronization).`,
        };
    }
    return { readOnly: false, noteHTML: "" };
}
/** extIdCell is the External ID of an action row; the default action is marked as such. */
function extIdCell(d) {
    const id = d.externalId ? `<code>${esc(d.externalId)}</code>` : "";
    const dflt = d.isDefault ? `<span class="muted">(default action)</span>` : "";
    return [id, dflt].filter(Boolean).join(" ") || `<span class="muted">-</span>`;
}
/** resultTableHTML renders the synchronization result: sections plus project sum. */
export function resultTableHTML(p, s) {
    const label = new Map(sectionStates(p).map((x) => [x.name, x.label]));
    let tracked = 0, actions = 0;
    const rows = s.sections.map((sec) => {
        tracked += sec.tracked;
        actions += sec.actions;
        const head = `<tr class="mon-sec"><td>${esc(label.get(sec.name) ?? sec.name)}</td>
      <td class="num">${sec.tracked} / ${sec.actions}</td><td class="num">${esc(sec.estTracked)}</td>
      <td class="num">${esc(sec.est)}</td><td class="num">${esc(sec.actual)}</td><td class="num">${esc(sec.remaining)}</td>
      <td class="num">${esc(sec.remainingPct)}</td><td class="num">${sec.entries}</td></tr>`;
        const details = sec.details.map((d) => {
            const row = `<tr class="mon-act${d.isDefault ? " mon-defact" : ""}"><td>${esc(d.name)}</td>
        <td class="mon-extid">${extIdCell(d)}</td><td class="num">${esc(d.est)}</td><td></td><td class="num">${esc(d.actual)}</td>
        <td class="num">${esc(d.remaining)}</td><td class="num">${esc(d.remainingPct)}</td><td class="num">${d.entries}</td></tr>`;
            // the unassigned bookings credited to the default action, shown apart
            const dflt = d.isDefault && d.defaultEntries > 0
                ? `<tr class="mon-act mon-default"><td>of which unassigned bookings (default action)</td>
            <td></td><td></td><td></td><td class="num">${esc(d.defaultActual)}</td><td></td><td></td><td class="num">${d.defaultEntries}</td></tr>` : "";
            return row + dflt;
        }).join("");
        return head + details;
    }).join("");
    return `<table class="mon-table"><thead><tr><th>Section / Action</th><th>Tracked actions / External ID</th>
      <th>Estimated (tracked)</th><th>Estimated (all)</th><th>Actual</th><th>Remaining</th><th>Remaining %</th><th>Entries</th></tr></thead>
    <tbody>${rows}
      <tr class="mon-total"><td>Project</td><td class="num">${tracked} / ${actions}</td><td class="num">${esc(s.estTracked)}</td>
        <td class="num">${esc(s.est)}</td><td class="num">${esc(s.actual)}</td><td class="num">${esc(s.remaining)}</td><td class="num">${esc(s.remainingPct)}</td><td class="num">${s.entries}</td></tr>
      ${s.defaultAction && s.unmatchedEntries === 0 ? "" : `<tr class="mon-unmatched"><td>Bookings without a matching External ID</td><td></td><td></td><td></td>
        <td class="num">${esc(s.unmatchedActual)}</td><td></td><td></td><td class="num">${s.unmatchedEntries}</td></tr>`}
    </tbody></table>
    <p class="muted">${s.fetched} bookings fetched for ${esc(s.from)} to ${esc(s.to)} (synchronized ${esc(s.at)}). Values in ${esc(p.unit)}.</p>`;
}
/** defaultOptions lists the actions (grouped by section) that can be the default action. */
function defaultOptions(p, current) {
    const isPhases = (c) => c.trim().toLowerCase() === "phases"; // the Phases lane is a view
    const groups = sectionStates(p).filter((s) => !isPhases(s.name)).map((s) => {
        const acts = p.packages.filter((w) => w.cluster === s.name && w.contour !== "None"); // None takes no bookings
        if (acts.length === 0)
            return "";
        return `<optgroup label="${esc(s.label)}">${acts.map((w) => `<option value="${esc(w.id)}"${current === w.id ? " selected" : ""}>${esc(w.displayId)} ${esc(w.name)}</option>`).join("")}</optgroup>`;
    }).join("");
    // a stored default that no longer exists is shown instead of silently looking like "none"
    const missing = current && !p.packages.some((w) => w.id === current)
        ? `<option value="${esc(current)}" selected>(action no longer exists)</option>` : "";
    return groups + missing;
}
function ensureBox(id, extra = "") {
    let box = document.querySelector("#" + id);
    if (!box) {
        box = document.createElement("div");
        box.id = id;
        box.className = "drill hidden " + extra;
        document.body.appendChild(box);
    }
    return box;
}
/** openMonitor shows the Monitor page. */
export async function openMonitor(host) {
    const box = ensureBox("monitorbox");
    box.classList.remove("hidden");
    const systems = await backend.monitorSystems();
    let summary = await backend.getMonitorSummary();
    let msg = null;
    let running = false;
    let testing = false; // a connection test is in flight
    let off = null;
    const proj = () => host.getProject();
    const close = () => { off?.(); off = null; box.classList.add("hidden"); };
    const say = (text, bad) => { msg = { text, bad }; draw(); };
    function draw() {
        const p = proj();
        const m = p.monitor;
        const secs = sectionStates(p);
        box.innerHTML = `
      <header><strong>Monitor</strong><span class="muted">compare the plan with real effort booked in an external system</span>
        <span style="flex:1"></span><button id="mon_close">Close</button></header>
      <div class="body">
        <fieldset class="mon-set"><legend>External system</legend>
          <p><label><input type="checkbox" id="mon_on" ${m.enabled ? "checked" : ""}/> <b>Monitoring on</b></label>
            <span class="muted">requires a system and a project ID</span></p>
          <p><label>System
            <select id="mon_sys">
              <option value=""${m.system ? "" : " selected"}>- choose -</option>
              ${systems.map((s) => `<option value="${esc(s.id)}"${s.available ? "" : " disabled"}${m.system === s.id ? " selected" : ""}>${esc(s.label)}</option>`).join("")}
            </select></label>
            <button id="mon_cfg"${m.system ? "" : " disabled"}>Configure System…</button></p>
          <p><label>Project ID in the system <input id="mon_pid" size="14" value="${esc(m.projectId)}" /></label>
            <button id="mon_test"${m.system && !testing ? "" : " disabled"}>Test</button></p>
          <p id="mon_msg" class="${testing ? "mon-busy" : msg ? (msg.bad ? "mon-bad" : "mon-ok") : ""}" role="status" aria-live="polite">${testing ? "Testing connection…" : msg ? esc(msg.text) : ""}</p>
          <p><label>Default action for unassigned bookings
            <select id="mon_def">
              <option value=""${m.defaultAction ? "" : " selected"}>- none: show them separately -</option>
              ${defaultOptions(p, m.defaultAction)}
            </select></label>
            <span class="muted">an action; bookings without a matching External ID are counted on it</span></p>
        </fieldset>
        <h3>Work packages (sections)</h3>
        <ul class="mon-secs">${secs.map((s) => `<li class="${s.tracked > 0 ? "mon-green" : "mon-grey"}"><span class="mon-dot" aria-hidden="true"></span>
            ${esc(s.label)} <span class="muted">${s.tracked > 0 ? `${s.tracked} of ${s.actions} actions have an External ID` : "no External ID"}${s.hasDefault ? " · contains the default action" : ""}</span></li>`).join("")
            || `<li class="muted">no sections yet</li>`}</ul>
        <fieldset class="mon-set mon-range"><legend>Period to synchronize</legend>
          <label><input type="checkbox" id="mon_before" ${m.includeBefore ? "checked" : ""}${running ? " disabled" : ""}/>
            also one month <b>before</b> the project start</label><br/>
          <label><input type="checkbox" id="mon_after" ${m.includeAfter ? "checked" : ""}${running ? " disabled" : ""}/>
            also one month <b>after</b> the project end</label>
          <div id="mon_period" class="muted">${m.syncFrom && m.syncTo ? `Bookings from ${esc(m.syncFrom)} to ${esc(m.syncTo)} are fetched.` : "The project needs a start and an end date."}</div>
        </fieldset>
        <p><button id="mon_sync" class="primary"${m.enabled && !running ? "" : " disabled"}>Synchronize</button>
          <button id="mon_cancel"${running ? "" : " disabled"}>Cancel</button>
          <button id="mon_export"${summary ? "" : " disabled"} title="Write the result as markdown: a compact file (work packages) and a detailed file (with actions)">Export</button>
          <progress id="mon_prog" max="100"${running ? "" : " hidden"}></progress>
          <span id="mon_cnt" class="muted" role="status"></span></p>
        <div id="mon_result">${summary ? resultTableHTML(p, summary) : `<p class="muted">No synchronization yet.</p>`}</div>
      </div>`;
        wire();
    }
    async function commit(enabled) {
        const sys = box.querySelector("#mon_sys").value;
        const pid = box.querySelector("#mon_pid").value;
        try {
            host.setProject(await backend.setMonitor(enabled, sys, pid));
            summary = await backend.getMonitorSummary();
            msg = null;
            host.refresh();
        }
        catch (e) {
            msg = { text: String(e), bad: true };
        }
        draw();
    }
    function onProgress(ev) {
        const d = (Array.isArray(ev.data) ? ev.data[0] : ev.data);
        const prog = box.querySelector("#mon_prog");
        const cnt = box.querySelector("#mon_cnt");
        if (!d || !prog || !cnt)
            return;
        if (d.total > 0) {
            prog.max = d.total;
            prog.value = d.fetched;
            cnt.textContent = `${d.fetched} of ${d.total} entries fetched`;
        }
        else {
            prog.removeAttribute("value");
            cnt.textContent = "connecting…";
        }
    }
    async function sync() {
        running = true;
        msg = null;
        draw();
        off = Events.On("monitor:progress", onProgress);
        try {
            summary = await backend.syncMonitor();
            msg = { text: `Synchronized: ${summary.fetched} entries fetched.`, bad: false };
            host.refresh(); // an open action view now shows its actual effort
        }
        catch (e) {
            msg = { text: String(e), bad: true };
        }
        finally {
            running = false;
            off?.();
            off = null;
        }
        draw();
    }
    function wire() {
        box.querySelector("#mon_close").addEventListener("click", close);
        box.querySelector("#mon_on").addEventListener("change", (e) => commit(e.target.checked));
        box.querySelector("#mon_sys").addEventListener("change", () => commit(proj().monitor.enabled));
        box.querySelector("#mon_pid").addEventListener("change", () => commit(proj().monitor.enabled));
        const onRange = async () => {
            const before = box.querySelector("#mon_before").checked;
            const after = box.querySelector("#mon_after").checked;
            try {
                host.setProject(await backend.setMonitorRange(before, after));
                summary = await backend.getMonitorSummary();
                msg = null;
                host.refresh();
            }
            catch (err) {
                msg = { text: String(err), bad: true };
            }
            draw();
        };
        box.querySelector("#mon_before").addEventListener("change", onRange);
        box.querySelector("#mon_after").addEventListener("change", onRange);
        box.querySelector("#mon_def").addEventListener("change", async (e) => {
            try {
                host.setProject(await backend.setMonitorDefaultAction(e.target.value));
                summary = await backend.getMonitorSummary();
                msg = null;
                host.refresh();
            }
            catch (err) {
                msg = { text: String(err), bad: true };
            }
            draw();
        });
        box.querySelector("#mon_cfg").addEventListener("click", () => openSystemConfig(proj().monitor.system, say));
        box.querySelector("#mon_sync").addEventListener("click", sync);
        box.querySelector("#mon_export").addEventListener("click", async () => {
            try {
                const files = await backend.exportMonitorMarkdown();
                if (files && files.length)
                    say(`Exported: ${files.join(", ")}`, false);
            }
            catch (err) {
                say(errorText(err), true);
            }
        });
        box.querySelector("#mon_cancel").addEventListener("click", () => { void backend.cancelSync(); });
        box.querySelector("#mon_test").addEventListener("click", async () => {
            if (testing)
                return; // one test at a time
            const sys = box.querySelector("#mon_sys").value;
            const pid = box.querySelector("#mon_pid").value;
            testing = true;
            msg = null;
            draw(); // button disabled + "Testing connection…" line
            try {
                const r = await backend.testSystemConnection(sys, pid);
                msg = { text: `Connected: project "${r.projectName}"${r.start ? ` (${r.start} to ${r.end || "open"})` : ""}.`, bad: false };
            }
            catch (e) {
                msg = { text: String(e), bad: true };
            }
            finally {
                testing = false;
            }
            draw();
        });
    }
    draw();
}
/** openSystemConfig shows the "Configure System" dialog (Aerion settings). */
export async function openSystemConfig(system, say) {
    const box = ensureBox("sysbox", "sysdlg");
    box.classList.remove("hidden");
    let cfg;
    try {
        cfg = await backend.getSystemConfig(system);
    }
    catch (e) {
        box.classList.add("hidden");
        say(String(e), true);
        return;
    }
    box.innerHTML = `
    <header><strong>Configure System: Aerion</strong><span style="flex:1"></span></header>
    <div class="body">
      <p class="muted">Stored for your user only in <code>~/.pt-pm/config.yaml</code> (not in the project file).</p>
      <p><label>Base URL <input id="sc_url" size="40" value="${esc(cfg.baseUrl)}" placeholder="https://your-company.aerion.cloud" /></label></p>
      <p><label>Username <input id="sc_user" size="30" value="${esc(cfg.username)}" autocomplete="off" /></label></p>
      <p><label>Password <input id="sc_pw" type="password" size="30" autocomplete="new-password"
          placeholder="${cfg.passwordSet ? "stored - leave empty to keep" : ""}" /></label></p>
      <p><label>Client ID <input id="sc_cid" size="30" value="${esc(cfg.clientId)}" autocomplete="off" /></label></p>
      <p><label>Client secret <input id="sc_cs" type="password" size="30" autocomplete="new-password"
          placeholder="${cfg.clientSecretSet ? "stored - leave empty to keep" : ""}" /></label></p>
      <p id="sc_msg" class="mon-bad" role="alert"></p>
      <p><button id="sc_save" class="primary">Save</button> <button id="sc_cancel">Cancel</button></p>
    </div>`;
    const v = (id) => box.querySelector("#" + id).value;
    box.querySelector("#sc_cancel").addEventListener("click", () => box.classList.add("hidden"));
    box.querySelector("#sc_save").addEventListener("click", async () => {
        try {
            await backend.saveSystemConfig(system, v("sc_url"), v("sc_user"), v("sc_pw"), v("sc_cid"), v("sc_cs"));
            box.classList.add("hidden");
            say("System configuration saved.", false);
        }
        catch (e) {
            box.querySelector("#sc_msg").textContent = String(e);
        }
    });
}
