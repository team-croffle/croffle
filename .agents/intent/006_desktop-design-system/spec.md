---
intent: "006"
status: draft
approvals: []
summary_for_user: |
  1.3.2 를 8개 작업 단위로 나눕니다. 순서는 W-1 토큰(디자인 시스템 값, `--croffle-*` 별칭 유지, 강조 색조는 4개 토큰만, Pretendard 500·600 추가) → W-2 대비·색 규칙 → W-3 테마 좌클릭 저장·부제·다가오는 일정(7일·5개) → W-4 알림 팝오버(울린 알림은 메모리, 곧 울릴 알림 24시간) → W-5 업데이트 확인·버전 타일 → W-6 설정 바로 적용·확장 삭제 확인 창 → W-7 ko 영어 문구·도움말 → W-8 DESIGN.md 입니다.
  검증은 V-1~V-10 입니다. 게이트와 범위 확인, 토큰 대비 자동 테스트(색조 0–359° 전부), 화면 코드 grep, 카탈로그·다가오는 일정·알림·업데이트 단위 테스트, 라이트·다크 화면 확인 8항목, unpack 빌드의 업데이트 확인, DESIGN.md 대조로 이루어집니다.
  명령은 지금 트리에서 사전 실행했습니다(`.loctx/reports/1.3.2/dryrun/`). qa-designer(gemini) 호출은 권한 판정에 막혀 V 는 PM 이 썼습니다.
  확장이 읽는 `--croffle-*` 의 값이 바뀌어 확장 화면 색도 바뀝니다(릴리스 노트에 적음).
