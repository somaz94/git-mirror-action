# CLAUDE.md - multi-git-mirror

GitHub Action to mirror repositories to multiple Git hosting providers (GitLab, GitHub, Bitbucket, CodeCommit, etc.) using Go.

## Commit Guidelines

- Do not include `Co-Authored-By` lines in commit messages.

## Project Structure

```
cmd/main.go                  # Entrypoint
internal/
  config/                    # Input parsing, target detection, env bools, validation
  mirror/
    mirror.go                # Mirror logic (push branches/tags, auth URL injection, log masking)
    ssh.go                   # SSH key setup/cleanup for git operations
  output/                    # GitHub Actions output writer (JSON result, counts)
Makefile                     # Build, test, lint commands
Dockerfile                   # Multi-stage (golang:1.24-alpine → alpine:3.21)
action.yml                   # GitHub Action definition (15 inputs, 3 outputs)
cliff.toml                   # git-cliff config for release notes
```

## Build & Test

```bash
make test            # Run unit tests (alias for test-unit)
make test-unit       # go test ./internal/... ./cmd/... -v -race -cover
make test-all        # Run all tests
make cover           # Generate coverage report
make cover-html      # Open coverage in browser
make bench           # Run benchmarks
make lint            # go vet
make fmt             # gofmt
make build           # Build binary
make clean           # Remove artifacts
```

## Key Inputs

- **Required**: `targets` (newline-separated, `provider::url` or auto-detect)
- **Auth**: `gitlab_token`, `github_token`, `bitbucket_username`, `bitbucket_api_token`, `ssh_private_key`
- **Options**: `mirror_branches` (default: all), `mirror_tags` (default: true), `force_push` (default: true), `exclude_branches`, `parallel`
- **Retry**: `retry_count` (default: 0), `retry_delay` (default: 5s)
- **Debug**: `dry_run` (includes pre-check via `git ls-remote`), `debug`

## Key Outputs

- `result`: JSON array of mirror results per target
- `mirrored_count`: Number of successfully mirrored targets
- `failed_count`: Number of failed mirror targets

## Supported Providers

- `gitlab` — oauth2 token auth
- `github` — x-access-token auth
- `bitbucket` — username:app-password auth
- `codecommit` — IAM/credential-helper (URL as-is)
- `generic` — SSH key or URL as-is

## Workflow Structure

| Workflow | Name | Trigger |
|----------|------|---------|
| `ci.yml` | `Continuous Integration` | push(main), PR, dispatch |
| `release.yml` | `Create release` | tag push `v*` |
| `changelog-generator.yml` | `Generate changelog` | after release, PR merge |
| `contributors.yml` | `Generate Contributors` | after changelog |
| `use-action.yml` | `Smoke Test (Released Action)` | after release, dispatch |
| `gitlab-mirror.yml` | `Backup GitHub to GitLab` | push(main) |
| `stale-issues.yml` | `Close Stale Issues` | daily cron |
| `dependabot-auto-merge.yml` | `Dependabot auto-merge` | PR (dependabot) |
| `issue-greeting.yml` | `Issue Greeting Bot` | issue opened |

### Workflow Chain
```
tag push v* → Create release
                ├→ Smoke Test (Released Action)
                └→ Generate changelog
                      └→ Generate Contributors
```

## Conventions

- **Commits**: Conventional Commits (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `ci:`, `chore:`)
- **Secrets**: `PAT_TOKEN` (cross-repo ops), `GITHUB_TOKEN` (releases), `GITLAB_TOKEN` (mirror)
- **Docker**: Multi-stage build, alpine base with git + openssh-client
- **Comments**: English only
- **Release**: git-cliff for RELEASE.md, major-tag-action for `v1` tag
- **cliff.toml**: Skip `^Merge`, `^Update changelog`, `^Auto commit`
- **paths-ignore**: `.github/workflows/**`, `**/*.md`

## Language

- Communicate with the user in Korean.
