---
name: croffle-workflow
description: Croffle-specific rules for the `.ai/` planning workspace and the planning / work / test / release workflows. Referenced by /planning-next, /planning-to, /work, /work-to, /test, /release. Read it when asked about plan or work files, "다음 작업", or release steps.
---

# Croffle workflow

Follows the global instructions (`~/.claude/CLAUDE.md`) and `AGENTS.md`; this file only adds the Croffle-specific parts. `.ai/` files are written in **Korean**, are gitignored, and are never committed. Commit messages stay English.

## Versions

- **Released version**: `version` in `apps/desktop/package.json` (bumped only by the release workflow).
- **Roadmap versions**: headings in `.ai/ROADMAP.md` (`## v1.1.1`, `## v1.2`, …). Server / docs roadmaps live in `.ai/ROADMAP.server.md`, `.ai/ROADMAP.docs.md`; the public summary is `docs/ROADMAP.md` (= `.ai/ROADMAP.pub.md`). The internal roadmap wins when they disagree.
- **Next version**: the first roadmap version, in semver order, greater than the released version that has no plan file.
- **Current work version**: the lowest version that still has a file under `.ai/work/` (work files are deleted when finished, so any remaining file is pending).
- Versions are written without `v` in file names (`1.2`, `1.1.1`, `1.3.0-rc.1`).

## `.ai/` file names (global layout, Croffle naming)

```
.ai/
  README.md                          layout description (read at session start)
  ROADMAP.md                         internal desktop roadmap (source of truth for planning)
  ROADMAP.server.md / ROADMAP.docs.md / ROADMAP.pub.md
  conversation/<date>-<topic>.md     decision logs
  plan/<version>_<feature>.md        one plan per version; <feature> = short slug of the version theme
  work/<version>-rc.<N>_<task>.md    one work item per file = one release candidate, N = 1, 2, … ; deleted when finished
  history/<YYYY-MM-DD-HHmm>_<task>.md  written when a work item finishes; first line `decisions: …`
  test/<YYYY-MM-DD-HHmm>_<version>.md  test run report
  release/<version>_<Release|Pre-Release>.md   GitHub release note body (templates: release-notes.md)
  pr/<branch>.md                     PR message when not pushing immediately
```

## Plan file — `.ai/plan/<version>_<feature>.md`

```
# <version> 계획 — <feature>
기준: ROADMAP.md `## v<version>` (최종 갱신 <date>)
상태: draft | ready | in-progress | done
브랜치: feat/<version>-<feature>   (또는 작업별 브랜치 목록)

## 조사
로드맵 항목이 닿는 코드와 현재 사실 (파일 경로).

## 의존 / 위험
types → desktop 순서, 네이티브 모듈, 마이그레이션, i18n 등.

## 접근
어떻게 나눌지, 왜 그 순서인지.

## 작업 목록
| rc | 작업 | 영역 (main/preload/renderer/common/types/cli/ci/docs) | 의존 | 상태 |
|---|---|---|---|---|

## 검증 계획
버전 전체를 확인하는 수동 시나리오 + 자동 게이트(typecheck/lint/build).

## 릴리스 메모
release_type (patch|minor|major|rc), 공개 문서에서 갱신할 곳, 패키지 Changeset 필요 여부.
```

Planning rules: read the roadmap section **and the code it touches** before writing; every roadmap bullet maps to at least one work item; **one work item = one release candidate `<version>-rc.N`** (numbered in execution order, so finishing rc.N leaves the app in a `<version>-rc.N` state that can be shipped with `/release rc` when useful); one work item = one focused commit (or a small series); order by dependency; `types` changes before `desktop` changes that need them; end with `docs` / `i18n` items when user-facing text changes.

## Work file — `.ai/work/<version>-rc.<N>_<task>.md`

```
---
version: <version>
id: rc.N
title: <제목>
area: main | preload | renderer | common | types | cli | ci | docs
depends: []            # e.g. ["rc.1"] or ["1.1.1-rc.3"] (other version)
status: todo | doing | blocked
branch: feat/<version>-<feature>
---

