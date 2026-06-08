package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"math/rand"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/abadojack/whatlanggo"
	fcolor "github.com/fatih/color"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
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
	TemplateImage        string `toml:"template_image"`
	OutputDir            string `toml:"output_dir"`
	LogFile              string `toml:"log_file"`
	LabelConfigFile      string `toml:"label_config"`
	CollectionConfigFile string `toml:"collection_config"`
	CollectionBaseURI    string `toml:"-"`
	Font                 struct {
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
	Key   string `xml:"key,attr"`
	Title string `xml:"title,attr"`
	Type  string `xml:"type,attr"`
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

const selectionMemoryFile = ".frantic-postr-selection.json"

var (
	colorOutputEnabled     = true
	trailModeEnabled       = false
	translateRateLimitMu   sync.Mutex
	nextTranslateRequestAt time.Time
)

// videoExtRe matches a dot-prefixed video container extension, case-insensitively.
var videoExtRe = regexp.MustCompile(`(?i)\.(mp4|mov|mpg|mpeg|mkv|avi|wmv|flv|webm|m4v|3gp|ts|vob|rm|rmvb|f4v|divx|xvid)\b`)

var collectionSectionKeyRe = regexp.MustCompile(`(?:/library/sections/|%2Flibrary%2Fsections%2F|%2flibrary%2fsections%2f)([0-9]+)(?:/|%2F|%2f)`)

type AppLogger struct {
	console *log.Logger
	file    *log.Logger
}

func (l *AppLogger) log(level, message string) {
	plain := fmt.Sprintf("%s %s", level, message)
	if l.file != nil {
		l.file.Println(plain)
	}
	if l.console != nil {
		l.console.Printf("%s %s", colorizeLevel(level), message)
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

func colorizeLevel(level string) string {
	if !colorOutputEnabled {
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

func main() {
	configPath := flag.String("config", "config.toml", "Path to config file")
	noColor := flag.Bool("no-color", false, "Disable ANSI colors in terminal output")
	trailMode := flag.Bool("trail", false, "Process as normal but do not write updates to Plex")
	trialMode := flag.Bool("trial", false, "Alias for -trail")
	upload := flag.Bool("upload", false, "Upload generated posters to Plex collections")
	collExport := flag.Bool("coll-export", false, "Export all collections (including smart filters) from a selected library")
	collImport := flag.Bool("coll-import", false, "Import collections from -coll-file into a selected library")
	collImpot := flag.Bool("coll-impot", false, "Alias for -coll-import")
	cloneLibraryMode := flag.Bool("clone", false, "Clone a selected library (settings + path mappings) with a new name")
	labelMode := flag.Bool("label", false, "Scan a selected library and add labels to items with titles matching -find")
	colInject := flag.Bool("col-inject", false, "Inject smart collections from collections.toml into a selected library")
	updateCategoryMode := flag.Bool("update-category", false, "When used with -label, also add values from -add to item category tags")
	onlyCategoryMode := flag.Bool("only-category", false, "When used with -label, update only category tags from -add and skip label updates")
	cleanMode := flag.Bool("clean", false, "Clean titles in a selected library by removing unsafe characters")
	translateMode := flag.Bool("translate", false, "Translate non-English titles to English in a selected library; combine with -clean to translate then clean")
	labelFind := flag.String("find", "", "Case-insensitive title text to search for when using -label")
	labelAdd := flag.String("add", "", "Comma-separated labels to add when using -label")
	collFile := flag.String("coll-file", "collections-export.json", "Path to the collections import/export file")
	flag.Parse()
	if *noColor {
		colorOutputEnabled = false
		fcolor.NoColor = true
	}
	if *trailMode || *trialMode {
		trailModeEnabled = true
	}

	importMode := *collImport || *collImpot
	translateOnlyMode := *translateMode && !*cleanMode
	modeCount := 0
	if *collExport {
		modeCount++
	}
	if importMode {
		modeCount++
	}
	if *cloneLibraryMode {
		modeCount++
	}
	if *labelMode {
		modeCount++
	}
	if *colInject {
		modeCount++
	}
	if *cleanMode {
		modeCount++
	}
	if translateOnlyMode {
		modeCount++
	}
	if modeCount > 1 {
		log.Fatal("invalid flags: use only one mode among -coll-export, -coll-import/-coll-impot, -clone, -label, -col-inject, -clean, -translate")
	}
	if *translateMode && (*cloneLibraryMode || *collExport || importMode || *labelMode || *colInject) {
		log.Fatal("invalid flags: -translate can only be used by itself or together with -clean")
	}
	if (*cloneLibraryMode || *collExport || importMode || *labelMode || *colInject || *cleanMode || translateOnlyMode) && *upload {
		log.Fatal("invalid flags: -upload is not used with clone/import/export modes")
	}
	labelsToAdd, err := parseLabelList(*labelAdd)
	if err != nil {
		log.Fatalf("invalid -add labels: %v", err)
	}
	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger, closeLogger, err := setupLogger(cfg.LogFile)
	if err != nil {
		log.Fatalf("failed to setup logger: %v", err)
	}
	defer closeLogger()

	logger.Printf("startup: frantic-postr config=%s", *configPath)
	logConfig(logger, cfg)
	effectiveUpdateCategoryMode := *updateCategoryMode
	effectiveOnlyCategoryMode := *onlyCategoryMode
	if !*labelMode && (effectiveUpdateCategoryMode || effectiveOnlyCategoryMode) {
		logger.Errorf("invalid flags: -update-category and -only-category only work with -label; ignoring")
		effectiveUpdateCategoryMode = false
		effectiveOnlyCategoryMode = false
	}
	if effectiveOnlyCategoryMode && effectiveUpdateCategoryMode {
		logger.Warningf("label mode: -only-category takes precedence over -update-category")
	}
	logger.Printf("config: no_color=%t trail=%t upload=%t clone=%t label=%t col_inject=%t update_category=%t only_category=%t clean=%t translate=%t coll_export=%t coll_import=%t coll_file=%s", *noColor, trailModeEnabled, *upload, *cloneLibraryMode, *labelMode, *colInject, effectiveUpdateCategoryMode, effectiveOnlyCategoryMode, *cleanMode, *translateMode, *collExport, importMode, *collFile)

	client := &http.Client{Timeout: 30 * time.Second}
	sections, err := fetchSections(client, cfg, logger)
	if err != nil {
		logger.Fatalf("failed to fetch libraries: %v", err)
	}
	if len(sections) == 0 {
		logger.Fatal("no libraries returned by Plex")
	}
	if *cloneLibraryMode {
		if err := cloneSelectedLibrary(client, cfg, sections, logger); err != nil {
			logger.Fatalf("library clone failed: %v", err)
		}
		logger.Println("shutdown: frantic-postr completed")
		return
	}
	if *labelMode {
		hasCLIArgs := strings.TrimSpace(*labelFind) != "" || len(labelsToAdd) > 0

		// Validate before prompting for a library.
		if !hasCLIArgs && len(cfg.Label.Lookups) == 0 {
			logger.Fatal("invalid flags: -label requires either -find with -add or one or more [[label.lookup]] entries in config")
		}
		if hasCLIArgs {
			if strings.TrimSpace(*labelFind) == "" || len(labelsToAdd) == 0 {
				logger.Fatal("invalid flags: when using -label CLI matching, both -find and -add are required")
			}
		}

		// Log the full plan before asking the user to select a library.
		if hasCLIArgs {
			logger.Infof("label mode plan: CLI find=%q labels=%v categories=%v update_category=%t only_category=%t",
				*labelFind, labelsToAdd, labelsToAdd, effectiveUpdateCategoryMode, effectiveOnlyCategoryMode)
		}
		for i, lookup := range cfg.Label.Lookups {
			var finds []string
			if strings.TrimSpace(lookup.TitleContains) != "" {
				finds = append(finds, lookup.TitleContains)
			}
			finds = append(finds, lookup.TitleContainsAny...)
			if strings.TrimSpace(lookup.Find) != "" {
				finds = append(finds, lookup.Find)
			}
			logger.Infof("label mode plan: lookup %d/%d finds=%v labels=%v categories=%v update_category=%t only_category=%t",
				i+1, len(cfg.Label.Lookups), finds, lookup.Labels, lookup.Categories, lookup.UpdateCategory, lookup.OnlyCategory)
		}

		// Select the library once — not once per lookup.
		selectedSection, err := selectSingleSection(sections)
		if err != nil {
			logger.Fatalf("label mode: library selection failed: %v", err)
		}
		logger.Infof("label mode: selected library=%s (%s)", selectedSection.Title, selectedSection.Key)

		if hasCLIArgs {
			categoriesToAdd := labelsToAdd
			if err := labelMatchingItems(client, cfg, selectedSection, []string{*labelFind}, labelsToAdd, categoriesToAdd, effectiveUpdateCategoryMode, effectiveOnlyCategoryMode, logger); err != nil {
				logger.Fatalf("label mode failed: %v", err)
			}
		}

		for i, lookup := range cfg.Label.Lookups {
			var finds []string
			if strings.TrimSpace(lookup.TitleContains) != "" {
				finds = append(finds, lookup.TitleContains)
			}
			finds = append(finds, lookup.TitleContainsAny...)
			if strings.TrimSpace(lookup.Find) != "" {
				finds = append(finds, lookup.Find)
			}
			logger.Infof("label mode: running config lookup %d/%d finds=%d labels=%d categories=%d", i+1, len(cfg.Label.Lookups), len(finds), len(lookup.Labels), len(lookup.Categories))
			if err := labelMatchingItems(client, cfg, selectedSection, finds, lookup.Labels, lookup.Categories, lookup.UpdateCategory, lookup.OnlyCategory, logger); err != nil {
				logger.Fatalf("label mode lookup %d failed: %v", i+1, err)
			}
		}

		logger.Println("shutdown: frantic-postr completed")
		return
	}
	if *colInject {
		if err := injectCollections(client, cfg, sections, logger); err != nil {
			logger.Fatalf("collection inject failed: %v", err)
		}
		logger.Println("shutdown: frantic-postr completed")
		return
	}
	if *cleanMode {
		if err := cleanLibraryTitles(client, cfg, sections, *translateMode, logger); err != nil {
			logger.Fatalf("clean mode failed: %v", err)
		}
		logger.Println("shutdown: frantic-postr completed")
		return
	}
	if translateOnlyMode {
		if err := translateLibraryTitles(client, cfg, sections, logger); err != nil {
			logger.Fatalf("translate mode failed: %v", err)
		}
		logger.Println("shutdown: frantic-postr completed")
		return
	}
	if *collExport {
		if err := exportCollections(client, cfg, sections, *collFile, logger); err != nil {
			logger.Fatalf("collection export failed: %v", err)
		}
		logger.Println("shutdown: frantic-postr completed")
		return
	}
	if importMode {
		if err := importCollections(client, cfg, sections, *collFile, logger); err != nil {
			logger.Fatalf("collection import failed: %v", err)
		}
		logger.Println("shutdown: frantic-postr completed")
		return
	}

	selectedSections, err := selectSections(sections)
	if err != nil {
		logger.Fatalf("failed to select library: %v", err)
	}
	for _, section := range selectedSections {
		logger.Printf("selected library: %s (%s)", section.Title, section.Key)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(selectedSections))
	for _, section := range selectedSections {
		section := section
		wg.Add(1)
		go func() {
			defer wg.Done()
			collections, err := fetchCollections(client, cfg, section.Key, logger)
			if err != nil {
				errCh <- fmt.Errorf("fetch collections for %s: %w", section.Title, err)
				return
			}
			logger.Printf("collections fetched: library=%s count=%d", section.Title, len(collections))

			if err := processCollections(client, cfg, section.Title, collections, *upload, logger); err != nil {
				errCh <- fmt.Errorf("process %s: %w", section.Title, err)
				return
			}
		}()
	}
	wg.Wait()
	close(errCh)

	if len(errCh) > 0 {
		errs := make([]string, 0, len(errCh))
		for err := range errCh {
			errs = append(errs, err.Error())
		}
		logger.Fatalf("processing failed: %s", strings.Join(errs, "; "))
	}

	logger.Println("shutdown: frantic-postr completed")
}

func loadConfig(path string) (Config, error) {
	var cfg Config
	bytes, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := toml.Unmarshal(bytes, &cfg); err != nil {
		return cfg, err
	}
	if cfg.LabelConfigFile != "" {
		var labelCfg LabelConfig
		if err := loadSupplementalConfig(path, cfg.LabelConfigFile, "label_config", &labelCfg); err != nil {
			return cfg, err
		}
		cfg.Label.Lookups = append(cfg.Label.Lookups, labelCfg.Label.Lookups...)
	}
	if cfg.CollectionConfigFile != "" {
		var collectionCfg CollectionConfig
		if err := loadSupplementalConfig(path, cfg.CollectionConfigFile, "collection_config", &collectionCfg); err != nil {
			return cfg, err
		}
		collectionCfg.BaseURI = strings.TrimSpace(collectionCfg.BaseURI)
		cfg.Collection.Lookups = append(cfg.Collection.Lookups, collectionCfg.Collection.Lookups...)
		cfg.CollectionBaseURI = collectionCfg.BaseURI
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "output"
	}
	if cfg.LogFile == "" {
		cfg.LogFile = "frantic-postr.log"
	}
	if cfg.Font.Size <= 0 {
		cfg.Font.Size = 64
	}
	if cfg.Font.Color == "" {
		cfg.Font.Color = "#FFFFFF"
	}
	if cfg.Font.ShadowColor == "" {
		cfg.Font.ShadowColor = "#000000"
	}
	if cfg.Font.GlowColor == "" {
		cfg.Font.GlowColor = "#000000"
	}
	if cfg.Font.GlowRadius < 0 {
		cfg.Font.GlowRadius = 0
	}
	if cfg.Font.GlowAlpha <= 0 || cfg.Font.GlowAlpha > 1 {
		cfg.Font.GlowAlpha = 0.35
	}
	if cfg.Plex.BaseURL == "" || cfg.Plex.Token == "" {
		return cfg, errors.New("plex config not found: set plex.base_url and plex.token")
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
	for i := range cfg.Label.Lookups {
		lookup := &cfg.Label.Lookups[i]
		lookup.TitleContains = strings.TrimSpace(lookup.TitleContains)
		lookup.Find = strings.TrimSpace(lookup.Find)
		matchList, err := normalizeFindList(lookup.TitleContainsAny)
		if err != nil {
			return cfg, fmt.Errorf("label.lookup[%d] title_contains_any: %w", i, err)
		}
		if len(matchList) == 0 {
			singleMatch := lookup.TitleContains
			if singleMatch == "" {
				singleMatch = lookup.Find
			}
			singleMatch = strings.TrimSpace(singleMatch)
			if singleMatch == "" {
				return cfg, fmt.Errorf("label.lookup[%d]: title_contains, title_contains_any (or find) is required", i)
			}
			matchList = []string{singleMatch}
		}
		lookup.TitleContainsAny = matchList
		lookup.TitleContains = lookup.TitleContainsAny[0]

		labels, err := normalizeTagList(lookup.Labels)
		if err != nil {
			return cfg, fmt.Errorf("label.lookup[%d] labels: %w", i, err)
		}
		categories, err := normalizeTagList(lookup.Categories)
		if err != nil {
			return cfg, fmt.Errorf("label.lookup[%d] categories: %w", i, err)
		}

		lookup.Labels = labels
		lookup.Categories = categories
		if len(lookup.Categories) == 0 {
			lookup.Categories = lookup.Labels
		}
		if len(lookup.Labels) == 0 && len(lookup.Categories) == 0 {
			return cfg, fmt.Errorf("label.lookup[%d]: at least one label or category is required", i)
		}
	}
	for i := range cfg.Collection.Lookups {
		lookup := &cfg.Collection.Lookups[i]
		lookup.Title = strings.TrimSpace(lookup.Title)
		lookup.Content = strings.TrimSpace(lookup.Content)
		if lookup.Title == "" {
			return cfg, fmt.Errorf("collection.lookup[%d]: title is required", i)
		}
		if lookup.Content != "" {
			lookup.Smart = true
		}
		if lookup.Smart && lookup.Content == "" {
			return cfg, fmt.Errorf("collection.lookup[%d]: smart collections require content", i)
		}
		lookup.Content = normalizeCollectionLookupContent(lookup.Content)
	}
	if strings.TrimSpace(cfg.Clean.TranslateAPIHTTPAddress) != "" {
		cfg.Clean.TranslateEndpoint = cfg.Clean.TranslateAPIHTTPAddress
	}
	if strings.TrimSpace(cfg.Clean.TranslateEndpoint) == "" {
		cfg.Clean.TranslateEndpoint = "https://libretranslate.com/translate"
	}
	if cfg.Clean.TranslateRateLimitPerMinute <= 0 {
		cfg.Clean.TranslateRateLimitPerMinute = 10
	}
	if _, err := url.ParseRequestURI(cfg.Plex.BaseURL); err != nil {
		return cfg, fmt.Errorf("invalid plex.base_url: %w", err)
	}
	if cfg.TemplateImage == "" {
		return cfg, errors.New("config must set template_image")
	}
	if err := requireFileExists("template_image", cfg.TemplateImage); err != nil {
		return cfg, err
	}
	if err := requireDirExists("output_dir", cfg.OutputDir); err != nil {
		return cfg, err
	}
	if cfg.Font.File != "" {
		if err := requireFileExists("font.file", cfg.Font.File); err != nil {
			return cfg, err
		}
	}
	logDir := filepath.Dir(cfg.LogFile)
	if logDir != "" && logDir != "." {
		if err := requireDirExists("log_file directory", logDir); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

func loadSupplementalConfig(basePath, supplementalPath, fieldName string, target any) error {
	if strings.TrimSpace(supplementalPath) == "" {
		return nil
	}
	resolvedPath := supplementalPath
	if !filepath.IsAbs(resolvedPath) {
		resolvedPath = filepath.Join(filepath.Dir(basePath), resolvedPath)
	}
	bytes, err := os.ReadFile(resolvedPath)
	if err != nil {
		return fmt.Errorf("%s: %w", fieldName, err)
	}
	if err := toml.Unmarshal(bytes, target); err != nil {
		return fmt.Errorf("%s: %w", fieldName, err)
	}
	return nil
}

func requireFileExists(name, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s not found: %s", name, path)
		}
		return fmt.Errorf("failed to read %s %s: %w", name, path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s must be a file, got directory: %s", name, path)
	}
	return nil
}

func requireDirExists(name, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s not found: %s", name, path)
		}
		return fmt.Errorf("failed to read %s %s: %w", name, path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s must be a directory, got file: %s", name, path)
	}
	return nil
}

func setupLogger(path string) (*AppLogger, func(), error) {
	runLogPath := uniqueRunLogPath(path, time.Now())
	f, err := os.OpenFile(runLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}
	logger := &AppLogger{
		console: log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds),
		file:    log.New(f, "", log.LstdFlags|log.Lmicroseconds),
	}
	logger.Infof("log file: %s", runLogPath)
	return logger, func() { _ = f.Close() }, nil
}

func uniqueCleanReportPath(outputDir string, now time.Time) string {
	timestamp := now.Format("20060102-150405")
	return filepath.Join(outputDir, "clean", fmt.Sprintf("clean-%s.csv", timestamp))
}

func uniqueLabelReportPath(outputDir string, now time.Time) string {
	timestamp := now.Format("20060102-150405")
	return filepath.Join(outputDir, "labels", fmt.Sprintf("labels-%s.csv", timestamp))
}

func writeLabelReport(path string, rows []labelReportRow) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create label report dir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create label report file: %w", err)
	}
	defer f.Close()
	csvField := func(s string) string {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	fmt.Fprintf(f, "%s|%s|%s|%s|%s|%s\n",
		csvField("RatingKey"), csvField("Title"),
		csvField("LabelsBefore"), csvField("LabelsAfter"),
		csvField("CategoriesBefore"), csvField("CategoriesAfter"))
	for _, r := range rows {
		fmt.Fprintf(f, "%s|%s|%s|%s|%s|%s\n",
			csvField(r.RatingKey), csvField(r.Title),
			csvField(r.LabelsBefore), csvField(r.LabelsAfter),
			csvField(r.CategoriesBefore), csvField(r.CategoriesAfter))
	}
	return nil
}

func writeCleanReport(path string, rows []cleanReportRow) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create clean report dir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create clean report file: %w", err)
	}
	defer f.Close()
	csvField := func(s string) string {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	fmt.Fprintf(f, "%s|%s|%s|%s|%s\n",
		csvField("RatingKey"), csvField("TitleBefore"), csvField("TitleAfter"),
		csvField("SortTitleBefore"), csvField("SortTitleAfter"))
	for _, r := range rows {
		fmt.Fprintf(f, "%s|%s|%s|%s|%s\n",
			csvField(r.RatingKey), csvField(r.TitleBefore), csvField(r.TitleAfter),
			csvField(r.SortTitleBefore), csvField(r.SortTitleAfter))
	}
	return nil
}

func uniqueRunLogPath(path string, now time.Time) string {
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	name := strings.TrimSuffix(filepath.Base(path), ext)
	if name == "" {
		name = "frantic-postr"
	}
	timestamp := now.Format("20060102-150405")
	fileName := fmt.Sprintf("%s-%s", name, timestamp)
	if ext != "" {
		fileName += ext
	}
	if dir == "" || dir == "." {
		return fileName
	}
	return filepath.Join(dir, fileName)
}

func logConfig(logger *AppLogger, cfg Config) {
	logger.Printf("config: plex.base_url=%s", cfg.Plex.BaseURL)
	logger.Printf("config: plex.token=*** (len=%d)", len(cfg.Plex.Token))
	logger.Printf("config: plex.retries=%d", cfg.Plex.Retries)
	logger.Printf("config: plex.workers=%d", cfg.Plex.Workers)
	logger.Printf("config: plex.retry_base_ms=%d", cfg.Plex.RetryBaseMs)
	logger.Printf("config: plex.retry_max_ms=%d", cfg.Plex.RetryMaxMs)
	logger.Printf("config: label_config=%s", cfg.LabelConfigFile)
	logger.Printf("config: label.lookup_count=%d", len(cfg.Label.Lookups))
	logger.Printf("config: collection_config=%s", cfg.CollectionConfigFile)
	logger.Printf("config: collection_base_uri=%s", cfg.CollectionBaseURI)
	logger.Printf("config: collection.lookup_count=%d", len(cfg.Collection.Lookups))
	logger.Printf("config: clean.translate_to_english=%t", cfg.Clean.TranslateToEnglish)
	logger.Printf("config: clean.translate_endpoint=%s", cfg.Clean.TranslateEndpoint)
	logger.Printf("config: clean.translate_rate_limit_per_minute=%d", cfg.Clean.TranslateRateLimitPerMinute)
	logger.Printf("config: template_image=%s", cfg.TemplateImage)
	logger.Printf("config: output_dir=%s", cfg.OutputDir)
	logger.Printf("config: log_file=%s", cfg.LogFile)
	logger.Printf("config: font.file=%s", cfg.Font.File)
	logger.Printf("config: font.size=%v", cfg.Font.Size)
	logger.Printf("config: font.color=%s", cfg.Font.Color)
	logger.Printf("config: font.shadow_color=%s", cfg.Font.ShadowColor)
	logger.Printf("config: font.shadow_offset_x=%d", cfg.Font.ShadowOffsetX)
	logger.Printf("config: font.shadow_offset_y=%d", cfg.Font.ShadowOffsetY)
	logger.Printf("config: font.glow_color=%s", cfg.Font.GlowColor)
	logger.Printf("config: font.glow_radius=%d", cfg.Font.GlowRadius)
	logger.Printf("config: font.glow_alpha=%v", cfg.Font.GlowAlpha)
	logger.Printf("config: font.y_offset=%d", cfg.Font.YOffset)
}

func fetchSections(client *http.Client, cfg Config, logger *AppLogger) ([]plexSection, error) {
	endpoint := strings.TrimRight(cfg.Plex.BaseURL, "/") + "/library/sections"
	logger.APIf("plex call: GET %s", endpoint)
	respBody, err := doPlexGET(client, endpoint, cfg, logger)
	if err != nil {
		return nil, err
	}
	var out plexSectionsResponse
	if err := xml.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	logger.Printf("plex result: sections=%d", len(out.Directories))
	return out.Directories, nil
}

func fetchCollectionDetails(client *http.Client, cfg Config, ratingKey string, logger *AppLogger) (collectionTransferRecord, error) {
	endpoint := strings.TrimRight(cfg.Plex.BaseURL, "/") + "/library/collections/" + ratingKey
	logger.APIf("plex call: GET %s", endpoint)
	respBody, err := doPlexGET(client, endpoint, cfg, logger)
	if err != nil {
		return collectionTransferRecord{}, err
	}
	var out plexCollectionDetailResponse
	if err := xml.Unmarshal(respBody, &out); err != nil {
		return collectionTransferRecord{}, err
	}
	if len(out.Directories) == 0 {
		return collectionTransferRecord{}, fmt.Errorf("collection details not found for ratingKey=%s", ratingKey)
	}
	detail := out.Directories[0]
	return collectionTransferRecord{
		Title:     normalizeCollectionName(detail.Title),
		GUID:      detail.GUID,
		RatingKey: detail.RatingKey,
		Subtype:   detail.Subtype,
		Smart:     detail.Smart == 1,
		Content:   detail.Content,
	}, nil
}

func fetchSectionDetail(client *http.Client, cfg Config, sectionKey string, logger *AppLogger) (plexSectionDetail, error) {
	endpoint := strings.TrimRight(cfg.Plex.BaseURL, "/") + "/library/sections/" + sectionKey
	logger.APIf("plex call: GET %s", endpoint)
	respBody, err := doPlexGET(client, endpoint, cfg, logger)
	if err != nil {
		return plexSectionDetail{}, err
	}
	var out plexSectionDetailResponse
	if err := xml.Unmarshal(respBody, &out); err != nil {
		return plexSectionDetail{}, err
	}
	if len(out.Directories) == 0 {
		return plexSectionDetail{}, fmt.Errorf("library details not found for section key=%s", sectionKey)
	}
	return out.Directories[0], nil
}

func fetchSectionPreferences(client *http.Client, cfg Config, sectionKey string, logger *AppLogger) ([]plexSectionPref, error) {
	endpoint := strings.TrimRight(cfg.Plex.BaseURL, "/") + "/library/sections/" + sectionKey + "/prefs"
	logger.APIf("plex call: GET %s", endpoint)
	respBody, err := doPlexGET(client, endpoint, cfg, logger)
	if err != nil {
		return nil, err
	}
	var out plexSectionPrefsResponse
	if err := xml.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	return out.Settings, nil
}

func fetchCollections(client *http.Client, cfg Config, sectionKey string, logger *AppLogger) ([]plexCollection, error) {
	endpoint := strings.TrimRight(cfg.Plex.BaseURL, "/") + "/library/sections/" + sectionKey + "/collections"
	logger.APIf("plex call: GET %s", endpoint)
	respBody, err := doPlexGET(client, endpoint, cfg, logger)
	if err != nil {
		return nil, err
	}
	var out plexCollectionsResponse
	if err := xml.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	collections := make([]plexCollection, 0, len(out.Directories))
	for _, collection := range out.Directories {
		collection.Title = normalizeCollectionName(collection.Title)
		if collection.Title != "" {
			collections = append(collections, collection)
		}
	}
	return collections, nil
}

func fetchSectionItems(client *http.Client, cfg Config, sectionKey string, logger *AppLogger) ([]plexLibraryItem, error) {
	allItems := make([]plexLibraryItem, 0, 256)
	start := 0
	const pageSize = 200
	for {
		endpoint := strings.TrimRight(cfg.Plex.BaseURL, "/") + "/library/sections/" + sectionKey + "/all"
		u, err := url.Parse(endpoint)
		if err != nil {
			return nil, err
		}
		q := u.Query()
		q.Set("X-Plex-Container-Start", strconv.Itoa(start))
		q.Set("X-Plex-Container-Size", strconv.Itoa(pageSize))
		u.RawQuery = q.Encode()

		logger.APIf("plex call: GET %s", endpoint)
		respBody, err := doPlexGET(client, u.String(), cfg, logger)
		if err != nil {
			return nil, err
		}
		var out plexSectionAllResponse
		if err := xml.Unmarshal(respBody, &out); err != nil {
			return nil, err
		}
		chunk := make([]plexLibraryItem, 0, len(out.Metadata)+len(out.Videos))
		chunk = append(chunk, out.Metadata...)
		chunk = append(chunk, out.Videos...)
		allItems = append(allItems, chunk...)

		if len(chunk) < pageSize {
			break
		}
		start += len(chunk)
	}
	return allItems, nil
}

func doPlexGET(client *http.Client, endpoint string, cfg Config, logger *AppLogger) ([]byte, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("X-Plex-Token", cfg.Plex.Token)
	u.RawQuery = q.Encode()
	if logger != nil {
		logger.APIf("plex curl: %s", curlCommand("GET", u.String()))
	}

	resp, err := doRequestWithRetry(client, cfg.Plex.Retries, cfg.Plex.RetryBaseMs, cfg.Plex.RetryMaxMs, logger, "GET "+u.String(), func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, u.String(), nil)
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("plex request failed: status=%s body=%s", resp.Status, string(body))
	}
	return io.ReadAll(resp.Body)
}

// retryBackoff returns the duration to wait before attempt n (0-based).
// Uses full-jitter exponential backoff: sleep = random(0, min(base*2^n, cap)).
func retryBackoff(attempt, baseMs, maxMs int) time.Duration {
	cap := math.Min(float64(maxMs), float64(baseMs)*math.Pow(2, float64(attempt)))
	// math/rand is fine here — this is not security-sensitive.
	jitter := cap * randFloat64()
	return time.Duration(jitter) * time.Millisecond
}

// randFloat64 is a package-level variable so tests can override it.
var randFloat64 = func() float64 {
	// math/rand is fine here — jitter is not security-sensitive.
	//nolint:gosec
	return rand.Float64()
}

func doRequestWithRetry(client *http.Client, retries, baseMs, maxMs int, logger *AppLogger, operation string, build func() (*http.Request, error)) (*http.Response, error) {
	if retries < 0 {
		retries = 0
	}
	for attempt := 0; attempt <= retries; attempt++ {
		req, err := build()
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err == nil {
			return resp, nil
		}
		if !isTimeoutError(err) || attempt == retries {
			return nil, err
		}
		delay := retryBackoff(attempt, baseMs, maxMs)
		if logger != nil {
			logger.Warningf("%s timed out, retry %d/%d in %s: %v", operation, attempt+1, retries, delay.Round(time.Millisecond), err)
		}
		time.Sleep(delay)
	}
	return nil, fmt.Errorf("request retries exhausted")
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "context deadline exceeded")
}

func curlCommand(method, requestURL string) string {
	return fmt.Sprintf("curl -X %s %s", shellQuote(method), shellQuote(requestURL))
}

func curlFileUploadCommand(method, endpoint, token, filePath string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Sprintf("curl -X %s -F %s %s", shellQuote(method), shellQuote("file=@"+filePath), shellQuote(endpoint))
	}
	q := u.Query()
	q.Set("X-Plex-Token", token)
	u.RawQuery = q.Encode()
	return fmt.Sprintf("curl -X %s -F %s %s", shellQuote(method), shellQuote("file=@"+filePath), shellQuote(u.String()))
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func selectSections(sections []plexSection) ([]plexSection, error) {
	defaultSelection := defaultSelectionInput(sections)
	for idx, s := range sections {
		fmt.Printf("[%d] %s (%s)\n", idx+1, s.Title, s.Type)
	}
	if defaultSelection == "" {
		fmt.Print("Select library number(s) (comma-separated) or 'all': ")
	} else {
		fmt.Printf("Select library number(s) (comma-separated) or 'all' [%s]: ", defaultSelection)
	}
	input, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if strings.TrimSpace(input) == "" && defaultSelection != "" {
		input = defaultSelection
	}

	indices, err := parseSelectionInput(input, len(sections))
	if err != nil {
		return nil, err
	}
	if err := saveSelectionMemory(selectionKeysFromIndices(sections, indices)); err != nil {
		fmt.Printf("%s failed to save selection memory: %v\n", colorizeLevel("WARNING"), err)
	}

	selected := make([]plexSection, 0, len(indices))
	for _, idx := range indices {
		selected = append(selected, sections[idx])
	}
	return selected, nil
}

func selectionKeysFromIndices(sections []plexSection, indices []int) []string {
	keys := make([]string, 0, len(indices))
	for _, idx := range indices {
		if idx < 0 || idx >= len(sections) {
			continue
		}
		keys = append(keys, sections[idx].Key)
	}
	return keys
}

func defaultSelectionInput(sections []plexSection) string {
	memory, err := loadSelectionMemory()
	if err != nil || len(memory.SectionKeys) == 0 {
		return ""
	}
	return defaultSelectionInputFromKeys(sections, memory.SectionKeys)
}

func defaultSelectionInputFromKeys(sections []plexSection, keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	indexByKey := make(map[string]int, len(sections))
	for idx, section := range sections {
		indexByKey[section.Key] = idx + 1
	}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		index, ok := indexByKey[key]
		if !ok {
			return ""
		}
		parts = append(parts, strconv.Itoa(index))
	}
	if len(parts) == len(sections) {
		return "all"
	}
	return strings.Join(parts, ",")
}

