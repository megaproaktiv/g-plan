package main

// The help about the plan's building blocks: sections and actions (including
// the contour settings). Text is data, in both GUI languages, in the same
// structure as the monitoring help. documentation/plan-objects*.md is generated
// from it (a test keeps it in sync).

// ObjectHelp returns the help page about sections and actions in a GUI language.
func (a *App) ObjectHelp(lang string) HelpPageDTO { return objectHelp(lang) }

func objectHelp(lang string) HelpPageDTO {
	if lang == "de" {
		return objectHelpDE()
	}
	return objectHelpEN()
}

func objectHelpEN() HelpPageDTO {
	return HelpPageDTO{
		Title: "Sections and actions",
		Intro: "A plan is built from sections, which group actions. This page explains both, and the settings of an action, above all its contour.",
		Sections: []HelpSectionDTO{
			{Key: "sections", Heading: "Sections", Blocks: []HelpBlockDTO{
				hp("A **section** is a named lane on the canvas that groups related actions and milestones. The planning name for a section is *work package*."),
				hul(
					"**Number.** Sections are numbered 1, 2, 3 ... from top to bottom. The number is the first part of the numbers of its actions: the second action of section 3 is `3.2`. The numbers follow the order on the screen, so they change when you change the order.",
					"**Color.** Each section can have its own color.",
					"**Create.** Click **New Section** in the menu of the empty canvas, or type `add section <name>` in the command bar. It is added at the end (after the last WP number). The new section becomes the current one, and new actions go into the current section.",
					"**Rename and delete.** Select the section (click its label) to rename it, change its color or delete it. Deleting a section does not delete its actions: they move to the default lane. Deleting can be undone with Cmd-Z.",
					"**Order.** The section can be moved up and down with the **▲ up** and **▼ down** buttons of the section, and with **Move Section Up / Down** in the right-click menu of the section label. This works for every section, **also for a section without actions**. The order is stored with the plan. The numbers of all actions follow.",
					"**The Phases lane** at the top is a view of the project phases, not a section. It has no number, cannot be moved and cannot be deleted. It can be switched on and off in the project settings.",
				),
				hnote("A section does not have an estimation of its own. Its effort is the sum of its actions."),
			}},
			{Key: "actions", Heading: "Actions", Blocks: []HelpBlockDTO{
				hp("An **action** is a piece of work with a start, an end, an estimation and, when it is under way, a real effort. It belongs to one section. Double-click an action, or choose **Edit Action** in its right-click menu, to open its editor."),
				htable([]string{"Field", "Meaning"}, [][]string{
					{"**Name**, **Description**", "What the action is. The name is shown on the bar."},
					{"**Start**, **End**", "The duration: the time span of the bar. Drag the bar to move it, drag its ends to resize it. A move can be undone with Cmd-Z."},
					{"**Estimated** (the Planned effort)", "The work you plan to spend, in the unit of the project. It does not depend on the duration. In the command bar: `est 3.1 20`."},
					{"**Real**", "The work really spent so far. It is entered by hand, or, when monitoring is on and the action is tracked, taken from the time bookings and greyed out."},
					{"**Contour**", "How the estimated work is spread over the duration. See the next section."},
					{"**External ID**", "Up to 16 characters that identify the action in the booking texts of the time tracking system. See the monitoring help."},
					{"**Details**", "Sub-actions with their own estimation. Typing an estimation on the action itself makes that value the fixed one, and the details share it."},
				}),
			}},
			{Key: "contour", Heading: "The contour settings of an action", Blocks: []HelpBlockDTO{
				hp("The **contour** is the planned *density* of work over the duration of the action: the height of the curve is the planned work per day at that point. The **area** under the curve is the estimation. So the contour tells where along the bar you expect the work to happen. It never changes the estimation, only where the progress marker is drawn."),
				htable([]string{"Contour", "Shape", "Use it when ..."}, [][]string{
					{"**Flat**", "Constant height. The work is spread evenly.", "the work is about the same every day. This is the default."},
					{"**Front-Loaded**", "High at the start, tapering off to the end.", "most of the work is done early, for example set-up or analysis."},
					{"**Back-Loaded**", "Low at the start, rising towards the end.", "the work builds up and most of it happens late, for example testing and roll-out."},
					{"**Double-Peak**", "High at both ends, light in the middle.", "there is a busy start and a busy end with a quiet time in between."},
					{"**None**", "No curve.", "the entry is not real work, but only a placeholder, for example a heading. See below."},
				}),
				hp("**How the progress is shown (Show Progress).** When **Show Progress** is on, the bar of an action shows two things:"),
				hul(
					"The **consumed** share: real effort divided by estimation, for example `16.4 %`. This is the *Work Complete* of the action.",
					"If the contour is not Flat, a second value follows, for example `Double-Peak 12.6 %`. It is the position on the duration up to which the **planned** work, following the curve, equals the work done so far. The black **marker** on the bar is drawn there.",
				),
				hp("Compare the marker with the *today* line. With a **Front-Loaded** contour much of the work is planned early, so the marker lags behind the share consumed: a lot of work must be done before it moves far. With a **Back-Loaded** contour the marker runs ahead. If the marker is left of today, the action is behind its plan. With **Flat** the marker is simply at the consumed share."),
				hp("**The contour None.** An action with the contour **None** has no estimation and no real effort: both fields are greyed out and the stored numbers are kept (they come back when you switch to another contour). It has no progress marker and no curve, it takes no time bookings, cannot carry an External ID and cannot be the default action. Use it for entries that only mark or group something."),
				hnote("Changing the contour changes how the progress is shown, but never the estimation, the real effort or the totals of the plan."),
			}},
		},
	}
}

