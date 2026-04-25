# 网易云音乐 Web API 探索记录

本记录来自 Playwright 真实打开 `https://music.163.com` 后，在已登录状态下通过页面运行时调用和抓包确认。仓库文档只保留接口形态，不记录真实 cookie、csrf、`params`、`encSecKey` 或 `checkToken`。

## 本地脚本

登录脚本：

```bash
node scripts/ncm-login.mjs
```

行为：

- 使用项目内 `.ncm/chrome-profile` 作为 Playwright 持久化浏览器目录。
- 如果已有登录态，会直接调用账号接口验证。
- 如果未登录，会打开网易云 Web 登录窗口，等待扫码或网页登录。
- 登录成功后保存：
  - `.ncm/session/storage-state.json`
  - `.ncm/session/user.json`

探索脚本：

```bash
node scripts/ncm-explore-api.mjs
```

可选环境变量：

```bash
NCM_SEARCH_KEYWORD=林俊杰 node scripts/ncm-explore-api.mjs
NCM_SKIP_WRITES=1 node scripts/ncm-explore-api.mjs
NCM_CLEANUP_WRITES=1 node scripts/ncm-explore-api.mjs
NCM_HEADLESS=1 node scripts/ncm-explore-api.mjs
```

行为：

- 复用 `.ncm/chrome-profile` 登录态。
- Hook `window.asrsea` 和 XHR，记录 `/api` 到 `/weapi` 的实际请求形态。
- 输出到 `.ncm/explore/ncm-api-explore-<timestamp>.json`。
- 默认会创建一个 `ncm-cli-api-test-<timestamp>` 私密测试歌单，并在该测试歌单里验证加歌、改名、改标签、改描述。
- 设置 `NCM_SKIP_WRITES=1` 时只做只读探索。
- 设置 `NCM_CLEANUP_WRITES=1` 时，会在验证后移除测试歌曲并删除本次新建的测试歌单。

最近一次写操作探索输出：

```text
.ncm/explore/ncm-api-explore-2026-04-25T13-04-29-246Z.json
```

最近一次只读复验输出：

```text
.ncm/explore/ncm-api-explore-2026-04-25T13-07-45-734Z.json
```

最近一次带清理的写操作探索输出：

```text
.ncm/explore/ncm-api-explore-2026-04-25T13-10-14-381Z.json
```

## Web 调用模型

页面运行时确认：

- 页面代码通常使用 `/api/...`，运行时实际请求变成 `/weapi/...`。
- 请求方式为 `POST application/x-www-form-urlencoded`。
- 请求体为 `params=<encrypted>&encSecKey=<encrypted>`。
- URL query 和明文 payload 都带 `csrf_token`。
- `checkToken` 可通过页面运行时 `nm.x.lc1x(callback)` 获取，创建歌单和更新描述会用到。

CLI 实现时需要实现 Web 端 `weapi` 加密层，或者先复用成熟实现，再封装稳定的 API client。

## 已验证结果

账号：

- 账号昵称：`<nickname>`
- userId：`<current-user-id>`
- 账号接口 `code: 200`

搜索：

- 关键词：`周杰伦`
- `/weapi/search/suggest/web`：`code: 200`
- `/weapi/search/suggest/multimatch`：`code: 200`
- `/weapi/cloudsearch/get/web` 歌曲搜索：`code: 200`，`songCount: 272`
- `/weapi/cloudsearch/get/web` 歌单搜索：`code: 200`

歌单：

- `/weapi/user/playlist`：`code: 200`
- 返回歌单总数：49
- 自建歌单：4
- 收藏歌单：45
- `/weapi/v6/playlist/detail`：`code: 200`

歌曲：

- 使用搜索首条歌曲 `210049` 验证。
- `/weapi/v3/song/detail`：`code: 200`
- `/weapi/song/lyric`：`code: 200`
- `/weapi/song/enhance/player/url/v1`：`code: 200`

推荐和播放记录：

- `/weapi/v2/discovery/recommend/songs`：`code: 200`
- `/weapi/v1/play/record`：`code: 200`

写操作：

- `/weapi/playlist/create`：`code: 200`
- 创建测试歌单 ID：`17924063236`
- `/weapi/playlist/manipulate/tracks` 添加歌曲：`code: 200`
- `/weapi/playlist/update/name`：`code: 200`
- `/weapi/playlist/tags/update`：`code: 200`
- `/weapi/playlist/desc/update`：`code: 200`
- `/weapi/playlist/manipulate/tracks` 移除歌曲：`code: 200`
- `/weapi/playlist/delete` 删除歌单：`code: 200`
- 本轮创建的 `ncm-cli-api-test-*` 测试歌单已清理，复查列表无残留。

