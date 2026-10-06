package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
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

			if reviewFile == "" {
				home, err := os.UserHomeDir()
				if err == nil {
					if reviewID != "" {
						reviewFile = filepath.Join(home, ".wtf2pr", fmt.Sprintf("review_%s.json", reviewID))
					} else {
						reviewFile = filepath.Join(home, ".wtf2pr", "review.json")
					}
				}
			}
			if reviewFile != "" {
				_ = os.MkdirAll(filepath.Dir(reviewFile), 0755)
			}

			srv := server.NewServer(workDir, staticFS, reviewFile)
			fmt.Printf("Starting server at http://%s:%d (workdir: %s)\n", host, port, workDir)
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
	webCmd.Flags().StringVar(&reviewID, "review-id", "", "Review ID for default file naming (default: ~/.wtf2pr/review.json)")

	rootCmd.AddCommand(webCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
