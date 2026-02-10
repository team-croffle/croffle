import { contextBridge } from 'electron';
import { windowApi } from './api/window.api';
import { tagApi } from './api/tag.api';
import { pluginInfoApi } from './api/pluginInfo.api';
import { settingsApi } from './api/settings.api';
import { scheduleApi } from './api/schedule.api';
import { osApi } from './api/os.api';
import { httpApi } from './api/http.api';
import { pluginStorageApi } from './api/pluginStorage.api';
import { eventApi } from './api/event.api';

const croffleApi = {
  base: {
    windows: windowApi,
    tags: tagApi,
    schedules: scheduleApi,
    pluginInfo: pluginInfoApi,
    settings: settingsApi,
  },
  app: {
    os: osApi,
    http: httpApi,
    storage: pluginStorageApi,
    event: eventApi,
  },
};

contextBridge.exposeInMainWorld('croffle', croffleApi);
