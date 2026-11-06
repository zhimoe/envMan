package domain

import "testing"

func TestMergeChangesPreservesUnrelatedUpdates(t *testing.T) {
	before := []Variable{
		{Name: "Path", Value: `C:\Old`, Type: "REG_EXPAND_SZ"},
		{Name: "KEEP", Value: "old", Type: "REG_SZ"},
	}
	desired := []Variable{
		{Name: "Path", Value: `C:\New`, Type: "REG_EXPAND_SZ"},
		{Name: "KEEP", Value: "old", Type: "REG_SZ"},
	}
	latest := []Variable{
		{Name: "Path", Value: `C:\Old`, Type: "REG_EXPAND_SZ"},
		{Name: "KEEP", Value: "changed elsewhere", Type: "REG_SZ"},
		{Name: "ADDED_ELSEWHERE", Value: "1", Type: "REG_SZ"},
	}

	merged := MergeChanges(latest, before, desired)
	path, _ := Find(merged, "path")
	if path.Value != `C:\New` {
		t.Fatalf("Path = %q", path.Value)
	}
	keep, _ := Find(merged, "keep")
	if keep.Value != "changed elsewhere" {
		t.Fatalf("unrelated update was overwritten: %+v", keep)
	}
	if _, index := Find(merged, "ADDED_ELSEWHERE"); index < 0 {
		t.Fatal("concurrently added variable was removed")
	}
}

func TestDiffAndValueType(t *testing.T) {
	before := []Variable{{Name: "Path", Value: "old", Type: "REG_SZ"}, {Name: "DROP", Value: "1", Type: "REG_SZ"}}
	desired := []Variable{{Name: "PATH", Value: `%SystemRoot%`, Type: ""}, {Name: "ADD", Value: "2", Type: "REG_SZ"}}
	upserts, deletions := Diff(before, desired)
	if len(upserts) != 2 || len(deletions) != 1 || deletions[0] != "DROP" {
		t.Fatalf("Diff() = upserts %#v, deletions %#v", upserts, deletions)
	}
	if got := ValueType("", `%SystemRoot%`); got != "REG_EXPAND_SZ" {
		t.Fatalf("ValueType() = %q", got)
	}
}
