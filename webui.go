package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pelletier/go-toml/v2"
)

var appVersion = "development"

type webServer struct {
	configPath string
	port       int
	logger     *AppLogger
	startedAt  time.Time
	busy       atomic.Bool
	busyMu     sync.Mutex
	actionMu   sync.Mutex
	actionRun  webActionRuntime
}

type webActionRuntime struct {
	action      string
	startedAt   time.Time
	completedAt time.Time
	running     bool
	ok          bool
	message     string
	err         string
	logs        strings.Builder
	hasProgress bool
	progress    webActionProgressSnapshot
}

type webActionProgressSnapshot struct {
	Label     string `json:"label"`
	Current   int    `json:"current"`
	Total     int    `json:"total"`
	Percent   int    `json:"percent"`
	Final     bool   `json:"final"`
	UpdatedAt string `json:"updated_at"`
}

type webActionStatusResponse struct {
	Running     bool                       `json:"running"`
	Action      string                     `json:"action,omitempty"`
	StartedAt   string                     `json:"started_at,omitempty"`
	CompletedAt string                     `json:"completed_at,omitempty"`
	OK          bool                       `json:"ok"`
	Message     string                     `json:"message,omitempty"`
	Error       string                     `json:"error,omitempty"`
	Logs        string                     `json:"logs,omitempty"`
	Progress    *webActionProgressSnapshot `json:"progress,omitempty"`
}

type webPlexSettings struct {
	BaseURL     string `json:"base_url"`
	Token       string `json:"token"`
	Retries     int    `json:"retries"`
	Workers     int    `json:"workers"`
	RetryBaseMs int    `json:"retry_base_ms"`
	RetryMaxMs  int    `json:"retry_max_ms"`
}

type webGeneralSettings struct {
	TemplateImage             string  `json:"template_image"`
	TypeTemplateImage         string  `json:"type_template_image"`
	StudioTemplateImage       string  `json:"studio_template_image"`
	AdminTemplateImage        string  `json:"admin_template_image"`
	TypeCollectionsFile       string  `json:"type_collections_file"`
	StudioCollectionsFile     string  `json:"studio_collections_file"`
	AdminCollectionsFile      string  `json:"admin_collections_file"`
	OutputDir                string `json:"output_dir"`
	LogFile                  string `json:"log_file"`
	PlexConfigFile           string  `json:"plex_config"`
	LabelConfigFile          string  `json:"label_config"`
	CollectionConfigFile     string  `json:"collection_config"`
	TranslateToEnglish       bool    `json:"translate_to_english"`
	TranslateEndpoint        string `json:"translate_endpoint"`
	TranslateAPIKey          string `json:"translate_api_key"`
	TranslateRateLimitMinute int    `json:"translate_rate_limit_per_minute"`
	CleanReplacements        string  `json:"clean_replacements"`
	StatsExcludeWords        string  `json:"stats_exclude_words"`
	BackupRetentionDays      int     `json:"backup_retention_days"`
	FontFile                 string  `json:"font_file"`
	FontSize                 float64 `json:"font_size"`
	FontColor                string  `json:"font_color"`
	FontShadowColor          string  `json:"font_shadow_color"`
	FontShadowOffsetX        int     `json:"font_shadow_offset_x"`
	FontShadowOffsetY        int     `json:"font_shadow_offset_y"`
	FontGlowColor            string  `json:"font_glow_color"`
	FontGlowRadius           int     `json:"font_glow_radius"`
	FontGlowAlpha            float64 `json:"font_glow_alpha"`
	FontYOffset              int     `json:"font_y_offset"`
}

type webBackupSummary struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	ModTime string `json:"mod_time"`
	Label   string `json:"label"`
	Path    string `json:"path"`
}

type webStateResponse struct {
	AppName       string             `json:"app_name"`
	Version       string             `json:"version"`
	GoVersion     string             `json:"go_version"`
	StartedAt     string             `json:"started_at"`
	ConfigPath    string             `json:"config_path"`
	OutputDir     string             `json:"output_dir"`
	LogFile       string             `json:"log_file"`
	Plex          webPlexSettings    `json:"plex"`
	General       webGeneralSettings `json:"general"`
	Sections      []plexSection      `json:"sections"`
	Backups       []webBackupSummary `json:"backups"`
	ConfigValid   bool               `json:"config_valid"`
	ConfigError   string             `json:"config_error,omitempty"`
	SectionsError string             `json:"sections_error,omitempty"`
	Defaults      webDefaults        `json:"defaults"`
	About         webAbout           `json:"about"`
}

type webDefaults struct {
	ImportPath string `json:"import_path"`
	ExportPath string `json:"export_path"`
}

type webAbout struct {
	Module     string `json:"module"`
	Version    string `json:"version"`
	GoVersion  string `json:"go_version"`
	CommitHint string `json:"commit_hint,omitempty"`
	StartedAt  string `json:"started_at"`
	Mode       string `json:"mode"`
}

type webConfigUpdateRequest struct {
	BaseURL     string `json:"base_url"`
	Token       string `json:"token"`
	Retries     int    `json:"retries"`
	Workers     int    `json:"workers"`
	RetryBaseMs int    `json:"retry_base_ms"`
	RetryMaxMs  int    `json:"retry_max_ms"`
	OutputDir                string `json:"output_dir"`
	LogFile                  string `json:"log_file"`
	TemplateImage            string  `json:"template_image"`
	TypeTemplateImage        string  `json:"type_template_image"`
	StudioTemplateImage      string  `json:"studio_template_image"`
	AdminTemplateImage       string  `json:"admin_template_image"`
	TypeCollectionsFile      string  `json:"type_collections_file"`
	StudioCollectionsFile    string  `json:"studio_collections_file"`
	AdminCollectionsFile     string  `json:"admin_collections_file"`
	PlexConfigFile           string  `json:"plex_config"`
	LabelConfigFile          string  `json:"label_config"`
	CollectionConfigFile     string  `json:"collection_config"`
	TranslateToEnglish       bool    `json:"translate_to_english"`
	TranslateEndpoint        string `json:"translate_endpoint"`
	TranslateAPIKey          string `json:"translate_api_key"`
	TranslateRateLimitMinute int    `json:"translate_rate_limit_per_minute"`
	CleanReplacements        string  `json:"clean_replacements"`
	StatsExcludeWords        string  `json:"stats_exclude_words"`
	BackupRetentionDays      int     `json:"backup_retention_days"`
	FontFile                 string  `json:"font_file"`
	FontSize                 float64 `json:"font_size"`
	FontColor                string  `json:"font_color"`
	FontShadowColor          string  `json:"font_shadow_color"`
	FontShadowOffsetX        int     `json:"font_shadow_offset_x"`
	FontShadowOffsetY        int     `json:"font_shadow_offset_y"`
	FontGlowColor            string  `json:"font_glow_color"`
	FontGlowRadius           int     `json:"font_glow_radius"`
	FontGlowAlpha            float64 `json:"font_glow_alpha"`
	FontYOffset              int     `json:"font_y_offset"`
}

type webActionRequest struct {
	SectionKey     string   `json:"section_key"`
	SectionKeys    []string `json:"section_keys"`
	CollectionKey  string   `json:"collection_key"`
	UploadPosters  bool     `json:"upload_posters"`
	LabelTypes     bool     `json:"label_types"`
	Trail          bool     `json:"trail"`
	Translate      bool     `json:"translate"`
	Find           string   `json:"find"`
	Add            string   `json:"add"`
	Categories     string   `json:"categories"`
	UpdateCategory bool     `json:"update_category"`
	OnlyCategory   bool     `json:"only_category"`
	CollFile       string   `json:"coll_file"`
	RestoreFile    string   `json:"restore_file"`
	CloneName      string   `json:"clone_name"`
}

type webActionResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
	Logs    string `json:"logs,omitempty"`
}

type webPageData struct {
	AppName   string
	Version   string
	GoVersion string
	StartedAt string
	Port      int
	HelpPath  string
	Mode      string
	About     webAbout
}

