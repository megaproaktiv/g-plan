# Changelog

All notable changes to pt-plan are recorded here. The format is loosely based
on [Keep a Changelog](https://keepachangelog.com/), and the project aims to
follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Fixed

- The text position of a **summary action** (Sammelvorgang) and of a summary detail
  was ignored: the label was always drawn to the right. It now follows the setting:
  up/down above/below the bar, outleft before it, outright and Auto after it. The
  rollup bar is too thin for a label inside, so left and center put it above the
  bar (left-aligned / centered).

### Changed

- **Command bar: `add` always needs the kind.** The short form `add <name>` (whose
  meaning depended on the mode) is gone because it was mixed up too often. Use
  `add action`, `add section`, `add role`, `add milestone` or `add detail`; a bare
  `add ...` is refused with a hint to the full forms.
- `add action <name> [from <date>] [to <date>] [est <value>] [after <id>]`: the name
  ends at the first reserved word (`from`, `to`, `est`, `after`, `on`, `in`), so
  `add action "Scheduled Callback" from 2026-11-06 to 2026-11-10` creates
  "Scheduled Callback" from 6 to 10 November. Before, a bare `add` put the whole
  line into the name. Quote a name that contains a reserved word. Without dates an
  action starts today and lasts 7 days; with only `from` it lasts 7 days.
  `add action <name> after <id>` replaces `add <name> :after <id>` (`:after` still
  works). In an open detail plane, `add detail <name> ...` adds a sub-action.
  The command help, the placeholder, the manual and the walkthroughs are updated.

- **New sections go to the end.** "Add section" (and the chat command) appends the
  section after the existing ones, so with WP-1 present the next one is WP-2. Before,
  an empty section could land in front of the others (empty sections were ordered by
  name).
- **The date row stays at the top.** The month, day and week labels are fixed to the
  top of the view while the chart is scrolled down. The PDF/PNG export is unchanged:
  it always shows the date row at its normal place.

- External ID recognition ignores hyphens as well as case: an action with `WP1`
  is found for `WP1`, `wp1`, `Wp1`, `WP-1` and `wp-1` (and the other way round),
  but not for `WP 1` (a space is not a hyphen). `WP1` and `WP-1` count as the same
  ID, so a second action cannot use either. Help and manual are updated.
- Monitoring ignores the "Phases" section. The Phases lane is a view, not a work
  package, so its actions need no External ID: they no longer appear as a section
  in the Monitor page or the result table, do not count toward the totals, cannot
  match bookings or be the default action, and keep a manually entered real effort.

### Added

- `LICENSE`: the project is MIT licensed (also linked from the README and shipped in
  the release archive).

- Project setup: **Show week numbers**. The ISO calendar week (W12, "KW 12" in
  German projects) is written at every Monday in the date row when a week is wide
  enough for the label. Off by default; saved with the project.
- Moving an action or a milestone shows a tick in the date row for its start and its
  end (one tick for a milestone).

- Manual: `monitoring.md` and the generated monitoring guides (English, German) describe
  the Remaining columns, the Export and that the Phases lane is ignored.
- Monitoring result table: new columns **Remaining** (estimated of the tracked
  actions minus actual; negative = over budget) and **Remaining %** (share of the
  estimation), per work package, per action (which now also shows its estimate)
  and for the project.
- Monitor page: **Export** button. It writes the result as markdown in two files,
  `<name>-compact.md` (work packages and project sum only) and `<name>-detailed.md`
  (with the tracked actions of each work package).

## [1.2.2] - 2026-11-06

### Added - one script to build and publish a release

- `scripts/release.sh` builds the macOS app package of a tag and publishes it as a
  GitHub release, replacing a manual procedure of about a dozen commands. It can
  **recreate** a package without uploading (`--no-publish`) and **republish** an
  existing release by replacing its assets and notes (`--republish`).
- It builds the version-stamped universal binary (Apple Silicon + Intel), the
  ad-hoc signed `ptplan.app` and the `ptplan.dmg`, verifies them (architectures,
  version stamp, signature, icon files, dmg contents), writes the release notes
  from the version's `CHANGELOG.md` section, and checks the upload by downloading
  the assets again and comparing their checksums.
- It refuses to run when the tag is not pushed, HEAD is not that tag, files that go
  into the build have uncommitted changes, or the changelog has no section for the
  version. It also copes with a `GITHUB_TOKEN` that cannot see the private
  repository by falling back to the stored `gh` login.
- Documented in `docs/release.md`.

### Changed

- The app's Get Info no longer shows the template placeholders ("My Company",
  "Some Product Comments"); it reads "(c) <year> pt-plan" and "Local graphical
  project-plan editor". The app itself behaves exactly as in 1.2.1.

