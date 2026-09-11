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
  work/<version>_<NN>-<task>.md      one work item per file, NN = 01, 02, … ; deleted when finished
  history/<YYYY-MM-DD-HHmm>_<task>.md  written when a work item finishes; first line `decisions: …`
  test/<YYYY-MM-DD-HHmm>_<version>.md  test run report
  release/<version>_<Release|Pre-Release>.md   release notes + checklist
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
| NN | 작업 | 영역 (main/preload/renderer/common/types/cli/ci/docs) | 의존 | 상태 |
|---|---|---|---|---|

## 검증 계획
버전 전체를 확인하는 수동 시나리오 + 자동 게이트(typecheck/lint/build).

## 릴리스 메모
release_type (patch|minor|major|rc), 공개 문서에서 갱신할 곳, 패키지 Changeset 필요 여부.
```

Planning rules: read the roadmap section **and the code it touches** before writing; every roadmap bullet maps to at least one work item; one work item = one focused commit (or a small series); order by dependency; `types` changes before `desktop` changes that need them; end with `docs` / `i18n` items when user-facing text changes.

## Work file — `.ai/work/<version>_<NN>-<task>.md`

```
---
version: <version>
id: NN
title: <제목>
area: main | preload | renderer | common | types | cli | ci | docs
depends: []            # e.g. ["01"] or ["1.1.1/03"]
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

1. Pick the item (see each command). Refuse if `status` is `blocked` or a dependency has not finished (its work file still exists); say why and stop.
2. Branch: never commit on `master`. Use the branch named in the plan (`feat/<version>-<feature>`); create it from `master` and sync (rebase) if it does not exist. Set `status: doing`.
3. Re-read the files listed in **현재 코드**; if the plan no longer matches the code, fix the work file first. Check prior decisions: `grep -l 'decisions:.*<keyword>' .ai/history/*`.
4. Implement following `AGENTS.md` conventions, ticking checklist items as they complete. Stay inside the item's scope; if something else is broken, add a new work item instead of fixing it silently.
5. Verify: `pnpm typecheck`, `pnpm lint` (and `oxfmt --check` on touched files), `pnpm build` when main/preload/renderer changed, `pnpm --filter @croffledev/croffle-cli build` when cli changed. Run the item's manual scenario when it can be checked without the user; otherwise list it as "사용자 확인 필요".
6. Add a Changeset if `packages/types` or `packages/cli` changed.
7. Commit with `type(scope): title` (scope = desktop | types | cli | ci; omit for repo-wide), one-line summary, bulleted details, `Co-Authored-By` footer only. One commit per item unless the item says otherwise. Do not push.
8. Write `.ai/history/<YYYY-MM-DD-HHmm>_<task>.md`: first line `decisions: <keyword>, …` or `decisions: none`; then summary (what was done, commit hash, actual result / errors, fixes) and a decisions section (≤10 lines, omit if none). Update the row in the plan, then **delete the work file**.
9. If no work file for the version remains, set the plan `상태: done` and say the version is ready for `/test` and `/release`.

Report at the end: item, branch, commit hash, verification results, anything left for the user.

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
- The version's feature branch is merged (Rebase and Merge), local `master` synced with `origin/master`, working tree clean, on `master`.
- The plan is `done`, no work file for the version remains, and the latest `.ai/test/*_<version>.md` has no failing gate.
- Public docs updated where the version changed behavior: `README.md` / `README.ko.md`, `docs/ROADMAP.md` ("현재 데스크톱" line), `CONTRIBUTING*.md` if the workflow changed.
- Pending Changesets exist for any `packages/*` change in this version.

Steps:
1. Write `.ai/release/<version>_<Release|Pre-Release>.md` (Pre-Release for `rc`): release type, highlights (from history files), breaking changes, manual QA summary, the exact `gh workflow run` command.
2. Update `.ai/ROADMAP.md` header ("현재 데스크톱 **<version>**") and move the version's section to a `## 완료` block, keeping later versions intact.
3. Doc changes go on a branch (`docs/<version>-release`), are committed (`docs: prepare v<version> release`), and merged the normal way after the user approves the push.
4. Trigger **Croffle Release** (`.github/workflows/release.yml`) with `gh workflow run release.yml -f release_type=<patch|minor|major|rc> [-f version=<x.y.z>] [-f version_suffix=rc.1] -f draft=false -f dry_run=false`. Releasing is outward-facing: **always ask before running it**, and offer `dry_run=true` first if anything is uncertain.
5. Watch the run (`gh run watch`), then verify the GitHub Release has win/mac/linux assets and is not a draft (electron-updater cannot see drafts). Record the run URL in the release file.
