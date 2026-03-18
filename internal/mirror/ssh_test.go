package mirror

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/somaz94/multi-git-mirror/internal/config"
)

func TestSetupSSHNoKey(t *testing.T) {
	cfg := &config.Config{}
	m := New(cfg)

	err := m.setupSSH()
	if err != nil {
		t.Fatalf("unexpected error with no SSH key: %v", err)
	}
}

func TestCleanupSSHNoKey(t *testing.T) {
	cfg := &config.Config{}
	m := New(cfg)

	// Should not panic or error with no SSH key
	m.cleanupSSH()
}

func TestSetupSSHWritesAllFiles(t *testing.T) {
	tmpDir := t.TempDir()
	sshPath := filepath.Join(tmpDir, ".ssh")

	cfg := &config.Config{
		SSHPrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key-data\n-----END OPENSSH PRIVATE KEY-----",
	}
	m := New(cfg)
	m.sshDir = sshPath

	err := m.setupSSH()
	if err != nil {
		t.Fatalf("setupSSH failed: %v", err)
	}

	// Verify key file
	keyPath := filepath.Join(sshPath, sshKeyFile)
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("key file not found: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected key permission 0600, got %o", info.Mode().Perm())
	}
	data, _ := os.ReadFile(keyPath)
	if string(data) != cfg.SSHPrivateKey+"\n" {
		t.Error("key content mismatch")
	}

	// Verify SSH config file
	configPath := filepath.Join(sshPath, sshConfigFile)
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file not found: %v", err)
	}
	configStr := string(configData)
	if !strings.Contains(configStr, "IdentityFile") {
		t.Error("SSH config missing IdentityFile")
	}
	if !strings.Contains(configStr, "StrictHostKeyChecking no") {
		t.Error("SSH config missing StrictHostKeyChecking")
	}
	if !strings.Contains(configStr, keyPath) {
		t.Error("SSH config does not reference correct key path")
	}

	// Verify known_hosts file
	knownHostsPath := filepath.Join(sshPath, knownHosts)
	khInfo, err := os.Stat(knownHostsPath)
	if err != nil {
		t.Fatalf("known_hosts not found: %v", err)
	}
	if khInfo.Size() != 0 {
		t.Error("known_hosts should be empty")
	}

	// Verify GIT_SSH_COMMAND env var
	sshCmd := os.Getenv("GIT_SSH_COMMAND")
	if !strings.Contains(sshCmd, configPath) {
		t.Errorf("GIT_SSH_COMMAND should reference config path, got: %s", sshCmd)
	}
	if !strings.Contains(sshCmd, "BatchMode=yes") {
		t.Errorf("GIT_SSH_COMMAND should contain BatchMode=yes, got: %s", sshCmd)
	}

	// Cleanup env
	os.Unsetenv("GIT_SSH_COMMAND")
}

func TestCleanupSSHRemovesFiles(t *testing.T) {
	tmpDir := t.TempDir()
	sshPath := filepath.Join(tmpDir, ".ssh")

	cfg := &config.Config{
		SSHPrivateKey: "test-key",
	}
	m := New(cfg)
	m.sshDir = sshPath

	// Setup first
	err := m.setupSSH()
	if err != nil {
		t.Fatalf("setupSSH failed: %v", err)
	}

	// Verify files exist
	for _, f := range []string{sshKeyFile, sshConfigFile, knownHosts} {
		if _, err := os.Stat(filepath.Join(sshPath, f)); err != nil {
			t.Fatalf("expected file %s to exist before cleanup", f)
		}
	}

	// Verify GIT_SSH_COMMAND is set
	if os.Getenv("GIT_SSH_COMMAND") == "" {
		t.Fatal("expected GIT_SSH_COMMAND to be set before cleanup")
	}

	// Run cleanup
	m.cleanupSSH()

	// Verify files removed
	for _, f := range []string{sshKeyFile, sshConfigFile, knownHosts} {
		if _, err := os.Stat(filepath.Join(sshPath, f)); !os.IsNotExist(err) {
			t.Errorf("expected file %s to be removed after cleanup", f)
		}
	}

	// Verify env var unset
	if os.Getenv("GIT_SSH_COMMAND") != "" {
		t.Error("expected GIT_SSH_COMMAND to be unset after cleanup")
	}
}

func TestSetupSSHInvalidDir(t *testing.T) {
	cfg := &config.Config{
		SSHPrivateKey: "test-key",
	}
	m := New(cfg)
	// Point to a path that can't be created
	m.sshDir = "/dev/null/invalid"

	err := m.setupSSH()
	if err == nil {
		t.Fatal("expected error for invalid SSH directory")
	}
	if !strings.Contains(err.Error(), "failed to create .ssh directory") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSetupSSHKeyWriteFails(t *testing.T) {
	tmpDir := t.TempDir()
	sshPath := filepath.Join(tmpDir, ".ssh")

	cfg := &config.Config{
		SSHPrivateKey: "test-key",
	}
	m := New(cfg)
	m.sshDir = sshPath

	// Create the dir, then make it read-only so writing the key file fails
	if err := os.MkdirAll(sshPath, 0700); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	os.Chmod(sshPath, 0500)
	defer os.Chmod(sshPath, 0700)

	err := m.setupSSH()
	if err == nil {
		t.Fatal("expected error when key write fails")
	}
	if !strings.Contains(err.Error(), "failed to write SSH key") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunSSHSetupAndCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	sshPath := filepath.Join(tmpDir, ".ssh")

	cfg := &config.Config{
		SSHPrivateKey: "test-key",
		Targets: []config.Target{
			{Provider: config.ProviderGeneric, URL: "git@example.com:org/repo.git"},
		},
		MirrorAllBranches: true,
	}
	m := New(cfg)
	m.sshDir = sshPath
	m.gitFn = mockGitOK()

	results := m.Run()

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Success {
		t.Errorf("expected success, got: %s", results[0].Message)
	}

	// Verify cleanup happened — files should be gone
	for _, f := range []string{sshKeyFile, sshConfigFile, knownHosts} {
		if _, err := os.Stat(filepath.Join(sshPath, f)); !os.IsNotExist(err) {
			t.Errorf("expected file %s to be cleaned up after Run", f)
		}
	}
	if os.Getenv("GIT_SSH_COMMAND") != "" {
		t.Error("expected GIT_SSH_COMMAND to be unset after Run")
	}
}

func TestRunSSHSetupFails(t *testing.T) {
	cfg := &config.Config{
		SSHPrivateKey: "test-key",
		Targets: []config.Target{
			{Provider: config.ProviderGeneric, URL: "git@example.com:org/repo.git"},
		},
		MirrorAllBranches: true,
	}
	m := New(cfg)
	m.sshDir = "/dev/null/invalid"
	m.gitFn = mockGitOK()

	results := m.Run()

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Success {
		t.Error("expected failure when SSH setup fails")
	}
	if !strings.Contains(results[0].Message, "SSH setup failed") {
		t.Errorf("expected SSH setup error, got: %s", results[0].Message)
	}
}
