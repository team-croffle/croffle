import { existsSync, readFileSync, unlinkSync, writeFileSync } from 'node:fs';
import path from 'node:path';

import { AppCloseBehavior, AppEventType } from '@croffledev/common';
import type { AppSettings } from '@croffledev/croffle-types';
import type { BrowserWindow } from 'electron';
import { app, Menu, Tray, shell } from 'electron';
import type { UpdateInfo } from 'electron-updater';
import { autoUpdater } from 'electron-updater';

import icon from '../../../resources/logo-no-border.png?asset';
import { eventService } from '../event-bus/event-service';
import { t } from '../i18n';
import { logger } from '../logger';
import { settingService } from '../setting/setting-service';

class WindowService {
  private mainWindow: BrowserWindow | null = null;
  private tray: Tray | null = null;
  public isQuitting: boolean = false; // Service가 상태 관리

  // ======== Update 관련 변수 ========
  private readonly updateStatePath: string = path.join(
    app.getPath('userData'),
    'update-state.json',
  );
  private pendingUpdateInfo: UpdateInfo | null = null;
  private installOnQuit = false;

  constructor() {
    logger.info('WindowService', 'Initializing Service...');
    this.registerAppLifecycle();
    this.registerUpdateListeners();
    this.registerUpdateActionListeners();
  }

  private registerAppLifecycle(): void {
    app.on('before-quit', () => {
      this.isQuitting = true;
      this.tray?.destroy();
    });
  }

  public init(window: BrowserWindow): void {
    this.mainWindow = window;
    this.createTray();
    this.registerWindowEvents();
    this.registerSettingsListeners();
    logger.info('WindowService', 'Window initialized.');
  }

  private registerSettingsListeners(): void {
    eventService.on(AppEventType.SETTINGS_UPDATE, (payload) => {
      const settings = payload as AppSettings | undefined;
      if (!settings?.general?.language) {
        return;
      }
      this.refreshTrayMenu();
    });
  }

  /**
   * @deprecated Kept for the public `window.setCloseToTrayMode` API. Writes the
   * `general.closeBehavior` setting so the choice survives restarts.
   */
  public setCloseToTrayMode(enabled: boolean): void {
    const closeBehavior = enabled ? AppCloseBehavior.TRAY : AppCloseBehavior.QUIT;
    const updated = settingService.update({
      general: { ...settingService.get().general, closeBehavior },
    });
    eventService.emit(AppEventType.SETTINGS_UPDATE, updated);
    logger.info('WindowService', `Close behavior set to: ${closeBehavior}`);
  }

  /**
   * Close button behavior from settings. Old settings files (≤ 1.2.1) have no value;
   * the merge in SettingService fills in `ASK` so the first close prompts once.
   */
  private get closeBehavior(): AppCloseBehavior {
    return settingService.get().general.closeBehavior ?? AppCloseBehavior.ASK;
  }

  private registerWindowEvents(): void {
    if (!this.mainWindow) {
      return;
    }

    this.mainWindow.on('close', (event) => {
      if (this.isQuitting) {
        return true;
      }

      switch (this.closeBehavior) {
        case AppCloseBehavior.TRAY:
          event.preventDefault();
          this.hideWindow();
          return false;
        case AppCloseBehavior.QUIT:
          this.isQuitting = true;
          return true;
        default:
          // ASK: keep the window, let the renderer prompt. Its answer is saved as the
          // setting and the next close follows it.
          event.preventDefault();
          logger.info('WindowService', 'Close requested with closeBehavior=ask; prompting.');
          eventService.emit(AppEventType.WINDOW_CLOSE_REQUESTED);
          return false;
      }
    });

    this.mainWindow.webContents.setWindowOpenHandler(({ url }) => {
      try {
        const parsedUrl = new URL(url);
        const allowedProtocols = ['http:', 'https:'];

        if (allowedProtocols.includes(parsedUrl.protocol)) {
          shell.openExternal(url);
        }
      } catch (err) {
        logger.error('WindowService', `Invalid URL: ${url}`, err);
      }
      return { action: 'deny' };
    });
  }

  private createTray(): void {
    if (this.tray) {
      return;
    }

    try {
      this.tray = new Tray(icon);
      this.tray.setToolTip('CROFFLE');
      this.refreshTrayMenu();
      this.tray.on('double-click', () => this.showWindow());
    } catch (err) {
      logger.error('WindowService', 'Tray error:', err);
    }
  }

  private refreshTrayMenu(): void {
    if (!this.tray) {
      return;
    }

    const contextMenu = Menu.buildFromTemplate([
      { label: t('tray.openWindow'), click: () => this.showWindow() },
      { type: 'separator' },
      { label: t('tray.exitApp'), click: () => this.exitApp() },
    ]);
    this.tray.setContextMenu(contextMenu);
  }

