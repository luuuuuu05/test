package api

const dashboardHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>MR Flag 管理端</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f6f7f9;
      --panel: #ffffff;
      --text: #111827;
      --muted: #667085;
      --line: #d9dee7;
      --accent: #0f766e;
      --accent-dark: #115e59;
      --info: #2563eb;
      --warn: #b45309;
      --danger: #b42318;
      --soft-info: #dbeafe;
      --soft-warn: #fef3c7;
      --soft-ok: #ccfbf1;
      --soft-danger: #fee4e2;
      --shadow: 0 10px 28px rgba(17, 24, 39, 0.08);
    }

    * {
      box-sizing: border-box;
    }

    body {
      margin: 0;
      background: var(--bg);
      color: var(--text);
      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      letter-spacing: 0;
    }

    button,
    table {
      font: inherit;
    }

    .shell {
      width: min(1180px, calc(100% - 32px));
      margin: 0 auto;
      padding: 28px 0 40px;
    }

    .topbar {
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      gap: 16px;
      margin-bottom: 18px;
    }

    h1 {
      margin: 0;
      font-size: 26px;
      line-height: 1.18;
      font-weight: 760;
    }

    .subtitle {
      margin-top: 6px;
      color: var(--muted);
      font-size: 14px;
    }

    .actions,
    .row-actions {
      display: flex;
      align-items: center;
      gap: 8px;
      flex-wrap: wrap;
    }

    .actions {
      justify-content: flex-end;
    }

    button {
      height: 34px;
      border: 1px solid var(--line);
      border-radius: 8px;
      background: #ffffff;
      color: var(--text);
      cursor: pointer;
      font-weight: 680;
      padding: 0 12px;
      white-space: nowrap;
    }

    button:hover {
      border-color: #aab3c2;
      background: #fbfcfe;
    }

    button:disabled {
      cursor: not-allowed;
      opacity: 0.55;
    }

    .primary {
      border-color: var(--accent);
      background: var(--accent);
      color: #ffffff;
    }

    .primary:hover {
      background: var(--accent-dark);
    }

    .danger {
      border-color: #f3b1aa;
      color: var(--danger);
    }

    .updated {
      min-height: 20px;
      color: var(--muted);
      font-size: 13px;
      text-align: right;
    }

    .metrics {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 12px;
      margin-bottom: 14px;
    }

    .metric {
      min-width: 0;
      border: 1px solid var(--line);
      border-radius: 8px;
      background: var(--panel);
      padding: 14px 16px;
      box-shadow: var(--shadow);
    }

    .metric-label {
      color: var(--muted);
      font-size: 13px;
      white-space: nowrap;
    }

    .metric-value {
      margin-top: 6px;
      font-size: 28px;
      line-height: 1;
      font-weight: 780;
    }

    .panel {
      overflow: hidden;
      border: 1px solid var(--line);
      border-radius: 8px;
      background: var(--panel);
      box-shadow: var(--shadow);
    }

    .notice {
      border-bottom: 1px solid #e6ebf2;
      background: #fbfcfe;
      color: var(--muted);
      padding: 10px 16px;
      font-size: 13px;
    }

    .error {
      display: none;
      border-bottom: 1px solid #fecaca;
      background: #fff1f2;
      color: var(--danger);
      padding: 12px 16px;
      font-size: 14px;
      font-weight: 620;
    }

    .error.is-visible {
      display: block;
    }

    .table-wrap {
      overflow-x: auto;
    }

    table {
      width: 100%;
      min-width: 920px;
      border-collapse: collapse;
    }

    th,
    td {
      padding: 14px 16px;
      border-bottom: 1px solid #edf0f5;
      text-align: left;
      vertical-align: middle;
    }

    th {
      background: #fbfcfe;
      color: var(--muted);
      font-size: 12px;
      font-weight: 720;
    }

    td {
      font-size: 14px;
    }

    tbody tr:last-child td {
      border-bottom: 0;
    }

    .mono {
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
      font-size: 13px;
    }

    .room-name {
      font-weight: 720;
    }

    .status-badge {
      display: inline-flex;
      align-items: center;
      min-height: 24px;
      border-radius: 6px;
      padding: 3px 8px;
      font-size: 12px;
      font-weight: 760;
      white-space: nowrap;
    }

    .status-waiting {
      background: var(--soft-info);
      color: var(--info);
    }

    .status-countdown {
      background: var(--soft-warn);
      color: var(--warn);
    }

    .status-playing {
      background: var(--soft-ok);
      color: var(--accent);
    }

    .status-ended {
      background: var(--soft-danger);
      color: var(--danger);
    }

    .capacity {
      display: flex;
      align-items: center;
      gap: 10px;
      min-width: 130px;
      white-space: nowrap;
    }

    .bar {
      width: 86px;
      height: 8px;
      overflow: hidden;
      border-radius: 4px;
      background: #e5e7eb;
    }

    .bar-fill {
      height: 100%;
      width: 0;
      border-radius: 4px;
      background: var(--accent);
    }

    .empty {
      padding: 42px 16px;
      color: var(--muted);
      text-align: center;
      font-size: 14px;
    }

    @media (max-width: 720px) {
      .shell {
        width: min(100% - 24px, 1180px);
        padding-top: 20px;
      }

      .topbar {
        display: block;
      }

      .actions {
        justify-content: space-between;
        margin-top: 14px;
      }

      .updated {
        text-align: left;
      }

      .metrics {
        grid-template-columns: 1fr;
      }

      h1 {
        font-size: 23px;
      }
    }
  </style>
