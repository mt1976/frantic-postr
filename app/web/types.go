package web

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mt1976/frantic-postr/app/core"
)

type AppLogger = core.AppLogger
type Config = core.Config
type PlexSection = core.PlexSection
type PlexCollection = core.PlexCollection
type plexSection = core.PlexSection
type plexSectionDetail = core.PlexSectionDetail
type plexSectionPref = core.PlexSectionPref
type plexCollection = core.PlexCollection
type plexLibraryItem = core.PlexLibraryItem
type collectionLookupConfig = core.CollectionLookupConfig
type backupCandidate = core.BackupCandidate
type collectionTransferFile = core.CollectionTransferFile
type collectionTransferRecord = core.CollectionTransferRecord
type collectionInventoryEntry = core.CollectionInventoryEntry

var appVersion = "development"

func SetAppVersion(version string) {
	trimmed := strings.TrimSpace(version)
	if trimmed != "" {
		appVersion = trimmed
	}
}

type webServer struct {
	configPath string
	port       int
	logger     *AppLogger
	startedAt  time.Time
	busy       atomic.Bool
	actionMu   sync.Mutex
	actionRun  webActionRuntime
	actionStop context.CancelFunc
}

type webActionRuntime struct {
	action      string
	startedAt   time.Time
	completedAt time.Time
	running     bool
	ok          bool
	canceled    bool
	stopAsked   bool
	message     string
	err         string
	outputFile  string
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
	Canceled    bool                       `json:"canceled,omitempty"`
	StopAsked   bool                       `json:"stop_asked,omitempty"`
	Message     string                     `json:"message,omitempty"`
	Error       string                     `json:"error,omitempty"`
	OutputFile  string                     `json:"output_file,omitempty"`
	DownloadURL string                     `json:"download_url,omitempty"`
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
	TemplateImage            string  `json:"template_image"`
	TypeTemplateImage        string  `json:"type_template_image"`
	StudioTemplateImage      string  `json:"studio_template_image"`
	AdminTemplateImage       string  `json:"admin_template_image"`
	TypeCollectionsFile      string  `json:"type_collections_file"`
	StudioCollectionsFile    string  `json:"studio_collections_file"`
	AdminCollectionsFile     string  `json:"admin_collections_file"`
	OutputDir                string  `json:"output_dir"`
	LogFile                  string  `json:"log_file"`
	PlexConfigFile           string  `json:"plex_config"`
	LabelConfigFile          string  `json:"label_config"`
	CollectionConfigFile     string  `json:"collection_config"`
	TranslateToEnglish       bool    `json:"translate_to_english"`
	TranslateEndpoint        string  `json:"translate_endpoint"`
	TranslateAPIKey          string  `json:"translate_api_key"`
	TranslateRateLimitMinute int     `json:"translate_rate_limit_per_minute"`
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
	Sections      []PlexSection      `json:"sections"`
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
	BaseURL                  string  `json:"base_url"`
	Token                    string  `json:"token"`
	Retries                  int     `json:"retries"`
	Workers                  int     `json:"workers"`
	RetryBaseMs              int     `json:"retry_base_ms"`
	RetryMaxMs               int     `json:"retry_max_ms"`
	OutputDir                string  `json:"output_dir"`
	LogFile                  string  `json:"log_file"`
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
	TranslateEndpoint        string  `json:"translate_endpoint"`
	TranslateAPIKey          string  `json:"translate_api_key"`
	TranslateRateLimitMinute int     `json:"translate_rate_limit_per_minute"`
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
	SectionKey         string   `json:"section_key"`
	SectionKeys        []string `json:"section_keys"`
	CollectionKey      string   `json:"collection_key"`
	UploadPosters      bool     `json:"upload_posters"`
	MissingPostersOnly bool     `json:"missing_posters_only"`
	LabelTypes         bool     `json:"label_types"`
	Trail              bool     `json:"trail"`
	Translate          bool     `json:"translate"`
	Find               string   `json:"find"`
	Add                string   `json:"add"`
	Categories         string   `json:"categories"`
	UpdateCategory     bool     `json:"update_category"`
	OnlyCategory       bool     `json:"only_category"`
	CollFile           string   `json:"coll_file"`
	RestoreFile        string   `json:"restore_file"`
	CloneName          string   `json:"clone_name"`
}

type webActionResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
	Logs    string `json:"logs,omitempty"`
}

type webConfigContentResponse struct {
	Scope   string `json:"scope"`
	Path    string `json:"path,omitempty"`
	Content string `json:"content"`
}

type webConfigContentUpdateRequest struct {
	Scope   string `json:"scope"`
	Content string `json:"content"`
}

type webTemplatePreviewRequest struct {
	TemplateKind        string  `json:"template_kind"`
	SampleText          string  `json:"sample_text"`
	TemplateImage       string  `json:"template_image"`
	TypeTemplateImage   string  `json:"type_template_image"`
	StudioTemplateImage string  `json:"studio_template_image"`
	AdminTemplateImage  string  `json:"admin_template_image"`
	FontFile            string  `json:"font_file"`
	FontSize            float64 `json:"font_size"`
	FontColor           string  `json:"font_color"`
	FontShadowColor     string  `json:"font_shadow_color"`
	FontShadowOffsetX   int     `json:"font_shadow_offset_x"`
	FontShadowOffsetY   int     `json:"font_shadow_offset_y"`
	FontGlowColor       string  `json:"font_glow_color"`
	FontGlowRadius      int     `json:"font_glow_radius"`
	FontGlowAlpha       float64 `json:"font_glow_alpha"`
	FontYOffset         int     `json:"font_y_offset"`
}

type webTemplatePreviewResponse struct {
	OK           bool   `json:"ok"`
	TemplateKind string `json:"template_kind"`
	TemplatePath string `json:"template_path"`
	SampleText   string `json:"sample_text"`
	ImageDataURL string `json:"image_data_url"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}

type webFileOption struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	ModTime string `json:"mod_time,omitempty"`
	Size    int64  `json:"size"`
}

type webFileListResponse struct {
	Scope   string          `json:"scope"`
	Default string          `json:"default,omitempty"`
	Files   []webFileOption `json:"files"`
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

type WebConfigUpdateRequest = webConfigUpdateRequest
type WebActionRequest = webActionRequest
type WebActionResponse = webActionResponse
type WebActionStatusResponse = webActionStatusResponse
type WebStateResponse = webStateResponse
type WebTemplatePreviewRequest = webTemplatePreviewRequest
type WebTemplatePreviewResponse = webTemplatePreviewResponse
