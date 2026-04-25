#!/usr/bin/env node

import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import https from 'node:https';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptPath = fileURLToPath(import.meta.url);
const rootDir = path.resolve(path.dirname(scriptPath), '..');
const githubRepo = 'Davied-H/ncm-cli';
const githubPackageURL = `https://raw.githubusercontent.com/${githubRepo}/main/package.json`;
const githubCommitURL = `https://api.github.com/repos/${githubRepo}/commits/main`;

function usage() {
  console.log(`ncm-cli installer

Usage:
  npx github:<owner>/<repo> [install] [--dir <bin-dir>] [--name <binary-name>]
  npx github:<owner>/<repo> check-update [--dir <bin-dir>] [--name <binary-name>] [--json]
  npx github:<owner>/<repo> update [--dir <bin-dir>] [--name <binary-name>] [--with-playwright-driver]
  npx ncm-cli@latest [install|check-update|update] [--dir <bin-dir>] [--name <binary-name>]
  node scripts/ncm-install.mjs [install|check-update|update] [--dir <bin-dir>] [--name <binary-name>]

Options:
  --dir <bin-dir>          Install directory. Defaults to NCM_CLI_INSTALL_DIR or ~/.local/bin.
  --name <binary-name>     Installed binary name. Defaults to ncm.
  --with-playwright-driver Install required Go Playwright driver after building.
  --with-playwright-browser
                          Also install Playwright Chromium. Only needed when Chrome is unavailable.
  --json                   Output JSON for check-update.
  --force                  Reinstall even when update reports the latest GitHub version.
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
    withPlaywrightBrowser: false,
    json: false,
    force: false,
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
    } else if (arg === '--with-playwright-browser') {
      opts.withPlaywrightBrowser = true;
    } else if (arg === '--json') {
      opts.json = true;
    } else if (arg === '--force') {
      opts.force = true;
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

function readPackageMetadata() {
  const packagePath = path.join(rootDir, 'package.json');
  return JSON.parse(fs.readFileSync(packagePath, 'utf8'));
}

function localGitCommit() {
  const result = spawnSync('git', ['rev-parse', '--short=12', 'HEAD'], {
    cwd: rootDir,
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'ignore'],
  });
  if (result.error || result.status !== 0) {
    return '';
  }
  const commit = result.stdout.trim();
  if (!commit) return '';

  const dirty = spawnSync('git', ['diff', '--quiet'], {
    cwd: rootDir,
    stdio: 'ignore',
  });
  return dirty.status === 0 ? commit : `${commit}-dirty`;
}

function fetchJSON(url) {
  return new Promise((resolve, reject) => {
    const request = https.get(url, {
      headers: {
        Accept: 'application/json',
        'User-Agent': 'ncm-cli-installer',
      },
    }, (response) => {
      const status = response.statusCode || 0;
      if ([301, 302, 303, 307, 308].includes(status) && response.headers.location) {
        response.resume();
        fetchJSON(new URL(response.headers.location, url).toString()).then(resolve, reject);
        return;
      }
      let body = '';
      response.setEncoding('utf8');
      response.on('data', (chunk) => {
        body += chunk;
      });
      response.on('end', () => {
        if (status < 200 || status >= 300) {
          reject(new Error(`GitHub request failed with HTTP ${status}: ${url}`));
          return;
        }
        try {
          resolve(JSON.parse(body));
        } catch (error) {
          reject(new Error(`GitHub response was not valid JSON: ${error.message}`));
        }
      });
    });
    request.setTimeout(15000, () => {
      request.destroy(new Error(`GitHub request timed out: ${url}`));
    });
    request.on('error', reject);
  });
}

let latestGitHubMetadataCache = null;

async function latestGitHubMetadata() {
  if (latestGitHubMetadataCache) return latestGitHubMetadataCache;

  const [pkg, commit] = await Promise.all([
    fetchJSON(githubPackageURL),
    fetchJSON(githubCommitURL),
  ]);

  latestGitHubMetadataCache = {
    version: String(pkg.version || ''),
    commit: String(commit.sha || ''),
    packageURL: githubPackageURL,
    commitURL: githubCommitURL,
  };
  return latestGitHubMetadataCache;
}

async function buildMetadata() {
  const pkg = readPackageMetadata();
  let buildCommit = localGitCommit();

  if (!buildCommit) {
    try {
      const latest = await latestGitHubMetadata();
      if (latest.version === pkg.version && latest.commit) {
        buildCommit = latest.commit.slice(0, 12);
      }
    } catch {
      buildCommit = 'unknown';
    }
  }

  return {
    version: String(pkg.version || 'dev'),
    commit: buildCommit || 'unknown',
  };
}

async function install(opts) {
  ensureGo();

  const installDir = path.resolve(opts.dir);
  const outPath = path.join(installDir, opts.name);
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'ncm-cli-install-'));
  const builtPath = path.join(tmpDir, opts.name);
  const metadata = await buildMetadata();
  const ldflags = `-X main.version=${metadata.version} -X main.commit=${metadata.commit}`;

  fs.mkdirSync(installDir, { recursive: true });
  try {
    console.log(`Building ncm CLI ${metadata.version} (${metadata.commit}) from ${rootDir}`);
    run('go', ['build', '-ldflags', ldflags, '-o', builtPath, './cmd/ncm']);

    fs.copyFileSync(builtPath, outPath);
    fs.chmodSync(outPath, 0o755);
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }

  console.log(`Installed ${opts.name} to ${outPath}`);
  console.log(`Run: ${opts.name} login`);

  if (opts.withPlaywrightBrowser) {
    console.log('Installing Go Playwright driver and Chromium browser');
    run('go', ['run', 'github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1', 'install', 'chromium']);
  } else if (opts.withPlaywrightDriver) {
    console.log('Installing Go Playwright driver');
    run('go', ['run', 'github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1', '--version']);
  } else {
    console.log('Playwright driver is required for ncm login. Install it before logging in:');
    console.log('  go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 --version');
    console.log('If Chrome is unavailable, install Playwright Chromium too:');
    console.log('  go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 install chromium');
  }
}

function shellQuote(value) {
  return `'${String(value).replaceAll("'", "'\\''")}'`;
}

function findOnPath(name) {
  const result = spawnSync('sh', ['-lc', `command -v ${shellQuote(name)}`], {
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'ignore'],
  });
  if (result.error || result.status !== 0) return '';
  return result.stdout.trim().split('\n')[0] || '';
}

function installedBinaryPath(opts) {
  const configuredPath = path.join(path.resolve(opts.dir), opts.name);
  if (fs.existsSync(configuredPath)) return configuredPath;

  const pathBinary = findOnPath(opts.name);
  if (pathBinary && fs.existsSync(pathBinary)) return pathBinary;

  return configuredPath;
}

function parseVersionText(text) {
  const match = String(text).match(/\bv?(\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?)\b/);
  return match ? match[1] : '';
}

function readInstalledVersion(opts) {
  const binaryPath = installedBinaryPath(opts);
  if (!fs.existsSync(binaryPath)) {
    return {
      installed: false,
      path: binaryPath,
      version: '',
      commit: '',
      error: '',
    };
  }

  const versionResult = spawnSync(binaryPath, ['version', '--json'], {
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  if (!versionResult.error && versionResult.status === 0) {
    try {
      const data = JSON.parse(versionResult.stdout);
      return {
        installed: true,
        path: binaryPath,
        version: String(data.version || ''),
        commit: String(data.commit || ''),
        error: '',
      };
    } catch (error) {
      return {
        installed: true,
        path: binaryPath,
        version: '',
        commit: '',
        error: `无法解析版本 JSON：${error.message}`,
      };
    }
  }

  const fallbackResult = spawnSync(binaryPath, ['--version'], {
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  if (!fallbackResult.error && fallbackResult.status === 0) {
    return {
      installed: true,
      path: binaryPath,
      version: parseVersionText(fallbackResult.stdout),
      commit: '',
      error: '',
    };
  }

  const stderr = versionResult.stderr?.trim() || fallbackResult.stderr?.trim() || versionResult.error?.message || fallbackResult.error?.message || '';
  return {
    installed: true,
    path: binaryPath,
    version: '',
    commit: '',
    error: stderr,
  };
}

function versionParts(value) {
  const text = String(value || '').trim().replace(/^v/, '').split(/[+-]/)[0];
  if (!text) return null;
  const parts = text.split('.').map((part) => Number.parseInt(part, 10));
  if (parts.some((part) => Number.isNaN(part))) return null;
  return parts;
}

function compareVersions(left, right) {
  const a = versionParts(left);
  const b = versionParts(right);
  if (!a || !b) return null;
  const length = Math.max(a.length, b.length);
  for (let i = 0; i < length; i += 1) {
    const av = a[i] || 0;
    const bv = b[i] || 0;
    if (av > bv) return 1;
    if (av < bv) return -1;
  }
  return 0;
}

function commitMatches(current, latest) {
  if (!current || !latest || current === 'unknown') return false;
  const cleanCurrent = String(current).replace(/-dirty$/, '');
  return latest.startsWith(cleanCurrent) || cleanCurrent.startsWith(latest);
}

async function checkUpdateReport(opts) {
  const latest = await latestGitHubMetadata();
  const current = readInstalledVersion(opts);
  const versionCompare = compareVersions(current.version, latest.version);

  let updateAvailable = false;
  let reason = 'installed version matches latest GitHub version';

  if (!current.installed) {
    updateAvailable = true;
    reason = 'ncm is not installed';
  } else if (!current.version || versionCompare === null) {
    updateAvailable = true;
    reason = 'installed ncm does not report a comparable version';
  } else if (versionCompare < 0) {
    updateAvailable = true;
    reason = 'installed version is older than latest GitHub version';
  } else if (versionCompare === 0 && !commitMatches(current.commit, latest.commit)) {
    updateAvailable = true;
    reason = 'installed commit differs from latest GitHub main';
  } else if (versionCompare > 0) {
    reason = 'installed version is newer than latest GitHub version';
  }

  return {
    installed: current.installed,
    installedPath: current.path,
    currentVersion: current.version || null,
    currentCommit: current.commit || null,
    latestVersion: latest.version || null,
    latestCommit: latest.commit || null,
    latestSource: latest.packageURL,
    updateAvailable,
    reason,
    error: current.error || null,
  };
}

function shortCommit(commitValue) {
  return commitValue ? commitValue.slice(0, 12) : 'unknown';
}

function printCheckUpdateReport(report, opts) {
  if (!report.installed) {
    console.log(`ncm is not installed at ${report.installedPath}`);
  } else {
    console.log(`Installed ncm: ${report.currentVersion || 'unknown'} (${report.currentCommit || 'unknown'})`);
    console.log(`Installed path: ${report.installedPath}`);
  }

  console.log(`Latest GitHub: ${report.latestVersion || 'unknown'} (${shortCommit(report.latestCommit)})`);
  console.log(`Update available: ${report.updateAvailable ? 'yes' : 'no'}`);
  console.log(`Reason: ${report.reason}`);
  if (report.error) {
    console.log(`Version check warning: ${report.error}`);
  }
  if (report.updateAvailable) {
    console.log('Update command:');
    console.log(`  npx --yes github:${githubRepo} update --dir ${shellQuote(opts.dir)} --name ${shellQuote(opts.name)} --with-playwright-driver`);
  }
}

async function checkUpdate(opts) {
  const report = await checkUpdateReport(opts);
  if (opts.json) {
    console.log(JSON.stringify(report, null, 2));
  } else {
    printCheckUpdateReport(report, opts);
  }
}

async function update(opts) {
  const before = await checkUpdateReport(opts);
  if (!before.updateAvailable && !opts.force) {
    printCheckUpdateReport(before, opts);
    console.log('Already up to date. Use --force to reinstall anyway.');
    return;
  }

  printCheckUpdateReport(before, opts);
  await install(opts);

  const after = await checkUpdateReport(opts);
  console.log(`After update: ${after.currentVersion || 'unknown'} (${after.currentCommit || 'unknown'})`);
}

try {
  const opts = parseArgs(process.argv.slice(2));
  if (opts.command === 'help') {
    usage();
  } else if (opts.command === 'install') {
    await install(opts);
  } else if (opts.command === 'check' || opts.command === 'check-update') {
    await checkUpdate(opts);
  } else if (opts.command === 'update' || opts.command === 'upgrade') {
    await update(opts);
  } else {
    throw new Error(`未知命令：${opts.command}`);
  }
} catch (error) {
  console.error(error.message);
  console.error('Run with --help for usage.');
  process.exit(1);
}
