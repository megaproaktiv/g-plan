package plan

import (
	"fmt"
	"strings"
)

// Role identity and references. A role is referenced by NAME from detail
// packages (DetailPackage.Role) and owns its assignments to actions
// (Role.Assignments). A role may only be deleted when nothing references it,
// and renaming must carry the references along.

// NextRoleID returns an unused role ID (highest existing + 1, at least 1).
func (p *Project) NextRoleID() int {
	max := 0
	for i := range p.Roles {
		if p.Roles[i].ID > max {
			max = p.Roles[i].ID
		}
	}
	return max + 1
}

// EnsureRoleIDs gives every role a unique positive ID. Roles that already have
// a unique ID keep it; roles without one (older files) or with a duplicate get
// the next free number, in list order, so a legacy file's first role becomes
// R1, the second R2, and so on.
func (p *Project) EnsureRoleIDs() {
	used := map[int]bool{}
	for i := range p.Roles {
		r := &p.Roles[i]
		if r.ID > 0 && !used[r.ID] {
			used[r.ID] = true
			continue
		}
		r.ID = p.NextRoleID()
		used[r.ID] = true
	}
}

// RoleByName returns the role with exactly this name, or nil.
func (p *Project) RoleByName(name string) *Role {
	for i := range p.Roles {
		if p.Roles[i].Name == name {
			return &p.Roles[i]
		}
	}
	return nil
}

// RoleByID returns the role with this ID, or nil.
func (p *Project) RoleByID(id int) *Role {
	for i := range p.Roles {
		if p.Roles[i].ID == id {
			return &p.Roles[i]
		}
	}
	return nil
}

// RoleUsage lists what still references a role.
type RoleUsage struct {
	Actions []string // names of actions the role is assigned to (each once)
	Details []string // names of detail packages that name the role
}

// InUse reports whether anything references the role.
func (u RoleUsage) InUse() bool { return len(u.Actions) > 0 || len(u.Details) > 0 }

// RoleUsage returns the references to the role with the given name. Any
// assignment counts, even one at 0%.
func (p *Project) RoleUsage(name string) RoleUsage {
	var u RoleUsage
	if r := p.RoleByName(name); r != nil {
		seen := map[string]bool{}
		for _, as := range r.Assignments {
			if seen[as.WorkPackageID] {
				continue
			}
			seen[as.WorkPackageID] = true
			if wp := p.WorkPackage(as.WorkPackageID); wp != nil {
				u.Actions = append(u.Actions, wp.Name)
			} else {
				u.Actions = append(u.Actions, as.WorkPackageID)
			}
		}
	}
	for i := range p.Packages {
		for j := range p.Packages[i].Details {
			if d := &p.Packages[i].Details[j]; d.Role == name {
				u.Details = append(u.Details, d.Name)
			}
		}
	}
	return u
}

// String describes the usage for an error message (at most 5 names per kind).
func (u RoleUsage) String() string {
	list := func(kind string, names []string) string {
		shown := names
		more := ""
		if len(shown) > 5 {
			more = fmt.Sprintf(" and %d more", len(shown)-5)
			shown = shown[:5]
		}
		return fmt.Sprintf("%s %s%s", kind, strings.Join(quoteAll(shown), ", "), more)
	}
	var parts []string
	if len(u.Actions) > 0 {
		parts = append(parts, list("assigned to action", u.Actions))
	}
	if len(u.Details) > 0 {
		parts = append(parts, list("used by detail", u.Details))
	}
	return strings.Join(parts, "; ")
}

func quoteAll(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = fmt.Sprintf("%q", s)
	}
	return out
}

// RenameRole renames a role and updates every reference to it (the detail
// packages that name the role). The new name must not be used by another role
// (case-insensitive). Assignments live on the role itself and move with it.
func (p *Project) RenameRole(oldName, newName string) error {
	newName = strings.TrimSpace(newName)
	r := p.RoleByName(oldName)
	if r == nil {
		return fmt.Errorf("role %q not found", oldName)
	}
	if newName == "" {
		return fmt.Errorf("a role needs a name")
	}
	for i := range p.Roles {
		if &p.Roles[i] != r && strings.EqualFold(p.Roles[i].Name, newName) {
			return fmt.Errorf("a role named %q already exists", p.Roles[i].Name)
		}
	}
	if newName == oldName {
		return nil
	}
	r.Name = newName
	for i := range p.Packages {
		for j := range p.Packages[i].Details {
			if p.Packages[i].Details[j].Role == oldName {
				p.Packages[i].Details[j].Role = newName
			}
		}
	}
	return nil
}

// DeleteRole removes a role. It refuses (and changes nothing) while the role is
// still referenced by an assignment or a detail package.
func (p *Project) DeleteRole(name string) error {
	if p.RoleByName(name) == nil {
		return fmt.Errorf("role %q not found", name)
	}
	if u := p.RoleUsage(name); u.InUse() {
		return fmt.Errorf("role %q cannot be deleted: still %s", name, u)
	}
	out := p.Roles[:0]
	for _, r := range p.Roles {
		if r.Name != name {
			out = append(out, r)
		}
	}
	p.Roles = out
	return nil
}
