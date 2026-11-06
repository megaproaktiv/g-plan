package plan

import (
	"encoding/json"
	"fmt"
	"os"
)

// rolesFile is the on-disk shape of <project>.roles.json. It holds the roles
// and the project-wide configuration that does not belong in the .mmd graphic
// source (requirement R-3.2, R-6.*).
type rolesFile struct {
	Schema         string            `json:"schema"`
	Project        projectMeta       `json:"project"`
	Roles          []Role            `json:"roles"`
	Details        []detailMeta      `json:"details,omitempty"`
	ActionColors   map[string]string `json:"actionColors,omitempty"`
	ActionFontCols map[string]string `json:"actionFontColors,omitempty"`
	ActionTextPos  map[string]string `json:"actionTextPos,omitempty"`
	ActionDescs    map[string]string `json:"actionDescs,omitempty"`
	ActionSameLine []string          `json:"actionSameLine,omitempty"`
	SectionColors  map[string]string `json:"sectionColors,omitempty"`
	LinkColors     []linkColor       `json:"linkColors,omitempty"`
	Baselines      []Baseline        `json:"baselines,omitempty"`
}

// linkColor stores a dependency's color + anchors (non-graphic metadata that
// the .mmd link directive cannot hold).
type linkColor struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Color      string `json:"color,omitempty"`
	FromAnchor string `json:"fromAnchor,omitempty"`
	ToAnchor   string `json:"toAnchor,omitempty"`
}

// detailMeta carries the non-graphic fields of a level-2 detail package
// (description + role) that cannot be stored in the .mmd Gantt source. Keyed
// by the stable detail id.
type detailMeta struct {
	ID        string `json:"id"`
	Desc      string `json:"desc,omitempty"`
	Role      string `json:"role,omitempty"`
	FontColor string `json:"fontColor,omitempty"`
	TextPos   string `json:"textPos,omitempty"`
}

type projectMeta struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	Unit            Unit   `json:"unit"`
	HoursPerDay     int    `json:"hoursPerDay"`
	ArrowStyle      string `json:"arrowStyle,omitempty"`
	ShowDetails     bool   `json:"showDetails,omitempty"`
	ShowRoleActions bool   `json:"showRoleActions,omitempty"`
	// ShowPhases is a pointer so a saved false is distinguishable from unset
	// (unset => default true).
	ShowPhases *bool `json:"showPhases,omitempty"`
}

const rolesSchema = "pt-plan/roles@1"

// LoadRoles reads a <project>.roles.json into the project (unit, title, roles).
// A missing file is not an error: the project simply has no roles yet.
func (p *Project) LoadRoles(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var rf rolesFile
	if err := json.Unmarshal(b, &rf); err != nil {
		return fmt.Errorf("parse roles json: %w", err)
	}
	if rf.Project.Unit != "" {
		p.Unit = rf.Project.Unit
	}
	if rf.Project.HoursPerDay > 0 {
		p.HoursPerDay = rf.Project.HoursPerDay
	}
	if rf.Project.Title != "" && p.Title == "" {
		p.Title = rf.Project.Title
	}
	if rf.Project.Description != "" {
		p.Description = rf.Project.Description
	}
	if rf.Project.ArrowStyle != "" {
		p.ArrowStyle = rf.Project.ArrowStyle
	}
	p.ShowDetails = rf.Project.ShowDetails
	p.ShowRoleActions = rf.Project.ShowRoleActions
	if rf.Project.ShowPhases != nil {
		p.ShowPhases = *rf.Project.ShowPhases
	} else {
		p.ShowPhases = true // unset in older files => default on
	}
	p.Baselines = rf.Baselines
	p.Roles = rf.Roles
	p.EnsureRoleIDs()
	// apply detail metadata (desc/role) onto the already-parsed packages
	if len(rf.Details) > 0 {
		meta := make(map[string]detailMeta, len(rf.Details))
		for _, m := range rf.Details {
			meta[m.ID] = m
		}
		for i := range p.Packages {
			for j := range p.Packages[i].Details {
				if m, ok := meta[p.Packages[i].Details[j].ID]; ok {
					p.Packages[i].Details[j].Desc = m.Desc
					p.Packages[i].Details[j].Role = m.Role
					p.Packages[i].Details[j].FontColor = m.FontColor
					p.Packages[i].Details[j].TextPos = m.TextPos
				}
			}
		}
	}
	// apply action colors/descriptions
	for i := range p.Packages {
		wp := &p.Packages[i]
		if c, ok := rf.ActionColors[wp.ID]; ok {
			wp.Color = c
		}
		if c, ok := rf.ActionFontCols[wp.ID]; ok {
			wp.FontColor = c
		}
		if tp, ok := rf.ActionTextPos[wp.ID]; ok {
			wp.TextPos = tp
		}
		if d, ok := rf.ActionDescs[wp.ID]; ok {
			wp.Desc = d
		}
	}
	// apply same-line flags
	if len(rf.ActionSameLine) > 0 {
		sl := make(map[string]bool, len(rf.ActionSameLine))
		for _, id := range rf.ActionSameLine {
			sl[id] = true
		}
		for i := range p.Packages {
			if sl[p.Packages[i].ID] {
				p.Packages[i].SameLine = true
			}
		}
	}
	// section colors
	if len(rf.SectionColors) > 0 {
		p.SectionColors = map[string]string{}
		for k, v := range rf.SectionColors {
			p.SectionColors[k] = v
		}
	}
	// link colors/anchors
	for _, lc := range rf.LinkColors {
		for i := range p.Deps {
			if p.Deps[i].From == lc.From && p.Deps[i].To == lc.To {
				p.Deps[i].Color = lc.Color
				p.Deps[i].FromAnchor = lc.FromAnchor
				p.Deps[i].ToAnchor = lc.ToAnchor
			}
		}
	}
	return nil
}

