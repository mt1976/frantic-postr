package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
	"golang.org/x/image/font/basicfont"
)

func newTestLogger(console, file io.Writer) *AppLogger {
	return &AppLogger{
		console: log.New(console, "", 0),
		file:    log.New(file, "", 0),
	}
}

func TestBuildOutputPathUsesTemplateExtension(t *testing.T) {
	out := buildOutputPath("/tmp/output", "Sci-Fi Classics", "/tmp/template.jpeg")
	expected := filepath.Join("/tmp/output", "Sci-Fi Classics.jpg")
	if out != expected {
		t.Fatalf("expected %q got %q", expected, out)
	}
}

func TestResolveCollectionTransferPathUsesOutputCollectionsExportForBareFilename(t *testing.T) {
	cfg := Config{OutputDir: "/tmp/output"}
	got := resolveCollectionTransferPath(cfg, "collections-export.json")
	expected := filepath.Join("/tmp/output", "collections-export", "collections-export.json")
	if got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
}

func TestResolveCollectionTransferPathKeepsExplicitDirectory(t *testing.T) {
	cfg := Config{OutputDir: "/tmp/output"}
	got := resolveCollectionTransferPath(cfg, "custom/collections-export.json")
	expected := filepath.Clean("custom/collections-export.json")
	if got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
}

func TestResolveCollectionExportPathAddsDateSuffix(t *testing.T) {
	cfg := Config{OutputDir: "/tmp/output"}
	now := time.Date(2026, time.June, 22, 10, 0, 0, 0, time.UTC)
	got := resolveCollectionExportPath(cfg, "collections-export.json", now)
	expected := filepath.Join("/tmp/output", "collections-export", "collections-export_20260622.json")
	if got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
}

func TestResolveCollectionExportPathDoesNotDoubleAppendDate(t *testing.T) {
	cfg := Config{OutputDir: "/tmp/output"}
	now := time.Date(2026, time.June, 22, 10, 0, 0, 0, time.UTC)
	got := resolveCollectionExportPath(cfg, "collections-export_20260622.json", now)
	expected := filepath.Join("/tmp/output", "collections-export", "collections-export_20260622.json")
	if got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
}

func TestTokenizeStatsWords(t *testing.T) {
	got := tokenizeStatsWords("The.Big-Brown_Fox+1080p 123")
	expected := []string{"the", "big", "brown", "fox", "1080p"}
	if len(got) != len(expected) {
		t.Fatalf("expected %d tokens got %d: %+v", len(expected), len(got), got)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("expected token %d to be %q got %q", i, expected[i], got[i])
		}
	}
}

func TestIsNumericToken(t *testing.T) {
	if !isNumericToken("123") {
		t.Fatal("expected numeric token to be true")
	}
	if isNumericToken("1080p") {
		t.Fatal("expected alpha-numeric token to be false")
	}
}

func TestBuildStatsExcludedWordSetIncludesDefaultsAndConfigWords(t *testing.T) {
	set := buildStatsExcludedWordSet([]string{"  custom  ", "THE", "custom"})
	if _, ok := set["the"]; !ok {
		t.Fatal("expected default stop word to be included")
	}
	if _, ok := set["custom"]; !ok {
		t.Fatal("expected custom word to be included")
	}
}

func TestBuildStatsRowsSortsByCountThenWord(t *testing.T) {
	rows := buildStatsRows(map[string]int{"beta": 2, "alpha": 3, "zeta": 2})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows got %d", len(rows))
	}
	if rows[0][0] != "alpha" || rows[0][1] != "3" {
		t.Fatalf("unexpected first row: %+v", rows[0])
	}
	if rows[1][0] != "beta" || rows[1][1] != "2" {
		t.Fatalf("unexpected second row: %+v", rows[1])
	}
	if rows[2][0] != "zeta" || rows[2][1] != "2" {
		t.Fatalf("unexpected third row: %+v", rows[2])
	}
}

func TestSanitizeFileName(t *testing.T) {
	got := sanitizeFileName(`  A/B:C*D?"E<F>G|  `)
	if got != "A_B-C_D'EFG_" {
		t.Fatalf("unexpected sanitized value: %q", got)
	}
}

func TestNormalizeCollectionMatchKey(t *testing.T) {
	got := normalizeCollectionMatchKey("  My   Admin   Queue ")
	if got != "myadminqueue" {
		t.Fatalf("unexpected normalized key: %q", got)
	}
	withPunctuation := normalizeCollectionMatchKey("Bears (Chubs)")
	if withPunctuation != "bearschubs" {
		t.Fatalf("unexpected punctuation-normalized key: %q", withPunctuation)
	}
}

func TestSelectPosterTemplateUsesCaseAndSpaceInsensitiveMatching(t *testing.T) {
	cfg := Config{
		TemplateImage:      "default.png",
		TypeTemplateImage:  "type.png",
		StudioTemplateImage: "studio.png",
		AdminTemplateImage: "admin.png",
		TypeCollectionSet: map[string]struct{}{
			"mytypecollection": {},
			"myclashcollection": {},
		},
		StudioCollectionSet: map[string]struct{}{
			"mystudiocollection": {},
			"myclashcollection":  {},
		},
		AdminCollectionSet: map[string]struct{}{
			"myadminqueue": {},
		},
	}

	path, background := selectPosterTemplate(cfg, " My Admin Queue ")
	if path != "admin.png" || background != "admin" {
		t.Fatalf("expected admin template, got path=%q background=%q", path, background)
	}

	path, background = selectPosterTemplate(cfg, "My Type Collection")
	if path != "type.png" || background != "type" {
		t.Fatalf("expected type template, got path=%q background=%q", path, background)
	}

	path, background = selectPosterTemplate(cfg, "My Studio Collection")
	if path != "studio.png" || background != "studio" {
		t.Fatalf("expected studio template, got path=%q background=%q", path, background)
	}

	// Clash case: studio should override type.
	path, background = selectPosterTemplate(cfg, "My Clash Collection")
	if path != "studio.png" || background != "studio" {
		t.Fatalf("expected studio to override type, got path=%q background=%q", path, background)
	}

	path, background = selectPosterTemplate(cfg, "Other")
	if path != "default.png" || background != "default" {
		t.Fatalf("expected default template, got path=%q background=%q", path, background)
	}
}

func TestRenderCollectionPosterCreatesOutput(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "template.png")

	template := image.NewRGBA(image.Rect(0, 0, 200, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 200; x++ {
			template.Set(x, y, color.RGBA{R: 25, G: 25, B: 50, A: 255})
		}
	}
	f, err := os.Create(templatePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, template); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	_ = f.Close()

	cfg := Config{TemplateImage: templatePath}
	cfg.Font.Color = "#FFFFFF"
	cfg.Font.ShadowColor = "#000000"
	cfg.Font.GlowColor = "#000000"
	cfg.Font.GlowAlpha = 0.4
	cfg.Font.GlowRadius = 1
	cfg.Font.ShadowOffsetX = 1
	cfg.Font.ShadowOffsetY = 1

	outputPath := filepath.Join(dir, "Collection.png")
	if err := renderCollectionPoster(cfg, "Collection", outputPath); err != nil {
		t.Fatalf("renderCollectionPoster failed: %v", err)
	}

	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected output file to exist: %v", err)
	}
}

func TestRetryBackoff(t *testing.T) {
	// Pin jitter to 1.0 so the result equals the cap exactly — deterministic.
	orig := randFloat64
	randFloat64 = func() float64 { return 1.0 }
	defer func() { randFloat64 = orig }()

	tests := []struct {
		attempt   int
		baseMs    int
		maxMs     int
		wantExact time.Duration // with jitter=1.0, result == cap * 1ms
	}{
		// attempt 0: cap = min(30000, 500*2^0) = 500ms
		{0, 500, 30000, 500 * time.Millisecond},
		// attempt 1: cap = min(30000, 500*2^1) = 1000ms
		{1, 500, 30000, 1000 * time.Millisecond},
		// attempt 2: cap = min(30000, 500*2^2) = 2000ms
		{2, 500, 30000, 2000 * time.Millisecond},
		// attempt 10: cap = min(30000, 500*1024) = 30000ms (capped)
		{10, 500, 30000, 30000 * time.Millisecond},
	}
	for _, tt := range tests {
		got := retryBackoff(tt.attempt, tt.baseMs, tt.maxMs)
		if got != tt.wantExact {
			t.Errorf("retryBackoff(attempt=%d, base=%d, max=%d) = %v, want %v",
				tt.attempt, tt.baseMs, tt.maxMs, got, tt.wantExact)
		}
	}

	// With jitter=0.0 the result should always be 0.
	randFloat64 = func() float64 { return 0.0 }
	if got := retryBackoff(5, 500, 30000); got != 0 {
		t.Errorf("with jitter=0, expected 0, got %v", got)
	}
}

