import type { Tag } from '@croffledev/common';
import { defineStore } from 'pinia';
import { ref } from 'vue';

/** 새 태그에 자동 배정하는 색. 일정 색 프리셋과 같은 팔레트. */
export const TAG_COLOR_PRESETS = [
  '#DCA780',
  '#F87171',
  '#FBBF24',
  '#34D399',
  '#60A5FA',
  '#A78BFA',
  '#F472B6',
];

export const TAG_NAME_MAX_LENGTH = 50;

const normalizeName = (name: string) => name.trim().toLowerCase();

/** 같은 이름이면 항상 같은 색이 나오도록 이름 해시로 프리셋을 고른다. */
export function pickTagColor(name: string): string {
  let hash = 0;
  for (const ch of normalizeName(name)) {
    hash = (hash * 31 + ch.codePointAt(0)!) >>> 0;
  }
  return TAG_COLOR_PRESETS[hash % TAG_COLOR_PRESETS.length];
}

export const useTagStore = defineStore('tag', () => {
  const tags = ref<Tag[]>([]);
  const loaded = ref(false);
  let loading: Promise<void> | null = null;

  const loadTags = async (force = false) => {
    if (loaded.value && !force) {
      return;
    }
    if (loading) {
      return loading;
    }
    loading = (async () => {
      try {
        tags.value = await croffle.calendar.tags.getAll();
        loaded.value = true;
      } finally {
        loading = null;
      }
    })();
    return loading;
  };

  const findByName = (name: string) => {
    const target = normalizeName(name);
    return tags.value.find((tag) => normalizeName(tag.name) === target);
  };

  const upsertTag = (tag: Tag) => {
    const index = tags.value.findIndex((t) => t.id === tag.id);
    if (index === -1) {
      tags.value.push(tag);
      return;
    }
    tags.value[index] = tag;
  };

  const createTag = async (name: string, color = pickTagColor(name)) => {
    const created = await croffle.calendar.tags.create(name.trim(), color);
    upsertTag(created);
    return created;
  };

  /**
   * 이름으로 태그를 보장한다. 있으면 그 태그, 없으면 만든다.
   * 생성이 중복 이름으로 실패하면(다른 곳에서 먼저 만든 경우) 이름으로 다시 조회한다.
   */
  const ensureTag = async (name: string): Promise<Tag> => {
    await loadTags();
    const existing = findByName(name);
    if (existing) {
      return existing;
    }
    try {
      return await createTag(name);
    } catch (error) {
      const fetched = await croffle.calendar.tags.getByName(name.trim());
      if (fetched) {
        upsertTag(fetched);
        return fetched;
      }
      throw error;
    }
  };

  return {
    tags,
    loaded,
    loadTags,
    findByName,
    createTag,
    ensureTag,
  };
});
