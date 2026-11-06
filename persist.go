package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const maxSnapshots = 20

// Snapshot is a saved state of one scope, taken before a change.
type Snapshot struct {
	ID        string     `json:"id"`
	Time      time.Time  `json:"time"`
	Scope     Scope      `json:"scope"`
	Note      string     `json:"note"`
	Variables []Variable `json:"variables"`
}

// PendingEdits is a system-scope write waiting for an elevated restart.
type PendingEdits struct {
	Scope     Scope      `json:"scope"`
	Note      string     `json:"note"`
	Focus     string     `json:"focus,omitempty"`
	Before    []Variable `json:"before,omitempty"`
	Variables []Variable `json:"variables"`
}

var dataDirOverride string

func userDataDir() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		dir = os.TempDir()
	}
	dir = filepath.Join(dir, "EnvMan")
	if dataDirOverride != "" {
		dir = dataDirOverride
	}
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

func snapshotsPath() string { return filepath.Join(userDataDir(), "snapshots.json") }
func pendingPath() string   { return filepath.Join(userDataDir(), "pending.json") }
func settingsPath() string  { return filepath.Join(userDataDir(), "settings.json") }

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func loadSnapshots() []Snapshot {
	var out []Snapshot
	if err := readJSON(snapshotsPath(), &out); err != nil {
		return nil
	}
	return out
}

func saveSnapshots(list []Snapshot) error {
	if len(list) > maxSnapshots {
		list = list[:maxSnapshots]
	}
	return writeJSON(snapshotsPath(), list)
}

func takeSnapshot(scope Scope, note string) (Snapshot, error) {
	vars, err := loadEnv(scope)
	if err != nil {
		return Snapshot{}, err
	}
	now := time.Now()
	snap := Snapshot{
		ID:        now.Format("20060102-150405.000000000"),
		Time:      now,
		Scope:     scope,
		Note:      note,
		Variables: vars,
	}
	list := loadSnapshots()
	list = append([]Snapshot{snap}, list...)
	if len(list) > maxSnapshots {
		list = list[:maxSnapshots]
	}
	if err := saveSnapshots(list); err != nil {
		return snap, err
	}
	return snap, nil
}

func readPending() *PendingEdits {
	var p PendingEdits
	if err := readJSON(pendingPath(), &p); err != nil {
		return nil
	}
	return &p
}

func writePending(p PendingEdits) error {
	return writeJSON(pendingPath(), p)
}

func clearPending() {
	_ = os.Remove(pendingPath())
}
