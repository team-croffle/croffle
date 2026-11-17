# Croffle에 기여하기

기여해 주셔서 감사합니다. 이 문서는 로컬 개발, 검사, PR, 그리고 관리자를 위한 CI와 릴리스 방법을 정리합니다.

English: [CONTRIBUTING.md](./CONTRIBUTING.md)

---

## 준비물

- **Node.js ≥ 24**
- **pnpm 12** (`packageManager`에 고정, `corepack enable`로 활성화)
- 데스크톱 앱 빌드 시 네이티브 모듈(`better-sqlite3`)용 OS 툴체인
- **Go** (`apps/server/go.mod`의 버전) — `apps/server`를 다룰 때만

---

## 저장소 구조

| 경로                     | 패키지                      | 설명                                               |
| ------------------------ | --------------------------- | -------------------------------------------------- |
| `apps/desktop`           | `@croffledev/desktop`       | Electron 앱 (private, 설치 파일은 GitHub Releases) |
| `apps/server`            | `croffle-server` (Go)       | 선택적 동기화 서버, 컨테이너 이미지는 ghcr에만     |
| `apps/server/web`        | —                           | 서버 대시보드 (Vue), 별도 pnpm 루트·lockfile       |
| `packages/types`         | `@croffledev/croffle-types` | 확장용 배포 타입 정의                              |
| `packages/cli`           | `@croffledev/croffle-cli`   | 배포용 확장 CLI                                    |
| `packages/sync-fixtures` | —                           | 데스크톱·서버가 함께 읽는 동기화 프로토콜 시나리오 |

`apps/server`는 pnpm 워크스페이스 밖에 있고, 작업 명령은 `Makefile`에 있습니다. [`apps/server/README.ko.md`](./apps/server/README.ko.md)의 개발 절을 참고하세요.

데스크톱은 `src/main`(Node: 데이터베이스, IPC, 확장, 리마인더, 트레이), `src/preload`(contextBridge → `window.croffle`), `src/renderer`(Vue, 샌드박스), `src/common`(공유 코드, alias `@croffledev/common`)으로 나뉩니다. renderer는 Node API를 쓰지 않고 preload와 IPC를 거칩니다.

호스트 API는 **도메인 단위 flat** 형태입니다.

```ts
croffle.window;
croffle.calendar.schedules;
croffle.extensions.info;
croffle.event;
// …
```

API 추가 순서: IPC 핸들러(`src/main/ipc/<domain>.handler.ts`) → preload(`src/preload/api/**`) → 공개 타입(`packages/types`).

용어: 설치 단위는 **extension**입니다. 확장 옵션은 **configuration**, 앱 전역 설정은 **settings**(`croffle.settings`)입니다. 확장 패키지 매니페스트는 `croffle-manifest.json`입니다.

타입은 `@croffledev/croffle-types`에, 앱 런타임 enum은 `apps/desktop/src/common`에 있습니다.

---

## 시작하기

```bash
git clone https://github.com/team-croffle/croffle.git
cd croffle
pnpm install
pnpm dev                 # 데스크톱 개발 모드
```

서버 대시보드는 `pnpm --dir apps/server/web install`로 따로 설치합니다.

루트 주요 스크립트:

| 스크립트                      | 용도                                |
| ----------------------------- | ----------------------------------- |
| `pnpm dev`                    | 데스크톱 개발 실행                  |
| `pnpm build`                  | 데스크톱 빌드 (electron-vite)       |
| `pnpm build:unpack`           | 설치 파일 없이 데스크톱 빌드·패키징 |
| `pnpm typecheck`              | 워크스페이스 전체 타입체크          |
| `pnpm lint` / `pnpm lint:fix` | oxlint                              |
| `pnpm format:check`           | 포맷 검사 (oxfmt)                   |
| `pnpm format`                 | 전체 포맷 (oxfmt)                   |
| `pnpm -r test`                | 테스트 (데스크톱: `node --test`)    |
| `pnpm changeset`              | 배포 패키지용 Changeset 추가        |

그 밖의 명령:

```bash
pnpm --filter @croffledev/croffle-cli build      # cli (tsup)
pnpm --dir apps/server/web build                 # 서버 대시보드
make -C apps/server check                        # 서버: gofmt, vet, go test
pnpm --filter @croffledev/desktop db:generate    # 새 Drizzle 마이그레이션
```

DB 스키마를 바꿀 때는 Drizzle 스키마를 고치고 `db:generate`를 실행한 뒤, `apps/desktop/drizzle/` 아래 새 마이그레이션을 커밋합니다. 이미 커밋된 마이그레이션은 고치지 않습니다.

---

## Pull Request

