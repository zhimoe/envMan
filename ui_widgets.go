package main

import (
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *App) iconButtonBase(c *ui.Context, s *ui.SVG, tip string, size float32) ui.Element {
	b := ui.ButtonBase(c).Size(size, size).Radius(5).Center().Cursor(ui.CursorPointer)
	if tip != "" {
		b.Label(tip).Tooltip(tip)
	}
	b.Children(func() {
		ui.Icon(c, s).Size(size*0.54, size*0.54).TextColor(a.pal.text3)
	})
	return b
}

func (a *App) iconButton(c *ui.Context, s *ui.SVG, tip string, size float32, onClick func()) {
	b := a.iconButtonBase(c, s, tip, size)
	if b.Hovered() {
		b.Background(a.pal.surface3)
	}
	if b.Clicked() && onClick != nil {
		onClick()
	}
}

func (a *App) iconButtonColor(c *ui.Context, s *ui.SVG, tip string, size float32, color ui.Color, onClick func()) {
	b := ui.ButtonBase(c).Size(size, size).Radius(5).Center().Cursor(ui.CursorPointer)
	b.Label(tip).Tooltip(tip)
	b.Children(func() {
		ui.Icon(c, s).Size(size*0.54, size*0.54).TextColor(color)
	})
	if b.Hovered() {
		b.Background(a.pal.surface3)
	}
	if b.Clicked() && onClick != nil {
		onClick()
	}
}

type buttonKind int

const (
	buttonPrimary buttonKind = iota
	buttonSecondary
	buttonGhost
	buttonDanger
	buttonText
)

func (a *App) button(c *ui.Context, label string, s *ui.SVG, kind buttonKind, enabled bool, onClick func()) {
	p := a.pal
	h := float32(31)
	px := float32(10)
	fg := p.text2
	bg := p.surface
	border := p.line
	hover := p.surface2
	if kind == buttonPrimary {
		fg = p.accentText
		bg = p.accent
		border = p.accent
		hover = p.accentHover
	} else if kind == buttonGhost {
		fg = p.text2
		bg = p.surface3
		border = p.surface3
		hover = p.line
	} else if kind == buttonDanger {
		fg = ui.Hex("#ffffff")
		bg = p.danger
		border = p.danger
		hover = p.danger
	} else if kind == buttonText {
		h = 22
		px = 5
		fg = p.accent
		bg = ui.Transparent
		border = ui.Transparent
		hover = p.accentSoft
	}
	if !enabled {
		fg = p.text3
		bg = p.surface3
		border = p.line
		if kind == buttonText {
			bg = ui.Transparent
			border = ui.Transparent
		}
	}
	b := ui.ButtonBase(c).Height(h).Padding(0, px).Radius(8).Gap(6).Cursor(ui.CursorPointer)
	b.Background(bg).Border(1, border).TextColor(fg).FontSize(12).FontWeight(600)
	b.Label(label)
	if !enabled {
		b.Disabled(true)
	}
	if b.Hovered() && enabled {
		b.Background(hover)
	}
	b.Children(func() {
		if s != nil {
			ui.Icon(c, s).Size(14, 14).TextColor(fg)
		}
		ui.Text(c, label).FontSize(12).FontWeight(600).TextColor(fg).SingleLine()
	})
	if enabled && b.Clicked() && onClick != nil {
		onClick()
	}
}

func (a *App) textButton(c *ui.Context, label string, enabled bool, onClick func()) {
	a.button(c, label, nil, buttonText, enabled, onClick)
}

func (a *App) chip(c *ui.Context, text string, fg, bg ui.Color, size float32) ui.Element {
	return ui.Row(c).Padding(1.5, 5).Radius(99).Background(bg).AlignItems(ui.Center).Children(func() {
		ui.Text(c, text).FontSize(size).FontWeight(600).TextColor(fg).SingleLine()
	})
}

func (a *App) sectionLabel(c *ui.Context, text string) {
	ui.Text(c, text).FontSize(9.5).FontWeight(600).LetterSpacing(0.6).TextColor(a.pal.text3).Padding(0, 8)
}

func (a *App) scopePill(c *ui.Context, scope Scope) {
	fg, bg := a.pal.accent, a.pal.accentSoft
	if scope == ScopeSystem {
		fg, bg = a.pal.warn, a.pal.warnSoft
	}
	ui.Row(c).Background(bg).Radius(99).Padding(3, 7).Gap(4).AlignItems(ui.Center).Children(func() {
		s := iconUser
		if scope == ScopeSystem {
			s = iconShield
		}
		ui.Icon(c, s).Size(10, 10).TextColor(fg)
		ui.Text(c, scope.String()).FontSize(9.5).FontWeight(600).TextColor(fg)
	})
}

