# pt-plan — Requirements

RFC-2119 keywords (MUST / SHOULD / MAY) are used normatively.

## 1. Scope and vision

pt-plan is a local, file-based, multi-project graphical plan editor. It renders
a project plan as a Gantt-style chart and lets the user edit it with the mouse,
drill into work packages to plan detail tasks, and manage per-project roles and
their assignments. It is, in effect, a small, focused MS-Project / Gantt editor.

## 2. File model

### 2.1 Single-file project

R-2.1.1 Each project MUST be stored as a single self-contained JSON file
`<project>.ptplan` (schema `pt-plan/project@2`).

R-2.1.2 The file MUST contain EVERYTHING needed to restore the project:
configuration (unit, hours/day, canvas zoom, view options, arrow style,
language, autosave and report settings), the graphic content (sections/clusters
and their order, work packages, their level-2 detail sub-packages, milestones,
phases, dependencies, dates), roles with their percentage assignments, section
colors, baselines, the monitoring configuration, the work-package IDs (§16)
and the task metadata (§17). The ONLY exception is the task description
content (Markdown text and its images), which lives in the work-package folder
next to the project file (§17.3).

R-2.1.3 The project's **unit**, **hours/day**, and **zoom** MUST be persisted in
the file and restored on reopen.

R-2.1.4 Saving MUST write this one file as the project's source of truth (no
sibling `.mmd` / `.roles.json`). Opening a legacy three-file project (`.mmd` +
`.roles.json`) MAY still be supported for migration, but saving converts it to
the single file.

R-2.1.5 The app MAY additionally write a read-only `<project>.md` projection
(for humans and LLMs) and export a rendered Gantt (PNG) and a PDF report. These
are OUTPUTS: they are never parsed back and are never the source of truth.

R-2.1.6 New optional configuration keys MUST have defaults so that files written
before the key existed open with the default, not a zero value.

R-2.1.7 **Saving MUST be atomic.** The project file MUST be written to a
temporary file in the same directory, flushed to disk and then moved over the
target in one rename, so that a crash, a full disk or a power cut leaves either
the complete old or the complete new file, never a truncated one. The previous
content SHOULD be kept as `<file>.bak` (the backup is not a rotated version of
R-14.4.3). A failed save MUST leave the existing file untouched and MUST be
reported to the user. A symbolic link at the target MUST stay a link. The
other files the app writes (`.md` projection, settings) are written the same
way. [IMPL]

R-2.1.8 **Dates are calendar dates.** A date MUST NOT depend on the time zone
of the machine: it is a day (`YYYY-MM-DD`) without time of day, and the same
file MUST show the same days on every computer. [PART: work-package, detail
and dependency dates are days; the project Start/End milestones are still
stored with a time of day and a UTC offset, and the default start of a new
action is computed from the current UTC day. TODO: store plain dates.]

R-2.1.9 **Files from newer versions.** The app MUST NOT silently lose data of
a file written by a newer version. A file whose schema is newer than the app
knows MUST be refused or opened read-only with a clear message; unknown keys
of a known schema SHOULD be kept on save. [TODO: today only the schema string
is checked.]

### 2.2 gmaid/Gmate compatibility (legacy import/export only)

R-2.2.1 The serializer MUST emit only syntax gmaid accepts: `gantt` header,
`title`, `dateFormat`, `axisFormat`, `tickInterval`, `section`, task lines
`Label :[tags,] id, start, end|dur`, the `milestone`/`crit`/`active`/`done`
tags, the `summary` tag, the `&` same-line prefix, `<br/>` line breaks, and the
`link <from> FS|SS|FF|SF <to>` directive.

R-2.2.2 Every task the app creates MUST carry a unique, stable task **id** (last
`:id` field) so CSS id-selectors and dependency links remain valid.

## 3. Project configuration

R-3.1 A project MUST be configured as either **days** or **hours**, applied to
the whole project. There is no per-work-package unit override.

R-3.2 The configured unit MUST be stored in the project file (R-2.1.3) and
reflected in the generated `.md` projection. Switching the unit MUST NOT change
any stored effort value (see R-4.3); it only changes the display.

## 4. Units, estimation, and integer arithmetic

R-4.1 A work package (level 1) estimation MUST be a whole number of the project
unit (whole days or whole hours). Fractions are NOT allowed at level 1.

R-4.2 A detail package (level 2) estimation MAY carry up to two decimal places.

R-4.3 All efforts (estimated and real) MUST be stored as **integer minutes**,
independent of the display unit, to avoid floating-point error and so that
switching between days and hours never reinterprets a stored value. One display
unit is 60 minutes (hours) or 60 × hours/day minutes (days). All sums and
comparisons MUST be done in integer space; conversion to a decimal display (up
to two places) happens only at render time.

R-4.4 **Duration ≠ estimation.** A bar’s time span (start→end) is independent of
its estimation; a bar MAY run longer than its estimation implies.

## 5. Work packages and detail packages (two levels only)

R-5.1 There MUST be exactly two levels: work packages (level 1) and their detail
packages (level 2). No deeper nesting.

