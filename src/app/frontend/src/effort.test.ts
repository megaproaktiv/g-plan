import { expect, test } from "bun:test";
import fc from "fast-check";
import { effortHTML, effortParts, effortText } from "./effort";

// The case from the review: project unit is days.
test("days project: bold value in days, hours in brackets", () => {
  expect(effortHTML("days", "320", "40")).toBe("<b>40 days</b> (320 h)");
  expect(effortHTML("days", "52,5", "6,56")).toBe("<b>6,56 days</b> (52,5 h)");
  expect(effortHTML("days", "4", "0,5")).toBe("<b>0,5 days</b> (4 h)");
  expect(effortHTML("days", "48,5", "6,06")).toBe("<b>6,06 days</b> (48,5 h)");
  expect(effortText("days", "52,5", "6,56")).toBe("6,56 days (52,5 h)");
});

test("hours project: bold value in hours, days in brackets", () => {
  expect(effortHTML("hours", "52,5", "6,56")).toBe("<b>52,5 h</b> (6,56 days)");
  expect(effortHTML("hours", "320", "40")).toBe("<b>320 h</b> (40 days)");
  expect(effortText("hours", "4", "0,5")).toBe("4 h (0,5 days)");
});

test("singular day, and an unknown unit counts as days", () => {
  expect(effortText("days", "8", "1")).toBe("1 day (8 h)");
  expect(effortText("hours", "8", "1")).toBe("8 h (1 day)");
  expect(effortParts("whatever", "8", "1").main).toBe("1 day");
});

// Feature: the value in the project's unit always comes first; the other is the
// only thing in brackets and nothing is lost or added.
test("property: project unit first, both values present, HTML escaped", () => {
  const num = fc.tuple(fc.integer({ min: 0, max: 9999 }), fc.integer({ min: 0, max: 99 }))
    .map(([a, b]) => (b === 0 ? `${a}` : `${a},${b}`));
  fc.assert(fc.property(num, num, fc.constantFrom("days", "hours"), (h, d, unit) => {
    const t = effortText(unit, h, d);
    const open = t.indexOf(" (");
    const main = t.slice(0, open), other = t.slice(open + 2, -1);
    if (unit === "days") {
      expect(main.endsWith(d === "1" ? " day" : " days")).toBe(true);
      expect(main.startsWith(d)).toBe(true);
      expect(other).toBe(`${h} h`);
    } else {
      expect(main).toBe(`${h} h`);
      expect(other.startsWith(d)).toBe(true);
    }
    expect(t.endsWith(")")).toBe(true);
    expect(effortHTML(unit, h, d).startsWith("<b>")).toBe(true);
  }), { numRuns: 100 });
  expect(effortHTML("days", "<i>", "&")).not.toContain("<i>");
});
