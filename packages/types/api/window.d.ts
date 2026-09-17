export interface WindowApi {
  minimize(): Promise<void>;
  maximize(): Promise<void>;
  close(): Promise<void>;
  exitApp(): Promise<void>;
  checkForUpdates(): Promise<void>;
  /**
   * @deprecated Since 1.2.2 the close behavior is a user setting. This call now writes
   * `general.closeBehavior` (`true` → `tray`, `false` → `quit`). Use `settings.update` instead.
   * Will be removed in 1.3.
   */
  setCloseToTrayMode(enabled: boolean): Promise<void>;
}
