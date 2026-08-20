package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type LocalController struct {
	server *webServer
}

func NewLocalController(configPath string, logger *AppLogger) *LocalController {
	return &LocalController{server: &webServer{
		configPath: configPath,
		logger:     logger,
		startedAt:  time.Now(),
	}}
}

func (c *LocalController) State() (WebStateResponse, error) {
	if c == nil || c.server == nil {
		return WebStateResponse{}, errors.New("local controller is not initialized")
	}
	return c.server.stateResponse()
}

func (c *LocalController) Status() WebActionStatusResponse {
	if c == nil || c.server == nil {
		return WebActionStatusResponse{}
	}
	return c.server.snapshotActionRuntime()
}

func (c *LocalController) Stop() WebActionResponse {
	if c == nil || c.server == nil {
		return WebActionResponse{OK: false, Error: "Local controller is not initialized."}
	}
	c.server.actionMu.Lock()
	running := c.server.actionRun.running
	if running {
		c.server.actionRun.stopAsked = true
	}
	stop := c.server.actionStop
	c.server.actionMu.Unlock()
	if !running {
		return WebActionResponse{OK: false, Error: "No active operation to stop."}
	}
	if stop != nil {
		stop()
	}
	return WebActionResponse{OK: true, Message: "Stop requested. The active operation will halt shortly."}
}

func (c *LocalController) RunAction(action string, request WebActionRequest) WebActionResponse {
	if c == nil || c.server == nil {
		return WebActionResponse{OK: false, Error: "Local controller is not initialized."}
	}
	action = strings.TrimSpace(strings.Trim(action, "/"))
	if action == "" {
		return WebActionResponse{OK: false, Error: "Action name is required."}
	}
	if !c.server.busy.CompareAndSwap(false, true) {
		return WebActionResponse{OK: false, Error: "Another operation is already running. Wait for it to finish and retry."}
	}
	defer c.server.busy.Store(false)
	c.server.resetActionRuntime(action)
	actionCtx, cancel := context.WithCancel(context.Background())
	c.server.actionMu.Lock()
	c.server.actionStop = cancel
	startedAt := c.server.actionRun.startedAt
	c.server.actionMu.Unlock()
	restoreRequestContext := setRequestContext(actionCtx)
	defer restoreRequestContext()
	defer cancel()

	requestLogger, logBuffer := newBufferedLogger(c.server.logger)
	requestLogger.LogCallback = func(line string) {
		c.server.appendActionLog(line)
	}
	requestLogger.ProgressCallback = func(label string, current, total int, final bool) {
		c.server.updateActionProgress(label, current, total, final)
	}
	err := c.server.runAction(action, request, requestLogger)
	outputFile := c.server.detectLatestDownloadableOutput(startedAt, time.Now())
	response := WebActionResponse{OK: err == nil, Logs: logBuffer.String()}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			response.OK = false
			response.Error = "Operation stopped by user."
			c.server.completeActionRuntime(false, true, "Operation stopped.", response.Error, outputFile)
			return response
		}
		response.Error = err.Error()
		c.server.completeActionRuntime(false, false, "", err.Error(), outputFile)
		return response
	}
	response.Message = fmt.Sprintf("%s completed.", action)
	c.server.completeActionRuntime(true, false, response.Message, "", outputFile)
	return response
}

func (c *LocalController) SaveConfig(request WebConfigUpdateRequest) WebActionResponse {
	if c == nil || c.server == nil {
		return WebActionResponse{OK: false, Error: "Local controller is not initialized."}
	}
	requestLogger, logBuffer := newBufferedLogger(c.server.logger)
	if err := SaveWebConfig(c.server.configPath, request, requestLogger); err != nil {
		return WebActionResponse{OK: false, Error: err.Error(), Logs: logBuffer.String()}
	}
	return WebActionResponse{OK: true, Message: "Configuration saved.", Logs: logBuffer.String()}
}

func (c *LocalController) TestPlexConnection(request WebConfigUpdateRequest) WebActionResponse {
	if c == nil || c.server == nil {
		return WebActionResponse{OK: false, Error: "Local controller is not initialized."}
	}
	if err := validateWebPlexConnectionRequest(request); err != nil {
		return WebActionResponse{OK: false, Error: err.Error()}
	}
	requestLogger, logBuffer := newBufferedLogger(c.server.logger)
	if err := c.server.testPlexConnection(request, requestLogger); err != nil {
		return WebActionResponse{OK: false, Error: err.Error(), Logs: logBuffer.String()}
	}
	return WebActionResponse{OK: true, Message: "Plex connection succeeded.", Logs: logBuffer.String()}
}

func (c *LocalController) Collections(sectionKey string) ([]PlexCollection, error) {
	if c == nil || c.server == nil {
		return nil, errors.New("local controller is not initialized")
	}
	cfg, err := loadWebRuntimeConfig(c.server.configPath, c.server.logger)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	return fetchCollections(client, cfg, strings.TrimSpace(sectionKey), c.server.logger)
}