---
## 요구 해석
기준: 디자인 시스템 "croffle" 버전 `1795072829-768e`(https://claude.ai/artifact/9RJ3iMqwtduoMasBnMe4ky 의 `project/README.md` · `tokens.json` · `components/*/README.md`)와 디자인 캔버스 Desktop 페이지. 아래에서 "시스템"은 이 버전을 말합니다.

- S-1 (바라는 결과 "모든 화면이 … 토큰(색 · 반경 · 그림자)을 씁니다", 계획 1) — `index.css` 의 `:root`(Croffle Light) · `.dark`(Croffle Dark) 값을 시스템 `tokens.json` 의 값으로 바꿉니다. 시스템에만 있는 `success` · `warning` · `info`(+ `-foreground`)를 더하고 `@theme inline` 에 연결해 `bg-success` · `text-warning` 같은 클래스가 생기게 합니다. 반경은 `--radius: 12px` 에 시스템 단계(sm 8 · md 10 · lg 12 · xl 16 · 2xl 20 · 3xl 24), 그림자는 `shadow-xs` · `shadow-sm` · `shadow-surface` · `shadow-lg` 4종(라이트는 잉크색, 다크는 검정)입니다.
- S-2 (제약 "`--croffle-*` 이름은 지우거나 바꾸지 않습니다", 바라는 결과 "값은 새 토큰을 따릅니다") — 지금 있는 `--croffle-*` 16개는 모두 남기고, 시스템 표대로 의미 토큰의 별칭으로 둡니다(`croffle-bg` → `background`, `croffle-text` → `muted-foreground` …). `--croffle-accent-hue` 도 남깁니다. 시스템에서 새로 생긴 `--croffle-primary-strong` 을 더합니다.
- S-3 (바라는 결과 "강조 색상 설정은 강조색 계열만 바꾸고, 기본 색조는 70°") — 강조색 계열 `primary` · `ring` · `croffle-primary-strong`(= `accent-foreground`) · `croffle-today-bg` 만 `--croffle-accent-hue` 를 따르고, 시스템 기본값과 같은 명도 · 채도를 씁니다(oklch). 그 밖의 토큰(배경 · 사이드바 · 테두리 · 호버)은 색조를 따르지 않습니다(지금은 따름). `DEFAULT_ACCENT_HUE` 는 시스템 `primary`(#98641b)의 oklch 색조를 반올림한 70 이고, 설정 미리보기 색(`hueToPreviewHex`)도 새 `primary` 의 명도 · 채도를 씁니다.
- S-4 (바라는 결과 "본문 글꼴은 Pretendard 그대로") — 글꼴은 Pretendard · Fredoka 그대로입니다. 시스템의 글자 단계가 500 · 600 굵기를 쓰므로(라벨 14/500, 대화상자 제목 18/600) 지금 없는 Pretendard Medium · SemiBold woff2 를 시스템 `project/fonts/` 에서 가져와 같이 넣습니다(PM 해석; 지금은 브라우저가 굵기를 흉내 냅니다).
- S-5 (바라는 결과 "색과 대비 문제가 모두 고쳐집니다", 계획 2) — 다음을 고칩니다.
  - 강조색 위 글자는 `primary-foreground`(흰색 아님). 화면 코드에 강조색 위 `text-white` 가 있으면 바꿉니다.
  - 강조색 글자(달력 제목, 오늘 숫자, 링크)는 `croffle-primary-strong`.
  - 입력칸 · 선택 · 스위치 테두리는 `input`(3:1 이상).
  - 다크 표면은 `background` < `card` = `popover` 순서로 밝아집니다(토큰 값으로 해결).
  - 우선순위 표시는 점 + 글자, 낮음 `success` · 보통 `warning` · 높음 `destructive`.
  - 경고 안내의 `amber-*` 와 그 `dark:` 짝은 `warning` 토큰으로 바꿉니다.
  - 화면 코드에 남은 팔레트 색 · hex 는 토큰으로 바꿉니다(`DESIGN.md` 의 검사 grep 기준). 예외는 제목 표시줄 닫기 버튼의 빨간 호버 하나입니다.
- S-6 (바라는 결과 "오류 글자는 `text-destructive` 이고, `destructive-foreground` 는 빨간 바탕 위 글자에만") — 오류 글자에 쓴 `text-destructive-foreground` 를 모두 `text-destructive` 로 바꾸고, `destructive-foreground` 는 `bg-destructive` 위 글자에만 남깁니다.
- S-7 (사용자 "알림 목록 팝오버 붙이기", 질의서 1 "가") — 왼쪽 사이드바의 종 버튼이 Popover(340px)를 엽니다.
  - 목록 두 묶음: 이번 실행에서 울린 알림(최근 순)과 곧 울릴 알림(시각 순). PM 해석: intent 의 "DB 에서 조회"는 리마인더 스케줄러가 DB 에서 읽어 둔 후보(`reminder-scheduler.ts` `candidates`)를 쓰는 것으로 봅니다. 범위는 앞으로 24시간입니다.
  - 울린 알림은 메인 프로세스 메모리에만 둡니다. 앱을 끄면 사라지고, DB 마이그레이션은 없습니다.
  - 종 아이콘에 읽지 않은 수를 붙이고, 머리글에 "모두 읽음"을 둡니다. 팝오버를 열기만 해서는 읽음이 되지 않습니다(PM 해석: 시스템 Popover 의 "모두 읽음" 동작).
  - 항목을 고르면 그 일정의 보기 창을 엽니다. 비면 "새 알림이 없습니다".
- S-8 (사용자 "기능 구현", 에이전트가 정한 것 "지금부터 7일 안의 일정을 최대 5개") — 오른쪽 사이드바 "다가오는 일정"에 지금 이후 7일 안에 시작하는 일정을 시작 시각 순으로 최대 5개 보여 줍니다.
  - 이미 시작했지만 끝나지 않은 일정은 넣지 않습니다(PM 해석).
  - 반복 일정은 펼친 발생 하나하나를 셉니다. 예외(건너뜀 · 바뀐 발생)도 반영합니다.
  - 항목은 제목, 날짜 · 시각(종일이면 날짜), 일정 색 점이고, 더블클릭하면 보기 창이 열립니다. 일정이 바뀌면 다시 계산합니다.
- S-9 (사용자 "좌클릭도 저장되게 하기") — 제목 표시줄 테마 버튼 좌클릭은 라이트 ↔ 다크를 바꾸고 그 값을 설정 `general.theme` 에 저장합니다. 그래서 설정 화면 · 다음 실행과 같은 값이 됩니다. 우클릭 메뉴(라이트 / 다크 / 시스템)는 그대로입니다.
- S-10 (사용자 "의미 있는 내용으로 변경", 에이전트가 정한 것 "오늘 일정 N개") — 왼쪽 사이드바 항목의 부제는 정보를 더합니다. 달력은 "오늘 일정 N개" / "오늘 일정 없음"입니다. 다른 항목(확장 경로)은 부제를 두지 않거나 확장이 준 설명을 씁니다. 제목을 다른 언어로 되풀이하지 않습니다.
- S-11 (사용자 "버전 타일: 변경", "업데이트 쪽에 수동으로 누를 수 있는 업데이트 확인 버튼/기능 추가") — 오른쪽 사이드바 아래의 버전 글자를 버전 타일로 바꿉니다. 누르면 Popover 가 열리고, 앱 이름 · 버전 · 업데이트 확인 버튼 · 확인 상태를 보여 줍니다.
- S-12 (같은 사용자 말, 계획 4) — 수동 업데이트 확인을 새로 둡니다.
  - 메인 IPC 가 `autoUpdater.checkForUpdates()` 를 부르고, 상태를 `확인 중` → `최신 버전` / `새 버전 X`(기존 업데이트 대화상자를 엶) / `실패` 로 알립니다.
  - 설정 › 일반에 "업데이트" 행("지금 확인" outline 버튼 + 상태)을 두고, 버전 타일과 같은 상태를 씁니다.
  - 개발 빌드에서는 버튼을 끄고 그 이유를 힌트로 보입니다.
  - 시작할 때의 자동 확인은 지금처럼 설정을 따릅니다. 수동 확인의 "최신 버전"은 그 자리에 보이므로 토스트를 띄우지 않습니다.
- S-13 (사용자 "바로 적용으로 통일", 에이전트가 정한 것) — 설정 화면의 저장 · 취소 푸터를 없앱니다.
  - 스위치 · 선택 · 색 선택은 바꾸는 즉시 저장 · 적용합니다.
  - 글자 · 숫자 입력은 칸을 벗어나거나 Enter 를 누를 때 적용합니다. 잘못된 값은 저장하지 않고, 칸 아래에 `text-destructive` 로 이유를 보입니다.
  - 저장이 실패하면 오류 토스트를 띄우고 화면 값을 저장된 값으로 되돌립니다.
  - 동기화 패널은 이미 자기 버튼(연결 · 로그인)으로 동작하므로 바꾸지 않습니다.
- S-14 (사용자 "앱의 확인 모달로 변경") — 확장 삭제는 `confirm()` 대신 앱의 `confirm-modal`(destructive 확인 버튼)로 묻습니다.
- S-15 (사용자 "i18n을 사용하여 번역") — ko 카탈로그의 `settings.errors.install` · `installLocal` 을 한국어로 바꿉니다. 같은 검사로 ko 카탈로그에 남은 다른 영어 문장도 찾아 고칩니다. en 카탈로그에는 같은 키가 그대로 있습니다.
- S-16 (사용자 "실제 동작에 맞춰서 수정") — 도움말의 각 항목 문구를 이 버전의 실제 동작에 맞춥니다(테마 좌클릭 저장, 설정 바로 적용, 알림 팝오버, 다가오는 일정, 업데이트 확인). 문구는 해요체입니다(시스템 "Voice").
- S-17 (계획 5) — `apps/desktop/DESIGN.md` 를 시스템에 맞춰 다시 씁니다.
  - 토큰 표: 상태 토큰, `croffle-primary-strong`.
  - `--croffle-*` 규칙: "이름은 유지, 값은 의미 토큰의 별칭".
  - 오류 글자 규칙(S-6).
  - 앰버 예외를 없애고 `warning` 으로.
  - 반경 표(12px 기준), 그림자 4종, 강조 색상 설정이 바꾸는 범위(S-3).
  - 행동 규칙: 설정 바로 적용, 되돌릴 수 없는 동작은 앱 확인 창, 알림 · 업데이트 · 테마 버튼, 부제.
  - 디자인 시스템 아티팩트 링크.

## 설계
- **토큰 (`index.css`)** — 시스템 hex 를 그대로 옮기되, 강조색 계열 4개만 oklch(L C `var(--croffle-accent-hue)`)로 씁니다. L · C 는 시스템 hex 를 oklch 로 바꾼 값(라이트 · 다크 각각)입니다. `--croffle-today-bg` 는 같은 L C 에 알파(라이트 0.16, 다크 0.18)입니다.
  - 색조를 바꾸면 sRGB 밖으로 나가는 색조가 있을 수 있습니다. 그때도 대비가 유지되는지 V 에서 0–359° 전부를 검사합니다. 깨지면 L · C 를 낮추는 조정을 A-n 으로 적습니다.
  - shadcn 의 `sidebar-*` 는 시스템 값(데스크톱 활성 항목 = `sidebar-primary` = `primary`)입니다.
- **Tailwind 연결** — `@theme inline` 에 `--color-success` 등 상태 6개, `--color-croffle-primary-strong`, `--shadow-xs/sm/surface/lg`, `--radius-2xl/3xl` 을 더합니다. `components/ui/**` 는 고치지 않는 것이 원칙입니다. 토큰만으로 대비가 안 맞는 부분(예: 스위치 트랙 테두리)이 있으면 그 파일만 고치고 A-n 에 적습니다.
- **알림 (S-7)** — 메인: `reminder-scheduler` 가 울린 알림을 메모리 목록(최대 50개; 일정 id, 발생 시각, 제목, 울린 시각, 읽음)에 넣고, 렌더러에 이벤트를 보냅니다. 곧 울릴 알림은 스케줄러가 가진 후보에서 고릅니다.
  - IPC 는 `notifications.handler.ts`(list, markAllRead) → `preload/api/notifications.ts` → `window.croffle` 입니다.
  - 렌더러: `notification-store.ts` + `notification-popover.vue`.
  - 앱 화면 전용이라 `packages/types` 에 넣지 않습니다(PM 해석: 업데이트 이벤트처럼 호스트 UI 용). 확장에 열 필요가 생기면 그때 공개 타입으로 냅니다.
- **다가오는 일정 (S-8)** — 고르는 일(범위 · 정렬 · 5개 자르기 · 진행 중 제외)은 순수 함수 `src/common/upcoming.ts` 로 두고, 단위 테스트를 둡니다. 발생 펼치기는 달력이 쓰는 기존 경로(범위 조회 → rrule 전개)를 그대로 씁니다. 새 DB 조회는 두지 않습니다.
- **업데이트 확인 (S-11 · S-12)** — 메인에 `update.handler.ts`(check)를 두고 `autoUpdater.checkForUpdates()` 결과를 기존 `UPDATE_*` 이벤트로 알립니다. 개발 빌드에서는 `{ status: 'unsupported' }` 를 돌려줍니다. `update-store` 에 `status: idle | checking | latest | available | error | unsupported` 를 두고, 버전 타일 Popover 와 설정 행이 같은 상태를 읽습니다.
- **설정 바로 적용 (S-13)** — `settings-modal.vue` 의 초안(draft) · `handleSave` · `handleCancel` 흐름을 행 단위 저장으로 바꿉니다(`settings.update` 부분 갱신). 테마 초안(`setThemeDraft`)은 즉시 적용으로 합칩니다. 값 검사 · 저장 · 실패 시 되돌리기는 `src/common/settings-apply.ts` 의 순수 함수로 두어 테스트합니다.
- **테스트할 수 있게 두는 모듈** — 알림 목록(`src/main/notification/notification-list.ts`), 업데이트 상태(`src/common/update-status.ts`), 다가오는 일정(`src/common/upcoming.ts`), 설정 적용(`src/common/settings-apply.ts`)은 Electron · DOM 없이 `node --test` 로 불러올 수 있어야 합니다(V-3 ~ V-8).
- **i18n** — 새 문구는 ko · en 두 카탈로그에 같은 키로 넣습니다(`notifications.*`, `rightSidebar.upcoming*`, `sidebar.subtitle.*`, `update.check*`, `settings.general.update*`).
- **브랜치** — `feat/desktop-1.3.2-design-system`. W 마다 커밋하고, 커밋 제목에 W 번호를 씁니다.

## 작업 단위
- [ ] W-1 토큰 — `index.css` 를 시스템 토큰으로(S-1 · S-2 · S-3 · S-4), `@theme` 연결, `appearance.ts` 기본 색조 70 · 미리보기 L/C, Pretendard 500 · 600 글꼴. 토큰 대비 검사 테스트(`test/tokens-contrast.test.ts`: `index.css` 를 읽어 글자 쌍 4.5:1, `input` 3:1, 강조 색조 0–359°)를 이 단위에 넣습니다.
- [ ] W-2 대비 · 색 규칙 — 화면 코드의 오류 글자(S-6), 우선순위 표시, `amber-*` → `warning`, 팔레트 색 · hex · 강조색 위 흰 글자 정리(S-5), 달력 강조 글자 → `croffle-primary-strong`.
- [ ] W-3 셸: 테마 · 부제 · 다가오는 일정 — S-9, S-10, S-8(+ `src/common/upcoming.ts` 와 테스트).
- [ ] W-4 알림 팝오버 — S-7 (메인 메모리 목록 `src/main/notification/notification-list.ts` · IPC · preload · store · popover · 종의 수, `test/notification-list.test.ts`).
- [ ] W-5 업데이트 확인 · 버전 타일 — S-11 · S-12 (IPC · 상태 `src/common/update-status.ts` · store · 버전 타일 Popover · 설정 › 일반 › 업데이트 행, `test/update-status.test.ts`).
- [ ] W-6 설정 바로 적용 · 확장 삭제 확인 — S-13 · S-14 (저장 · 되돌리기 순수 함수 `src/common/settings-apply.ts`, `test/settings-apply.test.ts` 의 `settings save failure restores the stored value`).
- [ ] W-7 문구 — S-15 (ko 영어 문장, `test/i18n-catalogs.test.ts`), S-16 (도움말).
- [ ] W-8 문서 — S-17 (`DESIGN.md`).

W-1 이 먼저이고, W-2 는 W-1 위에 올라갑니다. W-3 ~ W-7 은 서로 독립이라 각각 되돌릴 수 있습니다. W-8 은 마지막입니다.

## 검증 기준
작성: PM. 위임표의 qa-designer(gemini-4-argon-high, `agy`) 호출은 2026-11-19 이 세션의 자동 권한 판정이 막았습니다(외부 모델로 보내기). 사용자가 허용하면 qa-designer 의 검토를 받아 바꾼 점을 A-n 으로 적습니다.

공통 사항:
- 원본은 `.loctx/reports/1.3.2/test/` 에 둡니다. 명령 하나가 파일 하나이고, 첫 줄은 `$ 명령`, 본문은 stdout · stderr 전체, 끝 줄은 `exit=N` 입니다. `00-context.txt` 에 SHA · 시각 · `git status` 를 적습니다.
- 판정은 하위 조건마다 "기대 / 원본 파일:줄 / 일치 · 불일치 · 미확인"으로 적습니다. 하나라도 일치가 아니면 그 V 는 통과가 아닙니다.
- 사전 실행은 spec 요약 전에 했습니다. 결과는 `.loctx/reports/1.3.2/dryrun/` 에 있고, 각 V 끝에 적었습니다.

- **V-1 게이트 (전 W)** — 저장소 루트에서 `pnpm typecheck`, `pnpm lint`, `pnpm format:check`, `pnpm -r test`, `pnpm build` 를 각각 실행합니다.
  - 기대: 다섯 개 모두 `exit=0`.
  - `pnpm -r test` 원본에 `$ node --test --disable-warning=ExperimentalWarning --disable-warning=MODULE_TYPELESS_PACKAGE_JSON test/**/*.test.ts` 줄이 있고, 요약 줄이 `ℹ fail 0` 이며, `ℹ pass` 의 수가 V-3 · V-5 ~ V-8 에서 이름을 든 테스트를 포함할 만큼 늘었습니다(사전 실행 34).
  - 사전 실행: `pnpm -r test` 는 데스크톱만 test 스크립트가 있어 머리 줄이 `Scope: 3 of 4 workspace projects` 다음의 `$ node --test …` 입니다(`dryrun/v1-test.txt:2–3`).
  - 실패: 하나라도 `exit≠0`, 또는 `ℹ fail` 이 0 이 아님.
  - 서버 게이트(`make -C apps/server check`, `server-web`)와 `build-cli` 는 돌리지 않습니다. 이유는 V-2 가 이 범위를 건드리지 않았음을 보이기 때문입니다.
  - 사전 실행: 지금 트리에서 다섯 개를 돌려 출력 형식을 확인했습니다.
- **V-2 범위 (제약 · 범위 밖)** — 루트에서 `git diff --name-only origin/master...HEAD -- apps/server packages apps/desktop/drizzle` 를 실행합니다.
  - 기대: 출력 없음, `exit=0`.
  - 실패: 경로가 하나라도 나옴. 이 경우 V-1 에 해당 게이트를 더하고 사용자에게 알립니다.
  - 사전 실행: 형식 확인.
- **V-3 토큰 대비 (S-1 · S-2 · S-3, W-1)** — `apps/desktop` 에서 `node --test --disable-warning=ExperimentalWarning --disable-warning=MODULE_TYPELESS_PACKAGE_JSON test/tokens-contrast.test.ts` 를 실행합니다. 테스트는 `src/renderer/src/index.css` 를 파일로 읽어 `:root` · `.dark` 의 값을 계산합니다.
  - 기대: `exit=0`. 다음 이름의 테스트가 각각 `✔` 로 나옵니다.
    - `text pairs hold 4.5:1 in both themes` — `foreground` · `muted-foreground` · `accent-foreground` 를 `background` · `card` · `popover` · `sidebar` · `muted` 위에서, 그리고 `primary-foreground`/`primary`, `destructive`(글자)/`background` · `card`, `destructive-foreground`/`destructive`, `success`·`warning`·`info` 의 `-foreground` / 각 바탕을 검사합니다.
    - `input holds 3:1 against background card sidebar in both themes`
    - `accent hue sweep keeps accent pairs for every hue 0..359` — 모든 정수 색조, 두 테마에서 `primary-foreground`/`primary` 4.5:1, `croffle-primary-strong`/`background` · `card` 4.5:1 을 봅니다. sRGB 밖의 색은 잘라 낸 뒤 계산합니다.
    - `non-accent tokens do not use the accent hue` — `--croffle-accent-hue` 를 쓰는 선언이 `primary` · `ring` · `accent-foreground` · `croffle-primary-strong` · `croffle-today-bg` 와 그 별칭뿐입니다.
    - `every croffle variable of 1.3.1 is still defined` — 1.3.1 의 16개 이름 목록(테스트에 고정)과 `--croffle-accent-hue` 가 `:root` 와 `.dark` 에 모두 있습니다.
    - `dark surfaces rise: background < card = popover` — oklch 명도로 비교합니다.
  - 실패: 위 이름 중 `✖` 가 있거나 이름이 출력에 없음.
  - 사전 실행: 아직 파일이 없어 구현 뒤에 의미가 있습니다. `node --test` 의 출력 형식(`✔ 이름 (ms)`, `ℹ fail 0`)은 기존 테스트로 확인했습니다.
- **V-4 화면 코드 규칙 (S-5 · S-6 · S-14, W-2 · W-6)** — `apps/desktop/src/renderer/src` 에서 아래를 실행하고 원본을 따로 남깁니다.
  - a) `grep -rnE '(bg|text|border|ring)-(neutral|gray|zinc|slate|red|amber|yellow|green|blue|white|black)(-[0-9]+)?\b|\[#[0-9A-Fa-f]{3,8}\]' --include='*.vue' --include='*.ts' --exclude-dir=ui .`
    - 기대: 남는 줄은 `app.vue` 의 닫기 버튼 한 줄뿐입니다(`hover:bg-red-600 hover:text-red-100 dark:hover:bg-red-700`).
    - 사전 실행: 지금은 9줄입니다. 닫기 버튼 1줄과 `right-sidebar.vue:75`, `schedule-modal.vue:732 · 733 · 1067`, `sync-panel.vue:143 · 246 · 284 · 306` 입니다(`dryrun/v4a.txt`).
  - b) `grep -rn 'text-destructive-foreground' --include='*.vue' --exclude-dir=ui .`
    - 기대: 남는 줄마다 같은 요소에 `bg-destructive` 가 있습니다. 판정할 때 줄을 하나씩 적습니다.
    - 사전 실행: 지금 2줄(`sync-panel.vue:276 · 281`, 둘 다 바탕 없는 오류 글자, `dryrun/v4b.txt`).
  - c) `grep -rnE '\bconfirm\(' --include='*.vue' --include='*.ts' --exclude-dir=ui .`
    - 기대: 출력 없음(`exit=1`).
    - 사전 실행: 지금 `settings-modal.vue:268` 1줄.
  - 실패: 기대 밖의 줄이 하나라도 있음.
