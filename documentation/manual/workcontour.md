# Work contour and progress

How pt-plan tracks an action's completion and draws it on the bar. Covers the
**Estimated/Real effort** split, the **Work Contour** (effort-distribution
shape), the integer **Progress Percentage**, the (non-linear) **Completion
Position**, and the project-level **Show Progress** toggle that gates the
on-screen marker and curve.

Scope: these rules govern the on-screen canvas only
(`src/app/frontend/src/gantt.ts`), backed by pure Go math in
`src/app/plan/progress.go`. The PDF/PNG export is unaffected.

---

## 1. Estimated effort vs. real effort

Every action already carries an **Estimated effort** (`est`, unchanged). This
feature adds a second value, **Real effort** (`realEst`): the effort actually
booked against the action so far. Both are integer amounts in the project's
display unit (days or hours), stored internally as minutes, in the inclusive
range `0` to `5,256,000` minutes (100 working years at 8h/day).

Each is edited on its own line in the action editor — the two-line
*Headline effort* — and each commits independently on Enter or blur. Rejecting
an edit (empty, non-numeric, negative, or over the maximum) leaves the stored
value untouched and marks the input invalid; it never partially writes.

A project file saved before this feature has no `realEst` key; loading it
yields `0`, which is the correct "no progress booked yet" default.

## 2. Work Contour: four effort-distribution shapes

Each action has a **Contour**, chosen from a four-value picker next to the
Headline effort:

```
Flat          constant effort rate across the duration
Front-Loaded  more effort spent early, tapering off
Back-Loaded   effort ramps up, most of it spent late
Double-Peak   effort spent at both ends, light in the middle
```

New actions default to `Flat`; an absent/unrecognized value on load also
normalizes to `Flat`. Picking an invalid value (anything other than an exact
match of the four names) is rejected and leaves the stored contour unchanged.

Sketch of each `Contour_Curve` shape as drawn inside the bar (height = effort
density, not time):

```
Flat           Front-Loaded      Back-Loaded       Double-Peak
 ______          \                      /           \        /
                   \__                _/              \__  __/
                      \______      __/                    \/
```

## 3. Progress Percentage (integer, always 0..100)

The **Progress Percentage** is computed from the two efforts using integer
arithmetic only — no floating point enters this calculation:

- `0` when Estimated effort is `0` (nothing to divide by).
- `100` when Real effort (clamped to `0` if negative) is at or above Estimated
  effort.
- otherwise `(real * 100) / est`, truncating toward zero.

This value is shown next to the Headline effort and is independent of the
Work Contour — the contour only affects *where on the bar* that progress is
drawn, never the percentage itself.

## 4. Completion Position: contour-shaped placement

The **Completion Position** is a fraction from `0.0` (start of the bar) to
`1.0` (end of the bar) that places the Progress Marker. It starts from the
same effort ratio as the percentage, `t = real / est`, but reshapes it through
the action's contour before turning it into a position:

The contour curve is the planned effort DENSITY over the duration (its height
at each point), so the planned effort for any stretch is the AREA under the
curve there. The marker sits at the point where the area under the curve, from
the start, equals the real-effort share already booked:

| Contour | Where the effort is planned | Effect on the marker |
|---|---|---|
| Flat | even across the duration | tracks the ratio exactly: `position = t` |
| Front-Loaded | a large share near the start | a lot of booking is needed before the marker moves far; with little real effort the marker lags BEHIND the ratio |
| Back-Loaded | a large share near the end | little booking is needed early on; with little real effort the marker runs AHEAD of the ratio, lagging only near the end |
| Double-Peak | large shares at both ends, little in the middle | the marker crawls near both ends and advances quickly through the middle |

This is the inverse of "the marker moves fastest where the curve is highest":
a high point in the curve means a lot of PLANNED effort sits there, so a lot of
REAL effort has to be booked to get the marker past it.

**Boundary behaviour**, true for every contour:

A worked example (10 days planned, Front-Loaded): to move the marker past the
first 20% of the duration takes roughly 6 of the 10 days of real effort,
because the front-loaded curve holds about 60% of the total planned effort in
that first fifth.

