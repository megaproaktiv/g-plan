# Test Case TC-001 — pt-plan full regression

Manual regression checklist. Work top to bottom. For each step, perform the
action and confirm the expected result, then tick **PASS** or **FAIL** (add a
note after `FAIL:`). A process passes only when every one of its steps is PASS.

- App under test: `src/app` Wails v3 desktop app
- Start it with: `cd src/app && wails3 dev` (live) — the window must open and
  show an empty **Untitled** plan.
- Sample data for load tests: `src/app/testdata/plan.mmd`,
  `src/app/testdata/wp4-email.mmd`, `src/app/testdata/work-package-plan.mmd`.
- Unit (`day`/`hour`) and `h/day` live in the toolbar; the chat/command bar is
  at the bottom; the roles column is on the right.

## Data

Tester:
Date:

# Format

## Checks

- [ ] **S0.1 Launch** Testactiondescription
  - [ ] PASS
  - [ ] FAIL:

1) Checkbox for "done Test"
2) Pass
3) Fail and reason. Also write FIX CHANGE ADD for implementation commands

## Test Role Names

- Project Manager
- Architect
- Engineer
- Developer
- Tester

---
# Testcase

## P0 — Launch & smoke

- [ ] **P0.1 Launch** Run `wails3 dev`. Expected: window opens, toolbar visible (Project…, Open, Save, Export PDF report, Link (L), Unit, h/day, Zoom, Mode); canvas shows an empty plan with a **Start** and **End** milestone marker.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P0.2 No console error** The canvas area does NOT show a red "Frontend error / Unhandled rejection / Backend not connected" box.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P0.3 Project summary line** The toolbar status reads `Untitled · <start>–<end> · Total 0 days`.
  - [ ] PASS
  - [ ] FAIL:

---

## P1 — Project configuration & details

- [ ] **P1.1 Open project dialog** Click **Project…**. Expected: a dialog with Name (max 16), Description, Start date, End date, and a blue **Apply** button.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P1.2 Set name** Type a name > 16 chars (input must refuse beyond 16 chars). Enter a valid name, click **Apply**. Expected: toolbar title and window title update to the name.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P1.3 Set description** Enter a description, **Apply**, reopen **Project…**. Expected: the description is still there.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P1.4 Set range** Set Start and End dates, **Apply**. Expected: the canvas Start/End milestone markers move to those dates; the status line shows the new range.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P1.5 Unit = days, h/day** Toolbar Unit = `days`, h/day = `8`. Expected: estimations display in days.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P1.6 Switch to hours** Set Unit = `hours`. Expected: estimation labels now interpret as hours (no crash, values reformat).
  - [ ] PASS
  - [ ] FAIL:

---

## P2 — Actions: create, move, resize (Epic 1)

