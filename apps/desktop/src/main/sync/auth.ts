import { hostname } from 'node:os';

import { ERROR_CODES, type LoginResponse, type WhoAmIResponse } from '@croffledev/common';

import { getClientId } from './client-id';
import { getServer, resetBinding, saveToken, updateServerState } from './sync-config';
import { checkServer, configureTls, request, SyncError } from './transport';

export type SignInResult = { accountId: string; teamId: string | null };

export type PasswordSignIn = { username: string; password: string; otp?: string };

function requireServerUrl(): string {
  const server = getServer();
  if (!server) {
    throw new SyncError('server', 'No sync server is set');
  }
  configureTls(server.url, server.allowInsecureTls);
  return server.url;
}

function bind(token: string, accountId: string, teamId: string | null | undefined): SignInResult {
  // Another account's cursors and snapshot do not apply to this one.
  const previous = getServer()?.accountId;
  if (previous && previous !== accountId) {
    resetBinding();
  }
  saveToken(token);
  updateServerState({ accountId, teamId: teamId ?? null, lastErrorKind: null, lastError: null });
  return { accountId, teamId: teamId ?? null };
}

/**
 * Username and password → a device token. The password is sent once and never stored, which is
 * why an expired token can only be replaced by signing in again. A 401 with `otp_required`
 * surfaces as that code so the settings screen can ask for the one-time code and retry.
 */
export async function signInWithPassword(input: PasswordSignIn): Promise<SignInResult> {
  const url = requireServerUrl();
  await checkServer(url);
  const response = await request<LoginResponse>(url, 'POST', '/auth/login', {
    retry: false,
    body: {
      username: input.username,
      password: input.password,
      ...(input.otp ? { otp: input.otp } : {}),
      clientId: getClientId(),
      deviceName: hostname(),
    },
  });
  return bind(response.token, response.accountId, response.teamId);
}

/** A pasted token is checked with `/auth/whoami` first, so a typo fails here, not silently later. */
export async function signInWithToken(token: string): Promise<SignInResult> {
  const trimmed = token.trim();
  if (!trimmed) {
    throw new SyncError('auth', 'The token is empty', 0, ERROR_CODES.unauthorized);
  }
  const url = requireServerUrl();
  await checkServer(url);
  const response = await request<WhoAmIResponse>(url, 'POST', '/auth/whoami', {
    token: trimmed,
    retry: false,
  });
  return bind(trimmed, response.accountId, response.teamId);
}

/** Forgets the token and the account binding. Queued local changes stay for the next sign-in. */
export function signOut(): void {
  resetBinding();
}
