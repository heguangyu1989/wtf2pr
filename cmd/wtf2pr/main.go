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

func main() {
	var port int
	var host string
	var workDir string
	var reviewFile string
	var reviewID string
	var token string
	var configFile string

	rootCmd := &cobra.Command{
		Use:   "wtf2pr",
		Short: "A tool for reviewing git diffs in web UI",
	}

	webCmd := &cobra.Command{
		Use:   "web",
		Short: "Start the web server for diff review",
		Run: func(cmd *cobra.Command, args []string) {
			if workDir == "" {
				var err error
				workDir, err = os.Getwd()
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to get current dir: %v\n", err)
					os.Exit(1)
				}
			}

			if reviewFile != "" {
				_ = os.MkdirAll(filepath.Dir(reviewFile), 0755)
			}

			if token == "" {
				var err error
				token, err = generateToken()
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to generate token: %v\n", err)
					os.Exit(1)
				}
			}

			allowedDirs, err := loadAllowedDirs(configFile, cmd.Flags().Changed("config"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
				os.Exit(1)
			}
			if len(allowedDirs) > 0 && !config.IsAllowed(allowedDirs, workDir) {
				fmt.Fprintf(os.Stderr, "warning: workdir %s is outside the allowed directories in config\n", workDir)
			}

			srv := server.NewServer(server.Options{
				WorkDir:     workDir,
				StaticFS:    staticFS,
				ReviewFile:  reviewFile,
				ReviewID:    reviewID,
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

	webCmd.Flags().IntVar(&port, "port", 8322, "Server port")
	webCmd.Flags().StringVar(&host, "host", "0.0.0.0", "Server host")
	webCmd.Flags().StringVar(&workDir, "workdir", "", "Working directory (git repo)")
	webCmd.Flags().StringVar(&reviewFile, "review-file", "", "Review storage JSON file path")
	webCmd.Flags().StringVar(&reviewID, "review-id", "", "Review ID for default file naming")
	webCmd.Flags().StringVar(&token, "token", "", "Access token for web UI (default: randomly generated on startup)")
	webCmd.Flags().StringVar(&configFile, "config", defaultConfigPath(), "YAML config file path (restrict accessible paths etc.)")

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

// loadAllowedDirs 加载配置文件中的允许目录列表；显式指定 --config 但文件不存在时返回错误
func loadAllowedDirs(configFile string, explicit bool) ([]string, error) {
	cfg, err := config.Load(configFile)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		if explicit {
			return nil, fmt.Errorf("config file not found: %s", configFile)
		}
		return nil, nil
	}
	return cfg.ResolvedAllowedDirs()
}

// generateToken 生成随机访问令牌
func generateToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
