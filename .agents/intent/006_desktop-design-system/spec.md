---
intent: "006"
status: accepted
approvals:
  - { role: user, by: bluenyang, at: "2026-11-19", words: "1. gemini한데 요청해. 2. 수정했어. 다시 봐줘 3. PM 해석: ㅇㅋ 재확인 후 진행", rev: "6f50d7b", kind: summary, note: "진행 on condition of the recheck; the recheck (qa-designer review, design system 1795075960-81fe) is folded into 4238b83" }
  - { role: user, by: bluenyang, at: "2026-11-20", words: "F-7. 승인", rev: "84e9404", kind: change, note: "S-16 and V-8 9: adding is right-click menu or the add button; double-clicking a date opens that day (review1 F-7, PM wrote the spec wrongly)" }
  - { role: user, by: bluenyang, at: "2026-11-20", words: "정정 3건은 승인", rev: "383c2bf", kind: change, note: "V-4 c regex, V-8 run conditions (isolated dev run), V-9 log location and evidence file names; judged on the same raw output (test/judgement.md)" }
  - { role: user, by: bluenyang, at: "2026-11-20", words: "spec 경로를 정정. 경로만 일치하면 추가 리뷰 없음", rev: "a854b7d", kind: change, note: "V-8 evidence paths to the ones used (first audit A-4); path-only change, no further review by the user's word" }
  - { role: user, by: bluenyang, at: "2026-11-20", words: "응 진행해", rev: "7acf76c", kind: summary, note: "rc.2 changes: S-3 change, S-18, S-19, W-9..W-11, V-3 additions, V-12..V-14 (decisions: 1 \"추천으로\", 2 \"맞아! 되돌려줘.\" + \"가 방식으로 하고 싶어. 색조를 조금 따라가는 느낌.\", 3 \"변경해줘.\")" }
  - { role: user, by: bluenyang, at: "2026-11-20", words: "3건 모두 승인", rev: "95ace28", kind: change, note: "check corrections: V-1 pass count 75 → 74 (PM arithmetic), V-3 9 → 10 tests (rc.2 change), V-8 8 expectation follows the S-3 change (review12 R-3, R-4)" }
