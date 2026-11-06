package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// Chat is the command-bubble interface (Epic 4). It mutates the same model the
// graphical editor uses (requirement R-10.2) and is an accessible, mouse-free
// alternative.
//
// It is MODE-BASED around a central object. The current mode is one of
// action | role | section. A bare verb acts on the current mode and,
// where relevant, the current selected object:
//
//	mode action|role|section       switch the central object
//	add action|section|role|milestone|detail <name> ...   add an object (the kind is required)
//	use <id|name>                  select the current object
//	move <n><unit>                 shift current action by +/- days/weeks (e.g. move 3d, move -1w)
//	lengthen <n><unit>             extend current action's end
//	shorten <n><unit>              reduce current action's end
//	move <id> to <date>            explicit move of an action
//	resize <id> to <date>          explicit end change
//	est <id> <value>               set estimation
//	add action <name> [from <d>] [to <d>] [est <v>] [after <id>]
//	assign <role> to <wpID> <pct>%
//	help
func (a *App) Chat(line string) (string, error) {
	fields := tokenize(line)
	if len(fields) == 0 {
		return "", nil
	}
	verb := strings.ToLower(fields[0])

	switch verb {
	case "help", "?":
		return a.ChatHelp(), nil

	case "mode":
		if len(fields) < 2 {
			return "current mode: " + a.mode, nil
		}
		m := strings.ToLower(fields[1])
		switch m {
		case "action", "role", "section", "milestone":
			a.mode = m
			a.current = ""
			return "mode: " + m, nil
		default:
			return "", fmt.Errorf("mode must be action|role|section|milestone")
		}

	case "use", "select":
		if len(fields) < 2 {
			return "", fmt.Errorf("usage: select <action|section> <ref>  (or  use <id|name>)")
		}
		// `select <kind> <ref>` explicitly selects an action or a section and
		// shows a border around it. `select <ref>` / `use <ref>` select in the
		// current mode (back-compat).
		if verb == "select" && len(fields) >= 3 &&
			(strings.EqualFold(fields[1], "action") || strings.EqualFold(fields[1], "section")) {
			kind := strings.ToLower(fields[1])
			if _, err := a.Select(kind, fields[2]); err != nil {
				return "", err
			}
			return fmt.Sprintf("selected %s %s", kind, fields[2]), nil
		}
		// In action mode, resolve #n / n / n.k / internal-id to a real action
		// id so relative commands (move/lengthen/shorten) can find it. In other
		// modes the current object is referenced by name.
		if a.mode == "action" {
			if id := a.resolveActionRef(fields[1]); id != "" {
				a.current = id
				a.selKind, a.selID = "action", id // also show the selection border
				return a.mode + " selected: " + id, nil
			}
			return "", fmt.Errorf("action %q not found", fields[1])
		}
		if a.mode == "section" {
			if name := a.resolveSectionRef(fields[1]); name != "" {
				a.current = name
				a.selKind, a.selID = "section", name
				a.currentSection = name // new actions go into the selected section
				return "section selected: " + name, nil
			}
		}
		a.current = fields[1]
		return a.mode + " selected: " + a.current, nil

	case "add":
		return a.chatAdd(fields)

	case "move":
		// Forms:
		//   move <id> after <id>     place <id> to start right after another
		//   move <id> before <id>    place <id> to end right before another
		//   move <id> to <date>      move <id> so it starts on <date>
		//   move <id> <n>d|w left|right|future|past    relative shift
		//   move <n>d|w              relative shift of the CURRENT action
		// <id> may be an action or a detail sub-action (n.k / n.k.m).
		if len(fields) == 2 && isDelta(fields[1]) { // move 3d  (current)
			return a.shiftCurrent(fields[1])
		}
		if hasKeyword(fields, "after") {
			return a.moveRelativeTo(pick(fields, 1, a.current), afterKeyword(fields, "after"), "after")
		}
		if hasKeyword(fields, "before") {
			return a.moveRelativeTo(pick(fields, 1, a.current), afterKeyword(fields, "before"), "before")
		}
		if hasKeyword(fields, "to") {
			return a.moveTaskTo(pick(fields, 1, a.current), afterKeyword(fields, "to"))
		}
		// directional delta: move <id> 10d left|right|future|past
		if dir := lastDirection(fields); dir != "" {
			return a.moveTaskDelta(pick(fields, 1, a.current), fields, dir)
		}
		// fallback: move <id> <n>d|w (default = right/future)
		if d := lastDelta(fields); d != "" {
			return a.moveTaskDelta(pick(fields, 1, a.current), fields, "right")
		}
		return "", fmt.Errorf("usage: move <id> after|before <id> | move <id> to <date> | move <id> <n>d left|right|future|past")

	case "lengthen", "extend":
		if len(fields) < 2 {
			return "", fmt.Errorf("usage: lengthen <n>d|w")
		}
		return a.resizeCurrent(fields[1], +1)

	case "shorten":
		if len(fields) < 2 {
			return "", fmt.Errorf("usage: shorten <n>d|w")
		}
		return a.resizeCurrent(fields[1], -1)

	case "resize":
		id := pick(fields, 1, a.current)
		to := afterKeyword(fields, "to")
		wp := a.project.WorkPackage(id)
		if wp == nil {
			return "", fmt.Errorf("action %q not found", id)
		}
		if _, err := a.MoveWorkPackage(id, wp.Start.Format(plan.DateLayout), to); err != nil {
			return "", err
		}
		return fmt.Sprintf("resized %s end to %s", id, to), nil

	case "est":
		// est <id> <value> — <id> may be #n, n, a WBS code (1.1 action or
		// 1.1.1 detail sub-action) or an internal id. Detail sub-actions are
		// actions too, so estimation works on them as well.
		ref := pick(fields, 1, a.current)
		val := fields[len(fields)-1]
		taskID := a.resolveTaskRef(ref)
		if taskID == "" {
			return "", fmt.Errorf("task %q not found (use #n, n, a WBS code like 1.1 or 1.1.1, or a name)", ref)
		}
		if d, wp := a.project.DetailByID(taskID); d != nil {
			if _, err := a.SetDetailEstimation(wp.ID, taskID, val); err != nil {
				return "", err
			}
			return fmt.Sprintf("set detail %s estimation to %s", ref, val), nil
		}
		if _, err := a.SetWorkPackageEstimation(taskID, val); err != nil {
			return "", err
		}
		return fmt.Sprintf("set %s estimation to %s", ref, val), nil

	case "milestone", "ms":
		// milestone <name> [on <YYYY-MM-DD>] [in <section>]
		// Creates a milestone (main level only). Defaults: today-ish (project
		// start) + current section.
		if len(fields) < 2 {
			return "", fmt.Errorf("usage: milestone <name> [on <date>] [in <section>]")
		}
		name := fields[1]
		date := afterKeyword(fields, "on")
		if date == "" {
			if s, ok := a.project.Start(); ok {
				date = s.Format(plan.DateLayout)
			} else {
				date = time.Now().Format(plan.DateLayout)
			}
		}
		section := afterKeyword(fields, "in")
		if _, err := a.AddMilestone(name, date, section); err != nil {
			return "", err
		}
		return fmt.Sprintf("added milestone %q on %s", name, date), nil

	case "link":
		// link <a> <b> [FS|SS|FF|SF]
		// Endpoints may be actions OR detail sub-actions (WBS n.k or n.k.m).
		if len(fields) < 3 {
			return "", fmt.Errorf("usage: link <a> <b> [FS|SS|FF|SF]")
		}
		from := a.resolveTaskRef(fields[1])
		to := a.resolveTaskRef(fields[2])
		if from == "" || to == "" {
			return "", fmt.Errorf("could not resolve task references %q %q", fields[1], fields[2])
		}
		kind := "FS"
		if len(fields) >= 4 {
			kind = strings.ToUpper(fields[3])
		}
		if _, err := a.AddDependency(from, to, kind, "r", "l"); err != nil {
			return "", err
		}
		return fmt.Sprintf("linked %s %s %s", fields[1], kind, fields[2]), nil

	case "assign":
		// assign <role> to <wp> [<pct>%]
		//   <role> may be an R-number (R1 / 1) or a role name.
		//   <wp>   may be #n / n / a WBS code (1.2) or an internal id.
		//   percent defaults to 50% when omitted (e.g. "assign 1 to 1.1").
		role := a.resolveRoleRef(fields[1])
		if role == "" {
			return "", fmt.Errorf("role %q not found (use R<n> or a role name)", fields[1])
		}
		wpRef := afterKeyword(fields, "to")
		wp := a.resolveTaskRef(wpRef)
		if wp == "" {
			return "", fmt.Errorf("action %q not found (to assign)", wpRef)
		}
		// percent, if given, is the token AFTER the wp ref (e.g.
		// "assign 1 to 1.1 100" or "... 100%"). A bare "assign 1 to 1.1" (no
		// trailing number) uses the default share. We must NOT mistake the role
		// ref (fields[1]) or the wp ref for the percent.
		pct := 0
		pctSet := false
		// find the index of the "to" keyword; the wp ref is the next token and
		// the percent (if any) is the token after that.
		toIdx := -1
		for i, f := range fields {
			if strings.EqualFold(f, "to") {
				toIdx = i
				break
			}
		}
		pctIdx := -1
		if toIdx >= 0 {
			pctIdx = toIdx + 2 // skip "to" and the wp ref
		}
		if pctIdx >= 0 && pctIdx < len(fields) {
			t := strings.TrimSuffix(fields[pctIdx], "%")
			var v int
			if n, err := fmt.Sscanf(t, "%d", &v); err == nil && n == 1 && t == fmt.Sprintf("%d", v) {
				pct = v
				pctSet = true
			}
		}
		// also accept an explicit NN% appearing anywhere
		if !pctSet {
			for _, f := range fields {
				if strings.HasSuffix(f, "%") {
					t := strings.TrimSuffix(f, "%")
					var v int
					if n, err := fmt.Sscanf(t, "%d", &v); err == nil && n == 1 && t == fmt.Sprintf("%d", v) {
						pct = v
						pctSet = true
					}
				}
			}
		}
		if !pctSet {
			pct = 50 // default share when no explicit percent is given
		}
		if _, err := a.AssignRole(role, wp, pct); err != nil {
			return "", err
		}
		return fmt.Sprintf("assigned %s to %s at %d%%", role, wp, pct), nil

	default:
		return "", fmt.Errorf("unknown command %q (type: help)", verb)
	}
}

