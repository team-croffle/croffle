# Contributing to Croffle

Thanks for helping improve Croffle. This guide covers local development, checks, pull requests and, for maintainers, CI and releases.

한국어: [CONTRIBUTING.ko.md](./CONTRIBUTING.ko.md)

---

## Prerequisites

- **Node.js ≥ 24**
- **pnpm 12** (pinned in `packageManager`; enable with `corepack enable`)
- OS-appropriate toolchain for native modules (`better-sqlite3`) when building the desktop app
- **Go** (version in `apps/server/go.mod`) — only for `apps/server`

---

## Repository layout

| Path                     | Package                     | Notes                                                  |
| ------------------------ | --------------------------- | ------------------------------------------------------ |
| `apps/desktop`           | `@croffledev/desktop`       | Electron app (private; installers on GitHub Releases)  |
| `apps/server`            | `croffle-server` (Go)       | Optional sync server; container image on ghcr only     |
| `apps/server/web`        | —                           | Server dashboard (Vue); its own pnpm root and lockfile |
| `packages/types`         | `@croffledev/croffle-types` | Published TypeScript definitions for extensions        |
| `packages/cli`           | `@croffledev/croffle-cli`   | Published extension CLI                                |
| `packages/sync-fixtures` | —                           | Sync protocol scenarios shared by desktop and server   |

`apps/server` is outside the pnpm workspace: its tasks live in its `Makefile`. See [`apps/server/README.md`](./apps/server/README.md) → Development.

The desktop is split into `src/main` (Node: database, IPC, extensions, reminders, tray), `src/preload` (contextBridge → `window.croffle`), `src/renderer` (Vue, sandboxed) and `src/common` (shared code, alias `@croffledev/common`). The renderer never uses Node APIs; it goes through preload and IPC.

The host API is **domain-flat**, for example:

```ts
croffle.window;
croffle.calendar.schedules;
croffle.extensions.info;
croffle.event;
// …
```

Adding an API: IPC handler (`src/main/ipc/<domain>.handler.ts`) → preload (`src/preload/api/**`) → public types (`packages/types`).

Terminology: installable packages are **extensions**. Extension options are **configuration**; app-wide preferences are **settings** (`croffle.settings`). Extension packages use `croffle-manifest.json`.

Types live in `@croffledev/croffle-types`. Runtime enums used by the app live under `apps/desktop/src/common`.

---

## Setup

```bash
git clone https://github.com/team-croffle/croffle.git
cd croffle
pnpm install
pnpm dev                 # Electron desktop
```

For the server dashboard: `pnpm --dir apps/server/web install`.

Useful scripts (root):

| Script                        | Purpose                                         |
| ----------------------------- | ----------------------------------------------- |
| `pnpm dev`                    | Run desktop in development                      |
| `pnpm build`                  | Build desktop (electron-vite)                   |
| `pnpm build:unpack`           | Build and package the desktop without installer |
| `pnpm typecheck`              | Typecheck all workspace packages                |
| `pnpm lint` / `pnpm lint:fix` | oxlint                                          |
| `pnpm format:check`           | Check formatting (oxfmt)                        |
| `pnpm format`                 | Format everything (oxfmt)                       |
| `pnpm -r test`                | Tests (desktop: `node --test`)                  |
| `pnpm changeset`              | Add a Changeset for publishable packages        |

Other places:

```bash
pnpm --filter @croffledev/croffle-cli build      # cli (tsup)
pnpm --dir apps/server/web build                 # server dashboard
make -C apps/server check                        # server: gofmt, vet, go test
pnpm --filter @croffledev/desktop db:generate    # new Drizzle migration
```

Database schema changes: edit the Drizzle schema, run `db:generate`, and commit the new migration under `apps/desktop/drizzle/`. Never edit a migration that is already committed.

---

## Pull requests

1. Branch from `master`: `feat/…`, `fix/…`, `refactor/…`, `chore/…`.
2. Keep changes focused; prefer package-scoped PRs when practical.
3. Open a PR and pick a template from [`.github/pull_request_template/`](./.github/pull_request_template/): default (refactor / chore), bug fix, or feature.
4. Fill **package scope** (`desktop` / `server` / `types` / `cli`) and **Electron surface** (main / preload / renderer) when relevant.
5. Before requesting review, run the checks for what you touched — the same ones CI runs: `pnpm typecheck`, `pnpm lint`, `pnpm format:check`, `pnpm -r test`, and the build of the changed package (`make -C apps/server check` for the server).
6. A change to `packages/types` or `packages/cli` needs a Changeset (`pnpm changeset`).
7. Change `package.json` and the lockfile in the same commit.

PRs are merged with **Rebase and Merge**.

Renderer screens follow [`apps/desktop/DESIGN.md`](./apps/desktop/DESIGN.md) (control heights, radius, button variants, tokens).

### Labels

[`.github/labeler.yaml`](./.github/labeler.yaml) auto-labels PRs by path and branch prefix (`feat`, `fix`, `refactor`, …). Locale / i18n file changes get the `i18n` label. Labels are created automatically if missing.

### Commit style

`type(scope): title`

- Types: `feat`, `fix`, `refactor`, `chore`, `docs`, `ci`, `release`, `test`
- Scopes: `desktop`, `server`, `types`, `cli`, `ci` — omit when the change is repo-wide

