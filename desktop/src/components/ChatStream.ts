import type { ChatLine } from '../types';

export class ChatStream {
  private el: HTMLElement;

  constructor(el: HTMLElement) {
    this.el = el;
    this.el.className = 'chat-stream';
  }

  append(line: ChatLine, outgoing: boolean) {
    const bubble = document.createElement('div');
    bubble.className = outgoing ? 'bubble-out' : 'bubble-in';
    if (line.type === 'text') {
      bubble.textContent = line.body || '';
    } else {
      bubble.innerHTML = `<div style="font-size:10px;color:var(--text-muted);margin-bottom:4px;">${line.from} · 文件</div>📄 ${line.filename}`;
    }
    this.el.appendChild(bubble);
    this.el.scrollTop = this.el.scrollHeight;
  }
}
