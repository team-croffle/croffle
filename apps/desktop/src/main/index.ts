import { join } from 'node:path';

import { electronApp, optimizer, is } from '@electron-toolkit/utils';
import { app, shell, BrowserWindow, protocol } from 'electron';
import { autoUpdater } from 'electron-updater';

import icon from '../../resources/logo-no-border.png?asset';
import { reminderScheduler } from './calendar/reminder-scheduler';
import { databaseManager } from './database';
import { registerAllIpcHandlers } from './ipc';
import { logger } from './logger';
import { runVersionMigrations } from './setting/migrations';
import {
  applyLoginItem,
  applyStartupPresentation,
  resolveStartupPresentation,
  shouldCheckForUpdates,
  LOGIN_HIDDEN_ARG,
} from './setting/setting-applies';
import { settingService } from './setting/setting-service';
import { closeSplash, showSplash } from './splash';
import { windowService } from './window/window-service';

// Must be called before app is ready
protocol.registerSchemesAsPrivileged([
  {
    scheme: 'extension',
    privileges: {
      standard: true,
      secure: true,
      corsEnabled: true,
      supportFetchAPI: true,
      allowServiceWorkers: false,
    },
  },
]);

const DEV_URL = 'http://localhost:5173';

function createWindow(): void {
  logger.info('Main', 'Creating main window');
  // Create the browser window.
  const mainWindow = new BrowserWindow({
    width: 1500,
    height: 800,
    minWidth: 960,
    minHeight: 600,
    frame: false,
    icon: icon,
    show: false,
    autoHideMenuBar: true,
    ...(process.platform === 'linux' ? { icon } : {}),
    webPreferences: {
      preload: join(__dirname, '../preload/index.js'),
      sandbox: true,
      contextIsolation: true,
      nodeIntegration: false,
    },
  });

  windowService.init(mainWindow);

  mainWindow.on('ready-to-show', () => {
    logger.info('Main', 'Main window is ready to show');
    const settings = settingService.get();
    applyStartupPresentation(settings);
    const shouldHideOnLogin = resolveStartupPresentation(settings).hidden;

    // WHY HERE: Splash window is closed after the main window is created.
    closeSplash();

    if (!shouldHideOnLogin) {
      logger.debug('Main', 'Showing main window');
      mainWindow.show();
    } else {
      logger.debug('Main', 'Main window is hidden on login');
    }
  });

  mainWindow.webContents.setWindowOpenHandler((details) => {
    shell.openExternal(details.url);
    return { action: 'deny' };
  });

  // HMR for renderer base on electron-vite cli.
  // Load the remote URL for development or the local html file for production.
  if (is.dev) {
    logger.debug('Main', `Loading DEV_URL: ${DEV_URL}`);
    mainWindow.loadURL(DEV_URL);
    mainWindow.webContents.openDevTools();
  } else {
    logger.debug('Main', 'Loading local HTML file for production');
    mainWindow.loadFile(join(__dirname, '../renderer/index.html'));
  }
}

// This method will be called when Electron has finished
// initialization and is ready to create browser windows.
// Some APIs can only be used after this event occurs.

const gotTheLock = app.requestSingleInstanceLock();

if (!gotTheLock) {
  logger.warn('Main', 'Another instance is already running. Quitting this instance.');
  app.quit();
} else {
  app.on('second-instance', (_event, argv) => {
    // 로그인 자동 시작(숨김 인자)으로 뜬 두 번째 인스턴스는 창을 띄우지 않는다.
    // 옛 로그인 항목이 남아 있어도 트레이만 유지되도록.
    if (argv.includes(LOGIN_HIDDEN_ARG)) {
      logger.info('Main', 'Second instance started hidden at login; keeping the window as is.');
      return;
    }
    logger.info('Main', 'Second instance requested. Focusing existing window.');
    // Someone tried to run a second instance, we should focus our window.
    windowService.showWindow();
  });

  app.whenReady().then(async (): Promise<void> => {
    logger.info('Main', 'Application is ready. Starting initialization...');
    // Set app user model id for windows
    electronApp.setAppUserModelId('kr.croffledev.croffle');

    // Default open or close DevTools by F12 in development
    // and ignore CommandOrControl + R in production.
    // see https://github.com/alex8088/electron-toolkit/tree/master/packages/utils
    app.on('browser-window-created', (_, window) => {
      optimizer.watchWindowShortcuts(window);
    });

    // 업데이트 마이그레이션(옛 로그인 항목 정리 등)은 설정이 준비된 직후, 창을 만들기 전에.
    await runVersionMigrations(settingService.get());

    // 로그인 항목은 AppUserModelId가 설정된 뒤 여기서 한 번만 등록한다 (설정 저장 시에는 applyPersisted).
    applyLoginItem(settingService.get());

    // Splash window is shown before the main window is created,
    // unless this is a hidden login start (tray only).
    if (resolveStartupPresentation(settingService.get()).hidden) {
      logger.info('Main', 'Hidden login start: skipping splash');
    } else {
      showSplash();
    }

    // IPC test
    try {
      await databaseManager.initialize();

      logger.info('Main', 'Registering IPC handlers');
      registerAllIpcHandlers();

      await reminderScheduler.start();

      createWindow();

      if (!is.dev && shouldCheckForUpdates(settingService.get())) {
        logger.info('Main', 'Checking for application updates');
        autoUpdater.checkForUpdatesAndNotify();
      }
    } catch (error) {
      logger.error('Main', 'Failed to initialize the application', error);
      reminderScheduler.stop();
      app.quit();
      closeSplash();
    }

    app.on('before-quit', () => {
      reminderScheduler.stop();
    });

    app.on('activate', function () {
      // On macOS it's common to re-create a window in the app when the
      // dock icon is clicked and there are no other windows open.
      if (BrowserWindow.getAllWindows().length === 0) {
        createWindow();
      }
    });
  });

  // Quit when all windows are closed, except on macOS. There, it's common
  // for applications and their menu bar to stay active until the user quits
  // explicitly with Cmd + Q.
  app.on('window-all-closed', () => {
    logger.info('Main', 'All windows closed');
    if (process.platform !== 'darwin') {
      app.quit();
    }
  });
}

// In this file you can include the rest of your app's specific main process
// code. You can also put them in separate files and require them here.
