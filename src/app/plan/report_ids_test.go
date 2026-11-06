package plan

import (
	"strings"
	"testing"
)

// The report's "#" column shows the on-screen WBS number (e.g. "1.1"), not the
// long internal id (e.g. "ACC_CORE_CONFIGURATION), which overflowed the table.
func TestReportShowsWBSNumberNotInternalID(t *testing.T) {
	p := skynetLike()
	md := p.Markdown()
	for _, id := range []string{"PM", "AWS", "EMAIL"} {
		// the bare upper-cased internal id should not be a table cell "| PM |"
		if strings.Contains(md, "| "+id+" |") {
			t.Errorf("report still shows the internal id %q as the # column", id)
		}
	}
	// the WBS number of the first numbered section's action must appear
	wbs := p.WBS()
	// pick any package that has a display number and assert it is in the report
	var anyNum string
	for id, disp := range wbs.ActionDisplay {
		if disp != "" && !strings.HasPrefix(disp, "P") { // skip Phases Pn
			anyNum = disp
			_ = id
			break
		}
	}
	if anyNum == "" {
		t.Fatal("no numbered action in the fixture")
	}
	if !strings.Contains(md, "| "+anyNum+" |") {
		t.Errorf("report should show the WBS number %q as the # column", anyNum)
	}
}

// A report page's orientation follows the embedded image's shape: a tall
// (detail/drill) image gets a PORTRAIT page so it fills the page instead of
// being squeezed into a sliver on a landscape page; a wide (overview) image
// stays LANDSCAPE. A square image stays landscape (the report default).
func TestPageOrientationFollowsImageShape(t *testing.T) {
	cases := []struct {
		name string
		w, h float64
		want string
	}{
		{"tall detail view", 2035, 5316, "P"},
		{"wide overview", 2805, 2628, "L"},
		{"square", 1000, 1000, "L"},
		{"very wide", 4000, 500, "L"},
	}
	for _, c := range cases {
		if got := pageOrientationFor(c.w, c.h); got != c.want {
			t.Errorf("%s (%gx%g): orientation = %q, want %q", c.name, c.w, c.h, got, c.want)
		}
	}
}