1. `master`에서 브랜치를 만듭니다: `feat/…`, `fix/…`, `refactor/…`, `chore/…`.
2. 변경 범위를 좁게 유지하고, 가능하면 패키지 단위로 PR을 나눕니다.
3. [`.github/pull_request_template/`](./.github/pull_request_template/)에서 템플릿을 고릅니다: default(refactor / chore), bug fix, feature.
4. 해당하면 **package scope**(`desktop` / `server` / `types` / `cli`)와 **Electron surface**(main / preload / renderer)를 채웁니다.
5. 리뷰 요청 전에 바꾼 부분의 검사를 돌립니다. CI와 같은 검사입니다: `pnpm typecheck`, `pnpm lint`, `pnpm format:check`, `pnpm -r test`, 바뀐 패키지의 빌드(서버는 `make -C apps/server check`).
6. `packages/types`나 `packages/cli`를 바꾸면 Changeset을 추가합니다(`pnpm changeset`).
7. `package.json`과 lockfile은 같은 커밋에서 바꿉니다.

PR은 **Rebase and Merge**로만 머지합니다.

renderer 화면은 [`apps/desktop/DESIGN.md`](./apps/desktop/DESIGN.md)(컨트롤 높이, radius, 버튼 종류, 토큰)를 따릅니다.

### 라벨

[`.github/labeler.yaml`](./.github/labeler.yaml)이 경로와 브랜치 prefix(`feat`, `fix`, `refactor`, …)에 따라 라벨을 붙입니다. locale / i18n 파일 변경에는 `i18n` 라벨이 붙습니다. 없는 라벨은 자동으로 만듭니다.

### 커밋 메시지

`type(scope): title`

- type: `feat`, `fix`, `refactor`, `chore`, `docs`, `ci`, `release`, `test`
- scope: `desktop`, `server`, `types`, `cli`, `ci` — 저장소 전체에 걸친 변경이면 생략

예: `fix(desktop): keep the time when a schedule is moved to another day`

---

## 언어 및 번역 (i18n)

**main·renderer** UI 문구는 **언어당 카탈로그 하나**에 모읍니다.

| 경로                                               | 역할                                                               |
| -------------------------------------------------- | ------------------------------------------------------------------ |
| `apps/desktop/src/common/i18n/locales/{lang}.json` | 공유 메시지 카탈로그 (`en`, `ko`, …)                               |
| `apps/desktop/src/common/i18n/index.ts`            | main/공유용 `t()` · `localeMessages`                               |
| `apps/desktop/src/main/i18n.ts`                    | `settings.general.language`를 읽는 main 헬퍼                       |
| `apps/desktop/src/renderer/src/i18n/index.ts`      | vue-i18n 연결 (`localeMessages`를 `@croffledev/common`에서 import) |

`renderer/.../locales`를 따로 만들지 마세요. 키는 중첩 구조(`tray.openWindow`, `settings.general.language` 등)를 유지하고, 플레이스홀더는 `{name}` 형식으로 통일합니다.

### 언어 추가 PR (환영)

지금은 **앱 내장** 언어만 지원합니다(확장으로 언어 팩을 넣는 방식은 아직 없음). 번역 PR은 보통 다음을 포함합니다.

1. `en.json`을 복사해 `apps/desktop/src/common/i18n/locales/<code>.json`을 만들고 값을 번역합니다(키는 빠짐없이).
2. 로케일 등록:
   - `apps/desktop/src/common/enums.ts`의 `AppSettingLanguage`
   - `packages/types/models/app-settings.d.ts`의 `AppSettingLanguage` (배포 패키지 — Changeset 추가)
   - `apps/desktop/src/common/i18n/index.ts`의 `localeMessages` / `resolveAppLocale`
3. 설정 UI 옵션(`settings-modal.vue`)과 locale JSON의 `language.<code>`를 추가합니다.
4. 필요하면 캘린더 표시를 연동합니다(`calendar-settings.ts`의 `languageToLocale`, FullCalendar locale).
5. 스모크 테스트: 설정에서 언어 전환 → renderer UI, 트레이 메뉴, 리마인더 알림 본문 확인.

PR 제목 예: `feat(desktop): add <language> locale`. Feature 템플릿을 쓰면 `i18n` 라벨이 붙습니다.

---

## 확장 개발

```bash
pnpm --package=@croffledev/croffle-cli dlx croffle-plugin create my-extension
```

`build`, `pack`은 [`packages/cli/README.md`](./packages/cli/README.md)를, API 타입(`ExtensionContext`, `CroffleAPI` 등)은 `@croffledev/croffle-types`를 설치한 뒤 [`packages/types/README.md`](./packages/types/README.md)를 참고하세요.

---

## 관리자: CI와 릴리스

### 워크플로 (`.github/workflows`)

