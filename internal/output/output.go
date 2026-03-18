package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/somaz94/multi-git-mirror/internal/mirror"
)

// Write writes mirror results to GitHub Actions outputs.
func Write(results []mirror.Result) error {
	outputFile := os.Getenv("GITHUB_OUTPUT")

	var mirrored, failed int
	for _, r := range results {
		if r.Success {
			mirrored++
		} else {
			failed++
		}
	}

	resultJSON, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if outputFile != "" {
		if err := writeGitHubOutput(outputFile, "result", string(resultJSON)); err != nil {
			return err
		}
		if err := writeGitHubOutput(outputFile, "mirrored_count", fmt.Sprintf("%d", mirrored)); err != nil {
			return err
		}
		if err := writeGitHubOutput(outputFile, "failed_count", fmt.Sprintf("%d", failed)); err != nil {
			return err
		}
	}

	// Always print summary to stdout
	fmt.Printf("::notice::Mirror complete: %d succeeded, %d failed\n", mirrored, failed)

	if failed > 0 {
		return fmt.Errorf("%d mirror target(s) failed", failed)
	}

	return nil
}

func writeGitHubOutput(path, key, value string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open GITHUB_OUTPUT: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s=%s\n", key, value)
	if err != nil {
		return fmt.Errorf("failed to write output %s: %w", key, err)
	}

	return nil
}
