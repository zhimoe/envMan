package storage

import (
	"fmt"
	"testing"

	"envman/internal/domain"
)

func TestSnapshotRetentionAndPendingRoundTrip(t *testing.T) {
	store := New(t.TempDir())
	variables := []domain.Variable{{Name: "Path", Value: `C:\Windows`, Type: "REG_SZ"}}
	for i := range MaxSnapshots + 5 {
		if _, err := store.SaveSnapshot(domain.ScopeUser, fmt.Sprintf("snapshot-%02d", i), variables); err != nil {
			t.Fatal(err)
		}
	}
	snapshots := store.LoadSnapshots()
	if len(snapshots) != MaxSnapshots {
		t.Fatalf("snapshot count = %d, want %d", len(snapshots), MaxSnapshots)
	}
	if snapshots[0].Note != "snapshot-24" || snapshots[len(snapshots)-1].Note != "snapshot-05" {
		t.Fatalf("retained snapshots = %q ... %q", snapshots[0].Note, snapshots[len(snapshots)-1].Note)
	}

	pending := PendingEdits{Scope: domain.ScopeSystem, Note: "pending", Variables: variables}
	if err := store.WritePending(pending); err != nil {
		t.Fatal(err)
	}
	loaded := store.ReadPending()
	if loaded == nil || loaded.Note != pending.Note || len(loaded.Variables) != 1 {
		t.Fatalf("pending round trip = %+v", loaded)
	}
	store.ClearPending()
	if loaded := store.ReadPending(); loaded != nil {
		t.Fatalf("pending file survived clear: %+v", loaded)
	}
}
