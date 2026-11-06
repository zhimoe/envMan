package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

func (a *App) sideItem(c *ui.Context, s *ui.SVG, label string, count int, active bool, accent ui.Color, locked bool, onClick func()) {
	p := a.pal
	bg := ui.Transparent
	border := ui.Transparent
	fg := p.text2
	iconColor := p.text3
	if active {
		bg = p.surface
		border = p.line
		fg = p.text
		iconColor = accent
	}
	b := ui.ButtonBase(c).Height(31).FillWidth().Padding(0, 8).Radius(7).Gap(8).
		Background(bg).Border(1, border).Cursor(ui.CursorPointer)
	b.Children(func() {
		if s != nil {
			ui.Icon(c, s).Size(15, 15).TextColor(iconColor)
		}
		ui.Text(c, label).FontSize(12).FontWeight(500).TextColor(fg).Grow(1).SingleLine()
		if locked {
			ui.Icon(c, iconLock).Size(11, 11).TextColor(p.text3)
		}
		if count >= 0 {
			fg2, bg2 := p.text3, p.surface3
			if active {
				fg2, bg2 = accent, p.accentSoft
			}
			ui.Row(c).Padding(1.5, 5).Radius(99).Background(bg2).Children(func() {
				ui.Text(c, fmt.Sprint(count)).FontSize(9.5).FontWeight(600).TextColor(fg2)
			})
		}
	})
	if b.Hovered() && !active {
		b.Background(p.surface2)
		b.Border(1, p.lineSoft)
	}
	if b.Clicked() && onClick != nil {
		onClick()
	}
}

func (a *App) sidebar(c *ui.Context) {
	p := a.pal
	ui.Column(c).Width(a.sidebarWidth(c)).FillHeight().Background(p.sidebar).
		BorderWidth(0, 1, 0, 0).BorderColor(p.line).Padding(12, 10, 10, 10).Children(func() {
		a.sectionLabel(c, "变量范围")
		ui.Box(c).Height(3)
		a.sideItem(c, iconUser, "用户变量", a.scopeCount(ScopeUser), a.scope == ScopeUser, p.accent, false, func() {
			a.switchScope(ScopeUser)
		})
		a.sideItem(c, iconShield, "系统变量", a.scopeCount(ScopeSystem), a.scope == ScopeSystem, p.warn, !a.admin, func() {
			a.switchScope(ScopeSystem)
		})

		ui.Box(c).Height(15)
		a.sectionLabel(c, "快速查看")
		ui.Box(c).Height(3)
		a.sideItem(c, iconLayers, "全部变量", a.scopeCount(a.scope), a.filter == ViewAll, p.accent, false, func() {
			a.switchView(ViewAll)
		})
		a.sideItem(c, iconRoute, "PATH 路径", -1, a.filter == ViewPath, p.accent, false, func() {
			a.switchView(ViewPath)
		})
		a.sideItem(c, iconList, "多值变量", a.multiCount(a.scope), a.filter == ViewMulti, p.accent, false, func() {
			a.switchView(ViewMulti)
		})
		a.sideItem(c, iconBraces, "含变量引用", a.refCount(a.scope), a.filter == ViewRefs, p.accent, false, func() {
			a.switchView(ViewRefs)
		})

		ui.Spacer(c).Grow(1)
		a.adminCard(c)
		ui.Box(c).Height(6)
		a.sideItem(c, iconClock, "备份历史", len(a.snapshots), false, p.accent, false, func() {
			a.backupsOpen = true
		})
	})
}

func (a *App) adminCard(c *ui.Context) {
	p := a.pal
	bg := p.surface
	border := p.line
	icoBg := p.warnSoft
	icoFg := p.warn
	title := "标准权限"
	sub := "系统变量为只读"
	if a.admin {
		bg = p.successSoft
		border = p.successSoft
		icoBg = p.successSoft
		icoFg = p.success
		title = "管理员权限"
		sub = "可修改系统变量"
	}
	card := ui.Column(c).FillWidth().Radius(10).Background(bg).Border(1, border).Padding(9, 10).Gap(5)
	card.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Size(24, 24).Radius(7).Background(icoBg).Center().Children(func() {
				icon := iconShield
				if a.admin {
					icon = iconShieldDone
				}
				ui.Icon(c, icon).Size(14, 14).TextColor(icoFg)
			})
			ui.Column(c).Grow(1).Gap(0).Children(func() {
				ui.Text(c, title).FontSize(11).FontWeight(600).TextColor(p.text)
				ui.Text(c, sub).FontSize(10).TextColor(p.text3)
			})
		})
		if !a.admin {
			link := ui.ButtonBase(c).Height(20).Padding(0).Gap(3).Cursor(ui.CursorPointer)
			link.Label("以管理员身份重启")
			link.Children(func() {
				ui.Text(c, "以管理员身份重启").FontSize(10).FontWeight(600).TextColor(p.warn)
				ui.Icon(c, iconArrowRight).Size(10, 10).TextColor(p.warn)
			})
			if link.Hovered() {
				link.Background(p.warnSoft)
			}
			if link.Clicked() {
				a.requestAdmin()
			}
		}
	})
}

func (a *App) sidebarWidth(c *ui.Context) float32 {
	w, _ := c.Size()
	switch {
	case w < 1000:
		return 184
	case w < 1120:
		return 200
	default:
		return 214
	}
}
