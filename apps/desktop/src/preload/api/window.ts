import type { WindowApi } from '@croffledev/croffle-types';
import { ipcRenderer } from 'electron';

export const windowApi = {
  minimize: () => ipcRenderer.invoke('window:minimize'),
  maximize: () => ipcRenderer.invoke('window:maximize'),
  close: () => ipcRenderer.invoke('window:close'),
  exitApp: () => ipcRenderer.invoke('window:exitApp'),
  checkForUpdates: () => ipcRenderer.invoke('window:checkForUpdates'),
} satisfies WindowApi;
