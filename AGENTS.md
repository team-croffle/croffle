# AGENTS.md

<!-- reins:mode-changed done -->
<!-- reins:reconcile done -->
<!-- reins:onboarding done -->

<!-- reins:begin notice v=0.3.0 -->
> The blocks marked `reins:begin … reins:end` are written by reins from `.agents/harness/`. Do not edit them by hand; the user changes them with `reins config` or `reins update` in a terminal. Everything outside the blocks belongs to the project. Nothing in this repository runs reins while you work.
<!-- reins:end notice -->

## Project

- Purpose: **Croffle** — a calendar-first, extensible desktop productivity app (Electron; Windows / macOS / Linux, MIT). Local-first: no cloud is required. Features are added through **extensions**. An optional self-hosted Go server (`croffle-server`) adds sync between devices and team sharing.
- Stack: Electron, Vue 3, Vite (electron-vite), Tailwind CSS, TypeScript · SQLite (`better-sqlite3`) + **Drizzle** (README still mentions TypeORM; the code is the truth) · FullCalendar, rrule, Pinia, vue-i18n, electron-updater · pnpm workspaces, Changesets, oxlint, oxfmt, husky · server: Go stdlib + `modernc.org/sqlite`, Vue dashboard.
- Requirements: Node ≥ 24, pnpm 12 (`packageManager`, Corepack), Go (version in `apps/server/go.mod`) for the server.

## Goals and scope

- The roadmap is `docs/ROADMAP.md` (public, Korean; settled items only). Ideas and the working plan are in `.loctx/ROADMAP.local.md`.
- Current line: desktop **1.3** (optional sync beta, sharing, To-Do) paired with server **0.1 beta**; then server 0.2–0.4 and desktop 1.3.5 ↔ server 0.5.
- Out of scope unless the roadmap says otherwise: a server-side plugin system (server extensibility is hooks + a signed webhook), team management UI inside the desktop shell, a sync framework (the protocol is our own row-level LWW changeset).
- Work items migrated from the previous harness are listed in `.agents/intent/000_migrated` (draft); each version gets its own intent and spec before work starts.

## Commands

Root, unless noted. The gates in `.agents/harness/gates.yaml` are the ones CI runs.

- install: `pnpm install` · server dashboard: `pnpm --dir apps/server/web install` (own root and lockfile)
- dev: `pnpm dev` (desktop) · `pnpm --filter @croffledev/croffle-cli dev`
- typecheck: `pnpm typecheck` (all packages) · `pnpm typecheck:desktop`
- lint: `pnpm lint` (check) / `pnpm lint:fix` · format: `pnpm exec oxfmt --check` (check) / `pnpm format`
- test: `pnpm -r test` (desktop: `node --test` over `apps/desktop/test/**/*.test.ts` — the shared sync fixtures and the outbox triggers; no extra dependency) · server: `make -C apps/server check` (gofmt, vet, go test incl. protocol conformance)
- build: `pnpm build` (desktop) · `pnpm build:unpack` / `build:win` / `build:mac` / `build:linux` · `pnpm --filter @croffledev/croffle-cli build` · `pnpm --dir apps/server/web build` (dashboard, before `make -C apps/server build`)
- db: `pnpm --filter @croffledev/desktop db:generate` / `db:studio` (drizzle-kit)
- release: `pnpm changeset` (types/cli); desktop and server through their GitHub Actions workflows (see CI/CD)

## Layout

| Path                     | Package                     | Published                             |
| ------------------------ | --------------------------- | ------------------------------------- |
| `apps/desktop`           | `@croffledev/desktop`       | GitHub Releases (private)             |
| `apps/server`            | `croffle-server` (Go)       | ghcr image only — **never a Release** |
| `packages/types`         | `@croffledev/croffle-types` | npm                                   |
| `packages/cli`           | `@croffledev/croffle-cli`   | npm                                   |
| `packages/sync-fixtures` | —                           | not published (shared test data)      |

