package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Provider represents a git hosting provider.
type Provider string

const (
	ProviderGitLab     Provider = "gitlab"
	ProviderGitHub     Provider = "github"
	ProviderBitbucket  Provider = "bitbucket"
	ProviderCodeCommit Provider = "codecommit"
	ProviderGeneric    Provider = "generic"
)

// validProviders is the set of recognized provider names.
var validProviders = map[Provider]bool{
	ProviderGitLab:     true,
	ProviderGitHub:     true,
	ProviderBitbucket:  true,
	ProviderCodeCommit: true,
	ProviderGeneric:    true,
}

// Target represents a single mirror target.
type Target struct {
	Provider Provider
	URL      string
}

// Config holds all configuration for the mirror action.
type Config struct {
	Targets           []Target
	GitLabToken       string
	GitHubToken       string
	BitbucketUsername string
	BitbucketAPIToken string
	SSHPrivateKey     string
	MirrorBranches    []string
	MirrorAllBranches bool
	MirrorTags        bool
	ForcePush         bool
	DryRun            bool
	Debug             bool
	RetryCount        int
	RetryDelay        int // seconds
	ExcludeBranches   []string
	Parallel          bool
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	targetsRaw := os.Getenv("INPUT_TARGETS")
	if targetsRaw == "" {
		return nil, fmt.Errorf("targets input is required")
	}

	targets, err := parseTargets(targetsRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse targets: %w", err)
	}

	branches := os.Getenv("INPUT_MIRROR_BRANCHES")
	mirrorAll := strings.TrimSpace(strings.ToLower(branches)) == "all"

	var branchList []string
	if !mirrorAll && branches != "" {
		for _, b := range strings.Split(branches, ",") {
			if trimmed := strings.TrimSpace(b); trimmed != "" {
				branchList = append(branchList, trimmed)
			}
		}
	}

	var excludeList []string
	if excl := os.Getenv("INPUT_EXCLUDE_BRANCHES"); excl != "" {
		for _, b := range strings.Split(excl, ",") {
			if trimmed := strings.TrimSpace(b); trimmed != "" {
				excludeList = append(excludeList, trimmed)
			}
		}
	}

	cfg := &Config{
		Targets:           targets,
		GitLabToken:       os.Getenv("INPUT_GITLAB_TOKEN"),
		GitHubToken:       os.Getenv("INPUT_GITHUB_TOKEN"),
		BitbucketUsername: os.Getenv("INPUT_BITBUCKET_USERNAME"),
		BitbucketAPIToken: os.Getenv("INPUT_BITBUCKET_API_TOKEN"),
		SSHPrivateKey:     os.Getenv("INPUT_SSH_PRIVATE_KEY"),
		MirrorBranches:    branchList,
		MirrorAllBranches: mirrorAll,
		MirrorTags:        envBool("INPUT_MIRROR_TAGS", true),
		ForcePush:         envBool("INPUT_FORCE_PUSH", true),
		DryRun:            envBool("INPUT_DRY_RUN", false),
		Debug:             envBool("INPUT_DEBUG", false),
		RetryCount:        envInt("INPUT_RETRY_COUNT", 0),
		RetryDelay:        envInt("INPUT_RETRY_DELAY", 5),
		ExcludeBranches:   excludeList,
		Parallel:          envBool("INPUT_PARALLEL", false),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks the configuration for consistency and required values.
func (c *Config) Validate() error {
	if !c.MirrorAllBranches && len(c.MirrorBranches) == 0 {
		return fmt.Errorf("mirror_branches must specify branch names or 'all'")
	}

	for _, t := range c.Targets {
		switch t.Provider {
		case ProviderGitLab:
			if c.GitLabToken == "" && c.SSHPrivateKey == "" {
				logWarning("target %s: no gitlab_token or ssh_private_key provided", t.URL)
			}
		case ProviderGitHub:
			if c.GitHubToken == "" && c.SSHPrivateKey == "" {
				logWarning("target %s: no github_token or ssh_private_key provided", t.URL)
			}
		case ProviderBitbucket:
			if (c.BitbucketUsername == "" || c.BitbucketAPIToken == "") && c.SSHPrivateKey == "" {
				logWarning("target %s: no bitbucket credentials or ssh_private_key provided", t.URL)
			}
		case ProviderCodeCommit:
			logWarning("target %s: ensure AWS credentials and credential-helper are configured", t.URL)
		}
	}

	return nil
}

func logWarning(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "::warning::%s\n", fmt.Sprintf(format, args...))
}

// parseTargets parses the newline-separated targets input.
// Format: "provider::url" or just "url" (auto-detect provider).
func parseTargets(raw string) ([]Target, error) {
	var targets []Target

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var t Target
		if parts := strings.SplitN(line, "::", 2); len(parts) == 2 {
			provider := Provider(strings.ToLower(strings.TrimSpace(parts[0])))
			if !validProviders[provider] {
				return nil, fmt.Errorf("unknown provider %q in target: %q (valid: gitlab, github, bitbucket, codecommit, generic)", provider, line)
			}
			t.Provider = provider
			t.URL = strings.TrimSpace(parts[1])
		} else {
			t.URL = line
			t.Provider = detectProvider(line)
		}

		if t.URL == "" {
			return nil, fmt.Errorf("empty URL in target: %q", line)
		}

		if !strings.HasPrefix(t.URL, "https://") && !strings.HasPrefix(t.URL, "http://") && !strings.HasPrefix(t.URL, "git@") {
			return nil, fmt.Errorf("invalid URL format in target: %q (must start with https://, http://, or git@)", t.URL)
		}

		targets = append(targets, t)
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no valid targets found")
	}

	return targets, nil
}

// detectProvider auto-detects the provider from the URL.
func detectProvider(url string) Provider {
	lower := strings.ToLower(url)
	switch {
	case strings.Contains(lower, "gitlab.com") || strings.Contains(lower, "gitlab"):
		return ProviderGitLab
	case strings.Contains(lower, "github.com") || strings.Contains(lower, "github"):
		return ProviderGitHub
	case strings.Contains(lower, "bitbucket.org") || strings.Contains(lower, "bitbucket"):
		return ProviderBitbucket
	case strings.Contains(lower, "codecommit"):
		return ProviderCodeCommit
	default:
		return ProviderGeneric
	}
}

func envInt(key string, defaultVal int) int {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return n
}

func envBool(key string, defaultVal bool) bool {
	val := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch val {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	default:
		return defaultVal
	}
}
