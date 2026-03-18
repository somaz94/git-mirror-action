package mirror

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/somaz94/git-mirror-action/internal/config"
)

// mockGit returns a gitRunner that always succeeds.
func mockGitOK() gitRunner {
	return func(args ...string) error {
		return nil
	}
}

// mockGitFail returns a gitRunner that always fails.
func mockGitFail(msg string) gitRunner {
	return func(args ...string) error {
		return fmt.Errorf("%s", msg)
	}
}

// mockGitFailOn returns a gitRunner that fails when the first arg matches.
func mockGitFailOn(failCmd string) gitRunner {
	return func(args ...string) error {
		if len(args) > 0 && args[0] == failCmd {
			return fmt.Errorf("mock %s failure", failCmd)
		}
		return nil
	}
}

func TestInjectTokenAuth(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		user     string
		pass     string
		expected string
	}{
		{
			name:     "https url",
			url:      "https://gitlab.com/org/repo.git",
			user:     "oauth2",
			pass:     "my-token",
			expected: "https://oauth2:my-token@gitlab.com/org/repo.git",
		},
		{
			name:     "ssh url unchanged",
			url:      "git@github.com:org/repo.git",
			user:     "x-access-token",
			pass:     "token",
			expected: "git@github.com:org/repo.git",
		},
		{
			name:     "http url unchanged",
			url:      "http://example.com/repo.git",
			user:     "user",
			pass:     "pass",
			expected: "http://example.com/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := injectTokenAuth(tt.url, tt.user, tt.pass)
			if got != tt.expected {
				t.Errorf("injectTokenAuth(%q, %q, %q) = %q, want %q", tt.url, tt.user, tt.pass, got, tt.expected)
			}
		})
	}
}

func TestBuildAuthURL(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		target   config.Target
		expected string
	}{
		{
			name: "gitlab with token",
			cfg:  &config.Config{GitLabToken: "gl-token"},
			target: config.Target{
				Provider: config.ProviderGitLab,
				URL:      "https://gitlab.com/org/repo.git",
			},
			expected: "https://oauth2:gl-token@gitlab.com/org/repo.git",
		},
		{
			name: "github with token",
			cfg:  &config.Config{GitHubToken: "gh-token"},
			target: config.Target{
				Provider: config.ProviderGitHub,
				URL:      "https://github.com/org/repo.git",
			},
			expected: "https://x-access-token:gh-token@github.com/org/repo.git",
		},
		{
			name: "bitbucket with credentials",
			cfg:  &config.Config{BitbucketUsername: "user", BitbucketPassword: "pass"},
			target: config.Target{
				Provider: config.ProviderBitbucket,
				URL:      "https://bitbucket.org/org/repo.git",
			},
			expected: "https://user:pass@bitbucket.org/org/repo.git",
		},
		{
			name: "codecommit uses url as-is",
			cfg:  &config.Config{},
			target: config.Target{
				Provider: config.ProviderCodeCommit,
				URL:      "https://git-codecommit.us-east-1.amazonaws.com/v1/repos/repo",
			},
			expected: "https://git-codecommit.us-east-1.amazonaws.com/v1/repos/repo",
		},
		{
			name: "generic uses url as-is",
			cfg:  &config.Config{},
			target: config.Target{
				Provider: config.ProviderGeneric,
				URL:      "https://custom-git.example.com/repo.git",
			},
			expected: "https://custom-git.example.com/repo.git",
		},
		{
			name: "gitlab without token",
			cfg:  &config.Config{},
			target: config.Target{
				Provider: config.ProviderGitLab,
				URL:      "https://gitlab.com/org/repo.git",
			},
			expected: "https://gitlab.com/org/repo.git",
		},
		{
			name: "bitbucket missing password",
			cfg:  &config.Config{BitbucketUsername: "user"},
			target: config.Target{
				Provider: config.ProviderBitbucket,
				URL:      "https://bitbucket.org/org/repo.git",
			},
			expected: "https://bitbucket.org/org/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.cfg)
			got, err := m.buildAuthURL(tt.target)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("buildAuthURL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewMirror(t *testing.T) {
	cfg := &config.Config{Debug: true}
	m := New(cfg)
	if m.cfg != cfg {
		t.Error("expected mirror to hold the provided config")
	}
	if m.gitFn == nil {
		t.Error("expected gitFn to be set")
	}
}

func TestLogInfo(t *testing.T) {
	cfg := &config.Config{}
	m := New(cfg)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	m.logInfo("test %s", "message")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "::notice::test message") {
		t.Errorf("expected notice output, got: %s", output)
	}
}

func TestLogError(t *testing.T) {
	cfg := &config.Config{}
	m := New(cfg)

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	m.logError("err %s", "msg")

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "::error::err msg") {
		t.Errorf("expected error output, got: %s", output)
	}
}

