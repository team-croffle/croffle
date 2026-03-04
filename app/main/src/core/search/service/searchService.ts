import { databaseManager } from '../../../services/DatabaseManager';
import { Schedule } from '../../schedules/model/Schedule';
import { Tag } from '../../tags/model/Tag';
import { SearchQuery } from 'croffle';

export class SearchService {
  async searchSchedules(query: SearchQuery): Promise<Schedule[]> {
    const repo = databaseManager.getRepository(Schedule);

    const qb = repo.createQueryBuilder('schedule').leftJoinAndSelect('schedule.tags', 'tag');

    // 텍스트 검색
    if (query.text?.trim()) {
      const keyword = `%${query.text.trim()}%`;
      qb.andWhere('(schedule.title LIKE :keyword OR schedule.description LIKE :keyword)', {
        keyword,
      });
    }

    // 날짜 범위 검색
    const start = query.dateRange?.start ? new Date(query.dateRange.start) : null;
    const end = query.dateRange?.end ? new Date(query.dateRange.end) : null;

    if (start && end) {
      qb.andWhere('schedule.endDate >= :start AND schedule.startDate <= :end', { start, end });
    } else if (start) {
      qb.andWhere('schedule.endDate >= :start', { start });
    } else if (end) {
      qb.andWhere('schedule.startDate <= :end', { end });
    }

    // 태그 필터
    if (query.tags?.length) {
      const validTags = query.tags.filter((tag) => tag?.id);
      if (validTags.length > 0) {
        const tagIds = validTags.map((tag: Tag) => tag.id);
        qb.andWhere('tag.id IN (:...tagIds)', { tagIds });
      }
    }

    // 정렬 및 중복 제거
    qb.orderBy('schedule.startDate', 'ASC');

    return qb.getMany();
  }
}

export const searchService = new SearchService();
