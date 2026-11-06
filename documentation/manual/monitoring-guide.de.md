# Monitoring

> Generated from `src/app/monitoringhelp.go` (the in-program help shows the same text). Do not edit by hand:
> run `UPDATE_GLOSSARY=1 go test . -run GuideDoc` after changing the help.

Monitoring vergleicht Ihren Plan mit den Stunden, die in einem externen Zeiterfassungssystem (derzeit Aerion) wirklich gebucht wurden. Es liest dort nur und ändert nichts.

## Wie alles zusammenhängt

Drei Dinge verbinden Ihren Plan mit den Buchungen:

- **System** und **Projekt-ID** (einmal je Projekt auf der Seite **Monitor**): welches Zeiterfassungssystem und welche Nummer das Projekt dort hat.
- **Externe ID** (je Aktion im Aktions-Editor): ein kurzes Kennzeichen mit bis zu 16 Zeichen, zum Beispiel `WP2`.
- Der **Buchungstext**: Wer Zeit bucht, schreibt die externe ID in den Text der Buchung im externen System.

Eine Synchronisation liest alle Buchungen des externen Projekts vom Projektstart bis zum Projektende und sucht in den Buchungstexten nach den externen IDs Ihrer Aktionen. Die gebuchte Zeit wird zum **Ist-Aufwand** (Arbeit) der Aktion, und der Plan zeigt, wie viel der Schätzung verbraucht ist.

## Wie die externe ID gedeutet wird

Die externe ID wird **im Text jeder Buchung** gesucht, sonst nirgends. Eine Buchung gehört zu einer Aktion, wenn ihr Text die externe ID der Aktion enthält:

- **Als eigenes Wort.** Die ID zählt nur, wenn das Zeichen direkt davor und direkt danach kein Buchstabe und keine Ziffer ist. `WP1` wird in `WP1 kickoff`, in `(wp1)` und in `design / WP1 / review` gefunden, aber nicht in `WP12` und nicht in `xWP1`.
- **Groß- und Kleinschreibung und Bindestriche spielen keine Rolle.** `WP1`, `wp1`, `Wp1`, `WP-1` und `wp-1` sind dieselbe ID, im ID-Feld wie im Buchungstext. Ein Leerzeichen zählt nicht als Bindestrich: `WP 1` ist nicht `WP1`.
- **Mehrere IDs in einer Buchung** (zum Beispiel `WP2 and WP3 pairing session`): Die gebuchte Zeit wird **gleichmäßig auf diese Aktionen verteilt**. Sekunden, die sich nicht gleichmäßig teilen lassen, gehen an die zuerst genannten, so geht keine Zeit verloren.
- **Dieselbe ID zweimal** in einem Text zählt einmal. Eine ID, die nur innerhalb einer längeren ID vorkommt (die `1` in `Phase 1`), zählt nicht als eigene.
- **Keine ID im Text:** Die Buchung ist *nicht zugeordnet*. Sie wird der **Standard-Aktion** zugerechnet, falls Sie eine gewählt haben, sonst erscheint sie nur in der Zeile *Bookings without a matching External ID*.
- **Welche Zeit:** die erfasste Dauer jeder Buchung. Gelesen werden nur Arbeitsbuchungen, keine Pausen und keine Arbeitszeit-Einträge.
- **Regeln für die ID selbst:** höchstens 16 Zeichen, im Projekt eindeutig (Groß-/Kleinschreibung und Bindestriche ignoriert: `WP1` und `WP-1` gelten als gleich), und eine Aktion mit der Kontur **None** kann keine haben. Sie lässt sich jederzeit ändern, auch bei eingeschaltetem Monitoring.

Welche dieser Buchungstexte werden für eine Aktion mit der externen ID `WP1` gefunden?

| Buchungstext | Für WP1 gefunden? | Warum |
|---|---|---|
| `WP1 kickoff meeting` | ja | die ID steht als eigenes Wort |
| `wp1: design` | ja | Groß- und Kleinschreibung spielt keine Rolle |
| `(WP1)` | ja | Klammern und Satzzeichen sind in Ordnung |
| `design / WP1 / review` | ja | sie darf mitten im Text stehen |
| `WP12 review` | nein | WP12 ist eine andere, längere ID |
| `xWP1 review` | nein | direkt davor steht ein Buchstabe: kein eigenes Wort |
| `WP-1 kickoff` | ja | ein Bindestrich in der ID spielt keine Rolle: WP-1 ist WP1 |
| `WP 1 kickoff` | nein | ein Leerzeichen ist kein Bindestrich: WP 1 sind zwei Wörter |
| `Daily standup` | nein | keine ID im Text |

> Tipp: Wählen Sie IDs, die in normalen Wörtern nicht vorkommen, etwa `WP1`, `WP2`, `WP3`, und bitten Sie das Team, sie an den Anfang des Buchungstexts zu schreiben.

## Schritt für Schritt, mit Beispiel

Das Beispielprojekt ist *Skynet MVP*, geplant in Tagen mit 8 Stunden pro Tag. Die Projektnummer in Aerion ist `555555`. Es hat drei Aktionen: **Project Management** (externe ID `WP1`, 40 Tage geplant), **AWS Landing Zone** (`WP2`, 10 Tage = 80 h) und **ACC Core configuration** (`WP3`, 12 Tage = 96 h).

