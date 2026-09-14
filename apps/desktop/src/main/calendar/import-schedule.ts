import { promises as fs } from 'node:fs';

import type { Schedule as ScheduleInterface, Tag } from '@croffledev/common';
import { eq } from 'drizzle-orm';

import { databaseManager } from '../database';
import { schedules, type TagRow } from '../database/schema';
import { t } from '../i18n';
import { scheduleMapper } from '../mapper/schedule-mapper';
import { colorValidation } from '../utils/color-validator';
import { openJsonFileDialog } from '../window/json-file-dialog';
import type { ExportShapeV1 } from './export.type';
import { reminderScheduler } from './reminder-scheduler';
import { createSchedule, updateSchedule } from './schedule-service';
import { createTag, getTagByName } from './tag';

const FALLBACK_TAG_COLOR = '#DCA780';

/**
 * 파일의 태그는 다른 기기에서 만든 id일 수 있으므로 **이름**으로 이 DB의 태그에 맞춘다.
 * 없으면 파일의 색(유효하지 않으면 기본색)으로 만든다. 같은 파일 안에서는 캐시로 한 번만 조회한다.
 */
async function resolveTagsByName(
  fileTags: Tag[] | undefined,
  cache: Map<string, TagRow>,
): Promise<TagRow[] | undefined> {
  if (!Array.isArray(fileTags)) {
    return undefined;
  }

  const resolved: TagRow[] = [];
  for (const fileTag of fileTags) {
    const name = typeof fileTag?.name === 'string' ? fileTag.name.trim() : '';
    if (!name) {
      continue;
    }
    const key = name.toLowerCase();
    let row = cache.get(key);
    if (!row) {
      row =
        (await getTagByName(name)) ??
        (await createTag(
          name,
          typeof fileTag.color === 'string' && colorValidation(fileTag.color)
            ? fileTag.color
            : FALLBACK_TAG_COLOR,
        ));
      cache.set(key, row);
    }
    if (!resolved.some((item) => item.id === row!.id)) {
      resolved.push(row);
    }
  }
  return resolved;
}

function parseImportFile(raw: string): ScheduleInterface[] {
  let parsed: ExportShapeV1 | ScheduleInterface[];
  try {
    parsed = JSON.parse(raw) as ExportShapeV1 | ScheduleInterface[];
  } catch {
    throw new Error('Import file is not valid JSON');
  }

  const scheduleList: ScheduleInterface[] = Array.isArray(parsed)
    ? parsed
    : Array.isArray(parsed?.schedules)
      ? parsed.schedules
      : [];

  if (scheduleList.length === 0) {
    throw new Error('Import file has no schedules');
  }
  return scheduleList;
}

export async function importScheduleFromFile(
  mode: 'merge' | 'duplicate' = 'merge',
): Promise<{ created: number; updated: number } | null> {
  const filePath = await openJsonFileDialog({ title: t('importExport.importDialogTitle') });
  if (!filePath) {
    return null;
  }

  const raw = await fs.readFile(filePath, 'utf-8');
  const scheduleList = parseImportFile(raw);

  const db = databaseManager.getDb();
  const tagCache = new Map<string, TagRow>();

  let created = 0;
  let updated = 0;

  try {
    for (const s of scheduleList) {
      const entityData = scheduleMapper.toEntity(s);
      entityData.tags = await resolveTagsByName(s.tags, tagCache);

      if (mode === 'duplicate') {
        // 같은 기기에서 내보낸 파일이면 id가 이미 있으므로 새 id로 만든다.
        delete entityData.id;
        await createSchedule(entityData);
        created += 1;
        continue;
      }

      if (s.id) {
        const exists = await db.query.schedules.findFirst({
          where: eq(schedules.id, s.id),
        });
        if (exists) {
          await updateSchedule(s.id, entityData);
          updated += 1;
          continue;
        }
      }

      await createSchedule(entityData);
      created += 1;
    }
  } catch (error) {
    const reason = error instanceof Error ? error.message : String(error);
    const index = created + updated;
    const title = scheduleList[index]?.title ?? '?';
    throw new Error(
      `Imported ${created + updated} of ${scheduleList.length}, failed at "${title}": ${reason}`,
      { cause: error },
    );
  } finally {
    if (created + updated > 0) {
      void reminderScheduler.rebuild();
    }
  }

  return { created, updated };
}
