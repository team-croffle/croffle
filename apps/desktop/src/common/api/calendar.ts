import type { SchedulesApi } from '@croffledev/croffle-types';

/** Public schedule CRUD plus the host-only native file dialogs. */
export interface InternalSchedulesApi extends SchedulesApi {
  exportSchedulesToFile(period?: {
    start: string;
    end: string;
  }): Promise<{ filePath: string; count: number } | null>;
  importScheduleFromFile(
    mode?: 'merge' | 'duplicate',
  ): Promise<{ created: number; updated: number } | null>;
}
