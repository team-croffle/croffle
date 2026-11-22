import { t } from './i18n';
import { reminderBody, type ReminderCandidate } from './reminder';

/** The notification body of a reminder in `locale`. */
export function formatReminderBody(candidate: ReminderCandidate, locale?: string | null): string {
  const { key, values } = reminderBody(candidate);
  return t(key, locale, values);
}
