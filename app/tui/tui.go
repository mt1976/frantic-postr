package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mt1976/frantic-postr/app/core"
	"github.com/mt1976/frantic-postr/app/web"
	"github.com/rivo/tview"
)

type terminalUI struct {
	app        *tview.Application
	pages      *tview.Pages
	main       *tview.Flex
	tabBar     *tview.TextView
	status     *tview.TextView
	logs       *tview.TextView
	content    *tview.Pages
	controller *web.LocalController
	configPath string
	logger     *core.AppLogger

	state       web.WebStateResponse
	activeTab   int
	sectionKeys []string
	sections    []web.PlexSection
	collections []web.PlexCollection
}

type tabSpec struct {
	name  string
	build func() tview.Primitive
}

var tabNames = []string{"Config", "Runtime", "Posters", "Library", "Collections", "Backup & Restore"}

func Start(configPath string, logger *core.AppLogger) error {
	if logger != nil {
		consoleLogger := logger.Console
		logger.Console = nil
		defer func() {
			logger.Console = consoleLogger
		}()
	}
	ui := &terminalUI{
		app:        tview.NewApplication(),
		pages:      tview.NewPages(),
		content:    tview.NewPages(),
		configPath: configPath,
		logger:     logger,
		controller: web.NewLocalController(configPath, logger),
	}
	ui.app.SetRoot(ui.pages, true)
	ui.pages.AddPage("splash", ui.buildSplash(), true, true)
	ui.app.SetInputCapture(ui.captureKeys)
	return ui.app.Run()
}

func (ui *terminalUI) buildSplash() tview.Primitive {
	header := `
  ========================================
          frantic-postr terminal UI
  ========================================

  Poster generation. Library cleanup.
  Collection maintenance. Backup workflows.

  Press Enter to open the control room.
  Press Esc at any time to quit.
`
	view := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText("[#8bd5ca]" + header)
	view.SetBorder(true).SetTitle(" Splash ")
	view.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			ui.showMain()
		}
	})
	return tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(view, 14, 1, true).
		AddItem(nil, 0, 1, false)
}

func (ui *terminalUI) showMain() {
	ui.main = tview.NewFlex().SetDirection(tview.FlexRow)
	ui.tabBar = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	ui.status = tview.NewTextView().SetDynamicColors(true).SetText("Loading state...")
	ui.logs = tview.NewTextView().SetDynamicColors(true).SetScrollable(true).SetChangedFunc(func() {
		ui.app.Draw()
	})
	ui.logs.SetBorder(true).SetTitle(" Operation Log ")
	ui.content.SetBorder(true)
	ui.main.AddItem(ui.tabBar, 1, 0, false)
	ui.main.AddItem(ui.status, 1, 0, false)
	ui.main.AddItem(ui.content, 0, 7, true)
	ui.main.AddItem(ui.logs, 0, 3, false)
	ui.pages.RemovePage("splash")
	ui.pages.AddPage("main", ui.main, true, true)
	ui.rebuildTabs()
	ui.refreshStateAsync()
}

func (ui *terminalUI) captureKeys(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEsc:
		ui.app.Stop()
		return nil
	case tcell.KeyF1, tcell.KeyF2, tcell.KeyF3, tcell.KeyF4, tcell.KeyF5, tcell.KeyF6:
		ui.setTab(int(event.Key() - tcell.KeyF1))
		return nil
	case tcell.KeyCtrlR:
		ui.refreshStateAsync()
		return nil
	case tcell.KeyCtrlS:
		ui.stopAction()
		return nil
	}
	if ui.pages.HasPage("splash") && event.Key() == tcell.KeyEnter {
		ui.showMain()
		return nil
	}
	return event
}

func (ui *terminalUI) rebuildTabs() {
	tabs := []tabSpec{
		{name: "config", build: ui.buildConfigTab},
		{name: "runtime", build: ui.buildRuntimeTab},
		{name: "posters", build: ui.buildPostersTab},
		{name: "library", build: ui.buildLibraryTab},
		{name: "collections", build: ui.buildCollectionsTab},
		{name: "backup", build: ui.buildBackupTab},
	}
	for _, tab := range tabs {
		ui.content.RemovePage(tab.name)
		ui.content.AddPage(tab.name, tab.build(), true, false)
	}
	ui.setTab(ui.activeTab)
	ui.updateStatus("Ready. F1-F6 switch tabs. Ctrl-R refreshes state. Ctrl-S stops a running action. Esc quits.")
}

