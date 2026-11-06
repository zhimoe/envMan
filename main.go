package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func main() {
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "--data-dir=") {
			dataDirOverride = strings.TrimPrefix(arg, "--data-dir=")
		}
	}
	a := newApp()
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
			var w, h int
			if _, err := fmt.Sscanf(size, "%dx%d", &w, &h); err == nil && w > 0 && h > 0 {
				a.win.SetSize(w, h)
			}
		}
		if path := os.Getenv("ENVMAN_CAPTURE"); path != "" {
			go func() {
				time.Sleep(500 * time.Millisecond)
				a.win.Update(func() {
					switch os.Getenv("ENVMAN_SCENE") {
					case "sheet":
						name := os.Getenv("ENVMAN_VAR")
						if name == "" {
							name = "Path"
						}
						if v, ok := a.findVariable(a.scope, name); ok {
							a.openEdit(v, true)
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
							if txt := os.Getenv("ENVMAN_BATCH_TEXT"); txt != "" {
								a.batch.text = txt
							}
						}
						if os.Getenv("ENVMAN_SCENE") == "applybatch" && a.batch != nil {
							entries, _ := batchParse(a.batch.text)
							a.applyBatchValue(a.batch.scope, a.batch.target, strings.Join(entries, ";"))
						}
					case "save":
						if v, ok := a.findVariable(a.scope, "JAVA_HOME"); ok {
							a.openEdit(v, false)
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
						if v, ok := a.findVariable(a.scope, "Path"); ok {
							a.scope = ScopeUser
							a.confirmDelete([]Variable{v})
						}
					}
				})
				time.Sleep(900 * time.Millisecond)
				png, err := a.win.CapturePage()
				if err == nil {
					_ = os.WriteFile(path, png, 0o644)
				}
				mygo.App.Quit()
			}()
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