R-5.2 A work package MUST have: id, name, start, end (→ duration), an estimation,
and an optional cluster membership. It MAY have detail packages and role
assignments.

R-5.3 Work packages MAY be **clustered** (grouped). Clustering is a visual/
organizational grouping and MUST NOT change the estimation arithmetic.

R-5.4 **Estimation mode per work package.** Each work package MUST declare how
its estimation relates to its details, one of:
  (a) **fixed-at-level-1** — the level-1 estimation is authoritative and is
      *distributed* down to the detail packages; or
  (b) **derived-from-level-1** — the level-1 estimation is the *sum* of its
      detail-package estimations (roll-up).

R-5.5 The project total estimation MUST be **calculated** as the sum of all
work-package estimations. It MUST NOT be entered directly.

R-5.6 When a work package is `derived`, editing a detail estimate MUST update the
work-package total (and thus the project total). When `fixed`, the level-1 value
is held and the details MUST reconcile against it (the UI SHOULD surface any
over-/under-allocation).

## 6. Roles and assignments (people)

R-6.1 A project MUST have a set of roles. Each role MUST have a name and a
description.

R-6.2 A role MAY be assigned to one or more work packages. An assignment MUST
reference a work-package id and carry the role’s share as a **percentage** of
that work package (stored as an integer percent).

R-6.3 At level 1, roles are assigned to work packages by percentage only (not to
detail packages). Assignment to detail packages is out of scope for now.

R-6.4 An assignment's **consumed** effort MUST be derived, never entered: the
work package's real effort (R-13.5) scaled by the assignment percentage. It is
shown in the people/roles report. (A stored per-assignment `consumed` field is a
legacy leftover and is ignored.)

R-6.5 The roles view MUST let the user pick one role and see that role’s planned
load across the whole project timeline (per time bucket).

R-6.6 A role’s total effort MUST be derived (sum of percentage × work-package
estimation across its assignments), never entered directly.

R-6.7 A role MAY be assigned to the same work package more than once (e.g. two
`Developer` assignments representing two people). Each **individual assignment**
(one role line) MUST be capped at 100%; a single assignment above 100% MUST be
rejected or clamped to 100%.

R-6.8 The 100% limit MUST be checked **per role line only** — i.e. against the
percentage of a single assignment. The app MUST NOT sum percentages across
different role lines on a work package and warn when their total exceeds 100%;
multiple roles (or multiple instances of a role) each at 100% on the same work
package are valid and MUST NOT raise a warning.

R-6.9 The action editor MUST allow **adding** further role assignments via an
**Add Role** control near the role-assignment area, and MUST allow **removing**
assignments. A small **×** control MUST appear on each role line beyond the
first three (i.e. the 4th and later lines are individually removable). The
editor MUST support adding and then deleting additional roles above 5 so the
assignment count can grow and be reduced again.

R-6.10 A role assignment's **percentage** is the fraction of the **working day**
the role spends on the action — NOT a share of the effort. A role at `p%`
contributes `p% × HoursPerDay` hours on each working day (Mon–Fri) in the
action's window. Weekends (Saturday/Sunday) carry **zero** work.

R-6.11 The editor MUST show a READ-ONLY capacity line below the roles that
reports two checks: (1) whether the **assigned capacity** — the sum over roles
of `p% × HoursPerDay × working-days-in-window` — covers the action's effort;
and (2) whether the effort **fits the window** at all (effort ≤
`HoursPerDay × working-days-in-window`, i.e. the window is long enough even for
a single 100% role). An action failing either check MUST be flagged.

R-6.12 Every work package MUST lie within the project's overall Start..End
window. An action starting before the project Start or ending after the project
End MUST be flagged.

## 7. Graphical editing (Epic 1)

R-7.1 The app MUST render the plan on an interactive canvas: a time axis, section
lanes, work-package bars, milestones (diamonds), and dependency arrows.

R-7.2 The user MUST be able, with the mouse, to: move a work package earlier or
later in time (drag horizontally); move it to another section (drag vertically);
and lengthen or shorten it (drag an edge handle).

R-7.3 Edits MUST snap to the current zoom granularity and MUST keep whole-unit
level-1 estimation invariants (R-4.1) where estimation is coupled to duration.

R-7.4 The user MUST be able to **zoom** the time axis across four levels only:
**days ↔ weeks ↔ months ↔ year**. Zooming further in than days or further out
than a year MUST NOT be possible.

R-7.5 The live canvas SHOULD redraw smoothly during drag/resize (hand-drawn from
the in-memory model, not via an external renderer).

## 8. Drill-in / third dimension (Epic 2)

R-8.1 The user MUST be able to drill **into** a work package to a second plane
that shows and edits its detail packages on their own time axis.

R-8.2 Inside a work package, the user MAY add, move, resize, and remove detail
packages, subject to the two-level limit (R-5.1) and the estimation mode (R-5.4).

R-8.3 Role set is fixed at the work-package level: inside a work package the user
MUST NOT invent new roles; only the roles already assigned to that work package
(or the project roles) apply.

