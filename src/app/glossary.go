package main

// The glossary: the project's vocabulary, defined once and shown in the help in
// both GUI languages. The English text is the reference;
// documentation/glossary.md mirrors it (a test keeps the term names in sync).

// Glossary groups.
const (
	GlossaryCore   = "core"   // the planning vocabulary
	GlossaryScreen = "screen" // terms that appear on screen and tie back to the core terms
)

// GlossaryTermDTO is one glossary entry.
type GlossaryTermDTO struct {
	Key        string `json:"key"`
	Group      string `json:"group"`
	Term       string `json:"term"`
	AlsoCalled string `json:"alsoCalled"` // synonyms, comma separated
	Definition string `json:"definition"`
	Example    string `json:"example"`
	Note       string `json:"note"`
}

// Glossary returns the glossary in a GUI language ("en" | "de"). Both languages
// have the same keys in the same order.
func (a *App) Glossary(lang string) []GlossaryTermDTO { return glossaryTerms(lang) }

func glossaryTerms(lang string) []GlossaryTermDTO {
	if lang == "de" {
		return glossaryDE()
	}
	return glossaryEN()
}

func glossaryEN() []GlossaryTermDTO {
	return []GlossaryTermDTO{
		{Key: "section", Group: GlossaryCore, Term: "Section", AlsoCalled: "Work Package",
			Definition: "A named lane on the canvas that groups actions (and milestones). It has a number - 1, 2, 3 ... - which is the first part of the numbers of its actions (1.1, 1.2 ...), and it can have its own color. The planning name for a section is Work Package."},
		{Key: "workpackage", Group: GlossaryCore, Term: "Work Package", AlsoCalled: "Section",
			Definition: "The planning name for a section: a group of related actions that belong together and are added up together, for example in the Monitor table. A work package contains actions."},
		{Key: "action", Group: GlossaryCore, Term: "Action",
			Definition: "A piece of work to be done, drawn as a bar with a start, an end, an estimation and optional roles. An action can be broken down into detail actions (open them with the drill icon or Details...)."},
		{Key: "role", Group: GlossaryCore, Term: "Role",
			Definition: "A named kind of worker or skill, for example Developer, that can be assigned to actions. The percentage of an assignment is the share of the working day the role spends on that action."},
		{Key: "milestone", Group: GlossaryCore, Term: "Milestone", AlsoCalled: "Gateway, Gate",
			Definition: "A certain point in time, without duration, that is used to measure the success of the project. A milestone is reached or missed on its date."},
		{Key: "estimation", Group: GlossaryCore, Term: "Estimation", AlsoCalled: "planned effort",
			Definition: "The planned amount of work for an action, in hours or days. It says how much work is planned, not how long the action takes (see Duration). The estimation of a work package is the sum of its actions."},
		{Key: "work", Group: GlossaryCore, Term: "Work", AlsoCalled: "real effort (field \"Real\")",
			Definition: "The time actually spent on an action, in hours or days. It is entered by hand or - with monitoring on - taken from the bookings in the external time tracking system."},
		{Key: "duration", Group: GlossaryCore, Term: "Duration",
			Definition: "The time between the start and the end of an action. It usually differs from the estimation: an action can last 10 days while the estimated work is 80 hours. The program also shows the working days of the duration (Monday to Friday, without weekends)."},
		{Key: "durationcomplete", Group: GlossaryCore, Term: "Duration Complete", AlsoCalled: "Duration Progress",
			Definition: "The percentage of the duration that has gone by since the start of the action: days passed / days of the duration. It depends only on the calendar, not on how much work has been done.",
			Example:    "Day 2 of a 10-day action: Duration Complete = 2 / 10 = 20 %."},
		{Key: "workcomplete", Group: GlossaryCore, Term: "Work Complete", AlsoCalled: "Effort Progress",
			Definition: "The percentage of the estimation that has been worked: Work / Estimation. It answers: how many of the planned work hours have gone into the action? In the program it is shown as \"consumed\" and never above 100 %.",
			Example:    "32 of 80 estimated hours worked: Work Complete = 32 / 80 = 40 %.",
			Note:       "Duration Complete and Work Complete differ even when a project is exactly on plan. After the first 2 of 10 days only 20 % of the duration has passed, but with a strongly front-loaded plan, for example 40 % of the work is planned for that time."},
		{Key: "taskcomplete", Group: GlossaryCore, Term: "Task Complete",
			Definition: "Only a definition - it is not shown in the program. The task is really finished: it has been tested against its requirements. The estimation is only an estimate, not the truth, so Task Complete can differ from Duration Complete and from Work Complete: all the estimated work may be spent and the task still not done, or the task may be done with less work than estimated.",
			Note:       "Not shown in the program."},

		{Key: "contour", Group: GlossaryScreen, Term: "Contour",
			Definition: "The shape that says how the estimated work is spread over the duration of an action: Flat (evenly), Front-Loaded (more at the start), Back-Loaded (more at the end), Double-Peak (more at both ends). The area under the curve is the estimation. The contour None means the action has no estimation and no real effort: the fields are greyed out, the action has no progress bar and takes no bookings."},
		{Key: "consumed", Group: GlossaryScreen, Term: "Consumed", AlsoCalled: "rating, rating with contour",
			Definition: "Shown on the action bar and in the action view when Show Progress is on. The first value, for example 16.4 %, is Work Complete. If the contour is not Flat, a second value follows, for example \"Double-Peak 12.6 %\": the position on the duration up to which the contour's planned work equals the work done so far. If the black marker on the bar is left of the today line, the work is behind the plan."},
		{Key: "externalid", Group: GlossaryScreen, Term: "External ID",
			Definition: "Up to 16 characters. It identifies an action in the booking texts of the external time tracking system: bookings that name it count as Work of this action. If a booking names several IDs, its time is split evenly between them. It can be changed at any time. An action with the contour None is not a real action and cannot carry an External ID."},
		{Key: "defaultaction", Group: GlossaryScreen, Term: "Default Action",
			Definition: "The action that receives the bookings which name no External ID, so that no booked time is lost. It is chosen on the Monitor page."},
	}
}

