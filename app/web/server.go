package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mt1976/frantic-postr/app/core"
	"github.com/pelletier/go-toml/v2"
)

func startWebServer(configPath string, port int, logger *AppLogger) error {
	server := &webServer{
		configPath: configPath,
		port:       port,
		logger:     logger,
		startedAt:  time.Now(),
	}
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	logger.Successf("web UI available at http://127.0.0.1:%d (local)", port)
	logger.Successf("web UI available at http://<this-machine-ip>:%d (LAN)", port)
	logger.Infof("web help available at http://127.0.0.1:%d/help", port)
	return http.ListenAndServe(addr, server.routes())
}

func (s *webServer) routes() http.Handler {
	mux := http.NewServeMux()
	resourceDir := locateWebResourcesDir()
	if resourceDir != "" {
		mux.Handle("/res/", http.StripPrefix("/res/", http.FileServer(http.Dir(resourceDir))))
	}
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/help", s.handleHelp)
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/config/content", s.handleConfigContent)
	mux.HandleFunc("/api/action/status", s.handleActionStatus)
	mux.HandleFunc("/api/action/stop", s.handleActionStop)
	mux.HandleFunc("/api/action/output", s.handleActionOutput)
	mux.HandleFunc("/api/files/list", s.handleFileList)
	mux.HandleFunc("/api/files/upload", s.handleFileUpload)
	mux.HandleFunc("/api/config", s.handleConfigUpdate)
	mux.HandleFunc("/api/template/preview", s.handleTemplatePreview)
	mux.HandleFunc("/api/plex/test", s.handlePlexTest)
	mux.HandleFunc("/api/action/", s.handleAction)
	mux.HandleFunc("/api/sections/", s.handleSectionCollections)
	return s.withConnectionLogging(mux)
}

func locateWebResourcesDir() string {
	candidates := []string{}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, wd)
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(exe))
	}
	seen := map[string]struct{}{}
	for _, start := range candidates {
		current := start
		for current != "" {
			if _, ok := seen[current]; ok {
				break
			}
			seen[current] = struct{}{}
			resDir := filepath.Join(current, "res")
			if info, err := os.Stat(resDir); err == nil && info.IsDir() {
				return resDir
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			current = parent
		}
	}
	return ""
}

func (s *webServer) withConnectionLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, port, hostLabel := resolveRemotePeer(r)
		message := fmt.Sprintf("connection from %s port %s host %s method=%s path=%s", ip, port, hostLabel, r.Method, r.URL.Path)
		s.logger.APIf("%s", message)
		s.appendActionLog("API " + message)
		next.ServeHTTP(w, r)
	})
}

func resolveRemotePeer(r *http.Request) (string, string, string) {
	remote := strings.TrimSpace(r.RemoteAddr)
	if remote == "" {
		return "unknown", "unknown", "unknown"
	}
	host, port, err := net.SplitHostPort(remote)
	if err != nil {
		return remote, "unknown", "unknown"
	}
	ip := strings.TrimSpace(host)
	if ip == "" {
		ip = "unknown"
	}
	if strings.TrimSpace(port) == "" {
		port = "unknown"
	}
	hostLabel := "unknown"
	if parsed := net.ParseIP(ip); parsed != nil {
		switch {
		case parsed.IsLoopback():
			hostLabel = "loopback"
		case parsed.IsPrivate():
			hostLabel = "private-network"
		case parsed.IsLinkLocalUnicast():
			hostLabel = "link-local"
		case parsed.IsUnspecified():
			hostLabel = "unspecified"
		default:
			hostLabel = parsed.String()
		}
	} else {
		hostLabel = ip
	}
	return ip, port, hostLabel
}

func (s *webServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writeHTML(w, http.StatusOK, webIndexTemplate, webPageData{
		AppName:   appDisplayName,
		Version:   effectiveAppVersion(),
		GoVersion: currentGoVersion(),
		StartedAt: s.startedAt.Format(time.RFC822),
		Port:      s.port,
		HelpPath:  "/help",
		Mode:      "Web UI",
		About:     buildWebAbout(s.startedAt),
	})
}

func (s *webServer) handleHelp(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/help" {
		http.NotFound(w, r)
		return
	}
	writeHTML(w, http.StatusOK, webHelpTemplate, webPageData{AppName: appDisplayName})
}

func (s *webServer) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	response, err := s.stateResponse()
	if err != nil {
		s.logger.Errorf("web state load failed: config=%s err=%v", s.configPath, err)
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *webServer) stateResponse() (webStateResponse, error) {
	s.logger.Infof("web state requested: config=%s", s.configPath)
	displayCfg, err := loadWebDisplayConfig(s.configPath, s.logger)
	if err != nil {
		s.logger.Errorf("web state display load failed: config=%s err=%v", s.configPath, err)
		return webStateResponse{}, err
	}
	runtimeCfg, err := loadWebRuntimeConfig(s.configPath, s.logger)
	if err != nil {
		s.logger.Errorf("web state runtime load failed: config=%s err=%v", s.configPath, err)
		return webStateResponse{}, err
	}
	strictCfg, strictErr := loadConfig(s.configPath)
	sections := []plexSection{}
	sectionsErr := ""
	if strings.TrimSpace(runtimeCfg.Plex.BaseURL) != "" && strings.TrimSpace(runtimeCfg.Plex.Token) != "" {
		client := &http.Client{Timeout: 30 * time.Second}
		sections, err = fetchSections(client, runtimeCfg, s.logger)
		if err != nil {
			sectionsErr = err.Error()
		}
	} else {
		sectionsErr = "Plex URL and token are required before libraries can be loaded"
	}
	backups, err := s.backups()
	if err != nil {
		sectionsErr = joinMessages(sectionsErr, fmt.Sprintf("backup list: %v", err))
	}
	response := webStateResponse{
		AppName:    appDisplayName,
		Version:    effectiveAppVersion(),
		GoVersion:  currentGoVersion(),
		StartedAt:  s.startedAt.Format(time.RFC3339),
		ConfigPath: s.configPath,
		OutputDir:  displayCfg.OutputDir,
		LogFile:    displayCfg.LogFile,
		Plex: webPlexSettings{
			BaseURL:     displayCfg.Plex.BaseURL,
			Token:       displayCfg.Plex.Token,
			Retries:     displayCfg.Plex.Retries,
			Workers:     displayCfg.Plex.Workers,
			RetryBaseMs: displayCfg.Plex.RetryBaseMs,
			RetryMaxMs:  displayCfg.Plex.RetryMaxMs,
		},
		General: webGeneralSettings{
			TemplateImage:            displayCfg.TemplateImage,
			TypeTemplateImage:        displayCfg.TypeTemplateImage,
			StudioTemplateImage:      displayCfg.StudioTemplateImage,
			AdminTemplateImage:       displayCfg.AdminTemplateImage,
			TypeCollectionsFile:      displayCfg.TypeCollectionsFile,
			StudioCollectionsFile:    displayCfg.StudioCollectionsFile,
			AdminCollectionsFile:     displayCfg.AdminCollectionsFile,
			OutputDir:                displayCfg.OutputDir,
			LogFile:                  displayCfg.LogFile,
			PlexConfigFile:           displayCfg.PlexConfigFile,
			LabelConfigFile:          displayCfg.LabelConfigFile,
			CollectionConfigFile:     displayCfg.CollectionConfigFile,
			TranslateToEnglish:       displayCfg.Clean.TranslateToEnglish,
			TranslateEndpoint:        displayCfg.Clean.TranslateEndpoint,
			TranslateAPIKey:          displayCfg.Clean.TranslateAPIKey,
			TranslateRateLimitMinute: displayCfg.Clean.TranslateRateLimitPerMinute,
			CleanReplacements:        serializeCleanReplacements(displayCfg.Clean.Replacements),
			StatsExcludeWords:        strings.Join(displayCfg.Stats.ExcludeWords, ", "),
			BackupRetentionDays:      displayCfg.Backup.RetentionDays,
			FontFile:                 displayCfg.Font.File,
			FontSize:                 displayCfg.Font.Size,
			FontColor:                displayCfg.Font.Color,
			FontShadowColor:          displayCfg.Font.ShadowColor,
			FontShadowOffsetX:        displayCfg.Font.ShadowOffsetX,
			FontShadowOffsetY:        displayCfg.Font.ShadowOffsetY,
			FontGlowColor:            displayCfg.Font.GlowColor,
			FontGlowRadius:           displayCfg.Font.GlowRadius,
			FontGlowAlpha:            displayCfg.Font.GlowAlpha,
			FontYOffset:              displayCfg.Font.YOffset,
		},
		Sections:      sections,
		Backups:       backups,
		ConfigValid:   strictErr == nil,
		SectionsError: sectionsErr,
		Defaults: webDefaults{
			ImportPath: resolveCollectionTransferPath(runtimeCfg, "collections-export.json"),
			ExportPath: resolveCollectionExportPath(runtimeCfg, "collections-export.json", time.Now()),
		},
		About: buildWebAbout(s.startedAt),
	}
	if strictErr != nil {
		s.logger.Warningf("web state config validation failed: config=%s err=%v", s.configPath, strictErr)
		response.ConfigError = strictErr.Error()
	} else {
		response.OutputDir = strictCfg.OutputDir
		response.LogFile = strictCfg.LogFile
	}
	s.logger.Infof("web state loaded: config=%s plex_url=%q token_present=%t sections=%d config_valid=%t", s.configPath, strings.TrimSpace(response.Plex.BaseURL), strings.TrimSpace(response.Plex.Token) != "", len(response.Sections), response.ConfigValid)
	return response, nil
}