// ChatHelp is exposed as a binding so the UI can show it in a panel.
func (a *App) ChatHelp() string {
	return fmt.Sprintf(`pt-plan commands   (current mode: %s)

Central object & modes
  mode action|role|section    switch the central object
  use <id|name>               select the current object
  link #<n> #<m> [FS|SS|FF|SF] link action number n to m
                              (also accepts WBS codes; details link in drill)

Adding - always say WHAT to add (there is no short "add <name>")
  add action <name> [from <YYYY-MM-DD>] [to <YYYY-MM-DD>] [est <value>]
                              e.g. add action "Scheduled Callback" from 2026-11-06 to 2026-11-10
                              without dates: today + 7 days; only from: + 7 days
  add action <name> after <id>
                              start after another action and FS-link them
                              <id> may be #n, n, or a WBS code like 1.2
  add section <name>          new section at the end (WP-n+1); new actions go into it
  add role <name>
  add milestone <name> [on <YYYY-MM-DD>] [in <section>]
  add detail <name> to <action> [from <d>] [to <d>] [est <value>]
                              in an open detail plane: add detail <name> [from ...]
  Reserved words: from, to, est, after, on, in end the name. Put a name that
  contains one in quotes: add action "Go to market" from 2027-01-04

Actions (work packages) — ids may be #n, n, a WBS code (1.2), or a
detail sub-action (1.2.1); detail sub-actions are actions too.
  move <n>d|w                 shift current action (move 3d, move -1w)
  move <id> after <id>        start right after another task
  move <id> before <id>       end right before another task
  move <id> to <YYYY-MM-DD>   move a specific task to a date
  move <id> <n>d|w left|right|future|past   shift a task in a direction
  lengthen <n>d|w             extend current action's end
  shorten <n>d|w              reduce current action's end
  resize <id> to <YYYY-MM-DD> set end date explicitly
  est <id> <value>            set effort (action OR detail sub-action)
  link <a> <b> [FS|SS|FF|SF]  link two tasks (actions or detail sub-actions)

Roles
  assign <role> to <wpID> <percent>%%

help                          show this`, a.mode)
}

