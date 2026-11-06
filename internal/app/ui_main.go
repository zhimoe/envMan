package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const (
	colCheck   float32 = 34
	colName    float32 = 200
	colActions float32 = 72
	rowHeight  float32 = 44
)

func (a *App) mainView(c *ui.Context) {
	ui.Row(c).Grow(1).FillWidth().AlignItems(ui.Stretch).Children(func() {
		a.sidebar(c)
		a.mainPane(c)
	})
}

func (a *App) mainPane(c *ui.Context) {
	p := a.pal
	ui.Column(c).Grow(1).FillHeight().Background(p.window).Children(func() {
		a.mainHeader(c)
		a.toolbar(c)
		if err := a.loadErrors[a.scope]; err != "" {
			a.errorBanner(c, err)
		}
		if a.scope == ScopeSystem && !a.admin {
			a.readonlyBanner(c)
		}
		if a.checkedCount() > 0 {
			a.selectionBar(c)
		}
		rows := a.currentRows()
		a.table(c, rows)
	})
}

func (a *App) mainHeader(c *ui.Context) {
	p := a.pal
	ui.Row(c).Padding(15, 22, 0, 22).FillWidth().AlignItems(ui.Start).Children(func() {
		ui.Column(c).Grow(1).Gap(4).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				dot := p.accent
				if a.scope == ScopeSystem {
					dot = p.warn
				}
				ui.Box(c).Size(7, 7).Radius(99).Background(dot)
				ui.Text(c, a.scope.String()).FontSize(18).FontWeight(650).TextColor(p.text)
				ui.Row(c).Padding(1.5, 5).Radius(5).Background(p.surface3).Children(func() {
					ui.Text(c, fmt.Sprint(a.scopeCount(a.scope))).FontSize(9.5).FontWeight(600).TextColor(p.text3)
				})
			})
			sub := ui.Row(c).Gap(5).AlignItems(ui.Center)
			sub.Children(func() {
				if a.scope == ScopeSystem {
					icon := iconLock
					color := p.warn
					if a.admin {
						icon = iconShieldDone
						color = p.success
					}
					ui.Icon(c, icon).Size(11, 11).TextColor(color)
				}
				textColor := p.text3
				if a.scope == ScopeSystem && !a.admin {
					textColor = p.warn
				}
				ui.Text(c, a.scopeSubtitle()).FontSize(10.5).TextColor(textColor)
			})
		})
		ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
			a.button(c, "批量编辑", iconFileEdit, buttonSecondary, true, func() {
				a.openBatch("")
			})
			a.button(c, "新建变量", iconPlus, buttonPrimary, true, func() {
				a.openCreate("")
			})
		})
	})
}

func (a *App) toolbar(c *ui.Context) {
	p := a.pal
	ui.Row(c).Padding(11, 22, 10, 22).FillWidth().AlignItems(ui.Center).Gap(10).Children(func() {
		search := ui.SearchField(c, &a.query).Grow(1).MaxWidth(500).Height(34).
			Placeholder("搜索变量名或值…").Label("搜索变量名或值").Bind(&a.search)
		if search.Changed() {
			a.clearChecked()
			a.setFocus(-1)
		}
		if search.Submitted() {
			rows := a.filtered(a.scope)
			if len(rows) == 1 {
				a.openEdit(rows[0], false)
			} else if len(rows) == 0 && strings.TrimSpace(a.query) != "" {
				a.openCreate(strings.TrimSpace(a.query))
			} else if len(rows) > 0 {
				a.setFocus(0)
			}
		}
		rows := a.filtered(a.scope)
		ui.Spacer(c).Grow(1)
		if a.query != "" || a.filter != ViewAll {
			ui.Textf(c, "%d 个结果", len(rows)).FontSize(10.5).TextColor(p.text3)
		}
		multi := 0
		allExpanded := true
		for _, v := range rows {
			if isMultiValue(v.Value) {
				multi++
				if !a.isExpanded(a.scope, v.Name) {
					allExpanded = false
				}
			}
		}
		label := "展开全部多值"
		icon := iconUnfold
		if multi > 0 && allExpanded {
			label = "收起全部多值"
			icon = iconFold
		}
		a.quietButton(c, label, icon, multi > 0, func() {
			for _, v := range rows {
				if !isMultiValue(v.Value) {
					continue
				}
				if allExpanded {
					delete(a.expanded, a.expandKey(a.scope, v.Name))
				} else {
					a.expand(a.scope, v.Name)
				}
			}
		})
	})
}

