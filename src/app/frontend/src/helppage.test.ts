import { expect, test } from "bun:test";
import { blockHTML, helpPageHTML, inline } from "./helppage";
import type { HelpBlock, HelpPage } from "./types";

const blk = (over: Partial<HelpBlock>): HelpBlock => ({ kind: "p", text: "", items: [], head: [], rows: [], ...over });

test("inline markup: bold, code and italic", () => {
  expect(inline("Click **Synchronize** now")).toBe("Click <b>Synchronize</b> now");
  expect(inline("type `WP1` here")).toBe("type <code>WP1</code> here");
  expect(inline("the *Skynet MVP* plan")).toBe("the <i>Skynet MVP</i> plan");
  expect(inline("**Bold** and `code` and *it*")).toBe("<b>Bold</b> and <code>code</code> and <i>it</i>");
});

test("code is shown literally: no bold or italic inside it", () => {
  expect(inline("`**not bold** *nor italic*`")).toBe("<code>**not bold** *nor italic*</code>");
  expect(inline("`a < b`")).toBe("<code>a &lt; b</code>");
});

test("all text is escaped, whatever the markup", () => {
  expect(inline("<script>alert(1)</script>")).toBe("&lt;script&gt;alert(1)&lt;/script&gt;");
  expect(inline("**<img src=x onerror=alert(1)>**")).toBe("<b>&lt;img src=x onerror=alert(1)&gt;</b>");
  expect(inline('a & "b"')).toBe("a &amp; &quot;b&quot;");
  expect(inline(undefined as unknown as string)).toBe("");
});

test("blocks: paragraph, note, lists, table", () => {
  expect(blockHTML(blk({ kind: "p", text: "Hello **you**" }))).toBe("<p>Hello <b>you</b></p>");
  expect(blockHTML(blk({ kind: "note", text: "Tip" }))).toBe('<p class="hp-note">Tip</p>');
  expect(blockHTML(blk({ kind: "ul", items: ["a", "`b`"] }))).toBe("<ul><li>a</li><li><code>b</code></li></ul>");
  expect(blockHTML(blk({ kind: "ol", items: ["one", "two"] }))).toBe("<ol><li>one</li><li>two</li></ol>");
  const t = blockHTML(blk({ kind: "table", head: ["Text", "Found?"], rows: [["`WP1 kickoff`", "yes"], ["`WP12`", "no"]] }));
  expect(t).toContain("<th>Text</th><th>Found?</th>");
  expect(t).toContain("<tr><td><code>WP1 kickoff</code></td><td>yes</td></tr>");
  expect(t).toContain("<tr><td><code>WP12</code></td><td>no</td></tr>");
  expect(blockHTML(blk({ kind: "unknown" }))).toBe("");
});

test("a whole page: intro, headed sections in order, anchored by key", () => {
  const page: HelpPage = {
    title: "Monitoring", intro: "Compare **plan** and bookings.",
    sections: [
      { key: "a", heading: "First", blocks: [blk({ kind: "p", text: "one" })] },
      { key: "b", heading: "Second `x`", blocks: [blk({ kind: "ol", items: ["s1", "s2"] }), blk({ kind: "note", text: "n" })] },
    ],
  };
  const html = helpPageHTML(page);
  expect(html).toContain('<p class="hp-intro">Compare <b>plan</b> and bookings.</p>');
  expect(html.indexOf("First")).toBeLessThan(html.indexOf("Second"));
  expect(html).toContain('<section id="hp_a"><h3>First</h3><p>one</p></section>');
  expect(html).toContain("<h3>Second <code>x</code></h3>");
  expect(html).toContain("<ol><li>s1</li><li>s2</li></ol>");
  expect(html.match(/<section /g)!.length).toBe(2);
});
