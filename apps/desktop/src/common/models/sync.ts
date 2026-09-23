/**
 * Row-level LWW bookkeeping shared by every synchronised table (1.2.3 schema prep).
 * Internal only: never surfaced through `packages/types`.
 */
export type SyncColumns = {
  /** Millisecond wall clock of the last write. LWW comparison key (`updatedAt` is second-resolution, UI only). */
  updatedAtMs: number;
  /** Monotonic per-row counter, starts at 1 and bumps on every write. */
  version: number;
  /** Tombstone. Non-null rows are hidden from every query and never physically deleted. */
  deletedAt: Date | null;
  /** `device:<clientId>` · `<provider>:<accountId>` · `server`. Null for rows written before 1.2.3. */
  lastWriterId: string | null;
  /** Owning account once sync is enabled (1.3). Null while local-only. */
  ownerId: string | null;
};

export const LAST_WRITER_DEVICE_PREFIX = 'device:';