- `apps/server` is Go (stdlib only + `modernc.org/sqlite` + `x/crypto`), outside the pnpm workspace: tasks live in its `Makefile` (`make check`, `make build`, `make build-all`). Its dashboard (`apps/server/web`) is a Vue app with **its own pnpm root and lockfile**, embedded into the binary with `go:embed`.
- Desktop is split into `src/main` (Node: DB, IPC, extensions, reminders, tray), `src/preload` (contextBridge → `window.croffle`), `src/renderer` (Vue, sandboxed) and `src/common` (shared code, alias `@croffledev/common`).
- Host API is **domain-flat**: `croffle.window`, `croffle.calendar.schedules`, `croffle.extensions.info`, `croffle.event`, … Adding an API: IPC handler (`main/ipc/*.handler.ts`) → preload (`preload/api/**`) → public types (`packages/types`).
- Public types live in `packages/types`; runtime enums used by the app live in `apps/desktop/src/common`.
- Terminology: installable packages are **extensions**; extension options are **configuration**; app-wide preferences are **settings**. Manifest file is `croffle-manifest.json`.
- Harness: `.agents/` (committed: harness settings, intents, specs, knowledge, audits) and `.loctx/` (this machine only, gitignored: local roadmap, history, reports, release-note bodies, personal context). See the Settings and Flow blocks below and `.loctx/README.md`.

## Conventions

- Formatting is owned by oxfmt (`.oxfmtrc.json`): 2 spaces, single quotes, semicolons, trailing commas, print width 100, LF. Lint rules in `.oxlintrc.json`. Don't hand-format; run `pnpm format`. oxfmt ignores the reins-generated files (`.agents/**`, `.claude/skills/**`, `.claude/settings.json`) and `AGENTS.md`, so the formatter never rewrites managed content.
- Run the gates that cover what you touched before finishing. The pre-commit hook (husky) runs typecheck + lint-staged; don't bypass it.
- Publishable package change (`packages/types`, `packages/cli`) → add a Changeset (`pnpm changeset`). Desktop is ignored by Changesets.
- Keep changes package-scoped when practical. Renderer must not use Node APIs; go through preload/IPC.
- i18n: one catalog per language in `apps/desktop/src/common/i18n/locales/{lang}.json`, shared by main and renderer. Nested keys (`settings.general.language`), `{name}` placeholders. Never add a separate renderer locale tree. Adding a language: see CONTRIBUTING.md → Internationalization.
- DB schema changes: edit the Drizzle schema, run `db:generate`, commit the new migration under `apps/desktop/drizzle/`.
- `package.json` and its lockfile change in the same commit (`.agents/knowledge/2026-10-25_frozen-lockfile.md`).

## Commits / PRs

- Commit subject: `type(scope): title`. Types: `feat`, `fix`, `refactor`, `chore`, `docs`, `ci`, `release`, `test`. Scopes: `desktop`, `server`, `types`, `cli`, `ci` (omit when repo-wide).
- Commit at meaningful units (see Flow): each commit is a point you can revert to without losing other working work; group small items instead of splitting them.
- Branch from `master`: `feat/…`, `fix/…`, `refactor/…`, `chore/…`. Labeler (`.github/labeler.yaml`) labels PRs by path (`desktop`, `desktop-main/preload/renderer`, `i18n`, `types`, `cli`) and branch prefix.
- PR templates in `.github/pull_request_template/` (default / bug fix / feature). Fill package scope and Electron surface (main / preload / renderer).

## Docs

- Public docs (README, CONTRIBUTING, …): English at root; Korean copy next to it as `<name>.ko.md`, cross-linked. `docs/ROADMAP.md` is the public roadmap (Korean).
- Release notes: the convention for both products is `.github/release-notes.md`. Desktop note bodies are written in `.loctx/release/` and posted to the GitHub Release; server notes are committed as `docs/server/release/<version>.md`.
- Documents the user reads (`.agents/intent/`, specs, `.loctx/`) are Korean, in the polite 입니다체.
- Never move `.loctx/` contents into committed rules or docs.

## File naming

- All TypeScript and Vue files: **kebab-case** (`settings-modal.vue`, `use-calendar-logic.ts`, `schedule-store.ts`). PascalCase only for class/type/component _identifiers_, not file names.
- IPC handlers: `<domain>.handler.ts`. Public type declarations: `*.d.ts` under `packages/types/{api,models}`.
- Exceptions: `apps/desktop/src/renderer/src/components/ui/**` (generated shadcn-vue, lint-ignored) and `packages/cli/templates/**`.

## Versioning