func (ui *terminalUI) setTab(index int) {
	if index < 0 || index >= len(tabNames) || ui.content == nil {
		return
	}
	ui.activeTab = index
	names := []string{"config", "runtime", "posters", "library", "collections", "backup"}
	for _, name := range names {
		ui.content.HidePage(name)
	}
	ui.content.ShowPage(names[index])
	parts := make([]string, 0, len(tabNames))
	for i, name := range tabNames {
		label := fmt.Sprintf(" F%d %s ", i+1, name)
		if i == index {
			parts = append(parts, "[black:#8bd5ca:b]"+label+"[-:-:-]")
		} else {
			parts = append(parts, "[#c7d5e0]"+label+"[-]")
		}
	}
	ui.tabBar.SetText(strings.Join(parts, " "))
}

func (ui *terminalUI) refreshStateAsync() {
	ui.updateStatus("Refreshing config, backups, and Plex libraries...")
	go func() {
		state, err := ui.controller.State()
		ui.app.QueueUpdateDraw(func() {
			if err != nil {
				ui.updateStatus("Refresh failed: " + err.Error())
				ui.appendLog("[red]state refresh failed:[-] " + err.Error())
				return
			}
			ui.state = state
			ui.sections = state.Sections
			ui.sectionKeys = make([]string, 0, len(state.Sections))
			for _, section := range state.Sections {
				ui.sectionKeys = append(ui.sectionKeys, section.Key)
			}
			ui.rebuildTabs()
			summary := fmt.Sprintf("Config %s. Libraries: %d. Output: %s", validity(state.ConfigValid), len(state.Sections), blankDefault(state.OutputDir, "(not set)"))
			if state.SectionsError != "" {
				summary += " | Plex: " + state.SectionsError
			}
			ui.updateStatus(summary)
		})
	}()
}

