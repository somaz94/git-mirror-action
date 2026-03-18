# Examples

Complete workflow examples for common mirroring scenarios.

<br/>

## Table of Contents

- [Basic Mirror on Push](#basic-mirror-on-push)
- [Scheduled Mirror](#scheduled-mirror)
- [Release and Mirror](#release-and-mirror)
- [Multi-Provider Backup](#multi-provider-backup)
- [Selective Branch Mirror](#selective-branch-mirror)
- [Exclude Branches](#exclude-branches)
- [Parallel Mirror with Retry](#parallel-mirror-with-retry)
- [Dry Run Testing](#dry-run-testing)
- [Conditional Mirror with Output](#conditional-mirror-with-output)

<br/>

## Basic Mirror on Push

Mirror to GitLab every time code is pushed to main.

```yaml
name: Mirror to GitLab

on:
  push:
    branches:
      - main

jobs:
  mirror:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0

      - uses: somaz94/git-mirror-action@v1
        with:
          targets: |
            gitlab::https://gitlab.com/myorg/myrepo.git
          gitlab_token: ${{ secrets.GITLAB_TOKEN }}
```

<br/>

## Scheduled Mirror

Run a full mirror every day at midnight UTC.

```yaml
name: Scheduled Mirror

on:
  schedule:
    - cron: '0 0 * * *'
  workflow_dispatch:

jobs:
  mirror:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0

      - uses: somaz94/git-mirror-action@v1
        with:
          targets: |
            gitlab::https://gitlab.com/myorg/myrepo.git
            bitbucket::https://bitbucket.org/myorg/myrepo.git
          gitlab_token: ${{ secrets.GITLAB_TOKEN }}
          bitbucket_username: ${{ secrets.BITBUCKET_USERNAME }}
          bitbucket_api_token: ${{ secrets.BITBUCKET_API_TOKEN }}
          mirror_branches: 'all'
          mirror_tags: 'true'
```

<br/>

## Release and Mirror

Mirror to backup providers after creating a release.

```yaml
name: Release and Mirror

on:
  push:
    tags:
      - "v[0-9]+.[0-9]+.[0-9]+"

permissions:
  contents: write

jobs:
  release:
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

## Multi-Provider Backup

Mirror to three providers simultaneously.

```yaml
name: Multi-Provider Backup

on:
  push:
    branches:
      - main
  workflow_dispatch:

jobs:
  mirror:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0

      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: us-east-1

      - name: Setup CodeCommit credential helper
        run: git config --global credential.helper '!aws codecommit credential-helper $@'

      - name: Mirror to all providers
        uses: somaz94/git-mirror-action@v1
        with:
          targets: |
            gitlab::https://gitlab.com/myorg/myrepo.git
            bitbucket::https://bitbucket.org/myorg/myrepo.git
            codecommit::https://git-codecommit.us-east-1.amazonaws.com/v1/repos/myrepo
          gitlab_token: ${{ secrets.GITLAB_TOKEN }}
          bitbucket_username: ${{ secrets.BITBUCKET_USERNAME }}
          bitbucket_api_token: ${{ secrets.BITBUCKET_API_TOKEN }}
```

<br/>

## Selective Branch Mirror

Mirror only specific branches.

```yaml
- uses: somaz94/git-mirror-action@v1
  with:
    targets: |
      gitlab::https://gitlab.com/myorg/myrepo.git
    gitlab_token: ${{ secrets.GITLAB_TOKEN }}
    mirror_branches: 'main,develop,release'
    mirror_tags: 'false'
```

<br/>

## Exclude Branches

Mirror all branches except specific ones.

```yaml
- uses: somaz94/git-mirror-action@v1
  with:
    targets: |
      gitlab::https://gitlab.com/myorg/myrepo.git
    gitlab_token: ${{ secrets.GITLAB_TOKEN }}
    mirror_branches: 'all'
    exclude_branches: 'staging,experiment,tmp'
```

<br/>

## Parallel Mirror with Retry

Mirror to multiple providers concurrently with automatic retry on failure.

```yaml
- uses: somaz94/git-mirror-action@v1
  with:
    targets: |
      gitlab::https://gitlab.com/myorg/myrepo.git
      bitbucket::https://bitbucket.org/myorg/myrepo.git
      codecommit::https://git-codecommit.us-east-1.amazonaws.com/v1/repos/myrepo
    gitlab_token: ${{ secrets.GITLAB_TOKEN }}
    bitbucket_username: ${{ secrets.BITBUCKET_USERNAME }}
    bitbucket_api_token: ${{ secrets.BITBUCKET_API_TOKEN }}
    parallel: 'true'
    retry_count: '3'
    retry_delay: '10'
```

<br/>

## Dry Run Testing

Test your configuration without actually pushing.

```yaml
- uses: somaz94/git-mirror-action@v1
  id: mirror
  with:
    targets: |
      gitlab::https://gitlab.com/myorg/myrepo.git
      bitbucket::https://bitbucket.org/myorg/myrepo.git
    gitlab_token: ${{ secrets.GITLAB_TOKEN }}
    bitbucket_username: ${{ secrets.BITBUCKET_USERNAME }}
    bitbucket_api_token: ${{ secrets.BITBUCKET_API_TOKEN }}
    dry_run: 'true'
    debug: 'true'

- name: Show results
  run: |
    echo "Result: ${{ steps.mirror.outputs.result }}"
    echo "Would mirror to: ${{ steps.mirror.outputs.mirrored_count }} targets"
```

<br/>

## Conditional Mirror with Output

Use mirror results in subsequent steps.

```yaml
- name: Mirror repositories
  uses: somaz94/git-mirror-action@v1
  id: mirror
  with:
    targets: |
      gitlab::https://gitlab.com/myorg/myrepo.git
      bitbucket::https://bitbucket.org/myorg/myrepo.git
    gitlab_token: ${{ secrets.GITLAB_TOKEN }}
    bitbucket_username: ${{ secrets.BITBUCKET_USERNAME }}
    bitbucket_api_token: ${{ secrets.BITBUCKET_API_TOKEN }}

- name: Notify on failure
  if: steps.mirror.outputs.failed_count != '0'
  run: |
    echo "::warning::${{ steps.mirror.outputs.failed_count }} mirror target(s) failed"
    echo "Details: ${{ steps.mirror.outputs.result }}"

- name: Report success
  if: steps.mirror.outputs.failed_count == '0'
  run: echo "All ${{ steps.mirror.outputs.mirrored_count }} mirrors succeeded"
```
