import { NOTIFICATIONS_CHANNEL, type InternalNotificationsApi } from '@croffledev/common';
import { ipcRenderer } from 'electron';

export const notificationsApi = {
  list: () => ipcRenderer.invoke('notifications:list'),
  markAllRead: () => ipcRenderer.invoke('notifications:markAllRead'),
  onChanged: (callback) => {
    const subscription = () => callback();
    ipcRenderer.on(NOTIFICATIONS_CHANNEL, subscription);
    return () => {
      ipcRenderer.off(NOTIFICATIONS_CHANNEL, subscription);
    };
  },
} satisfies InternalNotificationsApi;
