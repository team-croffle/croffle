import type { Tag } from './tag.js';

export type SearchQuery = {
  text?: string;
  dateRange?: {
    start: string; // ISO 8601
    end: string; // ISO 8601
  };
  tags?: Tag[];
};
