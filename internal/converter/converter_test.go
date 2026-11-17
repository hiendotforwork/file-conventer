package converter

import (
	"archive/zip"
	"bytes"
	"reflect"
	"testing"
)

func TestSupportedTargets(t *testing.T) {
	for input, want := range map[string][]string{
		"report.docx": {"pdf", "txt", "html"},
		"slides.pptx": {"pdf"},
		"sheet.xlsx":  {"pdf", "html", "csv"},
		"page.html":   {"pdf", "txt"},
		"notes.md":    {"pdf", "docx", "html", "txt"},
		"scan.pdf":    nil,
		"unknown.bin": nil,
	} {
		if got := SupportedTargets(input); !reflect.DeepEqual(got, want) {
			t.Errorf("SupportedTargets(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestValidateInputRejectsMalformedFormats(t *testing.T) {
	for name, data := range map[string][]byte{
		"table.csv":  []byte("a,\"b\n"),
		"page.html":  []byte("just text"),
		"sheet.xlsx": []byte("PK\x03\x04not a workbook"),
		"notes.txt":  []byte("not utf-8: \xff"),
	} {
		if _, err := ValidateInput(name, data); err == nil {
			t.Errorf("ValidateInput(%q) accepted malformed content", name)
		}
	}
}

func TestValidateInputRejectsOfficeArchiveExpansionLimit(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	main, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := main.Write([]byte("<document/>")); err != nil {
		t.Fatal(err)
	}
	large, err := zw.Create("large.bin")
	if err != nil {
		t.Fatal(err)
	}
	block := bytes.Repeat([]byte("x"), 1<<20)
	for i := 0; i < 101; i++ {
		if _, err := large.Write(block); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateInput("bomb.docx", buf.Bytes()); err == nil {
		t.Fatal("accepted an Office archive above the expansion limit")
	}
}
