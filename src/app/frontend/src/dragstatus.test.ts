import { expect, test } from "bun:test";
import fc from "fast-check";
import { changeText, dragStatusText, isoDay, milestoneDragStatusText } from "./dragstatus";

const D = (s: string) => Date.parse(s + "T00:00:00Z");

test("moving an action: the start comes first, then end, duration and the change", () => {
  const t = dragStatusText({
    name: "AWS Landing Zone", mode: "move",
    origStart: D("2026-12-02"), origEnd: D("2027-01-07"),
    start: D("2026-12-05"), end: D("2027-01-10"),
  });
  expect(t).toBe('Moving "AWS Landing Zone": start 2026-12-05 (+3 days) · end 2027-01-10 (+3 days) · 37 days (25 working days)');
  expect(t.indexOf("start 2026-12-05")).toBeLessThan(t.indexOf("end 2027-01-10"));
});

test("resizing names the edge that changes; the other edge reads unchanged", () => {
  const left = dragStatusText({ name: "X", mode: "l", origStart: D("2026-03-02"), origEnd: D("2026-03-13"), start: D("2026-03-04"), end: D("2026-03-13") });
  expect(left).toContain('Changing the start of "X"');
  expect(left).toContain("start 2026-03-04 (+2 days)");
  expect(left).toContain("end 2026-03-13 (unchanged)");
  const right = dragStatusText({ name: "X", mode: "r", origStart: D("2026-03-02"), origEnd: D("2026-03-13"), start: D("2026-03-02"), end: D("2026-03-11") });
  expect(right).toContain('Changing the end of "X"');
  expect(right).toContain("start 2026-03-02 (unchanged)");
  expect(right).toContain("end 2026-03-11 (-2 days)");
});

test("a move into another section says so", () => {
  const t = dragStatusText({ name: "X", mode: "move", origStart: D("2026-03-02"), origEnd: D("2026-03-06"),
    start: D("2026-03-02"), end: D("2026-03-06"), intoSection: "Voice rollout" });
  expect(t).toContain('into section "Voice rollout"');
  expect(dragStatusText({ name: "X", mode: "move", origStart: 0, origEnd: 0, start: 0, end: 0, intoSection: "" })).toContain('into section "(none)"');
  expect(dragStatusText({ name: "X", mode: "move", origStart: 0, origEnd: 0, start: 0, end: 0 })).not.toContain("into section");
});

test("singular and plural, working days exclude weekends", () => {
  const one = dragStatusText({ name: "X", mode: "move", origStart: D("2026-03-02"), origEnd: D("2026-03-02"), start: D("2026-03-03"), end: D("2026-03-03") });
  expect(one).toContain("(+1 day)");
  expect(one).toContain("1 day (1 working day)");
  const weekend = dragStatusText({ name: "X", mode: "move", origStart: D("2026-03-07"), origEnd: D("2026-03-08"), start: D("2026-03-07"), end: D("2026-03-08") });
  expect(weekend).toContain("2 days (0 working days)");
});

test("milestone drag shows the date it would move to", () => {
  expect(milestoneDragStatusText("Go-live", D("2027-01-10"), D("2027-01-07"))).toBe('Moving milestone "Go-live": date 2027-01-07 (-3 days)');
  expect(changeText(0)).toBe("unchanged");
  expect(isoDay(D("2026-11-04"))).toBe("2026-11-04");
});

// Feature: the status text always states the exact start the action would have.
test("property: the start shown is the start that would be applied", () => {
  const day = fc.integer({ min: 0, max: 800 });
  fc.assert(fc.property(day, fc.integer({ min: 0, max: 60 }), fc.integer({ min: -30, max: 30 }),
    fc.constantFrom("move", "l", "r"), (a, len, shift, mode) => {
      const origStart = D("2026-01-01") + a * 86400000, origEnd = origStart + len * 86400000;
      const start = origStart + shift * 86400000, end = origEnd + shift * 86400000;
      const t = dragStatusText({ name: "N", mode, origStart, origEnd, start, end });
      expect(t).toContain(`start ${isoDay(start)} (${changeText(start - origStart)})`);
      expect(t).toContain(`end ${isoDay(end)} (${changeText(end - origEnd)})`);
      expect(t.indexOf(`start ${isoDay(start)}`)).toBeLessThan(t.indexOf(`end ${isoDay(end)}`));
    }), { numRuns: 100 });
});