R-8.4 The user MUST be able to drill back **up** to the plan plane.

## 9. Rendering strategy

R-9.1 The interactive canvas MUST render directly from the in-memory model
(low-detail, fast).

R-9.2 Exported graphics (PNG, and the Gantt pages of the PDF report) MUST be
rendered from the live canvas, so the export matches the screen (notes, arrow
style, colors). Editor-only chrome (selection borders, anchors, red warning
borders) MUST NOT appear in exports.

R-9.3 The PDF report MUST be able to contain both a compacted view (no detail
bars) and a detailed view (detail bars shown), each on its own page. Which of
them are printed is chosen in the report dialog (§15).

R-9.4 The legacy gmaid / `.mmd` rendering path (gmaid embedded as a Go package,
no external binary) MAY be kept for legacy export only.

## 10. Chat text interface (Epic 4, optional)

R-10.1 The app MAY provide a chat/command bubble accepting base commands that
mutate the plan (move/resize/add/remove/assign), mirroring the My-Memory-Master
command bubble. The command forms are specified in §14.5.

R-10.2 The chat interface, if present, MUST operate on the same in-memory model
and MUST produce the same project file and `.md` projection as the graphical
editor. It exists partly as an accessibility alternative to the mouse.

## 11. Persistence and multi-project

R-11.1 The app MUST open and save projects as the single project file (§2.1) at
a user-chosen path. Multiple projects MUST be independent.

R-11.2 On save, the app MUST write the project file and regenerate the
read-only `.md` projection (R-2.1.5).

R-11.3 Saving MUST be consistency-checked: a single role assignment above 100%
MUST be reported (per line only, R-6.8; percentages are NOT summed across role
lines), and the derived project total MUST equal the sum of work-package
estimations.

## 12. Non-functional

R-12.1 Core logic (model, file I/O, serialization, md projection, arithmetic)
MUST live in a pure-Go package with no Wails dependency, and MUST be unit-tested.

R-12.2 The project MUST be driven by a `Taskfile.yaml`; documentation file names
MUST be lower-case (`readme.md`, `requirements.md`).

R-12.3 Estimation arithmetic MUST be exact (integer-based, R-4.3); no test may
rely on float equality.

R-12.4 Each epic MUST be developed on its own feature branch and merged into
`main` only when complete.

R-12.5 **Target platforms.** The app MUST run on macOS (tested) and SHOULD run
on Windows. Windows is built (cross-compiled, no CGo) but NOT yet tested; it
MUST NOT be described as supported until it has been tested. Linux is out of
scope for now.
R-12.6 **Distribution.** The macOS app is ad-hoc signed, not notarized, so
Gatekeeper warns on other computers. Notarization with a Developer ID, and a
signed Windows installer, are [TODO] before the app is handed to third parties.
R-12.7 The project is published under the MIT license (`LICENSE`).

## 13. Advanced planning features (MS-Project-grade)

These extend the core editor toward a professional scheduling tool. Status is
tracked per item: [IMPL] implemented, [PART] partial, [TODO] not yet.

### 13.1 Baselines (Basislinien) — [IMPL core]
R-13.1.1 The app MUST support snapshotting (freezing) the current schedule and
effort plan as a named **baseline** for later historical comparison.
R-13.1.2 A baseline MUST capture, per work package and detail, the start, end
and estimation at capture time, plus a label and timestamp.
R-13.1.3 Baselines MUST persist in the project file and MUST NOT change when
the live plan changes.
R-13.1.4 The UI SHOULD allow comparing the live plan against a chosen baseline
(variance in start/end/effort). [PART: variance list in the Baselines dialog;
baseline bars on the Gantt are TODO.]

### 13.2 WBS-code numbering (WBS-Code) — [IMPL]
R-13.2.1 The app MUST assign a dynamic WBS code per hierarchy level
(e.g. `1`, `1.1`, `1.1.2`): work package = `n`, action = `n.k`, drill-down
detail = `n.k.m`. Once Epic 6 is implemented, `n` is the work-package ID
(R-16.2.1), not the section's position.
R-13.2.2 WBS codes MUST cascade/recompute automatically when tasks are moved,
reordered, added or deleted (they are display-only, dense, recomputed each
render). Superseded by Epic 6 (§16.3): once implemented, work-package and
action numbers are stored and stable, and only change by manual edit.

### 13.3 Task types (Aufgaben-Berechnungstypen) — [TODO]
R-13.3.1 Each work package MAY declare a calculation type governing how
effort/duration/units interact on change:
  - **Fixed Work**: total effort (hours) is rigid; duration adapts to resource
    changes.
  - **Fixed Duration**: the time window is rigid; daily resource load adapts.
  - **Fixed Units**: daily resource capacity (e.g. 8h/day) is rigid.
R-13.3.2 Editing one of {work, duration, units} MUST recompute the others per
the task type, using integer math.