## [1.2.1] - 2026-11-06

### Changed - new application icon

- pt-plan now has its own icon, a project plan in its simplest form: two actions
  joined by a link. It replaces the default Wails "W" everywhere: the app in the
  Dock and Finder, the `.dmg` and its drive icon (which carried the Wails logo),
  and the Windows `.ico`.
- On macOS 26 and later the app ships the new Icon Composer icon (light, dark and
  tinted appearances); older macOS versions use the classic `icons.icns` with all
  sizes up to 1024 px.
- The icon is kept as source in `assets/icon/` (SVG artwork, the Icon Composer
  layer and settings) with `render-icon.sh`, which builds the PNG, `.icns`,
  `.ico`, `Assets.car` and the `.dmg` file icon. The Wails build folder is
  git-ignored, so the icon used to exist on one machine only.

## [1.2.0] - 2026-11-05

### Added - the report always prints the details

- The report now contains the **whole-project overview** on its own page, followed
  by **one page per action that has details**, showing only that action's detail
  (drill) view. This no longer depends on the *Show Details* toggle. Each detail
  page is captioned with its WBS number and name (e.g. `Details - 1.2 Tunnel`).
  The work-package table and the people / roles pages follow as before.
- A PNG report (project setting *Report format = png*) still writes a single
  image, the overview.
- Binding change: `App.ExportReport(overviewBase64, detailsJSON)`, where
  `detailsJSON` is a JSON array of `{name, png}`. Core: `plan.WritePDFReportWithViews`
  and `plan.ReportImage`. Frontend: `gantt.exportReportImages`, `exportDrillPNG`.

### Fixed - report pages are scaled to fit (portrait or landscape)

- A tall, narrow detail view used to be squeezed into a thin strip on a landscape
  page. Each Gantt page now picks its **orientation from the image shape**:
  portrait for a taller-than-wide image, landscape otherwise, and the image is
  centred and fills the page. The table pages stay landscape.

### Added - role actions in the detail view, with Show percent / Show days

- **Show role actions now also works in the detail (drill) view.** A detail that
  has a role gets a `↳` role sub-row beneath it; as a detail carries a single
  role, it is shown as 100 % of the detail's own effort.
- Two new checkboxes under *Show role actions* in the Roles panel control the
  label. By default only the **role name** is shown; **Show percent** adds the
  assignment percent and **Show days** adds the effort share, for example
  `Wizard`, `Wizard 60%`, `Wizard = 6 d` or `Wizard 60% = 6 d`. They apply to the
  main view and the detail view alike, are saved with the project, and files
  from earlier versions open with both off. Backend: `App.SetRoleLabelOptions`.
- Behaviour change: the role sub-row label used to always read `name  60% = 6 d`;
  it is now name-only until the options are switched on.

### Fixed - actions added from the command bar did not appear

- After `mode section`, a bare `add <name>` creates a *section*, not an action
  (by design), but the mode selector and the status line kept showing
  "action". Users typed `add one`, `add two`, `add three` and saw nothing. The
  mode selector, canvas and status line are now synchronised with the backend
  after every command and batch run, and the reply to a section add says how to
  add an action (`mode action`, or `add action "<name>"`).
- **New** now resets the current section, selection and drill state, so the
  first action of a new plan no longer lands in the previous plan's section.

### Added - walkthroughs and the walkthrough manual

- Five feature walkthroughs in `documentation/walkthrough/` (getting started,
  sections and colours, roles and assignments, dependencies and milestones,
  details / notes / report) with screenshots captured from the running app.
