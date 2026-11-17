# @croffledev/croffle-cli

CLI for scaffolding, building, and packaging Croffle **extensions**.

The package ships two names for the same command, `croffle-plugin` and `create-croffle-plugin`, so name one when you run it without installing:

```bash
pnpm --package=@croffledev/croffle-cli dlx croffle-plugin create my-extension
# or: npx -p @croffledev/croffle-cli croffle-plugin create my-extension
```

Inside the extension project:

```bash
pnpm --package=@croffledev/croffle-cli dlx croffle-plugin build   # Vite or tsup
pnpm --package=@croffledev/croffle-cli dlx croffle-plugin pack    # .zip to install in Croffle
```

Extension types live in `@croffledev/croffle-types` (`ExtensionContext`, `ExtensionInfo`, …).  
Manifest file is `croffle-manifest.json` (`id`, `name`, `version`, `author`, `main?`, `engines?`, `contributes?`).  
In `activated(ctx)`, use `ctx.storage` / `ctx.session` / `ctx.configuration` (id is bound). Since types 2.0 the context carries only the public surface — no window control, settings writes or extension management.

Templates declare a right-click menu item under `contributes.contextMenus` and attach its action with `ctx.ui.onContextMenu(id, handler)`; they require Croffle 1.2.5 or later.
