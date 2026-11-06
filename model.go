package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Scope is where an environment variable lives.
type Scope int

const (
	ScopeUser Scope = iota
	ScopeSystem
)

func (s Scope) String() string {
	if s == ScopeSystem {
		return "系统变量"
	}
	return "用户变量"
}

// ViewFilter selects which variables the table shows.
type ViewFilter int

const (
	ViewAll ViewFilter = iota
	ViewPath
	ViewMulti
	ViewRefs
)

// Variable is one environment variable.
type Variable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

// Key identifies a variable within a scope.
func (v Variable) Key(scope Scope) string {
	return fmt.Sprintf("%d\x00%s", scope, strings.ToLower(v.Name))
}

func varKey(scope Scope, name string) string {
	return fmt.Sprintf("%d\x00%s", scope, strings.ToLower(name))
}

var refPattern = regexp.MustCompile(`%[^%\s]+%`)

func splitValue(value string) []string {
	if value == "" {
		return []string{""}
	}
	value = strings.ReplaceAll(value, "\r\n", "\n")
	var parts []string
	for p := range strings.FieldsFuncSeq(value, func(r rune) bool { return r == ';' || r == '\n' || r == '\r' }) {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return []string{""}
	}
	return parts
}

func isMultiValue(value string) bool {
	return len(splitValue(value)) > 1
}

func hasRef(value string) bool {
	return refPattern.MatchString(value)
}

func isListLike(name string) bool {
	switch strings.ToLower(name) {
	case "path", "psmodulepath", "pathext", "pythonpath", "classpath", "include", "lib", "cmakeprefixpath", "node_path":
		return true
	}
	return false
}

func isBatchEditable(v Variable) bool {
	return isMultiValue(v.Value) || isListLike(v.Name)
}

func trimTo(s string, n int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "…"
}

func (a *App) variables(scope Scope) []Variable {
	if a.vars == nil {
		return nil
	}
	return a.vars[scope]
}

func (a *App) findVariable(scope Scope, name string) (Variable, bool) {
	for _, v := range a.variables(scope) {
		if strings.EqualFold(v.Name, name) {
			return v, true
		}
	}
	return Variable{}, false
}

func (a *App) overlap(scope Scope, name string) int {
	// 0: none, 1: overridden by user (system view), 2: overrides system (user view)
	other := ScopeSystem
	if scope == ScopeSystem {
		other = ScopeUser
	}
	if _, ok := a.findVariable(other, name); !ok {
		return 0
	}
	if scope == ScopeUser {
		return 2
	}
	return 1
}

func (a *App) filtered(scope Scope) []Variable {
	vars := append([]Variable(nil), a.variables(scope)...)
	switch a.filter {
	case ViewPath:
		filtered := vars[:0]
		for _, v := range vars {
			if strings.EqualFold(v.Name, "Path") {
				filtered = append(filtered, v)
			}
		}
		vars = filtered
	case ViewMulti:
		filtered := vars[:0]
		for _, v := range vars {
			if isMultiValue(v.Value) {
				filtered = append(filtered, v)
			}
		}
		vars = filtered
	case ViewRefs:
		filtered := vars[:0]
		for _, v := range vars {
			if hasRef(v.Value) {
				filtered = append(filtered, v)
			}
		}
		vars = filtered
	}
	if q := strings.TrimSpace(a.query); q != "" {
		q = strings.ToLower(q)
		filtered := vars[:0]
		for _, v := range vars {
			if strings.Contains(strings.ToLower(v.Name), q) || strings.Contains(strings.ToLower(v.Value), q) {
				filtered = append(filtered, v)
			}
		}
		vars = filtered
	}
	sort.SliceStable(vars, func(i, j int) bool {
		ai, aj := strings.ToLower(vars[i].Name), strings.ToLower(vars[j].Name)
		if a.sortDesc {
			return ai > aj
		}
		return ai < aj
	})
	return vars
}

func (a *App) scopeCount(scope Scope) int {
	return len(a.variables(scope))
}

func (a *App) multiCount(scope Scope) int {
	n := 0
	for _, v := range a.variables(scope) {
		if isMultiValue(v.Value) {
			n++
		}
	}
	return n
}

func (a *App) refCount(scope Scope) int {
	n := 0
	for _, v := range a.variables(scope) {
		if hasRef(v.Value) {
			n++
		}
	}
	return n
}

func (a *App) batchTargets(scope Scope) []Variable {
	var out []Variable
	for _, v := range a.variables(scope) {
		if isBatchEditable(v) {
			out = append(out, v)
		}
	}
	return out
}

func (a *App) checkedCount() int {
	n := 0
	prefix := fmt.Sprintf("%d\x00", a.scope)
	for k, on := range a.checked {
		if on && strings.HasPrefix(k, prefix) {
			n++
		}
	}
	return n
}

func (a *App) isChecked(v Variable) bool {
	return a.checked[v.Key(a.scope)]
}

func (a *App) setChecked(v Variable, on bool) {
	if a.checked == nil {
		a.checked = make(map[string]bool)
	}
	if on {
		a.checked[v.Key(a.scope)] = true
	} else {
		delete(a.checked, v.Key(a.scope))
	}
}

func (a *App) clearChecked() {
	prefix := fmt.Sprintf("%d\x00", a.scope)
	for k := range a.checked {
		if strings.HasPrefix(k, prefix) {
			delete(a.checked, k)
		}
	}
}

func (a *App) expandKey(scope Scope, name string) string {
	return varKey(scope, name)
}

func (a *App) isExpanded(scope Scope, name string) bool {
	return a.expanded[a.expandKey(scope, name)]
}

func (a *App) toggleExpanded(scope Scope, name string) {
	if a.expanded == nil {
		a.expanded = make(map[string]bool)
	}
	k := a.expandKey(scope, name)
	if a.expanded[k] {
		delete(a.expanded, k)
	} else {
		a.expanded[k] = true
	}
}

func (a *App) expand(scope Scope, name string) {
	if a.expanded == nil {
		a.expanded = make(map[string]bool)
	}
	a.expanded[a.expandKey(scope, name)] = true
}

func (a *App) focusValue() int {
	if a.scope == ScopeSystem {
		return a.focusSystem
	}
	return a.focusUser
}

func (a *App) setFocus(v int) {
	if a.scope == ScopeSystem {
		a.focusSystem = v
	} else {
		a.focusUser = v
	}
}

func (a *App) focusPtr() *int {
	if a.scope == ScopeSystem {
		return &a.focusSystem
	}
	return &a.focusUser
}

// currentRows returns the rows for the active scope, clamped focus.
func (a *App) currentRows() []Variable {
	rows := a.filtered(a.scope)
	f := a.focusPtr()
	if *f >= len(rows) {
		*f = len(rows) - 1
	}
	if *f < -1 {
		*f = -1
	}
	return rows
}
