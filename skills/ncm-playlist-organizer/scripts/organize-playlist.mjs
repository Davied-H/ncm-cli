#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { execFileSync } from "node:child_process";

const args = parseArgs(process.argv.slice(2));

if (!args["playlist-id"] && !args.input) {
  usage();
  process.exit(1);
}

const outDir = path.resolve(args["out-dir"] || process.cwd());
const prefix = sanitizePrefix(args.prefix || "ncm-playlist");
fs.mkdirSync(outDir, { recursive: true });

const cn = /[\u4e00-\u9fff]/;
const kana = /[\u3040-\u30ff]/;
const hangul = /[\uac00-\ud7af]/;

const jpArtists = new Set([
  "majiko",
  "玉置浩二",
  "SEKAI NO OWARI",
  "Reol",
  "德永英明",
  "Aimer",
  "あいみょん",
  "北野武",
  "ハンバート ハンバート",
  "中島美嘉",
  "KOKIA",
  "平井 大",
  "手嶌葵",
  "Ms.OOJA",
  "米津玄師",
  "高梨康治",
  "Akie秋绘",
  "和田光司",
  "宮崎歩",
  "つじあやの",
  "Goose house",
  "seven oops",
  "いきものがかり",
  "浜崎あゆみ",
  "RADWIMPS",
  "高橋優",
  "當山みれい",
  "EGOIST",
  "川嶋あい",
  "まふまふ",
  "久石譲",
  "小田和正",
  "中森明菜",
  "放課後ティータイム",
  "茶太",
  "平井堅",
  "神山羊",
  "wacci",
  "Naomile",
  "MONKEY MAJIK",
  "前田愛",
  "星野源",
  "柴田淳",
  "岡村孝子",
  "安全地帯",
  "杏里",
  "西城秀樹",
  "希良梨",
  "Eve",
  "Daoko",
  "eill",
  "GARNiDELiA",
  "GOING UNDER GROUND",
  "jyA-Me",
  "King Gnu",
  "m-flo",
  "m.o.v.e",
  "MAGIC OF LiFE",
  "Polkadot Stingray",
  "back number",
  "SawanoHiroyuki[nZk]",
  "sumika",
  "yama",
  "一十三十一",
  "中孝介",
  "v flower",
  "ONE☆DRAFT",
  "bassy",
  "miu-clips",
  "電波少女",
]);

const krArtists = new Set(["IU", "Sik-K"]);
const westernArtists = new Set([
  "Ludwig van Beethoven",
  "Carpenters",
  "Rihanna",
  "Adele",
  "Bruno Mars",
  "Michael Jackson",
  "Charlie Puth",
  "Jason Mraz",
  "Queen",
  "Alan Walker",
  "Taylor Swift",
  "Eminem",
  "Justin Bieber",
  "Johnny Cash",
  "Ed Sheeran",
  "P!nk",
  "Fleurie",
  "Matthias Reim",
  "Maroon 5",
  "Daddy Yankee",
]);
const instrumentalArtists = new Set(["Ludwig van Beethoven", "久石譲", "高梨康治"]);