Example: `fix(desktop): keep the time when a schedule is moved to another day`.

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
3. Expose it in the Settings UI (`settings-modal.vue` language options) and add `language.<code>` to the locale JSON.
4. Wire calendar display if needed (`languageToLocale` in `calendar-settings.ts`, FullCalendar locale import when available).
5. Smoke-test: switch language in Settings → confirm renderer UI, tray menu, and a reminder notification body.

Prefer a focused PR titled like `feat(desktop): add <language> locale`. Use the Feature PR template and expect the `i18n` label.

---

## Extension development

```bash
pnpm --package=@croffledev/croffle-cli dlx croffle-plugin create my-extension
```

See [`packages/cli/README.md`](./packages/cli/README.md) for `build` and `pack`, and install `@croffledev/croffle-types` for the API typings (`ExtensionContext`, `CroffleAPI`, …) — see [`packages/types/README.md`](./packages/types/README.md).

---

## Maintainers: CI and releases

### Workflows (`.github/workflows`)

| Workflow                                    | When                                                                                       | What                                                                                                                                                                                                                                                                                                                                  |
| ------------------------------------------- | ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ci`                                        | push to `master`, PR (opened, synchronize, reopened, ready_for_review), manual             | Detects the changed scopes (`desktop`, `types`, `cli`, `server`, `serverWeb`; shared config = every Node scope) and runs only those: oxlint, oxfmt check, typecheck, test, then the build of the changed app or package. The server scope runs gofmt, vet, build and test without pnpm. **`CI result` is the single required check.** |
| `secret-scan`                               | every branch push and PR                                                                   | TruffleHog (verified + unknown) and Gitleaks over the full history                                                                                                                                                                                                                                                                    |
| `labeler`                                   | PR open / reopen / sync                                                                    | Labels by path and branch prefix; creates missing labels                                                                                                                                                                                                                                                                              |
| `publish-packages`                          | push to `master` touching `.changeset/**`, `packages/**`, `package.json`, lockfile; manual | Opens the "chore: version packages" PR, or stages the packages on npm (`scripts/stage-packages.mjs`, OIDC trusted publishing)                                                                                                                                                                                                         |
| `release` ("Croffle Release")               | manual                                                                                     | Desktop installers to a draft GitHub Release                                                                                                                                                                                                                                                                                          |
| `server-release` ("Croffle Server Release") | manual                                                                                     | Server image to ghcr — never a GitHub Release                                                                                                                                                                                                                                                                                         |

### Tags

Per product: `desktop-vX.Y.Z` and `server-vX.Y.Z`, pre-releases `…-rc.N`. Desktop tags up to `v1.2.6` are unprefixed and are never renamed. Packages get `<name>@<version>`.

### npm packages

`@croffledev/croffle-types` and `@croffledev/croffle-cli` are released with **Changesets**. The desktop is not published to npm.

1. A change to a package comes with `pnpm changeset`.
2. After the merge, `publish-packages` opens the **chore: version packages** PR. It runs no CI, so check it locally before merging:
   - only `packages/*/package.json`, `CHANGELOG.md` and the consumed `.changeset/*.md` change;
   - the versions match the Changeset bumps;
   - `pnpm install --frozen-lockfile`, the typecheck of types and cli, the cli build, and `pnpm pack` of each package pass.
3. Merging it **stages** the new versions on npm (`npm stage publish`) and puts the git tag `<name>@<version>` on that commit. Nothing is installable until a maintainer approves the staged versions on npmjs.com.
4. If a staged version is rejected, delete its tag by hand. A later run cannot tag an older commit.

### Desktop

1. Actions → **Croffle Release** → **Run workflow**.
2. Inputs: `release_type` (`patch` | `minor` | `major` | `rc`; `patch` on an rc finalizes it), optional exact `version` and `version_suffix` (e.g. `rc.1`), `draft` (default `true`), `dry_run` (build only — no commit, tag or release).
3. The workflow bumps `apps/desktop/package.json`, commits, tags `desktop-v<version>`, creates one draft Release, then the macOS, Linux and Windows jobs attach the unsigned installers: Windows `.exe`, macOS arm64 `.dmg`/`.zip`, Linux `.AppImage`/`.deb`/`.rpm`.
4. Write the note body following [`.github/release-notes.md`](./.github/release-notes.md) and publish with `gh release edit <tag> --title "<title>" --notes-file <file> --draft=false` (add `--prerelease` for an rc). Titles: `Pre-Release vX.Y.Z-rc.N` for an rc, `Croffle vX.Y.Z — Release Note` for stable.

electron-updater cannot see a draft, so a release that stays a draft never reaches users.

### Server

1. Actions → **Croffle Server Release** → **Run workflow**.
2. Inputs: `version` (exact, no `v`), optional `prerelease_suffix` (e.g. `rc.1`), `dry_run`.
3. The workflow runs the Go gates, builds the multi-arch image, pushes `ghcr.io/team-croffle/croffle-server:<version>` (plus `:latest` for a stable release) and tags `server-v<version>`.
4. The notes are committed as `docs/server/release/<version>.md` (see [`docs/server/release/README.md`](./docs/server/release/README.md)).

**Never create a GitHub Release for the server**, and never attach a server artifact to one: electron-updater resolves the repository's latest release, so a server release there would stop every desktop client from updating.

---

## Code of conduct & license

Be respectful in issues and reviews. The project is licensed under the **MIT License** (`LICENSE`).
