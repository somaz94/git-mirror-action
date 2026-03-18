package mirror

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/somaz94/multi-git-mirror/internal/config"
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
	cfg       *config.Config
	gitFn     gitRunner
	secrets   []string   // values to mask in debug logs
	sshDir    string     // directory for SSH key files
	configMu  sync.Mutex // protects git config operations (remote add/remove)
}

// New creates a new Mirror instance.
func New(cfg *config.Config) *Mirror {
	m := &Mirror{cfg: cfg, sshDir: defaultSSHDir}
	m.gitFn = m.execGit
	m.secrets = collectSecrets(cfg)
	return m
}

// collectSecrets gathers all sensitive values for log masking.
func collectSecrets(cfg *config.Config) []string {
	var secrets []string
	for _, s := range []string{
		cfg.GitLabToken,
		cfg.GitHubToken,
		cfg.BitbucketUsername,
		cfg.BitbucketAPIToken,
		cfg.SSHPrivateKey,
	} {
		if s != "" {
			secrets = append(secrets, s)
		}
	}
	return secrets
}

// ensureGitRepo initializes a temporary git repo if none exists (needed for dry-run in Docker).
func (m *Mirror) ensureGitRepo() error {
	if err := m.gitFn("rev-parse", "--git-dir"); err != nil {
		m.logDebug("No git repository found, initializing temporary repo for dry-run")
		if err := m.gitFn("init", "-b", "main"); err != nil {
			return fmt.Errorf("failed to initialize temporary git repo: %w", err)
		}
	}
	return nil
}

// Run executes mirroring to all configured targets.
func (m *Mirror) Run() []Result {
	// Ensure we are inside a git repository
	if err := m.ensureGitRepo(); err != nil {
		return []Result{{
			Target:  config.Target{},
			Success: false,
			Message: err.Error(),
		}}
	}

	// Setup SSH if configured
	if err := m.setupSSH(); err != nil {
		return []Result{{
			Target:  config.Target{},
			Success: false,
			Message: fmt.Sprintf("SSH setup failed: %v", err),
		}}
	}
	defer m.cleanupSSH()

	if m.cfg.Parallel && len(m.cfg.Targets) > 1 {
		return m.runParallel()
	}
	return m.runSequential()
}

func (m *Mirror) runSequential() []Result {
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

func (m *Mirror) runParallel() []Result {
	results := make([]Result, len(m.cfg.Targets))
	var wg sync.WaitGroup

	for i, target := range m.cfg.Targets {
		wg.Add(1)
		go func(idx int, t config.Target) {
			defer wg.Done()
			m.logInfo("Mirroring to %s (%s)...", t.URL, t.Provider)

			result := m.mirrorTo(t)
			results[idx] = result

			if result.Success {
				m.logInfo("Successfully mirrored to %s", t.URL)
			} else {
				m.logError("Failed to mirror to %s: %s", t.URL, result.Message)
			}
		}(i, target)
	}

	wg.Wait()
	return results
}

func (m *Mirror) mirrorTo(target config.Target) Result {
	authURL, err := m.buildAuthURL(target)
	if err != nil {
		return Result{Target: target, Success: false, Message: err.Error()}
	}

	remoteName := fmt.Sprintf("mirror-%s-%s", target.Provider, sanitizeRemoteName(target.URL))

	// Lock to prevent concurrent .git/config modifications
	m.configMu.Lock()
	_ = m.git("remote", "remove", remoteName)
	addErr := m.git("remote", "add", remoteName, authURL)
	m.configMu.Unlock()

	if addErr != nil {
		return Result{Target: target, Success: false, Message: fmt.Sprintf("failed to add remote: %v", addErr)}
	}

	// Clean up remote on exit
	defer func() {
		m.configMu.Lock()
		_ = m.git("remote", "remove", remoteName)
		m.configMu.Unlock()
	}()

	if m.cfg.DryRun {
		// Pre-check: verify remote connectivity
		if err := m.git("ls-remote", "--exit-code", remoteName); err != nil {
			m.logError("[DRY RUN] Pre-check failed for %s: %v", target.URL, err)
			return Result{Target: target, Success: false, Message: fmt.Sprintf("pre-check failed: %v", err)}
		}
		m.logInfo("[DRY RUN] Pre-check passed for %s", target.URL)
		return Result{Target: target, Success: true, Message: "dry run - pre-check passed"}
	}

	// Push branches with retry
	if err := m.withRetry("push branches", func() error {
		return m.pushBranches(remoteName)
	}); err != nil {
		return Result{Target: target, Success: false, Message: fmt.Sprintf("failed to push branches: %v", err)}
	}

	// Push tags with retry
	if m.cfg.MirrorTags {
		if err := m.withRetry("push tags", func() error {
			return m.pushTags(remoteName)
		}); err != nil {
			return Result{Target: target, Success: false, Message: fmt.Sprintf("failed to push tags: %v", err)}
		}
	}

	return Result{Target: target, Success: true, Message: "mirrored successfully"}
}

// sanitizeRemoteName creates a safe remote name suffix from a URL.
func sanitizeRemoteName(rawURL string) string {
	s := rawURL
	for _, ch := range []string{"https://", "http://", "git@", ":", "/", "."} {
		s = strings.ReplaceAll(s, ch, "-")
	}
	return strings.Trim(s, "-")
}

// withRetry executes fn with retry logic based on config.
func (m *Mirror) withRetry(operation string, fn func() error) error {
	var lastErr error
	attempts := 1 + m.cfg.RetryCount
	for i := 0; i < attempts; i++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if i < attempts-1 {
			m.logInfo("Retry %d/%d for %s after error: %v", i+1, m.cfg.RetryCount, operation, lastErr)
			time.Sleep(time.Duration(m.cfg.RetryDelay) * time.Second)
		}
	}
	return lastErr
}

