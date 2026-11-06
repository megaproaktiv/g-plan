# Glossary

> Generated from `src/app/glossary.go` (the in-program help shows the same text). Do not edit by hand:
> run `UPDATE_GLOSSARY=1 go test ./... -run GlossaryDoc` after changing the glossary.

The words used in this program and what they mean.

## Terms

### Section

*Also called:* Work Package

A named lane on the canvas that groups actions (and milestones). It has a number - 1, 2, 3 ... - which is the first part of the numbers of its actions (1.1, 1.2 ...), and it can have its own color. The planning name for a section is Work Package.

### Work Package

*Also called:* Section

The planning name for a section: a group of related actions that belong together and are added up together, for example in the Monitor table. A work package contains actions.

### Action

A piece of work to be done, drawn as a bar with a start, an end, an estimation and optional roles. An action can be broken down into detail actions (open them with the drill icon or Details...).

### Role

A named kind of worker or skill, for example Developer, that can be assigned to actions. The percentage of an assignment is the share of the working day the role spends on that action.

### Milestone

*Also called:* Gateway, Gate

A certain point in time, without duration, that is used to measure the success of the project. A milestone is reached or missed on its date.

### Estimation

*Also called:* planned effort

The planned amount of work for an action, in hours or days. It says how much work is planned, not how long the action takes (see Duration). The estimation of a work package is the sum of its actions.

### Work

*Also called:* real effort (field "Real")

The time actually spent on an action, in hours or days. It is entered by hand or - with monitoring on - taken from the bookings in the external time tracking system.

### Duration

The time between the start and the end of an action. It usually differs from the estimation: an action can last 10 days while the estimated work is 80 hours. The program also shows the working days of the duration (Monday to Friday, without weekends).

### Duration Complete

*Also called:* Duration Progress

The percentage of the duration that has gone by since the start of the action: days passed / days of the duration. It depends only on the calendar, not on how much work has been done.

*Example:* Day 2 of a 10-day action: Duration Complete = 2 / 10 = 20 %.

### Work Complete

*Also called:* Effort Progress

The percentage of the estimation that has been worked: Work / Estimation. It answers: how many of the planned work hours have gone into the action? In the program it is shown as "consumed" and never above 100 %.

*Example:* 32 of 80 estimated hours worked: Work Complete = 32 / 80 = 40 %.

*Note:* Duration Complete and Work Complete differ even when a project is exactly on plan. After the first 2 of 10 days only 20 % of the duration has passed, but with a strongly front-loaded plan, for example 40 % of the work is planned for that time.

### Task Complete

Only a definition - it is not shown in the program. The task is really finished: it has been tested against its requirements. The estimation is only an estimate, not the truth, so Task Complete can differ from Duration Complete and from Work Complete: all the estimated work may be spent and the task still not done, or the task may be done with less work than estimated.

*Note:* Not shown in the program.

## Terms used on screen

### Contour

The shape that says how the estimated work is spread over the duration of an action: Flat (evenly), Front-Loaded (more at the start), Back-Loaded (more at the end), Double-Peak (more at both ends). The area under the curve is the estimation. The contour None means the action has no estimation and no real effort: the fields are greyed out, the action has no progress bar and takes no bookings.

### Consumed

*Also called:* rating, rating with contour

Shown on the action bar and in the action view when Show Progress is on. The first value, for example 16.4 %, is Work Complete. If the contour is not Flat, a second value follows, for example "Double-Peak 12.6 %": the position on the duration up to which the contour's planned work equals the work done so far. If the black marker on the bar is left of the today line, the work is behind the plan.

### External ID

Up to 16 characters. It identifies an action in the booking texts of the external time tracking system: bookings that name it count as Work of this action. If a booking names several IDs, its time is split evenly between them. It can be changed at any time. An action with the contour None is not a real action and cannot carry an External ID.

### Default Action

The action that receives the bookings which name no External ID, so that no booked time is lost. It is chosen on the Monitor page.
