# Monitoring: real effort from an external system

> A step-by-step guide with a worked example, and how the External ID is
> interpreted, is in the in-program help (the `⏱` tab) and in
> [monitoring-guide.md](monitoring-guide.md) / [monitoring-guide.de.md](monitoring-guide.de.md).

How pt-plan compares the plan with real effort booked in an external time
tracking system. The first supported system is **Aerion** (CoffeeCup). Other
types (CSV) are listed in the pull-down but greyed out until implemented.

Everything here is **read-only** towards the external system: pt-plan only
reads bookings (HTTP GET) and never creates or changes anything there.

---

## 1. The three components

| Component | Where it lives | Meaning |
|---|---|---|
| **Driver / system type** | project (`monitor.system`) | which external system: `aerion` (CSV not yet) |
| **Project ID** | project (`monitor.projectId`) | the project's id in that system (Aerion: a number) |
| **Work package ID** | action (`externalId`) | text that identifies an action inside the booking line |

The credentials of a system are **not** part of the project file. They are
stored per user in `~/.pt-pm/config.yaml` (directory mode 0700, file mode 0600).

## 2. Workflow

1. Open the project.
2. Click **Monitor**. Choose the system, enter the project ID, switch
   **Monitoring on**. Both system and project ID are mandatory; the switch is
   refused with a message until they are set. Use **Test** to check that the
   credentials work and that the project exists.
3. In each action's editor, enter an **External ID** (up to 16 characters,
   unique within the project). It can be changed at any time, **also while
   monitoring is on**: the last synchronization result is then cleared, the next
   synchronization uses the new ID, and the real effort follows at that
   synchronization. A refused value (already used, too long) is explained in
   words next to the field.
   An action with the contour **None** is not a real action and **cannot carry
   an External ID**: its field is greyed out, and switching an action that has
   an ID to None is refused until the ID is removed.
4. In the Monitor page, the list of sections (work packages) shows a green dot
   for every section in which at least one action has an External ID.
5. Click **Synchronize**. A progress bar and the number of entries fetched are
   shown. The result is a table with one row per section, the whole project as a
   sum, and a line for bookings that matched no External ID.
6. Optional: click **Export** to write the result as markdown (see 5a).

The **Phases** lane is a view, not a work package, so monitoring ignores it: its
actions need no External ID and do not appear in the section list or the result
table, count in no total, cannot match bookings and cannot be the default action.
Their real effort stays manually editable.

### Configure System (Aerion)

The **Configure System** button opens a form with exactly the entries of the
`aerion:` section of the reporting tool's `config.yaml`:

```yaml
aerion:
  base_url: "https://your-company.aerion.cloud"
  username: "..."
  password: "..."
  client_id: "..."
  client_secret: "..."
```

Saved to `~/.pt-pm/config.yaml`; other top-level keys in that file are kept.
The password and client secret are never sent back to the screen: the form shows
only whether one is stored, and leaving a secret field empty keeps the stored
value. The base URL must be `https` (plain `http` only for localhost).

## 3. What is fetched

For the project ID, **all single-line bookings from the project's start date to
its end date** are fetched (optionally widened, see below) from `GET /v1/time-entries` (Aerion API reference,
TimeEntry), filtered to `trackingType = WORK`, sorted by id and read page by
page (`skip` / `limit`) until the `meta.total` reported by the API is reached.
(The older reporting code reads a single page of at most 5000 entries; pt-plan
pages, so long projects are complete.)

**Period options.** On the Monitor page two check boxes extend the period:
*also one month before the project start* and *also one month after the project
end* (project settings `monitor.includeBefore` / `monitor.includeAfter`, both off
by default). Bookings often begin early or run over, so this makes them visible.
The page shows the exact dates that will be fetched, and the result table
reports the range that was actually used. The month is added keeping the day of
month, clamped to the last day of the target month (31 March minus one month is
28 February). Changing the options clears the previous result, and they cannot
be changed while a synchronization is running.

Per booking the used fields are `id`, `day`, `duration` (seconds actually
tracked) and `comment` (the booking text).

## 4. How a booking finds its action

The work package ID is searched **in the booking text** (`comment`). An action's
External ID matches when it occurs in the text:

- case-insensitive, and **hyphens do not matter**: `WP1`, `wp1`, `Wp1`, `WP-1` and
  `wp-1` are the same ID, in the ID field as well as in the booking text. A space
  is not a hyphen: `WP 1` does not match `WP1`. Because of this, two actions cannot
  carry `WP1` and `WP-1` (the second is refused as already used);
- as a whole token: the characters directly before and after must not be
  letters or digits, so `WP-1` does not match inside `WP-12` or `xWP-1`;
- an occurrence that lies completely inside a longer matching ID (for example
  `1` inside `Phase 1`) is not a reference of its own;
- **if several External IDs occur in one text, the time is spread evenly over
  those actions.** Each action gets the same share; seconds that do not divide
  evenly go to the first references in text order, so nothing is lost or
  invented. A booking naming the same ID twice counts once for it. In the
  result table a booking is counted on every action it references, while the
  project line counts each booking once.

