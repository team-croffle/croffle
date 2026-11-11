# Contributing to Croffle

Thanks for helping improve Croffle. This guide covers local development, PRs, and releases in the monorepo.

한국어: [CONTRIBUTING.ko.md](./CONTRIBUTING.ko.md)

---

## Prerequisites

- **Node.js ≥ 24**
- **pnpm** (enable with `corepack enable`)
- OS-appropriate toolchain for native modules (`better-sqlite3`) when building the desktop app

---

## Repository layout

| Path             | Package                     | Notes                            |
| ---------------- | --------------------------- | -------------------------------- |
| `apps/desktop`   | `@croffledev/desktop`       | Electron app (private)           |
| `packages/types` | `@croffledev/croffle-types` | Published TypeScript definitions |
| `packages/cli`   | `@croffledev/croffle-cli`   | Published extension CLI          |

Desktop host API (preload) is exposed as a **domain-flat** surface, for example:

```ts
croffle.window;
croffle.calendar.schedules;
croffle.extensions.info;
croffle.extensions.configuration;
croffle.event;
// …
```

Terminology: installable packages are **extensions**. Extension options are **configuration**; app-wide preferences remain **settings** (`croffle.settings`). Extension packages use `croffle-manifest.json`.

Types live in `@croffledev/croffle-types`. Runtime enums used by the app live under `apps/desktop/src/common` (`@croffledev/common` alias).

---

## Setup

```bash
git clone https://github.com/team-croffle/croffle.git
cd croffle
pnpm install
pnpm dev                 # Electron desktop
```

Useful scripts (root):

| Script                        | Purpose                                  |
| ----------------------------- | ---------------------------------------- |
| `pnpm dev`                    | Run desktop in development               |
| `pnpm build`                  | Build desktop (electron-vite)            |
| `pnpm typecheck`              | Typecheck workspace packages             |
| `pnpm lint` / `pnpm lint:fix` | oxlint                                   |
| `pnpm format`                 | oxfmt                                    |
| `pnpm changeset`              | Add a Changeset for publishable packages |

Filter examples:

```bash
pnpm --filter @croffledev/desktop typecheck
pnpm --filter @croffledev/croffle-cli build
```

---

## Pull requests

1. Fork / branch from `master` (e.g. `feat/…`, `fix/…`, `refactor/…`).
2. Keep changes focused; prefer package-scoped PRs when practical.
3. Open a PR and pick a template from [`.github/pull_request_template.md`](./.github/pull_request_template.md):
   - Default / refactor / chore
   - Bug fix
   - Feature
4. Fill **package scope** (`desktop` / `types` / `cli`) and **Electron surface** (main / preload / renderer) when relevant.
5. Ensure `pnpm typecheck` (and lint for touched files) pass before requesting review.

### Labels

[`.github/labeler.yaml`](./.github/labeler.yaml) auto-labels PRs by path and branch prefix (`feat`, `fix`, `refactor`, …). Locale / i18n file changes get the `i18n` label. Labels are created automatically if missing.

### Commit style

Prefer concise conventional-style subjects used in this repo, for example:

- `feat: …`
- `fix: …`
- `refactor: …`
- `chore: …`

---

## Internationalization (i18n)

UI copy for **main and renderer** lives in **one catalog per language**:

| Path                                               | Role                                                                 |
| -------------------------------------------------- | -------------------------------------------------------------------- |
| `apps/desktop/src/common/i18n/locales/{lang}.json` | Shared message catalogs (`en`, `ko`, …)                              |
| `apps/desktop/src/common/i18n/index.ts`            | Lightweight `t()` / `localeMessages` for main + shared code          |
| `apps/desktop/src/main/i18n.ts`                    | Main helper that reads `settings.general.language`                   |
| `apps/desktop/src/renderer/src/i18n/index.ts`      | vue-i18n wiring (imports `localeMessages` from `@croffledev/common`) |

Do **not** add a separate `renderer/.../locales` tree. Keep keys nested (e.g. `tray.openWindow`, `settings.general.language`) and use `{name}` placeholders consistently.

### Adding a language (PR welcome)

Language packs are **built into the app** for now (not extension-contributed). A translation PR typically:

1. Add `apps/desktop/src/common/i18n/locales/<code>.json` by copying `en.json` and translating values (keep every key).
2. Register the locale in:
   - `AppSettingLanguage` in `apps/desktop/src/common/enums.ts`
   - `AppSettingLanguage` in `packages/types/models/app-settings.d.ts` (publishable — add a Changeset)
   - `localeMessages` / `resolveAppLocale` in `apps/desktop/src/common/i18n/index.ts`
3. Expose it in Settings UI (`settings-modal.vue` language options) and `language.<code>` in the locale JSON.
4. Wire calendar display if needed (`languageToLocale` in `calendar-settings.ts`, FullCalendar locale import when available).
5. Smoke-test: switch language in Settings → confirm renderer UI, tray menu, and a reminder notification body.

Prefer a focused PR titled like `feat(desktop): add <language> locale`. Use the Feature PR template and expect the `i18n` label from [`.github/labeler.yaml`](./.github/labeler.yaml).

---

## Publishable packages (npm)

`@croffledev/croffle-types` and `@croffledev/croffle-cli` are released with **Changesets**.

1. After a user-facing package change: `pnpm changeset`
2. Merge; the **Publish Packages** workflow on `master` opens a version PR or publishes when appropriate.

Desktop (`@croffledev/desktop`) is **not** published to npm.

---

## Desktop app releases (GitHub Releases)

Desktop installers are produced by the **Croffle Release** workflow (`workflow_dispatch` only)—not by npm.

1. Actions → **Croffle Release** → **Run workflow**
2. Choose `release_type`: `patch` | `minor` | `major` | `rc`
3. Optionally set exact `version` and/or `version_suffix` (e.g. `rc.1`)
4. Artifacts upload to GitHub Releases (unsigned): Windows `.exe`, macOS arm64 `.dmg`/`.zip`, Linux `.AppImage`/`.deb`/`.rpm`

Use `dry_run` to build without tagging/publishing.

---

## Plugin development

```bash
pnpm dlx @croffledev/croffle-cli create my-plugin
# or
pnpm dlx create-croffle-plugin my-plugin
```

See [`packages/cli/README.md`](./packages/cli/README.md) and install `@croffledev/croffle-types` for API typings (`CroffleAPI`, `PluginContext`, …).

---

## Code of conduct & license

Be respectful in issues and reviews. The project is licensed under the **MIT License** (`LICENSE`).
