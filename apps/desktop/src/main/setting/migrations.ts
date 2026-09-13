import type { AppSettings } from '@croffledev/croffle-types';
import { app } from 'electron';

import { logger } from '../logger';
import { readAppState, writeAppState } from './app-state';
import { cleanupLoginItems } from './login-items';
import { applyLoginItem } from './setting-applies';

type Migration = {
  /** 이 버전 **이하**에서 올라온 경우 실행 (직전 버전 미기록이면 무조건 실행) */
  upTo: string;
  name: string;
  run: (settings: AppSettings) => Promise<void>;
};

/**
 * 버전 마이그레이션 목록. 실행 순서 = 배열 순서.
 * 새 항목은 `upTo`를 "고쳐야 하는 마지막 버전"으로 두고 뒤에 추가한다.
 */
const MIGRATIONS: Migration[] = [
  {
    // 1.1.1-rc.4까지: 이전 버전 인스턴스가 시작 프로그램에 남거나, AppUserModelId 설정 전 등록으로
    // 같은 exe가 electron.app.croffle / kr.croffledev.croffle 두 이름으로 등록됨 (rc.1~rc.4도 해당).
    // Croffle 항목을 전부 지우고 현재 설정대로 재등록한다 (off면 등록 안 함).
    upTo: '1.1.1-rc.4',
    name: 'cleanup-login-items',
    run: async (settings) => {
      const result = await cleanupLoginItems();
      applyLoginItem(settings);
      logger.info(
        'Migrations',
        `Login items re-registered (startOnSystemBoot=${settings.general.startOnSystemBoot}, startMinimized=${settings.general.startMinimized}); removed ${result.removed.length} stale item(s)`,
      );
    },
  },
];

const parseVersion = (v: string) => {
  const [core = '', pre] = v.replace(/^v/, '').split('-', 2);
  const nums = core.split('.').map((n) => Number.parseInt(n, 10) || 0);
  while (nums.length < 3) {
    nums.push(0);
  }
  return { nums, pre: pre ?? null };
};

/** `a`가 `b`보다 작으면 음수, 같으면 0, 크면 양수. `1.1.1-rc.2 < 1.1.1`. */
export function compareVersions(a: string, b: string): number {
  const pa = parseVersion(a);
  const pb = parseVersion(b);
  for (let i = 0; i < 3; i += 1) {
    const diff = pa.nums[i]! - pb.nums[i]!;
    if (diff !== 0) {
      return diff;
    }
  }
  if (pa.pre === pb.pre) {
    return 0;
  }
  if (pa.pre === null) {
    return 1;
  }
  if (pb.pre === null) {
    return -1;
  }
  return pa.pre.localeCompare(pb.pre, undefined, { numeric: true });
}

/**
 * 직전 실행 버전을 읽어 필요한 마이그레이션을 돌리고 현재 버전을 기록한다.
 * 개발 빌드는 로그인 항목 등록이 없으므로 기록만 하고 건너뛴다.
 */
export async function runVersionMigrations(settings: AppSettings): Promise<void> {
  const current = app.getVersion();
  const { lastRunVersion } = readAppState();

  if (lastRunVersion === current) {
    return;
  }

  if (!app.isPackaged) {
    logger.debug('Migrations', `Dev build: skip migrations (${lastRunVersion} → ${current})`);
    writeAppState({ lastRunVersion: current });
    return;
  }

  for (const migration of MIGRATIONS) {
    const shouldRun =
      lastRunVersion === null || compareVersions(lastRunVersion, migration.upTo) <= 0;
    if (!shouldRun) {
      continue;
    }
    logger.info(
      'Migrations',
      `Running "${migration.name}" (last=${lastRunVersion ?? 'unknown'}, upTo=${migration.upTo}, current=${current})`,
    );
    try {
      await migration.run(settings);
    } catch (err) {
      // 마이그레이션 실패로 앱 시작을 막지 않는다. 버전을 기록하지 않아 다음 실행에 재시도한다.
      logger.error('Migrations', `"${migration.name}" failed; will retry on next launch`, err);
      return;
    }
  }

  writeAppState({ lastRunVersion: current });
}