func objectHelpDE() HelpPageDTO {
	return HelpPageDTO{
		Title: "Abschnitte und Aktionen",
		Intro: "Ein Plan besteht aus Abschnitten, die Aktionen zusammenfassen. Diese Seite erklärt beide und die Einstellungen einer Aktion, vor allem ihre Kontur.",
		Sections: []HelpSectionDTO{
			{Key: "sections", Heading: "Abschnitte", Blocks: []HelpBlockDTO{
				hp("Ein **Abschnitt** ist eine benannte Bahn auf der Zeichenfläche, die zusammengehörige Aktionen und Meilensteine zusammenfasst. Der Planungsbegriff für einen Abschnitt ist *Arbeitspaket*."),
				hul(
					"**Nummer.** Abschnitte sind von oben nach unten mit 1, 2, 3 ... nummeriert. Die Nummer ist der erste Teil der Nummern seiner Aktionen: Die zweite Aktion von Abschnitt 3 ist `3.2`. Die Nummern folgen der Reihenfolge auf dem Bildschirm und ändern sich daher, wenn Sie die Reihenfolge ändern.",
					"**Farbe.** Jeder Abschnitt kann eine eigene Farbe haben.",
					"**Anlegen.** **New Section** im Menü der leeren Zeichenfläche anklicken oder in der Befehlszeile `add section <Name>` eingeben. Er wird am Ende angefügt (nach der letzten WP-Nummer). Der neue Abschnitt wird zum aktuellen, und neue Aktionen kommen in den aktuellen Abschnitt.",
					"**Umbenennen und löschen.** Den Abschnitt auswählen (auf sein Etikett klicken), um ihn umzubenennen, seine Farbe zu ändern oder ihn zu löschen. Beim Löschen eines Abschnitts werden seine Aktionen nicht gelöscht: Sie wandern in die Standard-Bahn. Löschen lässt sich mit Cmd-Z rückgängig machen.",
					"**Reihenfolge.** Der Abschnitt lässt sich mit den Schaltflächen **▲ up** und **▼ down** des Abschnitts und mit **Move Section Up / Down** im Rechtsklick-Menü des Abschnitts-Etiketts nach oben und unten schieben. Das geht für jeden Abschnitt, **auch für einen Abschnitt ohne Aktionen**. Die Reihenfolge wird mit dem Plan gespeichert. Die Nummern aller Aktionen folgen.",
					"**Die Bahn Phases** ganz oben ist eine Ansicht der Projektphasen, kein Abschnitt. Sie hat keine Nummer, lässt sich weder verschieben noch löschen und kann in den Projekteinstellungen ein- und ausgeschaltet werden.",
				),
				hnote("Ein Abschnitt hat keine eigene Schätzung. Sein Aufwand ist die Summe seiner Aktionen."),
			}},
			{Key: "actions", Heading: "Aktionen", Blocks: []HelpBlockDTO{
				hp("Eine **Aktion** ist ein Stück Arbeit mit Start, Ende, Schätzung und, wenn sie läuft, einem Ist-Aufwand. Sie gehört zu genau einem Abschnitt. Per Doppelklick auf die Aktion oder **Edit Action** im Rechtsklick-Menü öffnet sich der Editor."),
				htable([]string{"Feld", "Bedeutung"}, [][]string{
					{"**Name**, **Beschreibung**", "Was die Aktion ist. Der Name steht auf dem Balken."},
					{"**Start**, **Ende**", "Die Dauer: die Zeitspanne des Balkens. Zum Verschieben den Balken ziehen, zum Ändern der Länge seine Enden. Eine Verschiebung lässt sich mit Cmd-Z rückgängig machen."},
					{"**Estimated** (der geplante Aufwand)", "Die Arbeit, die Sie aufwenden wollen, in der Einheit des Projekts. Sie hängt nicht von der Dauer ab. In der Befehlszeile: `est 3.1 20`."},
					{"**Real**", "Die bisher wirklich geleistete Arbeit. Sie wird von Hand eingegeben oder, wenn das Monitoring eingeschaltet und die Aktion verfolgt ist, aus den Zeitbuchungen übernommen und ausgegraut."},
					{"**Contour**", "Wie die geschätzte Arbeit über die Dauer verteilt ist. Siehe nächster Abschnitt."},
					{"**External ID**", "Bis zu 16 Zeichen, die die Aktion in den Buchungstexten des Zeiterfassungssystems kennzeichnen. Siehe Monitoring-Hilfe."},
					{"**Details**", "Unteraktionen mit eigener Schätzung. Wird eine Schätzung an der Aktion selbst eingegeben, gilt dieser Wert fest, und die Details teilen ihn untereinander auf."},
				}),
			}},
			{Key: "contour", Heading: "Die Kontur-Einstellungen einer Aktion", Blocks: []HelpBlockDTO{
				hp("Die **Kontur** ist die geplante *Dichte* der Arbeit über die Dauer der Aktion: Die Höhe der Kurve ist die geplante Arbeit pro Tag an dieser Stelle. Die **Fläche** unter der Kurve ist die Schätzung. Die Kontur sagt also, wo auf dem Balken Sie die Arbeit erwarten. Sie ändert nie die Schätzung, sondern nur die Stelle, an der die Fortschrittsmarke gezeichnet wird."),
				htable([]string{"Kontur", "Form", "Verwenden, wenn ..."}, [][]string{
					{"**Flat**", "Gleichbleibende Höhe. Die Arbeit ist gleichmäßig verteilt.", "die Arbeit an jedem Tag etwa gleich ist. Das ist die Vorgabe."},
					{"**Front-Loaded**", "Hoch am Anfang, zum Ende hin abnehmend.", "der größte Teil der Arbeit früh anfällt, etwa Aufbau oder Analyse."},
					{"**Back-Loaded**", "Niedrig am Anfang, zum Ende hin ansteigend.", "die Arbeit anwächst und der größte Teil spät anfällt, etwa Test und Einführung."},
					{"**Double-Peak**", "Hoch an beiden Enden, in der Mitte wenig.", "es am Anfang und am Ende viel zu tun gibt und dazwischen eine ruhige Zeit liegt."},
					{"**None**", "Keine Kurve.", "der Eintrag keine echte Arbeit ist, sondern nur ein Platzhalter, etwa eine Überschrift. Siehe unten."},
				}),
				hp("**Wie der Fortschritt angezeigt wird (Show Progress).** Bei eingeschaltetem **Show Progress** zeigt der Balken einer Aktion zwei Dinge:"),
				hul(
					"Den **verbrauchten** Anteil: Ist-Aufwand geteilt durch Schätzung, zum Beispiel `16,4 %`. Das ist die *erledigte Arbeit* (Work Complete) der Aktion.",
					"Ist die Kontur nicht Flat, folgt ein zweiter Wert, zum Beispiel `Double-Peak 12,6 %`. Das ist die Stelle auf der Dauer, bis zu der die **geplante** Arbeit entlang der Kurve der bisher geleisteten Arbeit entspricht. Dort wird die schwarze **Marke** auf dem Balken gezeichnet.",
				),
				hp("Vergleichen Sie die Marke mit der *Heute*-Linie. Bei **Front-Loaded** ist viel Arbeit früh geplant, die Marke bleibt daher hinter dem verbrauchten Anteil zurück: Es muss viel Arbeit geleistet werden, bis sie weit wandert. Bei **Back-Loaded** läuft die Marke voraus. Liegt die Marke links von heute, hinkt die Aktion ihrem Plan hinterher. Bei **Flat** steht die Marke einfach beim verbrauchten Anteil."),
				hp("**Die Kontur None.** Eine Aktion mit der Kontur **None** hat keine Schätzung und keinen Ist-Aufwand: Beide Felder sind ausgegraut, die gespeicherten Zahlen bleiben erhalten (sie kommen zurück, wenn Sie eine andere Kontur wählen). Sie hat keine Fortschrittsmarke und keine Kurve, nimmt keine Zeitbuchungen auf, kann keine externe ID tragen und nicht die Standard-Aktion sein. Verwenden Sie sie für Einträge, die nur etwas markieren oder gruppieren."),
				hnote("Eine andere Kontur ändert, wie der Fortschritt angezeigt wird, aber nie die Schätzung, den Ist-Aufwand oder die Summen des Plans."),
			}},
		},
	}
}
