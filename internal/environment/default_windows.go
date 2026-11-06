//go:build windows

package environment

import (
	"fmt"
	"os"
	"unsafe"

	"envman/internal/domain"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

type registryBackend struct{}

func NewDefault() Backend { return registryBackend{} }

func registryKey(scope domain.Scope, write bool) (registry.Key, error) {
	access := uint32(registry.QUERY_VALUE)
	if write {
		access |= registry.SET_VALUE
	}
	if scope == domain.ScopeSystem {
		return registry.OpenKey(registry.LOCAL_MACHINE,
			`SYSTEM\CurrentControlSet\Control\Session Manager\Environment`, access)
	}
	return registry.OpenKey(registry.CURRENT_USER, `Environment`, access)
}

func typeName(typ uint32) string {
	if typ == registry.EXPAND_SZ {
		return "REG_EXPAND_SZ"
	}
	return "REG_SZ"
}

func (registryBackend) Load(scope domain.Scope) ([]domain.Variable, error) {
	key, err := registryKey(scope, false)
	if err != nil {
		return nil, err
	}
	defer key.Close()
	names, err := key.ReadValueNames(-1)
	if err != nil {
		return nil, err
	}
	variables := make([]domain.Variable, 0, len(names))
	for _, name := range names {
		value, typ, err := key.GetStringValue(name)
		if err != nil {
			// Environment keys can hold non-string values; skip them.
			continue
		}
		variables = append(variables, domain.Variable{Name: name, Value: value, Type: typeName(typ)})
	}
	return variables, nil
}

func setOne(key registry.Key, name, value, typ string) error {
	if typ == "REG_EXPAND_SZ" {
		return key.SetExpandStringValue(name, value)
	}
	return key.SetStringValue(name, value)
}

// ApplyChanges writes only the user-requested difference, preserving unrelated
// registry updates made after the application snapshot was taken.
func (backend registryBackend) ApplyChanges(scope domain.Scope, before, desired []domain.Variable) error {
	upserts, deletions := domain.Diff(before, desired)
	if len(upserts) == 0 && len(deletions) == 0 {
		return nil
	}
	key, err := registryKey(scope, true)
	if err != nil {
		return backend.writeError(scope, err)
	}
	defer key.Close()

	// Apply writes before deletions so a partial failure leaves the snapshot
	// available and avoids deleting requested values prematurely.
	for _, variable := range upserts {
		if err := setOne(key, variable.Name, variable.Value, domain.ValueType(variable.Type, variable.Value)); err != nil {
			return backend.writeError(scope, fmt.Errorf("写入 %s：%w", variable.Name, err))
		}
	}
	for _, name := range deletions {
		if err := key.DeleteValue(name); err != nil && err != registry.ErrNotExist {
			return backend.writeError(scope, fmt.Errorf("删除 %s：%w", name, err))
		}
	}
	broadcastEnvChange()
	return nil
}

func (backend registryBackend) writeError(scope domain.Scope, err error) error {
	if scope == domain.ScopeSystem && !backend.Elevated() {
		return ErrNeedElevation
	}
	return err
}

func (registryBackend) Elevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

func (registryBackend) RelaunchElevated(params string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(executable)
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
