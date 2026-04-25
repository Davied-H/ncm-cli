# AGENTS.md

本文件给 Codex、Claude Code 等代码代理使用。README 面向 CLI 使用者和开发者；代理交接、接口复验和安全边界放在这里。

## 工作入口

接手本项目时按顺序阅读：

1. `README.md`：了解当前 CLI 能力、使用方式和开发命令。
2. `docs/ncm-api-exploration.md`：了解已验证过的网易云 Web API。
3. `docs/ncm-cli-plan.md`：了解 CLI 命令设计和后续阶段。
4. `scripts/ncm-explore-api.mjs`：需要复验接口时再读脚本细节。

## 当前状态

- Go module 已生成，命令名为 `ncm`。
- 第一版只读命令已实现：`login`、`me`、`playlist list/show`、`song`、`lyric`、`search suggest/song/playlist`。
- 主要实现目录：
  - `cmd/ncm/`
  - `internal/config/`
  - `internal/crypto/`
  - `internal/login/`
  - `internal/ncm/`
  - `internal/output/`
- Node 脚本仍保留，用于登录态探索和 Web API 复验。

## 实现约定

- Go CLI 使用 `cobra` 管命令结构。
- HTTP 请求使用标准库 `net/http`、`cookiejar` 和 `encoding/json`。
- 默认配置目录解析顺序：`--config-dir` > `NCM_CONFIG_DIR` > `~/.config/ncm-cli`。
- 只读命令如果默认配置目录无登录态，可兼容读取项目内 `.ncm/session/storage-state.json`。
- `internal/ncm.Client.WeAPI` 负责：
  - 从 cookie jar/登录态读取 `__csrf`
  - 给 payload 补 `csrf_token`
  - 将 `/api/...` 规范化为 `/weapi/...`
  - 生成 `params` 和 `encSecKey`
  - POST 到 `https://music.163.com/weapi/...?...`
  - 处理 HTTP 错误、JSON 解析错误和 `code != 200` 业务错误

## 接口复验

网易云 Web API 不稳定。新增接口、接口报错或准备做写操作前，优先运行只读探索：

```bash
npm run explore:read
```

需要验证写操作时，优先使用带清理的探索：

```bash
npm run explore:cleanup
```

探索输出位于 `.ncm/explore/`，可以辅助判断接口形态，但不要把敏感字段写进仓库文档。

## 验证命令

基础验证：

```bash
go test ./...
go build -o /tmp/ncm-cli-ncm ./cmd/ncm
```

只读端到端验证：

```bash
go run ./cmd/ncm me --json
go run ./cmd/ncm playlist list --json
go run ./cmd/ncm playlist show 490155105 --limit 2
go run ./cmd/ncm song 210049 --json
go run ./cmd/ncm lyric 210049 --raw
go run ./cmd/ncm search suggest 周杰伦 --json
go run ./cmd/ncm search song 周杰伦 --limit 3 --json
go run ./cmd/ncm search playlist 周杰伦 --limit 2 --json
```

## 安全边界

- 不提交 `.ncm/`、`~/.config/ncm-cli/` 或任何真实登录态。
- 不把真实 cookie、csrf、`params`、`encSecKey`、`checkToken` 写入 README、docs、测试快照或日志。
- 写操作尚未实现；后续实现时默认只允许操作当前账号自建歌单。
- 删除歌单、批量删除、移除歌曲等命令必须有确认机制。
- 创建歌单和更新描述依赖页面运行时 `checkToken`，历史探索来源为 `nm.x.lc1x(callback)`。
- 播放 URL 受版权、会员、地区和登录态影响，不能假定一定可播放。
- 如果后续接入桌面端播放，Web 端同款 URL Scheme 是 `orpheus://base64(json)`，不是移动端常见的 `orpheus://song/<id>`。
