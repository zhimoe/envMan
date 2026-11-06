package main

import (
	"errors"
	"fmt"
	"strings"
)

var errNeedElevation = errors.New("需要管理员权限")

func valueType(typ, value string) string {
	if typ != "" {
		return typ
	}
	if strings.Contains(value, "%") {
		return "REG_EXPAND_SZ"
	}
	return "REG_SZ"
}

func findVar(vars []Variable, name string) (Variable, int) {
	for i, v := range vars {
		if strings.EqualFold(v.Name, name) {
			return v, i
		}
	}
	return Variable{}, -1
}

// envDiff returns only the writes and deletions needed to turn before into
// desired. Names are case-insensitive, matching Windows environment semantics.
func envDiff(before, desired []Variable) (upserts []Variable, deletions []string) {
	beforeByName := make(map[string]Variable, len(before))
	desiredByName := make(map[string]Variable, len(desired))
	for _, v := range before {
		beforeByName[strings.ToLower(v.Name)] = v
	}
	for _, v := range desired {
		desiredByName[strings.ToLower(v.Name)] = v
		previous, exists := beforeByName[strings.ToLower(v.Name)]
		if !exists || previous != v {
			upserts = append(upserts, v)
		}
	}
	for _, v := range before {
		if _, exists := desiredByName[strings.ToLower(v.Name)]; !exists {
			deletions = append(deletions, v.Name)
		}
	}
	return upserts, deletions
}

func describeError(scope Scope, err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, errNeedElevation) {
		return "系统变量需要管理员权限"
	}
	return fmt.Sprintf("无法读取%s：%v", scope, err)
}