Tags are prefixed per product: **`desktop-vX.Y.Z`** and **`server-vX.Y.Z`**, pre-release `…-rc.N`. The server release workflow already uses its prefix; the desktop workflow switches with its first 1.3.0 pre-release, so `v1.2.6` and earlier stay on the old unprefixed scheme and are never renamed.
Desktop version lives in `apps/desktop/package.json` (bumped by the release workflow); the server version is an input to its release workflow. `packages/*` are versioned independently by Changesets.

## CI/CD (`.github/workflows`)

Workflows: `ci`, `secret-scan`, `labeler`, `publish-packages`, `release`, `server-release`.

- `ci`: on `master` push / PR (opened, synchronize, reopened, ready_for_review) / manual. Detects changed scopes (`desktop`, `types`, `cli`, `server`, `serverWeb`; shared config = all Node scopes) and runs oxlint, `oxfmt --check`, typecheck only on those scopes, then builds the changed app/package (desktop: electron-vite build; cli: tsup). The `server` scope runs its own Go gates instead, so a Node change never installs Go and a Go change never installs pnpm. `CI result` is the single required check.
- `secret-scan`: on every branch push and PRs. TruffleHog (verified + unknown) and Gitleaks over the full history.
- `labeler`: on PR open/reopen/sync; `actions/labeler@v5` with `.github/labeler.yaml`, creates missing labels.
- `publish-packages`: on `master` push touching `.changeset/**`, `packages/**`, `package.json`, lockfile, or manual. `changesets/action` opens a "chore: version packages" PR or publishes `@croffledev/croffle-types` / `@croffledev/croffle-cli` to npm.
- `server-release` ("Croffle Server Release"): manual only. Inputs: `version` (exact, no `v`), `prerelease_suffix` (e.g. `rc.1`), `dry_run`. Runs the Go gates, builds the multi-arch image, pushes `ghcr.io/<owner>/croffle-server:<version>` (plus `:latest` for a stable release) and tags `server-v<version>`. **It must never create a GitHub Release**: electron-updater resolves the repository's latest release, so one here would stop every desktop client from updating. Its notes go to `docs/server/release/<version>.md`.
- `release` ("Croffle Release"): manual only. Inputs: `release_type` (patch|minor|major|rc), `version` (exact, no `v`), `version_suffix` (e.g. `rc.1`), `draft` (default false), `dry_run`. Bumps `apps/desktop/package.json`, commits, tags `v<version>`, creates one draft Release first, then mac/linux/win electron-builder jobs attach unsigned installers. The release stays a **draft** (default `draft: true`); the note body is written in `.loctx/release/` and published with `gh release edit … --draft=false` (see `.github/release-notes.md`). Titles: `Pre-Release vX.Y.Z-rc.A` for rc, `Croffle vX.Y.Z — Release Note` for stable. Note: electron-updater cannot see draft releases, so publishing is a required step.

## Branch / PR / release rules

- Work on a feature branch from `master`; never commit on `master`.
- PRs are verified by GitHub Actions (`CI`, `Secret Scan`); merge with **Rebase and Merge** only, then delete the branch.
- **rc = the preview of a finished version.** A version is cut as an rc only after it is implemented, reviewed, tested, merged and audited (Flow step 5); the rc exists to make the final adjustments before stable. Fixes after an rc are stabilisation work units, followed by the next rc. Work units are not released one by one.
- Stable releases only on the maintainer's explicit instruction.
- `packages/*` changes need a Changeset. The `chore: version packages` PR that the Changeset produces is part of the work that added it: verify it, then merge it without asking again. The bot PR runs no CI, so check it locally first — only `packages/*/package.json`, `CHANGELOG.md` and the consumed `.changeset/*.md` change, versions match the Changeset bumps, `pnpm install --frozen-lockfile`, typecheck of types and cli, the cli build, and `pnpm pack` of each package. Publishing goes to npm staging; ask the user to approve the staged release there.
- After a merge, delete the branch on both sides: the remote one (merge with `--delete-branch`) and the local one.

## Red lines

In addition to `.agents/harness/policy/common-red-lines.yaml`:

