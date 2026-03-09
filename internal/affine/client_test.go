package affine

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeWriter struct {
	req createDocRequest
	err error
}

func (f *fakeWriter) CreateDoc(_ context.Context, req createDocRequest) error {
	f.req = req
	return f.err
}

func TestImportMarkdownCreatesDocViaWriter(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "note.md")
	if err := os.WriteFile(p, []byte("# hello\n\nParagraph"), 0o644); err != nil {
		t.Fatal(err)
	}

	writer := &fakeWriter{}
	client := newWithWriter("http://affine", "abc", "ws-1", http.DefaultClient, writer)
	if err := client.ImportMarkdown(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	if writer.req.Title != "hello" {
		t.Fatalf("unexpected title: %q", writer.req.Title)
	}
	if writer.req.Markdown != "Paragraph" {
		t.Fatalf("unexpected markdown: %q", writer.req.Markdown)
	}
	if writer.req.BaseURL != "http://affine" || writer.req.Token != "abc" || writer.req.WorkspaceID != "ws-1" {
		t.Fatalf("unexpected request: %+v", writer.req)
	}
}

func TestImportMarkdownFallsBackToFilenameWhenNoH1(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "note.md")
	if err := os.WriteFile(p, []byte("Paragraph"), 0o644); err != nil {
		t.Fatal(err)
	}

	writer := &fakeWriter{}
	client := newWithWriter("http://affine", "abc", "ws-1", http.DefaultClient, writer)
	if err := client.ImportMarkdown(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	if writer.req.Title != "note" {
		t.Fatalf("unexpected title: %q", writer.req.Title)
	}
	if writer.req.Markdown != "Paragraph" {
		t.Fatalf("unexpected markdown: %q", writer.req.Markdown)
	}
}

func TestImportMarkdownPropagatesWriterError(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "note.md")
	if err := os.WriteFile(p, []byte("# hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	writer := &fakeWriter{err: errors.New("boom")}
	client := newWithWriter("http://affine", "abc", "ws-1", http.DefaultClient, writer)
	if err := client.ImportMarkdown(context.Background(), p); err == nil || err.Error() != "boom" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImportFileUploadsBlobThenCreatesDocument(t *testing.T) {
	var graphqlAuth string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/graphql":
			graphqlAuth = r.Header.Get("Authorization")
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatal(err)
			}
			if r.FormValue("operations") == "" || r.FormValue("map") == "" {
				t.Fatal("expected graphql multipart payload")
			}
			file, _, err := r.FormFile("0")
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			b, err := io.ReadAll(file)
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != "abc" {
				t.Fatalf("unexpected upload body: %q", string(b))
			}
			_, _ = w.Write([]byte(`{"data":{"setBlob":"blob-id"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer s.Close()

	d := t.TempDir()
	p := filepath.Join(d, "a.bin")
	if err := os.WriteFile(p, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}

	writer := &fakeWriter{}
	client := newWithWriter(s.URL, "abc", "ws-1", s.Client(), writer)
	if err := client.ImportFile(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if graphqlAuth != "Bearer abc" {
		t.Fatalf("unexpected graphql auth: %q", graphqlAuth)
	}
	if writer.req.Title != "a.bin" {
		t.Fatalf("unexpected title: %q", writer.req.Title)
	}
	if !strings.Contains(writer.req.Markdown, "[a.bin]("+s.URL+"/api/workspaces/ws-1/blobs/blob-id)") {
		t.Fatalf("unexpected markdown: %q", writer.req.Markdown)
	}
}

func TestWriteJSONPart(t *testing.T) {
	var buf strings.Builder
	mw := multipart.NewWriter(&buf)
	if err := writeJSONPart(mw, "x", map[string]string{"a": "b"}); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"a":"b"`) {
		t.Fatalf("unexpected body: %q", buf.String())
	}
}
