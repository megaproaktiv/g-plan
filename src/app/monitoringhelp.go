package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// The monitoring help: what the External ID means and a step-by-step guide with
// a worked example, in both GUI languages. Text is data (like the glossary) and
// the example is COMPUTED with the real matching and rating code, so the numbers
// in the help cannot disagree with what the program does.
//
// Inline markup in the texts: **bold** for names of buttons and fields, and
// `code` for texts the user types. documentation/monitoring-guide*.md is
// generated from this data (a test keeps it in sync).

// HelpBlockDTO is one block of a help section.
type HelpBlockDTO struct {
	Kind  string     `json:"kind"`  // p | ul | ol | table | note
	Text  string     `json:"text"`  // p, note
	Items []string   `json:"items"` // ul, ol
	Head  []string   `json:"head"`  // table
	Rows  [][]string `json:"rows"`  // table
}

// HelpSectionDTO is a headed part of a help page.
type HelpSectionDTO struct {
	Key     string         `json:"key"`
	Heading string         `json:"heading"`
	Blocks  []HelpBlockDTO `json:"blocks"`
}

// HelpPageDTO is a whole help page.
type HelpPageDTO struct {
	Title    string           `json:"title"`
	Intro    string           `json:"intro"`
	Sections []HelpSectionDTO `json:"sections"`
}

// MonitoringHelp returns the monitoring help page in a GUI language.
func (a *App) MonitoringHelp(lang string) HelpPageDTO { return monitoringHelp(lang) }

func hp(text string) HelpBlockDTO      { return HelpBlockDTO{Kind: "p", Text: text} }
func hnote(text string) HelpBlockDTO   { return HelpBlockDTO{Kind: "note", Text: text} }
func hul(items ...string) HelpBlockDTO { return HelpBlockDTO{Kind: "ul", Items: items} }
func hol(items ...string) HelpBlockDTO { return HelpBlockDTO{Kind: "ol", Items: items} }
func htable(head []string, rows [][]string) HelpBlockDTO {
	return HelpBlockDTO{Kind: "table", Head: head, Rows: rows}
}

// --- the worked example ------------------------------------------------------

// The example project: "Skynet MVP", planned in days with 8 hours per day, and
// three actions that carry the External IDs WP1, WP2 and WP3.
const exampleProjectNumber = "555555"

type exampleBooking struct {
	Text    string
	Seconds int
}

// exampleBookings are the bookings of the example, as they could look in the
// external system. The texts are the same in both languages (they are what
// people type into the time tracking tool).
var exampleBookings = []exampleBooking{
	{"WP2 VPC design", 3 * 3600},
	{"wp2: security review", 2 * 3600},
	{"WP2 and WP3 pairing session", 4 * 3600},
	{"WP3 account structure", 5 * 3600},
	{"Daily standup", 1 * 3600},
}

// exampleMatchRows are the texts used to explain how an External ID is found.
// The ID asked about is WP1; Match is what the program really answers.
var exampleMatchTexts = []string{
	"WP1 kickoff meeting",
	"wp1: design",
	"(WP1)",
	"design / WP1 / review",
	"WP12 review",
	"xWP1 review",
	"WP-1 kickoff",
	"WP 1 kickoff",
	"Daily standup",
}

// exampleProject builds the project of the example.
func exampleProject() *plan.Project {
	pr := plan.NewProject("Skynet MVP", plan.UnitDays)
	pr.SetRange(exampleDate("2026-11-17"), exampleDate("2027-07-31"))
	add := func(id, name, ext string, days int64) {
		pr.Packages = append(pr.Packages, plan.WorkPackage{
			ID: id, Name: name, ExternalID: ext, Mode: plan.ModeFixed, Contour: plan.ContourFlat,
			Start: exampleDate("2026-11-17"), End: exampleDate("2027-07-31"), Est: pr.BaseFromWholeDisplay(days), Cluster: "Plan",
		})
	}
	add("pm", "Project Management", "WP1", 40)
	add("aws", "AWS Landing Zone", "WP2", 10)
	add("acc", "ACC Core configuration", "WP3", 12)
	pr.Monitor = plan.Monitor{Enabled: true, System: plan.SystemAerion, ProjectID: exampleProjectNumber, DefaultAction: "pm"}
	return pr
}

func exampleDate(s string) (t time.Time) { t, _ = plan.ParseDateStr(s); return }

// exampleOutcome is what the program makes of the example.
type exampleOutcome struct {
	// per booking: the hours counted for each action, e.g. "WP2 2 h, WP3 2 h"
	Split    []string
	Projects map[string]exampleAction // by action id
}

