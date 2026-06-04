package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/image/font/basicfont"
)

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
	logger := log.New(&buf, "", 0)

	if _, err := doPlexGET(server.Client(), server.URL+"/library/sections", "secret-token", logger); err != nil {
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

	collections, err := fetchCollections(server.Client(), cfg, "73", log.New(io.Discard, "", 0))
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

	if err := uploadCollectionPoster(server.Client(), cfg, collection, imagePath, log.New(io.Discard, "", 0)); err != nil {
		t.Fatalf("uploadCollectionPoster failed: %v", err)
	}
}
