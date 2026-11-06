package plan

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
)

// a4Portrait is the A4 page size in PORTRAIT extents (mm). fpdf's AddPageFormat
// swaps these for the orientation string: "P" -> 210x297, "L" -> 297x210.
var a4Portrait = fpdf.SizeType{Wd: 210, Ht: 297}

// ReportImage is one captioned Gantt image embedded in the report (e.g. a work
// package's detail/drill view). PNG is the raw PNG bytes.
type ReportImage struct {
	Caption string
	PNG     []byte
}

// WritePDFReport builds a single combined PDF report (requirement: one document
// with the rendered Gantt image PLUS reports about the people/roles) and writes
// it to outPath. The Gantt is rendered via the embedded gmaid pipeline, so no
// external binary and no separate files are involved.
func (p *Project) WritePDFReport(outPath, css string) error {
	png, err := p.RenderPNGBytes(css, 2)
	if err != nil {
		return fmt.Errorf("render gantt for report: %w", err)
	}
	return p.writePDFReport(outPath, png, nil)
}

// WritePDFReportWithImage builds the same combined PDF report but embeds a
// CALLER-PROVIDED Gantt PNG (e.g. the live-canvas SVG rasterized by the
// frontend) so the report graphic matches the on-screen view exactly.
func (p *Project) WritePDFReportWithImage(outPath string, png []byte) error {
	return p.writePDFReport(outPath, png, nil)
}

// WritePDFReportWithViews builds the combined PDF report with the whole-project
// overview image on its own page, then one page per supplied detail view (each
// a work package's detail/drill image), then the work-packages table and the
// people/roles pages. details may be empty (→ overview only).
func (p *Project) WritePDFReportWithViews(outPath string, overview []byte, details []ReportImage) error {
	return p.writePDFReport(outPath, overview, details)
}

// writePDFReport is the shared report builder: page 1 = title + the overview
// Gantt PNG; then one page per detail view; then a work-packages table page and
// the people/roles pages.
func (p *Project) writePDFReport(outPath string, overview []byte, details []ReportImage) error {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetTitle(p.Title, true)
	pdf.SetAutoPageBreak(true, 12)

	// --- Gantt image page(s): whole-project overview, then each detail view ---
	p.ganttPage(pdf, overview, "Overview", 0)
	for i, d := range details {
		caption := d.Caption
		if caption == "" {
			caption = "Detail view"
		}
		p.ganttPage(pdf, d.PNG, "Details - "+caption, i+1)
	}

	// --- Work packages table (always landscape, independent of the last
	// Gantt page's orientation) ---
	pdf.AddPageFormat("L", a4Portrait)
	unit := p.unitWord()
	sectionTitle(pdf, "Work packages")
	wpHeader := []string{"#", "Name", "Start", "End", "Est (" + unit + ")", "Mode"}
	wpWidths := []float64{18, 90, 28, 28, 30, 24}
	tableHeader(pdf, wpHeader, wpWidths)
	pdf.SetFont("Helvetica", "", 9)
	wbs := p.WBS()
	for i := range p.Packages {
		wp := &p.Packages[i]
		row := []string{
			wbs.ActionDisplay[wp.ID], wp.Name,
			wp.Start.Format(DateLayout), wp.End.Format(DateLayout),
			p.Format(wp.EffectiveEst()), string(wp.Mode),
		}
		tableRow(pdf, row, wpWidths, []string{"L", "L", "L", "L", "R", "L"})
	}
	pdf.SetFont("Helvetica", "B", 9)
	tableRow(pdf, []string{"", "Total", "", "", p.Format(p.TotalEst()), ""}, wpWidths,
		[]string{"L", "L", "L", "L", "R", "L"})

	// --- Page 3+: people / roles reports (landscape) ---
	pdf.AddPageFormat("L", a4Portrait)
	sectionTitle(pdf, "People / roles")
	for i := range p.Roles {
		r := &p.Roles[i]
		pdf.SetFont("Helvetica", "B", 12)
		pdf.CellFormat(0, 7, tr(fmt.Sprintf("%s  -  %s %s total", r.Name, p.Format(p.RoleEst(r)), unit)), "", 1, "L", false, 0, "")
		if r.Description != "" {
			pdf.SetFont("Helvetica", "I", 9)
			pdf.MultiCell(0, 5, tr(r.Description), "", "L", false)
		}
		if len(r.Assignments) > 0 {
			hdr := []string{"Work package", "%", "Share (" + unit + ")", "Consumed"}
			w := []float64{120, 20, 35, 35}
			tableHeader(pdf, hdr, w)
			pdf.SetFont("Helvetica", "", 9)
			for _, a := range r.Assignments {
				wp := p.WorkPackage(a.WorkPackageID)
				name := a.WorkPackageID
				var share Base
				if wp != nil {
					num := wbs.ActionDisplay[wp.ID]
					name = strings.TrimSpace(num + " " + wp.Name)
					share = Base(int64(wp.EffectiveEst()) * int64(a.Percent) / 100)
				}
				tableRow(pdf, []string{name, fmt.Sprintf("%d", a.Percent), p.Format(share), p.AssignmentConsumed(a)},
					w, []string{"L", "R", "R", "R"})
			}
		}
		pdf.Ln(4)
	}

	return pdf.OutputFileAndClose(outPath)
}

