import './styles.css';
import * as api from './api';

function showConnectScreen() {
  const app = document.getElementById('app')!;
  app.innerHTML = `
    <div class="connect-panel">
      <h2>bitExchange Web</h2>
      <p style="color:var(--text-secondary);font-size:13px;">
        连接到 bitexchange-core 实例。
        请在另一台设备上运行 <code style="background:rgba(255,255,255,0.1);padding:2px 6px;border-radius:4px;">bitexchange-core --root ~/bitExchange</code>，
        然后输入其地址。
      </p>
      <input class="input-field" id="api-url" placeholder="http://192.168.1.10:8080" value="http://127.0.0.1:8080" />
      <button class="btn btn-primary" id="connect-btn">连接 →</button>
      <p id="connect-error" style="color:var(--error);font-size:12px;display:none;"></p>
    </div>
  `;

  const urlInput = app.querySelector('#api-url') as HTMLInputElement;
  const connectBtn = app.querySelector('#connect-btn') as HTMLButtonElement;
  const errorP = app.querySelector('#connect-error') as HTMLElement;

  connectBtn.addEventListener('click', async () => {
    const url = urlInput.value.trim();
    if (!url) return;
    api.setApiBase(url);
    try {
      await api.getPeers();
      showMainUI();
    } catch {
      errorP.textContent = `无法连接到 ${url}，请检查 core 是否运行`;
      errorP.style.display = 'block';
    }
  });

  urlInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') connectBtn.click();
  });
}

