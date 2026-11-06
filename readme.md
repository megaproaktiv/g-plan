# pt-plan

A local, graphical **project-plan editor**. Draw a project plan as a Gantt-style
chart and edit it directly with the mouse: move work packages between sections,
shift them earlier or later in time, and lengthen or shorten them. Drill *into* a
work package to plan its detail tasks. Assign roles (people) to work packages by
percentage and see each role's load across the whole timeline.

The basic idea is to have a robust project plan editor for small and medium projects, and that automatic recalculation is only done if it's really necessary. 
Usually, you just get a warning, and then you have to make the adjustment steps, and then you are really aware of the problem and of the solution. 
Also, most of the thing should be doable with shortcuts from the keyboards and with text commands. 
Because the text command makes integration with MCP servers easy 

pt-plan is **multi-project** and **file-based**. Every project is a single
self-contained JSON file:

| File | Role | Edited by |
|---|---|---|
| `<project>.ptplan` | **The whole project** — config (unit, hours/day, zoom, view options, arrow style), the graphic content (sections, work packages, detail sub-packages, milestones, dependencies, dates), roles + percentage assignments, section colors, and baselines. | app |

See [`requirements.md`](requirements.md) for the full business requirements
(RFC-2119) and the rationale behind the file model.

## Why this file model

- Earlier versions stored a project as three sibling files (a Mermaid-Gantt
  `.mmd` graphic source, a `.roles.json` for people, and a generated `.md`
  projection). As the editor diverged from plain Gantt/Mermaid, the canonical
  format is now **one JSON file** that holds everything — simpler to save,
  move, diff, and reason over. Legacy three-file projects still open (for
  migration); saving writes the single `.ptplan`.
- The JSON is the single source of truth: configuration, the plan graphic, and
  the people/roles all live together, so `unit`, `hours/day`, and the canvas
  `zoom` are saved with the model and restored on reopen.

## Concepts

- **Project** — one `<project>.ptplan`. Configured as
  either **days** or **hours** throughout.
- **Section** — a horizontal lane (milestones, phases, or a work package with
  its sub-bars).
- **Work package (level 1)** — a bar with a duration **and** an estimation
  (whole days or whole hours). Work packages may be **clustered**.
- **Detail package (level 2)** — a task *inside* a work package. Only two levels
  exist. Detail estimates may carry two-decimal fractions (stored as integers to
  avoid floating-point error).
- **Milestone** — a zero-duration marker.
- **Dependency** — a typed link between tasks (FS / SS / FF / SF).
- **Role** — a named person/skill (name + description), assigned to work packages
  by percentage, with an optional **consumed** value for progress.

## Epics

1. **Graphical plan editor** — render the plan; mouse-move work packages across
   sections and time; lengthen/shorten; zoom days ↔ weeks ↔ months ↔ year.
2. **Third dimension (drill-in)** — open a work package to plan its detail
   packages (two levels only).
3. **Roles & people** — per-project roles assigned by percentage; a role view
   showing one role's planned load across the whole timeline.
4. **Chat text interface (optional)** — a command bubble for base commands
   (e.g. *"move WP4 to March"*, *"add detail Tests to WP4"*), for users who
   prefer text or need an accessible alternative to the mouse.

## Rendering strategy

- **Live canvas:** the app draws the Gantt itself (SVG) from the in-memory
  model, so drag / resize / zoom are instant.
- **Export:** PNG and PDF report are rendered from the live canvas, so the
  export looks like the screen (without selection and warning chrome). The PDF
  holds a compacted and a detailed view.
- **Legacy:** the gmaid / `.mmd` path (gmaid embedded as a Go package) is kept
  for legacy import/export only.

## Technology

- **Go + [Wails v3](https://v3.wails.io/)** — native desktop shell with a
  WebView UI. Services are bound and typed TS bindings are generated into
  `frontend/bindings`.
- No CGo, no bundled browser. gmaid (pure Go) is an embedded package.
- Cross-platform: macOS first, then Windows.

## Build and run

From `src/app`:

```sh
wails3 build   # build the ptplan binary into bin/ptplan
wails3 dev     # live-reload development
go test ./...  # run the pure-Go core + app tests
```

## Status

Core (model + file I/O + gmaid serialization + .md projection + PDF report)
implemented and unit-tested. gmaid is used as an **embedded Go package**
(`github.com/megaproaktiv/gmaid`), not an external binary, so the rendered
Gantt and the people/role tables can be assembled into a single PDF report
in-process. Wails desktop app builds and packages: interactive canvas (Epic 1:
zoom days/weeks/months/year, drag-move, edge-resize, vertical drag to another
section, Phases lane, milestones), drill-in detail editor (Epic 2), roles panel
with per-month load (Epic 3), and a chat command bubble with batch mode
(Epic 4). Also done: native open/save dialogs, autosave, undo, baselines, WBS
numbering, progress and work contours, and monitoring (sync of real effort from
Aerion). Open items are marked [TODO] / [PART] in
[`requirements.md`](requirements.md) §13–14.

## License

[MIT](LICENSE) - Copyright (c) 2026 Gernot Glawe.
