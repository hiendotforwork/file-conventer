package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"file-converter/internal/converter"
	"file-converter/internal/storage"
)

const (
	MaxFiles       = 10
	MaxInputBytes  = 25 << 20
	MaxOutputBytes = 100 << 20
)

var (
	ErrInvalid     = errors.New("invalid file")
	ErrUnsupported = errors.New("unsupported conversion")
	ErrTooLarge    = errors.New("file too large")
	ErrBusy        = errors.New("converter busy")
	ErrFailed      = errors.New("conversion failed")
)

type File struct {
	Name string
	Data []byte
}

type Result struct {
	Name string
	Data []byte
}

type Service struct {
	soffice string
	slots   chan struct{}
}

func New(soffice string) *Service { return &Service{soffice: soffice, slots: make(chan struct{}, 2)} }

func SupportedTargets(files []File) []string {
	if len(files) == 0 {
		return nil
	}
	common := converter.SupportedTargets(files[0].Name)
	for _, f := range files[1:] {
		next := map[string]bool{}
		for _, x := range converter.SupportedTargets(f.Name) {
			next[x] = true
		}
		filtered := common[:0]
		for _, x := range common {
			if next[x] {
				filtered = append(filtered, x)
			}
		}
		common = filtered
	}
	return common
}

func (s *Service) Convert(ctx context.Context, files []File, target string) ([]Result, error) {
	if len(files) == 0 || len(files) > MaxFiles {
		return nil, ErrInvalid
	}
	if !contains(SupportedTargets(files), target) {
		return nil, ErrUnsupported
	}
	total := 0
	extensions := make([]string, len(files))
	for i, f := range files {
		if len(f.Data) == 0 {
			return nil, ErrInvalid
		}
		total += len(f.Data)
		if len(f.Data) > MaxInputBytes || total > MaxInputBytes {
			return nil, ErrTooLarge
		}
		ext, err := converter.ValidateInput(f.Name, f.Data)
		if err != nil {
			return nil, ErrInvalid
		}
		extensions[i] = ext
	}
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, ErrBusy
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	dir, err := storage.NewRequestDir()
	if err != nil {
		return nil, ErrFailed
	}
	defer os.RemoveAll(dir)
	profile := filepath.Join(dir, "profile")
	results := make([]Result, 0, len(files))
	var outputBytes int64
	for i, f := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		ext := extensions[i]
		source := filepath.Join(dir, fmt.Sprintf("input-%02d%s", i, ext))
		if err := os.WriteFile(source, f.Data, 0600); err != nil {
			return nil, ErrFailed
		}
		out, err := converter.Convert(ctx, s.soffice, profile, source, dir, target)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, ErrFailed
		}
		info, err := os.Stat(out)
		if err != nil || info.Size() == 0 || info.Size() > MaxOutputBytes {
			return nil, ErrFailed
		}
		outputBytes += info.Size()
		if outputBytes > MaxOutputBytes {
			return nil, ErrFailed
		}
		data, err := os.ReadFile(out)
		if err != nil {
			return nil, ErrFailed
		}
		stem := strings.TrimSuffix(filepath.Base(f.Name), filepath.Ext(f.Name))
		name := stem + "." + target
		if len(files) > 1 {
			name = fmt.Sprintf("%02d-%s", i+1, name)
		}
		results = append(results, Result{Name: name, Data: data})
	}
	return results, nil
}

func contains(values []string, value string) bool {
	for _, x := range values {
		if x == value {
			return true
		}
	}
	return false
}
