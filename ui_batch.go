package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

type batchInfo struct {
	original   []string
	added      []string
	removed    []string
	reordered  bool
	hasChanges bool
	duplicates []string
}

func batchParse(text string) (entries []string, ignored int) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	for p := range strings.SplitSeq(strings.ReplaceAll(text, ";", "\n"), "\n") {
		p = strings.TrimSpace(p)
		if p == "" {
			ignored++
			continue
		}
		entries = append(entries, p)
	}
	return entries, ignored
}

func normalizeEntry(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func analyzeBatch(originalValue, text string) batchInfo {
	original := splitValue(originalValue)
	if originalValue == "" {
		original = nil
	}
	entries, _ := batchParse(text)
	oldKeys := make(map[string]bool)
	for _, o := range original {
		oldKeys[normalizeEntry(o)] = true
	}
	newKeys := make(map[string]bool)
	for _, n := range entries {
		newKeys[normalizeEntry(n)] = true
	}
	info := batchInfo{original: original}
	for _, e := range entries {
		if !oldKeys[normalizeEntry(e)] {
			info.added = append(info.added, e)
		}
	}
	for _, o := range original {
		if !newKeys[normalizeEntry(o)] {
			info.removed = append(info.removed, o)
		}
	}
	commonOld := make([]string, 0)
	seenOld := make(map[string]bool)
	for _, o := range original {
		key := normalizeEntry(o)
		if newKeys[key] && !seenOld[key] {
			commonOld = append(commonOld, key)
			seenOld[key] = true
		}
	}
	commonNew := make([]string, 0)
	seenNew := make(map[string]bool)
	for _, n := range entries {
		key := normalizeEntry(n)
		if oldKeys[key] && !seenNew[key] {
			commonNew = append(commonNew, key)
			seenNew[key] = true
		}
	}
	info.reordered = strings.Join(commonOld, "\x00") != strings.Join(commonNew, "\x00")
	oldNorm := make([]string, len(original))
	for i, o := range original {
		oldNorm[i] = normalizeEntry(o)
	}
	newNorm := make([]string, len(entries))
	for i, n := range entries {
		newNorm[i] = normalizeEntry(n)
	}
	info.hasChanges = strings.Join(oldNorm, "\x00") != strings.Join(newNorm, "\x00")
	seen := make(map[string]bool)
	for _, n := range entries {
		k := normalizeEntry(n)
		if seen[k] {
			info.duplicates = append(info.duplicates, n)
		}
		seen[k] = true
	}
	return info
}

func (a *App) openBatch(preferred string) {
	if a.batch != nil {
		return
	}
	targets := a.batchTargets(a.scope)
	if len(targets) == 0 {
		a.toast("当前范围没有可批量编辑的多值变量")
		return
	}
	target := targets[0]
	for _, t := range targets {
		if preferred != "" && strings.EqualFold(t.Name, preferred) {
			target = t
			break
		}
	}
	if preferred == "" {
		for _, t := range targets {
			if strings.EqualFold(t.Name, "Path") {
				target = t
				break
			}
		}
	}
	entries := splitValue(target.Value)
	text := strings.Join(entries, "\n")
	if target.Value == "" {
		text = ""
	}
	a.batch = &batchState{
		scope:         a.scope,
		target:        target.Name,
		originalValue: target.Value,
		text:          text,
	}
	a.sheet = nil
	a.backupsOpen = false
}

func (a *App) closeBatch() {
	b := a.batch
	if b == nil {
		return
	}
	info := analyzeBatch(b.originalValue, b.text)
	if info.hasChanges {
		a.modal = &modalState{
			icon:    iconInfo,
			title:   "放弃批量编辑？",
			message: "粘贴的内容还没有应用，离开后会丢失。",
			confirm: "放弃更改",
			cancel:  "继续编辑",
			danger:  true,
			onConfirm: func() {
				a.batch = nil
			},
		}
		return
	}
	a.batch = nil
}

func (a *App) switchBatchTarget(name string) {
	b := a.batch
	if b == nil || name == b.target {
		return
	}
	target, ok := a.findVariable(b.scope, name)
	if !ok {
		return
	}
	b.target = target.Name
	b.originalValue = target.Value
	b.text = strings.Join(splitValue(target.Value), "\n")
	if target.Value == "" {
		b.text = ""
	}
}

func (a *App) batchView(c *ui.Context) {
	b := a.batch
	if b == nil {
		return
	}
	p := a.pal
	info := analyzeBatch(b.originalValue, b.text)
	entries, ignored := batchParse(b.text)
	targets := a.batchTargets(b.scope)
	names := make([]string, 0, len(targets))
	for _, t := range targets {
		names = append(names, t.Name)
	}
	validTarget := false
	for _, n := range names {
		if n == b.target {
			validTarget = true
			break
		}
	}
	if !validTarget && len(names) > 0 {
		a.switchBatchTarget(names[0])
		b = a.batch
		info = analyzeBatch(b.originalValue, b.text)
		entries, ignored = batchParse(b.text)
	}

	ui.Column(c).Grow(1).FillWidth().FillHeight().Background(p.window).Children(func() {
		// Header
		ui.Row(c).Height(52).FillWidth().Padding(0, 14, 0, 11).Background(p.sidebar).
			BorderWidth(0, 0, 1, 0).BorderColor(p.line).AlignItems(ui.Center).Gap(8).Children(func() {
			a.iconButton(c, iconArrowLeft, "返回变量列表（Esc）", 25, func() { a.closeBatch() })
			ui.Text(c, "批量编辑值").FontSize(13.5).FontWeight(650).TextColor(p.text)
			a.scopePill(c, b.scope)
			ui.Box(c).Width(4)
			ui.Text(c, "变量").FontSize(9).TextColor(p.text3)
			oldTarget := b.target
			sel := ui.Select(c, &b.target, names).Width(170).Label("目标变量")
			if sel.Changed() && b.target != oldTarget {
				next := b.target
				b.target = oldTarget
				if info.hasChanges {
					a.modal = &modalState{
						icon:    iconInfo,
						title:   "切换变量？",
						message: "切换后，当前粘贴的内容不会保留。",
						confirm: "放弃并切换",
						cancel:  "继续编辑",
						danger:  true,
						onConfirm: func() {
							a.switchBatchTarget(next)
						},
					}
				} else {
					a.switchBatchTarget(next)
				}
			}
			w, _ := c.Size()
			if w > 1020 {
				ui.Row(c).MarginStart(5).Gap(5).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconInfo).Size(11, 11).TextColor(p.text3)
					ui.Text(c, "直接粘贴完整值 · 支持 ; 与换行分隔").FontSize(9.5).TextColor(p.text3).SingleLine()
				})
			}
			ui.Spacer(c).Grow(1)
			a.button(c, "取消", nil, buttonSecondary, true, func() { a.closeBatch() })
			label := "应用更改"
			if info.hasChanges && len(entries) == 0 && len(info.original) > 0 {
				label = "清空变量"
			}
			a.button(c, label, nil, buttonPrimary, info.hasChanges, func() { a.applyBatch(c) })
		})

		// Body
		ui.Row(c).Grow(1).FillWidth().AlignItems(ui.Stretch).Children(func() {
			ui.Column(c).Grow(1.12).Basis(0).FillHeight().Background(p.surface).Children(func() {
				ui.Row(c).Height(38).FillWidth().Padding(0, 13).BorderWidth(0, 0, 1, 0).BorderColor(p.lineSoft).
					AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Text(c, "粘贴完整值").FontSize(10.5).FontWeight(600).TextColor(p.text2)
					ui.Textf(c, "当前 %d 个条目", len(splitValue(b.originalValue))).FontSize(9.5).TextColor(p.text3)
					ui.Spacer(c).Grow(1)
					a.quietButton(c, "粘贴", iconPaste, true, func() {
						text := c.ReadClipboard()
						if strings.TrimSpace(text) == "" {
							a.toast("剪贴板为空")
							return
						}
						b.text = text
					})
					a.quietButton(c, "整理为多行", iconList, true, func() {
						b.text = strings.Join(entries, "\n")
					})
				})
				ui.Box(c).Grow(1).FillWidth().Padding(11, 13).Children(func() {
					ui.TextArea(c, &b.text).Grow(1).FillWidth().Font(monoFont).FontSize(12).
						Placeholder("C:\\Program Files\\Tool\\bin\nC:\\Users\\Ada\\.local\\bin\nC:\\Windows\\System32").
						Label("粘贴变量值").AutoFocus()
				})
				ui.Row(c).Height(33).FillWidth().Padding(0, 13).BorderWidth(1, 0, 0, 0).BorderColor(p.lineSoft).
					AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconShieldDone).Size(11, 11).TextColor(p.success)
					ui.Text(c, "应用前自动创建备份").FontSize(9.5).TextColor(p.text3).Grow(1)
					meta := fmt.Sprintf("%d 个条目", len(entries))
					if ignored > 0 {
						meta += fmt.Sprintf(" · 忽略 %d 个空条目", ignored)
					}
					ui.Text(c, meta).FontSize(9.5).TextColor(p.text3)
				})
			})
			a.divider(c, false, p.line)
			ui.Column(c).Grow(0.88).Basis(0).FillHeight().Background(p.surface2).Children(func() {
				ui.Row(c).Height(38).FillWidth().Padding(0, 13).BorderWidth(0, 0, 1, 0).BorderColor(p.lineSoft).
					AlignItems(ui.Center).Gap(7).Children(func() {
					ui.Text(c, "解析结果").FontSize(10.5).FontWeight(600).TextColor(p.text2)
					a.chip(c, fmt.Sprint(len(entries)), p.text3, p.surface3, 9)
					ui.Spacer(c).Grow(1)
					ui.Textf(c, "%s · %d → %d 项", b.target, len(info.original), len(entries)).FontSize(9).TextColor(p.text3).SingleLine()
				})
				ui.Scroll(c).Grow(1).Padding(7, 9, 14, 9).Children(func() {
					a.batchStats(c, info)
					if len(entries) == 0 && len(info.original) > 0 {
						a.batchCallout(c, iconAlert, p.dangerSoft, p.danger, "应用后此变量的值为空。")
					} else if len(entries) == 0 {
						a.batchCallout(c, iconInfo, p.surface2, p.text3, "粘贴内容后，这里会显示解析出的条目。")
					}
					if info.reordered {
						a.batchCallout(c, iconInfo, p.warnSoft, p.warn, "条目顺序已调整，PATH 会按新顺序查找程序。")
					}
					if len(info.duplicates) > 0 {
						uniq := uniqueStrings(info.duplicates)
						a.batchCallout(c, iconAlert, p.warnSoft, p.warn,
							fmt.Sprintf("%d 个重复条目 · %s。重复项通常不会再次生效。", len(uniq), strings.Join(firstN(uniq, 2), "、")))
					}
					if len(entries) > 0 {
						ui.Textf(c, "新值条目  %d", len(entries)).FontSize(8.5).FontWeight(600).
							LetterSpacing(0.5).TextColor(p.text3).Padding(3, 4)
						for i, e := range entries {
							isNew := !containsFold(info.original, e)
							a.batchEntry(c, i+1, e, isNew, false)
						}
					}
					if len(info.removed) > 0 {
						ui.Textf(c, "将移除  %d", len(info.removed)).FontSize(8.5).FontWeight(600).
							LetterSpacing(0.5).TextColor(p.text3).Padding(8, 4, 3, 4)
						for i, e := range info.removed {
							a.batchEntry(c, i+1, e, false, true)
						}
					}
				})
			})
		})
	})
	c.OnShortcut(0, ui.KeyEscape, func() { a.closeBatch() })
	c.OnShortcut(ui.Cmd, ui.KeyEnter, func() { a.applyBatch(c) })
	c.OnShortcut(ui.Cmd, ui.KeyS, func() { a.applyBatch(c) })
}