## 接口清单

### 当前账号

```text
POST /weapi/w/nuser/account/get?csrf_token=<csrf>
payload: { "csrf_token": "<csrf>" }
```

关键返回：

- `profile.userId`
- `profile.nickname`

CLI 用途：

- `ncm me`
- 登录态校验
- 后续默认 `uid`

### 用户歌单列表

```text
POST /weapi/user/playlist?csrf_token=<csrf>
payload: {
  "uid": "<current-user-id>",
  "limit": 100,
  "offset": 0,
  "includeVideo": true,
  "csrf_token": "<csrf>"
}
```

关键返回：

- `playlist[].id`
- `playlist[].name`
- `playlist[].userId`
- `playlist[].trackCount`
- `playlist[].subscribed`
- `playlist[].specialType`
- `playlist[].privacy`
- `playlist[].coverImgUrl`

CLI 用途：

- `ncm playlist list`
- 区分自建和收藏歌单

### 歌单详情

```text
POST /weapi/v6/playlist/detail?csrf_token=<csrf>
payload: {
  "id": 7292354428,
  "n": 1000,
  "s": 8,
  "csrf_token": "<csrf>"
}
```

关键返回：

- `playlist.tracks[]`
- `privileges[]`
- `playlist.creator.userId`

CLI 用途：

- `ncm playlist show <id>`
- 导出歌单
- 判断歌曲播放权限

### 搜索建议

```text
POST /weapi/search/suggest/web?csrf_token=<csrf>
payload: {
  "s": "周杰伦",
  "limit": 8,
  "csrf_token": "<csrf>"
}
```

CLI 用途：

- `ncm search suggest <keyword>`

### 综合匹配

```text
POST /weapi/search/suggest/multimatch?csrf_token=<csrf>
payload: {
  "s": "周杰伦",
  "csrf_token": "<csrf>"
}
```

CLI 用途：

- 搜索时展示最佳匹配歌手、专辑、歌单等。

### 云搜索

```text
POST /weapi/cloudsearch/get/web?csrf_token=<csrf>
payload: {
  "s": "周杰伦",
  "type": 1,
  "limit": 10,
  "offset": 0,
  "total": true,
  "hlpretag": "<span class=\"s-fc7\">",
  "hlposttag": "</span>",
  "csrf_token": "<csrf>"
}
```

常用 `type`：

- `1`：歌曲
- `1000`：歌单

CLI 用途：

- `ncm search song <keyword>`
- `ncm search playlist <keyword>`

### 歌曲详情

```text
POST /weapi/v3/song/detail?csrf_token=<csrf>
payload: {
  "c": "[{\"id\":210049}]",
  "csrf_token": "<csrf>"
}
```

CLI 用途：

- `ncm song <id>`

### 歌词

```text
POST /weapi/song/lyric?csrf_token=<csrf>
payload: {
  "id": 210049,
  "lv": -1,
  "kv": -1,
  "tv": -1,
  "csrf_token": "<csrf>"
}
```

CLI 用途：

- `ncm lyric <id>`

### 播放地址

```text
POST /weapi/song/enhance/player/url/v1?csrf_token=<csrf>
payload: {
  "ids": "[210049]",
  "level": "exhigh",
  "encodeType": "aac",
  "csrf_token": "<csrf>"
}
```

注意：

- 返回 `code: 200` 不代表一定有可播放 URL。
- `data[].url` 可能为空，受会员、版权、地区和账号状态影响。

CLI 用途：

- `ncm url <song-id> --level exhigh`

### 桌面端播放

网易云 Web 歌曲页的「打开客户端」入口不会跳转到移动端常见的
`orpheus://song/<id>`，而是把 JSON payload 做 Base64 后拼到
`orpheus://` 后面。

Web 端已验证的歌曲播放 payload：

```json
{"type":"song","id":29816860,"cmd":"play"}
```

对应 URL：

```text
orpheus://eyJ0eXBlIjoic29uZyIsImlkIjoyOTgxNjg2MCwiY21kIjoicGxheSJ9
```

2026-04-25 本机验证：通过 macOS `open` 调用该 URL 后，网易云音乐桌面端
3.1.6 播放历史新增 `29816860 / brave heart`。

CLI 用途：

- `ncm play <song-id>`

### 每日推荐歌曲

```text
POST /weapi/v2/discovery/recommend/songs?csrf_token=<csrf>
payload: { "csrf_token": "<csrf>" }
```

CLI 用途：

