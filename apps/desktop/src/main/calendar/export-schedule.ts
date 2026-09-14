import { promises as fs } from 'node:fs';
import { dirname } from 'node:path';

import { t } from '../i18n';
import { scheduleMapper } from '../mapper/schedule-mapper';
import { saveJsonFileDialog } from '../window/json-file-dialog';
import type { ExportShapeV1 } from './export.type';
import { getSchedules } from './schedule-service';

export async function exportSchedulesToFile(period?: {
  start: string;
  end: string;
}): Promise<{ filePath: string; count: number } | null> {
  const filePath = await saveJsonFileDialog({
    title: t('importExport.exportDialogTitle'),
    defaultFileName: 'schedules.json',
  });

  if (!filePath) {
    return null;
  }

  const schedules = await getSchedules(
    period
      ? { start: new Date(period.start), end: new Date(period.end) }
      : {
          start: new Date('1970-01-01T00:00:00.000Z'),
          end: new Date('2999-12-31T23:59:59.999Z'),
        },
  );

  const payload: ExportShapeV1 = {
    version: 1,
    exportedAt: new Date().toISOString(),
    schedules: schedules.map(scheduleMapper.toInterface),
  };

  await fs.mkdir(dirname(filePath), { recursive: true });
  await fs.writeFile(filePath, JSON.stringify(payload, null, 2), 'utf-8');

  return { filePath, count: payload.schedules.length };
}
