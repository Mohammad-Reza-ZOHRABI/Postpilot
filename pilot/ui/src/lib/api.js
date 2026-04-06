const BASE = '/api/v1';

async function request(method, path, body = null) {
  const opts = {
    method,
    headers: {},
    credentials: 'same-origin',
  };
  if (body) {
    opts.headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(body);
  }
  const res = await fetch(BASE + path, opts);
  if (res.status === 401) {
    window.location.hash = '#/login';
    throw new Error('Unauthorized');
  }
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
  return data;
}

export const api = {
  // Auth
  check: () => request('GET', '/auth/check'),
  login: (email, password, totp_code) => request('POST', '/auth/login', { email, password, totp_code }),
  logout: () => request('POST', '/auth/logout'),
  setupStep1: (email, password) => request('POST', '/auth/setup', { step: 1, email, password }),
  setupStep2: (email, password, totp_secret, totp_code) =>
    request('POST', '/auth/setup', { step: 2, email, password, totp_secret, totp_code }),

  // Dashboard
  dashboard: () => request('GET', '/dashboard'),

  // Settings
  getSettings: () => request('GET', '/settings'),
  saveSettings: (settings) => request('POST', '/settings', settings),

  // API Keys
  getKeys: () => request('GET', '/keys'),
  createKey: (name, permissions, rate_limit) => request('POST', '/keys', { name, permissions, rate_limit }),
  revokeKey: (id) => request('POST', `/keys/${id}/revoke`),

  // DNS
  checkDns: (domain) => request('POST', '/dns/check', { domain }),

  // Users
  listUsers: () => request('GET', '/users'),
  createUser: (email, password, role) => request('POST', '/users', { email, password, role }),
  deleteUser: (id) => request('POST', `/users/${id}/delete`),
  updateUserRole: (id, role) => request('POST', `/users/${id}/role`, { role }),

  // Send (public API)
  health: () => request('GET', '/health'),
};
