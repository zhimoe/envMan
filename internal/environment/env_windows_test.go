//go:build windows

package environment

import (
	"os"
	"testing"
	"uuid"

	"envman/internal/domain"

	"golang.org/x/sys/windows/registry"
)

func TestWindowsUserRegistryIntegration(t *testing.T) {
	if os.Getenv("ENVMAN_WINDOWS_INTEGRATION") != "1" {
		t.Skip("set ENVMAN_WINDOWS_INTEGRATION=1 to test the real user environment registry")
	}
	testWindowsRegistryScope(t, domain.ScopeUser)
}

func TestWindowsSystemRegistryIntegration(t *testing.T) {
	if os.Getenv("ENVMAN_WINDOWS_SYSTEM_INTEGRATION") != "1" {
		t.Skip("set ENVMAN_WINDOWS_SYSTEM_INTEGRATION=1 in an elevated shell to test the real system environment registry")
	}
	if !(registryBackend{}).Elevated() {
		t.Fatal("system registry integration test requires an elevated shell")
	}
	testWindowsRegistryScope(t, domain.ScopeSystem)
}

func testWindowsRegistryScope(t *testing.T, scope domain.Scope) {
	t.Helper()
	backend := registryBackend{}
	prefix := "ENVMAN_INTEGRATION_" + uuid.New().String()
	updatedName := prefix + "_UPDATED"
	deletedName := prefix + "_DELETED"
	addedName := prefix + "_ADDED"
	externalName := prefix + "_EXTERNAL"
	nonStringName := prefix + "_DWORD"
	stringNames := []string{updatedName, deletedName, addedName, externalName}

	t.Cleanup(func() {
		key, err := registryKey(scope, true)
		if err == nil {
			for _, name := range stringNames {
				_ = key.DeleteValue(name)
			}
			_ = key.DeleteValue(nonStringName)
			_ = key.Close()
		}
	})

	key, err := registryKey(scope, true)
	if err != nil {
		t.Fatalf("open environment key: %v", err)
	}
	if err := setOne(key, updatedName, `%SystemRoot%\EnvMan\before`, "REG_EXPAND_SZ"); err != nil {
		_ = key.Close()
		t.Fatalf("create expandable value: %v", err)
	}
	if err := setOne(key, deletedName, "delete-me", "REG_SZ"); err != nil {
		_ = key.Close()
		t.Fatalf("create value to delete: %v", err)
	}
	if err := key.SetDWordValue(nonStringName, 42); err != nil {
		_ = key.Close()
		t.Fatalf("create non-string value: %v", err)
	}
	if err := key.Close(); err != nil {
		t.Fatalf("close environment key: %v", err)
	}

	before, err := backend.Load(scope)
	if err != nil {
		t.Fatalf("load initial environment: %v", err)
	}
	updated, updatedIndex := domain.Find(before, updatedName)
	if updatedIndex < 0 || updated.Type != "REG_EXPAND_SZ" {
		t.Fatalf("expandable value was not loaded with its type: %+v", updated)
	}
	if _, index := domain.Find(before, nonStringName); index >= 0 {
		t.Fatal("non-string registry value was exposed as an environment variable")
	}

	desired := domain.Clone(before)
	desired[updatedIndex].Value = `%SystemRoot%\EnvMan\after`
	_, deletedIndex := domain.Find(desired, deletedName)
	if deletedIndex < 0 {
		t.Fatal("value prepared for deletion was not loaded")
	}
	desired = append(desired[:deletedIndex], desired[deletedIndex+1:]...)
	desired = append(desired, domain.Variable{Name: addedName, Value: "added-by-envman", Type: "REG_SZ"})

	// Simulate an unrelated program writing after EnvMan took its snapshot.
	key, err = registryKey(scope, true)
	if err != nil {
		t.Fatalf("reopen environment key for concurrent write: %v", err)
	}
	if err := setOne(key, externalName, "external-change", "REG_SZ"); err != nil {
		_ = key.Close()
		t.Fatalf("create concurrent external value: %v", err)
	}
	if err := key.Close(); err != nil {
		t.Fatalf("close environment key after concurrent write: %v", err)
	}
	if err := backend.ApplyChanges(scope, before, desired); err != nil {
		t.Fatalf("apply environment diff: %v", err)
	}

	after, err := backend.Load(scope)
	if err != nil {
		t.Fatalf("reload updated environment: %v", err)
	}
	updated, updatedIndex = domain.Find(after, updatedName)
	if updatedIndex < 0 || updated.Value != `%SystemRoot%\EnvMan\after` || updated.Type != "REG_EXPAND_SZ" {
		t.Fatalf("updated expandable value = %+v", updated)
	}
	if _, index := domain.Find(after, deletedName); index >= 0 {
		t.Fatal("requested registry deletion was not applied")
	}
	if added, index := domain.Find(after, addedName); index < 0 || added.Value != "added-by-envman" {
		t.Fatalf("requested registry addition = %+v", added)
	}
	if external, index := domain.Find(after, externalName); index < 0 || external.Value != "external-change" {
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