func (s *webServer) handleConfigUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request webConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if err := validateWebConfigUpdate(request); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	requestLogger, logBuffer := newBufferedLogger(s.logger)
	requestLogger.Infof("web config save requested: config=%s plex_url=%q token_present=%t output_dir=%q log_file=%q", s.configPath, strings.TrimSpace(request.BaseURL), strings.TrimSpace(request.Token) != "", strings.TrimSpace(request.OutputDir), strings.TrimSpace(request.LogFile))
	if err := saveWebConfig(s.configPath, request, requestLogger); err != nil {
		requestLogger.Errorf("web config save failed: config=%s err=%v", s.configPath, err)
		writeJSON(w, http.StatusInternalServerError, webActionResponse{OK: false, Error: err.Error(), Logs: logBuffer.String()})
		return
	}
	requestLogger.Successf("web config save complete: config=%s plex_url=%q", s.configPath, strings.TrimSpace(request.BaseURL))
	writeJSON(w, http.StatusOK, webActionResponse{OK: true, Message: "Configuration saved.", Logs: logBuffer.String()})
}

func (s *webServer) handlePlexTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request webConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if err := validateWebPlexConnectionRequest(request); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	requestLogger, logBuffer := newBufferedLogger(s.logger)
	requestLogger.Infof("plex test requested: config=%s base_url=%q token_present=%t retries=%d workers=%d", s.configPath, strings.TrimSpace(request.BaseURL), strings.TrimSpace(request.Token) != "", request.Retries, request.Workers)
	if err := s.testPlexConnection(request, requestLogger); err != nil {
		requestLogger.Errorf("plex test failed: base_url=%q err=%v", strings.TrimSpace(request.BaseURL), err)
		writeJSON(w, http.StatusBadRequest, webActionResponse{OK: false, Error: err.Error(), Logs: logBuffer.String()})
		return
	}
	requestLogger.Successf("plex test succeeded: base_url=%q", strings.TrimSpace(request.BaseURL))
	writeJSON(w, http.StatusOK, webActionResponse{OK: true, Message: "Plex connection succeeded.", Logs: logBuffer.String()})
}

func (s *webServer) handleTemplatePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request webTemplatePreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	cfg, err := loadWebRuntimeConfig(s.configPath, s.logger)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	applyTemplatePreviewRequest(&cfg, request, s.configPath)
	preview, err := generateTemplatePreview(cfg, request.TemplateKind, request.SampleText)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, webTemplatePreviewResponse{
		OK:           true,
		TemplateKind: preview.TemplateKind,
		TemplatePath: pathForConfigDisplay(s.configPath, preview.TemplatePath),
		SampleText:   preview.SampleText,
		ImageDataURL: preview.ImageDataURL,
		Width:        preview.Width,
		Height:       preview.Height,
	})
}

func applyTemplatePreviewRequest(cfg *Config, request webTemplatePreviewRequest, configPath string) {
	if cfg == nil {
		return
	}
	if value := strings.TrimSpace(request.TemplateImage); value != "" {
		cfg.TemplateImage = resolvePathRelativeToConfig(configPath, value)
	}
	if value := strings.TrimSpace(request.TypeTemplateImage); value != "" {
		cfg.TypeTemplateImage = resolvePathRelativeToConfig(configPath, value)
	}
	if value := strings.TrimSpace(request.StudioTemplateImage); value != "" {
		cfg.StudioTemplateImage = resolvePathRelativeToConfig(configPath, value)
	}
	if value := strings.TrimSpace(request.AdminTemplateImage); value != "" {
		cfg.AdminTemplateImage = resolvePathRelativeToConfig(configPath, value)
	}
	if value := strings.TrimSpace(request.FontFile); value != "" {
		cfg.Font.File = resolvePathRelativeToConfig(configPath, value)
	}
	if request.FontSize > 0 {
		cfg.Font.Size = request.FontSize
	}
	if value := strings.TrimSpace(request.FontColor); value != "" {
		cfg.Font.Color = value
	}
	if value := strings.TrimSpace(request.FontShadowColor); value != "" {
		cfg.Font.ShadowColor = value
	}
	if value := strings.TrimSpace(request.FontGlowColor); value != "" {
		cfg.Font.GlowColor = value
	}
	cfg.Font.ShadowOffsetX = request.FontShadowOffsetX
	cfg.Font.ShadowOffsetY = request.FontShadowOffsetY
	if request.FontGlowRadius >= 0 {
		cfg.Font.GlowRadius = request.FontGlowRadius
	}
	if request.FontGlowAlpha >= 0 && request.FontGlowAlpha <= 1 {
		cfg.Font.GlowAlpha = request.FontGlowAlpha
	}
	cfg.Font.YOffset = request.FontYOffset
}

func (s *webServer) handleSectionCollections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/sections/")
	if !strings.HasSuffix(path, "/collections") {
		http.NotFound(w, r)
		return
	}
	sectionKey := strings.TrimSuffix(path, "/collections")
	sectionKey = strings.Trim(sectionKey, "/")
	if sectionKey == "" {
		writeJSONError(w, http.StatusBadRequest, "section key is required")
		return
	}
	cfg, err := loadWebRuntimeConfig(s.configPath, s.logger)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	client := &http.Client{Timeout: 30 * time.Second}
	collections, err := fetchCollections(client, cfg, sectionKey, s.logger)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	type collectionPayload struct {
		RatingKey string `json:"rating_key"`
		Title     string `json:"title"`
	}
	payload := struct {
		Collections []collectionPayload `json:"collections"`
	}{Collections: make([]collectionPayload, 0, len(collections))}
	for _, collection := range collections {
		payload.Collections = append(payload.Collections, collectionPayload{RatingKey: collection.RatingKey, Title: collection.Title})
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *webServer) handleActionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, s.snapshotActionRuntime())
}

func (s *webServer) handleActionStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.actionMu.Lock()
	running := s.actionRun.running
	if running {
		s.actionRun.stopAsked = true
	}
	stop := s.actionStop
	s.actionMu.Unlock()
	if !running {
		writeJSON(w, http.StatusConflict, webActionResponse{OK: false, Error: "No active operation to stop."})
		return
	}
	if stop != nil {
		stop()
	}
	writeJSON(w, http.StatusOK, webActionResponse{OK: true, Message: "Stop requested. The active operation will halt shortly."})
}

func (s *webServer) handleActionOutput(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.actionMu.Lock()
	path := strings.TrimSpace(s.actionRun.outputFile)
	s.actionMu.Unlock()
	if path == "" {
		writeJSONError(w, http.StatusNotFound, "No output file available for download yet.")
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("output file unavailable: %v", err))
		return
	}
	if !info.Mode().IsRegular() {
		writeJSONError(w, http.StatusBadRequest, "output path is not a file")
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(path)))
	http.ServeFile(w, r, path)
}

func (s *webServer) handleFileList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	scope := strings.TrimSpace(r.URL.Query().Get("scope"))
	if scope == "" {
		writeJSONError(w, http.StatusBadRequest, "scope is required")
		return
	}
	files, defaultValue, err := s.filesForScope(scope)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, webFileListResponse{
		Scope:   scope,
		Default: defaultValue,
		Files:   files,
	})
}

