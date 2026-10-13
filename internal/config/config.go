package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 启动配置文件（YAML）
type Config struct {
	// Port 服务监听端口，0 表示使用默认值
	Port int `yaml:"port"`
	// Host 服务监听地址
	Host string `yaml:"host"`
	// WorkDir Git 工作目录
	WorkDir string `yaml:"workdir"`
	// Token Web 访问令牌，为空时每次启动随机生成
	Token string `yaml:"token"`
	// AllowedDirs 允许访问的目录列表；为空表示不限制
	AllowedDirs []string `yaml:"allowed_dirs"`
}

// Load 读取 YAML 配置文件；文件不存在时返回 nil, nil（由调用方决定是否视为错误）
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config file %s: %w", path, err)
	}
	return &cfg, nil
}

// ResolvedAllowedDirs 返回规范化后的允许目录列表（展开 ~、转为绝对路径、解析符号链接）
func (c *Config) ResolvedAllowedDirs() ([]string, error) {
	if c == nil {
		return nil, nil
	}
	var out []string
	for _, dir := range c.AllowedDirs {
		resolved, err := NormalizePath(dir)
		if err != nil {
			return nil, fmt.Errorf("invalid allowed_dir %q: %w", dir, err)
		}
		out = append(out, resolved)
	}
	return out, nil
}

// NormalizePath 展开 ~ 并转为绝对路径；目标存在时解析符号链接
func NormalizePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, strings.TrimPrefix(path[1:], "/"))
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	return abs, nil
}

// IsAllowed 判断路径是否处于允许列表内；列表为空表示不限制
func IsAllowed(allowedDirs []string, path string) bool {
	if len(allowedDirs) == 0 {
		return true
	}
	normalized, err := NormalizePath(path)
	if err != nil {
		return false
	}
	for _, dir := range allowedDirs {
		if normalized == dir || strings.HasPrefix(normalized, dir+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}
