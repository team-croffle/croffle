import { ipcMain } from 'electron';
import { HttpResponse } from 'croffle';
import { httpService } from '../core/http/service/HttpService';
import { eventService } from '../core/events/service/EventService';
import { AppEventType } from '../shared/enums';

export const registerHttpIpcHandlers = (): void => {
  ipcMain.handle(
    'http:get',
    async (
      _,
      url: string,
      params?: Record<string, string>,
      headers?: Record<string, string>
    ): Promise<HttpResponse> => {
      if (!url || typeof url !== 'string') {
        throw new Error('[HTTP] Invalid URL.');
      }

      // Add app event emit
      eventService.emit(AppEventType.HTTP_SERVICE_GET, url, params, headers);

      return httpService.get(url, params, headers);
    }
  );

  ipcMain.handle(
    'http:post',
    async (
      _,
      url: string,
      body?: unknown,
      headers?: Record<string, string>
    ): Promise<HttpResponse> => {
      if (!url || typeof url !== 'string') {
        throw new Error('[HTTP] Invalid URL.');
      }

      // Add app event emit
      eventService.emit(AppEventType.HTTP_SERVICE_POST, url, body, headers);

      return httpService.post(url, body, headers);
    }
  );
};