func TestUniqueRunLogPathAddsTimestampBeforeExtension(t *testing.T) {
	now := time.Date(2026, time.June, 3, 14, 5, 6, 0, time.UTC)
	got := uniqueRunLogPath(filepath.Join("logs", "frantic-postr.log"), now)
	expected := filepath.Join("logs", "frantic-postr-20260603-140506.log")
	if got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
}

func TestDoPlexGETLogsRunnableCurlCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("X-Plex-Token"); got != "secret-token" {
			t.Fatalf("expected token query param, got %q", got)
		}
		_, _ = w.Write([]byte("<MediaContainer/>"))
	}))
	defer server.Close()

	var buf bytes.Buffer
	logger := newTestLogger(&buf, io.Discard)

	if _, err := doPlexGET(server.Client(), server.URL+"/library/sections", Config{Plex: struct {
		BaseURL     string `toml:"base_url"`
		Token       string `toml:"token"`
		Retries     int    `toml:"retries"`
		Workers     int    `toml:"workers"`
		RetryBaseMs int    `toml:"retry_base_ms"`
		RetryMaxMs  int    `toml:"retry_max_ms"`
	}{Token: "secret-token", Retries: 3, RetryBaseMs: 500, RetryMaxMs: 30000}}, logger); err != nil {
		t.Fatalf("doPlexGET failed: %v", err)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "plex curl: curl -X 'GET' '") {
		t.Fatalf("expected curl command in log output, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "X-Plex-Token=secret-token") {
		t.Fatalf("expected token in logged curl command, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "/library/sections") {
		t.Fatalf("expected endpoint in logged curl command, got %q", logOutput)
	}
}

func TestQuietLoggerSuppressesAPIOutputOnConsole(t *testing.T) {
	var console bytes.Buffer
	var file bytes.Buffer
	logger := &AppLogger{
		console: log.New(&console, "", 0),
		file:    log.New(&file, "", 0),
		quiet:   true,
	}

	logger.Infof("progress message")
	logger.Successf("poster created: %s", "Collection.png")
	logger.APIf("plex call: GET %s", "https://example.test/library/sections")

	consoleOutput := console.String()
	if strings.Contains(consoleOutput, "progress message") {
		t.Fatalf("expected info log to be hidden from console, got %q", consoleOutput)
	}
	if !strings.Contains(consoleOutput, "poster created: Collection.png") {
		t.Fatalf("expected success log to remain visible on console, got %q", consoleOutput)
	}
	if strings.Contains(consoleOutput, "plex call:") {
		t.Fatalf("expected API log to be hidden from console, got %q", consoleOutput)
	}
	if !strings.Contains(file.String(), "INFO progress message") {
		t.Fatalf("expected progress message in file log, got %q", file.String())
	}
	if !strings.Contains(file.String(), "SUCCESS poster created: Collection.png") {
		t.Fatalf("expected success log in file log, got %q", file.String())
	}
	if !strings.Contains(file.String(), "API plex call: GET https://example.test/library/sections") {
		t.Fatalf("expected API log in file log, got %q", file.String())
	}
}

func TestQuietProgressTrackerRendersToConsole(t *testing.T) {
	var console bytes.Buffer
	logger := &AppLogger{console: log.New(&console, "", 0), quiet: true}
	progress := newProgressTracker(logger, "gen posters", 3)
	if progress == nil {
		t.Fatal("expected progress tracker in quiet mode")
	}
	progress.Advance()
	progress.Advance()
	progress.Finish()

	output := console.String()
	if !strings.Contains(output, "gen posters [") {
		t.Fatalf("expected progress bar output, got %q", output)
	}
	if !strings.Contains(output, "3/3") {
		t.Fatalf("expected completed count in progress output, got %q", output)
	}
	if strings.Contains(output, "plex call:") {
		t.Fatalf("expected API log to be hidden from console, got %q", console.String())
	}
}

func TestMainWithoutModeFlagsPrintsHelp(t *testing.T) {
	origArgs := os.Args
	origCommandLine := flag.CommandLine
	defer func() {
		os.Args = origArgs
		flag.CommandLine = origCommandLine
	}()

	var buf bytes.Buffer
	flag.CommandLine = flag.NewFlagSet(origArgs[0], flag.ContinueOnError)
	flag.CommandLine.SetOutput(&buf)
	os.Args = []string{origArgs[0]}

	main()

	output := buf.String()
	if !strings.Contains(output, "-gen-posters") {
		t.Fatalf("expected help output to include -gen-posters, got %q", output)
	}
	if !strings.Contains(output, "-coll-dupes") {
		t.Fatalf("expected help output to include -coll-dupes, got %q", output)
	}
	if !strings.Contains(output, "-coll-delete-non-smart") {
		t.Fatalf("expected help output to include -coll-delete-non-smart, got %q", output)
	}
	if !strings.Contains(output, "-upload-posters") {
		t.Fatalf("expected help output to include -upload-posters, got %q", output)
	}
	oldFlagRe := regexp.MustCompile(`(?m)^\s+-process(\s|$)|^\s+-upload(\s|$)|^\s+-col-inject(\s|$)|^\s+-coll-impot(\s|$)`)
	if oldFlagRe.MatchString(output) {
		t.Fatalf("expected help output to omit old aliases, got %q", output)
	}
}

func TestFetchCollectionsReturnsTitlesAndGUIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<MediaContainer size="2"><Directory ratingKey="357900" guid="collection://21b2716a-a84a-429b-abbc-6c89312b636d" title="D4ddy_T"></Directory><Directory ratingKey="333373" guid="collection://9c80b1ca-358f-4068-864d-d57e420ab705" title="RuggerLad69"></Directory></MediaContainer>`))
	}))
	defer server.Close()

	var cfg Config
	cfg.Plex.BaseURL = server.URL
	cfg.Plex.Token = "secret-token"

	collections, err := fetchCollections(server.Client(), cfg, "73", newTestLogger(io.Discard, io.Discard))
	if err != nil {
		t.Fatalf("fetchCollections failed: %v", err)
	}
	if len(collections) != 2 {
		t.Fatalf("expected 2 collections got %d", len(collections))
	}
	if collections[0].Title != "D4ddy T" || collections[0].GUID != "collection://21b2716a-a84a-429b-abbc-6c89312b636d" {
		t.Fatalf("unexpected first collection: %+v", collections[0])
	}
	if collections[0].RatingKey != "357900" {
		t.Fatalf("unexpected first collection rating key: %+v", collections[0])
	}
	if collections[1].Title != "RuggerLad69" || collections[1].GUID != "collection://9c80b1ca-358f-4068-864d-d57e420ab705" {
		t.Fatalf("unexpected second collection: %+v", collections[1])
	}
}

func TestFetchCollectionItemCountUsesAllLeavesPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/library/collections/357900/allLeaves" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`<MediaContainer size="4"></MediaContainer>`))
	}))
	defer server.Close()

	var cfg Config
	cfg.Plex.BaseURL = server.URL
	cfg.Plex.Token = "secret-token"

	count, err := fetchCollectionItemCount(server.Client(), cfg, "357900", newTestLogger(io.Discard, io.Discard))
	if err != nil {
		t.Fatalf("fetchCollectionItemCount failed: %v", err)
	}
	if count != 4 {
		t.Fatalf("expected count 4, got %d", count)
	}
}

func TestFetchCollectionItemsUsesItemsPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/library/collections/357900/items" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`<MediaContainer size="1"><Video ratingKey="365142" title="Example"><Media><Part file="V:\\FILTH\\TORRENT\\chalate2000\\0gnfqk81vt7lmlg8b4ykb_source.mp4"/></Media></Video></MediaContainer>`))
	}))
	defer server.Close()

	var cfg Config
	cfg.Plex.BaseURL = server.URL
	cfg.Plex.Token = "secret-token"

	items, err := fetchCollectionItems(server.Client(), cfg, "357900", newTestLogger(io.Discard, io.Discard))
	if err != nil {
		t.Fatalf("fetchCollectionItems failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].RatingKey != "365142" {
		t.Fatalf("unexpected rating key: %q", items[0].RatingKey)
	}
}

func TestLabelTypeCollectionItemsFromCollectionSkipsWhenNotTypeCollection(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		t.Fatalf("expected no Plex requests when collection is not in type set, got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	var cfg Config
	cfg.Plex.BaseURL = server.URL
	cfg.Plex.Token = "secret-token"
	cfg.TypeCollectionSet = map[string]struct{}{
		"sometype": {},
	}

	updated, skipped, err := labelTypeCollectionItemsFromCollection(
		server.Client(),
		cfg,
		plexCollection{RatingKey: "357900", Title: "Other"},
		"Other",
		newTestLogger(io.Discard, io.Discard),
	)
	if err != nil {
		t.Fatalf("labelTypeCollectionItemsFromCollection failed: %v", err)
	}
	if updated != 0 || skipped != 0 {
		t.Fatalf("expected no-op result, got updated=%d skipped=%d", updated, skipped)
	}
	if requestCount != 0 {
		t.Fatalf("expected zero requests, got %d", requestCount)
	}
}

func TestLabelTypeCollectionItemsFromCollectionAddsLabelWithoutDuplicates(t *testing.T) {
	putCalls := 0
	var putLabels []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/library/collections/357900/items":
			_, _ = w.Write([]byte(`<MediaContainer size="2"><Video ratingKey="365142" title="Needs Label"><Label tag="Existing"/></Video><Video ratingKey="365143" title="Already Labeled"><Label tag="My Type Collection"/></Video></MediaContainer>`))
			return
		case r.Method == http.MethodPut && r.URL.Path == "/library/metadata/365142":
			putCalls++
			q := r.URL.Query()
			putLabels = append(putLabels, q.Get("label[0].tag.tag"), q.Get("label[1].tag.tag"))
			w.WriteHeader(http.StatusOK)
			return
		case r.Method == http.MethodPut && r.URL.Path == "/library/metadata/365143":
			t.Fatalf("did not expect PUT for item that already has target label")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	var cfg Config
	cfg.Plex.BaseURL = server.URL
	cfg.Plex.Token = "secret-token"
	cfg.Plex.Retries = 1
	cfg.Plex.RetryBaseMs = 1
	cfg.Plex.RetryMaxMs = 1
	cfg.TypeCollectionSet = map[string]struct{}{
		"mytypecollection": {},
	}

	updated, skipped, err := labelTypeCollectionItemsFromCollection(
		server.Client(),
		cfg,
		plexCollection{RatingKey: "357900", Title: "My Type Collection"},
		"My Type Collection",
		newTestLogger(io.Discard, io.Discard),
	)
	if err != nil {
		t.Fatalf("labelTypeCollectionItemsFromCollection failed: %v", err)
	}
	if updated != 1 {
		t.Fatalf("expected 1 updated item, got %d", updated)
	}
	if skipped != 1 {
		t.Fatalf("expected 1 skipped item, got %d", skipped)
	}
	if putCalls != 1 {
		t.Fatalf("expected exactly one label update call, got %d", putCalls)
	}
	if len(putLabels) != 2 || putLabels[0] != "Existing" || putLabels[1] != "My Type Collection" {
		t.Fatalf("unexpected merged labels in PUT call: %v", putLabels)
	}
}

func TestBuildDuplicateCollectionRowsGroupsByTitle(t *testing.T) {
	rows, dupGroups := buildDuplicateCollectionRows([]collectionInventoryEntry{
		{Title: "Duplicate", RatingKey: "1", ItemCount: 3, Smart: true},
		{Title: "duplicate", RatingKey: "2", ItemCount: 5, Smart: false},
		{Title: "Unique", RatingKey: "3", ItemCount: 7, Smart: false},
	})

	if dupGroups != 1 {
		t.Fatalf("expected 1 duplicate group, got %d", dupGroups)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 duplicate rows, got %d", len(rows))
	}
	if rows[0][0] != "Duplicate" || rows[1][0] != "duplicate" {
		t.Fatalf("unexpected duplicate rows: %+v", rows)
	}
	if rows[0][4] != "2" || rows[1][4] != "2" {
		t.Fatalf("expected duplicate group size 2, got %+v", rows)
	}
}

func TestDeleteNonSmartCollectionEntriesDeletesOnlyNonSmart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/library/sections/42/collection/357900" {
			t.Fatalf("unexpected delete path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("X-Plex-Token"); got != "secret-token" {
			t.Fatalf("unexpected token: %s", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	var cfg Config
	cfg.Plex.BaseURL = server.URL
	cfg.Plex.Token = "secret-token"
	cfg.Plex.Retries = 1
	cfg.Plex.RetryBaseMs = 1
	cfg.Plex.RetryMaxMs = 1

	rows, deleted, failures, err := deleteNonSmartCollectionEntries(server.Client(), cfg, []collectionInventoryEntry{
		{Title: "Smart", RatingKey: "111", SectionKey: "42", ItemCount: 2, Smart: true},
		{Title: "Plain", RatingKey: "357900", SectionKey: "42", ItemCount: 7, Smart: false},
	}, newTestLogger(io.Discard, io.Discard))
	if err != nil {
		t.Fatalf("deleteNonSmartCollectionEntries failed: %v", err)
	}
	if deleted != 1 || failures != 0 {
		t.Fatalf("unexpected delete summary: deleted=%d failures=%d", deleted, failures)
	}
	if len(rows) != 1 || rows[0].Title != "Plain" || rows[0].Status != "deleted" {
		t.Fatalf("unexpected deletion rows: %+v", rows)
	}
}

func TestNormalizeCollectionNameReplacesSpecialCharsWithSpaces(t *testing.T) {
	got := normalizeCollectionName("  A.B*C_D  ")
	if got != "A B C D" {
		t.Fatalf("expected normalized collection name, got %q", got)
	}
}

func TestWrapTextToWidthBreaksOnWordBoundaries(t *testing.T) {
	maxWidth := textPixelWidth(basicfont.Face7x13, "Alpha") + 2
	lines := wrapTextToWidth(basicfont.Face7x13, "Alpha Beta Gamma", maxWidth)
	if len(lines) < 2 {
		t.Fatalf("expected wrapped lines, got %v", lines)
	}
	if lines[0] != "Alpha" {
		t.Fatalf("expected first line to break naturally at word boundary, got %q", lines[0])
	}
}

func TestPosterDisplayTextUppercasesAndPreservesBreaks(t *testing.T) {
	got := posterDisplayText("Top 100 Movies")
	if got != "TOP 100\nMOVIES" {
		t.Fatalf("expected uppercase poster text with preserved break, got %q", got)
	}
}

func TestPosterTextPaddingLeavesExtraMargin(t *testing.T) {
	cfg := Config{}
	cfg.Font.Size = 64
	cfg.Font.GlowRadius = 1
	cfg.Font.ShadowOffsetX = 1
	cfg.Font.ShadowOffsetY = 1

	horizontal, vertical := posterTextPadding(cfg, 1000, 1500)
	if horizontal <= 20 || vertical <= 20 {
		t.Fatalf("expected padding larger than the previous inset, got x=%d y=%d", horizontal, vertical)
	}
	if horizontal < 48 || vertical < 57 {
		t.Fatalf("expected a safer inset for poster text, got x=%d y=%d", horizontal, vertical)
	}
}

func TestDisambiguateCollectionsByGUID(t *testing.T) {
	in := []plexCollection{
		{Title: "Duplicate Name", GUID: "guid-1"},
		{Title: "Duplicate Name", GUID: "guid-2"},
		{Title: "Duplicate Name", GUID: "guid-1"},
		{Title: "Unique", GUID: "guid-3"},
	}
	out := disambiguateCollectionsByGUID(in)

	if out[0].Title != "Duplicate Name 1" {
		t.Fatalf("unexpected first disambiguated title: %q", out[0].Title)
	}
	if out[1].Title != "Duplicate Name 2" {
		t.Fatalf("unexpected second disambiguated title: %q", out[1].Title)
	}
	if out[2].Title != "Duplicate Name 1" {
		t.Fatalf("expected same GUID to keep same sequence, got %q", out[2].Title)
	}
	if out[3].Title != "Unique" {
		t.Fatalf("unexpected unique title mutation: %q", out[3].Title)
	}
}

func TestForceLineBreakAfterNumber(t *testing.T) {
	got := forceLineBreakAfterNumber("Top 100 Movies")
	if got != "Top 100\nMovies" {
		t.Fatalf("expected forced line break after number, got %q", got)
	}
}

func TestForceLineBreakAfterNumberSkipsEmbeddedDigits(t *testing.T) {
	if got := forceLineBreakAfterNumber("Version 5.1 Ready"); got != "Version 5.1 Ready" {
		t.Fatalf("expected no break for dotted number, got %q", got)
	}
	if got := forceLineBreakAfterNumber("d4ddy t"); got != "d4ddy t" {
		t.Fatalf("expected no break for embedded number, got %q", got)
	}
}

func TestForceLineBreakAfterNumberBreaksSeparatedNumbers(t *testing.T) {
	got := forceLineBreakAfterNumber("hello 123 345")
	if got != "hello 123\n345" {
		t.Fatalf("expected breaks after standalone numbers, got %q", got)
	}
}

func TestParseSelectionInput(t *testing.T) {
	indices, err := parseSelectionInput("1,3,2", 3)
	if err != nil {
		t.Fatalf("parseSelectionInput failed: %v", err)
	}
	expected := []int{0, 2, 1}
	if len(indices) != len(expected) {
		t.Fatalf("expected %d indices got %d", len(expected), len(indices))
	}
	for i := range expected {
		if indices[i] != expected[i] {
			t.Fatalf("unexpected index at %d: got %d want %d", i, indices[i], expected[i])
		}
	}
}

func TestParseSelectionInputAll(t *testing.T) {
	indices, err := parseSelectionInput("all", 3)
	if err != nil {
		t.Fatalf("parseSelectionInput all failed: %v", err)
	}
	expected := []int{0, 1, 2}
	for i := range expected {
		if indices[i] != expected[i] {
			t.Fatalf("unexpected index at %d: got %d want %d", i, indices[i], expected[i])
		}
	}
}

func TestParseSelectionInputInvalid(t *testing.T) {
	if _, err := parseSelectionInput("4", 3); err == nil {
		t.Fatal("expected invalid selection error")
	}
}

func TestParsePageCommand(t *testing.T) {
	if got := parsePageCommand("f"); got != 1 {
		t.Fatalf("expected f to be next page, got %d", got)
	}
	if got := parsePageCommand("N"); got != 1 {
		t.Fatalf("expected N to be next page, got %d", got)
	}
	if got := parsePageCommand("b"); got != -1 {
		t.Fatalf("expected b to be previous page, got %d", got)
	}
	if got := parsePageCommand("3"); got != 0 {
		t.Fatalf("expected non-page command to return 0, got %d", got)
	}
}

func TestPageSlice(t *testing.T) {
	start, end, pages := pageSlice(23, 0, 10)
	if start != 0 || end != 10 || pages != 3 {
		t.Fatalf("unexpected first page: start=%d end=%d pages=%d", start, end, pages)
	}
	start, end, pages = pageSlice(23, 2, 10)
	if start != 20 || end != 23 || pages != 3 {
		t.Fatalf("unexpected last page: start=%d end=%d pages=%d", start, end, pages)
	}
	start, end, pages = pageSlice(23, 99, 10)
	if start != 20 || end != 23 || pages != 3 {
		t.Fatalf("expected out-of-range page to clamp to last: start=%d end=%d pages=%d", start, end, pages)
	}
}

func TestUploadCollectionPoster(t *testing.T) {
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "poster.png")
	if err := os.WriteFile(imagePath, []byte("image-data"), 0o644); err != nil {
		t.Fatalf("failed to create image file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/library/collections/357900/posters" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("X-Plex-Token"); got != "secret-token" {
			t.Fatalf("expected token query param, got %q", got)
		}

		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("failed parsing content type: %v", err)
		}
		if mediaType != "multipart/form-data" {
			t.Fatalf("expected multipart content type, got %s", mediaType)
		}

		if err := r.ParseMultipartForm(2 << 20); err != nil {
			t.Fatalf("failed parsing multipart form: %v", err)
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("missing file form field: %v", err)
		}
		defer file.Close()
		bytes, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("failed reading upload: %v", err)
		}
		if string(bytes) != "image-data" {
			t.Fatalf("unexpected upload content: %q", string(bytes))
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	var cfg Config
	cfg.Plex.BaseURL = server.URL
	cfg.Plex.Token = "secret-token"
	collection := plexCollection{RatingKey: "357900", Title: "Sample", GUID: "guid-1"}

	if err := uploadCollectionPoster(server.Client(), cfg, collection, imagePath, newTestLogger(io.Discard, io.Discard)); err != nil {
		t.Fatalf("uploadCollectionPoster failed: %v", err)
	}
}

func TestRewriteCollectionContentURI(t *testing.T) {
	in := "server://abc/com.plexapp.plugins.library/library/sections/73/all?type=2&sort=titleSort%3Aasc"
	out := rewriteCollectionContentURI(in, "73", "15")
	if !strings.Contains(out, "/library/sections/15/") {
		t.Fatalf("expected rewritten section key, got %q", out)
	}
}

func TestCreateCollectionSmartRequestUsesRewrittenURI(t *testing.T) {
	var captured url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.Query()
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	var cfg Config
	cfg.Plex.BaseURL = server.URL
	cfg.Plex.Token = "secret-token"

	record := collectionTransferRecord{
		Title:   "Recent",
		Smart:   true,
		Content: "server://abc/com.plexapp.plugins.library/library/sections/73/all?type=2",
	}
	if err := createCollection(server.Client(), cfg, "73", "15", "Recent", 2, record, newTestLogger(io.Discard, io.Discard)); err != nil {
		t.Fatalf("createCollection failed: %v", err)
	}

	if captured.Get("smart") != "1" {
		t.Fatalf("expected smart=1 got %q", captured.Get("smart"))
	}
	if captured.Get("sectionId") != "15" {
		t.Fatalf("expected sectionId=15 got %q", captured.Get("sectionId"))
	}
	if !strings.Contains(captured.Get("uri"), "/library/sections/15/") {
		t.Fatalf("expected uri section rewrite, got %q", captured.Get("uri"))
	}
}

func TestCollectionTransferFileJSONRoundTrip(t *testing.T) {
	in := collectionTransferFile{
		Version:       1,
		ExportedAtUTC: "2026-06-05T10:00:00Z",
		SourceLibrary: plexSection{Key: "73", Title: "Source", Type: "show"},
		Collections: []collectionTransferRecord{{
			Title:   "Smart One",
			Smart:   true,
			Content: "server://abc/com.plexapp.plugins.library/library/sections/73/all?type=2",
		}},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var out collectionTransferFile
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if out.SourceLibrary.Key != "73" || len(out.Collections) != 1 || !out.Collections[0].Smart {
		t.Fatalf("unexpected round-trip value: %+v", out)
	}
}

func TestDefaultCloneLibraryName(t *testing.T) {
	if got := defaultCloneLibraryName("Movies"); got != "Movies-clone" {
		t.Fatalf("expected Movies-clone got %q", got)
	}
	if got := defaultCloneLibraryName("   "); got != "library-clone" {
		t.Fatalf("expected library-clone for empty source name, got %q", got)
	}
}

func TestBuildCreateLibraryURLIncludesSettingsAndLocations(t *testing.T) {
	source := plexSectionDetail{
		Type:     "show",
		Agent:    "tv.plex.agents.series",
		Scanner:  "Plex TV Series",
		Language: "en-US",
	}
	u, err := buildCreateLibraryURL("http://localhost:32400", "secret-token", source, "TV Clone", []string{"/media/tv", "/media/tv2"})
	if err != nil {
		t.Fatalf("buildCreateLibraryURL failed: %v", err)
	}

	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	q := parsed.Query()
	if q.Get("name") != "TV Clone" {
		t.Fatalf("expected name query param, got %q", q.Get("name"))
	}
	if q.Get("type") != "show" {
		t.Fatalf("expected type query param, got %q", q.Get("type"))
	}
	if q.Get("agent") != "tv.plex.agents.series" {
		t.Fatalf("expected agent query param, got %q", q.Get("agent"))
	}
	if q.Get("scanner") != "Plex TV Series" {
		t.Fatalf("expected scanner query param, got %q", q.Get("scanner"))
	}
	if q.Get("language") != "en-US" {
		t.Fatalf("expected language query param, got %q", q.Get("language"))
	}
	if q.Get("X-Plex-Token") != "secret-token" {
		t.Fatalf("expected token query param, got %q", q.Get("X-Plex-Token"))
	}
	locations := q["location"]
	if len(locations) != 2 || locations[0] != "/media/tv" || locations[1] != "/media/tv2" {
		t.Fatalf("expected two location mappings, got %+v", locations)
	}
}

func TestParseLabelList(t *testing.T) {
	labels, err := parseLabelList("urbsex, abandoned ,urbsex")
	if err != nil {
		t.Fatalf("parseLabelList failed: %v", err)
	}
	if len(labels) != 2 || labels[0] != "urbsex" || labels[1] != "abandoned" {
		t.Fatalf("unexpected parsed labels: %+v", labels)
	}
}

func TestNormalizeTagList(t *testing.T) {
	labels, err := normalizeTagList([]string{" urbsex ", "", "abandoned", "URBSEX"})
	if err != nil {
		t.Fatalf("normalizeTagList failed: %v", err)
	}
	if len(labels) != 2 || labels[0] != "urbsex" || labels[1] != "abandoned" {
		t.Fatalf("unexpected normalized labels: %+v", labels)
	}
}

func TestLoadConfigNormalizesLabelLookups(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "template.png")
	if err := os.WriteFile(templatePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	configPath := filepath.Join(dir, "config.toml")
	configBody := strings.Join([]string{
		"template_image = \"" + templatePath + "\"",
		"output_dir = \"" + dir + "\"",
		"log_file = \"" + filepath.Join(dir, "run.log") + "\"",
		"",
		"[plex]",
		"base_url = \"http://127.0.0.1:32400\"",
		"token = \"token\"",
		"retries = 3",
		"",
		"[[label.lookup]]",
		"find = \"abandoned\"",
		"labels = [\"urbsex\", \"abandoned\", \"URBSEX\"]",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(configBody), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	if len(cfg.Label.Lookups) != 1 {
		t.Fatalf("expected 1 lookup, got %d", len(cfg.Label.Lookups))
	}
	lookup := cfg.Label.Lookups[0]
	if lookup.TitleContains != "abandoned" {
		t.Fatalf("expected title_contains from find alias, got %q", lookup.TitleContains)
	}
	if len(lookup.TitleContainsAny) != 1 || lookup.TitleContainsAny[0] != "abandoned" {
		t.Fatalf("expected normalized title_contains_any, got %+v", lookup.TitleContainsAny)
	}
	if len(lookup.Labels) != 2 || lookup.Labels[0] != "urbsex" || lookup.Labels[1] != "abandoned" {
		t.Fatalf("unexpected lookup labels: %+v", lookup.Labels)
	}
	if len(lookup.Categories) != 2 || lookup.Categories[0] != "urbsex" || lookup.Categories[1] != "abandoned" {
		t.Fatalf("expected categories fallback from labels, got %+v", lookup.Categories)
	}
}

func TestLoadConfigResolvesFontPathRelativeToConfig(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "config")
	fontsDir := filepath.Join(dir, "fonts")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.MkdirAll(fontsDir, 0o755); err != nil {
		t.Fatalf("failed to create fonts dir: %v", err)
	}

	templatePath := filepath.Join(dir, "template.png")
	if err := os.WriteFile(templatePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	fontPath := filepath.Join(fontsDir, "Drips.ttf")
	if err := os.WriteFile(fontPath, []byte("font"), 0o644); err != nil {
		t.Fatalf("failed to write font: %v", err)
	}

	configPath := filepath.Join(configDir, "config.toml")
	configBody := strings.Join([]string{
		"template_image = \"" + templatePath + "\"",
		"output_dir = \"" + dir + "\"",
		"log_file = \"" + filepath.Join(dir, "run.log") + "\"",
		"",
		"[font]",
		"file = \"../fonts/Drips.ttf\"",
		"",
		"[plex]",
		"base_url = \"http://127.0.0.1:32400\"",
		"token = \"token\"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(configBody), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	if cfg.Font.File != fontPath {
		t.Fatalf("expected resolved font path %q got %q", fontPath, cfg.Font.File)
	}
}

func TestNormalizeFindList(t *testing.T) {
	finds, err := normalizeFindList([]string{" Chem ", "", "PnP", "chem"})
	if err != nil {
		t.Fatalf("normalizeFindList failed: %v", err)
	}
	if len(finds) != 2 || finds[0] != "Chem" || finds[1] != "PnP" {
		t.Fatalf("unexpected normalized find list: %+v", finds)
	}
}

func TestFirstMatchedFind(t *testing.T) {
	find, ok := firstMatchedFind("Party PnP mix", []string{"Chem", "PnP"})
	if !ok {
		t.Fatal("expected a match")
	}
	if find != "PnP" {
		t.Fatalf("unexpected first matched find: %q", find)
	}
}

func TestTitleMatchesFindIsCaseInsensitiveSubstring(t *testing.T) {
	if !titleMatchesFind(".abanDONED.", "abandoned") {
		t.Fatal("expected punctuation-surrounded title to match")
	}
	if !titleMatchesFind("_abandonedHouse_", "abandoned") {
		t.Fatal("expected embedded substring to match")
	}
	if titleMatchesFind("Other", "abandoned") {
		t.Fatal("expected non-matching title to fail")
	}
}

func TestMergeLabelsAddsOnlyMissingCaseInsensitive(t *testing.T) {
	existing := []plexLabel{{Tag: "urbsex"}, {Tag: "Archive"}}
	merged, changed := mergeLabels(existing, []string{"URBSEX", "abandoned"})
	if !changed {
		t.Fatal("expected label merge to report changes")
	}
	if len(merged) != 3 {
		t.Fatalf("expected 3 labels got %d (%+v)", len(merged), merged)
	}
	if merged[2] != "abandoned" {
		t.Fatalf("expected added label at end, got %+v", merged)
	}
}

func TestBuildUpdateLibraryItemLabelsURL(t *testing.T) {
	u, err := buildUpdateLibraryItemLabelsURL("http://localhost:32400", "secret-token", "1234", []string{"urbsex", "abandoned"})
	if err != nil {
		t.Fatalf("buildUpdateLibraryItemLabelsURL failed: %v", err)
	}
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	q := parsed.Query()
	if q.Get("X-Plex-Token") != "secret-token" {
		t.Fatalf("expected token query param, got %q", q.Get("X-Plex-Token"))
	}
	if q.Get("label[0].tag.tag") != "urbsex" {
		t.Fatalf("expected first label param, got %q", q.Get("label[0].tag.tag"))
	}
	if q.Get("label[1].tag.tag") != "abandoned" {
		t.Fatalf("expected second label param, got %q", q.Get("label[1].tag.tag"))
	}
}

func TestBuildUpdateLibraryItemCategoriesURL(t *testing.T) {
	u, err := buildUpdateLibraryItemCategoriesURL("http://localhost:32400", "secret-token", "1234", []string{"urbsex", "abandoned"})
	if err != nil {
		t.Fatalf("buildUpdateLibraryItemCategoriesURL failed: %v", err)
	}
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	q := parsed.Query()
	if q.Get("X-Plex-Token") != "secret-token" {
		t.Fatalf("expected token query param, got %q", q.Get("X-Plex-Token"))
	}
	if q.Get("genre[0].tag.tag") != "urbsex" {
		t.Fatalf("expected first category param, got %q", q.Get("genre[0].tag.tag"))
	}
	if q.Get("genre[1].tag.tag") != "abandoned" {
		t.Fatalf("expected second category param, got %q", q.Get("genre[1].tag.tag"))
	}
}

func TestSectionAllResponseParsesVideoItems(t *testing.T) {
	xmlBody := `<MediaContainer size="1"><Video ratingKey="365142" title=""><Media><Part file="M:\\NEW\\WISH\\abandonedHouse.mp4"/></Media></Video></MediaContainer>`
	var out plexSectionAllResponse
	if err := xml.Unmarshal([]byte(xmlBody), &out); err != nil {
		t.Fatalf("xml unmarshal failed: %v", err)
	}
	if len(out.Videos) != 1 {
		t.Fatalf("expected one video item, got %d", len(out.Videos))
	}
	if out.Videos[0].RatingKey != "365142" {
		t.Fatalf("unexpected rating key: %q", out.Videos[0].RatingKey)
	}
}

func TestSectionAllResponseParsesGenres(t *testing.T) {
	xmlBody := `<MediaContainer size="1"><Video ratingKey="365142" title="Example"><Genre tag="urbsex"/></Video></MediaContainer>`
	var out plexSectionAllResponse
	if err := xml.Unmarshal([]byte(xmlBody), &out); err != nil {
		t.Fatalf("xml unmarshal failed: %v", err)
	}
	if len(out.Videos) != 1 {
		t.Fatalf("expected one video item, got %d", len(out.Videos))
	}
	if len(out.Videos[0].Genres) != 1 || out.Videos[0].Genres[0].Tag != "urbsex" {
		t.Fatalf("expected one genre tag, got %+v", out.Videos[0].Genres)
	}
}

func TestLibraryItemMatchTextFallsBackToPartFile(t *testing.T) {
	item := plexLibraryItem{
		Title: "",
		Media: []plexMedia{{Parts: []plexPart{{File: `M:\NEW\WISH\_abandonedHouse_.mp4`}}}},
	}
	if !titleMatchesFind(libraryItemMatchText(item), "abandoned") {
		t.Fatal("expected file path fallback to match abandoned")
	}
}

func TestLibraryItemFileStem(t *testing.T) {
	item := plexLibraryItem{
		Media: []plexMedia{{Parts: []plexPart{{File: `V:\FILTH\TORRENT\My.Video.Name.mkv`}}}},
	}
	if got := libraryItemFileStem(item); got != "My.Video.Name" {
		t.Fatalf("unexpected file stem: %q", got)
	}
}

func TestPathCleanTitleFromFilePath(t *testing.T) {
	got := pathCleanTitleFromFilePath(`V:\FILTH\TORRENT\chalate2000\0gnfqk81vt7lmlg8b4ykb_source.mp4`, nil)
	want := "0gnfqk81vt7lmlg8b4ykb source - chalate2000 - TORRENT - FILTH"
	if got != want {
		t.Fatalf("unexpected path-clean title: got %q want %q", got, want)
	}
}

func TestSeedCleanTitles(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		sortTitle     string
		fileStem      string
		wantTitle     string
		wantSortTitle string
	}{
		{
			name:          "both populated — no change",
			title:         "My Film",
			sortTitle:     "Film My",
			fileStem:      "some_file",
			wantTitle:     "My Film",
			wantSortTitle: "Film My",
		},
		{
			name:          "blank title — filled from filename",
			title:         "",
			sortTitle:     "",
			fileStem:      "My.Film.2024",
			wantTitle:     "My.Film.2024",
			wantSortTitle: "My.Film.2024",
		},
		{
			name:          "blank title and blank sort title — sort falls back to seeded title",
			title:         "",
			sortTitle:     "",
			fileStem:      "Great Movie",
			wantTitle:     "Great Movie",
			wantSortTitle: "Great Movie",
		},
		{
			name:          "title present, sort title blank — sort filled from title",
			title:         "My Film",
			sortTitle:     "",
			fileStem:      "some_file",
			wantTitle:     "My Film",
			wantSortTitle: "My Film",
		},
		{
			name:          "title present, sort title blank, no filename — sort filled from title",
			title:         "My Film",
			sortTitle:     "",
			fileStem:      "",
			wantTitle:     "My Film",
			wantSortTitle: "My Film",
		},
		{
			name:          "all blank, no filename — nothing seeded",
			title:         "",
			sortTitle:     "",
			fileStem:      "",
			wantTitle:     "",
			wantSortTitle: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitle, gotSort := seedCleanTitles(tt.title, tt.sortTitle, tt.fileStem)
			if gotTitle != tt.wantTitle {
				t.Errorf("title: got %q, want %q", gotTitle, tt.wantTitle)
			}
			if gotSort != tt.wantSortTitle {
				t.Errorf("sortTitle: got %q, want %q", gotSort, tt.wantSortTitle)
			}
		})
	}
}

func TestDefaultSelectionInputFromKeys(t *testing.T) {
	sections := []plexSection{{Key: "10"}, {Key: "20"}, {Key: "30"}}
	if got := defaultSelectionInputFromKeys(sections, []string{"20", "10"}); got != "2,1" {
		t.Fatalf("expected 2,1 got %q", got)
	}
	if got := defaultSelectionInputFromKeys(sections, []string{"10", "20", "30"}); got != "all" {
		t.Fatalf("expected all got %q", got)
	}
	if got := defaultSelectionInputFromKeys(sections, []string{"99"}); got != "" {
		t.Fatalf("expected empty default for unknown key, got %q", got)
	}
}

func TestCleanTitleForSearch(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  string
	}{
		{name: "blank to unknown", in: "", out: "Unknown"},
		{name: "only special chars to unknown", in: "---", out: "Unknown"},
		{name: "only quotes to unknown", in: `"'"`, out: "Unknown"},
		{name: "special chars to spaces", in: "hello:world/again", out: "Hello world again"},
		{name: "ampersand to and", in: "rock & roll", out: "Rock and roll"},
		{name: "hash number to No.", in: "chapter #12", out: "Chapter No. 12"},
		{name: "at allowed", in: "name@home", out: "Name@home"},
		{name: "compress spaces", in: "one   two,,,   three", out: "One two three"},
		// Quote stripping: all quote types removed without adding a space.
		{name: "ascii double quotes stripped", in: `"Hello World"`, out: "Hello World"},
		{name: "ascii single quote stripped", in: "it's here", out: "Its here"},
		{name: "backtick stripped", in: "`code`", out: "Code"},
		{name: "curly double quotes stripped", in: "\u201cHello\u201d", out: "Hello"},
		{name: "curly single quotes stripped", in: "\u2018title\u2019", out: "Title"},
		{name: "angle quotes stripped", in: "\u00abfilm\u00bb", out: "Film"},
		// Video extension stripping.
		{name: "mp4 extension stripped", in: "myvideo.mp4", out: "Myvideo"},
		{name: "mov extension stripped", in: "clip.mov", out: "Clip"},
		{name: "mkv extension stripped", in: "show.mkv", out: "Show"},
		{name: "avi extension stripped", in: "old_file.avi", out: "Old file"},
		{name: "extension case insensitive", in: "video.MP4", out: "Video"},
		{name: "extension mid title stripped", in: "cool.mp4 part2", out: "Cool part2"},
		// Emoji stripping.
		{name: "emoji stripped", in: "hello \U0001F600 world", out: "Hello world"},
		{name: "dingbat stripped", in: "star \u2728 title", out: "Star title"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanTitleForSearch(tt.in, nil); got != tt.out {
				t.Fatalf("expected %q got %q", tt.out, got)
			}
		})
	}
}

func TestCleanTitleForSearchSecondReplacementPass(t *testing.T) {
	// Verify the second replacement pass catches patterns exposed after the char-clean pass.
	// "&" is replaced with " and " in the first pass; the surrounding text then forms "rock and roll".
	// A custom replacement for "rock and roll" (whole phrase after cleaning) should fire in pass 2.
	replacements := map[string]string{
		"rock and roll": "RnR",
	}
	got := cleanTitleForSearch("rock & roll", replacements)
	if got != "RnR" {
		t.Fatalf("expected second replacement pass to catch pattern, got %q", got)
	}
}

func TestCleanTitleForSearchWithCustomReplacements(t *testing.T) {
	replacements := map[string]string{
		"£":    " gbp ",
		"cum#": " climax number ",
	}
	got := cleanTitleForSearch("best £ clip cum#2", replacements)
	if got != "Best gbp clip climax number 2" {
		t.Fatalf("unexpected cleaned value: %q", got)
	}
}

func TestStampUnknown(t *testing.T) {
	seq := 0
	date := "20260605"
	stamp := func(s string) string {
		if s != "Unknown" {
			return s
		}
		seq++
		return fmt.Sprintf("Unknown-%s-%04d", date, seq)
	}
	if got := stamp("Hello"); got != "Hello" {
		t.Fatalf("non-unknown should pass through, got %q", got)
	}
	if got := stamp("Unknown"); got != "Unknown-20260605-0001" {
		t.Fatalf("first unknown, got %q", got)
	}
	if got := stamp("Unknown"); got != "Unknown-20260605-0002" {
		t.Fatalf("second unknown, got %q", got)
	}
}

func TestUniqueCleanReportPath(t *testing.T) {
	now := time.Date(2026, time.June, 5, 9, 30, 0, 0, time.UTC)
	got := uniqueCleanReportPath("output", now)
	expected := filepath.Join("output", "clean", "clean-20260605-093000.csv")
	if got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
}

func TestWriteCleanReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clean", "clean-test.csv")
	rows := []cleanReportRow{
		{RatingKey: "1", TitleBefore: "old title", TitleAfter: "Old Title", SortTitleBefore: "", SortTitleAfter: ""},
		{RatingKey: "2", TitleBefore: `say "hello"`, TitleAfter: "Say hello", SortTitleBefore: "", SortTitleAfter: ""},
	}
	if err := writeCleanReport(path, rows); err != nil {
		t.Fatalf("writeCleanReport error: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read report: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 rows), got %d: %q", len(lines), string(b))
	}
	if lines[0] != `"RatingKey"|"TitleBefore"|"TitleAfter"|"SortTitleBefore"|"SortTitleAfter"` {
		t.Fatalf("unexpected header: %q", lines[0])
	}
	if lines[1] != `"1"|"old title"|"Old Title"|""|""` {
		t.Fatalf("unexpected row 1: %q", lines[1])
	}
	// Embedded double-quote in TitleBefore should be escaped as ""
	if lines[2] != `"2"|"say ""hello"""|"Say hello"|""|""` {
		t.Fatalf("unexpected row 2: %q", lines[2])
	}
}

func TestMergeTOMLDocumentsAddsAndOverwrites(t *testing.T) {
	current := []byte(strings.Join([]string{
		"template_image = \"a.png\"",
		"",
		"[plex]",
		"workers = 4",
		"retries = 3",
		"",
		"[clean]",
		"translate_endpoint = \"https://old.example\"",
	}, "\n"))
	backup := []byte(strings.Join([]string{
		"template_image = \"b.png\"",
		"",
		"[plex]",
		"workers = 10",
		"retry_base_ms = 750",
		"",
		"[clean]",
		"translate_endpoint = \"https://new.example\"",
	}, "\n"))

	merged, err := mergeTOMLDocuments(current, backup)
	if err != nil {
		t.Fatalf("mergeTOMLDocuments failed: %v", err)
	}

	var out map[string]any
	if err := toml.Unmarshal(merged, &out); err != nil {
		t.Fatalf("failed to parse merged TOML: %v", err)
	}
	if out["template_image"] != "b.png" {
		t.Fatalf("expected template_image overwrite from backup, got %#v", out["template_image"])
	}
	plex, ok := out["plex"].(map[string]any)
	if !ok {
		t.Fatalf("expected [plex] table in merged output, got %#v", out["plex"])
	}
	if plex["workers"] != int64(10) {
		t.Fatalf("expected workers overwritten to 10, got %#v", plex["workers"])
	}
	if plex["retries"] != int64(3) {
		t.Fatalf("expected retries preserved at 3, got %#v", plex["retries"])
	}
	if plex["retry_base_ms"] != int64(750) {
		t.Fatalf("expected new key retry_base_ms added, got %#v", plex["retry_base_ms"])
	}
}

func TestNormalizeArchivePathRejectsTraversal(t *testing.T) {
	if _, ok := normalizeArchivePath("../config/config.toml"); ok {
		t.Fatal("expected traversal path to be rejected")
	}
	if got, ok := normalizeArchivePath("./config/config.toml"); !ok || got != "config/config.toml" {
		t.Fatalf("expected normalized safe path, got=%q ok=%t", got, ok)
	}
}

func TestListBackupArchivesSortsNewestFirst(t *testing.T) {
	dir := t.TempDir()
	older := filepath.Join(dir, "frantic-postr-backup-20260101-010101.zip")
	newer := filepath.Join(dir, "frantic-postr-backup-20260101-020202.zip")
	if err := os.WriteFile(older, []byte("older"), 0o644); err != nil {
		t.Fatalf("failed to write older archive: %v", err)
	}
	if err := os.WriteFile(newer, []byte("newer"), 0o644); err != nil {
		t.Fatalf("failed to write newer archive: %v", err)
	}

	now := time.Now()
	if err := os.Chtimes(older, now.Add(-2*time.Minute), now.Add(-2*time.Minute)); err != nil {
		t.Fatalf("failed to set older mtime: %v", err)
	}
	if err := os.Chtimes(newer, now, now); err != nil {
		t.Fatalf("failed to set newer mtime: %v", err)
	}

	backups, err := listBackupArchives(dir)
	if err != nil {
		t.Fatalf("listBackupArchives failed: %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("expected 2 backups, got %d", len(backups))
	}
	if backups[0].Name != filepath.Base(newer) {
		t.Fatalf("expected newest backup first, got %q", backups[0].Name)
	}
}

func TestFormatBackupDateTime(t *testing.T) {
	value := time.Date(2026, time.July, 5, 14, 22, 39, 0, time.UTC)
	got := formatBackupDateTime(value)
	expected := "05 Jul 2026 at 14:22:39"
	if got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
}

func TestIsPromptExitCommand(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{input: "q", want: true},
		{input: "Q", want: true},
		{input: "quit", want: true},
		{input: "EX", want: true},
		{input: "exit", want: true},
		{input: "Bye", want: true},
		{input: "  quit  ", want: true},
		{input: "", want: false},
		{input: "y", want: false},
		{input: "next", want: false},
	}

	for _, tc := range cases {
		got := isPromptExitCommand(tc.input)
		if got != tc.want {
			t.Fatalf("input %q: expected %t got %t", tc.input, tc.want, got)
		}
	}
}

func TestParseBackupHostname(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		expect   string
	}{
		{name: "hostname present", fileName: "frantic-postr-backup-media-node-20260705-111500.zip", expect: "media-node"},
		{name: "legacy filename no hostname", fileName: "frantic-postr-backup-20260705-111500.zip", expect: ""},
		{name: "uppercase extension", fileName: "frantic-postr-backup-host-a-20260705-111500.ZIP", expect: "host-a"},
		{name: "invalid timestamp", fileName: "frantic-postr-backup-host-invalid.zip", expect: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseBackupHostname(tc.fileName)
			if got != tc.expect {
				t.Fatalf("expected %q got %q", tc.expect, got)
			}
		})
	}
}

func TestApplyRestoredFileMergesTOMLAndCapturesRollback(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "config", "config.toml")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}
	before := []byte("template_image = \"old.png\"\n[plex]\nworkers = 3\n")
	if err := os.WriteFile(dest, before, 0o644); err != nil {
		t.Fatalf("failed to write destination file: %v", err)
	}

	rollbackZipPath := filepath.Join(dir, "rollback.zip")
	f, err := os.Create(rollbackZipPath)
	if err != nil {
		t.Fatalf("failed to create rollback zip: %v", err)
	}
	zw := zip.NewWriter(f)

	manifest := rollbackManifest{}
	seen := map[string]struct{}{}
	backup := []byte("template_image = \"new.png\"\n[plex]\nworkers = 8\nretry_base_ms = 500\n")
	action, _, changed, _, err := applyRestoredFile(dest, "config/config.toml", backup, zw, &manifest, seen, false)
	if err != nil {
		t.Fatalf("applyRestoredFile failed: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close rollback zip writer: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close rollback zip file: %v", err)
	}

	if action != "merged" || !changed {
		t.Fatalf("expected merged changed result, got action=%q changed=%t", action, changed)
	}
	after, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read destination file after restore: %v", err)
	}
	if !strings.Contains(string(after), "template_image = 'new.png'") {
		t.Fatalf("expected merged file to include overwritten value, got %q", string(after))
	}
	if len(manifest.Entries) != 1 || !manifest.Entries[0].ExistedPrior {
		t.Fatalf("unexpected rollback manifest: %+v", manifest.Entries)
	}

	r, err := zip.OpenReader(rollbackZipPath)
	if err != nil {
		t.Fatalf("failed to open rollback zip: %v", err)
	}
	defer r.Close()
	if len(r.File) != 1 || r.File[0].Name != "config/config.toml" {
		t.Fatalf("unexpected rollback zip entries: %+v", r.File)
	}
	storedBefore, err := readZipFile(r.File[0])
	if err != nil {
		t.Fatalf("failed to read rollback entry: %v", err)
	}
	if !bytes.Equal(storedBefore, before) {
		t.Fatalf("rollback zip should store pre-restore bytes")
	}
}

func TestMergeTOMLDocumentsWithChangesUsesPropertyDiffFormat(t *testing.T) {
	current := []byte("[plex]\nworkers = 4\n")
	backup := []byte("[plex]\nworkers = 9\nretry_base_ms = 500\n")
	_, changes, err := mergeTOMLDocumentsWithChanges(current, backup)
	if err != nil {
		t.Fatalf("mergeTOMLDocumentsWithChanges failed: %v", err)
	}
	if len(changes) == 0 {
		t.Fatal("expected property-level changes")
	}
	joined := strings.Join(changes, " | ")
	if !strings.Contains(joined, "plex.workers : 4 -> 9") {
		t.Fatalf("expected workers change in property format, got %q", joined)
	}
	if !strings.Contains(joined, "plex.retry_base_ms : <missing> -> 500") {
		t.Fatalf("expected added property in diff, got %q", joined)
	}
}

func TestApplyRestoredFileDryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "config", "config.toml")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatalf("failed creating directory: %v", err)
	}
	before := []byte("[plex]\nworkers = 4\n")
	if err := os.WriteFile(dest, before, 0o644); err != nil {
		t.Fatalf("failed writing file: %v", err)
	}
	backup := []byte("[plex]\nworkers = 12\n")

	action, _, changed, changes, err := applyRestoredFile(dest, "config/config.toml", backup, nil, nil, nil, true)
	if err != nil {
		t.Fatalf("applyRestoredFile dry run failed: %v", err)
	}
	if !changed || action != "merged" {
		t.Fatalf("expected merged+changed dry run, got action=%q changed=%t", action, changed)
	}
	if len(changes) == 0 || !strings.Contains(changes[0], " : ") {
		t.Fatalf("expected property style changes, got %+v", changes)
	}
	after, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed reading file after dry run: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("dry run should not write file contents")
	}
}

func TestApplyBackupRetentionRemovesOldArchives(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "frantic-postr-backup-old.zip")
	newPath := filepath.Join(dir, "frantic-postr-backup-new.zip")
	if err := os.WriteFile(oldPath, []byte("old"), 0o644); err != nil {
		t.Fatalf("failed to create old backup: %v", err)
	}
	if err := os.WriteFile(newPath, []byte("new"), 0o644); err != nil {
		t.Fatalf("failed to create new backup: %v", err)
	}
	now := time.Now()
	if err := os.Chtimes(oldPath, now.AddDate(0, 0, -40), now.AddDate(0, 0, -40)); err != nil {
		t.Fatalf("failed to age old backup: %v", err)
	}
	if err := os.Chtimes(newPath, now, now); err != nil {
		t.Fatalf("failed to set new backup mtime: %v", err)
	}

	cfg := Config{}
	cfg.Backup.RetentionDays = 30
	if err := applyBackupRetention(cfg, dir, newTestLogger(io.Discard, io.Discard)); err != nil {
		t.Fatalf("applyBackupRetention failed: %v", err)
	}
	if _, err := os.Stat(oldPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected old backup to be removed, err=%v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("expected new backup to remain, err=%v", err)
	}
}

func TestUniqueLabelReportPath(t *testing.T) {
	now := time.Date(2026, time.June, 6, 10, 0, 0, 0, time.UTC)
	got := uniqueLabelReportPath("output", now)
	expected := filepath.Join("output", "labels", "labels-20260606-100000.csv")
	if got != expected {
		t.Fatalf("expected %q got %q", expected, got)
	}
}

func TestWriteLabelReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "labels", "labels-test.csv")
	rows := []labelReportRow{
		{
			RatingKey:        "1",
			Title:            "My Film",
			LabelsBefore:     "Action",
			LabelsAfter:      "Action, Drama",
			CategoriesBefore: "",
			CategoriesAfter:  "Drama",
		},
		{
			RatingKey:        "2",
			Title:            `Film "Two"`,
			LabelsBefore:     "",
			LabelsAfter:      "Horror",
			CategoriesBefore: "",
			CategoriesAfter:  "",
		},
	}
	if err := writeLabelReport(path, rows); err != nil {
		t.Fatalf("writeLabelReport error: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read report: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 rows), got %d:\n%s", len(lines), string(b))
	}
	wantHeader := `"RatingKey"|"Title"|"LabelsBefore"|"LabelsAfter"|"CategoriesBefore"|"CategoriesAfter"`
	if lines[0] != wantHeader {
		t.Fatalf("unexpected header:\n got  %q\n want %q", lines[0], wantHeader)
	}
	wantRow1 := `"1"|"My Film"|"Action"|"Action, Drama"|""|"Drama"`
	if lines[1] != wantRow1 {
		t.Fatalf("unexpected row 1:\n got  %q\n want %q", lines[1], wantRow1)
	}
	// Embedded double-quote must be escaped as ""
	wantRow2 := `"2"|"Film ""Two"""|""|"Horror"|""|""`
	if lines[2] != wantRow2 {
		t.Fatalf("unexpected row 2:\n got  %q\n want %q", lines[2], wantRow2)
	}
}

func TestJoinPlexLabels(t *testing.T) {
	labels := []plexLabel{{Tag: "Action"}, {Tag: " Drama "}, {Tag: ""}, {Tag: "Horror"}}
	got := joinPlexLabels(labels)
	want := "Action, Drama, Horror"
	if got != want {
		t.Fatalf("expected %q got %q", want, got)
	}
	if joinPlexLabels(nil) != "" {
		t.Fatalf("expected empty string for nil labels")
	}
}

func TestBuildUpdateLibraryItemTitleURL(t *testing.T) {
	u, err := buildUpdateLibraryItemTitleURL("http://localhost:32400", "secret-token", "1234", "Clean Title")
	if err != nil {
		t.Fatalf("buildUpdateLibraryItemTitleURL failed: %v", err)
	}
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	q := parsed.Query()
	if q.Get("X-Plex-Token") != "secret-token" {
		t.Fatalf("expected token query param, got %q", q.Get("X-Plex-Token"))
	}
	if q.Get("title.value") != "Clean Title" {
		t.Fatalf("expected title.value query param, got %q", q.Get("title.value"))
	}
}

func TestBuildUpdateLibraryItemTitleAndSortURL(t *testing.T) {
	u, err := buildUpdateLibraryItemTitleAndSortURL("http://localhost:32400", "secret-token", "1234", "Clean Title", true, "Clean Title")
	if err != nil {
		t.Fatalf("buildUpdateLibraryItemTitleAndSortURL failed: %v", err)
	}
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	q := parsed.Query()
	if q.Get("title.value") != "Clean Title" {
		t.Fatalf("expected title.value query param, got %q", q.Get("title.value"))
	}
	if q.Get("titleSort.value") != "Clean Title" {
		t.Fatalf("expected titleSort.value query param, got %q", q.Get("titleSort.value"))
	}
}

func TestTranslateTextToEnglish(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST got %s", r.Method)
		}
		_, _ = w.Write([]byte(`{"translatedText":"Fat hairy guy masturbates and satisfies his lust"}`))
	}))
	defer server.Close()

	var cfg Config
	cfg.Plex.Retries = 1
	cfg.Clean.TranslateEndpoint = server.URL

	translated, err := translateTextToEnglish(server.Client(), cfg, "Толстыи волосатыи парень", newTestLogger(io.Discard, io.Discard))
	if err != nil {
		t.Fatalf("translateTextToEnglish failed: %v", err)
	}
	if translated == "" {
		t.Fatal("expected translated text")
	}
}

func TestMaybeTranslateToEnglishSkipsEnglish(t *testing.T) {
	var cfg Config
	cfg.Clean.TranslateEndpoint = "http://localhost:65535"
	translated, lang, ok := maybeTranslateToEnglish(&http.Client{Timeout: 100 * time.Millisecond}, cfg, "Simple english title", newTestLogger(io.Discard, io.Discard))
	if ok {
		t.Fatal("expected no translation for english text")
	}
	if translated != "Simple english title" {
		t.Fatalf("unexpected translated value: %q", translated)
	}
	if lang != "" {
		t.Fatalf("expected empty language code when translation gate is not met, got %q", lang)
	}
}

func TestHasClearNonEnglishMarkers(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "plain ascii english", in: "Simple english title", want: false},
		{name: "ascii with punctuation", in: "title #12 @home", want: false},
		{name: "french accent", in: "Caf\u00e9 du Monde", want: true},
		{name: "russian", in: "\u0422\u043e\u043b\u0441\u0442\u044b\u0438 \u0432\u043e\u043b\u043e\u0441\u0430\u0442\u044b\u0438 \u043f\u0430\u0440\u0435\u043d\u044c", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasClearNonEnglishMarkers(tt.in); got != tt.want {
				t.Fatalf("unexpected marker check result for %q: got %t want %t", tt.in, got, tt.want)
			}
		})
	}
}

func TestShouldSkipPlexWrite(t *testing.T) {
	trailModeEnabled = false
	if shouldSkipPlexWrite(newTestLogger(io.Discard, io.Discard), "op", "http://localhost") {
		t.Fatal("expected no skip when trail mode disabled")
	}
	trailModeEnabled = true
	if !shouldSkipPlexWrite(newTestLogger(io.Discard, io.Discard), "op", "http://localhost") {
		t.Fatal("expected skip when trail mode enabled")
	}
	trailModeEnabled = false
}
