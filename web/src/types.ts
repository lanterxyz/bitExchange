export interface Peer {
  device_id: string;
  device_name: string;
  fingerprint: string;
  path: string;
  rate: number;
  status: string;
}

export interface Task {
  id: string;
  kind: string;
  target: string;
  payload: string;
  status: string;
  progress: number;
  rate: number;
  path: string;
  error?: string;
}

export interface ClientConfig {
  device_name: string;
  default_save_root: string;
  server_url: string;
  relay_enabled: boolean;
  relay_max_bytes: number;
  listen_port: number;
  encrypt_default: boolean;
  trusted_devices_version: number;
}

export interface ChatLine {
  type: 'text' | 'file';
  from: string;
  body?: string;
  filename?: string;
  savedPath?: string;
}
