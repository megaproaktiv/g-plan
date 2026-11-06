import { expect, test } from "bun:test";
import { syncAfterChat, type ChatSyncDeps } from "./chatflow";

// Records what happens, in order, so the tests can check the sequence.
function deps(over: Partial<ChatSyncDeps> & { sel?: { kind: string; id: string } | Error; backendMode?: string } = {}) {
  const log: string[] = [];
  let current: any = { packages: ["old"] };
  const d: ChatSyncDeps = {
    getProject: async () => { log.push("getProject"); return { packages: ["old", "NEW"] } as any; },
    setProject: (p) => { current = p; log.push("setProject"); },
    mode: async () => { log.push("mode"); return over.backendMode ?? "action"; },
    applyMode: (m) => log.push("applyMode:" + m),
    refresh: () => { log.push("refresh:" + current.packages.join("+")); },
    selection: async () => {
      log.push("selection");
      if (over.sel instanceof Error) throw over.sel;
      return over.sel ?? { kind: "action", id: "x" };
    },
    selectAction: (id) => log.push("selectAction:" + id),
    selectSection: (id) => log.push("selectSection:" + id),
    selectMilestone: (id) => log.push("selectMilestone:" + id),
    ...over,
  };
  return { d, log };
}

// The bug: a newly added action was only drawn after Save, because the canvas
// never received the new project. It must be drawn before the selection is applied.
test("the new project is drawn first, then the selection follows", async () => {
  const { d, log } = deps({ sel: { kind: "action", id: "newaction" }, backendMode: "action" });
  await syncAfterChat(d);
  // mode "action" equals the UI default here, but applyMode is still called to
  // reflect whatever the backend reports; the ORDER is what matters.
  expect(log).toEqual(["getProject", "setProject", "mode", "applyMode:action", "refresh:old+NEW", "selection", "selectAction:newaction"]);
});

// The real-world bug this guards: after `mode section` (or a bare `add` that
// keeps section mode) the backend mode changed, and the UI must reflect it
// BEFORE the refresh so the mode selector + status line are not stale.
test("the backend mode is applied before the refresh", async () => {
  const { d, log } = deps({ backendMode: "section" });
  await syncAfterChat(d);
  expect(log).toContain("applyMode:section");
  expect(log.indexOf("applyMode:section")).toBeLessThan(log.indexOf("refresh:old+NEW"));
});

// A failing mode fetch must not stop the project from being drawn.
test("a failing mode lookup still leaves the project drawn", async () => {
  const { d, log } = deps({ mode: async () => { throw new Error("no mode"); } });
  await syncAfterChat(d);
  expect(log).toContain("refresh:old+NEW");
  expect(log.filter((l) => l.startsWith("applyMode"))).toEqual([]);
});

test("the canvas is refreshed with the NEW project (not the stale one)", async () => {
  const { d, log } = deps();
  await syncAfterChat(d);
  const refresh = log.find((l) => l.startsWith("refresh:"))!;
  expect(refresh).toContain("NEW");
});

test("a failing selection lookup still leaves the project drawn", async () => {
  const { d, log } = deps({ sel: new Error("no selection") });
  await syncAfterChat(d);
  expect(log).toContain("refresh:old+NEW");
  expect(log.filter((l) => l.startsWith("select") && l !== "selection")).toEqual([]);
});

test("each kind of selection goes to its own handler; no selection selects nothing", async () => {
  for (const [kind, expected] of [["section", "selectSection:s1"], ["milestone", "selectMilestone:m1"]] as const) {
    const { d, log } = deps({ sel: { kind, id: kind === "section" ? "s1" : "m1" } });
    await syncAfterChat(d);
    expect(log.at(-1)).toBe(expected);
    expect(log.indexOf("refresh:old+NEW")).toBeLessThan(log.indexOf(expected));
  }
  const { d, log } = deps({ sel: { kind: "", id: "" } });
  await syncAfterChat(d);
  expect(log.filter((l) => /^select(Action|Section|Milestone):/.test(l))).toEqual([]);
  expect(log).toContain("refresh:old+NEW");
});

test("a failing project fetch is reported to the caller and nothing is drawn or selected", async () => {
  const { d, log } = deps({ getProject: async () => { throw new Error("backend down"); } });
  await expect(syncAfterChat(d)).rejects.toThrow("backend down");
  expect(log).toEqual([]);
});
