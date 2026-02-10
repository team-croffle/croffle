import { ipcMain } from 'electron';
import { scheduleService } from '../core/schedules/service/ScheduleService';
import { Schedule } from 'croffle';
import { ScheduleMapper } from '../core/schedules/mapper/ScheduleMapper';
import { eventService } from '../core/events/service/EventService';
import { AppEventType } from '../shared/enums';

export const registerScheduleIpcHandlers = (): void => {
  ipcMain.handle(
    'schedule:get',
    async (_, period: { start: string; end: string }): Promise<Schedule[]> => {
      const schedules = await scheduleService.getSchedules({
        start: new Date(period.start),
        end: new Date(period.end),
      });

      // Add app event emit
      eventService.emit(AppEventType.SCHEDULE_GET, schedules);

      return schedules.map(ScheduleMapper.toInterface);
    }
  );

  ipcMain.handle('schedule:create', async (_, data: Partial<Schedule>): Promise<Schedule> => {
    const entityData = ScheduleMapper.toEntity(data);
    const createdEntity = await scheduleService.createSchedule(entityData);

    // Add app event emit
    eventService.emit(AppEventType.SCHEDULE_CREATE, createdEntity);

    return ScheduleMapper.toInterface(createdEntity);
  });

  ipcMain.handle(
    'schedule:update',
    async (_, id: string, data: Partial<Schedule>): Promise<Schedule> => {
      const entityData = ScheduleMapper.toEntity(data);
      const updatedEntity = await scheduleService.updateSchedule(id, entityData);

      // Add app event emit
      eventService.emit(AppEventType.SCHEDULE_UPDATE, updatedEntity);

      return ScheduleMapper.toInterface(updatedEntity);
    }
  );

  ipcMain.handle('schedule:delete', async (_, id: string): Promise<boolean> => {
    // Add app event emit
    eventService.emit(AppEventType.SCHEDULE_DELETE, id);

    return await scheduleService.deleteSchedule(id);
  });
};
