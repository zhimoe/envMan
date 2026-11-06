//go:build windows

package main

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func registryKey(scope Scope, write bool) (registry.Key, error) {
	access := uint32(registry.QUERY_VALUE)
	if write {
		access |= registry.SET_VALUE
	}
	if scope == ScopeSystem {
		return registry.OpenKey(registry.LOCAL_MACHINE,
			`SYSTEM\CurrentControlSet\Control\Session Manager\Environment`, access)
	}
	return registry.OpenKey(registry.CURRENT_USER, `Environment`, access)
}

func typeName(t uint32) string {
	if t == registry.EXPAND_SZ {
		return "REG_EXPAND_SZ"
	}
	return "REG_SZ"
}

func loadEnv(scope Scope) ([]Variable, error) {
	key, err := registryKey(scope, false)
	if err != nil {
		return nil, err
	}
	defer key.Close()
	names, err := key.ReadValueNames(-1)
	if err != nil {
		return nil, err
	}
	out := make([]Variable, 0, len(names))
	for _, name := range names {
		value, typ, err := key.GetStringValue(name)
		if err != nil {
			// Environment keys can hold non-string values; skip them.
			continue
		}
		out = append(out, Variable{Name: name, Value: value, Type: typeName(typ)})
	}
	return out, nil
}

func setOne(key registry.Key, name, value, typ string) error {
	if typ == "REG_EXPAND_SZ" {
		return key.SetExpandStringValue(name, value)
	}
	return key.SetStringValue(name, value)
}

func writeEnv(scope Scope, name, value, typ string) error {
	key, err := registryKey(scope, true)
	if err != nil {
		if scope == ScopeSystem && !elevated() {
			return errNeedElevation
		}
		return err
	}
	defer key.Close()
	if err := setOne(key, name, value, valueType(typ, value)); err != nil {
		if scope == ScopeSystem && !elevated() {
			return errNeedElevation
		}
		return err
	}
	broadcastEnvChange()
	return nil
}

func deleteEnv(scope Scope, name string) error {
	key, err := registryKey(scope, true)
	if err != nil {
		if scope == ScopeSystem && !elevated() {
			return errNeedElevation
		}
		return err
	}
	defer key.Close()
	if err := key.DeleteValue(name); err != nil && err != registry.ErrNotExist {
		if scope == ScopeSystem && !elevated() {
			return errNeedElevation
		}
		return err
	}
	broadcastEnvChange()
	return nil
}

// applyEnvChanges writes only the user-requested difference. This prevents an
// unrelated registry change made after the pre-write snapshot from being
// overwritten by a full-scope replacement.
func applyEnvChanges(scope Scope, before, desired []Variable) error {
	upserts, deletions := envDiff(before, desired)
	if len(upserts) == 0 && len(deletions) == 0 {
		return nil
	}
	key, err := registryKey(scope, true)
	if err != nil {
		if scope == ScopeSystem && !elevated() {
			return errNeedElevation
		}
		return err
	}
	defer key.Close()

	// Write new values first. If a write fails, no requested deletion has yet
	// happened and the pre-write snapshot remains available for recovery.
	for _, v := range upserts {
		if err := setOne(key, v.Name, v.Value, valueType(v.Type, v.Value)); err != nil {
			if scope == ScopeSystem && !elevated() {
				return errNeedElevation
			}
			return fmt.Errorf("写入 %s：%w", v.Name, err)
		}
	}
	for _, name := range deletions {
		if err := key.DeleteValue(name); err != nil && err != registry.ErrNotExist {
			if scope == ScopeSystem && !elevated() {
				return errNeedElevation
			}
			return fmt.Errorf("删除 %s：%w", name, err)
		}
	}
	broadcastEnvChange()
	return nil
}

func elevated() bool {
	token := windows.GetCurrentProcessToken()
	return token.IsElevated()
}

func relaunchElevated(params string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	var args *uint16
	if params != "" {
		args, err = windows.UTF16PtrFromString(params)
		if err != nil {
			return err
		}
	}
	return windows.ShellExecute(0, verb, file, args, nil, windows.SW_SHOWNORMAL)
}

func broadcastEnvChange() {
	user32 := windows.NewLazySystemDLL("user32.dll")
	proc := user32.NewProc("SendMessageTimeoutW")
	target, err := windows.UTF16PtrFromString("Environment")
	if err != nil {
		return
	}
	const (
		hwndBroadcast   = 0xFFFF
		wmSettingChange = 0x001A
		smtoAbortIfHung = 0x0002
	)
	_, _, _ = proc.Call(
		uintptr(hwndBroadcast),
		uintptr(wmSettingChange),
		0,
		uintptr(unsafe.Pointer(target)),
		uintptr(smtoAbortIfHung),
		uintptr(5000),
		0,
	)
}
