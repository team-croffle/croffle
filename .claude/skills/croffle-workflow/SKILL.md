---
name: croffle-workflow
description: Shared definitions for Croffle's `.ai/` planning workspace and the planning / work / test / release workflows. Referenced by /planning-next, /planning-to, /work, /work-to, /test, /release. Read it when asked about plan or work files, "다음 작업", or release steps.
---

# Croffle workflow

Read `AGENTS.md` first. All files under `.ai/` are written in **Korean**, are gitignored, and are never committed. Commit messages stay English (`type(scope): title`).

## Versions

- **Released version**: `version` in `apps/desktop/package.json` (bumped only by the release workflow).
- **Roadmap versions**: headings in `.ai/ROADMAP.md` (`## v1.1.1`, `## v1.2`, …). Server / docs roadmaps live in `.ai/ROADMAP.server.md`, `.ai/ROADMAP.docs.md`; the public summary is `docs/ROADMAP.md` (= `.ai/ROADMAP.pub.md`). The internal roadmap wins when they disagree.
- **Next version**: the first roadmap version, in semver order, greater than the released version that has no `.ai/plan/<version>.md`.
- **Current work version**: the lowest version under `.ai/work/` with any item not `done`.
- Versions are written without `v` in file names (`1.2`, `1.1.1`, `1.3.0-rc.1`).

## `.ai/` layout

```
.ai/
  ROADMAP.md                internal desktop roadmap (source of truth for planning)
  ROADMAP.server.md / ROADMAP.docs.md / ROADMAP.pub.md
  conversation/             decision logs (<date>-<topic>.md)
  plan/<version>.md         one plan per version
  work/<version>/NN-<slug>.md   one work item per file, NN = 01, 02, …
  history/<date>-<version>-NN-<slug>.md   log written when a work item is done
  test/<date>-<version>.md  test run report
  release/<version>.md      release checklist + notes
  pr/<branch>.md            PR description drafts (optional)
```

## Plan file — `.ai/plan/<version>.md`

```
# <version> 계획
기준: ROADMAP.md `## v<version>` (최종 갱신 <date>)
상태: draft | ready | in-progress | done

## 목표
한두 문장. 이 버전이 끝나면 사용자가 무엇을 할 수 있는가.

## 범위
- 포함: 로드맵 항목을 그대로 옮기고, 애매한 건 코드 확인 후 구체화
- 제외: 다음 버전으로 미루는 것과 이유

## 작업 목록
| NN | 작업 | 영역 (main/preload/renderer/common/types/cli/ci/docs) | 의존 | 상태 |
|---|---|---|---|---|

## 검증 계획
버전 전체를 확인하는 수동 시나리오 + 자동 게이트(typecheck/lint/build).

## 릴리스 메모
release_type (patch|minor|major|rc), 공개 문서에서 갱신할 곳, 패키지 Changeset 필요 여부.
```

Planning rules: read the roadmap section **and the code it touches** before writing; every roadmap bullet maps to at least one work item; split so that one work item is one focused commit (or a small series); order by dependency; put `types` changes before `desktop` changes that need them; end every plan with `docs` / `i18n` items when user-facing text changes.

## Work file — `.ai/work/<version>/NN-<slug>.md`

```
---
version: <version>
id: NN
title: <제목>
area: main | preload | renderer | common | types | cli | ci | docs
depends: []            # e.g. ["01"] or ["1.1.1/03"]
status: todo | doing | done | blocked
commit: ""             # filled when done
---