summary_for_user: |
  [rc.2 변경, 2026-11-20] rc.1 확인 뒤 작업 단위 3개를 더합니다. W-9 중립색 색조: 바탕 · 사이드바 · 테두리 · 호버 · 글자 등 중립색이 강조 색조를 색조만 따릅니다(L · C 는 시스템 값, 토큰마다 색조 차이 d). 기본 70° 에서 라이트는 지금과 같고, 다크는 같은 방향으로 따르게 해 기본 다크가 슬레이트에서 따뜻한 짙은 회색으로 바뀝니다. W-10 라이트의 입력칸 · 셀렉트 · 여러 줄 입력 · outline 버튼 바탕을 card(#ffffff)로. W-11 이 규칙과 컨트롤 바탕을 디자인 시스템 · 캔버스에 씁니다. primary 는 그대로(brand 토큰은 로드맵).
  검사: V-3 에 "중립색이 색조를 따르고 L · C 는 시스템 값" · "모든 글자 쌍이 0~359° 모든 색조에서 기준 이상"을 더하고, V-12 컨트롤 바탕 grep, V-13 시스템 · 캔버스 파일 대조, V-14 라이트 · 다크 × 70° · 250° 화면과 계산값을 더합니다. 이미 고친 결함 A-21 ~ A-23(버전 타일 · 보기 셀렉트 · 직접 색 칸)도 V-14 에서 봅니다. intent 의 "강조색 계열만" 문장도 함께 고쳤습니다.
  1.3.2 를 8개 작업 단위로 나눕니다: W-1 토큰(시스템 값, `--croffle-*` 별칭 유지, 강조 색조는 4개 토큰만, 기본 70, Pretendard 500·600 선언) → W-2 대비·색 규칙(화면 코드 + `components/ui` 6개 파일) → W-3 테마 좌클릭 저장·부제·다가오는 일정 → W-4 알림 팝오버 → W-5 업데이트 확인(기존 호스트 전용 `window.checkForUpdates` 를 고침, 확인할 수 없는 설치는 바로 "확인할 수 없음")·버전 타일 → W-6 설정 바로 적용·확장 삭제 확인 창 → W-7 ko 영어 값·도움말 4단계 → W-8 DESIGN.md.
  검증은 V-1~V-11 입니다(V-11 릴리스 노트는 release 단계). qa-designer(gemini) 검토로 빠진 검사(S-1~S-4·S-16), grep 정규식, `components/ui` 의 오류 글자·스위치, Linux unpack 의 업데이트 확인 동작을 보강했습니다.
  디자인 시스템은 고치신 버전(`1795075960-81fe`)을 기준으로 합니다. 바뀐 명령은 모두 다시 사전 실행했습니다(`.loctx/reports/1.3.2/dryrun/`).
---
## 요구 해석
기준: 디자인 시스템 "croffle" 버전 `1795075960-81fe`(https://claude.ai/artifact/9RJ3iMqwtduoMasBnMe4ky 의 `project/README.md` · `tokens.json` · `components/*/README.md`)와 디자인 캔버스 Desktop 페이지. 아래에서 "시스템"은 이 버전을 말합니다.

- S-1 (바라는 결과 "모든 화면이 … 토큰(색 · 반경 · 그림자)을 씁니다", 계획 1) — `index.css` 의 `:root`(Croffle Light) · `.dark`(Croffle Dark) 값을 시스템 `tokens.json` 의 값으로 바꿉니다. 시스템에만 있는 `success` · `warning` · `info`(+ `-foreground`)를 더하고 `@theme inline` 에 연결해 `bg-success` · `text-warning` 같은 클래스가 생기게 합니다. 반경은 `--radius: 12px` 에 시스템 단계(sm 8 · md 10 · lg 12 · xl 16 · 2xl 20 · 3xl 24), 그림자는 `shadow-xs` · `shadow-sm` · `shadow-surface` · `shadow-lg` 4종(라이트는 잉크색, 다크는 검정)입니다.
- S-2 (제약 "`--croffle-*` 이름은 지우거나 바꾸지 않습니다", 바라는 결과 "값은 새 토큰을 따릅니다") — 지금 있는 `--croffle-*` 16개는 모두 남기고, 시스템 표대로 의미 토큰의 별칭으로 둡니다(`croffle-bg` → `background`, `croffle-text` → `muted-foreground` …). `--croffle-accent-hue` 도 남깁니다. 시스템에서 새로 생긴 `--croffle-primary-strong` 을 더합니다.
- S-3 (바라는 결과 "강조 색상 설정은 강조색 계열만 바꾸고, 기본 색조는 70°") — 강조색 계열 `primary` · `ring` · `croffle-primary-strong`(= `accent-foreground`) · `croffle-today-bg` 만 `--croffle-accent-hue` 를 따르고, 시스템 기본값과 같은 명도 · 채도를 씁니다(oklch). 그 밖의 토큰(배경 · 사이드바 · 테두리 · 호버)은 색조를 따르지 않습니다(지금은 따름). `DEFAULT_ACCENT_HUE`(`appearance.ts`)와 메인 기본 설정 `DEFAULT_SETTINGS.appearance.accentHue`(`setting-service.ts:29`)는 시스템 `primary`(#98641b)의 oklch 색조를 반올림한 70 입니다. 이미 저장된 69.8 은 그대로 두며, `huesMatch` 가 기본값으로 봅니다. 설정 미리보기 색(`hueToPreviewHex`)도 새 `primary` 의 명도 · 채도를 씁니다.
  - **S-3 변경 (2026-11-20, rc.1 확인 뒤 사용자 결정):** 중립색도 강조 색조를 **색조만** 따릅니다. 따르는 토큰은 라이트 `background` · `foreground` · `secondary` · `secondary-foreground` · `muted` · `muted-foreground` · `accent` · `border` · `input` · `sidebar`, 다크는 여기에 `card` · `popover` · `sidebar-accent` · `sidebar-border` 를 더한 것입니다. 각 토큰은 `oklch(L C calc(var(--croffle-accent-hue) + d))` 이고 L · C 는 시스템 값, `d` 는 라이트에서 그 토큰의 시스템 색조 − 70 입니다. 다크는 같은 방향으로 따르도록 라이트 짝의 `d` 를 씁니다(`card` · `popover` → `background`, `sidebar-accent` · `sidebar-border` → `muted`). 그래서 기본 70° 에서 라이트는 시스템 hex 와 같고, 다크 중립색은 슬레이트(색조 264°)에서 따뜻한 짙은 회색(예: `background` #15171b → #191713)으로 바뀌며 시스템 다크 값도 그렇게 고칩니다. 고정: `destructive` · `success` · `warning` · `info` 와 각 `-foreground`, `primary-foreground`, 그림자, `croffle-day-select-bg`, 채도 0 인 색(라이트 `card` · `popover`). 위 "그 밖의 토큰은 색조를 따르지 않습니다"는 이것으로 바뀝니다.
- S-4 (바라는 결과 "본문 글꼴은 Pretendard 그대로") — 글꼴은 Pretendard · Fredoka 그대로입니다. 시스템의 글자 단계가 500 · 600 굵기를 쓰므로(라벨 14/500, 대화상자 제목 18/600) 이미 저장소에 있는 `assets/fonts/woff2/Pretendard-Medium.woff2` · `Pretendard-SemiBold.woff2` 의 `@font-face`(500 · 600)를 선언합니다(PM 해석; 지금은 선언이 없어 브라우저가 굵기를 흉내 냅니다).
- S-5 (바라는 결과 "색과 대비 문제가 모두 고쳐집니다", 계획 2) — 다음을 고칩니다.
  - 강조색 위 글자는 `primary-foreground`(흰색 아님). 화면 코드에 강조색 위 `text-white` 가 있으면 바꿉니다.
  - 강조색 글자(달력 제목, 오늘 숫자, 링크)는 `croffle-primary-strong`.
  - 입력칸 · 선택 · 스위치 테두리(스위치는 꺼진 트랙)는 `input`(3:1 이상)이고, 투명도로 묽히지 않습니다. 켜진 스위치의 손잡이는 `primary-foreground` 입니다.
  - 다크 표면은 `background` < `card` = `popover` 순서로 밝아집니다(토큰 값으로 해결).
  - 우선순위 표시는 점 + 글자, 낮음 `success` · 보통 `warning` · 높음 `destructive`.
  - 경고 안내의 `amber-*` 와 그 `dark:` 짝은 `warning` 토큰으로 바꿉니다.
  - 화면 코드에 남은 팔레트 색(`emerald` · `rose` 포함 Tailwind 팔레트 전부) · hex · `<style>` 안의 `rgba(` 는 토큰으로 바꿉니다. 예외는 제목 표시줄 닫기 버튼의 빨간 호버 하나입니다.
- S-6 (바라는 결과 "오류 글자는 `text-destructive` 이고, `destructive-foreground` 는 빨간 바탕 위 글자에만") — 오류 글자에 쓴 `text-destructive-foreground` 를 모두 `text-destructive` 로 바꾸고, `destructive-foreground` 는 `bg-destructive` 위 글자에만 남깁니다. `components/ui` 안도 같습니다. 바탕 없이 `text-destructive-foreground` 를 쓰는 `ContextMenuItem` · `CalendarCellTrigger` · `RangeCalendarCellTrigger` 는 `text-destructive` 로, `bg-destructive` 위에 `text-white` 를 쓰는 `button` · `badge` 의 `destructive` 변형은 `text-destructive-foreground` 로 바꿉니다(`dark:bg-destructive/60` 없앰).
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
- S-10 (사용자 "의미 있는 내용으로 변경", 에이전트가 정한 것 "오늘 일정 N개") — 왼쪽 사이드바 항목의 부제는 정보를 더합니다. 달력은 "오늘 일정 N개" / "오늘 일정 없음"입니다. 다른 항목(확장 경로)은 부제를 두지 않거나 확장이 준 설명을 씁니다. 제목을 다른 언어로 되풀이하지 않습니다(지금 ko `sidebar.calendar.subtitle` 은 "Calendar").
- S-11 (사용자 "버전 타일: 변경", "업데이트 쪽에 수동으로 누를 수 있는 업데이트 확인 버튼/기능 추가") — 오른쪽 사이드바 아래의 버전 글자를 버전 타일로 바꿉니다. 누르면 Popover 가 열리고, 앱 이름 · 버전 · 업데이트 확인 버튼 · 확인 상태를 보여 줍니다.
- S-12 (같은 사용자 말, 계획 4) — 수동 업데이트 확인을 둡니다.
  - 이미 있는 호스트 전용 `croffle.window.checkForUpdates()`(`window:checkForUpdates`, `packages/types` 에서는 host-only 로 빠져 있음)를 고쳐 씁니다.
  - 수동 확인은 자동 업데이트 설정이 꺼져 있어도 동작합니다(지금은 `window-service.ts:254` 에서 일찍 돌아감).
  - 상태는 `확인 중` → `최신 버전` / `새 버전 X`(기존 업데이트 대화상자를 엶) / `실패(이유)` 입니다.
  - 이 설치 형식에서 확인할 수 없으면(`app.isPackaged` 가 아님, 또는 `autoUpdater.isUpdaterActive()` 가 거짓 — 예: Linux 에서 AppImage 가 아닌 실행) 곧바로 `확인할 수 없음`(이유 힌트)이 됩니다. `확인 중`에 멈추지 않습니다.
  - 설정 › 일반에 "업데이트" 행("지금 확인" outline 버튼 + 상태)을 두고, 버전 타일과 같은 상태를 씁니다.
  - 개발 빌드에서는 두 곳 모두 버튼을 끄고 그 이유를 힌트로 보입니다.
  - 시작할 때의 자동 확인은 지금처럼 설정을 따릅니다. 수동 확인의 "최신 버전"은 그 자리에 보이므로 토스트를 띄우지 않습니다.
- S-13 (사용자 "바로 적용으로 통일", 에이전트가 정한 것) — 설정 화면의 저장 · 취소 푸터를 없앱니다.
  - 스위치 · 선택 · 색 선택은 바꾸는 즉시 저장 · 적용합니다.
  - 글자 · 숫자 입력은 칸을 벗어나거나 Enter 를 누를 때 적용합니다. 잘못된 값은 저장하지 않고, 칸 아래에 `text-destructive` 로 이유를 보입니다. 지금 이런 칸은 확장 구성 폼(`config-schema-form.vue`)에만 있습니다. 숫자는 `Number.isFinite` 가 아니면 잘못된 값입니다(스키마에 범위 필드가 없음). 확장 관리 탭의 GitHub 주소 칸은 "설치" 버튼으로 동작하는 작업 칸이라 그대로입니다.
  - 저장 · 취소 버튼을 전제로 한 문구(`settings.calendar.dataHint` "저장/취소 버튼과 무관합니다")는 없앱니다.
  - 저장이 실패하면 오류 토스트를 띄우고 화면 값을 저장된 값으로 되돌립니다.
  - 동기화 패널은 이미 자기 버튼(연결 · 로그인)으로 동작하므로 바꾸지 않습니다.
- S-14 (사용자 "앱의 확인 모달로 변경") — 확장 삭제는 `confirm()` 대신 앱의 `confirm-modal`(destructive 확인 버튼)로 묻습니다.
- S-15 (사용자 "i18n을 사용하여 번역") — ko 카탈로그의 `settings.errors.install` · `installLocal` 을 한국어로 바꿉니다. 같은 검사로 ko 카탈로그에 남은 다른 영어 문장도 찾아 고칩니다. en 카탈로그에는 같은 키가 그대로 있습니다.
- S-16 (사용자 "실제 동작에 맞춰서 수정") — 도움말의 각 항목 문구를 이 버전의 실제 동작에 맞춥니다. 단계는 지금처럼 4개(`welcome` · `addTodo` · `theme` · `ready`)이고 키도 유지합니다. `addTodo` 는 일정 추가(우클릭 메뉴 · 추가 버튼; 날짜 더블클릭은 그날 보기; 지금 문구의 "클릭"은 틀림)와 다가오는 일정, `theme` 은 테마 좌클릭 저장 · 우클릭 메뉴와 설정 바로 적용, `ready` 는 알림 종과 업데이트 확인을 담습니다. 문구는 해요체입니다(시스템 "Voice").
- S-17 (계획 5) — `apps/desktop/DESIGN.md` 를 시스템에 맞춰 다시 씁니다.
- S-18 (사용자 결정 3 "변경해줘.", 2026-11-20) — 라이트에서 `Input` · `SelectTrigger` · `Textarea` 와 `Button variant="outline"` 의 바탕을 `card`(#ffffff)로 합니다. 다크는 지금 그대로(`dark:bg-input/30`)입니다. 바탕과 컨트롤이 구분되게 하는 것이 목적입니다.
- S-19 (사용자 "이거 디자인 시스템을 통해서 추가로 작성해줄래?", 2026-11-20) — S-3 변경의 규칙(따르는 토큰 · 고정 토큰 · L · C · `d` · 기준 색조)과 S-18 을 디자인 시스템(`tokens.json` 의 규칙 항목과 값, README 의 "강조 색조" 절, Input · Button 문서, `components/bundle.css`)과 캔버스(`croffle-theme.css`, `ds/croffle/tokens.json`)에 씁니다. 다른 디자인 시스템이 기준 색조 · 토큰별 L · C · `d` · 고정 목록만 바꿔 쓸 수 있게 적습니다.
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
- **Tailwind 연결** — `@theme inline` 에 `--color-success` 등 상태 6개, `--color-croffle-primary-strong`, `--shadow-xs/sm/surface/lg`, `--radius-2xl/3xl` 을 더합니다. `components/ui/**` 는 토큰으로 해결하는 것이 원칙이지만, 토큰만으로는 S-5 · S-6 이 성립하지 않는 다음 파일은 고칩니다: `button/index.ts` · `badge/index.ts`(destructive 변형), `context-menu/ContextMenuItem.vue`, `calendar/CalendarCellTrigger.vue`, `range-calendar/RangeCalendarCellTrigger.vue`, `switch/Switch.vue`. 이 밖에 더 고쳐야 하는 파일이 나오면 A-n 에 적습니다.
- **알림 (S-7)** — 메인: `reminder-scheduler` 가 울린 알림을 메모리 목록(최대 50개; 일정 id, 발생 시각, 제목, 울린 시각, 읽음)에 넣고, 렌더러에 이벤트를 보냅니다. 곧 울릴 알림은 스케줄러가 가진 후보에서 고릅니다.
  - IPC 는 `notifications.handler.ts`(list, markAllRead) → `preload/api/notifications.ts` → `window.croffle` 입니다.
  - 렌더러: `notification-store.ts` + `notification-popover.vue`.
  - 앱 화면 전용이라 `packages/types` 에 넣지 않습니다(PM 해석: 업데이트 이벤트처럼 호스트 UI 용). 확장에 열 필요가 생기면 그때 공개 타입으로 냅니다.
- **다가오는 일정 (S-8)** — 고르는 일(범위 · 정렬 · 5개 자르기 · 진행 중 제외)은 순수 함수 `src/common/upcoming.ts` 로 두고, 단위 테스트를 둡니다. 발생 펼치기는 달력이 쓰는 기존 경로(범위 조회 → rrule 전개)를 그대로 씁니다. 새 DB 조회는 두지 않습니다.
- **업데이트 확인 (S-11 · S-12)** — 새 IPC 를 두지 않고 `window:checkForUpdates` → `windowService.checkForUpdates({ manual: true })` 를 고칩니다. 수동이면 `autoUpdate` 설정을 보지 않습니다. `!app.isPackaged` 또는 `!autoUpdater.isUpdaterActive()` 이면 `{ status: 'unsupported', reason }` 을 돌려주고, 아니면 `checkForUpdates()` 를 부른 뒤 `{ status: 'started' }` 를 돌려줍니다(결과는 기존 `UPDATE_*` 이벤트; 반환이 `null` 이면 `unsupported`; 예외는 `error`). 렌더러 `update-store` 는 순수 상태 함수(`src/common/update-status.ts`, `idle | checking | latest | available | error | unsupported`)를 쓰고, 버전 타일 Popover 와 설정 행이 같은 store 를 읽습니다. 수동 확인 중에 온 `UPDATE_NOT_AVAILABLE` 은 토스트를 띄우지 않습니다(시작 때 자동 확인은 지금처럼 토스트). 반환 타입이 바뀌는 `window.checkForUpdates` 는 호스트 전용이라 `packages/types` 는 바뀌지 않습니다.
- **설정 바로 적용 (S-13)** — `settings-modal.vue` 의 초안(draft) · `handleSave` · `handleCancel` 흐름을 행 단위 저장으로 바꿉니다(`settings.update` 부분 갱신). 테마 초안(`setThemeDraft`)은 즉시 적용으로 합칩니다. 값 검사 · 저장 · 실패 시 되돌리기는 `src/common/settings-apply.ts` 의 순수 함수로 두어 테스트합니다.
- **테스트할 수 있게 두는 모듈** — 알림 목록(`src/main/notification/notification-list.ts`), 업데이트 상태(`src/common/update-status.ts`), 다가오는 일정(`src/common/upcoming.ts`), 설정 적용(`src/common/settings-apply.ts`)은 Electron · DOM 없이 `node --test` 로 불러올 수 있어야 합니다(V-3 ~ V-8).
- **i18n** — 새 문구는 ko · en 두 카탈로그에 같은 키로 넣습니다(`notifications.*`, `rightSidebar.upcoming*`, `sidebar.subtitle.*`, `update.check*`, `settings.general.update*`).
- **브랜치** — `feat/desktop-1.3.2-design-system`. W 마다 커밋하고, 커밋 제목에 W 번호를 씁니다.

## 작업 단위
- [x] W-1 토큰 — `index.css` 를 시스템 토큰으로(S-1 · S-2 · S-3 · S-4), `@theme` 연결, `appearance.ts` · `setting-service.ts` 기본 색조 70 · 미리보기 L/C, Pretendard 500 · 600 `@font-face`. 토큰 대비 검사 테스트(`test/tokens-contrast.test.ts`: `index.css` 를 읽어 글자 쌍 4.5:1, `input` 3:1, 강조 색조 0–359°)를 이 단위에 넣습니다. — a91eafd
- [x] W-2 대비 · 색 규칙 — 화면 코드와 위 `components/ui` 6개 파일의 오류 글자(S-6), 스위치(S-5), 우선순위 점 + 글자, `amber-*` → `warning`, 팔레트 색 · hex · `rgba(` · 강조색 위 흰 글자 정리(S-5), 달력 강조 글자(`calendar.vue` 의 `var(--croffle-primary)` 글자색) · 설정 · 도움말의 강조 글자 → `croffle-primary-strong`. — 4458a7e
- [x] W-3 셸: 테마 · 부제 · 다가오는 일정 — S-9, S-10, S-8(+ `src/common/upcoming.ts` 와 테스트). — 0f1ce63
- [x] W-4 알림 팝오버 — S-7 (메인 메모리 목록 `src/main/notification/notification-list.ts` · IPC · preload · store · popover · 종의 수, `test/notification-list.test.ts`). — cd17d8a
- [x] W-5 업데이트 확인 · 버전 타일 — S-11 · S-12 (IPC · 상태 `src/common/update-status.ts` · store · 버전 타일 Popover · 설정 › 일반 › 업데이트 행, `test/update-status.test.ts`). — 0fe0860
- [x] W-6 설정 바로 적용 · 확장 삭제 확인 — S-13 · S-14 (저장 · 되돌리기 순수 함수 `src/common/settings-apply.ts`, `test/settings-apply.test.ts`, 확장 구성 폼의 blur · Enter 적용, `dataHint` 정리). — c80a644
- [x] W-7 문구 — S-15 (ko 영어 값, `test/i18n-catalogs.test.ts`), S-16 (도움말 4단계). — 1c1ba1b
- [x] W-8 문서 — S-17 (`DESIGN.md`). — 5eb9a5d
- [x] W-9 중립색 색조 — S-3 변경 (`index.css` `:root` · `.dark`, `test/tokens-contrast.test.ts`, DESIGN.md 의 강조 색조 절). — 845d998
- [x] W-10 라이트 컨트롤 바탕 — S-18 (`components/ui` 의 `input/Input.vue` · `select/SelectTrigger.vue` · `textarea/Textarea.vue` · `button/index.ts`, DESIGN.md). — 94305e6
- [x] W-11 디자인 시스템 · 캔버스 — S-19 (아티팩트 두 개; 저장소 파일 없음). — 시스템 버전 `1795164178-9247`(tokens.json · README · Input/Select/Button README · bundle.css · design-system.json), 캔버스 버전 `1795164205-d996`(croffle-theme.css · ds/croffle/tokens.json · ds/croffle/components/bundle.css). A-24 · A-25 의 값도 시스템에 넣었습니다.

W-1 이 먼저이고, W-2 는 W-1 위에 올라갑니다. W-3 ~ W-7 은 서로 독립이라 각각 되돌릴 수 있습니다(S-10 의 "Calendar" 부제는 W-3, 그 밖의 ko 영어 값은 W-7). W-8 은 마지막입니다. 새 테스트 파일은 모두 `apps/desktop/test/` 바로 아래에 둡니다(`test/**/*.test.ts` 는 셸 glob 이라 하위 폴더를 찾지 않음).

## 검증 기준
작성: PM 초안 → qa-designer(gemini-4-argon-high) 검토(`.loctx/reports/1.3.2/qa-design-answer.md`) → PM 이 코드와 사전 실행으로 확인해 반영. 처음 위임은 2026-11-19 자동 권한 판정에 막혔고, 사용자 "gemini한데 요청해." 뒤 다시 불렀습니다.

공통 사항:
- 원본은 `.loctx/reports/1.3.2/test/` 에 둡니다. 명령 하나가 파일 하나이고, 첫 줄은 `$ 명령`(정규식의 `\b` 는 글자 그대로), 본문은 stdout · stderr 전체, 끝 줄은 `exit=N` 입니다. `00-context.txt` 에 SHA · 시각 · `git status` 를 적습니다.
- 판정은 하위 조건마다 "기대 / 원본 파일:줄 / 일치 · 불일치 · 미확인"으로 적습니다. 하나라도 일치가 아니면 그 V 는 통과가 아닙니다.
- `node --test` 명령의 플래그는 모두 `--disable-warning=ExperimentalWarning --disable-warning=MODULE_TYPELESS_PACKAGE_JSON` 이고, 작업 디렉터리는 `apps/desktop` 입니다. 테스트 이름은 출력의 `✔ <이름> (<ms>ms)` 줄에서 `grep -F "✔ <이름>"` 으로 찾습니다.
- 사전 실행은 spec 요약 전에 했습니다. 결과는 `.loctx/reports/1.3.2/dryrun/` 에 있고, 각 V 끝에 적었습니다.

- **V-1 게이트 (전 W · S-4)** — 저장소 루트에서 `pnpm typecheck`, `pnpm lint`, `pnpm format:check`, `pnpm -r test`, `pnpm build` 를 각각 실행해 `v1-{typecheck,lint,format,test,build}.txt` 에 남깁니다.
  - 기대:
    1. 다섯 개 모두 `exit=0`.
    2. `v1-test.txt` 에 `$ node --test --disable-warning=ExperimentalWarning --disable-warning=MODULE_TYPELESS_PACKAGE_JSON test/**/*.test.ts` 줄과 `ℹ fail 0` 줄이 있고, `ℹ pass` 가 61 이상입니다(사전 실행 34 + V-3 9 + V-5 3 + V-6 6 + V-7 9).
    3. `v1-build.txt` 에 `Pretendard-Medium-` 와 `Pretendard-SemiBold-` 로 시작하는 `.woff2` 산출물 줄이 있습니다(S-4).
  - 실패: `exit≠0`, `ℹ fail` 이 0 이 아님, `ℹ pass` 가 61 미만(rc.2 부터는 74 미만: 73 − 옛 V-3 테스트 1 + 새 2; 2026-11-20 정정, 처음 문구 "75" 는 PM 산수 오류), 또는 두 글꼴 줄이 없음.
  - 서버 게이트(`make -C apps/server check`, `server-web`)와 `build-cli` 는 돌리지 않습니다. V-2 가 그 범위를 건드리지 않았음을 보입니다.
  - 사전 실행: 다섯 개 모두 `exit=0`, `ℹ pass 34` · `ℹ fail 0`(`dryrun/v1-test.txt:50–51`). 빌드 원본의 글꼴 줄은 `out/renderer/assets/Pretendard-Regular-<hash>.woff2` 형식이고, 지금은 Regular · Bold 만 있습니다(`dryrun/v1-build.txt:23–29`).
- **V-2 범위 (제약 · 범위 밖)** — 루트에서 `git diff --name-only origin/master...HEAD -- apps/server packages apps/desktop/drizzle .agents/harness .claude` 를 실행해 `v2-scope.txt` 에 남깁니다.
  - 기대: 출력 0줄, `exit=0`.
  - 실패: 경로가 한 줄이라도 나옴. 이 경우 V-1 에 해당 게이트를 더하고 사용자에게 알립니다.
  - 사전 실행: 출력 0줄(`dryrun/v2-scope-2.txt`).
- **V-3 토큰 · 대비 · 글꼴 선언 (S-1 · S-2 · S-3 · S-4, W-1)** — `node --test … test/tokens-contrast.test.ts` → `v3-tokens.txt`. 테스트는 `src/renderer/src/index.css` 와 관련 소스를 파일로 읽어 계산합니다.
  - 기대: `exit=0`, `ℹ fail 0`, 다음 9개가 모두 `✔` 입니다. (rc.2 부터 10개: 아래 "rc.2 변경" 대로 `non-accent tokens do not use the accent hue` 대신 `neutral tokens follow the accent hue with system L and C` 와 `every text pair holds for every hue 0..359`; 2026-11-20 정정)
    - `text pairs hold 4.5:1 in both themes` — `foreground` · `muted-foreground` · `accent-foreground` 를 `background` · `card` · `popover` · `sidebar` · `muted` 위에서, 그리고 `primary-foreground`/`primary`, `destructive`(글자)/`background` · `card`, `destructive-foreground`/`destructive`, `success` · `warning` · `info` 의 `-foreground`/각 바탕.
    - `input holds 3:1 against background card sidebar in both themes`
    - `accent hue sweep keeps accent pairs for every hue 0..359` — 정수 색조 전부, 두 테마에서 `primary-foreground`/`primary` ≥ 4.5:1, `croffle-primary-strong`/`background` · `card` ≥ 4.5:1(sRGB 밖은 잘라 낸 뒤 계산).
    - `non-accent tokens do not use the accent hue` — `--croffle-accent-hue` 를 직접 또는 별칭으로 쓰는 토큰이 `primary`(`croffle-primary` · `sidebar-primary`), `ring`(`sidebar-ring`), `accent-foreground`(`croffle-primary-strong` · `sidebar-accent-foreground`), `croffle-today-bg` 뿐입니다.
    - **rc.2 변경 (S-3 변경):** 위 항목 대신 `neutral tokens follow the accent hue with system L and C` — 따르는 토큰 목록이 S-3 변경과 같고, 각 토큰이 기본 70° 에서 라이트는 시스템 hex 와 sRGB 8비트로 같으며(다크는 S-3 변경의 L · C · `d` 그대로), 고정 목록은 `--croffle-accent-hue` 를 쓰지 않습니다. 그리고 `every text pair holds for every hue 0..359` — 위 `text pairs hold 4.5:1 in both themes` 의 쌍과 `input` 3:1 쌍을 정수 색조 전부에서 계산해 모두 기준 이상입니다.
    - `every croffle variable of 1.3.1 is still defined` — 1.3.1 의 16개 이름(테스트에 고정), `--croffle-accent-hue`, 새 `--croffle-primary-strong` 이 `:root` 와 `.dark` 에 모두 있습니다.
    - `croffle variables alias semantic tokens and default accent hue is 70` — 강조 계열이 아닌 `--croffle-*` 가 시스템 표대로 `var(--의미 토큰)` 이고, `index.css` 의 `--croffle-accent-hue`, `appearance.ts` 의 `DEFAULT_ACCENT_HUE`, `setting-service.ts` 의 기본 `accentHue` 가 모두 70 이며, `hueToPreviewHex` 가 라이트 `primary` 의 L · C 를 씁니다.
    - `dark surfaces rise: background < card = popover` — `.dark` 의 oklch 명도.
    - `theme inline exposes status colors radius steps and 4 shadows` — `--radius` 12px(0.75rem), `--radius-sm/md/lg/xl/2xl/3xl` = 8/10/12/16/20/24px, `--color-success` · `--color-warning` · `--color-info` 와 각 `-foreground`, `--color-croffle-primary-strong`, `--shadow-xs/sm/surface/lg`(`:root` 와 `.dark` 의 값 포함).
    - `pretendard 400 500 600 700 and fredoka 600 700 font faces are declared` — 각 `@font-face` 와 가리키는 `.woff2` 파일이 있습니다.
  - 실패: `exit≠0`, `ℹ fail` 이 0 이 아님, 또는 9개(rc.2 부터 10개) 중 하나라도 `✔` 줄이 없음.
  - 사전 실행: 파일이 아직 없어 구현 뒤에 의미가 있습니다. `✔ 이름 (ms)` 형식은 기존 테스트로 확인했습니다(`dryrun/v3-format.txt`).
- **V-4 화면 코드 규칙 (S-5 · S-6 · S-14, W-2 · W-6)** — `apps/desktop/src/renderer/src` 에서 a) ~ f) 를 각각 실행해 `v4{a..f}.txt` 에 남깁니다.
  - a) `grep -rnE '(bg|text|border|ring)-(slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose|white|black)(-[0-9]+)?(/[0-9]+)?\b|\[#[0-9A-Fa-f]{3,8}\]|rgba\(' --include='*.vue' --include='*.ts' --exclude-dir=ui .`
    - 기대: `./app.vue` 의 닫기 버튼 1줄(`hover:bg-red-600 hover:text-red-100 dark:hover:bg-red-700`)만 남고 `exit=0`.
  - b) `grep -rn 'text-destructive-foreground' --include='*.vue' --include='*.ts' .` (`ui/` 포함)
    - 기대: 나오는 줄마다 같은 요소 또는 같은 variant 문자열에 `bg-destructive` 가 있습니다(`components/ui/button/index.ts` · `badge/index.ts` 의 destructive 변형). 바탕 없이 쓰는 줄(`sync-panel.vue`, `ContextMenuItem.vue`, `CalendarCellTrigger.vue`, `RangeCalendarCellTrigger.vue`)은 0줄입니다.
  - c) `grep -rnE '(^|[^.[:alnum:]_])confirm\(' --include='*.vue' --include='*.ts' --exclude-dir=ui .` (점이나 식별자 뒤가 아닌 `confirm(` 만; 2026-11-20 정정, 처음 문구 `\bconfirm\(` 는 메서드 `patches.confirm(` 도 잡음)
    - 기대: 출력 없음, `exit=1`.
  - d) `grep -nE 'var\(--croffle-primary(-strong)?\)' components/calendar.vue`
    - 기대: 달력 제목, 오늘 숫자, 더보기 호버의 글자색 줄이 `var(--croffle-primary-strong)` 이고, 글자색(`color:`)에 `var(--croffle-primary)` 를 쓰는 줄이 0줄입니다. 바탕(`background`)에 쓰는 `var(--croffle-primary)` 는 허용하며 판정에 줄마다 적습니다.
  - e) `grep -nE 'success|warning|destructive' components/right-sidebar.vue components/schedule-modal.vue`
    - 기대: 두 파일 모두 우선순위 낮음 · 보통 · 높음에 `success` · `warning` · `destructive` 를 쓰는 줄이 있고 `exit=0`.
  - f) `grep -nE 'border-input|bg-input|bg-primary|bg-background|primary-foreground|text-white|bg-destructive' components/ui/input/Input.vue components/ui/select/SelectTrigger.vue components/ui/switch/Switch.vue components/ui/button/index.ts components/ui/badge/index.ts`
    - 기대: `Input.vue` · `SelectTrigger.vue` 에 `border-input`. `Switch.vue` 에 `bg-input/80` 이 없고, 켜진 손잡이가 `primary-foreground` 입니다(`data-[state=checked]` 의 `bg-background` 없음). `button` · `badge` 의 destructive 변형에 `text-white` 가 없고 `text-destructive-foreground` 가 있습니다.
  - 실패: a) 에 닫기 버튼 밖의 줄, b) 에 바탕 없는 줄, c) 에 출력, d) 에 글자색 `var(--croffle-primary)`, e) 에 상태 토큰 하나라도 빠짐, f) 의 기대와 다름.
  - 사전 실행(이 명령 그대로, `dryrun/v4*-2.txt`): a) 16줄, b) 5줄(`sync-panel.vue` 2 · `ui/` 3), c) `settings-modal.vue:268` 1줄, d) 3줄, e) 2줄, f) 9줄. 첫 줄의 `\b` 가 글자 그대로 남는 것도 확인했습니다.
