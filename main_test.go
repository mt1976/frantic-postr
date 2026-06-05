package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
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
	"strings"
	"testing"
	"time"

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

func TestSanitizeFileName(t *testing.T) {
	got := sanitizeFileName(`  A/B:C*D?"E<F>G|  `)
	if got != "A_B-C_D'EFG_" {
		t.Fatalf("unexpected sanitized value: %q", got)
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

	if _, err := doPlexGET(server.Client(), server.URL+"/library/sections", "secret-token", 3, logger); err != nil {
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
		Media: []plexMedia{{Parts: []plexPart{{File: `/mnt/media/My.Video.Name.mkv`}}}},
	}
	if got := libraryItemFileStem(item); got != "My.Video.Name" {
		t.Fatalf("unexpected file stem: %q", got)
	}
}

func TestSeedCleanTitles(t *testing.T) {
	tests := []struct {
		name         string
		title        string
		sortTitle    string
		fileStem     string
		wantTitle    string
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
		{name: "special chars to spaces", in: "hello:world/again", out: "Hello world again"},
		{name: "ampersand to and", in: "rock & roll", out: "Rock and roll"},
		{name: "hash number to No.", in: "chapter #12", out: "Chapter No. 12"},
		{name: "at allowed", in: "name@home", out: "Name@home"},
		{name: "compress spaces", in: "one   two,,,   three", out: "One two three"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanTitleForSearch(tt.in, nil); got != tt.out {
				t.Fatalf("expected %q got %q", tt.out, got)
			}
		})
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