func TestLogDebugEnabled(t *testing.T) {
	cfg := &config.Config{Debug: true}
	m := New(cfg)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	m.logDebug("debug %s", "info")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "::debug::debug info") {
		t.Errorf("expected debug output, got: %s", output)
	}
}

func TestLogDebugDisabled(t *testing.T) {
	cfg := &config.Config{Debug: false}
	m := New(cfg)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	m.logDebug("should not appear")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output != "" {
		t.Errorf("expected no output when debug disabled, got: %s", output)
	}
}

func TestMirrorToDryRun(t *testing.T) {
	cfg := &config.Config{
		DryRun:      true,
		GitLabToken: "test-token",
	}
	m := New(cfg)
	m.gitFn = mockGitOK()

	target := config.Target{
		Provider: config.ProviderGitLab,
		URL:      "https://gitlab.com/org/repo.git",
	}

	result := m.mirrorTo(target)

	if !result.Success {
		t.Errorf("expected success for dry run, got failure: %s", result.Message)
	}
	if !strings.Contains(result.Message, "dry run") {
		t.Errorf("expected dry run message, got: %s", result.Message)
	}
}

func TestMirrorToSuccess(t *testing.T) {
	cfg := &config.Config{
		GitLabToken:       "test-token",
		MirrorAllBranches: true,
		MirrorTags:        true,
		ForcePush:         true,
	}
	m := New(cfg)
	m.gitFn = mockGitOK()

	target := config.Target{
		Provider: config.ProviderGitLab,
		URL:      "https://gitlab.com/org/repo.git",
	}

	result := m.mirrorTo(target)

	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.Message)
	}
	if result.Message != "mirrored successfully" {
		t.Errorf("expected 'mirrored successfully', got: %s", result.Message)
	}
}

func TestMirrorToSuccessNoTags(t *testing.T) {
	cfg := &config.Config{
		MirrorAllBranches: true,
		MirrorTags:        false,
	}
	m := New(cfg)
	m.gitFn = mockGitOK()

	target := config.Target{
		Provider: config.ProviderGeneric,
		URL:      "https://example.com/repo.git",
	}

	result := m.mirrorTo(target)

	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.Message)
	}
}

func TestMirrorToAddRemoteFails(t *testing.T) {
	callCount := 0
	cfg := &config.Config{}
	m := New(cfg)
	m.gitFn = func(args ...string) error {
		callCount++
		// First call is "remote remove" (ignored), second is "remote add" (fail)
		if callCount == 2 {
			return fmt.Errorf("remote add failed")
		}
		return nil
	}

	target := config.Target{
		Provider: config.ProviderGeneric,
		URL:      "https://example.com/repo.git",
	}

	result := m.mirrorTo(target)

	if result.Success {
		t.Error("expected failure when remote add fails")
	}
	if !strings.Contains(result.Message, "failed to add remote") {
		t.Errorf("expected 'failed to add remote' message, got: %s", result.Message)
	}
}

func TestMirrorToPushBranchesFails(t *testing.T) {
	callCount := 0
	cfg := &config.Config{
		MirrorAllBranches: true,
	}
	m := New(cfg)
	m.gitFn = func(args ...string) error {
		callCount++
		// 1: remote remove, 2: remote add, 3: push --all (fail)
		if callCount == 3 {
			return fmt.Errorf("push failed")
		}
		return nil
	}

	target := config.Target{
		Provider: config.ProviderGeneric,
		URL:      "https://example.com/repo.git",
	}

	result := m.mirrorTo(target)

	if result.Success {
		t.Error("expected failure when push branches fails")
	}
	if !strings.Contains(result.Message, "failed to push branches") {
		t.Errorf("expected push branches error, got: %s", result.Message)
	}
}

