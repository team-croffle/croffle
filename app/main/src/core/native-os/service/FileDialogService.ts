import { app, dialog } from 'electron';

export class FileDialogService {
  async openJsonFile(options?: { title?: string }): Promise<string | null> {
    const result = await dialog.showOpenDialog({
      title: options?.title ?? 'Import',
      defaultPath: app.getPath('documents'),
      properties: ['openFile'],
      filters: [{ name: 'JSON', extensions: ['json'] }],
    });

    if (result.canceled || result.filePaths.length === 0) return null;
    return result.filePaths[0];
  }

  async saveJsonFile(options?: {
    title?: string;
    defaultFileName?: string;
  }): Promise<string | null> {
    const result = await dialog.showSaveDialog({
      title: options?.title ?? 'Export',
      defaultPath: `${app.getPath('documents')}/${options?.defaultFileName ?? 'schedules.json'}`,
      filters: [{ name: 'JSON', extensions: ['json'] }],
    });

    if (result.canceled || !result.filePath) return null;
    return result.filePath;
  }
}

export const fileDialogService = new FileDialogService();