func (ui *terminalUI) buildConfigTab() tview.Primitive {
	cfg := ui.state
	req := web.WebConfigUpdateRequest{
		BaseURL:                  cfg.Plex.BaseURL,
		Token:                    cfg.Plex.Token,
		Retries:                  cfg.Plex.Retries,
		Workers:                  cfg.Plex.Workers,
		RetryBaseMs:              cfg.Plex.RetryBaseMs,
		RetryMaxMs:               cfg.Plex.RetryMaxMs,
		OutputDir:                cfg.General.OutputDir,
		LogFile:                  cfg.General.LogFile,
		TemplateImage:            cfg.General.TemplateImage,
		TypeTemplateImage:        cfg.General.TypeTemplateImage,
		StudioTemplateImage:      cfg.General.StudioTemplateImage,
		AdminTemplateImage:       cfg.General.AdminTemplateImage,
		TypeCollectionsFile:      cfg.General.TypeCollectionsFile,
		StudioCollectionsFile:    cfg.General.StudioCollectionsFile,
		AdminCollectionsFile:     cfg.General.AdminCollectionsFile,
		PlexConfigFile:           cfg.General.PlexConfigFile,
		LabelConfigFile:          cfg.General.LabelConfigFile,
		CollectionConfigFile:     cfg.General.CollectionConfigFile,
		TranslateToEnglish:       cfg.General.TranslateToEnglish,
		TranslateEndpoint:        cfg.General.TranslateEndpoint,
		TranslateAPIKey:          cfg.General.TranslateAPIKey,
		TranslateRateLimitMinute: cfg.General.TranslateRateLimitMinute,
		CleanReplacements:        cfg.General.CleanReplacements,
		StatsExcludeWords:        cfg.General.StatsExcludeWords,
		BackupRetentionDays:      cfg.General.BackupRetentionDays,
		FontFile:                 cfg.General.FontFile,
		FontSize:                 cfg.General.FontSize,
		FontColor:                cfg.General.FontColor,
		FontShadowColor:          cfg.General.FontShadowColor,
		FontShadowOffsetX:        cfg.General.FontShadowOffsetX,
		FontShadowOffsetY:        cfg.General.FontShadowOffsetY,
		FontGlowColor:            cfg.General.FontGlowColor,
		FontGlowRadius:           cfg.General.FontGlowRadius,
		FontGlowAlpha:            cfg.General.FontGlowAlpha,
		FontYOffset:              cfg.General.FontYOffset,
	}
	form := newForm("Configuration")
	form.AddInputField("Plex URL", req.BaseURL, 60, nil, func(v string) { req.BaseURL = v })
	form.AddPasswordField("Token", req.Token, 60, '*', func(v string) { req.Token = v })
	form.AddInputField("Retries", itoa(req.Retries), 8, numberOnly, func(v string) { req.Retries = atoi(v) })
	form.AddInputField("Workers", itoa(req.Workers), 8, numberOnly, func(v string) { req.Workers = atoi(v) })
	form.AddInputField("Retry base ms", itoa(req.RetryBaseMs), 8, numberOnly, func(v string) { req.RetryBaseMs = atoi(v) })
	form.AddInputField("Retry max ms", itoa(req.RetryMaxMs), 8, numberOnly, func(v string) { req.RetryMaxMs = atoi(v) })
	form.AddInputField("Output directory", req.OutputDir, 60, nil, func(v string) { req.OutputDir = v })
	form.AddInputField("Log file", req.LogFile, 60, nil, func(v string) { req.LogFile = v })
	form.AddInputField("Template image", req.TemplateImage, 60, nil, func(v string) { req.TemplateImage = v })
	form.AddInputField("Type template", req.TypeTemplateImage, 60, nil, func(v string) { req.TypeTemplateImage = v })
	form.AddInputField("Studio template", req.StudioTemplateImage, 60, nil, func(v string) { req.StudioTemplateImage = v })
	form.AddInputField("Admin template", req.AdminTemplateImage, 60, nil, func(v string) { req.AdminTemplateImage = v })
	form.AddInputField("Font file", req.FontFile, 60, nil, func(v string) { req.FontFile = v })
	form.AddInputField("Font size", fmt.Sprintf("%.1f", req.FontSize), 8, nil, func(v string) { req.FontSize = atof(v) })
	form.AddInputField("Font color", req.FontColor, 16, nil, func(v string) { req.FontColor = v })
	form.AddInputField("Shadow color", req.FontShadowColor, 16, nil, func(v string) { req.FontShadowColor = v })
	form.AddInputField("Glow color", req.FontGlowColor, 16, nil, func(v string) { req.FontGlowColor = v })
	form.AddCheckbox("Translate to English", req.TranslateToEnglish, func(v bool) { req.TranslateToEnglish = v })
	form.AddInputField("Translate endpoint", req.TranslateEndpoint, 60, nil, func(v string) { req.TranslateEndpoint = v })
	form.AddInputField("Stats exclude words", req.StatsExcludeWords, 60, nil, func(v string) { req.StatsExcludeWords = v })
	form.AddInputField("Backup retention days", itoa(req.BackupRetentionDays), 8, numberOnly, func(v string) { req.BackupRetentionDays = atoi(v) })
	form.AddButton("Save config", func() { ui.saveConfig(req) })
	form.AddButton("Test Plex", func() { ui.testPlex(req) })
	form.AddButton("Refresh", ui.refreshStateAsync)
	return wrap(form)
}

