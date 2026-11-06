package app

import (
	"github.com/egoist/mygo/ui"
)

func (a *App) dismissModal(confirm bool) {
	m := a.modal
	if m == nil {
		return
	}
	a.modal = nil
	if confirm {
		if m.onConfirm != nil {
			m.onConfirm()
		}
	} else if m.onCancel != nil {
		m.onCancel()
	}
}

func (a *App) modalView(c *ui.Context) {
	m := a.modal
	if m == nil {
		return
	}
	p := a.pal
	icon := m.icon
	if icon == nil {
		if m.danger {
			icon = iconTrash
		} else {
			icon = iconInfo
		}
	}
	iconBg, iconFg := p.accentSoft, p.accent
	if m.danger {
		iconBg, iconFg = p.dangerSoft, p.danger
	} else if icon == iconShield || icon == iconAlert {
		iconBg, iconFg = p.warnSoft, p.warn
	}
	ui.Overlay(c, func() {
		back := ui.Row(c).Absolute().Left(0).Top(42).Right(0).Bottom(0).Center().
			Background(ui.RGBA(0, 0, 0, 0.44)).Modal()
		back.Children(func() {
			panel := ui.Column(c).Width(400).Padding(18).Radius(12).Background(p.surface).
				Border(1, p.line).Shadow(0, 18, 50, 0, ui.RGBA(0, 0, 0, 0.35)).
				Role(ui.RoleDialog).Label(m.title)
			panel.Children(func() {
				ui.Row(c).Size(34, 34).Radius(10).Background(iconBg).Center().Children(func() {
					ui.Icon(c, icon).Size(17, 17).TextColor(iconFg)
				})
				ui.Box(c).Height(10)
				ui.Text(c, m.title).FontSize(14).FontWeight(650).TextColor(p.text)
				ui.Box(c).Height(5)
				ui.Text(c, m.message).FontSize(11).TextColor(p.text2).MaxLines(12)
				ui.Box(c).Height(16)
				ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Children(func() {
					cancelLabel := m.cancel
					if cancelLabel == "" {
						cancelLabel = "取消"
					}
					confirmLabel := m.confirm
					if confirmLabel == "" {
						confirmLabel = "确定"
					}
					a.button(c, cancelLabel, nil, buttonSecondary, true, func() {
						a.dismissModal(false)
					})
					kind := buttonPrimary
					if m.danger {
						kind = buttonDanger
					}
					a.button(c, confirmLabel, nil, kind, true, func() {
						a.dismissModal(true)
					})
				})
			})
			if panel.PressedOutside() || back.OverlayShortcut(0, ui.KeyEscape) {
				a.dismissModal(false)
			}
		})
	})
}
