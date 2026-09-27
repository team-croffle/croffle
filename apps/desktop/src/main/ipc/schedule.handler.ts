import { AppEventType } from '@croffledev/common';
import type { Schedule } from '@croffledev/common';
import { ipcMain } from 'electron';

import { exportSchedulesToFile } from '../calendar/export-schedule';
import { importScheduleFromFile } from '../calendar/import-schedule';
import { moveOccurrence } from '../calendar/schedule-exceptions';
import {
  getSchedules,
  createSchedule,
  updateSchedule,
  deleteSchedule,
} from '../calendar/schedule-service';
import { eventService } from '../event-bus/event-service';
import { scheduleMapper } from '../mapper/schedule-mapper';

export const registerScheduleIpcHandlers = (): void => {
  ipcMain.handle(
    'schedule:get',
    async (_, period: { start: string; end: string }): Promise<Schedule[]> => {
      const schedules = await getSchedules({
        start: new Date(period.start),
        end: new Date(period.end),
      });

      return schedules.map(scheduleMapper.toInterface);
    },
  );

  ipcMain.handle('schedule:create', async (_, data: Partial<Schedule>): Promise<Schedule> => {
    const entityData = scheduleMapper.toEntity(data);
    const createdEntity = await createSchedule(entityData);

    const dto = scheduleMapper.toInterface(createdEntity);
    eventService.emit(AppEventType.SCHEDULE_CREATE, dto);

    return dto;
  });

  ipcMain.handle(
    'schedule:update',
    async (_, id: string, data: Partial<Schedule>): Promise<Schedule> => {
      const entityData = scheduleMapper.toEntity(data);
      const updatedEntity = await updateSchedule(id, entityData);

      const dto = scheduleMapper.toInterface(updatedEntity);
      eventService.emit(AppEventType.SCHEDULE_UPDATE, dto);

      return dto;
    },
  );

  ipcMain.handle(
    'schedule:moveOccurrence',
    async (
      _,
      id: string,
      occurrenceStart: Date | string,
      newStart: Date | string,
    ): Promise<Schedule> => {
      const updatedEntity = await moveOccurrence(id, occurrenceStart, newStart);

      const dto = scheduleMapper.toInterface(updatedEntity);
      eventService.emit(AppEventType.SCHEDULE_UPDATE, dto);

      return dto;
    },
  );

  ipcMain.handle('schedule:delete', async (_, id: string): Promise<boolean> => {
    const result = await deleteSchedule(id);
    if (result) {
      eventService.emit(AppEventType.SCHEDULE_DELETE, id);
    }
    return result;
  });

  ipcMain.handle(
    'schedule:exportSchedulesToFile',
    async (
      _,
      period?: { start: string; end: string },
    ): Promise<{ filePath: string; count: number } | null> => {
      const result = await exportSchedulesToFile(period);
      if (result) {
        eventService.emit(AppEventType.SCHEDULE_EXPORT_TO_FILE, result);
      }
      return result;
    },
  );

  ipcMain.handle(
    'schedule:importScheduleFromFile',
    async (
      _,
      mode?: 'merge' | 'duplicate',
    ): Promise<{ created: number; updated: number } | null> => {
      const result = await importScheduleFromFile(mode ?? 'merge');
      if (result) {
        eventService.emit(AppEventType.SCHEDULE_IMPORT_FROM_FILE, result);
      }
      return result;
    },
  );
};