- [ ] **P2.1 Add via `+` key** With the canvas focused (not an input), press `+`. Expected: a new action "New action" appears and its inline editor opens on the right with the **name field focused**.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P2.2 Add via right-click** Right-click empty canvas → **Add action**. Expected: a new action is created and selected (editor opens).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P2.3 Name has no WP prefix** Add an action named `demo` (edit the name in the editor, Apply). Expected: the bar label reads `demo` (NOT `demo0`, `WP_DEMO`, or `demo 0`).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P2.4 Drag move in time** Drag an action bar left/right. Expected: it moves in time, snapping to the zoom granularity; it stays in its lane.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P2.5 Resize** Drag the left or right edge handle. Expected: start or end changes; the bar lengthens/shortens.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P2.6 Dates editor** Double-click the action; in the editor set Start and End, **Apply**. Expected: the bar matches the new dates.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P2.7 Delete** In the editor, click **Delete**. Expected: the action and its bar disappear; remaining action numbers (#n) stay dense.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P2.8 Edit- Readonly** In the canvas, click an action. This now is selected. In the right window, the view for the action is shown. Its it not changeable. So apply, delete are deactivated and light grey. A slide button "Edit on/off" is show
  - [ ] PASS
  - [ ] FAIL: 
- [ ] **P2.9 Edit** Open Edit for an action, Toggle Edit. The Fileds are now editable, Apply, delete are activated. 
  - [ ] PASS
  - [ ] FAIL: 
---

## P3 — Estimation & total

- [ ] **P3.1 Estimation flows to total** Double-click a detail-less action, set Estimation = `5`, **Apply**. Expected: the bar shows `name 5`; the toolbar **Total** increases by 5 (the action became fixed-mode automatically).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P3.2 Whole-unit rule (days)** In day mode, set an action estimation to `1,25`. Expected: rejected or snapped — level-1 must be a whole day (error message or whole value).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P3.3 Two actions sum** Add a second action with estimation `3`. Expected: Total = `8`.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P3.4 Total is calculated** Confirm there is no field to type the project total directly; it only reflects the sum of actions.
  - [ ] PASS
  - [ ] FAIL:

---

## P4 — Sections / swimlanes

- [ ] **P4.1 Add section (mode)** In the command bar type `add section Design`. Expected: a new empty swimlane labeled **Design** appears on the left.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P4.2 Add section (right-click)** Right-click canvas → **Add section**. Expected: another swimlane appears.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P4.3 New action lands in current section** With a current section set, add an action. Expected: it appears inside that swimlane.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P4.3a Add action follows the newly selected section** Select **Section 1** and add an action (e.g. `add Alpha`). Expected: the action is created in Section 1. Then select **Section 2** and type `add action` (or use **Add action**). Expected: the new action is created in **Section 2** (the currently selected section), NOT in Section 1. Changing the current section must re-target where new actions land.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P4.4 Drag between swimlanes** Drag an action vertically onto another lane. Expected: while dragging, the target lane is highlighted (dashed); on drop the action moves into that lane.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P4.5 Horizontal drag keeps lane** Drag an action only left/right. Expected: it stays in the same swimlane.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P4.6 Select section** Click a lane band. Expected: bottom-right selection panel shows "Section", action count, a Name field, color picker.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P4.7 Rename section** In that panel, change the Name and click **Rename**. Expected: the lane label updates immediately and the panel stays open showing the new name.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P4.8 Section color** Pick a color in the panel. Expected: the lane band takes that color; **default** clears it.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P4.9 Section ordering** Create to sections, Section 1 should be top, Section 2 below
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P4.10 Section Up Down Buttons** Edit a section. If the section is the top section, then the move up button is disabled. If the section is the lowest section, then the move down button is disabled. Note that this is logical: if the phases section is disabled, work package 1 is at the top. If it's enabled, it's at the top. If it's rendered second 
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P4.11 Section Edit color picker** Edit a section. The color picker is a square box, so width and height are the same
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P4.12 Delete section (with confirm)** Edit a section. There is a **Delete section** button. Clicking it asks **"Are you sure?"** (a confirm dialog). Cancelling keeps the section; confirming deletes it (its actions move to the default lane) and the deletion is undoable with Cmd-Z. The fixed **Phases** lane has no Delete button.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P4.13 Action move up/down within a section** Add action **PM**, then add action **email** (both in the same section, so PM = n.1, email = n.2). Select **email** and click **▲ up**. Expected: email moves above PM (email = n.1, PM = n.2). **▼ down** moves it back. The top action's up and the bottom action's down are no-ops. Moving stays within the action's own section.
  - [ ] PASS
  - [ ] FAIL:
---

## P5 — Drill-in / detail packages (Epic 2)

- [ ] **P5.1 Open details** Double-click an action → editor → **Details…**. Expected: the detail table modal opens.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P5.2 Add detail row** Click **+ detail**. Expected: a new editable row ("New detail") appears (no prompt popup).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P5.3 Edit detail** Set a detail's Title, Description, Est, Role (from the role dropdown), click **save**. Expected: values persist in the row.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P5.4 Derived roll-up** Set the action mode to `derived`; add two details with Est `2` and `3`. Expected: the action's estimation shows `5`.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P5.5 Detail bars visible on canvas** Close the modal. Expected: detail bars render as indented sub-rows under the action (colored by role).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P5.6 Remove detail** Reopen details, click `×` on a row. Expected: the row and its sub-row bar disappear.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P5.7 Link detail packages graphically** With detail bars visible on the canvas (P5.5), create a graphical link between two detail packages — hover a detail sub-row to reveal its anchor dots and drag from one detail bar to another (or use Link mode `L`: click the source detail, then the target detail). Expected: a red routing line follows the cursor and, on drop, an arrow connects the two detail packages (not their parent actions). The link is between the detail bars themselves and the arrow endpoints attach to the detail sub-rows.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P5.8 Detail link persists and lists** Create a link between two detail packages, then confirm it appears in the right-side dependency table with the correct source/target detail and survives a save/reload (P13.2).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P5.9 Estimate a sub-work-package in the detail view** Open the detail view of an action and set the **Est** on a detail row (sub-work-package / sub-action) to a value, e.g. `4`, then **save**. Expected: the estimation is accepted and persists on that detail row, its sub-row bar sizes to the estimate, and in `derived` mode the estimate rolls up into the parent action's total. Estimation must work on sub-work-packages in the detail view, not only on top-level actions.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P5.10 Drill-down cursor** In the main view, hover the mouse over an action's **drill-down icon**. Expected: the mouse cursor changes to a drill-down icon (a drill-down affordance), indicating the drill-down is clickable; moving off the icon restores the normal cursor.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P5.11 Drill-down icon opens only the drill-down (bug fix)** In the main view, click the action's **drill-down icon**. Expected: ONLY the drill-down opens (the level-2 drill-in view for that action). The detail table/editor must NOT also appear. Regression guard: previously clicking the drill-down icon incorrectly opened the detail as well.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P5.12 Click a detail bar shows its editor (no crash)** With **Show Details** on, single-click a detail (sub-action) bar on the main canvas. Expected: the right pane shows that detail's editor (Title, Description, Effort, Role, dates, Save/Delete). It must NOT throw a frontend error (regression guard: the old `#selpanel` path threw `null is not an object (classList)`).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P5.13 Detail links shown on the main view** Create a link between two detail sub-actions (in the drill plane). Turn **Show Details** on in the Project dialog. Expected: the detail→detail link is drawn as an arrow on the main canvas between the two detail sub-rows (not only inside the drill plane).
  - [ ] PASS
  - [ ] FAIL:

---

## P6 — Roles (Epic 3)

- [ ] **P6.1 Add role focuses name** Click **+ role** in the roles column. Expected: a role is added and its editor opens with the **name field focused**.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P6.2 Edit role** Click a role name → editor; change Name + Description, **Save**. Expected: the role row shows the new name.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P6.3 Delete role** Open a role editor, **Delete**. Expected: the role disappears.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P6.4 Up to 3 roles on an action** Double-click an action; in the editor set up to three roles each with a percent; **Apply**. Expected: accepted; a 4th role is not allowed.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P6.4a Same role twice, each capped at 100%** On an action, add the **same role twice** (e.g. two `Developer` assignments — representing two developers). Expected: duplicate assignments of one role are allowed (both appear). Each individual assignment is capped at `100%` — entering more than 100% for a single assignment is rejected or clamped to 100%. (Two Developer assignments at 100% each are fine; a single assignment above 100% is not.)
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P6.4b Two different roles at 100% give no warning** Edit an action and set its estimation to `20` days. Assign a **Developer** role at `100%` and an **Architect** role at `100%`, then **Apply**. Expected: both assignments are accepted and **no warning** is shown — different roles each at 100% on the same action is valid and must not trigger an over-allocation/over-100% warning.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P6.4c 100% checked per role line only** On one action, add several role lines whose percentages together far exceed 100% (e.g. Developer 100%, Architect 100%, Tester 100%). Expected: no warning is raised from summing across lines; the only constraint is that each single line is ≤ 100% (R-6.8). A warning MUST appear only if an individual line is set above 100%, not because the lines add up to more than 100%.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P6.4d Add Role control** In the action editor, locate the **Add Role** control near the role-assignment area and click it. Expected: a new empty role-assignment line is added that you can set to a role + percent (R-6.9).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P6.4e Remove additional roles with ×** Add role lines beyond three. Expected: each line beyond the first three shows a small **×** control; clicking it removes that assignment. Add and then delete additional roles above 5 (grow past five lines, then remove them) — the count must grow and shrink again correctly (R-6.9).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P6.5 Role load** Confirm each role row shows a load value (`<n> days/hours`) derived from its assignments.
  - [ ] PASS
  - [ ] FAIL:

---

## P7 — Role filter

- [ ] **P7.1 All roles default** The **All roles** checkbox is checked; all actions are shown.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P7.2 Filter shows only assigned actions** Click **filter** on a role (or uncheck All roles and check one). Expected: the canvas shows only actions that have that role attached; actions with no matching role are hidden.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P7.3 Clear filter** Re-check **All roles**. Expected: all actions show again.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P7.4 Multi-role filter via checkboxes** With three roles each assigned to a different action, use the checkboxes left of the roles to select **more than one** role (e.g. keep Dev and QA checked, uncheck PM). Expected: the canvas shows the actions of ALL checked roles at once (DevTask and QATask visible, PMTask hidden). Clicking a single role's **filter** button narrows to just that one role.
  - [ ] PASS
  - [ ] FAIL:

---

## P8 — Dependencies / links

- [ ] **P8.1 Link via anchors** Hover an action to reveal 4 anchor dots (l/r/t/b). Drag from one anchor to another action. Expected: a red routing line follows the cursor; on drop an arrow connects the two.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P8.2 Link mode (L)** Press `L` (or Link (L) button), click a source action, then a target. Expected: an arrow is created; link mode exits.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P8.3 Chat link by number** In the command bar type `link #1 #2`. Expected: action #1 is linked to #2 (arrow appears).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P8.4 Arrow does not cross a bar** For two actions that immediately follow each other, create a link. Expected: the arrow routes around/between the bars and does not pass through the target action body.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P8.5 Anchors honored** Link from the **bottom** anchor of one action. Expected: the arrow leaves the bottom edge (not the right edge).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P8.6 Link appears accurately in the right table** Create a link from one action to another. Expected: the right-side table lists the dependency with the correct source and target (matching the actions you linked, by name or #number) and the correct direction.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P8.7 Delete link from the table** In the right-side table, click the link's delete control. Expected: the link row is removed from the table and the corresponding arrow disappears from the canvas.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P8.8 Start-start (SS) dependency** Link from action A's **left (start)** anchor to action B's **left (start)** anchor. Expected: a **start-start (SS)** dependency is created — the right-side table shows the link type as `SS`. The rule checks that both actions **begin at the same time**: if A's start differs from B's start, the dependency is flagged/enforced (e.g. moving A's start drags B's start to match, or an SS violation is reported). When A.start == B.start the SS link is satisfied with no warning.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P8.9 Finish-finish (FF / end-end) dependency** Link from action A's **right (end)** anchor to action B's **right (end)** anchor. Expected: an **end-end / finish-finish (FF)** dependency is created — the right-side table shows the link type as `FF`. The rule checks that both actions **end at the same time**: if A's end differs from B's end, the dependency is flagged/enforced (e.g. moving A's end drags B's end to match, or an FF violation is reported). When A.end == B.end the FF link is satisfied with no warning.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P8.10 Finish-start (FS) dependency, middle-to-middle, min one unit** Link from the **middle** of action A to the **middle** of action B. Expected: a **finish-start (FS)** dependency is created (shown as `FS` in the table). The rule checks that **B starts after A ends**, with a minimum gap of **at least one unit** — one day in day mode, one hour in hour mode (so `B.start >= A.end + 1 unit`). If B starts on or before A's end (gap < 1 unit), the dependency is flagged/enforced (B is pushed to `A.end + 1 unit`, or an FS violation is reported). When B starts at least one unit after A's end the FS link is satisfied with no warning.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P8.11 Middle-to-middle = overlap (OV)** Link from the **middle** of action A to the **middle** of action B. Expected: an **overlap (OV)** dependency — the two actions MUST intersect in time. It is flagged (red) ONLY when they do NOT overlap (one entirely before the other). Overlapping (or touching) actions raise no warning. (This replaces the old middle=FS behavior: middle-to-middle means "must overlap", nothing else.)
  - [ ] PASS
  - [ ] FAIL:

---

## P8b — Undo (Cmd-Z)

- [ ] **P8b.1 Undo delete link** Create a link between two actions, then delete it (from the table or the canvas). Press **Cmd-Z**. Expected: the link is restored — the arrow reappears on the canvas and the row returns to the right-side table, with the same source and target.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P8b.2 Undo delete action** Delete an action (editor **Delete** or `×`). Press **Cmd-Z**. Expected: the action is restored with its original name, dates, section, estimation, roles, and any links it had.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P8b.3 Undo delete section** Delete a section/swimlane. Press **Cmd-Z**. Expected: the section is restored at its original position, with its name, color, and the actions it contained.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P8b.4 Undo is multi-step** Delete a link, then an action, then a section (in that order). Press **Cmd-Z** three times. Expected: each press undoes the most recent deletion in reverse order (section, then action, then link), each fully restored.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P8b.5 Undo add section** Add a section (e.g. `add Program Management` in section mode, or Add section). Press **Cmd-Z**. Expected: the added section is removed again.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P8b.6 Undo add action** Add an action. Press **Cmd-Z**. Expected: the added action is removed again.
  - [ ] PASS
  - [ ] FAIL:

---

## P9 — Summary (Summiervorgang)

- [ ] **P9.1 Toggle summary** Double-click an action; tick **summary**, **Apply**. Expected: the bar changes to a rollup graphic (thin spine with triangular end caps), distinct from a normal bar.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P9.2 Summary has duration not estimation** In the editor for a summary, Expected: no estimation field; duration is set via Start/End dates.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P9.3 Summary whole-block click area** A summary bar is a thin spine, which is hard to click. Expected: clicking anywhere in the summary's whole block (the row span across the bar, including the label area) selects it — not only the thin spine. Double-click opens its editor.
  - [ ] PASS
  - [ ] FAIL:

---

## P10 — Chat / command bubble (Epic 4)

- [ ] **P10.1 Help** Click **help** (or type `help`). Expected: a panel lists all commands, including the current mode.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P10.2 Add action** Type `add action Build` (any mode). Expected: an action "Build" is created. A bare `add Build` is refused with a hint to the full form.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P10.3 Add after + link** Type `add action Test after #<n>` (n = an existing action number). Expected: a new action starts at that action's end and is FS-linked to it (arrow appears).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P10.4 Relative move** Type `use #<n>` then `move 3d`. Expected: the current action shifts 3 days later; `move -1w` shifts a week earlier.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P10.5 Lengthen / shorten** Type `lengthen 2d` then `shorten 1w`. Expected: the current action's end changes accordingly.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P10.6 Assign** Type `assign <RoleName> to #<n> 50%`. Expected: that role gets a 50% assignment on the action (visible in role load / filter).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P10.6b Assign by role number** Roles have numeric ids (1, 2, 3…). Type `assign 1 to 1.1` → role #1 is assigned to WP 1.1 (a bare `assign 1 to 1.1` uses the default percent; `assign 1 to 1.1 100` sets 100%). The role number must NOT be mistaken for the percent.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P10.6a Estimate by action ID** In the command bar type `est 1.1 10` (where `1.1` is an existing action ID). Expected: the command resolves action `1.1` by its ID and sets its estimation to `10` days — the bar label updates (e.g. `name 10`) and the toolbar **Total** reflects the new value. An unknown ID (e.g. `est 9.9 10`) is rejected with an error and no action is changed.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P10.7 Command history** Press Up/Down in the command bar. Expected: it cycles through previously entered commands.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P10.8 Assign works in any mode, bare percent** Switch Mode = `role` (or `section`). Type `assign Architect to 1.1 100`. Expected: the command is accepted in any mode (assign is a global verb, not mode-gated) and the role is assigned at **100%** — a bare number (no `%`) is read as the percent, not defaulted to 50%.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P10.9 Move after / before** In action mode, type `move 1.2 after 1.1`. Expected: action 1.2 is placed to start one unit after 1.1 ends (duration preserved). Then `move 1.2 before 1.1`. Expected: 1.2 ends one unit before 1.1 starts.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P10.10 Move to a date** Type `move 1.2 to 2026-06-10`. Expected: action 1.2 moves so it starts on 2026-06-10, keeping its duration.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P10.11 Move directional delta** Type `move 1.1 10d right` (or `future`). Expected: 1.1 shifts 10 days later. Then `move 1.1 10d left` (or `past`). Expected: it shifts 10 days earlier.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P10.12 Chat commands on detail sub-actions** Drill into an action that has details (so they get WBS ids `n.k.m`). Type `est 1.1.1 3` — the detail sub-action's effort becomes 3. Type `link 1.1.1 1.1.2` — the two detail sub-actions are linked. Detail sub-actions are actions too, so `add`/`est`/`move`/`resize`/`link`/`assign` accept them.
  - [ ] PASS
  - [ ] FAIL:

---

## P11 — Zoom

- [ ] **P11.1 Dropdown zoom** Change Zoom to `days`. Expected: the chart reloads immediately showing per-day gridlines with day numbers; Mondays emphasized.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P11.2 Weeks** Zoom = `weeks`. Expected: per-week gridlines labeled `d.m`.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P11.3 Months / year** Zoom = `months`, then `year`. Expected: coarser gridlines; cannot zoom finer than days or coarser than year.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P11.4 Pinch zoom** Pinch the trackpad (or Ctrl/Cmd + mouse wheel) over the canvas. Expected: the zoom level steps in/out and the Zoom dropdown syncs.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P11.5 Zoom Window (fit to view)** Zoom in (e.g. set Zoom = `days` or pinch in) until the plan no longer fits and some actions/milestones are off-screen. Then choose **Window** from the Zoom control (**Zoom Window**). Expected: the view zooms back so the entire plan — from the first to the last element (Start/End milestones and all actions) — fits inside the visible window with everything in view and no content clipped off either edge.
  - [ ] PASS
  - [ ] FAIL:

---

## P12 — Selection detail panel

- [ ] **P12.1 Select action** Single-click an action bar. Expected: the bottom-right panel shows the action's estimation, mode, dates, section, and description, with an **Edit** button.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P12.2 Edit from panel** Click **Edit**. Expected: the inline action editor opens.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P12.3 Close panel** Click `×`. Expected: the panel hides.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P12.4 Clicking an action in role mode shows the action edit form** Set Mode = `role` so the **role form** appears on the right. Then single-click an action bar on the canvas. Expected: the right panel switches away from the role form and shows the **action edit form** for the clicked action (its estimation, mode, dates, section, description). Clicking an action must surface the action form even while role mode is active.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P12.5 Section order up/down from the action panel** Click an action that belongs to a named section. Its panel shows **Action order ▲/▼** and, below it, **Section order ▲/▼**. The section-order buttons move the action's whole section up/down (same as the section panel's up/down). An action in the default (unnamed) lane shows no Section order row.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P12.6 No frontend error on detail/selection interactions** Clicking actions, sections, and detail bars does NOT surface a red frontend-error box (regression guard for `TypeError: null is not an object (classList)`).
  - [ ] PASS
  - [ ] FAIL:

---

## P13 — Save / Load (single JSON file)

- [ ] **P13.1 Save** Click **Save**. Expected: a native save dialog appears; choosing a path writes exactly ONE file `<name>.ptplan` (extension is `.ptplan`, NOT `.ptplan.json`; no sibling `.mmd`/`.roles.json`/`.md`). A confirmation shows the real saved path (no `:Users:` mangling).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P13.2 Reload persists everything** Click **Open**, pick the saved `.ptplan.json`. Expected: actions, sections, roles, role assignments, colors, dependencies, summary flags, descriptions, project range, AND the project **unit**, **hours/day**, and **zoom** all return.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P13.3 Config persisted (unit / h-day / zoom)** Set Unit = hours, h/day = 6, Zoom = weeks; Save; New; Open the file. Expected: unit=hours, h/day=6, and the canvas zoom is restored to weeks.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P13.4 New project (auto-saves old)** With a saved project open, click **New**. Expected: the old project is auto-saved to its existing path, and a fresh empty Untitled project opens. If the current project was never saved, you are asked to save it first.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P13.5 Save path shown in Project dialog** Open **Project…**. Expected: it shows "Saved to: <path>" (or "(not saved yet)"), i.e. the save directory/file of the single JSON.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P13.6 Legacy import** Opening an old three-file `.mmd` project still loads (migration); saving then writes the single `.ptplan.json`.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P13.7 Unsaved-changes prompt on Open/New** Make a change, then click **Open** or **New**. Expected: you are asked whether to save the current project first (OK = save, then proceed; Cancel = confirm discard or abort). With no unsaved changes, Open/New proceeds without asking.
  - [ ] PASS
  - [ ] FAIL:

---

## P18 — Right-pane view switcher (icons)

- [ ] **P18.1 Icons present, correct order** The right pane shows a row of view-mode icons at the top, in the order **action, role, section** (same order as the toolbar/chat mode). The current view is highlighted.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P18.2 Switch view in view mode** Not in edit mode, click the **role** icon → the role view appears; click **section** → section context; click **action** → the current action/section view. Switching updates the highlight and the status-line Mode.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P18.3 Disabled in edit mode** Open an action and click **Edit** (enter edit mode). Expected: the view-switcher icons are **disabled** (greyed, with a "leave edit to switch" hint). Leaving edit mode re-enables them.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P18.4 Object icon follows an action selection** In role view, click an action bar. Expected: the right-pane view switcher's **action** icon becomes highlighted (the object-type icon follows the selection) and the action panel shows.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P18.5 Selection persists across view switch** Select an action (border appears). Switch to role/section view. Expected: the action stays selected — its canvas border remains — while the panel reflects the chosen view.
  - [ ] PASS
  - [ ] FAIL:

---

## P19 — Selection & the `select` command

- [ ] **P19.1 Click selects with a border** Click an action bar → a dashed selection border is drawn around it. Click a section band → a border is drawn around the lane.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P19.2 `select` chat command** Type `select action 1.1` → action 1.1 is selected (border + action panel). Type `select section <name>` → that section is selected. (Roles get numeric ids so `assign 1 to 1.1` assigns role #1 to WP 1.1.)
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P19.3 Deselect** Press **Escape** (or Ctrl) → the selection border clears. Clicking empty canvas also deselects.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P19.4 Arrow-key navigation** Select an action. Press **→** to select the nearest action to the right, **←** left, **↑** the nearest above, **↓** below. The selection (and border) moves accordingly.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P19.5 Add action deselects first** Select an action, then add a new action (`+` key or right-click → New Action). Expected: the new action is created, selected, and its editor opens — the prior selection does NOT swallow it (regression: a stale selection made the new action "not found").
  - [ ] PASS
  - [ ] FAIL:

---

## P20 — Milestones

- [ ] **P20.1 Add milestone (mode / chat)** Type `add milestone Kickoff` (or `milestone Kickoff on <date> in <section>`). Expected: a milestone is created in the current section, drawn as a selectable **diamond ◆** in that lane.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P20.2 Shared WBS numbering** With an action 1.1 in a section, add a milestone in the same section. Expected: the milestone shares the section's n.k sequence — e.g. it becomes **1.2** (ordered by date alongside actions).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P20.3 Select + edit a milestone** Click the diamond → the right pane shows a milestone editor with **name**, **date**, and a **color** picker; the view switcher's milestone icon ◆ highlights. Changing name/color updates the diamond.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P20.4 Main level only** Milestones can be set in any section at the main level, but NOT inside a drill-down (level-3). 
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P20.5 Project Start/End protected** The project Start/End markers render across the chart and cannot be deleted (no Delete button / delete is rejected).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P20.6 Link an action end to a milestone** Select an action, press **L**, then click a milestone diamond (or drag from the action's right/end anchor to the milestone). Expected: a link is created from the action's **end** to the milestone (drawn as an arrow). It is flagged only if the milestone is before the action's end.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P20.7 Link a milestone to an action start** Select a milestone, press **L**, then click an action. Expected: a link from the milestone to the action's **start** (not its middle) is created.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P20.8 Milestone dialog section up/down** Open a milestone (in a section). Its editor has **Section order ▲/▼** that moves the milestone's whole section up/down (same as the action/section panels).
  - [ ] PASS
  - [ ] FAIL:

---

## P21 — Right-click context menu (by target)

- [ ] **P21.1 On an action** Right-click an action bar. Expected: the menu shows **Edit Action** and **Delete Action** only. Delete asks to confirm and is undoable (Cmd-Z).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P21.2 On a section label** Right-click the left label column of a lane. Expected: **Edit Section** and **Delete Section** (Delete confirms, undoable). The fixed Phases lane has no Delete.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P21.3 On a milestone** Right-click a milestone diamond. Expected: **Edit Milestone** and **Delete Milestone** (Delete confirms).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P21.4 On empty space** Right-click free space inside a section. Expected: **New Action** and **New Milestone** (plus New Section / Add Role) — created in that section. There is NO "Toggle link mode" entry.
  - [ ] PASS
  - [ ] FAIL:

---

## P14 — Rendering & PDF report

- [ ] **P14.1 Export PDF** Click **Export PDF report**. Expected: a native save dialog; the written PDF contains the rendered Gantt image on page 1 and per-role / work-package tables on later pages.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P14.2 PDF opens** Open the PDF. Expected: it is valid and shows the current plan plus people tables.
  - [ ] PASS
  - [ ] FAIL:

---

## P15 — Status line (mode + view)

- [ ] **P15.1 Mode shown in status line** The toolbar status line (right of the name/range/total) shows the current **chat mode** — `Mode: action|role|section`. Switching Mode updates it immediately.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P15.2 View (canvas zoom) shown in status line** The status line also shows the current **canvas view / zoom** — `View: days|weeks|months|year|window`. Changing the Zoom control updates it immediately.
  - [ ] PASS
  - [ ] FAIL:

---

## P16 — Project window & weekends

- [ ] **P16.1 Work package outside the project window** Move an action so it starts before the project Start or ends after the project End. Expected: the action is flagged (red border / HasWarning) and a warning reads `WP <id>: <range> is outside the project window <start>..<end>`. Bringing it back inside clears the flag.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P16.2 Weekend coloring** At a fine zoom (days/weeks or a fitted window), Saturdays and Sundays are drawn with a distinct band color in the date frame.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P16.3 Weekends carry zero work** The capacity calculation counts only Mon–Fri as working days; Saturdays and Sundays contribute zero work (verify via the role-fit line below).
  - [ ] PASS
  - [ ] FAIL:

---

## P17 — Role capacity (fit) line

Percentage on a role assignment is the fraction of the working day the role
spends on the action (NOT a share of the effort). A role at `p%` delivers
`p% × h/day` hours on every working day in the window.

- [ ] **P17.1 Read-only fit line** Give an action effort and assign a role. Below the roles a READ-ONLY line shows the assigned capacity vs the effort and whether it is covered, e.g. `Assigned capacity 5 of 5 days — effort is covered.`
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P17.2 Capacity covers effort (check 1)** With one role at `100%` over a window of N working days and effort = N days, the line reports **covered** (fits). Dropping the role to `50%` halves the capacity so it no longer covers the effort and the action is flagged.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P17.3 Effort fits the window (check 2)** If the effort exceeds what a single 100% role could do in the window (effort > h/day × workdays), the line reports the window is too short even at 100% and the action is flagged.
  - [ ] PASS
  - [ ] FAIL:

---

## P22 — Dependency relationship rules (7 kinds)

A = predecessor (link From), B = successor (link To). Mid(x) = start +
(end−start)/2. Touching endpoints satisfy the ≥ / ≤ rules (not a violation).
A violated link is drawn with a RED arrow and flagged in the right-pane Links
table.

- [ ] **P22.1 FS (Finish-to-Start)** Rule `Start(B) ≥ End(A)`. B starting exactly when A ends is OK; B starting before A ends is violated.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P22.2 SS (Start-to-Start)** Rule `Start(B) ≥ Start(A)`. B starting at/after A's start is OK; before is violated.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P22.3 FF (Finish-to-Finish)** Rule `End(B) ≥ End(A)`. B ending at/after A's end is OK; before is violated.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P22.4 SF (Start-to-Finish)** Rule `End(B) ≥ Start(A)`. B ending at/after A's start is OK; before is violated.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P22.5 MS (Middle-to-Start)** Rule `Start(B) ≥ Mid(A)`. B starting at/after A's midpoint is OK; before is violated.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P22.6 ME (Middle-to-End)** Rule `End(B) ≥ Mid(A)`. B ending at/after A's midpoint is OK; before is violated.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P22.7 MM (Middle-to-Middle)** Rule `Start(A) ≤ Mid(B) ≤ End(A)`. B's midpoint must fall within A's span; outside (before start or after end) is violated.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P22.8 Anchor → kind** Drawing a link from an end edge to a start edge gives FS; start/start = SS; end/end = FF; start→end = SF; from a middle (top/bottom) anchor to start/end/middle gives MS/ME/MM. A link touching a milestone is FS.
  - [ ] PASS
  - [ ] FAIL:

---

## P23 — GUI language (German / English)

The GUI language is set in Project settings and persists with the project. The
command bar / chat language is ALWAYS English and is never translated.

- [ ] **P23.1 Language pulldown** Project settings has a Language pulldown (English / Deutsch). Switching it updates the GUI.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P23.2 Number format** English uses a dot decimal and comma thousands (`1.5`, `2,500`); German uses a comma decimal and dot thousands (`1,5`, `2.500`).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P23.3 Date format** English shows ISO `2026-03-15`; German shows `15.03.2026`. The stored/transport format stays ISO.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P23.4 Link abbreviation follows language** The Links table kind column shows the language abbreviation: English FS/SS/FF/SF/MS/ME/MM, German EA/AA/EE/AE/MA/ME/MM.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P23.5 Persist** Saving and reopening a German project reopens it in German.
  - [ ] PASS
  - [ ] FAIL:

---

## P24 — Help (multiple pages, icon-chosen)

Help is reached from the `help` button and shows a small row of object-style
icon buttons; hovering an icon shows a tooltip; selecting one shows that help
page in the chosen GUI language.

- [ ] **P24.1 Icon buttons + tooltips** The help overlay has icon buttons (⌨ command help, ⇄ relationship help) with hover tooltips; the active page is highlighted.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P24.2 Command help is English** The command-help page lists the CLI commands in English regardless of the GUI language (the command language is never translated).
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P24.3 Relationship help table** The relationship-help page shows the 7 rules as a table (abbreviation, relationship name, rule, example) in the chosen language, using that language's abbreviations.
  - [ ] PASS
  - [ ] FAIL:
- [ ] **P24.4 Localized chrome** In German, the help tabs/columns/close button are localized (Befehlshilfe, Beziehungshilfe, Kürzel/Beziehung/Regel/Beispiel, Schließen).
  - [ ] PASS
  - [ ] FAIL:

---

## Result

- Date run: ____________________
- Build / commit: ____________________
- Overall: [ ] PASS   [ ] FAIL
- Failing steps / notes:
  -
