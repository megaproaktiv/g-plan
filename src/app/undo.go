package main

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// Undo support: destructive edits (deleting a link, an action, or a section)
// push a deep-copy snapshot of the whole project onto a stack. Undo (Cmd-Z in
// the UI) pops the most recent snapshot and restores it, so deletions are
// reversible in reverse order (R: undoable deletion of links/actions/sections).

const maxUndo = 100

// pushUndo captures a deep copy of the current project onto the undo stack. It
// is called BEFORE a destructive mutation so Undo can restore the prior state.
func (a *App) pushUndo() {
	if a.project == nil {
		return
	}
	cp := deepCopyProject(a.project)
	if cp == nil {
		return
	}
	a.undoStack = append(a.undoStack, cp)
	if len(a.undoStack) > maxUndo {
		a.undoStack = a.undoStack[len(a.undoStack)-maxUndo:]
	}
}

// CanUndo reports whether there is at least one snapshot to restore.
func (a *App) CanUndo() bool { return len(a.undoStack) > 0 }

// Undo restores the most recently captured snapshot (the last destructive
// edit). It is a no-op (returns the current project) when the stack is empty.
func (a *App) Undo() (*ProjectDTO, error) {
	n := len(a.undoStack)
	if n == 0 {
		return toDTO(a.project), nil
	}
	a.project = a.undoStack[n-1]
	a.undoStack = a.undoStack[:n-1]
	return toDTO(a.project), nil
}

// deepCopyProject makes an independent deep copy of a project via gob so a
// restored snapshot shares no slices/maps with the live project. gob is used
// (not JSON) because most graphic fields (Packages, Deps, Milestones, ...) are
// tagged json:"-" and would be dropped by a JSON round-trip.
func deepCopyProject(p *plan.Project) *plan.Project {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(p); err != nil {
		return nil
	}
	var cp plan.Project
	if err := gob.NewDecoder(&buf).Decode(&cp); err != nil {
		return nil
	}
	return &cp
}

// RemoveSection deletes a section/swimlane: its actions are moved to the
// default (unclustered) lane and its color entry is removed. The deletion is
// undoable (Cmd-Z). Returns an error if the section does not exist.
func (a *App) RemoveSection(name string) (*ProjectDTO, error) {
	found := false
	for i := range a.project.Packages {
		if a.project.Packages[i].Cluster == name {
			found = true
			break
		}
	}
	if _, ok := a.project.SectionColors[name]; ok {
		found = true
	}
	if !found {
		return nil, fmt.Errorf("section %q not found", name)
	}
	a.pushUndo()
	for i := range a.project.Packages {
		if a.project.Packages[i].Cluster == name {
			a.project.Packages[i].Cluster = ""
		}
	}
	if a.project.SectionColors != nil {
		delete(a.project.SectionColors, name)
	}
	out := a.project.SectionOrder[:0]
	for _, c := range a.project.SectionOrder {
		if c != name {
			out = append(out, c)
		}
	}
	a.project.SectionOrder = out
	if a.currentSection == name {
		a.currentSection = ""
	}
	return toDTO(a.project), nil
}