function showMainUI() {
  const app = document.getElementById('app')!;
  app.innerHTML = '';

  const left = document.createElement('div');
  left.className = 'panel panel-left';
  app.appendChild(left);

  const center = document.createElement('div');
  center.className = 'panel panel-center';
  app.appendChild(center);

  const right = document.createElement('div');
  right.className = 'panel panel-right';
  app.appendChild(right);

  const inputBar = document.createElement('div');
  app.appendChild(inputBar);

  // Device list
  let selectedTargets: string[] = [];
  const deviceTitle = document.createElement('div');
  deviceTitle.className = 'panel-title';
  deviceTitle.textContent = '在线设备';
  left.appendChild(deviceTitle);

  const deviceListEl = document.createElement('div');
  left.appendChild(deviceListEl);

  function renderDevices(peers: any[]) {
    deviceListEl.innerHTML = '';
    if (peers.length === 0) {
      deviceListEl.innerHTML = '<div class="device-meta">暂无在线设备</div>';
      return;
    }
    for (const p of peers) {
      const card = document.createElement('div');
      card.className = 'device-card';
      if (selectedTargets.includes(p.device_id)) card.classList.add('selected');
      card.innerHTML = `
        <div class="device-name">${p.device_name}</div>
        <div class="device-meta">${p.path} · ${(p.rate || 0).toFixed(1)}MB/s</div>
      `;
      card.onclick = () => {
        const idx = selectedTargets.indexOf(p.device_id);
        if (idx >= 0) selectedTargets.splice(idx, 1);
        else selectedTargets.push(p.device_id);
        renderDevices(peers);
      };
      deviceListEl.appendChild(card);
    }
  }

  // Chat stream
  const chatStream = document.createElement('div');
  chatStream.className = 'chat-stream';
  center.appendChild(chatStream);

  function appendChat(line: any, outgoing: boolean) {
    const bubble = document.createElement('div');
    bubble.className = outgoing ? 'bubble-out' : 'bubble-in';
    if (line.type === 'text') {
      bubble.textContent = line.body || '';
    } else {
      bubble.innerHTML = `<div style="font-size:10px;color:var(--text-muted);margin-bottom:4px;">${line.from} · 文件</div>📄 ${line.filename}`;
    }
    chatStream.appendChild(bubble);
    chatStream.scrollTop = chatStream.scrollHeight;
  }

  // Task panel
  const taskTitle = document.createElement('div');
  taskTitle.className = 'panel-title';
  taskTitle.textContent = '传输任务';
  right.appendChild(taskTitle);

  const taskListEl = document.createElement('div');
  right.appendChild(taskListEl);

  function renderTasks(tasks: any[]) {
    taskListEl.innerHTML = '';
    if (tasks.length === 0) {
      taskListEl.innerHTML = '<div class="device-meta">暂无任务</div>';
      return;
    }
    for (const t of tasks) {
      const card = document.createElement('div');
      card.className = 'task-card';
      if (t.status === 'done') card.classList.add('done');
      if (t.status === 'failed' || t.status === 'cancelled') card.classList.add('failed');
      const pct = Math.round(t.progress * 100);
      card.innerHTML = `
        <div style="color:var(--text-primary);font-size:12px;">${t.payload} → ${t.target}</div>
        <div class="progress-bar"><div class="progress-fill" style="width:${pct}%"></div></div>
        <div style="color:var(--text-secondary);font-size:10px;">${pct}% · ${t.path || 'pending'}</div>
      `;
      taskListEl.appendChild(card);
    }
  }

  // Input bar
  inputBar.className = 'input-bar';
  inputBar.innerHTML = `
    <input class="input-field" type="text" placeholder="输入消息，或拖入/粘贴文件..." />
    <button class="btn" id="file-btn">📎 文件</button>
    <button class="btn btn-toggle" id="encrypt-btn">🔒 加密</button>
    <button class="btn btn-primary" id="send-btn">发送 →</button>
    <input type="file" id="file-input" style="display:none" multiple />
  `;

  const input = inputBar.querySelector('.input-field') as HTMLInputElement;
  const fileInput = inputBar.querySelector('#file-input') as HTMLInputElement;
  const encryptBtn = inputBar.querySelector('#encrypt-btn') as HTMLButtonElement;
  let encrypted = false;

  inputBar.querySelector('#send-btn')!.addEventListener('click', async () => {
    if (selectedTargets.length === 0) { alert('请先选择至少一个目标设备'); return; }
    if (input.value.trim()) {
      const tasks = await api.sendText(selectedTargets, input.value, encrypted);
      appendChat({ type: 'text', from: 'me', body: input.value }, true);
      input.value = '';
      refreshTasks();
      for (const t of tasks) api.subscribeTask(t.id, () => refreshTasks());
    }
  });

  inputBar.querySelector('#file-btn')!.addEventListener('click', () => fileInput.click());

  fileInput.addEventListener('change', async () => {
    if (selectedTargets.length === 0) { alert('请先选择至少一个目标设备'); return; }
    if (fileInput.files && fileInput.files.length) {
      for (const f of Array.from(fileInput.files)) {
        const tasks = await api.sendFile(selectedTargets, f, encrypted);
        appendChat({ type: 'file', from: 'me', filename: f.name }, true);
        for (const t of tasks) api.subscribeTask(t.id, () => refreshTasks());
      }
    }
    fileInput.value = '';
  });

  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') inputBar.querySelector('#send-btn')!.dispatchEvent(new Event('click'));
  });

  inputBar.addEventListener('dragover', (e) => e.preventDefault());
  inputBar.addEventListener('drop', async (e) => {
    e.preventDefault();
    if (selectedTargets.length === 0) { alert('请先选择至少一个目标设备'); return; }
    if (e.dataTransfer?.files) {
      for (const f of Array.from(e.dataTransfer.files)) {
        const tasks = await api.sendFile(selectedTargets, f, encrypted);
        appendChat({ type: 'file', from: 'me', filename: f.name }, true);
        for (const t of tasks) api.subscribeTask(t.id, () => refreshTasks());
      }
    }
  });

  encryptBtn.addEventListener('click', () => {
    encrypted = !encrypted;
    encryptBtn.classList.toggle('active', encrypted);
  });

  api.subscribeInbox((_ev, data) => {
    try { appendChat(JSON.parse(data), false); } catch {}
  });

  async function refreshPeers() {
    try { renderDevices(await api.getPeers()); } catch {}
  }

  async function refreshTasks() {
    try { renderTasks(await api.getTasks()); } catch {}
  }

  refreshPeers();
  refreshTasks();
  setInterval(refreshPeers, 2000);
  setInterval(refreshTasks, 1000);
}

showConnectScreen();
