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

func TestNormalizePathBoundaries(t *testing.T) {
	// 空路径应报错
	if _, err := NormalizePath(""); err == nil {
		t.Errorf("expected error for empty path")
	}

	// 裸 ~ 展开为 home
	home, _ := os.UserHomeDir()
	got, err := NormalizePath("~")
	if err != nil || got != home {
		t.Errorf("expected ~ -> %s, got %s, err=%v", home, got, err)
	}

	// 相对路径转绝对
	got, err = NormalizePath(".")
	cwd, _ := os.Getwd()
	if err != nil || got != cwd {
		t.Errorf("expected . -> %s, got %s, err=%v", cwd, got, err)
	}

	// 尾部斜杠被清理（先归一化，兼容 macOS /var 符号链接）
	dir := t.TempDir()
	dir, err = NormalizePath(dir)
	if err != nil {
		t.Fatalf("failed to normalize temp dir: %v", err)
	}
	got, err = NormalizePath(dir + "/")
	if err != nil || got != dir {
		t.Errorf("expected trailing slash cleaned to %s, got %s, err=%v", dir, got, err)
	}

	// 不存在的目标不报错，返回清理后的绝对路径
	missing := filepath.Join(dir, "not-exist")
	got, err = NormalizePath(missing)
	if err != nil || got != missing {
		t.Errorf("expected %s, got %s, err=%v", missing, got, err)
	}
}

func TestIsAllowedRootDir(t *testing.T) {
	// 允许列表包含根目录时全部放行
	if !IsAllowed([]string{string(os.PathSeparator)}, "/etc") {
		t.Errorf("root dir in allowed list should allow everything")
	}
	if !IsAllowed([]string{string(os.PathSeparator)}, "/") {
		t.Errorf("root dir itself should be allowed")
	}
}
