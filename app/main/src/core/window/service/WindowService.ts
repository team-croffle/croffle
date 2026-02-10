import { app, BrowserWindow, Menu, Tray, nativeImage, shell } from 'electron';
import { autoUpdater } from 'electron-updater';
import path from 'path';
import { eventService } from '../../events/service/EventService';
import { AppEventType } from '../../../shared/enums';

class WindowService {
  private mainWindow: BrowserWindow | null = null;
  private tray: Tray | null = null;
  public isQuitting: boolean = false; // Service가 상태 관리
  private shouldCloseToTray: boolean = true; // 닫기 시 트레이로 최소화 여부

  constructor() {
    console.info('[WindowService] Initializing Service...');
    this.registerAppLifecycle();
    this.registerUpdateListeners();
  }

  private registerAppLifecycle(): void {
    app.on('before-quit', () => {
      this.isQuitting = true;
      this.tray?.destroy();
    });
  }

  private registerUpdateListeners(): void {
    if (!app.isPackaged) return;

    autoUpdater.on('checking-for-update', () => console.info('[Updater] Checking...'));
    autoUpdater.on('update-available', () => console.info('[Updater] Available.'));
    autoUpdater.on('update-downloaded', () => console.info('[Updater] Downloaded.'));
    autoUpdater.on('error', (err) => console.error('[Updater] Error:', err));
  }

  public init(window: BrowserWindow): void {
    this.mainWindow = window;
    this.createTray();
    this.registerWindowEvents();
    console.info('[WindowService] Window initialized.');
  }

  public setCloseToTrayMode(enabled: boolean): void {
    this.shouldCloseToTray = enabled;
    console.info(`[WindowService] Close-to-Tray mode set to: ${enabled}`);
  }

  private registerWindowEvents(): void {
    if (!this.mainWindow) return;

    this.mainWindow.on('close', (event) => {
      if (this.isQuitting) return true;

      if (this.shouldCloseToTray) {
        event.preventDefault();
        this.hideWindow();
      } else {
        this.isQuitting = true;
        return true;
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
        console.error('[WindowService] Invalid URL:', url, err);
      }
      return { action: 'deny' };
    });
  }

  private createTray(): void {
    if (this.tray) return;

    try {
      const iconPath = app.isPackaged
        ? path.join(process.resourcesPath, 'icons/Logo2OnlyNoBorderIcon.png')
        : path.join(app.getAppPath(), '../../icons/Logo2OnlyNoBorderIcon.png');

      const icon = nativeImage.createFromPath(iconPath);
      if (icon.isEmpty()) {
        console.error('[WindowService] Failed to load tray icon at:', iconPath);
      }
      this.tray = new Tray(icon);
      this.tray.setToolTip('CROFFLE');

      const contextMenu = Menu.buildFromTemplate([
        { label: '열기', click: () => this.showWindow() },
        { type: 'separator' },
        { label: '종료', click: () => this.exitApp() },
      ]);

      this.tray.setContextMenu(contextMenu);
      this.tray.on('double-click', () => this.showWindow());
    } catch (err) {
      console.error('[WindowService] Tray error:', err);
    }
  }

  public showWindow(): void {
    this.mainWindow?.show();

    // Add app event emit
    eventService.emit(AppEventType.WINDOW_SHOW, this.mainWindow);
  }

  public hideWindow(): void {
    this.mainWindow?.hide();

    // Add app event emit
    eventService.emit(AppEventType.WINDOW_HIDE, this.mainWindow);
  }

  public exitApp(): void {
    this.isQuitting = true;
    app.quit();

    // Add app event emit
    eventService.emit(AppEventType.WINDOW_EXIT, this.mainWindow);
  }

  public async checkForUpdates(): Promise<void> {
    if (!app.isPackaged) return;
    await autoUpdater.checkForUpdatesAndNotify();

    eventService.emit(AppEventType.WINDOW_CHECK_FOR_UPDATES, this.mainWindow);
  }
}

export const windowService = new WindowService();
