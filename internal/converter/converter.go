package converter

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

var targets = map[string][]string{
	".docx": {"pdf", "txt", "html"},
	".pptx": {"pdf"},
	".xlsx": {"pdf", "html", "csv"}, // shortcut: CSV export keeps only the first sheet; switch to XLSX to retain all sheets.
	".csv":  {"xlsx", "pdf", "html"},
	".txt":  {"docx", "pdf", "html"},
	".html": {"pdf", "txt"},
	".md":   {"pdf", "docx", "html", "txt"},
}

func SupportedTargets(filename string) []string {
	return append([]string(nil), targets[strings.ToLower(filepath.Ext(filename))]...)
}

func ValidateInput(filename string, data []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if _, ok := targets[ext]; !ok || len(data) == 0 {
		return "", errors.New("unsupported file")
	}
	switch ext {
	case ".docx", ".pptx", ".xlsx":
		if !validOffice(data, map[string]string{".docx": "word/document.xml", ".pptx": "ppt/presentation.xml", ".xlsx": "xl/workbook.xml"}[ext]) {
			return "", errors.New("invalid file")
		}
	case ".html":
		if !validText(data) || !strings.Contains(strings.ToLower(string(data)), "<html") {
			return "", errors.New("invalid file")
		}
	case ".csv":
		if !validText(data) {
			return "", errors.New("invalid file")
		}
		r := csv.NewReader(bytes.NewReader(data))
		r.FieldsPerRecord = -1
		if _, err := r.ReadAll(); err != nil {
			return "", errors.New("invalid file")
		}
	case ".txt", ".md":
		if !validText(data) {
			return "", errors.New("invalid file")
		}
	}
	return ext, nil
}

func validText(data []byte) bool {
	if !utf8.Valid(data) {
		return false
	}
	for _, b := range data {
		if b == 0 {
			return false
		}
	}
	return true
}

func validOffice(data []byte, required string) bool {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(zr.File) > 10000 {
		return false
	}
	var size uint64
	found := false
	for _, f := range zr.File {
		if f.UncompressedSize64 > (100<<20)-size {
			return false
		}
		size += f.UncompressedSize64
		if f.Name == required {
			found = true
		}
	}
	return found
}

func Convert(ctx context.Context, executable, profile, source, outDir, target string) (string, error) {
	profileURL := "file://" + filepath.ToSlash(profile)
	cmd := exec.CommandContext(ctx, executable, "-env:UserInstallation="+profileURL, "--headless", "--convert-to", target, "--outdir", outDir, source)
	env := make([]string, 0, len(os.Environ())+1)
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "TMPDIR=") {
			env = append(env, value)
		}
	}
	cmd.Env = append(env, "TMPDIR="+filepath.Dir(profile))
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("conversion failed: %s", strings.TrimSpace(string(output)))
	}
	return filepath.Join(outDir, strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))+"."+target), nil
}
