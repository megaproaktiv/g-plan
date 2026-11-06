# Dependency relationships

> **CHANGE LOCK (since v0.2.0).** The seven relationship kinds below and their
> timing rules are a **locked, stable contract**. Do not change a rule's
> comparison, rename/add/remove a kind, or alter the touching-endpoint
> semantics without a deliberate decision. Any change must be mirrored in:
> `src/app/plan/model.go` (`DepViolated`), the `RelationshipRules` binding
> (`src/app/i18n.go`), the link abbreviations (`kindAbbr` in `src/app/i18n.go`
> and `src/app/frontend/src/main.ts`), the help pages, and the Go test
> `TestAllRelationshipRules`. This file is the canonical reference.

A dependency links two tasks. In a link, **A** is the predecessor (the link's
*From*) and **B** is the successor (the link's *To*). Endpoints may be actions,
detail sub‑actions, or milestones (a milestone is a zero‑duration point).

`Mid(x)` is the midpoint of a task's span: `start + (end − start) / 2`.

Touching endpoints **satisfy** the rule (the comparisons are `≥` / `≤`, not
strictly greater/less). A link whose rule is breached is drawn with a **red**
arrow and flagged in the right‑pane *Links* table.

## The seven kinds

| Relationship | English | German | Abbr (EN / DE) | Rule | Example |
|---|---|---|---|---|---|
| Finish‑to‑Start | Finish-to-Start | Ende‑Anfang (Normalfolge) | FS / EA | `Start(B) ≥ End(A)` | Implementation starts after the design is done. |
| Start‑to‑Start | Start-to-Start | Anfang‑Anfang (Anfangsfolge) | SS / AA | `Start(B) ≥ Start(A)` | Testing starts once development has started. |
| Finish‑to‑Finish | Finish-to-Finish | Ende‑Ende (Endfolge) | FF / EE | `End(B) ≥ End(A)` | Documentation finishes when development finishes. |
| Start‑to‑Finish | Start-to-Finish | Anfang‑Ende (Sprungfolge) | SF / AE | `End(B) ≥ Start(A)` | The old system runs until the new one goes live. |
| Middle‑to‑Start | Middle-to-Start | Mitte‑Anfang | MS / MA | `Start(B) ≥ Mid(A)` | Integration starts once half the components are built. |
| Middle‑to‑End | Middle-to-End | Mitte‑Ende | ME / ME | `End(B) ≥ Mid(A)` | The interim report can't be finished before half the analysis is done. |
| Middle‑to‑Middle | Middle-to-Middle | Mitte‑Mitte | MM / MM | `Start(A) ≤ Mid(B) ≤ End(A)` | Hypercare peaks while the migration is still running. |

## Notes

- **FS semantics.** As of v0.2.0, FS is the literal `Start(B) ≥ End(A)` —
  B starting *exactly* when A ends is allowed (touching). (Earlier builds used a
  "+1 unit gap"; that is no longer the rule.)
- **MM bounds.** MM checks that **B's midpoint** falls within A's span. It is
  violated when `Mid(B) < Start(A)` (B too early) **or** `Mid(B) > End(A)`
  (B too late).
- **`OV` is a legacy alias of `MM`.** Older project files that stored the
  `OV` ("overlap") kind load and validate exactly as `MM`.

## Creating a link (anchor → kind)

Graphically, the kind is inferred from which anchor the link is drawn from/to.
`l` = the start edge, `r` = the end edge, `t`/`b` = the middle (top/bottom):

| From anchor (A) | To anchor (B) | Kind |
|---|---|---|
| `r` (end) | `l` (start) | FS |
| `l` (start) | `l` (start) | SS |
| `r` (end) | `r` (end) | FF |
| `l` (start) | `r` (end) | SF |
| `t`/`b` (middle) | `l` (start) | MS |
| `t`/`b` (middle) | `r` (end) | ME |
| `t`/`b` (middle) | `t`/`b` (middle) | MM |

A link that touches a **milestone** is always FS (a point link).

In the command bar you can also type `link <A> <B> [FS|SS|FF|SF|MS|ME|MM]`
(the command language is always English).

## Language

The GUI language (English / Deutsch, set in Project settings) controls the
abbreviation shown in the *Links* table and the relationship **help** page
(the `⇄` icon in Help): English shows `FS SS FF SF MS ME MM`, German shows
`EA AA EE AE MA ME MM`. The rules themselves are language‑neutral.
