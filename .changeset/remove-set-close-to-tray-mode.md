---
'@croffledev/croffle-types': major
---

Remove the deprecated `WindowApi.setCloseToTrayMode`. The close-button behavior is the `general.closeBehavior` setting (`ask` | `tray` | `quit`); write it with `settings.update({ general: { ...general, closeBehavior } })`.
