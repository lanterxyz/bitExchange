import type { ClientConfig } from '../types';
import * as api from '../api';

export class SettingsPage {
  private el: HTMLElement;
  private onBack: () => void;

  constructor(el: HTMLElement, onBack: () => void) {
    this.el = el;
    this.onBack = onBack;
  }

  async render() {
    const cfg = await api.getConfig();
    this.el.innerHTML = `
      <div style="padding:20px;overflow:auto;height:100vh;background:linear-gradient(135deg,#0f0c29,#302b63);color:#fff;">
        <button class="btn" id="back-btn">← 返回</button>
        <h2 style="margin:16px 0;color:var(--accent-end);">设置</h2>

        <div class="panel" style="margin-bottom:14px;">
          <div class="panel-title">设备</div>
          <label style="display:block;margin-bottom:8px;color:var(--text-secondary);font-size:12px;">设备名称</label>
          <input class="input-field" id="device-name" value="${cfg.device_name || ''}" style="width:100%;" />
          <label style="display:block;margin:14px 0 8px;color:var(--text-secondary);font-size:12px;">默认保存根目录</label>
          <input class="input-field" id="save-root" value="${cfg.default_save_root || ''}" style="width:100%;" />
        </div>

        <div class="panel" style="margin-bottom:14px;">
          <div class="panel-title">网络</div>
          <label style="display:block;margin-bottom:8px;color:var(--text-secondary);font-size:12px;">信令服务器</label>
          <input class="input-field" id="server-url" value="${cfg.server_url || ''}" style="width:100%;" />
          <label style="display:flex;align-items:center;gap:8px;margin-top:14px;color:var(--text-primary);font-size:13px;">
            <input type="checkbox" id="relay-enabled" ${cfg.relay_enabled ? 'checked' : ''} /> 允许公网中继
          </label>
        </div>

        <div class="panel" style="margin-bottom:14px;">
          <div class="panel-title">安全</div>
          <label style="display:flex;align-items:center;gap:8px;color:var(--text-primary);font-size:13px;">
            <input type="checkbox" id="encrypt-default" ${cfg.encrypt_default ? 'checked' : ''} /> 默认加密传输
          </label>
        </div>

        <button class="btn btn-primary" id="save-btn">保存设置</button>
      </div>
    `;

    this.el.querySelector('#back-btn')!.addEventListener('click', this.onBack);
    this.el.querySelector('#save-btn')!.addEventListener('click', async () => {
      const updated: ClientConfig = {
        device_name: (this.el.querySelector('#device-name') as HTMLInputElement).value,
        default_save_root: (this.el.querySelector('#save-root') as HTMLInputElement).value,
        server_url: (this.el.querySelector('#server-url') as HTMLInputElement).value,
        relay_enabled: (this.el.querySelector('#relay-enabled') as HTMLInputElement).checked,
        relay_max_bytes: cfg.relay_max_bytes,
        listen_port: cfg.listen_port,
        encrypt_default: (this.el.querySelector('#encrypt-default') as HTMLInputElement).checked,
        trusted_devices_version: cfg.trusted_devices_version,
      };
      await api.putConfig(updated);
      alert('设置已保存');
      this.onBack();
    });
  }
}
