# Monitoring

> Generated from `src/app/monitoringhelp.go` (the in-program help shows the same text). Do not edit by hand:
> run `UPDATE_GLOSSARY=1 go test . -run GuideDoc` after changing the help.

Monitoring compares your plan with the hours that were really booked in an external time tracking system (Aerion today). It only reads from that system and never changes anything in it.

## How it fits together

Three things connect your plan with the bookings:

- **System** and **project ID** (set once per project on the **Monitor** page): which time tracking system, and the number of the project in it.
- **External ID** (set per action in the action editor): a short label of up to 16 characters, for example `WP2`.
- The **booking text**: whoever books time writes the External ID into the text of the booking in the external system.

A synchronization reads all bookings of the external project from the project start to the project end and looks for the External IDs of your actions in the booking texts. The booked time becomes the **real effort** (Work) of the action, and the plan shows how much of the estimation is consumed.

## How the External ID is interpreted

The External ID is searched **in the text of every booking**, and nowhere else. A booking belongs to an action when its text contains the action's External ID:

- **As a word of its own.** The ID counts only when the character right before and right after it is not a letter or a digit. `WP1` is found in `WP1 kickoff`, in `(wp1)` and in `design / WP1 / review`, but not in `WP12` and not in `xWP1`.
- **Capital letters and hyphens do not matter.** `WP1`, `wp1`, `Wp1`, `WP-1` and `wp-1` are all the same ID, in the ID field and in the booking text. A space does not count as a hyphen: `WP 1` is not `WP1`.
- **Several IDs in one booking** (for example `WP2 and WP3 pairing session`): the booked time is **split evenly** between those actions. Seconds that do not divide evenly go to the first ones mentioned, so no time is lost.
- **The same ID twice** in one text counts once. An ID that only occurs inside a longer ID (the `1` in `Phase 1`) does not count as its own.
- **No ID in the text:** the booking is *unassigned*. It is counted for the **default action** if you chose one, otherwise it only appears on the line *Bookings without a matching External ID*.
- **Which time:** the tracked duration of each booking. Only work bookings are read, not breaks or working-time entries.
- **Rules for the ID itself:** at most 16 characters, unique in the project (capital letters and hyphens ignored: `WP1` and `WP-1` count as the same), and an action with the contour **None** cannot carry one. It can be changed at any time, also while monitoring is on.

Which of these booking texts are found for an action with the External ID `WP1`?

| Booking text | Found for WP1? | Why |
|---|---|---|
| `WP1 kickoff meeting` | yes | the ID stands as a word of its own |
| `wp1: design` | yes | capital letters do not matter |
| `(WP1)` | yes | brackets and punctuation are fine |
| `design / WP1 / review` | yes | it may sit in the middle of the text |
| `WP12 review` | no | WP12 is a different, longer ID |
| `xWP1 review` | no | a letter directly before it: not a separate word |
| `WP-1 kickoff` | yes | a hyphen inside the ID does not matter: WP-1 is WP1 |
| `WP 1 kickoff` | no | a space is not a hyphen: WP 1 is two words |
| `Daily standup` | no | no ID in the text |

> Tip: choose IDs that do not occur in everyday words, such as `WP1`, `WP2`, `WP3`, and ask the team to write them at the beginning of the booking text.

## Step by step, with an example

The example project is *Skynet MVP*, planned in days with 8 hours per day. Its project number in Aerion is `555555`. It has three actions: **Project Management** (External ID `WP1`, 40 days planned), **AWS Landing Zone** (`WP2`, 10 days = 80 h) and **ACC Core configuration** (`WP3`, 12 days = 96 h).

1. **Open the project** and check its start and end date. Bookings are fetched from the project start to the project end.
2. Click **Monitor**, then **Configure System…**. Enter the base URL, user name, password, client ID and client secret of Aerion and save. They are stored only for you, in `~/.pt-pm/config.yaml`, and are never shown again.
3. Still on the **Monitor** page, choose the system **Aerion** and enter the project number (`555555`). Click **Test**: *Connected: project …* shows that the credentials and the number are right.
4. **Give every action an External ID.** Open the action (double-click), type `WP1`, `WP2` or `WP3` into **External ID**, and press Enter or click elsewhere. Tell the team to write the ID into the text of their bookings.
5. **Optional: choose the Default action** for bookings without an ID, so that no booked time is lost. In the example it is *Project Management*.
6. **Optional:** tick *also one month before the project start* or *also one month after the project end* if bookings begin early or run over.
7. Switch **Monitoring on**. This needs a system and a project number.
8. Click **Synchronize**. A progress bar and the number of bookings fetched are shown.
9. **Read the result.** The table shows, per section, the estimation of the tracked actions against the actual effort, the **Remaining** effort (estimation minus actual, negative when over budget) and **Remaining %** (its share of the estimation), the project total, and the line for unassigned bookings. With *Show Progress* switched on, the bars show the consumed percentage, and the action view shows the planned and actual effort. **Export** (next to Synchronize) writes the table as two markdown files: `<name>-compact.md` with the work packages (sections) and the project total, and `<name>-detailed.md` which also lists the tracked actions of each work package.
10. **Correct and repeat.** If bookings are unassigned or in the wrong place, fix the booking text in the external system or the External ID, then synchronize again. Each synchronization overwrites the real effort of the tracked actions.

These are the bookings in the external system, and what the program makes of them:

| Booking text | Time | Counted for |
|---|---|---|
| `WP2 VPC design` | 3 h | WP2 3 h |
| `wp2: security review` | 2 h | WP2 2 h |
| `WP2 and WP3 pairing session` | 4 h | WP2 2 h, WP3 2 h |
| `WP3 account structure` | 5 h | WP3 5 h |
| `Daily standup` | 1 h | Project Management: 1 h |

After the synchronization the actions show:

| Action | Planned | Actual | Consumed |
|---|---|---|---|
| AWS Landing Zone (WP2) | 10 days (80 h) | 0,88 days (7 h) | 8.7% |
| ACC Core configuration (WP3) | 12 days (96 h) | 0,88 days (7 h) | 7.2% |
| Project Management (WP1) | 40 days (320 h) | 0,13 days (1 h) | 0.3% |

> *Consumed* is the actual effort divided by the planned effort. Project Management also received the 1 h of the booking without an ID, because it is the default action.

## If it does not add up

- **Nothing is assigned to an action:** the External ID is not in the booking text, or it is not a separate word. `WP 1` (with a space) is not `WP1`.
- **Everything is unassigned:** no action has an External ID yet, or the project number is wrong. Use **Test** on the Monitor page.
- **Too many unassigned bookings:** ask the team to add IDs, or choose a default action.
- **The Real field of an action is greyed out:** tracked actions (with an External ID, or the default action) get their real effort from the synchronization. To enter it by hand, remove the External ID or switch monitoring off.
- **An action does not take part at all:** the contour **None** means no estimation, no real effort and no External ID. The **Phases** lane is a view, not a work package: it is ignored by monitoring and its actions need no External ID.
- **"Aerion is not configured":** open **Monitor** and **Configure System…**.
- **The result disappeared:** changing an External ID, the default action or the period clears the last result. Synchronize again.