func (ui *terminalUI) buildRuntimeTab() tview.Primitive {
	view := tview.NewTextView().SetDynamicColors(true).SetScrollable(true)
	view.SetBorder(true).SetTitle(" Runtime ")
	state := ui.state
	var b strings.Builder
	fmt.Fprintf(&b, "[#8bd5ca]%s[-] %s\n\n", blankDefault(state.AppName, "frantic-postr"), blankDefault(state.Version, "development"))
	fmt.Fprintf(&b, "Mode: Terminal UI\nConfig: %s\nGo: %s\nStarted: %s\nOutput: %s\nLog file: %s\nConfig valid: %s\n\n", state.ConfigPath, state.GoVersion, state.StartedAt, state.OutputDir, state.LogFile, validity(state.ConfigValid))
	if state.ConfigError != "" {
		fmt.Fprintf(&b, "[red]Config error:[-] %s\n\n", state.ConfigError)
	}
	if state.SectionsError != "" {
		fmt.Fprintf(&b, "[yellow]Plex notice:[-] %s\n\n", state.SectionsError)
	}
	fmt.Fprintf(&b, "[#8bd5ca]Libraries (%d)[-]\n", len(state.Sections))
	for _, section := range state.Sections {
		fmt.Fprintf(&b, "  %s  key=%s  type=%s\n", section.Title, section.Key, section.Type)
	}
	fmt.Fprintf(&b, "\n[#8bd5ca]Backups (%d)[-]\n", len(state.Backups))
	for _, backup := range state.Backups {
		fmt.Fprintf(&b, "  %s  %s\n", backup.Name, backup.Label)
	}
	view.SetText(b.String())
	return wrap(view)
}

func (ui *terminalUI) buildPostersTab() tview.Primitive {
	var upload, missingOnly, labelTypes, trail bool
	selected := map[string]bool{}
	form := newForm("Posters")
	for _, section := range ui.sections {
		key := section.Key
		form.AddCheckbox(section.Title+" ("+section.Type+")", false, func(v bool) { selected[key] = v })
	}
	form.AddCheckbox("Upload posters to Plex", false, func(v bool) { upload = v })
	form.AddCheckbox("Missing posters only", false, func(v bool) { missingOnly = v })
	form.AddCheckbox("Label type collection items", false, func(v bool) { labelTypes = v })
	form.AddCheckbox("Trail mode", false, func(v bool) { trail = v })
	form.AddButton("Generate posters", func() {
		keys := []string{}
		for _, section := range ui.sections {
			if selected[section.Key] {
				keys = append(keys, section.Key)
			}
		}
		ui.runAction("gen-posters", web.WebActionRequest{SectionKeys: keys, UploadPosters: upload, MissingPostersOnly: missingOnly, LabelTypes: labelTypes, Trail: trail})
	})
	return wrap(form)
}

func (ui *terminalUI) buildLibraryTab() tview.Primitive {
	selectedKey := firstSectionKey(ui.sections)
	find, add, categories, cloneName := "", "", "", ""
	updateCategory, onlyCategory, translateDuringClean, trail := false, false, false, false
	form := newForm("Library")
	form.AddDropDown("Library", ui.sectionLabels(), 0, func(_ string, idx int) { selectedKey = ui.sectionKeyAt(idx) })
	form.AddCheckbox("Translate during clean", false, func(v bool) { translateDuringClean = v })
	form.AddCheckbox("Trail mode", false, func(v bool) { trail = v })
	form.AddInputField("Label find", find, 40, nil, func(v string) { find = v })
	form.AddInputField("Labels", add, 40, nil, func(v string) { add = v })
	form.AddInputField("Categories", categories, 40, nil, func(v string) { categories = v })
	form.AddCheckbox("Update categories", false, func(v bool) { updateCategory = v })
	form.AddCheckbox("Only categories", false, func(v bool) { onlyCategory = v })
	form.AddInputField("Clone name", cloneName, 40, nil, func(v string) { cloneName = v })
	form.AddButton("Clean titles", func() {
		ui.runAction("clean", web.WebActionRequest{SectionKey: selectedKey, Translate: translateDuringClean, Trail: trail})
	})
	form.AddButton("Translate titles", func() { ui.runAction("translate", web.WebActionRequest{SectionKey: selectedKey, Trail: trail}) })
	form.AddButton("Stats report", func() { ui.runAction("stats", web.WebActionRequest{SectionKey: selectedKey, Trail: trail}) })
	form.AddButton("Apply labels", func() {
		ui.runAction("label", web.WebActionRequest{SectionKey: selectedKey, Find: find, Add: add, Categories: categories, UpdateCategory: updateCategory, OnlyCategory: onlyCategory, Trail: trail})
	})
	form.AddButton("Clone library", func() {
		ui.runAction("clone", web.WebActionRequest{SectionKey: selectedKey, CloneName: cloneName, Trail: trail})
	})
	return wrap(form)
}