type exampleAction struct {
	Name, ExternalID string
	Planned          string // "10 days (80 h)"
	Actual           string // "0,88 days (7 h)"
	Consumed         string // "8.7%"
}

func computeExample(lang string) exampleOutcome {
	pr := exampleProject()
	var bookings []plan.Booking
	for i, b := range exampleBookings {
		bookings = append(bookings, plan.Booking{ID: int64(i + 1), Seconds: b.Seconds, Text: b.Text})
	}
	sum := pr.MonitorSummary(bookings, pr.Monitor.DefaultAction)
	pr.ApplySyncedEffort(sum)

	var ids []string
	var owner []string
	for _, w := range pr.Packages {
		ids = append(ids, w.ExternalID)
		owner = append(owner, w.ID)
	}
	out := exampleOutcome{Projects: map[string]exampleAction{}}
	for _, b := range bookings {
		refs := plan.MatchExternalIDs(b.Text, ids)
		shares := plan.SpreadSeconds(b.Seconds, len(refs))
		if len(refs) == 0 {
			out.Split = append(out.Split, fmt.Sprintf("%s: %s", pr.WorkPackage(pr.Monitor.DefaultAction).Name, hoursText(int64(b.Seconds))))
			continue
		}
		var parts []string
		for k, r := range refs {
			parts = append(parts, fmt.Sprintf("%s %s", ids[r], hoursText(shares[k])))
		}
		out.Split = append(out.Split, strings.Join(parts, ", "))
	}
	dec := lang == "de"
	for _, w := range pr.Packages {
		perm := w.ProgressPermille()
		pct := fmt.Sprintf("%.1f%%", float64(perm)/10)
		if dec {
			pct = strings.Replace(pct, ".", ",", 1)
		}
		out.Projects[w.ID] = exampleAction{
			Name: w.Name, ExternalID: w.ExternalID,
			Planned:  fmt.Sprintf("%s %s (%s h)", pr.FormatDays(w.Est), daysWord(pr.FormatDays(w.Est), lang), pr.FormatHours(w.Est)),
			Actual:   fmt.Sprintf("%s %s (%s h)", pr.FormatDays(w.RealEst), daysWord(pr.FormatDays(w.RealEst), lang), pr.FormatHours(w.RealEst)),
			Consumed: pct,
		}
	}
	return out
}

func hoursText(sec int64) string {
	pr := plan.NewProject("x", plan.UnitDays)
	return pr.FormatHours(plan.SecondsToBase(sec)) + " h"
}

func daysWord(n, lang string) string {
	if lang == "de" {
		if n == "1" {
			return "Tag"
		}
		return "Tage"
	}
	if n == "1" {
		return "day"
	}
	return "days"
}

// matchAnswers returns, for each of exampleMatchTexts, whether the ID WP1 is
// found - answered by the real matcher.
func matchAnswers() []bool {
	var out []bool
	for _, txt := range exampleMatchTexts {
		out = append(out, len(plan.MatchExternalIDs(txt, []string{"WP1"})) == 1)
	}
	return out
}

func monitoringHelp(lang string) HelpPageDTO {
	if lang == "de" {
		return monitoringHelpDE()
	}
	return monitoringHelpEN()
}

