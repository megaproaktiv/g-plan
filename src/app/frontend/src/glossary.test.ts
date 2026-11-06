import { expect, test } from "bun:test";
import { glossaryHTML, termHTML } from "./glossary";
import type { GlossaryTerm } from "./types";

const t = (over: Partial<GlossaryTerm> = {}): GlossaryTerm => ({
  key: "k", group: "core", term: "Term", alsoCalled: "", definition: "A definition.", example: "", note: "", ...over,
});
const tr = (_k: string, d: string) => d;
const de: Record<string, string> = { "glossary.also": "Auch genannt", "glossary.example": "Beispiel", "glossary.note": "Hinweis",
  "glossary.group.core": "Begriffe", "glossary.group.screen": "Begriffe in der Oberfläche", "glossary.intro": "Die Begriffe." };
const trDe = (k: string, d: string) => de[k] ?? d;

test("a term shows its name, synonyms, definition, example and note", () => {
  const html = termHTML(t({ term: "Work Complete", alsoCalled: "Effort Progress", example: "32 of 80 = 40 %", note: "Careful." }), tr);
  expect(html).toContain("<dt>Work Complete");
  expect(html).toContain("Also called: Effort Progress");
  expect(html).toContain("A definition.");
  expect(html).toContain("<b>Example:</b> 32 of 80 = 40 %");
  expect(html).toContain("<b>Note:</b> Careful.");
  // optional parts are left out when empty
  const bare = termHTML(t(), tr);
  expect(bare).not.toContain("Also called");
  expect(bare).not.toContain("Example");
  expect(bare).not.toContain("Note:");
});

test("the labels follow the language", () => {
  const html = termHTML(t({ alsoCalled: "Aufwandsfortschritt", example: "x", note: "y" }), trDe);
  expect(html).toContain("Auch genannt: Aufwandsfortschritt");
  expect(html).toContain("<b>Beispiel:</b>");
  expect(html).toContain("<b>Hinweis:</b>");
});

test("groups: core terms first, then the on-screen terms; an empty group is omitted", () => {
  const terms = [t({ key: "b", group: "screen", term: "Consumed" }), t({ key: "a", group: "core", term: "Section" })];
  const html = glossaryHTML(terms, tr);
  expect(html.indexOf("Section")).toBeLessThan(html.indexOf("Consumed"));
  expect(html.indexOf("<h3>Terms</h3>")).toBeLessThan(html.indexOf("<h3>Terms used on screen</h3>"));
  const onlyCore = glossaryHTML([t()], tr);
  expect(onlyCore).not.toContain("Terms used on screen");
  expect(glossaryHTML(terms, trDe)).toContain("<h3>Begriffe in der Oberfläche</h3>");
});

test("text from the data is escaped", () => {
  const html = glossaryHTML([t({ term: "<script>x</script>", definition: "a & b <i>", alsoCalled: '"q"' })], tr);
  expect(html).not.toContain("<script>");
  expect(html).not.toContain("<i>");
  expect(html).toContain("&lt;script&gt;");
  expect(html).toContain("a &amp; b");
});

test("every entry is anchored by its key", () => {
  expect(termHTML(t({ key: "workcomplete" }), tr)).toContain('id="gl_workcomplete"');
});