### 13.4 Resource utilization view (Auslastungsansicht) — [PART]
R-13.4.1 The app SHOULD show planned capacity vs actual effort per role (and
team), per time bucket, to surface overload. (Per-month role load exists;
overload highlighting + team rollups are TODO.)

### 13.5 Progress & remaining effort (Fortschritt & Restaufwand) — [IMPL core]
R-13.5.1 Each action MUST carry an estimated effort and a **real effort**
(entered, or synced per §13.8). The app MUST compute the progress percentage
from real ÷ estimated effort and show a progress marker on the bar when
**Show Progress** is on. Detailed spec: `.kiro/specs/progress/`.
R-13.5.2 The monitoring result MUST show the **remaining effort** = estimated
(tracked) − actual, per work package, per tracked action and for the project,
as a signed value (negative = over budget), and **Remaining %** = remaining as a
whole percent of the estimation (empty without an estimation). [IMPL in the
Monitor table.] "Duration complete" is defined in the glossary but not yet
displayed. [TODO]

### 13.6 Export functions — [PART]
R-13.6.1 The app MUST export a vector PDF status report (done). It SHOULD also
export tabular CSV and Excel. (CSV TODO; Excel TODO.)

### 13.7 Work contours (Arbeitskurven) — [PART]
R-13.7.1 A task MAY use a non-linear effort distribution (Flat, Front-Loaded,
Back-Loaded, Double-Peak) instead of a rigid per-day spread. [IMPL: contour per
action, drawn in the bar, used to place the progress marker.]
R-13.7.2 A contour of **None** marks an action that tracks no effort: its
effective estimated and real effort are 0, it shows no progress graphics, and it
MUST NOT carry an External ID (§13.8). [IMPL]
R-13.7.3 The capacity/load calculations (R-6.11, R-13.4) MUST honor the
contour. [TODO: role load still spreads effort evenly.]

### 13.8 Monitoring: sync real effort from external systems — [IMPL core]
R-13.8.1 The app MUST provide a driver interface for external time-tracking
systems. Drivers: Aerion [IMPL]; CSV [TODO].
R-13.8.2 An action MAY carry an **External ID** (max 16 characters, unique). A
booking is matched to actions by the External IDs found in its text
(case-insensitive, hyphens ignored, whole token, longest match wins); a booking naming several
IDs is split evenly between them. Unmatched bookings go to a configurable
**default action**.
R-13.8.3 A sync MUST write the matched effort into the real effort of the
actions (R-13.5). Real effort of a synced action MUST NOT be editable by hand.
The sync period is the project window, optionally extended by months before and
after.
R-13.8.4 Credentials MUST be stored outside the project file
(`~/.pt-pm/config.yaml`, mode 0600) and MUST never be returned to the UI.
R-13.8.2a Matching ignores case and hyphens: `WP1`, `wp1`, `WP-1` and `wp-1`
are the same External ID, but `WP 1` (with a space) is not. Two actions MUST
NOT carry IDs that differ only in case or hyphens.
R-13.8.2b The **Phases** lane is a view, not a work package: monitoring MUST
ignore it. Its actions need no External ID, do not appear in the Monitor page or
the result table, count in no total, cannot match bookings or be the default
action, and keep a manually entered real effort.
R-13.8.6 The Monitor page MUST offer an **Export** of the last result as
Markdown in two files, `<name>-compact.md` (work packages and the project sum)
and `<name>-detailed.md` (also the tracked actions). [IMPL]
R-13.8.5 Matching on booking attributes (not just text) and persisting the last
sync result are [TODO].

### 13.9 Implementation order
Done: Baselines (13.1), WBS codes (13.2), Progress (13.5), Work contours
(13.7, display), Monitoring (13.8, Aerion). Remaining: CSV export (13.6) and
overload view (13.4), then contour-aware load (13.7.3), then Task types (13.3).

### 13.10 Working calendar — [TODO]
R-13.10.1 The capacity checks (R-6.10, R-6.11) and the load calculations
(R-13.4) SHOULD use a project calendar with public holidays and other
non-working days. Today only Saturdays and Sundays count as non-working, so
the capacity of a week with a holiday is overstated.

## 14. Implemented editor features (documented after the fact)

These features exist in the app but had no requirement. They are recorded here
so the requirements describe the product. All are [IMPL] unless marked.

