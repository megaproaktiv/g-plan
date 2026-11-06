import { describe, expect, test } from "bun:test";
import { contextMenuItems, type CtxActions } from "./ctxmenu";

function spy() {
  const calls: string[] = [];
  const a = new Proxy({}, { get: (_t, name: string) => (...args: unknown[]) => { calls.push(`${name}(${args.join(",")})`); } }) as unknown as CtxActions;
  return { a, calls };
}

describe("context menu", () => {
  test("section label offers Move Up/Down and Delete, wired to the section", async () => {
    const { a, calls } = spy();
    const items = contextMenuItems({ kind: "section-label", cluster: "WP4" }, a);
    expect(items.map((i) => i.label)).toEqual(["Edit Section", "Move Section Up", "Move Section Down", "Delete Section"]);
    for (const i of items) await i.run();
    expect(calls).toEqual(["editSection(WP4)", "moveSection(WP4,-1)", "moveSection(WP4,1)", "deleteSection(WP4)"]);
  });
  test("the Phases lane is neither movable nor deletable", () => {
    const { a } = spy();
    expect(contextMenuItems({ kind: "section-label", cluster: "Phases" }, a).map((i) => i.label)).toEqual(["Edit Section"]);
  });
  test("action: Delete runs immediately (no confirm dialog) and Move works", async () => {
    const { a, calls } = spy();
    const items = contextMenuItems({ kind: "action", id: "x" }, a);
    expect(items.map((i) => i.label)).toContain("Delete Action");
    await items.find((i) => i.label === "Delete Action")!.run();
    await items.find((i) => i.label === "Move Action Up")!.run();
    expect(calls).toEqual(["deleteAction(x)", "moveAction(x,-1)"]);
  });
});