func (a *App) errorBanner(c *ui.Context, message string) {
	p := a.pal
	ui.Row(c).Margin(0, 22, 8, 22).Height(32).Padding(0, 8).Radius(9).
		Background(p.dangerSoft).Border(1, p.dangerSoft).AlignItems(ui.Center).Gap(7).Children(func() {
		ui.Icon(c, iconAlert).Size(14, 14).TextColor(p.danger)
		ui.Text(c, message).FontSize(10.5).TextColor(p.danger).Grow(1).SingleLine()
		a.textButton(c, "重试", true, func() { a.reload() })
	})
}

func (a *App) readonlyBanner(c *ui.Context) {
	p := a.pal
	ui.Row(c).Margin(0, 22, 8, 22).Height(32).Padding(0, 8).Radius(9).
		Background(p.warnSoft).Border(1, p.warnSoft).AlignItems(ui.Center).Gap(7).Children(func() {
		ui.Icon(c, iconShield).Size(14, 14).TextColor(p.warn)
		ui.Text(c, "系统变量为只读。以管理员身份重新启动后可修改。").FontSize(10.5).TextColor(p.warn).Grow(1).SingleLine()
		a.textButton(c, "立即提权", true, func() {
			a.requestAdmin()
		})
	})
}

func (a *App) selectionBar(c *ui.Context) {
	p := a.pal
	n := a.checkedCount()
	ui.Row(c).Margin(0, 22, 8, 22).Height(38).Padding(0, 6, 0, 12).Radius(9).
		Background(p.accentSoft).Border(1, p.accentSoft2).AlignItems(ui.Center).Gap(8).Children(func() {
		ui.Row(c).Size(15, 15).Radius(99).Center().Background(p.accent).Children(func() {
			ui.Icon(c, iconCheck).Size(10, 10).TextColor(p.accentText)
		})
		ui.Textf(c, "已选择 %d 项", n).FontSize(11).TextColor(p.text2)
		ui.Spacer(c).Grow(1)
		a.textButton(c, "复制", true, func() { a.copyChecked(c) })
		a.textButton(c, "导出 .env", true, func() { a.exportChecked() })
		a.divider(c, false, p.accentSoft2).Height(16)
		a.dangerTextButton(c, "删除", func() { a.confirmDeleteChecked() })
		a.iconButton(c, iconX, "取消选择", 25, func() { a.clearChecked() })
	})
}

func (a *App) dangerTextButton(c *ui.Context, label string, onClick func()) {
	p := a.pal
	b := ui.ButtonBase(c).Height(22).Padding(0, 5).Radius(5).Cursor(ui.CursorPointer).Gap(4)
	b.Label(label).Children(func() {
		ui.Icon(c, iconTrash).Size(12, 12).TextColor(p.danger)
		ui.Text(c, label).FontSize(10.5).FontWeight(600).TextColor(p.danger)
	})
	if b.Hovered() {
		b.Background(p.dangerSoft)
	}
	if b.Clicked() && onClick != nil {
		onClick()
	}
}

func (a *App) checkedVariables() []Variable {
	var out []Variable
	prefix := a.showKey()
	for _, v := range a.variables(a.scope) {
		if a.checked[prefix+strings.ToLower(v.Name)] {
			out = append(out, v)
		}
	}
	return out
}