</head>
<body>
  <main class="shell">
    <header class="topbar">
      <div>
        <h1>MR Flag 管理端</h1>
        <div class="subtitle">默认房间为 ROOM_DEFAULT，Unity 客户端接入后在这里开始游戏。</div>
      </div>
      <div>
        <div class="actions">
          <button class="primary" id="ensureDefault" type="button">准备默认房间</button>
          <button id="refresh" type="button">刷新</button>
        </div>
        <div class="updated" id="updated">正在读取</div>
      </div>
    </header>

    <section class="metrics" aria-label="房间统计">
      <div class="metric">
        <div class="metric-label">房间数</div>
        <div class="metric-value" id="roomCount">0</div>
      </div>
      <div class="metric">
        <div class="metric-label">玩家数</div>
        <div class="metric-value" id="playerCount">0</div>
      </div>
      <div class="metric">
        <div class="metric-label">进行中</div>
        <div class="metric-value" id="playingCount">0</div>
      </div>
    </section>

    <section class="panel">
      <div class="notice">开始游戏需要 2 名玩家在同一房间。按钮调用 REST：POST /api/rooms/:id/start。</div>
      <div class="error" id="error"></div>
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>房间</th>
              <th>Room ID</th>
              <th>邀请码</th>
              <th>状态</th>
              <th>玩家</th>
              <th>创建时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody id="rooms"></tbody>
        </table>
      </div>
      <div class="empty" id="empty" hidden>暂无房间</div>
    </section>
  </main>

  <script>
    const els = {
      ensureDefault: document.getElementById('ensureDefault'),
      refresh: document.getElementById('refresh'),
      updated: document.getElementById('updated'),
      error: document.getElementById('error'),
      roomCount: document.getElementById('roomCount'),
      playerCount: document.getElementById('playerCount'),
      playingCount: document.getElementById('playingCount'),
      rooms: document.getElementById('rooms'),
      empty: document.getElementById('empty')
    };

    const statusText = {
      waiting: '等待中',
      countdown: '倒计时',
      playing: '进行中',
      ended: '已结束'
    };

    const statusClass = {
      waiting: 'status-waiting',
      countdown: 'status-countdown',
      playing: 'status-playing',
      ended: 'status-ended'
    };

    const timeFmt = new Intl.DateTimeFormat('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    });

    let loading = false;

    function numberOf(value) {
      const n = Number(value);
      return Number.isFinite(n) ? n : 0;
    }

    function formatTime(ms) {
      const n = numberOf(ms);
      return n > 0 ? timeFmt.format(new Date(n)) : '-';
    }

    function setError(message) {
      if (!message) {
        els.error.classList.remove('is-visible');
        els.error.textContent = '';
        return;
      }
      els.error.textContent = message;
      els.error.classList.add('is-visible');
    }

    async function requestJSON(path, options) {
      const res = await fetch(path, Object.assign({ cache: 'no-store' }, options || {}));
      const text = await res.text();
      let data = null;
      if (text) {
        try {
          data = JSON.parse(text);
        } catch (_) {
          data = { message: text };
        }
      }
      if (!res.ok) {
        const msg = data && (data.message || data.code) ? (data.message || data.code) : 'HTTP ' + res.status;
        throw new Error(msg);
      }
      return data;
    }

    function makeCell(text, className) {
      const td = document.createElement('td');
      if (className) {
        td.className = className;
      }
      td.textContent = text;
      return td;
    }

    function renderStats(rooms) {
      const playerCount = rooms.reduce(function (sum, room) {
        return sum + numberOf(room.player_count);
      }, 0);
      const playingCount = rooms.filter(function (room) {
        return room.status === 'playing' || room.status === 'countdown';
      }).length;

      els.roomCount.textContent = String(rooms.length);
      els.playerCount.textContent = String(playerCount);
      els.playingCount.textContent = String(playingCount);
    }

    function actionButton(label, className, disabled, onClick) {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.textContent = label;
      if (className) {
        btn.className = className;
      }
      btn.disabled = disabled;
      btn.addEventListener('click', onClick);
      return btn;
    }

    function renderRooms(rooms) {
      els.rooms.textContent = '';
      els.empty.hidden = rooms.length !== 0;

      rooms.forEach(function (room) {
        const tr = document.createElement('tr');
        const roomID = room.room_id || '';
        const players = numberOf(room.player_count);
        const maxPlayers = Math.max(numberOf(room.max_players), players, 1);
        const status = room.status || 'waiting';
        const canStart = status === 'waiting' || status === 'ended';
        const canStop = status === 'countdown' || status === 'playing';

        tr.appendChild(makeCell(room.room_name || roomID || '-', 'room-name'));
        tr.appendChild(makeCell(roomID || '-', 'mono'));
        tr.appendChild(makeCell(room.join_code || '-', 'mono'));

        const statusCell = document.createElement('td');
        const badge = document.createElement('span');
        badge.className = 'status-badge ' + (statusClass[status] || 'status-waiting');
        badge.textContent = statusText[status] || status;
        statusCell.appendChild(badge);
        tr.appendChild(statusCell);

        const capacityCell = document.createElement('td');
        const capacity = document.createElement('div');
        const label = document.createElement('span');
        const bar = document.createElement('span');
        const fill = document.createElement('span');
        capacity.className = 'capacity';
        label.textContent = String(players) + '/' + String(maxPlayers);
        bar.className = 'bar';
        fill.className = 'bar-fill';
        fill.style.width = String(Math.min(100, Math.round((players / maxPlayers) * 100))) + '%';
        bar.appendChild(fill);
        capacity.appendChild(label);
        capacity.appendChild(bar);
        capacityCell.appendChild(capacity);
        tr.appendChild(capacityCell);

        tr.appendChild(makeCell(formatTime(room.created_at)));

        const actionCell = document.createElement('td');
        const actions = document.createElement('div');
        actions.className = 'row-actions';
        actions.appendChild(actionButton('开始游戏', 'primary', !canStart, function () {
          startRoom(roomID);
        }));
        actions.appendChild(actionButton('停止', 'danger', !canStop, function () {
          stopRoom(roomID);
        }));
        actions.appendChild(actionButton('详情', '', false, function () {
          inspectRoom(roomID);
        }));
        actionCell.appendChild(actions);
        tr.appendChild(actionCell);

        els.rooms.appendChild(tr);
      });
    }

    async function ensureDefaultRoom() {
      await requestJSON('/api/default-room', { method: 'POST' });
    }

    async function startRoom(roomID) {
      try {
        setError('');
        await requestJSON('/api/rooms/' + encodeURIComponent(roomID) + '/start', { method: 'POST' });
        els.updated.textContent = '已发送开始游戏';
        await loadRooms();
      } catch (err) {
        setError('开始失败：' + err.message);
      }
    }

    async function stopRoom(roomID) {
      try {
        setError('');
        await requestJSON('/api/rooms/' + encodeURIComponent(roomID) + '/stop', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ reason: 'admin_abort' })
        });
        els.updated.textContent = '已停止';
        await loadRooms();
      } catch (err) {
        setError('停止失败：' + err.message);
      }
    }

    async function inspectRoom(roomID) {
      try {
        const room = await requestJSON('/api/rooms/' + encodeURIComponent(roomID));
        const scene = room && room.scene_map ? '地图版本 ' + room.scene_map.version : '暂无地图';
        setError('房间 ' + roomID + '：' + room.status + '，玩家 ' + (room.players || []).length + '，' + scene);
      } catch (err) {
        setError('读取详情失败：' + err.message);
      }
    }

    async function loadRooms() {
      if (loading) {
        return;
      }
      loading = true;
      els.refresh.disabled = true;
      els.ensureDefault.disabled = true;

      try {
        const rooms = await requestJSON('/api/rooms');
        const list = Array.isArray(rooms) ? rooms : [];
        renderStats(list);
        renderRooms(list);
        if (!els.error.textContent.startsWith('房间 ')) {
          setError('');
        }
        els.updated.textContent = '更新于 ' + new Date().toLocaleTimeString('zh-CN');
      } catch (err) {
        setError('读取房间失败：' + err.message);
        els.updated.textContent = '读取失败';
      } finally {
        loading = false;
        els.refresh.disabled = false;
        els.ensureDefault.disabled = false;
      }
    }

    els.ensureDefault.addEventListener('click', async function () {
      try {
        setError('');
        await ensureDefaultRoom();
        await loadRooms();
      } catch (err) {
        setError('默认房间准备失败：' + err.message);
      }
    });

    els.refresh.addEventListener('click', loadRooms);

    (async function init() {
      try {
        await ensureDefaultRoom();
      } catch (err) {
        setError('默认房间准备失败：' + err.message);
      }
      await loadRooms();
      window.setInterval(loadRooms, 2000);
    })();
  </script>
</body>
</html>
`
