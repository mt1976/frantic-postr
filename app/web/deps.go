package web

import (
	"context"
	"net/http"
	"time"
)

var appDisplayName = "frantic-postr"

var (
	loadConfig                    func(path string) (Config, error)
	fetchSections                 func(client *http.Client, cfg Config, logger *AppLogger) ([]plexSection, error)
	fetchCollections              func(client *http.Client, cfg Config, sectionKey string, logger *AppLogger) ([]plexCollection, error)
	fetchSectionDetail            func(client *http.Client, cfg Config, sectionKey string, logger *AppLogger) (plexSectionDetail, error)
	fetchSectionPreferences       func(client *http.Client, cfg Config, sectionKey string, logger *AppLogger) ([]plexSectionPref, error)
	extractSectionLocations       func(detail plexSectionDetail) []string
	ensureLibraryNameAvailable    func(sections []plexSection, name string) error
	resolveCollectionTransferPath func(cfg Config, collFilePath string) string
	resolveCollectionExportPath   func(cfg Config, collFilePath string, now time.Time) string
	resolvePathRelativeToConfig   func(configPath, candidate string) string
	backupArchiveDir              func(workspaceRoot string) string
	formatBackupDateTime          func(value time.Time) string
	mergePlexConfig               func(target, supplemental *Config)
	loadSupplementalConfig        func(basePath, supplementalPath, fieldName string, target any) error
	loadOpsConfig                 func(path string) (Config, error)
	createBackupArchive           func(cfg Config, configPath string, logger *AppLogger) error
	restoreFromBackup             func(cfg Config, restoreFilter string, logger *AppLogger) error
	rollbackLastRestore           func(cfg Config, logger *AppLogger) error
	listBackupArchives            func(backupsDir string) ([]backupCandidate, error)

	parseLabelList                  func(raw string) ([]string, error)
	selectSingleSection             func(sections []plexSection) (plexSection, error)
	labelMatchingItems              func(client *http.Client, cfg Config, selectedSection plexSection, finds []string, labelsToAdd []string, categoriesToAdd []string, updateCategory bool, onlyCategory bool, logger *AppLogger) error
	cleanLibraryTitles              func(client *http.Client, cfg Config, sections []plexSection, translateEnabled bool, logger *AppLogger) error
	translateLibraryTitles          func(client *http.Client, cfg Config, sections []plexSection, logger *AppLogger) error
	analyzeLibraryFileNameStats     func(client *http.Client, cfg Config, sections []plexSection, logger *AppLogger) error
	exportCollections               func(client *http.Client, cfg Config, sections []plexSection, exportPath string, logger *AppLogger) error
	importCollections               func(client *http.Client, cfg Config, sections []plexSection, importPath string, logger *AppLogger) error
	injectCollections               func(client *http.Client, cfg Config, sections []plexSection, logger *AppLogger) error
	reportDuplicateCollections      func(client *http.Client, cfg Config, sections []plexSection, logger *AppLogger) error
	deleteNonSmartCollections       func(client *http.Client, cfg Config, sections []plexSection, logger *AppLogger) error
	pathCleanCollectionTitles       func(client *http.Client, cfg Config, sections []plexSection, logger *AppLogger) error
	processCollections              func(client *http.Client, cfg Config, libraryName string, collections []plexCollection, upload bool, labelTypeCollectionItems bool, missingPostersOnly bool, logger *AppLogger) error
	fetchCollectionDetails          func(client *http.Client, cfg Config, ratingKey string, logger *AppLogger) (collectionTransferRecord, error)
	createCollection                func(client *http.Client, cfg Config, sourceSectionKey, targetSectionKey, title string, targetTypeCode int, collection collectionTransferRecord, logger *AppLogger) error
	createLibraryFromSection        func(client *http.Client, cfg Config, sourceDetail plexSectionDetail, newName string, locations []string, logger *AppLogger) (plexSection, error)
	applySectionPreferences         func(client *http.Client, cfg Config, sectionKey string, prefs []plexSectionPref, logger *AppLogger) error
	deleteNonSmartCollectionEntries func(client *http.Client, cfg Config, entries []collectionInventoryEntry, logger *AppLogger) ([]CoreCollectionDeleteRow, int, int, error)
	buildDuplicateCollectionRows    func(entries []collectionInventoryEntry) ([][]string, int)
	fetchCollectionInventory        func(client *http.Client, cfg Config, sectionKey string, logger *AppLogger) ([]collectionInventoryEntry, error)
	fetchCollectionItems            func(client *http.Client, cfg Config, ratingKey string, logger *AppLogger) ([]plexLibraryItem, error)
	updateLibraryItemTitle          func(client *http.Client, cfg Config, ratingKey, title string, logger *AppLogger) error
	sectionTypeToPlexTypeCode       func(sectionType string) (int, error)
	normalizeCollectionName         func(in string) string
	composeCollectionContent        func(baseURI, content string) string
	libraryItemFilePath             func(item plexLibraryItem) string
	pathCleanTitleFromFilePath      func(filePath string, replacements map[string]string) string
	defaultCloneLibraryName         func(sourceTitle string) string

	writeCSVReport                        func(path string, header []string, rows [][]string) error
	uniqueCollectionReportPath            func(outputDir, prefix string, now time.Time) string
	uniqueCollectionReportPathForLibrary  func(outputDir, prefix, libraryName string, now time.Time) string
	uniquePathCleanReportPath             func(outputDir string, now time.Time) string
	uniquePathCleanReportPathForLibrary   func(outputDir, libraryName string, now time.Time) string
	uniquePosterReportPath                func(outputDir string, now time.Time) string
	uniqueStatsReportPath                 func(outputDir string, now time.Time) string
	uniqueStatsReportPathForLibrary       func(outputDir, libraryName string, now time.Time) string
	uniqueCleanReportPath                 func(outputDir string, now time.Time) string
	uniqueCleanReportPathForLibrary       func(outputDir, libraryName string, now time.Time) string
	uniqueLabelReportPath                 func(outputDir string, now time.Time) string
	uniqueLabelReportPathForLibrary       func(outputDir, libraryName string, now time.Time) string
	resolveCollectionExportPathForLibrary func(baseExportPath, libraryName string, now time.Time) string

	sanitizeFileName   func(in string) string
	setRequestContext  func(ctx context.Context) func()
	newProgressTracker func(logger *AppLogger, label string, total int) ProgressTrackerCompat
)

