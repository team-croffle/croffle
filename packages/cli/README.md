# @croffledev/croffle-cli

CLI for scaffolding, building, and packaging Croffle **extensions**.

```bash
pnpm dlx @croffledev/croffle-cli create my-extension
# or
pnpm dlx create-croffle-plugin my-extension
```

```bash
pnpm dlx @croffledev/croffle-cli build
pnpm dlx @croffledev/croffle-cli pack
```

Extension types live in `@croffledev/croffle-types` (`ExtensionContext`, `ExtensionInfo`, …).  
Manifest file is `croffle-manifest.json` (`id`, `name`, `version`, `author`, `main?`, `engines?`, `contributes?`).  
In `activated(ctx)`, prefer `ctx.storage` / `ctx.session` / `ctx.configuration` (id is bound); host UI keeps using `croffle.extensions.*`.

Templates declare a right-click menu item under `contributes.contextMenus` and attach its action with `ctx.ui.onContextMenu(id, handler)`; they require Croffle 1.2.5 or later.
