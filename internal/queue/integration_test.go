package queue

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"secondbrain-folder-drop/internal/affine"
	"secondbrain-folder-drop/internal/blinko"
	"secondbrain-folder-drop/internal/metrics"
	"secondbrain-folder-drop/internal/processor"
	"secondbrain-folder-drop/internal/store"
)

func TestMarkdownCreatesNoteAndDeletesFile(t *testing.T) {
	var upserts atomic.Int64
	var affineDocs atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/note/upsert":
			upserts.Add(1)
			w.WriteHeader(http.StatusOK)
		case "/api/workspaces/ws-1/mcp/":
			handleAffineMCP(t, w, r, &affineDocs, true)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	f := filepath.Join(dir, "note.md")
	mustWrite(t, f, "# hello")

	failedDir := filepath.Join(dir, "failed")
	q, cancel := newTestQueue(t, srv.URL, failedDir, true, "", 1, 2, 10*time.Millisecond)
	defer cancel()
	q.EnqueuePath(f)

	waitFor(t, func() bool {
		_, err := os.Stat(f)
		return os.IsNotExist(err)
	}, 2*time.Second)

	if upserts.Load() != 1 {
		t.Fatalf("expected 1 upsert, got %d", upserts.Load())
	}
	if affineDocs.Load() != 1 {
		t.Fatalf("expected 1 affine document, got %d", affineDocs.Load())
	}
}

func TestNonMarkdownUploadsThenCreatesNoteAndDeletesFile(t *testing.T) {
	var uploads atomic.Int64
	var upserts atomic.Int64
	var affineUploads atomic.Int64
	var affineDocs atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/file/upload":
			uploads.Add(1)
			_ = json.NewEncoder(w).Encode(blinko.UploadResponse{Path: "/api/file/a", Name: "a.bin", Size: 3, Type: "application/octet-stream"})
		case "/api/v1/note/upsert":
			upserts.Add(1)
			w.WriteHeader(http.StatusOK)
		case "/graphql":
			affineUploads.Add(1)
			_, _ = w.Write([]byte(`{"data":{"setBlob":"blob-id"}}`))
		case "/api/workspaces/ws-1/mcp/":
			handleAffineMCP(t, w, r, &affineDocs, true)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	f := filepath.Join(dir, "a.bin")
	mustWrite(t, f, "abc")

	failedDir := filepath.Join(dir, "failed")
	q, cancel := newTestQueue(t, srv.URL, failedDir, true, "", 1, 2, 10*time.Millisecond)
	defer cancel()
	q.EnqueuePath(f)

	waitFor(t, func() bool {
		_, err := os.Stat(f)
		return os.IsNotExist(err)
	}, 2*time.Second)

	if uploads.Load() != 1 {
		t.Fatalf("expected 1 upload, got %d", uploads.Load())
	}
	if upserts.Load() != 1 {
		t.Fatalf("expected 1 upsert, got %d", upserts.Load())
	}
	if affineUploads.Load() != 1 || affineDocs.Load() != 1 {
		t.Fatalf("expected 1 affine upload/doc, got uploads=%d docs=%d", affineUploads.Load(), affineDocs.Load())
	}
}

func TestUploadFailureRetriesThenQuarantine(t *testing.T) {
	var uploads atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/file/upload":
			uploads.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		case "/api/workspaces/ws-1/mcp/":
			handleAffineMCP(t, w, r, nil, true)
		case "/graphql":
			_, _ = w.Write([]byte(`{"data":{"setBlob":"blob-id"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	f := filepath.Join(dir, "a.bin")
	mustWrite(t, f, "abc")
	failedDir := filepath.Join(dir, "failed")

	q, cancel := newTestQueue(t, srv.URL, failedDir, true, "", 1, 2, 10*time.Millisecond)
	defer cancel()
	q.EnqueuePath(f)

	failedFile := filepath.Join(failedDir, "a.bin")
	waitFor(t, func() bool {
		_, err := os.Stat(failedFile)
		return err == nil
	}, 3*time.Second)
	waitFor(t, func() bool {
		_, err := os.Stat(failedFile + ".error.json")
		return err == nil
	}, 2*time.Second)

	if uploads.Load() != 3 {
		t.Fatalf("expected 3 upload attempts, got %d", uploads.Load())
	}
}

func TestUpsertFailureAfterUploadRetriesThenQuarantine(t *testing.T) {
	var uploads atomic.Int64
	var upserts atomic.Int64
	var affineUploads atomic.Int64
	var affineDocs atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/file/upload":
			uploads.Add(1)
			_ = json.NewEncoder(w).Encode(blinko.UploadResponse{Path: "/api/file/a", Name: "a.bin", Size: 3, Type: "application/octet-stream"})
		case "/api/v1/note/upsert":
			upserts.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("upsert failed"))
		case "/graphql":
			affineUploads.Add(1)
			_, _ = w.Write([]byte(`{"data":{"setBlob":"blob-id"}}`))
		case "/api/workspaces/ws-1/mcp/":
			handleAffineMCP(t, w, r, &affineDocs, true)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	f := filepath.Join(dir, "a.bin")
	mustWrite(t, f, "abc")
	failedDir := filepath.Join(dir, "failed")

	q, cancel := newTestQueue(t, srv.URL, failedDir, true, "", 1, 2, 10*time.Millisecond)
	defer cancel()
	q.EnqueuePath(f)

	failedFile := filepath.Join(failedDir, "a.bin")
	waitFor(t, func() bool {
		_, err := os.Stat(failedFile)
		return err == nil
	}, 3*time.Second)

	if uploads.Load() != 3 || upserts.Load() != 3 {
		t.Fatalf("expected 3 upload/upsert attempts, got uploads=%d upserts=%d", uploads.Load(), upserts.Load())
	}
	if affineUploads.Load() != 3 || affineDocs.Load() != 3 {
		t.Fatalf("expected 3 affine upload/doc attempts, got uploads=%d docs=%d", affineUploads.Load(), affineDocs.Load())
	}
}

func newTestQueue(t *testing.T, baseURL, failedDir string, deleteOnOK bool, archiveDir string, workers, maxRetries int, retryDelay time.Duration) (*Queue, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	hc := &http.Client{Timeout: 2 * time.Second}
	proc := processor.New([]processor.Sink{
		blinko.New(baseURL, "token", hc),
		affine.New(baseURL, "token", "ws-1", hc),
	}, processor.Config{DeleteOnOK: deleteOnOK, ArchiveDir: archiveDir, FailedDir: failedDir})
	q := New(128, workers, maxRetries, retryDelay, proc, metrics.New(), store.NewDedupe(5*time.Millisecond), func(string, ...any) {})
	go func() {
		_ = q.Run(ctx)
	}()
	return q, cancel
}

func handleAffineMCP(t *testing.T, w http.ResponseWriter, r *http.Request, docs *atomic.Int64, hasCreateTool bool) {
	t.Helper()

	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Fatal(err)
	}

	method, _ := req["method"].(string)
	switch method {
	case "tools/list":
		if hasCreateTool {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"create_document"}]}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`))
	case "tools/call":
		if docs != nil {
			docs.Add(1)
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"{\"success\":true}"}]}}`))
	default:
		http.NotFound(w, r)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func waitFor(t *testing.T, cond func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}
