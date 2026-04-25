# 网易云音乐 CLI 规划

## 目标

做一个面向个人使用的网易云音乐命令行工具，优先支持登录态读取、歌单浏览、歌曲元数据/歌词查询，后续再扩展播放地址、收藏/歌单管理等写操作。

本轮接口探索使用 Playwright 真实打开 `https://music.163.com`，登录账号后抓取运行时请求。项目内已沉淀登录和探索脚本：

- `scripts/ncm-login.mjs`
- `scripts/ncm-explore-api.mjs`
- `docs/ncm-api-exploration.md`

仓库文档只记录接口形态，不保存真实 cookie、csrf token、`params`、`encSecKey`。

登录态和探索输出保存于项目内 `.ncm/`，该目录已加入 `.gitignore`。

## Web 端调用模型

网易云 Web 页面运行时确认：

- `window.GEnc === true`
- `window.NEJ_CONF.p_csrf = { cookie: "__csrf", param: "csrf_token" }`
- 页面代码通常先声明 `/api/...`，运行时统一改写成 `/weapi/...`
- 实际请求为 `POST application/x-www-form-urlencoded`
- 请求体为 `params=<encrypted>&encSecKey=<encrypted>`
- 明文 payload 会带 `csrf_token`，URL query 也会带 `?csrf_token=<cookie __csrf>`

CLI 需要实现 Web 加密层，或者复用已验证的网易云 Web 加密实现。请求发送时必须维护 cookie jar，登录后至少依赖 `MUSIC_U` 和 `__csrf` 一类 cookie。

## 已验证接口

### 当前账号

```text
POST https://music.163.com/weapi/w/nuser/account/get?csrf_token=<csrf>
```

明文 payload：

```json
{
  "csrf_token": "<csrf>"
}
```

返回里 `profile.userId` 是后续获取个人歌单的 `uid`。

### 用户歌单列表

```text
POST https://music.163.com/weapi/user/playlist?csrf_token=<csrf>
```

明文 payload：

```json
{
  "uid": "<current-user-id>",
  "limit": 100,
  "offset": 0,
  "includeVideo": true,
  "csrf_token": "<csrf>"
}
```

已登录账号实测返回：

- `code: 200`
- 歌单总数：49
- 自建歌单：4
- 收藏歌单：45

关键字段：

- `playlist[].id`
- `playlist[].name`
- `playlist[].userId`
- `playlist[].trackCount`
- `playlist[].subscribed`
- `playlist[].specialType`
- `playlist[].privacy`
- `playlist[].coverImgUrl`

### 歌单详情

```text
POST https://music.163.com/weapi/v6/playlist/detail?csrf_token=<csrf>
```

明文 payload：

```json
{
  "id": 490155105,
  "n": 1000,
  "s": 8,
  "csrf_token": "<csrf>"
}
```

返回结构：

- `playlist.id`
- `playlist.name`
- `playlist.trackCount`
- `playlist.creator.userId`
- `playlist.tracks[]`
- `privileges[]`

歌曲关键字段：

- `tracks[].id`
- `tracks[].name`
- `tracks[].ar[].name`
- `tracks[].al.name`
- `tracks[].dt`
- `tracks[].fee`

权限关键字段：

- `privileges[].id`
- `privileges[].fee`
- `privileges[].pl`
- `privileges[].dl`
- `privileges[].maxbr`

### 其他可用接口

搜索建议：

```text
POST /weapi/search/suggest/web?csrf_token=<csrf>
payload: { "s": "周杰伦", "limit": 8, "csrf_token": "<csrf>" }
```

歌曲详情：

```text
POST /weapi/v3/song/detail?csrf_token=<csrf>
payload: { "c": "[{\"id\":185809}]", "csrf_token": "<csrf>" }
```

歌词：

```text
POST /weapi/song/lyric?csrf_token=<csrf>
payload: { "id": 185809, "lv": -1, "kv": -1, "tv": -1, "csrf_token": "<csrf>" }
```

播放地址：

```text
POST /weapi/song/enhance/player/url/v1?csrf_token=<csrf>
payload: { "ids": "[185809]", "level": "exhigh", "encodeType": "aac", "csrf_token": "<csrf>" }
```

注意：播放地址会受版权、会员、地区、登录态影响，可能返回 `url: null` 或 `code: 404`，CLI 不能假定一定可播放。

桌面端播放：

```text
orpheus://<base64({"type":"song","id":185809,"cmd":"play"})>
```

说明：该格式来自网易云 Web 端「打开客户端」入口，并已在 macOS 网易云音乐桌面端 3.1.6 验证；不要使用移动端常见的 `orpheus://song/<id>`。

搜索结果接口：

```text
POST /weapi/cloudsearch/get/web?csrf_token=<csrf>
payload: { "s": "...", "type": 1, "limit": 30, "offset": 0, "total": true, "csrf_token": "<csrf>" }
```

使用项目内登录态重新验证后，歌曲搜索和歌单搜索均返回 `code: 200`。`type: 1` 为歌曲，`type: 1000` 为歌单。

