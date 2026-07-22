package core

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	fcolor "github.com/fatih/color"
)

type Config struct {
	Plex struct {
		BaseURL     string `toml:"base_url"`
		Token       string `toml:"token"`
		Retries     int    `toml:"retries"`
		Workers     int    `toml:"workers"`
		RetryBaseMs int    `toml:"retry_base_ms"`
		RetryMaxMs  int    `toml:"retry_max_ms"`
	} `toml:"plex"`
	Label struct {
		Lookups []labelLookupConfig `toml:"lookup"`
	} `toml:"label"`
	Collection struct {
		Lookups []collectionLookupConfig `toml:"lookup"`
	} `toml:"collection"`
	Clean struct {
		Replacements                map[string]string `toml:"replacements"`
		TranslateToEnglish          bool              `toml:"translate_to_english"`
		TranslateEndpoint           string            `toml:"translate_endpoint"`
		TranslateAPIHTTPAddress     string            `toml:"translate_api_http_address"`
		TranslateAPIKey             string            `toml:"translate_api_key"`
		TranslateRateLimitPerMinute int               `toml:"translate_rate_limit_per_minute"`
	} `toml:"clean"`
	Stats struct {
		ExcludeWords []string `toml:"exclude_words"`
	} `toml:"stats"`
	Backup struct {
		RetentionDays int `toml:"retention_days"`
	} `toml:"backup"`
	TemplateImage         string              `toml:"template_image"`
	TypeTemplateImage     string              `toml:"type_template_image"`
	StudioTemplateImage   string              `toml:"studio_template_image"`
	AdminTemplateImage    string              `toml:"admin_template_image"`
	TypeCollectionsFile   string              `toml:"type_collections_file"`
	StudioCollectionsFile string              `toml:"studio_collections_file"`
	AdminCollectionsFile  string              `toml:"admin_collections_file"`
	OutputDir             string              `toml:"output_dir"`
	LogFile               string              `toml:"log_file"`
	PlexConfigFile        string              `toml:"plex_config"`
	LabelConfigFile       string              `toml:"label_config"`
	CollectionConfigFile  string              `toml:"collection_config"`
	CollectionBaseURI     string              `toml:"-"`
	TypeCollectionSet     map[string]struct{} `toml:"-"`
	StudioCollectionSet   map[string]struct{} `toml:"-"`
	AdminCollectionSet    map[string]struct{} `toml:"-"`
	Font                  struct {
		File          string  `toml:"file"`
		Size          float64 `toml:"size"`
		Color         string  `toml:"color"`
		ShadowColor   string  `toml:"shadow_color"`
		ShadowOffsetX int     `toml:"shadow_offset_x"`
		ShadowOffsetY int     `toml:"shadow_offset_y"`
		GlowColor     string  `toml:"glow_color"`
		GlowRadius    int     `toml:"glow_radius"`
		GlowAlpha     float64 `toml:"glow_alpha"`
		YOffset       int     `toml:"y_offset"`
	} `toml:"font"`
}

type plexSectionsResponse struct {
	Directories []plexSection `xml:"Directory"`
}

type plexCollectionsResponse struct {
	Directories []plexCollection `xml:"Directory"`
}

type plexSectionAllResponse struct {
	Size     int               `xml:"size,attr"`
	Metadata []plexLibraryItem `xml:"Metadata"`
	Videos   []plexLibraryItem `xml:"Video"`
}

type plexCollectionDetailResponse struct {
	Directories []struct {
		RatingKey string `xml:"ratingKey,attr"`
		Title     string `xml:"title,attr"`
		GUID      string `xml:"guid,attr"`
		Subtype   string `xml:"subtype,attr"`
		Smart     int    `xml:"smart,attr"`
		Content   string `xml:"content,attr"`
	} `xml:"Directory"`
}

type plexSectionDetailResponse struct {
	Directories []plexSectionDetail `xml:"Directory"`
}

type plexSectionDetail struct {
	Key       string                `xml:"key,attr"`
	Title     string                `xml:"title,attr"`
	Type      string                `xml:"type,attr"`
	Agent     string                `xml:"agent,attr"`
	Scanner   string                `xml:"scanner,attr"`
	Language  string                `xml:"language,attr"`
	Locations []plexSectionLocation `xml:"Location"`
}

type plexSectionLocation struct {
	Path string `xml:"path,attr"`
}

type plexSectionPrefsResponse struct {
	Settings []plexSectionPref `xml:"Setting"`
}

type plexSectionPref struct {
	ID    string `xml:"id,attr"`
	Value string `xml:"value,attr"`
}

