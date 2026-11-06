//go:build windows

package main

import (
	"os"
	"testing"
	"uuid"

	"golang.org/x/sys/windows/registry"
)

func TestWindowsUserRegistryIntegration(t *testing.T) {
	if os.Getenv("ENVMAN_WINDOWS_INTEGRATION") != "1" {
		t.Skip("set ENVMAN_WINDOWS_INTEGRATION=1 to test the real user environment registry")
	}
	testWindowsRegistryScope(t, ScopeUser)
}

func TestWindowsSystemRegistryIntegration(t *testing.T) {
	if os.Getenv("ENVMAN_WINDOWS_SYSTEM_INTEGRATION") != "1" {
		t.Skip("set ENVMAN_WINDOWS_SYSTEM_INTEGRATION=1 in an elevated shell to test the real system environment registry")
	}
	if !elevated() {
		t.Fatal("system registry integration test requires an elevated shell")
	}
	testWindowsRegistryScope(t, ScopeSystem)
}

func testWindowsRegistryScope(t *testing.T, scope Scope) {
	t.Helper()
	prefix := "ENVMAN_INTEGRATION_" + uuid.New().String()
	updatedName := prefix + "_UPDATED"
	deletedName := prefix + "_DELETED"
	addedName := prefix + "_ADDED"
	externalName := prefix + "_EXTERNAL"
	nonStringName := prefix + "_DWORD"
	stringNames := []string{updatedName, deletedName, addedName, externalName}

	t.Cleanup(func() {
		for _, name := range stringNames {
			_ = deleteEnv(scope, name)
		}
		key, err := registryKey(scope, true)
		if err == nil {
			_ = key.DeleteValue(nonStringName)
			_ = key.Close()
		}
	})

	if err := writeEnv(scope, updatedName, `%SystemRoot%\EnvMan\before`, "REG_EXPAND_SZ"); err != nil {
		t.Fatalf("create expandable value: %v", err)
	}
	if err := writeEnv(scope, deletedName, "delete-me", "REG_SZ"); err != nil {
		t.Fatalf("create value to delete: %v", err)
	}
	key, err := registryKey(scope, true)
	if err != nil {
		t.Fatalf("open environment key: %v", err)
	}
	if err := key.SetDWordValue(nonStringName, 42); err != nil {
		_ = key.Close()
		t.Fatalf("create non-string value: %v", err)
	}
	if err := key.Close(); err != nil {
		t.Fatalf("close environment key: %v", err)
	}

	before, err := loadEnv(scope)
	if err != nil {
		t.Fatalf("load initial environment: %v", err)
	}
	updated, updatedIndex := findVar(before, updatedName)
	if updatedIndex < 0 || updated.Type != "REG_EXPAND_SZ" {
		t.Fatalf("expandable value was not loaded with its type: %+v", updated)
	}
	if _, index := findVar(before, nonStringName); index >= 0 {
		t.Fatal("non-string registry value was exposed as an environment variable")
	}

	desired := cloneVars(before)
	desired[updatedIndex].Value = `%SystemRoot%\EnvMan\after`
	_, deletedIndex := findVar(desired, deletedName)
	if deletedIndex < 0 {
		t.Fatal("value prepared for deletion was not loaded")
	}
	desired = append(desired[:deletedIndex], desired[deletedIndex+1:]...)
	desired = append(desired, Variable{Name: addedName, Value: "added-by-envman", Type: "REG_SZ"})

	// Simulate an unrelated program writing after EnvMan took its snapshot.
	if err := writeEnv(scope, externalName, "external-change", "REG_SZ"); err != nil {
		t.Fatalf("create concurrent external value: %v", err)
	}
	if err := applyEnvChanges(scope, before, desired); err != nil {
		t.Fatalf("apply environment diff: %v", err)
	}

	after, err := loadEnv(scope)
	if err != nil {
		t.Fatalf("reload updated environment: %v", err)
	}
	updated, updatedIndex = findVar(after, updatedName)
	if updatedIndex < 0 || updated.Value != `%SystemRoot%\EnvMan\after` || updated.Type != "REG_EXPAND_SZ" {
		t.Fatalf("updated expandable value = %+v", updated)
	}
	if _, index := findVar(after, deletedName); index >= 0 {
		t.Fatal("requested registry deletion was not applied")
	}
	if added, index := findVar(after, addedName); index < 0 || added.Value != "added-by-envman" {
		t.Fatalf("requested registry addition = %+v", added)
	}
	if external, index := findVar(after, externalName); index < 0 || external.Value != "external-change" {
		t.Fatalf("concurrent external registry value = %+v", external)
	}

	key, err = registryKey(scope, false)
	if err != nil {
		t.Fatalf("reopen environment key: %v", err)
	}
	number, valueType, err := key.GetIntegerValue(nonStringName)
	_ = key.Close()
	if err != nil || number != 42 || valueType != registry.DWORD {
		t.Fatalf("non-string registry value changed: value=%d type=%d err=%v", number, valueType, err)
	}
}