写操作已通过测试歌单验证：

```text
POST /weapi/playlist/create
payload: { "name": "...", "privacy": 10, "checkToken": "<checkToken>", "csrf_token": "<csrf>" }

POST /weapi/playlist/manipulate/tracks
payload: { "op": "add", "pid": 17924063236, "trackIds": "[210049]", "imme": true, "csrf_token": "<csrf>" }

POST /weapi/playlist/manipulate/tracks
payload: { "op": "del", "pid": 17924174661, "trackIds": "[210049]", "imme": true, "csrf_token": "<csrf>" }

POST /weapi/playlist/update/name
payload: { "id": 17924063236, "name": "...", "csrf_token": "<csrf>" }

POST /weapi/playlist/tags/update
payload: { "id": 17924063236, "tags": "华语", "csrf_token": "<csrf>" }

POST /weapi/playlist/desc/update
payload: { "id": 17924063236, "desc": "...", "checkToken": "<checkToken>", "csrf_token": "<csrf>" }

POST /weapi/playlist/delete
payload: { "pid": 17924174661, "csrf_token": "<csrf>" }
```

创建歌单和更新描述需要页面运行时的 `checkToken`，本轮通过 `nm.x.lc1x(callback)` 获取。
删除歌单当前验证可用的 payload 是 `pid`，`ids` 和 `id` 形态返回 `400`。

## CLI 命令设计

建议命令名先用 `ncm`：

```text
ncm login
ncm me
ncm playlist list [--uid <uid>] [--limit 100] [--offset 0] [--json]
ncm playlist show <playlist-id> [--limit 1000] [--json]
ncm playlist create <name> [--private]
ncm playlist add <playlist-id> <song-id...>
ncm playlist remove <playlist-id> <song-id...>
ncm playlist delete <playlist-id>
ncm playlist rename <playlist-id> <name>
ncm playlist tags <playlist-id> <tag...>
ncm playlist desc <playlist-id> <text>
ncm song <song-id> [--json]
ncm lyric <song-id> [--raw]
ncm search suggest <keyword> [--json]
ncm search song <keyword> [--limit 30] [--offset 0] [--json]
ncm search playlist <keyword> [--limit 30] [--offset 0] [--json]
ncm url <song-id> [--level standard|exhigh|lossless|hires]
ncm recommend songs
ncm record [--week|--all]
```

第一阶段只读功能：

1. `login`：打开二维码登录或复用 cookie 导入。
2. `me`：调用 `/weapi/w/nuser/account/get` 校验登录态。
3. `playlist list`：列出当前用户自建/收藏歌单。
4. `playlist show`：显示歌单歌曲列表和可播放权限。
5. `song` / `lyric`：查询歌曲元数据与歌词。
6. `play`：通过桌面端 URL Scheme 推送歌曲到网易云音乐桌面端播放。

第二阶段增强：

1. 搜索结果分页。
2. 播放地址解析和外部播放器集成。
3. 导出歌单为 JSON/CSV/M3U。
4. 创建歌单、添加/删除歌曲、删除歌单、更新歌单元数据等写操作。

## 技术方案

推荐用 Go 实现 CLI：

- `cobra` 管命令结构。
- `net/http` + cookie jar 管请求。
- `encoding/json` 定义响应 DTO。
- `go-keyring` 或本地配置文件保存 cookie，默认路径可用 `~/.config/ncm-cli/`。
- Web 加密层封装成 `internal/crypto/weapi.go`。
- API client 封装成 `internal/ncm/client.go`。

项目结构：

```text
cmd/ncm/
internal/crypto/
internal/ncm/
internal/config/
internal/output/
docs/
```

核心 client 形态：

```go
type Client struct {
    BaseURL string
    HTTP    *http.Client
    CSRF    string
}

func (c *Client) WeAPI(ctx context.Context, path string, payload any, out any) error
```

`WeAPI` 负责：

1. 从 cookie jar 读取 `__csrf`。
2. 给明文 payload 补 `csrf_token`。
3. 把 `/api/...` 规范化成 `/weapi/...`。
4. 生成 `params` 和 `encSecKey`。
5. POST 到 `https://music.163.com/weapi/...?...`。
6. 处理 `code != 200` 的业务错误。

## 实现顺序

1. 初始化 Go module 和 `ncm` CLI 骨架。
2. 实现 cookie/config 存储。
3. 实现 WeAPI 加密和通用请求。
4. 实现 `me`、`playlist list`、`playlist show`。
5. 加入 JSON/table 两种输出。
6. 补充登录流程和端到端验证。
7. 扩展歌曲、歌词、搜索、导出。

## 风险

- 网易云 Web 接口不是公开稳定 API，路径和加密参数可能变化。
- 登录、搜索、播放地址更容易触发风控。
- 播放 URL 受版权和会员状态影响，CLI 应明确展示不可播放原因。
- 写操作要谨慎，先只做只读接口。
