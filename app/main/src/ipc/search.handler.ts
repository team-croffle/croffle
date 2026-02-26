import { ipcMain } from 'electron';
import { searchService } from '../core/search/service/searchService';

export function registerSearchIpcHandlers() {
  ipcMain.handle('search:search', async (_event, query) => {
    console.log('main query raw:', query.text);
    return searchService.searchSchedules(query);
  });
}
