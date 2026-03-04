import { promises as fs } from 'node:fs';
import { dirname } from 'node:path';

import type { Schedule as ScheduleInterface } from 'croffle';

import { fileDialogService } from '../../native-os/service/FileDialogService';
import { scheduleService } from './ScheduleService';
import { ScheduleMapper } from '../mapper/ScheduleMapper';

import { databaseManager } from '../../../services/DatabaseManager';
import { Schedule as ScheduleEntity } from '../model/Schedule';

type ExportShapeV1 = {
  version: 1;
  exportedAt: string;
  schedules: ScheduleInterface[];
};

export class ScheduleImportExportService {
  async exportSchedulesToFile(period?: {
    start: string;
    end: string;
  }): Promise<{ filePath: string; count: number } | null> {
    const filePath = await fileDialogService.saveJsonFile({
      title: 'Export schedules',
      defaultFileName: 'schedules.json',
    });

    if (!filePath) return null;

    const schedules = await scheduleService.getSchedules(
      period
        ? { start: new Date(period.start), end: new Date(period.end) }
        : { start: new Date('1970-01-01T00:00:00.000Z'), end: new Date('2999-12-31T23:59:59.999Z') }
    );

    const payload: ExportShapeV1 = {
      version: 1,
      exportedAt: new Date().toISOString(),
      schedules: schedules.map(ScheduleMapper.toInterface),
    };

    await fs.mkdir(dirname(filePath), { recursive: true });
    await fs.writeFile(filePath, JSON.stringify(payload, null, 2), 'utf-8');

    return { filePath, count: payload.schedules.length };
  }

  async importScheduleFromFile(
    mode: 'merge' | 'duplicate' = 'merge'
  ): Promise<{ created: number; updated: number } | null> {
    const filePath = await fileDialogService.openJsonFile({ title: 'Import schedules' });
    if (!filePath) return null;

    const raw = await fs.readFile(filePath, 'utf-8');
    const parsed = JSON.parse(raw) as ExportShapeV1 | ScheduleInterface[];

    // 허용 포맷:
    // 1) { version, exportedAt, schedules: [...] }
    // 2) Schedule[] 단독 배열
    const schedules: ScheduleInterface[] = Array.isArray(parsed)
      ? parsed
      : Array.isArray(parsed.schedules)
        ? parsed.schedules
        : [];

    if (schedules.length === 0) {
      throw new Error('Import file has no schedules');
    }

    const repo = databaseManager.getRepository(ScheduleEntity);

    let created = 0;
    let updated = 0;

    for (const s of schedules) {
      const entityData = ScheduleMapper.toEntity(s);

      if (mode === 'duplicate') {
        await scheduleService.createSchedule(entityData);
        created += 1;
        continue;
      }

      // merge: id 기반 upsert
      if (s.id) {
        const exists = await repo.findOne({ where: { id: s.id } });
        if (exists) {
          await scheduleService.updateSchedule(s.id, entityData);
          updated += 1;
          continue;
        }
      }

      await scheduleService.createSchedule(entityData);
      created += 1;
    }

    return { created, updated };
  }
}

export const scheduleImportExportService = new ScheduleImportExportService();