var webIndexTemplate = template.Must(template.New("index").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.AppName}} Web UI</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@400;500;700&family=IBM+Plex+Mono:wght@400;500&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/uikit@3.23.8/dist/css/uikit.min.css">
  <script defer src="https://cdn.jsdelivr.net/npm/uikit@3.23.8/dist/js/uikit.min.js"></script>
  <script defer src="https://cdn.jsdelivr.net/npm/uikit@3.23.8/dist/js/uikit-icons.min.js"></script>
  <style>
    :root {
      --fp-ink: #182126;
      --fp-deep: #23353c;
      --fp-sand: #f5efe2;
      --fp-paper: rgba(255,255,255,0.78);
      --fp-accent: #d36b2d;
      --fp-accent-soft: #f0bc6c;
      --fp-line: rgba(24,33,38,0.12);
      --fp-ok: #2d8b57;
      --fp-warn: #a65b21;
      --fp-error: #9d2f2f;
    }
    html, body {
      min-height: 100%;
      background:
        radial-gradient(circle at top left, rgba(240,188,108,0.4), transparent 30%),
        linear-gradient(160deg, #f8f1e4 0%, #f2e9d8 42%, #ebe5da 100%);
      color: var(--fp-ink);
      font-family: "Space Grotesk", sans-serif;
    }
    .fp-shell {
      max-width: 1380px;
      margin: 0 auto;
      padding: 24px;
    }
    .fp-hero {
      background: linear-gradient(135deg, rgba(35,53,60,0.96), rgba(54,78,85,0.92));
      color: #f7f3ea;
      border-radius: 28px;
      padding: 32px;
      box-shadow: 0 24px 70px rgba(31, 36, 41, 0.16);
      overflow: hidden;
      position: relative;
    }
    .fp-hero::after {
      content: "";
      position: absolute;
      right: -80px;
      top: -40px;
      width: 260px;
      height: 260px;
      background: radial-gradient(circle, rgba(240,188,108,0.46), transparent 70%);
    }
    .fp-kicker, .fp-meta, .fp-log, .uk-input, .uk-select, .uk-textarea, .uk-button {
      font-family: "IBM Plex Mono", monospace;
    }
    .fp-card, .uk-card {
      border-radius: 22px;
      background: var(--fp-paper);
      border: 1px solid var(--fp-line);
      backdrop-filter: blur(10px);
      box-shadow: 0 18px 50px rgba(24,33,38,0.08);
    }
    .fp-title {
      letter-spacing: -0.04em;
      margin: 0;
    }
    .fp-kicker {
      display: inline-flex;
      gap: 10px;
      align-items: center;
      text-transform: uppercase;
      letter-spacing: 0.12em;
      font-size: 12px;
      color: rgba(247,243,234,0.82);
    }
    .fp-summary-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
      gap: 14px;
    }
    .fp-stat {
      padding: 16px 18px;
      border-radius: 18px;
      background: rgba(255,255,255,0.08);
      border: 1px solid rgba(255,255,255,0.08);
    }
    .fp-stat strong {
      display: block;
      font-size: 24px;
      line-height: 1.1;
      margin-top: 10px;
    }
    .fp-section-title {
      margin: 0 0 6px;
      font-size: 1.15rem;
    }
    .fp-banner {
      border-radius: 18px;
      padding: 14px 16px;
      display: none;
    }
    .fp-banner.active {
      display: block;
    }
    .fp-banner.ok { background: rgba(45,139,87,0.12); color: var(--fp-ok); }
    .fp-banner.warn { background: rgba(166,91,33,0.14); color: var(--fp-warn); }
    .fp-banner.error { background: rgba(157,47,47,0.12); color: var(--fp-error); }
    .fp-log {
      min-height: 280px;
      max-height: 480px;
      overflow: auto;
      padding: 18px;
      border-radius: 18px;
      background: #151d21;
      color: #e9e6dd;
      white-space: pre-wrap;
      border: 1px solid rgba(255,255,255,0.07);
    }
		.fp-progress-wrap {
			border-radius: 14px;
			border: 1px solid var(--fp-line);
			background: rgba(255,255,255,0.6);
			padding: 12px;
		}
		.fp-progress-head {
			display: flex;
			justify-content: space-between;
			align-items: center;
			gap: 10px;
			font-family: "IBM Plex Mono", monospace;
			font-size: 12px;
			color: rgba(24,33,38,0.78);
			margin-bottom: 8px;
		}
		.fp-progress-title {
			margin: 0;
			font-size: 13px;
			letter-spacing: 0.04em;
			text-transform: uppercase;
		}
		.fp-progress-bar {
			width: 100%;
			height: 14px;
			appearance: none;
			border: 0;
			border-radius: 999px;
			overflow: hidden;
			background: rgba(24,33,38,0.1);
		}
		.fp-progress-bar::-webkit-progress-bar {
			background: rgba(24,33,38,0.1);
			border-radius: 999px;
		}
		.fp-progress-bar::-webkit-progress-value {
			background: linear-gradient(90deg, var(--fp-accent), var(--fp-accent-soft));
			border-radius: 999px;
			transition: width 120ms linear;
		}
		.fp-progress-bar::-moz-progress-bar {
			background: linear-gradient(90deg, var(--fp-accent), var(--fp-accent-soft));
			border-radius: 999px;
		}
		.fp-progress-meta {
			margin-top: 8px;
			font-family: "IBM Plex Mono", monospace;
			font-size: 12px;
			color: rgba(24,33,38,0.66);
		}
    .fp-chip {
      display: inline-flex;
      align-items: center;
      border-radius: 999px;
      padding: 5px 10px;
      background: rgba(211,107,45,0.12);
      color: var(--fp-accent);
      font-size: 12px;
      margin: 4px 6px 0 0;
    }
    .fp-actions-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      gap: 16px;
    }
    .fp-muted {
      color: rgba(24,33,38,0.68);
    }
    .fp-path {
      word-break: break-all;
    }
    .fp-footer-note {
      font-size: 12px;
      color: rgba(24,33,38,0.62);
    }
		.fp-test-button {
			position: relative;
			border: 1px solid rgba(24,33,38,0.16);
			transition: border-color 180ms ease, box-shadow 180ms ease, color 180ms ease, background 180ms ease;
		}
		.fp-test-button:hover,
		.fp-test-button:focus {
			border-color: rgba(24,33,38,0.32);
		}
		.fp-test-button.is-loading {
			border-color: rgba(211,107,45,0.6);
			box-shadow: 0 0 0 3px rgba(211,107,45,0.12);
		}
		.fp-test-button.is-success {
			border-color: rgba(45,139,87,0.9);
			box-shadow: 0 0 0 3px rgba(45,139,87,0.14);
			color: var(--fp-ok);
			background: rgba(45,139,87,0.08);
		}
		.fp-test-button.is-error {
			border-color: rgba(157,47,47,0.9);
			box-shadow: 0 0 0 3px rgba(157,47,47,0.14);
			color: var(--fp-error);
			background: rgba(157,47,47,0.08);
		}
		.fp-test-button-icon {
			display: inline-flex;
			align-items: center;
			justify-content: center;
			width: 1.1rem;
			margin-left: 8px;
			opacity: 0;
			transform: scale(0.8);
			transition: opacity 180ms ease, transform 180ms ease;
		}
		.fp-test-button.is-success .fp-test-button-icon,
		.fp-test-button.is-error .fp-test-button-icon,
		.fp-test-button.is-loading .fp-test-button-icon {
			opacity: 1;
			transform: scale(1);
		}
		.fp-test-button.is-success .fp-test-button-icon {
			color: var(--fp-ok);
		}
		.fp-test-button.is-error .fp-test-button-icon {
			color: var(--fp-error);
		}
		.fp-test-button.is-loading .fp-test-button-icon {
			color: var(--fp-accent);
			animation: fp-pulse 1s ease-in-out infinite;
		}
		@keyframes fp-pulse {
			0%, 100% { opacity: 0.45; }
			50% { opacity: 1; }
		}
    @media (max-width: 640px) {
      .fp-shell { padding: 14px; }
      .fp-hero { padding: 22px; border-radius: 22px; }
      .fp-log { min-height: 220px; }
    }
  </style>
