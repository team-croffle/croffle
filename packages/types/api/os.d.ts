import type {
  ClipboardImageData,
  ClipboardResult,
  ClipboardTextData,
} from '../models/clipboard.js';

export interface OsApi {
  showNotification(title: string, body: string): Promise<void>;
  getClipboard(): Promise<ClipboardResult>;
  setClipboard(data: ClipboardTextData | ClipboardImageData): Promise<void>;
}
