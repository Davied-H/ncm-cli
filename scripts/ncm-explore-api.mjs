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
const exploreDir = path.join(rootDir, '.ncm/explore');
const keyword = process.env.NCM_SEARCH_KEYWORD || '周杰伦';
const skipWrites = process.env.NCM_SKIP_WRITES === '1';
const cleanupWrites = process.env.NCM_CLEANUP_WRITES === '1';
const headless = process.env.NCM_HEADLESS === '1';

const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

function compact(value, max = 1400) {
  if (!value) return '';
  const text = typeof value === 'string' ? value : JSON.stringify(value);
  return text.length > max ? `${text.slice(0, max)}...` : text;
}

function summarizeSong(song) {
  if (!song) return null;
  return {
    id: song.id,
    name: song.name,
    artists: (song.ar || song.artists || []).map((artist) => artist.name),
    album: song.al?.name || song.album?.name,
    duration: song.dt || song.duration,
    fee: song.fee,
    privilege: song.privilege && {
      fee: song.privilege.fee,
      pl: song.privilege.pl,
      dl: song.privilege.dl,
      maxbr: song.privilege.maxbr,
    },
  };
}

function summarizePlaylist(playlist) {
  if (!playlist) return null;
  return {
    id: playlist.id,
    name: playlist.name,
    userId: playlist.userId,
    creatorUserId: playlist.creator?.userId,
    trackCount: playlist.trackCount,
    subscribed: playlist.subscribed,
    specialType: playlist.specialType,
    privacy: playlist.privacy,
    coverImgUrl: playlist.coverImgUrl,
  };
}

function responseMeta(call) {
  return {
    ok: call.ok,
    code: call.res?.code ?? call.err?.code,
    message: call.res?.message || call.res?.msg || call.err?.message || call.err?.msg || '',
  };
}

function redactSensitive(value) {
  if (typeof value === 'string') {
    return value.replace(/([?&]csrf_token=)[^&]+/g, '$1<redacted>');
  }
  if (Array.isArray(value)) {
    return value.map((item) => redactSensitive(item));
  }
  if (value && typeof value === 'object') {
    return Object.fromEntries(Object.entries(value).map(([key, item]) => {
      if (key === 'csrf_token' || key === 'checkToken') return [key, '<redacted>'];
      return [key, redactSensitive(item)];
    }));
  }
  return value;
}

await fs.mkdir(sessionDir, { recursive: true });
await fs.mkdir(profileDir, { recursive: true });
await fs.mkdir(exploreDir, { recursive: true });

const context = await chromium.launchPersistentContext(profileDir, {
  channel: 'chrome',
  headless,
  viewport: { width: 1440, height: 1000 },
  locale: 'zh-CN',
  timezoneId: 'Asia/Shanghai',
  ignoreHTTPSErrors: true,
  args: ['--no-proxy-server'],
  userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36',
});

await context.addInitScript(() => {
  window.__ncmPlainQueue = [];
  window.__ncmPlainEvents = [];

  const installAsrseaHook = (fn) => function wrappedAsrsea(text, ...rest) {
    let payload = text;
    try {
      payload = JSON.parse(text);
    } catch {}
    window.__ncmPlainQueue.push(payload);
    return fn.call(this, text, ...rest);
  };

  Object.defineProperty(window, 'asrsea', {
    configurable: true,
    get() {
      return this.__ncmAsrsea;
    },
    set(fn) {
      this.__ncmAsrsea = typeof fn === 'function' ? installAsrseaHook(fn) : fn;
    },
  });

  const open = XMLHttpRequest.prototype.open;
  XMLHttpRequest.prototype.open = function patchedOpen(method, url) {
    this.__ncmMeta = { method, url: String(url) };
    return open.apply(this, arguments);
  };

  const send = XMLHttpRequest.prototype.send;
  XMLHttpRequest.prototype.send = function patchedSend(body) {
    const meta = this.__ncmMeta || {};
    if (/\/(?:weapi|api|eapi)\//.test(meta.url || '')) {
      window.__ncmPlainEvents.push({
        method: meta.method,
        url: meta.url,
        payload: window.__ncmPlainQueue.shift() || null,
        bodyPreview: typeof body === 'string' ? body.slice(0, 700) : '',
      });
    }
    return send.apply(this, arguments);
  };
});

const page = context.pages()[0] || await context.newPage();
page.setDefaultTimeout(30000);

