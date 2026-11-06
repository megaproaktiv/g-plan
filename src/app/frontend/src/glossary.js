const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
/** termHTML renders one glossary entry. */
export function termHTML(t, tr) {
    return `<div class="gl-term" id="gl_${esc(t.key)}">
    <dt>${esc(t.term)}${t.alsoCalled ? ` <span class="gl-also">(${esc(tr("glossary.also", "Also called"))}: ${esc(t.alsoCalled)})</span>` : ""}</dt>
    <dd>${esc(t.definition)}
      ${t.example ? `<div class="gl-example"><b>${esc(tr("glossary.example", "Example"))}:</b> ${esc(t.example)}</div>` : ""}
      ${t.note ? `<div class="gl-note"><b>${esc(tr("glossary.note", "Note"))}:</b> ${esc(t.note)}</div>` : ""}
    </dd></div>`;
}
/** glossaryHTML renders all terms, grouped (core terms first, then terms used on screen). */
export function glossaryHTML(terms, tr) {
    const groups = [
        ["core", tr("glossary.group.core", "Terms")],
        ["screen", tr("glossary.group.screen", "Terms used on screen")],
    ];
    const body = groups.map(([g, title]) => {
        const list = terms.filter((t) => t.group === g);
        if (list.length === 0)
            return "";
        return `<h3>${esc(title)}</h3><dl class="gl-list">${list.map((t) => termHTML(t, tr)).join("")}</dl>`;
    }).join("");
    return `<div class="body"><p class="muted">${esc(tr("glossary.intro", "The words used in this program and what they mean."))}</p>${body}</div>`;
}
