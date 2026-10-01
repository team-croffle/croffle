---
'@croffledev/croffle-cli': minor
---

Templates declare their context menu item in `croffle-manifest.json` (`contributes.contextMenus`) and attach the action with `context.ui.onContextMenu`, detaching it in `deactivated`. They require Croffle 1.2.5 (`engines.croffle: ">=1.2.5"`) and `@croffledev/croffle-types` ^1.8.0, and use `moduleResolution: "bundler"` so `tsc` checks pass in new projects.
