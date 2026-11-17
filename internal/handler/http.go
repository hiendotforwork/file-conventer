package handler

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"file-converter/internal/service"
)

const maxRequestBytes = service.MaxInputBytes + (1 << 20)

var errUploadTooLarge = errors.New("upload too large")

type Handler struct {
	service *service.Service
	index   *template.Template
	static  http.Handler
}

func New(s *service.Service, index []byte, static http.Handler) http.Handler {
	h := &Handler{service: s, index: template.Must(template.New("index.html").Parse(string(index))), static: static}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.home)
	mux.HandleFunc("GET /formats", h.formats)
	mux.HandleFunc("POST /convert", h.convert)
	mux.Handle("GET /static/", http.StripPrefix("/static/", static))
	return mux
}

func (h *Handler) home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.index.Execute(w, nil)
}

func (h *Handler) formats(w http.ResponseWriter, r *http.Request) {
	files := make([]service.File, 0)
	for _, ext := range r.URL.Query()["ext"] {
		files = append(files, service.File{Name: "input." + strings.TrimPrefix(strings.ToLower(ext), ".")})
	}
	writeJSON(w, http.StatusOK, map[string][]string{"formats": service.SupportedTargets(files)})
}

func (h *Handler) convert(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	files, target, err := readMultipart(r)
	if err != nil {
		status := http.StatusBadRequest
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) || errors.Is(err, errUploadTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		writeJSON(w, status, map[string]string{"error": "Upload is invalid or too large."})
		return
	}
	results, err := h.service.Convert(r.Context(), files, target)
	if err != nil {
		status, message := http.StatusUnprocessableEntity, "Conversion failed. Check the file and try again."
		switch {
		case errors.Is(err, service.ErrUnsupported):
			message = "This output format is not supported for the selected files."
		case errors.Is(err, service.ErrInvalid):
			message = "One or more files are invalid or unsupported."
		case errors.Is(err, service.ErrTooLarge):
			status, message = http.StatusRequestEntityTooLarge, "The selected files exceed the upload limit."
		case errors.Is(err, service.ErrBusy):
			status, message = http.StatusServiceUnavailable, "The converter is busy. Try again shortly."
		case errors.Is(err, context.DeadlineExceeded):
			status, message = http.StatusGatewayTimeout, "Conversion took too long."
		}
		writeJSON(w, status, map[string]string{"error": message})
		return
	}
	if len(results) == 1 {
		w.Header().Set("Content-Type", contentType(target))
		w.Header().Set("Content-Disposition", disposition(results[0].Name))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(results[0].Data)
		return
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, result := range results {
		part, err := zw.Create(result.Name)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not package converted files."})
			return
		}
		if _, err = part.Write(result.Data); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not package converted files."})
			return
		}
	}
	if err := zw.Close(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not package converted files."})
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", disposition("converted-files.zip"))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

func readMultipart(r *http.Request) ([]service.File, string, error) {
	mr, err := r.MultipartReader()
	if err != nil {
		return nil, "", err
	}
	var files []service.File
	var target string
	total := 0
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, "", err
		}
		switch part.FormName() {
		case "files":
			if len(files) == service.MaxFiles {
				return nil, "", errors.New("too many files")
			}
			data, readErr := io.ReadAll(io.LimitReader(part, service.MaxInputBytes+1))
			if readErr != nil {
				return nil, "", readErr
			}
			total += len(data)
			if len(data) > service.MaxInputBytes || total > service.MaxInputBytes {
				return nil, "", errUploadTooLarge
			}
			name := safeName(part.FileName())
			if name == "" || len(data) == 0 {
				return nil, "", errors.New("invalid upload")
			}
			files = append(files, service.File{Name: name, Data: data})
		case "target":
			v, readErr := io.ReadAll(io.LimitReader(part, 16))
			if readErr != nil || len(v) == 16 {
				return nil, "", errors.New("invalid target")
			}
			target = strings.TrimPrefix(strings.ToLower(string(v)), ".")
		default:
			if _, err := io.Copy(io.Discard, part); err != nil {
				return nil, "", err
			}
		}
		_ = part.Close()
	}
	if len(files) == 0 || target == "" {
		return nil, "", errors.New("missing upload or target")
	}
	return files, target, nil
}

func safeName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)
	if name == "." || name == ".." {
		return ""
	}
	return name
}

func contentType(ext string) string {
	if t := mime.TypeByExtension("." + ext); t != "" {
		return t
	}
	return "application/octet-stream"
}

func disposition(filename string) string {
	return mime.FormatMediaType("attachment", map[string]string{"filename": filename})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
