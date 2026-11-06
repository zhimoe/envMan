package app

import (
	"envman/internal/domain"
	"envman/internal/storage"
)

type Scope = domain.Scope

const (
	ScopeUser   = domain.ScopeUser
	ScopeSystem = domain.ScopeSystem
)

type Variable = domain.Variable
type Snapshot = storage.Snapshot
type PendingEdits = storage.PendingEdits

func cloneVars(vars []Variable) []Variable {
	return domain.Clone(vars)
}

func findVar(vars []Variable, name string) (Variable, int) {
	return domain.Find(vars, name)
}

func valueType(typ, value string) string {
	return domain.ValueType(typ, value)
}

func mergeVariableChanges(latest, before, desired []Variable) []Variable {
	return domain.MergeChanges(latest, before, desired)
}
