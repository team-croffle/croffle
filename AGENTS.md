# AGENTS.md

<!-- reins:tidy done -->

<!-- reins:mode-changed done -->
<!-- reins:reconcile done -->
<!-- reins:onboarding done -->

<!-- reins:begin notice v=0.3.0 -->
> The blocks marked `reins:begin … reins:end` are written by reins from `.agents/harness/`. Do not edit them by hand; the user changes them with `reins config` or `reins update` in a terminal. Everything outside the blocks belongs to the project. Nothing in this repository runs reins while you work.
<!-- reins:end notice -->

## Project

- Purpose: **Croffle** — a calendar-first, extensible desktop productivity app (Electron; Windows / macOS / Linux, MIT). Local-first: no cloud is required. Features are added through **extensions**. An optional self-hosted Go server (`croffle-server`) adds sync between devices and team sharing.
- Stack: Electron, Vue 3, Vite (electron-vite), Tailwind CSS, TypeScript · SQLite (`better-sqlite3`) + **Drizzle** · FullCalendar, rrule, Pinia, vue-i18n, electron-updater · pnpm workspaces, Changesets, oxlint, oxfmt, husky · server: Go (stdlib first, a few vetted libraries) + `modernc.org/sqlite`, Vue dashboard.
- Requirements: Node ≥ 24, pnpm 12 (`packageManager`, Corepack), Go (version in `apps/server/go.mod`) for the server.

## Goals and scope

- The roadmap is `docs/ROADMAP.md` (public, Korean; settled items only). Ideas and the working plan are in `.loctx/ROADMAP.local.md`.
- Out of scope unless the roadmap says otherwise: a server-side plugin system (server extensibility is hooks + a signed webhook), team management UI inside the desktop shell, a sync framework (the protocol is our own row-level LWW changeset).
- Each version gets its own intent and spec before work starts; the order of versions is the roadmap's.

## Commands

Root, unless noted. The gates in `.agents/harness/gates.yaml` are the ones CI runs.

- install: `pnpm install` · server dashboard: `pnpm --dir apps/server/web install` (own root and lockfile)
- dev: `pnpm dev` (desktop) · `pnpm --filter @croffledev/croffle-cli dev`
- typecheck: `pnpm typecheck` (all packages) · `pnpm typecheck:desktop`
- lint: `pnpm lint` (check) / `pnpm lint:fix` · format: `pnpm format:check` (check) / `pnpm format`
- test: `pnpm -r test` (desktop: `node --test` over `apps/desktop/test/**/*.test.ts`) · server: `make -C apps/server check` (gofmt, vet, go test incl. protocol conformance)
- build: `pnpm build` (desktop) · `pnpm build:unpack` / `build:win` / `build:mac` / `build:linux` · `pnpm --filter @croffledev/croffle-cli build` · `pnpm --dir apps/server/web build` (dashboard, before `make -C apps/server build`)
- db: `pnpm --filter @croffledev/desktop db:generate` / `db:studio` (drizzle-kit)
- release: `pnpm changeset` (types/cli); desktop and server through their GitHub Actions workflows (see CONTRIBUTING.md → Maintainers)

## Layout

| Path                     | Package                     | Published                             |
| ------------------------ | --------------------------- | ------------------------------------- |
| `apps/desktop`           | `@croffledev/desktop`       | GitHub Releases (private)             |
| `apps/server`            | `croffle-server` (Go)       | ghcr image only — **never a Release** |
| `packages/types`         | `@croffledev/croffle-types` | npm                                   |
| `packages/cli`           | `@croffledev/croffle-cli`   | npm                                   |
| `packages/sync-fixtures` | —                           | not published (shared test data)      |

- `apps/server` is Go — stdlib first, plus a few vetted libraries listed in `apps/server/README.md` → Dependencies (each new one is justified in its spec) — outside the pnpm workspace: tasks live in its `Makefile` (`make check`, `make build`, `make build-all`). Its dashboard (`apps/server/web`) is a Vue app with **its own pnpm root and lockfile**, embedded into the binary with `go:embed`.
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
- Renderer UI follows `apps/desktop/DESIGN.md` (control heights, radius, button variants, input widths, tokens). Read it before adding or changing a screen; when a screen needs something it does not cover, extend that file in the same change.
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

