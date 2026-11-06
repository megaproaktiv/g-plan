const isPhases = (c) => c.toLowerCase() === "phases";
export function contextMenuItems(target, a) {
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
            const items = [{ label: "Edit Section", run: () => a.editSection(c) }];
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