func (a *App) batchStats(c *ui.Context, info batchInfo) {
	p := a.pal
	ui.Row(c).FillWidth().Gap(4).Padding(2, 4, 7, 4).Children(func() {
		if len(info.added) > 0 {
			a.statChip(c, fmt.Sprintf("+%d 新增", len(info.added)), iconPlus, p.success, p.successSoft)
		}
		if len(info.removed) > 0 {
			a.statChip(c, fmt.Sprintf("−%d 移除", len(info.removed)), iconMinus, p.danger, p.dangerSoft)
		}
		if info.reordered {
			a.statChip(c, "顺序调整", iconUnfold, p.warn, p.warnSoft)
		}
		if !info.hasChanges {
			a.statChip(c, "与当前值一致", iconCheck, p.success, p.successSoft)
		}
	})
}

func (a *App) statChip(c *ui.Context, label string, ic *ui.SVG, fg, bg ui.Color) {
	ui.Row(c).Padding(3, 5).Radius(4).Background(bg).Gap(3).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, ic).Size(9, 9).TextColor(fg)
		ui.Text(c, label).FontSize(9).FontWeight(600).TextColor(fg)
	})
}

func (a *App) batchCallout(c *ui.Context, ic *ui.SVG, bg, fg ui.Color, text string) {
	ui.Row(c).FillWidth().Margin(0, 4, 7, 4).Padding(6, 8).Radius(7).Background(bg).Gap(6).AlignItems(ui.Start).Children(func() {
		ui.Icon(c, ic).Size(10, 10).TextColor(fg)
		ui.Text(c, text).FontSize(9.5).TextColor(fg).MaxLines(4)
	})
}