func (ui *terminalUI) buildCollectionsTab() tview.Primitive {
	selectedKey := firstSectionKey(ui.sections)
	collectionKey, collFile := "", ui.state.Defaults.ImportPath
	trail := false
	form := newForm("Collections")
	form.AddDropDown("Library", ui.sectionLabels(), 0, func(_ string, idx int) {
		selectedKey = ui.sectionKeyAt(idx)
	})
	form.AddInputField("Collection file", collFile, 60, nil, func(v string) { collFile = v })
	form.AddInputField("Path clean collection key", collectionKey, 30, nil, func(v string) { collectionKey = v })
	form.AddCheckbox("Trail mode", false, func(v bool) { trail = v })
	form.AddButton("Load collection keys", func() { ui.loadCollections(selectedKey) })
	form.AddButton("Export", func() {
		ui.runAction("coll-export", web.WebActionRequest{SectionKey: selectedKey, CollFile: collFile, Trail: trail})
	})
	form.AddButton("Import", func() {
		ui.runAction("coll-import", web.WebActionRequest{SectionKey: selectedKey, CollFile: collFile, Trail: trail})
	})
	form.AddButton("Inject", func() { ui.runAction("coll-inject", web.WebActionRequest{SectionKey: selectedKey, Trail: trail}) })
	form.AddButton("Duplicates", func() { ui.runAction("coll-dupes", web.WebActionRequest{SectionKey: selectedKey, Trail: trail}) })
	form.AddButton("Delete non-smart", func() {
		ui.runAction("coll-delete-non-smart", web.WebActionRequest{SectionKey: selectedKey, Trail: trail})
	})
	form.AddButton("Path clean", func() {
		ui.runAction("coll-path-clean", web.WebActionRequest{SectionKey: selectedKey, CollectionKey: collectionKey, Trail: trail})
	})
	return wrap(form)
}

func (ui *terminalUI) buildBackupTab() tview.Primitive {
	restoreFile := ""
	trail := false
	form := newForm("Backup & Restore")
	form.AddInputField("Restore file filter", restoreFile, 60, nil, func(v string) { restoreFile = v })
	form.AddCheckbox("Trail mode for restore", false, func(v bool) { trail = v })
	form.AddButton("Create backup", func() { ui.runAction("backup", web.WebActionRequest{}) })
	form.AddButton("Restore", func() { ui.runAction("restore", web.WebActionRequest{RestoreFile: restoreFile, Trail: trail}) })
	form.AddButton("Rollback last restore", func() { ui.runAction("rollback", web.WebActionRequest{}) })
	return wrap(form)
}

func (ui *terminalUI) saveConfig(req web.WebConfigUpdateRequest) {
	ui.updateStatus("Saving configuration...")
	go func() {
		resp := ui.controller.SaveConfig(req)
		ui.handleActionResponse("save config", resp)
		if resp.OK {
			ui.refreshStateAsync()
		}
	}()
}

func (ui *terminalUI) testPlex(req web.WebConfigUpdateRequest) {
	ui.updateStatus("Testing Plex connection...")
	go func() {
		resp := ui.controller.TestPlexConnection(req)
		ui.handleActionResponse("test Plex", resp)
	}()
}

func (ui *terminalUI) runAction(action string, req web.WebActionRequest) {
	ui.updateStatus("Running " + action + "...")
	ui.logs.SetText("")
	done := make(chan web.WebActionResponse, 1)
	go func() {
		done <- ui.controller.RunAction(action, req)
	}()
	go ui.pollAction(done, action)
}

func (ui *terminalUI) pollAction(done <-chan web.WebActionResponse, action string) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case resp := <-done:
			ui.handleActionResponse(action, resp)
			return
		case <-ticker.C:
			status := ui.controller.Status()
			ui.app.QueueUpdateDraw(func() {
				message := "Running " + blankDefault(status.Action, action)
				if status.Progress != nil {
					message = fmt.Sprintf("%s: %s %d/%d (%d%%)", message, status.Progress.Label, status.Progress.Current, status.Progress.Total, status.Progress.Percent)
				}
				if status.StopAsked {
					message += " | stop requested"
				}
				ui.updateStatus(message)
				if status.Logs != "" {
					ui.logs.SetText(status.Logs)
					ui.logs.ScrollToEnd()
				}
			})
		}
	}
}

