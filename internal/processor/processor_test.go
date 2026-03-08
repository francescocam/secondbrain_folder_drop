package processor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestMoveFile(t *testing.T) {
	d := t.TempDir()
	src := filepath.Join(d, "a.txt")
	dst := filepath.Join(d, "sub", "a.txt")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := moveFile(src, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("expected destination file: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("expected source removed, err=%v", err)
	}
}

type fakeSink struct {
	markdownCalls int
	fileCalls     int
	lastMarkdown  string
	lastFile      string
	markdownErr   error
	fileErr       error
}

func (f *fakeSink) ImportMarkdown(_ context.Context, path string) error {
	f.markdownCalls++
	f.lastMarkdown = path
	return f.markdownErr
}

func (f *fakeSink) ImportFile(_ context.Context, path string) error {
	f.fileCalls++
	f.lastFile = path
	return f.fileErr
}

func TestProcessMarkdownCallsAllSinks(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "a.md")
	if err := os.WriteFile(p, []byte("# title"), 0o644); err != nil {
		t.Fatal(err)
	}

	sink1 := &fakeSink{}
	sink2 := &fakeSink{}
	proc := New([]Sink{sink1, sink2}, Config{DeleteOnOK: true, FailedDir: filepath.Join(d, "failed")})
	if err := proc.Process(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if sink1.markdownCalls != 1 || sink2.markdownCalls != 1 {
		t.Fatalf("unexpected markdown calls sink1=%d sink2=%d", sink1.markdownCalls, sink2.markdownCalls)
	}
	if sink1.fileCalls != 0 || sink2.fileCalls != 0 {
		t.Fatalf("expected no file imports, got sink1=%d sink2=%d", sink1.fileCalls, sink2.fileCalls)
	}
}

func TestProcessBinaryCallsAllSinks(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "a.bin")
	if err := os.WriteFile(p, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}

	sink1 := &fakeSink{}
	sink2 := &fakeSink{}
	proc := New([]Sink{sink1, sink2}, Config{DeleteOnOK: true, FailedDir: filepath.Join(d, "failed")})
	if err := proc.Process(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if sink1.fileCalls != 1 || sink2.fileCalls != 1 {
		t.Fatalf("unexpected file calls sink1=%d sink2=%d", sink1.fileCalls, sink2.fileCalls)
	}
	if sink1.markdownCalls != 0 || sink2.markdownCalls != 0 {
		t.Fatalf("expected no markdown imports, got sink1=%d sink2=%d", sink1.markdownCalls, sink2.markdownCalls)
	}
}

func TestProcessCallsRemainingSinksAfterFailure(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "a.md")
	if err := os.WriteFile(p, []byte("# title"), 0o644); err != nil {
		t.Fatal(err)
	}

	sink1 := &fakeSink{markdownErr: os.ErrPermission}
	sink2 := &fakeSink{}
	proc := New([]Sink{sink1, sink2}, Config{DeleteOnOK: true, FailedDir: filepath.Join(d, "failed")})

	if err := proc.Process(context.Background(), p); err == nil {
		t.Fatal("expected error")
	}
	if sink2.markdownCalls != 1 {
		t.Fatalf("expected second sink called once, got %d", sink2.markdownCalls)
	}
}
