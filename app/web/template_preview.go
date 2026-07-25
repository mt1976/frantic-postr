package web

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode"

	_ "image/gif"
	_ "image/jpeg"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	_ "golang.org/x/image/webp"
)

type templatePreviewResult struct {
	TemplateKind string
	TemplatePath string
	SampleText   string
	ImageDataURL string
	Width        int
	Height       int
}

func generateTemplatePreview(cfg Config, requestedKind, sampleText string) (templatePreviewResult, error) {
	kind := strings.ToLower(strings.TrimSpace(requestedKind))
	if kind == "" {
		kind = "default"
	}
	templatePath, kindLabel := pickPreviewTemplatePath(cfg, kind)
	if strings.TrimSpace(templatePath) == "" {
		return templatePreviewResult{}, fmt.Errorf("template image is not configured for type %q", kind)
	}

	templateFile, err := os.Open(templatePath)
	if err != nil {
		return templatePreviewResult{}, err
	}
	defer templateFile.Close()

	src, _, err := image.Decode(templateFile)
	if err != nil {
		return templatePreviewResult{}, err
	}

	rgba := image.NewRGBA(src.Bounds())
	draw.Draw(rgba, rgba.Bounds(), src, src.Bounds().Min, draw.Src)

	face, closeFace, err := loadPreviewFontFace(cfg)
	if err != nil {
		return templatePreviewResult{}, err
	}
	defer closeFace()

	if cfg.Font.Color == "" {
		cfg.Font.Color = "#FFFFFF"
	}
	if cfg.Font.ShadowColor == "" {
		cfg.Font.ShadowColor = "#000000"
	}
	if cfg.Font.GlowColor == "" {
		cfg.Font.GlowColor = "#000000"
	}
	if cfg.Font.GlowAlpha <= 0 || cfg.Font.GlowAlpha > 1 {
		cfg.Font.GlowAlpha = 0.35
	}
	if cfg.Font.Size <= 0 {
		cfg.Font.Size = 64
	}

	mainColor, err := parsePreviewHexColor(cfg.Font.Color, 255)
	if err != nil {
		return templatePreviewResult{}, err
	}
	shadowColor, err := parsePreviewHexColor(cfg.Font.ShadowColor, 210)
	if err != nil {
		return templatePreviewResult{}, err
	}
	glowAlpha := uint8(math.Round(cfg.Font.GlowAlpha * 255))
	glowColor, err := parsePreviewHexColor(cfg.Font.GlowColor, glowAlpha)
	if err != nil {
		return templatePreviewResult{}, err
	}

	if strings.TrimSpace(sampleText) == "" {
		sampleText = defaultPreviewText(kindLabel)
	}
	previewText := posterPreviewDisplayText(sampleText)
	w := rgba.Bounds().Dx()
	h := rgba.Bounds().Dy()
	paddingX, paddingY := posterPreviewPadding(cfg, w, h)
	maxWidth := w - paddingX*2
	if maxWidth < 20 {
		paddingX = 0
		maxWidth = w
	}
	lines := wrapPreviewText(face, previewText, maxWidth)
	positions := centeredPreviewTextDots(face, lines, w, h, cfg.Font.YOffset, paddingX, paddingY)

	if cfg.Font.GlowRadius > 0 {
		for _, line := range positions {
			for dx := -cfg.Font.GlowRadius; dx <= cfg.Font.GlowRadius; dx++ {
				for dy := -cfg.Font.GlowRadius; dy <= cfg.Font.GlowRadius; dy++ {
					if dx == 0 && dy == 0 {
						continue
					}
					if dx*dx+dy*dy > cfg.Font.GlowRadius*cfg.Font.GlowRadius {
						continue
					}
					drawPreviewText(rgba, face, line.Text, line.X+dx, line.Y+dy, glowColor)
				}
			}
		}
	}

	for _, line := range positions {
		if cfg.Font.ShadowOffsetX != 0 || cfg.Font.ShadowOffsetY != 0 {
			drawPreviewText(rgba, face, line.Text, line.X+cfg.Font.ShadowOffsetX, line.Y+cfg.Font.ShadowOffsetY, shadowColor)
		}
		drawPreviewText(rgba, face, line.Text, line.X, line.Y, mainColor)
	}

	var out bytes.Buffer
	if err := png.Encode(&out, rgba); err != nil {
		return templatePreviewResult{}, err
	}
	encoded := base64.StdEncoding.EncodeToString(out.Bytes())

	return templatePreviewResult{
		TemplateKind: kindLabel,
		TemplatePath: templatePath,
		SampleText:   sampleText,
		ImageDataURL: "data:image/png;base64," + encoded,
		Width:        w,
		Height:       h,
	}, nil
}

func pickPreviewTemplatePath(cfg Config, kind string) (string, string) {
	switch kind {
	case "type":
		if strings.TrimSpace(cfg.TypeTemplateImage) == "" {
			return "", "type"
		}
		return cfg.TypeTemplateImage, "type"
	case "studio":
		if strings.TrimSpace(cfg.StudioTemplateImage) == "" {
			return "", "studio"
		}
		return cfg.StudioTemplateImage, "studio"
	case "admin":
		if strings.TrimSpace(cfg.AdminTemplateImage) == "" {
			return "", "admin"
		}
		return cfg.AdminTemplateImage, "admin"
	default:
		if strings.TrimSpace(cfg.TemplateImage) == "" {
			return "", "default"
		}
		return cfg.TemplateImage, "default"
	}
}

