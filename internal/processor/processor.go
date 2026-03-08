package processor

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	DeleteOnOK bool
	ArchiveDir string
	FailedDir  string
}

type Sink interface {
	ImportMarkdown(ctx context.Context, path string) error
	ImportFile(ctx context.Context, path string) error
}

type Processor struct {
	sinks []Sink
	cfg   Config
}

type FailureRecord struct {
	File      string    `json:"file"`
	Error     string    `json:"error"`
	At        time.Time `json:"at"`
	Attempts  int       `json:"attempts"`
	Permanent bool      `json:"permanent"`
}

func New(sinks []Sink, cfg Config) *Processor { return &Processor{sinks: sinks, cfg: cfg} }

func (p *Processor) Process(ctx context.Context, filePath string) error {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".md" || ext == ".markdown" {
		return p.processAll(func(sink Sink) error {
			return sink.ImportMarkdown(ctx, filePath)
		})
	}

	return p.processAll(func(sink Sink) error {
		return sink.ImportFile(ctx, filePath)
	})
}

func (p *Processor) processAll(run func(Sink) error) error {
	var firstErr error
	for _, sink := range p.sinks {
		if err := run(sink); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (p *Processor) FinalizeSuccess(path string) (bool, error) {
	if p.cfg.DeleteOnOK {
		return true, os.Remove(path)
	}
	base := filepath.Base(path)
	dst := filepath.Join(p.cfg.ArchiveDir, base)
	return false, moveFile(path, dst)
}

func (p *Processor) Quarantine(path string, rec FailureRecord) error {
	if err := os.MkdirAll(p.cfg.FailedDir, 0o755); err != nil {
		return err
	}
	base := filepath.Base(path)
	dst := filepath.Join(p.cfg.FailedDir, base)
	if err := moveFile(path, dst); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}

	rec.File = dst
	b, _ := json.MarshalIndent(rec, "", "  ")
	return os.WriteFile(dst+".error.json", b, 0o644)
}

func moveFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := out.ReadFrom(in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Remove(src)
}
