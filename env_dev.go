//go:build !windows

package main

import "strings"

var devVars = map[Scope][]Variable{
	ScopeUser: {
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
	ScopeSystem: {
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

func loadEnv(scope Scope) ([]Variable, error) {
	return append([]Variable(nil), devVars[scope]...), nil
}

func writeEnv(scope Scope, name, value, typ string) error {
	for i := range devVars[scope] {
		if strings.EqualFold(devVars[scope][i].Name, name) {
			devVars[scope][i].Value = value
			devVars[scope][i].Type = valueType(typ, value)
			return nil
		}
	}
	devVars[scope] = append(devVars[scope], Variable{Name: name, Value: value, Type: valueType(typ, value)})
	return nil
}

func deleteEnv(scope Scope, name string) error {
	for i := range devVars[scope] {
		if strings.EqualFold(devVars[scope][i].Name, name) {
			devVars[scope] = append(devVars[scope][:i], devVars[scope][i+1:]...)
			return nil
		}
	}
	return nil
}

func replaceEnv(scope Scope, vars []Variable) error {
	out := make([]Variable, 0, len(vars))
	for _, v := range vars {
		v.Type = valueType(v.Type, v.Value)
		out = append(out, v)
	}
	devVars[scope] = out
	return nil
}

func applyEnvChanges(scope Scope, before, desired []Variable) error {
	upserts, deletions := envDiff(before, desired)
	for _, v := range upserts {
		if err := writeEnv(scope, v.Name, v.Value, v.Type); err != nil {
			return err
		}
	}
	for _, name := range deletions {
		if err := deleteEnv(scope, name); err != nil {
			return err
		}
	}
	return nil
}

func elevated() bool { return true }

func relaunchElevated(params string) error { return nil }