func monitoringHelpEN() HelpPageDTO {
	ex := computeExample("en")
	yes, no := "yes", "no"
	why := []string{
		"the ID stands as a word of its own", "capital letters do not matter", "brackets and punctuation are fine",
		"it may sit in the middle of the text", "WP12 is a different, longer ID", "a letter directly before it: not a separate word",
		"a hyphen inside the ID does not matter: WP-1 is WP1", "a space is not a hyphen: WP 1 is two words", "no ID in the text",
	}
	var matchRows [][]string
	for i, txt := range exampleMatchTexts {
		answer := no
		if matchAnswers()[i] {
			answer = yes
		}
		matchRows = append(matchRows, []string{"`" + txt + "`", answer, why[i]})
	}
	var bookingRows [][]string
	for i, b := range exampleBookings {
		bookingRows = append(bookingRows, []string{"`" + b.Text + "`", hoursText(int64(b.Seconds)), ex.Split[i]})
	}
	var resultRows [][]string
	for _, id := range []string{"aws", "acc", "pm"} {
		r := ex.Projects[id]
		resultRows = append(resultRows, []string{fmt.Sprintf("%s (%s)", r.Name, r.ExternalID), r.Planned, r.Actual, r.Consumed})
	}

	return HelpPageDTO{
		Title: "Monitoring",
		Intro: "Monitoring compares your plan with the hours that were really booked in an external time tracking system (Aerion today). It only reads from that system and never changes anything in it.",
		Sections: []HelpSectionDTO{
			{Key: "idea", Heading: "How it fits together", Blocks: []HelpBlockDTO{
				hp("Three things connect your plan with the bookings:"),
				hul(
					"**System** and **project ID** (set once per project on the **Monitor** page): which time tracking system, and the number of the project in it.",
					"**External ID** (set per action in the action editor): a short label of up to 16 characters, for example `WP2`.",
					"The **booking text**: whoever books time writes the External ID into the text of the booking in the external system.",
				),
				hp("A synchronization reads all bookings of the external project from the project start to the project end and looks for the External IDs of your actions in the booking texts. The booked time becomes the **real effort** (Work) of the action, and the plan shows how much of the estimation is consumed."),
			}},
			{Key: "extid", Heading: "How the External ID is interpreted", Blocks: []HelpBlockDTO{
				hp("The External ID is searched **in the text of every booking**, and nowhere else. A booking belongs to an action when its text contains the action's External ID:"),
				hul(
					"**As a word of its own.** The ID counts only when the character right before and right after it is not a letter or a digit. `WP1` is found in `WP1 kickoff`, in `(wp1)` and in `design / WP1 / review`, but not in `WP12` and not in `xWP1`.",
					"**Capital letters and hyphens do not matter.** `WP1`, `wp1`, `Wp1`, `WP-1` and `wp-1` are all the same ID, in the ID field and in the booking text. A space does not count as a hyphen: `WP 1` is not `WP1`.",
					"**Several IDs in one booking** (for example `WP2 and WP3 pairing session`): the booked time is **split evenly** between those actions. Seconds that do not divide evenly go to the first ones mentioned, so no time is lost.",
					"**The same ID twice** in one text counts once. An ID that only occurs inside a longer ID (the `1` in `Phase 1`) does not count as its own.",
					"**No ID in the text:** the booking is *unassigned*. It is counted for the **default action** if you chose one, otherwise it only appears on the line *Bookings without a matching External ID*.",
					"**Which time:** the tracked duration of each booking. Only work bookings are read, not breaks or working-time entries.",
					"**Rules for the ID itself:** at most 16 characters, unique in the project (capital letters and hyphens ignored: `WP1` and `WP-1` count as the same), and an action with the contour **None** cannot carry one. It can be changed at any time, also while monitoring is on.",
				),
				hp("Which of these booking texts are found for an action with the External ID `WP1`?"),
				htable([]string{"Booking text", "Found for WP1?", "Why"}, matchRows),
				hnote("Tip: choose IDs that do not occur in everyday words, such as `WP1`, `WP2`, `WP3`, and ask the team to write them at the beginning of the booking text."),
			}},
			{Key: "steps", Heading: "Step by step, with an example", Blocks: []HelpBlockDTO{
				hp(fmt.Sprintf("The example project is *Skynet MVP*, planned in days with 8 hours per day. Its project number in Aerion is `%s`. It has three actions: **Project Management** (External ID `WP1`, 40 days planned), **AWS Landing Zone** (`WP2`, 10 days = 80 h) and **ACC Core configuration** (`WP3`, 12 days = 96 h).", exampleProjectNumber)),
				hol(
					"**Open the project** and check its start and end date. Bookings are fetched from the project start to the project end.",
					"Click **Monitor**, then **Configure System…**. Enter the base URL, user name, password, client ID and client secret of Aerion and save. They are stored only for you, in `~/.pt-pm/config.yaml`, and are never shown again.",
					fmt.Sprintf("Still on the **Monitor** page, choose the system **Aerion** and enter the project number (`%s`). Click **Test**: *Connected: project …* shows that the credentials and the number are right.", exampleProjectNumber),
					"**Give every action an External ID.** Open the action (double-click), type `WP1`, `WP2` or `WP3` into **External ID**, and press Enter or click elsewhere. Tell the team to write the ID into the text of their bookings.",
					"**Optional: choose the Default action** for bookings without an ID, so that no booked time is lost. In the example it is *Project Management*.",
					"**Optional:** tick *also one month before the project start* or *also one month after the project end* if bookings begin early or run over.",
					"Switch **Monitoring on**. This needs a system and a project number.",
					"Click **Synchronize**. A progress bar and the number of bookings fetched are shown.",
					"**Read the result.** The table shows, per section, the estimation of the tracked actions against the actual effort, the **Remaining** effort (estimation minus actual, negative when over budget) and **Remaining %** (its share of the estimation), the project total, and the line for unassigned bookings. With *Show Progress* switched on, the bars show the consumed percentage, and the action view shows the planned and actual effort. **Export** (next to Synchronize) writes the table as two markdown files: `<name>-compact.md` with the work packages (sections) and the project total, and `<name>-detailed.md` which also lists the tracked actions of each work package.",
					"**Correct and repeat.** If bookings are unassigned or in the wrong place, fix the booking text in the external system or the External ID, then synchronize again. Each synchronization overwrites the real effort of the tracked actions.",
				),
				hp("These are the bookings in the external system, and what the program makes of them:"),
				htable([]string{"Booking text", "Time", "Counted for"}, bookingRows),
				hp("After the synchronization the actions show:"),
				htable([]string{"Action", "Planned", "Actual", "Consumed"}, resultRows),
				hnote("*Consumed* is the actual effort divided by the planned effort. Project Management also received the 1 h of the booking without an ID, because it is the default action."),
			}},
			{Key: "problems", Heading: "If it does not add up", Blocks: []HelpBlockDTO{
				hul(
					"**Nothing is assigned to an action:** the External ID is not in the booking text, or it is not a separate word. `WP 1` (with a space) is not `WP1`.",
					"**Everything is unassigned:** no action has an External ID yet, or the project number is wrong. Use **Test** on the Monitor page.",
					"**Too many unassigned bookings:** ask the team to add IDs, or choose a default action.",
					"**The Real field of an action is greyed out:** tracked actions (with an External ID, or the default action) get their real effort from the synchronization. To enter it by hand, remove the External ID or switch monitoring off.",
					"**An action does not take part at all:** the contour **None** means no estimation, no real effort and no External ID. The **Phases** lane is a view, not a work package: it is ignored by monitoring and its actions need no External ID.",
					"**\"Aerion is not configured\":** open **Monitor** and **Configure System…**.",
					"**The result disappeared:** changing an External ID, the default action or the period clears the last result. Synchronize again.",
				),
			}},
		},
	}
}

