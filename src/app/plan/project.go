package plan

import (
	"path/filepath"
	"strings"
)

// Paths holds the three sibling file paths for a project given a base path
// (the .mmd path chosen by the user).
type Paths struct {
	MMD   string // <base>.mmd  (graphic source)
	Roles string // <base>.roles.json (roles + config)
	MD    string // <base>.md   (generated projection, read-only)
}

// PathsFor derives the three-file set from any one of the project's paths.
// base may be the .mmd, .roles.json, or .md path (or a bare base name).
func PathsFor(base string) Paths {
	dir := filepath.Dir(base)
	name := filepath.Base(base)
	// strip known suffixes to get the bare project name
	for _, suf := range []string{".roles.json", ".mmd", ".md", ".json"} {
		if strings.HasSuffix(name, suf) {
			name = strings.TrimSuffix(name, suf)
			break
		}
	}
	return Paths{
		MMD:   filepath.Join(dir, name+".mmd"),
		Roles: filepath.Join(dir, name+".roles.json"),
		MD:    filepath.Join(dir, name+".md"),
	}
}

// Open loads a project from its three-file set, reading the .mmd graphic source
// and the .roles.json. The .md is not read (it is generated output).
func Open(base string) (*Project, Paths, error) {
	paths := PathsFor(base)
	p := NewProject("", UnitDays)
	if err := p.ReadMMD(paths.MMD); err != nil {
		return nil, paths, err
	}
	if err := p.LoadRoles(paths.Roles); err != nil {
		return nil, paths, err
	}
	return p, paths, nil
}

// Save writes all three files (requirement R-11.2): the .mmd graphic source,
// the .roles.json, and the regenerated .md projection.
func (p *Project) Save(base string) (Paths, error) {
	paths := PathsFor(base)
	if err := p.WriteMMD(paths.MMD); err != nil {
		return paths, err
	}
	if err := p.SaveRoles(paths.Roles); err != nil {
		return paths, err
	}
	if err := p.WriteMarkdown(paths.MD); err != nil {
		return paths, err
	}
	return paths, nil
}