</head>
<body>
  <div class="fp-shell">
    <section class="fp-hero uk-margin-large-bottom uk-animation-slide-top-small">
      <div class="uk-flex uk-flex-between uk-flex-wrap uk-flex-middle uk-gap-small">
        <div>
          <div class="fp-kicker">{{.Mode}} <span>•</span> Version {{.Version}}</div>
          <h1 class="fp-title uk-heading-small">{{.AppName}} control room</h1>
          <p class="uk-text-large uk-margin-small-top uk-margin-medium-bottom" style="max-width: 760px; color: rgba(247,243,234,0.86);">Run poster generation, library cleanup, collection maintenance, and backup workflows from a local UIKit dashboard without losing the existing Plex-aware Go logic.</p>
          <div class="uk-flex uk-flex-wrap uk-gap-small">
            <button class="uk-button uk-button-secondary" id="refresh-state">Refresh state</button>
            <a class="uk-button uk-button-default" href="{{.HelpPath}}">Help & tips</a>
            <button class="uk-button uk-button-default" uk-toggle="target: #about-modal">About</button>
          </div>
        </div>
        <div class="fp-summary-grid" style="min-width: min(100%, 380px); max-width: 420px;">
          <div class="fp-stat">
            <span class="fp-kicker">Listening</span>
            <strong>127.0.0.1:{{.Port}}</strong>
          </div>
          <div class="fp-stat">
            <span class="fp-kicker">Started</span>
            <strong>{{.StartedAt}}</strong>
          </div>
        </div>
      </div>
    </section>

    <div id="status-banner" class="fp-banner uk-margin-bottom"></div>

    <div class="uk-grid-large" uk-grid>
      <div class="uk-width-1-3@l">
        <div class="uk-card uk-card-body fp-card uk-margin-bottom uk-animation-slide-left-small">
					<h2 class="fp-section-title">Configuration</h2>
					<p class="fp-muted uk-margin-small-top">If you do not pass <code>--config</code>, the web UI uses <code>config/config.toml</code>. Saving here writes the exposed settings back to the active config files, and keeps Plex secrets in <code>plex_config</code> when you use a split file.</p>
          <form id="config-form" class="uk-grid-small" uk-grid>
            <div class="uk-width-1-1">
              <label class="uk-form-label" for="base-url">Plex URL</label>
              <input class="uk-input" id="base-url" name="base_url" type="url" required>
            </div>
            <div class="uk-width-1-1">
              <label class="uk-form-label" for="token">API key / token</label>
			  <input class="uk-input" id="token" name="token" type="text" autocomplete="off">
            </div>
            <div class="uk-width-1-2">
              <label class="uk-form-label" for="retries">Retries</label>
              <input class="uk-input" id="retries" name="retries" type="number" min="1">
            </div>
            <div class="uk-width-1-2">
              <label class="uk-form-label" for="workers">Workers</label>
              <input class="uk-input" id="workers" name="workers" type="number" min="1">
            </div>
            <div class="uk-width-1-2">
              <label class="uk-form-label" for="retry-base-ms">Retry base ms</label>
              <input class="uk-input" id="retry-base-ms" name="retry_base_ms" type="number" min="1">
            </div>
            <div class="uk-width-1-2">
              <label class="uk-form-label" for="retry-max-ms">Retry max ms</label>
              <input class="uk-input" id="retry-max-ms" name="retry_max_ms" type="number" min="1">
            </div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="output-dir-input">Output directory</label>
							<input class="uk-input" id="output-dir-input" name="output_dir" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="log-file-input">Log file</label>
							<input class="uk-input" id="log-file-input" name="log_file" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="translate-endpoint">Translate endpoint</label>
							<input class="uk-input" id="translate-endpoint" name="translate_endpoint" type="url">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="translate-api-key">Translate API key</label>
							<input class="uk-input" id="translate-api-key" name="translate_api_key" type="password" autocomplete="off">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="translate-rate-limit">Translate rate limit per minute</label>
							<input class="uk-input" id="translate-rate-limit" name="translate_rate_limit_per_minute" type="number" min="1">
						</div>
						<div class="uk-width-1-1">
							<h3 class="fp-section-title uk-margin-small-top">Config files and templates</h3>
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="template-image">Template image</label>
							<input class="uk-input" id="template-image" name="template_image" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="type-template-image">Type template image</label>
							<input class="uk-input" id="type-template-image" name="type_template_image" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="studio-template-image">Studio template image</label>
							<input class="uk-input" id="studio-template-image" name="studio_template_image" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="admin-template-image">Admin template image</label>
							<input class="uk-input" id="admin-template-image" name="admin_template_image" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="type-collections-file">Type collections file</label>
							<input class="uk-input" id="type-collections-file" name="type_collections_file" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="studio-collections-file">Studio collections file</label>
							<input class="uk-input" id="studio-collections-file" name="studio_collections_file" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="admin-collections-file">Admin collections file</label>
							<input class="uk-input" id="admin-collections-file" name="admin_collections_file" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="plex-config-file">Plex config file</label>
							<input class="uk-input" id="plex-config-file" name="plex_config" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="label-config-file">Label config file</label>
							<input class="uk-input" id="label-config-file" name="label_config" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="collection-config-file">Collection config file</label>
							<input class="uk-input" id="collection-config-file" name="collection_config" type="text">
						</div>
						<div class="uk-width-1-1">
							<h3 class="fp-section-title uk-margin-small-top">Cleaning, stats, backup, and font</h3>
						</div>
						<div class="uk-width-1-1">
							<label><input class="uk-checkbox" id="translate-to-english" type="checkbox"> Translate to English in clean config</label>
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="clean-replacements">Clean replacements</label>
							<textarea class="uk-textarea" id="clean-replacements" rows="6" placeholder="& =  and 
		FULL MOVIE =  "></textarea>
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="stats-exclude-words">Stats exclude words</label>
							<textarea class="uk-textarea" id="stats-exclude-words" rows="4" placeholder="comma,separated,words"></textarea>
						</div>
						<div class="uk-width-1-2">
							<label class="uk-form-label" for="backup-retention-days">Backup retention days</label>
							<input class="uk-input" id="backup-retention-days" type="number" min="0">
						</div>
						<div class="uk-width-1-2">
							<label class="uk-form-label" for="font-file">Font file</label>
							<input class="uk-input" id="font-file" type="text">
						</div>
						<div class="uk-width-1-2">
							<label class="uk-form-label" for="font-size">Font size</label>
							<input class="uk-input" id="font-size" type="number" min="0" step="0.1">
						</div>
						<div class="uk-width-1-2">
							<label class="uk-form-label" for="font-color">Font color</label>
							<input class="uk-input" id="font-color" type="text" placeholder="#FFFFFF">
						</div>
						<div class="uk-width-1-2">
							<label class="uk-form-label" for="font-shadow-color">Shadow color</label>
							<input class="uk-input" id="font-shadow-color" type="text">
						</div>
						<div class="uk-width-1-2">
							<label class="uk-form-label" for="font-glow-color">Glow color</label>
							<input class="uk-input" id="font-glow-color" type="text">
						</div>
						<div class="uk-width-1-3">
							<label class="uk-form-label" for="font-shadow-offset-x">Shadow X</label>
							<input class="uk-input" id="font-shadow-offset-x" type="number">
						</div>
						<div class="uk-width-1-3">
							<label class="uk-form-label" for="font-shadow-offset-y">Shadow Y</label>
							<input class="uk-input" id="font-shadow-offset-y" type="number">
						</div>
						<div class="uk-width-1-3">
							<label class="uk-form-label" for="font-glow-radius">Glow radius</label>
							<input class="uk-input" id="font-glow-radius" type="number" min="0">
						</div>
						<div class="uk-width-1-2">
							<label class="uk-form-label" for="font-glow-alpha">Glow alpha</label>
							<input class="uk-input" id="font-glow-alpha" type="number" min="0" max="1" step="0.01">
						</div>
						<div class="uk-width-1-2">
							<label class="uk-form-label" for="font-y-offset">Y offset</label>
							<input class="uk-input" id="font-y-offset" type="number">
						</div>
		            <div class="uk-width-1-1 uk-flex uk-flex-between uk-flex-middle uk-margin-small-top uk-flex-wrap uk-gap-small">
		              <span class="fp-footer-note">The server stays local to 127.0.0.1.</span>
		              <div class="uk-flex uk-gap-small uk-flex-wrap">
								<button class="uk-button uk-button-default fp-test-button" id="test-plex-connection" type="button">Test Plex connection <span class="fp-test-button-icon" id="test-plex-connection-icon" aria-hidden="true"></span></button>
		                <button class="uk-button uk-button-primary" type="submit">Save config</button>
		              </div>
		            </div>
          </form>
        </div>

        <div class="uk-card uk-card-body fp-card uk-animation-slide-left-small">
          <h2 class="fp-section-title">Runtime</h2>
          <dl class="uk-description-list uk-description-list-divider">
            <dt>Config</dt>
            <dd id="config-path" class="fp-path"></dd>
            <dt>Output</dt>
            <dd id="output-dir" class="fp-path"></dd>
            <dt>Logs</dt>
            <dd id="log-file" class="fp-path"></dd>
            <dt>Backups</dt>
            <dd id="backup-list"></dd>
          </dl>
        </div>
      </div>

      <div class="uk-width-expand@l">
        <ul uk-tab>
          <li><a href="#">Posters</a></li>
          <li><a href="#">Library</a></li>
          <li><a href="#">Collections</a></li>
          <li><a href="#">Safety</a></li>
        </ul>
        <ul class="uk-switcher uk-margin">
          <li>
            <div class="fp-actions-grid">
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Generate posters</h3>
                <form id="posters-form">
                  <label class="uk-form-label" for="poster-sections">Libraries</label>
                  <select class="uk-select" id="poster-sections" multiple size="8"></select>
                  <div class="uk-margin-small-top uk-grid-small" uk-grid>
                    <label><input class="uk-checkbox" id="poster-upload" type="checkbox"> Upload posters to Plex</label>
                    <label><input class="uk-checkbox" id="poster-label-types" type="checkbox"> Label type collections</label>
                    <label><input class="uk-checkbox" id="poster-trail" type="checkbox"> Dry run only</label>
                  </div>
                  <button class="uk-button uk-button-primary uk-margin-top" type="submit">Run poster workflow</button>
                </form>
              </div>
            </div>
          </li>
          <li>
            <div class="fp-actions-grid">
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Clean titles</h3>
                <form id="clean-form">
                  <label class="uk-form-label" for="clean-section">Library</label>
                  <select class="uk-select" id="clean-section"></select>
                  <div class="uk-margin-small-top uk-grid-small" uk-grid>
                    <label><input class="uk-checkbox" id="clean-translate" type="checkbox"> Translate before cleaning</label>
                    <label><input class="uk-checkbox" id="clean-trail" type="checkbox"> Dry run only</label>
                  </div>
                  <button class="uk-button uk-button-primary uk-margin-top" type="submit">Run clean</button>
                </form>
              </div>
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Translate titles</h3>
                <form id="translate-form">
                  <label class="uk-form-label" for="translate-section">Library</label>
                  <select class="uk-select" id="translate-section"></select>
                  <label class="uk-margin-small-top"><input class="uk-checkbox" id="translate-trail" type="checkbox"> Dry run only</label>
                  <button class="uk-button uk-button-primary uk-margin-top" type="submit">Run translation</button>
                </form>
              </div>
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Stats and labels</h3>
                <form id="stats-form" class="uk-margin-small-bottom">
                  <label class="uk-form-label" for="stats-section">Stats library</label>
                  <select class="uk-select" id="stats-section"></select>
                  <button class="uk-button uk-button-secondary uk-margin-top" type="submit">Run stats</button>
                </form>
                <hr>
                <form id="label-form" class="uk-grid-small" uk-grid>
                  <div class="uk-width-1-1">
                    <label class="uk-form-label" for="label-section">Label library</label>
                    <select class="uk-select" id="label-section"></select>
                  </div>
                  <div class="uk-width-1-1">
                    <label class="uk-form-label" for="label-find">Find text</label>
                    <input class="uk-input" id="label-find" type="text" placeholder="studio, actor, tag fragment">
                  </div>
                  <div class="uk-width-1-1">
                    <label class="uk-form-label" for="label-add">Labels</label>
                    <input class="uk-input" id="label-add" type="text" placeholder="comma,separated,labels">
                  </div>
                  <div class="uk-width-1-1">
                    <label class="uk-form-label" for="label-categories">Categories</label>
                    <input class="uk-input" id="label-categories" type="text" placeholder="optional override">
                  </div>
                  <div class="uk-width-1-1 uk-grid-small" uk-grid>
                    <label><input class="uk-checkbox" id="label-update-category" type="checkbox"> Update categories</label>
                    <label><input class="uk-checkbox" id="label-only-category" type="checkbox"> Categories only</label>
                    <label><input class="uk-checkbox" id="label-trail" type="checkbox"> Dry run only</label>
                  </div>
                  <div class="uk-width-1-1">
                    <button class="uk-button uk-button-primary" type="submit">Run labels</button>
                  </div>
                </form>
              </div>
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Clone library</h3>
                <form id="clone-form">
                  <label class="uk-form-label" for="clone-section">Source library</label>
                  <select class="uk-select" id="clone-section"></select>
                  <label class="uk-form-label uk-margin-small-top" for="clone-name">New library name</label>
                  <input class="uk-input" id="clone-name" type="text" placeholder="movies-clone">
                  <button class="uk-button uk-button-primary uk-margin-top" type="submit">Clone</button>
                </form>
              </div>
            </div>
          </li>
          <li>
            <div class="fp-actions-grid">
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Export and import</h3>
                <form id="export-form" class="uk-margin-small-bottom">
                  <label class="uk-form-label" for="export-section">Export library</label>
                  <select class="uk-select" id="export-section"></select>
                  <label class="uk-form-label uk-margin-small-top" for="export-file">Export file</label>
                  <input class="uk-input" id="export-file" type="text">
                  <button class="uk-button uk-button-secondary uk-margin-top" type="submit">Export collections</button>
                </form>
                <hr>
                <form id="import-form">
                  <label class="uk-form-label" for="import-section">Import target library</label>
                  <select class="uk-select" id="import-section"></select>
                  <label class="uk-form-label uk-margin-small-top" for="import-file">Import file</label>
                  <input class="uk-input" id="import-file" type="text">
                  <label class="uk-margin-small-top"><input class="uk-checkbox" id="import-trail" type="checkbox"> Dry run only</label>
                  <button class="uk-button uk-button-primary uk-margin-top" type="submit">Import collections</button>
                </form>
              </div>
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Collection maintenance</h3>
                <form id="inject-form" class="uk-margin-small-bottom">
                  <label class="uk-form-label" for="inject-section">Inject target library</label>
                  <select class="uk-select" id="inject-section"></select>
                  <label class="uk-margin-small-top"><input class="uk-checkbox" id="inject-trail" type="checkbox"> Dry run only</label>
                  <button class="uk-button uk-button-primary uk-margin-top" type="submit">Inject configured collections</button>
                </form>
                <form id="dupes-form" class="uk-margin-small-bottom">
                  <label class="uk-form-label" for="dupes-section">Duplicates library</label>
                  <select class="uk-select" id="dupes-section"></select>
                  <button class="uk-button uk-button-secondary uk-margin-top" type="submit">Audit duplicates</button>
                </form>
                <form id="delete-non-smart-form" class="uk-margin-small-bottom">
                  <label class="uk-form-label" for="delete-non-smart-section">Delete non-smart library</label>
                  <select class="uk-select" id="delete-non-smart-section"></select>
                  <label class="uk-margin-small-top"><input class="uk-checkbox" id="delete-non-smart-trail" type="checkbox"> Dry run only</label>
                  <button class="uk-button uk-button-danger uk-margin-top" type="submit">Delete non-smart collections</button>
                </form>
              </div>
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Path clean a collection</h3>
                <form id="path-clean-form">
                  <label class="uk-form-label" for="path-clean-section">Library</label>
                  <select class="uk-select" id="path-clean-section"></select>
                  <label class="uk-form-label uk-margin-small-top" for="path-clean-collection">Collection</label>
                  <select class="uk-select" id="path-clean-collection"></select>
                  <label class="uk-margin-small-top"><input class="uk-checkbox" id="path-clean-trail" type="checkbox"> Dry run only</label>
                  <button class="uk-button uk-button-primary uk-margin-top" type="submit">Path clean collection</button>
                </form>
              </div>
            </div>
          </li>
          <li>
            <div class="fp-actions-grid">
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Backup</h3>
                <p class="fp-muted">Archive config, templates, fonts, and selection state into the repo-level backups folder.</p>
                <form id="backup-form">
                  <button class="uk-button uk-button-primary" type="submit">Create backup</button>
                </form>
              </div>
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Restore</h3>
                <p class="fp-muted">Leave blank to restore the newest backup, or paste part of a filename to target a specific archive.</p>
                <form id="restore-form">
                  <label class="uk-form-label" for="restore-file">Backup filter</label>
                  <input class="uk-input" id="restore-file" type="text" placeholder="20260704 or frantic-postr-backup-host">
                  <label class="uk-margin-small-top"><input class="uk-checkbox" id="restore-trail" type="checkbox"> Dry run only</label>
                  <button class="uk-button uk-button-primary uk-margin-top" type="submit">Restore backup</button>
                </form>
              </div>
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Rollback</h3>
                <p class="fp-muted">Revert the most recent restore run using the generated rollback archive and manifest.</p>
                <form id="rollback-form">
                  <button class="uk-button uk-button-danger" type="submit">Rollback last restore</button>
                </form>
              </div>
            </div>
          </li>
        </ul>

        <div class="uk-card uk-card-body fp-card uk-margin-top uk-animation-slide-bottom-small">
          <div class="uk-flex uk-flex-between uk-flex-wrap uk-flex-middle">
            <div>
              <h2 class="fp-section-title">Operation log</h2>
              <p class="fp-muted uk-margin-small-top">Every action returns inline output so you can verify what ran without checking a terminal buffer.</p>
            </div>
            <span class="fp-chip" id="config-status-chip">Loading runtime status…</span>
          </div>
					<div class="fp-progress-wrap uk-margin-top">
						<div class="fp-progress-head">
							<p class="fp-progress-title" id="progress-label">No active operation</p>
							<span id="progress-count">0/0</span>
						</div>
						<progress id="progress-bar" class="fp-progress-bar" max="100" value="0"></progress>
						<div class="fp-progress-meta" id="progress-meta">Idle</div>
					</div>
          <pre id="action-log" class="fp-log uk-margin-top">No action has been run yet.</pre>
        </div>
      </div>
    </div>
  </div>

  <div id="about-modal" uk-modal>
    <div class="uk-modal-dialog uk-modal-body fp-card">
      <button class="uk-modal-close-default" type="button" uk-close></button>
      <h2 class="uk-modal-title">About {{.AppName}}</h2>
      <p class="fp-muted">A local control surface for the existing frantic-postr Go workflows. The UI stays on top of the same Plex-aware code paths used by the CLI.</p>
      <dl class="uk-description-list uk-description-list-divider">
        <dt>Version</dt>
        <dd>{{.About.Version}}</dd>
        <dt>Go runtime</dt>
        <dd>{{.About.GoVersion}}</dd>
        <dt>Module</dt>
        <dd>{{.About.Module}}</dd>
        <dt>Mode</dt>
        <dd>{{.About.Mode}}</dd>
        <dt>Started</dt>
        <dd>{{.About.StartedAt}}</dd>
        {{if .About.CommitHint}}<dt>Build hint</dt><dd>{{.About.CommitHint}}</dd>{{end}}
      </dl>
    </div>
  </div>

  <script>
	const state = {
		sections: [],
		backups: [],
		defaults: {},
		plexTestResetTimer: null,
		actionPollTimer: null,
		actionPollBusy: false,
		actionPollFast: 300,
		actionPollSlow: 1200,
		lastActionStatusHash: ''
	};

    const sectionSelectIds = [
      'clean-section', 'translate-section', 'stats-section', 'label-section', 'clone-section',
      'export-section', 'import-section', 'inject-section', 'dupes-section', 'delete-non-smart-section',
      'path-clean-section'
    ];

    function showBanner(type, message) {
      const banner = document.getElementById('status-banner');
	banner.className = 'fp-banner active ' + type;
      banner.textContent = message;
    }

		function showToast(type, message, timeout) {
			if (typeof UIkit === 'undefined' || !UIkit.notification) {
				return;
			}
			const status = type === 'error' ? 'danger' : type;
			UIkit.notification({
				message: message,
				status: status,
				pos: 'bottom-right',
				timeout: timeout || 4000
			});
		}

    function setLog(text) {
      document.getElementById('action-log').textContent = text || 'No output returned.';
    }

		function updateProgress(status) {
			const label = document.getElementById('progress-label');
			const count = document.getElementById('progress-count');
			const bar = document.getElementById('progress-bar');
			const meta = document.getElementById('progress-meta');
			if (!label || !count || !bar || !meta) {
				return;
			}
			const progress = status && status.progress ? status.progress : null;
			if (!status || !status.running) {
				if (progress && progress.total > 0) {
					label.textContent = progress.label || 'Completed';
					count.textContent = progress.current + '/' + progress.total;
					bar.value = progress.percent || 100;
					meta.textContent = status.error ? 'Failed' : 'Completed';
				} else {
					label.textContent = 'No active operation';
					count.textContent = '0/0';
					bar.value = 0;
					meta.textContent = 'Idle';
				}
				return;
			}
			if (progress) {
				label.textContent = progress.label || (status.action || 'processing');
				count.textContent = progress.current + '/' + progress.total;
				bar.value = progress.percent || 0;
				meta.textContent = 'Running ' + (status.action || 'operation') + '...';
				return;
			}
			label.textContent = status.action ? ('Running ' + status.action) : 'Running operation';
			count.textContent = '0/0';
			bar.value = 0;
			meta.textContent = 'Waiting for first progress update...';
		}

		function ensureLogPinnedBottom() {
			const logEl = document.getElementById('action-log');
			if (!logEl) {
				return;
			}
			logEl.scrollTop = logEl.scrollHeight;
		}

		async function pollActionStatus() {
			if (state.actionPollBusy) {
				return;
			}
			state.actionPollBusy = true;
			try {
				const response = await fetch('/api/action/status');
				if (!response.ok) {
					return;
				}
				const payload = await response.json();
				const currentHash = JSON.stringify({
					running: payload.running,
					action: payload.action,
					error: payload.error,
					message: payload.message,
					logs: payload.logs,
					progress: payload.progress
				});
				if (currentHash !== state.lastActionStatusHash) {
					state.lastActionStatusHash = currentHash;
					updateProgress(payload);
					if (payload.logs) {
						setLog(payload.logs);
						ensureLogPinnedBottom();
					}
				}
			} catch (error) {
				// Keep polling; transient issues should not break UX.
			} finally {
				state.actionPollBusy = false;
			}
		}

		function startActionPolling() {
			if (state.actionPollTimer) {
				return;
			}
			pollActionStatus();
			state.actionPollTimer = setInterval(pollActionStatus, state.actionPollFast);
		}

		function stopActionPolling() {
			if (!state.actionPollTimer) {
				return;
			}
			clearInterval(state.actionPollTimer);
			state.actionPollTimer = null;
			setTimeout(pollActionStatus, 0);
		}

    function updateStatusChip(valid, message) {
      const chip = document.getElementById('config-status-chip');
      chip.textContent = message;
      chip.style.background = valid ? 'rgba(45,139,87,0.12)' : 'rgba(157,47,47,0.12)';
      chip.style.color = valid ? 'var(--fp-ok)' : 'var(--fp-error)';
    }

		function clearPlexTestButtonState() {
			const button = document.getElementById('test-plex-connection');
			const icon = document.getElementById('test-plex-connection-icon');
			if (!button || !icon) {
				return;
			}
			button.classList.remove('is-loading', 'is-success', 'is-error');
			button.disabled = false;
			icon.textContent = '';
			if (state.plexTestResetTimer) {
				clearTimeout(state.plexTestResetTimer);
				state.plexTestResetTimer = null;
			}
		}

		function setPlexTestButtonState(nextState) {
			const button = document.getElementById('test-plex-connection');
			const icon = document.getElementById('test-plex-connection-icon');
			if (!button || !icon) {
				return;
			}
			clearPlexTestButtonState();
			if (nextState === 'loading') {
				button.classList.add('is-loading');
				button.disabled = true;
				icon.textContent = '...';
				return;
			}
			if (nextState === 'success') {
				button.classList.add('is-success');
				icon.textContent = '✓';
				state.plexTestResetTimer = setTimeout(clearPlexTestButtonState, 4500);
				return;
			}
			if (nextState === 'error') {
				button.classList.add('is-error');
				icon.textContent = '✕';
				state.plexTestResetTimer = setTimeout(clearPlexTestButtonState, 4500);
			}
		}

    function makeOption(section) {
      const option = document.createElement('option');
      option.value = section.key;
	option.textContent = section.title + ' (' + section.type + ')';
      return option;
    }

    function fillSectionSelect(selectId, allowPlaceholder) {
      const select = document.getElementById(selectId);
      if (!select) return;
      const previous = select.multiple ? Array.from(select.selectedOptions).map((option) => option.value) : select.value;
      select.innerHTML = '';
      if (allowPlaceholder) {
        const placeholder = document.createElement('option');
        placeholder.value = '';
        placeholder.textContent = state.sections.length ? 'Choose a library' : 'No libraries loaded';
        select.appendChild(placeholder);
      }
      state.sections.forEach((section) => select.appendChild(makeOption(section)));
      if (select.multiple) {
        previous.forEach((value) => {
          const match = Array.from(select.options).find((option) => option.value === value);
          if (match) match.selected = true;
        });
      } else if (previous) {
        select.value = previous;
      }
      if (!select.multiple && !select.value && state.sections.length && !allowPlaceholder) {
        select.value = state.sections[0].key;
      }
    }

    async function loadCollectionsForPathClean() {
      const sectionKey = document.getElementById('path-clean-section').value;
      const collectionSelect = document.getElementById('path-clean-collection');
      collectionSelect.innerHTML = '';
      if (!sectionKey) {
        return;
      }
      try {
		const response = await fetch('/api/sections/' + encodeURIComponent(sectionKey) + '/collections');
        const payload = await response.json();
        if (!response.ok) throw new Error(payload.error || 'Failed to load collections');
        payload.collections.forEach((collection) => {
          const option = document.createElement('option');
          option.value = collection.rating_key;
          option.textContent = collection.title;
          collectionSelect.appendChild(option);
        });
      } catch (error) {
        showBanner('error', error.message);
      }
    }

    async function refreshState() {
      const response = await fetch('/api/state');
      const payload = await response.json();
      if (!response.ok) {
        showBanner('error', payload.error || 'Failed to load state');
        return;
      }
      state.sections = payload.sections || [];
      state.backups = payload.backups || [];
      state.defaults = payload.defaults || {};

      document.getElementById('base-url').value = payload.plex.base_url || '';
      document.getElementById('token').value = payload.plex.token || '';
      document.getElementById('retries').value = payload.plex.retries || 1;
      document.getElementById('workers').value = payload.plex.workers || 1;
      document.getElementById('retry-base-ms').value = payload.plex.retry_base_ms || 500;
      document.getElementById('retry-max-ms').value = payload.plex.retry_max_ms || 30000;
	document.getElementById('output-dir-input').value = payload.general.output_dir || '';
	document.getElementById('log-file-input').value = payload.general.log_file || '';
	document.getElementById('translate-endpoint').value = payload.general.translate_endpoint || '';
	document.getElementById('translate-api-key').value = payload.general.translate_api_key || '';
	document.getElementById('translate-rate-limit').value = payload.general.translate_rate_limit_per_minute || 10;
	document.getElementById('template-image').value = payload.general.template_image || '';
	document.getElementById('type-template-image').value = payload.general.type_template_image || '';
	document.getElementById('studio-template-image').value = payload.general.studio_template_image || '';
	document.getElementById('admin-template-image').value = payload.general.admin_template_image || '';
	document.getElementById('type-collections-file').value = payload.general.type_collections_file || '';
	document.getElementById('studio-collections-file').value = payload.general.studio_collections_file || '';
	document.getElementById('admin-collections-file').value = payload.general.admin_collections_file || '';
	document.getElementById('plex-config-file').value = payload.general.plex_config || '';
	document.getElementById('label-config-file').value = payload.general.label_config || '';
	document.getElementById('collection-config-file').value = payload.general.collection_config || '';
	document.getElementById('translate-to-english').checked = !!payload.general.translate_to_english;
	document.getElementById('clean-replacements').value = payload.general.clean_replacements || '';
	document.getElementById('stats-exclude-words').value = payload.general.stats_exclude_words || '';
	document.getElementById('backup-retention-days').value = payload.general.backup_retention_days || 0;
	document.getElementById('font-file').value = payload.general.font_file || '';
	document.getElementById('font-size').value = payload.general.font_size || 0;
	document.getElementById('font-color').value = payload.general.font_color || '';
	document.getElementById('font-shadow-color').value = payload.general.font_shadow_color || '';
	document.getElementById('font-shadow-offset-x').value = payload.general.font_shadow_offset_x || 0;
	document.getElementById('font-shadow-offset-y').value = payload.general.font_shadow_offset_y || 0;
	document.getElementById('font-glow-color').value = payload.general.font_glow_color || '';
	document.getElementById('font-glow-radius').value = payload.general.font_glow_radius || 0;
	document.getElementById('font-glow-alpha').value = payload.general.font_glow_alpha || 0;
	document.getElementById('font-y-offset').value = payload.general.font_y_offset || 0;
      document.getElementById('config-path').textContent = payload.config_path;
      document.getElementById('output-dir').textContent = payload.output_dir;
      document.getElementById('log-file').textContent = payload.log_file;
      document.getElementById('export-file').value = payload.defaults.export_path || '';
      document.getElementById('import-file').value = payload.defaults.import_path || '';

      sectionSelectIds.forEach((id) => fillSectionSelect(id, false));
      fillSectionSelect('poster-sections', false);
      await loadCollectionsForPathClean();

      const backupList = document.getElementById('backup-list');
      if (!state.backups.length) {
        backupList.textContent = 'No backups found.';
      } else {
		backupList.innerHTML = state.backups.map((backup) => '<span class="fp-chip">' + backup.name + '</span>').join('');
      }

      if (payload.config_valid) {
		const sectionsSummary = payload.sections_error ? 'Config valid. Plex fetch issue: ' + payload.sections_error : 'Config valid. Libraries loaded: ' + state.sections.length;
        showBanner(payload.sections_error ? 'warn' : 'ok', sectionsSummary);
        updateStatusChip(true, sectionsSummary);
      } else {
        const errorMessage = payload.config_error || 'Config validation failed.';
        showBanner('warn', errorMessage);
        updateStatusChip(false, errorMessage);
      }
    }

    async function postJSON(url, body) {
      const response = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body || {})
      });
      const payload = await response.json();
      if (!response.ok || !payload.ok) {
        throw new Error(payload.error || 'Request failed');
      }
      return payload;
    }

    async function runAction(action, payload) {
	setLog('Running ' + action + '...');
	state.lastActionStatusHash = '';
	updateProgress({ running: true, action: action });
	startActionPolling();
      try {
		const result = await postJSON('/api/action/' + action, payload);
        setLog(result.logs || result.message || 'Action completed.');
		showBanner('ok', result.message || (action + ' completed.'));
        await refreshState();
      } catch (error) {
        setLog(error.message);
        showBanner('error', error.message);
		} finally {
			stopActionPolling();
      }
    }

    document.getElementById('refresh-state').addEventListener('click', refreshState);
    document.getElementById('path-clean-section').addEventListener('change', loadCollectionsForPathClean);

    document.getElementById('config-form').addEventListener('submit', async (event) => {
      event.preventDefault();
      try {
        const payload = {
          base_url: document.getElementById('base-url').value,
          token: document.getElementById('token').value,
          retries: Number(document.getElementById('retries').value),
          workers: Number(document.getElementById('workers').value),
          retry_base_ms: Number(document.getElementById('retry-base-ms').value),
					retry_max_ms: Number(document.getElementById('retry-max-ms').value),
					output_dir: document.getElementById('output-dir-input').value,
					log_file: document.getElementById('log-file-input').value,
					template_image: document.getElementById('template-image').value,
					type_template_image: document.getElementById('type-template-image').value,
					studio_template_image: document.getElementById('studio-template-image').value,
					admin_template_image: document.getElementById('admin-template-image').value,
					type_collections_file: document.getElementById('type-collections-file').value,
					studio_collections_file: document.getElementById('studio-collections-file').value,
					admin_collections_file: document.getElementById('admin-collections-file').value,
					plex_config: document.getElementById('plex-config-file').value,
					label_config: document.getElementById('label-config-file').value,
					collection_config: document.getElementById('collection-config-file').value,
					translate_to_english: document.getElementById('translate-to-english').checked,
					translate_endpoint: document.getElementById('translate-endpoint').value,
					translate_api_key: document.getElementById('translate-api-key').value,
					translate_rate_limit_per_minute: Number(document.getElementById('translate-rate-limit').value),
					clean_replacements: document.getElementById('clean-replacements').value,
					stats_exclude_words: document.getElementById('stats-exclude-words').value,
					backup_retention_days: Number(document.getElementById('backup-retention-days').value),
					font_file: document.getElementById('font-file').value,
					font_size: Number(document.getElementById('font-size').value),
					font_color: document.getElementById('font-color').value,
					font_shadow_color: document.getElementById('font-shadow-color').value,
					font_shadow_offset_x: Number(document.getElementById('font-shadow-offset-x').value),
					font_shadow_offset_y: Number(document.getElementById('font-shadow-offset-y').value),
					font_glow_color: document.getElementById('font-glow-color').value,
					font_glow_radius: Number(document.getElementById('font-glow-radius').value),
					font_glow_alpha: Number(document.getElementById('font-glow-alpha').value),
					font_y_offset: Number(document.getElementById('font-y-offset').value)
        };
        const result = await postJSON('/api/config', payload);
        setLog(result.logs || result.message || 'Configuration updated.');
        showBanner('ok', result.message || 'Configuration saved.');
				showToast('success', result.message || 'Configuration saved.', 3500);
        await refreshState();
      } catch (error) {
        setLog(error.message);
        showBanner('error', error.message);
				showToast('danger', error.message, 5000);
      }
    });

		document.getElementById('test-plex-connection').addEventListener('click', async () => {
			try {
				const payload = {
					base_url: document.getElementById('base-url').value,
					token: document.getElementById('token').value,
					retries: Number(document.getElementById('retries').value),
					workers: Number(document.getElementById('workers').value),
					retry_base_ms: Number(document.getElementById('retry-base-ms').value),
					retry_max_ms: Number(document.getElementById('retry-max-ms').value)
				};
				setPlexTestButtonState('loading');
				setLog('Testing Plex connection...');
				const result = await postJSON('/api/plex/test', payload);
				setLog(result.logs || result.message || 'Plex connection succeeded.');
				showBanner('ok', result.message || 'Plex connection succeeded.');
				setPlexTestButtonState('success');
				showToast('success', result.message || 'Plex connection succeeded.', 4500);
			} catch (error) {
				setLog(error.message);
				showBanner('error', error.message);
				setPlexTestButtonState('error');
				showToast('danger', error.message, 5000);
			}
		});

    document.getElementById('posters-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('gen-posters', {
        section_keys: Array.from(document.getElementById('poster-sections').selectedOptions).map((option) => option.value),
        upload_posters: document.getElementById('poster-upload').checked,
        label_types: document.getElementById('poster-label-types').checked,
        trail: document.getElementById('poster-trail').checked
      });
    });

    document.getElementById('clean-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('clean', {
        section_key: document.getElementById('clean-section').value,
        translate: document.getElementById('clean-translate').checked,
        trail: document.getElementById('clean-trail').checked
      });
    });

    document.getElementById('translate-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('translate', {
        section_key: document.getElementById('translate-section').value,
        trail: document.getElementById('translate-trail').checked
      });
    });

    document.getElementById('stats-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('stats', {
        section_key: document.getElementById('stats-section').value
      });
    });

    document.getElementById('label-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('label', {
        section_key: document.getElementById('label-section').value,
        find: document.getElementById('label-find').value,
        add: document.getElementById('label-add').value,
        categories: document.getElementById('label-categories').value,
        update_category: document.getElementById('label-update-category').checked,
        only_category: document.getElementById('label-only-category').checked,
        trail: document.getElementById('label-trail').checked
      });
    });

    document.getElementById('clone-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('clone', {
        section_key: document.getElementById('clone-section').value,
        clone_name: document.getElementById('clone-name').value
      });
    });

    document.getElementById('export-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('coll-export', {
        section_key: document.getElementById('export-section').value,
        coll_file: document.getElementById('export-file').value
      });
    });

    document.getElementById('import-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('coll-import', {
        section_key: document.getElementById('import-section').value,
        coll_file: document.getElementById('import-file').value,
        trail: document.getElementById('import-trail').checked
      });
    });

    document.getElementById('inject-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('coll-inject', {
        section_key: document.getElementById('inject-section').value,
        trail: document.getElementById('inject-trail').checked
      });
    });

    document.getElementById('dupes-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('coll-dupes', {
        section_key: document.getElementById('dupes-section').value
      });
    });

    document.getElementById('delete-non-smart-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('coll-delete-non-smart', {
        section_key: document.getElementById('delete-non-smart-section').value,
        trail: document.getElementById('delete-non-smart-trail').checked
      });
    });

    document.getElementById('path-clean-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('coll-path-clean', {
        section_key: document.getElementById('path-clean-section').value,
        collection_key: document.getElementById('path-clean-collection').value,
        trail: document.getElementById('path-clean-trail').checked
      });
    });

    document.getElementById('backup-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('backup', {});
    });

    document.getElementById('restore-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('restore', {
        restore_file: document.getElementById('restore-file').value,
        trail: document.getElementById('restore-trail').checked
      });
    });

    document.getElementById('rollback-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('rollback', {});
    });

		pollActionStatus();
		refreshState();
  </script>
