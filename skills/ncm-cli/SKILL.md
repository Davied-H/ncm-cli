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

If working inside the `ncm-cli` repository, install from the local package:

```bash
npx . install --dir ~/.local/bin
```

If the package has been published to npm, install remotely:

```bash
npx ncm-cli@latest install --dir ~/.local/bin
```

If `~/.local/bin` is not on `PATH`, either add it for the user or run the installed binary by absolute path.

The Go Playwright driver is required because `ncm login` opens NetEase Cloud Music Web in Chrome, and most CLI features require login state. Install it before login:

```bash
go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 install chromium
```

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
```

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
2. Run `ncm --help` to confirm the binary works.
3. If the user asks for account data and login is missing, run `ncm login` and wait for the user to complete browser login.
4. Prefer `--json` for parsing and automation.
5. Use table output for user-facing summaries.
6. If a command fails due to Web API changes, inspect the repository docs and run read-only exploration before changing code.

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

If the user wants to install this skill itself, use the skills tool against the repository. This installs the agent skill only; it does not install the `ncm` CLI binary:

```bash
npx skills add <repo-url-or-owner/repo> --skill ncm-cli --full-depth
```

For local testing from this repository:

```bash
npx skills add . --skill ncm-cli --full-depth --copy -y
```

After installing or updating a skill, tell the user to restart Claude Code/Codex if their client requires restart to reload skills.

## Safety

- Do not expose login state or encrypted request fields.
- Do not commit `.ncm/`, `~/.config/ncm-cli/`, or generated login files.
- Write operations are not part of the current CLI surface; do not invent write commands.
- Playback URL availability depends on copyright, membership, region, and login state.
