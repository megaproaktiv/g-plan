# Glossar

> Generated from `src/app/glossary.go` (the in-program help shows the same text). Do not edit by hand:
> run `UPDATE_GLOSSARY=1 go test ./... -run GlossaryDoc` after changing the glossary.

Die in diesem Programm verwendeten Begriffe und ihre Bedeutung.

## Begriffe

### Abschnitt (Section)

*Auch genannt:* Arbeitspaket

Eine benannte Zeile (Schwimmbahn) auf der Zeichenfläche, die Aktionen (und Meilensteine) gruppiert. Sie hat eine Nummer - 1, 2, 3 ... -, die der erste Teil der Nummern ihrer Aktionen ist (1.1, 1.2 ...), und kann eine eigene Farbe haben. Der Planungsbegriff für einen Abschnitt ist Arbeitspaket.

### Arbeitspaket (Work Package)

*Auch genannt:* Abschnitt

Der Planungsbegriff für einen Abschnitt: eine Gruppe zusammengehöriger Aktionen, die gemeinsam summiert werden, zum Beispiel in der Monitor-Tabelle. Ein Arbeitspaket enthält Aktionen.

### Aktion (Action)

Arbeit, die zu erledigen ist, dargestellt als Balken mit Start, Ende, Schätzung und optionalen Rollen. Eine Aktion kann in Detailaktionen zerlegt werden (öffnen mit dem Drill-Symbol oder Details...).

### Rolle (Role)

Eine benannte Art von Mitarbeitenden oder Fähigkeit, zum Beispiel Entwickler, die Aktionen zugewiesen werden kann. Der Prozentwert einer Zuweisung ist der Anteil des Arbeitstags, den die Rolle für diese Aktion aufwendet.

### Meilenstein (Milestone)

*Auch genannt:* Gateway, Gate

Ein bestimmter Zeitpunkt ohne Dauer, an dem der Projekterfolg gemessen wird. Ein Meilenstein wird zu seinem Datum erreicht oder verfehlt.

### Schätzung (Estimation)

*Auch genannt:* geplanter Aufwand

Die geplante Arbeitsmenge für eine Aktion, in Stunden oder Tagen. Sie sagt, wie viel Arbeit geplant ist, nicht wie lange die Aktion dauert (siehe Dauer). Die Schätzung eines Arbeitspakets ist die Summe seiner Aktionen.

### Arbeit (Work)

*Auch genannt:* Ist-Aufwand (Feld "Real")

Die tatsächlich für eine Aktion aufgewendete Zeit, in Stunden oder Tagen. Sie wird von Hand eingegeben oder - bei eingeschaltetem Monitoring - aus den Buchungen im externen Zeiterfassungssystem übernommen.

### Dauer (Duration)

Die Zeit zwischen Start und Ende einer Aktion. Sie weicht meist von der Schätzung ab: Eine Aktion kann 10 Tage dauern, während die geschätzte Arbeit 80 Stunden beträgt. Das Programm zeigt auch die Arbeitstage der Dauer (Montag bis Freitag, ohne Wochenenden).

### Dauer abgeschlossen (Duration Complete)

*Auch genannt:* Dauerfortschritt, zeitlicher Fortschritt (Duration Progress)

Der Prozentsatz der Dauer, der seit dem Beginn der Aktion verstrichen ist: vergangene Tage / Tage der Dauer. Er hängt nur vom Kalender ab, nicht davon, wie viel Arbeit geleistet wurde.

*Beispiel:* Tag 2 einer 10-Tage-Aktion: Dauer abgeschlossen = 2 / 10 = 20 %.

### Arbeit abgeschlossen (Work Complete)

*Auch genannt:* Aufwandsfortschritt (Effort Progress)

Der Prozentsatz der Schätzung, der bereits gearbeitet wurde: Arbeit / Schätzung. Er beantwortet: Wie viele der geplanten Arbeitsstunden sind in die Aktion geflossen? Im Programm heißt der Wert "consumed" (verbraucht) und wird höchstens mit 100 % angezeigt.

*Beispiel:* 32 von 80 geschätzten Stunden gearbeitet: Arbeit abgeschlossen = 32 / 80 = 40 %.

*Hinweis:* Dauer abgeschlossen und Arbeit abgeschlossen weichen auch dann voneinander ab, wenn ein Projekt genau im Plan liegt. Nach den ersten 2 von 10 Tagen sind erst 20 % der Dauer vergangen, aber bei einer stark vorgezogenen Verteilung (Front-Loaded) sind für diese Zeit zum Beispiel 40 % der Arbeit geplant.

### Aufgabe fertig (Task Complete)

Nur eine Definition - sie wird im Programm nicht angezeigt. Die Aufgabe ist wirklich erledigt: Sie wurde gegen ihre Anforderungen getestet. Die Schätzung ist nur eine Schätzung und nicht die Wahrheit; deshalb kann "Aufgabe fertig" von "Dauer abgeschlossen" und von "Arbeit abgeschlossen" abweichen: Die gesamte geschätzte Arbeit kann aufgebraucht sein, ohne dass die Aufgabe fertig ist, oder die Aufgabe ist mit weniger Arbeit als geschätzt fertig.

*Hinweis:* Wird im Programm nicht angezeigt.

## Begriffe in der Oberfläche

### Kontur (Contour)

Die Form, die angibt, wie die geschätzte Arbeit über die Dauer einer Aktion verteilt ist: Flat (gleichmäßig), Front-Loaded (mehr am Anfang), Back-Loaded (mehr am Ende), Double-Peak (mehr an beiden Enden). Die Fläche unter der Kurve ist die Schätzung. Die Kontur None bedeutet: Die Aktion hat keine Schätzung und keinen Ist-Aufwand; die Felder sind ausgegraut, die Aktion hat keinen Fortschrittsbalken und nimmt keine Buchungen auf.

### Verbraucht (Consumed)

*Auch genannt:* Bewertung, Bewertung mit Kontur

Wird mit Show Progress am Aktionsbalken und in der Aktionsansicht gezeigt. Der erste Wert, zum Beispiel 16.4 %, ist "Arbeit abgeschlossen". Ist die Kontur nicht Flat, folgt ein zweiter Wert, zum Beispiel "Double-Peak 12.6 %": die Stelle der Dauer, bis zu der die geplante Arbeit der Kontur der bisher geleisteten Arbeit entspricht. Liegt die schwarze Markierung im Balken links von der Heute-Linie, liegt die Arbeit hinter dem Plan.

### Externe ID (External ID)

Bis zu 16 Zeichen. Sie kennzeichnet eine Aktion in den Buchungstexten des externen Zeiterfassungssystems: Buchungen, die sie nennen, zählen als Arbeit dieser Aktion. Nennt eine Buchung mehrere IDs, wird ihre Zeit gleichmäßig aufgeteilt. Sie kann jederzeit geändert werden. Eine Aktion mit der Kontur None ist keine echte Aktion und kann keine externe ID haben.

### Standard-Aktion (Default Action)

Die Aktion, die Buchungen erhält, die keine externe ID nennen, damit keine gebuchte Zeit verloren geht. Sie wird auf der Monitor-Seite gewählt.
