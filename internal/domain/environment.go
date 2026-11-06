package domain

import (
	"fmt"
	"slices"
	"strings"
)

// Scope identifies the Windows environment registry scope.
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

// Variable is one Windows environment variable.
type Variable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

// Key identifies a variable within a scope using Windows' case-insensitive
// name semantics.
func (v Variable) Key(scope Scope) string {
	return fmt.Sprintf("%d\x00%s", scope, strings.ToLower(v.Name))
}

func Clone(vars []Variable) []Variable {
	return slices.Clone(vars)
}

func Find(vars []Variable, name string) (Variable, int) {
	for i, variable := range vars {
		if strings.EqualFold(variable.Name, name) {
			return variable, i
		}
	}
	return Variable{}, -1
}

// Diff returns only the writes and deletions needed to turn before into
// desired. Variable names are case-insensitive on Windows.
func Diff(before, desired []Variable) (upserts []Variable, deletions []string) {
	beforeByName := make(map[string]Variable, len(before))
	desiredByName := make(map[string]Variable, len(desired))
	for _, variable := range before {
		beforeByName[strings.ToLower(variable.Name)] = variable
	}
	for _, variable := range desired {
		name := strings.ToLower(variable.Name)
		desiredByName[name] = variable
		previous, exists := beforeByName[name]
		if !exists || previous != variable {
			upserts = append(upserts, variable)
		}
	}
	for _, variable := range before {
		if _, exists := desiredByName[strings.ToLower(variable.Name)]; !exists {
			deletions = append(deletions, variable.Name)
		}
	}
	return upserts, deletions
}

// MergeChanges applies only the changes between before and desired to latest.
// This preserves unrelated updates made by other programs after the snapshot.
func MergeChanges(latest, before, desired []Variable) []Variable {
	result := Clone(latest)
	desiredByName := make(map[string]Variable, len(desired))
	beforeByName := make(map[string]Variable, len(before))
	for _, variable := range before {
		beforeByName[strings.ToLower(variable.Name)] = variable
	}
	for _, variable := range desired {
		desiredByName[strings.ToLower(variable.Name)] = variable
	}

	for name := range beforeByName {
		if _, keep := desiredByName[name]; keep {
			continue
		}
		for i := len(result) - 1; i >= 0; i-- {
			if strings.EqualFold(result[i].Name, name) {
				result = append(result[:i], result[i+1:]...)
			}
		}
	}

	for name, next := range desiredByName {
		previous, existed := beforeByName[name]
		if existed && previous == next {
			continue
		}
		_, index := Find(result, next.Name)
		if index < 0 {
			result = append(result, next)
			continue
		}
		result[index] = next
	}
	return result
}

func ValueType(typ, value string) string {
	if typ != "" {
		return typ
	}
	if strings.Contains(value, "%") {
		return "REG_EXPAND_SZ"
	}
	return "REG_SZ"
}
