# Text rendering rules

How pt-plan draws the **label** on an action / detail bar in the hand-rolled SVG
Gantt (`src/app/frontend/src/gantt.ts`). Covers three settings that each bar
carries and how they combine: the label **text**, its **color**, and its
**position**.

Scope: these rules govern the on-screen canvas only. The high-resolution PNG /
PDF export is rendered from the `.mmd` via gmaid and does **not** carry any of
this styling (color/position are editor metadata, not Mermaid Gantt syntax).

---

## 1. What the label says (text content)

For every bar the label string is built the same way:

```
label = (est && est !== "0") ? `${name}  ${est}` : name
```

- The action/detail **name**, followed by the **effort** when it is non-zero.
- A zero (or empty) effort is omitted, so the bar shows just the name.
- Summary (Sammelvorgang) bars follow the same rule but the label always sits to
  the right of the rollup graphic (see §4.4).

Font: `system-ui, sans-serif`. Size: **11px** for level-1 action bars and
drill-plane detail bars, **10px** for main-view detail sub-rows and role rows.

---

## 2. Which fields drive rendering

Each `WorkPackage` and `DetailPackage` carries two optional label fields
(persisted in the single-file project JSON and in legacy roles.json; surfaced on
the DTO as `fontColor` and `textPos`):

| Field       | Values                                             | Empty / "auto" means                         |
|-------------|----------------------------------------------------|----------------------------------------------|
| `fontColor` | any hex (`#rrggbb` / `#rgb`)                        | **auto-contrast** (compute black or white)   |
| `textPos`   | `center` \| `left` \| `outleft` \| `right`/`outright` \| `up` \| `down` | **auto-placement** (inside if it fits, else outside-right) |

Backend setters validate and normalize:
`SetActionFontColor` / `SetDetailFontColor`,
`SetActionTextPos` / `SetDetailTextPos` (reject unknown positions; `"auto"` is
stored as `""`).

---

## 3. Color rules (`labelColor` + `contrastOn`)

`labelColor(bg, explicit)` picks the text fill:

1. **Explicit wins.** If `fontColor` is set, that color is used verbatim.
2. **Otherwise auto-contrast** against the bar's background `bg` via
   `contrastOn(bg)`.

`contrastOn(hex)` uses the W3C relative luminance of the background:

```
L = 0.2126·R + 0.7152·G + 0.0722·B      (each channel linearized)
return L > 0.179 ? "#111" : "#fff"
```

- Light background (L > 0.179) → near-black `#111`.
- Dark background (L ≤ 0.179) → white `#fff`.
- Unparseable color → defaults to white.

The background handed to `labelColor` is the bar's actual fill:
- Level-1 bar: `color || "#8a90dd"`.
- Detail bar: the role color if the detail has a role, else the default detail
  fill.

This is what keeps a label readable on a dark bar color — the previous
hard-coded white/`#333` labels were unreadable on dark backgrounds.

> One exception: a label drawn **outside** the bar (positions `right`, `up`,
> `down`, or the auto→right fallback) is not sitting on the bar color, so it
> uses the explicit `fontColor` if set, otherwise a neutral dark `#333` that is
> readable on the canvas background. See §4.

---

## 4. Position rules (`placeLabel`)

`placeLabel(g, x0, x1, midY, h, text, size, pos, insideColor, explicit)` draws
the label. `x0`/`x1` are the bar's left/right edges, `midY` its vertical center,
`h` its height. `insideColor` is the §3 auto-contrast color for labels that sit
on the bar.

Let `barW = x1 − x0` and `estW = measureText(text, size)` (see §5).

### 4.1 Auto (`textPos` empty or `"auto"`) — the default

```
place = (estW + 10 <= barW) ? "left" : "right"
```

- **Fits** (estimated width + 10px padding ≤ bar width) → drawn **inside**,
  left-aligned (same as `left`).
- **Does not fit** → drawn just to the **right** of the bar (same as `right`).

