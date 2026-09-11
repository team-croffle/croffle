import { existsSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';

import { is } from '@electron-toolkit/utils';
import { app } from 'electron';

import { logger } from '../logger';

/**
 * main 전용 앱 상태. 공개 `AppSettings`(packages/types)에 넣지 않는 내부 값만 둔다.
 * 현재는 마지막으로 실행된 앱 버전만 기록해 업데이트 마이그레이션의 기준으로 쓴다.
 */
export type AppState = {
  /** 직전 실행의 `app.getVersion()`. 파일이 없으면 1.1.0 이하로 간주한다. */
  lastRunVersion: string | null;
};

const EMPTY_STATE: AppState = { lastRunVersion: null };

function resolvePath(): string {
  return is.dev
    ? path.join(process.cwd(), 'dev/app-state.json')
    : path.join(app.getPath('userData'), 'app-state.json');
}

export function readAppState(): AppState {
  const filePath = resolvePath();
  try {
    if (!existsSync(filePath)) {
      return { ...EMPTY_STATE };
    }
    const parsed = JSON.parse(readFileSync(filePath, 'utf-8')) as Partial<AppState>;
    return {
      lastRunVersion: typeof parsed.lastRunVersion === 'string' ? parsed.lastRunVersion : null,
    };
  } catch (err) {
    logger.warn('AppState', `Failed to read ${filePath}; treating as first run.`, err);
    return { ...EMPTY_STATE };
  }
}

export function writeAppState(state: AppState): void {
  const filePath = resolvePath();
  try {
    writeFileSync(filePath, JSON.stringify(state, null, 2), 'utf-8');
  } catch (err) {
    // 쓰기 실패는 치명적이지 않다. 다음 실행에 마이그레이션이 다시 돌지만 멱등이다.
    logger.warn('AppState', `Failed to write ${filePath}.`, err);
  }
}
