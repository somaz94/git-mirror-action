package config

import (
	"os"
	"testing"
)

func TestLoadRequiresTargets(t *testing.T) {
	os.Unsetenv("INPUT_TARGETS")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when targets is empty")
	}
}

func TestLoadValidConfig(t *testing.T) {
	t.Setenv("INPUT_TARGETS", "gitlab::https://gitlab.com/org/repo.git")
	t.Setenv("INPUT_GITLAB_TOKEN", "test-token")
	t.Setenv("INPUT_MIRROR_BRANCHES", "main,develop")
	t.Setenv("INPUT_MIRROR_TAGS", "true")
	t.Setenv("INPUT_FORCE_PUSH", "true")
	t.Setenv("INPUT_DRY_RUN", "false")
	t.Setenv("INPUT_DEBUG", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(cfg.Targets))
	}
	if cfg.Targets[0].Provider != ProviderGitLab {
		t.Errorf("expected gitlab provider, got %s", cfg.Targets[0].Provider)
	}
	if cfg.GitLabToken != "test-token" {
		t.Errorf("expected test-token, got %s", cfg.GitLabToken)
	}
	if cfg.MirrorAllBranches {
		t.Error("expected MirrorAllBranches to be false")
	}
	if len(cfg.MirrorBranches) != 2 {
		t.Errorf("expected 2 branches, got %d", len(cfg.MirrorBranches))
	}
	if !cfg.Debug {
		t.Error("expected debug to be true")
	}
}

func TestLoadAllBranches(t *testing.T) {
	t.Setenv("INPUT_TARGETS", "https://gitlab.com/org/repo.git")
	t.Setenv("INPUT_MIRROR_BRANCHES", "all")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.MirrorAllBranches {
		t.Error("expected MirrorAllBranches to be true")
	}
}

func TestParseTargetsMultiple(t *testing.T) {
	raw := `gitlab::https://gitlab.com/org/repo.git
codecommit::https://git-codecommit.us-east-1.amazonaws.com/v1/repos/repo
https://bitbucket.org/org/repo.git`

	targets, err := parseTargets(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 3 {
		t.Fatalf("expected 3 targets, got %d", len(targets))
	}
	if targets[0].Provider != ProviderGitLab {
		t.Errorf("target[0]: expected gitlab, got %s", targets[0].Provider)
	}
	if targets[1].Provider != ProviderCodeCommit {
		t.Errorf("target[1]: expected codecommit, got %s", targets[1].Provider)
	}
	if targets[2].Provider != ProviderBitbucket {
		t.Errorf("target[2]: expected bitbucket, got %s", targets[2].Provider)
	}
}

func TestParseTargetsEmpty(t *testing.T) {
	_, err := parseTargets("")
	if err == nil {
		t.Fatal("expected error for empty targets")
	}
}

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		url      string
		expected Provider
	}{
		{"https://gitlab.com/org/repo.git", ProviderGitLab},
		{"https://github.com/org/repo.git", ProviderGitHub},
		{"https://bitbucket.org/org/repo.git", ProviderBitbucket},
		{"https://git-codecommit.us-east-1.amazonaws.com/v1/repos/repo", ProviderCodeCommit},
		{"https://custom-git.example.com/repo.git", ProviderGeneric},
	}

	for _, tt := range tests {
		got := detectProvider(tt.url)
		if got != tt.expected {
			t.Errorf("detectProvider(%q) = %s, want %s", tt.url, got, tt.expected)
		}
	}
}

func TestEnvBool(t *testing.T) {
	t.Setenv("TEST_BOOL_TRUE", "true")
	t.Setenv("TEST_BOOL_FALSE", "false")
	t.Setenv("TEST_BOOL_YES", "yes")

	if !envBool("TEST_BOOL_TRUE", false) {
		t.Error("expected true")
	}
	if envBool("TEST_BOOL_FALSE", true) {
		t.Error("expected false")
	}
	if !envBool("TEST_BOOL_YES", false) {
		t.Error("expected true for 'yes'")
	}
	if !envBool("NONEXISTENT", true) {
		t.Error("expected default true")
	}
}
