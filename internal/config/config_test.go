package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidation(t *testing.T) {
	d := t.TempDir()
	cfgPath := filepath.Join(d, "cfg.yaml")
	if err := os.WriteFile(cfgPath, []byte("blinko:\n  base_url: \"\"\n  jwt_token: \"\"\naffine:\n  base_url: \"\"\n  auth_token: \"\"\n  workspace_id: \"\"\nwatch:\n  input_dir: \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(cfgPath); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestLoadWithAffineEnvOverrides(t *testing.T) {
	d := t.TempDir()
	cfgPath := filepath.Join(d, "cfg.yaml")
	cfg := "" +
		"blinko:\n" +
		"  base_url: \"http://blinko\"\n" +
		"  jwt_token: \"blinko-token\"\n" +
		"affine:\n" +
		"  base_url: \"http://affine\"\n" +
		"  auth_token: \"affine-token\"\n" +
		"  workspace_id: \"ws-1\"\n" +
		"watch:\n" +
		"  input_dir: \"" + d + "/inbox\"\n"
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("BFD_AFFINE_BASE_URL", "http://override-affine")
	t.Setenv("BFD_AFFINE_AUTH_TOKEN", "override-token")
	t.Setenv("BFD_AFFINE_WORKSPACE_ID", "override-workspace")

	loaded, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Affine.BaseURL != "http://override-affine" {
		t.Fatalf("unexpected affine base url: %q", loaded.Affine.BaseURL)
	}
	if loaded.Affine.AuthToken != "override-token" {
		t.Fatalf("unexpected affine auth token: %q", loaded.Affine.AuthToken)
	}
	if loaded.Affine.WorkspaceID != "override-workspace" {
		t.Fatalf("unexpected affine workspace id: %q", loaded.Affine.WorkspaceID)
	}
}

func TestLoadForTargetBlinkoDoesNotRequireAffine(t *testing.T) {
	d := t.TempDir()
	cfgPath := filepath.Join(d, "cfg.yaml")
	cfg := "" +
		"blinko:\n" +
		"  base_url: \"http://blinko\"\n" +
		"  jwt_token: \"blinko-token\"\n" +
		"affine:\n" +
		"  base_url: \"\"\n" +
		"  auth_token: \"\"\n" +
		"  workspace_id: \"\"\n" +
		"watch:\n" +
		"  input_dir: \"" + filepath.Join(d, "inbox") + "\"\n"
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadForTarget(cfgPath, TargetBlinko); err != nil {
		t.Fatalf("expected blinko-only config to validate: %v", err)
	}
}

func TestLoadForTargetAffineDoesNotRequireBlinko(t *testing.T) {
	d := t.TempDir()
	cfgPath := filepath.Join(d, "cfg.yaml")
	cfg := "" +
		"blinko:\n" +
		"  base_url: \"\"\n" +
		"  jwt_token: \"\"\n" +
		"affine:\n" +
		"  base_url: \"http://affine\"\n" +
		"  auth_token: \"affine-token\"\n" +
		"  workspace_id: \"ws-1\"\n" +
		"watch:\n" +
		"  input_dir: \"" + filepath.Join(d, "inbox") + "\"\n"
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadForTarget(cfgPath, TargetAffine); err != nil {
		t.Fatalf("expected affine-only config to validate: %v", err)
	}
}

func TestParseTarget(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Target
		wantErr error
	}{
		{name: "default empty", input: "", want: TargetBoth},
		{name: "both", input: "both", want: TargetBoth},
		{name: "blinko", input: "blinko", want: TargetBlinko},
		{name: "affine", input: "affine", want: TargetAffine},
		{name: "case insensitive", input: "AFFINE", want: TargetAffine},
		{name: "invalid", input: "dropbox", wantErr: errors.New("invalid")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTarget(tt.input)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected target: got %q want %q", got, tt.want)
			}
		})
	}
}