Bookings that match no External ID belong to no action. What happens to them is
set by the **Default action** on the Monitor page (project setting
`monitor.defaultAction`, the id of an **action**; the list shows the actions
grouped by section):

- **an action is chosen:** all unassigned bookings are counted as that action's
  effort. Its row in the table shows the total (matched plus unassigned) and a
  second line *of which unassigned bookings (default action)*. The
  default action counts as tracked even when it has no External ID (it is then
  labelled "(default)"), so its estimate is compared with what it received. The
  hours are included in its section's and the project's *Actual*. Nothing is
  lost: matched + default = everything fetched.
- **none (default):** they are shown only on the line *Bookings without a
  matching External ID* and are not part of any action.

Any action can be chosen except those with the contour *None* (they take no bookings) and those in the *Phases* lane, which is a view and
not a work package. It can be changed while monitoring is on. If the chosen
action is deleted, the synchronization stops with a message until another one is
chosen. Matched bookings are never redirected to the default action.

Actions with the contour **None** have no estimation and no real effort and
cannot carry an External ID, so they take no part in the matching: they are not
synchronized and cannot be the default action. (Older files that still have an
ID on a None action are repaired when loaded: the ID is dropped, it never had an
effect.)

## 5. The result table

| Column | Meaning |
|---|---|
| Section | the section; indented rows below list its tracked actions |
| Tracked actions | actions with an External ID plus the default action / all actions in the section |
| Estimated (tracked) | estimated effort of the actions that have an External ID |
| Estimated (all) | estimated effort of all actions in the section |
| Actual | booked effort matched to the section's tracked actions |
| Remaining | *Estimated (tracked)* minus *Actual*; negative means over budget |
| Remaining % | *Remaining* as a whole percent of *Estimated (tracked)*; empty without an estimation |
| Entries | number of matched bookings |

"Actual" is comparable with "Estimated (tracked)". Efforts are shown in the
project's unit (days or hours); booked seconds are summed first and converted to
minutes once, so rounding errors do not add up. The last result is kept until
the project, the monitor setting or an External ID changes.

Remaining and Remaining % are shown for every section, for each tracked action
below it (the action row also shows its own estimate in *Estimated (tracked)*)
and for the project line. Example: 10 days estimated, 8,7 days booked gives
Remaining 1,3 and Remaining 13 %; 10 days estimated, 12 days booked gives
Remaining -2 and Remaining -20 %.

### 5a. Export as markdown

The **Export** button on the Monitor page (enabled once there is a result) opens a
save dialog for a file name, for example `Demo-monitoring.md`, and writes two files
next to each other:

| File | Content |
|---|---|
| `<name>-compact.md` | only the work packages (sections) and the project sum |
| `<name>-detailed.md` | the same, plus the tracked actions below each work package (External ID, default action marked) |

Both contain the columns *Estimated (tracked)*, *Actual*, *Remaining*,
*Remaining %* and *Entries*, the period that was fetched and the line for bookings
without a matching External ID (when it applies). Existing files with these names
are overwritten.

### The action view

With monitoring on, the small view of an action shows, besides the planned
effort in the project unit with its hours and days, the **actual effort** of the
last synchronization in hours and days, split into

- the bookings **assigned to the action's External ID** ("assigned to WP1"),
- the **unassigned** bookings credited to it when it is the default action.

Example (an action planned with 40 days = 320 h, External ID `WP1`, also the
default action; one booking of 4 h names WP1, 13 bookings have no
reference):

```
Planned effort: 40 days (320 h · 40 days)
Actual effort: 52,5 h (6,56 days) · 14 bookings
  - assigned to "WP1": 4 h (0,5 days) · 1 booking
  - unassigned (default action): 48,5 h (6,06 days) · 13 bookings
```

Actions without an External ID that are not the default show "not tracked"; before
the first synchronization "not synchronized yet".

## 6. Saved with the project

`monitor` (`enabled`, `system`, `projectId`) and the actions' `externalId`.
Files from earlier versions load unchanged (monitoring off, no External IDs).
The synchronized actions' real effort (`realEst`) is written by each
synchronization (see `workcontour.md` §4c); `monitor.syncedAt` records when.
Actions that are not synchronized keep their manually entered real effort.

## 7. Testing against the live system

The unit tests use a fake server. A read-only test against the real system is
skipped unless requested:

```
PTPM_LIVE=1 PTPM_LIVE_PROJECT=<aerion project id> go test ./extsys -run Live -v
PTPM_LIVE_PTPLAN=<file.ptplan> go test . -run LiveProjectFile -v   # a whole project file (read only, never saved)
```

It uses the credentials in `~/.pt-pm/config.yaml`, checks the project, fetches
all bookings of 2026, repeats the fetch with small pages to prove paging loses
nothing, and matches a token taken from the real booking texts.