func defaultPreviewText(kind string) string {
	switch kind {
	case "type":
		return "Sample Type Collection 2026"
	case "studio":
		return "Sample Studio Collection 2026"
	case "admin":
		return "Sample Admin Collection 2026"
	default:
		return "Sample Collection 2026"
	}
}

func loadPreviewFontFace(cfg Config) (font.Face, func(), error) {
	if strings.TrimSpace(cfg.Font.File) == "" {
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
	size := cfg.Font.Size
	if size <= 0 {
		size = 64
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, nil, err
	}
	return face, func() { _ = face.Close() }, nil
}

type previewTextDot struct {
	Text string
	X    int
	Y    int
}

func centeredPreviewTextDots(face font.Face, lines []string, width, height, yOffset, paddingX, paddingY int) []previewTextDot {
	if len(lines) == 0 {
		lines = []string{"UNTITLED"}
	}
	if paddingX < 0 {
		paddingX = 0
	}
	if paddingY < 0 {
		paddingY = 0
	}
	if width <= paddingX*2 {
		paddingX = 0
	}
	if height <= paddingY*2 {
		paddingY = 0
	}
	availableWidth := width - paddingX*2
	if availableWidth <= 0 {
		availableWidth = width
	}
	availableHeight := height - paddingY*2
	if availableHeight <= 0 {
		availableHeight = height
	}

	lineHeight := face.Metrics().Height.Ceil()
	if lineHeight <= 0 {
		lineHeight = 1
	}
	totalHeight := lineHeight * len(lines)
	top := paddingY + (availableHeight-totalHeight)/2 + yOffset
	if top < paddingY {
		top = paddingY
	}

	dots := make([]previewTextDot, 0, len(lines))
	for idx, line := range lines {
		bounds, _ := font.BoundString(face, line)
		textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
		x := paddingX + (availableWidth-textWidth)/2 - bounds.Min.X.Ceil()
		if x < paddingX {
			x = paddingX
		}
		yTop := top + idx*lineHeight
		y := yTop - bounds.Min.Y.Ceil()
		dots = append(dots, previewTextDot{Text: line, X: x, Y: y})
	}
	return dots
}

func posterPreviewPadding(cfg Config, width, height int) (int, int) {
	fontSizePad := int(math.Ceil(cfg.Font.Size * 0.75))
	if fontSizePad < 24 {
		fontSizePad = 24
	}
	effectPad := cfg.Font.GlowRadius + previewMaxInt(previewAbsInt(cfg.Font.ShadowOffsetX), previewAbsInt(cfg.Font.ShadowOffsetY)) + 8
	if effectPad < 24 {
		effectPad = 24
	}
	horizontal := previewMaxInt(fontSizePad, effectPad)
	vertical := previewMaxInt(int(math.Ceil(cfg.Font.Size*0.9)), effectPad)
	if width > 0 {
		horizontal = previewMinInt(horizontal, previewMaxInt(0, width/4))
	}
	if height > 0 {
		vertical = previewMinInt(vertical, previewMaxInt(0, height/4))
	}
	return horizontal, vertical
}

func posterPreviewDisplayText(text string) string {
	return strings.ToUpper(forcePreviewLineBreakAfterNumber(text))
}

func forcePreviewLineBreakAfterNumber(text string) string {
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

func wrapPreviewText(face font.Face, text string, maxWidth int) []string {
	raw := strings.TrimSpace(text)
	if raw == "" {
		return []string{"UNTITLED"}
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
		lines = append(lines, wrapPreviewSingleLine(face, clean, maxWidth)...)
	}
	if len(lines) == 0 {
		return []string{"UNTITLED"}
	}
	return lines
}

func wrapPreviewSingleLine(face font.Face, clean string, maxWidth int) []string {
	if clean == "" {
		return nil
	}
	words := strings.Fields(clean)
	lines := make([]string, 0, len(words))
	current := words[0]
	for _, word := range words[1:] {
		candidate := current + " " + word
		if previewTextPixelWidth(face, candidate) <= maxWidth {
			current = candidate
			continue
		}
		lines = append(lines, current)
		if previewTextPixelWidth(face, word) <= maxWidth {
			current = word
			continue
		}
		parts := splitPreviewWord(face, word, maxWidth)
		if len(parts) == 0 {
			current = word
			continue
		}
		lines = append(lines, parts[:len(parts)-1]...)
		current = parts[len(parts)-1]
	}
	if previewTextPixelWidth(face, current) <= maxWidth {
		return append(lines, current)
	}
	parts := splitPreviewWord(face, current, maxWidth)
	if len(parts) == 0 {
		return append(lines, current)
	}
	return append(lines, parts...)
}

func splitPreviewWord(face font.Face, word string, maxWidth int) []string {
	if word == "" {
		return nil
	}
	runes := []rune(word)
	parts := make([]string, 0, len(runes))
	start := 0
	for start < len(runes) {
		end := start + 1
		for end <= len(runes) && previewTextPixelWidth(face, string(runes[start:end])) <= maxWidth {
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

func previewTextPixelWidth(face font.Face, text string) int {
	bounds, _ := font.BoundString(face, text)
	return (bounds.Max.X - bounds.Min.X).Ceil()
}

func drawPreviewText(dst draw.Image, face font.Face, text string, x, y int, c color.Color) {
	d := &font.Drawer{Dst: dst, Src: image.NewUniform(c), Face: face, Dot: fixed.P(x, y)}
	d.DrawString(text)
}

func parsePreviewHexColor(hex string, alpha uint8) (color.Color, error) {
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

func previewMaxInt(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	best := values[0]
	for _, value := range values[1:] {
		if value > best {
			best = value
		}
	}
	return best
}

func previewMinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func previewAbsInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
