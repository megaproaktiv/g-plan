-- circled.lua — pandoc Lua filter for the walkthrough manual (LaTeX/PDF only).
--
-- The walkthroughs use a few symbol glyphs that the default LaTeX fonts do not
-- contain, so in the PDF they came out blank (pandoc "Missing character"
-- warnings). This filter rewrites each one to a font-independent LaTeX command
-- so it renders identically in every engine (the Docker image and host xelatex
-- alike). It only touches the specific code points below; all other text is
-- left exactly as written. For non-LaTeX output it is a no-op, so the markdown
-- still shows the real glyphs on GitHub.
--
-- Primary fix: the circled callout markers ① … ⑨ (U+2460..U+2468) that point at
-- numbered spots on the UI screenshots -> \ding{192}..\ding{200} (pifont).
-- Also mapped: ↔ ▲ ↳ ☺ used inline in the prose (amssymb / pifont).
-- Requires the LaTeX packages loaded by header.tex (pifont, amssymb).

-- code point (as a UTF-8 string) -> the raw LaTeX that renders it
local sym = {
  ["\u{2460}"] = "\\ding{192}", ["\u{2461}"] = "\\ding{193}", ["\u{2462}"] = "\\ding{194}",
  ["\u{2463}"] = "\\ding{195}", ["\u{2464}"] = "\\ding{196}", ["\u{2465}"] = "\\ding{197}",
  ["\u{2466}"] = "\\ding{198}", ["\u{2467}"] = "\\ding{199}", ["\u{2468}"] = "\\ding{200}",
  ["\u{2194}"] = "\\ensuremath{\\leftrightarrow}", -- ↔
  ["\u{25B2}"] = "\\ding{115}",                    -- ▲ (pifont filled up-triangle)
  ["\u{21B3}"] = "\\ensuremath{\\hookrightarrow}", -- ↳
  ["\u{263A}"] = "\\ensuremath{\\odot}",           -- ☺ (round "face" icon marker)
}

local function latex_output()
  return FORMAT:match("latex") or FORMAT:match("beamer")
end

-- Str: split a run of text around any mapped glyph, keeping order, so a marker
-- embedded in a word still renders.
function Str(el)
  if not latex_output() then return nil end
  local found = false
  for u in pairs(sym) do
    if el.text:find(u, 1, true) then found = true; break end
  end
  if not found then return nil end

  local inlines, buf = {}, {}
  for _, cp in utf8.codes(el.text) do
    local ch = utf8.char(cp)
    local tex = sym[ch]
    if tex then
      if #buf > 0 then
        table.insert(inlines, pandoc.Str(table.concat(buf))); buf = {}
      end
      table.insert(inlines, pandoc.RawInline("latex", tex))
    else
      table.insert(buf, ch)
    end
  end
  if #buf > 0 then table.insert(inlines, pandoc.Str(table.concat(buf))) end
  return inlines
end

-- Code: a mapped glyph inside an inline code span (e.g. `↳`) is verbatim, so
-- render the whole span as \texttt{...} with the glyph replaced by its command.
function Code(el)
  if not latex_output() then return nil end
  local found = false
  for u in pairs(sym) do
    if el.text:find(u, 1, true) then found = true; break end
  end
  if not found then return nil end

  local parts = {}
  for _, cp in utf8.codes(el.text) do
    local ch = utf8.char(cp)
    parts[#parts + 1] = sym[ch] or ch
  end
  return pandoc.RawInline("latex", "\\texttt{" .. table.concat(parts) .. "}")
end