func (a *App) copyChecked(c *ui.Context) {
	items := a.checkedVariables()
	var lines []string
	for _, v := range items {
		lines = append(lines, v.Name+"="+v.Value)
	}
	c.WriteClipboard(strings.Join(lines, "\n"))
	c.Toast(fmt.Sprintf("已复制 %d 个变量", len(items)))
}

func (a *App) exportChecked() {
	items := a.checkedVariables()
	var lines []string
	for _, v := range items {
		lines = append(lines, v.Name+"="+v.Value)
	}
	content := strings.Join(lines, "\n")
	scopeName := "user"
	if a.scope == ScopeSystem {
		scopeName = "system"
	}
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent:      a.win,
			DefaultPath: fmt.Sprintf("envman-%s.env", scopeName),
		})
		if err != nil || path == "" {
			return
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			a.postToast("导出失败：" + err.Error())
			return
		}
		a.postToast(fmt.Sprintf("已导出 %d 个变量", len(items)))
	}()
}

func (a *App) postToast(msg string) {
	if a.win == nil {
		a.pendingToast = msg
		return
	}
	a.win.Update(func() { a.toast(msg) })
}

func (a *App) table(c *ui.Context, rows []Variable) {
	p := a.pal
	ui.Column(c).Grow(1).Margin(0, 22, 12, 22).Radius(10).Background(p.surface).
		Border(1, p.line).Clip().Children(func() {
		a.tableHeader(c, rows)
		if len(rows) == 0 {
			a.emptyState(c)
		} else {
			a.tableList(c, rows)
		}
		a.tableFooter(c, rows)
	})
}

func (a *App) tableHeader(c *ui.Context, rows []Variable) {
	p := a.pal
	ui.Row(c).Height(31).FillWidth().Background(p.surface2).
		BorderWidth(0, 0, 1, 0).BorderColor(p.line).AlignItems(ui.Center).Children(func() {
		all := len(rows) > 0
		any := false
		for _, v := range rows {
			if a.isChecked(v) {
				any = true
			} else {
				all = false
			}
		}
		ui.Row(c).Width(colCheck).Padding(0, 0, 0, 11).AlignItems(ui.Center).Children(func() {
			next := !all
			a.checkBox(c, all, any && !all, "全选当前结果", func() {
				for _, v := range rows {
					a.setChecked(v, next)
				}
			})
		})
		ui.Row(c).Width(colName).Padding(0, 12, 0, 0).Children(func() {
			sort := ui.ButtonBase(c).Height(22).Padding(0, 3).Radius(4).Gap(4).Cursor(ui.CursorPointer)
			sort.Label("按变量名排序")
			sort.Children(func() {
				ui.Text(c, "变量名").FontSize(9.5).FontWeight(600).LetterSpacing(0.5).TextColor(p.text3)
				arrow := "↑"
				if a.sortDesc {
					arrow = "↓"
				}
				ui.Text(c, arrow).FontSize(9).TextColor(p.text3)
			})
			if sort.Hovered() {
				sort.Background(p.surface3)
			}
			if sort.Clicked() {
				a.sortDesc = !a.sortDesc
			}
		})
		ui.Row(c).Grow(1).MinWidth(0).Padding(0, 12, 0, 0).Children(func() {
			ui.Text(c, "值").FontSize(9.5).FontWeight(600).LetterSpacing(0.5).TextColor(p.text3)
		})
		ui.Box(c).Width(colActions)
	})
}

