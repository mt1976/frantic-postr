package reports

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mt1976/frantic-postr/app/core"
)

func SetupLogger(path string) (*core.AppLogger, func(), error) {
	runLogPath := UniqueRunLogPath(path, time.Now())
	if err := os.MkdirAll(filepath.Dir(runLogPath), 0o755); err != nil {
		return nil, nil, fmt.Errorf("create log dir: %w", err)
	}
	f, err := os.OpenFile(runLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}
	logger := &core.AppLogger{
		Console: log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds),
		File:    log.New(f, "", log.LstdFlags|log.Lmicroseconds),
	}
	logger.Infof("log file: %s", runLogPath)
	return logger, func() { _ = f.Close() }, nil
}

func UniqueCleanReportPath(outputDir string, now time.Time) string {
	timestamp := now.Format("20060102-150405")
	return filepath.Join(outputDir, "clean", fmt.Sprintf("clean-%s.csv", timestamp))
}

func UniqueLabelReportPath(outputDir string, now time.Time) string {
	timestamp := now.Format("20060102-150405")
	return filepath.Join(outputDir, "labels", fmt.Sprintf("labels-%s.csv", timestamp))
}

func WriteLabelReport(path string, rows []core.LabelReportRow) error {
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

func WriteCleanReport(path string, rows []core.CleanReportRow) error {
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

func WriteCSVReport(path string, header []string, rows [][]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create report dir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create report file: %w", err)
	}
	defer f.Close()
	writer := csv.NewWriter(f)
	if len(header) > 0 {
		if err := writer.Write(header); err != nil {
			return err
		}
	}
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func UniqueCollectionReportPath(outputDir, prefix string, now time.Time) string {
	timestamp := now.Format("20060102-150405")
	return filepath.Join(outputDir, "collections", fmt.Sprintf("%s-%s.csv", prefix, timestamp))
}

func UniqueCollectionReportPathForLibrary(outputDir, prefix, libraryName string, now time.Time) string {
	timestamp := now.Format("20060102-150405")
	namePrefix := PrefixWithLibraryToken(prefix, libraryName)
	return filepath.Join(outputDir, "collections", fmt.Sprintf("%s-%s.csv", namePrefix, timestamp))
}

func UniquePathCleanReportPath(outputDir string, now time.Time) string {
	timestamp := now.Format("20060102-150405")
	return filepath.Join(outputDir, "path-clean", fmt.Sprintf("path-clean-%s.csv", timestamp))
}

func UniquePathCleanReportPathForLibrary(outputDir, libraryName string, now time.Time) string {
	timestamp := now.Format("20060102-150405")
	namePrefix := PrefixWithLibraryToken("path-clean", libraryName)
	return filepath.Join(outputDir, "path-clean", fmt.Sprintf("%s-%s.csv", namePrefix, timestamp))
}

func UniquePosterReportPath(outputDir string, now time.Time) string {
	timestamp := now.Format("20060102-150405")
	return filepath.Join(outputDir, fmt.Sprintf("posters-%s.csv", timestamp))
}

func UniqueStatsReportPath(outputDir string, now time.Time) string {
	timestamp := now.Format("20060102-150405")
	return filepath.Join(outputDir, "stats", fmt.Sprintf("word-frequency-%s.csv", timestamp))
}

func UniqueStatsReportPathForLibrary(outputDir, libraryName string, now time.Time) string {
	timestamp := now.Format("20060102-150405")
	namePrefix := PrefixWithLibraryToken("word-frequency", libraryName)
	return filepath.Join(outputDir, "stats", fmt.Sprintf("%s-%s.csv", namePrefix, timestamp))
}

func UniqueCleanReportPathForLibrary(outputDir, libraryName string, now time.Time) string {
	timestamp := now.Format("20060102-150405")
	namePrefix := PrefixWithLibraryToken("clean", libraryName)
	return filepath.Join(outputDir, "clean", fmt.Sprintf("%s.csv", TimestampWithPrefix(namePrefix, timestamp)))
}

func UniqueLabelReportPathForLibrary(outputDir, libraryName string, now time.Time) string {
	timestamp := now.Format("20060102-150405")
	namePrefix := PrefixWithLibraryToken("labels", libraryName)
	return filepath.Join(outputDir, "labels", fmt.Sprintf("%s.csv", TimestampWithPrefix(namePrefix, timestamp)))
}

func PrefixWithLibraryToken(prefix, libraryName string) string {
	token := strings.TrimSpace(sanitizeFileName(libraryName))
	if token == "" {
		return prefix
	}
	lowerPrefix := strings.ToLower(prefix)
	lowerToken := strings.ToLower(token)
	if strings.Contains(lowerPrefix, lowerToken) {
		return prefix
	}
	return prefix + "-" + token
}

func TimestampWithPrefix(prefix, timestamp string) string {
	if strings.TrimSpace(prefix) == "" {
		return timestamp
	}
	return prefix + "-" + timestamp
}

func UniqueRunLogPath(path string, now time.Time) string {
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

func sanitizeFileName(in string) string {
	trimmed := strings.TrimSpace(in)
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "-", "*", "_", "?", "", "\"", "'", "<", "", ">", "", "|", "_")
	out := replacer.Replace(trimmed)
	if out == "" {
		return "untitled"
	}
	return out
}
