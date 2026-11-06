package environment

import (
	"testing"

	"envman/internal/domain"
)

func TestMemoryApplyChangesPreservesConcurrentUpdates(t *testing.T) {
	backend := NewMemory(DemoVariables())
	before, err := backend.Load(domain.ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
	desired := domain.Clone(before)
	_, javaIndex := domain.Find(desired, "JAVA_HOME")
	desired[javaIndex].Value = `C:\AppChange`

	current := domain.Clone(before)
	_, cargoIndex := domain.Find(current, "CARGO_HOME")
	current[cargoIndex].Value = `C:\ChangedElsewhere`
	current = append(current, domain.Variable{Name: "EXTERNAL", Value: "1", Type: "REG_SZ"})
	backend.Replace(domain.ScopeUser, current)

	if err := backend.ApplyChanges(domain.ScopeUser, before, desired); err != nil {
		t.Fatal(err)
	}
	after, err := backend.Load(domain.ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
	java, _ := domain.Find(after, "JAVA_HOME")
	if java.Value != `C:\AppChange` {
		t.Fatalf("requested update = %+v", java)
	}
	cargo, _ := domain.Find(after, "CARGO_HOME")
	if cargo.Value != `C:\ChangedElsewhere` {
		t.Fatalf("concurrent update = %+v", cargo)
	}
	if _, index := domain.Find(after, "EXTERNAL"); index < 0 {
		t.Fatal("concurrent addition was removed")
	}
}
