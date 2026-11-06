// Command render loads a plan's three-file set and renders its Gantt to PNG
// via the embedded gmaid pipeline. Usage: render <base.mmd> <out.png> [scale]
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: render <base.mmd> <out.png> [scale]")
		os.Exit(2)
	}
	base, out := os.Args[1], os.Args[2]
	scale := 2.0
	if len(os.Args) > 3 {
		if v, err := strconv.ParseFloat(os.Args[3], 64); err == nil {
			scale = v
		}
	}
	p, _, err := plan.Open(base)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open:", err)
		os.Exit(1)
	}
	if err := p.RenderPNG(out, "", scale); err != nil {
		fmt.Fprintln(os.Stderr, "render:", err)
		os.Exit(1)
	}
	fi, _ := os.Stat(out)
	fmt.Printf("rendered %q -> %s (%d bytes, scale %.1f)\n", p.Title, out, func() int64 {
		if fi != nil {
			return fi.Size()
		}
		return 0
	}(), scale)
}
