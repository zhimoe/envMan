package environment

import (
	"sync"

	"envman/internal/domain"
)

// Memory is a deterministic Backend used by non-Windows development and
// application tests.
type Memory struct {
	mu   sync.RWMutex
	vars map[domain.Scope][]domain.Variable
}

func NewMemory(vars map[domain.Scope][]domain.Variable) *Memory {
	backend := &Memory{vars: make(map[domain.Scope][]domain.Variable, len(vars))}
	for scope, values := range vars {
		backend.vars[scope] = domain.Clone(values)
	}
	return backend
}

func (m *Memory) Load(scope domain.Scope) ([]domain.Variable, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return domain.Clone(m.vars[scope]), nil
}

func (m *Memory) ApplyChanges(scope domain.Scope, before, desired []domain.Variable) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	upserts, deletions := domain.Diff(before, desired)
	current := domain.Clone(m.vars[scope])
	for _, variable := range upserts {
		variable.Type = domain.ValueType(variable.Type, variable.Value)
		_, index := domain.Find(current, variable.Name)
		if index < 0 {
			current = append(current, variable)
		} else {
			current[index] = variable
		}
	}
	for _, name := range deletions {
		_, index := domain.Find(current, name)
		if index >= 0 {
			current = append(current[:index], current[index+1:]...)
		}
	}
	m.vars[scope] = current
	return nil
}

func (m *Memory) Elevated() bool { return true }

func (m *Memory) RelaunchElevated(string) error { return nil }

// Replace is intended for deterministic tests and development fixtures.
func (m *Memory) Replace(scope domain.Scope, vars []domain.Variable) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vars[scope] = domain.Clone(vars)
}

func DemoVariables() map[domain.Scope][]domain.Variable {
	return map[domain.Scope][]domain.Variable{
		domain.ScopeUser: {
			{Name: "CARGO_HOME", Value: `C:\Users\Ada\.cargo`, Type: "REG_SZ"},
			{Name: "EDITOR", Value: "code --wait", Type: "REG_SZ"},
			{Name: "GOPATH", Value: `C:\Users\Ada\go`, Type: "REG_SZ"},
			{Name: "JAVA_HOME", Value: `C:\Program Files\Eclipse Adoptium\jdk-21.0.5.11-hotspot`, Type: "REG_SZ"},
			{Name: "NODE_OPTIONS", Value: "--max-old-space-size=8192", Type: "REG_SZ"},
			{Name: "Path", Value: `C:\Users\Ada\AppData\Local\Microsoft\WindowsApps;C:\Users\Ada\.cargo\bin;C:\Users\Ada\scoop\shims;C:\Users\Ada\AppData\Local\pnpm;C:\Users\Ada\AppData\Roaming\npm;C:\tools\legacy`, Type: "REG_EXPAND_SZ"},
			{Name: "pnpm_HOME", Value: `C:\Users\Ada\AppData\Local\pnpm`, Type: "REG_SZ"},
			{Name: "PYTHONPATH", Value: `C:\Users\Ada\projects\lib;C:\Users\Ada\projects\vendor`, Type: "REG_SZ"},
			{Name: "PYTHONUTF8", Value: "1", Type: "REG_SZ"},
			{Name: "RUSTUP_HOME", Value: `C:\Users\Ada\.rustup`, Type: "REG_SZ"},
			{Name: "TEMP", Value: `%USERPROFILE%\AppData\Local\Temp`, Type: "REG_EXPAND_SZ"},
			{Name: "TMP", Value: `%USERPROFILE%\AppData\Local\Temp`, Type: "REG_EXPAND_SZ"},
		},
		domain.ScopeSystem: {
			{Name: "ComSpec", Value: `C:\WINDOWS\system32\cmd.exe`, Type: "REG_EXPAND_SZ"},
			{Name: "DriverData", Value: `C:\Windows\System32\Drivers\DriverData`, Type: "REG_SZ"},
			{Name: "NUMBER_OF_PROCESSORS", Value: "16", Type: "REG_SZ"},
			{Name: "OS", Value: "Windows_NT", Type: "REG_SZ"},
			{Name: "Path", Value: `C:\Windows\system32;C:\Windows;C:\Windows\System32\Wbem;C:\Windows\System32\WindowsPowerShell\v1.0\;C:\Windows\System32\OpenSSH\;C:\Program Files\dotnet\;C:\Program Files\Git\cmd;C:\Program Files\PowerShell\7`, Type: "REG_EXPAND_SZ"},
			{Name: "PATHEXT", Value: ".COM;.EXE;.BAT;.CMD;.VBS;.VBE;.JS;.JSE;.WSF;.WSH;.MSC;.CPL", Type: "REG_SZ"},
			{Name: "PROCESSOR_ARCHITECTURE", Value: "AMD64", Type: "REG_SZ"},
			{Name: "PSModulePath", Value: `C:\Program Files\WindowsPowerShell\Modules;C:\WINDOWS\system32\WindowsPowerShell\v1.0\Modules;C:\Program Files\PowerShell\7\Modules`, Type: "REG_EXPAND_SZ"},
			{Name: "SystemRoot", Value: `C:\WINDOWS`, Type: "REG_SZ"},
			{Name: "TEMP", Value: `%SystemRoot%\TEMP`, Type: "REG_EXPAND_SZ"},
			{Name: "TMP", Value: `%SystemRoot%\TEMP`, Type: "REG_EXPAND_SZ"},
			{Name: "windir", Value: `C:\WINDOWS`, Type: "REG_SZ"},
		},
	}
}
