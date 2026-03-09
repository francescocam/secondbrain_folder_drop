package affine

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

//go:embed helper_bundle.cjs
var helperBundle []byte

type nodeWriter struct {
	once     sync.Once
	helper   string
	setupErr error
}

func (w *nodeWriter) CreateDoc(ctx context.Context, req createDocRequest) error {
	if err := w.ensureHelper(); err != nil {
		return err
	}

	nodePath, err := exec.LookPath("node")
	if err != nil {
		return fmt.Errorf("affine node runtime not found on PATH: %w", err)
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, nodePath, w.helper)
	cmd.Stdin = bytes.NewReader(payload)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) == 0 {
			return fmt.Errorf("affine node helper failed: %w", err)
		}
		return fmt.Errorf("affine node helper failed: %s", string(out))
	}

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return fmt.Errorf("parse affine helper response: %w", err)
	}
	if !result.OK {
		if result.Error == "" {
			result.Error = "unknown helper failure"
		}
		return fmt.Errorf("affine document write failed: %s", result.Error)
	}
	return nil
}

func (w *nodeWriter) ensureHelper() error {
	w.once.Do(func() {
		dir, err := os.MkdirTemp("", "bfd-affine-helper-*")
		if err != nil {
			w.setupErr = err
			return
		}
		path := filepath.Join(dir, "helper_bundle.cjs")
		if err := os.WriteFile(path, helperBundle, 0o700); err != nil {
			w.setupErr = err
			return
		}
		w.helper = path
	})
	return w.setupErr
}