func (s *webServer) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	scope := strings.TrimSpace(r.URL.Query().Get("scope"))
	if scope != "collections-import" && scope != "template-images" {
		writeJSONError(w, http.StatusBadRequest, "unsupported upload scope")
		return
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid upload body: %v", err))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "missing upload file")
		return
	}
	defer file.Close()
	baseName := filepath.Base(strings.TrimSpace(header.Filename))
	if baseName == "" {
		writeJSONError(w, http.StatusBadRequest, "upload filename is required")
		return
	}
	ext := strings.ToLower(filepath.Ext(baseName))
	if scope == "collections-import" && ext != ".json" {
		writeJSONError(w, http.StatusBadRequest, "only .json import files are supported")
		return
	}
	if scope == "template-images" && !isAllowedTemplateImageExtension(ext) {
		writeJSONError(w, http.StatusBadRequest, "only image files are supported for template uploads")
		return
	}
	cfg, err := loadWebRuntimeConfig(s.configPath, s.logger)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var targetDir string
	if scope == "collections-import" {
		defaultPath := resolveCollectionTransferPath(cfg, "collections-export.json")
		targetDir = filepath.Dir(defaultPath)
	} else {
		targetDir = resolveTemplateImageDir(cfg)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	targetPath := uniqueUploadPath(filepath.Join(targetDir, baseName))
	dst, err := os.Create(targetPath)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := dst.Close(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	files, defaultValue, err := s.filesForScope(scope)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	value := targetPath
	if scope == "template-images" {
		value = pathForConfigDisplay(s.configPath, targetPath)
	}
	writeJSON(w, http.StatusOK, struct {
		OK      bool            `json:"ok"`
		Message string          `json:"message"`
		File    string          `json:"file"`
		Value   string          `json:"value"`
		Scope   string          `json:"scope"`
		Default string          `json:"default"`
		Files   []webFileOption `json:"files"`
	}{
		OK:      true,
		Message: fmt.Sprintf("Uploaded %s", filepath.Base(targetPath)),
		File:    filepath.Base(targetPath),
		Value:   value,
		Scope:   scope,
		Default: defaultValue,
		Files:   files,
	})
}

func (s *webServer) filesForScope(scope string) ([]webFileOption, string, error) {
	cfg, err := loadWebRuntimeConfig(s.configPath, s.logger)
	if err != nil {
		return nil, "", err
	}
	switch scope {
	case "collections-import":
		defaultPath := resolveCollectionTransferPath(cfg, "collections-export.json")
		dir := filepath.Dir(defaultPath)
		files, err := listFilesForImportScope(dir)
		return files, defaultPath, err
	case "template-images":
		dir := resolveTemplateImageDir(cfg)
		files, err := listFilesForTemplateScope(s.configPath, dir)
		if err != nil {
			return nil, "", err
		}
		return files, pathForConfigDisplay(s.configPath, cfg.TemplateImage), nil
	default:
		return nil, "", fmt.Errorf("unsupported file scope: %s", scope)
	}
}

func listFilesForTemplateScope(configPath, dir string) ([]webFileOption, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []webFileOption{}, nil
		}
		return nil, err
	}
	out := make([]webFileOption, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !isAllowedTemplateImageExtension(ext) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		fullPath := filepath.Join(dir, entry.Name())
		out = append(out, webFileOption{
			Name:    entry.Name(),
			Value:   pathForConfigDisplay(configPath, fullPath),
			ModTime: info.ModTime().Format(time.RFC3339),
			Size:    info.Size(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ModTime > out[j].ModTime
	})
	return out, nil
}

func isAllowedTemplateImageExtension(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp", ".tif", ".tiff", ".avif":
		return true
	default:
		return false
	}
}

func resolveTemplateImageDir(cfg Config) string {
	candidates := []string{
		strings.TrimSpace(cfg.TemplateImage),
		strings.TrimSpace(cfg.TypeTemplateImage),
		strings.TrimSpace(cfg.StudioTemplateImage),
		strings.TrimSpace(cfg.AdminTemplateImage),
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		dir := filepath.Dir(candidate)
		if strings.TrimSpace(dir) != "" && dir != "." {
			return dir
		}
	}
	if wd, err := os.Getwd(); err == nil {
		return filepath.Join(wd, "templates")
	}
	return "templates"
}

func pathForConfigDisplay(configPath, targetPath string) string {
	trimmed := strings.TrimSpace(targetPath)
	if trimmed == "" {
		return ""
	}
	baseDir := filepath.Dir(configPath)
	rel, err := filepath.Rel(baseDir, trimmed)
	if err != nil || strings.TrimSpace(rel) == "" {
		return trimmed
	}
	if rel == "." {
		return "./"
	}
	if !strings.HasPrefix(rel, ".") {
		return "./" + rel
	}
	return rel
}

func (s *webServer) handleConfigContent(w http.ResponseWriter, r *http.Request) {
	scope := strings.TrimSpace(r.URL.Query().Get("scope"))
	if scope == "" {
		writeJSONError(w, http.StatusBadRequest, "scope is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		path, err := s.resolveConfigContentPath(scope)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		content, err := readConfigContentFile(path)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, webConfigContentResponse{Scope: scope, Path: pathForConfigDisplay(s.configPath, path), Content: content})
	case http.MethodPost:
		var request webConfigContentUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}
		request.Scope = strings.TrimSpace(request.Scope)
		if request.Scope == "" {
			request.Scope = scope
		}
		if request.Scope != scope {
			writeJSONError(w, http.StatusBadRequest, "scope mismatch")
			return
		}
		path, err := s.resolveConfigContentPath(scope)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		content := normalizeConfigContent(scope, request.Content)
		if isTomlContentScope(scope) {
			probe := map[string]any{}
			if strings.TrimSpace(content) != "" {
				if err := toml.Unmarshal([]byte(content), &probe); err != nil {
					writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid TOML: %v", err))
					return
				}
			}
		}
		if err := writeConfigContentFile(path, content); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, webActionResponse{OK: true, Message: "Config content updated."})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *webServer) resolveConfigContentPath(scope string) (string, error) {
	cfg, err := loadWebRuntimeConfig(s.configPath, s.logger)
	if err != nil {
		return "", err
	}
	switch scope {
	case "type-collections":
		if strings.TrimSpace(cfg.TypeCollectionsFile) == "" {
			return "", errors.New("type_collections_file is not configured")
		}
		return cfg.TypeCollectionsFile, nil
	case "studio-collections":
		if strings.TrimSpace(cfg.StudioCollectionsFile) == "" {
			return "", errors.New("studio_collections_file is not configured")
		}
		return cfg.StudioCollectionsFile, nil
	case "admin-collections":
		if strings.TrimSpace(cfg.AdminCollectionsFile) == "" {
			return "", errors.New("admin_collections_file is not configured")
		}
		return cfg.AdminCollectionsFile, nil
	case "label-config":
		if strings.TrimSpace(cfg.LabelConfigFile) == "" {
			return "", errors.New("label_config is not configured")
		}
		return cfg.LabelConfigFile, nil
	case "collection-config":
		if strings.TrimSpace(cfg.CollectionConfigFile) == "" {
			return "", errors.New("collection_config is not configured")
		}
		return cfg.CollectionConfigFile, nil
	default:
		return "", fmt.Errorf("unsupported config content scope: %s", scope)
	}
}

func readConfigContentFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimRight(string(content), "\n"), nil
}

func writeConfigContentFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func normalizeConfigContent(scope, content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !isCollectionListScope(scope) {
		return strings.TrimSpace(content)
	}
	lines := strings.Split(content, "\n")
	seen := map[string]struct{}{}
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, trimmed)
	}
	return strings.Join(filtered, "\n")
}

func isCollectionListScope(scope string) bool {
	switch scope {
	case "type-collections", "studio-collections", "admin-collections":
		return true
	default:
		return false
	}
}

func isTomlContentScope(scope string) bool {
	switch scope {
	case "label-config", "collection-config":
		return true
	default:
		return false
	}
}

func listFilesForImportScope(dir string) ([]webFileOption, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []webFileOption{}, nil
		}
		return nil, err
	}
	out := make([]webFileOption, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		fullPath := filepath.Join(dir, entry.Name())
		out = append(out, webFileOption{
			Name:    entry.Name(),
			Value:   fullPath,
			ModTime: info.ModTime().Format(time.RFC3339),
			Size:    info.Size(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ModTime > out[j].ModTime
	})
	return out, nil
}

func uniqueUploadPath(path string) string {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)
	dir := filepath.Dir(path)
	timestamp := time.Now().Format("20060102-150405")
	return filepath.Join(dir, fmt.Sprintf("%s-%s%s", base, timestamp, ext))
}