func (ui *terminalUI) handleActionResponse(label string, resp web.WebActionResponse) {
	ui.app.QueueUpdateDraw(func() {
		if resp.Logs != "" {
			ui.logs.SetText(resp.Logs)
			ui.logs.ScrollToEnd()
		}
		if resp.OK {
			ui.updateStatus(blankDefault(resp.Message, label+" complete."))
			return
		}
		ui.updateStatus(label + " failed: " + resp.Error)
		ui.appendLog("[red]" + label + " failed:[-] " + resp.Error)
	})
}

func (ui *terminalUI) loadCollections(sectionKey string) {
	ui.updateStatus("Loading collections...")
	go func() {
		collections, err := ui.controller.Collections(sectionKey)
		ui.app.QueueUpdateDraw(func() {
			if err != nil {
				ui.updateStatus("Collection load failed: " + err.Error())
				return
			}
			ui.collections = collections
			lines := make([]string, 0, len(collections)+1)
			lines = append(lines, fmt.Sprintf("Collections for section %s:", sectionKey))
			for _, collection := range collections {
				lines = append(lines, fmt.Sprintf("  %s  key=%s", collection.Title, collection.RatingKey))
			}
			ui.appendLog(strings.Join(lines, "\n"))
			ui.updateStatus(fmt.Sprintf("Loaded %d collections. Copy a key into the Path clean collection key field if needed.", len(collections)))
		})
	}()
}

func (ui *terminalUI) stopAction() {
	resp := ui.controller.Stop()
	if resp.OK {
		ui.updateStatus(resp.Message)
		return
	}
	ui.updateStatus(resp.Error)
}

func (ui *terminalUI) sectionLabels() []string {
	if len(ui.sections) == 0 {
		return []string{"No libraries loaded"}
	}
	labels := make([]string, 0, len(ui.sections))
	for _, section := range ui.sections {
		labels = append(labels, section.Title+" ("+section.Type+")")
	}
	return labels
}

func (ui *terminalUI) sectionKeyAt(index int) string {
	if index < 0 || index >= len(ui.sections) {
		return ""
	}
	return ui.sections[index].Key
}

func (ui *terminalUI) updateStatus(text string) {
	if ui.status != nil {
		ui.status.SetText("[#8bd5ca]Status:[-] " + text)
	}
}

func (ui *terminalUI) appendLog(text string) {
	if ui.logs == nil || strings.TrimSpace(text) == "" {
		return
	}
	current := ui.logs.GetText(false)
	if current != "" {
		current += "\n"
	}
	ui.logs.SetText(current + text)
	ui.logs.ScrollToEnd()
}

func newForm(title string) *tview.Form {
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(" " + title + " ")
	form.SetButtonsAlign(tview.AlignLeft)
	form.SetFieldBackgroundColor(tcell.ColorBlack)
	form.SetFieldTextColor(tcell.ColorWhite)
	form.SetLabelColor(tcell.ColorLightCyan)
	form.SetButtonBackgroundColor(tcell.ColorDarkCyan)
	return form
}

func wrap(p tview.Primitive) tview.Primitive {
	return tview.NewFlex().
		AddItem(p, 0, 1, true)
}

func firstSectionKey(sections []web.PlexSection) string {
	if len(sections) == 0 {
		return ""
	}
	return sections[0].Key
}

func validity(ok bool) string {
	if ok {
		return "valid"
	}
	return "invalid"
}

func blankDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func itoa(value int) string {
	return strconv.Itoa(value)
}

func atoi(value string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(value))
	return n
}

func atof(value string) float64 {
	n, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return n
}

func numberOnly(textToCheck string, lastChar rune) bool {
	return lastChar >= '0' && lastChar <= '9' || lastChar == 0 || strings.TrimSpace(textToCheck) == ""
}
