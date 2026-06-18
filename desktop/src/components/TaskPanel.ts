import type { Task } from '../types';

export class TaskPanel {
  private el: HTMLElement;
  private tasks: Task[] = [];

  constructor(el: HTMLElement) {
    this.el = el;
  }

  update(tasks: Task[]) {
    this.tasks = tasks;
    this.render();
  }

  private render() {
    this.el.innerHTML = '<div class="panel-title">传输任务</div>';
    if (this.tasks.length === 0) {
      const empty = document.createElement('div');
      empty.className = 'device-meta';
      empty.textContent = '暂无任务';
      this.el.appendChild(empty);
      return;
    }
    for (const t of this.tasks) {
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
      this.el.appendChild(card);
    }
  }
}
