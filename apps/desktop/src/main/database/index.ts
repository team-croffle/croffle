import * as path from 'node:path';

import { is } from '@electron-toolkit/utils';
import Database from 'better-sqlite3';
import { drizzle } from 'drizzle-orm/better-sqlite3';
import type { BetterSQLite3Database } from 'drizzle-orm/better-sqlite3';
import { app } from 'electron';

import type { PublicTypeIntegrity } from '../../common/assert-public-types';
import { logger } from '../logger';
import { ensureOutboxTriggers } from '../sync/outbox-triggers';
import { applyLegacySchemaFixes } from './ensure-schema';
import { runMigrations } from './migrations';
import * as schema from './schema';

export type { PublicTypeIntegrity };

export type AppDatabase = BetterSQLite3Database<typeof schema>;

class DatabaseManager {
  private sqlite: Database.Database | null = null;
  private db: AppDatabase | null = null;

  public async initialize(): Promise<void> {
    if (this.db) {
      return;
    }

    try {
      const dbPath = is.dev
        ? path.join(process.cwd(), 'dev/croffle.db')
        : path.join(app.getPath('userData'), 'croffle.db');
      logger.debug('DB', `Database path: ${dbPath}`);

      this.sqlite = new Database(dbPath);
      this.sqlite.pragma('journal_mode = WAL');
      this.sqlite.pragma('foreign_keys = ON');

      applyLegacySchemaFixes(this.sqlite);

      this.db = drizzle(this.sqlite, { schema });
      runMigrations(this.sqlite, this.db);
      ensureOutboxTriggers(this.sqlite);

      logger.info('DB', 'Database initialized successfully.');
    } catch (error) {
      logger.error('DB', 'Error during database initialization:', error);
      throw error;
    }
  }

  /** The raw connection, for pragmas that cannot run inside a drizzle transaction. */
  public getSqlite(): Database.Database {
    if (!this.sqlite) {
      throw new Error('Database not initialized. Call initialize() first.');
    }
    return this.sqlite;
  }

  public getDb(): AppDatabase {
    if (!this.db) {
      throw new Error('Database not initialized. Call initialize() first.');
    }
    return this.db;
  }
}

export const databaseManager = new DatabaseManager();
