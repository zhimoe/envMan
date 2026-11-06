package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const monoFont = "SF Mono, Consolas, monospace"

type ThemeMode int

const (
	ThemeSystem ThemeMode = iota
	ThemeLight
	ThemeDark
)

func (m ThemeMode) String() string {
	switch m {
	case ThemeLight:
		return "light"
	case ThemeDark:
		return "dark"
	default:
		return "system"
	}
}

func parseThemeMode(s string) ThemeMode {
	switch s {
	case "light":
		return ThemeLight
	case "dark":
		return ThemeDark
	default:
		return ThemeSystem
	}
}

type appSettings struct {
	Theme string `json:"theme,omitempty"`
}

type palette struct {
	window      ui.Color
	sidebar     ui.Color
	surface     ui.Color
	surface2    ui.Color
	surface3    ui.Color
	text        ui.Color
	text2       ui.Color
	text3       ui.Color
	line        ui.Color
	lineSoft    ui.Color
	accent      ui.Color
	accentHover ui.Color
	accentPress ui.Color
	accentText  ui.Color
	accentSoft  ui.Color
	accentSoft2 ui.Color
	warn        ui.Color
	warnSoft    ui.Color
	danger      ui.Color
	dangerSoft  ui.Color
	success     ui.Color
	successSoft ui.Color
	hit         ui.Color
	dark        bool
}

func lightPalette() palette {
	return palette{
		window:      ui.Hex("#f5f6f8"),
		sidebar:     ui.Hex("#eff1f4"),
		surface:     ui.Hex("#ffffff"),
		surface2:    ui.Hex("#f8f9fb"),
		surface3:    ui.Hex("#f1f2f6"),
		text:        ui.Hex("#1b1c21"),
		text2:       ui.Hex("#5b5e68"),
		text3:       ui.Hex("#7b7e89"),
		line:        ui.Hex("#e5e7ec"),
		lineSoft:    ui.Hex("#edeef2"),
		accent:      ui.Hex("#4a5cf0"),
		accentHover: ui.Hex("#3d4ddb"),
		accentPress: ui.Hex("#3342c4"),
		accentText:  ui.Hex("#ffffff"),
		accentSoft:  ui.RGBA(74, 92, 240, 0.09),
		accentSoft2: ui.RGBA(74, 92, 240, 0.16),
		warn:        ui.Hex("#a95c08"),
		warnSoft:    ui.RGBA(196, 108, 16, 0.11),
		danger:      ui.Hex("#dc3d43"),
		dangerSoft:  ui.RGBA(220, 61, 67, 0.09),
		success:     ui.Hex("#188a4a"),
		successSoft: ui.RGBA(24, 138, 74, 0.10),
		hit:         ui.RGBA(255, 198, 0, 0.28),
	}
}

func darkPalette() palette {
	return palette{
		window:      ui.Hex("#0f1014"),
		sidebar:     ui.Hex("#0c0d11"),
		surface:     ui.Hex("#17181e"),
		surface2:    ui.Hex("#1c1e25"),
		surface3:    ui.Hex("#20222b"),
		text:        ui.Hex("#eceef3"),
		text2:       ui.Hex("#a7abb7"),
		text3:       ui.Hex("#7f8492"),
		line:        ui.Hex("#242731"),
		lineSoft:    ui.Hex("#1f222a"),
		accent:      ui.Hex("#8090ff"),
		accentHover: ui.Hex("#93a1ff"),
		accentPress: ui.Hex("#6d7df0"),
		accentText:  ui.Hex("#10121a"),
		accentSoft:  ui.RGBA(128, 144, 255, 0.12),
		accentSoft2: ui.RGBA(128, 144, 255, 0.20),
		warn:        ui.Hex("#e5a44a"),
		warnSoft:    ui.RGBA(229, 164, 74, 0.12),
		danger:      ui.Hex("#ff6a70"),
		dangerSoft:  ui.RGBA(255, 106, 112, 0.12),
		success:     ui.Hex("#4ecb85"),
		successSoft: ui.RGBA(78, 203, 133, 0.12),
		hit:         ui.RGBA(255, 210, 80, 0.22),
		dark:        true,
	}
}

// buildTheme returns the app's compact theme and its color palette.
func (a *App) buildTheme(c *ui.Context) (*ui.Theme, palette) {
	prefs := c.Preferences()
	dark := c.Theme().Dark
	switch a.themeMode {
	case ThemeLight:
		dark = false
	case ThemeDark:
		dark = true
	}
	p := lightPalette()
	if dark {
		p = darkPalette()
	}
	scale := prefs.TextScale
	if scale < 1 {
		scale = 1
	}
	base := ui.LightTheme()
	if dark {
		base = ui.DarkTheme()
	}
	t := *base
	t.Dark = dark
	t.Spacing = 3
	t.FontSize = 12.5 * scale
	t.Radius = 7
	t.Font = ""
	t.Background = p.window
	t.Surface = p.surface
	t.SurfaceHover = p.surface2
	t.SurfacePressed = p.surface3
	t.Border = p.line
	t.Text = p.text
	t.TextMuted = p.text3
	t.Accent = p.accent
	t.AccentHover = p.accentHover
	t.AccentPressed = p.accentPress
	t.AccentText = p.accentText
	t.Danger = p.danger
	t.Warning = p.warn
	t.Success = p.success
	t.Selection = p.accentSoft2
	t.Focus = p.accent.Alpha(0.55)
	t.Scrollbar = p.text3.Alpha(0.35)
	t.ScrollbarWidth = 6
	if dark {
		t.Inverse = ui.Hex("#e9ebf1")
		t.InverseText = ui.Hex("#17181e")
	} else {
		t.Inverse = ui.Hex("#1d1f26")
		t.InverseText = ui.Hex("#f3f4f7")
	}
	if prefs.HighContrast {
		if dark {
			t.Background = ui.Hex("#000000")
			t.Surface = ui.Hex("#101010")
			t.Border = ui.Hex("#ffffff")
			t.Text = ui.Hex("#ffffff")
			t.TextMuted = ui.Hex("#e0e0e0")
		} else {
			t.Background = ui.Hex("#ffffff")
			t.Surface = ui.Hex("#ffffff")
			t.Border = ui.Hex("#000000")
			t.Text = ui.Hex("#000000")
			t.TextMuted = ui.Hex("#303030")
		}
		p.text3 = t.TextMuted
		p.line = t.Border
	}
	return &t, p
}

func (a *App) toggleTheme() {
	if a.themeMode == ThemeSystem {
		if a.lastDark {
			a.themeMode = ThemeLight
		} else {
			a.themeMode = ThemeDark
		}
	} else if a.themeMode == ThemeDark {
		a.themeMode = ThemeLight
	} else {
		a.themeMode = ThemeDark
	}
	a.saveSettings()
	if a.win != nil {
		src := mygo.ThemeLight
		if a.themeMode == ThemeDark {
			src = mygo.ThemeDark
		}
		a.win.SetWindowButtonsTheme(src)
	}
}

func (a *App) saveSettings() {
	_ = writeJSON(settingsPath(), appSettings{Theme: a.themeMode.String()})
}

func loadSettings() appSettings {
	var s appSettings
	_ = readJSON(settingsPath(), &s)
	return s
}
