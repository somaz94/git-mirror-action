package mirror

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/somaz94/git-mirror-action/internal/config"
)

// Result represents the outcome of mirroring to a single target.
type Result struct {
	Target  config.Target `json:"target"`
	Success bool          `json:"success"`
	Message string        `json:"message"`
}

// gitRunner is a function that executes a git command.
type gitRunner func(args ...string) error

// Mirror handles repository mirroring operations.
type Mirror struct {
	cfg    *config.Config
	gitFn  gitRunner
}

// New creates a new Mirror instance.
func New(cfg *config.Config) *Mirror {
	m := &Mirror{cfg: cfg}
	m.gitFn = m.execGit
	return m
}

// Run executes mirroring to all configured targets.
func (m *Mirror) Run() []Result {
	var results []Result

	for _, target := range m.cfg.Targets {
		m.logInfo("Mirroring to %s (%s)...", target.URL, target.Provider)

		result := m.mirrorTo(target)
		results = append(results, result)

		if result.Success {
			m.logInfo("Successfully mirrored to %s", target.URL)
		} else {
			m.logError("Failed to mirror to %s: %s", target.URL, result.Message)
		}
	}

	return results
}

func (m *Mirror) mirrorTo(target config.Target) Result {
	authURL, err := m.buildAuthURL(target)
	if err != nil {
		return Result{Target: target, Success: false, Message: err.Error()}
	}

	remoteName := fmt.Sprintf("mirror-%s", target.Provider)

	// Remove remote if it already exists
	_ = m.git("remote", "remove", remoteName)

	// Add the mirror remote
	if err := m.git("remote", "add", remoteName, authURL); err != nil {
		return Result{Target: target, Success: false, Message: fmt.Sprintf("failed to add remote: %v", err)}
	}

	if m.cfg.DryRun {
		m.logInfo("[DRY RUN] Would push to %s", target.URL)
		return Result{Target: target, Success: true, Message: "dry run - skipped"}
	}

	// Push branches
	if err := m.pushBranches(remoteName); err != nil {
		return Result{Target: target, Success: false, Message: fmt.Sprintf("failed to push branches: %v", err)}
	}

	// Push tags
	if m.cfg.MirrorTags {
		if err := m.pushTags(remoteName); err != nil {
			return Result{Target: target, Success: false, Message: fmt.Sprintf("failed to push tags: %v", err)}
		}
	}

	// Clean up remote
	_ = m.git("remote", "remove", remoteName)

	return Result{Target: target, Success: true, Message: "mirrored successfully"}
}

func (m *Mirror) pushBranches(remote string) error {
	args := []string{"push"}
	if m.cfg.ForcePush {
		args = append(args, "-f")
	}

	if m.cfg.MirrorAllBranches {
		args = append(args, "--all", remote)
	} else {
		for _, branch := range m.cfg.MirrorBranches {
			branchArgs := append(args, remote, fmt.Sprintf("refs/heads/%s:refs/heads/%s", branch, branch))
			if err := m.git(branchArgs...); err != nil {
				return fmt.Errorf("branch %s: %w", branch, err)
			}
		}
		return nil
	}

	return m.git(args...)
}

func (m *Mirror) pushTags(remote string) error {
	args := []string{"push"}
	if m.cfg.ForcePush {
		args = append(args, "-f")
	}
	args = append(args, "--tags", remote)
	return m.git(args...)
}

func (m *Mirror) buildAuthURL(target config.Target) (string, error) {
	url := target.URL

	switch target.Provider {
	case config.ProviderGitLab:
		if m.cfg.GitLabToken != "" {
			url = injectTokenAuth(url, "oauth2", m.cfg.GitLabToken)
		}
	case config.ProviderGitHub:
		if m.cfg.GitHubToken != "" {
			url = injectTokenAuth(url, "x-access-token", m.cfg.GitHubToken)
		}
	case config.ProviderBitbucket:
		if m.cfg.BitbucketUsername != "" && m.cfg.BitbucketPassword != "" {
			url = injectTokenAuth(url, m.cfg.BitbucketUsername, m.cfg.BitbucketPassword)
		}
	case config.ProviderCodeCommit:
		// CodeCommit uses credential-helper or IAM, URL is used as-is
	case config.ProviderGeneric:
		// Use URL as-is, or SSH key will be configured separately
	}

	return url, nil
}

// injectTokenAuth injects username:password into an HTTPS URL.
func injectTokenAuth(url, username, password string) string {
	if strings.HasPrefix(url, "https://") {
		return fmt.Sprintf("https://%s:%s@%s", username, password, strings.TrimPrefix(url, "https://"))
	}
	return url
}

func (m *Mirror) git(args ...string) error {
	return m.gitFn(args...)
}

func (m *Mirror) execGit(args ...string) error {
	m.logDebug("git %s", strings.Join(args, " "))
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m *Mirror) logInfo(format string, args ...interface{}) {
	fmt.Printf("::notice::%s\n", fmt.Sprintf(format, args...))
}

func (m *Mirror) logError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "::error::%s\n", fmt.Sprintf(format, args...))
}

func (m *Mirror) logDebug(format string, args ...interface{}) {
	if m.cfg.Debug {
		fmt.Printf("::debug::%s\n", fmt.Sprintf(format, args...))
	}
}