func loadSelectionMemory() (selectionMemory, error) {
	var memory selectionMemory
	bytes, err := os.ReadFile(selectionMemoryFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return memory, nil
		}
		return memory, err
	}
	if len(bytes) == 0 {
		return memory, nil
	}
	if err := json.Unmarshal(bytes, &memory); err != nil {
		return selectionMemory{}, err
	}
	return memory, nil
}

func saveSelectionMemory(sectionKeys []string) error {
	memory := selectionMemory{SectionKeys: sectionKeys}
	bytes, err := json.Marshal(memory)
	if err != nil {
		return err
	}
	return os.WriteFile(selectionMemoryFile, bytes, 0o644)
}

func selectSingleSection(sections []plexSection) (plexSection, error) {
	selected, err := selectSections(sections)
	if err != nil {
		return plexSection{}, err
	}
	if len(selected) != 1 {
		return plexSection{}, errors.New("select exactly one library")
	}
	return selected[0], nil
}

func promptCloneName(defaultName string) (string, error) {
	if strings.TrimSpace(defaultName) == "" {
		defaultName = "library-clone"
	}
	fmt.Printf("New library name [%s]: ", defaultName)
	input, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	name := strings.TrimSpace(input)
	if name == "" {
		name = defaultName
	}
	return name, nil
}

