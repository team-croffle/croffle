---
'@croffledev/croffle-types': patch
---

Fix type resolution for projects using `moduleResolution: "nodenext"` (such as the `croffle-cli` templates): relative imports in the declaration files now carry `.js` extensions, so `ExtensionContext` and every other export resolve instead of reporting "no exported member".