const primaryBucketDefs = [
  {
    key: "japanese_anime",
    name: "日系 / ACG",
    test: (t) =>
      kana.test(t.allText) ||
      /动漫|动画|anime|ost|op|ed|k-?on|けいおん|ヨルシカ|n-buna|aimyon|あいみょん|majiko|doraemon|哆啦|gto/i.test(t.allText),
  },
  {
    key: "chinese_pop",
    name: "华语流行",
    test: (t) => cn.test(t.allText) && !kana.test(t.allText) && !hangul.test(t.allText),
  },
  {
    key: "english_pop",
    name: "欧美 / 英文",
    test: (t) => /^[\x00-\x7f\s·'’&.,:!?()+/\-]+$/.test(t.allText) && /[a-z]/i.test(t.allText),
  },
  {
    key: "korean",
    name: "韩语 / K-Pop",
    test: (t) => hangul.test(t.allText) || /k-?pop|korea|korean/i.test(t.allText),
  },
  {
    key: "instrumental_focus",
    name: "纯音 / 阅读 / 专注",
    test: (t) => /纯音|钢琴|piano|instrumental|bgm|原声|阅读|看书|学习|咖啡|jazz|classical|古典|soundtrack/i.test(t.allText),
  },
  {
    key: "workout",
    name: "运动 / 节奏",
    test: (t) => /健身|跑步|运动|hiit|节奏|踩点|燃|前方高能|workout|gym|dance|edm/i.test(t.allText),
  },
];

const tagDefs = [
  {
    key: "live_versions",
    name: "Live / 翻唱 / 版本",
    test: (t) => /live|现场|翻唱|cover|acoustic|rehearsal|version|remix|remaster|伴奏|demo/i.test(t.allText),
  },
  primaryBucketDefs.find((bucket) => bucket.key === "instrumental_focus"),
  primaryBucketDefs.find((bucket) => bucket.key === "workout"),
  {
    key: "long_tracks",
    name: "超长曲目候选",
    test: (t) => t.durationMs >= 7 * 60 * 1000,
  },
  {
    key: "short_tracks",
    name: "短曲目候选",
    test: (t) => t.durationMs > 0 && t.durationMs <= 90 * 1000,
  },
].filter(Boolean);

const canonicalMarkers =
  /\s*(?:\(|（|\[|【).{0,36}?(?:live|现场|翻唱|cover|acoustic|rehearsal|version|remix|remaster|伴奏|demo|电影|电视剧|动画|动漫|主题曲|片头曲|片尾曲|插曲|ost).{0,36}?(?:\)|）|\]|】)\s*/gi;

const payload = args.input ? readJson(args.input) : await fetchPlaylist(args["playlist-id"]);
const playlist = payload.playlist || payload.result;
if (!playlist) throw new Error("Input JSON does not contain a playlist/result object.");

const tracks = playlist.tracks || [];
const privileges = new Map((payload.privileges || []).map((p) => [p.id, p]));
const expectedTrackCount = playlist.trackCount || tracks.length;

const enriched = tracks.map((track, index) => enrichTrack(track, index, privileges));
const duplicateGroups = [...groupBy(enriched, (t) => t.canonical).values()].filter((group) => group.length > 1);
const titleCollisionGroups = [...groupBy(enriched, (t) => t.titleCanonical).values()]
  .filter((group) => group.length > 1 && new Set(group.map((t) => t.artists.join(" / "))).size > 1)
  .sort((a, b) => b.length - a.length);

const primaryBucketCounts = countBy(enriched, (t) => t.primaryBucketKey);
const tagCounts = countBy(enriched, (t) => t.tagKeys);
const artistCounts = countBy(enriched.flatMap((t) => t.artists), (x) => x).slice(0, 40);
const albumCounts = countBy(enriched, (t) => t.album).filter(([name]) => name).slice(0, 30);
const feeCounts = countBy(enriched, (t) => `fee=${t.fee ?? "unknown"}`);
const knownPrivilegeCount = enriched.filter((t) => t.playableLevel !== null).length;
const unplayable = enriched.filter((t) => t.playableLevel === 0);

const primaryBuckets = Object.fromEntries(
  primaryBucketCounts.map(([key]) => [
    primaryName(key),
    enriched.filter((t) => t.primaryBucketKey === key).map(exportTrack),
  ]),
);

const tags = Object.fromEntries(
  tagCounts.map(([key]) => [
    tagName(key),
    enriched.filter((t) => t.tagKeys.includes(key)).map(exportTrack),
  ]),
);
tags["疑似重复版本"] = duplicateGroups.flat().map(exportTrack);
tags["同名不同歌手"] = titleCollisionGroups.flat().map(exportTrack);

writeJson(`${prefix}-primary-buckets.json`, primaryBuckets);
writeJson(`${prefix}-tags.json`, tags);
writeCsv(`${prefix}-tracks.csv`, enriched);
writeMarkdown(`${prefix}-analysis.md`, buildAnalysis());
writeMarkdown(`${prefix}-copy-plan.md`, buildCopyPlan());

console.log(
  JSON.stringify(
    {
      playlist: playlist.name,
      id: playlist.id,
      expectedTrackCount,
      tracks: enriched.length,
      complete: enriched.length === expectedTrackCount,
      analysis: path.join(outDir, `${prefix}-analysis.md`),
      csv: path.join(outDir, `${prefix}-tracks.csv`),
      primaryBuckets: path.join(outDir, `${prefix}-primary-buckets.json`),
      tags: path.join(outDir, `${prefix}-tags.json`),
      copyPlan: path.join(outDir, `${prefix}-copy-plan.md`),
    },
    null,
    2,
  ),
);

function parseArgs(argv) {
  const parsed = {};
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (!arg.startsWith("--")) continue;
    const key = arg.slice(2);
    const next = argv[i + 1];
    parsed[key] = next && !next.startsWith("--") ? argv[++i] : true;
  }
  return parsed;
}