type ProgressTrackerCompat interface {
	Advance()
	Finish()
}

type CoreCollectionDeleteRow struct {
	Title     string
	RatingKey string
	ItemCount int
	Smart     bool
	Status    string
}

type Deps struct {
	AppDisplayName                    string
	LoadConfig                        func(path string) (Config, error)
	FetchSections                     func(client *http.Client, cfg Config, logger *AppLogger) ([]plexSection, error)
	FetchCollections                  func(client *http.Client, cfg Config, sectionKey string, logger *AppLogger) ([]plexCollection, error)
	FetchSectionDetail                func(client *http.Client, cfg Config, sectionKey string, logger *AppLogger) (plexSectionDetail, error)
	FetchSectionPreferences           func(client *http.Client, cfg Config, sectionKey string, logger *AppLogger) ([]plexSectionPref, error)
	ExtractSectionLocations           func(detail plexSectionDetail) []string
	EnsureLibraryNameAvailable        func(sections []plexSection, name string) error
	ResolveCollectionTransferPath     func(cfg Config, collFilePath string) string
	ResolveCollectionExportPath       func(cfg Config, collFilePath string, now time.Time) string
	ResolveCollectionExportPathForLib func(baseExportPath, libraryName string, now time.Time) string
	ResolvePathRelativeToConfig       func(configPath, candidate string) string
	BackupArchiveDir                  func(workspaceRoot string) string
	FormatBackupDateTime              func(value time.Time) string
	MergePlexConfig                   func(target, supplemental *Config)
	LoadSupplementalConfig            func(basePath, supplementalPath, fieldName string, target any) error
	LoadOpsConfig                     func(path string) (Config, error)
	CreateBackupArchive               func(cfg Config, configPath string, logger *AppLogger) error
	RestoreFromBackup                 func(cfg Config, restoreFilter string, logger *AppLogger) error
	RollbackLastRestore               func(cfg Config, logger *AppLogger) error
	ListBackupArchives                func(backupsDir string) ([]backupCandidate, error)
	ParseLabelList                    func(raw string) ([]string, error)
	SelectSingleSection               func(sections []plexSection) (plexSection, error)
	LabelMatchingItems                func(client *http.Client, cfg Config, selectedSection plexSection, finds []string, labelsToAdd []string, categoriesToAdd []string, updateCategory bool, onlyCategory bool, logger *AppLogger) error
	CleanLibraryTitles                func(client *http.Client, cfg Config, sections []plexSection, translateEnabled bool, logger *AppLogger) error
	TranslateLibraryTitles            func(client *http.Client, cfg Config, sections []plexSection, logger *AppLogger) error
	AnalyzeLibraryFileNameStats       func(client *http.Client, cfg Config, sections []plexSection, logger *AppLogger) error
	ExportCollections                 func(client *http.Client, cfg Config, sections []plexSection, exportPath string, logger *AppLogger) error
	ImportCollections                 func(client *http.Client, cfg Config, sections []plexSection, importPath string, logger *AppLogger) error
	InjectCollections                 func(client *http.Client, cfg Config, sections []plexSection, logger *AppLogger) error
	ReportDuplicateCollections        func(client *http.Client, cfg Config, sections []plexSection, logger *AppLogger) error
	DeleteNonSmartCollections         func(client *http.Client, cfg Config, sections []plexSection, logger *AppLogger) error
	PathCleanCollectionTitles         func(client *http.Client, cfg Config, sections []plexSection, logger *AppLogger) error
	ProcessCollections                func(client *http.Client, cfg Config, libraryName string, collections []plexCollection, upload bool, labelTypeCollectionItems bool, missingPostersOnly bool, logger *AppLogger) error
	FetchCollectionDetails            func(client *http.Client, cfg Config, ratingKey string, logger *AppLogger) (collectionTransferRecord, error)
	CreateCollection                  func(client *http.Client, cfg Config, sourceSectionKey, targetSectionKey, title string, targetTypeCode int, collection collectionTransferRecord, logger *AppLogger) error
	CreateLibraryFromSection          func(client *http.Client, cfg Config, sourceDetail plexSectionDetail, newName string, locations []string, logger *AppLogger) (plexSection, error)
	ApplySectionPreferences           func(client *http.Client, cfg Config, sectionKey string, prefs []plexSectionPref, logger *AppLogger) error
	DeleteNonSmartCollectionEntries   func(client *http.Client, cfg Config, entries []collectionInventoryEntry, logger *AppLogger) ([]CoreCollectionDeleteRow, int, int, error)
	BuildDuplicateCollectionRows      func(entries []collectionInventoryEntry) ([][]string, int)
	FetchCollectionInventory          func(client *http.Client, cfg Config, sectionKey string, logger *AppLogger) ([]collectionInventoryEntry, error)
	FetchCollectionItems              func(client *http.Client, cfg Config, ratingKey string, logger *AppLogger) ([]plexLibraryItem, error)
	UpdateLibraryItemTitle            func(client *http.Client, cfg Config, ratingKey, title string, logger *AppLogger) error
	SectionTypeToPlexTypeCode         func(sectionType string) (int, error)
	NormalizeCollectionName           func(in string) string
	ComposeCollectionContent          func(baseURI, content string) string
	LibraryItemFilePath               func(item plexLibraryItem) string
	PathCleanTitleFromFilePath        func(filePath string, replacements map[string]string) string
	DefaultCloneLibraryName           func(sourceTitle string) string
	WriteCSVReport                    func(path string, header []string, rows [][]string) error
	UniqueCollectionReportPath        func(outputDir, prefix string, now time.Time) string
	UniqueCollectionReportPathForLib  func(outputDir, prefix, libraryName string, now time.Time) string
	UniquePathCleanReportPath         func(outputDir string, now time.Time) string
	UniquePathCleanReportPathForLib   func(outputDir, libraryName string, now time.Time) string
	UniquePosterReportPath            func(outputDir string, now time.Time) string
	UniqueStatsReportPath             func(outputDir string, now time.Time) string
	UniqueStatsReportPathForLib       func(outputDir, libraryName string, now time.Time) string
	UniqueCleanReportPath             func(outputDir string, now time.Time) string
	UniqueCleanReportPathForLib       func(outputDir, libraryName string, now time.Time) string
	UniqueLabelReportPath             func(outputDir string, now time.Time) string
	UniqueLabelReportPathForLib       func(outputDir, libraryName string, now time.Time) string
	SanitizeFileName                  func(in string) string
	SetRequestContext                 func(ctx context.Context) func()
	NewProgressTracker                func(logger *AppLogger, label string, total int) ProgressTrackerCompat
}