- **V-5 카탈로그 · 도움말 (S-10 · S-15 · S-16, W-3 · W-7)** — `node --test … test/i18n-catalogs.test.ts` → `v5-catalogs.txt`, 그리고 `apps/desktop` 에서 `grep -n '"install":\|"installLocal":\|"subtitle":\|"dataHint":' src/common/i18n/locales/ko.json` → `v5-grep.txt`.
  - 기대: 테스트는 `exit=0`, `ℹ fail 0` 이고 다음 3개가 `✔` 입니다.
    - `ko and en have the same keys`
    - `ko has no untranslated English values` — 모든 문자열 값에서 `{…}` 를 지운 뒤 한글(`[가-힣]`)이 없는 키는 허용 목록뿐입니다. 허용 목록은 테스트에 고정하고 판정에 그대로 옮깁니다(사전 실행으로 정한 처음 목록: `language.en`, `settings.sync.serverUrlPlaceholder`; 구현 중 더하면 A-n).
    - `help copy in ko uses polite haeyo style` — `help.*` 의 `description` · `content` 에 `습니다` · `입니다` · `십시오` 가 없습니다.
  - grep 은 `exit=0` 이고, `settings.errors.install` · `installLocal` 값에 한글이 있고 `Failed to` 가 없으며, `sidebar.calendar.subtitle` 이 "Calendar" 가 아니고, `"dataHint":` 줄이 없습니다.
  - 실패: `✖`, 이름 없음, 또는 grep 의 기대와 다름.
  - 사전 실행: grep 을 실행해 지금 `ko.json:83`(dataHint) · `107` · `171–172`(영어) · `197`("Calendar") · `203` 을 확인했습니다(`dryrun/v5-grep-2.txt`, 6줄). 한글 없는 ko 값은 5개입니다: 허용할 `language.en` · `settings.sync.serverUrlPlaceholder` 와, 고칠 `settings.errors.install` · `installLocal` · `sidebar.calendar.subtitle`(`dryrun/v5-nohangul.txt`).
