import { ipcRenderer } from 'electron';
import type { SearchQuery, Schedule } from 'croffle';

export const searchApi = {
  search: (query: SearchQuery): Promise<Schedule[]> => ipcRenderer.invoke('search:search', query),
};
