# Git Mirror Action

[![Continuous Integration](https://github.com/somaz94/git-mirror-action/actions/workflows/ci.yml/badge.svg)](https://github.com/somaz94/git-mirror-action/actions/workflows/ci.yml)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Latest Tag](https://img.shields.io/github/v/tag/somaz94/git-mirror-action)](https://github.com/somaz94/git-mirror-action/tags)
[![Top Language](https://img.shields.io/github/languages/top/somaz94/git-mirror-action)](https://github.com/somaz94/git-mirror-action)
[![GitHub Marketplace](https://img.shields.io/badge/Marketplace-Git%20Mirror%20Action-blue?logo=github)](https://github.com/marketplace/actions/git-mirror-action)

A Go-based GitHub Action that mirrors repositories to multiple Git hosting providers — GitLab, GitHub, Bitbucket, AWS CodeCommit, and more — in a single step.

<br/>

## Features

- Multi-target mirroring in a single workflow step
- Auto-detect provider from URL (GitLab, GitHub, Bitbucket, CodeCommit)
- Selective branch mirroring or mirror all branches
- Exclude specific branches from mirroring
- Tag mirroring support
- Force push option for exact replication
- Multiple authentication methods (token, app password, SSH key)
- Parallel mirroring to multiple targets concurrently
- Retry logic with configurable count and delay
- Dry run mode with remote connectivity pre-check (`git ls-remote`)
- JSON result output for downstream steps
- Credential masking in all log output

> For detailed documentation, see the [docs/](docs/) folder:
> [Authentication](docs/AUTHENTICATION.md) |
> [Configuration](docs/CONFIGURATION.md) |
> [Examples](docs/EXAMPLES.md) |
> [Development](docs/DEVELOPMENT.md)

<br/>

## Usage

<br/>

### Basic — Mirror to GitLab

```yaml
steps:
  - name: Checkout
    uses: actions/checkout@v6
    with:
      fetch-depth: 0

  - name: Mirror to GitLab
    uses: somaz94/git-mirror-action@v1
    with:
      targets: |
        gitlab::https://gitlab.com/myorg/myrepo.git
      gitlab_token: ${{ secrets.GITLAB_TOKEN }}
```

<br/>

### Multi-target — GitLab + CodeCommit

```yaml
- name: Mirror to multiple targets
  uses: somaz94/git-mirror-action@v1
  with:
    targets: |
      gitlab::https://gitlab.com/myorg/myrepo.git
      codecommit::https://git-codecommit.us-east-1.amazonaws.com/v1/repos/myrepo
    gitlab_token: ${{ secrets.GITLAB_TOKEN }}
    mirror_branches: 'main,develop'
    mirror_tags: 'true'
```

<br/>

### Mirror to Bitbucket

```yaml
- name: Mirror to Bitbucket
  uses: somaz94/git-mirror-action@v1
  with:
    targets: |
      bitbucket::https://bitbucket.org/myorg/myrepo.git
    bitbucket_username: ${{ secrets.BITBUCKET_USERNAME }}
    bitbucket_api_token: ${{ secrets.BITBUCKET_API_TOKEN }}
```

<br/>

### With SSH Key

```yaml
- name: Mirror via SSH
  uses: somaz94/git-mirror-action@v1
  with:
    targets: |
      generic::git@custom-git.example.com:org/repo.git
    ssh_private_key: ${{ secrets.SSH_PRIVATE_KEY }}
```

<br/>

### Dry Run with Output

```yaml
- name: Mirror (dry run)
  uses: somaz94/git-mirror-action@v1
  id: mirror
  with:
    targets: |
      gitlab::https://gitlab.com/myorg/myrepo.git
    gitlab_token: ${{ secrets.GITLAB_TOKEN }}
    dry_run: 'true'
    debug: 'true'

- name: Check results
  run: |
    echo "Result: ${{ steps.mirror.outputs.result }}"
    echo "Mirrored: ${{ steps.mirror.outputs.mirrored_count }}"
    echo "Failed: ${{ steps.mirror.outputs.failed_count }}"
```

<br/>

### In Release Workflow

```yaml
name: Release and Mirror

on:
  push:
    tags:
      - "v[0-9]+.[0-9]+.[0-9]+"

jobs:
  release-and-mirror:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0

      - name: Create GitHub release
        uses: softprops/action-gh-release@v2
        with:
          tag_name: ${{ github.ref_name }}

      - name: Mirror to backup providers
        uses: somaz94/git-mirror-action@v1
        with:
          targets: |
            gitlab::https://gitlab.com/myorg/myrepo.git
            bitbucket::https://bitbucket.org/myorg/myrepo.git
          gitlab_token: ${{ secrets.GITLAB_TOKEN }}
          bitbucket_username: ${{ secrets.BITBUCKET_USERNAME }}
          bitbucket_api_token: ${{ secrets.BITBUCKET_API_TOKEN }}
```

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
| `dry_run` | Dry run mode with remote pre-check | No | `false` |
| `retry_count` | Number of retry attempts on push failure | No | `0` |
| `retry_delay` | Delay in seconds between retries | No | `5` |
| `exclude_branches` | Branches to exclude (comma-separated) | No | `''` |
| `parallel` | Mirror to targets in parallel | No | `false` |
| `debug` | Enable debug logging | No | `false` |

<br/>

## Outputs

| Output | Description | Example |
|--------|-------------|---------|
| `result` | JSON array with mirror results per target | `[{"target":{...},"success":true,"message":"mirrored successfully"}]` |
| `mirrored_count` | Number of successfully mirrored targets | `2` |
| `failed_count` | Number of failed mirror targets | `0` |

<br/>

## Target Format

Targets are specified one per line. You can explicitly set the provider or let it auto-detect from the URL:

```
provider::url          # explicit provider
url                    # auto-detect from URL
```

### Supported Providers

| Provider | Auth Method | Example URL |
|----------|-------------|-------------|
| `gitlab` | OAuth2 token | `https://gitlab.com/org/repo.git` |
| `github` | x-access-token | `https://github.com/org/repo.git` |
| `bitbucket` | Username + App password | `https://bitbucket.org/org/repo.git` |
| `codecommit` | IAM / credential-helper | `https://git-codecommit.us-east-1.amazonaws.com/v1/repos/repo` |
| `generic` | SSH key or URL as-is | `git@custom-git.example.com:org/repo.git` |

<br/>

## Why?

Many teams need to keep repository mirrors in sync across multiple Git providers — for disaster recovery, CI/CD across platforms, compliance, or migration. This action replaces fragile shell scripts with a single, configurable step that handles authentication, branch/tag selection, and multi-target mirroring out of the box.

<br/>

## Project Structure

```
.
├── cmd/
│   └── main.go                  # Entry point
├── internal/
│   ├── config/
│   │   ├── config.go            # Configuration loading & target parsing
│   │   └── config_test.go       # Config tests
│   ├── mirror/
│   │   ├── mirror.go            # Mirror logic, retry, parallel, auth URL injection
│   │   ├── ssh.go               # SSH key setup/cleanup
│   │   └── mirror_test.go       # Mirror tests
│   └── output/
│       └── output.go            # GitHub Actions output writer
├── .github/
│   └── workflows/               # CI/CD workflows (9 files)
├── action.yml                   # Action metadata
├── Dockerfile                   # Multi-stage Docker build
├── Makefile                     # Build targets
├── cliff.toml                   # git-cliff changelog config
└── go.mod
```

<br/>

## Development

<br/>

### Prerequisites

- Go 1.24+
- Docker (for container builds)

<br/>

### Build

```bash
make build
```

<br/>

### Test

```bash
make test
```

<br/>

### Coverage

```bash
make cover
```

<br/>

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

<br/>

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
