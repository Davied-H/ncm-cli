# Claude Code / Codex 安装 ncm-cli 的步骤

这份文档只给 Claude Code、Codex 或其他代码代理使用。目标是在用户机器上安装 `ncm-cli`，安装登录所需的 Go Playwright driver，并完成最小可用验证。

## 1. 确认前置依赖

先检查 Go 和 npx 是否可用：

```bash
go version
npx --version
```

如果 `go` 不存在，先让用户安装 Go 1.24+。如果 `npx` 不存在，先让用户安装 Node.js/npm。

## 2. 安装 ncm CLI

如果当前目录就是 `ncm-cli` 仓库，执行本地安装：

```bash
npx . install --dir ~/.local/bin --with-playwright-driver
```

如果 `ncm-cli` 已发布到 npm，执行远程安装：

```bash
npx ncm-cli@latest install --dir ~/.local/bin --with-playwright-driver
```

`--with-playwright-driver` 是必须的。`ncm-cli` 需要打开网易云 Web 完成登录；不安装 Go Playwright driver，`ncm login` 无法正常工作，而多数功能都依赖登录态。

## 3. 确认 PATH

安装脚本默认写入：

```text
~/.local/bin/ncm
```

检查 `ncm` 是否可直接运行：

```bash
command -v ncm
ncm --help
```

如果找不到 `ncm`，把 `~/.local/bin` 加入用户的 `PATH`，或临时使用完整路径：

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