  public showWindow(): void {
    this.mainWindow?.show();

    // Add app event emit
    eventService.emit(AppEventType.WINDOW_SHOW);
  }

  public hideWindow(): void {
    this.mainWindow?.hide();

    // Add app event emit
    eventService.emit(AppEventType.WINDOW_HIDE);
  }

  public exitApp(): void {
    this.isQuitting = true;
    app.quit();

    // Add app event emit
    eventService.emit(AppEventType.WINDOW_EXIT);
  }

  // ======== Update 관련 로직 ========

  private registerUpdateListeners(): void {
    if (!app.isPackaged) {
      return;
    }

    autoUpdater.autoDownload = false; // 체크 후 자동 다운로드 방지
    autoUpdater.autoInstallOnAppQuit = true; // 자동 업데이트 설치 허용

    autoUpdater.on('checking-for-update', () => logger.info('Updater', 'Checking...'));
    autoUpdater.on('update-available', (info) => {
      if (this.getSkippedVersion() === info.version) {
        logger.info('Updater', `Version ${info.version} is skipped by user.`);
        return;
      }

      const releaseNotes =
        typeof info.releaseNotes === 'string'
          ? info.releaseNotes
          : Array.isArray(info.releaseNotes)
            ? info.releaseNotes
                .map((note) => note.note ?? '')
                .filter(Boolean)
                .join('\n')
            : '';

      this.pendingUpdateInfo = info;
      eventService.emit(AppEventType.UPDATE_AVAILABLE, {
        version: info.version,
        releaseNotes,
      });
    });

    autoUpdater.on('update-not-available', () => {
      eventService.emit(AppEventType.UPDATE_NOT_AVAILABLE);
    });

    autoUpdater.on('download-progress', (progress) => {
      eventService.emit(AppEventType.UPDATE_DOWNLOAD_PROGRESS, {
        percent: progress.percent,
        transferred: progress.transferred,
      });
    });

    autoUpdater.on('update-downloaded', () => {
      eventService.emit(AppEventType.UPDATE_DOWNLOADED);

      if (this.installOnQuit) {
        // "나중에 적용" 선택 시 알아서 적용됨
        // 이벤트 리스너에서 설치 예약 처리함
        return;
      }
      // "지금 적용" 선택 시
      this.isQuitting = true;
      autoUpdater.quitAndInstall();
    });

    autoUpdater.on('error', (err) => {
      logger.error('Updater', 'Error:', err);
      eventService.emit(AppEventType.UPDATE_ERROR, err);
    });
  }

  // Update 관련 이벤트 리스너 등록
  private registerUpdateActionListeners(): void {
    eventService.on(AppEventType.UPDATE_DOWNLOAD_NOW, () => {
      this.installOnQuit = false;
      autoUpdater.downloadUpdate();
    });

    eventService.on(AppEventType.UPDATE_DOWNLOAD_LATER, () => {
      this.installOnQuit = true;
      autoUpdater.downloadUpdate();
    });

    eventService.on(AppEventType.UPDATE_SKIP_THIS_VERSION, () => {
      if (this.pendingUpdateInfo) {
        this.setSkippedVersion(this.pendingUpdateInfo.version);
        this.pendingUpdateInfo = null;
      }
    });
  }

  public async checkForUpdates(): Promise<void> {
    if (!app.isPackaged) {
      return;
    }
    if (!settingService.get().general.autoUpdate) {
      return;
    }
    await autoUpdater.checkForUpdates(); // Notify제거: Event를 통해 모달로 처리함
  }

  // Update를 스킵한 버전 정보 조회
  private getSkippedVersion(): string | null {
    try {
      if (!existsSync(this.updateStatePath)) {
        return null;
      }
      const updateState = JSON.parse(readFileSync(this.updateStatePath, 'utf8'));
      return updateState.version;
    } catch (err) {
      logger.error('WindowService', 'Failed to get skipped version:', err);
      return null;
    }
  }

  // Update를 스킵한 버전 정보 설정
  private setSkippedVersion(version: string | null): void {
    try {
      if (version === null) {
        if (existsSync(this.updateStatePath)) {
          unlinkSync(this.updateStatePath);
        }
        return;
      }
      const updateState = { version };
      writeFileSync(this.updateStatePath, JSON.stringify(updateState, null, 2), 'utf8');
    } catch (err) {
      logger.error('WindowService', 'Failed to set skipped version:', err);
    }
  }
}

export const windowService = new WindowService();