</body>
</html>`))

var webHelpTemplate = template.Must(template.New("help").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.AppName}} Help</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@400;500;700&family=IBM+Plex+Mono:wght@400;500&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/uikit@3.23.8/dist/css/uikit.min.css">
  <style>
    body {
      background: linear-gradient(180deg, #f6eddc 0%, #f1e7d7 100%);
      color: #182126;
      font-family: "Space Grotesk", sans-serif;
      margin: 0;
    }
    .fp-help {
      max-width: 980px;
      margin: 0 auto;
      padding: 28px 18px 40px;
    }
    .fp-help-card {
      border-radius: 24px;
      background: rgba(255,255,255,0.8);
      padding: 28px;
      box-shadow: 0 18px 50px rgba(24,33,38,0.08);
      border: 1px solid rgba(24,33,38,0.1);
      margin-bottom: 18px;
    }
    code { font-family: "IBM Plex Mono", monospace; }
  </style>
</head>
<body>
  <main class="fp-help">
    <div class="fp-help-card">
      <div class="uk-flex uk-flex-between uk-flex-middle uk-flex-wrap uk-gap-small">
        <div>
          <p class="uk-text-meta" style="letter-spacing: 0.16em; text-transform: uppercase;">Useful tips</p>
          <h1 class="uk-heading-small uk-margin-small-top">{{.AppName}} web help</h1>
        </div>
        <a class="uk-button uk-button-default" href="/">Back to dashboard</a>
      </div>
      <p class="uk-text-large">The web UI is local-only, uses the same Go workflows as the CLI, and is best treated as an operations console rather than a separate product tier.</p>
    </div>

    <div class="fp-help-card">
      <h2>Recommended flow</h2>
      <ul class="uk-list uk-list-bullet">
        <li>Confirm the Plex URL and token first. A valid config unlocks library discovery and all Plex-backed actions.</li>
        <li>Use dry-run checkboxes before destructive actions such as cleaning titles, deleting non-smart collections, imports, and restores.</li>
        <li>Run stats or duplicate-collection audits before cleanup so you have fresh reports in the output folders.</li>
        <li>Use backup before large operations that touch multiple files or collections.</li>
      </ul>
    </div>

    <div class="fp-help-card">
      <h2>Action notes</h2>
      <ul class="uk-list uk-list-bullet">
        <li><code>Generate posters</code> can target multiple libraries at once and optionally upload finished posters back to Plex.</li>
        <li><code>Clean titles</code> can optionally translate titles first, then apply the configured cleanup replacements.</li>
        <li><code>Path clean</code> rewrites titles from collection item file paths and writes a CSV audit into <code>output/path-clean/</code>.</li>
        <li><code>Export collections</code> writes JSON under <code>output/collections-export/</code> when you provide only a filename.</li>
        <li><code>Restore</code> accepts a partial backup filename; leaving it blank restores the newest archive in non-interactive mode.</li>
      </ul>
    </div>

    <div class="fp-help-card">
      <h2>Troubleshooting</h2>
      <ul class="uk-list uk-list-bullet">
        <li>If the dashboard loads but libraries do not, the config is readable but the Plex connection is failing. Check URL, token, and network reachability.</li>
        <li>If an operation says config validation failed, fix missing template, font, or output paths before retrying that workflow.</li>
        <li>If a restore or rollback fails, review the operation log first, then inspect the latest files under <code>output/restore/</code>.</li>
      </ul>
    </div>
  </main>
</body>
</html>`))