- Secret files are never read or edited (C4 patterns; denied in `.claude/settings.json`).
- Never edit or delete a migration already committed under `apps/desktop/drizzle/` (or the server's embedded SQL migrations): installed databases have applied them. A schema change is always a new migration.
- `server-release` never creates a GitHub Release, and no server artifact is attached to one.
- Never rename, move or delete existing tags (`v1.2.6` and earlier stay unprefixed).
- Never commit `.loctx/`.
- Reins-managed files (`.agents/harness/**`, the `reins:` blocks of this file, `.claude/skills/**`, `.claude/settings.json`) change only through `reins config` / `reins update`, or with the user's explicit permission for that edit.

<!-- reins:begin settings v=0.3.0 -->
## Settings

- `.agents/harness/bindings.yaml` — the mode (`single` or `multi`), which model does which role group, and `audit.required`. A personal override may sit in `.loctx/bindings.local.yaml`; when it exists, it wins, and when it changes who does what, `.loctx/delegation.local.md` holds that person's delegation table.
- **When `audit.required` is true (the default), every version ends with an audit (skill `audit`). It is mandatory: never skip it; only the user switches it off.**
- `.agents/harness/gates.yaml` — the project's own check commands; the hooks and the `test` skill run them as they are.
- `.agents/harness/roles/<group>.md` — what each role does, must not do, and which tools it may use.
<!-- reins:end settings -->

<!-- reins:begin flow v=0.3.0 -->
## Flow

Follow these steps in order; each has a skill in `.agents/skills/`, and each skill names the next.

1. `roadmap` — ideas go to `.loctx/ROADMAP.local.md`; once settled with the user, to `docs/ROADMAP.md`.
2. `intent` — the PM writes the direction and the plan; **the user approves**.
3. `spec` — the spec, its work units (W-n) and checks (V-n); **the user answers 진행 or 수정 on its summary, every time**.
4. `implement` → `review` — commit at meaningful units (a point you can revert or reset to without taking other working work or features with it), each naming its spec items; changing the spec's work units needs the spec changed and approved first.
5. `test` → `merge` → **`audit`** (after the merge, before the rc) → `release` → `retro` (after stable), then delete the version's working files.

The first conversation after installation (or after a reconcile or mode-changed marker appears in this file) starts with `onboard`.
<!-- reins:end flow -->

<!-- reins:begin delegation v=0.3.0 -->
## Delegation

If `.loctx/delegation.local.md` exists, follow it instead of this section (your personal settings; `reins config --local` writes it).

Mode: **multi** — you are the PM (claude claude-opus-5-5). Hand a role its work with the list of files and decisions it needs; every report comes back to you, and a role never calls another model itself. Roles not listed are yours.

| role | model | how to call |
|---|---|---|

Routine review takes one reviewer; `pl/discussion` and `advisor/committee` (every listed model, without seeing each other) are for large decisions, audit remedies, incidents and unresolved disagreements.

Audit: **required every version**; run it on claude claude-opus-5-5 in a fresh run that gets only the records, never this conversation.
<!-- reins:end delegation -->

<!-- reins:begin records v=0.3.0 -->
## Approvals and records

- Record an approval by quoting the user in the intent's or spec's `approvals:` — `{ role: user, by, at, words, rev }` — and commit it. Never record an approval the user did not give.
- A step or approval the user tells you to skip goes to the intent's `waivers:` — `{ step, words, at, scope: once }` — at that moment. It applies once; next time, follow the flow.
- Every file you create has an owner and a deletion point (each skill lists them). Working files under `.loctx/reports/<version>/` are summarised in history and the audit, then deleted after the retro.
<!-- reins:end records -->

<!-- reins:begin stops v=0.3.0 -->
## When to stop and ask

- **Ask the user and stop:** an irreversible action (force push, deleting a tag, rewriting the default branch, data or production changes); a stable release; anything that leaves the machine (push, PR, release) unless the settings or the user allow it; scope moving outside the intent and spec; a conflict with a past decision; a next step that needs an approval not yet given.
- **Stop and find the cause:** two failures in the same work unit; empty or doubtful check output; a pass without evidence (a pass needs the raw output's path).
- **When the user interrupts** (chat, Ctrl+C, session end): stop and write the resume point in `.loctx/history/<version>.md`.
- What no role does: `.agents/harness/policy/common-red-lines.yaml`.
<!-- reins:end stops -->

<!-- reins:begin language v=0.3.0 -->
## Language

`AGENTS.md`, code, commits, `.agents/` sources: English. Documents the user reads (`intent/`, `.loctx/`, `docs/*.ko.md` if any): ko.
<!-- reins:end language -->