## 목적
## 현재 코드 (파일 경로 + 핵심 사실)
## 변경
- [ ] 단계별 체크리스트 (파일 단위)
## 검증
- [ ] typecheck / lint / build
- [ ] 수동 확인 시나리오 (앱에서 어떻게 확인하는지)
## 메모
```

## Work workflow (one work item)

1. Pick the item (see each command). Refuse if `status` is `blocked` or a dependency is not `done`; say why and stop.
2. Set `status: doing`. Re-read the files listed in **현재 코드**; if the plan no longer matches the code, fix the work file first and note it under **메모**.
3. Implement following `AGENTS.md` conventions. Stay inside the item's scope; if something else is broken, add a new work item instead of fixing it silently.
4. Verify: `pnpm typecheck`, `pnpm lint` (and `oxfmt --check` on touched files), `pnpm build` when main/preload/renderer changed, `pnpm --filter @croffledev/croffle-cli build` when cli changed. Run the item's manual scenario when it can be checked without the user; otherwise list it as "사용자 확인 필요".
5. Add a Changeset if `packages/types` or `packages/cli` changed.
6. Commit with `type(scope): title` (scope = desktop | types | cli | ci; omit for repo-wide). One commit per item unless the item says otherwise. Do not push.
7. Set `status: done`, fill `commit`, tick the checklist, update the row in `.ai/plan/<version>.md`, and write `.ai/history/<date>-<version>-NN-<slug>.md` (what changed, files, verification result, open questions).
8. If the version has no remaining `todo`/`doing` items, set the plan `상태: done` and say the version is ready for `/test` and `/release`.

Report at the end: item, commit hash, verification results, anything left for the user.

## Test workflow

There is no automated test suite yet. A test run is:

1. Gates: `pnpm install --frozen-lockfile` (if node_modules is stale), `pnpm typecheck`, `pnpm lint`, `oxfmt --check .`, `pnpm build`, `pnpm --filter @croffledev/croffle-cli build`. If a `test` script exists in any package, run `pnpm -r test` too.
2. Scenarios: for the version under test, collect the **검증** sections of every `done` work item plus the plan's **검증 계획**. Run what can be run (`pnpm dev`, dev DB at `apps/desktop/dev/croffle.db`, logs at `apps/desktop/dev/logs/main.log`), and mark the rest as manual for the user.
3. Always include the smoke set: app starts, calendar renders, create/edit/delete a schedule, settings open and language switch, an extension installs from a local zip (if one is available under `apps/desktop/dev/extensions`).
4. Write `.ai/test/<date>-<version>.md`: gates table (pass/fail + output on failure), scenario table (pass/fail/manual), failures with file paths. Failures become new work items in `.ai/work/<version>/`.

Never mark a scenario passed that was not actually executed.

## Release workflow

Releasing = the desktop version in `apps/desktop/package.json` (packages are released separately by Changesets on merge).

Preconditions, all must hold or stop and report:
- Working tree clean, on `master`, in sync with `origin/master`.
- `.ai/plan/<version>.md` is `done`; every work item is `done`; the latest `.ai/test/*-<version>.md` has no failing gate.
- Public docs updated where the version changed behavior: `README.md` / `README.ko.md`, `docs/ROADMAP.md` ("현재 데스크톱" line), `CONTRIBUTING*.md` if the workflow changed.
- Pending Changesets exist for any `packages/*` change in this version.

Steps:
1. Write `.ai/release/<version>.md`: release type, highlights (from history files), breaking changes, manual QA summary, the exact `gh workflow run` command.
2. Update `.ai/ROADMAP.md` header ("현재 데스크톱 **<version>**") and move the version's section to a `## 완료` block, keeping later versions intact.
3. Commit docs/roadmap changes (`docs: prepare v<version> release`) and push after the user confirms.
4. Trigger **Croffle Release** (`.github/workflows/release.yml`) with `gh workflow run release.yml -f release_type=<patch|minor|major|rc> [-f version=<x.y.z>] [-f version_suffix=rc.1] -f draft=false -f dry_run=false`. Triggering a release is outward-facing: **always ask before running it**, and offer `dry_run=true` first if anything is uncertain.
5. Watch the run (`gh run watch`), then verify the GitHub Release has win/mac/linux assets and is not a draft (electron-updater cannot see drafts). Record the run URL in the release file.