// Mode / SetMode bindings for the UI mode switch.
func (a *App) Mode() string { return a.mode }
func (a *App) SetMode(m string) (string, error) {
	switch m {
	case "action", "role", "section", "milestone":
		a.mode = m
		a.current = ""
		return m, nil
	}
	return "", fmt.Errorf("mode must be action|role|section|milestone")
}

// SetDrill tells the command bar which action's detail plane is open ("" when
// none). While a drill plane is open, `add ...` in the chat adds a DETAIL of
// that action instead of a new top-level action. The frontend calls this when
// it enters or leaves a drill plane.
func (a *App) SetDrill(id string) {
	if id != "" && a.project != nil && a.project.WorkPackage(id) == nil {
		id = "" // unknown id: fall back to top-level add
	}
	a.drillID = id
}

// Drill returns the action whose detail plane the command bar targets ("" = none).
func (a *App) Drill() string { return a.drillID }

// --- helpers implementing the mode-aware behavior ---

func (a *App) addAfter(name, afterRef string) (string, error) {
	afterID := a.resolveActionRef(afterRef)
	pred := a.project.WorkPackage(afterID)
	if pred == nil {
		return "", fmt.Errorf("action %q not found (to add after)", afterRef)
	}
	dur := pred.End.Sub(pred.Start)
	if dur <= 0 {
		dur = 7 * 24 * time.Hour
	}
	// Minimum buffer: the successor must start at least ONE display unit after
	// the predecessor ends, so the FS link is not immediately a violation
	// (DepViolated requires start >= pred.End + 1 unit).
	start := pred.End.Add(a.oneUnit())
	end := start.Add(dur)
	if _, err := a.AddWorkPackage(name, start.Format(plan.DateLayout), end.Format(plan.DateLayout)); err != nil {
		return "", err
	}
	id := a.project.Packages[len(a.project.Packages)-1].ID
	a.current = id
	a.selKind, a.selID = "action", id
	if _, err := a.AddDependency(afterID, id, "FS", "r", "l"); err != nil {
		return "", fmt.Errorf("created %s but link failed: %w", id, err)
	}
	return fmt.Sprintf("added %q after %s (FS link, +1 unit buffer) as %s", name, afterID, id), nil
}