This is the rule the user asked for: "if the text fits inside the action it is
inside; if not, it is to the right of the action."

### 4.2 Explicit positions

| `textPos`          | x                 | y (baseline)        | `text-anchor` | On/Off bar | Fill          |
|--------------------|-------------------|---------------------|---------------|------------|---------------|
| `left`             | `x0 + 6`          | `midY + size/2 − 1` | `start`       | inside     | `insideColor` |
| `center`           | `(x0 + x1) / 2`   | `midY + size/2 − 1` | `middle`      | inside     | `insideColor` |
| `outleft`          | `x0 − 8`          | `midY + size/2 − 1` | `end`         | outside L  | outside color |
| `right` / `outright`| `x1 + 8`         | `midY + size/2 − 1` | `start`       | outside R  | outside color |
| `up`               | `x0`              | `midY − h/2 − 4`    | `start`       | above bar  | outside color |
| `down`             | `x0`              | `midY + h/2 + size` | `start`       | below bar  | outside color |

Where "outside color" = `explicit` (the set `fontColor`) or `#333`.

- **`outleft`** places the label OUTSIDE to the left of the bar, right-aligned
  so it **ends** at the bar's left edge (an 8px gap before the edge).
- **`outright`** places it OUTSIDE just past the bar's right edge, left-aligned.
  **`right`** is a legacy alias of `outright` (kept so older saved plans still
  load; the picker always writes `outright`).

`text-anchor` is only set when it differs from the default `start` (i.e. for
`center` → `middle` and `outleft` → `end`).

### 4.3 The 5-point picker (UI)

In the action editor and the detail panel the position is chosen with a small
graphic widget (`textPosPickerHTML` / `wireTextPosPicker` in `main.ts`): a mini
action-bar SVG with clickable dots — **up / outleft / left / center / outright /
down** (outleft and outright sit just outside the mini bar's left and right
edges) — plus an **Auto** checkbox. The active choice is highlighted. Saving
sends the value through `setActionTextPos` / `setDetailTextPos`.

> Implementation note: the picker must **not** be wrapped in a `<label>`. A
> `<label>` forwards any click inside it to its single form control, so clicking
> an SVG dot would also toggle the Auto control and snap the choice back to
> `auto`. The picker is a plain `<div>`; Auto is its own separate checkbox.

### 4.4 Summary (Sammelvorgang) bars

A summary rolls up and renders as the MS-Project black polygon; its label is
always drawn to the **right** of the rollup (`xR + 8`) in a fixed `#333`,
independent of the `textPos` setting.

---

## 5. Width measurement (`measureText`)

```
measureText(s, size) = s.length · size · 0.58
```

An average-glyph estimate (0.58em). SVG `getComputedTextLength` would be exact
but requires the text node to be laid out first; the estimate is enough to make
the inside-vs-right auto decision. Because it is only an estimate, the auto
threshold keeps a 10px padding margin (`estW + 10 <= barW`) so a label that is
borderline does not spill past the bar edge.

---

## 6. Where each rule is applied

All three bar renderers funnel their label through `placeLabel` + `labelColor`,
so the rules are identical across views:

| Renderer        | What it draws                         | Size |
|-----------------|---------------------------------------|------|
| `drawBar`       | level-1 action bar (normal + summary) | 11   |
| `drawDetail`    | main-view detail sub-row (Show Details)| 10  |
| `drawDrillBar`  | detail bar inside the drill plane     | 11   |

Role sub-rows (`drawRoleRow`, role mode) also use `labelColor` for contrast but
keep a fixed inside-left placement (they are not user-positionable).

---

## 7. Precedence summary

For any bar:

1. **Text** = `name` + non-zero `est`.
2. **Position**: explicit `textPos` if set; else auto (inside when it fits, else
   right).
3. **Color**: explicit `fontColor` if set; else auto-contrast against the bar
   color — but only for labels that sit on the bar. Off-bar labels use the
   explicit color or neutral `#333`.
