/*
 * Session service: it signs the user in and is the only place that reads or
 * writes the stored session, so no component touches browser storage.
 */
import { request } from './api_client';

const STORAGE_KEY = 'workshop.session';

export interface Session {
  token: string;
  expiresAt: string;
  userId: string;
  username: string;
  fullName: string;
  role: 'ADMINISTRATOR' | 'TECHNICIAN';
  requiresPasswordChange?: boolean;
}

export async function signIn(username: string, password: string): Promise<Session> {
  return request<Session>('/session', { method: 'POST', body: { username, password } });
}

export async function changePassword(
  token: string,
  currentPassword: string,
  newPassword: string,
): Promise<void> {
  return request<void>('/session/change-password', {
    method: 'POST',
    body: { currentPassword, newPassword },
    token,
  });
}

export function readStoredSession(): Session | null {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return null;
    }
    const stored = JSON.parse(raw) as Session;
    if (!stored.token || new Date(stored.expiresAt).getTime() <= Date.now()) {
      window.localStorage.removeItem(STORAGE_KEY);
      return null;
    }
    return stored;
  } catch {
    return null;
  }
}

export function storeSession(session: Session): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
  } catch {
    // A browser with storage disabled still works for the current tab.
  }
}

export function clearStoredSession(): void {
  try {
    window.localStorage.removeItem(STORAGE_KEY);
  } catch {
    // Nothing to clean when storage is unavailable.
  }
}