// addKinds are the objects "add" can create. The kind is always required:
// a bare "add <name>" was too easily confused (it depended on the mode).
const addUsage = `say what to add: add action <name> [from <date>] [to <date>] [est <value>] [after <id>], ` +
	`add section <name>, add role <name>, add milestone <name> [on <date>] [in <section>], ` +
	`add detail <name> [to <action>] [from <date>] [to <date>] [est <value>]`

// addReserved are the words that end the name in "add action/detail ...". A
// name that contains one of them must be quoted ("add action \"Go to market\"").
var addReserved = map[string]bool{"from": true, "to": true, "est": true, "after": true, ":after": true, "on": true, "in": true}

// splitName returns the name (the tokens up to the first reserved word, joined
// by spaces and stripped of quotes) and the remaining tokens.
func splitName(tokens []string, reserved map[string]bool) (string, []string) {
	i := 0
	for i < len(tokens) && !reserved[strings.ToLower(tokens[i])] {
		i++
	}
	name := strings.Trim(strings.Join(tokens[:i], " "), " \"\u201c\u201d\u201e\u201f\u00ab\u00bb")
	return name, tokens[i:]
}

// opts reads "<keyword> <value>" pairs. A keyword that appears twice keeps
// every value in order (the detail form uses "to" for the action and the end).
func opts(tokens []string) map[string][]string {
	out := map[string][]string{}
	for i := 0; i < len(tokens); i++ {
		k := strings.ToLower(tokens[i])
		if addReserved[k] && i+1 < len(tokens) {
			if k == ":after" {
				k = "after"
			}
			out[k] = append(out[k], tokens[i+1])
			i++
		}
	}
	return out
}