- **V-5 카탈로그 (S-15, W-7)** — `apps/desktop` 에서 `node --test … test/i18n-catalogs.test.ts` 를 실행합니다(플래그는 V-3 과 같음).
  - 기대: `exit=0`. 다음 테스트가 `✔` 입니다.
    - `ko and en have the same keys`
    - `ko has no English sentences` — ko 값에서 `{…}` 자리표시자를 지운 뒤, 한글이 없고 영어 낱말(2글자 이상) 2개 이상이 공백으로 이어지는 값을 영어 문장으로 봅니다. 허용 목록은 테스트에 고정하고 판정에 그대로 옮깁니다(제품 · 고유 이름: `Croffle`, `Croffle Server`, `GitHub`, `Pretendard` 등).
  - 추가로 `grep -n '"install":\|"installLocal":' src/common/i18n/locales/ko.json` 의 두 `settings.errors` 값에 한글이 있어야 합니다.
  - 실패: `✖`, 또는 두 값 중 하나가 영어.
  - 사전 실행: grep 형식을 확인했습니다. 지금 `ko.json:171–172` 가 영어입니다.
- **V-6 다가오는 일정 선택 (S-8, W-3)** — `node --test … test/upcoming.test.ts`(`src/common/upcoming.ts` 대상).
  - 기대: `exit=0`. 다음 이름이 모두 `✔` 입니다.
    - `keeps schedules starting within 7 days` — 6일 23시간 뒤는 들고, 7일 1분 뒤는 빠집니다.
    - `sorts by start and keeps at most 5`
    - `excludes ongoing and past schedules` — 시작했지만 안 끝난 것, 끝난 것.
    - `counts recurring occurrences one by one` — 매일 반복은 5개를 채웁니다.
    - `applies exceptions` — 건너뛴 발생은 빠지고, 시각이 바뀐 발생은 바뀐 시각으로 정렬됩니다.
    - `all-day schedules starting tomorrow are included`
  - 실패: `✖` 또는 이름 없음.
  - 사전 실행: 구현 뒤.
