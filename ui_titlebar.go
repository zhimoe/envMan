package main

import (
	"github.com/egoist/mygo/ui"
)

func (a *App) titlebar(c *ui.Context) {
	p := a.pal
	bar := c.TitleBar()
	h := float32(42)
	if bar.Height > h {
		h = bar.Height
	}
	padL := bar.Left + 12
	padR := bar.Right + 10
	if bar.Left == 0 {
		padL = 12
	}
	if bar.Right == 0 {
		padR = 10
	}
	row := ui.Row(c).Height(h).FillWidth().Padding(0, padR, 0, padL).
		Background(p.sidebar).BorderWidth(0, 0, 1, 0).BorderColor(p.line).
		AlignItems(ui.Center).DragWindow()
	row.Children(func() {
		ui.Row(c).Gap(9).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Size(21, 21).Radius(6).Background(p.accent).Center().Children(func() {
				ui.Icon(c, iconSliders).Size(15, 15).TextColor(p.accentText)
			})
			ui.Text(c, "EnvMan").FontSize(12.5).FontWeight(650).TextColor(p.text)
		})
		ui.Spacer(c).Grow(1)
		themeIcon := iconMoon
		tip := "切换到深色外观"
		if a.lastDark {
			themeIcon = iconSun
			tip = "切换到浅色外观"
		}
		a.iconButtonColor(c, themeIcon, tip, 28, p.text3, func() {
			a.toggleTheme()
		})
	})
}
