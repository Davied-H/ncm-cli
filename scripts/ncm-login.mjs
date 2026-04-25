import { createRequire } from 'node:module';
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

for (const key of ['HTTP_PROXY', 'HTTPS_PROXY', 'ALL_PROXY', 'http_proxy', 'https_proxy', 'all_proxy']) {
  delete process.env[key];
}

function loadPlaywright() {
  const require = createRequire(import.meta.url);
  const candidates = [
    'playwright',
    '/tmp/ncm-api-explore/node_modules/playwright',
    '/Users/dong/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright',
  ];

  for (const candidate of candidates) {
    try {
      return require(candidate);
    } catch {}
  }
  throw new Error('未找到 playwright。可在项目根目录执行 npm install playwright 后重试。');
}

const { chromium } = loadPlaywright();
const rootDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const profileDir = path.join(rootDir, '.ncm/chrome-profile');
const sessionDir = path.join(rootDir, '.ncm/session');
const storagePath = path.join(sessionDir, 'storage-state.json');
const userPath = path.join(sessionDir, 'user.json');

const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

async function readExistingUserInfo() {
  try {
    return JSON.parse(await fs.readFile(userPath, 'utf8'));
  } catch {
    return {};
  }
}

async function accountGet(page) {
  return page.evaluate(() => new Promise((resolve) => {
    const ajax = window.NEJ?.P?.('nej.j')?.bc9T;
    if (!ajax) return resolve({ code: -1, message: 'NEJ ajax unavailable' });
    ajax('/api/w/nuser/account/get', {
      type: 'json',
      method: 'post',
      onload: (res) => resolve(res),
      onerror: (err) => resolve({ code: err?.code || -1, message: err?.message || err?.msg || 'error' }),
    });
  })).catch((error) => ({ code: -1, message: error.message }));
}

async function waitForLogin(page) {
  const deadline = Date.now() + 10 * 60 * 1000;
  let latest = null;
  while (Date.now() < deadline) {
    latest = await accountGet(page);
    if (latest?.profile?.userId) return latest;
    process.stdout.write('.');
    await wait(3000);
  }
  throw new Error(`等待登录超时，最后账号状态：${JSON.stringify(latest)}`);
}

await fs.mkdir(sessionDir, { recursive: true });
await fs.mkdir(profileDir, { recursive: true });

const context = await chromium.launchPersistentContext(profileDir, {
  channel: 'chrome',
  headless: false,
  viewport: { width: 1280, height: 900 },
  locale: 'zh-CN',
  timezoneId: 'Asia/Shanghai',
  ignoreHTTPSErrors: true,
  args: ['--no-proxy-server'],
});

const page = context.pages()[0] || await context.newPage();
await page.goto('https://music.163.com/#/discover', { waitUntil: 'domcontentloaded', timeout: 45000 });
await page.waitForFunction(() => window.NEJ?.P?.('nej.j')?.bc9T && window.GEnc === true, null, { timeout: 30000 });

let account = await accountGet(page);
if (!account?.profile?.userId) {
  console.log('当前未登录，已打开登录窗口。请扫码或完成登录。');
  try {
    await page.evaluate(() => {
      if (typeof window.login === 'function') window.login();
      else if (window.top && typeof window.top.login === 'function') window.top.login();
    });
  } catch {}
  account = await waitForLogin(page);
} else {
  console.log(`已复用登录态：${account.profile.nickname} (${account.profile.userId})`);
}

await context.storageState({ path: storagePath });
const cookies = await context.cookies('https://music.163.com');
const csrf = cookies.find((cookie) => cookie.name === '__csrf')?.value || '';
const existingUserInfo = await readExistingUserInfo();

const userInfo = {
  exportedAt: new Date().toISOString(),
  userId: account.profile.userId,
  nickname: account.profile.nickname,
  csrfPresent: Boolean(csrf),
  storageStatePath: storagePath,
  sourceProfileDir: profileDir,
  ...(existingUserInfo.lastExplorePath ? { lastExplorePath: existingUserInfo.lastExplorePath } : {}),
};

await fs.writeFile(userPath, `${JSON.stringify(userInfo, null, 2)}\n`, { mode: 0o600 });
await fs.chmod(storagePath, 0o600);
await fs.chmod(userPath, 0o600);

console.log(JSON.stringify(userInfo, null, 2));
await context.close();
