package main

import (
	"bufio"
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
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type Config struct {
	Plex struct {
		BaseURL string `json:"base_url"`
		Token   string `json:"token"`
	} `json:"plex"`
	TemplateImage string `json:"template_image"`
	OutputDir     string `json:"output_dir"`
	LogFile       string `json:"log_file"`
	Font          struct {
		File          string  `json:"file"`
		Size          float64 `json:"size"`
		Color         string  `json:"color"`
		ShadowColor   string  `json:"shadow_color"`
		ShadowOffsetX int     `json:"shadow_offset_x"`
		ShadowOffsetY int     `json:"shadow_offset_y"`
		GlowColor     string  `json:"glow_color"`
		GlowRadius    int     `json:"glow_radius"`
		GlowAlpha     float64 `json:"glow_alpha"`
		YOffset       int     `json:"y_offset"`
	} `json:"font"`
}

type plexSectionsResponse struct {
	Directories []struct {
		Key   string `xml:"key,attr"`
		Title string `xml:"title,attr"`
		Type  string `xml:"type,attr"`
	} `xml:"Directory"`
}

type plexCollectionsResponse struct {
	Metadata []struct {
		Title string `xml:"title,attr"`
	} `xml:"Metadata"`
}

func main() {
	configPath := flag.String("config", "config.json", "Path to config file")
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

	client := &http.Client{Timeout: 30 * time.Second}
	sections, err := fetchSections(client, cfg, logger)
	if err != nil {
		logger.Fatalf("failed to fetch libraries: %v", err)
	}
	if len(sections) == 0 {
		logger.Fatal("no libraries returned by Plex")
	}

	section, err := selectSection(sections)
	if err != nil {
		logger.Fatalf("failed to select library: %v", err)
	}
	logger.Printf("selected library: %s (%s)", section.Title, section.Key)

	collections, err := fetchCollections(client, cfg, section.Key, logger)
	if err != nil {
		logger.Fatalf("failed to fetch collections for %s: %v", section.Title, err)
	}
	logger.Printf("collections fetched: %d", len(collections))

	if err := processCollections(cfg, section.Title, collections, logger); err != nil {
		logger.Fatalf("processing failed: %v", err)
	}

	logger.Println("shutdown: frantic-postr completed")
}

func loadConfig(path string) (Config, error) {
	var cfg Config
	bytes, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(bytes, &cfg); err != nil {
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
	if cfg.TemplateImage == "" || cfg.Plex.BaseURL == "" || cfg.Plex.Token == "" {
		return cfg, errors.New("config must set plex.base_url, plex.token and template_image")
	}
	return cfg, nil
}

func setupLogger(path string) (*log.Logger, func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}
	mw := io.MultiWriter(os.Stdout, f)
	logger := log.New(mw, "", log.LstdFlags|log.Lmicroseconds)
	return logger, func() { _ = f.Close() }, nil
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
}, error) {
	endpoint := strings.TrimRight(cfg.Plex.BaseURL, "/") + "/library/sections"
	logger.Printf("plex call: GET %s", endpoint)
	respBody, err := doPlexGET(client, endpoint, cfg.Plex.Token)
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

func fetchCollections(client *http.Client, cfg Config, sectionKey string, logger *log.Logger) ([]string, error) {
	endpoint := strings.TrimRight(cfg.Plex.BaseURL, "/") + "/library/sections/" + sectionKey + "/collections"
	logger.Printf("plex call: GET %s", endpoint)
	respBody, err := doPlexGET(client, endpoint, cfg.Plex.Token)
	if err != nil {
		return nil, err
	}
	var out plexCollectionsResponse
	if err := xml.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	collections := make([]string, 0, len(out.Metadata))
	for _, m := range out.Metadata {
		if strings.TrimSpace(m.Title) != "" {
			collections = append(collections, m.Title)
		}
	}
	return collections, nil
}

func doPlexGET(client *http.Client, endpoint, token string) ([]byte, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("X-Plex-Token", token)
	u.RawQuery = q.Encode()
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

func selectSection(sections []struct {
	Key   string `xml:"key,attr"`
	Title string `xml:"title,attr"`
	Type  string `xml:"type,attr"`
}) (struct {
	Key   string `xml:"key,attr"`
	Title string `xml:"title,attr"`
	Type  string `xml:"type,attr"`
}, error) {
	for idx, s := range sections {
		fmt.Printf("[%d] %s (%s)\n", idx+1, s.Title, s.Type)
	}
	fmt.Print("Select a library number: ")
	input, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return struct {
			Key   string `xml:"key,attr"`
			Title string `xml:"title,attr"`
			Type  string `xml:"type,attr"`
		}{}, err
	}
	choice, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil || choice < 1 || choice > len(sections) {
		return struct {
			Key   string `xml:"key,attr"`
			Title string `xml:"title,attr"`
			Type  string `xml:"type,attr"`
		}{}, fmt.Errorf("invalid selection")
	}
	return sections[choice-1], nil
}

func processCollections(cfg Config, libraryName string, collections []string, logger *log.Logger) error {
	outDir := filepath.Join(cfg.OutputDir, sanitizeFileName(libraryName))
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	logger.Printf("output dir ready: %s", outDir)
	for _, collectionName := range collections {
		outputPath := buildOutputPath(outDir, collectionName, cfg.TemplateImage)
		logger.Printf("creating poster: collection=%q output=%s", collectionName, outputPath)
		if err := renderCollectionPoster(cfg, collectionName, outputPath); err != nil {
			return fmt.Errorf("render %q: %w", collectionName, err)
		}
		logger.Printf("poster created: %s", outputPath)
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
	baseX, baseY := centeredTextDot(face, text, w, h, cfg.Font.YOffset)

	if cfg.Font.GlowRadius > 0 {
		for dx := -cfg.Font.GlowRadius; dx <= cfg.Font.GlowRadius; dx++ {
			for dy := -cfg.Font.GlowRadius; dy <= cfg.Font.GlowRadius; dy++ {
				if dx == 0 && dy == 0 {
					continue
				}
				if dx*dx+dy*dy > cfg.Font.GlowRadius*cfg.Font.GlowRadius {
					continue
				}
				drawText(rgba, face, text, baseX+dx, baseY+dy, glowColor)
			}
		}
	}

	if cfg.Font.ShadowOffsetX != 0 || cfg.Font.ShadowOffsetY != 0 {
		drawText(rgba, face, text, baseX+cfg.Font.ShadowOffsetX, baseY+cfg.Font.ShadowOffsetY, shadowColor)
	}
	drawText(rgba, face, text, baseX, baseY, mainColor)

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

func centeredTextDot(face font.Face, text string, width, height, yOffset int) (int, int) {
	bounds, _ := font.BoundString(face, text)
	textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
	textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()
	x := (width-textWidth)/2 - bounds.Min.X.Ceil()
	yTop := (height-textHeight)/2 + yOffset
	y := yTop - bounds.Min.Y.Ceil()
	return x, y
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
