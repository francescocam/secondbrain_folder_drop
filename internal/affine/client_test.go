package affine

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportMarkdownUsesMCPCreateDocument(t *testing.T) {
	var authHeaders []string
	var calls []string

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/api/workspaces/ws-1/mcp/":
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal(err)
			}
			calls = append(calls, req["method"].(string))
			switch req["method"] {
			case "tools/list":
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"create_document"}]}}`))
			case "tools/call":
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"{\"success\":true}"}]}}`))
			default:
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer s.Close()

	d := t.TempDir()
	p := filepath.Join(d, "note.md")
	if err := os.WriteFile(p, []byte("# hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := New(s.URL, "abc", "ws-1", s.Client())
	if err := client.ImportMarkdown(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	if len(authHeaders) != 2 || authHeaders[0] != "Bearer abc" || authHeaders[1] != "Bearer abc" {
		t.Fatalf("unexpected auth headers: %+v", authHeaders)
	}
	if strings.Join(calls, ",") != "tools/list,tools/call" {
		t.Fatalf("unexpected calls: %+v", calls)
	}
}

func TestImportFileUploadsBlobThenCreatesDocument(t *testing.T) {
	var graphqlAuth string
	var mcpCalls []string

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
		case "/api/workspaces/ws-1/mcp/":
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal(err)
			}
			mcpCalls = append(mcpCalls, req["method"].(string))
			switch req["method"] {
			case "tools/list":
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"create_document"}]}}`))
			case "tools/call":
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"{\"success\":true}"}]}}`))
			default:
				http.NotFound(w, r)
			}
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

	client := New(s.URL, "abc", "ws-1", s.Client())
	if err := client.ImportFile(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if graphqlAuth != "Bearer abc" {
		t.Fatalf("unexpected graphql auth: %q", graphqlAuth)
	}
	if strings.Join(mcpCalls, ",") != "tools/list,tools/call" {
		t.Fatalf("unexpected mcp calls: %+v", mcpCalls)
	}
}

func TestImportMarkdownFailsWhenCreateDocumentToolMissing(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/workspaces/ws-1/mcp/" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"read_document"}]}}`))
	}))
	defer s.Close()

	d := t.TempDir()
	p := filepath.Join(d, "note.md")
	if err := os.WriteFile(p, []byte("# hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := New(s.URL, "abc", "ws-1", s.Client())
	if err := client.ImportMarkdown(context.Background(), p); err == nil {
		t.Fatal("expected error")
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
