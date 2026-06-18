import type { Peer } from '../types';

export class DeviceList {
  private el: HTMLElement;
  private peers: Peer[] = [];
  private selected = new Set<string>();
  private onChange: (selected: string[]) => void;

  constructor(el: HTMLElement, onChange: (selected: string[]) => void) {
    this.el = el;
    this.onChange = onChange;
  }

  update(peers: Peer[]) {
    this.peers = peers;
    this.render();
  }

  private render() {
    this.el.innerHTML = '<div class="panel-title">在线设备</div>';
    if (this.peers.length === 0) {
      const empty = document.createElement('div');
      empty.className = 'device-meta';
      empty.textContent = '暂无在线设备';
      this.el.appendChild(empty);
      return;
    }
    for (const p of this.peers) {
      const card = document.createElement('div');
      card.className = 'device-card';
      if (this.selected.has(p.device_id)) card.classList.add('selected');
      card.innerHTML = `
        <div class="device-name">${p.device_name}</div>
        <div class="device-meta">${p.path} · ${p.rate.toFixed(1)}MB/s</div>
      `;
      card.onclick = () => {
        if (this.selected.has(p.device_id)) {
          this.selected.delete(p.device_id);
        } else {
          this.selected.add(p.device_id);
        }
        this.render();
        this.onChange(Array.from(this.selected));
      };
      this.el.appendChild(card);
    }
  }

  getSelected(): string[] {
    return Array.from(this.selected);
  }
}
