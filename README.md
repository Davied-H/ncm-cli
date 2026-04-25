# ncm-cli

`ncm-cli` 是一个面向个人使用的网易云音乐命令行工具。当前已实现登录态校验、账号信息、歌单浏览与管理、歌曲、歌词和搜索。

项目使用 Go 实现 CLI，复用网易云 Web 端的 `weapi` 调用模型。播放 URL 解析、每日推荐和播放记录暂未接入。桌面端播放通过网易云音乐的 `orpheus://` URL Scheme 调起本机客户端。

适合这些场景：

- 在终端快速查看当前网易云账号、歌单和歌曲信息。
- 搜索歌曲或歌单，把结果用 `--json` 交给脚本继续处理。
- 创建和整理自己的歌单：加歌、移除歌曲、重命名、改标签、改描述、删除测试歌单。
- 在 macOS 上把歌曲推送给网易云音乐桌面端播放。
- 给自动化脚本、Agent 或个人工具链提供稳定的网易云音乐 CLI 入口。

如果要让 Claude Code 或 Codex 代为快速安装，请把下面这段任务发给它：

```text
请自动安装 ncm-cli。先访问并阅读安装文档：
https://github.com/Davied-H/ncm-cli/blob/main/CLAUDE_CODE_CODEX_INSTALL.md

按文档步骤执行安装：安装或更新 ncm-cli Skill，安装 ncm CLI 和必需的 Playwright driver，运行 `ncm --help` 与 `ncm version --json` 验证，并用 GitHub `check-update` 检测是否需要更新。需要登录时执行 `ncm login`，并等待我扫码或完成网页登录。不要输出或提交任何 cookie、csrf、storage-state.json 等登录态文件。
```

本仓库也包含代理 Skill：`skills/ncm-cli/SKILL.md`。发布到 GitHub 后，可用下面的方式安装到 Claude Code/Codex：

```bash
npx skills add Davied-H/ncm-cli --skill ncm-cli --full-depth -g -y
```

## 已实现功能

账号与登录：

- `ncm login`：打开网易云 Web 登录，保存本地登录态。
- `ncm me`：查看当前登录账号，支持 `--json`。

歌单浏览：

- `ncm playlist list`：列出当前账号或指定用户的歌单，区分自建和收藏。
- `ncm playlist show`：查看歌单歌曲、歌手、专辑、时长和播放权限。

歌单管理：

- `ncm playlist create`：创建公开或私密歌单。
- `ncm playlist add/remove`：批量添加或移除歌曲。
- `ncm playlist rename/tags/desc`：更新歌单名称、标签和描述。
- `ncm playlist delete`：删除当前账号自建歌单。
- 写操作默认只允许操作当前账号自建的普通歌单；移除歌曲和删除歌单默认需要确认，可用 `--yes` 做非交互脚本。

歌曲、歌词与搜索：

- `ncm song`：查看歌曲元数据和权限信息。
- `ncm lyric`：查看歌词，支持 `--raw` 只输出原始歌词。
- `ncm search suggest/song/playlist`：搜索建议、歌曲搜索和歌单搜索。

桌面端播放与自动化：

- `ncm play`：通过 macOS 网易云音乐桌面端播放指定歌曲。
- 所有主要读取命令和歌单写命令支持 `--json`，便于脚本和 Agent 解析。
- `ncm version`：输出版本和构建 commit，安装器可据此检查更新。

命令速查：

```bash
ncm login
ncm me
ncm playlist list [--uid <uid>] [--limit 100] [--offset 0] [--json]
ncm playlist show <playlist-id> [--limit 1000] [--json]
ncm playlist create <name> [--private] [--json]
ncm playlist add <playlist-id> <song-id...> [--json]
ncm playlist remove <playlist-id> <song-id...> [--yes] [--json]
ncm playlist delete <playlist-id> [--yes] [--json]
ncm playlist rename <playlist-id> <name> [--json]
ncm playlist tags <playlist-id> <tag...> [--json]
ncm playlist desc <playlist-id> <text> [--json]
ncm song <song-id> [--json]
ncm play <song-id> [--print-url]
ncm lyric <song-id> [--raw]
ncm search suggest <keyword> [--json]
ncm search song <keyword> [--limit 30] [--offset 0] [--json]
ncm search playlist <keyword> [--limit 30] [--offset 0] [--json]
ncm version [--json]
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
- Go Playwright driver。默认使用系统 Chrome，不会下载 Playwright Chromium 浏览器包

发布到 GitHub 后，推荐直接从 GitHub 安装 CLI 和必需的 Playwright driver：

```bash
npx --yes github:Davied-H/ncm-cli install --dir ~/.local/bin --with-playwright-driver
```

国内网络环境推荐显式使用镜像：

```bash
PLAYWRIGHT_DOWNLOAD_HOST=https://npmmirror.com/mirrors/playwright \
GOPROXY=https://goproxy.cn,direct \
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

检查本机 CLI 是否落后于 GitHub `main` 最新版本：

```bash
npx --yes github:Davied-H/ncm-cli check-update --dir ~/.local/bin --json
```

从 GitHub 拉取最新源码并更新本机 CLI：

```bash
PLAYWRIGHT_DOWNLOAD_HOST=https://npmmirror.com/mirrors/playwright \
GOPROXY=https://goproxy.cn,direct \
npx --yes github:Davied-H/ncm-cli update --dir ~/.local/bin --with-playwright-driver
```

也可以单独安装 Playwright driver：

```bash
PLAYWRIGHT_DOWNLOAD_HOST=https://npmmirror.com/mirrors/playwright \
GOPROXY=https://goproxy.cn,direct \
go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 --version
```

上面的命令只会准备 Go Playwright driver 并输出版本，不会下载 Chromium。如果机器没有 Chrome，可以额外安装 Playwright Chromium：

```bash
PLAYWRIGHT_DOWNLOAD_HOST=https://npmmirror.com/mirrors/playwright \
GOPROXY=https://goproxy.cn,direct \
go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 install chromium
```

或在安装 CLI 时一并安装：

```bash
PLAYWRIGHT_DOWNLOAD_HOST=https://npmmirror.com/mirrors/playwright \
GOPROXY=https://goproxy.cn,direct \
npx --yes github:Davied-H/ncm-cli install --dir ~/.local/bin --with-playwright-browser
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

管理歌单：

```bash
go run ./cmd/ncm playlist create "我的新歌单" --private
go run ./cmd/ncm playlist add 17924063236 210049 29816860
go run ./cmd/ncm playlist remove 17924063236 210049
go run ./cmd/ncm playlist rename 17924063236 "新的歌单名"
go run ./cmd/ncm playlist tags 17924063236 华语 流行
go run ./cmd/ncm playlist desc 17924063236 "歌单描述"
go run ./cmd/ncm playlist delete 17924063236
```

创建歌单和更新描述依赖网易云页面运行时的 `checkToken`，命令会短暂复用登录时的 Chrome profile。

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
internal/checktoken/ 页面运行时 checkToken 获取
internal/ncm/        API client 和接口封装
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
- 歌单写操作默认只允许操作当前账号自建歌单，不操作收藏歌单或特殊歌单。
- 删除歌单、移除歌曲默认需要确认；脚本场景可显式传 `--yes`。
- 播放 URL 解析受版权、会员、地区和登录态影响，不能假定一定可播放。
- 桌面端播放使用 `orpheus://base64(json)`，不是移动端常见的 `orpheus://song/<id>`。