func glossaryDE() []GlossaryTermDTO {
	return []GlossaryTermDTO{
		{Key: "section", Group: GlossaryCore, Term: "Abschnitt (Section)", AlsoCalled: "Arbeitspaket",
			Definition: "Eine benannte Zeile (Schwimmbahn) auf der Zeichenfläche, die Aktionen (und Meilensteine) gruppiert. Sie hat eine Nummer - 1, 2, 3 ... -, die der erste Teil der Nummern ihrer Aktionen ist (1.1, 1.2 ...), und kann eine eigene Farbe haben. Der Planungsbegriff für einen Abschnitt ist Arbeitspaket."},
		{Key: "workpackage", Group: GlossaryCore, Term: "Arbeitspaket (Work Package)", AlsoCalled: "Abschnitt",
			Definition: "Der Planungsbegriff für einen Abschnitt: eine Gruppe zusammengehöriger Aktionen, die gemeinsam summiert werden, zum Beispiel in der Monitor-Tabelle. Ein Arbeitspaket enthält Aktionen."},
		{Key: "action", Group: GlossaryCore, Term: "Aktion (Action)",
			Definition: "Arbeit, die zu erledigen ist, dargestellt als Balken mit Start, Ende, Schätzung und optionalen Rollen. Eine Aktion kann in Detailaktionen zerlegt werden (öffnen mit dem Drill-Symbol oder Details...)."},
		{Key: "role", Group: GlossaryCore, Term: "Rolle (Role)",
			Definition: "Eine benannte Art von Mitarbeitenden oder Fähigkeit, zum Beispiel Entwickler, die Aktionen zugewiesen werden kann. Der Prozentwert einer Zuweisung ist der Anteil des Arbeitstags, den die Rolle für diese Aktion aufwendet."},
		{Key: "milestone", Group: GlossaryCore, Term: "Meilenstein (Milestone)", AlsoCalled: "Gateway, Gate",
			Definition: "Ein bestimmter Zeitpunkt ohne Dauer, an dem der Projekterfolg gemessen wird. Ein Meilenstein wird zu seinem Datum erreicht oder verfehlt."},
		{Key: "estimation", Group: GlossaryCore, Term: "Schätzung (Estimation)", AlsoCalled: "geplanter Aufwand",
			Definition: "Die geplante Arbeitsmenge für eine Aktion, in Stunden oder Tagen. Sie sagt, wie viel Arbeit geplant ist, nicht wie lange die Aktion dauert (siehe Dauer). Die Schätzung eines Arbeitspakets ist die Summe seiner Aktionen."},
		{Key: "work", Group: GlossaryCore, Term: "Arbeit (Work)", AlsoCalled: "Ist-Aufwand (Feld \"Real\")",
			Definition: "Die tatsächlich für eine Aktion aufgewendete Zeit, in Stunden oder Tagen. Sie wird von Hand eingegeben oder - bei eingeschaltetem Monitoring - aus den Buchungen im externen Zeiterfassungssystem übernommen."},
		{Key: "duration", Group: GlossaryCore, Term: "Dauer (Duration)",
			Definition: "Die Zeit zwischen Start und Ende einer Aktion. Sie weicht meist von der Schätzung ab: Eine Aktion kann 10 Tage dauern, während die geschätzte Arbeit 80 Stunden beträgt. Das Programm zeigt auch die Arbeitstage der Dauer (Montag bis Freitag, ohne Wochenenden)."},
		{Key: "durationcomplete", Group: GlossaryCore, Term: "Dauer abgeschlossen (Duration Complete)", AlsoCalled: "Dauerfortschritt, zeitlicher Fortschritt (Duration Progress)",
			Definition: "Der Prozentsatz der Dauer, der seit dem Beginn der Aktion verstrichen ist: vergangene Tage / Tage der Dauer. Er hängt nur vom Kalender ab, nicht davon, wie viel Arbeit geleistet wurde.",
			Example:    "Tag 2 einer 10-Tage-Aktion: Dauer abgeschlossen = 2 / 10 = 20 %."},
		{Key: "workcomplete", Group: GlossaryCore, Term: "Arbeit abgeschlossen (Work Complete)", AlsoCalled: "Aufwandsfortschritt (Effort Progress)",
			Definition: "Der Prozentsatz der Schätzung, der bereits gearbeitet wurde: Arbeit / Schätzung. Er beantwortet: Wie viele der geplanten Arbeitsstunden sind in die Aktion geflossen? Im Programm heißt der Wert \"consumed\" (verbraucht) und wird höchstens mit 100 % angezeigt.",
			Example:    "32 von 80 geschätzten Stunden gearbeitet: Arbeit abgeschlossen = 32 / 80 = 40 %.",
			Note:       "Dauer abgeschlossen und Arbeit abgeschlossen weichen auch dann voneinander ab, wenn ein Projekt genau im Plan liegt. Nach den ersten 2 von 10 Tagen sind erst 20 % der Dauer vergangen, aber bei einer stark vorgezogenen Verteilung (Front-Loaded) sind für diese Zeit zum Beispiel 40 % der Arbeit geplant."},
		{Key: "taskcomplete", Group: GlossaryCore, Term: "Aufgabe fertig (Task Complete)",
			Definition: "Nur eine Definition - sie wird im Programm nicht angezeigt. Die Aufgabe ist wirklich erledigt: Sie wurde gegen ihre Anforderungen getestet. Die Schätzung ist nur eine Schätzung und nicht die Wahrheit; deshalb kann \"Aufgabe fertig\" von \"Dauer abgeschlossen\" und von \"Arbeit abgeschlossen\" abweichen: Die gesamte geschätzte Arbeit kann aufgebraucht sein, ohne dass die Aufgabe fertig ist, oder die Aufgabe ist mit weniger Arbeit als geschätzt fertig.",
			Note:       "Wird im Programm nicht angezeigt."},

		{Key: "contour", Group: GlossaryScreen, Term: "Kontur (Contour)",
			Definition: "Die Form, die angibt, wie die geschätzte Arbeit über die Dauer einer Aktion verteilt ist: Flat (gleichmäßig), Front-Loaded (mehr am Anfang), Back-Loaded (mehr am Ende), Double-Peak (mehr an beiden Enden). Die Fläche unter der Kurve ist die Schätzung. Die Kontur None bedeutet: Die Aktion hat keine Schätzung und keinen Ist-Aufwand; die Felder sind ausgegraut, die Aktion hat keinen Fortschrittsbalken und nimmt keine Buchungen auf."},
		{Key: "consumed", Group: GlossaryScreen, Term: "Verbraucht (Consumed)", AlsoCalled: "Bewertung, Bewertung mit Kontur",
			Definition: "Wird mit Show Progress am Aktionsbalken und in der Aktionsansicht gezeigt. Der erste Wert, zum Beispiel 16.4 %, ist \"Arbeit abgeschlossen\". Ist die Kontur nicht Flat, folgt ein zweiter Wert, zum Beispiel \"Double-Peak 12.6 %\": die Stelle der Dauer, bis zu der die geplante Arbeit der Kontur der bisher geleisteten Arbeit entspricht. Liegt die schwarze Markierung im Balken links von der Heute-Linie, liegt die Arbeit hinter dem Plan."},
		{Key: "externalid", Group: GlossaryScreen, Term: "Externe ID (External ID)",
			Definition: "Bis zu 16 Zeichen. Sie kennzeichnet eine Aktion in den Buchungstexten des externen Zeiterfassungssystems: Buchungen, die sie nennen, zählen als Arbeit dieser Aktion. Nennt eine Buchung mehrere IDs, wird ihre Zeit gleichmäßig aufgeteilt. Sie kann jederzeit geändert werden. Eine Aktion mit der Kontur None ist keine echte Aktion und kann keine externe ID haben."},
		{Key: "defaultaction", Group: GlossaryScreen, Term: "Standard-Aktion (Default Action)",
			Definition: "Die Aktion, die Buchungen erhält, die keine externe ID nennen, damit keine gebuchte Zeit verloren geht. Sie wird auf der Monitor-Seite gewählt."},
	}
}