1. **Projekt öffnen** und Start- und Enddatum prüfen. Die Buchungen werden vom Projektstart bis zum Projektende geholt.
2. **Monitor** anklicken, dann **Configure System…**. Basis-URL, Benutzername, Passwort, Client-ID und Client-Secret von Aerion eintragen und speichern. Sie werden nur für Sie in `~/.pt-pm/config.yaml` abgelegt und nie wieder angezeigt.
3. Weiter auf der Seite **Monitor**: das System **Aerion** wählen und die Projektnummer (`555555`) eintragen. Mit **Test** zeigt *Connected: project …*, dass Zugangsdaten und Nummer stimmen.
4. **Jeder Aktion eine externe ID geben.** Die Aktion öffnen (Doppelklick), `WP1`, `WP2` oder `WP3` in **External ID** eintragen und Enter drücken oder woanders hinklicken. Das Team bitten, die ID in den Text seiner Buchungen zu schreiben.
5. **Optional: die Standard-Aktion wählen** (Auswahlfeld *Default action*) für Buchungen ohne ID, damit keine gebuchte Zeit verloren geht. Im Beispiel ist es *Project Management*.
6. **Optional:** *also one month before the project start* oder *also one month after the project end* ankreuzen, wenn Buchungen früher beginnen oder überlaufen.
7. **Monitoring einschalten** (Kontrollkästchen *Monitoring on*). Dafür braucht es ein System und eine Projektnummer.
8. **Synchronize** anklicken. Ein Fortschrittsbalken und die Zahl der geholten Buchungen werden angezeigt.
9. **Ergebnis lesen.** Die Tabelle zeigt je Abschnitt die Schätzung der verfolgten Aktionen gegen den Ist-Aufwand, der **Rest** (Schätzung minus Ist, negativ bei Überschreitung) und **Rest %** (sein Anteil an der Schätzung), die Projektsumme und die Zeile für nicht zugeordnete Buchungen. Bei eingeschaltetem *Show Progress* zeigen die Balken den verbrauchten Anteil, und die Aktionsansicht zeigt geplanten und tatsächlichen Aufwand. **Export** (neben Synchronize) schreibt die Tabelle in zwei Markdown-Dateien: `<Name>-compact.md` mit den Arbeitspaketen (Abschnitten) und der Projektsumme, und `<Name>-detailed.md`, die zusätzlich die verfolgten Aktionen jedes Arbeitspakets auflistet.
10. **Korrigieren und wiederholen.** Sind Buchungen nicht oder falsch zugeordnet, den Buchungstext im externen System oder die externe ID korrigieren und erneut synchronisieren. Jede Synchronisation überschreibt den Ist-Aufwand der verfolgten Aktionen.

Das sind die Buchungen im externen System und was das Programm daraus macht:

| Buchungstext | Zeit | Gezählt für |
|---|---|---|
| `WP2 VPC design` | 3 h | WP2 3 h |
| `wp2: security review` | 2 h | WP2 2 h |
| `WP2 and WP3 pairing session` | 4 h | WP2 2 h, WP3 2 h |
| `WP3 account structure` | 5 h | WP3 5 h |
| `Daily standup` | 1 h | Project Management: 1 h |

Nach der Synchronisation zeigen die Aktionen:

| Aktion | Geplant | Ist | Verbraucht |
|---|---|---|---|
| AWS Landing Zone (WP2) | 10 Tage (80 h) | 0,88 Tage (7 h) | 8,7% |
| ACC Core configuration (WP3) | 12 Tage (96 h) | 0,88 Tage (7 h) | 7,2% |
| Project Management (WP1) | 40 Tage (320 h) | 0,13 Tage (1 h) | 0,3% |

> *Verbraucht* ist der Ist-Aufwand geteilt durch den geplanten Aufwand. Project Management hat auch die 1 h der Buchung ohne ID erhalten, weil es die Standard-Aktion ist.

## Wenn es nicht aufgeht

- **Einer Aktion wird nichts zugeordnet:** Die externe ID steht nicht im Buchungstext oder nicht als eigenes Wort. `WP 1` (mit Leerzeichen) ist nicht `WP1`.
- **Alles ist nicht zugeordnet:** Noch keine Aktion hat eine externe ID, oder die Projektnummer ist falsch. Auf der Seite Monitor **Test** benutzen.
- **Zu viele nicht zugeordnete Buchungen:** das Team bitten, IDs zu ergänzen, oder eine Standard-Aktion wählen.
- **Das Feld Real einer Aktion ist ausgegraut:** verfolgte Aktionen (mit externer ID oder die Standard-Aktion) bekommen ihren Ist-Aufwand aus der Synchronisation. Um ihn von Hand einzugeben, die externe ID entfernen oder das Monitoring ausschalten.
- **Eine Aktion nimmt gar nicht teil:** die Kontur **None** bedeutet keine Schätzung, keinen Ist-Aufwand und keine externe ID. Die Spur **Phases** ist eine Ansicht und kein Arbeitspaket: Das Monitoring ignoriert sie, und ihre Aktionen brauchen keine externe ID.
- **„Aerion is not configured“:** **Monitor** öffnen und **Configure System…** wählen.
- **Das Ergebnis ist verschwunden:** Eine geänderte externe ID, Standard-Aktion oder Zeitspanne löscht das letzte Ergebnis. Erneut synchronisieren.