- Real effort `0` → position `0.0` (marker sits at the start).
- Real effort at or above Estimated effort → position `1.0` (marker sits at
  the end), even if real effort overshoots further.
- Estimated effort `0` → position `0.0` (no basis to place a marker).

All four shapes are monotonically increasing and pinned at both ends, so the
marker never moves backward as real effort grows and always lands exactly on
the start/end boundary in those cases.

## 4a. The contour None

**None** is a fifth choice in the contour picker. An action with the contour
None has **no estimation and no real effort**:

- the *Estimated* and *Real* fields in the action editor are **greyed out** and
  cannot be changed (the backend refuses it as well);
- its effective estimation and real effort are 0, so it adds nothing to the
  project total, the role load, the capacity and over-allocation warnings or the
  reports;
- it has **no progress bar**: no curve, no marker, no shaded area and no rating
  on its bar, and no consumed value in the action view;
- it takes **no bookings** in the monitoring and **cannot carry an External
  ID** (the field is greyed out; an action that has one cannot be switched to
  None until the ID is removed). It is not synchronized and cannot be the
  default action (and the default action cannot be switched to None).

The numbers that were stored before are **kept**, only hidden: switching the
contour back to Flat (or another kind) restores the estimation and the real
effort. Use None for actions that are only a point or a container in the plan
and carry no work of their own.

## 4b. The two consumed ratings

Every action has two ratings of how much of it is consumed, shown to one decimal
(tenths of a percent, integer math; 164 = 16,4 %):

| Rating | Meaning |
|---|---|
| **Pure** | real effort ÷ planned effort. 52,5 h of 320 h = **16,4 %**. Does not depend on the contour. |
| **With contour** | how far along the planned curve the real effort has got: the Completion Position (§4) as a percentage. |

With the **Flat** contour both are the same. With another contour the same
effort rates differently: for 16,4 % effort a Front-Loaded action is at only
**12,3 %** (its early part needs a lot of effort), a Double-Peak action at
**12,6 %**, a Back-Loaded one at **34,2 %**.

Where the curve shapes come from: the drawn curves are the planned density. The
Front-Loaded curve peaks at 10 % of the duration (and tapers to the end), the
Back-Loaded one at 90 %, the Double-Peak one at 10 % and 90 % with a trough at
the middle. With the Front-Loaded curve about 29 % of the effort lies in the
first 20 % of the duration.

## 4c. Where the real effort comes from

An action is **synchronized** when monitoring is on (see `monitoring.md`) and
the action has an External ID or is the default action. Its real effort is
then written by every synchronization (hours booked in the external system), is
shown **read-only** in the action editor with a note saying where it came from
and when, and drives both ratings. Every other action keeps its **manual** real
effort, which is editable. Switching monitoring off frees the fields again (the
last values stay).

## 5. Show Progress: the project-level toggle

**Show Progress** is a project view option (next to Show Details / Show Role
Actions / Show Phases) that gates whether the Progress Marker and Contour
Curve are drawn at all. It defaults **off**; a project file predating the
feature loads with it off. When on, every non-summary action bar draws:

- the **Contour Curve** (bars 8px or wider only) — a thin line tracing the contour's effort-density
  shape across the full bar width, in a colour chosen to differ from the bar
  background by at least half the luminance range;
- the **shaded area** under the curve from the start up to the marker: that
  area *is* the effort done, so its share of the whole area under the curve
  equals the pure rating, and the marker's position along the bar is the
  contour rating;
- the **ratings in the label**, for example `16.4% (Double-Peak 12.6%)`
  (just `16.4%` for Flat), and a tooltip with planned and real hours;
- the **Progress Marker** — a vertical tick at the bar's x-position
  corresponding to the Completion Position, in a colour guaranteed at least a
  3:1 contrast ratio against the bar background.

Bars narrower than 8px omit the contour curve (the marker is still drawn), and
summary (rollup) bars draw neither — the bar itself always renders normally.
