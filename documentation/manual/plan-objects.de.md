# Abschnitte und Aktionen

> Generated from `src/app/objecthelp.go` (the in-program help shows the same text). Do not edit by hand:
> run `UPDATE_GLOSSARY=1 go test . -run ObjectGuideDoc` after changing the help.

Ein Plan besteht aus Abschnitten, die Aktionen zusammenfassen. Diese Seite erklärt beide und die Einstellungen einer Aktion, vor allem ihre Kontur.

## Abschnitte

Ein **Abschnitt** ist eine benannte Bahn auf der Zeichenfläche, die zusammengehörige Aktionen und Meilensteine zusammenfasst. Der Planungsbegriff für einen Abschnitt ist *Arbeitspaket*.

- **Nummer.** Abschnitte sind von oben nach unten mit 1, 2, 3 ... nummeriert. Die Nummer ist der erste Teil der Nummern seiner Aktionen: Die zweite Aktion von Abschnitt 3 ist `3.2`. Die Nummern folgen der Reihenfolge auf dem Bildschirm und ändern sich daher, wenn Sie die Reihenfolge ändern.
- **Farbe.** Jeder Abschnitt kann eine eigene Farbe haben.
- **Anlegen.** **New Section** im Menü der leeren Zeichenfläche anklicken oder in der Befehlszeile `add section <Name>` eingeben. Er wird am Ende angefügt (nach der letzten WP-Nummer). Der neue Abschnitt wird zum aktuellen, und neue Aktionen kommen in den aktuellen Abschnitt.
- **Umbenennen und löschen.** Den Abschnitt auswählen (auf sein Etikett klicken), um ihn umzubenennen, seine Farbe zu ändern oder ihn zu löschen. Beim Löschen eines Abschnitts werden seine Aktionen nicht gelöscht: Sie wandern in die Standard-Bahn. Löschen lässt sich mit Cmd-Z rückgängig machen.
- **Reihenfolge.** Der Abschnitt lässt sich mit den Schaltflächen **▲ up** und **▼ down** des Abschnitts und mit **Move Section Up / Down** im Rechtsklick-Menü des Abschnitts-Etiketts nach oben und unten schieben. Das geht für jeden Abschnitt, **auch für einen Abschnitt ohne Aktionen**. Die Reihenfolge wird mit dem Plan gespeichert. Die Nummern aller Aktionen folgen.
- **Die Bahn Phases** ganz oben ist eine Ansicht der Projektphasen, kein Abschnitt. Sie hat keine Nummer, lässt sich weder verschieben noch löschen und kann in den Projekteinstellungen ein- und ausgeschaltet werden.

> Ein Abschnitt hat keine eigene Schätzung. Sein Aufwand ist die Summe seiner Aktionen.

## Aktionen

Eine **Aktion** ist ein Stück Arbeit mit Start, Ende, Schätzung und, wenn sie läuft, einem Ist-Aufwand. Sie gehört zu genau einem Abschnitt. Per Doppelklick auf die Aktion oder **Edit Action** im Rechtsklick-Menü öffnet sich der Editor.

| Feld | Bedeutung |
|---|---|
| **Name**, **Beschreibung** | Was die Aktion ist. Der Name steht auf dem Balken. |
| **Start**, **Ende** | Die Dauer: die Zeitspanne des Balkens. Zum Verschieben den Balken ziehen, zum Ändern der Länge seine Enden. Eine Verschiebung lässt sich mit Cmd-Z rückgängig machen. |
| **Estimated** (der geplante Aufwand) | Die Arbeit, die Sie aufwenden wollen, in der Einheit des Projekts. Sie hängt nicht von der Dauer ab. In der Befehlszeile: `est 3.1 20`. |
| **Real** | Die bisher wirklich geleistete Arbeit. Sie wird von Hand eingegeben oder, wenn das Monitoring eingeschaltet und die Aktion verfolgt ist, aus den Zeitbuchungen übernommen und ausgegraut. |
| **Contour** | Wie die geschätzte Arbeit über die Dauer verteilt ist. Siehe nächster Abschnitt. |
| **External ID** | Bis zu 16 Zeichen, die die Aktion in den Buchungstexten des Zeiterfassungssystems kennzeichnen. Siehe Monitoring-Hilfe. |
| **Details** | Unteraktionen mit eigener Schätzung. Wird eine Schätzung an der Aktion selbst eingegeben, gilt dieser Wert fest, und die Details teilen ihn untereinander auf. |