function usage() {
  console.error(`Usage:
  node organize-playlist.mjs --playlist-id <id> [--out-dir dir] [--prefix name]
  node organize-playlist.mjs --input playlist.raw.json [--out-dir dir] [--prefix name]`);
}

function readJson(file) {
  return JSON.parse(fs.readFileSync(file, "utf8"));
}

function sanitizePrefix(value) {
  return String(value || "ncm-playlist")
    .trim()
    .replace(/[^\w.-]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 80) || "ncm-playlist";
}

async function fetchPlaylist(id) {
  try {
    return await fetchPublicFullPlaylist(id);
  } catch (error) {
    console.error(`public playlist fetch failed: ${error.message}`);
    console.error("falling back to `ncm playlist show`; large playlists may be incomplete");
    const text = execFileSync("ncm", ["playlist", "show", String(id), "--limit", "2000", "--json"], {
      encoding: "utf8",
      maxBuffer: 20 * 1024 * 1024,
    });
    return JSON.parse(text);
  }
}

async function fetchPublicFullPlaylist(id) {
  const detail = await getJson(`https://music.163.com/api/v6/playlist/detail?id=${id}&n=5000&s=0`);
  const playlistDetail = detail.playlist || {};
  const ids = (playlistDetail.trackIds || []).map((item) => item.id).filter(Boolean);
  if (!ids.length) throw new Error("public detail endpoint returned no trackIds");

  const songs = [];
  for (let i = 0; i < ids.length; i += 100) {
    const batch = ids.slice(i, i + 100);
    songs.push(...(await fetchSongDetails(batch)));
    console.error(`fetched ${Math.min(i + batch.length, ids.length)}/${ids.length}`);
  }

  const byId = new Map(songs.map((song) => [song.id, song]));
  const orderedSongs = ids.map((songId) => byId.get(songId)).filter(Boolean);
  return {
    code: 200,
    playlist: {
      id: playlistDetail.id,
      name: playlistDetail.name,
      userId: playlistDetail.userId,
      trackCount: playlistDetail.trackCount,
      subscribed: playlistDetail.subscribed,
      specialType: playlistDetail.specialType,
      privacy: playlistDetail.privacy,
      coverImgUrl: playlistDetail.coverImgUrl,
      creator: playlistDetail.creator
        ? { userId: playlistDetail.creator.userId, nickname: playlistDetail.creator.nickname }
        : undefined,
      tracks: orderedSongs.map(compactSong),
    },
    privileges: [],
  };
}

async function getJson(url) {
  const res = await fetch(url, {
    headers: {
      "User-Agent": "Mozilla/5.0",
      Referer: "https://music.163.com/",
    },
  });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return res.json();
}

async function fetchSongDetails(ids) {
  const c = encodeURIComponent(JSON.stringify(ids.map((id) => ({ id }))));
  const json = await getJson(`https://music.163.com/api/v3/song/detail?c=${c}`);
  if (json.code !== 200) throw new Error(`song detail failed: ${JSON.stringify(json).slice(0, 200)}`);
  return json.songs || [];
}

function compactSong(song) {
  return {
    id: song.id,
    name: song.name,
    ar: (song.ar || []).map((artist) => ({ id: artist.id, name: artist.name })),
    al: song.al ? { id: song.al.id, name: song.al.name } : undefined,
    dt: song.dt,
    fee: song.fee,
  };
}

function enrichTrack(track, index, privilegeMap) {
  const artists = (track.ar || []).map((artist) => artist.name).filter(Boolean);
  const allText = [track.name, artists.join(" "), track.al?.name].filter(Boolean).join(" ");
  const primaryBucketKey = primaryBucketFor(track, artists, allText);
  const tagKeys = tagDefs
    .filter((tag) => tag.test({ ...track, artists, allText, durationMs: track.dt || 0 }))
    .map((tag) => tag.key);
  const privilege = privilegeMap.get(track.id) || {};
  return {
    index: index + 1,
    id: track.id,
    name: track.name,
    artists,
    album: track.al?.name || "",
    durationMs: track.dt || 0,
    fee: track.fee,
    playableLevel: privilege.pl ?? null,
    allText,
    primaryBucketKey,
    tagKeys,
    canonical: `${norm(track.name)} — ${norm(artists.join(" / "))}`,
    titleCanonical: norm(track.name),
  };
}

