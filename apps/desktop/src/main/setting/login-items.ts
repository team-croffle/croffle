import { execFile } from 'node:child_process';
import { existsSync, readdirSync, unlinkSync } from 'node:fs';
import { homedir } from 'node:os';
import path from 'node:path';
import { promisify } from 'node:util';

import { app } from 'electron';

import { logger } from '../logger';
import { LOGIN_ITEM_NAME } from './setting-applies';

const execFileAsync = promisify(execFile);

const MATCH = /croffle/i;
const RUN_KEY = String.raw`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`;
const STARTUP_APPROVED_KEY = String.raw`HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`;

export type LoginItemCleanupResult = {
  removed: string[];
  errors: string[];
};

/**
 * 시작 프로그램(로그인 항목)에서 Croffle을 가리키는 항목을 **전부** 제거한다.
 *
 * WHY: `app.setLoginItemSettings`는 현재 이름·경로의 항목만 갱신한다. 이전 버전이
 * 다른 이름/경로로 등록해 둔 항목은 남아서 인자 없이 실행되고, 그 인스턴스가
 * `second-instance`로 창을 띄운다. 이름을 알 수 없으므로 경로에 croffle이 든
 * 항목을 열거해 지운 뒤, 호출자가 현재 설정대로 다시 등록한다.
 *
 * 실패는 로그와 결과에만 남기고 예외를 던지지 않는다.
 */
export async function cleanupLoginItems(): Promise<LoginItemCleanupResult> {
  const result: LoginItemCleanupResult = { removed: [], errors: [] };

  switch (process.platform) {
    case 'win32':
      await cleanupWindows(result);
      break;
    case 'darwin':
      await cleanupMac(result);
      break;
    case 'linux':
      cleanupLinux(result);
      break;
    default:
      break;
  }

  // 현재 이름/경로 항목도 해제한다. 호출자가 설정대로 재등록한다.
  try {
    app.setLoginItemSettings({ name: LOGIN_ITEM_NAME, openAtLogin: false, path: process.execPath });
  } catch (err) {
    result.errors.push(`setLoginItemSettings: ${String(err)}`);
  }

  logger.info(
    'LoginItems',
    `Cleanup done. removed=${JSON.stringify(result.removed)} errors=${result.errors.length}`,
  );
  return result;
}

// --- Windows -----------------------------------------------------------------

async function regQueryValues(key: string): Promise<{ name: string; data: string }[]> {
  const { stdout } = await execFileAsync('reg', ['query', key], { windowsHide: true });
  const values: { name: string; data: string }[] = [];
  for (const line of stdout.split(/\r?\n/)) {
    // "    Name    REG_SZ    C:\path\app.exe --arg"
    const match = /^\s{2,}(.+?)\s{2,}REG_\w+\s{2,}(.*)$/.exec(line);
    if (match) {
      values.push({ name: match[1]!.trim(), data: match[2]!.trim() });
    }
  }
  return values;
}

async function regDeleteValue(key: string, name: string): Promise<void> {
  await execFileAsync('reg', ['delete', key, '/v', name, '/f'], { windowsHide: true });
}

async function cleanupWindows(result: LoginItemCleanupResult): Promise<void> {
  // 1) HKCU\...\Run 값 중 데이터(실행 경로)에 croffle이 든 것
  try {
    const values = await regQueryValues(RUN_KEY);
    for (const value of values) {
      if (!MATCH.test(value.data)) {
        continue;
      }
      try {
        await regDeleteValue(RUN_KEY, value.name);
        result.removed.push(`Run\\${value.name}`);
      } catch (err) {
        result.errors.push(`reg delete ${value.name}: ${String(err)}`);
      }
      // StartupApproved에 같은 이름의 활성/비활성 플래그가 있으면 같이 정리
      try {
        await regDeleteValue(STARTUP_APPROVED_KEY, value.name);
      } catch {
        // 없으면 무시
      }
    }
  } catch (err) {
    result.errors.push(`reg query: ${String(err)}`);
  }

  // 2) 시작 폴더 바로가기
  const startupDir = path.join(
    process.env.APPDATA ?? path.join(homedir(), 'AppData', 'Roaming'),
    'Microsoft',
    'Windows',
    'Start Menu',
    'Programs',
    'Startup',
  );
  removeMatchingFiles(startupDir, /\.lnk$/i, result, 'Startup');
}

// --- macOS -------------------------------------------------------------------

async function cleanupMac(result: LoginItemCleanupResult): Promise<void> {
  try {
    const script =
      // "roffle": Croffle / croffle 둘 다 매칭 (AppleScript contains는 대소문자 구분)
      'tell application "System Events" to delete (every login item whose name contains "roffle")';
    await execFileAsync('osascript', ['-e', script]);
    result.removed.push('System Events login items matching croffle');
  } catch (err) {
    result.errors.push(`osascript: ${String(err)}`);
  }
}

// --- Linux -------------------------------------------------------------------

function cleanupLinux(result: LoginItemCleanupResult): void {
  const autostartDir = path.join(
    process.env.XDG_CONFIG_HOME ?? path.join(homedir(), '.config'),
    'autostart',
  );
  removeMatchingFiles(autostartDir, /\.desktop$/i, result, 'autostart');
}

// --- shared ------------------------------------------------------------------

function removeMatchingFiles(
  dir: string,
  ext: RegExp,
  result: LoginItemCleanupResult,
  label: string,
): void {
  if (!existsSync(dir)) {
    return;
  }
  let entries: string[];
  try {
    entries = readdirSync(dir);
  } catch (err) {
    result.errors.push(`readdir ${dir}: ${String(err)}`);
    return;
  }
  for (const entry of entries) {
    if (!ext.test(entry) || !MATCH.test(entry)) {
      continue;
    }
    try {
      unlinkSync(path.join(dir, entry));
      result.removed.push(`${label}\\${entry}`);
    } catch (err) {
      result.errors.push(`unlink ${entry}: ${String(err)}`);
    }
  }
}