### 14.1 Dependencies
R-14.1.1 The app MUST support seven dependency types between actions, details
and linkable milestones: FS, SS, FF, SF, and the mid-point types MS
(Start(B) ≥ Mid(A)), ME (End(B) ≥ Mid(A)) and MM (Start(A) ≤ Mid(B) ≤ End(A);
`OV` is a legacy alias). Touching dates satisfy the rule.
R-14.1.2 A violated dependency MUST be drawn red and MUST flag the action. The
app MUST NOT move tasks automatically to resolve it (warn, don't reschedule).
R-14.1.3 Dependency arrows MAY be drawn as splines (default) or square lines,
with a per-link color.

### 14.2 Sections, phases and milestones
R-14.2.1 The section order MUST be stored explicitly, including empty sections,
and MUST be changeable (move up/down). A new section (menu, `add section`) MUST
be added at the END, i.e. it gets the next number: with WP-1 present the next
one is WP-2. Existing sections keep their order.
R-14.2.2 A locked **Phases** lane MUST always come first, without a number, and
MAY be hidden ("Show Phases").
R-14.2.3 Project Start and End are milestones that bound the visible window.
R-14.2.4 Actions and details MAY be marked as a **summary** (Sammelvorgang): a
rollup bar with a duration but no effort. Its label follows the text position
(R-14.3.1); because the bar is only a few pixels high there is no "inside":
`left` and `center` put the label above the bar, `up`/`down` above/below,
`outleft` before and `outright`/auto after the bar.

### 14.3 Display options
R-14.3.1 Per action and detail: bar color, font color (empty = automatic
contrast), label position, and an optional note (the description) shown as a
sticky note at one of nine positions.
R-14.3.2 Project-wide toggles: Show Details (with a per-action override to hide
its detail bars), Show Progress, Show Notes, today line, and a day/week/month
grid. The today line, grid and notes settings also apply to the report.
R-14.3.4 The **date row** (month, day and week labels) MUST stay at the top of
the visible area while the chart is scrolled down. The PDF/PNG export MUST show
it at its normal place, unchanged.
R-14.3.5 A project setting **Show week numbers** (default off) MUST write the
ISO 8601 calendar week (`W12`, German `KW12`) at every Monday in the date row,
but only when a week is wide enough for the label.
R-14.3.6 While an action is moved or resized the date row MUST show a tick for
its start and one for its end; while a milestone is moved, one tick for its
date. The ticks disappear on release.
R-14.3.3 Each role MUST have a distinct color; the canvas MAY be filtered by role.

### 14.4 Undo and saving
R-14.4.1 Structural edits (add/remove/move/resize of sections, actions, details,
dependencies) MUST be undoable (Cmd/Ctrl-Z, up to 100 steps). A batch of chat
commands is one undo step. Redo and undo of detail reordering are [TODO].
R-14.4.5 The app SHOULD offer **redo** (Shift-Cmd/Ctrl-Z) for every undoable
step. [TODO]
R-14.4.2 Deletes MUST NOT ask for confirmation; they are undoable instead.
R-14.4.3 Autosave MUST be configurable: on/off, interval (every change or 5..99
minutes), save on exit, and optional rotated versions (5..10 kept).
R-14.4.4 The last-used directory MUST be remembered system-wide, outside the
project file.

### 14.5 Command bar
R-14.5.1 The chat (R-10) MUST address objects by the numbers shown on screen
(section `n`, action `n.k`, detail `n.k.m`, Phases action `Pk`), from the single
WBS numbering source (R-13.2).
R-14.5.2 A batch mode MUST run several commands, one per line, stopping at the
first error or at `quit`/`exit`/`stop`. Lines starting with `#` or `//` are
comments.
R-14.5.3 `add` MUST always name what is added: `add action`, `add section`,
`add role`, `add milestone` or `add detail`. A short `add <name>` whose meaning
depends on the mode MUST NOT exist (it was mixed up too often); a bare `add ...`
is refused with a hint to the full forms. While a detail plane is open,
`add detail <name>` (and `add action <name>`) MUST create a detail of the
drilled action.
R-14.5.6 The form is `add action <name> [from <date>] [to <date>] [est <value>]
[after <id>]`. `from`, `to`, `est`, `after`, `on` and `in` are **reserved
words** that end the name; a name containing one MUST be quoted. Without dates
an action starts today and lasts 7 days; with only `from` it lasts 7 days. A
bad date is an error and creates nothing.
R-14.5.4 Commands stay in English regardless of the UI language.

### 14.6 Language and help
R-14.6.1 The UI MUST support English and German, including date and number
formats (`de`: `02.01.2006`, decimal comma).
R-14.6.2 In-app help MUST include a glossary, the relationship rules, a
monitoring guide and a sections/actions guide, in English and German. These
come from a single source in code, which also generates the Markdown manual
pages. The generated pages MUST NOT be edited by hand.

## 15. Report composition (Epic 5) — [TODO]

Replaces the fixed report layout and the `Format pdf|png` choice in the project
dialog by a report dialog in which the user picks the parts to produce.

### 15.1 Report dialog
R-15.1.1 Clicking **Report** MUST open a report dialog. It MUST NOT export
directly.
R-15.1.2 The dialog MUST offer these PDF parts. Each part MUST be individually
selectable (checkbox):
  1. **Project plan**: the whole-project Gantt with details compacted.
  2. **Project detail plan**: one Gantt page per action that has detail
     packages (its drill-plane view).
  3. **Work packages**: the table of actions, grouped by work package.
  4. **Roles**: the people/roles pages.
  5. **Work package details** (new, §15.3).
  6. **Task descriptions**: the formatted task descriptions (§17.8). This
     option is only offered once Epic 7 exists.
R-15.1.3 The dialog MUST offer a separate **PNG** option. It is independent of
the PDF parts: the user MAY produce a PDF, PNGs, or both.
R-15.1.4 The dialog MUST have **Run** and **Cancel** buttons. Run MUST be
disabled while nothing is selected.
R-15.1.5 The PDF MUST print the selected parts in the fixed order of
R-15.1.2. Reordering parts by the user is not required (MAY be added later).
R-15.1.6 The selection MUST be persisted in the project file (R-2.1.2) and
restored as the dialog's initial state. Defaults (R-2.1.6): parts 1-4 on, parts
5-6 off, PNG off. A legacy file with `reportFormat: png` MUST open with PNG on
and all PDF parts off.
R-15.1.7 The `Format` selector MUST be removed from the project dialog. The
grid, today-line and notes settings stay where they are (they also drive the
live canvas); the report dialog MAY show them too.
R-15.1.8 Dialog labels MUST be available in English and German (R-14.6.1).

### 15.2 Output files
R-15.2.1 Run MUST show ONE native save dialog that chooses the target
directory and the base name (default `<title>-report`).
R-15.2.2 The PDF MUST be written as `<base>.pdf`.
R-15.2.3 With PNG selected, the app MUST write into the same directory:
`<base>-plan.png` (project plan) and one `<base>-detail-<wbs>.png` per action
with detail packages (`<wbs>` = its WBS number, e.g. `1.2`).
R-15.2.4 The app SHOULD warn before it overwrites existing PNG files that the
save dialog did not confirm.
R-15.2.5 A selected part without content (e.g. Project detail plan when no
action has details) MUST be skipped without error. A PDF with no pages MUST NOT
be written.
R-15.2.6 After Run the app MUST report the list of written files.
R-15.2.7 Producing "only the task descriptions as a separate file" is done by
selecting only part 6; no separate export button is required.

### 15.3 Report "Work package details"
R-15.3.1 The report MUST list every work package (section, §16) in WBS order
with its work-package ID and name, followed by its actions.
R-15.3.2 Each action MUST be listed with: WBS number, name, start, end,
estimation, estimation mode, real effort and progress (when tracked), assigned
roles with percent, and its plain-text description (`desc`).
R-15.3.3 Below each action the report MUST list its detail packages (WBS
number, name, start, end, estimation, role, description).
R-15.3.4 Entries without a description MUST still be listed. Descriptions MUST
wrap (multi-line), never be truncated.
R-15.3.5 Summary actions/details (R-14.2.4) MUST be marked as such and show no
effort.

## 16. Work-package IDs (Epic 6) — [TODO]

Prerequisite for Epic 7. Today a section's number is its position, so moving
a section renumbers it. A stable work-package ID decouples identity from
position.

### 16.1 Terminology
R-16.1.1 A **work package** is a numbered section, i.e. the cluster that
groups actions (WBS level `n`, R-13.2.1). Below it are **actions** (`n.k`) and
**details** (`n.k.m`).
R-16.1.2 The locked Phases lane (R-14.2.2) is not a work package and has no
work-package ID.

### 16.2 ID
R-16.2.1 Every work package MUST have a **work-package ID**: a positive
integer, displayed as `WP<n>` (e.g. `WP7`).
R-16.2.2 Work-package IDs MUST be unique within the project. A duplicate MUST
be rejected.
R-16.2.3 An ID MUST be issued only once in the life of a project: an ID that
was used by a deleted work package is retired and MUST NOT be assigned again,
neither automatically nor by manual entry. The app MUST therefore store the
highest ID ever issued (and the retired IDs) in the project file.
R-16.2.3a A new work package MUST get the highest ID ever issued + 1.
R-16.2.4 The ID MUST be editable in the section editor of the right pane.
R-16.2.5 If an edited ID leaves a gap (the IDs are no longer 1..N without
holes), the app MUST show a one-time hint at the moment of the edit. It MUST
NOT show a permanent warning or flag.
R-16.2.6 Moving a work package up or down (R-14.2.1) MUST NOT change its ID.
R-16.2.7 The ID MUST be stored in the project file. A file without IDs MUST
open with IDs 1..N assigned in the current section order (R-2.1.6).
R-16.2.8 Changing a work-package ID MUST be excluded from undo (an explicit
exception to R-14.4.1): it is not recorded on the undo stack, and undo/redo of
other edits MUST NOT change an ID.
R-16.2.9 The work-package ID MUST be shown in the section header on the
canvas and in the reports.
R-16.2.10 The WBS numbers of actions and details MUST derive from the
work-package ID: the actions of `WP7` are `7.1`, `7.2`, ... and their details
`7.1.1`, ... (R-13.2.1). Moving a work package up or down therefore changes no
number.
R-16.2.11 Before an ID change is applied, the app MUST warn that the numbers
of the work package's actions and details change with it (and with them the
references used by the command bar, R-14.5.1), and MUST let the user cancel.

