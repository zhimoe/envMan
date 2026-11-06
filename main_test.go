//go:build !windows

package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func resetDevEnv(t *testing.T) {
	t.Helper()
	user := append([]Variable(nil), devVars[ScopeUser]...)
	system := append([]Variable(nil), devVars[ScopeSystem]...)
	t.Cleanup(func() {
		devVars[ScopeUser] = user
		devVars[ScopeSystem] = system
	})
}

func newTestApp(t *testing.T) (*App, *ui.Tester) {
	t.Helper()
	previousDataDir := dataDirOverride
	dataDirOverride = t.TempDir()
	t.Cleanup(func() { dataDirOverride = previousDataDir })
	a := newApp()
	a.admin = true
	a.themeMode = ThemeLight
	a.scope = ScopeUser
	a.filter = ViewAll
	a.query = ""
	a.expand(ScopeUser, "Path")
	tt := ui.NewTester(a.view, 1180, 780)
	return a, tt
}

func TestMergeVariableChangesPreservesUnrelatedUpdates(t *testing.T) {
	before := []Variable{
		{Name: "Path", Value: `C:\\Old`, Type: "REG_EXPAND_SZ"},
		{Name: "KEEP", Value: "old", Type: "REG_SZ"},
	}
	desired := []Variable{
		{Name: "Path", Value: `C:\\New`, Type: "REG_EXPAND_SZ"},
		{Name: "KEEP", Value: "old", Type: "REG_SZ"},
	}
	latest := []Variable{
		{Name: "Path", Value: `C:\\Old`, Type: "REG_EXPAND_SZ"},
		{Name: "KEEP", Value: "changed elsewhere", Type: "REG_SZ"},
		{Name: "ADDED_ELSEWHERE", Value: "1", Type: "REG_SZ"},
	}

	merged := mergeVariableChanges(latest, before, desired)
	path, _ := findVar(merged, "Path")
	if path.Value != `C:\\New` {
		t.Fatalf("Path = %q", path.Value)
	}
	keep, _ := findVar(merged, "KEEP")
	if keep.Value != "changed elsewhere" {
		t.Fatalf("unrelated update was overwritten: %+v", keep)
	}
	if _, index := findVar(merged, "ADDED_ELSEWHERE"); index < 0 {
		t.Fatal("concurrently added variable was removed")
	}
}

func TestApplyEnvChangesPreservesPostSnapshotUpdates(t *testing.T) {
	resetDevEnv(t)
	before := cloneVars(devVars[ScopeUser])
	desired := cloneVars(before)
	_, javaIndex := findVar(desired, "JAVA_HOME")
	desired[javaIndex].Value = `C:\AppChange`
	_, editorIndex := findVar(desired, "EDITOR")
	desired = append(desired[:editorIndex], desired[editorIndex+1:]...)
	desired = append(desired, Variable{Name: "ADDED_BY_APP", Value: "1", Type: "REG_SZ"})

	_, cargoIndex := findVar(devVars[ScopeUser], "CARGO_HOME")
	devVars[ScopeUser][cargoIndex].Value = `C:\ChangedAfterSnapshot`
	_, goPathIndex := findVar(devVars[ScopeUser], "GOPATH")
	devVars[ScopeUser] = append(devVars[ScopeUser][:goPathIndex], devVars[ScopeUser][goPathIndex+1:]...)
	devVars[ScopeUser] = append(devVars[ScopeUser], Variable{
		Name: "ADDED_AFTER_SNAPSHOT", Value: "external", Type: "REG_SZ",
	})

	if err := applyEnvChanges(ScopeUser, before, desired); err != nil {
		t.Fatal(err)
	}
	if java, _ := findVar(devVars[ScopeUser], "JAVA_HOME"); java.Value != `C:\AppChange` {
		t.Fatalf("requested update was not applied: %+v", java)
	}
	if _, index := findVar(devVars[ScopeUser], "EDITOR"); index >= 0 {
		t.Fatal("requested deletion was not applied")
	}
	if _, index := findVar(devVars[ScopeUser], "ADDED_BY_APP"); index < 0 {
		t.Fatal("requested addition was not applied")
	}
	if cargo, _ := findVar(devVars[ScopeUser], "CARGO_HOME"); cargo.Value != `C:\ChangedAfterSnapshot` {
		t.Fatalf("unrelated update was overwritten: %+v", cargo)
	}
	if _, index := findVar(devVars[ScopeUser], "GOPATH"); index >= 0 {
		t.Fatal("unrelated deletion was undone")
	}
	if _, index := findVar(devVars[ScopeUser], "ADDED_AFTER_SNAPSHOT"); index < 0 {
		t.Fatal("unrelated addition was removed")
	}
}