func (a *App) tableList(c *ui.Context, rows []Variable) {
	st := &a.userList
	if a.scope == ScopeSystem {
		st = &a.systemList
	}
	st.Key = func(i int) any { return rows[i].Key(a.scope) }
	st.Label = func(i int) string { return rows[i].Name }
	st.Selected = a.focusPtr()
	st.Selection = nil
	// The List paints its chosen row in the accent color. Our rows paint their
	// own focus surface, so keep the List's own choice invisible: both the
	// choice fill and its check mark take the color of that surface.
	p := a.pal
	selBg := p.accentSoft.Over(p.surface)
	lt := *c.Theme()
	lt.Accent = selBg
	lt.AccentHover = selBg
	lt.AccentPressed = selBg
	lt.AccentText = selBg
	var list ui.Element
	ui.Themed(c, &lt).Grow(1).FillWidth().Children(func() {
		list = ui.List(c, st, len(rows), func(i int) {
			a.variableRow(c, rows[i], i)
		}).Grow(1).Padding(0).Gap(0)
	})
	if list.Submitted() {
		f := *a.focusPtr()
		if f >= 0 && f < len(rows) {
			a.openEdit(rows[f], false)
		}
	}
	list.OnShortcut(0, ui.KeySpace, func() {
		f := *a.focusPtr()
		if f >= 0 && f < len(rows) && isMultiValue(rows[f].Value) {
			a.toggleExpanded(a.scope, rows[f].Name)
		}
	})
	list.OnShortcut(0, ui.KeyDelete, func() {
		f := *a.focusPtr()
		if f >= 0 && f < len(rows) {
			a.confirmDelete(rows[f : f+1])
		}
	})
}

func (a *App) tableFooter(c *ui.Context, rows []Variable) {
	p := a.pal
	ui.Row(c).Height(30).FillWidth().Background(p.surface2).
		BorderWidth(1, 0, 0, 0).BorderColor(p.line).Padding(0, 10).AlignItems(ui.Center).Gap(6).Children(func() {
		ui.Box(c).Size(5, 5).Radius(99).Background(p.success)
		total := a.scopeCount(a.scope)
		text := fmt.Sprintf("%d 个变量 · 保存前自动备份", total)
		if len(rows) != total {
			text = fmt.Sprintf("显示 %d / %d 个变量 · 保存前自动备份", len(rows), total)
		}
		ui.Text(c, text).FontSize(9.5).TextColor(p.text3).SingleLine()
		ui.Spacer(c).Grow(1)
		w, _ := c.Size()
		if w > 1040 {
			ui.Textf(c, "双击编辑  空格展开  %s 搜索", cmdKeyLabel()).FontSize(9.5).TextColor(p.text3)
		} else {
			ui.Text(c, "双击编辑 · 空格展开").FontSize(9.5).TextColor(p.text3)
		}
	})
}

func (a *App) emptyState(c *ui.Context) {
	p := a.pal
	ui.Column(c).Grow(1).FillWidth().Center().Gap(9).Padding(36).Children(func() {
		ui.Column(c).Size(48, 48).Radius(14).Background(p.surface2).Border(1, p.line).Center().Children(func() {
			icon := iconLayers
			if strings.TrimSpace(a.query) != "" {
				icon = iconSearch
			}
			ui.Icon(c, icon).Size(22, 22).TextColor(p.text3)
		})
		title := "这里还没有变量"
		desc := "新建一个变量，或者切换范围查看。"
		if strings.TrimSpace(a.query) != "" {
			title = fmt.Sprintf("没有找到 “%s”", a.query)
			desc = "搜索变量名或值；也可以直接新建。"
		} else if a.filter == ViewPath {
			title = "当前范围没有 PATH 变量"
			desc = "PATH 决定终端按什么顺序查找可执行文件。"
		} else if a.filter == ViewMulti {
			title = "没有多值变量"
			desc = "包含 ; 分隔条目的变量会显示在这里。"
		} else if a.filter == ViewRefs {
			title = "没有含变量引用的值"
			desc = "像 %SystemRoot% 这样的引用会显示在这里。"
		}
		ui.Text(c, title).FontSize(13.5).FontWeight(600).TextColor(p.text)
		ui.Text(c, desc).FontSize(11).TextColor(p.text3).MaxLines(2).TextAlign(ui.Center)
		ui.Row(c).Gap(8).Children(func() {
			if strings.TrimSpace(a.query) != "" {
				a.button(c, "清除搜索", nil, buttonSecondary, true, func() {
					a.query = ""
					a.setFocus(-1)
				})
			}
			a.button(c, "新建变量", iconPlus, buttonPrimary, true, func() {
				a.openCreate(strings.TrimSpace(a.query))
			})
		})
	})
}

