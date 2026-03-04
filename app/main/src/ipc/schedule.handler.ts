import { ipcMain } from 'electron';
import { scheduleService } from '../core/schedules/service/ScheduleService';
import { Schedule } from 'croffle';
import { ScheduleMapper } from '../core/schedules/mapper/ScheduleMapper';
import { eventService } from '../core/events/service/EventService';
import { AppEventType } from '../shared/enums';
import { scheduleImportExportService } from '../core/schedules/service/ScheduleImportExportService';

export const registerScheduleIpcHandlers = (): void => {
  ipcMain.handle(
    'schedule:get',
    async (_, period: { start: string; end: string }): Promise<Schedule[]> => {
      const schedules = await scheduleService.getSchedules({
        start: new Date(period.start),
        end: new Date(period.end),
      });

      const dto = schedules.map(ScheduleMapper.toInterface);
      // Add app event emit
      eventService.emit(AppEventType.SCHEDULE_GET, dto);

      return dto;
    }
  );

  ipcMain.handle('schedule:create', async (_, data: Partial<Schedule>): Promise<Schedule> => {
    const entityData = ScheduleMapper.toEntity(data);
    const createdEntity = await scheduleService.createSchedule(entityData);

    const dto = ScheduleMapper.toInterface(createdEntity);
    // Add app event emit
    eventService.emit(AppEventType.SCHEDULE_CREATE, dto);

    return dto;
  });

  ipcMain.handle(
    'schedule:update',
    async (_, id: string, data: Partial<Schedule>): Promise<Schedule> => {
      const entityData = ScheduleMapper.toEntity(data);
      const updatedEntity = await scheduleService.updateSchedule(id, entityData);

      const dto = ScheduleMapper.toInterface(updatedEntity);
      // Add app event emit
      eventService.emit(AppEventType.SCHEDULE_UPDATE, dto);

      return dto;
    }
  );

  ipcMain.handle('schedule:delete', async (_, id: string): Promise<boolean> => {
    // Add app event emit
    eventService.emit(AppEventType.SCHEDULE_DELETE, id);

    return await scheduleService.deleteSchedule(id);
  });

  ipcMain.handle(
    'schedule:exportSchedulesToFile',
    async (
      _,
      period?: { start: string; end: string }
    ): Promise<{ filePath: string; count: number } | null> => {
      const result = await scheduleImportExportService.exportSchedulesToFile(period);
      eventService.emit(AppEventType.SCHEDULE_EXPORT_TO_FILE, result);
      return result;
    }
  );

  ipcMain.handle(
    'schedule:importScheduleFromFile',
    async (
      _,
      mode?: 'merge' | 'duplicate'
    ): Promise<{ created: number; updated: number } | null> => {
      const result = await scheduleImportExportService.importScheduleFromFile(mode ?? 'merge');
      eventService.emit(AppEventType.SCHEDULE_IMPORT_FROM_FILE, result);
      return result;
    }
  );
};
