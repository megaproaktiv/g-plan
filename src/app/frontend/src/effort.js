// Display of an effort in the action views: the PROJECT'S STANDARD UNIT (days or
// hours) comes first and is bold, the other unit follows in brackets.
const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
const daysLabel = (d) => `${d} ${d === "1" ? "day" : "days"}`;
/** effortParts returns the text of the main value (project unit) and the other one. */
export function effortParts(unit, hours, days) {
    return unit === "hours"
        ? { main: `${hours} h`, other: daysLabel(days) }
        : { main: daysLabel(days), other: `${hours} h` };
}
/** effortText is the plain-text form: "6,56 days (52,5 h)". */
export function effortText(unit, hours, days) {
    const { main, other } = effortParts(unit, hours, days);
    return `${main} (${other})`;
}
/** effortHTML is the HTML form: the main value bold, the other in brackets. */
export function effortHTML(unit, hours, days) {
    const { main, other } = effortParts(unit, hours, days);
    return `<b>${esc(main)}</b> (${esc(other)})`;
}
