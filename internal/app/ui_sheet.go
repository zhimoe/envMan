package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *App) openEdit(v Variable, listMode bool) {
	if a.sheet != nil {
		return
	}
	mode := 0
	if listMode || isMultiValue(v.Value) {
		mode = 1
	}
	entries := splitValue(v.Value)
	if len(entries) == 0 {
		entries = []string{""}
	}
	a.sheet = &sheetState{
		scope:         a.scope,
		originalName:  v.Name,
		originalValue: v.Value,
		name:          v.Name,
		value:         v.Value,
		valueMode:     mode,
		entries:       entries,
	}
	a.batch = nil
	a.backupsOpen = false
}

func (a *App) openCreate(prefill string) {
	if a.sheet != nil {
		return
	}
	a.sheet = &sheetState{
		create:    true,
		scope:     a.scope,
		name:      prefill,
		entries:   []string{""},
		valueMode: 0,
	}
	a.batch = nil
	a.backupsOpen = false
}

func (s *sheetState) cleanEntries() []string {
	var out []string
	for _, e := range s.entries {
		e = strings.TrimSpace(e)
		if e != "" {
			out = append(out, e)
		}
	}
	return out
}

func (s *sheetState) currentValue() string {
	if s.valueMode == 1 {
		return strings.Join(s.cleanEntries(), ";")
	}
	return s.value
}

func (a *App) sheetDirty() bool {
	s := a.sheet
	if s == nil {
		return false
	}
	if s.create {
		return strings.TrimSpace(s.name) != "" || s.currentValue() != ""
	}
	return strings.TrimSpace(s.name) != s.originalName || s.currentValue() != s.originalValue
}

func (a *App) closeSheet() {
	if a.sheet == nil {
		return
	}
	if a.sheetDirty() {
		a.modal = &modalState{
			icon:    iconInfo,
			title:   "放弃未保存的更改？",
			message: "关闭后，这个面板中的修改不会写入环境变量。",
			confirm: "放弃更改",
			cancel:  "继续编辑",
			danger:  true,
			onConfirm: func() {
				a.sheet = nil
			},
		}
		return
	}
	a.sheet = nil
}

func (a *App) sheetView(c *ui.Context) {
	s := a.sheet
	if s == nil {
		return
	}
	p := a.pal
	ui.Overlay(c, func() {
		back := ui.Row(c).Absolute().Left(0).Top(42).Right(0).Bottom(0).Justify(ui.End).
			Background(ui.RGBA(0, 0, 0, 0.32)).Modal()
		back.Children(func() {
			panel := ui.Column(c).Width(430).FillHeight().Background(p.surface).
				BorderWidth(0, 1, 0, 0).BorderColor(p.line).
				Role(ui.RoleDialog).Label("变量编辑")
			panel.Children(func() {
				a.sheetContent(c, s)
			})
			if back.OverlayShortcut(ui.Cmd, ui.KeyS) || back.OverlayShortcut(ui.Cmd, ui.KeyEnter) {
				a.saveSheet(c)
			} else if panel.PressedOutside() || back.OverlayShortcut(0, ui.KeyEscape) {
				a.closeSheet()
			}
		})
	})
}

