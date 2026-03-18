# Configuration

Complete reference for all inputs, outputs, and target configuration.

<br/>

## Table of Contents

- [Inputs](#inputs)
- [Outputs](#outputs)
- [Target Format](#target-format)
- [Supported Providers](#supported-providers)
- [Branch Configuration](#branch-configuration)
- [Default Values](#default-values)

<br/>

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `targets` | Mirror target URLs (newline-separated, `provider::url` or auto-detect) | Yes | - |
| `gitlab_token` | GitLab personal access token | No | `''` |
| `github_token` | GitHub personal access token | No | `''` |
| `bitbucket_username` | Bitbucket username for app password auth | No | `''` |
| `bitbucket_api_token` | Bitbucket API token | No | `''` |
| `ssh_private_key` | SSH private key for SSH-based authentication | No | `''` |
| `mirror_branches` | Branches to mirror (comma-separated, or `all`) | No | `all` |
| `mirror_tags` | Mirror tags | No | `true` |
| `force_push` | Use force push | No | `true` |
| `dry_run` | Dry run mode with remote pre-check (`git ls-remote`) | No | `false` |
| `retry_count` | Number of retry attempts on push failure (0 = no retry) | No | `0` |
| `retry_delay` | Delay in seconds between retry attempts | No | `5` |
| `exclude_branches` | Branches to exclude from mirroring (comma-separated) | No | `''` |
| `parallel` | Mirror to multiple targets in parallel | No | `false` |
| `debug` | Enable debug logging | No | `false` |

<br/>

## Outputs

| Output | Description | Example |
|--------|-------------|---------|
| `result` | JSON array with mirror results per target | `[{"target":{...},"success":true,"message":"mirrored successfully"}]` |
| `mirrored_count` | Number of successfully mirrored targets | `2` |
| `failed_count` | Number of failed mirror targets | `0` |

### Result JSON Structure

Each element in the `result` array has the following structure:

```json
{
  "target": {
    "Provider": "gitlab",
    "URL": "https://gitlab.com/org/repo.git"
  },
  "success": true,
  "message": "mirrored successfully"
}
```

<br/>

## Target Format

Targets are specified one per line in the `targets` input. Two formats are supported:

```
provider::url          # explicit provider
url                    # auto-detect from URL
```

### Examples

```yaml
targets: |
  gitlab::https://gitlab.com/myorg/myrepo.git
  codecommit::https://git-codecommit.us-east-1.amazonaws.com/v1/repos/myrepo
  https://bitbucket.org/myorg/myrepo.git
```

In the example above, the third target will auto-detect `bitbucket` as the provider from the URL.

<br/>

## Supported Providers

| Provider | Auth Method | Auto-detect Pattern |
|----------|-------------|---------------------|
| `gitlab` | OAuth2 token (`oauth2:<token>`) | URL contains `gitlab` |
| `github` | x-access-token (`x-access-token:<token>`) | URL contains `github` |
| `bitbucket` | Username + App password (`user:pass`) | URL contains `bitbucket` |
| `codecommit` | IAM / credential-helper (URL as-is) | URL contains `codecommit` |
| `generic` | SSH key or URL as-is | Default fallback |

<br/>

## Branch Configuration

<br/>

### Mirror All Branches

```yaml
mirror_branches: 'all'    # default
```

<br/>

### Mirror Specific Branches

```yaml
mirror_branches: 'main,develop,release'
```

Branches are specified as a comma-separated list. Each branch is pushed using the refspec `refs/heads/<branch>:refs/heads/<branch>`.

<br/>

### Exclude Branches

Exclude specific branches from mirroring. Works with both `all` and specific branch lists.

```yaml
mirror_branches: 'all'
exclude_branches: 'staging,hotfix,experiment'
```

When using `all`, excluded branches are pushed first with `--all`, then deleted from the remote. When using specific branches, excluded branches are simply skipped.

<br/>

## Retry Configuration

Configure retry behavior for transient network failures.

```yaml
retry_count: '3'     # Retry up to 3 times
retry_delay: '10'    # Wait 10 seconds between retries
```

Retry applies to both branch push and tag push operations independently.

<br/>

## Parallel Execution

Mirror to multiple targets concurrently using goroutines.

```yaml
parallel: 'true'
```

Each target gets a unique remote name to avoid conflicts. Parallel mode is automatically disabled when there is only one target.

<br/>

## Dry Run & Pre-check

When `dry_run` is enabled, the action validates remote connectivity using `git ls-remote` without actually pushing.

```yaml
dry_run: 'true'
debug: 'true'    # recommended with dry_run for detailed logs
```

This verifies that authentication and network connectivity are working before doing a real push.

<br/>

## Default Values

| Setting | Default | Notes |
|---------|---------|-------|
| `mirror_branches` | `all` | Pushes all branches with `--all` flag |
| `mirror_tags` | `true` | Pushes tags with `--tags` flag |
| `force_push` | `true` | Uses `-f` flag for force push |
| `dry_run` | `false` | When `true`, runs pre-check via `git ls-remote` |
| `retry_count` | `0` | Number of retry attempts (0 = no retry) |
| `retry_delay` | `5` | Seconds to wait between retries |
| `exclude_branches` | `''` | Comma-separated list of branches to exclude |
| `parallel` | `false` | When `true`, mirrors targets concurrently |
| `debug` | `false` | When `true`, logs git commands |

Boolean inputs accept: `true`, `1`, `yes` (truthy) or `false`, `0`, `no` (falsy).