func monitoringHelpDE() HelpPageDTO {
	ex := computeExample("de")
	yes, no := "ja", "nein"
	why := []string{
		"die ID steht als eigenes Wort", "Groß- und Kleinschreibung spielt keine Rolle", "Klammern und Satzzeichen sind in Ordnung",
		"sie darf mitten im Text stehen", "WP12 ist eine andere, längere ID", "direkt davor steht ein Buchstabe: kein eigenes Wort",
		"ein Bindestrich in der ID spielt keine Rolle: WP-1 ist WP1", "ein Leerzeichen ist kein Bindestrich: WP 1 sind zwei Wörter", "keine ID im Text",
	}
	var matchRows [][]string
	for i, txt := range exampleMatchTexts {
		answer := no
		if matchAnswers()[i] {
			answer = yes
		}
		matchRows = append(matchRows, []string{"`" + txt + "`", answer, why[i]})
	}
	var bookingRows [][]string
	for i, b := range exampleBookings {
		bookingRows = append(bookingRows, []string{"`" + b.Text + "`", hoursText(int64(b.Seconds)), ex.Split[i]})
	}
	var resultRows [][]string
	for _, id := range []string{"aws", "acc", "pm"} {
		r := ex.Projects[id]
		resultRows = append(resultRows, []string{fmt.Sprintf("%s (%s)", r.Name, r.ExternalID), r.Planned, r.Actual, r.Consumed})
	}

	return HelpPageDTO{
		Title: "Monitoring",
		Intro: "Monitoring vergleicht Ihren Plan mit den Stunden, die in einem externen Zeiterfassungssystem (derzeit Aerion) wirklich gebucht wurden. Es liest dort nur und ändert nichts.",
		Sections: []HelpSectionDTO{
			{Key: "idea", Heading: "Wie alles zusammenhängt", Blocks: []HelpBlockDTO{
				hp("Drei Dinge verbinden Ihren Plan mit den Buchungen:"),
				hul(
					"**System** und **Projekt-ID** (einmal je Projekt auf der Seite **Monitor**): welches Zeiterfassungssystem und welche Nummer das Projekt dort hat.",
					"**Externe ID** (je Aktion im Aktions-Editor): ein kurzes Kennzeichen mit bis zu 16 Zeichen, zum Beispiel `WP2`.",
					"Der **Buchungstext**: Wer Zeit bucht, schreibt die externe ID in den Text der Buchung im externen System.",
				),
				hp("Eine Synchronisation liest alle Buchungen des externen Projekts vom Projektstart bis zum Projektende und sucht in den Buchungstexten nach den externen IDs Ihrer Aktionen. Die gebuchte Zeit wird zum **Ist-Aufwand** (Arbeit) der Aktion, und der Plan zeigt, wie viel der Schätzung verbraucht ist."),
			}},
			{Key: "extid", Heading: "Wie die externe ID gedeutet wird", Blocks: []HelpBlockDTO{
				hp("Die externe ID wird **im Text jeder Buchung** gesucht, sonst nirgends. Eine Buchung gehört zu einer Aktion, wenn ihr Text die externe ID der Aktion enthält:"),
				hul(
					"**Als eigenes Wort.** Die ID zählt nur, wenn das Zeichen direkt davor und direkt danach kein Buchstabe und keine Ziffer ist. `WP1` wird in `WP1 kickoff`, in `(wp1)` und in `design / WP1 / review` gefunden, aber nicht in `WP12` und nicht in `xWP1`.",
					"**Groß- und Kleinschreibung und Bindestriche spielen keine Rolle.** `WP1`, `wp1`, `Wp1`, `WP-1` und `wp-1` sind dieselbe ID, im ID-Feld wie im Buchungstext. Ein Leerzeichen zählt nicht als Bindestrich: `WP 1` ist nicht `WP1`.",
					"**Mehrere IDs in einer Buchung** (zum Beispiel `WP2 and WP3 pairing session`): Die gebuchte Zeit wird **gleichmäßig auf diese Aktionen verteilt**. Sekunden, die sich nicht gleichmäßig teilen lassen, gehen an die zuerst genannten, so geht keine Zeit verloren.",
					"**Dieselbe ID zweimal** in einem Text zählt einmal. Eine ID, die nur innerhalb einer längeren ID vorkommt (die `1` in `Phase 1`), zählt nicht als eigene.",
					"**Keine ID im Text:** Die Buchung ist *nicht zugeordnet*. Sie wird der **Standard-Aktion** zugerechnet, falls Sie eine gewählt haben, sonst erscheint sie nur in der Zeile *Bookings without a matching External ID*.",
					"**Welche Zeit:** die erfasste Dauer jeder Buchung. Gelesen werden nur Arbeitsbuchungen, keine Pausen und keine Arbeitszeit-Einträge.",
					"**Regeln für die ID selbst:** höchstens 16 Zeichen, im Projekt eindeutig (Groß-/Kleinschreibung und Bindestriche ignoriert: `WP1` und `WP-1` gelten als gleich), und eine Aktion mit der Kontur **None** kann keine haben. Sie lässt sich jederzeit ändern, auch bei eingeschaltetem Monitoring.",
				),
				hp("Welche dieser Buchungstexte werden für eine Aktion mit der externen ID `WP1` gefunden?"),
				htable([]string{"Buchungstext", "Für WP1 gefunden?", "Warum"}, matchRows),
				hnote("Tipp: Wählen Sie IDs, die in normalen Wörtern nicht vorkommen, etwa `WP1`, `WP2`, `WP3`, und bitten Sie das Team, sie an den Anfang des Buchungstexts zu schreiben."),
			}},
			{Key: "steps", Heading: "Schritt für Schritt, mit Beispiel", Blocks: []HelpBlockDTO{
				hp(fmt.Sprintf("Das Beispielprojekt ist *Skynet MVP*, geplant in Tagen mit 8 Stunden pro Tag. Die Projektnummer in Aerion ist `%s`. Es hat drei Aktionen: **Project Management** (externe ID `WP1`, 40 Tage geplant), **AWS Landing Zone** (`WP2`, 10 Tage = 80 h) und **ACC Core configuration** (`WP3`, 12 Tage = 96 h).", exampleProjectNumber)),
				hol(
					"**Projekt öffnen** und Start- und Enddatum prüfen. Die Buchungen werden vom Projektstart bis zum Projektende geholt.",
					"**Monitor** anklicken, dann **Configure System…**. Basis-URL, Benutzername, Passwort, Client-ID und Client-Secret von Aerion eintragen und speichern. Sie werden nur für Sie in `~/.pt-pm/config.yaml` abgelegt und nie wieder angezeigt.",
					fmt.Sprintf("Weiter auf der Seite **Monitor**: das System **Aerion** wählen und die Projektnummer (`%s`) eintragen. Mit **Test** zeigt *Connected: project …*, dass Zugangsdaten und Nummer stimmen.", exampleProjectNumber),
					"**Jeder Aktion eine externe ID geben.** Die Aktion öffnen (Doppelklick), `WP1`, `WP2` oder `WP3` in **External ID** eintragen und Enter drücken oder woanders hinklicken. Das Team bitten, die ID in den Text seiner Buchungen zu schreiben.",
					"**Optional: die Standard-Aktion wählen** (Auswahlfeld *Default action*) für Buchungen ohne ID, damit keine gebuchte Zeit verloren geht. Im Beispiel ist es *Project Management*.",
					"**Optional:** *also one month before the project start* oder *also one month after the project end* ankreuzen, wenn Buchungen früher beginnen oder überlaufen.",
					"**Monitoring einschalten** (Kontrollkästchen *Monitoring on*). Dafür braucht es ein System und eine Projektnummer.",
					"**Synchronize** anklicken. Ein Fortschrittsbalken und die Zahl der geholten Buchungen werden angezeigt.",
					"**Ergebnis lesen.** Die Tabelle zeigt je Abschnitt die Schätzung der verfolgten Aktionen gegen den Ist-Aufwand, der **Rest** (Schätzung minus Ist, negativ bei Überschreitung) und **Rest %** (sein Anteil an der Schätzung), die Projektsumme und die Zeile für nicht zugeordnete Buchungen. Bei eingeschaltetem *Show Progress* zeigen die Balken den verbrauchten Anteil, und die Aktionsansicht zeigt geplanten und tatsächlichen Aufwand. **Export** (neben Synchronize) schreibt die Tabelle in zwei Markdown-Dateien: `<Name>-compact.md` mit den Arbeitspaketen (Abschnitten) und der Projektsumme, und `<Name>-detailed.md`, die zusätzlich die verfolgten Aktionen jedes Arbeitspakets auflistet.",
					"**Korrigieren und wiederholen.** Sind Buchungen nicht oder falsch zugeordnet, den Buchungstext im externen System oder die externe ID korrigieren und erneut synchronisieren. Jede Synchronisation überschreibt den Ist-Aufwand der verfolgten Aktionen.",
				),
				hp("Das sind die Buchungen im externen System und was das Programm daraus macht:"),
				htable([]string{"Buchungstext", "Zeit", "Gezählt für"}, bookingRows),
				hp("Nach der Synchronisation zeigen die Aktionen:"),
				htable([]string{"Aktion", "Geplant", "Ist", "Verbraucht"}, resultRows),
				hnote("*Verbraucht* ist der Ist-Aufwand geteilt durch den geplanten Aufwand. Project Management hat auch die 1 h der Buchung ohne ID erhalten, weil es die Standard-Aktion ist."),
			}},
			{Key: "problems", Heading: "Wenn es nicht aufgeht", Blocks: []HelpBlockDTO{
				hul(
					"**Einer Aktion wird nichts zugeordnet:** Die externe ID steht nicht im Buchungstext oder nicht als eigenes Wort. `WP 1` (mit Leerzeichen) ist nicht `WP1`.",
					"**Alles ist nicht zugeordnet:** Noch keine Aktion hat eine externe ID, oder die Projektnummer ist falsch. Auf der Seite Monitor **Test** benutzen.",
					"**Zu viele nicht zugeordnete Buchungen:** das Team bitten, IDs zu ergänzen, oder eine Standard-Aktion wählen.",
					"**Das Feld Real einer Aktion ist ausgegraut:** verfolgte Aktionen (mit externer ID oder die Standard-Aktion) bekommen ihren Ist-Aufwand aus der Synchronisation. Um ihn von Hand einzugeben, die externe ID entfernen oder das Monitoring ausschalten.",
					"**Eine Aktion nimmt gar nicht teil:** die Kontur **None** bedeutet keine Schätzung, keinen Ist-Aufwand und keine externe ID. Die Spur **Phases** ist eine Ansicht und kein Arbeitspaket: Das Monitoring ignoriert sie, und ihre Aktionen brauchen keine externe ID.",
					"**„Aerion is not configured“:** **Monitor** öffnen und **Configure System…** wählen.",
					"**Das Ergebnis ist verschwunden:** Eine geänderte externe ID, Standard-Aktion oder Zeitspanne löscht das letzte Ergebnis. Erneut synchronisieren.",
				),
			}},
		},
	}
}
