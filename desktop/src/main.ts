import './styles.css';
import { DeviceList } from './components/DeviceList';
import { ChatStream } from './components/ChatStream';
import { TaskPanel } from './components/TaskPanel';
import { InputBar } from './components/InputBar';
import { SettingsPage } from './components/SettingsPage';
import * as api from './api';

async function init() {
  let base = 'http://127.0.0.1:8080';
  // @ts-ignore
  if (window.__TAURI__) {
    // @ts-ignore
    base = await window.__TAURI__.core.invoke('api_base');
  }
  api.setApiBase(base);

  const app = document.getElementById('app')!;

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

  let selectedTargets: string[] = [];
  const deviceList = new DeviceList(left, (sel) => { selectedTargets = sel; });
  const chat = new ChatStream(center);
  const taskPanel = new TaskPanel(right);

  new InputBar(inputBar,
    async (text, encrypted) => {
      if (selectedTargets.length === 0) { alert('请先选择至少一个目标设备'); return; }
      const tasks = await api.sendText(selectedTargets, text, encrypted);
      chat.append({ type: 'text', from: 'me', body: text }, true);
      refreshTasks();
      for (const t of tasks) {
        api.subscribeTask(t.id, () => refreshTasks());
      }
    },
    async (file, encrypted) => {
      if (selectedTargets.length === 0) { alert('请先选择至少一个目标设备'); return; }
      const tasks = await api.sendFile(selectedTargets, file, encrypted);
      chat.append({ type: 'file', from: 'me', filename: file.name }, true);
      refreshTasks();
      for (const t of tasks) {
        api.subscribeTask(t.id, () => refreshTasks());
      }
    }
  );

  api.subscribeInbox((_ev, data) => {
    try {
      const msg = JSON.parse(data);
      chat.append(msg, false);
    } catch {}
  });

  async function refreshPeers() {
    try { deviceList.update(await api.getPeers()); } catch {}
  }

  async function refreshTasks() {
    try { taskPanel.update(await api.getTasks()); } catch {}
  }

  // Settings button
  const settingsBtn = document.createElement('button');
  settingsBtn.className = 'btn';
  settingsBtn.textContent = '⚙ 设置';
  settingsBtn.style.cssText = 'margin-top:12px;width:100%;';
  left.appendChild(settingsBtn);

  let settingsEl: HTMLDivElement | null = null;
  settingsBtn.addEventListener('click', () => {
    app.style.display = 'none';
    settingsEl = document.createElement('div');
    document.body.appendChild(settingsEl);
    new SettingsPage(settingsEl, () => {
      settingsEl?.remove();
      app.style.display = 'grid';
    }).render();
  });

  refreshPeers();
  refreshTasks();
  setInterval(refreshPeers, 2000);
  setInterval(refreshTasks, 1000);
}

init();
