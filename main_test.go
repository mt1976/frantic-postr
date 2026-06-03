package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
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