func TestMirrorToPushTagsFails(t *testing.T) {
	callCount := 0
	cfg := &config.Config{
		MirrorAllBranches: true,
		MirrorTags:        true,
	}
	m := New(cfg)
	m.gitFn = func(args ...string) error {
		callCount++
		// 1: remote remove, 2: remote add, 3: push --all (ok), 4: push --tags (fail)
		if callCount == 4 {
			return fmt.Errorf("tags push failed")
		}
		return nil
	}

	target := config.Target{
		Provider: config.ProviderGeneric,
		URL:      "https://example.com/repo.git",
	}

	result := m.mirrorTo(target)

	if result.Success {
		t.Error("expected failure when push tags fails")
	}
	if !strings.Contains(result.Message, "failed to push tags") {
		t.Errorf("expected push tags error, got: %s", result.Message)
	}
}

func TestRunWithMockSuccess(t *testing.T) {
	cfg := &config.Config{
		MirrorAllBranches: true,
		MirrorTags:        true,
		Targets: []config.Target{
			{Provider: config.ProviderGitLab, URL: "https://gitlab.com/org/repo.git"},
			{Provider: config.ProviderGeneric, URL: "https://example.com/repo.git"},
		},
	}
	m := New(cfg)
	m.gitFn = mockGitOK()

	results := m.Run()

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for i, r := range results {
		if !r.Success {
			t.Errorf("result[%d]: expected success, got failure: %s", i, r.Message)
		}
	}
}

func TestRunWithMockFailure(t *testing.T) {
	callCount := 0
	cfg := &config.Config{
		MirrorAllBranches: true,
		Targets: []config.Target{
			{Provider: config.ProviderGeneric, URL: "https://example.com/repo.git"},
		},
	}
	m := New(cfg)
	m.gitFn = func(args ...string) error {
		callCount++
		// 1: remote remove, 2: remote add, 3: push (fail)
		if callCount == 3 {
			return fmt.Errorf("push failed")
		}
		return nil
	}

	results := m.Run()

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Success {
		t.Error("expected failure")
	}
}

func TestPushBranchesSpecificWithMock(t *testing.T) {
	cfg := &config.Config{
		MirrorAllBranches: false,
		MirrorBranches:    []string{"main", "develop"},
		ForcePush:         true,
	}
	m := New(cfg)
	m.gitFn = mockGitOK()

	err := m.pushBranches("test-remote")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPushBranchesSpecificFailsOnSecond(t *testing.T) {
	pushCount := 0
	cfg := &config.Config{
		MirrorAllBranches: false,
		MirrorBranches:    []string{"main", "develop"},
		ForcePush:         false,
	}
	m := New(cfg)
	m.gitFn = func(args ...string) error {
		if len(args) > 0 && args[0] == "push" {
			pushCount++
			if pushCount == 2 {
				return fmt.Errorf("push develop failed")
			}
		}
		return nil
	}

	err := m.pushBranches("test-remote")
	if err == nil {
		t.Error("expected error on second branch push")
	}
	if !strings.Contains(err.Error(), "branch develop") {
		t.Errorf("expected develop branch error, got: %v", err)
	}
}

func TestPushBranchesAllWithMock(t *testing.T) {
	cfg := &config.Config{
		MirrorAllBranches: true,
		ForcePush:         true,
	}
	m := New(cfg)
	m.gitFn = mockGitOK()

	err := m.pushBranches("test-remote")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPushTagsWithMock(t *testing.T) {
	cfg := &config.Config{ForcePush: true}
	m := New(cfg)
	m.gitFn = mockGitOK()

	err := m.pushTags("test-remote")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPushTagsNoForceWithMock(t *testing.T) {
	cfg := &config.Config{ForcePush: false}
	m := New(cfg)
	m.gitFn = mockGitOK()

	err := m.pushTags("test-remote")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExecGit(t *testing.T) {
	cfg := &config.Config{}
	m := New(cfg)

	// execGit should work with a valid git command
	err := m.execGit("version")
	if err != nil {
		t.Errorf("expected git version to succeed: %v", err)
	}

	// execGit should fail with an invalid command
	err = m.execGit("invalid-command-that-does-not-exist")
	if err == nil {
		t.Error("expected error for invalid git command")
	}
}
