/**
 * The rules of the schedule kinds (1.3.3). A To-Do (`kind: 'todo'`) has a deadline only: its
 * `startDate` equals `endDate`, it does not repeat, and `completedAt` says when it was done. An
 * event never has `completedAt`. No database here, so the rules can be tested on their own.
 */
export type ScheduleKind = 'event' | 'todo';

export const SCHEDULE_KINDS: readonly ScheduleKind[] = ['event', 'todo'];

export type KindFields = {
  kind?: ScheduleKind;
  completedAt?: Date | null;
  startDate?: Date;
  endDate?: Date;
  recurrenceRule?: string | null;
};

/** The kind a new schedule gets: missing (an older client, an import without it) is an event. */
export function kindForCreate(input: { kind?: ScheduleKind }): ScheduleKind {
  return input.kind ?? 'event';
}

/** Throws when the fields break a kind rule. */
export function validateKindFields(fields: KindFields): void {
  if (fields.kind !== undefined && !SCHEDULE_KINDS.includes(fields.kind)) {
    throw new Error('Schedule kind must be "event" or "todo"');
  }
  const kind = fields.kind ?? 'event';
  if (kind === 'todo' && fields.recurrenceRule?.trim()) {
    throw new Error('A To-Do cannot repeat');
  }
  if (kind === 'event' && fields.completedAt) {
    throw new Error('Only a To-Do can be completed');
  }
}

/** A To-Do's start is its deadline; an event keeps its dates. */
export function alignTodoDates<T extends KindFields>(fields: T): T {
  if (fields.kind !== 'todo' || !fields.endDate) {
    return fields;
  }
  return { ...fields, startDate: fields.endDate };
}

/**
 * The kind fields of an update: what the patch leaves out stays. Turning a To-Do into an event
 * clears `completedAt` unless the patch sets it (which validation then refuses).
 */
export function mergeKindFields(
  existing: { kind: ScheduleKind; completedAt: Date | null },
  patch: { kind?: ScheduleKind; completedAt?: Date | null },
): { kind: ScheduleKind; completedAt: Date | null } {
  const kind = patch.kind ?? existing.kind;
  if (patch.completedAt !== undefined) {
    return { kind, completedAt: patch.completedAt };
  }
  return { kind, completedAt: kind === 'todo' ? existing.completedAt : null };
}