- **V-6 다가오는 일정 선택 (S-8, W-3)** — `node --test … test/upcoming.test.ts` → `v6-upcoming.txt`.
  - 기대: `exit=0`, `ℹ fail 0`, 다음 6개가 `✔` 입니다.
    - `keeps schedules starting within 7 days` — 6일 23시간 뒤는 들고, 7일 1분 뒤는 빠집니다.
    - `sorts by start and keeps at most 5`
    - `excludes ongoing and past schedules`
    - `counts recurring occurrences one by one` — 매일 반복이 5개를 채웁니다.
    - `applies exceptions` — 건너뛴 발생은 빠지고, 시각이 바뀐 발생은 바뀐 시각으로 정렬됩니다.
    - `all-day schedules starting tomorrow are included`
  - 실패: `✖` 또는 이름 없음.
  - 사전 실행: 구현 뒤.
- **V-7 알림 목록 · 업데이트 상태 · 설정 적용 (S-7 · S-12 · S-13, W-4 · W-5 · W-6)** — `node --test … test/notification-list.test.ts test/update-status.test.ts test/settings-apply.test.ts` → `v7-units.txt`.
  - 기대: `exit=0`, `ℹ fail 0`, 다음 9개가 `✔` 입니다.
    - `fired items are newest first and capped at 50` (목록을 읽기만 해서는 읽음이 바뀌지 않음 포함)
    - `mark all read clears the unread count`
    - `upcoming keeps candidates within 24 hours sorted by time` (이미 울린 후보와 지난 시각은 제외)
    - `manual check goes checking then latest`
    - `available opens the update dialog state`
    - `error keeps the message`
    - `unsupported install never stays checking` (개발 빌드 · `isUpdaterActive` 거짓 · `null` 반환)
    - `invalid input is rejected without calling save`
    - `settings save failure restores the stored value`
  - 실패: `✖` 또는 이름 없음.
  - 사전 실행: 구현 뒤. 이 모듈들은 Electron · DOM 없이 불러올 수 있어야 합니다(설계).