Tags are prefixed per product: **`desktop-vX.Y.Z`** and **`server-vX.Y.Z`**, pre-release `…-rc.N`. Desktop tags up to `v1.2.6` are unprefixed and stay so.
Desktop version lives in `apps/desktop/package.json` (bumped by the release workflow); the server version is an input to its release workflow. `packages/*` are versioned independently by Changesets.

## CI/CD (`.github/workflows`)

What each workflow does, its inputs and the release steps: CONTRIBUTING.md → Maintainers. The rules:

- `CI result` is the single required check. `ci` runs only the changed scopes; the Go scope never installs pnpm and the Node scopes never install Go.
- `server-release` publishes only the ghcr image and the `server-v<version>` tag. **It never creates a GitHub Release**: electron-updater resolves the repository's latest release, so one there would stop every desktop client from updating. Its notes go to `docs/server/release/<version>.md`.
- `release` (desktop) leaves the GitHub Release a **draft** (`draft: true` by default). electron-updater cannot see a draft, so publishing it with the note body from `.loctx/release/` (`.github/release-notes.md`) is a required step.
- `publish-packages` only **stages** `packages/*` on npm; a staged version goes live when a maintainer approves it on npmjs.com.

## Branch / PR / release rules

- Work on a feature branch from `master`; never commit on `master`.
- PRs are verified by GitHub Actions (`CI`, `Secret Scan`); merge with **Rebase and Merge** only, then delete the branch.
- **rc = the preview of a finished version.** A version is cut as an rc only after it is implemented, reviewed, tested, merged and audited (Flow step 5); the rc exists to make the final adjustments before stable. Fixes after an rc are stabilisation work units, followed by the next rc. Work units are not released one by one.
- Stable releases only on the maintainer's explicit instruction.
- `packages/*` changes need a Changeset. The `chore: version packages` PR that the Changeset produces is part of the work that added it: verify it locally (it runs no CI; the checklist is in CONTRIBUTING.md → Maintainers → npm packages), then merge it without asking again, and ask the user to approve the staged versions on npmjs.com.
- After a merge, delete the branch on both sides: the remote one (merge with `--delete-branch`) and the local one.
- **Review covers what is merged.** Every change of content — code, tests, spec text, release documents — comes before the last review. After it, only audit records (`.agents/audit/*.md`) are committed. A review that raises findings is followed by a review of the fix. A waiver for a direct push to master skips the branch and the PR, never the review.
- **Review ledger.** Each review adds a line to `.loctx/reports/review-ledger.tsv`: `<PR number or master>\t<reviewed full SHA>\t<review file>\t<PASS|PASS-with-findings|FAIL>`. The Claude Code hook (`.claude/hooks/premerge-review-check.sh`) refuses `gh pr merge` — and a direct push to master — when the head changed beyond audit records since the last reviewed SHA. Run the same check before handing a PR to its first audit. Never edit the ledger to pass the check; review the head instead. The Bash hooks accept a push or a merge only in its exact form, alone (`git push [-u] [-q] <remote> <branch>`, `gh pr merge <n> [--rebase] [--delete-branch] --match-head-commit <full head SHA>`), and refuse any other command in which the word `push` or `merge` appears; text that mentions them goes through a file (`git commit -F <file>`, `gh pr comment --body-file`).
- **Accepting a repeated finding.** When an audit finding of the same kind comes back a second time, it is closed only with a mechanical remedy (a hook, a check, a CI job), not with a note to remember. A remedy deferred to later (for example "in reins") is an open item in `.loctx/ROADMAP.local.md` until it is checked in the files.

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
- **When `audit.required` is true (the default), every version is audited twice (skill `audit`): a first audit before the merge and a second audit after the release. Both are mandatory: never skip one; only the user switches them off.**
- `.agents/harness/gates.yaml` — the project's own check commands; the hooks and the `test` skill run them as they are.
- `.agents/harness/roles/<group>.md` — what each role does, must not do, and which tools it may use.
<!-- reins:end settings -->

<!-- reins:begin flow v=0.3.0 -->
## Flow

Follow these steps in order; each has a skill in `.agents/skills/`, and each skill names the next.

