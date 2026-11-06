package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type App struct {
	win *mygo.Window

	scope    Scope
	filter   ViewFilter
	query    string
	sortDesc bool

	vars       map[Scope][]Variable
	loadErrors map[Scope]string
	admin      bool

	expanded map[string]bool
	checked  map[string]bool

	focusUser   int
	focusSystem int

	userList   ui.ListState
	systemList ui.ListState

	themeMode ThemeMode
	lastDark  bool
	pal       palette

	search ui.Handle

	sheet       *sheetState
	batch       *batchState
	modal       *modalState
	backupsOpen bool
	snapshots   []Snapshot

	startupToast string
	pendingToast string
	pendingErr   bool
	toastAction  *toastRequest
}

type sheetState struct {
	create        bool
	scope         Scope
	originalName  string
	originalValue string
	name          string
	value         string
	valueMode     int // 0 text, 1 list
	entries       []string
	nameError     string
	returnFocus   ui.Handle
}

type batchState struct {
	scope         Scope
	target        string
	originalValue string
	text          string
	returnFocus   ui.Handle
}

type toastRequest struct {
	message string
	label   string
	action  func()
}

type modalState struct {
	title     string
	message   string
	confirm   string
	cancel    string
	danger    bool
	icon      *ui.SVG
	onConfirm func()
	onCancel  func()
}

func newApp() *App {
	a := &App{
		vars:        make(map[Scope][]Variable),
		loadErrors:  make(map[Scope]string),
		expanded:    make(map[string]bool),
		checked:     make(map[string]bool),
		focusUser:   -1,
		focusSystem: -1,
	}
	settings := loadSettings()
	a.themeMode = parseThemeMode(settings.Theme)
	if t := os.Getenv("ENVMAN_THEME"); t != "" {
		a.themeMode = parseThemeMode(t)
	}
	a.admin = elevated()
	if v := os.Getenv("ENVMAN_ADMIN"); v == "0" {
		a.admin = false
	} else if v == "1" {
		a.admin = true
	}

	var applied *PendingEdits
	if p := readPending(); p != nil {
		if elevated() {
			current, err := loadEnv(p.Scope)
			if err != nil {
				a.startupToast = "提权后读取环境变量失败：" + err.Error()
				a.pendingErr = true
			} else {
				desired := p.Variables
				if p.Before != nil {
					desired = mergeVariableChanges(current, p.Before, p.Variables)
				}
				if err := applyEnvChanges(p.Scope, current, desired); err != nil {
					a.startupToast = "提权后应用更改失败：" + err.Error()
					a.pendingErr = true
				} else {
					clearPending()
					applied = p
				}
			}
		}
	}
	a.reload()
	a.snapshots = loadSnapshots()
	a.expand(ScopeUser, "Path")
	if applied != nil {
		a.scope = applied.Scope
		if applied.Note != "" {
			a.startupToast = applied.Note
		} else {
			a.startupToast = "已应用系统变量更改"
		}
		if applied.Focus != "" {
			a.expand(applied.Scope, applied.Focus)
			rows := a.filtered(applied.Scope)
			for i, item := range rows {
				if strings.EqualFold(item.Name, applied.Focus) {
					a.setFocus(i)
					break
				}
			}
		}
	}
	return a
}

func (a *App) reload() {
	for _, s := range []Scope{ScopeUser, ScopeSystem} {
		vars, err := loadEnv(s)
		if err != nil {
			a.loadErrors[s] = describeError(s, err)
			continue
		}
		a.vars[s] = vars
		delete(a.loadErrors, s)
	}
	a.snapshots = loadSnapshots()
}

func (a *App) view(c *ui.Context) {
	t, p := a.buildTheme(c)
	c.SetTheme(t)
	a.pal = p
	a.lastDark = t.Dark
	a.shortcuts(c)

	ui.Column(c).Fill().Background(p.window).Children(func() {
		a.titlebar(c)
		if a.batch != nil {
			a.batchView(c)
		} else {
			a.mainView(c)
		}
	})

	if a.sheet != nil {
		a.sheetView(c)
	}
	if a.backupsOpen {
		a.backupsView(c)
	}
	if a.modal != nil {
		a.modalView(c)
	}

	if a.startupToast != "" {
		msg := a.startupToast
		a.startupToast = ""
		if a.pendingErr {
			c.AddToast(ui.Toast{Title: msg, Type: "error", Timeout: 6_000_000_000})
			a.pendingErr = false
		} else {
			c.Toast(msg)
		}
	}
	if a.pendingToast != "" {
		msg := a.pendingToast
		a.pendingToast = ""
		c.Toast(msg)
	}
	if a.toastAction != nil {
		t := a.toastAction
		a.toastAction = nil
		c.ToastAction(t.message, t.label, t.action)
	}
}

func (a *App) toast(msg string) {
	a.pendingToast = msg
	if a.win != nil {
		a.win.Invalidate()
	}
}

func (a *App) undoToast(msg string, action func()) {
	a.toastAction = &toastRequest{message: msg, label: "撤销", action: action}
	if a.win != nil {
		a.win.Invalidate()
	}
}

func (a *App) shortcuts(c *ui.Context) {
	c.OnShortcut(ui.Cmd, ui.KeyK, func() {
		a.search.RequestFocus(c)
	})
	c.OnShortcut(ui.Cmd, ui.KeyN, func() {
		if a.sheet == nil && a.batch == nil && a.modal == nil {
			a.openCreate("")
		}
	})
	c.OnShortcut(ui.Cmd, ui.KeyE, func() {
		if a.sheet == nil && a.batch == nil && a.modal == nil {
			a.openBatch("")
		}
	})
	// The sheet and batch pages handle Escape themselves.
	c.OnShortcut(0, ui.KeySlash, func() {
		if a.sheet == nil && a.batch == nil && a.modal == nil && a.backupsOpen == false {
			a.search.RequestFocus(c)
		}
	})
}

func (a *App) switchScope(s Scope) {
	if a.scope == s {
		return
	}
	a.scope = s
	a.clearChecked()
	a.setFocus(-1)
	a.sheet = nil
	a.batch = nil
}

func (a *App) switchView(v ViewFilter) {
	if a.filter == v {
		return
	}
	a.filter = v
	a.clearChecked()
	a.setFocus(-1)
	if v == ViewPath {
		for _, item := range a.variables(a.scope) {
			if strings.EqualFold(item.Name, "Path") {
				a.expand(a.scope, item.Name)
				break
			}
		}
	}
}

func (a *App) scopeSubtitle() string {
	if a.scope == ScopeUser {
		return "仅影响当前用户 · 新终端生效"
	}
	if a.admin {
		return "影响所有用户 · 管理员模式"
	}
	return "影响所有用户 · 需要管理员权限才能修改"
}

func (a *App) showKey() string {
	return fmt.Sprintf("%d\x00", a.scope)
}