## Die Kontur-Einstellungen einer Aktion

Die **Kontur** ist die geplante *Dichte* der Arbeit über die Dauer der Aktion: Die Höhe der Kurve ist die geplante Arbeit pro Tag an dieser Stelle. Die **Fläche** unter der Kurve ist die Schätzung. Die Kontur sagt also, wo auf dem Balken Sie die Arbeit erwarten. Sie ändert nie die Schätzung, sondern nur die Stelle, an der die Fortschrittsmarke gezeichnet wird.

| Kontur | Form | Verwenden, wenn ... |
|---|---|---|
| **Flat** | Gleichbleibende Höhe. Die Arbeit ist gleichmäßig verteilt. | die Arbeit an jedem Tag etwa gleich ist. Das ist die Vorgabe. |
| **Front-Loaded** | Hoch am Anfang, zum Ende hin abnehmend. | der größte Teil der Arbeit früh anfällt, etwa Aufbau oder Analyse. |
| **Back-Loaded** | Niedrig am Anfang, zum Ende hin ansteigend. | die Arbeit anwächst und der größte Teil spät anfällt, etwa Test und Einführung. |
| **Double-Peak** | Hoch an beiden Enden, in der Mitte wenig. | es am Anfang und am Ende viel zu tun gibt und dazwischen eine ruhige Zeit liegt. |
| **None** | Keine Kurve. | der Eintrag keine echte Arbeit ist, sondern nur ein Platzhalter, etwa eine Überschrift. Siehe unten. |

**Wie der Fortschritt angezeigt wird (Show Progress).** Bei eingeschaltetem **Show Progress** zeigt der Balken einer Aktion zwei Dinge:

- Den **verbrauchten** Anteil: Ist-Aufwand geteilt durch Schätzung, zum Beispiel `16,4 %`. Das ist die *erledigte Arbeit* (Work Complete) der Aktion.
- Ist die Kontur nicht Flat, folgt ein zweiter Wert, zum Beispiel `Double-Peak 12,6 %`. Das ist die Stelle auf der Dauer, bis zu der die **geplante** Arbeit entlang der Kurve der bisher geleisteten Arbeit entspricht. Dort wird die schwarze **Marke** auf dem Balken gezeichnet.

Vergleichen Sie die Marke mit der *Heute*-Linie. Bei **Front-Loaded** ist viel Arbeit früh geplant, die Marke bleibt daher hinter dem verbrauchten Anteil zurück: Es muss viel Arbeit geleistet werden, bis sie weit wandert. Bei **Back-Loaded** läuft die Marke voraus. Liegt die Marke links von heute, hinkt die Aktion ihrem Plan hinterher. Bei **Flat** steht die Marke einfach beim verbrauchten Anteil.

**Die Kontur None.** Eine Aktion mit der Kontur **None** hat keine Schätzung und keinen Ist-Aufwand: Beide Felder sind ausgegraut, die gespeicherten Zahlen bleiben erhalten (sie kommen zurück, wenn Sie eine andere Kontur wählen). Sie hat keine Fortschrittsmarke und keine Kurve, nimmt keine Zeitbuchungen auf, kann keine externe ID tragen und nicht die Standard-Aktion sein. Verwenden Sie sie für Einträge, die nur etwas markieren oder gruppieren.

> Eine andere Kontur ändert, wie der Fortschritt angezeigt wird, aber nie die Schätzung, den Ist-Aufwand oder die Summen des Plans.