- `task manual` in that folder builds them into one `manual.pdf`. Rendering is
  OS independent: the pinned `pandoc/latex` Docker image, or a local pandoc when
  Docker is not running. Symbols that LaTeX fonts lack (the circled callout
  numbers, arrows) are mapped to font-independent commands by a Lua filter.

### Changed

- The generated documentation (glossary, monitoring guide, plan objects,
  relationships, text rendering, work contour) moved from `documentation/` to
  `documentation/manual/`; the tests that check they are up to date follow.
- `go.mod` no longer carries a machine-specific `replace` for gmaid.
- The monitoring help example project is now *Skynet MVP*; the test-run records
  no longer name the tester.
- Developer tooling: `mcpdrive snap <file.png>` saves a screenshot of the canvas
  (used to produce the walkthrough images); its request timeout is 120 s.

### Fixed - report ID column and graphic sharpness

- **The report's "#" column showed the internal id** (e.g. a long
  `ACC_CORE_CONFIGURATION`-style string) that overflowed the table. It now shows
  the on-screen WBS number (e.g. `1.1`), in both the PDF and the Markdown report,
  including the "Work package" label in the People / roles tables.
- **The embedded Gantt graphic was soft.** The report now rasterizes the canvas
  at 3x (was 2x) with high-quality image smoothing, so lines and labels stay
  sharp when the PDF scales the image to the full A4 page.

### Added - per-action show/hide of detail bars

- When **Show Details** is on, each action that has details gets a **Hide detail
  bars / Show detail bars** button on its read-only panel (next to Edit and
  Details…, where Action order / Section order are). It toggles whether that one
  action's level-3 detail bars are drawn in the main view, independently of the
  other actions. Stored per action as `detailsHidden` (round-trips through the
  project file). Backend: `App.SetActionDetailsHidden`. The button only appears
  while the project-wide Show Details is on and the action actually has details.

## [1.1.0] - 2026-11-04

### Added - Sammelvorgang for sub-actions

- A **detail (sub-action) can now be marked as a summary (Sammelvorgang)**, like a
  level-1 action. On the detail plane it is drawn with the same solid rollup bar
  and end-cap wedges. Toggle it with the new column in the drill editor table
  (the "∑" checkbox). Backend: `App.SetDetailSummary`; stored as
  `DetailPackage.summary` (round-trips through the project file).

### Fixed - add in the detail plane

- **`add action "..." from ... to ...` typed while a detail plane is open now
  adds a sub-action (detail) of the drilled action,** instead of creating a new
  top-level action. The command bar learns which action is drilled (the frontend
  calls `App.SetDrill` on entering/leaving the plane). `add "<name>"` without
  dates defaults the detail to the parent action's span. An explicit
  `add detail ... to <wp>` still targets the named work package, and leaving the
  plane makes `add` a top-level action again. The drill-in status line says so.

### Added - batch mode for the command bar

- A **🧩 button** next to the command bar switches to batch mode: the bar grows to
  five lines and takes several commands, one per line (paste a whole textual
  description of a plan). **Run batch** or Cmd+Enter runs them one after another.
  Blank lines and lines starting with `#` or `//` are skipped. The batch **stops at
  the first command that gives an error** and at a line `quit`, `exit` or `stop`;
  commands before it stay applied and the script stays in the box for fixing. The
  log shows the summary, the line-by-line log is in its tooltip. The whole batch is
  **one undo step** (Cmd-Z). Backend: `App.ChatBatch`.

### Fixed - estimation, undo, right-click menu, role report

- **`est 4.1 20` had no effect on an action in derived mode** (an action whose
  estimation is the sum of its details): the value was stored but ignored. Typing an
  estimation now makes it the fixed value of the action. Not reproduced on the
  saved Skynet file (there the command worked), so this is the cause found by
  reading the code, covered by a test.
- **Moving is undoable (Cmd-Z):** moving or resizing an action or a detail, moving
  a section and re-ordering details, as well as deleting a detail, push an undo step.