func optFirst(m map[string][]string, k string) string {
	if v := m[k]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// chatAdd handles "add <kind> <name> ...". The kind (action, section, role,
// milestone, detail; "wp" = action) is required. For actions and details the
// name ends at the first reserved word (from, to, est, after, on, in); quote a
// name that contains one.
func (a *App) chatAdd(fields []string) (string, error) {
	if len(fields) < 2 {
		return "", fmt.Errorf("%s", addUsage)
	}
	kind := strings.ToLower(fields[1])
	if kind == "wp" {
		kind = "action"
	}
	switch kind {
	case "action", "section", "role", "milestone", "detail":
	default:
		return "", fmt.Errorf("%q is not something to add - %s", fields[1], addUsage)
	}
	// While a detail plane is open, "add action ..." and "add detail ..." (without
	// a target action) add a DETAIL of the drilled action.
	if (kind == "action" || kind == "detail") && a.drillID != "" && a.project.WorkPackage(a.drillID) != nil {
		if kind == "action" || !detailHasTarget(fields[2:]) {
			return a.addDetailInDrill(fields[2:])
		}
	}
	switch kind {
	case "section", "role":
		name := strings.Trim(strings.Join(fields[2:], " "), " \"")
		if name == "" {
			return "", fmt.Errorf("usage: add %s <name>", kind)
		}
		if kind == "role" {
			a.AddRole(name, "")
			a.current = name
			a.selKind, a.selID = "", "" // roles are not canvas selections
			return fmt.Sprintf("added role %q", name), nil
		}
		if _, err := a.AddSection(name); err != nil {
			return "", err
		}
		a.current = name
		a.currentSection = name // new actions go into the new section
		a.selKind, a.selID = "section", name
		return fmt.Sprintf("added section %q (new actions go into it)", name), nil
	case "milestone":
		name, rest := splitName(fields[2:], addReserved)
		if name == "" {
			return "", fmt.Errorf("usage: add milestone <name> [on <date>] [in <section>]")
		}
		o := opts(rest)
		date := optFirst(o, "on")
		if date == "" {
			date = time.Now().Format(plan.DateLayout)
			if s, ok := a.project.Start(); ok {
				date = s.Format(plan.DateLayout)
			}
		}
		if _, err := a.AddMilestone(name, date, optFirst(o, "in")); err != nil {
			return "", err
		}
		return fmt.Sprintf("added milestone %q on %s", name, date), nil
	case "detail":
		name, rest := splitName(fields[2:], addReserved)
		o := opts(rest)
		to := o["to"]
		if name == "" || len(to) == 0 {
			return "", fmt.Errorf("usage: add detail <name> to <action> [from <date>] [to <date>] [est <value>]")
		}
		wp := a.resolveActionRef(to[0])
		if wp == "" {
			wp = to[0]
		}
		parent := a.project.WorkPackage(wp)
		if parent == nil {
			return "", fmt.Errorf("action %q not found (add detail ... to <action>)", to[0])
		}
		from, end := optFirst(o, "from"), ""
		if len(to) > 1 {
			end = to[1]
		}
		if from == "" {
			from = parent.Start.Format(plan.DateLayout)
		}
		if end == "" {
			end = parent.End.Format(plan.DateLayout)
		}
		est := optFirst(o, "est")
		if est == "" {
			est = "0"
		}
		if _, err := a.AddDetail(wp, name, from, end, est, "", ""); err != nil {
			return "", err
		}
		return fmt.Sprintf("added detail %q to %s (%s..%s)", name, parent.Name, from, end), nil
	}
	// action
	name, rest := splitName(fields[2:], addReserved)
	if name == "" {
		return "", fmt.Errorf("usage: add action <name> [from <date>] [to <date>] [est <value>] [after <id>]")
	}
	o := opts(rest)
	if after := optFirst(o, "after"); after != "" {
		return a.addAfter(name, after)
	}
	from, to := optFirst(o, "from"), optFirst(o, "to")
	if from == "" && to == "" {
		if est := optFirst(o, "est"); est == "" {
			return a.addBareAction(name)
		}
	}
	start := time.Now().Truncate(24 * time.Hour)
	if from != "" {
		t, err := plan.ParseDateStr(from)
		if err != nil {
			return "", fmt.Errorf("start date %q: %w", from, err)
		}
		start = t
	}
	end := start.AddDate(0, 0, 7)
	if to != "" {
		t, err := plan.ParseDateStr(to)
		if err != nil {
			return "", fmt.Errorf("end date %q: %w", to, err)
		}
		end = t
	}
	if _, err := a.AddWorkPackage(name, start.Format(plan.DateLayout), end.Format(plan.DateLayout)); err != nil {
		return "", err
	}
	id := a.project.Packages[len(a.project.Packages)-1].ID
	a.current = id
	a.selKind, a.selID = "action", id
	if est := optFirst(o, "est"); est != "" {
		if _, err := a.SetWorkPackageEstimation(id, est); err != nil {
			return "", fmt.Errorf("created %s but estimation failed: %w", id, err)
		}
	}
	return fmt.Sprintf("added action %s %q (%s..%s)", id, name, start.Format(plan.DateLayout), end.Format(plan.DateLayout)), nil
}

// detailHasTarget reports whether "add detail <name> to <action> ..." names its
// action: the first reserved word after the name is "to" and it is followed by
// something that is not a date (a date there means the end of the span).
func detailHasTarget(tokens []string) bool {
	_, rest := splitName(tokens, addReserved)
	if len(rest) < 2 || !strings.EqualFold(rest[0], "to") {
		return false
	}
	_, err := plan.ParseDateStr(rest[1])
	return err != nil
}

// addDetailInDrill adds a DETAIL to the drilled action ("add action ..." or
// "add detail ..." while its detail plane is open). Dates default to the
// action's span, the estimation to 0.
func (a *App) addDetailInDrill(tokens []string) (string, error) {
	parent := a.project.WorkPackage(a.drillID)
	if parent == nil {
		return "", fmt.Errorf("the drilled action no longer exists")
	}
	name, rest := splitName(tokens, addReserved)
	if name == "" {
		return "", fmt.Errorf("usage (detail plane): add detail <name> [from <date>] [to <date>] [est <value>]")
	}
	o := opts(rest)
	from, to, est := optFirst(o, "from"), optFirst(o, "to"), optFirst(o, "est")
	if from == "" {
		from = parent.Start.Format(plan.DateLayout)
	}
	if to == "" {
		to = parent.End.Format(plan.DateLayout)
	}
	if est == "" {
		est = "0"
	}
	if _, err := a.AddDetail(a.drillID, name, from, to, est, "", ""); err != nil {
		return "", err
	}
	return fmt.Sprintf("added detail %q to %s (%s..%s)", name, parent.Name, from, to), nil
}

// addBareAction adds an action with the default span (today, +7 days) and
// selects it, regardless of the current mode. Used by a bare add in action mode
// and by "add action <name>" without dates in any mode.
func (a *App) addBareAction(name string) (string, error) {
	start := time.Now().Truncate(24 * time.Hour)
	end := start.AddDate(0, 0, 7)
	if _, err := a.AddWorkPackage(name, start.Format(plan.DateLayout), end.Format(plan.DateLayout)); err != nil {
		return "", err
	}
	id := a.project.Packages[len(a.project.Packages)-1].ID
	a.current = id
	a.selKind, a.selID = "action", id
	return fmt.Sprintf("added action #%d %q", len(a.project.Packages), name), nil
}

// OR a detail sub-action. The mover keeps the duration and applies new dates.
func (a *App) taskSpan2(ref string) (start, end time.Time, move func(s, e time.Time) error, label string, err error) {
	id := a.resolveTaskRef(ref)
	if id == "" {
		return start, end, nil, "", fmt.Errorf("task %q not found", ref)
	}
	if d, wp := a.project.DetailByID(id); d != nil {
		mv := func(s, e time.Time) error {
			_, err := a.MoveDetail(wp.ID, id, s.Format(plan.DateLayout), e.Format(plan.DateLayout))
			return err
		}
		return d.Start, d.End, mv, id, nil
	}
	wp := a.project.WorkPackage(id)
	if wp == nil {
		return start, end, nil, "", fmt.Errorf("task %q not found", ref)
	}
	mv := func(s, e time.Time) error {
		_, err := a.MoveWorkPackage(id, s.Format(plan.DateLayout), e.Format(plan.DateLayout))
		return err
	}
	return wp.Start, wp.End, mv, id, nil
}

// moveTaskTo moves a task (action or detail) so it STARTS on the given date,
// preserving its duration.
func (a *App) moveTaskTo(ref, dateStr string) (string, error) {
	s, e, move, label, err := a.taskSpan2(ref)
	if err != nil {
		return "", err
	}
	ns, err := time.Parse(plan.DateLayout, dateStr)
	if err != nil {
		return "", fmt.Errorf("bad date %q (use YYYY-MM-DD)", dateStr)
	}
	dur := e.Sub(s)
	if err := move(ns, ns.Add(dur)); err != nil {
		return "", err
	}
	return fmt.Sprintf("moved %s to %s", label, dateStr), nil
}

// moveRelativeTo places a task immediately AFTER (start = other.end + 1 unit)
// or BEFORE (end = other.start - 1 unit) another task, preserving duration.
func (a *App) moveRelativeTo(ref, otherRef, rel string) (string, error) {
	s, e, move, label, err := a.taskSpan2(ref)
	if err != nil {
		return "", err
	}
	os, oe, _, _, err := a.taskSpan2(otherRef)
	if err != nil {
		return "", err
	}
	dur := e.Sub(s)
	unit := a.oneUnit()
	if rel == "after" {
		ns := oe.Add(unit)
		if err := move(ns, ns.Add(dur)); err != nil {
			return "", err
		}
		return fmt.Sprintf("moved %s after %s", label, otherRef), nil
	}
	// before: end one unit before the other's start
	ne := os.Add(-unit)
	ns := ne.Add(-dur)
	if err := move(ns, ne); err != nil {
		return "", err
	}
	return fmt.Sprintf("moved %s before %s", label, otherRef), nil
}

// moveTaskDelta shifts a task by a +/- delta in a direction. left/past shift
// earlier; right/future shift later.
func (a *App) moveTaskDelta(ref string, fields []string, dir string) (string, error) {
	s, e, move, label, err := a.taskSpan2(ref)
	if err != nil {
		return "", err
	}
	deltaTok := lastDelta(fields)
	if deltaTok == "" {
		return "", fmt.Errorf("missing delta like 10d or 2w")
	}
	days, err := deltaDays(deltaTok)
	if err != nil {
		return "", err
	}
	days = abs(days)
	if dir == "left" || dir == "past" {
		days = -days
	}
	ns := s.AddDate(0, 0, days)
	ne := e.AddDate(0, 0, days)
	if err := move(ns, ne); err != nil {
		return "", err
	}
	return fmt.Sprintf("moved %s %s by %s", label, dir, deltaTok), nil
}

// oneUnit returns one display unit as a duration: 1 day in day mode, else 1h.
func (a *App) oneUnit() time.Duration {
	if a.project.Unit == plan.UnitDays {
		return 24 * time.Hour
	}
	return time.Hour
}

// lastDirection returns a trailing direction word (left|right|future|past) or "".
func lastDirection(fields []string) string {
	for i := len(fields) - 1; i >= 0; i-- {
		switch strings.ToLower(fields[i]) {
		case "left", "right", "future", "past":
			return strings.ToLower(fields[i])
		}
	}
	return ""
}

// lastDelta returns the last token that looks like a delta (Nd / Nw), or "".
func lastDelta(fields []string) string {
	for i := len(fields) - 1; i >= 0; i-- {
		if isDelta(fields[i]) {
			return fields[i]
		}
	}
	return ""
}

func (a *App) shiftCurrent(delta string) (string, error) {
	if a.current == "" {
		return "", fmt.Errorf("no current action; 'use <id>' first")
	}
	wp := a.project.WorkPackage(a.current)
	if wp == nil {
		return "", fmt.Errorf("current action %q not found", a.current)
	}
	days, err := deltaDays(delta)
	if err != nil {
		return "", err
	}
	ns := wp.Start.AddDate(0, 0, days)
	ne := wp.End.AddDate(0, 0, days)
	if _, err := a.MoveWorkPackage(a.current, ns.Format(plan.DateLayout), ne.Format(plan.DateLayout)); err != nil {
		return "", err
	}
	return fmt.Sprintf("moved %s by %s", a.current, delta), nil
}

func (a *App) resizeCurrent(delta string, sign int) (string, error) {
	if a.current == "" {
		return "", fmt.Errorf("no current action; 'use <id>' first")
	}
	wp := a.project.WorkPackage(a.current)
	if wp == nil {
		return "", fmt.Errorf("current action %q not found", a.current)
	}
	days, err := deltaDays(delta)
	if err != nil {
		return "", err
	}
	days = abs(days) * sign
	ne := wp.End.AddDate(0, 0, days)
	if !ne.After(wp.Start) {
		return "", fmt.Errorf("resulting end would not be after start")
	}
	if _, err := a.MoveWorkPackage(a.current, wp.Start.Format(plan.DateLayout), ne.Format(plan.DateLayout)); err != nil {
		return "", err
	}
	verb := "lengthened"
	if sign < 0 {
		verb = "shortened"
	}
	return fmt.Sprintf("%s %s to end %s", verb, a.current, ne.Format(plan.DateLayout)), nil
}

// --- small parsing helpers ---

// isDelta reports whether s looks like a +/-N d|w token.
func isDelta(s string) bool {
	_, err := deltaDays(s)
	return err == nil
}

// deltaDays parses "3d", "-1w", "2w" into a signed number of days.
func deltaDays(s string) (int, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty delta")
	}
	unit := s[len(s)-1]
	num := s[:len(s)-1]
	mult := 1
	switch unit {
	case 'd':
	case 'w':
		mult = 7
	default:
		return 0, fmt.Errorf("delta must end in d or w (e.g. 3d, -1w)")
	}
	var n int
	if _, err := fmt.Sscanf(num, "%d", &n); err != nil {
		return 0, fmt.Errorf("bad delta %q", s)
	}
	return n * mult, nil
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func pick(fields []string, idx int, fallback string) string {
	if idx < len(fields) && !strings.EqualFold(fields[idx], "to") {
		return fields[idx]
	}
	return fallback
}

func hasKeyword(fields []string, kw string) bool {
	for _, f := range fields {
		if strings.EqualFold(f, kw) {
			return true
		}
	}
	return false
}

// tokenize splits on spaces but keeps "quoted phrases" as single tokens.
func tokenize(s string) []string {
	var out []string
	var cur strings.Builder
	inQuote := false
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		switch {
		case isQuote(r):
			inQuote = !inQuote
			if !inQuote {
				flush()
			}
		case r == ' ' && !inQuote:
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}

// isQuote reports whether r opens or closes a quoted token. Straight, smart,
// German and guillemet quotes are all accepted so a name pasted from a document
// ("2 Email…", „2 Email…", «2 Email…") is kept whole.
func isQuote(r rune) bool {
	switch r {
	case '"', '\u201c', '\u201d', '\u201e', '\u201f', '\u00ab', '\u00bb':
		return true
	}
	return false
}

func afterKeyword(fields []string, kw string) string {
	for i, f := range fields {
		if strings.EqualFold(f, kw) && i+1 < len(fields) {
			return fields[i+1]
		}
	}
	return ""
}

func nthAfterKeyword(fields []string, kw string, n int) string {
	count := 0
	for i, f := range fields {
		if strings.EqualFold(f, kw) {
			count++
			if count == n && i+1 < len(fields) {
				return fields[i+1]
			}
		}
	}
	return ""
}
