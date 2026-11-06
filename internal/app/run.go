package app

import (
	"fmt"
	"os"
	"strings"
	"time"

	"envman/internal/environment"
	"envman/internal/storage"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Run starts the native EnvMan application using the platform environment
// backend and the requested data directory.
func Run(dataDir string) error {
	a := newAppWith(environment.NewDefault(), storage.New(dataDir))
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:          "EnvMan",
			Width:          1180,
			Height:         780,
			MinWidth:       900,
			MinHeight:      620,
			StateKey:       "main",
			TitleBarStyle:  mygo.TitleBarHidden,
			TitleBarHeight: 42,
			Content:        ui.View(a.view),
		})
		switch a.themeMode {
		case ThemeLight:
			a.win.SetWindowButtonsTheme(mygo.ThemeLight)
		case ThemeDark:
			a.win.SetWindowButtonsTheme(mygo.ThemeDark)
		}
		if size := os.Getenv("ENVMAN_SIZE"); size != "" {
			var width, height int
			if _, err := fmt.Sscanf(size, "%dx%d", &width, &height); err == nil && width > 0 && height > 0 {
				a.win.SetSize(width, height)
			}
		}
		configureCapture(a)
	})
	return mygo.App.Run()
}

func configureCapture(a *App) {
	path := os.Getenv("ENVMAN_CAPTURE")
	if path == "" {
		return
	}
	go func() {
		time.Sleep(500 * time.Millisecond)
		a.win.Update(func() {
			configureCaptureScene(a)
		})
		time.Sleep(900 * time.Millisecond)
		png, err := a.win.CapturePage()
		if err == nil {
			_ = os.WriteFile(path, png, 0o644)
		}
		mygo.App.Quit()
	}()
}

func configureCaptureScene(a *App) {
	switch os.Getenv("ENVMAN_SCENE") {
	case "sheet":
		name := os.Getenv("ENVMAN_VAR")
		if name == "" {
			name = "Path"
		}
		if variable, ok := a.findVariable(a.scope, name); ok {
			a.openEdit(variable, true)
		}
	case "create":
		a.openCreate("")
	case "batch", "batchdiff", "applybatch":
		name := os.Getenv("ENVMAN_VAR")
		if name == "" {
			name = "Path"
		}
		a.openBatch(name)
		if a.batch != nil {
			if text := os.Getenv("ENVMAN_BATCH_TEXT"); text != "" {
				a.batch.text = text
			}
		}
		if os.Getenv("ENVMAN_SCENE") == "applybatch" && a.batch != nil {
			entries, _ := batchParse(a.batch.text)
			a.applyBatchValue(a.batch.scope, a.batch.target, strings.Join(entries, ";"))
		}
	case "save":
		if variable, ok := a.findVariable(a.scope, "JAVA_HOME"); ok {
			a.openEdit(variable, false)
			a.sheet.value = `C:\Program Files\Eclipse Adoptium\jdk-21.0.5.11-hotspot`
			a.saveSheet(nil)
		}
	case "system":
		a.scope = ScopeSystem
	case "readonly":
		a.scope = ScopeSystem
		a.admin = false
	case "toast":
		a.toast("测试提示 · 操作已完成")
	case "undo":
		a.undoToast("测试提示 · 操作已完成", func() {})
	case "backups":
		a.backupsOpen = true
	case "delete":
		if variable, ok := a.findVariable(a.scope, "Path"); ok {
			a.scope = ScopeUser
			a.confirmDelete([]Variable{variable})
		}
	}
}
