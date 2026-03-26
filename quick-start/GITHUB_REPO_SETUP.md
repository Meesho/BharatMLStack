# GitHub Repo Setup for Predator (Script-Only)

This guide covers using `setup-predator-github.sh` to **populate your GitHub repository** with the Predator Helm chart, `prd/applications`, and `configs/services` **without** creating any Kubernetes cluster or ArgoCD. You can optionally **add the repository to an existing ArgoCD** using a server URL and token.

**See also:** [PREDATOR_SETUP.md](./PREDATOR_SETUP.md) for full local development setup (cluster, ArgoCD, Docker Compose).

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Create a GitHub App](#create-a-github-app)
3. [Run the Script](#run-the-script)
4. [Add Repository to ArgoCD (Optional)](#add-repository-to-argocd-optional)
5. [Manual ArgoCD Repo Add with URL and Token](#manual-argocd-repo-add-with-url-and-token)

---

## Prerequisites

- **Git** installed
- **jq** (e.g. `brew install jq` or `apt install jq`)
- **openssl** (usually pre-installed)
- This repository (BharatMLStack) cloned locally; run the script from `quick-start/`
- A **GitHub repository** (empty or existing) where you want the Predator chart and configs
- A **GitHub App** with access to that repository (see below)

---

## Create a GitHub App

You must create a GitHub App and install it on your repository so the script can clone and push via an installation token. The steps below are based on [PREDATOR_SETUP.md Step 2–5](./PREDATOR_SETUP.md#step-2-create-a-github-app).

### 2.1 Navigate to GitHub App Settings

1. Go to https://github.com/settings/apps
2. Click **New GitHub App** (top right)

### 2.2 Basic Information

- **GitHub App name**: e.g. `horizon-bot`
- **Homepage URL**: `https://github.com` (required)
- **User authorization callback URL**: Leave empty
- **Webhook URL / secret**: Leave empty (optional)

### 2.3 Permissions

- **Repository permissions**
  - **Contents**: **Read and write** (required for clone/push)
  - **Metadata**: Read-only (auto)
- **Account permissions**: No access

### 2.4 Where the app can be installed

- **Only on this account** or **Any account**, as needed

### 2.5 Create the app

Click **Create GitHub App**. Note the **App ID** on the app’s settings page.

### 2.6 Install the app on your repository

1. In the app settings, go to **Install App**
2. Click **Install** for your user/org
3. Select the repository that will hold the Helm chart
4. After installation, the URL will look like:  
   `https://github.com/settings/installations/100732634`  
   The number at the end is your **Installation ID**.

### 2.7 Generate a private key

1. In the app settings, under **Private keys**, click **Generate a private key**
2. Save the downloaded `.pem` file (e.g. `github.pem` or `horizon/configs/github.pem`)

You now have:

- **App ID**
- **Installation ID**
- **Private key file path**

---

## Run the Script

From the **BharatMLStack** repo, inside `quick-start/`:

```bash
cd quick-start
./setup-predator-github.sh
```

The script runs **interactively** (like `setup-predator-k8s.sh`): it prompts for each value one by one:

1. **Repository URL** (e.g. `https://github.com/username/repo.git`)
2. **GitHub App ID**
3. **GitHub App Installation ID**
4. **GitHub App Private Key path** (e.g. `horizon/configs/github.pem`)
5. **Branch to push to** (default: `main`)
6. After pushing, **Add this repository to ArgoCD? (y/n)** — if yes, then **ArgoCD server URL** and **ArgoCD auth token**

You can still pass options on the command line to pre-fill or skip prompts: `--repo-url`, `--app-id`, `--installation-id`, `--private-key`, `--branch`, `--repo-name`, `--argocd-server`, `--argocd-token`. See `./setup-predator-github.sh --help`.

The script will:

1. Obtain a GitHub App installation token
2. Clone your repository (specified branch or default)
3. Copy into the repo:
   - `predator/1.0.0` → `1.0.0/`
   - `prd/applications/` (with `.gitkeep`)
   - `horizon/configs/services` → `configs/services/`
4. Commit and push

No Docker, Kubernetes, or ArgoCD are required for this step.

---

## Add Repository to ArgoCD (Optional)

If you have an **existing ArgoCD** and want the script to **register the repository** there, pass the ArgoCD server URL and an auth token. The script will use the ArgoCD CLI to add the repo with the same GitHub App credentials.

**Requirements:**

- **ArgoCD CLI** installed (e.g. `brew install argocd` or [Argo CD CLI install](https://argo-cd.readthedocs.io/en/stable/cli_installation/))
- ArgoCD server reachable (e.g. port-forward: `kubectl port-forward svc/argocd-server -n argocd 8087:443`)
- An **ArgoCD token** (from ArgoCD UI: User Info → Generate New Token, or `argocd account generate-token`)

**Example:**

```bash
./setup-predator-github.sh \
  --repo-url https://github.com/myorg/my-repo.git \
  --app-id 123 \
  --installation-id 456 \
  --private-key ./github.pem \
  --argocd-server https://localhost:8087 \
  --argocd-token <YOUR_ARGOCD_TOKEN>
```

The script will:

1. Populate the GitHub repo (as above)
2. Log in to ArgoCD with `argocd login <server> --auth-token <token> --insecure`
3. Run `argocd repo add <repo> --name <name> --type git --github-app-id ... --github-app-installation-id ... --github-app-private-key-path ... --insecure`

So **yes, you can create (register) a repository in ArgoCD at a given ArgoCD URL and token** in the same way: use `--argocd-server` and `--argocd-token`. The repo is added with GitHub App auth (no need to store a PAT in ArgoCD).

---

## Manual ArgoCD Repo Add with URL and Token

If you prefer not to use the script for ArgoCD, or the script’s ArgoCD step failed:

1. **Log in with token:**
   ```bash
   argocd login <ARGOCD_SERVER> --auth-token <YOUR_TOKEN> --insecure
   ```
   Example: `argocd login localhost:8087 --auth-token eyJ... --insecure`

2. **Add the repository (GitHub App):**
   ```bash
   argocd repo add https://github.com/<USER>/<REPO>.git \
     --name <REPO_NAME> \
     --type git \
     --github-app-id <GITHUB_APP_ID> \
     --github-app-installation-id <GITHUB_INSTALLATION_ID> \
     --github-app-private-key-path <PATH_TO_PEM> \
     --insecure
   ```

3. **Verify:**
   ```bash
   argocd repo list
   argocd repo get https://github.com/<USER>/<REPO>.git --insecure
   ```

---

## Summary

| Goal                         | Command / step |
|-----------------------------|----------------|
| Populate GitHub repo only   | `./setup-predator-github.sh --repo-url ... --app-id ... --installation-id ... --private-key ...` |
| Same + add repo to ArgoCD   | Add `--argocd-server <URL>` and `--argocd-token <TOKEN>` |
| ArgoCD repo add by hand     | Use [Manual ArgoCD Repo Add](#manual-argocd-repo-add-with-url-and-token) with your ArgoCD URL and token |

Prerequisites for the script: GitHub App (Steps 2–5 in this doc), `jq`, and (for ArgoCD step) ArgoCD CLI and token.