| 워크플로                                    | 실행 시점                                                                             | 하는 일                                                                                                                                                                                                                                                                |
| ------------------------------------------- | ------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ci`                                        | `master` push, PR(opened, synchronize, reopened, ready_for_review), 수동              | 바뀐 범위(`desktop`, `types`, `cli`, `server`, `serverWeb`; 공통 설정은 Node 범위 전체)만 골라 oxlint, oxfmt 검사, 타입체크, 테스트, 바뀐 앱·패키지 빌드를 돌립니다. 서버 범위는 pnpm 없이 gofmt, vet, build, test를 돌립니다. **필수 체크는 `CI result` 하나입니다.** |
| `secret-scan`                               | 모든 브랜치 push와 PR                                                                 | 전체 이력에 TruffleHog(verified + unknown)와 Gitleaks                                                                                                                                                                                                                  |
| `labeler`                                   | PR open / reopen / sync                                                               | 경로·브랜치 prefix로 라벨, 없는 라벨 생성                                                                                                                                                                                                                              |
| `publish-packages`                          | `.changeset/**`, `packages/**`, `package.json`, lockfile을 건드린 `master` push, 수동 | "chore: version packages" PR을 열거나, npm에 패키지를 stage합니다(`scripts/stage-packages.mjs`, OIDC trusted publishing)                                                                                                                                               |
| `release` ("Croffle Release")               | 수동                                                                                  | 데스크톱 설치 파일을 draft GitHub Release에                                                                                                                                                                                                                            |
| `server-release` ("Croffle Server Release") | 수동                                                                                  | 서버 이미지를 ghcr에 — GitHub Release는 만들지 않음                                                                                                                                                                                                                    |

### 태그

제품별 접두사를 씁니다: `desktop-vX.Y.Z`, `server-vX.Y.Z`, pre-release는 `…-rc.N`. 데스크톱 `v1.2.6` 이하는 접두사가 없으며 이름을 바꾸지 않습니다. 패키지 태그는 `<name>@<version>`입니다.

### npm 패키지

`@croffledev/croffle-types`, `@croffledev/croffle-cli`는 **Changesets**로 배포합니다. 데스크톱은 npm에 올리지 않습니다.

1. 패키지 변경에는 `pnpm changeset`을 함께 넣습니다.
2. 머지 후 `publish-packages`가 **chore: version packages** PR을 엽니다. 이 PR에는 CI가 돌지 않으므로 머지 전에 로컬에서 확인합니다.
   - 바뀐 파일이 `packages/*/package.json`, `CHANGELOG.md`, 소비된 `.changeset/*.md`뿐인지
   - 버전이 Changeset의 bump와 맞는지
   - `pnpm install --frozen-lockfile`, types·cli 타입체크, cli 빌드, 각 패키지 `pnpm pack`이 통과하는지
3. 머지하면 새 버전이 npm에 **stage**되고(`npm stage publish`), 그 커밋에 git 태그 `<name>@<version>`이 붙습니다. 관리자가 npmjs.com에서 stage된 버전을 승인하기 전에는 설치할 수 없습니다.
4. stage된 버전을 거절했다면 그 태그를 직접 지웁니다. 나중 실행은 예전 커밋에 태그를 붙이지 못합니다.

### 데스크톱

1. Actions → **Croffle Release** → **Run workflow**
2. 입력: `release_type`(`patch` | `minor` | `major` | `rc`; rc에 `patch`를 주면 정식 버전으로 확정), 선택 입력 `version`과 `version_suffix`(예: `rc.1`), `draft`(기본 `true`), `dry_run`(빌드만 — 커밋·태그·릴리스 없음)
3. 워크플로가 `apps/desktop/package.json`의 버전을 올려 커밋하고 `desktop-v<version>` 태그를 붙인 뒤 draft Release 하나를 만듭니다. 이어서 macOS, Linux, Windows 작업이 서명 없는 설치 파일을 붙입니다: Windows `.exe`, macOS arm64 `.dmg`/`.zip`, Linux `.AppImage`/`.deb`/`.rpm`.
4. [`.github/release-notes.md`](./.github/release-notes.md)에 따라 노트 본문을 쓰고 `gh release edit <tag> --title "<title>" --notes-file <file> --draft=false`로 발행합니다(rc는 `--prerelease` 추가). 제목: rc는 `Pre-Release vX.Y.Z-rc.N`, stable은 `Croffle vX.Y.Z — Release Note`.

electron-updater는 draft를 보지 못하므로, draft로 남은 릴리스는 사용자에게 가지 않습니다.

### 서버

1. Actions → **Croffle Server Release** → **Run workflow**
2. 입력: `version`(정확한 버전, `v` 없이), 선택 입력 `prerelease_suffix`(예: `rc.1`), `dry_run`
3. 워크플로가 Go 검사를 돌리고 멀티 아키텍처 이미지를 만들어 `ghcr.io/team-croffle/croffle-server:<version>`(stable이면 `:latest`도)으로 push하고 `server-v<version>` 태그를 붙입니다.
4. 노트는 `docs/server/release/<version>.md`로 커밋합니다([`docs/server/release/README.md`](./docs/server/release/README.md) 참고).

**서버용 GitHub Release는 절대 만들지 않고**, 서버 산출물을 Release에 붙이지도 않습니다. electron-updater가 저장소의 최신 Release를 찾기 때문에, 서버 Release가 생기면 모든 데스크톱 클라이언트의 업데이트가 멈춥니다.

---

## 행동 강령과 라이선스

이슈와 리뷰에서는 서로 존중해 주세요. 프로젝트는 **MIT License**(`LICENSE`)를 따릅니다.