- **V-7 알림 목록 · 업데이트 상태 (S-7 · S-12, W-4 · W-5)** — `node --test … test/notification-list.test.ts test/update-status.test.ts`.
  - 기대: `exit=0`. 다음 이름이 모두 `✔` 입니다.
    - 알림: `fired items are newest first and capped at 50`, `mark all read clears the unread count`, `upcoming keeps candidates within 24 hours sorted by time`.
    - 업데이트: `manual check goes checking then latest`, `available opens the update dialog state`, `error keeps the message`, `dev build is unsupported and never checking`.
  - 실패: `✖` 또는 이름 없음.
  - 사전 실행: 구현 뒤. 설계상 알림 목록과 업데이트 상태는 순수 모듈(`src/main/notification/notification-list.ts`, `src/common/update-status.ts`)로 두어야 이 테스트가 됩니다.
- **V-8 화면 확인 (S-7 ~ S-14, W-3 ~ W-6)** — `pnpm dev` 로 띄운 앱에서 라이트와 다크 각각 확인합니다. PM 이 Playwright `_electron` 으로 돌리거나 사용자가 확인하며, 누가 무엇으로 했는지 적습니다. 스크린샷은 `.loctx/reports/1.3.2/test/screens/<번호>-<light|dark>.png` 입니다. 하위 조건:
  1. 종: 일정에 1분 뒤 알림을 두고 기다리면 OS 알림 뒤 종에 `1` 이 붙습니다. 팝오버를 열면 그 항목이 굵은 글자로 맨 위에 있습니다. "모두 읽음"을 누르면 수가 사라지고, 항목을 누르면 그 일정 보기 창이 열립니다. 알림이 없을 때는 "새 알림이 없습니다"가 보입니다.
  2. 다가오는 일정: 오늘 +1일, +3일, +8일 일정을 만들면 앞의 둘만 보입니다. 하나를 지우면 바로 사라집니다.
  3. 테마: 좌클릭으로 다크로 바꾸고 설정 › 일반을 열면 테마가 "다크"입니다. 앱을 다시 켜도 다크입니다.
  4. 부제: 오늘 일정 2개면 "오늘 일정 2개", 0개면 "오늘 일정 없음"입니다.
  5. 버전 타일: 누르면 버전과 업데이트 확인 버튼이 있는 팝오버가 열립니다. dev 에서는 버튼이 꺼져 있고 힌트가 보입니다.
  6. 설정 바로 적용: 저장 · 취소 버튼이 없습니다.
     - 스위치를 바꾸고 창을 닫았다 열면 바뀐 값입니다.
     - 숫자 칸에 범위 밖 값을 넣고 칸을 벗어나면 칸 아래 빨간 글자가 보이고, 다시 열면 이전 값입니다.
     - 언어를 바꾸면 바로 문구가 바뀝니다.
  7. 확장 삭제: 앱의 확인 창(destructive 버튼)이 뜨고, 취소하면 확장이 남습니다.
  8. 강조 색상: 색조를 200 으로 바꾸면 주 버튼 · 오늘 칸 · 링 · 달력 제목만 바뀌고 사이드바 바탕은 그대로입니다.
  - 실패: 하위 조건 하나라도 다르게 보임, 또는 스크린샷 없음.
  - 사전 실행: 수동이라 하지 않았습니다. 저장 실패 경우(S-13 "저장이 실패하면")는 화면에서 만들 수 없어 W-6 의 단위 테스트 `settings save failure restores the stored value` 로 봅니다. 이 테스트도 V-7 처럼 이름으로 판정합니다.
