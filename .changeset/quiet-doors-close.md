---
'@croffledev/croffle-types': minor
---

Add `general.closeBehavior` to `AppSettings` and deprecate `window.setCloseToTrayMode`

The close button behavior is now a user setting instead of a hard-coded "hide to tray".

- `AppSettings.general.closeBehavior: AppCloseBehavior` — `ask` (default; the host prompts once and stores the answer), `tray`, or `quit`.
- `AppCloseBehavior` enum.
- `AppEventType.WINDOW_CLOSE_REQUESTED` — emitted when a close is requested while the setting is `ask`.
- `WindowApi.setCloseToTrayMode` still works but now writes the setting (`true` → `tray`, `false` → `quit`). It is deprecated and will be removed in 1.3; use `settings.update` instead.
