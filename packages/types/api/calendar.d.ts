import type { Schedule } from '../models/schedule';
import type { SearchQuery } from '../models/search';
import type { Tag } from '../models/tag';

export interface SchedulesApi {
  getAll(period: { start: string; end: string }): Promise<Schedule[]>;
  create(data: Partial<Schedule>): Promise<Schedule>;
  update(id: string, data: Partial<Schedule>): Promise<Schedule>;
  remove(id: string): Promise<boolean>;
  /**
   * Move one occurrence of a recurring schedule, keeping its duration.
   * `occurrenceStart` is the occurrence's original start; the series is unchanged.
   * Rejects for non-recurring schedules. Returns the schedule with its updated `exceptions`.
   */
  moveOccurrence(
    id: string,
    occurrenceStart: Date | string,
    newStart: Date | string,
  ): Promise<Schedule>;
  exportSchedulesToFile(period?: {
    start: string;
    end: string;
  }): Promise<{ filePath: string; count: number } | null>;
  importScheduleFromFile(
    mode?: 'merge' | 'duplicate',
  ): Promise<{ created: number; updated: number } | null>;
}

export interface TagsApi {
  getAll(): Promise<Tag[]>;
  getByName(name: string): Promise<Tag | null>;
  create(name: string, color: string): Promise<Tag>;
  modify(id: string, name: string, color: string): Promise<Tag>;
  remove(id: string): Promise<boolean>;
}

export interface SearchApi {
  q(query: SearchQuery): Promise<Schedule[]>;
}

export interface CalendarApi {
  schedules: SchedulesApi;
  tags: TagsApi;
  search: SearchApi;
}