- **Right-click menu:** *Delete Action* (and the other deletes) did nothing: they
  asked with `confirm()`, which the desktop web view does not show reliably. Deletes
  run at once and are undone with Cmd-Z. The section menu now has **Move Section
  Up / Down** (and the action menu **Move Action Up / Down**). The menu content is
  in `ctxmenu.ts` and tested.
- **Report, people and roles: *Consumed* was always 0.** It is now derived from the
  completed (real) effort of the assigned action: the days scaled by the
  assignment percent and the consumed percent of the action, e.g. `1.88 (4.6%)`.

### Changed - synchronization result

- The result table shows the **External ID** of every action in its own cell
  (the default action is marked).

### Added - help on sections and actions

- New help tab **Sections and actions** (EN/DE): numbering, color, order (also for
  sections without actions), the Phases lane, the fields of an action and all
  contour settings (Flat, Front-Loaded, Back-Loaded, Double-Peak, None) with how the
  progress marker is read. Generated to `documentation/plan-objects*.md`.

### Fixed - section order and typed numbers

- **"Order up" now works on a section without actions.** Sections were ordered
  only through the position of their actions, so an empty section (numbered last,
  alphabetically) could not move. The order of the sections is now stored
  explicitly in the project file (`sectionOrder`), so empty sections can be moved
  like any other, and renames and removals keep it consistent (also under Undo).
- **`move 3.1 to ...` moved 2.1.** The number typed in a command and the number
  shown on screen came from different calculations; the command side counted the
  Phases lane as section 1. All of them (action numbers `n.k`, the Phases lane
  `P1`..., section numbers for `select`, milestone codes, detail codes `n.k.m`)
  now come from one shared numbering, so a number in a command always means what
  the screen shows. The Phases lane actions can be addressed as `P1`, `P2`, ...
  Note: `n.k` follows the dates, so after a move changes the date order the
  numbers shift, as the screen shows.

### Added - drag status line and monitoring help

- **The status line follows a drag.** While an action, a detail action or a
  milestone is dragged, the status line (top) shows the state it is moving to,
  start first: `Moving "AWS Landing Zone": start 2026-12-05 (+3 days) · end
  2027-01-10 (+3 days) · 37 days (25 working days)`, with the target section for
  a move into another lane. The normal text returns when the drag ends.
- **Monitoring help** (new help tab): explains how the External ID is
  interpreted (whole word in the booking text, capital letters ignored, several
  IDs split the time evenly, no ID goes to the default action), a ten-step guide
  and a worked example whose numbers are computed by the program itself. In
  English and German. Also generated into `documentation/monitoring-guide.md` and
  `monitoring-guide.de.md`.

### Fixed - section mode and the External ID field

- **`add <name>` in section mode no longer switches to action mode.** The add
  left an earlier action selection in place, and the UI follows the selection.
  The new section is now selected (and a role add clears the selection), so the
  mode stays as it is.
- **External IDs are editable while monitoring is on.** (They used to be
  read-only then, which gave no visible reason when clicking the field.) A
  change clears the last synchronization result; the next synchronization uses
  the new ID. A refused value (already used, too long, ...) is now explained in
  words next to the field instead of only a red outline.
- **An action with the contour None cannot carry an External ID.** The field is
  greyed out with an explanation, setting one is refused, switching an action
  that has one to None is refused until the ID is removed, and older files are
  repaired on load (the ID of a None action is dropped; it never had an effect).

### Fixed - adding actions in the command bar

- **A new action is shown immediately.** After a command the canvas was not
  given the new project (only the selection border and the side panel were
  updated), so an added action appeared only after something else, like Save,
  redrew the canvas. The canvas is now refreshed first, then the selection
  follows.
- **`add action "2 Email and chat rollout" from 2027-05-01 to 2027-06-30` keeps
  the whole text as the name.** The name is everything between `add action` and
  `from`, so quotes are optional (the unquoted form used to keep only the first
  word, giving an action named "2"), and typographic quotes (curly, German and
  guillemets) are understood as well as straight ones. The command bar also
  turns off the system's automatic text replacement.
- `add action "<name>"` without dates works in every mode (it used to create an
  action named `action "<name>`), and `add <name>` in action mode adds the
  action with the whole name.
- An action id can no longer consist only of digits (it would be read as the
  action number, "2" = the second action).