### 16.3 Stable action numbers
R-16.3.1 Every action MUST have a stored **action number** `k`, unique within
its work package. Its WBS number is `<work-package ID>.<k>` (R-16.2.10).
R-16.3.2 The action number MUST NOT change by any move: not by dragging the
action in time, not by changing its dates, not by moving its work package up
or down, and not by re-sorting. It changes ONLY by manual edit (R-16.3.5).
R-16.3.3 A new action MUST get the highest action number ever issued in its
work package + 1. Like work-package IDs (R-16.2.3), an action number of a
deleted action is retired and MUST NOT be issued again in that work package.
R-16.3.4 Actions MAY still be drawn in date order within their work package;
their numbers then need not be ascending on screen.
R-16.3.5 The action number MUST be editable in the action editor of the right
pane. A duplicate or retired number MUST be rejected. A gap MUST give a
one-time hint only (as R-16.2.5). Before the change is applied the app MUST
warn that the numbers of the action's details and the command-bar references
change, and MUST let the user cancel. The change MUST be excluded from undo
(as R-16.2.8).
R-16.3.6 Moving an action to another work package MUST be allowed. The action
loses its number: it MUST get the target work package's highest ever issued
number + 1 (its old number stays retired in the source work package), and the
app MUST tell the user the new number.
R-16.3.7 Detail numbers `m` (`n.k.m`) MUST follow the same rules within their
action: stored, stable under moves, issued as highest ever + 1, changed only
by manual edit.
R-16.3.8 Milestones MUST be numbered like actions, from the SAME number range
of their work package (an action and a milestone never share a `k`). All
rules of §16.3 apply to them.
R-16.3.9 A file without stored action, milestone and detail numbers MUST open
with the numbers they currently display (date order), which are then stored.

