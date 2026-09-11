import type { AppSettings } from '@croffledev/croffle-types';
import { app } from 'electron';

import { logger } from '../logger';
import { windowService } from '../window/window-service';

export const LOGIN_HIDDEN_ARG = '--croffle-start-hidden';
export const STARTUP_ARG = '--startup';

/** 로그인 항목(시작 프로그램) 등록 상태 반영 */
export function applyLoginItem(settings: AppSettings): void {
  if (!app.isPackaged) {
    return;
  }

  const { startOnSystemBoot, startMinimized } = settings.general;
  const args: string[] = [];

  if (startOnSystemBoot) {
    args.push(STARTUP_ARG);
    if (startMinimized) {
      args.push(LOGIN_HIDDEN_ARG);
    }
  }

  app.setLoginItemSettings({
    openAtLogin: startOnSystemBoot,
    openAsHidden: startMinimized,
    path: process.execPath,
    args: args,
  });
}

export function applyPersisted(settings: AppSettings): void {
  applyLoginItem(settings);
}

export type StartupPresentation = {
  wasOpenedAtLogin: boolean;
  /** true면 스플래시도 메인 창도 띄우지 않고 트레이만 */
  hidden: boolean;
};

/**
 * 이번 실행이 로그인 자동 시작인지, 창을 숨겨야 하는지 한 번만 계산한다.
 * Windows/Linux는 `wasOpenedAtLogin`이 없으므로 argv(`--startup`, `--croffle-start-hidden`)로 판단.
 * `showSplash()` 전에 호출해 숨김 시작이면 스플래시도 생략한다.
 */
export function resolveStartupPresentation(settings: AppSettings): StartupPresentation {
  const loginSettings = app.getLoginItemSettings();
  const wasOpenedAtLogin = loginSettings.wasOpenedAtLogin || process.argv.includes(STARTUP_ARG);
  const wasOpenedAsHidden = process.argv.includes(LOGIN_HIDDEN_ARG);
  const { startMinimized, startOnSystemBoot } = settings.general;
  const hidden = wasOpenedAtLogin && (startMinimized || wasOpenedAsHidden);

  logger.debug(
    'Startup',
    `presentation: wasOpenedAtLogin=${wasOpenedAtLogin} hiddenArg=${wasOpenedAsHidden} startOnSystemBoot=${startOnSystemBoot} startMinimized=${startMinimized} → hidden=${hidden}`,
  );

  return { wasOpenedAtLogin, hidden };
}

/** 앱 최초 표시 시(로그인 시작 포함) 창 표시 결정 */
export function applyStartupPresentation(settings: AppSettings): void {
  applyLoginItem(settings);

  if (resolveStartupPresentation(settings).hidden) {
    windowService.hideWindow();
  }
}

export function shouldCheckForUpdates(settings: AppSettings): boolean {
  return settings.general.autoUpdate;
}