func (a *App) variableRow(c *ui.Context, v Variable, index int) {
	p := a.pal
	focused := *a.focusPtr() == index
	outer := ui.Column(c)
	outer.Children(func() {
		var actionEls []ui.Element
		body := ui.Column(c)
		body.Children(func() {
			ui.Row(c).Height(rowHeight).FillWidth().Children(func() {
				// checkbox
				ui.Row(c).Width(colCheck).Padding(0, 0, 0, 11).AlignItems(ui.Center).Children(func() {
					checked := a.isChecked(v)
					a.checkBox(c, checked, false, "选择 "+v.Name, func() {
						a.setChecked(v, !checked)
					})
				})
				// name
				ui.Row(c).Width(colName).Padding(0, 12, 0, 0).AlignItems(ui.Center).Gap(5).Children(func() {
					a.richName(c, v.Name, a.query).Grow(1).MinWidth(0)
					if rel := a.overlap(a.scope, v.Name); rel != 0 {
						text := "覆盖系统"
						fg, bg := p.warn, p.warnSoft
						if rel == 1 {
							text = "被覆盖"
							fg, bg = p.text3, p.surface3
						}
						a.chip(c, text, fg, bg, 8.5)
					}
				})
				// value
				ui.Row(c).Grow(1).MinWidth(0).Padding(0, 12, 0, 0).AlignItems(ui.Center).Children(func() {
					a.valueCell(c, v, index)
				})
				// actions
				ui.Row(c).Width(colActions).Padding(0, 7, 0, 0).Justify(ui.End).AlignItems(ui.Center).Gap(1).Children(func() {
					edit := a.iconButtonBase(c, iconPencil, "编辑", 25)
					if edit.Hovered() {
						edit.Background(p.surface3)
					}
					if edit.Clicked() {
						a.openEdit(v, false)
					}
					more := a.iconButtonBase(c, iconMore, "更多操作", 25)
					if more.Hovered() {
						more.Background(p.surface3)
					}
					more.PopoverMenuButton(func(m *ui.Menu) {
						if m.Item("编辑").Shortcut(0, ui.KeyEnter).Chosen() {
							a.openEdit(v, false)
						}
						if isBatchEditable(v) && m.Item("批量编辑值").Chosen() {
							a.openBatch(v.Name)
						}
						if isMultiValue(v.Value) {
							label := "展开条目"
							if a.isExpanded(a.scope, v.Name) {
								label = "收起条目"
							}
							if m.Item(label).Chosen() {
								a.toggleExpanded(a.scope, v.Name)
							}
						}
						m.Separator()
						if m.Item("复制值").Chosen() {
							c.WriteClipboard(v.Value)
							a.toast("已复制变量值")
						}
						if m.Item("复制 KEY=VALUE").Chosen() {
							c.WriteClipboard(v.Name + "=" + v.Value)
							a.toast("已复制 KEY=VALUE")
						}
						m.Separator()
						if m.Item("删除变量").Chosen() {
							a.confirmDelete([]Variable{v})
						}
					})
					actionEls = append(actionEls, edit, more)
				})
			})
			if isMultiValue(v.Value) && a.isExpanded(a.scope, v.Name) {
				w, _ := c.Size()
				if w < 1050 {
					ui.Row(c).Padding(0, 12, 6, colCheck).Children(func() {
						a.entriesPanel(c, v)
					})
				} else {
					ui.Row(c).Padding(0, 84, 6, colCheck+colName).Children(func() {
						a.entriesPanel(c, v)
					})
				}
			}
		})
		hovered := body.Hovered()
		body.TextColor(p.text)
		body.Background(p.surface)
		if focused {
			body.Background(p.accentSoft.Over(p.surface))
		} else if hovered {
			body.Background(p.surface2)
		}
		opacity := float32(0.45)
		if focused || hovered {
			opacity = 1
		}
		for _, el := range actionEls {
			el.Opacity(opacity)
		}
	})
	ui.Box(c).Height(1).FillWidth().Background(p.lineSoft)
}

