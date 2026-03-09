package affine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Client struct {
	baseURL     string
	token       string
	workspaceID string
	http        *http.Client
	writer      docWriter
}

type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("http status %d: %s", e.StatusCode, e.Body)
}

type createDocRequest struct {
	BaseURL     string `json:"baseURL"`
	Token       string `json:"token"`
	WorkspaceID string `json:"workspaceID"`
	Title       string `json:"title"`
	Markdown    string `json:"markdown"`
}

type docWriter interface {
	CreateDoc(context.Context, createDocRequest) error
}

func New(baseURL, token, workspaceID string, hc *http.Client) *Client {
	return newWithWriter(baseURL, token, workspaceID, hc, &nodeWriter{})
}

func newWithWriter(baseURL, token, workspaceID string, hc *http.Client, writer docWriter) *Client {
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		token:       token,
		workspaceID: workspaceID,
		http:        hc,
		writer:      writer,
	}
}

func (c *Client) ImportMarkdown(ctx context.Context, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	title, body := splitMarkdownTitle(string(content), strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	return c.writer.CreateDoc(ctx, createDocRequest{
		BaseURL:     c.baseURL,
		Token:       c.token,
		WorkspaceID: c.workspaceID,
		Title:       title,
		Markdown:    body,
	})
}

func (c *Client) ImportFile(ctx context.Context, path string) error {
	blobID, err := c.setBlob(ctx, path)
	if err != nil {
		return err
	}
	name := filepath.Base(path)
	blobURL := c.baseURL + "/api/workspaces/" + url.PathEscape(c.workspaceID) + "/blobs/" + url.PathEscape(blobID)
	content := fmt.Sprintf("Imported by folder-drop service.\n\n[%s](%s)", name, blobURL)
	return c.writer.CreateDoc(ctx, createDocRequest{
		BaseURL:     c.baseURL,
		Token:       c.token,
		WorkspaceID: c.workspaceID,
		Title:       name,
		Markdown:    content,
	})
}

func splitMarkdownTitle(markdown, fallback string) (string, string) {
	normalized := strings.ReplaceAll(markdown, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			if title == "" {
				break
			}
			bodyLines := append([]string{}, lines[:i]...)
			bodyLines = append(bodyLines, lines[i+1:]...)
			body := strings.TrimLeft(strings.Join(bodyLines, "\n"), "\n")
			return title, body
		}
		break
	}
	return fallback, normalized
}

func (c *Client) setBlob(ctx context.Context, path string) (string, error) {
	query := `mutation setBlob($workspaceId: String!, $blob: Upload!) { setBlob(workspaceId: $workspaceId, blob: $blob) }`
	operations := map[string]any{
		"query": query,
		"variables": map[string]any{
			"workspaceId": c.workspaceID,
			"blob":        nil,
		},
	}
	mapPart := map[string][]string{"0": {"variables.blob"}}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := writeJSONPart(mw, "operations", operations); err != nil {
		return "", err
	}
	if err := writeJSONPart(mw, "map", mapPart); err != nil {
		return "", err
	}
	part, err := mw.CreateFormFile("0", filepath.Base(path))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", err
	}
	if err := mw.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/graphql", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", &HTTPError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	var payload struct {
		Data struct {
			SetBlob string `json:"setBlob"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("parse affine setBlob response: %w", err)
	}
	if len(payload.Errors) > 0 {
		return "", fmt.Errorf("affine setBlob failed: %s", payload.Errors[0].Message)
	}
	if payload.Data.SetBlob == "" {
		return "", fmt.Errorf("affine setBlob response missing blob id")
	}
	return payload.Data.SetBlob, nil
}

func writeJSONPart(mw *multipart.Writer, field string, payload any) error {
	part, err := mw.CreateFormField(field)
	if err != nil {
		return err
	}
	return json.NewEncoder(part).Encode(payload)
}
