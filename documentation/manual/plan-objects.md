# Sections and actions

> Generated from `src/app/objecthelp.go` (the in-program help shows the same text). Do not edit by hand:
> run `UPDATE_GLOSSARY=1 go test . -run ObjectGuideDoc` after changing the help.

A plan is built from sections, which group actions. This page explains both, and the settings of an action, above all its contour.

## Sections

A **section** is a named lane on the canvas that groups related actions and milestones. The planning name for a section is *work package*.

- **Number.** Sections are numbered 1, 2, 3 ... from top to bottom. The number is the first part of the numbers of its actions: the second action of section 3 is `3.2`. The numbers follow the order on the screen, so they change when you change the order.
- **Color.** Each section can have its own color.
- **Create.** Click **New Section** in the menu of the empty canvas, or type `add section <name>` in the command bar. It is added at the end (after the last WP number). The new section becomes the current one, and new actions go into the current section.
- **Rename and delete.** Select the section (click its label) to rename it, change its color or delete it. Deleting a section does not delete its actions: they move to the default lane. Deleting can be undone with Cmd-Z.
- **Order.** The section can be moved up and down with the **▲ up** and **▼ down** buttons of the section, and with **Move Section Up / Down** in the right-click menu of the section label. This works for every section, **also for a section without actions**. The order is stored with the plan. The numbers of all actions follow.
- **The Phases lane** at the top is a view of the project phases, not a section. It has no number, cannot be moved and cannot be deleted. It can be switched on and off in the project settings.

> A section does not have an estimation of its own. Its effort is the sum of its actions.

## Actions

An **action** is a piece of work with a start, an end, an estimation and, when it is under way, a real effort. It belongs to one section. Double-click an action, or choose **Edit Action** in its right-click menu, to open its editor.

| Field | Meaning |
|---|---|
| **Name**, **Description** | What the action is. The name is shown on the bar. |
| **Start**, **End** | The duration: the time span of the bar. Drag the bar to move it, drag its ends to resize it. A move can be undone with Cmd-Z. |
| **Estimated** (the Planned effort) | The work you plan to spend, in the unit of the project. It does not depend on the duration. In the command bar: `est 3.1 20`. |
| **Real** | The work really spent so far. It is entered by hand, or, when monitoring is on and the action is tracked, taken from the time bookings and greyed out. |
| **Contour** | How the estimated work is spread over the duration. See the next section. |
| **External ID** | Up to 16 characters that identify the action in the booking texts of the time tracking system. See the monitoring help. |
| **Details** | Sub-actions with their own estimation. Typing an estimation on the action itself makes that value the fixed one, and the details share it. |

## The contour settings of an action

The **contour** is the planned *density* of work over the duration of the action: the height of the curve is the planned work per day at that point. The **area** under the curve is the estimation. So the contour tells where along the bar you expect the work to happen. It never changes the estimation, only where the progress marker is drawn.

| Contour | Shape | Use it when ... |
|---|---|---|
| **Flat** | Constant height. The work is spread evenly. | the work is about the same every day. This is the default. |
| **Front-Loaded** | High at the start, tapering off to the end. | most of the work is done early, for example set-up or analysis. |
| **Back-Loaded** | Low at the start, rising towards the end. | the work builds up and most of it happens late, for example testing and roll-out. |
| **Double-Peak** | High at both ends, light in the middle. | there is a busy start and a busy end with a quiet time in between. |
| **None** | No curve. | the entry is not real work, but only a placeholder, for example a heading. See below. |

**How the progress is shown (Show Progress).** When **Show Progress** is on, the bar of an action shows two things:

- The **consumed** share: real effort divided by estimation, for example `16.4 %`. This is the *Work Complete* of the action.
- If the contour is not Flat, a second value follows, for example `Double-Peak 12.6 %`. It is the position on the duration up to which the **planned** work, following the curve, equals the work done so far. The black **marker** on the bar is drawn there.

Compare the marker with the *today* line. With a **Front-Loaded** contour much of the work is planned early, so the marker lags behind the share consumed: a lot of work must be done before it moves far. With a **Back-Loaded** contour the marker runs ahead. If the marker is left of today, the action is behind its plan. With **Flat** the marker is simply at the consumed share.

**The contour None.** An action with the contour **None** has no estimation and no real effort: both fields are greyed out and the stored numbers are kept (they come back when you switch to another contour). It has no progress marker and no curve, it takes no time bookings, cannot carry an External ID and cannot be the default action. Use it for entries that only mark or group something.

> Changing the contour changes how the progress is shown, but never the estimation, the real effort or the totals of the plan.
