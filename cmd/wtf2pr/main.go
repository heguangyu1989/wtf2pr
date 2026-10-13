package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/wtf2pr/wtf2pr/internal/config"
	"github.com/wtf2pr/wtf2pr/internal/server"
)

//go:embed all:dist
var staticFS embed.FS

const (
	defaultPort = 8322
	defaultHost = "127.0.0.1"
)

func main() {
	var port int
	var workDir string
	var configFile string

	rootCmd := &cobra.Command{
		Use:   "wtf2pr",
		Short: "A tool for reviewing git diffs in web UI",
	}

	webCmd := &cobra.Command{
		Use:   "web",
		Short: "Start the web server for diff review",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := loadConfig(configFile, cmd.Flags().Changed("config"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
				os.Exit(1)
			}

			// 配置优先级：CLI 显式指定 > 配置文件 > 默认值
			if !cmd.Flags().Changed("port") && cfg.Port != 0 {
				port = cfg.Port
			}
			host := defaultHost
			if cfg.Host != "" {
				host = cfg.Host
			}
			if !cmd.Flags().Changed("workdir") && cfg.WorkDir != "" {
				workDir = cfg.WorkDir
			}
			if workDir == "" {
				workDir, err = os.Getwd()
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to get current dir: %v\n", err)
					os.Exit(1)
				}
			} else {
				// 与 allowed_dirs 行为一致：展开 ~、绝对化并解析符号链接
				workDir, err = config.NormalizePath(workDir)
				if err != nil {
					fmt.Fprintf(os.Stderr, "invalid workdir: %v\n", err)
					os.Exit(1)
				}
			}

			token := cfg.Token
			if token == "" {
				token, err = generateToken()
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to generate token: %v\n", err)
					os.Exit(1)
				}
			}

			allowedDirs, err := cfg.ResolvedAllowedDirs()
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
				os.Exit(1)
			}
			if port < 1 || port > 65535 {
				fmt.Fprintf(os.Stderr, "invalid port %d: must be between 1 and 65535\n", port)
				os.Exit(1)
			}
			if len(allowedDirs) > 0 && !config.IsAllowed(allowedDirs, workDir) {
				fmt.Fprintf(os.Stderr, "warning: workdir %s is outside the allowed directories in config\n", workDir)
			}

			srv := server.NewServer(server.Options{
				WorkDir:     workDir,
				StaticFS:    staticFS,
				Token:       token,
				AllowedDirs: allowedDirs,
			})
			fmt.Printf("Starting server at http://%s:%d (workdir: %s)\n", host, port, workDir)
			fmt.Printf("Access token: %s\n", token)
			if len(allowedDirs) > 0 {
				fmt.Printf("Path restriction enabled (%s), allowed dirs:\n", configFile)
				for _, dir := range allowedDirs {
					fmt.Printf("  - %s\n", dir)
				}
			}
			fmt.Printf("Open in browser: http://localhost:%d/?token=%s\n", port, token)
			if err := srv.Run(host, port); err != nil {
				fmt.Fprintf(os.Stderr, "server error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	webCmd.Flags().IntVar(&port, "port", defaultPort, "Server port (overrides config file)")
	webCmd.Flags().StringVar(&workDir, "workdir", "", "Working directory / git repo (overrides config file)")
	webCmd.Flags().StringVar(&configFile, "config", defaultConfigPath(), "YAML config file path")

	rootCmd.AddCommand(webCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// defaultConfigPath 默认配置文件路径
func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.yaml"
	}
	return filepath.Join(home, ".wtf2pr", "config.yaml")
}

// loadConfig 加载 YAML 配置文件；显式指定 --config 但文件不存在时返回错误
func loadConfig(configFile string, explicit bool) (*config.Config, error) {
	cfg, err := config.Load(configFile)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		if explicit {
			return nil, fmt.Errorf("config file not found: %s", configFile)
		}
		return &config.Config{}, nil
	}
	// 配置文件可能包含明文 token，权限过宽时提醒
	if info, statErr := os.Stat(configFile); statErr == nil && info.Mode().Perm()&0o077 != 0 {
		fmt.Fprintf(os.Stderr, "warning: config file %s is accessible by others, recommend chmod 600\n", configFile)
	}
	return cfg, nil
}

// generateToken 生成随机访问令牌
func generateToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