1. `roadmap` — ideas go to `.loctx/ROADMAP.local.md`; once settled with the user, to `docs/ROADMAP.md`.
2. `intent` — the PM writes the direction and the plan; **the user approves**.
3. `spec` — the spec, its work units (W-n) and checks (V-n); **the user answers 진행 or 수정 on its summary, every time**.
4. `implement` → `review` — commit at meaningful units (a point you can revert or reset to without taking other working work or features with it), each naming its spec items. Adding, dropping or rescoping a work unit needs the spec changed and approved first; a smaller adjustment goes on, is recorded in the spec (`A-n`) and is reported at the end of the flow.
5. `test` → `merge` (push, PR, CI) → **`audit`** (the first audit: risk, before the merge) — from the accepted spec to the first audit this is **one flow without asking**: do not stop for permission between these steps; report once at its end. Then the merge on the user's instruction → `release` (the rc the way the repository releases; stable on the user's instruction) → **`audit`** (the second audit: procedure only, handed to the retro) → `retro`, then delete the version's working files.

A loop is the work the user started — usually one version — up to its retro. **The retro ends the loop**: report that it is closed and stop; the next `intent` is written when the user starts the next work. If the user named another end ("go on to …", "do not stop"), follow it for that instruction only and quote it in `.loctx/ROADMAP.local.md`; the intent's approval, the spec's 진행 or 수정, every merge and the stable release still wait for the user. Ideas for later work go to the roadmap whenever they come. Naming a version is not an instruction to start it.

The first conversation after installation, and any conversation that finds a pending `reins:` marker in this file (onboarding, reconcile, mode-changed, tidy), starts with `onboard`.
<!-- reins:end flow -->

<!-- reins:begin delegation v=0.3.0 -->
## Delegation

If `.loctx/delegation.local.md` exists, follow it instead of this section (your personal settings; `reins config --local` writes it).

Mode: **multi** — you are the PM — the model in this conversation. Hand a role its work with the list of files and decisions it needs; every report comes back to you, and a role never calls another model itself. Roles not listed are yours.

| role | model | how to call |
|---|---|---|

Routine review takes one reviewer; `pl/discussion` and `advisor/committee` (every listed model, without seeing each other) are for large decisions, audit remedies, incidents and unresolved disagreements.

Audit: **required every version, before the merge and after the release**; run it in a fresh run of your own model (or another one the user chooses) that gets only the records, never this conversation.
<!-- reins:end delegation -->

<!-- reins:begin records v=0.3.0 -->
## Approvals and records

- Record an approval by quoting the user in the intent's or spec's `approvals:` — `{ role: user, by, at, words, rev }` — and commit it. Never record an approval the user did not give.
- A step or approval the user tells you to skip goes to the intent's `waivers:` — `{ step, words, at, scope: once }` — at that moment. It applies once; next time, follow the flow.
- Every file you create has an owner and a deletion point (each skill lists them). Working files under `.loctx/reports/<version>/` are summarised in history and the audit, then deleted after the retro.
- `.agents/audit/<version>.md` holds both audits of a version; an adjustment made while working is a line in the spec's adjustment record (`A-n`).
<!-- reins:end records -->

<!-- reins:begin stops v=0.3.0 -->
## When to stop and ask

- **Ask the user and stop:** an irreversible action (force push, deleting a tag, rewriting the default branch, data or production changes); a merge; a stable release; work that would touch a red line; a work unit that has to be added, dropped or rescoped; scope moving outside the intent and spec; a conflict with a past decision; a next step that needs an approval not yet given. Pushing a branch of the work in progress and opening its PR is part of the flow (an accepted spec's work units, their records, release documents); taking anything else out of the machine still needs the user's word.
- **While you wait for the user** — a release approval, a merge instruction, a stable instruction — report and wait; do not start another unit's work. Stabilising the same version is this loop's work.
- **The retro ends the loop:** report that it is closed and stop.
- **Stop and find the cause:** two failures in the same work unit; empty or doubtful check output; a pass without evidence (a pass needs the raw output's path).
- **When the user interrupts** (chat, Ctrl+C, session end): stop and write the resume point in `.loctx/history/<version>.md`.
- What no role does: `.agents/harness/policy/common-red-lines.yaml`.
<!-- reins:end stops -->

<!-- reins:begin language v=0.3.0 -->
## Language

`AGENTS.md`, code, commits, `.agents/` sources: English. Documents the user reads (`intent/`, `.loctx/`, `docs/*.ko.md` if any): ko.
<!-- reins:end language -->