## 목적
## 현재 코드 (파일 경로 + 핵심 사실)
## 작업
- [ ] 단계별 체크리스트 (파일 단위)
## 기대 결과
## 예상 오류 / 주의
## 검증
- [ ] typecheck / lint / build
- [ ] 수동 확인 시나리오 (앱에서 어떻게 확인하는지)
```

## Work workflow (one work item)

1. Pick the item (see each command; the lowest pending rc.N whose dependencies have finished). Refuse if `status` is `blocked` or a dependency has not finished (its work file still exists); say why and stop.
2. Branch: never commit on `master`. Use the branch named in the plan (`feat/<version>-<feature>`); create it from `master` and sync (rebase) if it does not exist. Set `status: doing`.
3. Re-read the files listed in **현재 코드**; if the plan no longer matches the code, fix the work file first. Check prior decisions: `grep -l 'decisions:.*<keyword>' .ai/history/*`.
4. Implement following `AGENTS.md` conventions, ticking checklist items as they complete. Stay inside the item's scope; if something else is broken, add a new work item instead of fixing it silently.
5. Verify: `pnpm typecheck`, `pnpm lint` (and `oxfmt --check` on touched files), `pnpm build` when main/preload/renderer changed, `pnpm --filter @croffledev/croffle-cli build` when cli changed. Run the item's manual scenario when it can be checked without the user; otherwise list it as "사용자 확인 필요".
6. Add a Changeset if `packages/types` or `packages/cli` changed.
7. Commit with `type(scope): title` (scope = desktop | types | cli | ci; omit for repo-wide), one-line summary, bulleted details, `Co-Authored-By` footer only. One commit per item unless the item says otherwise. Do not push.
8. Write `.ai/history/<YYYY-MM-DD-HHmm>_<task>.md`: first line `decisions: <keyword>, …` or `decisions: none`; then summary (what was done, commit hash, actual result / errors, fixes) and a decisions section (≤10 lines, omit if none). Update the row in the plan, then **delete the work file**.
9. If no work file for the version remains, set the plan `상태: done` and say the version is ready for `/test` and `/release`.

Report at the end: item, branch, commit hash, verification results, anything left for the user.

## Branch → PR → merge → release (per rc)

1. **Branch**: all work for a version runs on the branch named in the plan (`fix/<version>-<feature>` or `feat/…`), created from a synced `master`. Never on `master`.
2. **≥ 3 commits per branch/PR**. If a work item finishes with fewer than 3 commits, do **not** open a PR yet: continue with the next pending work item on the same branch and ship them together as one rc. Then renumber the remaining rc.N in the plan and work files so numbering stays contiguous.
3. **Verify** with `/test` (gates always; run the app with `pnpm dev` only when a scenario cannot be judged from code, or the user asks).
4. **Publish**: sync (rebase on `origin/master`), push the branch, open the PR (`gh pr create`, template from `.github/pull_request_template/`, labels come from the labeler). Draft the body in `.ai/pr/<branch>.md` first.
5. **GitHub Actions decide**: wait for `CI` and `Secret Scan` (`gh pr checks --watch`). Red → fix on the branch, push, wait again. Never merge red.
6. **Merge**: Rebase and Merge (`gh pr merge --rebase --delete-branch`). Then sync: `git checkout master && git pull --rebase`, delete the local branch.
7. **Release the rc**: run the Release workflow with `release_type=rc`, then write `.ai/release/X.Y.Z-rc.A_Pre-Release.md` from the pre-release template and publish the draft with `gh release edit … --notes-file … --draft=false --prerelease` (title `Pre-Release vX.Y.Z-rc.A`). This is part of the loop and needs no extra approval; announce it. Version file bumps are done by the workflow, never by hand.
8. **Stable release** (`Release vX.Y.Z`, `release_type=patch|minor|major`) happens **only on an explicit user instruction** or the user's manual run. `/release` without an explicit type must not produce a stable release.
9. **Packages** (`packages/types`, `packages/cli`): code changes ship with a Changeset in the same PR. After merge, the `Publish Packages` workflow (github-actions bot) opens a "chore: version packages" PR; merge that PR (same checks) and the bot publishes to npm. Do not publish by hand.

Operating rules for the loop:

- All GitHub operations go through `gh` (`gh pr create/checks/merge`, `gh workflow run`, `gh run watch`, `gh release view`). Never use the web UI or raw API calls when `gh` can do it.
- On any error (failed gate, red check, failed run, merge conflict): first search `.ai/history/*` (`grep -l 'decisions:.*<keyword>'`) for an earlier fix, apply or adapt it, and retry. Only stop and notify the user when a decision would cause a serious problem that is hard to undo (force push over shared history, deleting released tags, publishing a wrong stable version, data loss).
- Do not wait for the workflows that run on `master` after a merge (`CI`, `Secret Scan`, `Publish Packages`); the PR checks already covered that commit. Do wait for the `Croffle Release` run when the next step depends on its result (tag, version bump, assets).

## Test workflow

There is no automated test suite yet. A test run is:

1. Gates: `pnpm install --frozen-lockfile` (if node_modules is stale), `pnpm typecheck`, `pnpm lint`, `oxfmt --check .`, `pnpm build`, `pnpm --filter @croffledev/croffle-cli build`. If a `test` script exists in any package, run `pnpm -r test` too.
2. Scenarios: for the version under test, collect the **검증** sections from the version's history files plus the plan's **검증 계획**. Run what can be run (`pnpm dev`, dev DB at `apps/desktop/dev/croffle.db`, logs at `apps/desktop/dev/logs/main.log`), and mark the rest as manual for the user.
3. Always include the smoke set: app starts, calendar renders, create/edit/delete a schedule, settings open and language switch, an extension installs from a local zip (if one is available under `apps/desktop/dev/extensions`).
4. Write `.ai/test/<YYYY-MM-DD-HHmm>_<version>.md`: gates table (pass/fail + output on failure), scenario table (pass/fail/manual), failures with file paths. Failures become new work files for the version.

Never mark a scenario passed that was not actually executed.

## Release workflow

Releasing = the desktop version in `apps/desktop/package.json` (packages are released separately by Changesets on merge).

Preconditions, all must hold or stop and report:

- The branch is merged (Rebase and Merge) with green `CI` + `Secret Scan`, local `master` synced with `origin/master`, working tree clean, on `master`.
- For an **rc**: the rc's work items are finished (history written, work files deleted) and the latest `.ai/test/*_<version>.md` has no failing gate.
- For a **stable** release: the plan is `done`, no work file for the version remains, the last rc was released, and the user explicitly asked for the stable release.
- Public docs updated where the version changed behavior: `README.md` / `README.ko.md`, `docs/ROADMAP.md` ("현재 데스크톱" line), `CONTRIBUTING*.md` if the workflow changed.
- Pending Changesets exist for any `packages/*` change in this version.

Steps:

1. Write `.ai/release/<version>_<Release|Pre-Release>.md` as the **release note body** using the templates in [release-notes.md](./release-notes.md): Korean block in `<details>`, `---`, English block, and a final `## Changelogs` section pasted from `gh api … releases/generate-notes` with its `## What's Changed` heading renamed. No commands, run URLs, or QA logs in this file.
2. Update `.ai/ROADMAP.md` header ("현재 데스크톱 **<version>**") and move the version's section to a `## 완료` block, keeping later versions intact (stable releases only).
3. Doc changes go on a branch (`docs/<version>-release`), are committed (`docs: prepare v<version> release`), and merged the normal way after the user approves the push.
4. Trigger **Croffle Release** (`.github/workflows/release.yml`) with `gh workflow run release.yml -f release_type=<patch|minor|major|rc> [-f version=<x.y.z>] [-f version_suffix=rc.1] -f dry_run=false`. `draft` defaults to **true**: the workflow uploads assets to a draft. An rc release is part of the branch loop (announce, no extra approval). A stable release needs the user's explicit instruction; if anything is uncertain, offer `dry_run=true` first.
5. Watch the run (`gh run watch`), confirm win/mac/linux assets are attached, then publish with the notes: `gh release edit <tag> --title "<title>" --notes-file .ai/release/<file> --draft=false [--prerelease]`. Titles: `Pre-Release vX.Y.Z-rc.A` / `Croffle vX.Y.Z — Release Note`. Verify `gh release view <tag> --json name,isDraft,isPrerelease,assets` shows `isDraft=false` (electron-updater cannot see drafts). Record the run URL in `.ai/history/`, not in the release file.
