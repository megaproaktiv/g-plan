// Right-click menu content, kept free of DOM and backend so it can be tested.
// Every entry runs a callback of CtxActions; main.ts supplies the callbacks.
//
// Deleting is NOT confirmed with window.confirm(): the desktop web view does not
// reliably show JS dialogs (confirm() returned false, so "Delete" appeared dead).
// All deletes are undoable with Cmd-Z instead.
import type { CtxTarget } from "./gantt";

export interface CtxItem { label: string; run: () => void | Promise<void>; }

export interface CtxActions {
  editAction(id: string): void;
  deleteAction(id: string): void | Promise<void>;
  moveAction(id: string, dir: -1 | 1): void | Promise<void>;
  editMilestone(id: string): void;
  deleteMilestone(id: string): void | Promise<void>;
  editSection(cluster: string): void;
  deleteSection(cluster: string): void | Promise<void>;
  moveSection(cluster: string, dir: -1 | 1): void | Promise<void>;
  newAction(cluster: string): void;
  newMilestone(cluster: string, date: string): void;
  newSection(): void | Promise<void>;
  addRole(): void | Promise<void>;
}

const isPhases = (c: string) => c.toLowerCase() === "phases";

export function contextMenuItems(target: CtxTarget, a: CtxActions): CtxItem[] {
  switch (target.kind) {
    case "action": {
      const id = target.id;
      return [
        { label: "Edit Action", run: () => a.editAction(id) },
        { label: "Move Action Up", run: () => a.moveAction(id, -1) },
        { label: "Move Action Down", run: () => a.moveAction(id, 1) },
        { label: "Delete Action", run: () => a.deleteAction(id) },
      ];
    }
    case "milestone": {
      const id = target.id;
      return [
        { label: "Edit Milestone", run: () => a.editMilestone(id) },
        { label: "Delete Milestone", run: () => a.deleteMilestone(id) },
      ];
    }
    case "section-label": {
      const c = target.cluster;
      const items: CtxItem[] = [{ label: "Edit Section", run: () => a.editSection(c) }];
      if (!isPhases(c)) {
        items.push({ label: "Move Section Up", run: () => a.moveSection(c, -1) });
        items.push({ label: "Move Section Down", run: () => a.moveSection(c, 1) });
        items.push({ label: "Delete Section", run: () => a.deleteSection(c) });
      }
      return items;
    }
    default: {
      const cluster = target.cluster ?? "";
      const date = target.date ?? "";
      return [
        { label: "New Action", run: () => a.newAction(cluster) },
        { label: "New Milestone", run: () => a.newMilestone(cluster, date) },
        { label: "New Section", run: () => a.newSection() },
        { label: "Add Role", run: () => a.addRole() },
      ];
    }
  }
}
