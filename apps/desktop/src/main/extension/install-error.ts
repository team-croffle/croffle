/** Why an extension install failed; the renderer maps the code to a localized message. */
export type ExtensionInstallErrorCode = 'invalid-archive';

export class ExtensionInstallError extends Error {
  constructor(
    readonly code: ExtensionInstallErrorCode,
    message: string,
  ) {
    super(`[${code}] ${message}`);
    this.name = 'ExtensionInstallError';
  }
}