func (a *App) valueCell(c *ui.Context, v Variable, index int) {
	p := a.pal
	value := v.Value
	if value == "" {
		ui.Text(c, "（空值）").FontSize(11).TextColor(p.text3)
		return
	}
	if !isMultiValue(value) {
		a.richValue(c, value, a.query, 12, p.text2, 2).Grow(1).MinWidth(0)
		return
	}
	entries := splitValue(value)
	expanded := a.isExpanded(a.scope, v.Name)
	toggle := ui.ButtonBase(c).Grow(1).Padding(0).Gap(6).Cursor(ui.CursorPointer)
	toggle.Label(v.Name + " 条目")
	toggle.Children(func() {
		ui.Row(c).Size(17, 17).Radius(5).Center().
			Background(chooseColor(expanded, p.accentSoft, p.surface3)).Children(func() {
			col := p.text3
			if expanded {
				col = p.accent
			}
			chev := ui.Icon(c, iconChevron).Size(11, 11).TextColor(col)
			if expanded {
				chev.Rotate(90)
			}
		})
		ui.Row(c).Padding(1.5, 5).Radius(99).Background(p.surface2).Border(1, p.line).Children(func() {
			ui.Textf(c, "%d 项", len(entries)).FontSize(9).FontWeight(600).TextColor(p.text3)
		})
		if expanded {
			ui.Text(c, "收起条目列表").FontSize(12).TextColor(p.text3).SingleLine()
		} else {
			a.richValue(c, entries[0], a.query, 12, p.text2, 1).Grow(1).MinWidth(0)
			if len(entries) > 1 {
				ui.Textf(c, "+%d", len(entries)-1).FontSize(10.5).TextColor(p.text3)
			}
		}
	})
	if toggle.Clicked() {
		a.toggleExpanded(a.scope, v.Name)
	}
}

func chooseColor(cond bool, yes, no ui.Color) ui.Color {
	if cond {
		return yes
	}
	return no
}

func (a *App) entriesPanel(c *ui.Context, v Variable) {
	p := a.pal
	entries := splitValue(v.Value)
	ui.Column(c).FillWidth().Radius(7).Background(p.surface2).Border(1, p.line).Padding(4).Children(func() {
		ui.Row(c).Padding(0, 6, 3, 6).AlignItems(ui.Center).Children(func() {
			ui.Textf(c, "%d 个条目 · 以 ; 分隔", len(entries)).FontSize(9).TextColor(p.text3).Grow(1)
			a.textButton(c, "整段粘贴", true, func() {
				a.openBatch(v.Name)
			})
			a.textButton(c, "在列表中编辑", true, func() {
				a.openEdit(v, true)
			})
		})
		for i, entry := range entries {
			ui.Row(c).Padding(1, 4).AlignItems(ui.Start).Gap(3).Children(func() {
				ui.Textf(c, "%02d", i+1).Font(monoFont).FontSize(9).TextColor(p.text3).Width(24).TextAlign(ui.End)
				a.richValue(c, entry, a.query, 11.5, p.text2, 0).Grow(1).MinWidth(0)
				a.iconButtonColor(c, iconCopy, "复制此条目", 18, p.text3, func() {
					c.WriteClipboard(entry)
					a.toast("已复制条目")
				})
			})
		}
	})
}
