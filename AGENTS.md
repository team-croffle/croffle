# AGENTS.md

## Project
- Purpose: **Croffle** — a calendar-first, extensible desktop productivity app (Electron; Windows / macOS / Linux, MIT). Local-first: no cloud is required. Features are added through **extensions**.
- Stack: Electron, Vue 3, Vite (electron-vite), Tailwind CSS, TypeScript · SQLite (`better-sqlite3`) + **Drizzle** (README still mentions TypeORM; the code is the truth) · FullCalendar, rrule, Pinia, vue-i18n, electron-updater · pnpm workspaces, Changesets, oxlint, oxfmt, husky.
- Requirements: Node ≥ 24, pnpm (Corepack).
- Commands (root):
  - install: `pnpm install`
  - dev: `pnpm dev` (desktop) · `pnpm --filter @croffledev/croffle-cli dev`
  - lint: `pnpm lint` / `pnpm lint:fix` · format: `pnpm format`
  - typecheck: `pnpm typecheck` (all packages) · `pnpm typecheck:desktop`
  - test: none yet
  - build: `pnpm build` (desktop) · `pnpm build:unpack` / `build:win` / `build:mac` / `build:linux` · `pnpm --filter @croffledev/croffle-cli build`
  - db: `pnpm --filter @croffledev/desktop db:generate` / `db:studio` (drizzle-kit)
  - release: `pnpm changeset` (types/cli), desktop via GitHub Actions (see CI/CD)

## Layout
| Path | Package | Published |
|---|---|---|
| `apps/desktop` | `@croffledev/desktop` | GitHub Releases (private) |
| `packages/types` | `@croffledev/croffle-types` | npm |
| `packages/cli` | `@croffledev/croffle-cli` | npm |

- Desktop is split into `src/main` (Node: DB, IPC, extensions, reminders, tray), `src/preload` (contextBridge → `window.croffle`), `src/renderer` (Vue, sandboxed) and `src/common` (shared code, alias `@croffledev/common`).
- Host API is **domain-flat**: `croffle.window`, `croffle.calendar.schedules`, `croffle.extensions.info`, `croffle.event`, … Adding an API: IPC handler (`main/ipc/*.handler.ts`) → preload (`preload/api/**`) → public types (`packages/types`).
- Public types live in `packages/types`; runtime enums used by the app live in `apps/desktop/src/common`.
- Terminology: installable packages are **extensions**; extension options are **configuration**; app-wide preferences are **settings**. Manifest file is `croffle-manifest.json`.

## Conventions
- Formatting is owned by oxfmt (`.oxfmtrc.json`): 2 spaces, single quotes, semicolons, trailing commas, print width 100, LF. Lint rules in `.oxlintrc.json`. Don't hand-format; run `pnpm format`.
- Run `pnpm typecheck` and lint the touched files before finishing. The pre-commit hook (husky) runs typecheck + lint-staged; don't bypass it.
- Publishable package change (`packages/types`, `packages/cli`) → add a Changeset (`pnpm changeset`). Desktop is ignored by Changesets.
- Keep changes package-scoped when practical. Renderer must not use Node APIs; go through preload/IPC.
- i18n: one catalog per language in `apps/desktop/src/common/i18n/locales/{lang}.json`, shared by main and renderer. Nested keys (`settings.general.language`), `{name}` placeholders. Never add a separate renderer locale tree. Adding a language: see CONTRIBUTING.md → Internationalization.
- DB schema changes: edit the Drizzle schema, run `db:generate`, commit the migration under `apps/desktop/drizzle/`.

## Commits / PRs
- Commit subject: `type(scope): title`. Types: `feat`, `fix`, `refactor`, `chore`, `docs`, `ci`, `release`. Scopes: `desktop`, `types`, `cli`, `ci` (omit when repo-wide).
- Branch from `master`: `feat/…`, `fix/…`, `refactor/…`. Labeler (`.github/labeler.yaml`) labels PRs by path (`desktop`, `desktop-main/preload/renderer`, `i18n`, `types`, `cli`) and branch prefix.
- PR templates in `.github/pull_request_template/` (default / bug fix / feature). Fill package scope and Electron surface (main / preload / renderer).

## Docs
- Public docs (README, CONTRIBUTING, …): English at root; Korean copy next to it as `<name>.ko.md`, cross-linked. `docs/ROADMAP.md` is the public roadmap (Korean).
- `.ai/` is a gitignored local planning dir (roadmaps, plan/work/history/test/release files, conversations, interview notes). Layout and the planning / work / test / release workflows are defined in `.claude/skills/croffle-workflow/SKILL.md` and driven by `/planning-next`, `/planning-to`, `/work`, `/work-to`, `/test`, `/release`. Never move `.ai/` contents into committed rules or docs.

## File naming
- All TypeScript and Vue files: **kebab-case** (`settings-modal.vue`, `use-calendar-logic.ts`, `schedule-store.ts`). PascalCase only for class/type/component *identifiers*, not file names.
- IPC handlers: `<domain>.handler.ts`. Public type declarations: `*.d.ts` under `packages/types/{api,models}`.
- Exceptions: `apps/desktop/src/renderer/src/components/ui/**` (generated shadcn-vue, lint-ignored) and `packages/cli/templates/**`.

## Versioning
`vX.Y.Z`; pre-release `vX.Y.Z-rc.N`. Desktop version lives in `apps/desktop/package.json` (bumped by the release workflow). `packages/*` are versioned independently by Changesets.

## CI/CD (`.github/workflows`)
Workflows: `labeler`, `publish-packages`, `release`. There is no separate `ci` workflow yet; the husky pre-commit hook is the gate.
- `labeler`: on PR open/reopen/sync; `actions/labeler@v5` with `.github/labeler.yaml`, creates missing labels.
- `publish-packages`: on `master` push touching `.changeset/**`, `packages/**`, `package.json`, lockfile, or manual. `changesets/action` opens a "chore: version packages" PR or publishes `@croffledev/croffle-types` / `@croffledev/croffle-cli` to npm.
- `release` ("Croffle Release"): manual only. Inputs: `release_type` (patch|minor|major|rc), `version` (exact, no `v`), `version_suffix` (e.g. `rc.1`), `draft` (default false), `dry_run`. Bumps `apps/desktop/package.json`, commits, tags `v<version>`, creates one draft Release first, then mac/linux/win electron-builder jobs attach unsigned installers. Note: electron-updater cannot see draft releases.