func (a *App) batchEntry(c *ui.Context, index int, text string, added, removed bool) {
	p := a.pal
	bg := p.surface
	border := p.line
	fg := p.text2
	mark := "·"
	if added {
		bg = p.successSoft
		border = p.successSoft
		fg = p.success
		mark = "+"
	}
	if removed {
		bg = p.dangerSoft
		border = p.dangerSoft
		fg = p.danger
		mark = "−"
	}
	ui.Row(c).FillWidth().Margin(0, 0, 3, 0).Padding(5, 6).Radius(6).Background(bg).Border(1, border).
		AlignItems(ui.Start).Gap(4).Children(func() {
		ui.Textf(c, "%02d", index).Font(monoFont).FontSize(8.5).TextColor(p.text3).Width(24).TextAlign(ui.End)
		ui.Text(c, mark).Font(monoFont).FontSize(11).TextColor(fg).Width(13).TextAlign(ui.Center)
		ui.Text(c, text).Font(monoFont).FontSize(10.5).TextColor(fg).Grow(1).MinWidth(0)
	})
}

func (a *App) applyBatch(c *ui.Context) {
	b := a.batch
	if b == nil {
		return
	}
	entries, _ := batchParse(b.text)
	info := analyzeBatch(b.originalValue, b.text)
	if !info.hasChanges {
		return
	}
	if len(entries) == 0 && len(info.original) > 0 {
		a.modal = &modalState{
			icon:    iconAlert,
			title:   "清空变量值？",
			message: fmt.Sprintf("将把 “%s” 的值清空。应用前会自动备份，可在 5 秒内撤销。", b.target),
			confirm: "清空并应用",
			cancel:  "取消",
			danger:  true,
			onConfirm: func() {
				a.applyBatchValue(b.scope, b.target, "")
			},
		}
		return
	}
	value := strings.Join(entries, ";")
	a.applyBatchValue(b.scope, b.target, value)
}

func containsFold(list []string, s string) bool {
	for _, v := range list {
		if normalizeEntry(v) == normalizeEntry(s) {
			return true
		}
	}
	return false
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range in {
		if !seen[normalizeEntry(s)] {
			seen[normalizeEntry(s)] = true
			out = append(out, s)
		}
	}
	return out
}

func firstN(in []string, n int) []string {
	if len(in) <= n {
		return in
	}
	return in[:n]
}