- Adding an action whose end is before its start is now refused (it used to
  create it).

### Added - contour "None"

- **New contour None:** an action with no estimation and no real effort. Its
  Estimated and Real fields in the editor are greyed out, it has no progress bar
  (no curve, marker, shaded area or rating), and it adds nothing to totals, role
  load, capacity warnings or reports. It takes no bookings in the monitoring
  (not matched, not synchronized, cannot be the default action). The stored
  numbers are kept, so switching back restores them. Described in the glossary
  (Contour) and in `documentation/workcontour.md`.

### Added - glossary in the help (English and German)

- **New help page "Glossary"** (the `Aa` tab in the help, next to the command and
  relationship help) with the vocabulary: Section, Work Package, Action, Role,
  Milestone, Estimation, Work, Duration, Duration Complete (Duration Progress),
  Work Complete (Effort Progress) and Task Complete, plus four on-screen terms
  (Contour, Consumed, External ID, Default Action). Available in both languages.
  The same text is generated into `documentation/glossary.md` and
  `documentation/glossary.de.md` (a test fails if they drift).
- Definitions were checked while writing: Work Complete is a percentage (Work /
  Estimation, e.g. 32 of 80 h = 40 %), not an hours count; Section and Work
  Package name each other as the same thing; Task Complete is stated as
  definition only.
- **Wording:** the "default work package" is now called **default action** in the
  Monitor page, because a work package is a section and this setting picks an
  action.
- **Action views show the project's unit first:** in the action view, the editor
  note, the actual-effort block and the bar tooltip, the bold value is in the
  project's standard unit (days or hours) and the other unit follows in brackets,
  for example `6,56 days (52,5 h)` in a days project and `52,5 h (6,56 days)` in
  an hours project.

### Fixed / changed - progress and contour (corrected direction)

- **The contour now works the way the curve reads:** the curve is the planned
  effort density and the effort is the **area under the curve**. A Front-Loaded
  action needs a lot of real effort before its marker gets far, so with little
  effort booked the marker is *behind* the linear position (it was wrongly
  ahead). Back-Loaded is the mirror image; Double-Peak crawls at both ends. The
  marker is found by solving the area exactly. Spec documents corrected.
- **Two consumed ratings per action**, to one decimal: the pure rating (real ÷
  planned, e.g. 16,4 %) and the rating with the contour (how far along the
  planned curve the effort gets, e.g. 12,6 %). Flat gives the same value.
- **The action bar shows it:** with Show Progress the label carries the ratings
  (`16.4% (Double-Peak 12.6%)`), the area under the curve up to the marker is
  shaded, and a tooltip gives planned and real hours. The label is drawn above
  the graphics.
- **Synchronization feeds the real effort:** actions that are tracked (External
  ID or default action) get their real effort from every sync. The Real
  field in the action editor is read-only for them, with a note on the source
  and time; other actions stay manual and editable. The action view shows the
  real effort and both ratings.

### Added - monitoring of real effort from an external system (epic, part 1)

- **Monitor page** (toolbar button *Monitor*): switch monitoring on for the
  project, choose the external system type (Aerion; CSV is listed but greyed out
  until implemented) and the project ID in that system. Both are mandatory to
  switch monitoring on.
- **External ID** on actions (up to 16 characters, unique per project). It can
  be changed at any time, also while monitoring is on.
- **Section overview** in the Monitor page: a section (work package) is green
  when at least one of its actions has an External ID.
- **Synchronize**: fetches all single-line bookings of the external project from
  the project start to its end, with a progress bar and the number of entries
  fetched, and shows a table per section (estimated vs. actual) plus the project
  sum and the bookings that matched no External ID. The work package ID is
  searched in the booking text (case-insensitive, whole token, longest match).
- **Configure System** (Aerion): base URL, username, password, client ID and
  client secret, stored per user in `~/.pt-pm/config.yaml` (mode 0600). Secrets
  are never sent back to the screen. A **Test** button checks credentials and
  project.
- **Period options** (Monitor page): a synchronization can also cover **one month
  before the project start** and/or **one month after the project end** (two
  check boxes, saved with the project). The exact dates are shown; month
  arithmetic is clamped to the month end (31 Mar minus a month is 28 Feb).
