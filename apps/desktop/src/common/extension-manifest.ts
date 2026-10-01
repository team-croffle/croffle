import type { ContextMenuManifest } from './models';

export type SanitizedContextMenus = {
  items: ContextMenuManifest[];
  /** One line per dropped item, for the main-process log. */
  dropped: string[];
};

const isNonEmptyString = (value: unknown): value is string =>
  typeof value === 'string' && value.trim() !== '';

/**
 * Keep only well-formed `contributes.contextMenus` items so a typo in one manifest entry cannot
 * break the host's context menu. Duplicate ids keep the first item.
 */
export function sanitizeContextMenus(raw: unknown): SanitizedContextMenus {
  const items: ContextMenuManifest[] = [];
  const dropped: string[] = [];
  if (raw === undefined) {
    return { items, dropped };
  }
  if (!Array.isArray(raw)) {
    return { items, dropped: ['contextMenus is not an array'] };
  }

  const seen = new Set<string>();
  raw.forEach((entry: unknown, index) => {
    const item = (entry ?? {}) as Record<string, unknown>;
    const at = `contextMenus[${index}]`;
    if (typeof entry !== 'object' || entry === null) {
      dropped.push(`${at}: not an object`);
      return;
    }
    if (!isNonEmptyString(item.id) || !isNonEmptyString(item.label)) {
      dropped.push(`${at}: id and label must be non-empty strings`);
      return;
    }
    if (
      item.targetView !== undefined &&
      !(Array.isArray(item.targetView) && item.targetView.every(isNonEmptyString))
    ) {
      dropped.push(`${at} (${item.id}): targetView must be an array of strings`);
      return;
    }
    if (item.disabled !== undefined && typeof item.disabled !== 'boolean') {
      dropped.push(`${at} (${item.id}): disabled must be a boolean`);
      return;
    }
    if (seen.has(item.id)) {
      dropped.push(`${at} (${item.id}): duplicate id`);
      return;
    }
    seen.add(item.id);
    items.push({
      id: item.id,
      label: item.label,
      ...(item.targetView !== undefined && { targetView: item.targetView as string[] }),
      ...(item.disabled !== undefined && { disabled: item.disabled }),
    });
  });

  return { items, dropped };
}
