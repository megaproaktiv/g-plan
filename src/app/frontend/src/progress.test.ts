import { describe, expect, test } from "bun:test";
import fc from "fast-check";
import {
  contourAreaPoints, contourCurvePoints, curveColor, markerColor, formatPermille, ratingLabel,
  contrastRatio, relativeLuminance, MIN_CURVE_WIDTH, type BarRect,
} from "./progress";

const kinds = ["Flat", "Front-Loaded", "Back-Loaded", "Double-Peak"] as const;

const rectArb = fc.record({
  x0: fc.double({ min: 0, max: 200, noNaN: true }),
  // keep a margin above the minimum so x1-x0 can't round below it
  w: fc.double({ min: MIN_CURVE_WIDTH + 0.5, max: 800, noNaN: true }),
  by: fc.double({ min: 0, max: 400, noNaN: true }),
  h: fc.double({ min: 10, max: 60, noNaN: true }),
}).map(({ x0, w, by, h }): BarRect => ({ x0, x1: x0 + w, by, h }));

const hexArb = fc.tuple(
  fc.integer({ min: 0, max: 255 }), fc.integer({ min: 0, max: 255 }), fc.integer({ min: 0, max: 255 }),
).map(([r, g, b]) => `#${[r, g, b].map((v) => v.toString(16).padStart(2, "0")).join("")}`);

// Feature: progress, Property 13: The contour curve stays inside the action bar bounds
test("Property 13: curve stays within bar bounds and spans full width", () => {
  fc.assert(fc.property(rectArb, fc.constantFrom(...kinds), (rect, kind) => {
    const pts = contourCurvePoints(rect, kind);
    expect(pts.length).toBeGreaterThan(0);
    for (const [x, y] of pts) {
      expect(x).toBeGreaterThanOrEqual(rect.x0 - 1e-6);
      expect(x).toBeLessThanOrEqual(rect.x1 + 1e-6);
      expect(y).toBeGreaterThanOrEqual(rect.by - 1e-6);
      expect(y).toBeLessThanOrEqual(rect.by + rect.h + 1e-6);
    }
    expect(pts[0][0]).toBeCloseTo(rect.x0, 5);
    expect(pts[pts.length - 1][0]).toBeCloseTo(rect.x1, 5);
  }), { numRuns: 100 });
});

test("curve omitted below the minimum width", () => {
  fc.assert(fc.property(
    fc.double({ min: 0, max: MIN_CURVE_WIDTH - 0.01, noNaN: true }),
    fc.constantFrom(...kinds),
    (w, kind) => {
      const pts = contourCurvePoints({ x0: 0, x1: w, by: 0, h: 20 }, kind);
      expect(pts).toEqual([]);
    },
  ), { numRuns: 100 });
});

// Feature: progress, Property 14: Flat curve is a constant-height horizontal line
test("Property 14: Flat (and unset/invalid) curve is constant height", () => {
  fc.assert(fc.property(rectArb, fc.constantFrom("Flat", "", "bogus", "flat"), (rect, kind) => {
    const pts = contourCurvePoints(rect, kind);
    const ys = pts.map(([, y]) => y);
    for (const y of ys) expect(y).toBeCloseTo(ys[0], 5);
  }), { numRuns: 100 });
});

// Feature: progress, Property 15: Front-Loaded curve peaks in the left quarter and descends
test("Property 15: Front-Loaded peaks in the left quarter and descends", () => {
  fc.assert(fc.property(rectArb, (rect) => {
    const pts = contourCurvePoints(rect, "Front-Loaded");
    const w = rect.x1 - rect.x0;
    let peakIdx = 0;
    for (let i = 1; i < pts.length; i++) if (pts[i][1] < pts[peakIdx][1]) peakIdx = i;
    expect(pts[peakIdx][0] - rect.x0).toBeLessThanOrEqual(0.25 * w + 1e-6);
    for (let i = peakIdx; i < pts.length - 1; i++) {
      expect(pts[i][1]).toBeLessThanOrEqual(pts[i + 1][1] + 1e-6);
    }
  }), { numRuns: 100 });
});

