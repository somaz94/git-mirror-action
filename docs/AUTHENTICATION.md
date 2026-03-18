# Authentication

Guide for configuring authentication for each supported Git provider, including step-by-step token creation instructions.

<br/>

## Table of Contents

- [GitLab](#gitlab)
- [GitHub](#github)
- [Bitbucket](#bitbucket)
- [AWS CodeCommit](#aws-codecommit)
- [SSH Key](#ssh-key)
- [Adding Secrets to GitHub Actions](#adding-secrets-to-github-actions)
- [Security Best Practices](#security-best-practices)

<br/>

## GitLab

Uses OAuth2 token authentication. The token is injected into the HTTPS URL as `https://oauth2:<token>@gitlab.com/...`.

```yaml
- uses: somaz94/multi-git-mirror@v1
  with:
    targets: |
      gitlab::https://gitlab.com/myorg/myrepo.git
    gitlab_token: ${{ secrets.GITLAB_TOKEN }}
```

<br/>

### Required Token Scopes

| Scope | Purpose | Required |
|-------|---------|----------|
| `write_repository` | Push access to the target repository | Yes |
| `read_repository` | Read access (for private repos) | Optional |

<br/>

### Step-by-Step: Creating a GitLab Personal Access Token

1. Log in to [GitLab](https://gitlab.com)
2. Click your **avatar** (top-right) → **Preferences**
3. In the left sidebar, click **Access Tokens**
4. Click **Add new token**
5. Configure:
   - **Token name**: `multi-git-mirror`
   - **Expiration date**: Set an appropriate date (recommended: 1 year max)
   - **Scopes**: Check `write_repository`
6. Click **Create personal access token**
7. **Copy the token immediately** — it will not be shown again
8. Add it as a GitHub secret named `GITLAB_TOKEN` ([how to add secrets](#adding-secrets-to-github-actions))

<br/>

### Alternative: GitLab Project Access Token

For organization-level access, use a Project Access Token instead:

1. Go to your **GitLab project** → **Settings** → **Access Tokens**
2. Create a token with `write_repository` scope and **Maintainer** role
3. Use the token the same way as a personal access token

> **Note**: Project Access Tokens are available on GitLab Premium and higher. Free tier users should use Personal Access Tokens.

<br/>

## GitHub

Uses x-access-token authentication. The token is injected as `https://x-access-token:<token>@github.com/...`.

```yaml
- uses: somaz94/multi-git-mirror@v1
  with:
    targets: |
      github::https://github.com/myorg/myrepo.git
    github_token: ${{ secrets.MIRROR_GITHUB_TOKEN }}
```

<br/>

### Required Token Scopes

| Scope | Purpose | Required |
|-------|---------|----------|
| `repo` | Full control of private repositories | Yes (private repos) |
| `public_repo` | Push to public repositories only | Yes (public repos) |

> **Important**: Do NOT use `${{ secrets.GITHUB_TOKEN }}` for cross-repository mirroring. It only has access to the current repository. Use a Personal Access Token (PAT) instead.

<br/>

### Step-by-Step: Creating a GitHub Fine-Grained PAT (Recommended)

1. Log in to [GitHub](https://github.com)
2. Click your **avatar** (top-right) → **Settings**
3. Scroll down in the left sidebar → **Developer settings**
4. Click **Personal access tokens** → **Fine-grained tokens**
5. Click **Generate new token**
6. Configure:
   - **Token name**: `multi-git-mirror`
   - **Expiration**: Select an appropriate duration
   - **Resource owner**: Select the target organization or your account
   - **Repository access**: Select **Only select repositories** → choose the target repo
   - **Permissions** → **Repository permissions**:
     - **Contents**: `Read and write`
7. Click **Generate token**
8. **Copy the token immediately**
9. Add it as a GitHub secret named `MIRROR_GITHUB_TOKEN`

<br/>

### Alternative: Classic PAT

1. Go to **Settings** → **Developer settings** → **Personal access tokens** → **Tokens (classic)**
2. Click **Generate new token (classic)**
3. Check `repo` scope (or `public_repo` for public repos only)
4. Generate and copy the token

<br/>

## Bitbucket

Uses access token authentication. Credentials are injected as `https://user:token@bitbucket.org/...`.

```yaml
- uses: somaz94/multi-git-mirror@v1
  with:
    targets: |
      bitbucket::https://bitbucket.org/myorg/myrepo.git
    bitbucket_username: ${{ secrets.BITBUCKET_USERNAME }}
    bitbucket_api_token: ${{ secrets.BITBUCKET_API_TOKEN }}
```

<br/>

### Required Scopes

| Scope | Purpose | Required |
|-------|---------|----------|
| Repositories: Write | Push access to repositories | Yes |
| Repositories: Read | Read access (for private repos) | Optional |

> **Note**: Both `bitbucket_username` and `bitbucket_api_token` must be provided. If either is missing, the URL is used as-is.

> **Important**: Bitbucket deprecated App Passwords (can no longer be created as of Sep 9, 2025; fully disabled June 9, 2026). Use **Repository Access Token** or **Workspace Access Token** instead. Atlassian API Tokens (`id.atlassian.com`) are for REST APIs only and do **NOT** work for git HTTPS authentication.

<br/>

### Step-by-Step: Creating a Repository Access Token (Recommended)

Scoped to a single repository. Best for minimal-privilege access.

1. Go to your **Bitbucket repository** → **Repository settings**
2. In the left sidebar under **Security**, click **Access tokens**
3. Click **Create Repository Access Token**
4. Configure:
   - **Name**: `multi-git-mirror`
   - **Scopes**: Check **Repositories** → **Write**
5. Click **Create**
6. **Copy the token immediately** — it will not be shown again
7. Note the git clone command shown — the username is `x-token-auth`:
   ```
   git clone https://x-token-auth:<token>@bitbucket.org/<workspace>/<repo>.git
   ```
8. Add two GitHub secrets:
   - `BITBUCKET_USERNAME`: `x-token-auth` (the auto-generated username)
   - `BITBUCKET_API_TOKEN`: The token you just created

> **Note**: Repository Access Tokens always use `x-token-auth` as the username for git HTTPS operations. Use that as `bitbucket_username`, not your personal username.

<br/>

### Alternative: Creating a Workspace Access Token

Scoped to all repositories in a workspace. Useful when mirroring multiple repos.

> **Note**: Workspace Access Tokens require a **Bitbucket Premium** plan. Free/Standard plans should use Repository Access Tokens instead.

1. Go to your **Bitbucket workspace** → **Settings** (or visit `https://bitbucket.org/<workspace>/workspace/settings/access-tokens`)
2. In the left sidebar under **Security**, click **Access tokens**
3. Click **Create Workspace Access Token**
4. Configure:
   - **Name**: `multi-git-mirror`
   - **Scopes**: Check **Repositories** → **Write**
5. Click **Create**
6. **Copy the token immediately** — it will not be shown again
7. Add two GitHub secrets:
   - `BITBUCKET_USERNAME`: The auto-generated username shown with the token
   - `BITBUCKET_API_TOKEN`: The token you just created

<br/>

### Finding Your Bitbucket Username

When using **Repository/Workspace Access Tokens**, the username is auto-generated and shown when the token is created. Use that username, not your personal one.

For personal username (used with legacy App Passwords):
1. Click the **gear icon** (⚙️, top-right) → **Personal Bitbucket settings**
2. Your username is shown under **Atlassian account settings** or in the URL: `bitbucket.org/<username>/`

<br/>

## AWS CodeCommit

CodeCommit uses IAM-based authentication via the Git credential helper. The URL is used as-is without token injection.

```yaml
- name: Configure AWS credentials
  uses: aws-actions/configure-aws-credentials@v4
  with:
    aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
    aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
    aws-region: us-east-1

- name: Setup CodeCommit credential helper
  run: git config --global credential.helper '!aws codecommit credential-helper $@'

- uses: somaz94/multi-git-mirror@v1
  with:
    targets: |
      codecommit::https://git-codecommit.us-east-1.amazonaws.com/v1/repos/myrepo
```

<br/>

### Required IAM Permissions

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "codecommit:GitPush",
        "codecommit:CreateBranch",
        "codecommit:GetRepository"
      ],
      "Resource": "arn:aws:codecommit:us-east-1:123456789012:myrepo"
    }
  ]
}
```

<br/>

### Step-by-Step: Creating AWS IAM Credentials

#### Option A: IAM User with Access Keys

1. Log in to [AWS Console](https://console.aws.amazon.com) → **IAM**
2. Click **Users** → **Create user**
3. Enter username: `multi-git-mirror`
4. Click **Next** → **Attach policies directly**
5. Click **Create policy** and paste the JSON above (adjust the Resource ARN)
6. Attach the policy to the user
7. Go to the user → **Security credentials** → **Create access key**
8. Select **Third-party service** → **Create access key**
9. **Copy both keys immediately**
10. Add two GitHub secrets:
    - `AWS_ACCESS_KEY_ID`: The access key ID
    - `AWS_SECRET_ACCESS_KEY`: The secret access key

#### Option B: OIDC (Recommended for Production)

For keyless authentication, use GitHub OIDC provider:

```yaml
- name: Configure AWS credentials (OIDC)
  uses: aws-actions/configure-aws-credentials@v4
  with:
    role-to-assume: arn:aws:iam::123456789012:role/git-mirror-role
    aws-region: us-east-1
```

See [AWS docs](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_create_oidc.html) for OIDC setup.

<br/>

## SSH Key

For providers that support SSH, use the `ssh_private_key` input with the `generic` provider.

```yaml
- uses: somaz94/multi-git-mirror@v1
  with:
    targets: |
      generic::git@custom-git.example.com:org/repo.git
    ssh_private_key: ${{ secrets.SSH_PRIVATE_KEY }}
```

<br/>

### How It Works

When `ssh_private_key` is provided, the action automatically:

1. Writes the key to `/root/.ssh/mirror_key` with `0600` permissions
2. Creates an SSH config with `StrictHostKeyChecking no`
3. Sets `GIT_SSH_COMMAND` to use the configured key
4. Cleans up all SSH files after mirroring completes

<br/>

### Step-by-Step: Creating an SSH Key Pair

1. Generate an Ed25519 key pair (no passphrase):

   ```bash
   ssh-keygen -t ed25519 -C "multi-git-mirror" -f mirror_key -N ""
   ```

2. This creates two files:
   - `mirror_key` — Private key (add to GitHub secrets)
   - `mirror_key.pub` — Public key (add to target Git server)

3. **Add the public key** to the target Git server:
   - **GitLab**: Settings → Repository → Deploy keys → Add key (check "Grant write permissions")
   - **GitHub**: Settings → Deploy keys → Add deploy key (check "Allow write access")
   - **Bitbucket**: Repository settings → Access keys → Add key
   - **Custom server**: Add to `~/.ssh/authorized_keys` on the server

4. **Add the private key** as a GitHub secret:
   ```bash
   # Copy the private key content
   cat mirror_key
   ```
   Add this as `SSH_PRIVATE_KEY` in GitHub secrets

5. **Delete local key files** after setup:
   ```bash
   rm mirror_key mirror_key.pub
   ```

<br/>

## Adding Secrets to GitHub Actions

All credentials must be stored as GitHub secrets. Never hardcode them in workflow files.

<br/>

### Step-by-Step: Adding a Repository Secret

1. Go to your **GitHub repository**
2. Click **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Enter:
   - **Name**: The secret name (e.g., `GITLAB_TOKEN`)
   - **Secret**: The token/password value
5. Click **Add secret**

<br/>

### Step-by-Step: Adding an Organization Secret

1. Go to your **GitHub organization**
2. Click **Settings** → **Secrets and variables** → **Actions**
3. Click **New organization secret**
4. Enter the name and value
5. Under **Repository access**, select which repos can use this secret
6. Click **Add secret**

<br/>

### Quick Reference: Required Secrets Per Provider

| Provider | Secrets Needed |
|----------|---------------|
| GitLab | `GITLAB_TOKEN` |
| GitHub | `MIRROR_GITHUB_TOKEN` |
| Bitbucket | `BITBUCKET_USERNAME`, `BITBUCKET_API_TOKEN` |
| CodeCommit | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` |
| SSH | `SSH_PRIVATE_KEY` |

<br/>

## Security Best Practices

- **Always use GitHub Secrets** — Never hardcode tokens or passwords in workflow files
- **Use least-privilege tokens** — Grant only the minimum required scopes
- **Rotate tokens regularly** — Set expiration dates and rotate periodically
- **Use separate tokens per target** — Avoid reusing a single token across providers
- **Audit access** — Regularly review which tokens have access to your repositories
- **Prefer fine-grained tokens** — GitHub Fine-Grained PATs and GitLab Project Access Tokens over classic tokens
- **Use OIDC where possible** — For AWS CodeCommit, prefer OIDC over static access keys

<br/>

### Built-in Security Features

- **Credential URL encoding** — Special characters (`@`, `:`, `/`, etc.) in passwords are URL-encoded to prevent URL parsing issues
- **Log masking** — Tokens, passwords, and usernames are replaced with `***` in all log output (including git stderr)
- **SSH key cleanup** — SSH key files are automatically removed after mirroring
- **Config validation** — Warns when required credentials are missing for a target provider
