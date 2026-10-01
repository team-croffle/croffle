---
'@croffledev/croffle-types': minor
---

Add `UiApi.onContextMenu(id, handler, options?)` to attach an action to a menu item declared in the manifest's `contributes.contextMenus`; it returns a function that detaches it. `registerContextMenu`'s callback now receives the right-clicked element (`HTMLElement | null`), matching what the app has always passed. `ContextMenuManifest` fields are documented.
