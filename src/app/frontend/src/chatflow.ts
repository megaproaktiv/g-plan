// What the UI does after a command-bar command ran in the backend: fetch the
// changed project, DRAW it, then apply the selection the command made.
import type { Project } from "./types";

export interface ChatSyncDeps {
  getProject(): Promise<Project>;
  setProject(p: Project): void;
  /** re-renders the canvas, the title and the side panel from the current project */
  refresh(): void;
  selection(): Promise<{ kind: string; id: string }>;
  selectAction(id: string): void;
  selectSection(id: string): void;
  selectMilestone(id: string): void;
  /** the central-object mode the backend is now in (action|role|section|milestone) */
  mode(): Promise<string>;
  /** reflect the backend mode in the UI (selector, canvas, currentMode) without
   *  calling backend.setMode (which would reset the current object) */
  applyMode(m: string): void;
}

/**
 * syncAfterChat shows the result of a command. It FIRST pulls the mode the
 * command may have changed (e.g. `mode section`, or a bare `add` that keeps
 * section mode) and reflects it in the UI, so the mode selector and status line
 * are never stale — otherwise the user cannot tell they are in section mode and
 * their bare `add`s are creating sections, not actions. THEN it refreshes: the
 * refresh MUST come after the mode sync (the status line reads the mode) and
 * before the select* calls (which only move the selection border and render the
 * side panel, they do not hand the new project to the canvas; without the
 * refresh a newly added action stayed invisible until something else redrew it).
 */
export async function syncAfterChat(d: ChatSyncDeps): Promise<void> {
  d.setProject(await d.getProject());
  try {
    const m = await d.mode();
    if (m) d.applyMode(m);
  } catch {
    // the mode sync is best-effort; the project is still drawn below
  }
  d.refresh();
  try {
    const sel = await d.selection();
    if (sel.kind === "action") d.selectAction(sel.id);
    else if (sel.kind === "section") d.selectSection(sel.id);
    else if (sel.kind === "milestone") d.selectMilestone(sel.id);
  } catch {
    // the selection is optional: the project is already drawn
  }
}