- **Several references in one booking:** when a booking text names more than one
  External ID, its time is **spread evenly** over those actions (the remainder
  seconds go to the first ones, so no time is lost). The project total counts
  the booking once; a row counts the bookings that reference it.
- **Action view shows planned and actual effort:** the small action view now
  shows "Planned effort" (project unit, hours and days) and, while monitoring is
  on, "Actual effort" from the last synchronization in hours and days, split
  into the bookings assigned through the External ID and the unassigned ones
  credited to the default action.
- **Default action** (Monitor page): an **action** chosen from a list
  grouped by section. Bookings that match no External ID are counted as that
  action's effort and shown on a marked line ("of which unassigned bookings")
  under it, so nothing is lost. The default action counts as tracked even
  without an External ID, so its estimate is compared with what it received.
  Without a default they stay on the separate "unassigned" line. A default
  action that was deleted makes the sync stop with a clear message.
- New Go package `extsys` (Aerion driver adapted from the reporting tool, now
  with paging, re-authentication on 401, and back-off on 429); read-only towards
  Aerion. Documentation: `documentation/monitoring.md`.
- Project file: optional `monitor` setting and action `externalId`; older files
  load unchanged.

## [1.0.0] - 2026-11-04

### Added — progress tracking and visualization

- **Estimated vs. real effort.** Every action now carries a *Real effort* next
  to its Estimated effort (two-line *Headline effort* in the action editor, each
  line committed on Enter or blur; invalid input is rejected, marked
  `aria-invalid`, and the stored value is kept). Range 0 to 5,256,000 minutes.
- **Progress percentage.** Integer 0 to 100, computed with integer math only
  from real and estimated effort (0 when the estimate is 0, 100 when real
  reaches the estimate).
- **Work contour.** Per-action effort-distribution shape: Flat (default),
  Front-Loaded, Back-Loaded or Double-Peak, chosen in the action editor. New
  actions default to Flat.
- **Completion position.** The progress marker is placed along the bar through
  the action's contour (not simply linear), pinned to the start at 0 effort and
  to the end at full effort.
- **Show Progress** project option (default off). When on, each action bar draws
  a contour curve (bars 8px or wider) and a progress marker, with colours chosen
  for contrast against the bar color (marker at least 3:1, curve at least half
  the luminance range).
- New bindings `SetWorkPackageRealEffort` and `SetWorkPackageContour`; DTO fields
  `realEst`, `contour`, `progressPercent`, `completionPos`, `showProgress`.
- Documentation: `documentation/workcontour.md`.
- Tests: property-based tests (`pgregory.net/rapid` for Go, `fast-check` for the
  frontend) and happy-dom render tests; `bun test` runs the frontend suite.

### Changed

- **Breaking (API):** `SetViewOptions` takes a fourth argument, `showProgress`.
- Progress for derived-mode actions is computed against the effective (summed)
  estimate, matching the displayed effort.
- Estimation entry now rejects values above the 5,256,000-minute maximum.

### Compatibility

- Project files written by earlier versions load unchanged: missing `realEst`
  is 0, missing `contour` is Flat, missing `showProgress` is off. The file
  schema (`pt-plan/project@2`) is unchanged.

## [0.3.0] - 2026-11-04

### Changed — report / export

- **The report no longer shows editor chrome.** The link-draw anchor points and
  the dashed selection borders (and any selected-object highlight) are omitted
  from the exported PDF/PNG — the report shows the plan only, not the editing
  affordances.
- **The report now matches the on-screen chart exactly.** Export renders the
  live canvas image (SVG → PNG) instead of a separate gmaid drawing, so notes,
  rounded (spline) vs square arrows, and the `WP-n` section headers in the
  report are identical to what you see. (Previously the PDF used a different
  renderer: notes were missing, section headers differed, and arrows were
  always square even when the screen used splines.)
- **"Report…" button** (renamed from "Export PDF report").
- **Report settings** in Project settings: choose **PDF** (chart + tables) or
  **PNG** (chart only); a **Today line** toggle (default on); a **grid** interval
  (days / weeks / months, default months); and **Show notes** (default on). The
  today line, grid, and notes also drive the live canvas so the screen and the
  export stay in sync. These settings persist with the project.