// Feature: progress, Property 16: Back-Loaded curve rises to a peak in the right quarter
test("Property 16: Back-Loaded rises to a peak in the right quarter", () => {
  fc.assert(fc.property(rectArb, (rect) => {
    const pts = contourCurvePoints(rect, "Back-Loaded");
    const w = rect.x1 - rect.x0;
    let peakIdx = 0;
    for (let i = 1; i < pts.length; i++) if (pts[i][1] < pts[peakIdx][1]) peakIdx = i;
    expect(pts[peakIdx][0] - rect.x0).toBeGreaterThanOrEqual(0.75 * w - 1e-6);
    for (let i = 0; i < peakIdx; i++) {
      expect(pts[i][1]).toBeGreaterThanOrEqual(pts[i + 1][1] - 1e-6);
    }
  }), { numRuns: 100 });
});

// Feature: progress, Property 17: Double-Peak curve has two edge peaks and a central trough
test("Property 17: Double-Peak has two edge peaks and a central trough", () => {
  fc.assert(fc.property(rectArb, (rect) => {
    const pts = contourCurvePoints(rect, "Double-Peak");
    const w = rect.x1 - rect.x0;
    const left = pts.filter(([x]) => x - rect.x0 <= 0.25 * w + 1e-6);
    const right = pts.filter(([x]) => x - rect.x0 >= 0.75 * w - 1e-6);
    const mid = pts.filter(([x]) => x - rect.x0 > 0.25 * w + 1e-6 && x - rect.x0 < 0.75 * w - 1e-6);
    const minY = Math.min(...pts.map(([, y]) => y));
    const leftHasPeak = left.some(([, y]) => Math.abs(y - minY) < 1e-6);
    const rightHasPeak = right.some(([, y]) => Math.abs(y - minY) < 1e-6);
    expect(leftHasPeak).toBe(true);
    expect(rightHasPeak).toBe(true);
    const maxY = Math.max(...pts.map(([, y]) => y));
    const midHasTrough = mid.some(([, y]) => Math.abs(y - maxY) < 1e-6);
    expect(midHasTrough).toBe(true);
  }), { numRuns: 100 });
});

// Feature: progress, Property 18: Progress marker colour meets the 3:1 contrast ratio
test("Property 18: marker colour meets 3:1 contrast", () => {
  fc.assert(fc.property(hexArb, (bg) => {
    expect(contrastRatio(markerColor(bg), bg)).toBeGreaterThanOrEqual(3);
  }), { numRuns: 100 });
});

// Feature: progress, Property 19: Contour curve colour differs in luminance by at least half the range
test("Property 19: curve colour luminance differs by at least 0.5", () => {
  fc.assert(fc.property(hexArb, (bg) => {
    const diff = Math.abs(relativeLuminance(curveColor(bg)) - relativeLuminance(bg));
    expect(diff).toBeGreaterThanOrEqual(0.5 - 1e-9);
  }), { numRuns: 100 });
});

describe("sanity", () => {
  test("parseHex round-trips contrastOn-style black/white choice", () => {
    expect(markerColor("#000000")).toBe("#ffffff");
    expect(markerColor("#ffffff")).toBe("#000000");
  });
});

// ---- area under the curve (the effort done) -------------------------------

/** shoelace area of a polygon */
const polyArea = (pts: Array<[number, number]>) => {
  let s = 0;
  for (let i = 0; i < pts.length; i++) {
    const [x1, y1] = pts[i], [x2, y2] = pts[(i + 1) % pts.length];
    s += x1 * y2 - x2 * y1;
  }
  return Math.abs(s) / 2;
};
const whole = (rect: BarRect, kind: string) => polyArea(contourAreaPoints(rect, kind, 1));

