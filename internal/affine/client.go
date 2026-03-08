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
}

type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("http status %d: %s", e.StatusCode, e.Body)
}

func New(baseURL, token, workspaceID string, hc *http.Client) *Client {
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		token:       token,
		workspaceID: workspaceID,
		http:        hc,
	}
}

func (c *Client) ImportMarkdown(ctx context.Context, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return c.createDocument(ctx, title, string(content))
}

func (c *Client) ImportFile(ctx context.Context, path string) error {
	blobID, err := c.setBlob(ctx, path)
	if err != nil {
		return err
	}
	name := filepath.Base(path)
	blobURL := c.baseURL + "/api/workspaces/" + url.PathEscape(c.workspaceID) + "/blobs/" + url.PathEscape(blobID)
	content := fmt.Sprintf("Imported by folder-drop service.\n\n[%s](%s)", name, blobURL)
	return c.createDocument(ctx, name, content)
}

func (c *Client) createDocument(ctx context.Context, title, content string) error {
	if err := c.ensureCreateDocumentTool(ctx); err != nil {
		return err
	}

	body := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "create_document",
			"arguments": map[string]any{
				"title":   title,
				"content": content,
			},
		},
	}
	var resp rpcEnvelope
	if err := c.postJSON(ctx, c.baseURL+"/api/workspaces/"+url.PathEscape(c.workspaceID)+"/mcp/", body, &resp); err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("affine create_document failed: %s", resp.Error.Message)
	}
	if resp.Result.IsError {
		return fmt.Errorf("affine create_document returned error")
	}
	return nil
}

func (c *Client) ensureCreateDocumentTool(ctx context.Context) error {
	body := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}
	var resp rpcEnvelope
	if err := c.postJSON(ctx, c.baseURL+"/api/workspaces/"+url.PathEscape(c.workspaceID)+"/mcp/", body, &resp); err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("affine tools/list failed: %s", resp.Error.Message)
	}
	for _, tool := range resp.Result.Tools {
		if tool.Name == "create_document" {
			return nil
		}
	}
	return fmt.Errorf("affine MCP tool create_document is unavailable for workspace %s", c.workspaceID)
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

func (c *Client) postJSON(ctx context.Context, endpoint string, payload any, out any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return &HTTPError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("parse affine response: %w", err)
	}
	return nil
}

func writeJSONPart(mw *multipart.Writer, field string, payload any) error {
	part, err := mw.CreateFormField(field)
	if err != nil {
		return err
	}
	return json.NewEncoder(part).Encode(payload)
}

type rpcEnvelope struct {
	Result struct {
		IsError bool `json:"isError"`
		Tools   []struct {
			Name string `json:"name"`
		} `json:"tools"`
	} `json:"result"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}
