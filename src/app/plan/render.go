package plan

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/megaproaktiv/gmaid/gcss"
	"github.com/megaproaktiv/gmaid/glayout"
	"github.com/megaproaktiv/gmaid/gparser"
	"github.com/megaproaktiv/gmaid/grender"
)

// RenderPNG renders the project's current graphic content to a high-resolution
// PNG at outPath, using the EMBEDDED gmaid pipeline (parse -> layout -> render)
// in-process — no external binary (chosen so a combined PDF report can embed
// the image without a process round-trip). css may be empty.
//
// scale is the output resolution multiplier (gmaid default is 2).
func (p *Project) RenderPNG(outPath, css string, scale float64) error {
	chart, err := gparser.Parse(p.MMD())
	if err != nil {
		return fmt.Errorf("gmaid parse: %w", err)
	}
	eng := glayout.NewEngine(glayout.DefaultConfig())
	diagram := eng.Layout(chart)

	r := grender.New(grender.DefaultTheme(), glayout.DefaultConfig(), scale)
	if css != "" {
		r.SetStyle(gcss.Parse(css))
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	if err := r.Render(diagram, outPath); err != nil {
		return fmt.Errorf("gmaid render: %w", err)
	}
	return nil
}

// RenderPNGBytes renders to a temp file and returns the PNG bytes. Convenient
// for the UI preview and for embedding into the PDF report.
func (p *Project) RenderPNGBytes(css string, scale float64) ([]byte, error) {
	tmp, err := os.CreateTemp("", "ptplan-*.png")
	if err != nil {
		return nil, err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	if err := p.RenderPNG(tmp.Name(), css, scale); err != nil {
		return nil, err
	}
	return os.ReadFile(tmp.Name())
}
