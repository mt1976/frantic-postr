package main

import (
	"bufio"
	"bytes"
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
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/pelletier/go-toml/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type Config struct {
	Plex struct {
		BaseURL string `toml:"base_url"`
		Token   string `toml:"token"`
	} `toml:"plex"`
	TemplateImage string `toml:"template_image"`
	OutputDir     string `toml:"output_dir"`
	LogFile       string `toml:"log_file"`
	Font          struct {
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
	Directories []struct {
		Key   string `xml:"key,attr"`
		Title string `xml:"title,attr"`
		Type  string `xml:"type,attr"`
	} `xml:"Directory"`
}

type plexCollectionsResponse struct {
	Directories []plexCollection `xml:"Directory"`
}

type plexCollection struct {
	RatingKey string `xml:"ratingKey,attr"`
	Title     string `xml:"title,attr"`
	GUID      string `xml:"guid,attr"`
}

func main() {
	configPath := flag.String("config", "config.toml", "Path to config file")
	upload := flag.Bool("upload", false, "Upload generated posters to Plex collections")
	flag.Parse()

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
	logger.Printf("config: upload=%t", *upload)

	client := &http.Client{Timeout: 30 * time.Second}
	sections, err := fetchSections(client, cfg, logger)
	if err != nil {
		logger.Fatalf("failed to fetch libraries: %v", err)
	}
	if len(sections) == 0 {
		logger.Fatal("no libraries returned by Plex")
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

func setupLogger(path string) (*log.Logger, func(), error) {
	runLogPath := uniqueRunLogPath(path, time.Now())
	f, err := os.OpenFile(runLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}
	mw := io.MultiWriter(os.Stdout, f)
	logger := log.New(mw, "", log.LstdFlags|log.Lmicroseconds)
	logger.Printf("log file: %s", runLogPath)
	return logger, func() { _ = f.Close() }, nil
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

func logConfig(logger *log.Logger, cfg Config) {
	logger.Printf("config: plex.base_url=%s", cfg.Plex.BaseURL)
	logger.Printf("config: plex.token=*** (len=%d)", len(cfg.Plex.Token))
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

func fetchSections(client *http.Client, cfg Config, logger *log.Logger) ([]struct {
	Key   string `xml:"key,attr"`
	Title string `xml:"title,attr"`
	Type  string `xml:"type,attr"`
}, error,
) {
	endpoint := strings.TrimRight(cfg.Plex.BaseURL, "/") + "/library/sections"
	logger.Printf("plex call: GET %s", endpoint)
	respBody, err := doPlexGET(client, endpoint, cfg.Plex.Token, logger)
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

func fetchCollections(client *http.Client, cfg Config, sectionKey string, logger *log.Logger) ([]plexCollection, error) {
	endpoint := strings.TrimRight(cfg.Plex.BaseURL, "/") + "/library/sections/" + sectionKey + "/collections"
	logger.Printf("plex call: GET %s", endpoint)
	respBody, err := doPlexGET(client, endpoint, cfg.Plex.Token, logger)
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

func doPlexGET(client *http.Client, endpoint, token string, logger *log.Logger) ([]byte, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("X-Plex-Token", token)
	u.RawQuery = q.Encode()
	if logger != nil {
		logger.Printf("plex curl: %s", curlCommand("GET", u.String()))
	}
	resp, err := client.Get(u.String())
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

func selectSections(sections []struct {
	Key   string `xml:"key,attr"`
	Title string `xml:"title,attr"`
	Type  string `xml:"type,attr"`
}) ([]struct {
	Key   string `xml:"key,attr"`
	Title string `xml:"title,attr"`
	Type  string `xml:"type,attr"`
}, error,
) {
	for idx, s := range sections {
		fmt.Printf("[%d] %s (%s)\n", idx+1, s.Title, s.Type)
	}
	fmt.Print("Select library number(s) (comma-separated) or 'all': ")
	input, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	indices, err := parseSelectionInput(input, len(sections))
	if err != nil {
		return nil, err
	}

	selected := make([]struct {
		Key   string `xml:"key,attr"`
		Title string `xml:"title,attr"`
		Type  string `xml:"type,attr"`
	}, 0, len(indices))
	for _, idx := range indices {
		selected = append(selected, sections[idx])
	}
	return selected, nil
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

func processCollections(client *http.Client, cfg Config, libraryName string, collections []plexCollection, upload bool, logger *log.Logger) error {
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
		logger.Printf("poster created: %s", outputPath)
		if upload {
			if err := uploadCollectionPoster(client, cfg, collection, outputPath, logger); err != nil {
				return fmt.Errorf("upload poster for %q: %w", collection.Title, err)
			}
		}
	}
	return nil
}

func uploadCollectionPoster(client *http.Client, cfg Config, collection plexCollection, imagePath string, logger *log.Logger) error {
	if strings.TrimSpace(collection.RatingKey) == "" {
		return errors.New("missing collection ratingKey")
	}

	endpoint := strings.TrimRight(cfg.Plex.BaseURL, "/") + "/library/collections/" + collection.RatingKey + "/posters"
	if logger != nil {
		logger.Printf("plex call: POST %s", endpoint)
		logger.Printf("plex curl: %s", curlFileUploadCommand("POST", endpoint, cfg.Plex.Token, imagePath))
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

	req, err := http.NewRequest(http.MethodPost, u.String(), &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("plex upload failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	if logger != nil {
		logger.Printf("plex upload complete: collection=%q rating_key=%s", collection.Title, collection.RatingKey)
	}
	return nil
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