func (a *App) sheetContent(c *ui.Context, s *sheetState) {
	p := a.pal
	// Header
	ui.Row(c).Height(52).FillWidth().Padding(0, 12, 0, 18).
		BorderWidth(0, 0, 1, 0).BorderColor(p.line).AlignItems(ui.Center).Gap(9).Children(func() {
		title := "编辑变量"
		if s.create {
			title = "新建变量"
		}
		ui.Text(c, title).FontSize(14).FontWeight(650).TextColor(p.text)
		a.scopePill(c, s.scope)
		ui.Spacer(c).Grow(1)
		a.iconButton(c, iconX, "关闭（Esc）", 25, func() { a.closeSheet() })
	})

	ui.Scroll(c).Grow(1).Padding(16, 18, 22, 18).Children(func() {
		// Name
		ui.Row(c).FillWidth().AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
			ui.Text(c, "变量名").FontSize(11).FontWeight(600).TextColor(p.text2)
			hint := "不区分大小写"
			if !s.create {
				hint = "名称不可修改"
			}
			ui.Text(c, hint).FontSize(10).TextColor(p.text3)
		})
		ui.Box(c).Height(6)
		nameInput := ui.TextInput(c, &s.name).Height(34).Font(monoFont).FontSize(12).
			Placeholder("例如：JAVA_HOME").Label("变量名")
		if !s.create {
			nameInput.ReadOnly(true)
		} else {
			nameInput.AutoFocus()
		}
		if s.nameError != "" {
			ui.Box(c).Height(4)
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconAlert).Size(11, 11).TextColor(p.danger)
				ui.Text(c, s.nameError).FontSize(10.5).TextColor(p.danger)
			})
		}

		// Value label and mode
		ui.Box(c).Height(16)
		ui.Row(c).FillWidth().AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
			ui.Text(c, "值").FontSize(11).FontWeight(600).TextColor(p.text2)
			seg := ui.Segmented(c, &s.valueMode, "文本", "列表").Label("值编辑方式")
			_ = seg
			if seg.Changed() {
				if s.valueMode == 1 {
					entries := splitValue(s.value)
					if len(entries) == 0 {
						entries = []string{""}
					}
					s.entries = entries
				} else {
					s.value = strings.Join(s.cleanEntries(), ";")
				}
			}
		})
		ui.Box(c).Height(6)

		if s.valueMode == 1 {
			a.sheetValueList(c, s)
		} else {
			ui.TextArea(c, &s.value).Height(132).Font(monoFont).FontSize(12).
				Placeholder("可以是空值，也可以包含 %USERPROFILE% 这样的引用")
		}

		value := s.currentValue()
		entries := splitValue(value)
		if value == "" {
			entries = nil
		}
		ui.Box(c).Height(5)
		ui.Row(c).FillWidth().AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
			count := len(entries)
			meta := "当前为空值"
			if s.valueMode == 1 {
				meta = fmt.Sprintf("%d 个条目", count)
			} else if count > 1 {
				meta = fmt.Sprintf("%d 个条目 · 以 ; 分隔", count)
			} else if value != "" {
				meta = fmt.Sprintf("单值 · %d 个字符", len([]rune(value)))
			}
			ui.Text(c, meta).FontSize(9.5).TextColor(p.text3)
			ui.Text(c, "保存前自动备份").FontSize(9.5).TextColor(p.text3)
		})

		a.sheetWarnings(c, entries)

		if hasRef(value) {
			ui.Box(c).Height(8)
			a.resolvePreview(c, value)
		}

		if isListLike(s.name) {
			a.sheetCallout(c, iconInfo, p.surface2, p.line, p.text3,
				"PATH 按从左到右的顺序查找程序。修改后会对新打开的终端生效。")
		}

		// Advanced facts: they fill out the form and make the value type visible.
		ui.Box(c).Height(14)
		ui.Box(c).Height(1).FillWidth().Background(p.lineSoft)
		ui.Box(c).Height(9)
		typ := valueType("", value)
		if !s.create {
			if v, ok := a.findVariable(s.scope, s.originalName); ok && v.Type != "" {
				typ = v.Type
			}
		}
		typeLabel := "字符串（REG_SZ）"
		if typ == "REG_EXPAND_SZ" {
			typeLabel = "可展开字符串（REG_EXPAND_SZ）"
		}
		a.sheetInfoRow(c, "值类型", typeLabel)
		a.sheetInfoRow(c, "作用范围", s.scope.String()+" · 新终端生效")

		if !s.create {
			ui.Box(c).Height(18)
			ui.Row(c).FillWidth().BorderWidth(1, 0, 0, 0).BorderColor(p.line).Padding(13, 0, 0, 0).
				AlignItems(ui.Center).Children(func() {
				ui.Text(c, "删除后，新终端不再包含此变量").FontSize(9.5).TextColor(p.text3).Grow(1)
				a.dangerTextButton(c, "删除变量", func() {
					v, ok := a.findVariable(s.scope, s.originalName)
					if ok {
						a.confirmDelete([]Variable{v})
					}
				})
			})
		}
	})

	// Footer
	ui.Row(c).Height(58).FillWidth().Padding(0, 16).BorderWidth(1, 0, 0, 0).BorderColor(p.line).
		AlignItems(ui.Center).Gap(8).Children(func() {
		ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconShieldDone).Size(12, 12).TextColor(p.success)
			ui.Text(c, "保存前自动备份").FontSize(9.5).TextColor(p.text3)
		})
		ui.Spacer(c).Grow(1)
		a.button(c, "取消", nil, buttonSecondary, true, func() { a.closeSheet() })
		label := "保存更改"
		if s.create {
			label = "创建变量"
		}
		a.button(c, label, nil, buttonPrimary, true, func() { a.saveSheet(c) })
	})
}