- **V-8 화면 확인 (S-5 · S-7 ~ S-14 · S-16, W-2 ~ W-7)** — `pnpm dev` 와 같은 개발 실행(렌더러는 프로젝트 설정의 Vite 개발 서버, 메인은 미패키지 실행)을 격리된 cwd · user-data 로 띄운 앱에서 라이트와 다크 각각 아래 9개를 확인합니다(2026-11-20 정정: `pnpm dev` 는 실제 `~/.config/Croffle` 과 작업 폴더를 쓰고 화면에 창을 띄움).
  - PM 이 Playwright `_electron` 으로 하거나 사용자가 하며, 누가 무엇으로 했는지와 조건별 결과를 `v8-light.txt` · `v8-dark.txt` 에 적습니다.
  - 스크린샷 18장(`v8/screens/1-light.png` … `9-dark.png`)을 남기고 `ls -l .loctx/reports/1.3.2/test/v8/screens/` 를 `v8-screens-ls.txt` 에 남깁니다(2026-11-20 경로 정정, 1차 감사 A-4).
  1. 종(S-7):
     - 알림이 없을 때 팝오버에 "새 알림이 없습니다"가 보입니다.
     - 2분 뒤 일정에 1분 전 알림을 걸면, 울리기 전에는 "곧 울릴 알림"에 보입니다.
     - 울리면 종에 `1` 이 붙고, 팝오버의 울린 알림 맨 위에 굵은 글자로 보입니다.
     - 팝오버를 닫았다 다시 열어도 `1` 이 남습니다. "모두 읽음"을 누르면 사라집니다.
     - 항목을 누르면 그 일정 보기 창이 열립니다.
     - 앱을 다시 켜면 울린 알림 목록은 비어 있습니다.
  2. 다가오는 일정 · 우선순위(S-5 · S-8):
     - 오늘 +1일(시각 지정), +3일(종일), +8일 일정을 만들면 앞의 둘만 시작 순으로 보입니다.
     - 각 항목에 색 점 · 제목 · 날짜와 시각(종일은 날짜만)이 있습니다.
     - 더블클릭하면 보기 창이 열립니다.
     - 하나를 지우면 바로 사라집니다.
     - 오른쪽 사이드바와 일정 창의 우선순위가 점 + 글자(낮음 · 보통 · 높음)로 보입니다.
  3. 테마(S-9):
     - 좌클릭으로 다크로 바꾸면 설정 › 일반의 테마가 "다크"이고, 다시 켜도 다크입니다.
     - 우클릭하면 라이트 / 다크 / 시스템 메뉴가 열립니다.
  4. 부제(S-10): 오늘 일정이 2개면 "오늘 일정 2개", 0개면 "오늘 일정 없음"이고, 제목을 되풀이하는 부제가 없습니다.
  5. 버전 타일(S-11 · S-12): 누르면 앱 이름 · 버전 · 업데이트 확인 버튼 · 상태가 보입니다. `pnpm dev` 에서는 타일과 설정 › 일반 › 업데이트 행 모두 버튼이 꺼져 있고 이유 힌트가 보입니다.
  6. 설정 바로 적용(S-13):
     - 저장 · 취소 버튼이 없습니다.
     - 스위치(예: 주차 표시)를 바꾸고 설정을 닫았다 열면 바뀐 값입니다.
     - 언어를 바꾸면 바로 문구가 바뀝니다.
     - 달력 탭에 "저장/취소 버튼과 무관합니다" 문구가 없습니다.
     - 동기화 탭의 연결 · 로그인 버튼은 그대로입니다.
     - (잘못된 값 · 저장 실패는 V-7 로 봅니다.)
  7. 확장 삭제(S-14): 확장이 하나 이상 설치된 상태에서 삭제를 누르면 앱 확인 창(destructive 버튼)이 뜹니다. 취소하면 남고, 확인하면 지워집니다.
  8. 강조 색상(S-3 · S-5): 색조를 200 으로 바꾸면 주 버튼 · 오늘 칸 · 포커스 링 · 달력 제목 · 오늘 숫자가 그 색조가 되고, 배경 · 사이드바 바탕 · 테두리 · 호버는 밝기는 그대로인 채 그 색조로 살짝 물듭니다. 상태색(오류 · 성공 · 경고 · 정보)은 그대로입니다. (2026-11-20 정정, S-3 변경에 맞춤; 처음 문구는 "배경 · 사이드바 바탕 · 테두리 · 호버는 그대로")
  9. 도움말(S-16): 4단계를 모두 넘기면 문구가 해요체이고, 일정 추가(우클릭 메뉴 · 추가 버튼; 날짜 더블클릭은 그날 보기), 다가오는 일정, 테마 좌클릭 저장, 설정 바로 적용, 알림 종, 업데이트 확인과 맞습니다.
  - 실패: 하위 조건 하나라도 다르거나 미확인, 또는 스크린샷이 없거나 0바이트.
  - 사전 실행: 수동이라 하지 않았습니다.
- **V-9 패키지 빌드의 업데이트 확인 (S-11 · S-12, W-5)** — 루트에서 `pnpm build:unpack` → `v9-build-unpack.txt`. 그다음 두 번 실행합니다.
  - (가) `./apps/desktop/dist/linux-unpacked/croffle`
  - (나) `APPIMAGE=/tmp/croffle.AppImage ./apps/desktop/dist/linux-unpacked/croffle`
  - 두 경우 모두 설정 › 일반 › 업데이트의 자동 업데이트를 끈 채 "지금 확인"을 누르고, 버전 타일 팝오버도 엽니다.
  - 관찰과 해당 실행의 로그 발췌는 `v9-ga.txt` · `v9-na.txt` 에 남깁니다. 로그는 실행에 쓴 user-data 의 `logs/main.log` 입니다(2026-11-20 정정: 격리된 `--user-data-dir` 로 실행).
  - 기대:
    1. 빌드가 `exit=0` 입니다.
    2. 두 경우 모두 버튼이 켜져 있습니다.
    3. (가) 는 5초 안에 "확인할 수 없음"과 이유가 보이고, "확인 중"에 머물지 않습니다.
    4. (나) 는 "확인 중" 뒤 30초 안에 "최신 버전"(패키지 버전 1.3.1 = 최신 stable) · "새 버전 X" · "실패(이유)" 중 하나가 되고, "최신 버전"이면 토스트가 없습니다.
    5. 두 경우 모두 버전 타일과 설정 행이 같은 상태입니다.
    6. (나) 의 로그에 확인 시작 줄이 있습니다.
  - 실패: 빌드 `exit≠0`, 버튼이 꺼짐, "확인 중"에 머묾, "최신 버전"에 토스트, 두 곳의 상태가 다름, 또는 (나) 로그에 확인 줄이 없음.
  - 사전 실행: 하지 않음(구현 뒤). gemini 가 `dist/linux-unpacked/resources/app-update.yml` 이 있음과 `AppImageUpdater.isUpdaterActive()` 를 확인했고, PM 이 `node_modules/electron-updater/out/AppImageUpdater.js` · `AppUpdater.js:254–256` 으로 다시 확인했습니다.