// Reference: the marker position reached for an effort share, computed
// independently (exact piecewise integration in Python). The shaded area up to
// that position must be exactly that share of the whole area.
const reference: Array<[string, number, number]> = [
  ["Flat", 0.25, 0.25], ["Flat", 0.5, 0.5],
  ["Front-Loaded", 0.1740, 0.25], ["Front-Loaded", 0.3417, 0.5], ["Front-Loaded", 0.5592, 0.75],
  ["Back-Loaded", 0.4408, 0.25], ["Back-Loaded", 0.6583, 0.5], ["Back-Loaded", 0.8260, 0.75],
  ["Double-Peak", 0.1827, 0.25], ["Double-Peak", 0.5, 0.5], ["Double-Peak", 0.8173, 0.75],
];
test("shaded area up to the marker equals the effort share (independent reference)", () => {
  const rect: BarRect = { x0: 10, x1: 410, by: 5, h: 20 };
  for (const [kind, pos, share] of reference) {
    const got = polyArea(contourAreaPoints(rect, kind, pos)) / whole(rect, kind);
    expect(Math.abs(got - share)).toBeLessThan(2e-4);
  }
});

// Feature: the shaded share does not depend on the bar's size, grows with the
// marker position, and ends exactly at the marker.
test("area polygon: size-independent, monotonic, ends at the marker", () => {
  fc.assert(fc.property(rectArb, fc.constantFrom(...kinds),
    fc.double({ min: 0.01, max: 0.98, noNaN: true }), fc.double({ min: 0.0, max: 0.01, noNaN: true }),
    (rect, kind, pos, extra) => {
      const a = polyArea(contourAreaPoints(rect, kind, pos)) / whole(rect, kind);
      const b = polyArea(contourAreaPoints(rect, kind, pos + extra)) / whole(rect, kind);
      expect(b).toBeGreaterThanOrEqual(a - 1e-9);
      const unit: BarRect = { x0: 0, x1: 100, by: 0, h: 20 };
      const ref = polyArea(contourAreaPoints(unit, kind, pos)) / whole(unit, kind);
      expect(Math.abs(a - ref)).toBeLessThan(1e-9);
      const pts = contourAreaPoints(rect, kind, pos);
      const last = pts[pts.length - 1];
      expect(last[0]).toBeCloseTo(rect.x0 + pos * (rect.x1 - rect.x0), 6);
      for (const [x, y] of pts) { // stays inside the bar
        expect(x).toBeGreaterThanOrEqual(rect.x0 - 1e-6); expect(x).toBeLessThanOrEqual(rect.x1 + 1e-6);
        expect(y).toBeGreaterThanOrEqual(rect.by - 1e-6); expect(y).toBeLessThanOrEqual(rect.by + rect.h + 1e-6);
      }
    }), { numRuns: 100 });
});

test("area polygon: nothing for no progress or a tiny bar", () => {
  expect(contourAreaPoints({ x0: 0, x1: 100, by: 0, h: 20 }, "Flat", 0)).toEqual([]);
  expect(contourAreaPoints({ x0: 0, x1: 5, by: 0, h: 20 }, "Flat", 0.5)).toEqual([]);
  expect(contourAreaPoints({ x0: 0, x1: 100, by: 0, h: 20 }, "Flat", 7).length).toBeGreaterThan(2); // clamped to the bar end
});

test("rating text: pure rating, plus the contour rating when it differs", () => {
  expect(formatPermille(164)).toBe("16.4%");
  expect(formatPermille(164, true)).toBe("16,4%");
  expect(formatPermille(0)).toBe("0.0%");
  expect(formatPermille(1000)).toBe("100.0%");
  expect(ratingLabel(true, 164, 164, "Flat")).toBe("16.4%");
  expect(ratingLabel(true, 164, 126, "Double-Peak")).toBe("16.4% (Double-Peak 12.6%)");
  expect(ratingLabel(true, 164, 126, "Double-Peak", true)).toBe("16,4% (Double-Peak 12,6%)");
  expect(ratingLabel(true, 500, 500, "Front-Loaded")).toBe("50.0%"); // same value: no duplicate
  expect(ratingLabel(false, 0, 0, "Flat")).toBe(""); // nothing planned
});

test("contour None draws nothing: no curve and no shaded area", () => {
  const rect: BarRect = { x0: 0, x1: 300, by: 0, h: 20 };
  expect(contourCurvePoints(rect, "None")).toEqual([]);
  expect(contourAreaPoints(rect, "None", 0.5)).toEqual([]);
  // other kinds are unaffected
  expect(contourCurvePoints(rect, "Flat").length).toBeGreaterThan(2);
});