function primaryBucketFor(track, artists, allText) {
  const title = track.name || "";
  const hasCnTitle = cn.test(title);
  const hasKanaText = kana.test(allText);
  const hasHangulText = hangul.test(allText);
  const hasJpArtist = artists.some((artist) => jpArtists.has(artist) || kana.test(artist));
  const hasKrArtist = artists.some((artist) => krArtists.has(artist) || hangul.test(artist));
  const hasWesternArtist = artists.some((artist) => westernArtists.has(artist));
  const hasInstrumentalArtist = artists.some((artist) => instrumentalArtists.has(artist));
  const hasChineseArtist = artists.some((artist) => cn.test(artist) && !jpArtists.has(artist) && !krArtists.has(artist));
  const hasJapaneseSignals =
    hasJpArtist ||
    hasKanaText ||
    /动漫|动画|anime|ost|op|ed|k-?on|けいおん|ヨルシカ|n-buna|aimyon|あいみょん|daoko|米津|doraemon|哆啦|gto/i.test(allText);

  if (hasKrArtist || hasHangulText) return hasChineseArtist && hasCnTitle ? "chinese_pop" : "korean";
  if (/纯音|钢琴|piano|instrumental|classical|古典|交响|小品/i.test(allText) || (hasInstrumentalArtist && !hasCnTitle)) {
    return "instrumental_focus";
  }
  if (hasJapaneseSignals) return hasChineseArtist && hasCnTitle && !hasKanaText ? "chinese_pop" : "japanese_anime";
  if (hasWesternArtist || (/^[\x00-\x7f\s·'’&.,:!?()+/\-]+$/.test(allText) && /[a-z]/i.test(allText))) return "english_pop";
  if (hasCnTitle || hasChineseArtist || cn.test(allText)) return "chinese_pop";
  return "uncategorized";
}

function norm(value) {
  return String(value || "")
    .toLowerCase()
    .replace(canonicalMarkers, " ")
    .replace(/[-_·'"’“”‘’,，。.!！?？:：/\\|]+/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

function duration(ms) {
  const total = Math.round((ms || 0) / 1000);
  const minutes = Math.floor(total / 60);
  const seconds = String(total % 60).padStart(2, "0");
  return `${minutes}:${seconds}`;
}

function longDuration(ms) {
  const total = Math.round((ms || 0) / 1000);
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const seconds = String(total % 60).padStart(2, "0");
  return hours ? `${hours}h ${minutes}m ${seconds}s` : `${minutes}:${seconds}`;
}

function primaryName(key) {
  return primaryBucketDefs.find((bucket) => bucket.key === key)?.name || "未分类候选";
}

function tagName(key) {
  return tagDefs.find((tag) => tag.key === key)?.name || key;
}

function exportTrack(track) {
  return {
    id: track.id,
    name: track.name,
    artists: track.artists,
    album: track.album,
    duration: duration(track.durationMs),
  };
}

function countBy(items, keyFn) {
  const counts = new Map();
  for (const item of items) {
    const keys = Array.isArray(keyFn(item)) ? keyFn(item) : [keyFn(item)];
    for (const key of keys.filter(Boolean)) counts.set(key, (counts.get(key) || 0) + 1);
  }
  return [...counts.entries()].sort((a, b) => b[1] - a[1] || String(a[0]).localeCompare(String(b[0])));
}

function groupBy(items, keyFn) {
  const groups = new Map();
  for (const item of items) {
    const key = keyFn(item);
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(item);
  }
  return groups;
}

function table(rows, headers) {
  const body = rows.map((row) => headers.map((header) => String(row[header] ?? "")));
  const allRows = [headers, ...body];
  const widths = headers.map((_, i) => Math.min(60, Math.max(...allRows.map((row) => [...row[i]].length))));
  const format = (row) =>
    `| ${row
      .map((cell, i) => {
        const text = [...String(cell)].slice(0, widths[i]).join("");
        return text + " ".repeat(widths[i] - [...text].length);
      })
      .join(" | ")} |`;
  return [format(headers), format(widths.map((width) => "-".repeat(width))), ...body.map(format)].join("\n");
}

function buildAnalysis() {
  const lines = [];
  lines.push(`# ${playlist.name || "网易云歌单"} 整理分析`);
  lines.push("");
  lines.push(`- 歌单 ID：${playlist.id || ""}`);
  lines.push(`- 曲目数：${enriched.length}`);
  lines.push(`- 标称曲目数：${expectedTrackCount}`);
  lines.push(`- 数据完整：${enriched.length === expectedTrackCount ? "是" : "否"}`);
  lines.push(`- 总时长：${longDuration(enriched.reduce((sum, track) => sum + track.durationMs, 0))}`);
  lines.push(`- 无播放权限曲目：${knownPrivilegeCount ? unplayable.length : "未检测（公开补全接口不返回播放权限）"}`);
  lines.push(`- 同名同歌手疑似重复版本：${duplicateGroups.reduce((sum, group) => sum + group.length - 1, 0)}`);
  lines.push("");

  lines.push("## 建议主拆分桶（互斥）");
  lines.push("");
  lines.push(table(primaryBucketCounts.map(([key, count]) => ({ bucket: primaryName(key), count })), ["bucket", "count"]));
  lines.push("");

  lines.push("## 清理标签（可重叠）");
  lines.push("");
  lines.push(table(tagCounts.map(([key, count]) => ({ tag: tagName(key), count })), ["tag", "count"]));
  lines.push("");

  lines.push("## 歌手 Top 40");
  lines.push("");
  lines.push(table(artistCounts.map(([artist, count]) => ({ artist, count })), ["artist", "count"]));
  lines.push("");

  lines.push("## 专辑 Top 30");
  lines.push("");
  lines.push(table(albumCounts.map(([album, count]) => ({ album, count })), ["album", "count"]));
  lines.push("");

  lines.push("## 付费字段分布");
  lines.push("");
  lines.push(table(feeCounts.map(([fee, count]) => ({ fee, count })), ["fee", "count"]));
  lines.push("");

  lines.push("## 疑似重复版本");
  lines.push("");
  const duplicateRows = duplicateGroups.slice(0, 80).flatMap((group) =>
    group.map((track) => ({
      key: track.canonical,
      id: track.id,
      song: track.name,
      artist: track.artists.join(" / "),
      album: track.album,
      duration: duration(track.durationMs),
    })),
  );
  lines.push(duplicateRows.length ? table(duplicateRows, ["key", "id", "song", "artist", "album", "duration"]) : "未发现同名同歌手疑似重复。");
  lines.push("");

  lines.push("## 同名不同歌手");
  lines.push("");
  const titleRows = titleCollisionGroups.slice(0, 80).flatMap((group) =>
    group.map((track) => ({
      title: track.titleCanonical,
      id: track.id,
      song: track.name,
      artist: track.artists.join(" / "),
      album: track.album,
    })),
  );
  lines.push(titleRows.length ? table(titleRows, ["title", "id", "song", "artist", "album"]) : "未发现明显同名不同歌手。");
  lines.push("");
  return lines.join("\n");
}

function buildCopyPlan() {
  const baseName = playlist.name || "Playlist";
  const lines = [];
  lines.push(`# ${baseName} 复制计划`);
  lines.push("");
  lines.push("只建议复制到新歌单；默认不要删除或修改原歌单。");
  lines.push("");
  for (const [name, tracksInBucket] of Object.entries(primaryBuckets)) {
    if (!tracksInBucket.length) continue;
    lines.push(`## ${baseName} · ${name}`);
    lines.push("");
    lines.push(`- 曲目数：${tracksInBucket.length}`);
    lines.push(`- 创建命令：\`ncm playlist create "${baseName} · ${name}" --json\``);
    lines.push(`- 添加歌曲：从 \`${prefix}-primary-buckets.json\` 读取该分组 ID，按 100-200 首一批执行 \`ncm playlist add <new-playlist-id> <song-id...> --json\``);
    lines.push("");
  }
  return lines.join("\n");
}

function writeJson(name, value) {
  fs.writeFileSync(path.join(outDir, name), `${JSON.stringify(value, null, 2)}\n`, "utf8");
}

function writeMarkdown(name, value) {
  fs.writeFileSync(path.join(outDir, name), value, "utf8");
}

function writeCsv(name, tracksForCsv) {
  const rows = [
    ["index", "id", "name", "artists", "album", "duration", "fee", "playableLevel", "primaryBucket", "tags"],
    ...tracksForCsv.map((track) => [
      track.index,
      track.id,
      track.name,
      track.artists.join(" / "),
      track.album,
      duration(track.durationMs),
      track.fee ?? "",
      track.playableLevel ?? "",
      primaryName(track.primaryBucketKey),
      track.tagKeys.map(tagName).join(" / "),
    ]),
  ];
  fs.writeFileSync(path.join(outDir, name), `${rows.map((row) => row.map(csvCell).join(",")).join("\n")}\n`, "utf8");
}

function csvCell(value) {
  const text = String(value ?? "");
  return /[",\n]/.test(text) ? `"${text.replaceAll('"', '""')}"` : text;
}
