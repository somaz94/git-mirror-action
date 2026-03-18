package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/somaz94/multi-git-mirror/internal/config"
	"github.com/somaz94/multi-git-mirror/internal/mirror"
	"github.com/somaz94/multi-git-mirror/internal/output"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "::error::%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Trust the GitHub Actions workspace directory inside Docker
	if workspace := os.Getenv("GITHUB_WORKSPACE"); workspace != "" {
		exec.Command("git", "config", "--global", "--add", "safe.directory", workspace).Run()
	}
	exec.Command("git", "config", "--global", "--add", "safe.directory", "/github/workspace").Run()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	m := mirror.New(cfg)
	results := m.Run()

	return output.Write(results)
}
