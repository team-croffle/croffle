import { ipcMain } from 'electron';
import { searchService } from '../core/search/service/searchService';
import { SearchQuery } from 'croffle';

export function registerSearchIpcHandlers() {
  ipcMain.handle('search:search', async (_event, query: SearchQuery) => {
    return await searchService.searchSchedules(query);
  });
}
