package main

// GUI translations. These cover GUI labels AND the help content. The CLI/chat
// command language is NEVER translated (it stays English) — see CommandHelp /
// ChatHelp, which remain English regardless of the GUI language.

// relationshipRules returns the 7 dependency relationship rules localized for a
// GUI language code ("en"|"de"). The abbreviation follows the language:
//
//	en: FS SS FF SF MS ME MM
//	de: EA AA EE AE MA ME MM
//
// The rule column is language-neutral math (Start/End/Mid); the name and
// example are localized.
func relationshipRules(lang string) []RelationshipRuleDTO {
	if lang == "de" {
		return []RelationshipRuleDTO{
			{Kind: "FS", Abbr: "EA", Name: "Ende-Anfang (Normalfolge)", Rule: "Start(B) ≥ Ende(A)", Example: "Die Umsetzung beginnt, nachdem das Design fertig ist."},
			{Kind: "SS", Abbr: "AA", Name: "Anfang-Anfang (Anfangsfolge)", Rule: "Start(B) ≥ Start(A)", Example: "Das Testen beginnt, sobald die Entwicklung begonnen hat."},
			{Kind: "FF", Abbr: "EE", Name: "Ende-Ende (Endfolge)", Rule: "Ende(B) ≥ Ende(A)", Example: "Die Dokumentation endet, wenn die Entwicklung endet."},
			{Kind: "SF", Abbr: "AE", Name: "Anfang-Ende (Sprungfolge)", Rule: "Ende(B) ≥ Start(A)", Example: "Das alte System läuft, bis das neue live geht."},
			{Kind: "MS", Abbr: "MA", Name: "Mitte-Anfang", Rule: "Start(B) ≥ Mitte(A)", Example: "Die Integration beginnt, sobald die Hälfte der Komponenten gebaut ist."},
			{Kind: "ME", Abbr: "ME", Name: "Mitte-Ende", Rule: "Ende(B) ≥ Mitte(A)", Example: "Der Zwischenbericht kann nicht fertig sein, bevor die Hälfte der Analyse erledigt ist."},
			{Kind: "MM", Abbr: "MM", Name: "Mitte-Mitte", Rule: "Start(A) ≤ Mitte(B) ≤ Ende(A)", Example: "Die Hypercare-Phase hat ihren Höhepunkt, während die Migration noch läuft."},
		}
	}
	return []RelationshipRuleDTO{
		{Kind: "FS", Abbr: "FS", Name: "Finish-to-Start", Rule: "Start(B) ≥ End(A)", Example: "Implementation starts after the design is done."},
		{Kind: "SS", Abbr: "SS", Name: "Start-to-Start", Rule: "Start(B) ≥ Start(A)", Example: "Testing starts once development has started."},
		{Kind: "FF", Abbr: "FF", Name: "Finish-to-Finish", Rule: "End(B) ≥ End(A)", Example: "Documentation finishes when development finishes."},
		{Kind: "SF", Abbr: "SF", Name: "Start-to-Finish", Rule: "End(B) ≥ Start(A)", Example: "The old system runs until the new one goes live."},
		{Kind: "MS", Abbr: "MS", Name: "Middle-to-Start", Rule: "Start(B) ≥ Mid(A)", Example: "Integration starts once half the components are built."},
		{Kind: "ME", Abbr: "ME", Name: "Middle-to-End", Rule: "End(B) ≥ Mid(A)", Example: "The interim report can't be finished before half the analysis is done."},
		{Kind: "MM", Abbr: "MM", Name: "Middle-to-Middle", Rule: "Start(A) ≤ Mid(B) ≤ End(A)", Example: "Hypercare peaks while the migration is still running."},
	}
}

// kindAbbr maps a canonical dependency kind to its abbreviation in a GUI
// language (used by the links table + arrow labels). Unknown kinds echo back.
func kindAbbr(kind, lang string) string {
	if lang == "de" {
		switch kind {
		case "FS":
			return "EA"
		case "SS":
			return "AA"
		case "FF":
			return "EE"
		case "SF":
			return "AE"
		case "MS":
			return "MA"
		case "ME":
			return "ME"
		case "MM", "OV":
			return "MM"
		}
	}
	if kind == "OV" {
		return "MM"
	}
	return kind
}

// guiStrings returns the translated GUI label bundle for a language. Keys are
// stable identifiers the frontend looks up; values are the localized strings.
func guiStrings(lang string) map[string]string {
	en := map[string]string{
		"help.title":             "Help",
		"help.commands":          "Commands",
		"help.relationships":     "Relationships",
		"help.tab.commands":      "Command help",
		"help.tab.relationships": "Relationship help",
		"help.close":             "Close",
		"help.glossary":          "Glossary",
		"help.tab.glossary":      "Glossary",
		"help.monitoring":        "Monitoring",
		"help.tab.monitoring":    "Monitoring help",
		"help.objects":           "Sections and actions",
		"help.tab.objects":       "Help on sections and actions",
		"glossary.intro":         "The words used in this program and what they mean.",
		"glossary.group.core":    "Terms",
		"glossary.group.screen":  "Terms used on screen",
		"glossary.also":          "Also called",
		"glossary.example":       "Example",
		"glossary.note":          "Note",
		"help.col.rel":           "Relationship",
		"help.col.abbr":          "Abbr.",
		"help.col.rule":          "Rule",
		"help.col.example":       "Example",
		"help.commands.note":     "Commands are entered in the command bar at the bottom. The command language is always English.",
		"links.title":            "Links",
		"links.col.from":         "From",
		"links.col.kind":         "Kind",
		"links.col.to":           "To",
		"links.none":             "No dependencies yet.",
		"settings.language":      "Language",
	}
	if lang == "de" {
		de := map[string]string{
			"help.title":             "Hilfe",
			"help.commands":          "Befehle",
			"help.relationships":     "Beziehungen",
			"help.tab.commands":      "Befehlshilfe",
			"help.tab.relationships": "Beziehungshilfe",
			"help.close":             "Schließen",
			"help.glossary":          "Glossar",
			"help.tab.glossary":      "Glossar",
			"help.monitoring":        "Monitoring",
			"help.tab.monitoring":    "Monitoring-Hilfe",
			"help.objects":           "Abschnitte und Aktionen",
			"help.tab.objects":       "Hilfe zu Abschnitten und Aktionen",
			"glossary.intro":         "Die in diesem Programm verwendeten Begriffe und ihre Bedeutung.",
			"glossary.group.core":    "Begriffe",
			"glossary.group.screen":  "Begriffe in der Oberfläche",
			"glossary.also":          "Auch genannt",
			"glossary.example":       "Beispiel",
			"glossary.note":          "Hinweis",
			"help.col.rel":           "Beziehung",
			"help.col.abbr":          "Kürzel",
			"help.col.rule":          "Regel",
			"help.col.example":       "Beispiel",
			"help.commands.note":     "Befehle werden in der Befehlszeile unten eingegeben. Die Befehlssprache ist immer Englisch.",
			"links.title":            "Verknüpfungen",
			"links.col.from":         "Von",
			"links.col.kind":         "Art",
			"links.col.to":           "Nach",
			"links.none":             "Noch keine Abhängigkeiten.",
			"settings.language":      "Sprache",
		}
		// fall back to EN for any missing key
		for k, v := range en {
			if _, ok := de[k]; !ok {
				de[k] = v
			}
		}
		return de
	}
	return en
}
