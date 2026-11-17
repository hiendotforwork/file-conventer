package service

import (
	"reflect"
	"testing"
)

func TestSupportedTargetsRequiresAFormatForEveryFile(t *testing.T) {
	got := SupportedTargets([]File{{Name: "report.docx"}, {Name: "slides.pptx"}})
	if want := []string{"pdf"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SupportedTargets() = %v, want %v", got, want)
	}
	if got := SupportedTargets([]File{{Name: "report.docx"}, {Name: "sheet.xlsx"}}); !reflect.DeepEqual(got, []string{"pdf", "html"}) {
		t.Fatalf("shared outputs = %v, want [pdf html]", got)
	}
	if got := SupportedTargets([]File{{Name: "unknown.bin"}}); len(got) != 0 {
		t.Fatalf("unsupported file has outputs %v", got)
	}
}