func (m *Mirror) pushBranches(remote string) error {
	args := []string{"push"}
	if m.cfg.ForcePush {
		args = append(args, "-f")
	}

	if m.cfg.MirrorAllBranches {
		// When excluding branches with --all, use refspec exclusion
		if len(m.cfg.ExcludeBranches) > 0 {
			args = append(args, remote, "--all")
			// Push all first, then delete excluded branches on remote
			if err := m.git(args...); err != nil {
				return err
			}
			for _, branch := range m.cfg.ExcludeBranches {
				m.logDebug("Excluding branch %s from remote %s", branch, remote)
				_ = m.git("push", remote, "--delete", branch)
			}
			return nil
		}
		args = append(args, "--all", remote)
	} else {
		for _, branch := range m.cfg.MirrorBranches {
			if m.isExcluded(branch) {
				m.logDebug("Skipping excluded branch: %s", branch)
				continue
			}
			branchArgs := make([]string, len(args))
			copy(branchArgs, args)
			branchArgs = append(branchArgs, remote, fmt.Sprintf("refs/heads/%s:refs/heads/%s", branch, branch))
			if err := m.git(branchArgs...); err != nil {
				return fmt.Errorf("branch %s: %w", branch, err)
			}
		}
		return nil
	}

	return m.git(args...)
}

func (m *Mirror) isExcluded(branch string) bool {
	for _, excl := range m.cfg.ExcludeBranches {
		if excl == branch {
			return true
		}
	}
	return false
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
	rawURL := target.URL

	switch target.Provider {
	case config.ProviderGitLab:
		if m.cfg.GitLabToken != "" {
			rawURL = injectTokenAuth(rawURL, "oauth2", m.cfg.GitLabToken)
		}
	case config.ProviderGitHub:
		if m.cfg.GitHubToken != "" {
			rawURL = injectTokenAuth(rawURL, "x-access-token", m.cfg.GitHubToken)
		}
	case config.ProviderBitbucket:
		if m.cfg.BitbucketUsername != "" && m.cfg.BitbucketAPIToken != "" {
			rawURL = injectTokenAuth(rawURL, m.cfg.BitbucketUsername, m.cfg.BitbucketAPIToken)
		}
	case config.ProviderCodeCommit:
		// CodeCommit uses credential-helper or IAM, URL is used as-is
	case config.ProviderGeneric:
		// Use URL as-is; SSH key is configured via setupSSH
	}

	return rawURL, nil
}

// injectTokenAuth injects URL-encoded username:password into an HTTPS URL.
func injectTokenAuth(rawURL, username, password string) string {
	if strings.HasPrefix(rawURL, "https://") {
		encodedUser := url.QueryEscape(username)
		encodedPass := url.QueryEscape(password)
		return fmt.Sprintf("https://%s:%s@%s", encodedUser, encodedPass, strings.TrimPrefix(rawURL, "https://"))
	}
	return rawURL
}

func (m *Mirror) git(args ...string) error {
	return m.gitFn(args...)
}

func (m *Mirror) execGit(args ...string) error {
	m.logDebug("git %s", m.maskSecrets(strings.Join(args, " ")))
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	// Prevent interactive credential prompts (fails gracefully in CI/Docker)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	// Capture stderr and mask secrets before outputting
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if stderrBuf.Len() > 0 {
		fmt.Fprint(os.Stderr, m.maskSecrets(stderrBuf.String()))
	}
	return err
}

// maskSecrets replaces sensitive values in a string with ***.
func (m *Mirror) maskSecrets(s string) string {
	for _, secret := range m.secrets {
		s = strings.ReplaceAll(s, secret, "***")
	}
	return s
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