func startWebServer(configPath string, port int, logger *AppLogger) error {
	server := &webServer{
		configPath: configPath,
		port:       port,
		logger:     logger,
		startedAt:  time.Now(),
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	logger.Successf("web UI available at http://%s", addr)
	logger.Infof("web help available at http://%s/help", addr)
	return http.ListenAndServe(addr, server.routes())
}

func (s *webServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/help", s.handleHelp)
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/action/status", s.handleActionStatus)
	mux.HandleFunc("/api/config", s.handleConfigUpdate)
	mux.HandleFunc("/api/plex/test", s.handlePlexTest)
	mux.HandleFunc("/api/action/", s.handleAction)
	mux.HandleFunc("/api/sections/", s.handleSectionCollections)
	return mux
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
	s.logger.Infof("web state requested: config=%s", s.configPath)
	displayCfg, err := loadWebDisplayConfig(s.configPath, s.logger)
	if err != nil {
		s.logger.Errorf("web state display load failed: config=%s err=%v", s.configPath, err)
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	runtimeCfg, err := loadWebRuntimeConfig(s.configPath, s.logger)
	if err != nil {
		s.logger.Errorf("web state load failed: config=%s err=%v", s.configPath, err)
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
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
			TemplateImage:             displayCfg.TemplateImage,
			TypeTemplateImage:         displayCfg.TypeTemplateImage,
			StudioTemplateImage:       displayCfg.StudioTemplateImage,
			AdminTemplateImage:        displayCfg.AdminTemplateImage,
			TypeCollectionsFile:       displayCfg.TypeCollectionsFile,
			StudioCollectionsFile:     displayCfg.StudioCollectionsFile,
			AdminCollectionsFile:      displayCfg.AdminCollectionsFile,
			OutputDir:                 displayCfg.OutputDir,
			LogFile:                   displayCfg.LogFile,
			PlexConfigFile:            displayCfg.PlexConfigFile,
			LabelConfigFile:           displayCfg.LabelConfigFile,
			CollectionConfigFile:      displayCfg.CollectionConfigFile,
			TranslateToEnglish:        displayCfg.Clean.TranslateToEnglish,
			TranslateEndpoint:         displayCfg.Clean.TranslateEndpoint,
			TranslateAPIKey:           displayCfg.Clean.TranslateAPIKey,
			TranslateRateLimitMinute:  displayCfg.Clean.TranslateRateLimitPerMinute,
			CleanReplacements:         serializeCleanReplacements(displayCfg.Clean.Replacements),
			StatsExcludeWords:         strings.Join(displayCfg.Stats.ExcludeWords, ", "),
			BackupRetentionDays:       displayCfg.Backup.RetentionDays,
			FontFile:                  displayCfg.Font.File,
			FontSize:                  displayCfg.Font.Size,
			FontColor:                 displayCfg.Font.Color,
			FontShadowColor:           displayCfg.Font.ShadowColor,
			FontShadowOffsetX:         displayCfg.Font.ShadowOffsetX,
			FontShadowOffsetY:         displayCfg.Font.ShadowOffsetY,
			FontGlowColor:             displayCfg.Font.GlowColor,
			FontGlowRadius:            displayCfg.Font.GlowRadius,
			FontGlowAlpha:             displayCfg.Font.GlowAlpha,
			FontYOffset:               displayCfg.Font.YOffset,
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
	writeJSON(w, http.StatusOK, response)
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

func (s *webServer) resetActionRuntime(action string) {
	s.actionMu.Lock()
	defer s.actionMu.Unlock()
	s.actionRun = webActionRuntime{
		action:    action,
		startedAt: time.Now(),
		running:   true,
		ok:        false,
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

func (s *webServer) completeActionRuntime(ok bool, message, errText string) {
	s.actionMu.Lock()
	defer s.actionMu.Unlock()
	s.actionRun.running = false
	s.actionRun.completedAt = time.Now()
	s.actionRun.ok = ok
	s.actionRun.message = strings.TrimSpace(message)
	s.actionRun.err = strings.TrimSpace(errText)
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
		Running: s.actionRun.running,
		Action:  s.actionRun.action,
		OK:      s.actionRun.ok,
		Message: s.actionRun.message,
		Error:   s.actionRun.err,
		Logs:    s.actionRun.logs.String(),
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

	requestLogger, logBuffer := newBufferedLogger(s.logger)
	requestLogger.logCallback = func(line string) {
		s.appendActionLog(line)
	}
	requestLogger.progressCallback = func(label string, current, total int, final bool) {
		s.updateActionProgress(label, current, total, final)
	}
	err := s.runAction(action, request, requestLogger)
	response := webActionResponse{OK: err == nil, Logs: logBuffer.String()}
	if err != nil {
		response.Error = err.Error()
		s.completeActionRuntime(false, "", err.Error())
		writeJSON(w, http.StatusBadRequest, response)
		return
	}
	response.Message = fmt.Sprintf("%s completed.", action)
	s.completeActionRuntime(true, response.Message, "")
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
			return runPosterGeneration(client, cfg, selectedSections, request.UploadPosters, request.LabelTypes, logger)
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

func runPosterGeneration(client *http.Client, cfg Config, selectedSections []plexSection, uploadPosters bool, labelTypeCollectionItems bool, logger *AppLogger) error {
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
			if err := processCollections(client, cfg, section.Title, collections, uploadPosters, labelTypeCollectionItems, logger); err != nil {
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
	reportPath := uniqueCollectionReportPath(cfg.OutputDir, "duplicate-collections", time.Now())
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
	reportPath := uniqueCollectionReportPath(cfg.OutputDir, "deleted-non-smart-collections", time.Now())
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
	reportPath := uniquePathCleanReportPath(cfg.OutputDir, time.Now())
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
		quiet = base.quiet
		if base.console != nil {
			consoleWriter = io.MultiWriter(base.console.Writer(), buffer)
		}
		if base.file != nil {
			fileWriter = io.MultiWriter(base.file.Writer(), buffer)
		}
	}
	return &AppLogger{
		console: log.New(consoleWriter, "", log.LstdFlags|log.Lmicroseconds),
		file:    log.New(fileWriter, "", log.LstdFlags|log.Lmicroseconds),
		quiet:   quiet,
	}, buffer
}

func withTrailMode(enabled bool, fn func() error) error {
	previous := trailModeEnabled
	trailModeEnabled = enabled
	defer func() {
		trailModeEnabled = previous
	}()
	return fn()
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