func defaultCloneLibraryName(sourceName string) string {
	base := strings.TrimSpace(sourceName)
	if base == "" {
		base = "library"
	}
	return base + "-clone"
}

func cloneSelectedLibrary(client *http.Client, cfg Config, sections []plexSection, logger *AppLogger) error {
	source, err := selectSingleSection(sections)
	if err != nil {
		return err
	}
	logger.Printf("library clone: source library=%s (%s)", source.Title, source.Key)

	sourceDetail, err := fetchSectionDetail(client, cfg, source.Key, logger)
	if err != nil {
		return err
	}
	locations := extractSectionLocations(sourceDetail)
	if len(locations) == 0 {
		return fmt.Errorf("source library has no location mappings")
	}

	newName, err := promptCloneName(defaultCloneLibraryName(source.Title))
	if err != nil {
		return err
	}
	logger.Printf("library clone: requested target name=%q", newName)

	if err := ensureLibraryNameAvailable(sections, newName); err != nil {
		return err
	}

	newSection, err := createLibraryFromSection(client, cfg, sourceDetail, newName, locations, logger)
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

func extractSectionLocations(detail plexSectionDetail) []string {
	locations := make([]string, 0, len(detail.Locations))
	seen := map[string]struct{}{}
	for _, location := range detail.Locations {
		path := strings.TrimSpace(location.Path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		locations = append(locations, path)
	}
	return locations
}

func ensureLibraryNameAvailable(sections []plexSection, name string) error {
	for _, section := range sections {
		if strings.EqualFold(strings.TrimSpace(section.Title), strings.TrimSpace(name)) {
			return fmt.Errorf("library name already exists: %s", name)
		}
	}
	return nil
}

func createLibraryFromSection(client *http.Client, cfg Config, sourceDetail plexSectionDetail, newName string, locations []string, logger *AppLogger) (plexSection, error) {
	requestURL, err := buildCreateLibraryURL(cfg.Plex.BaseURL, cfg.Plex.Token, sourceDetail, newName, locations)
	if err != nil {
		return plexSection{}, err
	}
	if logger != nil {
		logger.APIf("plex call: POST %s", strings.TrimRight(cfg.Plex.BaseURL, "/")+"/library/sections")
		logger.APIf("plex curl: %s", curlCommand("POST", requestURL))
	}
	if shouldSkipPlexWrite(logger, "create library", requestURL) {
		return plexSection{Key: "trail-clone", Title: newName, Type: sourceDetail.Type}, nil
	}

	resp, err := doRequestWithRetry(client, cfg.Plex.Retries, cfg.Plex.RetryBaseMs, cfg.Plex.RetryMaxMs, logger, "POST "+requestURL, func() (*http.Request, error) {
		return http.NewRequest(http.MethodPost, requestURL, nil)
	})
	if err != nil {
		return plexSection{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return plexSection{}, fmt.Errorf("plex create library failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(body)))
	}

	var out plexSectionsResponse
	if err := xml.Unmarshal(body, &out); err == nil && len(out.Directories) > 0 {
		return out.Directories[0], nil
	}

	sections, err := fetchSections(client, cfg, logger)
	if err != nil {
		return plexSection{}, err
	}
	for _, section := range sections {
		if strings.EqualFold(strings.TrimSpace(section.Title), strings.TrimSpace(newName)) {
			return section, nil
		}
	}

	return plexSection{}, fmt.Errorf("created library %q but failed to resolve its section key", newName)
}

func buildCreateLibraryURL(baseURL, token string, sourceDetail plexSectionDetail, newName string, locations []string) (string, error) {
	if strings.TrimSpace(newName) == "" {
		return "", errors.New("library name is required")
	}
	if strings.TrimSpace(sourceDetail.Type) == "" {
		return "", errors.New("source library type is required")
	}
	if len(locations) == 0 {
		return "", errors.New("at least one source location is required")
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/library/sections"
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("X-Plex-Token", token)
	q.Set("name", newName)
	q.Set("type", sourceDetail.Type)
	if strings.TrimSpace(sourceDetail.Agent) != "" {
		q.Set("agent", sourceDetail.Agent)
	}
	if strings.TrimSpace(sourceDetail.Scanner) != "" {
		q.Set("scanner", sourceDetail.Scanner)
	}
	if strings.TrimSpace(sourceDetail.Language) != "" {
		q.Set("language", sourceDetail.Language)
	}
	for _, location := range locations {
		q.Add("location", location)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func applySectionPreferences(client *http.Client, cfg Config, sectionKey string, prefs []plexSectionPref, logger *AppLogger) error {
	applied := 0
	skipped := 0
	failed := 0
	for _, pref := range prefs {
		id := strings.TrimSpace(pref.ID)
		if id == "" {
			skipped++
			continue
		}
		if err := setSectionPreference(client, cfg, sectionKey, id, pref.Value, logger); err != nil {
			failed++
			if logger != nil {
				logger.Printf("library clone: preference copy failed id=%s error=%v", id, err)
			}
			continue
		}
		applied++
	}
	if logger != nil {
		logger.Printf("library clone: preferences copied applied=%d skipped=%d failed=%d", applied, skipped, failed)
	}
	return nil
}

func setSectionPreference(client *http.Client, cfg Config, sectionKey, prefID, prefValue string, logger *AppLogger) error {
	endpoint := strings.TrimRight(cfg.Plex.BaseURL, "/") + "/library/sections/" + sectionKey + "/prefs"
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("X-Plex-Token", cfg.Plex.Token)
	q.Set(prefID, prefValue)
	u.RawQuery = q.Encode()

	if logger != nil {
		logger.APIf("plex call: PUT %s", endpoint)
		logger.APIf("plex curl: %s", curlCommand("PUT", u.String()))
	}
	if shouldSkipPlexWrite(logger, "set section preference", u.String()) {
		return nil
	}

	resp, err := doRequestWithRetry(client, cfg.Plex.Retries, cfg.Plex.RetryBaseMs, cfg.Plex.RetryMaxMs, logger, "PUT "+u.String(), func() (*http.Request, error) {
		return http.NewRequest(http.MethodPut, u.String(), nil)
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("preference update failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func labelMatchingItems(client *http.Client, cfg Config, selectedSection plexSection, finds []string, labelsToAdd []string, categoriesToAdd []string, updateCategory bool, onlyCategory bool, logger *AppLogger) error {
	var err error
	finds, err = normalizeFindList(finds)
	if err != nil {
		return fmt.Errorf("invalid find patterns: %w", err)
	}
	if len(finds) == 0 {
		return errors.New("at least one find pattern is required")
	}
	if !onlyCategory && len(labelsToAdd) == 0 {
		return errors.New("at least one label is required unless -only-category is enabled")
	}
	if (updateCategory || onlyCategory) && len(categoriesToAdd) == 0 {
		categoriesToAdd = labelsToAdd
	}
	if (updateCategory || onlyCategory) && len(categoriesToAdd) == 0 {
		return errors.New("at least one category is required for category updates")
	}

	items, err := fetchSectionItems(client, cfg, selectedSection.Key, logger)
	if err != nil {
		return err
	}
	logger.Infof("label mode: scanned items=%d workers=%d", len(items), cfg.Plex.Workers)

	sem := make(chan struct{}, cfg.Plex.Workers)
	var wg sync.WaitGroup
	var resultsMu sync.Mutex
	var results []labelItemResult
	var activeWorkers atomic.Int32
	dispatched := 0
	totalItems := len(items)

	for _, item := range items {
		item := item
		displayTitle := libraryItemTitle(item)
		matchText := libraryItemMatchText(item)
		matchedFind, matched := firstMatchedFind(matchText, finds)
		if !matched {
			continue
		}
		logger.Matchf("matched title=%q", highlightFindMatches(displayTitle, matchedFind))

		dispatched++
		if dispatched%100 == 0 {
			logger.Infof("label mode: dispatched=%d/%d workers=%d/%d", dispatched, totalItems, activeWorkers.Load(), cfg.Plex.Workers)
		}
		wg.Add(1)
		sem <- struct{}{}
		activeWorkers.Add(1)
		logger.Infof("label mode: worker acquired ratingKey=%s active=%d/%d", item.RatingKey, activeWorkers.Load(), cfg.Plex.Workers)
		go func() {
			defer wg.Done()
			defer func() {
				<-sem
				activeWorkers.Add(-1)
				logger.Infof("label mode: worker released ratingKey=%s active=%d/%d", item.RatingKey, activeWorkers.Load(), cfg.Plex.Workers)
			}()

			r := labelItemResult{displayTitle: displayTitle, ratingKey: item.RatingKey}

			// Snapshot the before state.
			r.labelsBefore = joinPlexLabels(item.Labels)
			r.categoriesBefore = joinPlexLabels(item.Genres)

			if !onlyCategory {
				mergedLabels, labelsChanged := mergeLabels(item.Labels, labelsToAdd)
				if labelsChanged {
					if err := updateLibraryItemLabels(client, cfg, item.RatingKey, mergedLabels, logger); err != nil {
						r.err = fmt.Errorf("update labels for title=%q ratingKey=%s: %w", displayTitle, item.RatingKey, err)
						resultsMu.Lock()
						results = append(results, r)
						resultsMu.Unlock()
						return
					}
					r.labelsUpdated = true
					r.labelsAfter = strings.Join(mergedLabels, ", ")
					logger.Successf("labels added for title=%q ratingKey=%s", displayTitle, item.RatingKey)
				}
			}

			if updateCategory || onlyCategory {
				mergedCategories, categoriesChanged := mergeLabels(item.Genres, labelsToAdd)
				if categoriesChanged {
					if err := updateLibraryItemCategories(client, cfg, item.RatingKey, mergedCategories, logger); err != nil {
						r.err = fmt.Errorf("update categories for title=%q ratingKey=%s: %w", displayTitle, item.RatingKey, err)
						resultsMu.Lock()
						results = append(results, r)
						resultsMu.Unlock()
						return
					}
					r.categoriesUpdated = true
					r.categoriesAfter = strings.Join(mergedCategories, ", ")
					logger.Successf("categories added for title=%q ratingKey=%s", displayTitle, item.RatingKey)
				}
			}

			if !r.labelsUpdated && !r.categoriesUpdated {
				r.skipped = true
				logger.Warningf("label mode: no changes required title=%q ratingKey=%s", displayTitle, item.RatingKey)
			}
			resultsMu.Lock()
			results = append(results, r)
			resultsMu.Unlock()
		}()
	}
	wg.Wait()

	// Aggregate worker results.
	updatedLabels := 0
	updatedCategories := 0
	skipped := 0
	var reportRows []labelReportRow
	for _, r := range results {
		if r.err != nil {
			return r.err
		}
		if r.skipped {
			skipped++
			continue
		}
		if r.labelsUpdated {
			updatedLabels++
		}
		if r.categoriesUpdated {
			updatedCategories++
		}
		// Only include rows where something actually changed.
		if r.labelsUpdated || r.categoriesUpdated {
			reportRows = append(reportRows, labelReportRow{
				RatingKey:        r.ratingKey,
				Title:            r.displayTitle,
				LabelsBefore:     r.labelsBefore,
				LabelsAfter:      r.labelsAfter,
				CategoriesBefore: r.categoriesBefore,
				CategoriesAfter:  r.categoriesAfter,
			})
		}
	}

	reportPath := uniqueLabelReportPath(cfg.OutputDir, time.Now())
	if err := writeLabelReport(reportPath, reportRows); err != nil {
		logger.Warningf("label mode: failed to write report: %v", err)
	} else {
		logger.Infof("label mode: report written: %s (%d rows)", reportPath, len(reportRows))
	}

	logger.Successf("label mode complete: labels_updated=%d categories_updated=%d skipped=%d", updatedLabels, updatedCategories, skipped)
	return nil
}

func cleanLibraryTitles(client *http.Client, cfg Config, sections []plexSection, translateEnabled bool, logger *AppLogger) error {
	selectedSection, err := selectSingleSection(sections)
	if err != nil {
		return err
	}
	logger.Infof("clean mode: selected library=%s (%s)", selectedSection.Title, selectedSection.Key)

	items, err := fetchSectionItems(client, cfg, selectedSection.Key, logger)
	if err != nil {
		return err
	}
	logger.Infof("clean mode: scanned items=%d", len(items))
	logger.Infof("clean mode: workers=%d", cfg.Plex.Workers)

	unknownDate := time.Now().Format("20060102")
	var stampMu sync.Mutex
	unknownSeq := 0
	stampUnknown := func(s string) string {
		if s != "Unknown" {
			return s
		}
		stampMu.Lock()
		defer stampMu.Unlock()
		unknownSeq++
		return fmt.Sprintf("Unknown-%s-%04d", unknownDate, unknownSeq)
	}

	sem := make(chan struct{}, cfg.Plex.Workers)
	var wg sync.WaitGroup
	var resultsMu sync.Mutex
	var results []cleanItemResult
	var activeWorkers atomic.Int32
	dispatched := 0
	totalItems := len(items)

	for _, item := range items {
		item := item
		if strings.TrimSpace(item.RatingKey) == "" {
			resultsMu.Lock()
			results = append(results, cleanItemResult{skipped: true})
			resultsMu.Unlock()
			logger.Warningf("clean mode: skipping item with empty rating key")
			continue
		}
		dispatched++
		if dispatched%100 == 0 {
			logger.Infof("clean mode: dispatched=%d/%d workers=%d/%d", dispatched, totalItems, activeWorkers.Load(), cfg.Plex.Workers)
		}
		wg.Add(1)
		sem <- struct{}{}
		activeWorkers.Add(1)
		logger.Infof("clean mode: worker acquired ratingKey=%s active=%d/%d", item.RatingKey, activeWorkers.Load(), cfg.Plex.Workers)
		go func() {
			defer wg.Done()
			defer func() {
				<-sem
				activeWorkers.Add(-1)
				logger.Infof("clean mode: worker released ratingKey=%s active=%d/%d", item.RatingKey, activeWorkers.Load(), cfg.Plex.Workers)
			}()

			r := cleanItemResult{ratingKey: item.RatingKey}
			beforeTitle := strings.TrimSpace(item.Title)
			beforeSortTitle := strings.TrimSpace(item.SortTitle)
			workingTitle, workingSortTitle := seedCleanTitles(beforeTitle, beforeSortTitle, libraryItemFileStem(item))
			if workingTitle != beforeTitle {
				logger.Infof("clean mode: populated blank title from filename ratingKey=%s", item.RatingKey)
			}
			if workingSortTitle != beforeSortTitle && workingSortTitle != "" {
				if workingSortTitle == workingTitle {
					logger.Infof("clean mode: populated blank sort title from title ratingKey=%s", item.RatingKey)
				} else {
					logger.Infof("clean mode: populated blank sort title from filename ratingKey=%s", item.RatingKey)
				}
			}
			if translateEnabled {
				translated, detectedLang, translatedOk := maybeTranslateToEnglish(client, cfg, workingTitle, logger)
				if translatedOk && strings.TrimSpace(translated) != "" {
					logger.Infof("clean mode: translated title ratingKey=%s lang=%s", item.RatingKey, detectedLang)
					workingTitle = translated
				}
			}
			afterTitle := stampUnknown(cleanTitleForSearch(workingTitle, cfg.Clean.Replacements))
			updateSortTitle := beforeSortTitle == "" && strings.TrimSpace(workingSortTitle) != ""
			afterSortTitle := ""
			if updateSortTitle {
				afterSortTitle = stampUnknown(cleanTitleForSearch(workingSortTitle, cfg.Clean.Replacements))
			}
			if beforeTitle == afterTitle && (!updateSortTitle || beforeSortTitle == afterSortTitle) {
				r.skipped = true
				resultsMu.Lock()
				results = append(results, r)
				resultsMu.Unlock()
				return
			}
			if err := updateLibraryItemTitleWithOptionalSort(client, cfg, item.RatingKey, afterTitle, updateSortTitle, afterSortTitle, logger); err != nil {
				r.err = fmt.Errorf("update clean title ratingKey=%s: %w", item.RatingKey, err)
				resultsMu.Lock()
				results = append(results, r)
				resultsMu.Unlock()
				return
			}
			// If the title resolved to Unknown, add a REVIEW label so the item can be found easily.
			if strings.HasPrefix(afterTitle, "Unknown-") {
				reviewLabels, _ := mergeLabels(item.Labels, []string{"REVIEW"})
				if err := updateLibraryItemLabels(client, cfg, item.RatingKey, reviewLabels, logger); err != nil {
					logger.Warningf("clean mode: failed to add REVIEW label ratingKey=%s: %v", item.RatingKey, err)
				} else {
					logger.Infof("clean mode: REVIEW label added ratingKey=%s", item.RatingKey)
				}
			}
			r.beforeTitle = beforeTitle
			r.afterTitle = afterTitle
			r.beforeSortTitle = beforeSortTitle
			r.afterSortTitle = afterSortTitle
			r.updateSortTitle = updateSortTitle
			resultsMu.Lock()
			results = append(results, r)
			resultsMu.Unlock()
			if updateSortTitle {
				logger.Successf("clean mode: title/sort updated ratingKey=%s title_before=%q title_after=%q sort_before=%q sort_after=%q", item.RatingKey, beforeTitle, afterTitle, beforeSortTitle, afterSortTitle)
			} else {
				logger.Successf("clean mode: title updated ratingKey=%s before=%q after=%q", item.RatingKey, beforeTitle, afterTitle)
			}
		}()
	}
	wg.Wait()

	// Aggregate worker results.
	updated := 0
	skipped := 0
	var reportRows []cleanReportRow
	for _, r := range results {
		if r.err != nil {
			return r.err
		}
		if r.skipped {
			skipped++
			continue
		}
		updated++
		if r.beforeTitle != r.afterTitle {
			reportRows = append(reportRows, cleanReportRow{
				RatingKey:       r.ratingKey,
				TitleBefore:     r.beforeTitle,
				TitleAfter:      r.afterTitle,
				SortTitleBefore: r.beforeSortTitle,
				SortTitleAfter:  r.afterSortTitle,
			})
		}
	}

	reportPath := uniqueCleanReportPath(cfg.OutputDir, time.Now())
	if err := writeCleanReport(reportPath, reportRows); err != nil {
		logger.Warningf("clean mode: failed to write report: %v", err)
	} else {
		logger.Infof("clean mode: report written: %s (%d rows)", reportPath, len(reportRows))
	}

	logger.Successf("clean mode complete: updated=%d skipped=%d", updated, skipped)
	return nil
}

func translateLibraryTitles(client *http.Client, cfg Config, sections []plexSection, logger *AppLogger) error {
	selectedSection, err := selectSingleSection(sections)
	if err != nil {
		return err
	}
	logger.Infof("translate mode: selected library=%s (%s)", selectedSection.Title, selectedSection.Key)

	items, err := fetchSectionItems(client, cfg, selectedSection.Key, logger)
	if err != nil {
		return err
	}
	logger.Infof("translate mode: scanned items=%d", len(items))

	updated := 0
	skipped := 0
	for _, item := range items {
		if strings.TrimSpace(item.RatingKey) == "" {
			skipped++
			logger.Warningf("translate mode: skipping item with empty rating key")
			continue
		}
		before := strings.TrimSpace(item.Title)
		translated, detectedLang, translatedOk := maybeTranslateToEnglish(client, cfg, before, logger)
		if !translatedOk || strings.TrimSpace(translated) == "" || translated == before {
			skipped++
			continue
		}
		if err := updateLibraryItemTitle(client, cfg, item.RatingKey, translated, logger); err != nil {
			return fmt.Errorf("update translated title ratingKey=%s: %w", item.RatingKey, err)
		}
		updated++
		logger.Successf("translate mode: title updated ratingKey=%s lang=%s before=%q after=%q", item.RatingKey, detectedLang, before, translated)
	}

	logger.Successf("translate mode complete: updated=%d skipped=%d", updated, skipped)
	return nil
}

func cleanTitleForSearch(in string, replacements map[string]string) string {
	s := strings.TrimSpace(in)
	// Strip video container file extensions before anything else.
	s = stripVideoExtensions(s)
	// First pass of custom replacements.
	s = applyCustomReplacements(s, replacements)

	runes := []rune(s)
	var b strings.Builder
	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Strip emoji and quote characters without emitting a space.
		if isEmojiRune(r) || isQuoteRune(r) {
			continue
		}

		if r == '&' {
			b.WriteString(" and ")
			continue
		}

		if r == '#' {
			j := i + 1
			for j < len(runes) && unicode.IsSpace(runes[j]) {
				j++
			}
			k := j
			for k < len(runes) && unicode.IsDigit(runes[k]) {
				k++
			}
			if k > j {
				b.WriteString(" NoDOT ")
				b.WriteString(string(runes[j:k]))
				b.WriteByte(' ')
				i = k - 1
				continue
			}
			b.WriteByte(' ')
			continue
		}

		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) || r == '@' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte(' ')
	}

	clean := strings.Join(strings.Fields(b.String()), " ")
	clean = strings.ReplaceAll(clean, "NoDOT", "No.")
	// Second pass of custom replacements so patterns exposed after the char pass are caught.
	clean = applyCustomReplacements(clean, replacements)
	clean = strings.Join(strings.Fields(clean), " ")
	// Empty check is done after all transformations are complete.
	if clean == "" {
		return "Unknown"
	}
	return uppercaseFirstLetter(clean)
}

func stripVideoExtensions(s string) string {
	return videoExtRe.ReplaceAllString(s, "")
}

// isEmojiRune reports whether r is in a Unicode emoji/symbol block.
func isEmojiRune(r rune) bool {
	return (r >= 0x1F300 && r <= 0x1FAFF) || // Misc Symbols & Pictographs through Extended-A
		(r >= 0x2600 && r <= 0x27BF) || // Misc Symbols, Dingbats
		(r >= 0xFE00 && r <= 0xFE0F) || // Variation Selectors (emoji presentation)
		r == 0x200D // Zero Width Joiner (used in emoji sequences)
}

func isQuoteRune(r rune) bool {
	switch r {
	case
		'\'', '"', '`', // ASCII: ' " `
		'\u2018', '\u2019', // ' '
		'\u201A', '\u201B', // ‚ ‛
		'\u201C', '\u201D', // " "
		'\u201E', '\u201F', // „ ‟
		'\u2032', '\u2033', '\u2034', '\u2035', // ′ ″ ‴ ‵
		'\u2039', '\u203A', // ‹ ›
		'\u00AB', '\u00BB': // « »
		return true
	}
	return false
}

func applyCustomReplacements(in string, replacements map[string]string) string {
	if len(replacements) == 0 {
		return in
	}
	type pair struct {
		from string
		to   string
	}
	pairs := make([]pair, 0, len(replacements))
	for from, to := range replacements {
		if from == "" {
			continue
		}
		pairs = append(pairs, pair{from: from, to: to})
	}
	if len(pairs) == 0 {
		return in
	}
	// Longest patterns first so multi-char mappings win before single-char ones.
	slices.SortFunc(pairs, func(a, b pair) int {
		if len(a.from) == len(b.from) {
			if a.from < b.from {
				return -1
			}
			if a.from > b.from {
				return 1
			}
			return 0
		}
		if len(a.from) > len(b.from) {
			return -1
		}
		return 1
	})
	out := in
	for _, p := range pairs {
		out = strings.ReplaceAll(out, p.from, p.to)
	}
	return out
}

type translateRequest struct {
	Q      string `json:"q"`
	Source string `json:"source"`
	Target string `json:"target"`
	Format string `json:"format"`
	APIKey string `json:"api_key,omitempty"`
}

type translateResponse struct {
	TranslatedText string `json:"translatedText"`
}

func maybeTranslateToEnglish(client *http.Client, cfg Config, title string, logger *AppLogger) (string, string, bool) {
	text := strings.TrimSpace(title)
	if text == "" {
		return title, "", false
	}
	if !hasClearNonEnglishMarkers(text) {
		return title, "", false
	}
	info := whatlanggo.Detect(text)
	if info.Lang == whatlanggo.Eng {
		return title, info.Lang.Iso6391(), false
	}

	translated, err := translateTextToEnglish(client, cfg, text, logger)
	if err != nil {
		logger.Warningf("clean mode: translation failed lang=%s error=%v", info.Lang.Iso6391(), err)
		return title, info.Lang.Iso6391(), false
	}
	return translated, info.Lang.Iso6391(), true
}

func hasClearNonEnglishMarkers(text string) bool {
	for _, r := range text {
		if r > unicode.MaxASCII && unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func translateTextToEnglish(client *http.Client, cfg Config, text string, logger *AppLogger) (string, error) {
	requestBody := translateRequest{
		Q:      text,
		Source: "auto",
		Target: "en",
		Format: "text",
		APIKey: strings.TrimSpace(cfg.Clean.TranslateAPIKey),
	}
	bytesBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}
	endpoint := strings.TrimSpace(cfg.Clean.TranslateEndpoint)
	if endpoint == "" {
		return "", errors.New("translate endpoint is empty")
	}
	waitForTranslationRateLimit(cfg.Clean.TranslateRateLimitPerMinute, logger)
	if logger != nil {
		logger.APIf("translate call: POST %s", endpoint)
	}

	resp, err := doRequestWithRetry(client, cfg.Plex.Retries, cfg.Plex.RetryBaseMs, cfg.Plex.RetryMaxMs, logger, "POST "+endpoint, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bytesBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("translation request failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var out translateResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.TranslatedText) == "" {
		return "", errors.New("translation returned empty text")
	}
	return out.TranslatedText, nil
}

func waitForTranslationRateLimit(ratePerMinute int, logger *AppLogger) {
	if ratePerMinute <= 0 {
		return
	}
	interval := time.Minute / time.Duration(ratePerMinute)
	if interval <= 0 {
		interval = time.Second
	}

	translateRateLimitMu.Lock()
	now := time.Now()
	wait := time.Duration(0)
	if now.Before(nextTranslateRequestAt) {
		wait = nextTranslateRequestAt.Sub(now)
	}
	translateRateLimitMu.Unlock()

	if wait > 0 {
		if logger != nil {
			logger.Warningf("translate rate limit: waiting %s", wait.Round(time.Millisecond))
		}
		time.Sleep(wait)
	}

	translateRateLimitMu.Lock()
	nextTranslateRequestAt = time.Now().Add(interval)
	translateRateLimitMu.Unlock()
}

func uppercaseFirstLetter(s string) string {
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if unicode.IsLetter(runes[i]) {
			runes[i] = unicode.ToUpper(runes[i])
			break
		}
	}
	return string(runes)
}

func highlightFindMatches(text, find string) string {
	needle := strings.TrimSpace(find)
	if needle == "" || text == "" || !colorOutputEnabled {
		return text
	}
	lowerText := strings.ToLower(text)
	lowerNeedle := strings.ToLower(needle)
	if !strings.Contains(lowerText, lowerNeedle) {
		return text
	}
	var b strings.Builder
	start := 0
	for {
		idx := strings.Index(lowerText[start:], lowerNeedle)
		if idx < 0 {
			b.WriteString(text[start:])
			break
		}
		abs := start + idx
		b.WriteString(text[start:abs])
		b.WriteString(fcolor.New(fcolor.FgMagenta, fcolor.Bold).Sprint(text[abs : abs+len(needle)]))
		start = abs + len(needle)
	}
	return b.String()
}

func parseLabelList(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	parts := strings.Split(trimmed, ",")
	labels, err := normalizeTagList(parts)
	if err != nil {
		return nil, err
	}
	if len(labels) == 0 {
		return nil, errors.New("no valid labels found")
	}
	return labels, nil
}

func normalizeTagList(in []string) ([]string, error) {
	labels := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, part := range in {
		label := strings.TrimSpace(part)
		if label == "" {
			continue
		}
		key := strings.ToLower(label)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		labels = append(labels, label)
	}
	return labels, nil
}

func normalizeFindList(in []string) ([]string, error) {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, part := range in {
		find := strings.TrimSpace(part)
		if find == "" {
			continue
		}
		key := strings.ToLower(find)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, find)
	}
	return out, nil
}

func titleMatchesFind(title, find string) bool {
	needle := strings.TrimSpace(find)
	if needle == "" {
		return false
	}
	return strings.Contains(strings.ToLower(title), strings.ToLower(needle))
}

func firstMatchedFind(title string, finds []string) (string, bool) {
	for _, find := range finds {
		if titleMatchesFind(title, find) {
			return find, true
		}
	}
	return "", false
}

func libraryItemTitle(item plexLibraryItem) string {
	title := strings.TrimSpace(item.Title)
	if title != "" {
		return title
	}
	title = strings.TrimSpace(item.OriginalTitle)
	if title != "" {
		return title
	}
	for _, media := range item.Media {
		for _, part := range media.Parts {
			filePath := strings.TrimSpace(part.File)
			if filePath == "" {
				continue
			}
			base := filepath.Base(filePath)
			base = strings.TrimSuffix(base, filepath.Ext(base))
			if strings.TrimSpace(base) != "" {
				return base
			}
			return filePath
		}
	}
	return ""
}

func libraryItemMatchText(item plexLibraryItem) string {
	title := libraryItemTitle(item)
	if title != "" {
		return title
	}
	for _, media := range item.Media {
		for _, part := range media.Parts {
			filePath := strings.TrimSpace(part.File)
			if filePath != "" {
				return filePath
			}
		}
	}
	return ""
}

// seedCleanTitles returns working title and sort title, seeding blank values according to clean-mode rules:
// - blank title → filled from fileStem
// - blank sort title → filled from title (post-seeding), then fileStem if still blank
func seedCleanTitles(title, sortTitle, fileStem string) (string, string) {
	workingTitle := strings.TrimSpace(title)
	workingSortTitle := strings.TrimSpace(sortTitle)
	if workingTitle == "" && fileStem != "" {
		workingTitle = fileStem
	}
	if workingSortTitle == "" {
		if workingTitle != "" {
			workingSortTitle = workingTitle
		} else if fileStem != "" {
			workingSortTitle = fileStem
		}
	}
	return workingTitle, workingSortTitle
}

func libraryItemFileStem(item plexLibraryItem) string {
	for _, media := range item.Media {
		for _, part := range media.Parts {
			filePath := strings.TrimSpace(part.File)
			if filePath == "" {
				continue
			}
			base := filepath.Base(filePath)
			stem := strings.TrimSuffix(base, filepath.Ext(base))
			if strings.TrimSpace(stem) != "" {
				return stem
			}
		}
	}
	return ""
}

func joinPlexLabels(labels []plexLabel) string {
	parts := make([]string, 0, len(labels))
	for _, l := range labels {
		if t := strings.TrimSpace(l.Tag); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, ", ")
}

func mergeLabels(existing []plexLabel, toAdd []string) ([]string, bool) {
	merged := make([]string, 0, len(existing)+len(toAdd))
	seen := map[string]struct{}{}
	for _, label := range existing {
		tag := strings.TrimSpace(label.Tag)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, tag)
	}
	changed := false
	for _, label := range toAdd {
		tag := strings.TrimSpace(label)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, tag)
		changed = true
	}
	return merged, changed
}

func buildUpdateLibraryItemLabelsURL(baseURL, token, ratingKey string, labels []string) (string, error) {
	if strings.TrimSpace(ratingKey) == "" {
		return "", errors.New("missing rating key")
	}
	if len(labels) == 0 {
		return "", errors.New("at least one label is required")
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/library/metadata/" + ratingKey
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("X-Plex-Token", token)
	for idx, label := range labels {
		q.Set(fmt.Sprintf("label[%d].tag.tag", idx), label)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func updateLibraryItemLabels(client *http.Client, cfg Config, ratingKey string, labels []string, logger *AppLogger) error {
	requestURL, err := buildUpdateLibraryItemLabelsURL(cfg.Plex.BaseURL, cfg.Plex.Token, ratingKey, labels)
	if err != nil {
		return err
	}
	if logger != nil {
		logger.APIf("plex call: PUT %s", strings.TrimRight(cfg.Plex.BaseURL, "/")+"/library/metadata/"+ratingKey)
		logger.APIf("plex curl: %s", curlCommand("PUT", requestURL))
	}
	if shouldSkipPlexWrite(logger, "update item labels", requestURL) {
		return nil
	}
	resp, err := doRequestWithRetry(client, cfg.Plex.Retries, cfg.Plex.RetryBaseMs, cfg.Plex.RetryMaxMs, logger, "PUT "+requestURL, func() (*http.Request, error) {
		return http.NewRequest(http.MethodPut, requestURL, nil)
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("plex update labels failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func buildUpdateLibraryItemCategoriesURL(baseURL, token, ratingKey string, categories []string) (string, error) {
	if strings.TrimSpace(ratingKey) == "" {
		return "", errors.New("missing rating key")
	}
	if len(categories) == 0 {
		return "", errors.New("at least one category is required")
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/library/metadata/" + ratingKey
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("X-Plex-Token", token)
	for idx, category := range categories {
		q.Set(fmt.Sprintf("genre[%d].tag.tag", idx), category)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func updateLibraryItemCategories(client *http.Client, cfg Config, ratingKey string, categories []string, logger *AppLogger) error {
	requestURL, err := buildUpdateLibraryItemCategoriesURL(cfg.Plex.BaseURL, cfg.Plex.Token, ratingKey, categories)
	if err != nil {
		return err
	}
	if logger != nil {
		logger.APIf("plex call: PUT %s", strings.TrimRight(cfg.Plex.BaseURL, "/")+"/library/metadata/"+ratingKey)
		logger.APIf("plex curl: %s", curlCommand("PUT", requestURL))
	}
	if shouldSkipPlexWrite(logger, "update item categories", requestURL) {
		return nil
	}
	resp, err := doRequestWithRetry(client, cfg.Plex.Retries, cfg.Plex.RetryBaseMs, cfg.Plex.RetryMaxMs, logger, "PUT "+requestURL, func() (*http.Request, error) {
		return http.NewRequest(http.MethodPut, requestURL, nil)
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("plex update categories failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func buildUpdateLibraryItemTitleURL(baseURL, token, ratingKey, title string) (string, error) {
	return buildUpdateLibraryItemTitleAndSortURL(baseURL, token, ratingKey, title, false, "")
}

func buildUpdateLibraryItemTitleAndSortURL(baseURL, token, ratingKey, title string, updateSort bool, sortTitle string) (string, error) {
	if strings.TrimSpace(ratingKey) == "" {
		return "", errors.New("missing rating key")
	}
	if strings.TrimSpace(title) == "" {
		return "", errors.New("missing title")
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/library/metadata/" + ratingKey
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("X-Plex-Token", token)
	q.Set("title.value", title)
	if updateSort {
		if strings.TrimSpace(sortTitle) == "" {
			return "", errors.New("missing sort title")
		}
		q.Set("titleSort.value", sortTitle)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func updateLibraryItemTitle(client *http.Client, cfg Config, ratingKey, title string, logger *AppLogger) error {
	return updateLibraryItemTitleWithOptionalSort(client, cfg, ratingKey, title, false, "", logger)
}

func updateLibraryItemTitleWithOptionalSort(client *http.Client, cfg Config, ratingKey, title string, updateSort bool, sortTitle string, logger *AppLogger) error {
	requestURL, err := buildUpdateLibraryItemTitleAndSortURL(cfg.Plex.BaseURL, cfg.Plex.Token, ratingKey, title, updateSort, sortTitle)
	if err != nil {
		return err
	}
	if logger != nil {
		logger.APIf("plex call: PUT %s", strings.TrimRight(cfg.Plex.BaseURL, "/")+"/library/metadata/"+ratingKey)
		logger.APIf("plex curl: %s", curlCommand("PUT", requestURL))
	}
	if shouldSkipPlexWrite(logger, "update item title", requestURL) {
		return nil
	}
	resp, err := doRequestWithRetry(client, cfg.Plex.Retries, cfg.Plex.RetryBaseMs, cfg.Plex.RetryMaxMs, logger, "PUT "+requestURL, func() (*http.Request, error) {
		return http.NewRequest(http.MethodPut, requestURL, nil)
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("plex update title failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func exportCollections(client *http.Client, cfg Config, sections []plexSection, exportPath string, logger *AppLogger) error {
	selectedSection, err := selectSingleSection(sections)
	if err != nil {
		return err
	}
	logger.Printf("collection export: selected library=%s (%s)", selectedSection.Title, selectedSection.Key)

	collections, err := fetchCollections(client, cfg, selectedSection.Key, logger)
	if err != nil {
		return err
	}

	transfer := collectionTransferFile{
		Version:       1,
		ExportedAtUTC: time.Now().UTC().Format(time.RFC3339),
		SourceLibrary: selectedSection,
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
	}

	jsonBytes, err := json.MarshalIndent(transfer, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(exportPath, jsonBytes, 0o644); err != nil {
		return err
	}
	logger.Successf("collection export complete: file=%s count=%d", exportPath, len(transfer.Collections))
	return nil
}

func importCollections(client *http.Client, cfg Config, sections []plexSection, importPath string, logger *AppLogger) error {
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

	targetSection, err := selectSingleSection(sections)
	if err != nil {
		return err
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
			continue
		}

		if err := createCollection(client, cfg, transfer.SourceLibrary.Key, targetSection.Key, title, targetTypeCode, collection, logger); err != nil {
			return fmt.Errorf("create collection %q: %w", title, err)
		}
		existingByTitle[strings.ToLower(title)] = struct{}{}
		created++
	}

	logger.Successf("collection import complete: created=%d skipped=%d", created, skipped)
	return nil
}

func sectionTypeToPlexTypeCode(sectionType string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(sectionType)) {
	case "movie":
		return 1, nil
	case "show":
		return 2, nil
	case "artist":
		return 8, nil
	case "photo":
		return 13, nil
	default:
		return 0, fmt.Errorf("unsupported Plex library type for collections: %s", sectionType)
	}
}

func createCollection(client *http.Client, cfg Config, sourceSectionKey, targetSectionKey, title string, targetTypeCode int, collection collectionTransferRecord, logger *AppLogger) error {
	endpoint := strings.TrimRight(cfg.Plex.BaseURL, "/") + "/library/collections"
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("X-Plex-Token", cfg.Plex.Token)
	q.Set("type", strconv.Itoa(targetTypeCode))
	q.Set("title", title)
	q.Set("sectionId", targetSectionKey)
	if collection.Smart {
		if strings.TrimSpace(collection.Content) == "" {
			return errors.New("smart collection has no filter content")
		}
		q.Set("smart", "1")
		q.Set("uri", resolveCollectionContentURI(collection.Content, sourceSectionKey, targetSectionKey))
	} else {
		q.Set("smart", "0")
	}
	u.RawQuery = q.Encode()

	if logger != nil {
		logger.APIf("plex call: POST %s", endpoint)
		logger.APIf("plex curl: %s", curlCommand("POST", u.String()))
	}
	if shouldSkipPlexWrite(logger, "create collection", u.String()) {
		if logger != nil {
			logger.Successf("collection trail-only: title=%q smart=%t", title, collection.Smart)
		}
		return nil
	}

	resp, err := doRequestWithRetry(client, cfg.Plex.Retries, cfg.Plex.RetryBaseMs, cfg.Plex.RetryMaxMs, logger, "POST "+u.String(), func() (*http.Request, error) {
		return http.NewRequest(http.MethodPost, u.String(), nil)
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("plex create collection failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(body)))
	}
	if logger != nil {
		logger.Successf("collection created: title=%q smart=%t", title, collection.Smart)
	}
	return nil
}

func injectCollections(client *http.Client, cfg Config, sections []plexSection, logger *AppLogger) error {
	if len(cfg.Collection.Lookups) == 0 {
		return errors.New("invalid flags: -col-inject requires one or more [[collection.lookup]] entries in collections.toml")
	}

	targetSection, err := selectSingleSection(sections)
	if err != nil {
		return err
	}
	logger.Printf("collection inject: target library=%s (%s)", targetSection.Title, targetSection.Key)

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
	}

	logger.Successf("collection inject complete: created=%d skipped=%d", created, skipped)
	return nil
}

func normalizeCollectionLookupContent(content string) string {
	trimmed := strings.TrimSpace(content)
	trimmed = strings.TrimPrefix(trimmed, "&")
	trimmed = strings.TrimPrefix(trimmed, "?")
	return trimmed
}

func composeCollectionContent(baseURI, content string) string {
	trimmedContent := normalizeCollectionLookupContent(content)
	if trimmedContent == "" {
		return strings.TrimSpace(baseURI)
	}
	if strings.Contains(trimmedContent, "://") {
		return trimmedContent
	}
	trimmedBase := strings.TrimSpace(baseURI)
	if trimmedBase == "" {
		return trimmedContent
	}
	trimmedBase = strings.TrimRight(trimmedBase, "&?")
	if strings.Contains(trimmedBase, "?") {
		return trimmedBase + "&" + trimmedContent
	}
	return trimmedBase + "?" + trimmedContent
}

func resolveCollectionContentURI(content, sourceSectionKey, targetSectionKey string) string {
	out := strings.TrimSpace(content)
	if out == "" {
		return out
	}
	out = strings.ReplaceAll(out, "{{section_key}}", targetSectionKey)
	out = strings.ReplaceAll(out, "{{sectionKey}}", targetSectionKey)
	if sourceSectionKey == "" {
		sourceSectionKey = extractCollectionSourceSectionKey(out)
	}
	if sourceSectionKey == "" || targetSectionKey == "" {
		return out
	}
	return rewriteCollectionContentURI(out, sourceSectionKey, targetSectionKey)
}

func extractCollectionSourceSectionKey(content string) string {
	matches := collectionSectionKeyRe.FindStringSubmatch(content)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func rewriteCollectionContentURI(content, sourceSectionKey, targetSectionKey string) string {
	out := content
	if strings.TrimSpace(sourceSectionKey) == "" || strings.TrimSpace(targetSectionKey) == "" {
		return out
	}
	out = strings.ReplaceAll(out, "/library/sections/"+sourceSectionKey+"/", "/library/sections/"+targetSectionKey+"/")
	out = strings.ReplaceAll(out, "%2Flibrary%2Fsections%2F"+sourceSectionKey+"%2F", "%2Flibrary%2Fsections%2F"+targetSectionKey+"%2F")
	out = strings.ReplaceAll(out, "%2flibrary%2fsections%2f"+sourceSectionKey+"%2f", "%2flibrary%2fsections%2f"+targetSectionKey+"%2f")
	return out
}

func parseSelectionInput(input string, max int) ([]int, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, fmt.Errorf("invalid selection")
	}
	if strings.EqualFold(trimmed, "all") {
		indices := make([]int, max)
		for i := 0; i < max; i++ {
			indices[i] = i
		}
		return indices, nil
	}

	parts := strings.Split(trimmed, ",")
	seen := make(map[int]struct{}, len(parts))
	indices := make([]int, 0, len(parts))
	for _, part := range parts {
		choice, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || choice < 1 || choice > max {
			return nil, fmt.Errorf("invalid selection")
		}
		idx := choice - 1
		if _, ok := seen[idx]; ok {
			continue
		}
		seen[idx] = struct{}{}
		indices = append(indices, idx)
	}
	if len(indices) == 0 {
		return nil, fmt.Errorf("invalid selection")
	}
	return indices, nil
}

func processCollections(client *http.Client, cfg Config, libraryName string, collections []plexCollection, upload bool, logger *AppLogger) error {
	outDir := filepath.Join(cfg.OutputDir, sanitizeFileName(libraryName))
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	logger.Printf("output dir ready: %s", outDir)
	collections = disambiguateCollectionsByGUID(collections)
	for _, collection := range collections {
		outputPath := buildOutputPath(outDir, collection.Title, cfg.TemplateImage)
		logger.Printf("creating poster: collection=%q guid=%q output=%s", collection.Title, collection.GUID, outputPath)
		if err := renderCollectionPoster(cfg, collection.Title, outputPath); err != nil {
			return fmt.Errorf("render %q: %w", collection.Title, err)
		}
		logger.Successf("poster created: %s", outputPath)
		if upload {
			if err := uploadCollectionPoster(client, cfg, collection, outputPath, logger); err != nil {
				return fmt.Errorf("upload poster for %q: %w", collection.Title, err)
			}
		}
	}
	return nil
}

func uploadCollectionPoster(client *http.Client, cfg Config, collection plexCollection, imagePath string, logger *AppLogger) error {
	if strings.TrimSpace(collection.RatingKey) == "" {
		return errors.New("missing collection ratingKey")
	}

	endpoint := strings.TrimRight(cfg.Plex.BaseURL, "/") + "/library/collections/" + collection.RatingKey + "/posters"
	if logger != nil {
		logger.APIf("plex call: POST %s", endpoint)
		logger.APIf("plex curl: %s", curlFileUploadCommand("POST", endpoint, cfg.Plex.Token, imagePath))
	}
	if shouldSkipPlexWrite(logger, "upload poster", endpoint) {
		if logger != nil {
			logger.Successf("plex upload trail-only: collection=%q rating_key=%s", collection.Title, collection.RatingKey)
		}
		return nil
	}

	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return err
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(imagePath))
	if err != nil {
		return err
	}
	if _, err := part.Write(imageBytes); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("X-Plex-Token", cfg.Plex.Token)
	u.RawQuery = q.Encode()

	resp, err := doRequestWithRetry(client, cfg.Plex.Retries, cfg.Plex.RetryBaseMs, cfg.Plex.RetryMaxMs, logger, "POST "+u.String(), func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(body.Bytes()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())
		return req, nil
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("plex upload failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	if logger != nil {
		logger.Successf("plex upload complete: collection=%q rating_key=%s", collection.Title, collection.RatingKey)
	}
	return nil
}

func shouldSkipPlexWrite(logger *AppLogger, operation, requestURL string) bool {
	if !trailModeEnabled {
		return false
	}
	if logger != nil {
		logger.Warningf("trail mode: skipping Plex write operation=%q url=%s", operation, requestURL)
	}
	return true
}

func buildOutputPath(outDir, collectionName, templatePath string) string {
	ext := strings.ToLower(filepath.Ext(templatePath))
	if ext == "" {
		ext = ".png"
	}
	if ext == ".jpeg" {
		ext = ".jpg"
	}
	return filepath.Join(outDir, sanitizeFileName(collectionName)+ext)
}

func sanitizeFileName(in string) string {
	trimmed := strings.TrimSpace(in)
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "-", "*", "_", "?", "", "\"", "'", "<", "", ">", "", "|", "_")
	out := replacer.Replace(trimmed)
	if out == "" {
		return "untitled"
	}
	return out
}

func normalizeCollectionName(in string) string {
	trimmed := strings.TrimSpace(in)
	if trimmed == "" {
		return "untitled"
	}

	var b strings.Builder
	b.Grow(len(trimmed))
	for _, r := range trimmed {
		switch {
		case unicode.IsLetter(r), unicode.IsNumber(r):
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteByte(' ')
		default:
			b.WriteByte(' ')
		}
	}

	out := strings.Join(strings.Fields(b.String()), " ")
	if out == "" {
		return "untitled"
	}
	return out
}

func disambiguateCollectionsByGUID(collections []plexCollection) []plexCollection {
	if len(collections) == 0 {
		return collections
	}

	titleToGUIDs := make(map[string]map[string]struct{}, len(collections))
	for _, c := range collections {
		if _, ok := titleToGUIDs[c.Title]; !ok {
			titleToGUIDs[c.Title] = map[string]struct{}{}
		}
		titleToGUIDs[c.Title][c.GUID] = struct{}{}
	}

	seqByTitleAndGUID := make(map[string]map[string]int)
	nextSeqByTitle := make(map[string]int)
	out := make([]plexCollection, len(collections))
	for idx, c := range collections {
		out[idx] = c
		if len(titleToGUIDs[c.Title]) <= 1 {
			continue
		}
		if _, ok := seqByTitleAndGUID[c.Title]; !ok {
			seqByTitleAndGUID[c.Title] = map[string]int{}
		}
		if seqByTitleAndGUID[c.Title][c.GUID] == 0 {
			nextSeqByTitle[c.Title]++
			seqByTitleAndGUID[c.Title][c.GUID] = nextSeqByTitle[c.Title]
		}
		out[idx].Title = fmt.Sprintf("%s %d", c.Title, seqByTitleAndGUID[c.Title][c.GUID])
	}

	return out
}

func renderCollectionPoster(cfg Config, text, outputPath string) error {
	templateFile, err := os.Open(cfg.TemplateImage)
	if err != nil {
		return err
	}
	defer templateFile.Close()

	src, _, err := image.Decode(templateFile)
	if err != nil {
		return err
	}

	rgba := image.NewRGBA(src.Bounds())
	draw.Draw(rgba, rgba.Bounds(), src, src.Bounds().Min, draw.Src)

	face, closeFace, err := loadFontFace(cfg)
	if err != nil {
		return err
	}
	defer closeFace()

	mainColor, err := parseHexColor(cfg.Font.Color, 255)
	if err != nil {
		return err
	}
	shadowColor, err := parseHexColor(cfg.Font.ShadowColor, 210)
	if err != nil {
		return err
	}
	glowAlpha := uint8(math.Round(cfg.Font.GlowAlpha * 255))
	glowColor, err := parseHexColor(cfg.Font.GlowColor, glowAlpha)
	if err != nil {
		return err
	}

	w := rgba.Bounds().Dx()
	h := rgba.Bounds().Dy()
	maxTextWidth := w - 40
	if maxTextWidth < 20 {
		maxTextWidth = w
	}
	textLines := wrapTextToWidth(face, forceLineBreakAfterNumber(text), maxTextWidth)
	linePositions := centeredTextDots(face, textLines, w, h, cfg.Font.YOffset)

	if cfg.Font.GlowRadius > 0 {
		for _, line := range linePositions {
			for dx := -cfg.Font.GlowRadius; dx <= cfg.Font.GlowRadius; dx++ {
				for dy := -cfg.Font.GlowRadius; dy <= cfg.Font.GlowRadius; dy++ {
					if dx == 0 && dy == 0 {
						continue
					}
					if dx*dx+dy*dy > cfg.Font.GlowRadius*cfg.Font.GlowRadius {
						continue
					}
					drawText(rgba, face, line.Text, line.X+dx, line.Y+dy, glowColor)
				}
			}
		}
	}

	for _, line := range linePositions {
		if cfg.Font.ShadowOffsetX != 0 || cfg.Font.ShadowOffsetY != 0 {
			drawText(rgba, face, line.Text, line.X+cfg.Font.ShadowOffsetX, line.Y+cfg.Font.ShadowOffsetY, shadowColor)
		}
		drawText(rgba, face, line.Text, line.X, line.Y, mainColor)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(outputPath))
	if ext == ".jpg" || ext == ".jpeg" {
		return jpeg.Encode(f, rgba, &jpeg.Options{Quality: 95})
	}
	return png.Encode(f, rgba)
}

func loadFontFace(cfg Config) (font.Face, func(), error) {
	if cfg.Font.File == "" {
		return basicfont.Face7x13, func() {}, nil
	}
	fontBytes, err := os.ReadFile(cfg.Font.File)
	if err != nil {
		return nil, nil, err
	}
	parsed, err := opentype.Parse(fontBytes)
	if err != nil {
		return nil, nil, err
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: cfg.Font.Size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, nil, err
	}
	return face, func() { _ = face.Close() }, nil
}

type textDot struct {
	Text string
	X    int
	Y    int
}

func centeredTextDots(face font.Face, lines []string, width, height, yOffset int) []textDot {
	if len(lines) == 0 {
		lines = []string{"untitled"}
	}

	lineHeight := face.Metrics().Height.Ceil()
	if lineHeight <= 0 {
		lineHeight = 1
	}
	totalHeight := lineHeight * len(lines)
	top := (height-totalHeight)/2 + yOffset

	dots := make([]textDot, 0, len(lines))
	for idx, line := range lines {
		bounds, _ := font.BoundString(face, line)
		textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
		x := (width-textWidth)/2 - bounds.Min.X.Ceil()
		yTop := top + idx*lineHeight
		y := yTop - bounds.Min.Y.Ceil()
		dots = append(dots, textDot{Text: line, X: x, Y: y})
	}
	return dots
}

func wrapTextToWidth(face font.Face, text string, maxWidth int) []string {
	raw := strings.TrimSpace(text)
	if raw == "" {
		return []string{"untitled"}
	}
	if maxWidth <= 0 {
		return []string{raw}
	}

	segments := strings.Split(raw, "\n")
	lines := make([]string, 0, len(segments))
	for _, segment := range segments {
		clean := strings.Join(strings.Fields(segment), " ")
		if clean == "" {
			continue
		}
		lines = append(lines, wrapSingleLineToWidth(face, clean, maxWidth)...)
	}
	if len(lines) == 0 {
		return []string{"untitled"}
	}
	return lines
}

func wrapSingleLineToWidth(face font.Face, clean string, maxWidth int) []string {
	if clean == "" {
		return nil
	}

	words := strings.Fields(clean)
	lines := make([]string, 0, len(words))
	current := words[0]

	for _, word := range words[1:] {
		candidate := current + " " + word
		if textPixelWidth(face, candidate) <= maxWidth {
			current = candidate
			continue
		}

		lines = append(lines, current)
		if textPixelWidth(face, word) <= maxWidth {
			current = word
			continue
		}

		parts := splitWordToWidth(face, word, maxWidth)
		if len(parts) == 0 {
			current = word
			continue
		}
		lines = append(lines, parts[:len(parts)-1]...)
		current = parts[len(parts)-1]
	}

	if textPixelWidth(face, current) <= maxWidth {
		lines = append(lines, current)
		return lines
	}
	parts := splitWordToWidth(face, current, maxWidth)
	if len(parts) == 0 {
		return append(lines, current)
	}
	return append(lines, parts...)
}

func forceLineBreakAfterNumber(text string) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) == 0 {
		return ""
	}

	var b strings.Builder
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		b.WriteRune(r)
		if !unicode.IsDigit(r) {
			continue
		}

		nextIdx := i + 1
		for nextIdx < len(runes) && unicode.IsDigit(runes[nextIdx]) {
			i = nextIdx
			b.WriteRune(runes[nextIdx])
			nextIdx++
		}

		startIdx := i
		for startIdx > 0 && unicode.IsDigit(runes[startIdx-1]) {
			startIdx--
		}

		beforeIsBlank := startIdx == 0 || unicode.IsSpace(runes[startIdx-1])
		afterIsBlank := nextIdx == len(runes) || unicode.IsSpace(runes[nextIdx])
		if !beforeIsBlank || !afterIsBlank {
			continue
		}

		skipIdx := nextIdx
		for skipIdx < len(runes) && unicode.IsSpace(runes[skipIdx]) && runes[skipIdx] != '\n' {
			skipIdx++
		}
		if skipIdx < len(runes) && runes[skipIdx] != '\n' {
			b.WriteRune('\n')
			i = skipIdx - 1
		}
	}

	return strings.TrimSpace(b.String())
}

func splitWordToWidth(face font.Face, word string, maxWidth int) []string {
	if word == "" {
		return nil
	}
	runes := []rune(word)
	parts := make([]string, 0, len(runes))
	start := 0
	for start < len(runes) {
		end := start + 1
		for end <= len(runes) && textPixelWidth(face, string(runes[start:end])) <= maxWidth {
			end++
		}
		if end == start+1 {
			parts = append(parts, string(runes[start]))
			start++
			continue
		}
		parts = append(parts, string(runes[start:end-1]))
		start = end - 1
	}
	return parts
}

func textPixelWidth(face font.Face, text string) int {
	bounds, _ := font.BoundString(face, text)
	return (bounds.Max.X - bounds.Min.X).Ceil()
}

func drawText(dst draw.Image, face font.Face, text string, x, y int, c color.Color) {
	d := &font.Drawer{Dst: dst, Src: image.NewUniform(c), Face: face, Dot: fixed.P(x, y)}
	d.DrawString(text)
}

func parseHexColor(hex string, alpha uint8) (color.Color, error) {
	s := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(s) != 6 {
		return nil, fmt.Errorf("invalid color: %s", hex)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return nil, err
	}
	return color.RGBA{R: uint8(v >> 16), G: uint8((v >> 8) & 0xFF), B: uint8(v & 0xFF), A: alpha}, nil
}