func (s *webServer) resetActionRuntime(action string) {
	s.actionMu.Lock()
	defer s.actionMu.Unlock()
	s.actionRun = webActionRuntime{
		action:     action,
		startedAt:  time.Now(),
		running:    true,
		ok:         false,
		canceled:   false,
		stopAsked:  false,
		outputFile: "",
	}
}

func (s *webServer) appendActionLog(line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}
	s.actionMu.Lock()
	defer s.actionMu.Unlock()
	if s.actionRun.logs.Len() > 0 {
		s.actionRun.logs.WriteByte('\n')
	}
	s.actionRun.logs.WriteString(trimmed)
}

func (s *webServer) updateActionProgress(label string, current, total int, final bool) {
	if strings.TrimSpace(label) == "" {
		label = "processing"
	}
	if total < 0 {
		total = 0
	}
	if current < 0 {
		current = 0
	}
	if total > 0 && current > total {
		current = total
	}
	percent := 0
	if total > 0 {
		percent = (current * 100) / total
		if percent > 100 {
			percent = 100
		}
	}
	s.actionMu.Lock()
	defer s.actionMu.Unlock()
	s.actionRun.hasProgress = true
	s.actionRun.progress = webActionProgressSnapshot{
		Label:     label,
		Current:   current,
		Total:     total,
		Percent:   percent,
		Final:     final,
		UpdatedAt: time.Now().Format(time.RFC3339Nano),
	}
}

func (s *webServer) completeActionRuntime(ok bool, canceled bool, message, errText, outputFile string) {
	s.actionMu.Lock()
	defer s.actionMu.Unlock()
	s.actionRun.running = false
	s.actionRun.completedAt = time.Now()
	s.actionRun.ok = ok
	s.actionRun.canceled = canceled
	s.actionRun.message = strings.TrimSpace(message)
	s.actionRun.err = strings.TrimSpace(errText)
	s.actionRun.outputFile = strings.TrimSpace(outputFile)
	s.actionStop = nil
	if s.actionRun.hasProgress && s.actionRun.progress.Total > 0 && ok {
		s.actionRun.progress.Current = s.actionRun.progress.Total
		s.actionRun.progress.Percent = 100
		s.actionRun.progress.Final = true
		s.actionRun.progress.UpdatedAt = time.Now().Format(time.RFC3339Nano)
	}
}

func (s *webServer) snapshotActionRuntime() webActionStatusResponse {
	s.actionMu.Lock()
	defer s.actionMu.Unlock()
	response := webActionStatusResponse{
		Running:    s.actionRun.running,
		Action:     s.actionRun.action,
		OK:         s.actionRun.ok,
		Canceled:   s.actionRun.canceled,
		StopAsked:  s.actionRun.stopAsked,
		Message:    s.actionRun.message,
		Error:      s.actionRun.err,
		OutputFile: filepath.Base(strings.TrimSpace(s.actionRun.outputFile)),
		Logs:       s.actionRun.logs.String(),
	}
	if strings.TrimSpace(s.actionRun.outputFile) != "" {
		response.DownloadURL = "/api/action/output"
	}
	if !s.actionRun.startedAt.IsZero() {
		response.StartedAt = s.actionRun.startedAt.Format(time.RFC3339Nano)
	}
	if !s.actionRun.completedAt.IsZero() {
		response.CompletedAt = s.actionRun.completedAt.Format(time.RFC3339Nano)
	}
	if s.actionRun.hasProgress {
		progress := s.actionRun.progress
		response.Progress = &progress
	}
	return response
}

func (s *webServer) handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	action := strings.TrimPrefix(r.URL.Path, "/api/action/")
	action = strings.TrimSpace(strings.Trim(action, "/"))
	if action == "" {
		writeJSONError(w, http.StatusBadRequest, "action name is required")
		return
	}
	var request webActionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if !s.busy.CompareAndSwap(false, true) {
		writeJSON(w, http.StatusConflict, webActionResponse{OK: false, Error: "Another operation is already running. Wait for it to finish and retry."})
		return
	}
	defer s.busy.Store(false)
	s.resetActionRuntime(action)
	actionCtx, cancel := context.WithCancel(context.Background())
	s.actionMu.Lock()
	s.actionStop = cancel
	startedAt := s.actionRun.startedAt
	s.actionMu.Unlock()
	restoreRequestContext := setRequestContext(actionCtx)
	defer restoreRequestContext()
	defer cancel()

	requestLogger, logBuffer := newBufferedLogger(s.logger)
	requestLogger.LogCallback = func(line string) {
		s.appendActionLog(line)
	}
	requestLogger.ProgressCallback = func(label string, current, total int, final bool) {
		s.updateActionProgress(label, current, total, final)
	}
	err := s.runAction(action, request, requestLogger)
	outputFile := s.detectLatestDownloadableOutput(startedAt, time.Now())
	response := webActionResponse{OK: err == nil, Logs: logBuffer.String()}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			response.OK = false
			response.Error = "Operation stopped by user."
			s.completeActionRuntime(false, true, "Operation stopped.", response.Error, outputFile)
			writeJSON(w, http.StatusConflict, response)
			return
		}
		response.Error = err.Error()
		s.completeActionRuntime(false, false, "", err.Error(), outputFile)
		writeJSON(w, http.StatusBadRequest, response)
		return
	}
	response.Message = fmt.Sprintf("%s completed.", action)
	s.completeActionRuntime(true, false, response.Message, "", outputFile)
	writeJSON(w, http.StatusOK, response)
}

func (s *webServer) runAction(action string, request webActionRequest, logger *AppLogger) error {
	client := &http.Client{Timeout: 30 * time.Second}
	switch action {
	case "backup":
		opsCfg, err := loadOpsConfig(s.configPath)
		if err != nil {
			return err
		}
		return createBackupArchive(opsCfg, s.configPath, logger)
	case "restore":
		opsCfg, err := loadOpsConfig(s.configPath)
		if err != nil {
			return err
		}
		return withTrailMode(request.Trail, func() error {
			return restoreFromBackup(opsCfg, strings.TrimSpace(request.RestoreFile), logger)
		})
	case "rollback":
		opsCfg, err := loadOpsConfig(s.configPath)
		if err != nil {
			return err
		}
		return rollbackLastRestore(opsCfg, logger)
	}

	cfg, err := loadConfig(s.configPath)
	if err != nil {
		return err
	}
	sections, err := fetchSections(client, cfg, logger)
	if err != nil {
		return err
	}

	return withTrailMode(request.Trail, func() error {
		switch action {
		case "gen-posters":
			selectedSections, err := selectSectionsByKey(sections, request.SectionKeys)
			if err != nil {
				return err
			}
			return runPosterGeneration(client, cfg, selectedSections, request.UploadPosters, request.LabelTypes, request.MissingPostersOnly, logger)
		case "clean":
			section, err := requireSectionByKey(sections, request.SectionKey)
			if err != nil {
				return err
			}
			return cleanLibraryTitlesForSection(client, cfg, section, request.Translate, logger)
		case "translate":
			section, err := requireSectionByKey(sections, request.SectionKey)
			if err != nil {
				return err
			}
			return translateLibraryTitlesForSection(client, cfg, section, logger)
		case "stats":
			section, err := requireSectionByKey(sections, request.SectionKey)
			if err != nil {
				return err
			}
			return analyzeLibraryFileNameStatsForSection(client, cfg, section, logger)
		case "label":
			section, err := requireSectionByKey(sections, request.SectionKey)
			if err != nil {
				return err
			}
			labelsToAdd, err := parseLabelList(request.Add)
			if err != nil {
				return err
			}
			categoriesToAdd, err := parseLabelList(request.Categories)
			if err != nil {
				return err
			}
			if len(categoriesToAdd) == 0 {
				categoriesToAdd = labelsToAdd
			}
			if strings.TrimSpace(request.Find) == "" {
				return errors.New("find text is required for web label runs")
			}
			if len(labelsToAdd) == 0 && !request.OnlyCategory {
				return errors.New("at least one label is required")
			}
			if len(categoriesToAdd) == 0 && request.OnlyCategory {
				return errors.New("at least one category is required when using categories-only mode")
			}
			return labelMatchingItems(client, cfg, section, []string{strings.TrimSpace(request.Find)}, labelsToAdd, categoriesToAdd, request.UpdateCategory, request.OnlyCategory, logger)
		case "clone":
			section, err := requireSectionByKey(sections, request.SectionKey)
			if err != nil {
				return err
			}
			return cloneLibraryFromSection(client, cfg, sections, section, strings.TrimSpace(request.CloneName), logger)
		case "coll-export":
			section, err := requireSectionByKey(sections, request.SectionKey)
			if err != nil {
				return err
			}
			exportPath := resolveCollectionExportPath(cfg, blankDefault(request.CollFile, "collections-export.json"), time.Now())
			return exportCollectionsForSection(client, cfg, section, exportPath, logger)
		case "coll-import":
			section, err := requireSectionByKey(sections, request.SectionKey)
			if err != nil {
				return err
			}
			importPath := resolveCollectionTransferPath(cfg, blankDefault(request.CollFile, "collections-export.json"))
			return importCollectionsForSection(client, cfg, section, importPath, logger)
		case "coll-inject":
			section, err := requireSectionByKey(sections, request.SectionKey)
			if err != nil {
				return err
			}
			return injectCollectionsForSection(client, cfg, section, logger)
		case "coll-dupes":
			section, err := requireSectionByKey(sections, request.SectionKey)
			if err != nil {
				return err
			}
			return reportDuplicateCollectionsForSection(client, cfg, section, logger)
		case "coll-delete-non-smart":
			section, err := requireSectionByKey(sections, request.SectionKey)
			if err != nil {
				return err
			}
			return deleteNonSmartCollectionsForSection(client, cfg, section, logger)
		case "coll-path-clean":
			section, err := requireSectionByKey(sections, request.SectionKey)
			if err != nil {
				return err
			}
			return pathCleanCollectionTitlesForSelection(client, cfg, section, strings.TrimSpace(request.CollectionKey), logger)
		default:
			return fmt.Errorf("unsupported action: %s", action)
		}
	})
}