func (a *App) sheetInfoRow(c *ui.Context, label, value string) {
	p := a.pal
	ui.Row(c).FillWidth().Padding(2, 0).Gap(8).Children(func() {
		ui.Text(c, label).FontSize(9.5).TextColor(p.text3).Width(58)
		ui.Text(c, value).FontSize(10).TextColor(p.text2).Grow(1)
	})
}

func (a *App) sheetValueList(c *ui.Context, s *sheetState) {
	p := a.pal
	ui.Column(c).FillWidth().Radius(9).Background(p.surface2).Border(1, p.line).Padding(5).Children(func() {
		for i := range s.entries {
			i := i
			row := ui.Row(c).Height(32).FillWidth().Radius(5).Padding(0, 3).Gap(3).AlignItems(ui.Center)
			row.Children(func() {
				ui.Textf(c, "%02d", i+1).Font(monoFont).FontSize(9).TextColor(p.text3).Width(24).TextAlign(ui.End)
				ui.TextInput(c, &s.entries[i]).Height(27).Font(monoFont).FontSize(11.5).
					Label(fmt.Sprintf("条目 %d", i+1)).Grow(1)
				up := a.iconButtonBase(c, iconChevronDown, "上移", 18)
				up.Rotate(180)
				if i == 0 {
					up.Disabled(true).Opacity(0.3)
				}
				if up.Clicked() && i > 0 {
					s.entries[i-1], s.entries[i] = s.entries[i], s.entries[i-1]
				}
				down := a.iconButtonBase(c, iconChevronDown, "下移", 18)
				if i == len(s.entries)-1 {
					down.Disabled(true).Opacity(0.3)
				}
				if down.Clicked() && i < len(s.entries)-1 {
					s.entries[i+1], s.entries[i] = s.entries[i], s.entries[i+1]
				}
				remove := a.iconButtonBase(c, iconX, "移除条目", 18)
				if remove.Clicked() {
					s.entries = append(s.entries[:i], s.entries[i+1:]...)
					if len(s.entries) == 0 {
						s.entries = []string{""}
					}
				}
			})
			if row.Hovered() {
				row.Background(p.surface)
			}
		}
		ui.Box(c).Height(4)
		add := ui.ButtonBase(c).Height(29).FillWidth().Radius(6).Center().Gap(5).Cursor(ui.CursorPointer)
		add.Label("添加条目")
		add.Children(func() {
			ui.Icon(c, iconPlus).Size(12, 12).TextColor(p.text3)
			ui.Text(c, "添加条目").FontSize(10).TextColor(p.text3)
		})
		if add.Hovered() {
			add.Background(p.surface)
		}
		if add.Clicked() {
			s.entries = append(s.entries, "")
		}
	})
}

