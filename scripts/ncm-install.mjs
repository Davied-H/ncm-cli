#!/usr/bin/env node

import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptPath = fileURLToPath(import.meta.url);
const rootDir = path.resolve(path.dirname(scriptPath), '..');

function usage() {
  console.log(`ncm-cli installer

Usage:
  npx github:<owner>/<repo> [install] [--dir <bin-dir>] [--name <binary-name>]
  npx ncm-cli@latest [install] [--dir <bin-dir>] [--name <binary-name>]
  node scripts/ncm-install.mjs [install] [--dir <bin-dir>] [--name <binary-name>]

Options:
  --dir <bin-dir>          Install directory. Defaults to NCM_CLI_INSTALL_DIR or ~/.local/bin.
  --name <binary-name>     Installed binary name. Defaults to ncm.
  --with-playwright-driver Install required Go Playwright Chromium driver after building.
  --help                   Show this help.
`);
}

function parseArgs(argv) {
  const args = [...argv];
  const opts = {
    command: 'install',
    dir: process.env.NCM_CLI_INSTALL_DIR || path.join(os.homedir(), '.local', 'bin'),
    name: 'ncm',
    withPlaywrightDriver: false,
  };

  if (args[0] && !args[0].startsWith('-')) {
    opts.command = args.shift();
  }

  for (let i = 0; i < args.length; i += 1) {
    const arg = args[i];
    if (arg === '--help' || arg === '-h') {
      opts.command = 'help';
    } else if (arg === '--dir' || arg === '--bin-dir') {
      opts.dir = args[++i];
    } else if (arg === '--name') {
      opts.name = args[++i];
    } else if (arg === '--with-playwright-driver') {
      opts.withPlaywrightDriver = true;
    } else {
      throw new Error(`未知参数：${arg}`);
    }
  }

  if (!opts.dir) throw new Error('--dir 不能为空');
  if (!opts.name) throw new Error('--name 不能为空');
  return opts;
}

function run(cmd, args, options = {}) {
  const result = spawnSync(cmd, args, {
    cwd: options.cwd || rootDir,
    stdio: 'inherit',
    env: { ...process.env, ...(options.env || {}) },
  });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(`${cmd} ${args.join(' ')} failed with exit code ${result.status}`);
  }
}

function ensureGo() {
  const result = spawnSync('go', ['version'], { stdio: 'ignore' });
  if (result.error || result.status !== 0) {
    throw new Error('未找到 go。请先安装 Go 1.24+，然后重新运行安装命令。');
  }
}

function install(opts) {
  ensureGo();

  const installDir = path.resolve(opts.dir);
  const outPath = path.join(installDir, opts.name);
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'ncm-cli-install-'));
  const builtPath = path.join(tmpDir, opts.name);

  fs.mkdirSync(installDir, { recursive: true });
  console.log(`Building ncm CLI from ${rootDir}`);
  run('go', ['build', '-o', builtPath, './cmd/ncm']);

  fs.copyFileSync(builtPath, outPath);
  fs.chmodSync(outPath, 0o755);
  fs.rmSync(tmpDir, { recursive: true, force: true });

  console.log(`Installed ${opts.name} to ${outPath}`);
  console.log(`Run: ${opts.name} login`);

  if (opts.withPlaywrightDriver) {
    console.log('Installing Go Playwright Chromium driver');
    run('go', ['run', 'github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1', 'install', 'chromium']);
  } else {
    console.log('Playwright driver is required for ncm login. Install it before logging in:');
    console.log('  go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 install chromium');
  }
}

try {
  const opts = parseArgs(process.argv.slice(2));
  if (opts.command === 'help') {
    usage();
  } else if (opts.command === 'install') {
    install(opts);
  } else {
    throw new Error(`未知命令：${opts.command}`);
  }
} catch (error) {
  console.error(error.message);
  console.error('Run with --help for usage.');
  process.exit(1);
}