func TestMainView(t *testing.T) {
	resetDevEnv(t)
	a, tt := newTestApp(t)
	if !tt.HasText("用户变量") || !tt.HasText("CARGO_HOME") || !tt.HasText("PATH 路径") {
		t.Fatalf("main view texts: %q", tt.Texts())
	}
	if !tt.HasText("Path") {
		t.Fatalf("Path missing")
	}
	_ = a
}

func TestSearch(t *testing.T) {
	resetDevEnv(t)
	_, tt := newTestApp(t)
	if err := tt.Click("搜索变量名或值"); err != nil {
		t.Fatal(err)
	}
	tt.Type("java")
	if tt.HasText("CARGO_HOME") || !tt.HasText("JAVA_HOME") {
		t.Fatalf("search texts: %q", tt.Texts())
	}
}

func TestEditAndSave(t *testing.T) {
	resetDevEnv(t)
	a, tt := newTestApp(t)
	v, ok := a.findVariable(ScopeUser, "JAVA_HOME")
	if !ok {
		t.Fatal("JAVA_HOME missing")
	}
	a.openEdit(v, false)
	tt.Frame()
	if !tt.HasText("保存更改") {
		t.Fatalf("sheet not built: %q", tt.Texts())
	}
	a.sheet.value = `C:\Java\new`
	tt.Frame()
	if err := tt.Click("保存更改"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	got, _ := a.findVariable(ScopeUser, "JAVA_HOME")
	if got.Value != `C:\Java\new` {
		t.Fatalf("JAVA_HOME = %q", got.Value)
	}
	if a.sheet != nil {
		t.Fatal("sheet did not close")
	}
	if !tt.HasText("已保存") || !tt.HasText("撤销") {
		t.Fatalf("undo toast missing: %q", tt.Texts())
	}
	for i := range devVars[ScopeUser] {
		if strings.EqualFold(devVars[ScopeUser][i].Name, "CARGO_HOME") {
			devVars[ScopeUser][i].Value = `C:\ChangedElsewhere`
		}
	}
	devVars[ScopeUser] = append(devVars[ScopeUser], Variable{
		Name: "ADDED_ELSEWHERE", Value: "1", Type: "REG_SZ",
	})
	if err := tt.Click("撤销"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	got, _ = a.findVariable(ScopeUser, "JAVA_HOME")
	if got.Value != v.Value {
		t.Fatalf("undo JAVA_HOME = %q, want %q", got.Value, v.Value)
	}
	cargo, _ := a.findVariable(ScopeUser, "CARGO_HOME")
	if cargo.Value != `C:\ChangedElsewhere` {
		t.Fatalf("undo overwrote unrelated update: %+v", cargo)
	}
	if added, ok := a.findVariable(ScopeUser, "ADDED_ELSEWHERE"); !ok || added.Value != "1" {
		t.Fatalf("undo removed unrelated variable: %+v, %v", added, ok)
	}
}

func TestExpandValueUsesEffectiveScopeAndSkipsUnknownRefs(t *testing.T) {
	resetDevEnv(t)
	a, _ := newTestApp(t)

	got := a.expandValue(`%UNKNOWN%;%TEMP%;%JAVA_HOME%`)
	want := `%UNKNOWN%;%USERPROFILE%\AppData\Local\Temp;C:\Program Files\Eclipse Adoptium\jdk-21.0.5.11-hotspot`
	if got != want {
		t.Fatalf("expandValue() = %q, want %q", got, want)
	}
}

func TestAnalyzeBatchDetectsMixedReorder(t *testing.T) {
	info := analyzeBatch(`C:\A;C:\B`, "C:\\B\nC:\\A\nC:\\C")
	if !info.hasChanges || !info.reordered || len(info.added) != 1 {
		t.Fatalf("mixed batch analysis = %+v", info)
	}

	duplicateOnly := analyzeBatch(`C:\A;C:\B`, "C:\\A\nC:\\A\nC:\\B")
	if duplicateOnly.reordered {
		t.Fatalf("duplicate-only change reported as reorder: %+v", duplicateOnly)
	}
}

func TestCreateAndDuplicate(t *testing.T) {
	resetDevEnv(t)
	a, tt := newTestApp(t)
	a.openCreate("ENVMAN_TEST")
	tt.Frame()
	a.sheet.value = "hello"
	tt.Frame()
	if err := tt.Click("创建变量"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	got, ok := a.findVariable(ScopeUser, "ENVMAN_TEST")
	if !ok || got.Value != "hello" {
		t.Fatalf("created variable: %+v %v", got, ok)
	}

	a.openCreate("ENVMAN_TEST")
	tt.Frame()
	a.sheet.value = "again"
	tt.Frame()
	if err := tt.Click("创建变量"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.sheet == nil || !strings.Contains(a.sheet.nameError, "已存在") {
		t.Fatalf("duplicate not rejected: %+v", a.sheet)
	}
}

func TestBatchApply(t *testing.T) {
	resetDevEnv(t)
	a, tt := newTestApp(t)
	a.openBatch("Path")
	tt.Frame()
	if a.batch == nil {
		t.Fatal("batch did not open")
	}
	a.batch.text = strings.Join([]string{
		`C:\A`,
		`C:\B`,
		`C:\C`,
	}, "\n")
	tt.Frame()
	if err := tt.Click("应用更改"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	v, _ := a.findVariable(ScopeUser, "Path")
	if v.Value != `C:\A;C:\B;C:\C` {
		t.Fatalf("Path = %q", v.Value)
	}
	if a.batch != nil {
		t.Fatal("batch did not close")
	}
}

func TestDelete(t *testing.T) {
	resetDevEnv(t)
	a, tt := newTestApp(t)
	v, _ := a.findVariable(ScopeUser, "CARGO_HOME")
	a.confirmDelete([]Variable{v})
	tt.Frame()
	if !tt.HasText("删除这个变量？") {
		t.Fatalf("delete modal missing: %q", tt.Texts())
	}
	if err := tt.Click("删除"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if _, ok := a.findVariable(ScopeUser, "CARGO_HOME"); ok {
		t.Fatal("variable not deleted")
	}
}

func TestSelectionCheckbox(t *testing.T) {
	resetDevEnv(t)
	a, tt := newTestApp(t)
	r, ok := tt.Find("CARGO_HOME")
	if !ok {
		t.Fatal("row label not found")
	}
	tt.ClickAt(r.X+18, r.Y+r.H/2)
	tt.Frame()
	if a.checkedCount() != 1 {
		t.Fatalf("checked = %d", a.checkedCount())
	}
	if !tt.HasText("已选择 1 项") {
		t.Fatalf("selection bar missing: %q", tt.Texts())
	}
	if err := tt.Click("复制"); err != nil {
		t.Fatal(err)
	}
	if got, want := tt.Clipboard(), `CARGO_HOME=C:\Users\Ada\.cargo`; got != want {
		t.Fatalf("clipboard = %q, want %q", got, want)
	}
	r, _ = tt.Find("已选择 1 项")
	_ = r
	if err := tt.Click("取消选择"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.checkedCount() != 0 {
		t.Fatalf("checked after clear = %d", a.checkedCount())
	}
}

func TestSortByName(t *testing.T) {
	resetDevEnv(t)
	a, tt := newTestApp(t)
	ascending := a.filtered(ScopeUser)
	if len(ascending) < 2 || ascending[0].Name != "CARGO_HOME" {
		t.Fatalf("unexpected ascending order: %+v", ascending)
	}
	if err := tt.Click("按变量名排序"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	descending := a.filtered(ScopeUser)
	if !a.sortDesc || descending[0].Name != "TMP" {
		t.Fatalf("unexpected descending order: %+v", descending)
	}
}

func TestExpandAll(t *testing.T) {
	resetDevEnv(t)
	a, tt := newTestApp(t)
	if err := tt.Click("展开全部多值"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.isExpanded(ScopeUser, "PYTHONPATH") {
		t.Fatal("PYTHONPATH not expanded")
	}
	if !tt.HasText("收起全部多值") {
		t.Fatalf("expand label did not switch: %q", tt.Texts())
	}
	if err := tt.Click("收起全部多值"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.isExpanded(ScopeUser, "PYTHONPATH") {
		t.Fatal("PYTHONPATH still expanded")
	}
}

func TestKeyboardExpandAndEdit(t *testing.T) {
	resetDevEnv(t)
	a, tt := newTestApp(t)
	a.filter = ViewPath
	a.setFocus(-1)
	tt.Frame()
	if err := tt.Click("Path"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.focusValue() < 0 {
		t.Fatalf("row not focused: %d", a.focusValue())
	}
	was := a.isExpanded(ScopeUser, "Path")
	tt.Key(0, ui.KeySpace)
	tt.Frame()
	if a.isExpanded(ScopeUser, "Path") == was {
		t.Fatalf("Space did not toggle expansion")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.sheet == nil {
		t.Fatalf("Enter did not open the editor")
	}
}

func TestBackupRestore(t *testing.T) {
	resetDevEnv(t)
	a, tt := newTestApp(t)
	old, _ := a.findVariable(ScopeUser, "JAVA_HOME")
	vars := cloneVars(a.variables(ScopeUser))
	for i := range vars {
		if strings.EqualFold(vars[i].Name, "JAVA_HOME") {
			vars[i].Value = `C:\OldJava`
		}
	}
	if err := replaceEnv(ScopeUser, vars); err != nil {
		t.Fatal(err)
	}
	snap, err := takeSnapshot(ScopeUser, "测试快照")
	if err != nil {
		t.Fatal(err)
	}
	vars = cloneVars(a.variables(ScopeUser))
	for i := range vars {
		if strings.EqualFold(vars[i].Name, "JAVA_HOME") {
			vars[i].Value = `C:\NewJava`
		}
	}
	if err := replaceEnv(ScopeUser, vars); err != nil {
		t.Fatal(err)
	}
	a.reload()
	a.confirmRestore(nil, snap)
	tt.Frame()
	if !tt.HasText("恢复此快照？") {
		t.Fatalf("restore modal missing: %q", tt.Texts())
	}
	if err := tt.Click("恢复快照"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	got, _ := a.findVariable(ScopeUser, "JAVA_HOME")
	if got.Value != `C:\OldJava` {
		t.Fatalf("restored JAVA_HOME = %q (was %q)", got.Value, old.Value)
	}
}

func TestSnapshotRetentionAndPendingRoundTrip(t *testing.T) {
	resetDevEnv(t)
	previousDataDir := dataDirOverride
	dataDirOverride = t.TempDir()
	t.Cleanup(func() { dataDirOverride = previousDataDir })

	for i := range maxSnapshots + 5 {
		if _, err := takeSnapshot(ScopeUser, fmt.Sprintf("snapshot-%02d", i)); err != nil {
			t.Fatal(err)
		}
	}
	snapshots := loadSnapshots()
	if len(snapshots) != maxSnapshots {
		t.Fatalf("snapshot count = %d, want %d", len(snapshots), maxSnapshots)
	}
	if snapshots[0].Note != "snapshot-24" || snapshots[len(snapshots)-1].Note != "snapshot-05" {
		t.Fatalf("retained snapshot range = %q ... %q", snapshots[0].Note, snapshots[len(snapshots)-1].Note)
	}

	pending := PendingEdits{
		Scope: ScopeSystem,
		Note:  "pending test",
		Variables: []Variable{
			{Name: "Path", Value: `C:\Pending`, Type: "REG_EXPAND_SZ"},
		},
	}
	if err := writePending(pending); err != nil {
		t.Fatal(err)
	}
	got := readPending()
	if got == nil || got.Note != pending.Note || len(got.Variables) != 1 {
		t.Fatalf("pending round trip = %+v", got)
	}
	clearPending()
	if got := readPending(); got != nil {
		t.Fatalf("pending file survived clear: %+v", got)
	}
}

func TestElevatedStartupAppliesPendingDiff(t *testing.T) {
	resetDevEnv(t)
	previousDataDir := dataDirOverride
	dataDirOverride = t.TempDir()
	t.Cleanup(func() { dataDirOverride = previousDataDir })

	before := cloneVars(devVars[ScopeSystem])
	desired := cloneVars(before)
	_, pathIndex := findVar(desired, "Path")
	desired[pathIndex].Value = `C:\PendingPath`
	pending := PendingEdits{
		Scope: ScopeSystem, Note: "pending applied", Focus: "Path",
		Before: before, Variables: desired,
	}
	if err := writePending(pending); err != nil {
		t.Fatal(err)
	}

	_, osIndex := findVar(devVars[ScopeSystem], "OS")
	devVars[ScopeSystem][osIndex].Value = "Changed_Elsewhere"
	devVars[ScopeSystem] = append(devVars[ScopeSystem], Variable{
		Name: "ADDED_ELSEWHERE", Value: "1", Type: "REG_SZ",
	})

	a := newApp()
	path, _ := a.findVariable(ScopeSystem, "Path")
	if path.Value != `C:\PendingPath` {
		t.Fatalf("pending Path = %q", path.Value)
	}
	osVar, _ := a.findVariable(ScopeSystem, "OS")
	if osVar.Value != "Changed_Elsewhere" {
		t.Fatalf("pending apply overwrote unrelated update: %+v", osVar)
	}
	if _, ok := a.findVariable(ScopeSystem, "ADDED_ELSEWHERE"); !ok {
		t.Fatal("pending apply removed unrelated variable")
	}
	if readPending() != nil {
		t.Fatal("successfully applied pending edit was not cleared")
	}
}

func TestScopeAndView(t *testing.T) {
	resetDevEnv(t)
	a, tt := newTestApp(t)
	tt.Click("系统变量")
	tt.Frame()
	if a.scope != ScopeSystem {
		t.Fatal("scope did not switch")
	}
	a.admin = false
	tt.Frame()
	if !tt.HasText("系统变量为只读") {
		t.Fatalf("readonly banner missing: %q", tt.Texts())
	}
	a.admin = true
	a.switchView(ViewMulti)
	tt.Frame()
	if a.filter != ViewMulti {
		t.Fatal("view did not switch")
	}
	if tt.HasText("CARGO_HOME") || !tt.HasText("PATHEXT") {
		t.Fatalf("multi view rows: %q", tt.Texts())
	}
}