type plexSection struct {
	Key   string `xml:"key,attr" json:"key"`
	Title string `xml:"title,attr" json:"title"`
	Type  string `xml:"type,attr" json:"type"`
}

type cleanReportRow struct {
	RatingKey       string
	TitleBefore     string
	TitleAfter      string
	SortTitleBefore string
	SortTitleAfter  string
}

type cleanItemResult struct {
	ratingKey       string
	beforeTitle     string
	afterTitle      string
	beforeSortTitle string
	afterSortTitle  string
	updateSortTitle bool
	skipped         bool
	err             error
}

type labelItemResult struct {
	displayTitle      string
	ratingKey         string
	labelsUpdated     bool
	categoriesUpdated bool
	skipped           bool
	err               error
	// before/after snapshots for the report
	labelsBefore     string
	labelsAfter      string
	categoriesBefore string
	categoriesAfter  string
}

type labelReportRow struct {
	RatingKey        string
	Title            string
	LabelsBefore     string
	LabelsAfter      string
	CategoriesBefore string
	CategoriesAfter  string
}

type plexCollection struct {
	RatingKey string `xml:"ratingKey,attr"`
	Title     string `xml:"title,attr"`
	GUID      string `xml:"guid,attr"`
	Thumb     string `xml:"thumb,attr"`
}

type plexLibraryItem struct {
	RatingKey     string      `xml:"ratingKey,attr"`
	Title         string      `xml:"title,attr"`
	SortTitle     string      `xml:"titleSort,attr"`
	OriginalTitle string      `xml:"originalTitle,attr"`
	Labels        []plexLabel `xml:"Label"`
	Genres        []plexLabel `xml:"Genre"`
	Media         []plexMedia `xml:"Media"`
}

type plexMedia struct {
	Parts []plexPart `xml:"Part"`
}

type plexPart struct {
	File string `xml:"file,attr"`
}

type plexLabel struct {
	Tag string `xml:"tag,attr"`
}

type LabelConfig struct {
	Label struct {
		Lookups []labelLookupConfig `toml:"lookup"`
	} `toml:"label"`
}

type CollectionConfig struct {
	BaseURI    string `toml:"base_uri"`
	Collection struct {
		Lookups []collectionLookupConfig `toml:"lookup"`
	} `toml:"collection"`
}

type labelLookupConfig struct {
	TitleContains    string   `toml:"title_contains"`
	TitleContainsAny []string `toml:"title_contains_any"`
	Find             string   `toml:"find"`
	Labels           []string `toml:"labels"`
	Categories       []string `toml:"categories"`
	UpdateCategory   bool     `toml:"update_category"`
	OnlyCategory     bool     `toml:"only_category"`
}

type collectionLookupConfig struct {
	Title   string `toml:"title"`
	Smart   bool   `toml:"smart"`
	Content string `toml:"content"`
}

type selectionMemory struct {
	SectionKeys []string `json:"section_keys"`
}

type backupCandidate struct {
	Name    string
	Path    string
	ModTime time.Time
	Host    string
}

type restoreLogRow struct {
	Path   string
	Action string
	Detail string
}

type rollbackManifestEntry struct {
	Path         string `json:"path"`
	ExistedPrior bool   `json:"existed_prior"`
}

type rollbackManifest struct {
	RestoreTimestamp string                  `json:"restore_timestamp"`
	Entries          []rollbackManifestEntry `json:"entries"`
}

type lastRestoreState struct {
	RestoreTimestamp string `json:"restore_timestamp"`
	BackupArchive    string `json:"backup_archive"`
	RollbackZip      string `json:"rollback_zip"`
	ManifestFile     string `json:"manifest_file"`
	RolledBack       bool   `json:"rolled_back"`
}

const (
	selectionMemoryFile       = ".frantic-postr-selection.json"
	collectionTransferDirName = "collections-export"
	backupDirName             = "backups"
	restoreLogDirName         = "restore"
	lastRestoreStateFileName  = "last-restore-state.json"
)

var (
	ColorOutputEnabled     = true
	TrailModeEnabled       = false
	TranslateRateLimitMu   sync.Mutex
	NextTranslateRequestAt time.Time
	StdinReader            = bufio.NewReader(os.Stdin)
	PromptScreenModeLabel  = "Interactive"
	requestContextMu       sync.RWMutex
	requestContext         = context.Background()
)

