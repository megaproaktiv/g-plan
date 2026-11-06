// Pure, DOM-free helpers for progress visualization (contour curve geometry and
// marker/curve colour selection). Kept separate from gantt.ts so they can be
// unit/property-tested without a browser.
/** Bars narrower than this (px) get no contour curve (R8.9). */
export const MIN_CURVE_WIDTH = 8;
const SAMPLES = 40; // number of polyline segments across the bar
const PEAK_U = 0.1; // peak position as a fraction of width (inside 25%)
const PAD = 2; // vertical padding inside the bar, px
const H_LOW = 0.1; // lowest profile height (0..1)
/** Height profile h(u) in [0,1] (higher = more effort density), u in [0,1]. */
function frontProfile(u) {
    if (u <= PEAK_U)
        return 0.4 + 0.6 * (u / PEAK_U);
    return 1 - (1 - H_LOW) * ((u - PEAK_U) / (1 - PEAK_U));
}
function profile(kind, u) {
    switch (kind) {
        case "Front-Loaded": return frontProfile(u);
        case "Back-Loaded": return frontProfile(1 - u);
        case "Double-Peak": {
            // two edge peaks at PEAK_U and 1-PEAK_U, single trough at 0.5
            if (u <= 0.5) {
                if (u <= PEAK_U)
                    return 0.4 + 0.6 * (u / PEAK_U);
                return 1 - (1 - H_LOW) * ((u - PEAK_U) / (0.5 - PEAK_U));
            }
            return profile("Double-Peak", 1 - u);
        }
        default: return 0.5; // Flat, and any unset/invalid kind (R8.8)
    }
}
/**
 * contourCurvePoints returns the polyline points of the contour curve spanning
 * the full bar width inside [by, by+h]. Returns [] when the bar is narrower
 * than MIN_CURVE_WIDTH (R8.9). Unknown kinds draw as Flat (R8.8).
 */
export function contourCurvePoints(rect, kind) {
    if (kind === "None")
        return []; // no estimation: nothing to draw
    const w = rect.x1 - rect.x0;
    if (!(w >= MIN_CURVE_WIDTH))
        return [];
    const pts = [];
    const band = rect.h - 2 * PAD;
    for (let i = 0; i <= SAMPLES; i++) {
        const u = i / SAMPLES;
        const x = i === SAMPLES ? rect.x1 : rect.x0 + u * w;
        const y = rect.by + PAD + (1 - profile(kind, u)) * band;
        pts.push([x, y]);
    }
    return pts;
}
// ---- colour helpers -------------------------------------------------------
const DEFAULT_BG = "#8a90dd";
/** parseHex parses #rgb or #rrggbb to [r,g,b] 0..255, or null. */
export function parseHex(hex) {
    const m = /^#?([0-9a-f]{3}|[0-9a-f]{6})$/i.exec((hex || "").trim());
    if (!m)
        return null;
    let h = m[1];
    if (h.length === 3)
        h = h[0] + h[0] + h[1] + h[1] + h[2] + h[2];
    return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)];
}
/** WCAG relative luminance (0..1) of a hex colour; unparseable => default bar colour. */
export function relativeLuminance(hex) {
    const rgb = parseHex(hex) ?? parseHex(DEFAULT_BG);
    const lin = (c) => {
        const s = c / 255;
        return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
    };
    return 0.2126 * lin(rgb[0]) + 0.7152 * lin(rgb[1]) + 0.0722 * lin(rgb[2]);
}
/** WCAG contrast ratio (1..21) between two hex colours. */
export function contrastRatio(a, b) {
    const la = relativeLuminance(a), lb = relativeLuminance(b);
    const hi = Math.max(la, lb), lo = Math.min(la, lb);
    return (hi + 0.05) / (lo + 0.05);
}
const BLACK = "#000000";
const WHITE = "#ffffff";
/**
 * markerColor picks black or white, whichever has the higher WCAG contrast
 * ratio against bg. The best of the two is always >= 4.5:1, so the 3:1
 * requirement (R7.4) holds for every background.
 */
export function markerColor(bg) {
    return contrastRatio(BLACK, bg) >= contrastRatio(WHITE, bg) ? BLACK : WHITE;
}
/**
 * curveColor picks black or white, whichever differs more in relative
 * luminance from bg. max(L, 1-L) >= 0.5, so the "at least half the range"
 * requirement (R8.6) holds for every background.
 */
export function curveColor(bg) {
    const L = relativeLuminance(bg);
    return Math.abs(L - 0) >= Math.abs(1 - L) ? BLACK : WHITE;
}
// ---- shaded area under the contour curve + rating text ----------------------
/**
 * contourAreaPoints returns the polygon of the area under the contour curve from
 * the start of the bar up to the progress marker (pos, 0..1), closed along the
 * bar's baseline. The area is the effort done, so its share of the whole area
 * under the curve is the pure consumed rating. Returns [] for bars narrower than
 * MIN_CURVE_WIDTH or when nothing has been done yet.
 */
export function contourAreaPoints(rect, kind, pos) {
    if (kind === "None")
        return [];
    const w = rect.x1 - rect.x0;
    if (!(w >= MIN_CURVE_WIDTH) || !(pos > 0))
        return [];
    const p = Math.min(1, pos);
    const band = rect.h - 2 * PAD;
    const yAt = (u) => rect.by + PAD + (1 - profile(kind, u)) * band;
    const baseY = rect.by + rect.h - PAD;
    const pts = [[rect.x0, baseY]];
    for (let i = 0; i <= SAMPLES; i++) {
        const u = i / SAMPLES;
        if (u >= p)
            break;
        pts.push([rect.x0 + u * w, yAt(u)]);
    }
    pts.push([rect.x0 + p * w, yAt(p)], [rect.x0 + p * w, baseY]);
    return pts;
}
/** formatPermille renders tenths of a percent: 164 -> "16.4%" (or "16,4%" with a decimal comma). */
export function formatPermille(permille, decimalComma = false) {
    const s = (Math.max(0, permille) / 10).toFixed(1);
    return (decimalComma ? s.replace(".", ",") : s) + "%";
}
/**
 * ratingLabel is the text shown with an action bar: the pure consumed rating and,
 * when the contour changes it, the contour rating as well. Empty when nothing is
 * planned.
 */
export function ratingLabel(planned, pure, contour, kind, decimalComma = false) {
    if (!planned)
        return "";
    const a = formatPermille(pure, decimalComma);
    if (kind === "Flat" || kind === "" || pure === contour)
        return a;
    return `${a} (${kind} ${formatPermille(contour, decimalComma)})`;
}
