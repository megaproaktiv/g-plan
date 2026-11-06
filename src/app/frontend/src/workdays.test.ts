import { expect, test } from "bun:test";
import fc from "fast-check";
import { workdayCount } from "./workdays";

test("known ranges", () => {
  expect(workdayCount("2026-11-02", "2026-11-06")).toBe(5);  // Mon-Fri
  expect(workdayCount("2026-11-02", "2026-11-08")).toBe(5);  // Mon-Sun
  expect(workdayCount("2026-11-07", "2026-11-08")).toBe(0);  // Sat-Sun
  expect(workdayCount("2026-11-06", "2026-11-09")).toBe(2);  // Fri-Mon
  expect(workdayCount("2026-11-04", "2026-11-04")).toBe(1);  // single Wednesday
  expect(workdayCount("2026-11-07", "2026-11-07")).toBe(0);  // single Saturday
  expect(workdayCount("2026-01-01", "2026-12-31")).toBe(261);
  expect(workdayCount("2026-11-09", "2026-11-02")).toBe(0);  // reversed
  expect(workdayCount("", "2026-11-02")).toBe(0);            // invalid
});

// Feature: working days - agrees with a plain day-by-day count for any window.
test("matches the naive day-by-day count", () => {
  const naive = (s: string, e: string) => {
    let n = 0;
    for (let d = Date.parse(s + "T00:00:00Z"); d <= Date.parse(e + "T00:00:00Z"); d += 86400000) {
      const w = new Date(d).getUTCDay();
      if (w !== 0 && w !== 6) n++;
    }
    return n;
  };
  const day = fc.integer({ min: 0, max: 1500 }).map((i) => new Date(Date.UTC(2025, 0, 1) + i * 86400000).toISOString().slice(0, 10));
  fc.assert(fc.property(day, fc.integer({ min: 0, max: 800 }), (s, len) => {
    const e = new Date(Date.parse(s + "T00:00:00Z") + len * 86400000).toISOString().slice(0, 10);
    expect(workdayCount(s, e)).toBe(naive(s, e));
  }), { numRuns: 100 });
});
