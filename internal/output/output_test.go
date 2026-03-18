package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/somaz94/multi-git-mirror/internal/config"
	"github.com/somaz94/multi-git-mirror/internal/mirror"
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

func TestWriteMultipleSuccesses(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "github_output")
	os.WriteFile(outputFile, []byte{}, 0644)
	t.Setenv("GITHUB_OUTPUT", outputFile)

	results := []mirror.Result{
		{Target: config.Target{Provider: config.ProviderGitLab, URL: "https://gitlab.com/a"}, Success: true, Message: "ok"},
		{Target: config.Target{Provider: config.ProviderBitbucket, URL: "https://bitbucket.org/b"}, Success: true, Message: "ok"},
		{Target: config.Target{Provider: config.ProviderGeneric, URL: "https://example.com/c"}, Success: true, Message: "ok"},
	}

	err := Write(results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(outputFile)
	content := string(data)

	if !strings.Contains(content, "mirrored_count=3") {
		t.Errorf("expected mirrored_count=3, got: %s", content)
	}
	if !strings.Contains(content, "failed_count=0") {
		t.Errorf("expected failed_count=0, got: %s", content)
	}
}

func TestWriteAllFailures(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "github_output")
	os.WriteFile(outputFile, []byte{}, 0644)
	t.Setenv("GITHUB_OUTPUT", outputFile)

	results := []mirror.Result{
		{Target: config.Target{Provider: config.ProviderGitLab, URL: "https://gitlab.com/a"}, Success: false, Message: "err1"},
		{Target: config.Target{Provider: config.ProviderBitbucket, URL: "https://bitbucket.org/b"}, Success: false, Message: "err2"},
	}

	err := Write(results)
	if err == nil {
		t.Fatal("expected error when all targets fail")
	}
	if !strings.Contains(err.Error(), "2 mirror target(s) failed") {
		t.Errorf("expected 2 failures, got: %v", err)
	}

	data, _ := os.ReadFile(outputFile)
	content := string(data)

	if !strings.Contains(content, "mirrored_count=0") {
		t.Errorf("expected mirrored_count=0, got: %s", content)
	}
}

func TestWriteGitHubOutputDirectly(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "test_output")
	os.WriteFile(outputFile, []byte{}, 0644)

	err := writeGitHubOutput(outputFile, "test_key", "test_value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(outputFile)
	content := string(data)

	if !strings.Contains(content, "test_key=test_value") {
		t.Errorf("expected test_key=test_value, got: %s", content)
	}
}

func TestWriteReadOnlyOutputFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "readonly_output")
	os.WriteFile(outputFile, []byte{}, 0444)
	t.Setenv("GITHUB_OUTPUT", outputFile)

	results := []mirror.Result{
		{Target: config.Target{Provider: config.ProviderGeneric, URL: "https://example.com/a"}, Success: true, Message: "ok"},
	}

	err := Write(results)
	if err == nil {
		t.Fatal("expected error for read-only output file")
	}
}

func TestWriteNilResults(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "github_output")
	os.WriteFile(outputFile, []byte{}, 0644)
	t.Setenv("GITHUB_OUTPUT", outputFile)

	err := Write(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteGitHubOutputInvalidPath(t *testing.T) {
	err := writeGitHubOutput("/nonexistent/dir/file", "key", "val")
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
	if !strings.Contains(err.Error(), "failed to open GITHUB_OUTPUT") {
		t.Errorf("unexpected error: %v", err)
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