func runPosterGeneration(client *http.Client, cfg Config, selectedSections []plexSection, uploadPosters bool, labelTypeCollectionItems bool, missingPostersOnly bool, logger *AppLogger) error {
	if len(selectedSections) == 0 {
		return errors.New("select at least one library")
	}
	var wg sync.WaitGroup
	errCh := make(chan error, len(selectedSections))
	for _, section := range selectedSections {
		section := section
		logger.Printf("selected library: %s (%s)", section.Title, section.Key)
		wg.Add(1)
		go func() {
			defer wg.Done()
			collections, err := fetchCollections(client, cfg, section.Key, logger)
			if err != nil {
				errCh <- fmt.Errorf("fetch collections for %s: %w", section.Title, err)
				return
			}
			logger.Printf("collections fetched: library=%s count=%d", section.Title, len(collections))
			if err := processCollections(client, cfg, section.Title, collections, uploadPosters, labelTypeCollectionItems, missingPostersOnly, logger); err != nil {
				errCh <- fmt.Errorf("process %s: %w", section.Title, err)
				return
			}
		}()
	}
	wg.Wait()
	close(errCh)
	if len(errCh) == 0 {
		return nil
	}
	errs := make([]string, 0, len(errCh))
	for err := range errCh {
		errs = append(errs, err.Error())
	}
	return fmt.Errorf("processing failed: %s", strings.Join(errs, "; "))
}

func cleanLibraryTitlesForSection(client *http.Client, cfg Config, section plexSection, translateEnabled bool, logger *AppLogger) error {
	return cleanLibraryTitles(client, cfg, []plexSection{section}, translateEnabled, logger)
}

func translateLibraryTitlesForSection(client *http.Client, cfg Config, section plexSection, logger *AppLogger) error {
	return translateLibraryTitles(client, cfg, []plexSection{section}, logger)
}

func analyzeLibraryFileNameStatsForSection(client *http.Client, cfg Config, section plexSection, logger *AppLogger) error {
	return analyzeLibraryFileNameStats(client, cfg, []plexSection{section}, logger)
}