func (a *App) sheetWarnings(c *ui.Context, entries []string) {
	p := a.pal
	seen := make(map[string]bool)
	var dup []string
	for _, e := range entries {
		k := strings.ToLower(strings.TrimSpace(e))
		if seen[k] {
			dup = append(dup, e)
		}
		seen[k] = true
	}
	if len(dup) > 0 {
		ui.Box(c).Height(8)
		a.sheetCallout(c, iconAlert, p.warnSoft, p.warnSoft, p.warn,
			fmt.Sprintf("%d 个重复条目 · %s。重复项通常不会再次生效。", len(dup), strings.Join(dup[:min(2, len(dup))], "、")))
	}
}

func (a *App) sheetCallout(c *ui.Context, ic *ui.SVG, bg, border, fg ui.Color, text string) {
	ui.Row(c).FillWidth().Padding(7, 9).Radius(7).Background(bg).Border(1, border).Gap(6).AlignItems(ui.Start).Children(func() {
		ui.Icon(c, ic).Size(11, 11).TextColor(fg)
		ui.Text(c, text).FontSize(10).TextColor(fg).MaxLines(4)
	})
}

func (a *App) resolvePreview(c *ui.Context, value string) {
	p := a.pal
	expanded := a.expandValue(value)
	if expanded == value {
		return
	}
	ui.Row(c).FillWidth().Padding(7, 9).Radius(7).Background(p.surface2).Border(1, p.line).Children(func() {
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, "解析预览").FontSize(8.5).FontWeight(600).LetterSpacing(0.4).TextColor(p.text3)
			ui.Text(c, expanded).Font(monoFont).FontSize(10.5).TextColor(p.text2)
		})
	})
}

func (a *App) expandValue(value string) string {
	known := make(map[string]string)
	// User variables override system variables in the effective environment.
	for _, scope := range []Scope{ScopeSystem, ScopeUser} {
		for _, v := range a.variables(scope) {
			known[strings.ToLower(v.Name)] = v.Value
		}
	}
	result := value
	for i := 0; i < 3; i++ {
		before := result
		result = refPattern.ReplaceAllStringFunc(result, func(ref string) string {
			name := strings.ToLower(ref[1 : len(ref)-1])
			value, ok := known[name]
			if !ok {
				return ref
			}
			return value
		})
		if result == before {
			break
		}
	}
	return result
}

func (a *App) saveSheet(c *ui.Context) {
	s := a.sheet
	if s == nil {
		return
	}
	s.name = strings.TrimSpace(s.name)
	if s.name == "" {
		s.nameError = "请输入变量名"
		return
	}
	if strings.Contains(s.name, "=") {
		s.nameError = "变量名不能包含 ="
		return
	}
	for _, v := range a.variables(s.scope) {
		if strings.EqualFold(v.Name, s.name) && !(s.create == false && strings.EqualFold(v.Name, s.originalName)) {
			s.nameError = fmt.Sprintf("此范围中已存在变量 “%s”", v.Name)
			return
		}
	}
	s.nameError = ""
	if s.valueMode == 1 {
		s.entries = s.cleanEntries()
		if len(s.entries) == 0 {
			s.entries = []string{""}
		}
		s.value = strings.Join(s.entries, ";")
	}
	value := s.currentValue()
	scope := s.scope
	before := cloneVars(a.variables(scope))
	desired := cloneVars(a.variables(scope))
	if s.create {
		desired = append(desired, Variable{Name: s.name, Value: value, Type: valueType("", value)})
	} else {
		_, idx := findVar(desired, s.originalName)
		if idx < 0 {
			a.toast("变量已不存在")
			s.nameError = "变量已不存在，请重新打开"
			return
		}
		desired[idx].Value = value
		desired[idx].Type = valueType(desired[idx].Type, value)
		desired[idx].Name = s.name
	}
	success := "已保存 “" + s.name + "”"
	if s.create {
		success = "已创建 “" + s.name + "”"
	}
	note := success
	applied := a.commit(scope, desired, note, success, s.name, before)
	if applied {
		a.sheet = nil
	}
	_ = c
}
