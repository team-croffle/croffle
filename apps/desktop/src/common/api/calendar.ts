import type { SchedulesApi } from '@croffledev/croffle-types';

import type { Schedule } from '../models/schedule';

/** Public schedule CRUD plus the host-only native file dialogs. */
export interface InternalSchedulesApi extends SchedulesApi {
  /** Every To-Do not done yet, overdue ones too, earliest deadline first (host only). */
  listOpenTodos(): Promise<Schedule[]>;
  exportSchedulesToFile(period?: {
    start: string;
    end: string;
  }): Promise<{ filePath: string; count: number } | null>;
  importScheduleFromFile(
    mode?: 'merge' | 'duplicate',
  ): Promise<{ created: number; updated: number } | null>;
}