- **V-10 DESIGN.md (S-17, W-8)** — `apps/desktop` 에서 아래를 각각 실행해 `v10-1.txt` ~ `v10-8.txt` 에 남깁니다.
  1. ``grep -nE '`(success|warning|info)`' DESIGN.md`` → 세 이름이 모두 나옴.
  2. `grep -n 'croffle-primary-strong' DESIGN.md` → 1줄 이상.
  3. `grep -n '12px' DESIGN.md` → 1줄 이상.
  4. `grep -nE 'shadow-(xs|sm|surface|lg)' DESIGN.md` → 네 이름이 모두 나옴.
  5. `grep -n '9RJ3iMqwtduoMasBnMe4ky' DESIGN.md` → 1줄 이상.
  6. `grep -n 'amber' DESIGN.md` → 출력 없음, `exit=1`.
  7. ``grep -nE 'Do not rename them or change their values|Error text is `text-destructive-foreground`|`--radius` is 10px|follow the accent color the user picks' DESIGN.md`` → 출력 없음, `exit=1`.
  8. `nl -ba DESIGN.md` → `exit=0`. PM 이 S-17 의 항목을 하나씩 `v10-8.txt:줄` 과 대조해 판정에 적습니다.
     - ① 상태 토큰 · `croffle-primary-strong` 표
     - ② 별칭 규칙
     - ③ 오류 글자 규칙
     - ④ 앰버 예외 폐지
     - ⑤ 반경 표
     - ⑥ 그림자 4종
     - ⑦ 강조 범위 4개 토큰
     - ⑧ 행동 규칙과 링크
  - 실패: 1 ~ 5 · 8 이 기대와 다름, 6 · 7 에 출력, 또는 대조에서 빠진 항목.
  - 사전 실행(`dryrun/v10-*-2.txt`): 지금 1 · 2 · 3 · 5 는 출력이 없고, 4 는 2줄(옛 문장 속 `shadow-sm` · `shadow-lg`)이며, 6 은 1줄, 7 은 4줄입니다.
- **V-11 릴리스 노트 (제약 "릴리스 노트에 적습니다", release 단계)** — 이 V 는 test 단계가 아니라 release 단계(rc.1 노트를 쓴 뒤)에 실행합니다. 루트에서 다음을 각각 실행합니다.
  - `nl -ba .loctx/release/1.3.2-rc.1_Pre-Release.md` → `v11-note.txt`
  - `grep -n 'croffle-' .loctx/release/1.3.2-rc.1_Pre-Release.md` → `v11-vars.txt`
  - `git check-ignore -v .loctx/release/1.3.2-rc.1_Pre-Release.md` → `v11-ignored.txt`
  - 기대:
    - `.github/release-notes.md` 의 데스크톱 Pre-release 양식을 따릅니다.
    - 한국어와 영어 두 절 모두에 "`--croffle-*` 이름은 그대로, 값이 바뀌어 확장 화면 색이 달라짐" 안내가 있습니다(`v11-vars.txt` 2줄 이상).
    - `check-ignore` 가 `exit=0` 으로 무시 규칙을 보입니다.
  - 실패: 파일 없음, 한쪽 언어에 안내가 없음, 또는 추적 대상임.
  - 사전 실행: `.loctx/release/` 가 아직 없어 release 단계에서.
- **V-12 라이트 컨트롤 바탕 (S-18, W-10)** — `apps/desktop/src/renderer/src/components/ui` 에서 `grep -nE "bg-transparent|bg-background|bg-card" input/Input.vue select/SelectTrigger.vue textarea/Textarea.vue button/index.ts` → `v12-controls.txt`.
  - 기대: 네 파일의 컨트롤 바탕 줄이 `bg-card` 이고 같은 줄에 `dark:bg-input/30` 이 있습니다. 컨트롤 자신의 바탕에 `bg-transparent` · `bg-background` 가 없습니다(`file:bg-transparent` 같은 하위 요소는 판정에 줄마다 적고 허용).
  - 실패: 하나라도 `bg-card` 가 아니거나 다크 바탕이 빠짐.
- **V-13 디자인 시스템 · 캔버스 (S-19, W-11)** — 두 아티팩트에서 `project/tokens.json` · `project/README.md` · `project/components/Input/README.md` · `project/components/Button/README.md` · `project/components/Select/README.md` · `project/components/bundle.css`(시스템), `project/croffle-theme.css` · `project/ds/croffle/tokens.json` · `project/ds/croffle/components/bundle.css`(캔버스)을 읽어 `.loctx/reports/1.3.2/rc2/v13/` 에 받고, 버전과 함께 `grep -n` 결과를 `v13-ds.txt` 에 남깁니다.
  - 기대: 시스템 README 에 강조 색조 절이 있고 따르는 토큰 · 고정 토큰 · L · C · `d` · 기준 색조 70 을 말합니다. 두 tokens.json 의 다크 중립색 값이 `index.css` 의 기본 70° 값과 같습니다. 캔버스 테마의 중립색이 `calc(var(--croffle-accent-hue) + d)` 입니다. Input · Select · Button 문서와 두 bundle.css 의 라이트 컨트롤 바탕이 `card` 입니다.
  - 실패: 하나라도 없음, 또는 값이 앱과 다름.
- **V-14 rc.2 화면 (S-3 변경 · S-18, A-21 ~ A-23)** — 격리한 개발 실행(V-8 의 실행 조건)에서 라이트 · 다크 × 강조 색조 70 · 250 의 전체 화면, 설정 창, 일정 추가 창을 찍어 `.loctx/reports/1.3.2/rc2/v14/` 에 둡니다. 함께 각 경우의 `getComputedStyle` 값(`--background` · `--sidebar` · `--card` · `--input` 바탕 · `--primary`)을 `v14-values.txt` 에 남깁니다.
  - 기대: 70° 에서 라이트 값이 시스템 hex, 250° 에서 중립색이 푸른 쪽으로 바뀌고 명도는 같습니다. 라이트 입력칸 · 셀렉트 · outline 버튼의 바탕이 #ffffff 입니다. 버전 타일 · 보기 셀렉트 · 직접 색 칸이 A-21 ~ A-23 대로입니다.
  - 실패: 값이 다르거나 화면에서 하나라도 다름.

## 검토
- 2026-11-19 PM 자체 검토입니다(위임표에 plan-reviewer · qa-plan-reviewer 가 없음). 사용자 원문, intent, 제품 경계와 대조했습니다.
- intent 밖으로 나간 것은 없습니다. PM 해석으로 더한 것은 Pretendard 500 · 600(S-4), 알림 24시간 범위 · 열기만으로는 읽음이 아님(S-7), 진행 중 일정 제외(S-8), 수동 확인 때 토스트 생략(S-12)입니다. 사용자는 이 해석들을 받아들였습니다("PM 해석: ㅇㅋ", 2026-11-19).
- 2026-11-19 qa-designer(gemini-4-argon-high, `agy -p`, 사용자 "gemini한데 요청해.") 검토. 과제는 `.loctx/reports/1.3.2/qa-design-task.md`, 답은 `.loctx/reports/1.3.2/qa-design-answer.md`(원문 그대로) 입니다. PM 이 코드로 확인한 뒤 받아들인 것:
  - V 보강: S-1 · S-2 · S-3 · S-4 · S-16 의 빠진 검사, V-4a 정규식(`emerald` · `rose` · `rgba(`), V-4b 의 `ui/` 포함, V-5 의 한 단어 영어, V-8 의 하위 조건, V-10 의 옛 문장 음성 검사, V-11 · V-4f 추가.
  - spec 지적 1(설정에 글자 · 숫자 칸은 확장 구성 폼뿐) → S-13 에 대상을 적고, 잘못된 값의 확인은 단위 테스트로 합니다. `dataHint` 정리를 더했습니다.
  - 지적 2(Linux unpack 실행에서 `checkForUpdates()` 가 이벤트 없이 `null`) → S-12 · 설계에 `확인할 수 없음` 상태와 수동 확인의 `autoUpdate` 무시를 적었습니다. PM 이 더 찾은 것: 이미 있는 호스트 전용 `window.checkForUpdates` 를 고쳐 쓰므로 새 IPC 가 없습니다.
  - 지적 3(`components/ui` 6개 파일) → 설계 · W-2 에 적었습니다.
  - 지적 4(`setting-service.ts:29` 의 69.8) → S-3 · W-1.
  - 지적 5(Pretendard 500 · 600 파일은 이미 있음) → S-4 를 고치고 글꼴 용량 우려를 지웠습니다.
  - 지적 6(도움말 단계 구성) → S-16 에 4단계와 담을 내용을 적었습니다.
  - 받아들이지 않은 것: V-5 의 "help 에 합쇼체 0건" 은 받아들였지만 제목(`title`)은 명사라 검사에서 뺍니다. V-1 의 `All matched files use the correct format.` 문자열은 사전 실행 원본에서 확인한 뒤 씁니다.

