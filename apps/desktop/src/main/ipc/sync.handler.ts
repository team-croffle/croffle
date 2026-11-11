import type { SyncActionResult, SyncPasswordSignIn } from '@croffledev/common';
import { ipcMain } from 'electron';

import { signInWithPassword, signInWithToken, signOut } from '../sync/auth';
import { InvalidServerUrlError, setAllowInsecureTls, setServer } from '../sync/sync-config';
import { syncService } from '../sync/sync-service';
import { SyncError } from '../sync/transport';
import { validateSender } from './validate-sender';

function failure(error: unknown): SyncActionResult {
  if (error instanceof SyncError) {
    return { ok: false, kind: error.kind, code: error.code, message: error.message };
  }
  if (error instanceof InvalidServerUrlError) {
    return { ok: false, kind: 'invalid', message: error.message };
  }
  return {
    ok: false,
    kind: 'server',
    message: error instanceof Error ? error.message : String(error),
  };
}

/** `croffleInternal.sync`. Every channel checks that the app's own window is calling. */
export function registerSyncIpcHandlers() {
  ipcMain.handle('sync:getStatus', (event) => {
    validateSender(event);
    return syncService.getStatus();
  });

  ipcMain.handle('sync:setServer', (event, url: string): SyncActionResult => {
    validateSender(event);
    try {
      setServer(url);
      syncService.bindingChanged();
      return { ok: true };
    } catch (error) {
      return failure(error);
    }
  });

  ipcMain.handle('sync:setAllowInsecureTls', (event, allow: boolean) => {
    validateSender(event);
    setAllowInsecureTls(Boolean(allow));
    syncService.bindingChanged();
  });

  ipcMain.handle(
    'sync:signInWithPassword',
    async (event, input: SyncPasswordSignIn): Promise<SyncActionResult> => {
      validateSender(event);
      try {
        const { accountId } = await signInWithPassword(input);
        syncService.bindingChanged();
        return { ok: true, accountId };
      } catch (error) {
        return failure(error);
      }
    },
  );

  ipcMain.handle(
    'sync:signInWithToken',
    async (event, token: string): Promise<SyncActionResult> => {
      validateSender(event);
      try {
        const { accountId } = await signInWithToken(token);
        syncService.bindingChanged();
        return { ok: true, accountId };
      } catch (error) {
        return failure(error);
      }
    },
  );

  ipcMain.handle('sync:signOut', (event) => {
    validateSender(event);
    signOut();
    syncService.bindingChanged();
  });

  ipcMain.handle('sync:setEnabled', (event, enabled: boolean) => {
    validateSender(event);
    syncService.setEnabled(Boolean(enabled));
  });

  ipcMain.handle('sync:syncNow', async (event) => {
    validateSender(event);
    await syncService.syncNow();
  });
}
