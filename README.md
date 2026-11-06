# EnvMan

一个紧凑、原生的 Windows 环境变量管理工具。界面使用
[MyGo](https://github.com/egoist/mygo) native UI，以 Go 绘制，不使用
WebView、HTML、CSS 或 JavaScript。

## 已实现

- 同时浏览用户变量和系统变量，显示同名覆盖关系。
- 搜索、排序、键盘导航、批量选择、复制和 `.env` 导出。
- PATH 等多值变量逐条展开、列表编辑、排序和整段粘贴。
- 批量粘贴实时显示新增、移除、重复及顺序变化。
- 直接读写 Windows 注册表，不使用会截断长值的 `setx`。
- 系统变量写入前通过 UAC 以管理员身份重新启动。
- 每次写入前自动备份，保留最近 20 份，支持恢复和撤销。
- 浅色、深色和跟随系统三种主题，原生高 DPI 渲染。
- 保存时保留其他程序在 EnvMan 打开期间新增或修改的无关变量。

## 开发

需要 Go 1.27.3 或更高版本。MyGo CLI 已通过 `go.mod` 的 `tool` 指令固定为
v0.7.4；所有 Go 依赖都提交在 `vendor/` 中。

```sh
# 在当前平台运行；非 Windows 平台使用内置演示数据
go run ./cmd/envman

# 运行交互测试和静态检查
go test ./...
go vet ./...
go tool mygo vet .

# 构建 Windows x64 和 ARM64 可执行文件及安装器
go tool mygo build -platform windows/amd64,windows/arm64 .
```

构建产物位于 `build/windows-amd64/` 和 `build/windows-arm64/`：

- `EnvMan.exe`
- `envman-setup-<version>-<arch>.exe`

Go 1.14 及之后的工具链在存在 `vendor/` 时会自动使用它。也可以显式进行离线校验：

```sh
GOPROXY=off GOSUMDB=off go test -mod=vendor ./...
```

### Windows 原生集成测试

普通测试不会修改真实环境变量。需要在 Windows 上验证注册表类型、差异写入、
非字符串值保护和环境变化广播时，可显式运行临时变量集成测试；测试结束会自动清理：

```powershell
$env:ENVMAN_WINDOWS_INTEGRATION = "1"
go test -count=1 -run '^TestWindowsUserRegistryIntegration$' ./internal/environment
```

系统变量测试必须从“以管理员身份运行”的 PowerShell 执行：

```powershell
$env:ENVMAN_WINDOWS_SYSTEM_INTEGRATION = "1"
go test -count=1 -run '^TestWindowsSystemRegistryIntegration$' ./internal/environment
```

安装器最终验收还应确认：标准权限启动不弹 UAC；只有保存系统变量时弹 UAC；
提权重启后修改生效且待应用文件被清理；新开的终端能读到修改后的值。

升级依赖后必须同步 vendor 内容：

```sh
go get github.com/egoist/mygo@<version>
go get -tool github.com/egoist/mygo/cmd/mygo@<version>
go mod tidy
go mod vendor
```

## Windows 数据与权限

- 用户变量：`HKCU\Environment`
- 系统变量：`HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment`
- 备份和待应用更改：`%AppData%\EnvMan`
- 修改后广播 `WM_SETTINGCHANGE`；已经打开的终端不会自动刷新，请新开终端。

EnvMan 默认以标准权限启动。系统变量可以查看、复制和导出；只有实际写入时才请求
UAC。注册表中的非字符串值不会被 EnvMan 删除。

## 项目结构

- `cmd/envman/`：程序入口；`mygo.json` 的 `main` 字段指向此包。
- `internal/app/`：MyGo 原生界面、应用状态和提交/撤销等流程编排。
- `internal/domain/`：环境变量领域类型、差异计算和并发变更合并。
- `internal/environment/`：Windows 注册表后端与非 Windows 内存后端。
- `internal/storage/`：快照、待提权变更和设置的 JSON 持久化。
- 各包内的 `*_test.go`：领域、平台、存储与 native UI 交互测试。
- `vendor/`：应用与 MyGo 构建工具的完整依赖源码。