func exportCollectionsForSection(client *http.Client, cfg Config, section plexSection, exportPath string, logger *AppLogger) error {
	logger.Printf("collection export: selected library=%s (%s)", section.Title, section.Key)
	exportPath = resolveCollectionExportPathForLibrary(exportPath, section.Title, time.Now())
	collections, err := fetchCollections(client, cfg, section.Key, logger)
	if err != nil {
		return err
	}
	progress := newProgressTracker(logger, "export collections", len(collections))
	defer progress.Finish()
	transfer := collectionTransferFile{
		Version:       1,
		ExportedAtUTC: time.Now().UTC().Format(time.RFC3339),
		SourceLibrary: section,
		Collections:   make([]collectionTransferRecord, 0, len(collections)),
	}
	for _, collection := range collections {
		detail, err := fetchCollectionDetails(client, cfg, collection.RatingKey, logger)
		if err != nil {
			return fmt.Errorf("fetch details for collection %q: %w", collection.Title, err)
		}
		if detail.Title == "" {
			detail.Title = collection.Title
		}
		transfer.Collections = append(transfer.Collections, detail)
		progress.Advance()
	}
	jsonBytes, err := json.MarshalIndent(transfer, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(exportPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(exportPath, jsonBytes, 0o644); err != nil {
		return err
	}
	logger.Successf("collection export complete: file=%s count=%d", exportPath, len(transfer.Collections))
	return nil
}

func importCollectionsForSection(client *http.Client, cfg Config, targetSection plexSection, importPath string, logger *AppLogger) error {
	jsonBytes, err := os.ReadFile(importPath)
	if err != nil {
		return err
	}
	var transfer collectionTransferFile
	if err := json.Unmarshal(jsonBytes, &transfer); err != nil {
		return err
	}
	if len(transfer.Collections) == 0 {
		return errors.New("import file has no collections")
	}
	logger.Printf("collection import: target library=%s (%s)", targetSection.Title, targetSection.Key)
	targetTypeCode, err := sectionTypeToPlexTypeCode(targetSection.Type)
	if err != nil {
		return err
	}
	existing, err := fetchCollections(client, cfg, targetSection.Key, logger)
	if err != nil {
		return err
	}
	progress := newProgressTracker(logger, "import collections", len(transfer.Collections))
	defer progress.Finish()
	existingByTitle := make(map[string]struct{}, len(existing))
	for _, collection := range existing {
		existingByTitle[strings.ToLower(collection.Title)] = struct{}{}
	}
	created := 0
	skipped := 0
	for _, collection := range transfer.Collections {
		title := normalizeCollectionName(collection.Title)
		if title == "" {
			title = "untitled"
		}
		if _, ok := existingByTitle[strings.ToLower(title)]; ok {
			skipped++
			logger.Printf("collection import: skip existing title=%q", title)
			progress.Advance()
			continue
		}
		if err := createCollection(client, cfg, transfer.SourceLibrary.Key, targetSection.Key, title, targetTypeCode, collection, logger); err != nil {
			return fmt.Errorf("create collection %q: %w", title, err)
		}
		existingByTitle[strings.ToLower(title)] = struct{}{}
		created++
		progress.Advance()
	}
	logger.Successf("collection import complete: created=%d skipped=%d", created, skipped)
	return nil
}

func injectCollectionsForSection(client *http.Client, cfg Config, targetSection plexSection, logger *AppLogger) error {
	if len(cfg.Collection.Lookups) == 0 {
		return errors.New("invalid flags: -coll-inject requires one or more [[collection.lookup]] entries in config/collections.toml")
	}
	logger.Printf("collection inject: target library=%s (%s)", targetSection.Title, targetSection.Key)
	progress := newProgressTracker(logger, "inject collections", len(cfg.Collection.Lookups))
	defer progress.Finish()
	targetTypeCode, err := sectionTypeToPlexTypeCode(targetSection.Type)
	if err != nil {
		return err
	}
	existing, err := fetchCollections(client, cfg, targetSection.Key, logger)
	if err != nil {
		return err
	}
	existingByTitle := make(map[string]struct{}, len(existing))
	for _, collection := range existing {
		existingByTitle[strings.ToLower(collection.Title)] = struct{}{}
	}
	created := 0
	skipped := 0
	for i, lookup := range cfg.Collection.Lookups {
		title := normalizeCollectionName(lookup.Title)
		if title == "" {
			title = "untitled"
		}
		if _, ok := existingByTitle[strings.ToLower(title)]; ok {
			skipped++
			logger.Printf("collection inject: skip existing title=%q", title)
			progress.Advance()
			continue
		}
		content := composeCollectionContent(cfg.CollectionBaseURI, lookup.Content)
		record := collectionTransferRecord{Title: title, Smart: lookup.Smart, Content: content}
		if strings.TrimSpace(record.Content) != "" {
			record.Smart = true
		}
		if err := createCollection(client, cfg, "", targetSection.Key, title, targetTypeCode, record, logger); err != nil {
			return fmt.Errorf("create collection %q (lookup %d): %w", title, i+1, err)
		}
		existingByTitle[strings.ToLower(title)] = struct{}{}
		created++
		progress.Advance()
	}
	logger.Successf("collection inject complete: created=%d skipped=%d", created, skipped)
	return nil
}

func reportDuplicateCollectionsForSection(client *http.Client, cfg Config, section plexSection, logger *AppLogger) error {
	logger.Printf("collection dupes: selected library=%s (%s)", section.Title, section.Key)
	entries, err := fetchCollectionInventory(client, cfg, section.Key, logger)
	if err != nil {
		return err
	}
	rows, dupCount := buildDuplicateCollectionRows(entries)
	for _, row := range rows {
		logger.Successf("duplicate collection: title=%q items=%s rating_key=%s", row[0], row[2], row[1])
	}
	reportPath := uniqueCollectionReportPathForLibrary(cfg.OutputDir, "duplicate-collections", section.Title, time.Now())
	if err := writeCSVReport(reportPath, []string{"title", "rating_key", "item_count", "smart", "duplicate_group_size"}, rows); err != nil {
		return err
	}
	logger.Successf("collection dupes report complete: file=%s duplicate_groups=%d duplicate_rows=%d", reportPath, dupCount, len(rows))
	return nil
}

func deleteNonSmartCollectionsForSection(client *http.Client, cfg Config, section plexSection, logger *AppLogger) error {
	logger.Printf("collection delete non-smart: selected library=%s (%s)", section.Title, section.Key)
	entries, err := fetchCollectionInventory(client, cfg, section.Key, logger)
	if err != nil {
		return err
	}
	deletions, deleteCount, deleteFailures, err := deleteNonSmartCollectionEntries(client, cfg, entries, logger)
	if err != nil {
		return err
	}
	reportPath := uniqueCollectionReportPathForLibrary(cfg.OutputDir, "deleted-non-smart-collections", section.Title, time.Now())
	rows := make([][]string, 0, len(deletions))
	for _, row := range deletions {
		rows = append(rows, []string{row.Title, row.RatingKey, strconv.Itoa(row.ItemCount), strconv.FormatBool(row.Smart), row.Status})
	}
	if err := writeCSVReport(reportPath, []string{"title", "rating_key", "item_count", "smart", "status"}, rows); err != nil {
		return err
	}
	logger.Successf("delete non-smart collections complete: file=%s deleted=%d failed=%d skipped_smart=%d", reportPath, deleteCount, deleteFailures, len(entries)-len(deletions))
	if deleteFailures > 0 {
		return fmt.Errorf("deleted %d collections with %d failures", deleteCount, deleteFailures)
	}
	return nil
}

func pathCleanCollectionTitlesForSelection(client *http.Client, cfg Config, section plexSection, collectionKey string, logger *AppLogger) error {
	logger.Infof("path clean: selected library=%s (%s)", section.Title, section.Key)
	collections, err := fetchCollections(client, cfg, section.Key, logger)
	if err != nil {
		return err
	}
	logger.Infof("path clean: scanned collections=%d", len(collections))
	selectedCollection, err := requireCollectionByKey(collections, collectionKey)
	if err != nil {
		return err
	}
	logger.Infof("path clean: selected collection=%s (%s)", selectedCollection.Title, selectedCollection.RatingKey)
	items, err := fetchCollectionItems(client, cfg, selectedCollection.RatingKey, logger)
	if err != nil {
		return err
	}
	logger.Infof("path clean: scanned items=%d", len(items))
	progress := newProgressTracker(logger, "path clean", len(items))
	defer progress.Finish()
	updated := 0
	skipped := 0
	reportRows := make([][]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.RatingKey) == "" {
			skipped++
			logger.Warningf("path clean: skipping item with empty rating key")
			progress.Advance()
			continue
		}
		filePath := libraryItemFilePath(item)
		if strings.TrimSpace(filePath) == "" {
			skipped++
			logger.Warningf("path clean: skipping item with empty file path ratingKey=%s", item.RatingKey)
			progress.Advance()
			continue
		}
		before := strings.TrimSpace(item.Title)
		after := pathCleanTitleFromFilePath(filePath, cfg.Clean.Replacements)
		if after == "" || after == before {
			skipped++
			progress.Advance()
			continue
		}
		if err := updateLibraryItemTitle(client, cfg, item.RatingKey, after, logger); err != nil {
			return fmt.Errorf("update path-clean title ratingKey=%s: %w", item.RatingKey, err)
		}
		updated++
		reportRows = append(reportRows, []string{selectedCollection.Title, item.RatingKey, filePath, before, after})
		logger.Successf("path clean: title updated ratingKey=%s before=%q after=%q", item.RatingKey, before, after)
		progress.Advance()
	}
	reportPath := uniquePathCleanReportPathForLibrary(cfg.OutputDir, section.Title, time.Now())
	if err := writeCSVReport(reportPath, []string{"collection", "rating_key", "file_path", "title_before", "title_after"}, reportRows); err != nil {
		logger.Warningf("path clean: failed to write report: %v", err)
	} else {
		logger.Infof("path clean: report written: %s (%d rows)", reportPath, len(reportRows))
	}
	logger.Successf("path clean complete: updated=%d skipped=%d", updated, skipped)
	return nil
}

func cloneLibraryFromSection(client *http.Client, cfg Config, sections []plexSection, source plexSection, targetName string, logger *AppLogger) error {
	logger.Printf("library clone: source library=%s (%s)", source.Title, source.Key)
	if strings.TrimSpace(targetName) == "" {
		targetName = defaultCloneLibraryName(source.Title)
	}
	sourceDetail, err := fetchSectionDetail(client, cfg, source.Key, logger)
	if err != nil {
		return err
	}
	locations := extractSectionLocations(sourceDetail)
	if len(locations) == 0 {
		return fmt.Errorf("source library has no location mappings")
	}
	if err := ensureLibraryNameAvailable(sections, targetName); err != nil {
		return err
	}
	newSection, err := createLibraryFromSection(client, cfg, sourceDetail, targetName, locations, logger)
	if err != nil {
		return err
	}
	logger.Printf("library clone: created library=%s (%s)", newSection.Title, newSection.Key)
	prefs, err := fetchSectionPreferences(client, cfg, source.Key, logger)
	if err != nil {
		return err
	}
	if err := applySectionPreferences(client, cfg, newSection.Key, prefs, logger); err != nil {
		return err
	}
	logger.Successf("library clone complete: source=%s target=%s", source.Title, newSection.Title)
	return nil
}

func selectSectionsByKey(sections []plexSection, keys []string) ([]plexSection, error) {
	trimmedKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key != "" {
			trimmedKeys = append(trimmedKeys, key)
		}
	}
	if len(trimmedKeys) == 0 {
		return nil, errors.New("select at least one library")
	}
	selected := make([]plexSection, 0, len(trimmedKeys))
	for _, key := range trimmedKeys {
		section, err := requireSectionByKey(sections, key)
		if err != nil {
			return nil, err
		}
		selected = append(selected, section)
	}
	return selected, nil
}

func requireSectionByKey(sections []plexSection, key string) (plexSection, error) {
	needle := strings.TrimSpace(key)
	for _, section := range sections {
		if section.Key == needle {
			return section, nil
		}
	}
	return plexSection{}, fmt.Errorf("library not found: %s", key)
}

func requireCollectionByKey(collections []plexCollection, key string) (plexCollection, error) {
	needle := strings.TrimSpace(key)
	if needle == "" {
		return plexCollection{}, errors.New("collection key is required")
	}
	for _, collection := range collections {
		if collection.RatingKey == needle {
			return collection, nil
		}
	}
	return plexCollection{}, fmt.Errorf("collection not found: %s", key)
}

