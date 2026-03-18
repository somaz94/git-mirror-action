package mirror

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/somaz94/git-mirror-action/internal/config"
)

// mockGit returns a gitRunner that always succeeds.
func mockGitOK() gitRunner {
	return func(args ...string) error {
		return nil
	}
}


func TestEnsureGitRepoAlreadyExists(t *testing.T) {
	cfg := &config.Config{}
	m := New(cfg)
	m.gitFn = func(args ...string) error {
		return nil // rev-parse succeeds = already a git repo
	}
	if err := m.ensureGitRepo(); err != nil {
		t.Errorf("expected no error for existing repo, got: %v", err)
	}
}

func TestEnsureGitRepoInitializes(t *testing.T) {
	cfg := &config.Config{Debug: true}
	m := New(cfg)
	calls := []string{}
	m.gitFn = func(args ...string) error {
		calls = append(calls, args[0])
		if args[0] == "rev-parse" {
			return fmt.Errorf("not a git repo")
		}
		return nil
	}
	if err := m.ensureGitRepo(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(calls) < 2 || calls[1] != "init" {
		t.Errorf("expected git init call, got calls: %v", calls)
	}
}

func TestEnsureGitRepoInitFails(t *testing.T) {
	cfg := &config.Config{}
	m := New(cfg)
	m.gitFn = func(args ...string) error {
		return fmt.Errorf("fail")
	}
	if err := m.ensureGitRepo(); err == nil {
		t.Error("expected error when init fails")
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
			cfg:  &config.Config{BitbucketUsername: "user", BitbucketAPIToken: "pass"},
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

func TestMirrorToDryRunPreCheckFails(t *testing.T) {
	cfg := &config.Config{
		DryRun: true,
	}
	m := New(cfg)
	m.gitFn = func(args ...string) error {
		if len(args) > 0 && args[0] == "ls-remote" {
			return fmt.Errorf("connection refused")
		}
		return nil
	}

	target := config.Target{
		Provider: config.ProviderGeneric,
		URL:      "https://example.com/repo.git",
	}

	result := m.mirrorTo(target)

	if result.Success {
		t.Error("expected failure when pre-check fails")
	}
	if !strings.Contains(result.Message, "pre-check failed") {
		t.Errorf("expected pre-check error, got: %s", result.Message)
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

func TestMaskSecrets(t *testing.T) {
	cfg := &config.Config{
		GitLabToken: "my-secret-token",
		GitHubToken: "gh-token-123",
	}
	m := New(cfg)

	masked := m.maskSecrets("git remote add mirror https://oauth2:my-secret-token@gitlab.com/repo.git")
	if strings.Contains(masked, "my-secret-token") {
		t.Errorf("expected token to be masked, got: %s", masked)
	}
	if !strings.Contains(masked, "***") {
		t.Errorf("expected *** in masked output, got: %s", masked)
	}
}

func TestMaskSecretsNoSecrets(t *testing.T) {
	cfg := &config.Config{}
	m := New(cfg)

	input := "git push --all remote"
	masked := m.maskSecrets(input)
	if masked != input {
		t.Errorf("expected unchanged string, got: %s", masked)
	}
}

func TestCollectSecrets(t *testing.T) {
	cfg := &config.Config{
		GitLabToken:       "gl-tok",
		GitHubToken:       "",
		BitbucketUsername: "bb-user",
		BitbucketAPIToken: "bb-pass",
		SSHPrivateKey:     "ssh-key",
	}
	secrets := collectSecrets(cfg)

	if len(secrets) != 4 {
		t.Fatalf("expected 4 secrets, got %d", len(secrets))
	}
}

func TestCollectSecretsEmpty(t *testing.T) {
	cfg := &config.Config{}
	secrets := collectSecrets(cfg)

	if len(secrets) != 0 {
		t.Fatalf("expected 0 secrets, got %d", len(secrets))
	}
}

func TestMaskSecretsInStderr(t *testing.T) {
	cfg := &config.Config{
		BitbucketUsername: "myuser",
		BitbucketAPIToken: "mypass",
	}
	m := New(cfg)

	masked := m.maskSecrets("https://myuser:mypass@bitbucket.org/org/repo.git")
	if strings.Contains(masked, "myuser") {
		t.Errorf("expected username to be masked, got: %s", masked)
	}
	if strings.Contains(masked, "mypass") {
		t.Errorf("expected password to be masked, got: %s", masked)
	}
}

func TestInjectTokenAuthSpecialChars(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		user     string
		pass     string
		contains string
	}{
		{
			name:     "password with @",
			url:      "https://gitlab.com/org/repo.git",
			user:     "oauth2",
			pass:     "pass@word",
			contains: "pass%40word",
		},
		{
			name:     "password with :",
			url:      "https://gitlab.com/org/repo.git",
			user:     "oauth2",
			pass:     "pass:word",
			contains: "pass%3Aword",
		},
		{
			name:     "password with /",
			url:      "https://gitlab.com/org/repo.git",
			user:     "oauth2",
			pass:     "pass/word",
			contains: "pass%2Fword",
		},
		{
			name:     "username with special chars",
			url:      "https://bitbucket.org/org/repo.git",
			user:     "user@domain.com",
			pass:     "token",
			contains: "user%40domain.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := injectTokenAuth(tt.url, tt.user, tt.pass)
			if !strings.Contains(got, tt.contains) {
				t.Errorf("expected URL to contain %q, got: %s", tt.contains, got)
			}
		})
	}
}

func TestMirrorToRemoteCleanupOnFailure(t *testing.T) {
	// Verify that remote is cleaned up even when push fails
	var removeCalls int
	callCount := 0
	cfg := &config.Config{
		MirrorAllBranches: true,
	}
	m := New(cfg)
	m.gitFn = func(args ...string) error {
		callCount++
		if len(args) >= 2 && args[0] == "remote" && args[1] == "remove" {
			removeCalls++
		}
		// 1: remote remove (initial), 2: remote add, 3: push (fail), 4: remote remove (defer)
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
		t.Error("expected failure")
	}
	// Should have 2 remote remove calls: initial + cleanup defer
	if removeCalls != 2 {
		t.Errorf("expected 2 remote remove calls (init + defer cleanup), got %d", removeCalls)
	}
}

func TestWithRetrySuccess(t *testing.T) {
	cfg := &config.Config{RetryCount: 2, RetryDelay: 0}
	m := New(cfg)

	callCount := 0
	err := m.withRetry("test", func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestWithRetryEventualSuccess(t *testing.T) {
	cfg := &config.Config{RetryCount: 3, RetryDelay: 0}
	m := New(cfg)

	callCount := 0
	err := m.withRetry("test", func() error {
		callCount++
		if callCount < 3 {
			return fmt.Errorf("transient error")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestWithRetryAllFail(t *testing.T) {
	cfg := &config.Config{RetryCount: 2, RetryDelay: 0}
	m := New(cfg)

	callCount := 0
	err := m.withRetry("test", func() error {
		callCount++
		return fmt.Errorf("permanent error")
	})

	if err == nil {
		t.Fatal("expected error")
	}
	// 1 initial + 2 retries = 3
	if callCount != 3 {
		t.Errorf("expected 3 calls (1 + 2 retries), got %d", callCount)
	}
}

func TestWithRetryNoRetry(t *testing.T) {
	cfg := &config.Config{RetryCount: 0, RetryDelay: 0}
	m := New(cfg)

	callCount := 0
	err := m.withRetry("test", func() error {
		callCount++
		return fmt.Errorf("fail")
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retry), got %d", callCount)
	}
}

func TestExcludeBranchesSpecific(t *testing.T) {
	var pushedBranches []string
	cfg := &config.Config{
		MirrorAllBranches: false,
		MirrorBranches:    []string{"main", "develop", "staging"},
		ExcludeBranches:   []string{"develop"},
		Debug:             true,
	}
	m := New(cfg)
	m.gitFn = func(args ...string) error {
		if len(args) > 0 && args[0] == "push" {
			// Extract branch name from refspec
			for _, a := range args {
				if strings.HasPrefix(a, "refs/heads/") {
					parts := strings.Split(a, ":")
					branch := strings.TrimPrefix(parts[0], "refs/heads/")
					pushedBranches = append(pushedBranches, branch)
				}
			}
		}
		return nil
	}

	err := m.pushBranches("test-remote")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pushedBranches) != 2 {
		t.Fatalf("expected 2 branches pushed, got %d: %v", len(pushedBranches), pushedBranches)
	}
	for _, b := range pushedBranches {
		if b == "develop" {
			t.Error("excluded branch 'develop' should not be pushed")
		}
	}
}

func TestExcludeBranchesAll(t *testing.T) {
	var deletedBranches []string
	cfg := &config.Config{
		MirrorAllBranches: true,
		ExcludeBranches:   []string{"staging", "hotfix"},
	}
	m := New(cfg)
	m.gitFn = func(args ...string) error {
		if len(args) >= 3 && args[0] == "push" && args[2] == "--delete" {
			deletedBranches = append(deletedBranches, args[3])
		}
		return nil
	}

	err := m.pushBranches("test-remote")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deletedBranches) != 2 {
		t.Fatalf("expected 2 delete calls, got %d: %v", len(deletedBranches), deletedBranches)
	}
}

func TestRunParallel(t *testing.T) {
	cfg := &config.Config{
		MirrorAllBranches: true,
		MirrorTags:        true,
		Parallel:          true,
		Targets: []config.Target{
			{Provider: config.ProviderGitLab, URL: "https://gitlab.com/org/repo1.git"},
			{Provider: config.ProviderGitHub, URL: "https://github.com/org/repo2.git"},
			{Provider: config.ProviderGeneric, URL: "https://example.com/repo3.git"},
		},
	}
	m := New(cfg)
	m.gitFn = mockGitOK()

	results := m.Run()

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for i, r := range results {
		if !r.Success {
			t.Errorf("result[%d]: expected success, got failure: %s", i, r.Message)
		}
	}
}

func TestRunParallelWithFailure(t *testing.T) {
	cfg := &config.Config{
		MirrorAllBranches: true,
		Parallel:          true,
		Targets: []config.Target{
			{Provider: config.ProviderGeneric, URL: "https://example.com/repo1.git"},
			{Provider: config.ProviderGeneric, URL: "https://example.com/repo2.git"},
		},
	}
	m := New(cfg)
	var mu sync.Mutex
	callMap := make(map[string]int)
	m.gitFn = func(args ...string) error {
		if len(args) >= 2 && args[0] == "remote" && args[1] == "add" {
			remoteName := args[2]
			mu.Lock()
			callMap[remoteName]++
			mu.Unlock()
		}
		// Fail push for all
		if len(args) > 0 && args[0] == "push" {
			return fmt.Errorf("push failed")
		}
		return nil
	}

	results := m.Run()

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// At least one should fail
	failCount := 0
	for _, r := range results {
		if !r.Success {
			failCount++
		}
	}
	if failCount == 0 {
		t.Error("expected at least one failure")
	}
}

func TestSanitizeRemoteName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://gitlab.com/org/repo.git", "gitlab-com-org-repo-git"},
		{"git@github.com:org/repo.git", "github-com-org-repo-git"},
	}
	for _, tt := range tests {
		got := sanitizeRemoteName(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizeRemoteName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestIsExcluded(t *testing.T) {
	cfg := &config.Config{
		ExcludeBranches: []string{"staging", "hotfix"},
	}
	m := New(cfg)

	if !m.isExcluded("staging") {
		t.Error("expected staging to be excluded")
	}
	if !m.isExcluded("hotfix") {
		t.Error("expected hotfix to be excluded")
	}
	if m.isExcluded("main") {
		t.Error("expected main not to be excluded")
	}
}

func TestRunSingleTargetNotParallel(t *testing.T) {
	// Parallel with single target should still work (falls through to sequential)
	cfg := &config.Config{
		MirrorAllBranches: true,
		Parallel:          true,
		Targets: []config.Target{
			{Provider: config.ProviderGeneric, URL: "https://example.com/repo.git"},
		},
	}
	m := New(cfg)
	m.gitFn = mockGitOK()

	results := m.Run()

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Success {
		t.Errorf("expected success, got: %s", results[0].Message)
	}
}

func TestMirrorToWithRetry(t *testing.T) {
	pushCount := 0
	cfg := &config.Config{
		MirrorAllBranches: true,
		MirrorTags:        true,
		RetryCount:        2,
		RetryDelay:        0,
	}
	m := New(cfg)
	m.gitFn = func(args ...string) error {
		if len(args) > 0 && args[0] == "push" {
			pushCount++
			// Fail first push attempt, succeed second
			if pushCount == 1 {
				return fmt.Errorf("transient error")
			}
		}
		return nil
	}

	target := config.Target{
		Provider: config.ProviderGeneric,
		URL:      "https://example.com/repo.git",
	}

	result := m.mirrorTo(target)

	if !result.Success {
		t.Errorf("expected success after retry, got: %s", result.Message)
	}
}
