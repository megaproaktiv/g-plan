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
export async function syncAfterChat(d) {
    d.setProject(await d.getProject());
    try {
        const m = await d.mode();
        if (m)
            d.applyMode(m);
    }
    catch {
        // the mode sync is best-effort; the project is still drawn below
    }
    d.refresh();
    try {
        const sel = await d.selection();
        if (sel.kind === "action")
            d.selectAction(sel.id);
        else if (sel.kind === "section")
            d.selectSection(sel.id);
        else if (sel.kind === "milestone")
            d.selectMilestone(sel.id);
    }
    catch {
        // the selection is optional: the project is already drawn
    }
}