## 조정 기록
- A-1 (2026-11-19, W-1, A-20 으로 대체) 라이트 `--primary` 의 oklch 명도를 시스템 #98641b 의 0.547 에서 0.534 로 낮췄습니다 · 0.547 이면 색조 192° 부근에서 `primary-foreground` 대비가 4.39:1 로 떨어져 S-3 · V-3 의 "모든 색조 4.5:1" 을 지키지 못합니다. 0.534 면 최저 4.61:1 입니다. 기본 색조 70° 에서는 거의 같은 색(#93601a 부근)입니다. `--croffle-today-bg` · 설정 미리보기도 같은 명도를 씁니다.
- A-2 (2026-11-19, W-1) 다크의 강조색은 설정 색조에 차이를 더합니다(`primary` +5°, `accent-foreground` · `ring` +8.6°) · 시스템 다크 값(#d9a353 75°, #ecc079 78.6°)이 라이트(70°)와 색조가 달라, 색조 하나로 두 테마를 맞추려면 차이가 필요합니다. 기본 70° 에서 시스템 hex 와 같습니다.
- A-3 (2026-11-19, W-1) 라이트 `--input` 을 #988c74 에서 #968a72 로 한 단계 진하게 했습니다 · 시스템 값은 `--sidebar`(#f6f2e9) 위에서 2.97:1 로 S-5 · V-3 의 3:1 에 못 미칩니다. #968a72 는 sidebar 3.04 · background · card 그 이상입니다. 시스템도 사용자 지시("A-3. 고쳐.", 2026-11-20)로 같은 값으로 고쳤습니다(디자인 시스템 버전 `1795105637-a094`, `tokens.json` 의 `input` light 값과 usage 문장).
- A-4 (2026-11-19, W-3) `listOccurrenceStarts` 를 `src/common/reminder.ts` 에서 새 `src/common/occurrences.ts` 로 옮기고 `reminder.ts` 는 다시 내보냅니다. 테스트용 해석 훅 `test/support/{register.ts,ts-resolve.mjs}` 를 더했습니다 · `node --test` 의 타입 제거 모드는 `reminder.ts → i18n → enums.ts`(TS enum)를 읽지 못하고, 공용 코드는 확장자 없는 import · JSON · `rrule` CJS 를 씁니다. V 명령은 그대로입니다. 다가오는 일정은 달력이 보는 달과 상관없이 같은 범위 조회(`schedules.getAll`)로 오늘부터 8일을 읽습니다(`upcoming-store.ts`) · 0f1ce63
- A-5 (2026-11-19, W-2) 상태색을 글자로 쓰는 쌍(`success` · `warning` · `info` × `background` · `card` · `sidebar`)을 V-3 의 `text pairs hold 4.5:1 in both themes` 에 더했습니다 · 동기화 패널의 경고 글자를 `text-warning` 으로 바꾸면서 대비 근거가 필요했습니다. `warning` 은 `muted` 위에서 4.36:1 이라 그 조합은 쓰지 않습니다(DESIGN.md) · 4458a7e
- A-6 (2026-11-19, W-4) `components/ui/popover/PopoverContent.vue` 의 기본 모양을 `rounded-md shadow-md` 에서 `rounded-xl shadow-lg` 로 바꿨습니다 · 시스템 규칙(팝오버 `radius-xl` · `shadow-lg`). 설계에 적은 6개 밖의 `ui` 파일이며, 날짜 선택 팝오버도 같이 바뀝니다 · cd17d8a
- A-7 (2026-11-19, W-5) 수동 확인은 사용자가 건너뛰기로 한 버전도 "새 버전"으로 보입니다. `checkForUpdates()` 가 `null` 을 돌려주면 "실패"로 알립니다 · 건너뛴 버전이면 electron-updater 이벤트가 렌더러에 오지 않아 "확인 중"에 멈춥니다(S-12 "멈추지 않습니다") · 0fe0860
- A-8 (2026-11-19, W-6) 문구 키를 더했습니다: `settings.errors.notANumber`, `settings.extensions.uninstallTitle` · `uninstall`. 저장은 바뀐 뒤 250ms 를 모아 한 번 합니다(색 선택을 끄는 동안 저장이 몰리지 않게) · c80a644
- A-9 (2026-11-19, W-8) DESIGN.md 의 검사 grep 을 `apps/desktop/scripts/check-ui-tokens.sh` 로 옮겼습니다 · V-10 6 은 DESIGN.md 에 `amber` 가 없기를 기대하는데, 팔레트 이름을 나열한 검사 명령이 그 줄에 걸립니다. V-4 a ~ c 와 같은 명령입니다 · 5eb9a5d
- A-10 (2026-11-19, W-6, 리뷰 1 F-1 · F-2) 설정 화면은 화면이 바꾼 칸만 최신 저장값 위에 얹어 저장합니다(`mergeEdits`). 다른 곳의 저장(`SETTINGS_UPDATE`)을 바로 받아들이고, 저장은 차례로 하나씩 하며, 창을 닫을 때 기다리던 저장을 바로 보내고 다시 열 때 고친 내용을 버리지 않습니다. 테스트 `screen edits do not overwrite a change saved elsewhere` 를 더했습니다 · 318b3c0
- A-11 (2026-11-19, W-5, 리뷰 1 F-5) 수동 확인은 main 에 묻기 전에 "확인 중"으로 두고, main 의 답은 아직 확인 중일 때의 "확인할 수 없음"만 반영합니다(`applyCheckAnswer`). 테스트 `a late answer does not override a result that arrived first` 를 더했습니다 · bb7c3b2
- A-12 (2026-11-19, W-3 · W-4, 리뷰 1 F-3 · F-4 · F-6) 다가오는 일정 · 알림 항목을 열 때 그 일정을 달력 store 에 넣어 보기 창이 찾게 합니다. 알림 목록은 팝오버를 열 때와 1분마다 다시 읽습니다. 다가오는 일정은 조회에 성공한 뒤에만 날짜를 기록해 실패하면 다음 틱에 다시 읽습니다 · c7bcf95
- A-13 (2026-11-19, W-6, 리뷰 2 G-1 · G-2) 설정 화면의 편집 상태를 순수 클래스 `SettingsDraft`(`src/common/settings-apply.ts`)로 옮겼습니다. 저장이 도는 동안은 저장 중인 값을 기준으로 삼아, 그 사이의 편집(이전 값으로 되돌린 것 포함)을 지키고 다음에 저장합니다. 실패하면 저장하려던 칸만 되돌립니다. 테스트 `an edit made while saving is kept, even one back to the old value` · `a failed save puts back only the fields it tried to save` 를 더했습니다. 확장 구성 폼의 key 에 탭 id 를 넣었습니다 · 7203f17
- A-14 (2026-11-20, W-6, 리뷰 3 H-1 · H-2) 설정 저장을 패치 방식으로 다시 짰습니다(사용자 "가 방식으로."). 컨트롤마다 바뀐 칸 하나만 `settings.update` 로 보내고(메인이 최신 저장값과 합침), 화면은 최신 저장값 위에 끝나지 않은 패치를 얹은 것입니다(`PendingPatches`). 외부 반영은 저장값만 바꾸고, 실패한 패치는 버려 최신 저장값으로 돌아갑니다. A-10 · A-13 의 기준값 비교(`mergeEdits` · `SettingsDraft`)는 걷어 냈습니다. 원인: 화면 전체를 기준값과 비교해 편집을 추론한 설계가 외부 반영 · 저장 중 · 실패 순서마다 틀어짐(W-6 리뷰 3회 연속 실패). 테스트 `changes saved elsewhere during a save are never taken as edits` · `a failed save keeps the latest stored values received meanwhile` 를 더하고 V-7 의 두 이름은 그대로 둡니다 · fdd157b
- A-15 (2026-11-20, W-6 · W-3, 리뷰 4 K-1 ~ K-3) 저장값은 `SETTINGS_UPDATE` 와 조회로만 받고, 저장 응답은 패치를 끝내는 데만 씁니다. 색 선택은 고르는 즉시 대기 패치로 화면에 남기고 전송만 늦추며(`PendingPatches.revise`), 다른 설정의 이벤트가 와도 화면의 색 · 테마를 다시 얹습니다. 확장 구성 저장이 실패하면 그 뒤의 입력은 남깁니다. 제목 표시줄 테마 · 닫기 동작 창도 `general` 전체 대신 한 칸만 보냅니다. 테스트 `a late reply never brings back an older stored value` · `a held-back colour shows on screen until it is sent` 를 더했습니다 · 5540979
- A-16 (2026-11-20, W-6, 리뷰 5 L-1) 화면의 외관(테마 · 강조 색조)은 앱 설정 store 한 곳에서 "저장값 + 설정 화면의 미리보기"로 합성합니다(`lookOf`, `setThemeDraft` · `setAccentHueDraft` · `endPreview`). 리스너 등록 순서와 상관없이 미리보기가 남습니다. A-15 의 "이벤트가 응답보다 먼저 온다"는 가정은 버리고, 저장한 패치는 요청 전에 걸어 둔 "다음 저장값"(최대 1초)을 기다린 뒤 끝냅니다. 테스트 `a previewed colour or theme survives a settings event` 를 더했습니다 · 6da4edb
- A-17 (2026-11-20, W-6, 리뷰 6 M-1 ~ M-3) 메인이 설정을 저장할 때마다 revision 을 올리고, 저장 응답과 호스트 전용 채널(`croffle:settings:stored`)로 `{ revision, settings }` 를 보냅니다(사용자 "가"). 공개 `SETTINGS_UPDATE` · `packages/types` 는 그대로입니다. 설정 화면과 앱 설정 store 는 번호가 더 큰 저장값만 받고, 패치는 자기 번호 이상의 저장값이 왔을 때 끝납니다(`PendingPatches.confirm`). A-16 의 "다음 저장값을 1초 기다림"은 걷어 냈습니다. 앱 외관의 미리보기는 남은 패치 중 가장 새 값으로 한 번에 바꾸고(`setPreview`), 설정 화면이 받은 저장값은 store 에도 먼저 넘겨 미리보기가 끝날 때 옛 색이 칠해지지 않게 했습니다(가상 디스플레이에서 setProperty 기록으로 확인). 메인 `settings:update` · `settings:getStored` 에 `validateSender` 를 걸었습니다. 테스트 `a patch stays until stored settings of its revision arrive, in either order` · `the preview is the newest unfinished value; a later failure falls back to an earlier one` 를 더하고 기존 테스트를 revision 으로 고쳤습니다 · 629aa7b
- A-18 (2026-11-20, W-6, 리뷰 7 N-1) 메인의 설정 저장은 파일을 쓴 뒤에만 값과 revision 을 바꾸고, 쓰기가 실패하면 예외를 내 IPC 가 실패합니다(적용 · 방송 없음; 렌더러는 패치를 버리고 오류 토스트). 1.3.1 까지 쓰기 실패를 로그만 남기던 동작을 바꾼 것입니다. 병합 · 기록을 Electron 없는 `src/main/setting/settings-commit.ts` 로 빼고 테스트 `a one-field patch is merged into its section` · `a failed write keeps the stored settings and revision` 를 더했습니다 · f42d63c
- A-19 (2026-11-20, W-5, test 중 발견) 설정 › 일반 › 업데이트 행의 버튼 이름을 S-12 대로 "지금 확인"으로 바꿨습니다(버전 타일은 "업데이트 확인" 그대로; `UpdateCheck label="now"`, 키 `update.check.now`). V-9 준비 중 spec 과 구현의 차이를 PM 이 찾았습니다 · ed81e95
- A-20 (2026-11-20, W-1, A-1 대체) 라이트 `--primary` · `--croffle-today-bg` 의 명도를 시스템 값 0.5469 로 맞추되(사용자 "앱이 쓰는 라이트 primary 명도는 최신 디자인 시스템의 값에 맞춤", 방식 "나 방식"), 색조 185° 부근에서만 `0.0095 × max(0, cos(색조 − 185°))` 만큼 낮춥니다. 기본 색조 70° 에서는 정확히 0.5469(시스템 #98641b)이고, 모든 색조에서 `primary-foreground` 대비가 4.55:1 이상입니다(0.5469 고정이면 149–218° 에서 최저 4.39:1). 설정 미리보기는 같은 식(`accentLightness`)을 씁니다. Electron(Chromium)이 식을 계산하는 것을 가상 디스플레이에서 확인했습니다(70° → 0.5469, 185° → 0.5374). 테스트는 계산식을 읽고 기본 색조 값이 0.5469 인지도 봅니다(색 계산 도우미는 `test/support/color.ts` 로) · 21065fb
- A-21 (2026-11-20, W-5, rc.1 확인) 버전 타일을 캔버스(`D-Shell`)대로 다시 그렸습니다. 펼치면 정보 아이콘 · `CROFFLE v<버전>` 이 한 줄이고 새 버전을 알면 info 점이 붙습니다. 접으면 ghost 아이콘 버튼입니다. 오른쪽 사이드바의 `h-screen` 을 `h-full` 로 바꿨습니다 · `h-screen` 이 제목 표시줄(32px) 아래에서 창 밖으로 32px 넘쳐, rc.1 의 바닥 여백(`mb-4` + `pb-4`)이 그만큼을 가려 타일이 아래로 붙어 보였습니다. 이제 아래 끝이 왼쪽 사이드바의 아이콘 버튼과 같은 12px 입니다(가상 디스플레이에서 확인) · deb8555
- A-22 (2026-11-20, W-7, rc.1 확인) 달력 보기 셀렉트가 지금 보기의 이름을 직접 그립니다(`SelectValue` 슬롯) · 한국어 설정으로 띄우면 목록을 한 번 열기 전까지 "Month", 연 뒤에 "월간" 이 보였습니다(고치기 전 코드로 재현, 고친 뒤 ko "월간" · en "Month"). 1.3.1 이전부터 있던 문제입니다 · e108bc5
- A-23 (2026-11-20, W-2 · W-6, rc.1 확인) 일정 색 · 강조 색상의 직접 선택 칸은 값이 미리 준비된 색이 아니면 그 색으로 채우고 선택 테두리를 그립니다. 강조 색상의 색 입력은 `change` 도 받습니다 · 직접 고른 색이 어디에도 보이지 않았고, 운영체제 색 선택 창은 확인을 누를 때 `change` 만 보낼 수 있습니다. DESIGN.md 에 적었습니다 · d07a872
- A-24 (2026-11-20, W-9) 라이트 `--warning` 을 #b05519 에서 #af5519 로 한 단계 진하게 했습니다 · 중립색이 색조를 따르면 `warning` 글자가 `sidebar` 위에서 색조 337° 부근 4.497:1 로 4.5 에 못 미칩니다(#af5519 는 모든 색조에서 4.52 이상). 시스템 값도 W-11 에서 같이 고칩니다 · 845d998
- A-25 (2026-11-20, W-9) 라이트 `--input` 을 #968a72(L 0.6375)에서 #958971(L 0.6341)로 한 단계 진하게 했습니다 · 색조 146° 부근에서 `sidebar` 위 대비가 2.998:1 로 3:1 에 못 미칩니다(#958971 은 모든 색조에서 3.04 이상). 시스템 값도 W-11 에서 같이 고칩니다 · 845d998
- A-26 (2026-11-20, W-10) 시간 입력(`time-field.vue`)과 태그 입력(`ui/tags-input/TagsInput.vue`)의 바탕도 라이트 `bg-card` 로 했습니다(태그 입력은 다크 `dark:bg-input/30` 도 더함) · 같은 모양의 복합 컨트롤이라 W-10 의 4개만 바꾸면 한 창 안에서 바탕이 섞입니다. DESIGN.md 에 적었습니다 · 94305e6
- A-27 (2026-11-20, W-11 · W-9, 리뷰 12 R-1 · R-2) 캔버스와 시스템의 라이트 `primary` · 오늘 칠에 앱의 명도 보정식(A-20)을 넣고, 캔버스 라이트 `accent-foreground` 의 +0.6 차이를 앱처럼 없앴습니다(캔버스 `1795164795-8f09`, 시스템 `1795164798-de98`; 앱과 캔버스의 모든 토큰이 색조 0~330°(30° 간격)에서 8비트로 같고, 캔버스도 모든 색조 대비 검사를 통과함을 확인). 토큰 테스트는 중립색의 L · C · d 를 직접 비교하고 고정 토큰 값을 표(`test/support/system-tokens.ts`)와 맞춥니다(L · 고정값 · d 를 조금씩 바꾸면 각각 실패함을 확인) · 4624007
- A-28 (2026-11-20, W-11, test V-13 대조) 시스템 · 캔버스 tokens.json 의 라이트 `accent-foreground` 를 #8a5b18 에서 #8a5b19 로 고쳤습니다 · 앱은 rc.1 부터 `oklch(0.5109 0.0998 var(--croffle-accent-hue))`(색조 차이 없음)이고 그 값이 70° 에서 #8a5b19 입니다. 대비 검사를 거친 앱 값에 시스템을 맞췄습니다(시스템 `1795166278-8f9a`, 캔버스 `1795166281-0fe5`; 재대조 49개 hex 불일치 0, `.loctx/reports/1.3.2/rc2/v13-ds.txt`). 저장소 변경 없음

## 우려
- **확장 화면 색이 바뀝니다.** `--croffle-*` 값이 바뀌므로, 이 변수를 읽는 확장의 화면 색이 1.3.2 에서 달라집니다. 이름은 그대로라 깨지지는 않습니다. 릴리스 노트에 적습니다.
- **색조 · 대비**: 강조 색상의 일부 색조에서 고정 L/C 가 sRGB 밖이면 실제 색이 잘려 대비가 떨어질 수 있습니다. W-1 의 테스트가 잡고, 깨지면 A-n 으로 L · C 를 조정합니다.
- **글꼴 용량**: Pretendard 500 · 600 woff2(각 약 765KB)가 빌드 산출물에 새로 들어갑니다. 파일은 이미 저장소에 있습니다.
- **시스템 문서 정정 (해결)**: 초안 때 시스템의 `StateBlock` · `Input` README 는 오류 글자를 `destructive-foreground` 로, `CalendarMonth` README 는 토큰에 없는 `shadow-calendar` 를 적고 있었습니다. 사용자가 2026-11-19 고쳤습니다("수정했어. 다시 봐줘"). 버전 `1795075960-81fe` 에서 세 README 가 `text-destructive` · `shadow-surface` 를 쓰고, 토큰 값은 이전 버전과 같음을 PM 이 확인했습니다.
- **설정 바로 적용의 범위**: 언어를 바꾸면 화면 문구가 곧바로 바뀝니다. 시작 프로그램 등록처럼 OS 에 닿는 설정은 지금 저장 때 하던 적용(`setting-applies.ts`)을 행마다 부릅니다.
- **수동 업데이트 확인은 패키지 빌드에서만** 실제 결과가 나옵니다. V-7 은 상태 흐름을, V-9 는 unpack 빌드(AppImage 가 아니면 `확인할 수 없음`, `APPIMAGE` 를 주면 실제 조회)를 봅니다. 새 버전 감지는 rc 설치본에서 release 단계에 확인합니다.

## 범위 밖
- 일정 공유(1.3.3), 할 일(1.3.4), 서버 WebUI(서버 0.3.1), pipit-hub.
- 울린 알림의 DB 저장, 알림 설정의 새 항목.
- `components/ui/**` 의 전면 재생성.
- 시스템 · 캔버스 아티팩트 수정 (2026-11-20 부터 S-19 의 두 항목은 범위 안).
- 일정 편집 창(`schedule-modal.vue`)의 배치 변경. 이번에는 토큰 · 색 규칙만 적용합니다.