- **V-9 패키지 빌드의 업데이트 확인 (S-12, W-5)** — `pnpm build:unpack` 결과물을 실행하고 설정 › 일반 › 업데이트의 "지금 확인"을 누릅니다.
  - 기대: 버튼이 켜져 있습니다. "확인 중"이 보인 뒤 30초 안에 "최신 버전" · "새 버전 X" · "실패(이유)" 중 하나가 보입니다. 메인 로그(`~/.config/Croffle/logs/main.log`)에 확인 시작 줄이 있습니다.
  - 실패: 버튼이 꺼져 있음, 상태가 "확인 중"에 멈춤, 또는 앱 오류.
  - unpack 빌드에는 `app-update.yml` 이 없어 "실패"가 나올 수 있습니다. 이 V 는 상태 흐름만 봅니다. 실제 새 버전 감지는 rc 설치본에서 release 단계에 확인합니다.
  - 사전 실행: 하지 않음(구현 뒤).
- **V-10 DESIGN.md (S-17, W-8)** — `apps/desktop` 에서 ``grep -nE '`(success|warning|info)`' DESIGN.md``, `grep -n 'croffle-primary-strong' DESIGN.md`, `grep -n 'amber' DESIGN.md`, `grep -n 'Do not rename them or change their values' DESIGN.md`, `grep -nE '12px|shadow-surface' DESIGN.md`, `grep -n '9RJ3iMqwtduoMasBnMe4ky' DESIGN.md` 를 각각 실행합니다.
  - 기대:
    - 첫째 · 둘째 · 다섯째 · 여섯째는 1줄 이상입니다.
    - `amber` 와 옛 문장은 출력 없음(`exit=1`)입니다.
    - PM 이 S-17 의 항목(토큰 표, 별칭 규칙, 오류 글자, 반경, 그림자 4종, 강조 범위, 행동 규칙, 링크)을 하나씩 DESIGN.md 의 줄 번호와 대조해 판정에 적습니다.
  - 실패: 기대와 다른 grep 결과, 또는 대조에서 빠진 항목.
  - 사전 실행: 지금 `amber` · 옛 문장이 있고, `croffle-primary-strong` · `12px|shadow-surface` · 링크는 없습니다(`dryrun/v10-*.txt`).