func loadWebRuntimeConfig(path string, logger *AppLogger) (Config, error) {
	cfg, err := loadOpsConfig(path)
	if err != nil {
		return cfg, err
	}
	if logger != nil {
		logger.Infof("config load: main file=%s", path)
		logger.Infof("config load: resolved plex_config=%s", blankDefault(cfg.PlexConfigFile, "<none>"))
	}
	if cfg.PlexConfigFile != "" {
		var supplemental Config
		if err := loadResolvedSupplementalConfig(cfg.PlexConfigFile, "plex_config", &supplemental); err != nil {
			if !errors.Is(err, os.ErrNotExist) && !strings.Contains(err.Error(), "no such file or directory") {
				return cfg, err
			}
			if logger != nil {
				logger.Warningf("config load: plex config missing file=%s", cfg.PlexConfigFile)
			}
		} else {
			if logger != nil {
				logger.Infof("config load: merged plex config file=%s plex_url=%q token_present=%t", cfg.PlexConfigFile, strings.TrimSpace(supplemental.Plex.BaseURL), strings.TrimSpace(supplemental.Plex.Token) != "")
			}
			mergePlexConfig(&cfg, &supplemental)
		}
	}
	if cfg.Plex.Retries <= 0 {
		cfg.Plex.Retries = 3
	}
	if cfg.Plex.Workers <= 0 {
		cfg.Plex.Workers = 1
	}
	if cfg.Plex.RetryBaseMs <= 0 {
		cfg.Plex.RetryBaseMs = 500
	}
	if cfg.Plex.RetryMaxMs <= 0 {
		cfg.Plex.RetryMaxMs = 30000
	}
	if logger != nil {
		logger.Infof("config load effective: plex_url=%q token_present=%t retries=%d workers=%d", strings.TrimSpace(cfg.Plex.BaseURL), strings.TrimSpace(cfg.Plex.Token) != "", cfg.Plex.Retries, cfg.Plex.Workers)
	}
	return cfg, nil
}

func loadWebDisplayConfig(path string, logger *AppLogger) (Config, error) {
	var cfg Config
	if logger != nil {
		logger.Infof("config display load: main file=%s", path)
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := toml.Unmarshal(bytes, &cfg); err != nil {
		return cfg, err
	}
	if strings.TrimSpace(cfg.PlexConfigFile) != "" {
		var supplemental Config
		if err := loadSupplementalConfig(path, cfg.PlexConfigFile, "plex_config", &supplemental); err != nil {
			return cfg, err
		}
		mergePlexConfig(&cfg, &supplemental)
	}
	if logger != nil {
		logger.Infof("config display effective: plex_config=%s log_file=%s output_dir=%s", blankDefault(cfg.PlexConfigFile, "<none>"), cfg.LogFile, cfg.OutputDir)
	}
	return cfg, nil
}

func loadResolvedSupplementalConfig(path, fieldName string, target any) error {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return nil
	}
	bytes, err := os.ReadFile(trimmedPath)
	if err != nil {
		return fmt.Errorf("%s: %w", fieldName, err)
	}
	if err := toml.Unmarshal(bytes, target); err != nil {
		return fmt.Errorf("%s: %w", fieldName, err)
	}
	return nil
}

func saveWebConfig(configPath string, request webConfigUpdateRequest, logger *AppLogger) error {
	runtimeCfg, err := loadWebRuntimeConfig(configPath, logger)
	if err != nil {
		return err
	}
	baseDocument, err := readTOMLDocument(configPath, logger)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	baseDocument["template_image"] = strings.TrimSpace(request.TemplateImage)
	baseDocument["type_template_image"] = strings.TrimSpace(request.TypeTemplateImage)
	baseDocument["studio_template_image"] = strings.TrimSpace(request.StudioTemplateImage)
	baseDocument["admin_template_image"] = strings.TrimSpace(request.AdminTemplateImage)
	baseDocument["type_collections_file"] = strings.TrimSpace(request.TypeCollectionsFile)
	baseDocument["studio_collections_file"] = strings.TrimSpace(request.StudioCollectionsFile)
	baseDocument["admin_collections_file"] = strings.TrimSpace(request.AdminCollectionsFile)
	baseDocument["output_dir"] = strings.TrimSpace(request.OutputDir)
	baseDocument["log_file"] = strings.TrimSpace(request.LogFile)
	baseDocument["plex_config"] = strings.TrimSpace(request.PlexConfigFile)
	baseDocument["label_config"] = strings.TrimSpace(request.LabelConfigFile)
	baseDocument["collection_config"] = strings.TrimSpace(request.CollectionConfigFile)
	cleanSection := map[string]any{}
	if existing, ok := baseDocument["clean"].(map[string]any); ok {
		cleanSection = existing
	}
	cleanSection["translate_to_english"] = request.TranslateToEnglish
	cleanSection["translate_endpoint"] = strings.TrimSpace(request.TranslateEndpoint)
	cleanSection["translate_api_http_address"] = strings.TrimSpace(request.TranslateEndpoint)
	cleanSection["translate_api_key"] = strings.TrimSpace(request.TranslateAPIKey)
	cleanSection["translate_rate_limit_per_minute"] = request.TranslateRateLimitMinute
	replacements, err := parseCleanReplacements(request.CleanReplacements)
	if err != nil {
		return fmt.Errorf("clean replacements: %w", err)
	}
	cleanSection["replacements"] = replacements
	baseDocument["clean"] = cleanSection
	statsSection := map[string]any{}
	if existing, ok := baseDocument["stats"].(map[string]any); ok {
		statsSection = existing
	}
	statsSection["exclude_words"] = parseStatsExcludeWords(request.StatsExcludeWords)
	baseDocument["stats"] = statsSection
	backupSection := map[string]any{}
	if existing, ok := baseDocument["backup"].(map[string]any); ok {
		backupSection = existing
	}
	backupSection["retention_days"] = request.BackupRetentionDays
	baseDocument["backup"] = backupSection
	fontSection := map[string]any{}
	if existing, ok := baseDocument["font"].(map[string]any); ok {
		fontSection = existing
	}
	fontSection["file"] = strings.TrimSpace(request.FontFile)
	fontSection["size"] = request.FontSize
	fontSection["color"] = strings.TrimSpace(request.FontColor)
	fontSection["shadow_color"] = strings.TrimSpace(request.FontShadowColor)
	fontSection["shadow_offset_x"] = request.FontShadowOffsetX
	fontSection["shadow_offset_y"] = request.FontShadowOffsetY
	fontSection["glow_color"] = strings.TrimSpace(request.FontGlowColor)
	fontSection["glow_radius"] = request.FontGlowRadius
	fontSection["glow_alpha"] = request.FontGlowAlpha
	fontSection["y_offset"] = request.FontYOffset
	baseDocument["font"] = fontSection

	targetPath := configPath
	if strings.TrimSpace(request.PlexConfigFile) != "" {
		targetPath = resolvePathRelativeToConfig(configPath, request.PlexConfigFile)
	} else if strings.TrimSpace(runtimeCfg.PlexConfigFile) != "" {
		targetPath = runtimeCfg.PlexConfigFile
	}
	if logger != nil {
		logger.Infof("config save: main file=%s plex target=%s", configPath, targetPath)
	}
	plexDocument := baseDocument
	if targetPath != configPath {
		plexDocument, err = readTOMLDocument(targetPath, logger)
		if err != nil {
			return fmt.Errorf("read plex config: %w", err)
		}
	}
	plexSection := map[string]any{}
	if existing, ok := plexDocument["plex"].(map[string]any); ok {
		plexSection = existing
	}
	plexSection["base_url"] = strings.TrimSpace(request.BaseURL)
	plexSection["token"] = strings.TrimSpace(request.Token)
	plexSection["retries"] = request.Retries
	plexSection["workers"] = request.Workers
	plexSection["retry_base_ms"] = request.RetryBaseMs
	plexSection["retry_max_ms"] = request.RetryMaxMs
	plexDocument["plex"] = plexSection
	if err := writeTOMLDocument(configPath, baseDocument, logger); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	if targetPath == configPath {
		return nil
	}
	if err := writeTOMLDocument(targetPath, plexDocument, logger); err != nil {
		return fmt.Errorf("save plex config: %w", err)
	}
	return nil
}