// spansForText returns spans that highlight %VAR% references and the search
// query. base is the ordinary text color.
func (a *App) spansForText(text, query string, base ui.Color, withRefs bool) []ui.Span {
	p := a.pal
	refColor := p.accent
	refBg := p.accentSoft
	q := strings.ToLower(strings.TrimSpace(query))
	var spans []ui.Span

	addRun := func(s string, color ui.Color, bg ui.Color) {
		if s == "" {
			return
		}
		if q == "" {
			spans = append(spans, ui.Span{Text: s, Color: color, Background: bg, Radius: 2})
			return
		}
		lower := strings.ToLower(s)
		pos := 0
		for {
			i := strings.Index(lower[pos:], q)
			if i < 0 {
				break
			}
			i += pos
			if i > pos {
				spans = append(spans, ui.Span{Text: s[pos:i], Color: color, Background: bg, Radius: 2})
			}
			spans = append(spans, ui.Span{Text: s[i : i+len(q)], Color: color, Background: p.hit, Radius: 2})
			pos = i + len(q)
		}
		if pos < len(s) {
			spans = append(spans, ui.Span{Text: s[pos:], Color: color, Background: bg, Radius: 2})
		}
	}

	if !withRefs {
		addRun(text, base, ui.Transparent)
		return spans
	}

	pos := 0
	for pos < len(text) {
		start := strings.IndexByte(text[pos:], '%')
		if start < 0 {
			addRun(text[pos:], base, ui.Transparent)
			break
		}
		start += pos
		end := strings.IndexByte(text[start+1:], '%')
		if end < 0 {
			addRun(text[pos:], base, ui.Transparent)
			break
		}
		end += start + 1
		if start > pos {
			addRun(text[pos:start], base, ui.Transparent)
		}
		ref := text[start : end+1]
		// A reference gets its accent background; query matching inside it
		// still wins for the matched runes.
		if q == "" || !strings.Contains(strings.ToLower(ref), q) {
			spans = append(spans, ui.Span{Text: ref, Color: refColor, Background: refBg, Radius: 2})
		} else {
			addRun(ref, refColor, refBg)
		}
		pos = end + 1
	}
	return spans
}

func (a *App) richValue(c *ui.Context, value, query string, size float32, color ui.Color, lines int) ui.Element {
	spans := a.spansForText(value, query, color, true)
	e := ui.RichText(c, spans...).Font(monoFont).FontSize(size).TextColor(color)
	if lines == 1 {
		e.SingleLine()
	} else if lines > 1 {
		e.MaxLines(lines)
	}
	return e
}

func (a *App) richName(c *ui.Context, name, query string) ui.Element {
	spans := a.spansForText(name, query, a.pal.text, false)
	return ui.RichText(c, spans...).Font(monoFont).FontSize(11.5).FontWeight(600).TextColor(a.pal.text).SingleLine()
}

func (a *App) divider(c *ui.Context, horizontal bool, color ui.Color) ui.Element {
	if horizontal {
		return ui.Box(c).Height(1).FillWidth().Background(color)
	}
	return ui.Box(c).Width(1).FillHeight().Background(color)
}

func (a *App) quietButton(c *ui.Context, label string, s *ui.SVG, enabled bool, onClick func()) {
	p := a.pal
	b := ui.ButtonBase(c).Height(28).Padding(0, 8).Radius(7).Gap(6).Cursor(ui.CursorPointer)
	b.Label(label).FontSize(10.5).TextColor(p.text2)
	if s != nil {
		b.Children(func() {
			ui.Icon(c, s).Size(13, 13).TextColor(p.text3)
			ui.Text(c, label).FontSize(10.5).TextColor(p.text2)
		})
	} else {
		b.Children(func() {
			ui.Text(c, label).FontSize(10.5).TextColor(p.text2)
		})
	}
	if !enabled {
		b.Disabled(true).Opacity(0.4)
	}
	if b.Hovered() && enabled {
		b.Background(p.surface3)
	}
	if enabled && b.Clicked() && onClick != nil {
		onClick()
	}
}

func runtimeGOOS() string { return runtime.GOOS }

func cmdKeyLabel() string {
	if runtimeGOOS() == "darwin" {
		return "⌘K"
	}
	return "Ctrl K"
}

func (a *App) checkBox(c *ui.Context, checked, mixed bool, label string, onToggle func()) {
	p := a.pal
	box := ui.ButtonBase(c).Size(15, 15).Radius(4).Cursor(ui.CursorPointer).
		Border(1.4, p.line).Background(p.surface).Role(ui.RoleCheckBox).Label(label).FocusRing(false)
	box.Checked(checked)
	if mixed {
		box.Mixed()
	}
	if checked || mixed {
		box.Background(p.accent).Border(1.4, p.accent)
	}
	box.Children(func() {
		if mixed && !checked {
			ui.Icon(c, iconMinus).Size(9, 9).TextColor(p.accentText)
		} else if checked {
			ui.Icon(c, iconCheck).Size(10, 10).TextColor(p.accentText)
		}
	})
	if box.Hovered() && !checked && !mixed {
		box.Border(1.4, p.text3)
	}
	if box.Clicked() && onToggle != nil {
		onToggle()
	}
}