func sectionTitle(pdf *fpdf.Fpdf, s string) {
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 9, tr(s), "", 1, "L", false, 0, "")
	pdf.Ln(1)
}

// pageOrientationFor picks the PDF page orientation that best fits an image of
// the given pixel dimensions: "P" (portrait) when the image is taller than it
// is wide, "L" (landscape) otherwise. A square image stays landscape (the
// report default). This keeps a tall-and-narrow detail (drill) view from being
// squeezed into a sliver on a landscape page.
func pageOrientationFor(w, h float64) string {
	if h > w {
		return "P"
	}
	return "L"
}

// ganttPage adds one page with the project title, a caption, and the given
// Gantt PNG fit to the page width (height capped to the remaining space). The
// page ORIENTATION is chosen from the image's aspect ratio: a portrait (taller
// than wide) image gets a portrait page and a landscape image gets a landscape
// page, so a tall-and-narrow detail (drill) view fills the page instead of
// being squeezed into a sliver on a landscape page. idx disambiguates the
// registered fpdf image name so multiple images in one document (overview +
// per-WP detail views, which may share a name) do not collide.
func (p *Project) ganttPage(pdf *fpdf.Fpdf, png []byte, caption string, idx int) {
	// Register the image first (no page needed) so we can read its dimensions
	// and pick the page orientation before the page is created.
	var info *fpdf.ImageInfoType
	if len(png) > 0 {
		opt := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}
		name := fmt.Sprintf("gantt-%d", idx)
		info = pdf.RegisterImageOptionsReader(name, opt, bytes.NewReader(png))
		orient := "L"
		if info != nil && pageOrientationFor(info.Width(), info.Height()) == "P" {
			orient = "P" // taller than wide -> portrait page
		}
		pdf.AddPageFormat(orient, a4Portrait)
		p.ganttPageHeader(pdf, caption)
		p.placeGanttImage(pdf, name, info)
		return
	}
	// No image: fall back to a landscape page with just the header.
	pdf.AddPageFormat("L", a4Portrait)
	p.ganttPageHeader(pdf, caption)
}

// ganttPageHeader draws the title + one-line caption at the top of a Gantt page.
func (p *Project) ganttPageHeader(pdf *fpdf.Fpdf, caption string) {
	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(0, 12, tr(p.Title), "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	unit := p.unitWord()
	pdf.CellFormat(0, 6, tr(fmt.Sprintf("%s   -   Total estimation: %s %s   -   Work packages: %d   -   Roles: %d",
		caption, p.Format(p.TotalEst()), unit, len(p.Packages), len(p.Roles))), "", 1, "C", false, 0, "")
	pdf.Ln(2)
}

// placeGanttImage fits the registered image to the page width, capping the
// height to the space remaining below the header.
func (p *Project) placeGanttImage(pdf *fpdf.Fpdf, name string, info *fpdf.ImageInfoType) {
	if info == nil {
		return
	}
	opt := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}
	pw, ph := pdf.GetPageSize()
	lm, _, rm, _ := pdf.GetMargins()
	avail := pw - lm - rm
	imgW := info.Width()
	imgH := info.Height()
	scale := avail / imgW
	// cap height to remaining page space
	maxH := ph - pdf.GetY() - 14
	if imgH*scale > maxH {
		scale = maxH / imgH
	}
	drawW := imgW * scale
	// center horizontally so a portrait image that is width-capped by height
	// does not hug the left margin
	xLeft := lm + (avail-drawW)/2
	if xLeft < lm {
		xLeft = lm
	}
	pdf.ImageOptions(name, xLeft, pdf.GetY(), drawW, imgH*scale, true, opt, 0, "")
}

func tableHeader(pdf *fpdf.Fpdf, cells []string, widths []float64) {
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(230, 230, 240)
	for i, c := range cells {
		pdf.CellFormat(widths[i], 7, tr(c), "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)
}

func tableRow(pdf *fpdf.Fpdf, cells []string, widths []float64, align []string) {
	for i, c := range cells {
		a := "L"
		if i < len(align) {
			a = align[i]
		}
		pdf.CellFormat(widths[i], 6, tr(c), "1", 0, a, false, 0, "")
	}
	pdf.Ln(-1)
}

// tr transliterates text to the Windows-1252 subset fpdf's core fonts use,
// mapping the few special glyphs that appear in plans (macron p, dashes).
func tr(s string) string {
	s = strings.ReplaceAll(s, "p\u0304", "p") // p + combining macron -> p
	s = strings.ReplaceAll(s, "\u2013", "-")  // en dash
	s = strings.ReplaceAll(s, "\u2014", "-")  // em dash
	return s
}