const network = [];
page.on('request', (request) => {
  const url = request.url();
  if (/music\.163\.com\/(?:weapi|api|eapi)\//.test(url)) {
    network.push({
      type: 'request',
      method: request.method(),
      url,
      postData: compact(request.postData(), 900),
    });
  }
});

page.on('response', async (response) => {
  const url = response.url();
  if (!/music\.163\.com\/(?:weapi|api|eapi)\//.test(url)) return;
  const item = { type: 'response', status: response.status(), url, sample: '' };
  const contentType = response.headers()['content-type'] || '';
  if (contentType.includes('json')) {
    try {
      item.sample = compact(await response.text(), 1000);
    } catch {}
  }
  network.push(item);
});

async function callApi(pathname, data = {}, options = {}) {
  return page.evaluate(({ pathname, data, options }) => new Promise((resolve) => {
    const ajax = window.NEJ?.P?.('nej.j')?.bc9T;
    if (!ajax) return resolve({ ok: false, err: { message: 'NEJ ajax unavailable' } });
    ajax(pathname, {
      type: 'json',
      method: options.method || 'post',
      data,
      query: options.query,
      noescape: options.noescape,
      onload: (res) => resolve({ ok: true, res }),
      onerror: (err) => resolve({
        ok: false,
        err: {
          code: err?.code,
          message: err?.message || err?.msg,
          ext: err?.ext,
        },
      }),
    });
  }), { pathname, data, options });
}

async function getCheckToken() {
  return page.evaluate(() => new Promise((resolve) => {
    const x = window.NEJ?.P?.('nm.x');
    if (!x || typeof x.lc1x !== 'function') return resolve('');
    let done = false;
    const timer = setTimeout(() => {
      if (!done) {
        done = true;
        resolve('');
      }
    }, 8000);
    try {
      x.lc1x((token) => {
        if (!done) {
          done = true;
          clearTimeout(timer);
          resolve(token || '');
        }
      });
    } catch {
      clearTimeout(timer);
      resolve('');
    }
  }));
}

function redactPayloadForOutput(payload) {
  return redactSensitive(payload);
}

async function deletePlaylistById(playlistId) {
  const attempts = [
    { ids: JSON.stringify([playlistId]) },
    { id: playlistId },
    { pid: playlistId },
    { ids: String(playlistId) },
    { ids: [playlistId] },
  ];
  const checkToken = await getCheckToken();
  if (checkToken) {
    attempts.push(
      { id: playlistId, checkToken },
      { ids: JSON.stringify([playlistId]), checkToken },
    );
  }

  const results = [];
  for (const payload of attempts) {
    const call = await callApi('/api/playlist/delete', payload);
    const item = {
      ...responseMeta(call),
      payload: redactPayloadForOutput(payload),
    };
    results.push(item);
    if (item.code === 200) {
      return { ...item, attempts: results };
    }
  }
  return { ...results[results.length - 1], attempts: results };
}

await page.goto('https://music.163.com/#/discover', { waitUntil: 'domcontentloaded', timeout: 45000 });
await page.waitForFunction(() => window.NEJ?.P?.('nej.j')?.bc9T && window.GEnc === true, null, { timeout: 30000 });
await wait(2500);

const account = await callApi('/api/w/nuser/account/get');
if (!account.res?.profile?.userId) {
  await context.close();
  throw new Error('当前项目 .ncm/chrome-profile 未登录。请先执行 scripts/ncm-login.mjs。');
}

const uid = account.res.profile.userId;
const nickname = account.res.profile.nickname;
const csrf = await page.evaluate(() => window.NEJ?.P?.('nej.j')?.gM0x?.('__csrf') || '');

const result = {
  capturedAt: new Date().toISOString(),
  keyword,
  skipWrites,
  cleanupWrites,
  account: { userId: uid, nickname, csrfPresent: Boolean(csrf) },
  calls: {},
  endpointCatalog: {
    account: ['POST /weapi/w/nuser/account/get'],
    search: [
      'POST /weapi/search/suggest/web',
      'POST /weapi/search/suggest/multimatch',
      'POST /weapi/cloudsearch/get/web',
    ],
    song: [
      'POST /weapi/v3/song/detail',
      'POST /weapi/song/lyric',
      'POST /weapi/song/enhance/player/url/v1',
    ],
    playlist: [
      'POST /weapi/user/playlist',
      'POST /weapi/v6/playlist/detail',
      'POST /weapi/playlist/create',
      'POST /weapi/playlist/manipulate/tracks',
      'POST /weapi/playlist/update/name',
      'POST /weapi/playlist/tags/update',
      'POST /weapi/playlist/desc/update',
      'POST /weapi/playlist/delete',
      'POST /weapi/playlist/subscribe',
      'POST /weapi/playlist/unsubscribe',
    ],
    recommendations: [
      'POST /weapi/v2/discovery/recommend/songs',
      'POST /weapi/v1/play/record',
    ],
  },
};

result.calls.accountGet = {
  ...responseMeta(account),
  profile: { userId: uid, nickname },
};

const suggest = await callApi('/api/search/suggest/web', { s: keyword, limit: 8 });
result.calls.searchSuggest = {
  ...responseMeta(suggest),
  payload: { s: keyword, limit: 8 },
  songs: (suggest.res?.result?.songs || []).slice(0, 8).map(summarizeSong),
  artists: (suggest.res?.result?.artists || []).slice(0, 5).map((artist) => ({ id: artist.id, name: artist.name })),
  albums: (suggest.res?.result?.albums || []).slice(0, 5).map((album) => ({ id: album.id, name: album.name, artist: album.artist?.name })),
};

const multimatch = await callApi('/api/search/suggest/multimatch', { s: keyword });
result.calls.searchMultiMatch = {
  ...responseMeta(multimatch),
  payload: { s: keyword },
  orders: multimatch.res?.result?.orders || [],
};

const cloudSong = await callApi('/api/cloudsearch/get/web', {
  s: keyword,
  type: 1,
  limit: 10,
  offset: 0,
  total: true,
  hlpretag: '<span class="s-fc7">',
  hlposttag: '</span>',
}, { noescape: true });
result.calls.searchSongs = {
  ...responseMeta(cloudSong),
  payload: { s: keyword, type: 1, limit: 10, offset: 0, total: true },
  songCount: cloudSong.res?.result?.songCount,
  songs: (cloudSong.res?.result?.songs || []).slice(0, 10).map(summarizeSong),
};

const cloudPlaylist = await callApi('/api/cloudsearch/get/web', {
  s: keyword,
  type: 1000,
  limit: 5,
  offset: 0,
  total: true,
}, { noescape: true });
result.calls.searchPlaylists = {
  ...responseMeta(cloudPlaylist),
  payload: { s: keyword, type: 1000, limit: 5, offset: 0, total: true },
  playlistCount: cloudPlaylist.res?.result?.playlistCount,
  playlists: (cloudPlaylist.res?.result?.playlists || []).slice(0, 5).map(summarizePlaylist),
};

const playlistList = await callApi('/api/user/playlist', { uid, limit: 100, offset: 0, includeVideo: true });
const playlists = playlistList.res?.playlist || [];
const hostPlaylists = playlists.filter((playlist) => playlist.userId === uid);
const targetExisting = hostPlaylists.find((playlist) => playlist.specialType !== 5) || hostPlaylists[0] || playlists[0];
result.calls.userPlaylist = {
  ...responseMeta(playlistList),
  payload: { uid, limit: 100, offset: 0, includeVideo: true },
  total: playlists.length,
  created: hostPlaylists.length,
  subscribed: playlists.length - hostPlaylists.length,
  playlists: playlists.slice(0, 12).map(summarizePlaylist),
};

let targetSongId = cloudSong.res?.result?.songs?.[0]?.id || suggest.res?.result?.songs?.[0]?.id || 185809;
const songDetail = await callApi('/api/v3/song/detail', { c: JSON.stringify([{ id: targetSongId }]) });
result.calls.songDetail = {
  ...responseMeta(songDetail),
  payload: { c: JSON.stringify([{ id: targetSongId }]) },
  song: summarizeSong(songDetail.res?.songs?.[0]),
  privilege: songDetail.res?.privileges?.[0] && {
    id: songDetail.res.privileges[0].id,
    fee: songDetail.res.privileges[0].fee,
    pl: songDetail.res.privileges[0].pl,
    dl: songDetail.res.privileges[0].dl,
    maxbr: songDetail.res.privileges[0].maxbr,
  },
};

const lyric = await callApi('/api/song/lyric', { id: targetSongId, lv: -1, kv: -1, tv: -1 });
result.calls.lyric = {
  ...responseMeta(lyric),
  payload: { id: targetSongId, lv: -1, kv: -1, tv: -1 },
  firstLines: (lyric.res?.lrc?.lyric || '').split('\n').slice(0, 8),
};

const playerUrl = await callApi('/api/song/enhance/player/url/v1', {
  ids: JSON.stringify([targetSongId]),
  level: 'exhigh',
  encodeType: 'aac',
});
result.calls.playerUrl = {
  ...responseMeta(playerUrl),
  payload: { ids: JSON.stringify([targetSongId]), level: 'exhigh', encodeType: 'aac' },
  data: playerUrl.res?.data?.[0] && {
    id: playerUrl.res.data[0].id,
    code: playerUrl.res.data[0].code,
    urlPresent: Boolean(playerUrl.res.data[0].url),
    br: playerUrl.res.data[0].br,
    fee: playerUrl.res.data[0].fee,
    level: playerUrl.res.data[0].level,
    cannotListenReason: playerUrl.res.data[0].freeTrialPrivilege?.cannotListenReason,
  },
};

if (targetExisting?.id) {
  const detail = await callApi('/api/v6/playlist/detail', { id: targetExisting.id, n: 1000, s: 8 });
  result.calls.playlistDetail = {
    ...responseMeta(detail),
    payload: { id: targetExisting.id, n: 1000, s: 8 },
    playlist: detail.res?.playlist && {
      ...summarizePlaylist(detail.res.playlist),
      tracks: (detail.res.playlist.tracks || []).slice(0, 10).map(summarizeSong),
    },
    privileges: (detail.res?.privileges || []).slice(0, 10).map((privilege) => ({
      id: privilege.id,
      fee: privilege.fee,
      pl: privilege.pl,
      dl: privilege.dl,
      maxbr: privilege.maxbr,
    })),
  };
}

const daily = await callApi('/api/v2/discovery/recommend/songs', {});
result.calls.dailyRecommendSongs = {
  ...responseMeta(daily),
  songs: (daily.res?.recommend || daily.res?.data?.dailySongs || []).slice(0, 10).map(summarizeSong),
};

const playRecord = await callApi('/api/v1/play/record', { uid, type: -1 });
result.calls.playRecord = {
  ...responseMeta(playRecord),
  payload: { uid, type: -1 },
  weekData: (playRecord.res?.weekData || []).slice(0, 10).map((item) => ({
    playCount: item.playCount,
    score: item.score,
    song: summarizeSong(item.song),
  })),
  allData: (playRecord.res?.allData || []).slice(0, 10).map((item) => ({
    playCount: item.playCount,
    score: item.score,
    song: summarizeSong(item.song),
  })),
};

if (!skipWrites) {
  const stamp = new Date().toISOString().replace(/[-:TZ.]/g, '').slice(0, 14);
  const testName = `ncm-cli-api-test-${stamp}`;
  const createToken = await getCheckToken();
  const createPlaylist = await callApi('/api/playlist/create', {
    name: testName,
    privacy: 10,
    checkToken: createToken,
  });
  const createdPlaylist = createPlaylist.res?.playlist;

  result.calls.createPlaylist = {
    ...responseMeta(createPlaylist),
    payload: { name: testName, privacy: 10, checkToken: '<from nm.x.lc1x>' },
    playlist: summarizePlaylist(createdPlaylist),
  };

  if (createdPlaylist?.id) {
    const addTrack = await callApi('/api/playlist/manipulate/tracks', {
      op: 'add',
      pid: createdPlaylist.id,
      trackIds: JSON.stringify([targetSongId]),
      imme: true,
    });
    result.calls.addTrackToPlaylist = {
      ...responseMeta(addTrack),
      payload: { op: 'add', pid: createdPlaylist.id, trackIds: JSON.stringify([targetSongId]), imme: true },
      response: addTrack.res && {
        code: addTrack.res.code,
        message: addTrack.res.message || addTrack.res.msg,
        coverImgUrlPresent: Boolean(addTrack.res.coverImgUrl),
      },
    };

    const renamed = `${testName}-renamed`;
    const updateName = await callApi('/api/playlist/update/name', { id: createdPlaylist.id, name: renamed });
    result.calls.updatePlaylistName = {
      ...responseMeta(updateName),
      payload: { id: createdPlaylist.id, name: renamed },
    };

    const updateTags = await callApi('/api/playlist/tags/update', { id: createdPlaylist.id, tags: '华语' });
    result.calls.updatePlaylistTags = {
      ...responseMeta(updateTags),
      payload: { id: createdPlaylist.id, tags: '华语' },
    };

    const descToken = await getCheckToken();
    const updateDesc = await callApi('/api/playlist/desc/update', {
      id: createdPlaylist.id,
      desc: 'ncm-cli API exploration test playlist',
      checkToken: descToken,
    });
    result.calls.updatePlaylistDesc = {
      ...responseMeta(updateDesc),
      payload: { id: createdPlaylist.id, desc: 'ncm-cli API exploration test playlist', checkToken: '<from nm.x.lc1x>' },
    };

    const testDetail = await callApi('/api/v6/playlist/detail', { id: createdPlaylist.id, n: 1000, s: 8 });
    result.calls.createdPlaylistDetail = {
      ...responseMeta(testDetail),
      payload: { id: createdPlaylist.id, n: 1000, s: 8 },
      playlist: testDetail.res?.playlist && {
        ...summarizePlaylist(testDetail.res.playlist),
        tracks: (testDetail.res.playlist.tracks || []).slice(0, 10).map(summarizeSong),
      },
    };

    if (cleanupWrites) {
      const removeTrack = await callApi('/api/playlist/manipulate/tracks', {
        op: 'del',
        pid: createdPlaylist.id,
        trackIds: JSON.stringify([targetSongId]),
        imme: true,
      });
      result.calls.removeTrackFromPlaylist = {
        ...responseMeta(removeTrack),
        payload: { op: 'del', pid: createdPlaylist.id, trackIds: JSON.stringify([targetSongId]), imme: true },
        response: removeTrack.res && {
          code: removeTrack.res.code,
          message: removeTrack.res.message || removeTrack.res.msg,
        },
      };

      result.calls.deletePlaylist = await deletePlaylistById(createdPlaylist.id);
    }
  }
}

const plainEvents = await page.evaluate(() => window.__ncmPlainEvents || []);
result.plaintextEvents = plainEvents.map((event) => ({
  method: event.method,
  actualUrl: redactSensitive(event.url),
  payload: redactSensitive(event.payload),
  encryptedBodyShape: event.bodyPreview.includes('params=') && event.bodyPreview.includes('encSecKey=')
    ? 'params + encSecKey'
    : event.bodyPreview,
}));
result.network = network.map((entry) => ({
  ...entry,
  url: redactSensitive(entry.url),
  sample: redactSensitive(entry.sample),
  postData: entry.postData ? '<encrypted form body redacted>' : entry.postData,
}));

const outFile = path.join(exploreDir, `ncm-api-explore-${new Date().toISOString().replace(/[:.]/g, '-')}.json`);
await fs.writeFile(outFile, `${JSON.stringify(result, null, 2)}\n`, { mode: 0o600 });

const userInfo = {
  exportedAt: new Date().toISOString(),
  userId: uid,
  nickname,
  csrfPresent: Boolean(csrf),
  storageStatePath: path.join(sessionDir, 'storage-state.json'),
  sourceProfileDir: profileDir,
  lastExplorePath: outFile,
};
await fs.writeFile(path.join(sessionDir, 'user.json'), `${JSON.stringify(userInfo, null, 2)}\n`, { mode: 0o600 });
await context.storageState({ path: path.join(sessionDir, 'storage-state.json') });
await fs.chmod(path.join(sessionDir, 'storage-state.json'), 0o600);
await fs.chmod(path.join(sessionDir, 'user.json'), 0o600);
await fs.chmod(outFile, 0o600);

console.log(JSON.stringify({
  outFile,
  account: result.account,
  searchSongs: {
    code: result.calls.searchSongs.code,
    count: result.calls.searchSongs.songCount,
    first: result.calls.searchSongs.songs?.[0],
  },
  createdPlaylist: result.calls.createPlaylist?.playlist || null,
  addTrack: result.calls.addTrackToPlaylist || null,
  updateName: result.calls.updatePlaylistName || null,
  updateTags: result.calls.updatePlaylistTags || null,
  updateDesc: result.calls.updatePlaylistDesc || null,
  removeTrack: result.calls.removeTrackFromPlaylist || null,
  deletePlaylist: result.calls.deletePlaylist || null,
}, null, 2));

await context.close();