func SetRequestContext(ctx context.Context) func() {
	if ctx == nil {
		ctx = context.Background()
	}
	requestContextMu.Lock()
	previous := requestContext
	requestContext = ctx
	requestContextMu.Unlock()
	return func() {
		requestContextMu.Lock()
		requestContext = previous
		requestContextMu.Unlock()
	}
}

func CurrentRequestContext() context.Context {
	requestContextMu.RLock()
	defer requestContextMu.RUnlock()
	if requestContext == nil {
		return context.Background()
	}
	return requestContext
}

const AppDisplayName = "frantic-postr"

// VideoExtRe matches a dot-prefixed video container extension, case-insensitively.
var VideoExtRe = regexp.MustCompile(`(?i)\.(mp4|mov|mpg|mpeg|mkv|avi|wmv|flv|webm|m4v|3gp|ts|vob|rm|rmvb|f4v|divx|xvid)\b`)

var CollectionSectionKeyRe = regexp.MustCompile(`(?:/library/sections/|%2Flibrary%2Fsections%2F|%2flibrary%2fsections%2f)([0-9]+)(?:/|%2F|%2f)`)

type AppLogger struct {
	Console          *log.Logger
	File             *log.Logger
	Quiet            bool
	LogCallback      func(line string)
	ProgressCallback func(label string, current, total int, final bool)
}

type ProgressTracker struct {
	Console       *log.Logger
	renderConsole bool
	onUpdate      func(label string, current, total int, final bool)
	label         string
	total         int
	current       int
	mu            sync.Mutex
}

func NewProgressTracker(logger *AppLogger, label string, total int) *ProgressTracker {
	if logger == nil || total <= 0 {
		return nil
	}
	tracker := &ProgressTracker{
		label:    label,
		total:    total,
		onUpdate: logger.ProgressCallback,
	}
	if logger.Console != nil && logger.Quiet {
		tracker.Console = logger.Console
		tracker.renderConsole = true
	}
	if !tracker.renderConsole && tracker.onUpdate == nil {
		return nil
	}
	tracker.render(false)
	return tracker
}

func (p *ProgressTracker) render(final bool) {
	if p == nil {
		return
	}
	current := 0
	total := 0
	line := ""

	p.mu.Lock()
	defer p.mu.Unlock()
	barWidth := 24
	if p.total <= 0 {
		return
	}
	if p.current > p.total {
		p.current = p.total
	}
	filled := (p.current * barWidth) / p.total
	if filled < 0 {
		filled = 0
	}
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("#", filled) + strings.Repeat("-", barWidth-filled)
	percent := (p.current * 100) / p.total
	if percent > 100 {
		percent = 100
	}
	line = fmt.Sprintf("%s [%s] %d/%d (%d%%)", p.label, bar, p.current, p.total, percent)
	current = p.current
	total = p.total

	if p.onUpdate != nil {
		p.onUpdate(p.label, current, total, final)
	}
	if p.Console == nil || !p.renderConsole {
		return
	}
	if final {
		p.Console.Println(line)
		return
	}
	p.Console.Printf("\r%s", line)
}

func (p *ProgressTracker) Advance() {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.current++
	p.mu.Unlock()
	p.render(false)
}

func (p *ProgressTracker) Finish() {
	if p == nil {
		return
	}
	p.mu.Lock()
	if p.current < p.total {
		p.current = p.total
	}
	p.mu.Unlock()
	p.render(true)
}

func (l *AppLogger) log(level, message string) {
	plain := fmt.Sprintf("%s %s", level, message)
	if l.LogCallback != nil {
		l.LogCallback(plain)
	}
	if l.File != nil {
		l.File.Println(plain)
	}
	if l.Console != nil && (!l.Quiet || level == "SUCCESS" || level == "WARNING" || level == "ERROR") {
		l.Console.Printf("%s %s", ColorizeLevel(level), message)
	}
}

func (l *AppLogger) Printf(format string, args ...any) {
	l.log("INFO", fmt.Sprintf(format, args...))
}

func (l *AppLogger) Println(args ...any) {
	l.log("INFO", strings.TrimSpace(fmt.Sprintln(args...)))
}

func (l *AppLogger) Infof(format string, args ...any) {
	l.log("INFO", fmt.Sprintf(format, args...))
}

func (l *AppLogger) Successf(format string, args ...any) {
	l.log("SUCCESS", fmt.Sprintf(format, args...))
}

func (l *AppLogger) Warningf(format string, args ...any) {
	l.log("WARNING", fmt.Sprintf(format, args...))
}

func (l *AppLogger) APIf(format string, args ...any) {
	l.log("API", fmt.Sprintf(format, args...))
}