## 검토
- 2026-11-19 PM 자체 검토입니다(위임표에 plan-reviewer · qa-plan-reviewer 가 없음). 사용자 원문, intent, 제품 경계와 대조했습니다.
- intent 밖으로 나간 것은 없습니다. PM 해석으로 더한 것은 Pretendard 500 · 600(S-4), 알림 24시간 범위 · 열기만으로는 읽음이 아님(S-7), 진행 중 일정 제외(S-8), 수동 확인 때 토스트 생략(S-12)입니다.

## 조정 기록
—

## 우려
- **확장 화면 색이 바뀝니다.** `--croffle-*` 값이 바뀌므로, 이 변수를 읽는 확장의 화면 색이 1.3.2 에서 달라집니다. 이름은 그대로라 깨지지는 않습니다. 릴리스 노트에 적습니다.
- **색조 · 대비**: 강조 색상의 일부 색조에서 고정 L/C 가 sRGB 밖이면 실제 색이 잘려 대비가 떨어질 수 있습니다. W-1 의 테스트가 잡고, 깨지면 A-n 으로 L · C 를 조정합니다.
- **글꼴 용량**: Pretendard 500 · 600 woff2 가 약 1.5MB 늘어납니다(각 약 780KB). 지금 400 · 700 도 같은 크기입니다.
- **시스템 문서의 남은 문장**: 시스템 `StateBlock` · `Input` README 는 오류 글자를 `destructive-foreground` 로 적고 있습니다. 이는 사용자 결정("오류 글자는 `text-destructive`") · 시스템 README 의 "Colour" 절과 어긋납니다. `CalendarMonth` README 는 토큰에 없는 `shadow-calendar` 를 적습니다(토큰은 `shadow-surface`). 이 spec 은 사용자 결정과 토큰을 따릅니다. 시스템 문서를 고칠지는 사용자에게 묻습니다(intent 제약: 아티팩트는 읽기만).
- **설정 바로 적용의 범위**: 언어를 바꾸면 화면 문구가 곧바로 바뀝니다. 시작 프로그램 등록처럼 OS 에 닿는 설정은 지금 저장 때 하던 적용(`setting-applies.ts`)을 행마다 부릅니다.
- **수동 업데이트 확인은 패키지 빌드에서만** 실제 결과가 나옵니다. 그래서 V 에서 상태 흐름은 단위 수준에서 보고, 실제 확인은 rc 설치본에서 합니다.

## 범위 밖
- 일정 공유(1.3.3), 할 일(1.3.4), 서버 WebUI(서버 0.3.1), pipit-hub.
- 울린 알림의 DB 저장, 알림 설정의 새 항목.
- `components/ui/**` 의 전면 재생성.
- 시스템 · 캔버스 아티팩트 수정.
- 일정 편집 창(`schedule-modal.vue`)의 배치 변경. 이번에는 토큰 · 색 규칙만 적용합니다.