## 17. Tasks (Epic 7) — [TODO]

A task carries a formatted (Markdown) specification of a work package or an
action, in addition to the plain-text descriptions that already exist.

### 17.1 Scope
R-17.1.1 A task MUST be attached to a work package (the default) or to an
action. A work package or action MAY have **at most one** task (0..1).
Details (`n.k.m`) MUST NOT have tasks.
R-17.1.2 The Phases lane and its actions MUST NOT have tasks.

### 17.2 Task object
R-17.2.1 A task MUST have: a stable id, a title, a status, a **Spec Ready**
flag, and its owner (work-package ID or action id).
R-17.2.2 The status MUST be one of exactly three internal values `new`,
`open`, `closed`, displayed as English `New`, `In Progress`, `Closed` and
German `Neu`, `In Arbeit`, `Geschlossen`. No other status is allowed.
R-17.2.3 **Spec Ready** (German: `Spezifikation fertig`) means "the description is complete". It MUST be a
boolean, independent of the status. Setting a task to `closed` without Spec
Ready MUST be allowed; the app MUST show a one-time hint at that moment, not
a permanent warning.
R-17.2.4 A new task MUST start with status `new`, Spec Ready off, and the
owner's name as its title.
R-17.2.5 Task metadata (id, title, status, Spec Ready, owner) MUST be stored in
the project file. New keys get defaults (R-2.1.6).
R-17.2.6 Creating and removing a task and changing title, status or Spec Ready
MUST be undoable (R-14.4.1). Text edits inside the Markdown editor use the
editor's own undo.
R-17.2.7 Deleting a work package or action with a task MUST delete the task
with it (undoable); its files are handled by R-17.4.3.

### 17.3 Storage on disk
R-17.3.1 A project directory MUST hold only one project. The app SHOULD warn
when a project is saved into a directory that already holds another project
file.
R-17.3.2 The **task folder** MUST be configurable in the project dialog. The
default MUST be `<short name>-packages`. The **short name** is derived from
the project title: German umlauts and sharp s are transliterated (`ä`→`ae`,
`ö`→`oe`, `ü`→`ue`, `Ä`→`Ae`, `Ö`→`Oe`, `Ü`→`Ue`, `ß`→`ss`), then every
remaining character outside `A-Z a-z 0-9 - _` (spaces, punctuation, other
accents) is removed. The path is relative to the project directory.
R-17.3.3 Changing the task folder MUST show a hint that the new folder is
created empty. The app MUST NOT move, copy or overwrite any file when the
setting changes.
R-17.3.4 A work-package task MUST be stored as
`<task folder>/WP<n>/requirement-WP<n>.md`, `<n>` being the work-package ID.
R-17.3.5 An action task MUST be stored inside its work package's directory
as `<task folder>/WP<n>/action-<n>.<k>/requirement-action-<n>.<k>.md`, where
`<n>.<k>` is the action's current WBS number (R-16.2.10). Actions get no
additional fixed ID.
R-17.3.6 Images MUST be stored in the `img` subdirectory (lower case) of the
task's directory and referenced from the Markdown by a relative path
(`img/<file>`).
R-17.3.7 Directory and file names MUST NOT be translated.
R-17.3.8 A project that has never been saved has no directory. Task content
MUST then be kept in memory and written on the first save.
R-17.3.9 The Markdown files MUST be written on save and on autosave
(R-14.4.3), together with the project file.
R-17.3.10 Only PNG, JPEG and GIF images MUST be accepted (the types the PDF
report can embed). When a task file is saved and references an image of
another type, or a missing image, the app MUST show a warning.

