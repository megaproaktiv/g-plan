const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
/** inline escapes a text and applies the inline markup. Code is never styled further. */
export function inline(text) {
    return String(text ?? "").split(/(`[^`]*`)/).map((part) => {
        if (part.length >= 2 && part.startsWith("`") && part.endsWith("`"))
            return `<code>${esc(part.slice(1, -1))}</code>`;
        return esc(part)
            .replace(/\*\*([^*]+)\*\*/g, "<b>$1</b>")
            .replace(/\*([^*]+)\*/g, "<i>$1</i>");
    }).join("");
}
export function blockHTML(b) {
    switch (b.kind) {
        case "p": return `<p>${inline(b.text)}</p>`;
        case "note": return `<p class="hp-note">${inline(b.text)}</p>`;
        case "ul": return `<ul>${(b.items ?? []).map((i) => `<li>${inline(i)}</li>`).join("")}</ul>`;
        case "ol": return `<ol>${(b.items ?? []).map((i) => `<li>${inline(i)}</li>`).join("")}</ol>`;
        case "table":
            return `<table class="help-rel"><thead><tr>${(b.head ?? []).map((h) => `<th>${inline(h)}</th>`).join("")}</tr></thead>`
                + `<tbody>${(b.rows ?? []).map((r) => `<tr>${r.map((c) => `<td>${inline(c)}</td>`).join("")}</tr>`).join("")}</tbody></table>`;
        default: return "";
    }
}
/** helpPageHTML renders a whole page. */
export function helpPageHTML(page) {
    const sections = page.sections.map((s) => `<section id="hp_${esc(s.key)}"><h3>${inline(s.heading)}</h3>${s.blocks.map(blockHTML).join("")}</section>`).join("");
    return `<div class="body help-guide"><p class="hp-intro">${inline(page.intro)}</p>${sections}</div>`;
}