### Fixed

- **Adding an action after selecting a section no longer flips back to section
  mode.** Previously, after `select <n>` then (in action mode) `add <name>`, the
  action *was* created in the selected section but the command bar re-applied
  the stale section selection, flipping the view back to section mode so it
  looked like nothing happened. The command-bar `add` now selects the new
  action, so the view stays in action mode and focuses it.
- **Selecting a section now makes it the current section.** After selecting a
  section (by click, or `select <n>` / `use <n>` in the command bar), a new
  action — added via the command bar, the `+` shortcut, or the context menu —
  is created in that section instead of the default lane.
- **`select <n>` in section mode** now selects the section the UI labels as
  *n* (and draws the same selection box as clicking it). It previously used a
  different internal numbering that counted the Phases/default lanes, so the
  number could resolve to a different section than the one shown.

### Added — notes

- **Action notes (yellow sticky).** Each action can show its Description as a
  small yellow notepad near the bar. In the action editor a **Note** checkbox
  turns it on and a 3×3 position picker places it: edges (up / down / left /
  right / center) and the four **diagonal corners** (upper-left, upper-right,
  lower-left, lower-right) so the connector runs at ~45°. The note is drawn with
  a yellow background in small letters and is connected to the nearest point of
  the action by a plain line (no arrowhead). Hidden by default.

### Changed — milestones

- **The milestone dialog now shows a Links table**, like the action dialog: the
  dependencies that touch the selected milestone are listed (From / Kind / To,
  with the milestone shown by name) and can be deleted with the × control.

- **Right-click "New Milestone"** now creates the milestone in the section and
  at the date under the cursor (not at the project start), and selects it so it
  is immediately visible.
- **Both milestone types are draggable** by the mouse to change their date:
  section diamonds and the full-height markers (project Start/End and Phases-row
  milestones). Dragging the project Start/End marker moves the project window
  bound. The move is undoable.
- **Phases-row milestones draw as a full top-to-bottom line** (like the project
  Start/End markers) and **cannot be linked** to anything.
- **Dependencies to/from milestones use the normal before/after rules.** You can
  draw a link from an action's start/end edge to a milestone (or from a section
  milestone to an action); the relationship kind follows the drawn anchors
  (e.g. action end → milestone = FS, action start → milestone = SF) instead of
  being forced. A section milestone can be a link source or target by mouse
  (anchor drag / click) or keyboard (select + `L`).

## [0.2.0] - 2026-11-03

### Added — dependency relationships, language, help

- **Seven dependency relationship kinds** with explicit timing rules
  (A = predecessor, B = successor, Mid = midpoint):
  - FS `Start(B) ≥ End(A)`, SS `Start(B) ≥ Start(A)`, FF `End(B) ≥ End(A)`,
    SF `End(B) ≥ Start(A)`, MS `Start(B) ≥ Mid(A)`, ME `End(B) ≥ Mid(A)`,
    MM `Start(A) ≤ Mid(B) ≤ End(A)`.
  - Touching endpoints satisfy the rules; a violated link is drawn red. The
    anchor the link is drawn from picks the kind (end/start = FS, middle = MS/
    ME/MM, etc.). The old `OV` kind is kept as a legacy alias of MM.
- **GUI language** (English / Deutsch) in Project settings, persisted with the
  project. An internal config maps each language to its ISO code, date format,
  and number format (English: ISO dates, dot decimal; German: D.M.Y dates,
  comma decimal). Numbers, dates, the links-table abbreviation (FS… / EA…), and
  help are shown in the chosen language. **The command bar / chat language is
  always English and is never translated.**
- **Multi-page help**, reached from the help button: object-style icon buttons
  (⌨ command help, ⇄ relationship help) with hover tooltips select a page shown
  in the GUI language. The relationship page is a table (abbreviation, name,
  rule, example); the command page lists the CLI commands in English.

### Changed — drill-down & summary