- `ncm recommend songs`

### 播放记录

```text
POST /weapi/v1/play/record?csrf_token=<csrf>
payload: {
  "uid": "<current-user-id>",
  "type": -1,
  "csrf_token": "<csrf>"
}
```

CLI 用途：

- `ncm record [--week|--all]`

### 创建歌单

```text
POST /weapi/playlist/create?csrf_token=<csrf>
payload: {
  "name": "ncm-cli-api-test-20260425130425",
  "privacy": 10,
  "checkToken": "<checkToken>",
  "csrf_token": "<csrf>"
}
```

说明：

- `privacy: 10` 表示私密歌单。
- `checkToken` 来自页面运行时 `nm.x.lc1x(callback)`。

CLI 用途：

- `ncm playlist create <name> --private`

### 添加歌曲到歌单

```text
POST /weapi/playlist/manipulate/tracks?csrf_token=<csrf>
payload: {
  "op": "add",
  "pid": 17924063236,
  "trackIds": "[210049]",
  "imme": true,
  "csrf_token": "<csrf>"
}
```

说明：

- `trackIds` 是 JSON 字符串，不是数组。
- 删除歌曲时预计使用同接口并把 `op` 改成 `del`，需要单独验证。

CLI 用途：

- `ncm playlist add <playlist-id> <song-id...>`
- `ncm playlist remove <playlist-id> <song-id...>`

### 更新歌单名

```text
POST /weapi/playlist/update/name?csrf_token=<csrf>
payload: {
  "id": 17924063236,
  "name": "new-name",
  "csrf_token": "<csrf>"
}
```

CLI 用途：

- `ncm playlist rename <playlist-id> <name>`

### 更新歌单标签

```text
POST /weapi/playlist/tags/update?csrf_token=<csrf>
payload: {
  "id": 17924063236,
  "tags": "华语",
  "csrf_token": "<csrf>"
}
```

CLI 用途：

- `ncm playlist tags <playlist-id> <tag...>`

### 更新歌单描述

```text
POST /weapi/playlist/desc/update?csrf_token=<csrf>
payload: {
  "id": 17924063236,
  "desc": "description",
  "checkToken": "<checkToken>",
  "csrf_token": "<csrf>"
}
```

CLI 用途：

- `ncm playlist desc <playlist-id> <text>`

### 移除歌单歌曲

```text
POST /weapi/playlist/manipulate/tracks?csrf_token=<csrf>
payload: {
  "op": "del",
  "pid": 17924174661,
  "trackIds": "[210049]",
  "imme": true,
  "csrf_token": "<csrf>"
}
```

CLI 用途：

- `ncm playlist remove <playlist-id> <song-id...>`

### 删除歌单

```text
POST /weapi/playlist/delete?csrf_token=<csrf>
payload: {
  "pid": 17924174661,
  "csrf_token": "<csrf>"
}
```

说明：

- 当前 Web 端验证可用的是 `pid`。
- `ids: "[id]"` 和 `id: <id>` 在本轮验证中返回 `400 请求参数错误`。

CLI 用途：

- `ncm playlist delete <playlist-id>`

## CLI 初步规划

登录是所有命令的前置：

```text
ncm login
ncm me
ncm search song <keyword> [--limit 30] [--offset 0] [--json]
ncm search playlist <keyword> [--limit 30] [--offset 0] [--json]
ncm playlist list [--uid <uid>] [--json]
ncm playlist show <playlist-id> [--json]
ncm playlist create <name> [--private]
ncm playlist add <playlist-id> <song-id...>
ncm playlist remove <playlist-id> <song-id...>
ncm playlist delete <playlist-id>
ncm playlist rename <playlist-id> <name>
ncm playlist tags <playlist-id> <tag...>
ncm playlist desc <playlist-id> <text>
ncm song <song-id> [--json]
ncm play <song-id> [--print-url]
ncm lyric <song-id> [--raw]
ncm url <song-id> [--level standard|exhigh|lossless|hires]
ncm recommend songs
ncm record [--week|--all]
```

实现顺序建议：

1. 先实现 `login`，使用 Playwright 保存 `.ncm/session/storage-state.json` 和必要 cookie。
2. 实现 `me`，每次 CLI 启动先校验登录态。
3. 实现 `weapi` 加密、cookie jar、csrf 自动注入。
4. 先做只读命令：搜索、歌单列表、歌单详情、歌曲详情、歌词。
5. 写操作加交互确认，默认只允许操作当前账号自建歌单。
6. 创建歌单、加歌、改名、改标签、改描述可以作为第二阶段写命令。
