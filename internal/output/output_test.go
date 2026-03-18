package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/somaz94/git-mirror-action/internal/config"
	"github.com/somaz94/git-mirror-action/internal/mirror"
)

func TestWriteSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "github_output")
	os.WriteFile(outputFile, []byte{}, 0644)
	t.Setenv("GITHUB_OUTPUT", outputFile)

	results := []mirror.Result{
		{
			Target:  config.Target{Provider: config.ProviderGitLab, URL: "https://gitlab.com/org/repo.git"},
			Success: true,
			Message: "mirrored successfully",
		},
	}

	err := Write(results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(outputFile)
	content := string(data)

	if !strings.Contains(content, "mirrored_count=1") {
		t.Errorf("expected mirrored_count=1 in output, got: %s", content)
	}
	if !strings.Contains(content, "failed_count=0") {
		t.Errorf("expected failed_count=0 in output, got: %s", content)
	}
	if !strings.Contains(content, "result=") {
		t.Errorf("expected result= in output, got: %s", content)
	}
}

func TestWriteWithFailures(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "github_output")
	os.WriteFile(outputFile, []byte{}, 0644)
	t.Setenv("GITHUB_OUTPUT", outputFile)

	results := []mirror.Result{
		{
			Target:  config.Target{Provider: config.ProviderGitLab, URL: "https://gitlab.com/org/repo.git"},
			Success: true,
			Message: "mirrored successfully",
		},
		{
			Target:  config.Target{Provider: config.ProviderBitbucket, URL: "https://bitbucket.org/org/repo.git"},
			Success: false,
			Message: "auth failed",
		},
	}

	err := Write(results)
	if err == nil {
		t.Fatal("expected error when there are failures")
	}
	if !strings.Contains(err.Error(), "1 mirror target(s) failed") {
		t.Errorf("unexpected error message: %v", err)
	}

	data, _ := os.ReadFile(outputFile)
	content := string(data)

	if !strings.Contains(content, "mirrored_count=1") {
		t.Errorf("expected mirrored_count=1, got: %s", content)
	}
	if !strings.Contains(content, "failed_count=1") {
		t.Errorf("expected failed_count=1, got: %s", content)
	}
}

func TestWriteNoOutputFile(t *testing.T) {
	t.Setenv("GITHUB_OUTPUT", "")

	results := []mirror.Result{
		{
			Target:  config.Target{Provider: config.ProviderGeneric, URL: "https://example.com/repo.git"},
			Success: true,
			Message: "ok",
		},
	}

	err := Write(results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteInvalidOutputPath(t *testing.T) {
	t.Setenv("GITHUB_OUTPUT", "/nonexistent/path/output")

	results := []mirror.Result{
		{
			Target:  config.Target{Provider: config.ProviderGeneric, URL: "https://example.com/repo.git"},
			Success: true,
			Message: "ok",
		},
	}

	err := Write(results)
	if err == nil {
		t.Fatal("expected error for invalid output path")
	}
}

func TestWriteEmptyResults(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "github_output")
	os.WriteFile(outputFile, []byte{}, 0644)
	t.Setenv("GITHUB_OUTPUT", outputFile)

	results := []mirror.Result{}

	err := Write(results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(outputFile)
	content := string(data)

	if !strings.Contains(content, "mirrored_count=0") {
		t.Errorf("expected mirrored_count=0, got: %s", content)
	}
	if !strings.Contains(content, "failed_count=0") {
		t.Errorf("expected failed_count=0, got: %s", content)
	}
}