func (l *AppLogger) Errorf(format string, args ...any) {
	l.log("ERROR", fmt.Sprintf(format, args...))
}

func (l *AppLogger) Matchf(format string, args ...any) {
	l.log("MATCH", fmt.Sprintf(format, args...))
}

func (l *AppLogger) Fatalf(format string, args ...any) {
	l.log("ERROR", fmt.Sprintf(format, args...))
	os.Exit(1)
}

func (l *AppLogger) Fatal(args ...any) {
	l.log("ERROR", strings.TrimSpace(fmt.Sprintln(args...)))
	os.Exit(1)
}

func ColorizeLevel(level string) string {
	if !ColorOutputEnabled {
		return level
	}
	switch level {
	case "ERROR":
		return fcolor.New(fcolor.FgRed, fcolor.Bold).Sprint(level)
	case "SUCCESS":
		return fcolor.New(fcolor.FgGreen, fcolor.Bold).Sprint(level)
	case "WARNING":
		return fcolor.New(fcolor.FgYellow, fcolor.Bold).Sprint(level)
	case "API":
		return fcolor.New(fcolor.FgCyan, fcolor.Bold).Sprint(level)
	case "MATCH":
		return fcolor.New(fcolor.FgMagenta, fcolor.Bold).Sprint(level)
	default:
		return fcolor.New(fcolor.FgHiWhite, fcolor.Bold).Sprint(level)
	}
}

type collectionTransferFile struct {
	Version       int                        `json:"version"`
	ExportedAtUTC string                     `json:"exported_at_utc"`
	SourceLibrary plexSection                `json:"source_library"`
	Collections   []collectionTransferRecord `json:"collections"`
}

type collectionTransferRecord struct {
	Title     string `json:"title"`
	GUID      string `json:"guid,omitempty"`
	RatingKey string `json:"rating_key,omitempty"`
	Subtype   string `json:"subtype,omitempty"`
	Smart     bool   `json:"smart"`
	Content   string `json:"content,omitempty"`
}

type collectionInventoryEntry struct {
	Title      string
	RatingKey  string
	SectionKey string
	GUID       string
	Smart      bool
	ItemCount  int
}

type collectionDuplicateRow struct {
	Title          string
	RatingKey      string
	ItemCount      int
	Smart          bool
	DuplicateCount int
}

type collectionDeleteRow struct {
	Title     string
	RatingKey string
	ItemCount int
	Smart     bool
	Status    string
}

type pathCleanReportRow struct {
	Collection  string
	RatingKey   string
	FilePath    string
	TitleBefore string
	TitleAfter  string
}

type statsWordCount struct {
	Word  string
	Count int
}

const (
	SelectionMemoryFile       = selectionMemoryFile
	CollectionTransferDirName = collectionTransferDirName
	BackupDirName             = backupDirName
	RestoreLogDirName         = restoreLogDirName
	LastRestoreStateFileName  = lastRestoreStateFileName
)

type LabelLookupConfig = labelLookupConfig
type CollectionLookupConfig = collectionLookupConfig
type PlexSectionsResponse = plexSectionsResponse
type PlexCollectionsResponse = plexCollectionsResponse
type PlexSectionAllResponse = plexSectionAllResponse
type PlexCollectionDetailResponse = plexCollectionDetailResponse
type PlexSectionDetailResponse = plexSectionDetailResponse
type PlexSectionDetail = plexSectionDetail
type PlexSectionLocation = plexSectionLocation
type PlexSectionPrefsResponse = plexSectionPrefsResponse
type PlexSectionPref = plexSectionPref
type PlexSection = plexSection
type CleanReportRow = cleanReportRow
type CleanItemResult = cleanItemResult
type LabelItemResult = labelItemResult
type LabelReportRow = labelReportRow
type PlexCollection = plexCollection
type PlexLibraryItem = plexLibraryItem
type PlexMedia = plexMedia
type PlexPart = plexPart
type PlexLabel = plexLabel
type SelectionMemory = selectionMemory
type BackupCandidate = backupCandidate
type RestoreLogRow = restoreLogRow
type RollbackManifestEntry = rollbackManifestEntry
type RollbackManifest = rollbackManifest
type LastRestoreState = lastRestoreState
type CollectionTransferFile = collectionTransferFile
type CollectionTransferRecord = collectionTransferRecord
type CollectionInventoryEntry = collectionInventoryEntry
type CollectionDuplicateRow = collectionDuplicateRow
type CollectionDeleteRow = collectionDeleteRow
type PathCleanReportRow = pathCleanReportRow
type StatsWordCount = statsWordCount
