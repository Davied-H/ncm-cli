# Claude Code / Codex 从 GitHub 安装 ncm-cli

这份文档只给 Claude Code、Codex 或其他代码代理使用。目标是从 GitHub 仓库安装本项目的 Skill 和 `ncm` CLI，并完成登录验证。

本项目的 GitHub 仓库是 `Davied-H/ncm-cli`。

## 1. 安装代理 Skill

先安装本项目自带的 `ncm-cli` Skill，让 Claude Code / Codex 知道后续如何使用这个 CLI：

```bash
npx skills add Davied-H/ncm-cli --skill ncm-cli --full-depth -g -y
```

如果使用完整 GitHub URL：

```bash
npx skills add https://github.com/Davied-H/ncm-cli --skill ncm-cli --full-depth -g -y
```

## 2. 安装 ncm CLI 和登录 driver

从同一个 GitHub 仓库拉取并安装 CLI：

```bash
npx --yes github:Davied-H/ncm-cli install --dir ~/.local/bin --with-playwright-driver
```

`--with-playwright-driver` 必须保留。`ncm-cli` 需要打开网易云 Web 完成登录；不安装 Go Playwright driver，`ncm login` 无法正常工作，而主要功能都依赖登录态。

## 3. 确认 ncm 可运行

安装脚本默认写入：

```text
~/.local/bin/ncm
```

验证：

```bash
ncm --help
```

如果 `ncm` 不在 `PATH`，临时使用完整路径：

```bash
~/.local/bin/ncm --help
```

## 4. 登录网易云音乐

执行：

```bash
ncm login
```

等待浏览器打开后，让用户扫码或完成网页登录。登录态会保存到：

```text
~/.config/ncm-cli/
```

不要读取、输出、复制或提交该目录里的 cookie、csrf、`storage-state.json` 等登录态文件。

## 5. 最小验证

登录完成后执行：

```bash
ncm me --json
ncm playlist list --json
```

如果这两个命令返回 `code: 200` 和账号/歌单数据，安装完成。
