package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "not-exist.yaml"))
	if err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil config for missing file")
	}
}

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := "allowed_dirs:\n  - " + dir + "\n  - ~/code\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if len(cfg.AllowedDirs) != 2 {
		t.Fatalf("expected 2 allowed dirs, got %v", cfg.AllowedDirs)
	}

	resolved, err := cfg.ResolvedAllowedDirs()
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}
	if resolved[0] != dir {
		t.Errorf("expected %s, got %s", dir, resolved[0])
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, "code")
	if resolved[1] != want {
		t.Errorf("expected ~ expanded to %s, got %s", want, resolved[1])
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("allowed_dirs: [unclosed"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	if _, err := Load(cfgPath); err == nil {
		t.Fatalf("expected error for invalid yaml")
	}
}

func TestIsAllowed(t *testing.T) {
	root := t.TempDir()
	allowed := []string{root}

	cases := []struct {
		path string
		want bool
	}{
		{root, true},
		{filepath.Join(root, "sub"), true},
		{filepath.Join(root, "sub", "deeper"), true},
		{root + "2", false}, // 前缀相同但不在目录边界上
		{filepath.Dir(root), false},
		{"/nonexistent/path", false},
	}
	for _, tc := range cases {
		if got := IsAllowed(allowed, tc.path); got != tc.want {
			t.Errorf("IsAllowed(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestIsAllowedEmptyList(t *testing.T) {
	if !IsAllowed(nil, "/any/path") {
		t.Errorf("empty allowed list should allow everything")
	}
	if !IsAllowed([]string{}, "/any/path") {
		t.Errorf("empty allowed list should allow everything")
	}
}