func SetDeps(d Deps) {
	if d.AppDisplayName != "" {
		appDisplayName = d.AppDisplayName
	}
	loadConfig = d.LoadConfig
	fetchSections = d.FetchSections
	fetchCollections = d.FetchCollections
	fetchSectionDetail = d.FetchSectionDetail
	fetchSectionPreferences = d.FetchSectionPreferences
	extractSectionLocations = d.ExtractSectionLocations
	ensureLibraryNameAvailable = d.EnsureLibraryNameAvailable
	resolveCollectionTransferPath = d.ResolveCollectionTransferPath
	resolveCollectionExportPath = d.ResolveCollectionExportPath
	resolveCollectionExportPathForLibrary = d.ResolveCollectionExportPathForLib
	resolvePathRelativeToConfig = d.ResolvePathRelativeToConfig
	backupArchiveDir = d.BackupArchiveDir
	formatBackupDateTime = d.FormatBackupDateTime
	mergePlexConfig = d.MergePlexConfig
	loadSupplementalConfig = d.LoadSupplementalConfig
	loadOpsConfig = d.LoadOpsConfig
	createBackupArchive = d.CreateBackupArchive
	restoreFromBackup = d.RestoreFromBackup
	rollbackLastRestore = d.RollbackLastRestore
	listBackupArchives = d.ListBackupArchives
	parseLabelList = d.ParseLabelList
	selectSingleSection = d.SelectSingleSection
	labelMatchingItems = d.LabelMatchingItems
	cleanLibraryTitles = d.CleanLibraryTitles
	translateLibraryTitles = d.TranslateLibraryTitles
	analyzeLibraryFileNameStats = d.AnalyzeLibraryFileNameStats
	exportCollections = d.ExportCollections
	importCollections = d.ImportCollections
	injectCollections = d.InjectCollections
	reportDuplicateCollections = d.ReportDuplicateCollections
	deleteNonSmartCollections = d.DeleteNonSmartCollections
	pathCleanCollectionTitles = d.PathCleanCollectionTitles
	processCollections = d.ProcessCollections
	fetchCollectionDetails = d.FetchCollectionDetails
	createCollection = d.CreateCollection
	createLibraryFromSection = d.CreateLibraryFromSection
	applySectionPreferences = d.ApplySectionPreferences
	deleteNonSmartCollectionEntries = d.DeleteNonSmartCollectionEntries
	buildDuplicateCollectionRows = d.BuildDuplicateCollectionRows
	fetchCollectionInventory = d.FetchCollectionInventory
	fetchCollectionItems = d.FetchCollectionItems
	updateLibraryItemTitle = d.UpdateLibraryItemTitle
	sectionTypeToPlexTypeCode = d.SectionTypeToPlexTypeCode
	normalizeCollectionName = d.NormalizeCollectionName
	composeCollectionContent = d.ComposeCollectionContent
	libraryItemFilePath = d.LibraryItemFilePath
	pathCleanTitleFromFilePath = d.PathCleanTitleFromFilePath
	defaultCloneLibraryName = d.DefaultCloneLibraryName
	writeCSVReport = d.WriteCSVReport
	uniqueCollectionReportPath = d.UniqueCollectionReportPath
	uniqueCollectionReportPathForLibrary = d.UniqueCollectionReportPathForLib
	uniquePathCleanReportPath = d.UniquePathCleanReportPath
	uniquePathCleanReportPathForLibrary = d.UniquePathCleanReportPathForLib
	uniquePosterReportPath = d.UniquePosterReportPath
	uniqueStatsReportPath = d.UniqueStatsReportPath
	uniqueStatsReportPathForLibrary = d.UniqueStatsReportPathForLib
	uniqueCleanReportPath = d.UniqueCleanReportPath
	uniqueCleanReportPathForLibrary = d.UniqueCleanReportPathForLib
	uniqueLabelReportPath = d.UniqueLabelReportPath
	uniqueLabelReportPathForLibrary = d.UniqueLabelReportPathForLib
	sanitizeFileName = d.SanitizeFileName
	setRequestContext = d.SetRequestContext
	newProgressTracker = d.NewProgressTracker
}
