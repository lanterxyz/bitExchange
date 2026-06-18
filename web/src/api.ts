import type { Peer, Task, ClientConfig } from './types';

let apiBase = '';

export function setApiBase(base: string) { apiBase = base; }
export function getApiBase() { return apiBase; }

export async function getPeers(): Promise<Peer[]> {
  const resp = await fetch(`${apiBase}/api/peers`);
  return resp.json();
}

export async function getConfig(): Promise<ClientConfig> {
  const resp = await fetch(`${apiBase}/api/config`);
  return resp.json();
}

export async function putConfig(cfg: ClientConfig): Promise<ClientConfig> {
  const resp = await fetch(`${apiBase}/api/config`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(cfg),
  });
  return resp.json();
}

export async function getTasks(): Promise<Task[]> {
  const resp = await fetch(`${apiBase}/api/tasks`);
  const data = await resp.json();
  return data.tasks || [];
}

export async function sendText(targets: string[], body: string, encrypted: boolean): Promise<Task[]> {
  const resp = await fetch(`${apiBase}/api/send/text`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ to_device_ids: targets, body, encrypted }),
  });
  const data = await resp.json();
  return data.tasks || [];
}

export async function sendFile(targets: string[], file: File, encrypted: boolean): Promise<Task[]> {
  const form = new FormData();
  targets.forEach(t => form.append('to_device_ids', t));
  form.append('file', file);
  const resp = await fetch(`${apiBase}/api/send/file`, { method: 'POST', body: form });
  const data = await resp.json();
  return data.tasks || [];
}

export function subscribeTask(taskId: string, onEvent: (event: string, data: string) => void): () => void {
  const es = new EventSource(`${apiBase}/api/tasks/${taskId}/events`);
  es.addEventListener('progress', (e: MessageEvent) => onEvent('progress', e.data));
  es.addEventListener('status', (e: MessageEvent) => onEvent('status', e.data));
  return () => es.close();
}

export function subscribeInbox(onEvent: (event: string, data: string) => void): () => void {
  const es = new EventSource(`${apiBase}/api/inbox/events`);
  es.onmessage = (e) => onEvent('message', e.data);
  return () => es.close();
}
