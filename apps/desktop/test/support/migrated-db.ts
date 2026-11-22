import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { DatabaseSync } from 'node:sqlite';

const DRIZZLE_DIR = join(import.meta.dirname, '../../drizzle');

type Journal = { entries: { tag: string }[] };

/** The committed migration tags in journal order. */
export function migrationTags(): string[] {
  const journal = JSON.parse(
    readFileSync(join(DRIZZLE_DIR, 'meta/_journal.json'), 'utf8'),
  ) as Journal;
  return journal.entries.map((e) => e.tag);
}

export function migrationSql(tag: string): string {
  return readFileSync(join(DRIZZLE_DIR, `${tag}.sql`), 'utf8');
}

/** A memory database with every committed migration applied, split the way drizzle's migrator splits. */
export function migratedDatabase(): DatabaseSync {
  const db = new DatabaseSync(':memory:');
  db.exec('PRAGMA foreign_keys = ON');
  for (const tag of migrationTags()) {
    for (const statement of migrationSql(tag).split('--> statement-breakpoint')) {
      if (statement.trim()) {
        db.exec(statement);
      }
    }
  }
  return db;
}
