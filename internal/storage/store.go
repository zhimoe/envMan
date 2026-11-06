package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"envman/internal/domain"
)

const MaxSnapshots = 20

type Snapshot struct {
	ID        string            `json:"id"`
	Time      time.Time         `json:"time"`
	Scope     domain.Scope      `json:"scope"`
	Note      string            `json:"note"`
	Variables []domain.Variable `json:"variables"`
}

type PendingEdits struct {
	Scope     domain.Scope      `json:"scope"`
	Note      string            `json:"note"`
	Focus     string            `json:"focus,omitempty"`
	Before    []domain.Variable `json:"before,omitempty"`
	Variables []domain.Variable `json:"variables"`
}

type Store struct {
	dir string
}

func New(dir string) *Store {
	if dir == "" {
		base, err := os.UserConfigDir()
		if err != nil || base == "" {
			base = os.TempDir()
		}
		dir = filepath.Join(base, "EnvMan")
	}
	_ = os.MkdirAll(dir, 0o700)
	return &Store{dir: dir}
}

func (s *Store) Dir() string { return s.dir }

func (s *Store) snapshotsPath() string { return filepath.Join(s.dir, "snapshots.json") }
func (s *Store) pendingPath() string   { return filepath.Join(s.dir, "pending.json") }
func (s *Store) settingsPath() string  { return filepath.Join(s.dir, "settings.json") }

func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func (s *Store) LoadSnapshots() []Snapshot {
	var snapshots []Snapshot
	if err := readJSON(s.snapshotsPath(), &snapshots); err != nil {
		return nil
	}
	return snapshots
}

func (s *Store) SaveSnapshot(scope domain.Scope, note string, variables []domain.Variable) (Snapshot, error) {
	now := time.Now()
	snapshot := Snapshot{
		ID:        now.Format("20060102-150405.000000000"),
		Time:      now,
		Scope:     scope,
		Note:      note,
		Variables: domain.Clone(variables),
	}
	snapshots := append([]Snapshot{snapshot}, s.LoadSnapshots()...)
	if len(snapshots) > MaxSnapshots {
		snapshots = snapshots[:MaxSnapshots]
	}
	if err := writeJSON(s.snapshotsPath(), snapshots); err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

func (s *Store) ReadPending() *PendingEdits {
	var pending PendingEdits
	if err := readJSON(s.pendingPath(), &pending); err != nil {
		return nil
	}
	return &pending
}

func (s *Store) WritePending(pending PendingEdits) error {
	return writeJSON(s.pendingPath(), pending)
}

func (s *Store) ClearPending() {
	_ = os.Remove(s.pendingPath())
}

func (s *Store) LoadSettings(settings any) error {
	return readJSON(s.settingsPath(), settings)
}

func (s *Store) SaveSettings(settings any) error {
	return writeJSON(s.settingsPath(), settings)
}
