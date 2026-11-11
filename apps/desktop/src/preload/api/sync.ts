import { SYNC_STATUS_CHANNEL, type InternalSyncApi, type SyncStatus } from '@croffledev/common';
import { ipcRenderer, type IpcRendererEvent } from 'electron';

export const syncApi = {
  getStatus: () => ipcRenderer.invoke('sync:getStatus'),
  onStatus: (callback) => {
    const subscription = (_event: IpcRendererEvent, status: SyncStatus) => callback(status);
    ipcRenderer.on(SYNC_STATUS_CHANNEL, subscription);
    return () => {
      ipcRenderer.off(SYNC_STATUS_CHANNEL, subscription);
    };
  },
  setServer: (url) => ipcRenderer.invoke('sync:setServer', url),
  setAllowInsecureTls: (allow) => ipcRenderer.invoke('sync:setAllowInsecureTls', allow),
  signInWithPassword: (input) => ipcRenderer.invoke('sync:signInWithPassword', input),
  signInWithToken: (token) => ipcRenderer.invoke('sync:signInWithToken', token),
  signOut: () => ipcRenderer.invoke('sync:signOut'),
  setEnabled: (enabled) => ipcRenderer.invoke('sync:setEnabled', enabled),
  syncNow: () => ipcRenderer.invoke('sync:syncNow'),
} satisfies InternalSyncApi;