### 17.4 Moving and removing task directories
R-17.4.1 When the user changes a work-package ID (R-16.2.4) and the work
package already has a task directory, the app MUST ask at that moment whether
to move the directory to the new name. "Yes" MUST apply the ID change and
rename the directory, its Markdown file and the action-task directories inside
it (R-17.3.5) at once; "Cancel" MUST abort the ID change and leave everything
unchanged.
R-17.4.1a An action's WBS number changes only by a manual edit of its action
number (R-16.3.5) or by moving it to another work package (R-16.3.6). If the
action has a task directory, the app MUST ask at that moment whether to move
the directory to the new number; "Cancel" MUST abort the change (same rule
as R-17.4.1).
R-17.4.2 A move MUST NOT overwrite an existing directory or file; if the
target exists, the move MUST be refused and reported. Where several renames
are needed at once, the app MUST rename to temporary names first, then to the
final names.
R-17.4.3 When a task is deleted, its directory MUST NOT be deleted; on save it
MUST be moved to `<task folder>/_removed/<task id>/` (with a numeric suffix if
that exists) so that no description is lost.
R-17.4.4 A failed move MUST be reported and MUST NOT delete or overwrite any
file.

### 17.5 Markdown content
R-17.5.1 The supported Markdown subset MUST include: headings 1-3, paragraphs,
bold, italic, bullet and numbered lists, web links, images, and fenced
`mermaid` code blocks. Code blocks and tables MAY be supported.
R-17.5.2 Mermaid blocks MUST support `flowchart` and `sequenceDiagram`,
rendered by the embedded gmaid. Other diagram types MUST be shown as plain
code with a warning, not fail.
R-17.5.3 The Markdown file MUST stay plain, standard Markdown so it can be
edited offline with any editor.
R-17.5.4 Raw HTML and scripts in the Markdown MUST NOT be rendered or
executed (the file can be edited outside the app). Links MUST open in the
system browser, not in the app window. Images MUST be loaded only from the
task's `img` directory; remote images MUST NOT be loaded.

### 17.6 Markdown editor
R-17.6.1 The editor MUST work on the Markdown source and offer GitHub-style
**Write** and **Preview** modes. The preview MUST render images and Mermaid
diagrams.
R-17.6.2 The toolbar MUST offer: Heading 1, Heading 2, Heading 3, Bold,
Italic, Web link, Image. Lists SHOULD be offered too.
R-17.6.3 **Image** MUST let the user pick a local image file, copy it into the
task's `img` directory (a name clash gets a numeric suffix, never an
overwrite), and insert the Markdown reference at the cursor.
R-17.6.4 **Read from disk** MUST reload the Markdown file from the task's
directory, so files edited offline can be taken over.
R-17.6.5 The app MUST remember the version it last loaded or wrote (the
base). If both the editor and the file on disk changed since the base, the app
MUST NOT silently drop either side. It MUST offer: take the disk version, keep
the editor version, or merge line by line. A line-based merge MUST use the
base as common ancestor (three-way merge) and MUST show conflicting lines to
the user for a decision.
R-17.6.6 The same conflict check MUST run before the app writes a Markdown
file (R-17.3.9).

### 17.7 Views and panes
R-17.7.1 The main window MUST offer a view switch **Project | Tasks**.
R-17.7.2 The **Tasks** view MUST show a read-only list with one row per work
package and per action with a task: WP ID / WBS number and name, task title,
status, Spec Ready. Work packages without a task MUST be shown with an empty
task.
R-17.7.3 Selecting a row with a task MUST open the Markdown editor in the
center of the Tasks view. The list MUST be reachable again from the editor.
R-17.7.4 The right pane MUST get a new **Task** tab for the selected work
package or action. It MUST allow creating the task (when none exists),
editing title, status and Spec Ready, opening the editor, and removing the
task.
R-17.7.5 The task UI MUST be available in English and German (R-14.6.1).

### 17.8 Report "Task descriptions"
R-17.8.1 The report (part 6 of R-15.1.2) MUST print each task in WBS order
(work-package task first, then the tasks of its actions): owner number and
name, task title, status and Spec Ready, followed by the rendered Markdown
with its images and rendered Mermaid diagrams.
R-17.8.2 Each work package SHOULD start on a new page. Web links SHOULD be
clickable in the PDF.
R-17.8.3 A missing Markdown file or a broken image reference MUST NOT abort
the report; it MUST be shown as a visible placeholder and reported after Run.
R-17.8.4 Text outside the PDF core-font character set (e.g. emoji, arrows)
MUST NOT break the report; it SHOULD be printed correctly by embedding a
Unicode font.

### 17.9 Projection and out of scope
R-17.9.1 The `.md` projection (R-2.1.5) SHOULD list each task (owner, title,
status, Spec Ready, relative path of its Markdown file).
R-17.9.2 Out of scope: tasks on details, more than one task per owner, other
status values, chat commands for tasks (MAY be added later, R-10).
