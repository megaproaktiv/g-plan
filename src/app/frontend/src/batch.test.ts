import { expect, test } from "bun:test";
import { batchCommandCount, batchSummary } from "./batch";

const r = (o: object) => ({ total: 3, done: 3, failed: 0, stopped: 0, error: "", log: [], ...o });

test("summary: done, error, quit, empty", () => {
  expect(batchSummary(r({}))).toBe("Batch done: 3 of 3 commands.");
  expect(batchSummary(r({ done: 1, failed: 2, error: "task not found" }))).toContain("stopped at line 2 (1 of 3 commands done): task not found");
  expect(batchSummary(r({ done: 1, stopped: 2 }))).toContain("quit command at line 2");
  expect(batchSummary(r({ total: 0, done: 0 }))).toBe("Nothing to run: no commands.");
});

test("command count ignores blank lines and comments", () => {
  expect(batchCommandCount("# plan\n\nadd A\r\n// x\n  add B  \n")).toBe(2);
  expect(batchCommandCount("")).toBe(0);
});
