# ncm-cli

`ncm-cli` 是一个面向个人使用的网易云音乐命令行工具。当前第一版只实现只读能力：登录态校验、账号信息、歌单、歌曲、歌词和搜索。

项目使用 Go 实现 CLI，复用网易云 Web 端的 `weapi` 调用模型。写操作、播放 URL 解析、每日推荐和播放记录暂未接入。桌面端播放通过网易云音乐的 `orpheus://` URL Scheme 调起本机客户端。

如果要让 Claude Code 或 Codex 代为快速安装，请把 [CLAUDE_CODE_CODEX_INSTALL.md](CLAUDE_CODE_CODEX_INSTALL.md) 发给它。

本仓库也包含代理 Skill：`skills/ncm-cli/SKILL.md`。发布到 GitHub 后，可用下面的方式安装到 Claude Code/Codex：

```bash
npx skills add Davied-H/ncm-cli --skill ncm-cli --full-depth -g -y
```

## 功能

已实现命令：

```bash
ncm login
ncm me
ncm playlist list [--uid <uid>] [--limit 100] [--offset 0] [--json]
ncm playlist show <playlist-id> [--limit 1000] [--json]
ncm song <song-id> [--json]
ncm play <song-id> [--print-url]
ncm lyric <song-id> [--raw]
ncm search suggest <keyword> [--json]
ncm search song <keyword> [--limit 30] [--offset 0] [--json]
ncm search playlist <keyword> [--limit 30] [--offset 0] [--json]
```

全局参数：

```bash
--config-dir <path>   配置目录，默认使用 NCM_CONFIG_DIR 或 ~/.config/ncm-cli
--timeout <duration>  请求超时时间，默认 30s
```

## 安装与构建

环境要求：

- Go 1.24+
- Chrome，供 `ncm login` 打开网易云 Web 登录
- Go Playwright driver

发布到 GitHub 后，推荐直接从 GitHub 安装 CLI 和必需的 Playwright driver：

```bash
npx --yes github:Davied-H/ncm-cli install --dir ~/.local/bin --with-playwright-driver
```

在仓库目录内开发时，可以用本地 `npx` 安装：

```bash
npx . install --dir ~/.local/bin --with-playwright-driver
```

如果将来发布到 npm，也可以远程安装：

```bash
npx ncm-cli@latest install --dir ~/.local/bin --with-playwright-driver
```

安装器会编译 Go CLI，并把二进制写入指定目录，默认是 `~/.local/bin/ncm`。由于 CLI 需要登录网易云才能使用主要功能，Playwright driver 是必需依赖，安装命令应保留 `--with-playwright-driver`。

也可以单独安装 Playwright driver：

```bash
go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 install chromium
```

不安装，直接从源码运行：

```bash
go run ./cmd/ncm me
```

构建二进制：

```bash
go build -o bin/ncm ./cmd/ncm
```

## 登录

首次使用先登录：

```bash
go run ./cmd/ncm login
```

登录会打开网易云音乐 Web 页面。扫码或完成网页登录后，CLI 会保存登录态到：

```text
~/.config/ncm-cli/chrome-profile
~/.config/ncm-cli/session/storage-state.json
~/.config/ncm-cli/session/user.json
```

这些文件包含真实登录态，不要提交或分享。只读命令默认读取 `~/.config/ncm-cli/`；如果该目录没有登录态，开发环境下会兼容读取项目内 `.ncm/session/storage-state.json`。

## 使用示例

查看当前账号：

```bash
go run ./cmd/ncm me
go run ./cmd/ncm me --json
```

查看歌单：

```bash
go run ./cmd/ncm playlist list
go run ./cmd/ncm playlist show 490155105 --limit 20
```

查询歌曲和歌词：

```bash
go run ./cmd/ncm song 210049
go run ./cmd/ncm play 210049 --print-url
go run ./cmd/ncm lyric 210049 --raw
```

搜索：

```bash
go run ./cmd/ncm search suggest 周杰伦
go run ./cmd/ncm search song 周杰伦 --limit 3
go run ./cmd/ncm search playlist 周杰伦 --limit 3
```

需要脚本处理时使用 `--json`，例如：

```bash
go run ./cmd/ncm search song 周杰伦 --limit 3 --json
```

## 开发

主要目录：

```text
cmd/ncm/             CLI 入口和命令定义
internal/config/     配置目录、storage-state 读取、cookie 解析
internal/crypto/     网易云 Web weapi 加密
internal/login/      Go Playwright 登录流程
internal/ncm/        API client 和只读接口封装
internal/output/     JSON/table 输出工具
docs/                接口探索记录和 CLI 规划
scripts/             Node Playwright 登录/探索脚本
```

常用开发命令：

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
go run ./cmd/ncm play 29816860 --print-url
go run ./cmd/ncm lyric 210049 --raw
go run ./cmd/ncm search suggest 周杰伦 --json
go run ./cmd/ncm search song 周杰伦 --limit 3 --json
go run ./cmd/ncm search playlist 周杰伦 --limit 2 --json
```

## API 探索

网易云 Web API 不是公开稳定 API。接口报错、准备新增能力或调整请求层前，建议先复验接口。

只读探索：

```bash
npm run explore:read
```

带写操作和清理的探索：

```bash
npm run explore:cleanup
```

探索输出位于 `.ncm/explore/`。文档只记录接口形态和结论，不记录真实 cookie、csrf、`params`、`encSecKey`、`checkToken` 等敏感值。

## 安全边界

- 不提交 `.ncm/`、`~/.config/ncm-cli/` 或任何真实登录态。
- 不在文档、日志和普通输出中保留 cookie、csrf、`params`、`encSecKey`、`checkToken`。
- 写操作尚未接入；后续实现时应默认只允许操作当前账号自建歌单，并为删除类命令增加确认机制。
- 播放 URL 解析受版权、会员、地区和登录态影响，不能假定一定可播放。
- 桌面端播放使用 `orpheus://base64(json)`，不是移动端常见的 `orpheus://song/<id>`。