func readTOMLDocument(path string, logger *AppLogger) (map[string]any, error) {
	document := map[string]any{}
	if logger != nil {
		logger.Infof("config file read: path=%s", path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if logger != nil {
				logger.Warningf("config file read: missing path=%s", path)
			}
			return document, nil
		}
		return nil, err
	}
	if len(raw) == 0 {
		return document, nil
	}
	if err := toml.Unmarshal(raw, &document); err != nil {
		return nil, err
	}
	if logger != nil {
		logger.Infof("config file read complete: path=%s bytes=%d", path, len(raw))
	}
	return document, nil
}

func writeTOMLDocument(path string, document map[string]any, logger *AppLogger) error {
	out, err := toml.Marshal(document)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return err
	}
	if logger != nil {
		logger.Successf("config file updated: path=%s bytes=%d", path, len(out))
	}
	return nil
}

func validateWebConfigUpdate(request webConfigUpdateRequest) error {
	if strings.TrimSpace(request.BaseURL) == "" {
		return errors.New("base_url is required")
	}
	if request.Retries <= 0 {
		return errors.New("retries must be greater than zero")
	}
	if request.Workers <= 0 {
		return errors.New("workers must be greater than zero")
	}
	if request.RetryBaseMs <= 0 {
		return errors.New("retry_base_ms must be greater than zero")
	}
	if request.RetryMaxMs <= 0 {
		return errors.New("retry_max_ms must be greater than zero")
	}
	if request.TranslateRateLimitMinute <= 0 {
		return errors.New("translate_rate_limit_per_minute must be greater than zero")
	}
	if request.BackupRetentionDays < 0 {
		return errors.New("backup_retention_days must be greater than or equal to zero")
	}
	if request.FontSize < 0 {
		return errors.New("font_size must be greater than or equal to zero")
	}
	if request.FontGlowRadius < 0 {
		return errors.New("font_glow_radius must be greater than or equal to zero")
	}
	if request.FontGlowAlpha < 0 || request.FontGlowAlpha > 1 {
		return errors.New("font_glow_alpha must be between 0 and 1")
	}
	return nil
}

func validateWebPlexConnectionRequest(request webConfigUpdateRequest) error {
	if strings.TrimSpace(request.BaseURL) == "" {
		return errors.New("base_url is required")
	}
	if strings.TrimSpace(request.Token) == "" {
		return errors.New("token is required")
	}
	if request.Retries <= 0 {
		return errors.New("retries must be greater than zero")
	}
	if request.Workers <= 0 {
		return errors.New("workers must be greater than zero")
	}
	if request.RetryBaseMs <= 0 || request.RetryMaxMs <= 0 {
		return errors.New("retry timing values must be greater than zero")
	}
	return nil
}

func (s *webServer) testPlexConnection(request webConfigUpdateRequest, logger *AppLogger) error {
	cfg, err := loadWebRuntimeConfig(s.configPath, logger)
	if err != nil {
		return err
	}
	cfg.Plex.BaseURL = strings.TrimSpace(request.BaseURL)
	cfg.Plex.Token = strings.TrimSpace(request.Token)
	cfg.Plex.Retries = request.Retries
	cfg.Plex.Workers = request.Workers
	cfg.Plex.RetryBaseMs = request.RetryBaseMs
	cfg.Plex.RetryMaxMs = request.RetryMaxMs
	client := &http.Client{Timeout: 30 * time.Second}
	sections, err := fetchSections(client, cfg, logger)
	if err != nil {
		return err
	}
	logger.Infof("plex test result: sections=%d", len(sections))
	return nil
}

func serializeCleanReplacements(replacements map[string]string) string {
	if len(replacements) == 0 {
		return ""
	}
	keys := make([]string, 0, len(replacements))
	for key := range replacements {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+" = "+replacements[key])
	}
	return strings.Join(lines, "\n")
}

func parseCleanReplacements(raw string) (map[string]string, error) {
	replacements := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid replacement line %q; expected key = value", line)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, fmt.Errorf("invalid replacement line %q; key is required", line)
		}
		replacements[key] = value
	}
	return replacements, nil
}

func parseStatsExcludeWords(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t'
	})
	words := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		word := strings.TrimSpace(part)
		if word == "" {
			continue
		}
		key := strings.ToLower(word)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		words = append(words, word)
	}
	return words
}

func (s *webServer) backups() ([]webBackupSummary, error) {
	workspaceRoot, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	backups, err := listBackupArchives(backupArchiveDir(workspaceRoot))
	if err != nil {
		return nil, err
	}
	out := make([]webBackupSummary, 0, len(backups))
	for _, backup := range backups {
		out = append(out, webBackupSummary{
			Name:    backup.Name,
			Host:    backup.Host,
			ModTime: backup.ModTime.Format(time.RFC3339),
			Label:   formatBackupDateTime(backup.ModTime),
			Path:    backup.Path,
		})
	}
	return out, nil
}

func newBufferedLogger(base *AppLogger) (*AppLogger, *bytes.Buffer) {
	buffer := &bytes.Buffer{}
	consoleWriter := io.Writer(buffer)
	fileWriter := io.Writer(buffer)
	quiet := false
	if base != nil {
		quiet = base.Quiet
		if base.Console != nil {
			consoleWriter = io.MultiWriter(base.Console.Writer(), buffer)
		}
		if base.File != nil {
			fileWriter = io.MultiWriter(base.File.Writer(), buffer)
		}
	}
	return &AppLogger{
		Console: log.New(consoleWriter, "", log.LstdFlags|log.Lmicroseconds),
		File:    log.New(fileWriter, "", log.LstdFlags|log.Lmicroseconds),
		Quiet:   quiet,
	}, buffer
}

func withTrailMode(enabled bool, fn func() error) error {
	previous := core.TrailModeEnabled
	core.TrailModeEnabled = enabled
	defer func() {
		core.TrailModeEnabled = previous
	}()
	return fn()
}

func (s *webServer) detectLatestDownloadableOutput(startedAt, completedAt time.Time) string {
	if startedAt.IsZero() || completedAt.IsZero() {
		return ""
	}
	cfg, err := loadWebRuntimeConfig(s.configPath, s.logger)
	if err != nil {
		return ""
	}
	roots := []string{cfg.OutputDir}
	if cwd, err := os.Getwd(); err == nil {
		roots = append(roots, filepath.Join(cwd, core.BackupDirName))
	}
	windowStart := startedAt.Add(-2 * time.Second)
	windowEnd := completedAt.Add(2 * time.Second)
	type fileCandidate struct {
		path string
		mod  time.Time
	}
	var newest fileCandidate
	for _, root := range roots {
		cleanRoot := strings.TrimSpace(root)
		if cleanRoot == "" {
			continue
		}
		walkErr := filepath.WalkDir(cleanRoot, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !isDownloadableOutputPath(path) {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			mod := info.ModTime()
			if mod.Before(windowStart) || mod.After(windowEnd) {
				return nil
			}
			if newest.path == "" || mod.After(newest.mod) {
				newest = fileCandidate{path: path, mod: mod}
			}
			return nil
		})
		if walkErr != nil {
			continue
		}
	}
	return newest.path
}

func isDownloadableOutputPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp", ".tif", ".tiff", ".avif", ".svg":
		return false
	default:
		return true
	}
}

func buildWebAbout(startedAt time.Time) webAbout {
	about := webAbout{
		Module:    "github.com/mt1976/frantic-postr",
		Version:   effectiveAppVersion(),
		GoVersion: currentGoVersion(),
		StartedAt: startedAt.Format(time.RFC822),
		Mode:      "Local UIKit web UI",
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Path != "" {
			about.Module = info.Main.Path
		}
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && setting.Value != "" {
				about.CommitHint = shortRevision(setting.Value)
				break
			}
		}
	}
	return about
}

func effectiveAppVersion() string {
	trimmed := strings.TrimSpace(appVersion)
	if trimmed != "" && trimmed != "development" {
		return trimmed
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if strings.TrimSpace(info.Main.Version) != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return "development"
}

func currentGoVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && strings.TrimSpace(info.GoVersion) != "" {
		return info.GoVersion
	}
	return "unknown"
}

func shortRevision(in string) string {
	trimmed := strings.TrimSpace(in)
	if len(trimmed) > 12 {
		return trimmed[:12]
	}
	return trimmed
}

func blankDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func joinMessages(existing, next string) string {
	if strings.TrimSpace(existing) == "" {
		return next
	}
	if strings.TrimSpace(next) == "" {
		return existing
	}
	return existing + "; " + next
}

func writeHTML(w http.ResponseWriter, status int, tmpl *template.Template, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = tmpl.Execute(w, data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, webActionResponse{OK: false, Error: message})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
