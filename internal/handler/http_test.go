package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"file-converter/internal/service"
)

func TestFormatsEndpointReturnsOnlyCommonOutputs(t *testing.T) {
	h := New(service.New("unused"), []byte("ok"), http.NotFoundHandler())
	r := httptest.NewRequest("GET", "/formats?ext=docx&ext=pptx", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || w.Body.String() != "{\"formats\":[\"pdf\"]}\n" {
		t.Fatalf("GET /formats = %d %q", w.Code, w.Body.String())
	}
}

func TestSafeNameRemovesDirectoriesAndControls(t *testing.T) {
	if got := safeName(`..\\secret\report.docx`); got != "report.docx" {
		t.Fatalf("safeName() = %q", got)
	}
	if got := safeName("bad\nname.txt"); got != "badname.txt" {
		t.Fatalf("safeName() retained a control: %q", got)
	}
}
