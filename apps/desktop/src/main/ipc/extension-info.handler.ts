import { AppEventType } from '@croffledev/common';
import type { ExtensionInfo } from '@croffledev/common';
import { ipcMain, dialog } from 'electron';

import type { ExtensionInfoRow } from '../database/schema';
import { eventService } from '../event-bus/event-service';
import { extensionInfoService } from '../extension/info-service';
import { ExtensionInstallError } from '../extension/install-error';
import { extensionManager } from '../extension/manager';
import { readInstalledManifest } from '../extension/manifest';
import { extensionInfoMapper } from '../mapper/extension-info-mapper';
import { validateExtensionId } from '../utils/extension-validator';

function toExtensionInfo(row: ExtensionInfoRow): ExtensionInfo {
  return extensionInfoMapper.toInterface(row, readInstalledManifest(row.id));
}

export const registerExtensionInfoIpcHandlers = (): void => {
  ipcMain.handle('extensionInfo:getInstalledExtensions', async (): Promise<ExtensionInfo[]> => {
    const entity = await extensionInfoService.getInstalledExtensions();
    return entity.map(toExtensionInfo);
  });

  ipcMain.handle('extensionInfo:getEnabledExtensions', async (): Promise<ExtensionInfo[]> => {
    const entity = await extensionInfoService.getEnabledExtensions();
    return entity.map(toExtensionInfo);
  });

  ipcMain.handle(
    'extensionInfo:getExtensionByName',
    async (_, name: string): Promise<ExtensionInfo | null> => {
      validateExtensionId(name);
      const entity = await extensionInfoService.getExtensionByName(name);
      return entity ? toExtensionInfo(entity) : null;
    },
  );

  ipcMain.handle(
    'extensionInfo:installExtension',
    async (_, pluginData: Partial<ExtensionInfo>): Promise<ExtensionInfo> => {
      if (!pluginData.id) {
        throw new ExtensionInstallError('invalid-source', 'No GitHub repository given');
      }

      const entity = await extensionManager.installFromGitHub(pluginData.id);
      const dto = toExtensionInfo(entity);
      eventService.emit(AppEventType.EXTENSION_INFO_INSTALL, dto);
      return dto;
    },
  );

  ipcMain.handle('extensionInfo:installFromLocal', async (): Promise<ExtensionInfo | null> => {
    const result = await dialog.showOpenDialog({
      title: '로컬 확장 설치 (Zip 파일 선택)',
      filters: [{ name: 'Zip Files', extensions: ['zip'] }],
      properties: ['openFile'],
    });

    if (result.canceled || result.filePaths.length === 0) {
      return null;
    }

    const entity = await extensionManager.installFromLocalZip(result.filePaths[0]);
    const dto = toExtensionInfo(entity);
    eventService.emit(AppEventType.EXTENSION_INFO_INSTALL, dto);
    return dto;
  });

  ipcMain.handle(
    'extensionInfo:toggleExtension',
    async (_, name: string, enable: boolean): Promise<ExtensionInfo | null> => {
      validateExtensionId(name);
      if (typeof enable !== 'boolean') {
        throw new Error('[ExtensionInfo] Invalid enable value provided.');
      }

      const entity = await extensionInfoService.toggleExtension(name, enable);
      if (!entity) {
        return null;
      }

      const dto = toExtensionInfo(entity);
      eventService.emit(AppEventType.EXTENSION_INFO_TOGGLE, dto);
      return dto;
    },
  );

  ipcMain.handle('extensionInfo:uninstallExtension', async (_, name: string): Promise<boolean> => {
    validateExtensionId(name);
    const result = await extensionManager.uninstallExtension(name);
    if (result) {
      eventService.emit(AppEventType.EXTENSION_INFO_UNINSTALL, name);
    }
    return result;
  });
};
