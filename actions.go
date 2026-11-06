package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/egoist/mygo"
)

func cloneVars(vars []Variable) []Variable {
	return slices.Clone(vars)
}

// mergeVariableChanges applies only the changes between before and desired to
// latest. Environment variables can be changed by installers and other tools
// while EnvMan is open; preserving unrelated changes avoids overwriting them
// with the table's older snapshot.
func mergeVariableChanges(latest, before, desired []Variable) []Variable {
	result := cloneVars(latest)
	desiredByName := make(map[string]Variable, len(desired))
	beforeByName := make(map[string]Variable, len(before))
	for _, v := range before {
		beforeByName[strings.ToLower(v.Name)] = v
	}
	for _, v := range desired {
		desiredByName[strings.ToLower(v.Name)] = v
	}

	for name := range beforeByName {
		if _, keep := desiredByName[name]; keep {
			continue
		}
		for i := len(result) - 1; i >= 0; i-- {
			if strings.EqualFold(result[i].Name, name) {
				result = append(result[:i], result[i+1:]...)
			}
		}
	}

	for name, next := range desiredByName {
		previous, existed := beforeByName[name]
		if existed && previous == next {
			continue
		}
		_, index := findVar(result, next.Name)
		if index < 0 {
			result = append(result, next)
			continue
		}
		result[index] = next
	}
	return result
}

func (a *App) variablesWith(scope Scope, name, value string) ([]Variable, bool) {
	out := cloneVars(a.variables(scope))
	old, idx := findVar(out, name)
	if idx < 0 {
		typ := valueType("", value)
		out = append(out, Variable{Name: name, Value: value, Type: typ})
		return out, false
	}
	out[idx].Value = value
	out[idx].Type = valueType(old.Type, value)
	return out, true
}

func (a *App) variablesWithout(scope Scope, names []string) []Variable {
	drop := make(map[string]bool, len(names))
	for _, n := range names {
		drop[strings.ToLower(n)] = true
	}
	var out []Variable
	for _, v := range a.variables(scope) {
		if !drop[strings.ToLower(v.Name)] {
			out = append(out, v)
		}
	}
	return out
}

// commit writes a full scope and offers undo. It returns true when the change
// was applied in this process. A system change made without elevation asks for
// a restart instead.
func (a *App) commit(scope Scope, desired []Variable, note, success, focus string, before []Variable) bool {
	if scope == ScopeSystem && !a.admin {
		pending := PendingEdits{
			Scope:     scope,
			Note:      success,
			Focus:     focus,
			Before:    cloneVars(before),
			Variables: desired,
		}
		a.modal = &modalState{
			icon:    iconShield,
			title:   "需要管理员权限",
			message: "修改系统变量会影响这台电脑上的所有用户。EnvMan 将以管理员身份重新启动，Windows 会显示 UAC 授权提示。",
			confirm: "以管理员身份重启",
			cancel:  "取消",
			onConfirm: func() {
				if _, err := takeSnapshot(scope, note); err != nil {
					a.toast("创建备份失败，未应用更改：" + err.Error())
					return
				}
				if err := writePending(pending); err != nil {
					a.toast("无法准备提权后的写入：" + err.Error())
					return
				}
				if err := relaunchElevated("--data-dir=\"" + userDataDir() + "\""); err != nil {
					clearPending()
					a.toast("未获得管理员权限：" + err.Error())
					return
				}
				mygo.App.Quit()
			},
		}
		return false
	}

	snapshot, err := takeSnapshot(scope, note)
	if err != nil {
		a.toast("创建备份失败，未应用更改：" + err.Error())
		return false
	}
	toWrite := desired
	if before != nil {
		toWrite = mergeVariableChanges(snapshot.Variables, before, desired)
	}
	if err := applyEnvChanges(scope, snapshot.Variables, toWrite); err != nil {
		a.toast("保存失败：" + err.Error())
		return false
	}
	a.scope = scope
	a.reload()
	if focus != "" {
		a.expand(scope, focus)
		rows := a.filtered(scope)
		found := false
		for i, item := range rows {
			if strings.EqualFold(item.Name, focus) {
				a.setFocus(i)
				found = true
				break
			}
		}
		if !found && (a.filter != ViewAll || a.query != "") {
			a.filter = ViewAll
			a.query = ""
			rows = a.filtered(scope)
			for i, item := range rows {
				if strings.EqualFold(item.Name, focus) {
					a.setFocus(i)
					break
				}
			}
		}
	}
	if success != "" {
		if before != nil {
			undoBefore := cloneVars(toWrite)
			undoDesired := cloneVars(snapshot.Variables)
			a.undoToast(success, func() {
				a.commit(scope, undoDesired, "撤销更改", "已撤销更改", focus, undoBefore)
			})
		} else {
			a.toast(success)
		}
	}
	return true
}

func (a *App) requestAdmin() {
	if a.admin {
		return
	}
	a.modal = &modalState{
		icon:    iconShield,
		title:   "需要管理员权限",
		message: "以管理员身份重新启动后，才能修改系统变量。Windows 会显示 UAC 授权提示。",
		confirm: "以管理员身份重启",
		cancel:  "取消",
		onConfirm: func() {
			clearPending()
			if err := relaunchElevated("--data-dir=\"" + userDataDir() + "\""); err != nil {
				a.toast("未获得管理员权限：" + err.Error())
				return
			}
			mygo.App.Quit()
		},
	}
}

func (a *App) confirmDelete(vars []Variable) {
	if len(vars) == 0 {
		return
	}
	scope := a.scope
	names := make([]string, 0, len(vars))
	for _, v := range vars {
		names = append(names, v.Name)
	}
	before := cloneVars(a.variables(scope))
	title := "删除这个变量？"
	if len(names) > 1 {
		title = fmt.Sprintf("删除 %d 个变量？", len(names))
	}
	shown := names
	if len(shown) > 6 {
		shown = shown[:6]
	}
	listText := strings.Join(shown, "\n")
	if len(names) > len(shown) {
		listText += fmt.Sprintf("\n… 还有 %d 个", len(names)-len(shown))
	}
	message := "将从" + scope.String() + "中删除：\n" + listText + "\n\n保存前会自动创建备份，可在 5 秒内撤销。"
	a.modal = &modalState{
		icon:    iconTrash,
		title:   title,
		message: message,
		confirm: "删除",
		cancel:  "取消",
		danger:  true,
		onConfirm: func() {
			a.sheet = nil
			a.batch = nil
			a.clearChecked()
			success := "已删除 “" + names[0] + "”"
			if len(names) > 1 {
				success = fmt.Sprintf("已删除 %d 个变量", len(names))
			}
			a.commit(scope, a.variablesWithout(scope, names), "删除变量", success, "", before)
		},
	}
}

func (a *App) confirmDeleteChecked() {
	a.confirmDelete(a.checkedVariables())
}

func (a *App) applyBatchValue(scope Scope, target, value string) bool {
	before := cloneVars(a.variables(scope))
	desired, ok := a.variablesWith(scope, target, value)
	if !ok {
		a.toast("变量不存在：" + target)
		return false
	}
	applied := a.commit(scope, desired, "批量编辑 "+target, "已更新 “"+target+"”", target, before)
	if applied {
		a.batch = nil
		a.sheet = nil
	}
	return applied
}
