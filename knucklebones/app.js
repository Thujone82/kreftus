'use strict';

(() => {
  // =========================================================================
  // 1. CONSTANTS & THEMES (Exact parity with go/knucklebones/theme.go & rules.go)
  // =========================================================================
  const APP_VERSION = '1.0';
  const SAVE_STORAGE_KEY = 'knucklebones.save.v1';
  const DEFAULT_PORT = 8647;
  const GRID_COLS = 3;
  const GRID_ROWS = 3;
  const ROLL_DURATION_MS = 450;
  const EFFECT_DURATION_MS = 650;
  const AI_DELAY_MS = 950;
  const MAX_TIMEOUTS = 3;
  const DISCONNECT_GRACE_MS = 30000;
  const HARD_PLIES = 2;
  const PROTO_VERSION = 2;

  const THEME_ORDER = ['dark', 'light', 'contrast', 'parchment'];
  const THEMES = {
    dark: { name: 'Dark', bg: '#000000', fg: '#ffffff', dim: '#808088', accent: '#f0b840', title: '#e24040', win: '#60d070', lose: '#e85858' },
    light: { name: 'Light', bg: '#f6f2ea', fg: '#201c1c', dim: '#787470', accent: '#b06800', title: '#b82020', win: '#188830', lose: '#c02828' },
    contrast: { name: 'High Contrast', bg: '#000000', fg: '#ffffff', dim: '#ccccd6', accent: '#ffd400', title: '#ff5454', win: '#38f278', lose: '#ff4c4c' },
    parchment: { name: 'Parchment', bg: '#efe3c8', fg: '#2b1d14', dim: '#806c56', accent: '#b25612', title: '#9e1f1f', win: '#287c3a', lose: '#ac2222' }
  };

  const DIFF_ORDER = ['easy', 'medium', 'hard'];
  const DIFF_LABEL = { easy: 'Easy', medium: 'Medium', hard: 'Hard' };
  const AI_NAME = { easy: 'Ratau', medium: 'Flinky', hard: 'Shrumy' };
  const DIFF_HINT = {
    easy: 'Ratau picks a column at random.',
    medium: 'Flinky looks for the biggest swing on the current roll.',
    hard: 'Shrumy weighs every future roll you can get and looks ahead.'
  };

  const ROUND_CHOICES = [1, 3, 5, 0];
  const TIMER_CHOICES = [15, 30, 60, 0];
  const RECORD_KEYS = ['ai-easy', 'ai-medium', 'ai-hard', 'online', 'local'];
  const RECORD_LABELS = {
    'ai-easy': 'vs AI (Easy · Ratau)',
    'ai-medium': 'vs AI (Medium · Flinky)',
    'ai-hard': 'vs AI (Hard · Shrumy)',
    online: 'Online',
    local: 'Local (2 players)'
  };

  const PIP_GRID = [
    [],
    [0, 0, 0, 0, 1, 0, 0, 0, 0],
    [0, 0, 1, 0, 0, 0, 1, 0, 0],
    [0, 0, 1, 0, 1, 0, 1, 0, 0],
    [1, 0, 1, 0, 0, 0, 1, 0, 1],
    [1, 0, 1, 0, 1, 0, 1, 0, 1],
    [1, 0, 1, 1, 0, 1, 1, 0, 1]
  ];

  const GUIDE_CONTENT = [
    { kind: 'h', text: 'How to play' },
    { kind: 'p', text: 'Each turn, you get a random die to place in a column. The goal is to score more points than your opponent when the game ends, which happens when one of the boards is full.' },
    { kind: 'p', text: '• You score even more points when stacking the same dice in a column.' },
    { kind: 'p', text: '• You remove all of your opponent\'s dice with the same value when placing a die in the matching column.' },
    { kind: 'h', text: 'Taking a turn' },
    { kind: 'p', text: '1. A die is rolled for you. A coin toss decides who starts each round.' },
    { kind: 'p', text: '2. Place it in any of your columns with an empty slot. Columns fill from the middle of the table outward, and placed dice never move by choice.' },
    { kind: 'p', text: '3. Every die of the same value in your opponent\'s facing column is destroyed. Their other columns are untouched.' },
    { kind: 'h', text: 'Column scoring' },
    { kind: 'p', text: 'A die is worth its face value times the number of dice of that value in its column. For each value v appearing n times, the column scores n × n × v.' },
    {
      kind: 'table',
      text: [
        '  Column      Calculation           Score',
        '  6           6                         6',
        '  6 2         6 + 2                     8',
        '  6 6         (6×2) + (6×2)            24',
        '  6 6 2       24 + 2                   26',
        '  4 1 4       (4×2) + 1 + (4×2)        17',
        '  6 6 6       (6×3) × 3                54',
        '  3 3 3       (3×3) × 3                27',
        '  1 1 1       (1×3) × 3                 9'
      ].join('\n')
    },
    { kind: 'p', text: 'The best column is three 6s for 54, so the best board is 162.' },
    { kind: 'h', text: 'Ending the game' },
    { kind: 'p', text: 'The game stops the moment either board is full, even mid-comeback. The higher total wins; equal totals are a tie. A best-of series continues until someone wins a majority of rounds. Free play keeps going round after round.' },
    { kind: 'h', text: 'Reading the board' },
    { kind: 'p', text: 'Ivory dice are singles, gold dice are pairs, blue dice are triples. While you choose a column, a shaded die shows where yours will land, red dice are the ones it would destroy, and column totals show the change. Red pips mark the last die placed.' },
    { kind: 'h', text: 'House rules' },
    { kind: 'p', text: 'Set these in Game settings before you start.' },
    { kind: 'p', text: '• Destroy matching dice: Off plays the quiet, stacking-only game.' },
    { kind: 'p', text: '• Dice per turn: Two rolls two dice and you place one. Tab or W switches between them.' },
    { kind: 'p', text: '• Game ends when: Both boards are full removes the close-the-board tactic.' },
    { kind: 'h', text: 'Strategy' },
    { kind: 'p', text: '• Value swings, not scores. Gaining 6 while taking 24 away is worth 30.' },
    { kind: 'p', text: '• Stack high, spread low. Two 6s are 24; two 1s are only 4.' },
    { kind: 'p', text: '• Stacks get safer near the end. Your opponent rolls the value that destroys a stack 1 time in 6, every turn.' },
    { kind: 'p', text: '• Answer a stack with a single die: a lone 5 across from their 5, 5 removes 20.' },
    { kind: 'p', text: '• Control the clock. Ahead? Fill your board. Behind? Keep a column open so you are not forced to end the game.' },
    { kind: 'p', text: '• Use quiet columns, where the opposite dice cannot match yours, for low dice.' },
    { kind: 'h', text: 'Opponents' },
    { kind: 'p', text: '• Ratau (Easy) drops dice in any open column.' },
    { kind: 'p', text: '• Flinky (Medium) takes the biggest immediate swing.' },
    { kind: 'p', text: '• Shrumy (Hard) averages your best reply over every roll, looks one step further, and knows when ending the game wins or loses it.' },
    { kind: 'h', text: 'Online play & Fair Play' },
    { kind: 'p', text: 'Hosts run on the desktop/terminal Knucklebones app; pick Join Game and enter the host IP address and port (default 8647). Optional room codes authenticate the handshake with HMAC-SHA256. Host dice are verified via SHA-256 commit-reveal at round end. Turns can be timed (three timeouts forfeit the match), and dropped players have 30 seconds to reconnect.' },
    { kind: 'h', text: 'Credits' },
    { kind: 'p', text: 'The original Knucklebones game in Cult of the Lamb was created by Massive Monster. This is a fan-made web PWA edition with full parity to the desktop Go/Ebitengine and Terminal editions.' }
  ];

  // =========================================================================
  // 2. SYNCHRONOUS SHA-256 / HMAC-SHA256 / HKDF-SHA256 (crypto.go parity)
  // =========================================================================
  const K256 = new Uint32Array([
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
    0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
    0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
    0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
    0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
    0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2
  ]);

  const utf8Enc = new TextEncoder();

  function toBytes(input) {
    return input instanceof Uint8Array ? input : utf8Enc.encode(String(input));
  }

  function sha256Bytes(input) {
    const msg = toBytes(input);
    const bitLen = msg.length * 8;
    const paddedLen = Math.ceil((msg.length + 9) / 64) * 64;
    const padded = new Uint8Array(paddedLen);
    padded.set(msg);
    padded[msg.length] = 0x80;
    const dv = new DataView(padded.buffer);
    dv.setUint32(paddedLen - 8, Math.floor(bitLen / 0x100000000), false);
    dv.setUint32(paddedLen - 4, bitLen >>> 0, false);

    let h0 = 0x6a09e667, h1 = 0xbb67ae85, h2 = 0x3c6ef372, h3 = 0xa54ff53a;
    let h4 = 0x510e527f, h5 = 0x9b05688c, h6 = 0x1f83d9ab, h7 = 0x5be0cd19;
    const w = new Uint32Array(64);
    const rotr = (n, x) => (x >>> n) | (x << (32 - n));

    for (let off = 0; off < paddedLen; off += 64) {
      for (let i = 0; i < 16; i++) w[i] = dv.getUint32(off + i * 4, false);
      for (let i = 16; i < 64; i++) {
        const s0 = rotr(7, w[i - 15]) ^ rotr(18, w[i - 15]) ^ (w[i - 15] >>> 3);
        const s1 = rotr(17, w[i - 2]) ^ rotr(19, w[i - 2]) ^ (w[i - 2] >>> 10);
        w[i] = (w[i - 16] + s0 + w[i - 7] + s1) >>> 0;
      }
      let a = h0, b = h1, c = h2, d = h3, e = h4, f = h5, g = h6, h = h7;
      for (let i = 0; i < 64; i++) {
        const S1 = rotr(6, e) ^ rotr(11, e) ^ rotr(25, e);
        const ch = (e & f) ^ (~e & g);
        const temp1 = (h + S1 + ch + K256[i] + w[i]) >>> 0;
        const S0 = rotr(2, a) ^ rotr(13, a) ^ rotr(22, a);
        const maj = (a & b) ^ (a & c) ^ (b & c);
        const temp2 = (S0 + maj) >>> 0;
        h = g; g = f; f = e; e = (d + temp1) >>> 0;
        d = c; c = b; b = a; a = (temp1 + temp2) >>> 0;
      }
      h0 = (h0 + a) >>> 0; h1 = (h1 + b) >>> 0; h2 = (h2 + c) >>> 0; h3 = (h3 + d) >>> 0;
      h4 = (h4 + e) >>> 0; h5 = (h5 + f) >>> 0; h6 = (h6 + g) >>> 0; h7 = (h7 + h) >>> 0;
    }
    const out = new Uint8Array(32);
    const odv = new DataView(out.buffer);
    [h0, h1, h2, h3, h4, h5, h6, h7].forEach((val, idx) => odv.setUint32(idx * 4, val, false));
    return out;
  }

  function bytesToHex(bytes) {
    let hex = '';
    for (let i = 0; i < bytes.length; i++) hex += bytes[i].toString(16).padStart(2, '0');
    return hex;
  }

  function hmacSha256Bytes(keyInput, msgInput) {
    let key = toBytes(keyInput);
    if (key.length > 64) key = sha256Bytes(key);
    const ipad = new Uint8Array(64);
    const opad = new Uint8Array(64);
    ipad.fill(0x36);
    opad.fill(0x5c);
    for (let i = 0; i < key.length; i++) {
      ipad[i] ^= key[i];
      opad[i] ^= key[i];
    }
    const msg = toBytes(msgInput);
    const inner = new Uint8Array(64 + msg.length);
    inner.set(ipad);
    inner.set(msg, 64);
    const innerHash = sha256Bytes(inner);
    const outer = new Uint8Array(64 + 32);
    outer.set(opad);
    outer.set(innerHash, 64);
    return sha256Bytes(outer);
  }

  function hkdfSha256(ikm, salt, info) {
    const prk = hmacSha256Bytes(salt, ikm);
    const infoBytes = toBytes(info);
    const tInput = new Uint8Array(infoBytes.length + 1);
    tInput.set(infoBytes);
    tInput[infoBytes.length] = 1;
    return hmacSha256Bytes(prk, tInput);
  }

  function newRandomHex(nBytes) {
    const arr = new Uint8Array(nBytes);
    crypto.getRandomValues(arr);
    return bytesToHex(arr);
  }

  function commitSeed(seedHex) {
    return bytesToHex(sha256Bytes('kb-seed-v2|' + seedHex));
  }

  function verifySeedCommit(seedHex, commitHex) {
    if (!seedHex || !commitHex) return false;
    return commitSeed(seedHex) === commitHex;
  }

  function roundCoin(hostSeed, guestSeed) {
    return hmacSha256Bytes(hostSeed, 'coin-v2|' + guestSeed)[0] & 1;
  }

  function roundRoll(hostSeed, guestSeed, moveIdx, twoDice) {
    const need = twoDice ? 2 : 1;
    const out = [0, 0];
    let idx = 0;
    for (let ctr = 0; idx < need; ctr++) {
      const sum = hmacSha256Bytes(hostSeed, `roll-v2|${guestSeed}|${moveIdx}|${ctr}`);
      for (let i = 0; i < sum.length && idx < need; i++) {
        if (sum[i] < 252) {
          out[idx++] = (sum[i] % 6) + 1;
        }
      }
    }
    return out;
  }

  function deriveRoomKey(roomCode) {
    const clean = cleanRoomCode(roomCode);
    if (!clean) return null;
    return hkdfSha256(toBytes(clean), toBytes('knucklebones-v2-room'), 'room-auth');
  }

  function deriveWireKey(roomKey, nonce) {
    if (!roomKey || roomKey.length === 0) return null;
    return hkdfSha256(roomKey, toBytes(nonce), 'wire-mac-v2');
  }

  function helloAuthMAC(roomKey, nonce, name, token, guestSeed) {
    if (!roomKey) return '';
    return bytesToHex(hmacSha256Bytes(roomKey, `hello\x00${nonce}\x00${name}\x00${token}\x00${guestSeed}`));
  }

  function validHex(s, maxLen) {
    if (typeof s !== 'string' || s.length === 0 || s.length > maxLen) return false;
    return /^[0-9a-f]+$/i.test(s);
  }

  function canonicalRules(r) {
    const o = {};
    if (r && r.noDestroy) o.noDestroy = true;
    if (r && r.twoDice) o.twoDice = true;
    if (r && r.fillBoth) o.fillBoth = true;
    return o;
  }

  function canonicalMatch(m) {
    const src = m || {};
    const grids = [0, 1].map((pi) => [0, 1, 2].map((c) => {
      const col = src.grids && src.grids[pi] && src.grids[pi][c];
      return col ? [col[0] || 0, col[1] || 0, col[2] || 0] : [0, 0, 0];
    }));
    const dice = src.dice ? [src.dice[0] || 0, src.dice[1] || 0] : [0, 0];
    return {
      rules: canonicalRules(src.rules),
      grids,
      turn: src.turn || 0,
      dice,
      over: !!src.over,
      moves: src.moves || 0
    };
  }

  function canonicalLogEntry(e) {
    const o = { round: e.round || 0, kind: e.kind || '', player: e.player || 0 };
    if (e.die) o.die = e.die;
    if (e.col) o.col = e.col;
    if (e.destroyed) o.destroyed = e.destroyed;
    if (e.gain) o.gain = e.gain;
    if (e.lost) o.lost = e.lost;
    if (e.count) o.count = e.count;
    o.scores = e.scores ? [e.scores[0] || 0, e.scores[1] || 0] : [0, 0];
    if (e.reason) o.reason = e.reason;
    return o;
  }

  function canonicalSession(s) {
    const o = {
      mode: s.mode || '',
      names: s.names ? [String(s.names[0] || ''), String(s.names[1] || '')] : ['', '']
    };
    if (s.difficulty) o.difficulty = s.difficulty;
    o.bestOf = s.bestOf || 0;
    if (s.freePlay) o.freePlay = true;
    o.rules = canonicalRules(s.rules);
    o.match = canonicalMatch(s.match);
    o.series = s.series || 0;
    o.round = s.round || 0;
    o.wins = s.wins ? [s.wins[0] || 0, s.wins[1] || 0] : [0, 0];
    o.ties = s.ties || 0;
    o.phase = s.phase || '';
    o.log = Array.isArray(s.log) ? s.log.map(canonicalLogEntry) : [];
    o.history = Array.isArray(s.history) ? s.history.map((h) => ({
      round: h.round || 0,
      scores: h.scores ? [h.scores[0] || 0, h.scores[1] || 0] : [0, 0],
      winner: h.winner ?? 0,
      lead: h.lead ?? 0
    })) : [];
    o.timeouts = s.timeouts ? [s.timeouts[0] || 0, s.timeouts[1] || 0] : [0, 0];
    o.votes = s.votes ? [String(s.votes[0] || ''), String(s.votes[1] || '')] : ['', ''];
    if (s.forfeit) o.forfeit = { reason: s.forfeit.reason || '', loser: s.forfeit.loser || 0 };
    if (s.last) {
      o.last = {
        player: s.last.player || 0,
        col: s.last.col || 0,
        slot: s.last.slot || 0,
        die: s.last.die || 0,
        destroyed: s.last.destroyed || 0,
        gain: s.last.gain || 0,
        lost: s.last.lost || 0
      };
    }
    o.connected = s.connected ? [!!s.connected[0], !!s.connected[1]] : [false, false];
    if (s.turnLimit) o.turnLimit = s.turnLimit;
    if (s.turnLeftMs) o.turnLeftMs = s.turnLeftMs;
    if (s.waitLeftMs) o.waitLeftMs = s.waitLeftMs;
    if (s.aiming) o.aiming = true;
    if (s.aimCol) o.aimCol = s.aimCol;
    if (s.aimDie) o.aimDie = s.aimDie;
    if (s.hostCommit) o.hostCommit = s.hostCommit;
    if (s.guestSeed) o.guestSeed = s.guestSeed;
    if (s.hostReveal) o.hostReveal = s.hostReveal;
    if (s.nextHostCommit) o.nextHostCommit = s.nextHostCommit;
    if (s.roundFirst) o.roundFirst = s.roundFirst;
    if (Array.isArray(s.roundRolls) && s.roundRolls.length > 0) {
      o.roundRolls = s.roundRolls.map((r) => [r[0] || 0, r[1] || 0]);
    }
    if (Array.isArray(s.roundMoves) && s.roundMoves.length > 0) {
      o.roundMoves = s.roundMoves.map((mv) => ({
        p: mv.p || 0,
        c: mv.c || 0,
        d: mv.d || 0,
        dice: mv.dice ? [mv.dice[0] || 0, mv.dice[1] || 0] : [0, 0]
      }));
    }
    return o;
  }

  function canonicalMsgPayload(m) {
    const o = { type: m.type || '' };
    if (m.version) o.version = m.version;
    if (m.seq) o.seq = m.seq;
    // mac omitted intentionally when computing canonical payload
    if (m.nonce) o.nonce = m.nonce;
    if (m.authReq) o.authReq = true;
    if (m.auth) o.auth = m.auth;
    if (m.commit) o.commit = m.commit;
    if (m.seed) o.seed = m.seed;
    if (m.name) o.name = m.name;
    if (m.token) o.token = m.token;
    if (m.aiming) o.aiming = true;
    if (m.col) o.col = m.col;
    if (m.die) o.die = m.die;
    if (m.vote) o.vote = m.vote;
    if (m.text) o.text = m.text;
    if (m.state) o.state = canonicalSession(m.state);
    // Go json.Marshal escapes <, >, & as \u003c, \u003e, \u0026
    return JSON.stringify(o).replace(/[<>&]/g, (ch) => (ch === '<' ? '\\u003c' : ch === '>' ? '\\u003e' : '\\u0026'));
  }

  function signWireMsg(wireKey, seq, m) {
    if (!wireKey || wireKey.length === 0) return m;
    m.seq = seq;
    m.mac = bytesToHex(hmacSha256Bytes(wireKey, canonicalMsgPayload(m)));
    return m;
  }

  function verifyWireMsg(wireKey, expectedMinSeq, m) {
    if (!wireKey || wireKey.length === 0) return true;
    if (!m || typeof m.seq !== 'number' || m.seq < expectedMinSeq) return false;
    if (typeof m.mac !== 'string' || m.mac.length !== 64 || !validHex(m.mac, 64)) return false;
    const expected = bytesToHex(hmacSha256Bytes(wireKey, canonicalMsgPayload(m)));
    return m.mac.toLowerCase() === expected;
  }

  function splitHostPort(addr) {
    const s = String(addr || '').trim();
    if (!s) return null;
    if (s.startsWith('[')) {
      const end = s.indexOf(']');
      if (end < 0) return null;
      const host = s.slice(1, end).trim();
      const rest = s.slice(end + 1);
      if (rest === '') return { host, port: '' };
      if (!rest.startsWith(':')) return null;
      return { host, port: rest.slice(1).trim() };
    }
    const firstColon = s.indexOf(':');
    const lastColon = s.lastIndexOf(':');
    if (firstColon !== lastColon) {
      // Bare IPv6 address without brackets
      return { host: s, port: '', bareV6: true };
    }
    if (lastColon >= 0) {
      return { host: s.slice(0, lastColon).trim(), port: s.slice(lastColon + 1).trim() };
    }
    return { host: s, port: '' };
  }

  function formatHostPort(host, port) {
    return host.includes(':') ? `[${host}]:${port}` : `${host}:${port}`;
  }

  function normalizeJoinAddr(raw) {
    let addr = String(raw || '').trim();
    if (!addr) return '';
    addr = addr.replace(/^wss?:\/\//i, '').replace(/\/.*$/, '').trim();
    if (!addr) return '';
    const sp = splitHostPort(addr);
    if (!sp || !sp.host) return addr;
    const port = sp.port || String(DEFAULT_PORT);
    return formatHostPort(sp.host, port);
  }

  function isLoopbackHost(host) {
    const h = String(host || '').toLowerCase();
    return h === 'localhost' || h === '127.0.0.1' || h === '::1' || h === '[::1]' || h.endsWith('.localhost');
  }

  function buildWsUrl(rawAddr) {
    const s = String(rawAddr || '').trim();
    if (/^wss?:\/\//i.test(s)) {
      const scheme = s.slice(0, s.indexOf('://')).toLowerCase();
      const rest = s.slice(s.indexOf('://') + 3).replace(/\/.*$/, '');
      const sp = splitHostPort(rest);
      const hostPort = sp && sp.host ? formatHostPort(sp.host, sp.port || String(DEFAULT_PORT)) : rest;
      return `${scheme}://${hostPort}`;
    }
    const norm = normalizeJoinAddr(s);
    const sp = splitHostPort(norm);
    const loop = sp && isLoopbackHost(sp.host);
    const scheme = (location.protocol === 'https:' && !loop) ? 'wss' : 'ws';
    return `${scheme}://${norm}`;
  }

  // =========================================================================
  // 3. PROCEDURAL WEB AUDIO SYNTHESIZER (Exact parity with sound_gui.go)
  // =========================================================================
  const AudioEngine = {
    ctx: null,
    noiseBuffer: null,

    ensure() {
      if (this.ctx) {
        if (this.ctx.state === 'suspended') this.ctx.resume().catch(() => {});
        return this.ctx;
      }
      const Ctor = window.AudioContext || window.webkitAudioContext;
      if (!Ctor) return null;
      this.ctx = new Ctor();
      const len = Math.floor(this.ctx.sampleRate * 0.5);
      const buf = this.ctx.createBuffer(1, len, this.ctx.sampleRate);
      const data = buf.getChannelData(0);
      for (let i = 0; i < len; i++) data[i] = Math.random() * 2 - 1;
      this.noiseBuffer = buf;
      return this.ctx;
    },

    unlock() {
      this.ensure();
    },

    play(id, enabled) {
      if (!enabled || !id) return;
      const ctx = this.ensure();
      if (!ctx) return;
      const now = ctx.currentTime;
      switch (id) {
        case 'click':
          this.tone(now, 'triangle', 760, 520, 0.028, 0.14);
          break;
        case 'place':
          this.tone(now, 'triangle', 250, 115, 0.075, 0.28);
          this.noise(now, 540, 220, 0.024, 0.10);
          break;
        case 'destroy':
          this.noise(now, 360, 85, 0.130, 0.30);
          this.tone(now, 'square', 220, 110, 0.110, 0.24);
          break;
        case 'turn':
          this.tone(now, 'sine', 523.25, 523.25, 0.095, 0.16);
          break;
        case 'win':
          this.tone(now + 0.000, 'sine', 523.25, 523.25, 0.120, 0.18);
          this.tone(now + 0.085, 'sine', 659.25, 659.25, 0.130, 0.18);
          this.tone(now + 0.170, 'sine', 783.99, 783.99, 0.210, 0.20);
          break;
        case 'lose':
          this.tone(now + 0.000, 'triangle', 329.63, 311.13, 0.140, 0.20);
          this.tone(now + 0.120, 'triangle', 246.94, 220.00, 0.230, 0.20);
          break;
      }
    },

    tone(start, type, f0, f1, dur, amp) {
      const ctx = this.ctx;
      const osc = ctx.createOscillator();
      const gain = ctx.createGain();
      osc.type = type;
      osc.frequency.setValueAtTime(f0, start);
      if (f1 !== f0) osc.frequency.exponentialRampToValueAtTime(Math.max(20, f1), start + dur);
      gain.gain.setValueAtTime(0.001, start);
      gain.gain.linearRampToValueAtTime(amp, start + Math.min(0.005, dur * 0.2));
      gain.gain.exponentialRampToValueAtTime(0.0001, start + dur);
      osc.connect(gain);
      gain.connect(ctx.destination);
      osc.start(start);
      osc.stop(start + dur + 0.01);
    },

    noise(start, cut0, cut1, dur, amp) {
      const ctx = this.ctx;
      const src = ctx.createBufferSource();
      src.buffer = this.noiseBuffer;
      const filter = ctx.createBiquadFilter();
      filter.type = 'lowpass';
      filter.frequency.setValueAtTime(cut0, start);
      filter.frequency.exponentialRampToValueAtTime(Math.max(20, cut1), start + dur);
      const gain = ctx.createGain();
      gain.gain.setValueAtTime(amp, start);
      gain.gain.exponentialRampToValueAtTime(0.0001, start + dur);
      src.connect(filter);
      filter.connect(gain);
      gain.connect(ctx.destination);
      src.start(start);
      src.stop(start + dur + 0.01);
    }
  };

  // =========================================================================
  // 4. CORE GAME RULES & ENGINE (rules.go parity)
  // =========================================================================
  function emptyGrid() {
    return [[0, 0, 0], [0, 0, 0], [0, 0, 0]];
  }

  function cloneGrid(g) {
    return [g[0].slice(), g[1].slice(), g[2].slice()];
  }

  function columnScore(col) {
    const counts = [0, 0, 0, 0, 0, 0, 0];
    for (let k = 0; k < GRID_ROWS; k++) {
      const v = col[k];
      if (v >= 1 && v <= 6) counts[v]++;
    }
    let total = 0;
    for (let v = 1; v <= 6; v++) {
      if (counts[v] > 0) total += counts[v] * counts[v] * v;
    }
    return total;
  }

  function gridScore(grid) {
    return columnScore(grid[0]) + columnScore(grid[1]) + columnScore(grid[2]);
  }

  function matchesInCol(col, val) {
    if (val < 1 || val > 6) return 0;
    let n = 0;
    for (let k = 0; k < GRID_ROWS; k++) if (col[k] === val) n++;
    return n;
  }

  function openSlot(col) {
    for (let k = 0; k < GRID_ROWS; k++) if (col[k] === 0) return k;
    return -1;
  }

  function gridFull(grid) {
    for (let c = 0; c < GRID_COLS; c++) {
      if (openSlot(grid[c]) >= 0) return false;
    }
    return true;
  }

  function destroyInCol(col, val) {
    let removed = 0;
    let w = 0;
    for (let k = 0; k < GRID_ROWS; k++) {
      const x = col[k];
      if (x === 0) continue;
      if (x === val) {
        removed++;
        continue;
      }
      col[w++] = x;
    }
    while (w < GRID_ROWS) col[w++] = 0;
    return removed;
  }

  function createMatch(rules, first) {
    return {
      rules: { noDestroy: !!rules.noDestroy, twoDice: !!rules.twoDice, fillBoth: !!rules.fillBoth },
      grids: [emptyGrid(), emptyGrid()],
      turn: first ? 1 : 0,
      dice: [0, 0],
      moves: 0
    };
  }

  function cloneMatch(m) {
    return {
      rules: { ...m.rules },
      grids: [cloneGrid(m.grids[0]), cloneGrid(m.grids[1])],
      turn: m.turn,
      dice: [m.dice[0], m.dice[1]],
      moves: m.moves
    };
  }

  function matchFinished(m) {
    const f0 = gridFull(m.grids[0]);
    const f1 = gridFull(m.grids[1]);
    return m.rules.fillBoth ? (f0 && f1) : (f0 || f1);
  }

  function matchScores(m) {
    return [gridScore(m.grids[0]), gridScore(m.grids[1])];
  }

  function matchWinner(m) {
    const s = matchScores(m);
    if (s[0] > s[1]) return 0;
    if (s[1] > s[0]) return 1;
    return -1;
  }

  function legalMove(m, mv) {
    if (matchFinished(m)) return false;
    if (mv.col < 0 || mv.col >= GRID_COLS || mv.die < 0 || mv.die > 1) return false;
    return m.dice[mv.die] !== 0 && openSlot(m.grids[m.turn][mv.col]) >= 0;
  }

  function legalMoves(m) {
    if (matchFinished(m)) return [];
    const out = [];
    const diceCount = m.dice[1] !== 0 && m.dice[1] !== m.dice[0] ? 2 : 1;
    for (let d = 0; d < diceCount; d++) {
      if (m.dice[d] === 0) continue;
      for (let c = 0; c < GRID_COLS; c++) {
        if (openSlot(m.grids[m.turn][c]) >= 0) out.push({ col: c, die: d });
      }
    }
    return out;
  }

  function applyPlace(m, mv) {
    if (!legalMove(m, mv)) return null;
    const p = m.turn;
    const opp = 1 - p;
    const v = m.dice[mv.die];
    const slot = openSlot(m.grids[p][mv.col]);

    const beforeMy = columnScore(m.grids[p][mv.col]);
    const beforeOpp = columnScore(m.grids[opp][mv.col]);

    m.grids[p][mv.col][slot] = v;
    let destroyed = 0;
    if (!m.rules.noDestroy) {
      destroyed = destroyInCol(m.grids[opp][mv.col], v);
    }

    const afterMy = columnScore(m.grids[p][mv.col]);
    const afterOpp = columnScore(m.grids[opp][mv.col]);

    m.dice = [0, 0];
    m.moves++;
    if (!matchFinished(m) && !gridFull(m.grids[opp])) {
      m.turn = opp;
    }

    return {
      player: p,
      col: mv.col,
      slot,
      die: v,
      destroyed,
      gain: afterMy - beforeMy,
      lost: beforeOpp - afterOpp
    };
  }

  function previewPlace(m, mv) {
    const next = cloneMatch(m);
    const pl = applyPlace(next, mv);
    return { pl, next };
  }

  function rollForTurn(m, randDieFn) {
    m.dice = [randDieFn(), m.rules.twoDice ? randDieFn() : 0];
  }

  // =========================================================================
  // 5. AI PERSONALITIES & WIN PROBABILITY (ai.go parity)
  // =========================================================================
  const TWO_DIE_OUTCOMES = (() => {
    const list = [];
    for (let a = 1; a <= 6; a++) {
      for (let b = a; b <= 6; b++) {
        list.push({ a, b, prob: a === b ? 1 / 36 : 2 / 36 });
      }
    }
    return list;
  })();

  function chooseAIMove(m, diff) {
    const moves = legalMoves(m);
    if (moves.length === 0) return { col: 0, die: 0 };
    if (diff === 'easy') {
      return moves[Math.floor(Math.random() * moves.length)];
    }
    const player = m.turn;
    if (diff === 'medium') {
      return bestBy(moves, (mv) => {
        const { next } = previewPlace(m, mv);
        const s = matchScores(next);
        return s[player] - s[1 - player];
      });
    }
    const depth = m.rules.twoDice ? 1 : HARD_PLIES;
    return bestBy(moves, (mv) => {
      const { next } = previewPlace(m, mv);
      return -expectimax(next, depth);
    });
  }

  function bestBy(moves, evalFn) {
    let best = -Infinity;
    let tied = [];
    for (const mv of moves) {
      const val = evalFn(mv);
      if (val > best + 1e-9) {
        best = val;
        tied = [mv];
      } else if (Math.abs(val - best) <= 1e-9) {
        tied.push(mv);
      }
    }
    return tied[Math.floor(Math.random() * tied.length)];
  }

  function evaluateTerminal(m, p) {
    const s = matchScores(m);
    const diff = s[p] - s[1 - p];
    if (diff > 0) return 1000 + diff;
    if (diff < 0) return -1000 + diff;
    return 0;
  }

  function expectimax(m, depth) {
    const p = m.turn;
    if (matchFinished(m)) return evaluateTerminal(m, p);
    if (depth === 0) {
      const s = matchScores(m);
      return s[p] - s[1 - p];
    }
    let total = 0;
    if (!m.rules.twoDice) {
      for (let d = 1; d <= 6; d++) {
        const next = cloneMatch(m);
        next.dice = [d, 0];
        total += bestValue(next, depth) / 6;
      }
      return total;
    }
    for (const o of TWO_DIE_OUTCOMES) {
      const next = cloneMatch(m);
      next.dice = [o.a, o.b];
      total += bestValue(next, depth) * o.prob;
    }
    return total;
  }

  function bestValue(m, depth) {
    const p = m.turn;
    let best = -Infinity;
    for (const mv of legalMoves(m)) {
      const { next } = previewPlace(m, mv);
      let v = expectimax(next, depth - 1);
      if (next.turn !== p) v = -v;
      if (v > best) best = v;
    }
    return best;
  }

  function seedFromMatch(m) {
    let h = 2166136261 >>> 0;
    const mix = (x) => {
      h ^= x >>> 0;
      h = Math.imul(h, 16777619) >>> 0;
    };
    mix(m.turn + 1);
    mix(m.moves);
    mix(m.dice[0]);
    mix(m.dice[1]);
    mix(m.rules.noDestroy ? 1 : 0);
    mix(m.rules.twoDice ? 2 : 0);
    mix(m.rules.fillBoth ? 4 : 0);
    for (let p = 0; p < 2; p++) {
      for (let c = 0; c < GRID_COLS; c++) {
        for (let k = 0; k < GRID_ROWS; k++) mix(m.grids[p][c][k]);
      }
    }
    return h || 1;
  }

  function estimateWinProb(m, seat, rollouts = 96) {
    if (matchFinished(m)) {
      const w = matchWinner(m);
      if (w < 0) return 0.5;
      return w === seat ? 1 : 0;
    }
    let state = seedFromMatch(m);
    const rand = () => {
      state = (state + 0x6d2b79f5) | 0;
      let t = Math.imul(state ^ (state >>> 15), 1 | state);
      t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
      return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    };
    const randInt = (n) => Math.floor(rand() * n);

    let wins = 0;
    for (let r = 0; r < rollouts; r++) {
      const sim = cloneMatch(m);
      while (!matchFinished(sim)) {
        if (sim.dice[0] === 0) {
          rollForTurn(sim, () => randInt(6) + 1);
        }
        const moves = legalMoves(sim);
        if (moves.length === 0) break;
        let chosen;
        if (rand() < 0.85) {
          const p = sim.turn;
          let best = -Infinity;
          let tied = [];
          for (const mv of moves) {
            const { next } = previewPlace(sim, mv);
            const s = matchScores(next);
            let sc = s[p] - s[1 - p];
            if (matchFinished(next)) {
              if (sc > 0) sc += 500;
              else if (sc < 0) sc -= 500;
            }
            if (sc > best + 1e-9) { best = sc; tied = [mv]; }
            else if (Math.abs(sc - best) <= 1e-9) tied.push(mv);
          }
          chosen = tied[randInt(tied.length)];
        } else {
          chosen = moves[randInt(moves.length)];
        }
        applyPlace(sim, chosen);
      }
      const w = matchWinner(sim);
      if (w < 0) wins += 0.5;
      else if (w === seat) wins += 1;
    }
    return wins / rollouts;
  }

  // =========================================================================
  // 6. SAVE DATA, STATS & HELPERS (save.go & viewmodel.go parity)
  // =========================================================================
  function cleanName(raw, fallback = 'Lamb') {
    const cleaned = String(raw || '').replace(/[\r\n\t\u0000-\u001f\u007f]/g, ' ').trim().slice(0, 16);
    return cleaned || fallback;
  }

  function cleanRoomCode(raw) {
    return String(raw || '').replace(/\s+/g, '').replace(/[\u0000-\u001f\u007f]/g, '').slice(0, 12);
  }

  function cleanText(raw, maxLen) {
    const cleaned = String(raw || '').replace(/[\r\n\t\u0000-\u001f\u007f]/g, ' ').trim();
    return maxLen > 0 ? cleaned.slice(0, maxLen) : cleaned;
  }

  function defaultSave() {
    return {
      version: APP_VERSION,
      name: 'Lamb',
      theme: 'dark',
      displayMode: 'gui',
      settings: {
        difficulty: 'medium',
        bestOf: 1,
        rules: { noDestroy: false, twoDice: false, fillBoth: false },
        turnSeconds: 30,
        joinAddr: `localhost:${DEFAULT_PORT}`,
        player2: 'Player 2',
        sound: true,
        haptics: true,
        quickTap: false,
        colorblind: false,
        winMeter: false
      },
      records: {},
      highScores: []
    };
  }

  function sanitizeSave(raw) {
    const base = defaultSave();
    if (!raw || typeof raw !== 'object') return base;
    base.name = cleanName(raw.name, 'Lamb');
    if (THEMES[raw.theme]) base.theme = raw.theme;
    if (raw.displayMode === 'tui' || raw.displayMode === 'gui') base.displayMode = raw.displayMode;
    const st = raw.settings || {};
    if (DIFF_LABEL[st.difficulty]) base.settings.difficulty = st.difficulty;
    if ([0, 1, 3, 5].includes(st.bestOf)) base.settings.bestOf = st.bestOf;
    if ([0, 15, 30, 60].includes(st.turnSeconds)) base.settings.turnSeconds = st.turnSeconds;
    const ja = normalizeJoinAddr(cleanText(st.joinAddr, 80));
    base.settings.joinAddr = ja || `localhost:${DEFAULT_PORT}`;
    base.settings.player2 = cleanName(st.player2, 'Player 2');
    base.settings.sound = st.sound !== undefined ? !!st.sound : true;
    base.settings.haptics = st.haptics !== undefined ? !!st.haptics : true;
    base.settings.quickTap = !!st.quickTap;
    base.settings.colorblind = !!st.colorblind;
    base.settings.winMeter = !!st.winMeter;
    const rr = st.rules || {};
    base.settings.rules = {
      noDestroy: !!rr.noDestroy,
      twoDice: !!rr.twoDice,
      fillBoth: !!rr.fillBoth
    };
    if (raw.records && typeof raw.records === 'object') {
      for (const k of Object.keys(raw.records)) {
        const r = raw.records[k];
        if (!r || typeof r !== 'object') continue;
        base.records[k] = {
          games: Math.max(0, Number(r.games) || 0),
          wins: Math.max(0, Number(r.wins) || 0),
          losses: Math.max(0, Number(r.losses) || 0),
          draws: Math.max(0, Number(r.draws) || 0),
          best: Math.max(0, Number(r.best) || 0),
          seriesWins: Math.max(0, Number(r.seriesWins) || 0),
          seriesLosses: Math.max(0, Number(r.seriesLosses) || 0)
        };
      }
    }
    if (Array.isArray(raw.highScores)) {
      base.highScores = raw.highScores
        .filter((e) => e && typeof e.score === 'number')
        .map((e) => ({
          name: cleanName(e.name, 'Lamb'),
          score: Math.max(0, Math.round(e.score)),
          opponent: cleanName(e.opponent, 'Opponent'),
          against: Math.max(0, Math.round(Number(e.against) || 0)),
          mode: String(e.mode || 'ai'),
          date: String(e.date || todayISO())
        }))
        .sort((a, b) => b.score - a.score || a.against - b.against)
        .slice(0, 10);
    }
    return base;
  }

  function loadSave() {
    try {
      const txt = localStorage.getItem(SAVE_STORAGE_KEY);
      if (!txt) return defaultSave();
      return sanitizeSave(JSON.parse(txt));
    } catch {
      return defaultSave();
    }
  }

  function persistSave(save) {
    try {
      localStorage.setItem(SAVE_STORAGE_KEY, JSON.stringify(save));
    } catch { /* ignore quota errors */ }
  }

  function todayISO() {
    const d = new Date();
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${y}-${m}-${day}`;
  }

  function recordKey(mode, difficulty) {
    if (mode === 'ai') return 'ai-' + (difficulty || 'medium');
    if (mode === 'online') return 'online';
    return 'local';
  }

  function getRecord(save, key) {
    if (!save.records[key]) {
      save.records[key] = { games: 0, wins: 0, losses: 0, draws: 0, best: 0, seriesWins: 0, seriesLosses: 0 };
    }
    return save.records[key];
  }

  function addHighScore(save, entry) {
    const item = {
      name: cleanName(entry.name, 'Lamb'),
      score: entry.score,
      opponent: cleanName(entry.opponent, 'Opponent'),
      against: entry.against,
      mode: entry.mode,
      date: todayISO()
    };
    save.highScores.push(item);
    save.highScores.sort((a, b) => b.score - a.score || a.against - b.against);
    if (save.highScores.length > 10) save.highScores = save.highScores.slice(0, 10);
    const idx = save.highScores.indexOf(item);
    return idx >= 0 ? idx + 1 : 0;
  }

  function roundsLabel(bestOf) {
    if (bestOf === 0) return 'Free play';
    if (bestOf === 1) return 'Single game';
    return `Best of ${bestOf}`;
  }

  function rulesSummary(rules) {
    const parts = [];
    if (rules.noDestroy) parts.push('No destroy');
    if (rules.twoDice) parts.push('2 dice');
    if (rules.fillBoth) parts.push('Fill both');
    return parts.join(' · ');
  }

  function gainText(pl) {
    if (!pl) return '';
    let t = `+${pl.gain}`;
    if (pl.destroyed > 0) t += `, −${pl.lost}`;
    return t;
  }

  // =========================================================================
  // 7. SESSION & REPLAY (session.go & viewmodel.go parity)
  // =========================================================================
  function createSession(mode, names, difficulty, bestOf, rules, turnSeconds = 0, useCommit = false) {
    const s = {
      proto: PROTO_VERSION,
      mode,
      difficulty,
      bestOf,
      turnSeconds,
      freePlay: bestOf === 0,
      names: [names[0], names[1]],
      wins: [0, 0],
      round: 1,
      series: 1,
      phase: 'play',
      match: createMatch(rules, 0),
      roundFirst: 0,
      last: null,
      forfeit: null,
      votes: ['', ''],
      log: [],
      history: [],
      roundMoves: [],
      timeouts: [0, 0],
      turnDeadline: 0,
      aiming: false,
      aimCol: 0,
      aimDie: 0,
      hostCommit: '',
      hostSeed: '',
      guestSeed: '',
      connected: [true, mode !== 'online']
    };
    startRound(s, useCommit);
    return s;
  }

  function roundNeeded(bestOf) {
    return bestOf <= 0 ? 0 : Math.floor(bestOf / 2) + 1;
  }

  function seriesWinner(s) {
    if (s.forfeit) return 1 - s.forfeit.loser;
    if (s.bestOf === 1) return matchWinner(s.match);
    if (s.wins[0] > s.wins[1]) return 0;
    if (s.wins[1] > s.wins[0]) return 1;
    return -1;
  }

  function startRound(s, useCommit = false) {
    let first;
    if (useCommit) {
      s.hostSeed = newRandomHex(16);
      s.guestSeed = s.guestSeed || newRandomHex(16);
      s.hostCommit = commitSeed(s.hostSeed);
      first = roundCoin(s.hostSeed, s.guestSeed);
    } else {
      first = Math.random() < 0.5 ? 0 : 1;
      s.hostCommit = '';
      s.hostSeed = '';
    }
    s.roundFirst = first;
    s.match = createMatch(s.match.rules, first);
    s.last = null;
    s.forfeit = null;
    s.votes = ['', ''];
    s.roundMoves = [];
    s.phase = 'play';
    s.aiming = false;

    if (useCommit) {
      s.match.dice = roundRoll(s.hostSeed, s.guestSeed, s.match.moves, s.match.rules.twoDice);
    } else {
      rollForTurn(s.match, () => Math.floor(Math.random() * 6) + 1);
    }
    s.log.push({ kind: 'first', round: s.round, player: first });
    resetTurnDeadline(s);
  }

  function resetTurnDeadline(s) {
    s.turnDeadline = s.phase === 'play' && s.turnSeconds > 0 ? Date.now() + s.turnSeconds * 1000 : 0;
  }

  function finishRound(s) {
    const scores = matchScores(s.match);
    const w = matchWinner(s.match);
    if (w >= 0) s.wins[w]++;
    s.log.push({ kind: 'round', round: s.round, player: w, scores });
    s.history.push({ round: s.round, scores, winner: w });
    const need = roundNeeded(s.bestOf);
    if (need > 0 && (s.wins[0] >= need || s.wins[1] >= need || (s.bestOf === 1 && w >= 0))) {
      s.phase = 'game';
    } else if (s.bestOf === 1 && w < 0) {
      s.phase = 'game';
    } else {
      s.phase = 'round';
    }
    s.turnDeadline = 0;
  }

  function sessionPlace(s, p, mv) {
    if (s.phase !== 'play' || s.match.turn !== p) return false;
    const diceUsed = [s.match.dice[0], s.match.dice[1]];
    const pl = applyPlace(s.match, mv);
    if (!pl) return false;
    s.last = pl;
    s.aiming = false;
    s.roundMoves.push({ p, c: mv.col, d: mv.die, dice: diceUsed });
    s.log.push({
      kind: 'place',
      round: s.round,
      player: p,
      die: pl.die,
      col: pl.col,
      destroyed: pl.destroyed,
      gain: pl.gain,
      lost: pl.lost
    });

    if (matchFinished(s.match)) {
      finishRound(s);
    } else {
      if (s.hostSeed && s.guestSeed) {
        s.match.dice = roundRoll(s.hostSeed, s.guestSeed, s.match.moves, s.match.rules.twoDice);
      } else {
        rollForTurn(s.match, () => Math.floor(Math.random() * 6) + 1);
      }
      resetTurnDeadline(s);
    }
    return true;
  }

  function sessionForfeit(s, loser, reason) {
    if (s.phase === 'game') return;
    s.forfeit = { loser, reason };
    s.phase = 'game';
    s.turnDeadline = 0;
    s.aiming = false;
    s.log.push({ kind: 'forfeit', round: s.round, player: loser, reason });
  }

  function sessionOptions(s) {
    if (s.phase === 'round') return ['continue'];
    if (s.phase === 'game') return ['rematch', 'free'];
    return [];
  }

  function sessionApplyOutcomeChoice(s, choice, useCommit) {
    if (choice === 'continue') {
      s.round++;
      startRound(s, useCommit);
    } else if (choice === 'free') {
      s.freePlay = true;
      s.bestOf = 0;
      s.round++;
      startRound(s, useCommit);
    } else if (choice === 'rematch') {
      s.series++;
      s.round = 1;
      s.wins = [0, 0];
      s.timeouts = [0, 0];
      s.history = [];
      startRound(s, useCommit);
    }
  }

  function buildReplay(s) {
    if (!s || !s.roundMoves || s.roundMoves.length === 0) return [];
    const total = s.roundMoves.length;
    const frames = [];
    const m = createMatch(s.match.rules, s.roundFirst);
    m.dice = [...s.roundMoves[0].dice];
    frames.push({
      match: cloneMatch(m),
      last: null,
      step: 0,
      total,
      desc: `Round ${s.round} start · ${s.names[s.roundFirst]} goes first`
    });
    for (let i = 0; i < total; i++) {
      const rm = s.roundMoves[i];
      const rp = rm.p !== undefined ? rm.p : rm.player;
      const rc = rm.c !== undefined ? rm.c : rm.col;
      const rd = rm.d !== undefined ? rm.d : rm.die;
      m.turn = rp;
      m.dice = [...rm.dice];
      const pl = applyPlace(m, { col: rc, die: rd });
      if (!pl) break;
      m.dice = i + 1 < total ? [...s.roundMoves[i + 1].dice] : [0, 0];
      let desc = `${s.names[rp]} placed ${pl.die} in col ${rc + 1} (+${pl.gain})`;
      if (pl.destroyed > 0) desc += `, −${pl.lost}`;
      frames.push({
        match: cloneMatch(m),
        last: { ...pl },
        step: i + 1,
        total,
        desc
      });
    }
    return frames;
  }

  // =========================================================================
  // 8. ONLINE MULTIPLAYER TRANSPORT (WebSocket over TCP · Protocol v2)
  // =========================================================================
  function createWsGuestController(cfg, onUpdate) {
    const rawAddr = String(cfg.addr || '').trim();
    const displayAddr = normalizeJoinAddr(rawAddr);
    const wsUrl = buildWsUrl(rawAddr);
    const guestToken = newRandomHex(8); // 16 hex chars, matches net.go newToken()
    const roomKey = deriveRoomKey(cfg.roomCode || '');
    const waitingAt = Date.now();

    let ws = null;
    let sess = null;
    let lobbyErr = '';
    let fairState = 'locked';
    let hostGone = false;
    let closed = false;
    let fatal = false;
    let gotChallenge = false;
    let handshakeOK = false;
    let openedThisAttempt = false;
    let wireKey = null;
    let txSeq = 0;
    let rxSeq = 0;
    let gotAt = 0;
    let lostAt = 0;
    let retryAt = 0;
    let lastPing = 0;

    let expectedCommit = '';
    let nextExpectedCommit = '';
    let activeGuestSeed = '';
    let nextGuestSeed = '';
    let obsRoundKey = '';
    let verifiedKey = '';
    let obsRolls = {};

    function closeSocket() {
      if (ws) {
        const old = ws;
        ws = null;
        old.onopen = null;
        old.onmessage = null;
        old.onclose = null;
        old.onerror = null;
        try { old.close(); } catch {}
      }
      wireKey = null;
      gotChallenge = false;
      handshakeOK = false;
      txSeq = 0;
      rxSeq = 0;
    }

    function sendRaw(msgObj) {
      if (!ws || ws.readyState !== WebSocket.OPEN) return false;
      try {
        ws.send(JSON.stringify(msgObj) + '\n');
        return true;
      } catch {
        return false;
      }
    }

    function sendMsg(msgObj) {
      if (closed || !ws || ws.readyState !== WebSocket.OPEN) return;
      if (roomKey && roomKey.length > 0 && !gotChallenge) return;
      const pkg = { ...msgObj };
      if (wireKey && wireKey.length > 0) {
        txSeq++;
        signWireMsg(wireKey, txSeq, pkg);
      }
      if (!sendRaw(pkg)) closeSocket();
    }

    function checkFairPlay(st) {
      if (!st || st.mode !== 'online' || !expectedCommit || fairState === 'failed') return;
      const rk = `${st.series}:${st.round}`;
      if (obsRoundKey !== rk) {
        if (!obsRoundKey) {
          if (st.hostCommit !== expectedCommit || st.guestSeed !== activeGuestSeed) {
            fairState = 'failed';
            return;
          }
        } else {
          if (!nextExpectedCommit || !nextGuestSeed ||
              st.hostCommit !== nextExpectedCommit || st.guestSeed !== nextGuestSeed) {
            fairState = 'failed';
            return;
          }
          expectedCommit = st.hostCommit;
          activeGuestSeed = st.guestSeed;
          nextExpectedCommit = '';
          nextGuestSeed = '';
        }
        obsRoundKey = rk;
        obsRolls = {};
        if (st.hostCommit) fairState = 'locked';
      } else if (st.hostCommit !== expectedCommit || st.guestSeed !== activeGuestSeed) {
        fairState = 'failed';
        return;
      }

      if (st.phase === 'play') {
        if (st.match && !st.match.over) {
          const mvIdx = st.match.moves || 0;
          const d = st.match.dice || [0, 0];
          if (obsRolls[mvIdx]) {
            if (obsRolls[mvIdx][0] !== d[0] || obsRolls[mvIdx][1] !== d[1]) {
              fairState = 'failed';
            }
          } else {
            obsRolls[mvIdx] = [d[0], d[1]];
          }
        }
        return;
      }

      if (st.nextHostCommit) {
        if (!nextExpectedCommit) nextExpectedCommit = st.nextHostCommit;
        else if (st.nextHostCommit !== nextExpectedCommit) {
          fairState = 'failed';
          return;
        }
      }
      if (verifiedKey === rk) return;
      if (!verifySeedCommit(st.hostReveal, expectedCommit)) {
        fairState = 'failed';
        return;
      }
      const coin = roundCoin(st.hostReveal, activeGuestSeed);
      const rolls = Array.isArray(st.roundRolls) ? st.roundRolls : [];
      if ((st.roundFirst || 0) !== coin || rolls.length === 0) {
        fairState = 'failed';
        return;
      }
      const twoDice = !!(st.rules && st.rules.twoDice);
      for (let i = 0; i < rolls.length; i++) {
        const expRoll = roundRoll(st.hostReveal, activeGuestSeed, i, twoDice);
        if (rolls[i][0] !== expRoll[0] || rolls[i][1] !== expRoll[1]) {
          fairState = 'failed';
          return;
        }
      }
      for (const k of Object.keys(obsRolls)) {
        const idx = Number(k);
        const obs = obsRolls[k];
        if (idx < 0 || idx >= rolls.length || rolls[idx][0] !== obs[0] || rolls[idx][1] !== obs[1]) {
          fairState = 'failed';
          return;
        }
      }
      fairState = 'verified';
      verifiedKey = rk;
    }

    function normalizeIncomingSession(st) {
      if (!st.log) st.log = [];
      if (!st.history) st.history = [];
      if (!st.votes) st.votes = ['', ''];
      if (!st.wins) st.wins = [0, 0];
      if (!st.timeouts) st.timeouts = [0, 0];
      if (!st.connected) st.connected = [true, true];
      if (!st.roundRolls) st.roundRolls = [];
      if (!st.roundMoves) st.roundMoves = [];
      if (!st.rules) st.rules = {};
      if (!st.match) st.match = createMatch(st.rules, 0);
      if (!st.match.rules) st.match.rules = st.rules;
      st.turnSeconds = st.turnLimit || 0;
      return st;
    }

    function onWireMsg(msg) {
      if (closed || !msg || typeof msg !== 'object') return;
      if (msg.type === 'challenge') {
        if (gotChallenge) return;
        if (msg.version !== PROTO_VERSION) {
          lobbyErr = 'unsupported-protocol';
          fatal = true;
          closeSocket();
          onUpdate();
          return;
        }
        if (!!msg.authReq !== !!(roomKey && roomKey.length > 0)) {
          lobbyErr = 'bad-room-code';
          fatal = true;
          closeSocket();
          onUpdate();
          return;
        }
        if (!msg.nonce || !validHex(msg.nonce, 64) || typeof msg.commit !== 'string' || msg.commit.length !== 64 || !validHex(msg.commit, 64)) {
          lobbyErr = 'invalid-state';
          fatal = true;
          closeSocket();
          onUpdate();
          return;
        }
        wireKey = deriveWireKey(roomKey, msg.nonce);
        gotChallenge = true;
        if (!expectedCommit) {
          expectedCommit = msg.commit;
        } else if (msg.commit !== expectedCommit && (!nextExpectedCommit || msg.commit !== nextExpectedCommit)) {
          fairState = 'failed';
        }
        if (!activeGuestSeed) activeGuestSeed = newRandomHex(32);
        lastPing = Date.now();
        const hello = {
          type: 'hello',
          version: PROTO_VERSION,
          name: cfg.guestName,
          token: guestToken,
          seed: activeGuestSeed
        };
        if (roomKey && roomKey.length > 0) {
          hello.auth = helloAuthMAC(roomKey, msg.nonce, cfg.guestName, guestToken, activeGuestSeed);
        }
        if (!sendRaw(hello)) closeSocket();
        return;
      }

      if (msg.type === 'error') {
        if (handshakeOK && wireKey && !verifyWireMsg(wireKey, rxSeq + 1, msg)) return;
        lobbyErr = msg.text || 'connection-rejected';
        fatal = true;
        closeSocket();
        onUpdate();
        return;
      }

      if (roomKey && roomKey.length > 0 && !gotChallenge) {
        lobbyErr = 'bad-room-code';
        fatal = true;
        closeSocket();
        onUpdate();
        return;
      }

      if (wireKey && wireKey.length > 0) {
        if (!verifyWireMsg(wireKey, rxSeq + 1, msg)) {
          lobbyErr = 'bad-room-code';
          fatal = true;
          closeSocket();
          onUpdate();
          return;
        }
        rxSeq = msg.seq;
      }

      if (msg.type === 'state' && msg.state) {
        handshakeOK = true;
        checkFairPlay(msg.state);
        sess = normalizeIncomingSession(msg.state);
        gotAt = Date.now();
        lostAt = 0;
        lobbyErr = '';
        onUpdate();
      } else if (msg.type === 'aim' && sess && sess.phase === 'play') {
        if (msg.aiming) {
          const c = msg.col || 0;
          const d = msg.die || 0;
          if (c >= 0 && c < GRID_COLS && d >= 0 && d <= 1 && sess.match.dice[d] !== 0) {
            sess.aiming = true;
            sess.aimCol = c;
            sess.aimDie = d;
          }
        } else {
          sess.aiming = false;
        }
        onUpdate();
      } else if (msg.type === 'bye') {
        hostGone = true;
        if (sess) sess.connected[0] = false;
        closeSocket();
        onUpdate();
      }
    }

    function connectAttempt() {
      if (closed) return;
      closeSocket();
      lobbyErr = '';
      fatal = false;
      openedThisAttempt = false;
      if (!displayAddr || !wsUrl) {
        lobbyErr = 'invalid address';
        onUpdate();
        return;
      }
      try {
        ws = new WebSocket(wsUrl);
      } catch (err) {
        lobbyErr = err && err.message ? err.message : 'Could not open WebSocket connection';
        onUpdate();
        return;
      }
      const sock = ws;
      sock.onopen = () => {
        if (closed || ws !== sock) return;
        openedThisAttempt = true;
      };
      sock.onmessage = (ev) => {
        if (closed || ws !== sock) return;
        const chunks = String(ev.data || '').split('\n');
        for (const chunk of chunks) {
          const line = chunk.trim();
          if (!line) continue;
          try {
            onWireMsg(JSON.parse(line));
          } catch { /* ignore malformed frame */ }
        }
      };
      sock.onerror = () => { /* handled cleanly on close */ };
      sock.onclose = () => {
        if (closed || ws !== sock) return;
        closeSocket();
        if (fatal) {
          onUpdate();
          return;
        }
        if (!sess) {
          if (!lobbyErr) {
            lobbyErr = !openedThisAttempt && location.protocol === 'https:' && /^wss:\/\//.test(wsUrl)
              ? `could not reach ${displayAddr} (HTTPS pages require WSS/TLS; use localhost or desktop app)`
              : !openedThisAttempt
                ? `could not reach ${displayAddr}`
                : 'the host closed the connection';
          }
          onUpdate();
        } else if (sess.phase !== 'game' && !hostGone) {
          lostAt = Date.now();
          retryAt = Date.now() + 2000;
          onUpdate();
        } else {
          hostGone = true;
          sess.connected[0] = false;
          onUpdate();
        }
      };
    }

    connectAttempt();

    return {
      mode: 'online',
      seat: () => 1,
      state: () => sess,
      waitingSince: () => waitingAt,
      addr: () => displayAddr,
      roomCode: () => cfg.roomCode || '',
      lobbyError: () => lobbyErr,
      canRetry: () => lobbyErr !== '' && lobbyErr !== 'bad-room-code' && lobbyErr !== 'unsupported-protocol',
      retryConnect: () => connectAttempt(),
      fairness: () => fairState,
      opponentGone: () => hostGone,
      disconnectLeft: (now) => {
        if (!lostAt) return { seat: -1, left: 0 };
        return { seat: 0, left: Math.max(0, DISCONNECT_GRACE_MS - (now - lostAt)) };
      },
      turnLeft: (now) => {
        if (!sess || !sess.turnLeftMs || sess.phase !== 'play' || lostAt) return 0;
        const left = sess.turnLeftMs - (now - gotAt);
        return left > 0 ? left : 1;
      },
      place: (mv) => sendMsg({ type: 'move', col: mv.col, die: mv.die }),
      aim: (col, die) => sendMsg({ type: 'aim', col, die }),
      vote: (v) => {
        if (v === 'menu') return;
        if (sess) sess.votes[1] = v;
        if (!nextGuestSeed) nextGuestSeed = newRandomHex(32);
        sendMsg({ type: 'vote', vote: v, seed: nextGuestSeed });
        onUpdate();
      },
      resign: () => sendMsg({ type: 'resign' }),
      leave: () => {
        closed = true;
        try { sendMsg({ type: 'bye' }); } catch {}
        closeSocket();
      },
      tick: (now) => {
        if (closed) return false;
        if (ws && ws.readyState === WebSocket.OPEN && gotChallenge && now - lastPing >= 5000) {
          lastPing = now;
          sendMsg({ type: 'ping' });
        }
        if (lostAt > 0) {
          if (now - lostAt >= DISCONNECT_GRACE_MS) {
            lostAt = 0;
            hostGone = true;
            if (sess) {
              sess.connected[0] = false;
              sessionForfeit(sess, 0, 'disconnect');
            }
            return true;
          }
          if (!ws && now >= retryAt) {
            retryAt = now + 2000;
            connectAttempt();
          }
        }
        return false;
      }
    };
  }

  function createOfflineController(sess, isAI, onUpdate) {
    let aiKey = '';
    let aiTimer = null;
    return {
      mode: sess.mode,
      seat: () => (isAI ? 0 : -1),
      state: () => sess,
      fairness: () => 'none',
      opponentGone: () => false,
      disconnectLeft: () => ({ seat: -1, left: 0 }),
      turnLeft: () => 0,
      place: (mv) => {
        if (isAI && sess.match.turn !== 0) return;
        if (sessionPlace(sess, sess.match.turn, mv)) onUpdate();
      },
      aim: () => {},
      vote: (v) => {
        if (v === 'menu') return;
        sessionApplyOutcomeChoice(sess, v, false);
        onUpdate();
      },
      resign: () => {
        if (aiTimer) { clearTimeout(aiTimer); aiTimer = null; }
        if (sess.phase === 'play') {
          sessionForfeit(sess, isAI ? 0 : sess.match.turn, 'resignation');
        } else if (sess.phase === 'round' && isAI) {
          sessionForfeit(sess, 0, 'resignation');
        }
        onUpdate();
      },
      leave: () => {
        if (aiTimer) { clearTimeout(aiTimer); aiTimer = null; }
      },
      tick: () => {
        if (!isAI || sess.phase !== 'play' || sess.match.turn !== 1) return false;
        const key = `${sess.series}:${sess.round}:${sess.match.moves}`;
        if (key !== aiKey) {
          aiKey = key;
          if (aiTimer) clearTimeout(aiTimer);
          aiTimer = setTimeout(() => {
            aiTimer = null;
            if (sess.phase === 'play' && sess.match.turn === 1) {
              const mv = chooseAIMove(sess.match, sess.difficulty);
              sessionPlace(sess, 1, mv);
              onUpdate();
            }
          }, AI_DELAY_MS);
        }
        return false;
      }
    };
  }

  // =========================================================================
  // 9. APPLICATION STATE & VIEW CONTROLLER
  // =========================================================================
  const rootEl = document.getElementById('app');
  const save = loadSave();

  const app = {
    view: 'menu',
    menuIndex: 0,
    setupMode: 'ai',
    setIndex: 0,
    joinAddr: save.settings.joinAddr || `localhost:${DEFAULT_PORT}`,
    joinCode: '',
    joinField: 0,
    nameDraft: save.name,
    note: '',
    ctl: null,
    overlay: 'none',
    confirmIndex: 0,
    resultIndex: 0,
    hideResult: false,
    cursor: 1,
    dieSel: 0,
    rollKey: '',
    rollAt: 0,
    effectKey: '',
    effectAt: 0,
    effectLast: null,
    aimPubKey: '',
    recRoundKey: '',
    recSeriesKey: 0,
    recordNote: '',
    soundPhaseKey: '',
    winCacheKey: '',
    winCacheProb: 0.5,
    replayActive: false,
    replayStep: 0,
    replayFrames: [],
    flipP2: false,
    touchArmedCol: -1,
    wasRolling: false,
    wasFx: false,
    pendingDropAnim: false,
    pendingShakeAnim: false,
    installPromptEvent: null,
    swPendingUpdate: false,
    tickHandle: null
  };

  function applyThemeAndMode() {
    document.body.classList.remove('theme-dark', 'theme-light', 'theme-contrast', 'theme-parchment');
    document.body.classList.add('theme-' + save.theme);
    document.body.classList.toggle('mode-tui', save.displayMode === 'tui');
    document.body.classList.toggle('mode-gui', save.displayMode !== 'tui');
    const meta = document.getElementById('meta-theme-color');
    if (meta && THEMES[save.theme]) meta.setAttribute('content', THEMES[save.theme].bg);
  }

  function emitSound(id) {
    AudioEngine.play(id, save.settings.sound);
  }

  function triggerHaptic(kind) {
    if (!save.settings.haptics || !navigator.vibrate) return;
    try {
      if (kind === 'aim') navigator.vibrate(8);
      else if (kind === 'place') navigator.vibrate(18);
      else if (kind === 'destroy' || kind === 'win') navigator.vibrate([20, 30, 28]);
    } catch {}
  }

  function stepTheme(step = 1) {
    const idx = THEME_ORDER.indexOf(save.theme);
    save.theme = THEME_ORDER[(idx + step + THEME_ORDER.length) % THEME_ORDER.length];
    applyThemeAndMode();
    persistSave(save);
    emitSound('click');
  }

  function toggleDisplayMode() {
    save.displayMode = save.displayMode === 'gui' ? 'tui' : 'gui';
    applyThemeAndMode();
    persistSave(save);
    emitSound('click');
  }

  function bottomSeat() {
    if (!app.ctl) return 0;
    const seat = app.ctl.seat();
    return seat < 0 ? 0 : seat;
  }

  function isYou(s, p) {
    if (!app.ctl || !s) return false;
    const seat = app.ctl.seat();
    return seat >= 0 && p === seat && s.mode !== 'local';
  }

  function whoName(s, p) {
    return isYou(s, p) ? 'You' : s.names[p];
  }

  function panelName(s, p) {
    if (!s || s.mode !== 'ai') return s.names[p];
    if (p === 0) return s.names[0];
    return `${s.names[1]} (${DIFF_LABEL[s.difficulty] || ''})`;
  }

  function myTurn(s) {
    if (!s || s.phase !== 'play' || !app.ctl) return false;
    const seat = app.ctl.seat();
    return seat < 0 || s.match.turn === seat;
  }

  function isRolling() {
    return Date.now() - app.rollAt < ROLL_DURATION_MS;
  }

  function effectActive() {
    return app.effectAt > 0 && Date.now() - app.effectAt < EFFECT_DURATION_MS;
  }

  function syncSessionEvents() {
    if (!app.ctl) return;
    const s = app.ctl.state();
    if (!s) return;
    if (app.view === 'lobby') {
      app.view = 'play';
      app.overlay = 'none';
    }
    const key = `${s.series}:${s.round}:${s.match.moves + 1}`;
    if (s.last && s.match.moves > 0 && key !== app.effectKey) {
      app.effectKey = key;
      app.effectAt = Date.now();
      app.effectLast = { ...s.last };
      app.pendingDropAnim = true;
      app.pendingShakeAnim = s.last.destroyed > 0 && s.last.lost > 0;
      emitSound(s.last.destroyed > 0 ? 'destroy' : 'place');
      triggerHaptic(s.last.destroyed > 0 ? 'destroy' : 'place');
    }
    if (s.phase !== 'play') {
      if (app.overlay === 'resign' && s.phase === 'game') app.overlay = 'none';
      const pKey = `${s.series}:${s.round}`;
      if (pKey !== app.soundPhaseKey) {
        app.soundPhaseKey = pKey;
        if (s.mode !== 'local') {
          let w = matchWinner(s.match);
          if (s.forfeit) w = 1 - s.forfeit.loser;
          if (w === app.ctl.seat()) {
            emitSound('win');
            triggerHaptic('win');
          } else if (w >= 0) {
            emitSound('lose');
          }
        }
      }
      checkRecord(s);
      return;
    }

    if (app.replayActive) exitReplay();
    app.hideResult = false;
    if (key !== app.rollKey) {
      app.rollKey = key;
      app.rollAt = Date.now();
      app.dieSel = 0;
      app.touchArmedCol = -1;
      app.resultIndex = 0;
      app.recordNote = '';
      app.aimPubKey = '';
      const grid = s.match.grids[s.match.turn];
      for (let i = 0; i < GRID_COLS && openSlot(grid[app.cursor]) < 0; i++) {
        app.cursor = (app.cursor + 1) % GRID_COLS;
      }
      if (s.match.moves === 0 && myTurn(s)) emitSound('turn');
    }
    syncAim(s);
  }

  function syncAim(s) {
    if (!app.ctl || !s || s.mode !== 'online' || s.phase !== 'play') return;
    if (!myTurn(s) || isRolling() || app.overlay !== 'none') return;
    const k = `${app.cursor}:${app.dieSel}`;
    if (k === app.aimPubKey) return;
    app.aimPubKey = k;
    app.ctl.aim(app.cursor, app.dieSel);
  }

  function checkRecord(s) {
    if (!s || s.phase === 'play' || !app.ctl) return;
    const rKey = `${s.series}:${s.round}`;
    if (!s.forfeit && rKey !== app.recRoundKey) {
      app.recRoundKey = rKey;
      const rec = getRecord(save, recordKey(s.mode, s.difficulty));
      rec.games++;
      const sc = matchScores(s.match);
      const w = matchWinner(s.match);
      app.recordNote = '';
      if (s.mode === 'local') {
        if (w < 0) rec.draws++;
        for (let p = 0; p < 2; p++) {
          if (sc[p] > rec.best) rec.best = sc[p];
          addHighScore(save, { name: s.names[p], score: sc[p], opponent: s.names[1 - p], against: sc[1 - p], mode: s.mode });
        }
      } else {
        const me = app.ctl.seat();
        if (w < 0) rec.draws++;
        else if (w === me) rec.wins++;
        else rec.losses++;
        if (sc[me] > rec.best) rec.best = sc[me];
        const rank = addHighScore(save, { name: s.names[me], score: sc[me], opponent: s.names[1 - me], against: sc[1 - me], mode: s.mode });
        if (rank > 0) app.recordNote = `New high score! #${rank} on your Top 10`;
      }
      persistSave(save);
    }
    if (s.phase === 'game' && s.series !== app.recSeriesKey) {
      app.recSeriesKey = s.series;
      if (s.forfeit && rKey !== app.recRoundKey) {
        app.recRoundKey = rKey;
        const rec = getRecord(save, recordKey(s.mode, s.difficulty));
        rec.games++;
        if (s.mode !== 'local') {
          if (s.forfeit.loser === app.ctl.seat()) rec.losses++;
          else rec.wins++;
        }
      }
      if (s.mode !== 'local' && s.bestOf >= 3 && !s.freePlay) {
        const rec = getRecord(save, recordKey(s.mode, s.difficulty));
        const w = seriesWinner(s);
        if (w === app.ctl.seat()) rec.seriesWins++;
        else if (w >= 0) rec.seriesLosses++;
      }
      persistSave(save);
    }
  }

  function winProbCached(s) {
    const seat = bottomSeat();
    const k = `${s.series}:${s.round}:${s.match.moves}:${s.match.turn}:${s.match.dice[0]},${s.match.dice[1]}:${seat}`;
    if (k !== app.winCacheKey) {
      app.winCacheKey = k;
      app.winCacheProb = estimateWinProb(s.match, seat, 96);
    }
    const pct = Math.max(0, Math.min(100, Math.round(app.winCacheProb * 100)));
    const name = isYou(s, seat) ? 'You' : s.names[seat];
    return { pct, summary: `${name} ${pct}%` };
  }

  function enterReplay(s) {
    const frames = buildReplay(s);
    if (frames.length === 0) return;
    app.replayFrames = frames;
    app.replayStep = frames.length - 1;
    app.replayActive = true;
    app.overlay = 'none';
    emitSound('click');
    render();
  }

  function exitReplay() {
    app.replayActive = false;
    app.replayStep = 0;
    app.replayFrames = [];
  }

  function beginGame(ctl) {
    app.ctl = ctl;
    app.overlay = 'none';
    app.resultIndex = 0;
    app.cursor = 1;
    app.dieSel = 0;
    app.rollKey = '';
    app.effectKey = '';
    app.effectAt = 0;
    app.aimPubKey = '';
    app.recRoundKey = '';
    app.recSeriesKey = 0;
    app.recordNote = '';
    app.soundPhaseKey = '';
    app.winCacheKey = '';
    app.hideResult = false;
    app.flipP2 = false;
    app.touchArmedCol = -1;
    app.wasRolling = false;
    app.wasFx = false;
    app.pendingDropAnim = false;
    app.pendingShakeAnim = false;
    exitReplay();
    app.view = ctl.state() ? 'play' : 'lobby';
    syncSessionEvents();
    startClockLoop();
    render();
  }

  function leaveGame() {
    if (app.ctl) {
      app.ctl.leave();
      app.ctl = null;
    }
    app.overlay = 'none';
    exitReplay();
    stopClockLoop();
    app.view = 'menu';
    if (app.swPendingUpdate) applyPendingUpdateNow();
    render();
  }

  function startClockLoop() {
    stopClockLoop();
    app.tickHandle = setInterval(() => {
      if (!app.ctl) return;
      const now = Date.now();
      const changed = app.ctl.tick(now);
      const rollingNow = isRolling();
      const fxNow = effectActive();
      const s = app.ctl.state();
      const hasTimer = s && s.phase === 'play' && (s.turnDeadline > 0 || s.turnLeftMs > 0);
      const inLobby = app.view === 'lobby';
      const thinking = s && s.phase === 'play' && !myTurn(s) && !rollingNow;
      if (changed) {
        syncSessionEvents();
        render();
      } else if (rollingNow || fxNow || app.wasRolling || app.wasFx || hasTimer || inLobby || thinking) {
        if (app.view === 'lobby') updateLobbyClockOnly();
        else if (app.view === 'play') updatePlayClockInPlace(now);
      }
    }, 140);
  }

  function updatePlayClockInPlace(now) {
    const s = app.ctl && app.ctl.state();
    if (!s || app.replayActive) return;
    const rollingNow = isRolling();
    const fxNow = effectActive();
    const rollJustEnded = app.wasRolling && !rollingNow;
    const fxJustEnded = app.wasFx && !fxNow;
    app.wasRolling = rollingNow;
    app.wasFx = fxNow;
    if (rollJustEnded || fxJustEnded) {
      render();
      return;
    }

    const timerEl = document.getElementById('turn-timer-val');
    if (timerEl && s.phase === 'play') {
      const tLeft = app.ctl.turnLeft(now);
      if (tLeft > 0) {
        const secs = Math.ceil(tLeft / 1000);
        timerEl.textContent = `Time left: ${secs}s`;
        timerEl.classList.toggle('urgent', secs <= 5);
      }
    }

    const thinkEl = document.getElementById('thinking-text');
    if (thinkEl && s.phase === 'play' && !myTurn(s) && !rollingNow) {
      const dots = '.'.repeat(1 + (Math.floor(now / 350) % 3));
      thinkEl.textContent = `Thinking${dots}`;
    }

    if (rollingNow) {
      const wraps = document.querySelectorAll('#tray-dice .tray-die-wrap');
      wraps.forEach((wrap, i) => {
        const rollFace = ((Math.floor(now / 70) + i * 5) % 6) + 1;
        const dieEl = wrap.firstElementChild;
        if (dieEl) dieEl.innerHTML = renderDieInnerHTML(rollFace, '');
      });
    }
  }

  function stopClockLoop() {
    if (app.tickHandle) {
      clearInterval(app.tickHandle);
      app.tickHandle = null;
    }
  }

  function updateLobbyClockOnly() {
    const el = document.getElementById('lobby-wait-time');
    if (el && app.ctl && app.ctl.waitingSince) {
      el.textContent = formatWait(Date.now() - app.ctl.waitingSince());
    }
  }

  function formatWait(ms) {
    const sec = Math.max(0, Math.floor(ms / 1000));
    const m = Math.floor(sec / 60);
    const s = sec % 60;
    return `${m}:${String(s).padStart(2, '0')}`;
  }

  // =========================================================================
  // 10. UI HELPERS & RENDERING
  // =========================================================================
  function escHtml(str) {
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  function renderDieInnerHTML(v, mark) {
    const isTUI = save.displayMode === 'tui';
    const pips = PIP_GRID[v] || [];
    let inner = '';
    for (let i = 0; i < 9; i++) {
      if (pips[i]) {
        inner += isTUI ? '<span>●</span>' : '<span class="die-pip"></span>';
      } else {
        inner += '<span></span>';
      }
    }
    if (mark) {
      const sym = mark === 'pair' ? '=' : mark === 'triple' ? '#' : isTUI ? 'x' : '×';
      inner += `<span class="${isTUI ? 'tui-mark' : 'die-cb-mark'}">${sym}</span>`;
    }
    return inner;
  }

  function renderDieHTML(v, variant, mark, isLast, isJustPlaced = false) {
    const isTUI = save.displayMode === 'tui';
    const cls = [isTUI ? 'tui-die' : 'die-face'];
    if (variant) cls.push(variant);
    if (isLast) cls.push('last-placed');
    if (isJustPlaced && !isTUI) cls.push('just-placed');
    return `<div class="${cls.join(' ')}">${renderDieInnerHTML(v, mark)}</div>`;
  }

  function cellVariantAndMark(matches, doom, colorblind) {
    if (doom) return { variant: 'doom', mark: colorblind ? 'doom' : '' };
    if (matches >= 3) return { variant: 'triple', mark: colorblind ? 'triple' : '' };
    if (matches === 2) return { variant: 'pair', mark: colorblind ? 'pair' : '' };
    return { variant: '', mark: '' };
  }

  function logText(s, e) {
    switch (e.kind) {
      case 'first':
        return `Round ${e.round} · ${whoName(s, e.player)} ${isYou(s, e.player) ? 'go' : 'goes'} first.`;
      case 'place': {
        let t = `${whoName(s, e.player)} placed a ${e.die} in column ${e.col + 1} (+${e.gain})`;
        if (e.destroyed > 0) t += `, destroying ${e.destroyed} ${e.destroyed === 1 ? 'die' : 'dice'} (−${e.lost})`;
        return t + '.';
      }
      case 'timeout':
        return `${whoName(s, e.player)} ran out of time (${e.count}/${MAX_TIMEOUTS}).`;
      case 'round': {
        const base = `Round ${e.round} ended ${s.names[0]} ${e.scores[0]} – ${e.scores[1]} ${s.names[1]}.`;
        return e.player < 0 ? base + ' Tie.' : `${base} ${whoName(s, e.player)} won.`;
      }
      case 'forfeit':
        return forfeitText(s, e.player, e.reason);
      default:
        return '';
    }
  }

  function forfeitText(s, loser, reason) {
    const winnerName = s.names[1 - loser];
    const loserName = s.names[loser];
    const youWin = isYou(s, 1 - loser);
    const youLose = isYou(s, loser);
    if (reason === 'disconnect') {
      if (youWin) return 'Your opponent disconnected. You win!';
      if (youLose) return `You disconnected. ${winnerName} wins.`;
      return `${loserName} disconnected. ${winnerName} wins.`;
    }
    if (reason === 'timeout') {
      if (youWin) return 'Your opponent timed out three times. You win!';
      if (youLose) return `You timed out three times. ${winnerName} wins.`;
      return `${loserName} timed out three times. ${winnerName} wins.`;
    }
    if (youWin) return 'Your opponent resigned. You win!';
    if (youLose) return `You resigned. ${winnerName} wins.`;
    return `${loserName} resigned. ${winnerName} wins.`;
  }

  function headerHeadline(s, now) {
    if (app.replayActive) return { text: 'Round replay', color: 'var(--accent)' };
    if (app.ctl && app.ctl.mode === 'online') {
      const disc = app.ctl.disconnectLeft(now);
      if (disc.seat >= 0 && disc.left > 0) {
        return { text: `Waiting for ${s.names[disc.seat]} to reconnect (${Math.ceil(disc.left / 1000)}s)...`, color: 'var(--lose)' };
      }
      if (app.ctl.opponentGone() && s.phase !== 'game') {
        return { text: `${s.names[1 - bottomSeat()]} left`, color: 'var(--lose)' };
      }
    }
    if (s.phase !== 'play') {
      let w = s.phase === 'game' ? seriesWinner(s) : matchWinner(s.match);
      if (s.forfeit) w = 1 - s.forfeit.loser;
      if (w < 0) return { text: "It's a tie!", color: 'var(--accent)' };
      if (isYou(s, w)) return { text: s.phase === 'game' ? 'You won!' : 'You won the round!', color: 'var(--win)' };
      const c = app.ctl && app.ctl.seat() >= 0 && s.mode !== 'local' ? 'var(--lose)' : 'var(--accent)';
      return { text: `${s.names[w]} wins!`, color: c };
    }
    if (isRolling()) return { text: `${whoName(s, s.match.turn)} ${isYou(s, s.match.turn) ? 'roll' : 'rolls'}...`, color: 'var(--dim)' };
    return { text: isYou(s, s.match.turn) ? 'Your turn' : `${s.names[s.match.turn]}'s turn`, color: 'var(--accent)' };
  }

  function resultHeadline(s) {
    if (s.forfeit) {
      const c = isYou(s, 1 - s.forfeit.loser) ? 'var(--win)' : isYou(s, s.forfeit.loser) ? 'var(--lose)' : 'var(--accent)';
      return { text: forfeitText(s, s.forfeit.loser, s.forfeit.reason), color: c };
    }
    const sc = matchScores(s.match);
    const scope = s.phase === 'game' ? 'game' : 'round';
    const w = s.phase === 'game' ? seriesWinner(s) : matchWinner(s.match);
    if (w < 0) {
      return { text: scope === 'game' ? 'This is a tie! Nobody wins!' : 'This is a tie! Nobody wins this round!', color: 'var(--accent)' };
    }
    if (isYou(s, w)) return { text: `You won the ${scope} with ${sc[w]} points!`, color: 'var(--win)' };
    const c = app.ctl && app.ctl.seat() >= 0 && s.mode !== 'local' ? 'var(--lose)' : 'var(--accent)';
    return { text: `${s.names[w]} won the ${scope} with ${sc[w]} points!`, color: c };
  }

  function render() {
    applyThemeAndMode();
    if (app.view === 'menu') renderMenu();
    else if (app.view === 'settings') renderSettings();
    else if (app.view === 'name') renderNamePrompt();
    else if (app.view === 'join') renderJoin();
    else if (app.view === 'lobby') renderLobby();
    else if (app.view === 'guide') renderGuide();
    else if (app.view === 'stats') renderStats();
    else if (app.view === 'play') renderPlay();
  }

  function renderMenu() {
    const items = [
      { id: 'ai', label: 'Play vs AI' },
      { id: 'local', label: '2 Players (Same Device)' },
      { id: 'join', label: 'Join Game' },
      { id: 'guide', label: 'How to Play' },
      { id: 'stats', label: 'Stats & High Scores' },
      { id: 'name', label: `Change Name (${save.name})` },
      { id: 'options', label: 'Options' }
    ];
    const installBtn = !isPwaStandalone() && app.installPromptEvent
      ? `<button type="button" class="kb-btn sm pill on" data-action="install">Install PWA</button>`
      : '';
    const updatePill = app.swPendingUpdate
      ? `<button type="button" class="kb-btn sm pill on" data-action="apply-update">Update Available · Reload</button>`
      : '';

    const quickPills = `${installBtn}${updatePill}`;

    rootEl.innerHTML = `
      <div class="screen">
        ${save.displayMode === 'gui'
          ? `<div class="menu-banner-wrap"><img src="./banner-web.png" alt="Knucklebones" class="menu-banner"></div>`
          : `<h1 class="title-lettered">K N U C K L E B O N E S</h1>
             <div style="display:flex;gap:6px;margin:4px 0;">${[1, 2, 3, 4, 5, 6].map((v) => renderDieHTML(v, '', '', false)).join('')}</div>`}
        <p class="subtitle">The dice game from Cult of the Lamb</p>
        <div class="menu-list">
          ${items.map((it, idx) => `<button type="button" class="kb-btn ${idx === app.menuIndex ? 'selected' : ''}" data-menu="${it.id}">${escHtml(it.label)}</button>`).join('')}
        </div>
        ${quickPills ? `<div class="menu-quickbar">${quickPills}</div>` : ''}
        <div class="credits">
          <p>Original game by Massive Monster</p>
          <p>Fan-made Knucklebones PWA Edition</p>
        </div>
      </div>
      <div class="footer-hint">Click or ↑↓ to select · Enter · T theme · M GUI/TUI</div>
    `;
  }

  function settingRowsForMode() {
    if (app.setupMode === 'ai') return ['difficulty', 'rounds', 'destroy', 'dice', 'end', 'start'];
    if (app.setupMode === 'local') return ['player2', 'rounds', 'destroy', 'dice', 'end', 'start'];
    return ['theme', 'display', 'sound', 'haptics', 'quickTap', 'colorblind', 'winMeter', 'done'];
  }

  function settingDisplay(row) {
    const st = save.settings;
    switch (row) {
      case 'difficulty': return ['AI difficulty', `${DIFF_LABEL[st.difficulty]} · ${AI_NAME[st.difficulty]}`];
      case 'player2': return ['Player 2 name', st.player2];
      case 'rounds': return ['Number of rounds', roundsLabel(st.bestOf)];
      case 'destroy': return ['Destroy matching dice', st.rules.noDestroy ? 'Off' : 'On'];
      case 'dice': return ['Dice per turn', st.rules.twoDice ? 'Two (place one)' : 'One'];
      case 'end': return ['Game ends when', st.rules.fillBoth ? 'Both boards are full' : 'One board is full'];
      case 'theme': return ['Theme', THEMES[save.theme].name];
      case 'display': return ['Display mode', save.displayMode === 'tui' ? 'TUI (Terminal)' : 'GUI (Graphical)'];
      case 'sound': return ['Sound effects', st.sound ? 'On' : 'Off'];
      case 'haptics': return ['Vibration haptics', st.haptics ? 'On' : 'Off'];
      case 'quickTap': return ['Touch tap mode', st.quickTap ? 'Quick tap (1-tap)' : 'Preview first (2-tap)'];
      case 'colorblind': return ['Colorblind marks', st.colorblind ? 'On' : 'Off'];
      case 'winMeter': return ['Win probability', st.winMeter ? 'On' : 'Off'];
      case 'done': return ['Done', ''];
      default: return ['Start game', ''];
    }
  }

  function renderSettings() {
    const titles = { ai: 'Play vs AI', local: '2 Players (Same Device)', options: 'Options' };
    const subtitle = app.setupMode === 'options' ? 'Appearance & audio' : 'Game settings';
    const rows = settingRowsForMode();
    if (app.setIndex >= rows.length) app.setIndex = rows.length - 1;

    const hint = app.setupMode === 'ai' ? DIFF_HINT[save.settings.difficulty] : '';

    const showInstallBtn = app.setupMode === 'options' && !isPwaStandalone() && !!app.installPromptEvent;
    const showIosHint = app.setupMode === 'options' && !isPwaStandalone() && !app.installPromptEvent && isIosDevice();
    const pwaInstallHtml = showInstallBtn
      ? `<div class="pwa-install-box"><button type="button" class="kb-btn sm pill on" data-action="install">Install Knucklebones</button></div>`
      : showIosHint
        ? `<p class="ios-install-hint" aria-live="polite">To install on iPhone/iPad: open in Safari, tap Share, then Add to Home Screen.</p>`
        : '';

    rootEl.innerHTML = `
      <div class="screen screen-top">
        <p class="small-title">K N U C K L E B O N E S</p>
        <h2 class="section-heading">${escHtml(titles[app.setupMode])}</h2>
        <p class="subtitle">${escHtml(subtitle)}</p>
        <div class="settings-list">
          ${rows.map((row, idx) => {
            const sel = idx === app.setIndex;
            const [label, val] = settingDisplay(row);
            if (row === 'start' || row === 'done') {
              return `${app.setupMode === 'options' ? pwaInstallHtml : ''}<div class="btn-row"><button type="button" class="kb-btn ${sel ? 'selected' : 'primary'}" data-set-action="${row}">${escHtml(label)}</button></div>`;
            }
            if (row === 'player2') {
              return `
                <div class="setting-row ${sel ? 'selected' : ''}" data-set-idx="${idx}">
                  <span class="setting-label">${escHtml(label)}</span>
                  <input id="input-player2" class="kb-input" type="text" maxlength="16" value="${escHtml(save.settings.player2)}" placeholder="Player 2">
                </div>`;
            }
            return `
              <div class="setting-row ${sel ? 'selected' : ''}" data-set-idx="${idx}">
                <span class="setting-label">${escHtml(label)}</span>
                <div class="setting-ctrl">
                  <button type="button" class="arrow-btn" data-set-step="-1" data-set-idx="${idx}" aria-label="Previous">◀</button>
                  <span class="setting-val">${escHtml(val)}</span>
                  <button type="button" class="arrow-btn" data-set-step="1" data-set-idx="${idx}" aria-label="Next">▶</button>
                </div>
              </div>`;
          }).join('')}
        </div>
        ${hint ? `<p class="note-hint">${escHtml(hint)}</p>` : ''}
        ${app.note ? `<p class="error-text">${escHtml(app.note)}</p>` : ''}
        <div class="btn-row"><button type="button" class="kb-btn" data-action="back-menu">Back</button></div>
      </div>
      <div class="footer-hint">Click a setting or ↑↓ select · ←→ change · Enter · Esc back</div>
    `;
  }

  function renderNamePrompt() {
    rootEl.innerHTML = `
      <div class="screen">
        <img src="./die.png" alt="" class="die-emblem">
        <p class="small-title">K N U C K L E B O N E S</p>
        <div class="kb-card" style="max-width:480px;">
          <h2 class="kb-card-title">Your name</h2>
          <div class="form-field active">
            <label for="input-player-name">Shown to your opponent and on your Top 10.</label>
            <input id="input-player-name" class="kb-input" type="text" maxlength="16" value="${escHtml(app.nameDraft)}" placeholder="Lamb">
          </div>
          ${app.note ? `<p class="error-text">${escHtml(app.note)}</p>` : ''}
          <div class="btn-row">
            <button type="button" class="kb-btn" data-action="back-menu">Back</button>
            <button type="button" class="kb-btn selected" data-action="save-name">Save</button>
          </div>
        </div>
      </div>
      <div class="footer-hint">Type · Enter confirm · Esc back</div>
    `;
    const inp = document.getElementById('input-player-name');
    if (inp) { inp.focus(); inp.setSelectionRange(inp.value.length, inp.value.length); }
  }

  function renderJoin() {
    rootEl.innerHTML = `
      <div class="screen">
        <img src="./die.png" alt="" class="die-emblem">
        <p class="small-title">K N U C K L E B O N E S</p>
        <div class="kb-card" style="max-width:520px;">
          <h2 class="kb-card-title">Join Game</h2>
          <div class="form-field ${app.joinField === 0 ? 'active' : ''}">
            <label for="input-join-addr">Host address (host:port)</label>
            <input id="input-join-addr" class="kb-input" type="text" maxlength="80" value="${escHtml(app.joinAddr)}" placeholder="localhost:8647" autocomplete="off" spellcheck="false">
          </div>
          <div class="form-field ${app.joinField === 1 ? 'active' : ''}">
            <label for="input-join-code">Room code (optional, leave blank if open)</label>
            <input id="input-join-code" class="kb-input" type="text" maxlength="12" value="${escHtml(app.joinCode)}" placeholder="(optional)" autocomplete="off" spellcheck="false">
          </div>
          ${app.note ? `<p class="error-text">${escHtml(app.note)}</p>` : ''}
          ${location.protocol === 'https:' ? '<p class="note-hint" style="margin-top:8px;">From HTTPS, use <strong>localhost:8647</strong> to join a host on this computer (remote LAN IPs require WSS/TLS or the desktop app).</p>' : ''}
          <div class="btn-row">
            <button type="button" class="kb-btn" data-action="back-menu">Back</button>
            <button type="button" class="kb-btn selected" data-action="submit-join">Join</button>
          </div>
        </div>
      </div>
      <div class="footer-hint">Tab / ↑↓ switch field · Enter join · Esc back</div>
    `;
    const targetId = app.joinField === 1 ? 'input-join-code' : 'input-join-addr';
    const inp = document.getElementById(targetId);
    if (inp) {
      inp.focus();
      try { inp.setSelectionRange(inp.value.length, inp.value.length); } catch {}
    }
  }

  function renderLobby() {
    const ctl = app.ctl;
    if (!ctl) return;
    const wait = formatWait(Date.now() - ctl.waitingSince());
    const err = ctl.lobbyError();
    if (err) {
      const errMap = {
        'room-full': 'That game already has two players.',
        'unsupported-protocol': 'This game version is no longer compatible with the host.',
        'bad-room-code': 'Incorrect room code.',
        'invalid-state': 'The host sent invalid game data.',
        'connection-rejected': 'The host rejected the connection.'
      };
      const errText = errMap[err] || `Could not connect: ${err}`;
      const canRetry = ctl.canRetry();
      rootEl.innerHTML = `
        <div class="screen">
          <img src="./die.png" alt="" class="die-emblem">
          <p class="small-title">K N U C K L E B O N E S</p>
          <div class="kb-card">
            <h2 class="kb-card-title">Join Game</h2>
            <p class="error-text" style="font-size:1.05rem;">${escHtml(errText)}</p>
            <p class="note-hint" style="margin-top:10px;">${escHtml(ctl.addr())}</p>
            <div class="btn-row">
              <button type="button" class="kb-btn ${!canRetry ? 'selected' : ''}" data-action="back-join">Back</button>
              ${canRetry ? `<button type="button" class="kb-btn selected" data-action="retry-join">Retry</button>` : ''}
            </div>
          </div>
        </div>
        <div class="footer-hint">${canRetry ? 'Enter retry · ' : ''}Esc back</div>
      `;
      return;
    }

    rootEl.innerHTML = `
      <div class="screen">
        <img src="./die.png" alt="" class="die-emblem">
        <p class="small-title">K N U C K L E B O N E S</p>
        <div class="kb-card">
          <h2 class="kb-card-title">Join Game</h2>
          <p style="text-align:center;font-size:1.1rem;margin:18px 0 8px;">Waiting for game to start...</p>
          <p class="invite-code" style="text-align:center;font-size:1.25rem;">${escHtml(ctl.addr())}</p>
          <p class="note-hint">Waiting time: <span id="lobby-wait-time">${wait}</span></p>
          <div class="btn-row"><button type="button" class="kb-btn" data-action="back-join">Cancel</button></div>
        </div>
      </div>
      <div class="footer-hint">Esc cancel</div>
    `;
  }

  function renderGuide() {
    rootEl.innerHTML = `
      <div class="screen screen-top">
        <p class="small-title">K N U C K L E B O N E S</p>
        <div class="guide-card">
          ${GUIDE_CONTENT.map((item) => {
            if (item.kind === 'h') return `<h3 class="guide-h">${escHtml(item.text)}</h3>`;
            if (item.kind === 'table') return `<pre class="guide-table">${escHtml(item.text)}</pre>`;
            return `<p class="guide-p">${escHtml(item.text)}</p>`;
          }).join('')}
        </div>
        <div class="btn-row"><button type="button" class="kb-btn" data-action="back-menu">Back</button></div>
      </div>
      <div class="footer-hint">Scroll to read · Esc back</div>
    `;
  }

  function renderStats() {
    const rowsHtml = RECORD_KEYS.map((k) => {
      const r = save.records[k] || { games: 0, wins: 0, losses: 0, draws: 0, best: 0, seriesWins: 0, seriesLosses: 0 };
      const empty = r.games === 0;
      const wr = r.games > 0 ? `${Math.round((r.wins / r.games) * 100)}%` : '–';
      const wl = k === 'local' ? `${r.wins}–${r.losses}–${r.draws}` : `${r.wins}–${r.losses}–${r.draws}`;
      const series = k === 'local' ? '–' : `${r.seriesWins}–${r.seriesLosses}`;
      return `<tr class="${empty ? 'empty-row' : ''}"><td>${escHtml(RECORD_LABELS[k])}</td><td>${r.games}</td><td>${wl}</td><td>${wr}</td><td>${r.best}</td><td>${series}</td></tr>`;
    }).join('');

    const hsHtml = save.highScores.length === 0
      ? `<p class="note-hint" style="text-align:left;">Finish a game and your score is added here.</p>`
      : `<div class="stats-table-scroll"><table class="stats-table"><thead><tr><th>#</th><th style="text-align:left">Player</th><th>Score</th><th style="text-align:left">Opponent</th><th>Score</th><th>Date</th></tr></thead><tbody>
          ${save.highScores.map((e, i) => `<tr><td>${i + 1}.</td><td style="text-align:left"><strong>${escHtml(e.name)}</strong></td><td><strong>${e.score}</strong></td><td style="text-align:left">vs ${escHtml(e.opponent)}</td><td>${e.against}</td><td style="color:var(--dim)">${escHtml(e.date)}</td></tr>`).join('')}
        </tbody></table></div>`;

    rootEl.innerHTML = `
      <div class="screen screen-top">
        <p class="small-title">K N U C K L E B O N E S</p>
        <h2 class="section-heading">Stats</h2>
        <div class="stats-wrap">
          <div class="kb-card" style="max-width:none;">
            <div class="stats-table-scroll">
              <table class="stats-table">
                <thead><tr><th>Mode</th><th>Games</th><th>W–L–D</th><th>Win %</th><th>Best</th><th>Series</th></tr></thead>
                <tbody>${rowsHtml}</tbody>
              </table>
            </div>
            <h3 style="color:var(--accent);margin:18px 0 8px;">Top 10 final scores</h3>
            ${hsHtml}
            ${app.note ? `<p class="note-hint" style="color:var(--win);margin-top:8px;">${escHtml(app.note)}</p>` : ''}
            <div class="btn-row" style="margin-top:16px;">
              <button type="button" class="kb-btn sm" data-action="export-save">Export knucklebones.json</button>
              <button type="button" class="kb-btn sm" data-action="import-save">Import knucklebones.json</button>
              <button type="button" class="kb-btn sm" data-action="reset-stats">Reset Stats</button>
              <input id="import-save-file" type="file" accept=".json,application/json" class="hidden">
            </div>
          </div>
          <div class="btn-row"><button type="button" class="kb-btn" data-action="back-menu">Back</button></div>
        </div>
      </div>
      <div class="footer-hint">Esc back</div>
    `;
  }

  function renderPlay() {
    const s = app.ctl && app.ctl.state();
    if (!s) return;
    const now = Date.now();
    const bottom = bottomSeat();
    const topP = 1 - bottom;

    let m = s.match;
    let lastPl = s.last;
    let replayNote = '';
    if (app.replayActive && app.replayFrames.length > 0) {
      const rf = app.replayFrames[Math.max(0, Math.min(app.replayStep, app.replayFrames.length - 1))];
      m = rf.match;
      lastPl = rf.last;
      replayNote = `REPLAY ${rf.step}/${rf.total} · ${rf.desc}`;
    }

    const mover = m.turn;
    const rolling = !app.replayActive && isRolling();
    const aimingLocal = !app.replayActive && s.phase === 'play' && !rolling && app.overlay === 'none' && myTurn(s);
    const aimingRemote = !app.replayActive && s.phase === 'play' && !rolling && app.overlay === 'none' && !myTurn(s) && s.mode === 'online' && s.aiming;
    const aiming = aimingLocal || aimingRemote;
    const mv = aimingLocal ? { col: app.cursor, die: app.dieSel } : aimingRemote ? { col: s.aimCol, die: s.aimDie } : null;

    let preview = null;
    let previewPl = null;
    if (aiming && mv && legalMove(m, mv)) {
      const res = previewPlace(m, mv);
      preview = res.next;
      previewPl = res.pl;
    }

    const head = headerHeadline(s, now);
    const fxProg = !app.replayActive && effectActive() ? (now - app.effectAt) / EFFECT_DURATION_MS : -1;
    const playDrop = app.pendingDropAnim;
    const playShake = app.pendingShakeAnim;
    app.pendingDropAnim = false;
    app.pendingShakeAnim = false;
    app.wasRolling = rolling;
    app.wasFx = fxProg >= 0;

    let roundWinner = -1;
    if (!app.replayActive && s.phase !== 'play') {
      roundWinner = s.phase === 'game' ? seriesWinner(s) : matchWinner(s.match);
      if (s.forfeit) roundWinner = 1 - s.forfeit.loser;
    }

    function buildGridHTML(pi, isTop) {
      const grid = preview && pi === mover ? preview.grids[pi] : m.grids[pi];
      const baseGrid = m.grids[pi];
      const isTurn = !app.replayActive && s.phase === 'play' && mover === pi;
      const isWinnerBoard = roundWinner >= 0 && roundWinner === pi;

      let colsHtml = '';
      for (let c = 0; c < GRID_COLS; c++) {
        const colAimed = aiming && mv && mv.col === c && mover === pi;
        const isFullCol = openSlot(baseGrid[c]) < 0;
        const fxGain = fxProg >= 0 && app.effectLast && app.effectLast.player === pi && app.effectLast.col === c && app.effectLast.gain > 0;
        const fxLoss = fxProg >= 0 && app.effectLast && app.effectLast.player !== pi && app.effectLast.col === c && app.effectLast.destroyed > 0 && app.effectLast.lost > 0;

        let cellsHtml = '';
        for (let row = 0; row < GRID_ROWS; row++) {
          const k = isTop ? (GRID_ROWS - 1 - row) : row;
          const v = grid[c][k];
          const isGhost = preview && pi === mover && previewPl && c === previewPl.col && k === previewPl.slot;
          const isDoom = preview && pi !== mover && previewPl && !m.rules.noDestroy && c === previewPl.col && v === previewPl.die && v > 0;
          const isLast = !isGhost && lastPl && lastPl.player === pi && (s.phase === 'play' || app.replayActive) && lastPl.col === c && lastPl.slot === k && v === lastPl.die;
          const isJustPlaced = isLast && playDrop && !app.replayActive;

          if (isGhost) {
            cellsHtml += renderDieHTML(previewPl.die, 'ghost', '', false, false);
          } else if (v === 0) {
            cellsHtml += '<div class="die-slot"></div>';
          } else if (isDoom) {
            const { variant, mark } = cellVariantAndMark(0, true, save.settings.colorblind);
            cellsHtml += renderDieHTML(v, variant, mark, false, false);
          } else {
            const matches = matchesInCol(grid[c], v);
            const { variant, mark } = cellVariantAndMark(matches, false, save.settings.colorblind);
            cellsHtml += renderDieHTML(v, variant, mark, isLast, isJustPlaced);
          }
        }
        const badge = fxGain
          ? `<div class="impact-badge">+${app.effectLast.gain}</div>`
          : fxLoss
            ? `<div class="impact-badge loss">−${app.effectLast.lost}</div>`
            : '';

        const colCls = ['grid-col', colAimed ? 'aimed' : '', colAimed && isFullCol ? 'full' : '', fxLoss ? 'impact-loss' : '', fxLoss && playShake ? 'col-shake' : ''].filter(Boolean).join(' ');
        colsHtml += `<div class="${colCls}" data-col="${c}">${cellsHtml}${badge}</div>`;
      }

      const scoresHtml = [0, 1, 2].map((c) => {
        const curSc = columnScore(baseGrid[c]);
        if (preview) {
          const nextSc = columnScore(preview.grids[pi][c]);
          if (nextSc !== curSc) {
            const color = pi === mover ? 'var(--win)' : 'var(--lose)';
            return `<div style="color:${color}">${curSc}→${nextSc}</div>`;
          }
        }
        return `<div>${curSc}</div>`;
      }).join('');

      const boxCls = ['board-box', isTop ? 'grid-top' : 'grid-bot', isTurn ? 'active-board' : '', isWinnerBoard ? 'winner-board' : ''].filter(Boolean).join(' ');
      return {
        board: `<div class="${boxCls}"><div class="grid-3x3">${colsHtml}</div></div>`,
        scores: `<div class="col-scores-row">${scoresHtml}</div>`
      };
    }

    function buildPlayerPanel(pi, isTop) {
      const turn = !app.replayActive && s.phase === 'play' && mover === pi;
      const total = gridScore(m.grids[pi]);
      const winsStr = s.bestOf !== 1 || s.freePlay ? `Rounds won: ${s.wins[pi]}` : '';
      const toStr = (s.turnSeconds > 0 || s.turnLimit > 0) && s.timeouts[pi] > 0 ? `<div class="player-timeouts">Timeouts: ${s.timeouts[pi]} / ${MAX_TIMEOUTS}</div>` : '';

      let trayHtml = '';
      if (turn) {
        const selDie = !myTurn(s) && s.aiming ? s.aimDie : app.dieSel;
        const diceEls = [0, 1].map((i) => {
          const v = m.dice[i];
          if (!v) return '';
          if (rolling) {
            const rollFace = ((Math.floor(now / 70) + i * 5) % 6) + 1;
            return `<div class="tray-die-wrap rolling-die">${renderDieHTML(rollFace, '', '', false, false)}</div>`;
          }
          const variant = m.dice[1] !== 0 && i !== selDie ? 'ghost' : '';
          const selectedCls = m.dice[1] !== 0 && i === selDie ? 'selected-die' : '';
          const selectCls = myTurn(s) && m.dice[1] !== 0 ? 'selectable' : '';
          return `<div class="tray-die-wrap ${selectedCls} ${selectCls}" data-die="${i}">${renderDieHTML(v, variant, '', false, false)}</div>`;
        }).join('');

        let subNote = '';
        if (previewPl) subNote = `<div class="preview-delta">${escHtml(gainText(previewPl))}</div>`;
        else if (!myTurn(s) && !rolling) {
          const dots = '.'.repeat(1 + (Math.floor(now / 350) % 3));
          subNote = `<div class="thinking-text" id="thinking-text">Thinking${dots}</div>`;
        }
        const tLeft = app.ctl.turnLeft(now);
        const timerHtml = tLeft > 0
          ? `<div class="timer-text ${Math.ceil(tLeft / 1000) <= 5 ? 'urgent' : ''}" id="turn-timer-val">Time left: ${Math.ceil(tLeft / 1000)}s</div>`
          : '';
        const metaHtml = (subNote || timerHtml) ? `<div class="tray-meta">${subNote}${timerHtml}</div>` : '';
        trayHtml = `<div class="tray-side"><div class="tray-row" id="tray-dice">${diceEls}</div>${metaHtml}</div>`;
      }

      return `
        <div class="player-panel ${isTop ? 'panel-top' : 'panel-bottom'} ${turn ? 'active-turn' : ''}">
          <div class="player-info">
            <div class="player-name">${escHtml(panelName(s, pi))}</div>
            <div class="player-total"><span class="player-total-label">Total</span><span class="player-total-val">${total}</span></div>
            ${winsStr ? `<div class="player-sub">${escHtml(winsStr)}</div>` : ''}
            ${toStr}
          </div>
          ${trayHtml}
        </div>`;
    }

    const topGrid = buildGridHTML(topP, true);
    const botGrid = buildGridHTML(bottom, false);

    let dividerHtml = '';
    if (replayNote) {
      dividerHtml = `<div class="aim-divider replay-bar">${escHtml(replayNote)}</div>`;
    } else if (aiming && mv) {
      const arrow = mover === topP ? '▲' : '▼';
      const cells = [0, 1, 2].map((c) => {
        if (c !== mv.col) return '<div></div>';
        const full = openSlot(m.grids[mover][c]) < 0;
        return `<div class="aim-cell ${full ? 'full' : ''}">${arrow} ${c + 1} ${arrow}</div>`;
      }).join('');
      dividerHtml = `<div class="aim-divider">${cells}</div>`;
    } else {
      dividerHtml = `<div class="aim-divider"></div>`;
    }

    const probData = save.settings.winMeter && !app.replayActive ? winProbCached(s) : null;
    const probCls = probData ? (probData.pct < 40 ? 'prob-low' : probData.pct > 60 ? 'prob-high' : 'prob-mid') : 'prob-mid';
    const fairTxt = s.mode === 'online' && app.ctl.seat() === 1
      ? app.ctl.fairness() === 'verified'
        ? { t: 'Fair play verified', c: 'var(--win)' }
        : app.ctl.fairness() === 'failed'
          ? { t: '(!) Host dice unverifiable', c: 'var(--lose)' }
          : s.hostCommit ? { t: 'Fair play commit locked', c: 'var(--dim)' } : null
      : null;

    const actions = [];
    if (app.replayActive) {
      actions.push({ id: 'rep-prev', label: '< Prev [←]' }, { id: 'rep-next', label: 'Next > [→]' }, { id: 'rep-done', label: 'Done [Esc]' });
    } else if (s.phase !== 'play') {
      actions.push({ id: 'toggle-result', label: app.hideResult ? 'Result [V]' : 'Board [V]' });
      if (s.roundMoves && s.roundMoves.length > 0) actions.push({ id: 'enter-replay', label: 'Replay [X]' });
      actions.push(
        { id: 'open-log', label: 'Log [L]' },
        { id: 'open-history', label: 'History [H]' },
        { id: 'open-leave', label: 'Leave [Esc]' }
      );
    } else {
      actions.push(
        { id: 'open-log', label: 'Log [L]' },
        { id: 'open-history', label: 'History [H]' },
        { id: 'open-resign', label: 'Resign [R]' },
        { id: 'open-leave', label: 'Leave [Esc]' }
      );
    }

    const showThumbRow = !app.replayActive && s.phase === 'play';
    const thumbBarHtml = showThumbRow
      ? `<div class="thumb-drop-bar ${myTurn(s) ? '' : 'hidden-turn'}">${[0, 1, 2].map((c) => {
          const full = openSlot(m.grids[mover][c]) < 0;
          const armed = !save.settings.quickTap && app.touchArmedCol === c && !full;
          const lbl = full ? `Col ${c + 1} Full` : armed ? `Drop ${c + 1} ✓` : `Col ${c + 1}`;
          return `<button type="button" class="kb-btn sm thumb-drop-btn ${armed ? 'armed' : ''}" data-drop-col="${c}" ${(full || !myTurn(s)) ? 'disabled' : ''}>${lbl}</button>`;
        }).join('')}</div>`
      : '';

    const hasSeries = s.bestOf !== 1 || s.freePlay || s.history.length > 0;
    const hasMobileSideInfo = !!(hasSeries || fairTxt || probData);

    const flipBtnHtml = s.mode === 'local'
      ? `<button type="button" class="kb-btn sm pill ${app.flipP2 ? 'on' : ''}" data-action="toggle-flip-p2">Flip P2</button>`
      : '';

    const updateBtn = app.swPendingUpdate
      ? `<button type="button" class="kb-btn sm pill on" data-action="apply-update">Update Ready</button>`
      : '';

    const touchHint = app.replayActive
      ? 'Swipe board left/right to step · Tap Done to exit'
      : s.phase !== 'play'
        ? 'Tap Board/Result or Replay to review'
        : save.settings.quickTap
          ? (m.rules.twoDice ? 'Tap col to place · Tap die to switch' : 'Tap any column to place')
          : (m.rules.twoDice ? 'Tap col to preview · Tap again to drop · Tap die to switch' : 'Tap col to preview · Tap again to drop');

    const desktopHint = app.replayActive
      ? '←→ step · Home/End jump · Esc done'
      : s.phase !== 'play'
        ? 'Enter select · V board · X replay · L log · H history · Esc leave'
        : m.rules.twoDice
          ? 'Click col or ←→ · 1–3/Enter place · click die or Tab/W · C marks · P prob · S sound · L · H · R · Esc'
          : 'Click col or ←→ · 1–3/Enter place · C marks · P prob · S sound · L log · H history · R resign · Esc leave';

    rootEl.innerHTML = `
      <div class="play-shell ${app.flipP2 && s.mode === 'local' ? 'flip-p2' : ''}">
        <header class="play-header">
          <div class="header-brand">
            <img src="./die.png" alt="">
            <span>KNUCKLEBONES</span>
          </div>
          <div class="header-status" style="color:${head.color}">${escHtml(head.text)}</div>
          <div class="header-meta">
            <span>Round ${s.round} · ${escHtml(roundsLabel(s.bestOf))}</span>
            ${updateBtn}
          </div>
        </header>

        <div class="play-arena">
          <div class="left-panels">
            ${buildPlayerPanel(topP, true)}
            ${buildPlayerPanel(bottom, false)}
          </div>

          <div class="boards-column">
            ${topGrid.board}
            ${topGrid.scores}
            ${dividerHtml}
            ${botGrid.scores}
            ${botGrid.board}
          </div>

          ${thumbBarHtml}

          <aside class="side-panel ${hasMobileSideInfo ? 'has-mobile-info' : ''}">
            ${hasSeries
              ? `<div class="side-series">Series ${s.wins[0]} – ${s.wins[1]}</div>
                 <div class="side-meta side-meta-sub">${escHtml(s.names[0])} v ${escHtml(s.names[1])}</div>`
              : ''}
            ${rulesSummary(s.match.rules) ? `<div class="side-meta side-meta-sub">${escHtml(rulesSummary(s.match.rules))}</div>` : ''}
            ${fairTxt ? `<div class="side-meta" style="color:${fairTxt.c};font-weight:700;">${escHtml(fairTxt.t)}</div>` : ''}
            <div class="side-log-title">Log</div>
            <div class="side-log-list">
              ${s.log.length === 0
                ? '<div class="log-item">Nothing yet.</div>'
                : s.log.slice().reverse().slice(0, 14).map((e) => `<div class="log-item">• ${escHtml(logText(s, e))}</div>`).join('')}
            </div>
            ${probData
              ? `<div class="prob-box">
                   <div class="prob-label">Win odds · ${escHtml(probData.summary)}</div>
                   <div class="prob-bar-track"><div class="prob-bar-fill ${probCls}" style="width:${probData.pct}%"></div></div>
                 </div>`
              : ''}
            <div class="side-toggles">
              ${flipBtnHtml}
              <button type="button" class="kb-btn sm pill ${save.settings.winMeter ? 'on' : ''}" data-action="toggle-prob">Prob: ${save.settings.winMeter ? 'On' : 'Off'} [P]</button>
              <button type="button" class="kb-btn sm pill ${save.settings.sound ? 'on' : ''}" data-action="toggle-sound">Sound: ${save.settings.sound ? 'On' : 'Off'} [S]</button>
            </div>
          </aside>
        </div>

        <div class="play-actions">
          ${actions.map((a) => `<button type="button" class="kb-btn sm" data-play-act="${a.id}">${escHtml(a.label)}</button>`).join('')}
        </div>
        <div class="footer-hint">
          <span class="footer-desktop">${desktopHint}</span>
          <span class="footer-touch">${touchHint}</span>
        </div>
      </div>
      ${renderOverlayHTML(s)}
    `;
    fitPlayLayout();
  }

  let fitRaf = 0;
  function scheduleFitPlayLayout() {
    if (app.view !== 'play') return;
    if (fitRaf) cancelAnimationFrame(fitRaf);
    fitRaf = requestAnimationFrame(() => {
      fitRaf = 0;
      fitPlayLayout();
    });
  }

  function fitPlayLayout() {
    if (app.view !== 'play') return;
    const playShell = rootEl.querySelector('.play-shell');
    const arenaEl = playShell && playShell.querySelector('.play-arena');
    const boardsColEl = arenaEl && arenaEl.querySelector('.boards-column');
    const boardBoxEl = boardsColEl && boardsColEl.querySelector('.board-box');
    const gridColEl = boardsColEl && boardsColEl.querySelector('.grid-col');
    if (!playShell || !arenaEl || !boardsColEl || !boardBoxEl || !gridColEl) return;

    const rootStyle = getComputedStyle(rootEl);
    const padY = (parseFloat(rootStyle.paddingTop) || 0) + (parseFloat(rootStyle.paddingBottom) || 0);
    const vpH = Math.min(
      window.innerHeight || Infinity,
      document.documentElement.clientHeight || Infinity
    );
    const usableRootH = Math.max(320, vpH - padY);

    const boxHeight = (el) => (el ? el.getBoundingClientRect().height : 0);

    const shellStyle = getComputedStyle(playShell);
    const shellGap = parseFloat(shellStyle.rowGap) || 0;
    let nonArenaShellH = 0;
    let shellCount = 1;
    for (const sel of ['.play-header', '.play-actions', '.footer-hint']) {
      const el = playShell.querySelector(sel);
      const h = boxHeight(el);
      if (h > 0) {
        nonArenaShellH += h;
        shellCount++;
      }
    }
    const availArenaH = usableRootH - nonArenaShellH - Math.max(0, shellCount - 1) * shellGap;

    const arenaStyle = getComputedStyle(arenaEl);
    const isStacked = arenaStyle.flexDirection === 'column' ||
      arenaStyle.gridTemplateColumns.trim().split(/\s+/).length <= 1;

    let availBoardsH = availArenaH;
    let availBoardsW = boardsColEl.clientWidth;

    if (isStacked) {
      const arenaGap = parseFloat(arenaStyle.rowGap) || 0;
      let otherArenaH = 0;
      let arenaCount = 1;
      for (const sel of ['.panel-top', '.panel-bottom', '.thumb-drop-bar', '.side-panel']) {
        const el = arenaEl.querySelector(sel);
        if (el && getComputedStyle(el).display !== 'none') {
          otherArenaH += boxHeight(el);
          arenaCount++;
        }
      }
      availBoardsH = availArenaH - otherArenaH - Math.max(0, arenaCount - 1) * arenaGap;
      availBoardsW = arenaEl.clientWidth;
    }

    const boardsStyle = getComputedStyle(boardsColEl);
    const boardsGap = parseFloat(boardsStyle.rowGap) || 0;
    let scoresH = 0;
    boardsColEl.querySelectorAll('.col-scores-row').forEach((el) => { scoresH += boxHeight(el); });
    const dividerEl = boardsColEl.querySelector('.aim-divider');
    const dividerH = boxHeight(dividerEl);

    const boxStyle = getComputedStyle(boardBoxEl);
    const boxPadV = (parseFloat(boxStyle.paddingTop) || 0) + (parseFloat(boxStyle.paddingBottom) || 0) +
      (parseFloat(boxStyle.borderTopWidth) || 0) + (parseFloat(boxStyle.borderBottomWidth) || 0);
    const boxPadH = (parseFloat(boxStyle.paddingLeft) || 0) + (parseFloat(boxStyle.paddingRight) || 0) +
      (parseFloat(boxStyle.borderLeftWidth) || 0) + (parseFloat(boxStyle.borderRightWidth) || 0);

    const colStyle = getComputedStyle(gridColEl);
    const colPadV = (parseFloat(colStyle.paddingTop) || 0) + (parseFloat(colStyle.paddingBottom) || 0);
    const colPadH = (parseFloat(colStyle.paddingLeft) || 0) + (parseFloat(colStyle.paddingRight) || 0);

    const dieHFactor = document.body.classList.contains('mode-tui') ? 0.82 : 1;
    const minDie = 24;
    const maxDie = isStacked ? 108 : 116;
    const colCountGap = Math.max(0, boardsColEl.children.length - 1) * boardsGap;

    const solveForGap = (gap) => {
      const maxW = (availBoardsW - boxPadH - 3 * colPadH - 2 * gap - 4) / 3;
      const fixedH = scoresH + dividerH + colCountGap + 2 * boxPadV + 2 * colPadV + 4 * gap + 8;
      const maxH = (availBoardsH - fixedH) / (6 * dieHFactor);
      return Math.floor(Math.max(minDie, Math.min(maxW, maxH, maxDie)));
    };

    let dieSize = solveForGap(isStacked ? 7 : 10);
    let gridGap = Math.round(Math.max(4, Math.min(12, dieSize * 0.12)));
    dieSize = solveForGap(gridGap);
    gridGap = Math.round(Math.max(4, Math.min(12, dieSize * 0.12)));
    const colSize = dieSize + Math.ceil(colPadH);

    playShell.style.setProperty('--die-size', `${dieSize}px`);
    playShell.style.setProperty('--col-size', `${colSize}px`);
    playShell.style.setProperty('--grid-gap', `${gridGap}px`);
  }

  function renderOverlayHTML(s) {
    if (app.overlay === 'log') {
      const lines = s.log.length === 0
        ? '<p class="note-hint">Nothing yet.</p>'
        : s.log.map((e) => {
            const col = e.kind === 'round' ? 'var(--accent)' : e.kind === 'forfeit' || e.kind === 'timeout' ? 'var(--lose)' : 'var(--fg)';
            return `<div style="color:${col}">${escHtml(logText(s, e))}</div>`;
          }).join('');
      return `
        <div class="overlay-backdrop" data-close-overlay="1">
          <div class="kb-card overlay-card">
            <h2 class="kb-card-title">Logs</h2>
            <div class="overlay-scroll">${lines}</div>
            <div class="btn-row"><button type="button" class="kb-btn sm" data-action="close-overlay">Close</button></div>
          </div>
        </div>`;
    }
    if (app.overlay === 'history') {
      const lines = s.history.length === 0
        ? '<p class="note-hint">No completed rounds yet.</p>'
        : s.history.map((h) => {
            const wText = h.winner < 0 ? 'Tie' : `${whoName(s, h.winner)} won`;
            return `<div>Round ${h.round}: ${escHtml(s.names[0])} ${h.scores[0]} – ${h.scores[1]} ${escHtml(s.names[1])} (${escHtml(wText)})</div>`;
          }).join('');
      return `
        <div class="overlay-backdrop" data-close-overlay="1">
          <div class="kb-card overlay-card">
            <h2 class="kb-card-title">History</h2>
            <div class="overlay-scroll">${lines}</div>
            <div class="btn-row"><button type="button" class="kb-btn sm" data-action="close-overlay">Close</button></div>
          </div>
        </div>`;
    }
    if (app.overlay === 'resign' || app.overlay === 'leave') {
      const isRes = app.overlay === 'resign';
      const title = isRes ? 'Resign this match?' : 'Leave this game?';
      const warn = isRes
        ? (s.mode === 'local' ? `${s.names[s.match.turn]} will forfeit the current match.` : 'Resigning forfeits the current match and counts as a loss.')
        : (s.phase === 'play' ? 'Leaving ends the current match.' : 'Return to the main menu.');
      const noLabel = isRes ? 'Cancel' : 'Stay';
      const yesLabel = isRes ? 'Confirm resignation' : 'Leave';
      return `
        <div class="overlay-backdrop">
          <div class="kb-card" style="max-width:520px;">
            <h2 class="kb-card-title">${escHtml(title)}</h2>
            <p style="text-align:center;font-size:1.02rem;">${escHtml(warn)}</p>
            <div class="btn-row">
              <button type="button" class="kb-btn ${app.confirmIndex === 0 ? 'selected' : ''}" data-action="close-overlay">${escHtml(noLabel)}</button>
              <button type="button" class="kb-btn ${app.confirmIndex === 1 ? 'danger-selected' : ''}" data-action="${isRes ? 'confirm-resign' : 'leave-game'}">${escHtml(yesLabel)}</button>
            </div>
          </div>
        </div>`;
    }
    if (s.phase !== 'play' && !app.hideResult && !app.replayActive) {
      const res = resultHeadline(s);
      const sc = matchScores(s.match);
      const choices = [];
      if (!app.ctl.opponentGone()) choices.push(...sessionOptions(s));
      choices.push('menu');
      const labels = {
        continue: 'Continue game',
        rematch: 'Play another game',
        free: 'Go into free play',
        menu: 'Menu'
      };

      const me = bottomSeat();
      const myVote = s.votes[me];
      const oppVote = s.mode === 'online' ? s.votes[1 - me] : '';
      let voteStatus = app.recordNote || '';
      if (s.mode === 'online') {
        if (oppVote === 'menu') voteStatus = `${s.names[1 - me]} chose Menu`;
        else if (myVote && myVote !== oppVote) voteStatus = `Waiting for ${s.names[1 - me]}...`;
        else if (oppVote) voteStatus = `${s.names[1 - me]} chose ${labels[oppVote] || oppVote}`;
      }

      return `
        <div class="overlay-backdrop">
          <div class="kb-card" style="max-width:640px;">
            <h2 class="kb-card-title">${s.phase === 'game' ? 'Game over' : 'Round over'}</h2>
            <p style="text-align:center;font-size:1.2rem;font-weight:800;color:${res.color};margin:6px 0;">${escHtml(res.text)}</p>
            <p style="text-align:center;font-size:1.05rem;margin:4px 0;">${escHtml(s.names[0])} ${sc[0]}  –  ${sc[1]} ${escHtml(s.names[1])}</p>
            ${s.bestOf !== 1 || s.freePlay ? `<p class="note-hint">Rounds won ${s.wins[0]} – ${s.wins[1]} · ${escHtml(roundsLabel(s.bestOf))}</p>` : ''}
            ${voteStatus ? `<p class="note-hint" style="color:var(--accent);font-weight:700;">${escHtml(voteStatus)}</p>` : ''}
            <div class="btn-row">
              ${choices.map((ch, idx) => `<button type="button" class="kb-btn ${idx === app.resultIndex ? 'selected' : ''}" data-result-choice="${ch}">${escHtml(labels[ch])}</button>`).join('')}
              ${s.roundMoves && s.roundMoves.length > 0 ? `<button type="button" class="kb-btn" data-play-act="enter-replay">Replay</button>` : ''}
              <button type="button" class="kb-btn" data-play-act="toggle-result">Board</button>
            </div>
          </div>
        </div>`;
    }
    return '';
  }

  // =========================================================================
  // 11. EVENT HANDLERS (Click / Touch / Keyboard)
  // =========================================================================
  function openSettings(mode) {
    app.setupMode = mode;
    const rows = settingRowsForMode();
    app.setIndex = mode === 'options' ? 0 : rows.length - 1;
    app.note = '';
    app.view = 'settings';
    emitSound('click');
    render();
  }

  function changeSetting(row, step) {
    const st = save.settings;
    const cyc = (arr, cur) => arr[(arr.indexOf(cur) + step + arr.length) % arr.length];
    switch (row) {
      case 'difficulty': st.difficulty = cyc(DIFF_ORDER, st.difficulty); break;
      case 'rounds': st.bestOf = cyc(ROUND_CHOICES, st.bestOf); break;
      case 'destroy': st.rules.noDestroy = !st.rules.noDestroy; break;
      case 'dice': st.rules.twoDice = !st.rules.twoDice; break;
      case 'end': st.rules.fillBoth = !st.rules.fillBoth; break;
      case 'theme': stepTheme(step); render(); return;
      case 'display': toggleDisplayMode(); render(); return;
      case 'sound':
        st.sound = !st.sound;
        persistSave(save);
        if (st.sound) emitSound('click');
        render();
        return;
      case 'haptics':
        st.haptics = !st.haptics;
        persistSave(save);
        if (st.haptics) triggerHaptic('aim');
        emitSound('click');
        render();
        return;
      case 'quickTap':
        st.quickTap = !st.quickTap;
        app.touchArmedCol = -1;
        break;
      case 'colorblind': st.colorblind = !st.colorblind; break;
      case 'winMeter': st.winMeter = !st.winMeter; break;
      default: return;
    }
    persistSave(save);
    emitSound('click');
    render();
  }

  function startFromSettings() {
    save.settings.player2 = cleanName(save.settings.player2, 'Player 2');
    persistSave(save);
    app.note = '';

    if (app.setupMode === 'ai') {
      const names = [save.name, AI_NAME[save.settings.difficulty]];
      const sess = createSession('ai', names, save.settings.difficulty, save.settings.bestOf, save.settings.rules, 0, false);
      beginGame(createOfflineController(sess, true, () => { syncSessionEvents(); render(); }));
    } else if (app.setupMode === 'local') {
      let p2 = save.settings.player2;
      if (p2.toLowerCase() === save.name.toLowerCase()) p2 += ' (2)';
      const sess = createSession('local', [save.name, p2], '', save.settings.bestOf, save.settings.rules, 0, false);
      beginGame(createOfflineController(sess, false, () => { syncSessionEvents(); render(); }));
    }
  }

  function submitJoin() {
    const addrInput = document.getElementById('input-join-addr');
    const codeInput = document.getElementById('input-join-code');
    if (addrInput) app.joinAddr = addrInput.value;
    if (codeInput) app.joinCode = codeInput.value;
    const addr = normalizeJoinAddr(cleanText(app.joinAddr, 80));
    if (!addr) {
      app.note = 'Enter host address (host:port).';
      app.joinField = 0;
      render();
      return;
    }
    app.joinAddr = addr;
    save.settings.joinAddr = addr;
    persistSave(save);
    app.note = '';
    const g = createWsGuestController({
      addr,
      roomCode: cleanRoomCode(app.joinCode),
      guestName: save.name
    }, () => {
      if (app.ctl === g) {
        syncSessionEvents();
        render();
      }
    });
    beginGame(g);
  }

  function activateMenuItem(id) {
    emitSound('click');
    app.note = '';
    if (id === 'ai' || id === 'local' || id === 'options') {
      openSettings(id);
    } else if (id === 'join') {
      app.joinAddr = save.settings.joinAddr || `localhost:${DEFAULT_PORT}`;
      app.view = 'join';
      app.joinField = 0;
      render();
    } else if (id === 'guide') {
      app.view = 'guide';
      render();
    } else if (id === 'stats') {
      app.view = 'stats';
      render();
    } else if (id === 'name') {
      app.nameDraft = save.name;
      app.view = 'name';
      render();
    }
  }

  function placeAtColumn(col) {
    const s = app.ctl && app.ctl.state();
    if (!s || !myTurn(s) || isRolling() || app.overlay !== 'none' || app.replayActive) return;
    if (openSlot(s.match.grids[s.match.turn][col]) < 0) return;
    app.cursor = col;
    app.touchArmedCol = -1;
    app.ctl.place({ col, die: app.dieSel });
    syncSessionEvents();
    render();
  }

  function handleColumnTap(col) {
    const s = app.ctl && app.ctl.state();
    if (!s || !myTurn(s) || isRolling() || app.overlay !== 'none' || app.replayActive) return;
    if (openSlot(s.match.grids[s.match.turn][col]) < 0) {
      emitSound('click');
      return;
    }
    const needPreview = lastPointerType !== 'mouse' && !save.settings.quickTap;
    if (needPreview && app.touchArmedCol !== col) {
      app.cursor = col;
      app.touchArmedCol = col;
      emitSound('click');
      triggerHaptic('aim');
      syncAim(s);
      render();
      return;
    }
    placeAtColumn(col);
  }

  let lastPointerType = 'mouse';
  let touchSwipeStart = null;
  rootEl.addEventListener('pointerdown', (e) => {
    if (e.pointerType) lastPointerType = e.pointerType;
    AudioEngine.unlock();
  }, { passive: true });

  rootEl.addEventListener('touchstart', (e) => {
    lastPointerType = 'touch';
    if (e.touches && e.touches.length === 1 && app.view === 'play' && app.replayActive && app.overlay === 'none') {
      const t = e.touches[0];
      touchSwipeStart = { x: t.clientX, y: t.clientY, at: Date.now() };
    } else {
      touchSwipeStart = null;
    }
  }, { passive: true });

  rootEl.addEventListener('touchend', (e) => {
    if (!touchSwipeStart || app.view !== 'play' || !app.replayActive || app.overlay !== 'none') {
      touchSwipeStart = null;
      return;
    }
    const t = e.changedTouches && e.changedTouches[0];
    if (!t) { touchSwipeStart = null; return; }
    const dx = t.clientX - touchSwipeStart.x;
    const dy = t.clientY - touchSwipeStart.y;
    const dt = Date.now() - touchSwipeStart.at;
    touchSwipeStart = null;
    if (dt < 650 && Math.abs(dx) >= 44 && Math.abs(dx) > Math.abs(dy) * 1.4) {
      if (dx < 0 && app.replayStep + 1 < app.replayFrames.length) {
        app.replayStep++;
        emitSound('click');
        triggerHaptic('aim');
        render();
      } else if (dx > 0 && app.replayStep > 0) {
        app.replayStep--;
        emitSound('click');
        triggerHaptic('aim');
        render();
      }
    }
  }, { passive: true });

  rootEl.addEventListener('pointermove', (e) => {
    if (e.pointerType && e.pointerType !== 'mouse') return;
    lastPointerType = 'mouse';
    if (app.view !== 'play' || app.overlay !== 'none' || app.replayActive) return;
    const colEl = e.target.closest('[data-col]');
    if (!colEl) return;
    const s = app.ctl && app.ctl.state();
    if (!s || !myTurn(s) || isRolling()) return;
    const c = Number(colEl.dataset.col);
    if (c !== app.cursor) {
      app.cursor = c;
      syncAim(s);
      render();
    }
  });

  rootEl.addEventListener('input', (e) => {
    const id = e.target.id;
    if (id === 'input-player2') save.settings.player2 = e.target.value;
    else if (id === 'input-player-name') app.nameDraft = e.target.value;
    else if (id === 'input-join-addr') app.joinAddr = e.target.value;
    else if (id === 'input-join-code') app.joinCode = e.target.value;
  });

  rootEl.addEventListener('change', (e) => {
    if (e.target.id === 'import-save-file' && e.target.files && e.target.files[0]) {
      const reader = new FileReader();
      reader.onload = () => {
        try {
          const parsed = JSON.parse(String(reader.result));
          const clean = sanitizeSave(parsed);
          Object.assign(save, clean);
          persistSave(save);
          app.note = 'Save imported successfully!';
          emitSound('win');
          render();
        } catch {
          app.note = 'Invalid knucklebones.json file.';
          render();
        }
      };
      reader.readAsText(e.target.files[0]);
    }
  });

  rootEl.addEventListener('click', (e) => {
    AudioEngine.unlock();

    const menuEl = e.target.closest('[data-menu]');
    if (menuEl) {
      activateMenuItem(menuEl.dataset.menu);
      return;
    }

    const stepBtn = e.target.closest('[data-set-step]');
    if (stepBtn) {
      e.stopPropagation();
      const idx = Number(stepBtn.dataset.setIdx);
      const step = Number(stepBtn.dataset.setStep);
      app.setIndex = idx;
      changeSetting(settingRowsForMode()[idx], step);
      return;
    }

    const rowEl = e.target.closest('[data-set-idx]');
    if (rowEl && !e.target.closest('input')) {
      const idx = Number(rowEl.dataset.setIdx);
      app.setIndex = idx;
      const row = settingRowsForMode()[idx];
      if (row === 'player2') {
        render();
        const inp = rowEl.querySelector('input');
        if (inp) inp.focus();
      } else {
        changeSetting(row, 1);
      }
      return;
    }

    const setAct = e.target.closest('[data-set-action]');
    if (setAct) {
      if (setAct.dataset.setAction === 'done') {
        emitSound('click');
        app.view = 'menu';
        render();
      } else {
        startFromSettings();
      }
      return;
    }

    const dropEl = e.target.closest('[data-drop-col]');
    if (dropEl && app.view === 'play') {
      handleColumnTap(Number(dropEl.dataset.dropCol));
      return;
    }

    const colEl = e.target.closest('[data-col]');
    if (colEl && app.view === 'play') {
      handleColumnTap(Number(colEl.dataset.col));
      return;
    }

    const dieEl = e.target.closest('[data-die]');
    if (dieEl && app.view === 'play') {
      const s = app.ctl && app.ctl.state();
      if (s && myTurn(s) && s.match.dice[1] !== 0) {
        app.dieSel = Number(dieEl.dataset.die);
        app.touchArmedCol = -1;
        emitSound('click');
        triggerHaptic('aim');
        syncAim(s);
        render();
      }
      return;
    }

    const resChoice = e.target.closest('[data-result-choice]');
    if (resChoice && app.ctl) {
      const ch = resChoice.dataset.resultChoice;
      emitSound('click');
      if (ch === 'menu') leaveGame();
      else app.ctl.vote(ch);
      return;
    }

    const playAct = e.target.closest('[data-play-act]');
    if (playAct && app.ctl) {
      const s = app.ctl.state();
      switch (playAct.dataset.playAct) {
        case 'open-cb':
          save.settings.colorblind = !save.settings.colorblind;
          persistSave(save);
          emitSound('click');
          render();
          break;
        case 'toggle-mode':
          toggleDisplayMode();
          render();
          break;
        case 'open-log':
          app.overlay = app.overlay === 'log' ? 'none' : 'log';
          emitSound('click');
          render();
          break;
        case 'open-history':
          app.overlay = app.overlay === 'history' ? 'none' : 'history';
          emitSound('click');
          render();
          break;
        case 'open-resign':
          app.overlay = 'resign';
          app.confirmIndex = 0;
          emitSound('click');
          render();
          break;
        case 'open-leave':
          if (s && s.phase === 'game') leaveGame();
          else {
            app.overlay = 'leave';
            app.confirmIndex = 0;
            emitSound('click');
            render();
          }
          break;
        case 'toggle-result':
          app.hideResult = !app.hideResult;
          emitSound('click');
          render();
          break;
        case 'enter-replay':
          enterReplay(s);
          break;
        case 'rep-prev':
          if (app.replayStep > 0) { app.replayStep--; emitSound('click'); render(); }
          break;
        case 'rep-next':
          if (app.replayStep + 1 < app.replayFrames.length) { app.replayStep++; emitSound('click'); render(); }
          break;
        case 'rep-done':
          exitReplay();
          emitSound('click');
          render();
          break;
      }
      return;
    }

    const actEl = e.target.closest('[data-action]');
    if (actEl) {
      const act = actEl.dataset.action;
      if (act === 'back-menu') {
        emitSound('click');
        app.note = '';
        app.view = 'menu';
        render();
      } else if (act === 'cycle-theme') {
        stepTheme(1);
        render();
      } else if (act === 'toggle-mode') {
        toggleDisplayMode();
        render();
      } else if (act === 'toggle-prob') {
        save.settings.winMeter = !save.settings.winMeter;
        persistSave(save);
        emitSound('click');
        render();
      } else if (act === 'toggle-sound') {
        save.settings.sound = !save.settings.sound;
        persistSave(save);
        if (save.settings.sound) emitSound('click');
        render();
      } else if (act === 'toggle-flip-p2') {
        app.flipP2 = !app.flipP2;
        emitSound('click');
        render();
      } else if (act === 'save-name') {
        save.name = cleanName(app.nameDraft, 'Lamb');
        persistSave(save);
        emitSound('click');
        app.view = 'menu';
        render();
      } else if (act === 'submit-join') {
        submitJoin();
      } else if (act === 'leave-game') {
        emitSound('click');
        leaveGame();
      } else if (act === 'back-join') {
        const wasBadAuth = app.ctl && app.ctl.lobbyError && app.ctl.lobbyError() === 'bad_auth';
        if (app.ctl) { app.ctl.leave(); app.ctl = null; }
        stopClockLoop();
        app.note = '';
        app.joinField = wasBadAuth ? 1 : 0;
        app.view = 'join';
        emitSound('click');
        render();
      } else if (act === 'retry-join') {
        if (app.ctl && app.ctl.retryConnect) {
          emitSound('click');
          app.ctl.retryConnect();
          render();
        }
      } else if (act === 'confirm-resign') {
        app.overlay = 'none';
        emitSound('click');
        if (app.ctl) app.ctl.resign();
      } else if (act === 'close-overlay') {
        app.overlay = 'none';
        emitSound('click');
        render();
      } else if (act === 'export-save') {
        const blob = new Blob([JSON.stringify(save, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'knucklebones.json';
        a.click();
        URL.revokeObjectURL(url);
        emitSound('click');
      } else if (act === 'import-save') {
        document.getElementById('import-save-file')?.click();
      } else if (act === 'reset-stats') {
        if (confirm('Reset all saved statistics and Top 10 scores?')) {
          save.records = {};
          save.highScores = [];
          persistSave(save);
          app.note = 'Stats cleared.';
          emitSound('click');
          render();
        }
      } else if (act === 'install' && app.installPromptEvent) {
        const evt = app.installPromptEvent;
        if (typeof evt.prompt === 'function') {
          Promise.resolve(evt.prompt()).catch(() => {});
        }
        Promise.resolve(evt.userChoice).catch(() => {}).finally(() => {
          app.installPromptEvent = null;
          render();
        });
      } else if (act === 'apply-update') {
        applyPendingUpdateNow();
      }
      return;
    }

    if (e.target.dataset.closeOverlay === '1') {
      app.overlay = 'none';
      emitSound('click');
      render();
    }
  });

  document.addEventListener('keydown', (e) => {
    AudioEngine.unlock();
    const activeTag = document.activeElement && document.activeElement.tagName;
    const typing = activeTag === 'INPUT' || activeTag === 'TEXTAREA';

    if (e.key === 'Escape') {
      e.preventDefault();
      if (app.view === 'menu') return;
      if (app.view === 'settings' || app.view === 'name' || app.view === 'join' || app.view === 'guide' || app.view === 'stats') {
        app.note = '';
        app.view = 'menu';
        emitSound('click');
        render();
        return;
      }
      if (app.view === 'lobby') {
        if (app.ctl && app.ctl.seat() === 1) {
          document.querySelector('[data-action="back-join"]')?.click();
        } else {
          leaveGame();
        }
        return;
      }
      if (app.view === 'play') {
        if (app.overlay !== 'none') {
          app.overlay = 'none';
          render();
        } else if (app.replayActive) {
          exitReplay();
          render();
        } else {
          const s = app.ctl && app.ctl.state();
          if (s && s.phase === 'game') leaveGame();
          else {
            app.overlay = 'leave';
            app.confirmIndex = 0;
            render();
          }
        }
        return;
      }
    }

    if (typing) {
      if (e.key === 'Enter') {
        e.preventDefault();
        if (app.view === 'name') {
          save.name = cleanName(app.nameDraft, 'Lamb');
          persistSave(save);
          app.view = 'menu';
          render();
        } else if (app.view === 'join') {
          submitJoin();
        } else if (app.view === 'settings') {
          const rows = settingRowsForMode();
          app.setIndex = (app.setIndex + 1) % rows.length;
          document.activeElement.blur();
          render();
        }
      } else if ((e.key === 'Tab' || e.key === 'ArrowUp' || e.key === 'ArrowDown') && app.view === 'join') {
        e.preventDefault();
        app.joinField = 1 - app.joinField;
        render();
      }
      return;
    }

    const low = e.key.toLowerCase();
    if (app.view === 'menu') {
      const keys = ['ai', 'local', 'join', 'guide', 'stats', 'name', 'options'];
      if (e.key === 'ArrowUp' || low === 'w') {
        e.preventDefault();
        app.menuIndex = (app.menuIndex + keys.length - 1) % keys.length;
        render();
      } else if (e.key === 'ArrowDown' || low === 's') {
        e.preventDefault();
        app.menuIndex = (app.menuIndex + 1) % keys.length;
        render();
      } else if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        activateMenuItem(keys[app.menuIndex]);
      } else if (low === 't') {
        stepTheme(1);
        render();
      } else if (low === 'm') {
        toggleDisplayMode();
        render();
      }
      return;
    }

    if (app.view === 'settings') {
      const rows = settingRowsForMode();
      const row = rows[app.setIndex];
      if (e.key === 'ArrowUp' || low === 'w') {
        e.preventDefault();
        app.setIndex = (app.setIndex + rows.length - 1) % rows.length;
        render();
      } else if (e.key === 'ArrowDown' || low === 's') {
        e.preventDefault();
        app.setIndex = (app.setIndex + 1) % rows.length;
        render();
      } else if (e.key === 'ArrowLeft' || low === 'a') {
        e.preventDefault();
        changeSetting(row, -1);
      } else if (e.key === 'ArrowRight' || low === 'd') {
        e.preventDefault();
        changeSetting(row, 1);
      } else if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        if (row === 'start') startFromSettings();
        else if (row === 'done') { app.view = 'menu'; emitSound('click'); render(); }
        else if (row === 'player2') {
          render();
          document.querySelector('.setting-row.selected input')?.focus();
        } else changeSetting(row, 1);
      }
      return;
    }

    if (app.view === 'play' && app.ctl) {
      const s = app.ctl.state();
      if (!s) return;

      if (app.overlay === 'log' && low === 'l') { app.overlay = 'none'; render(); return; }
      if (app.overlay === 'history' && low === 'h') { app.overlay = 'none'; render(); return; }
      if (app.overlay === 'resign' || app.overlay === 'leave') {
        if (e.key === 'ArrowLeft' || e.key === 'ArrowRight' || low === 'a' || low === 'd') {
          app.confirmIndex = 1 - app.confirmIndex;
          render();
        } else if (e.key === 'Enter') {
          if (app.confirmIndex === 1) {
            if (app.overlay === 'resign') { app.overlay = 'none'; app.ctl.resign(); }
            else leaveGame();
          } else {
            app.overlay = 'none';
            render();
          }
        }
        return;
      }
      if (app.overlay !== 'none') return;

      if (app.replayActive) {
        if (e.key === 'ArrowLeft' || low === 'a') { if (app.replayStep > 0) { app.replayStep--; render(); } }
        else if (e.key === 'ArrowRight' || low === 'd') { if (app.replayStep + 1 < app.replayFrames.length) { app.replayStep++; render(); } }
        else if (e.key === 'Home') { app.replayStep = 0; render(); }
        else if (e.key === 'End') { app.replayStep = Math.max(0, app.replayFrames.length - 1); render(); }
        else if (e.key === 'Enter') { exitReplay(); render(); }
        return;
      }

      if (low === 'l') { app.overlay = 'log'; emitSound('click'); render(); return; }
      if (low === 'h') { app.overlay = 'history'; emitSound('click'); render(); return; }
      if (low === 's') {
        save.settings.sound = !save.settings.sound;
        persistSave(save);
        if (save.settings.sound) emitSound('click');
        render();
        return;
      }
      if (low === 'c') {
        save.settings.colorblind = !save.settings.colorblind;
        persistSave(save);
        emitSound('click');
        render();
        return;
      }
      if (low === 'p') {
        save.settings.winMeter = !save.settings.winMeter;
        persistSave(save);
        emitSound('click');
        render();
        return;
      }
      if (low === 't') { stepTheme(1); render(); return; }
      if (low === 'm') { toggleDisplayMode(); render(); return; }

      if (s.phase !== 'play') {
        const choices = [];
        if (!app.ctl.opponentGone()) choices.push(...sessionOptions(s));
        choices.push('menu');
        if (low === 'v') { app.hideResult = !app.hideResult; emitSound('click'); render(); return; }
        if (low === 'x' && s.roundMoves && s.roundMoves.length > 0) { enterReplay(s); return; }
        if (e.key === 'ArrowLeft' || low === 'a') { app.resultIndex = (app.resultIndex + choices.length - 1) % choices.length; render(); }
        else if (e.key === 'ArrowRight' || low === 'd') { app.resultIndex = (app.resultIndex + 1) % choices.length; render(); }
        else if (e.key === 'Enter' && !app.hideResult) {
          const ch = choices[app.resultIndex] || 'menu';
          emitSound('click');
          if (ch === 'menu') leaveGame();
          else app.ctl.vote(ch);
        }
        return;
      }

      if (low === 'r') { app.overlay = 'resign'; app.confirmIndex = 0; emitSound('click'); render(); return; }
      if (!myTurn(s) || isRolling()) return;

      if (e.key === 'ArrowLeft' || low === 'a') {
        e.preventDefault();
        app.cursor = (app.cursor + GRID_COLS - 1) % GRID_COLS;
        syncAim(s);
        render();
      } else if (e.key === 'ArrowRight' || low === 'd') {
        e.preventDefault();
        app.cursor = (app.cursor + 1) % GRID_COLS;
        syncAim(s);
        render();
      } else if ((e.key === 'Tab' || low === 'w' || low === 'q' || low === 'e' || e.key === 'ArrowUp' || e.key === 'ArrowDown') && s.match.dice[1] !== 0) {
        e.preventDefault();
        app.dieSel = 1 - app.dieSel;
        emitSound('click');
        syncAim(s);
        render();
      } else if (e.key === '1' || e.key === '2' || e.key === '3') {
        placeAtColumn(Number(e.key) - 1);
      } else if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        placeAtColumn(app.cursor);
      }
    }
  });

  // =========================================================================
  // 12. PWA INSTALL, SERVICE WORKER AUTO-UPDATE & URL SHORTCUTS
  // =========================================================================
  function isPwaStandalone() {
    return window.matchMedia('(display-mode: standalone)').matches ||
      window.navigator.standalone === true ||
      document.referrer.includes('android-app://');
  }

  function isIosDevice() {
    return /iPad|iPhone|iPod/.test(navigator.userAgent) ||
      (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1);
  }

  window.addEventListener('beforeinstallprompt', (e) => {
    e.preventDefault();
    if (isPwaStandalone()) {
      app.installPromptEvent = null;
      return;
    }
    app.installPromptEvent = e;
    if (app.view === 'menu' || (app.view === 'settings' && app.setupMode === 'options')) render();
  });

  window.addEventListener('appinstalled', () => {
    app.installPromptEvent = null;
    if (app.view === 'menu' || (app.view === 'settings' && app.setupMode === 'options')) render();
  });

  let swReg = null;
  let reloadingForUpdate = false;

  function isMidGame() {
    if (app.view !== 'play' || !app.ctl) return false;
    const s = app.ctl.state();
    return !!(s && s.phase === 'play');
  }

  function showUpdateToast() {
    const toast = document.getElementById('update-toast');
    if (toast) toast.classList.remove('hidden');
  }

  function applyPendingUpdateNow() {
    if (reloadingForUpdate) return;
    if (swReg && swReg.waiting) {
      swReg.waiting.postMessage({ type: 'SKIP_WAITING' });
    }
    reloadingForUpdate = true;
    window.location.reload();
  }

  document.getElementById('update-toast-btn')?.addEventListener('click', applyPendingUpdateNow);
  document.getElementById('update-toast-dismiss')?.addEventListener('click', () => {
    document.getElementById('update-toast')?.classList.add('hidden');
  });

  if ('serviceWorker' in navigator) {
    let hasActiveController = !!navigator.serviceWorker.controller;

    function triggerSwUpdate() {
      if (isMidGame()) {
        app.swPendingUpdate = true;
        showUpdateToast();
        render();
      } else if (!reloadingForUpdate) {
        reloadingForUpdate = true;
        window.location.reload();
      }
    }

    navigator.serviceWorker.addEventListener('controllerchange', () => {
      if (!hasActiveController) {
        hasActiveController = true;
        return; // first-time SW claim
      }
      triggerSwUpdate();
    });

    navigator.serviceWorker.addEventListener('message', (ev) => {
      if (ev.data && ev.data.type === 'SW_UPDATED') {
        if (!hasActiveController) return;
        triggerSwUpdate();
      }
    });

    window.addEventListener('load', () => {
      navigator.serviceWorker.register('./sw.js').then((reg) => {
        swReg = reg;
        reg.update().catch(() => {});

        document.addEventListener('visibilitychange', () => {
          if (document.visibilityState === 'visible') reg.update().catch(() => {});
        });
        window.addEventListener('online', () => {
          reg.update().catch(() => {});
        });
        setInterval(() => {
          reg.update().catch(() => {});
        }, 5 * 60 * 1000);
      }).catch((err) => {
        console.warn('[PWA] Service Worker registration failed:', err);
      });
    });
  }

  // Handle URL deep-links (?join=HOST:PORT or ?mode=ai|local|join|options)
  function handleInitialQuery() {
    const params = new URLSearchParams(location.search);
    const join = normalizeJoinAddr(cleanText(params.get('join') || '', 80));
    const mode = params.get('mode');
    if (join) {
      app.joinAddr = join;
      app.joinField = 1;
      app.view = 'join';
    } else if (mode === 'ai' || mode === 'local' || mode === 'options') {
      openSettings(mode);
      return;
    } else if (mode === 'join') {
      app.joinAddr = save.settings.joinAddr || `localhost:${DEFAULT_PORT}`;
      app.joinField = 0;
      app.view = 'join';
    }
    render();
  }

  window.addEventListener('resize', scheduleFitPlayLayout);
  if (window.visualViewport) window.visualViewport.addEventListener('resize', scheduleFitPlayLayout);

  applyThemeAndMode();
  handleInitialQuery();
})();