// SaveRoles writes the roles + project meta as <project>.roles.json.
func (p *Project) SaveRoles(path string) error {
	rf := rolesFile{
		Schema:    rolesSchema,
		Project:   projectMeta{Title: p.Title, Description: p.Description, Unit: p.Unit, HoursPerDay: p.hpd(), ArrowStyle: p.ArrowStyle, ShowDetails: p.ShowDetails, ShowRoleActions: p.ShowRoleActions, ShowPhases: &p.ShowPhases},
		Baselines: p.Baselines,
		Roles:     p.Roles,
	}
	for i := range p.Packages {
		for j := range p.Packages[i].Details {
			d := &p.Packages[i].Details[j]
			if d.Desc != "" || d.Role != "" || d.FontColor != "" || d.TextPos != "" {
				rf.Details = append(rf.Details, detailMeta{ID: d.ID, Desc: d.Desc, Role: d.Role, FontColor: d.FontColor, TextPos: d.TextPos})
			}
		}
	}
	for i := range p.Packages {
		wp := &p.Packages[i]
		if wp.Color != "" {
			if rf.ActionColors == nil {
				rf.ActionColors = map[string]string{}
			}
			rf.ActionColors[wp.ID] = wp.Color
		}
		if wp.FontColor != "" {
			if rf.ActionFontCols == nil {
				rf.ActionFontCols = map[string]string{}
			}
			rf.ActionFontCols[wp.ID] = wp.FontColor
		}
		if wp.TextPos != "" {
			if rf.ActionTextPos == nil {
				rf.ActionTextPos = map[string]string{}
			}
			rf.ActionTextPos[wp.ID] = wp.TextPos
		}
		if wp.Desc != "" {
			if rf.ActionDescs == nil {
				rf.ActionDescs = map[string]string{}
			}
			rf.ActionDescs[wp.ID] = wp.Desc
		}
		if wp.SameLine {
			rf.ActionSameLine = append(rf.ActionSameLine, wp.ID)
		}
	}
	if len(p.SectionColors) > 0 {
		rf.SectionColors = map[string]string{}
		for k, v := range p.SectionColors {
			rf.SectionColors[k] = v
		}
	}
	for _, dep := range p.Deps {
		if dep.Color != "" || dep.FromAnchor != "" || dep.ToAnchor != "" {
			rf.LinkColors = append(rf.LinkColors, linkColor{
				From: dep.From, To: dep.To, Color: dep.Color,
				FromAnchor: dep.FromAnchor, ToAnchor: dep.ToAnchor,
			})
		}
	}
	b, err := json.MarshalIndent(rf, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return WriteFileAtomic(path, b, 0o644)
}
