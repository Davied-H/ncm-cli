---
name: ncm-cli
description: Use this skill when Claude Code, Codex, or another coding agent needs to install, configure, validate, or operate the ncm-cli NetEase Cloud Music command-line tool; when the user asks to query their NetEase Cloud Music account, playlists, songs, lyrics, search results, or desktop playback through the CLI; or when the user asks to install this skill with npx skills.
---

# ncm-cli

Use `ncm-cli` to work with a user's NetEase Cloud Music account from the terminal. The CLI is implemented in Go and exposes a binary named `ncm`.

## Install

First check whether `ncm` already exists:

```bash
command -v ncm && ncm --help
```

If installing from the published GitHub repository, use:

```bash
PLAYWRIGHT_DOWNLOAD_HOST=https://npmmirror.com/mirrors/playwright \
GOPROXY=https://goproxy.cn,direct \
npx --yes github:Davied-H/ncm-cli install --dir ~/.local/bin --with-playwright-driver
```

The published GitHub repository is `Davied-H/ncm-cli`.

If working inside the `ncm-cli` repository, install from the local package:

```bash
PLAYWRIGHT_DOWNLOAD_HOST=https://npmmirror.com/mirrors/playwright \
GOPROXY=https://goproxy.cn,direct \
npx . install --dir ~/.local/bin --with-playwright-driver
```

If the package has also been published to npm, install remotely:

```bash
PLAYWRIGHT_DOWNLOAD_HOST=https://npmmirror.com/mirrors/playwright \
GOPROXY=https://goproxy.cn,direct \
npx ncm-cli@latest install --dir ~/.local/bin --with-playwright-driver
```

If `~/.local/bin` is not on `PATH`, either add it for the user or run the installed binary by absolute path.

Keep `--with-playwright-driver` in the install command. The Go Playwright driver is required because `ncm login` opens NetEase Cloud Music Web in Chrome, and most CLI features require login state. This installs the Playwright Go driver only; it does not download Playwright Chromium.

If Chrome is unavailable on the machine, install the bundled Playwright Chromium browser instead:

```bash
PLAYWRIGHT_DOWNLOAD_HOST=https://npmmirror.com/mirrors/playwright \
GOPROXY=https://goproxy.cn,direct \
npx --yes github:Davied-H/ncm-cli install --dir ~/.local/bin --with-playwright-browser
```

## Check for Updates

When the user asks to install, update, or verify whether their `ncm` CLI is current, check GitHub before assuming the local binary is up to date:

```bash
npx --yes github:Davied-H/ncm-cli check-update --dir ~/.local/bin --json
```

This compares the installed `ncm version --json` output with the latest `package.json` version and latest `main` commit from `Davied-H/ncm-cli` on GitHub.

If an update is available, reinstall from the latest GitHub source:

```bash
PLAYWRIGHT_DOWNLOAD_HOST=https://npmmirror.com/mirrors/playwright \
GOPROXY=https://goproxy.cn,direct \
npx --yes github:Davied-H/ncm-cli update --dir ~/.local/bin --with-playwright-driver
```

Use `--with-playwright-browser` instead of `--with-playwright-driver` only when Chrome is unavailable. Use `--force` only when the user explicitly wants to reinstall even though the check reports no update.

## Login

Run:

```bash
ncm login
```

This opens NetEase Cloud Music Web in Chrome. The user must scan or complete login. Login state is stored under:

```text
~/.config/ncm-cli/
```

Never print, copy, summarize, or commit cookies, csrf values, `storage-state.json`, `params`, `encSecKey`, or `checkToken`.

## Common Commands

Account:

```bash
ncm me
ncm me --json
```

Playlists:

```bash
ncm playlist list
ncm playlist list --json
ncm playlist show <playlist-id> --limit 20
ncm playlist show <playlist-id> --json
ncm playlist create <name> [--private] [--json]
ncm playlist add <playlist-id> <song-id...> [--json]
ncm playlist remove <playlist-id> <song-id...> [--yes] [--json]
ncm playlist delete <playlist-id> [--yes] [--json]
ncm playlist rename <playlist-id> <name> [--json]
ncm playlist tags <playlist-id> <tag...> [--json]
ncm playlist desc <playlist-id> <text> [--json]
```

`playlist create` and `playlist desc` briefly reuse the Chrome profile to obtain the NetEase Web runtime `checkToken`. Never print that token.

Songs and lyrics:

```bash
ncm song <song-id>
ncm song <song-id> --json
ncm lyric <song-id> --raw
```

Search:

```bash
ncm search suggest <keyword>
ncm search song <keyword> --limit 10
ncm search playlist <keyword> --limit 10
```

Desktop playback on macOS:

```bash
ncm play <song-id>
ncm play <song-id> --print-url
```

The desktop playback command uses the NetEase Cloud Music `orpheus://base64(json)` URL scheme.

## Agent Workflow

1. Install or locate `ncm`.
2. When the user asks for installation, update, or stale-version diagnosis, run `check-update` from GitHub and update if needed.
3. Run `ncm --help` and `ncm version --json` to confirm the binary works and reports version metadata.
4. If the user asks for account data and login is missing, run `ncm login` and wait for the user to complete browser login.
5. Prefer `--json` for parsing and automation.
6. Use table output for user-facing summaries.
7. If a command fails due to Web API changes, inspect the repository docs and run read-only exploration before changing code.

## Development and Validation

Inside the repository:

```bash
go test ./...
go build -o /tmp/ncm-cli-ncm ./cmd/ncm
```

Read-only end-to-end checks:

```bash
go run ./cmd/ncm me --json
go run ./cmd/ncm playlist list --json
go run ./cmd/ncm search song 周杰伦 --limit 3 --json
go run ./cmd/ncm lyric 210049 --raw
```

API exploration scripts:

```bash
npm run explore:read
npm run explore:cleanup
```

Use `explore:read` first. Only run write exploration when the user explicitly wants interface verification for write operations.

## npx skills Installation

If the user wants to install this skill itself, use the skills tool against the GitHub repository. This installs the agent skill only; it does not install the `ncm` CLI binary:

```bash
npx skills add Davied-H/ncm-cli --skill ncm-cli --full-depth -g -y
```

For a full GitHub-based setup, install the skill first, then install the CLI:

```bash
npx skills add Davied-H/ncm-cli --skill ncm-cli --full-depth -g -y
PLAYWRIGHT_DOWNLOAD_HOST=https://npmmirror.com/mirrors/playwright \
GOPROXY=https://goproxy.cn,direct \
npx --yes github:Davied-H/ncm-cli install --dir ~/.local/bin --with-playwright-driver
```

To refresh both the agent skill and the CLI from GitHub:

```bash
npx skills add Davied-H/ncm-cli --skill ncm-cli --full-depth -g -y
PLAYWRIGHT_DOWNLOAD_HOST=https://npmmirror.com/mirrors/playwright \
GOPROXY=https://goproxy.cn,direct \
npx --yes github:Davied-H/ncm-cli update --dir ~/.local/bin --with-playwright-driver
```

For local testing from this repository:

```bash
npx skills add . --skill ncm-cli --full-depth --copy -y
```

After installing or updating a skill, tell the user to restart Claude Code/Codex if their client requires restart to reload skills.

## Safety

- Do not expose login state or encrypted request fields.
- Do not commit `.ncm/`, `~/.config/ncm-cli/`, or generated login files.
- Playlist write operations only support the current account's own regular playlists.
- Removing songs and deleting playlists require confirmation by default; use `--yes` only when the user explicitly wants non-interactive execution.
- Playback URL availability depends on copyright, membership, region, and login state.
