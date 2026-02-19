import { ipcMain } from 'electron';
import { searchService, SearchQuery } from '../core/search/service/searchService';

export function registerSearchIpcHandlers() {
  ipcMain.handle('search:search', async (_event, query: SearchQuery) => {
    return await searchService.searchSchedules(query);
  });
}