- **Escape no longer drills up.** In the detail (drill) view, `Escape` only
  cancels link mode; use `U` to go up. This frees `Escape` for deselecting
  objects elsewhere.
- **Unified drill-down affordance.** Any action with detail sub-actions — normal
  OR summary — shows a drill-down icon at the **right edge** of its bar.
  A single click on the icon drills into the detail plane and the cursor shows a
  plus (zoom-in). On the thin summary (Sammelvorgang) bar the icon is just drawn
  smaller. Double-clicking an action opens its editor (same for normal and
  summary).

### Changed — open / save / new safety

- **Two distinct save actions.** The toolbar has a **Save** button
  (quick-save to the current file; greyed out when there are no unsaved changes;
  `⌘S`/`Ctrl-S`) and a separate always-active **Save As…** (choose a new
  location). If a project was never saved, Save falls back to Save As.
- **New enforces save, never discards.** "New" (the only blank-slate action)
  always saves a dirty project first — to its known path, or to a
  `Downloads/<title>.ptplan` fallback when it was never saved — then resets to
  an empty project ready to edit. No save/discard dialog. The same
  save-if-dirty rule runs before Open.
- **Autosave** (Project settings): a master **Autosave on/off** switch, plus
  `Autosave (minutes)` — `0` = after every change, otherwise `5`–`99` (values
  `1`–`4` round up to `5`), default `5`. When autosave is off, no timed save
  happens (the minutes field is disabled); manual Save and the New/Open
  must-save rule still apply. Autosave writes only when the project already has
  a path.
- **Autosave versions** (Project settings): an optional **Save versions**
  toggle with a kept-count of `5`–`10` (default `5`). When on, each autosave
  first rotates the previous file content into a numbered sibling
  `<name>.NNN.ptplan` (zero-padded so they sort by name then version number;
  highest number = newest) and prunes to the kept count.
- **Autosave on exit** (Project settings, default **on**): saves a dirty
  project on quit.
- **Last directory remembered** system-wide (in the user config dir) and used
  to pre-select the folder in Open / Save As.

## [0.1.1] - 2026-11-03

### Added — text rendering

- **Per-action and per-detail font color**, with a readable default. When no
  color is set, labels auto-contrast: the renderer picks black or white from the
  bar's background luminance so text stays legible on dark bars (fixes
  unreadable white text on dark detail/role colors).
- **Text position** for action and detail labels, chosen with a small graphic
  picker (a mini action bar with position dots plus an **Auto** toggle):
  - `auto` (default): the label is drawn **inside** the bar when it fits,
    otherwise just **outside to the right**.
  - `center`, `left` (inside positions).
  - `outleft`: outside to the left, ending at the bar's left edge.
  - `outright`: outside to the right, starting just past the bar's right edge
    (`right` is kept as a legacy alias).
  - `up` / `down`: above / below the bar.

### Fixed

- The Sammelvorgang (summary) bar is now drawn as a single clean polygon with
  snapped integer coordinates, removing the sub-pixel seams/gaps that appeared
  at its left and right end caps.
- The text-position picker no longer snaps back to **Auto** when a position dot
  is clicked (the widget is no longer wrapped in a `<label>` that forwarded
  clicks to the Auto control).

### Documentation

- Added `documentation/text-rendering.md` describing the label text, color, and
  position rules.

### Tooling

- Added `cmd/mcpdrive`, a small stdlib-only Go client for the Wails v3 MCP
  server, used to drive and verify the running app during development.

## [0.1.0] - 2026-11-01

- Initial working editor: graphical Gantt with drag move/resize, swimlane
  reassignment, zoom, milestones and dependency arrows; drill-in detail tasks;
  per-project roles with percentage assignments; baselines; WBS numbering;
  command bubble; single-file `.ptplan` project format; and PNG/PDF export.

[0.3.0]: https://github.com/megaproaktiv/g-plan/releases/tag/v0.3.0
[0.2.0]: https://github.com/megaproaktiv/g-plan/releases/tag/v0.2.0
[0.1.1]: https://github.com/megaproaktiv/g-plan/releases/tag/v0.1.1
[0.1.0]: https://github.com/megaproaktiv/g-plan/releases/tag/v0.1.0
