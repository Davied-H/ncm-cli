---
name: ncm-playlist-organizer
description: Use this skill when a user wants to organize, split, audit, deduplicate, or clean up a large NetEase Cloud Music playlist with ncm-cli. It exports read-only analysis reports, suggested playlist buckets, duplicate candidates, and optional copy plans before any playlist write operation.
---

# ncm-playlist-organizer

Use this skill to turn a large NetEase Cloud Music playlist into a reviewable organization plan. The bundled script is read-only: it fetches playlist metadata, classifies songs, and writes local reports. Use `ncm` for account lookup and any later playlist creation/copying.

## Workflow

1. Confirm `ncm` works:

```bash
command -v ncm && ncm me --json && ncm playlist list --json
```

If `ncm` is missing or login is required, use the `ncm-cli` skill first.

2. Identify the target playlist. If the user says "全部歌单", "ALL", "all songs", or similar, prefer the largest playlist owned by the current account unless the user names a specific playlist.

3. Run the read-only organizer script from this skill:

```bash
node <skill-dir>/scripts/organize-playlist.mjs \
  --playlist-id <playlist-id> \
  --out-dir <working-dir> \
  --prefix <safe-output-prefix>
```

For an existing `ncm playlist show ... --json` export:

```bash
node <skill-dir>/scripts/organize-playlist.mjs \
  --input playlist.raw.json \
  --out-dir <working-dir> \
  --prefix <safe-output-prefix>
```

The script uses public NetEase playlist/song-detail endpoints when `--playlist-id` is supplied, which avoids reading local cookies and can recover full track lists when `ncm playlist show` is capped at 1000 tracks. If public fetch fails, fall back to `ncm playlist show <id> --limit 2000 --json` and clearly tell the user if the result is incomplete.

4. Review the generated files with the user:

- `<prefix>-analysis.md`: summary, suggested primary buckets, cleanup tags, top artists/albums, duplicate candidates.
- `<prefix>-tracks.csv`: full track table for manual review.
- `<prefix>-primary-buckets.json`: mutually exclusive bucket plan for creating new playlists.
- `<prefix>-tags.json`: overlapping cleanup tags such as Live/version, instrumental/focus, workout, long/short tracks.
- `<prefix>-copy-plan.md`: proposed playlist names and counts.

## Default Classification

Primary buckets are mutually exclusive and designed for playlist creation:

- `日系 / ACG`
- `华语流行`
- `欧美 / 英文`
- `韩语 / K-Pop`
- `纯音 / 阅读 / 专注`
- `运动 / 节奏`
- `未分类候选`

Cleanup tags are overlapping and intended for review rather than direct splitting:

- `Live / 翻唱 / 版本`
- `纯音 / 阅读 / 专注`
- `运动 / 节奏`
- `超长曲目候选`
- `短曲目候选`
- `疑似重复版本`
- `同名不同歌手`

## Optional Write Step

Do not create, delete, rename, or remove songs until the user explicitly confirms the proposed plan.

After confirmation, create new playlists and copy songs in chunks:

```bash
ncm playlist create "ALL · 华语流行" --json
ncm playlist add <new-playlist-id> <song-id...> --json
```

Use chunks of 100-200 song IDs per `playlist add` command. Keep the original playlist intact by default. Only remove duplicate/version candidates or delete playlists when the user explicitly asks for that exact destructive operation.

## Safety

- Never print, copy, or commit cookies, csrf values, `storage-state.json`, `params`, `encSecKey`, or `checkToken`.
- Keep generated reports in the user's working directory, not inside this repository, unless the user explicitly wants fixtures.
- Treat duplicate/version detection as advisory. Show candidates first; do not auto-delete.
- If public APIs return fewer songs than the playlist's `trackCount`, call out the partial result and avoid write operations based on it.
