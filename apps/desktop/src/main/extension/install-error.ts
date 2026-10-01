import type { ExtensionInstallErrorCode } from '@croffledev/common';

/** An install failure the renderer can localize; the message starts with `[code]`. */
export class ExtensionInstallError extends Error {
  constructor(
    readonly code: ExtensionInstallErrorCode,
    message: string,
  ) {
    super(`[${code}] ${message}`);
    this.name = 'ExtensionInstallError';
  }
}
