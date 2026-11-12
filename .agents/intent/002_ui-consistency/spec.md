---
intent: "002"
status: accepted
approvals:
  - { role: user, by: bluenyang, at: "2026-11-12", words: "진행", rev: "7132f50", kind: summary }
summary_for_user: |
  작업 단위는 6개입니다: W-1 `apps/desktop/DESIGN.md` 와 AGENTS.md 연결 → W-2 토큰 연결(`--primary` · `--ring` · `--border` · `--input`)과 색 덮어쓰기 제거 → W-3 버튼 → W-4 입력 · 선택(설정 행 240px 포함) → W-5 다이얼로그 · 카드 · radius → W-6 글자 · 색 정리.
  intent 에 없던 세부 결정이 넷입니다: 툴팁은 강조색이 아니라 전경색 배경, 모든 다이얼로그 radius 를 14px 로(부품 기본값 변경), 제목 표시줄용 버튼 size `icon-xs`(28px) 추가, 태그 추천은 32px 로 키우지 않고 배지 모양으로. 중요도 선택기는 높이 규칙에서 뺐습니다.
  검증 V-1~V-10 은 qa-designer 가 쓰고 제가 고쳤습니다. 정적 검사(grep · 게이트) 6개, 앱을 띄워 높이 · radius · 색을 재는 검사 3개, 라이트 / 다크 전후 스크린샷 1개입니다. Playwright 는 저장소 밖(`.loctx`)에 설치해 쓰고 `package.json` 은 건드리지 않습니다.
  남는 우려: 강조색 위 흰 글자 대비가 낮은 상태(약 2:1)가 스위치와 확인 모달 버튼으로 넓어집니다. 이번 범위에서는 고치지 않습니다.
---
## 요구 해석
경로는 `apps/desktop/src/renderer/src/` 기준입니다. 조사 항목 번호(A-1 …)는 `.loctx/reports/1.3.0/rc2-ui-audit.md` 의 것입니다.

- S-1 규칙을 `apps/desktop/DESIGN.md` 한 파일에 적고 AGENTS.md 가 가리킨다. ← 바라는 결과 "규칙이 문서 하나에 적혀 있고, 이후 UI 작업이 그 문서를 따릅니다", 계획 1
- S-2 `--primary` · `--ring` · `--border` · `--input` 이 croffle 토큰을 따른다. 강조색 설정을 바꾸면 주 버튼 · 스위치 · 체크박스 · 포커스 링이 함께 바뀐다. ← 바라는 결과 "강조색 설정을 바꾸면 …", 계획 2, 질의서 2 "응. 그게 훨씬 낫다."
- S-3 버튼 · 입력 · 선택의 높이는 36px, 조밀한 칩 · 토글은 32px 이다. ← 규칙 표 "컨트롤 높이", 질의서 1 "통일해."
- S-4 아이콘 버튼은 본문 32px · 사이드바 레일 36px · 제목 표시줄 28px 이다. ← 규칙 표 "아이콘 버튼"
- S-5 주 버튼은 `Button` 기본 variant 이고 색 class 를 붙이지 않는다. 보조는 `outline`, 되돌릴 수 없는 확인은 `destructive`, 줄 안의 삭제는 `ghost` + `text-destructive` 이다. ← 규칙 표 "주 버튼", "보조 · 위험"
- S-6 버튼 패딩 · 굵기는 size 가 정한다. 다이얼로그 푸터 버튼은 최소 80px, 간격 `gap-2`, 주 버튼이 맨 오른쪽이다. 설정 창 취소 / 저장의 패딩과 굵기가 같다. ← 바라는 결과 "설정 창 취소 / 저장 버튼의 너비와 굵기가 같습니다", 규칙 표 "버튼 너비"
- S-7 버튼 안 아이콘은 16px 이고 간격은 `Button` 의 `gap-2` 만 쓴다. ← 규칙 표 "버튼 안 아이콘"
- S-8 `<button>` · `<input>` 대신 `Button` · `Input` 을 쓴다. 예외는 색 견본, 창 버튼, `type="color"` 이다. ← 규칙 표 "부품", 계획 3 · 4
- S-9 세로로 쌓인 폼의 입력은 `w-full`, 설정 행의 컨트롤은 행 오른쪽 240px(`w-60`), 숫자 입력은 `w-24` 이다. ← 규칙 표 "입력 너비", 질의서 3 "고정해"
- S-10 테두리와 포커스 링은 토큰만 쓰고 화면에서 덮어쓰지 않는다. ← 규칙 표 "테두리 · 포커스"
- S-11 radius 는 컨트롤 8px · 표면 14px · 원형이고, 달력 카드 24px 만 예외다. ← 규칙 표 "radius"
- S-12 확인형 다이얼로그는 `sm:max-w-md`, 폼형은 `sm:max-w-lg`. 구분된 푸터는 `border-t bg-muted/20 px-6 py-4` 이다. ← 규칙 표 "다이얼로그 폭", "다이얼로그 푸터"
- S-13 다이얼로그 제목 `text-lg font-semibold`, 소제목 `text-base font-bold`, 라벨 `text-sm font-medium` 이다. ← 규칙 표 "글자"
- S-14 화면에 `neutral-*` · hex 를 쓰지 않고 토큰을 쓴다(창 닫기 버튼 hover 의 빨강은 예외). 죽은 class 를 지우고 `config-schema-form.vue:47` 의 다크 모드 글자색을 고친다. ← 규칙 표 "색", 계획 6
- S-15 기능 · 문구 · 레이아웃 구조는 그대로다. `--croffle-*` 토큰의 이름과 값은 바뀌지 않는다. ← 제약, 바라는 결과 "기능과 문구는 바뀌지 않습니다"
- S-16 라이트 / 다크에서 주요 화면의 전후 스크린샷을 남긴다. ← 계획 7, 제약 "라이트 / 다크 모두 확인합니다"

## 설계
**토큰 (S-2, S-10).** `index.css` 의 `:root` 와 `.dark` 에서 shadcn 토큰을 croffle 토큰에 연결합니다. `--croffle-*` 는 그대로 둡니다.

| shadcn 토큰 | 값 |
|---|---|
| `--primary` | `var(--croffle-primary)` |
| `--primary-foreground` | 흰색 (`oklch(1 0 0)`, 지금 손으로 붙이는 `text-white` 와 같음) |
| `--ring` | `var(--croffle-primary)` (부품이 `ring-ring/50` 으로 씀) |
| `--border`, `--input` | `var(--croffle-border)` |

강조색은 `--croffle-accent-hue` 로 정해지므로 연결만 하면 설정을 따라갑니다. 주 버튼 hover 는 부품 기본인 `bg-primary/90` 입니다 (`bg-croffle-hover` 덮어쓰기는 지웁니다).

`--primary` 를 쓰는 부품 가운데 툴팁(`ui/tooltip/TooltipContent.vue`)은 작은 글자라 강조색 배경이면 읽기 어렵습니다. 툴팁만 `bg-foreground text-background` 로 바꿉니다. 스위치 · 체크박스 · 달력 선택일 · 기본 배지는 강조색이 됩니다(의도한 결과).

**부품 (S-3 ~ S-8).** `ui/button` 의 size 는 그대로 씁니다: `default` 36 · `sm` 32 · `icon` 36 · `icon-sm` 32. 제목 표시줄용으로 `icon-xs`(28px) 하나를 추가합니다. 화면의 `h-9` · `h-10` · `h-11` · `px-*` · `font-*` · `h-7 w-7` 덮어쓰기는 지웁니다. 푸터 버튼에는 `min-w-20` 을 붙입니다.

**radius (S-11).** `ui/dialog/DialogContent.vue` 의 `rounded-lg` 를 `rounded-xl` 로 바꿔 모든 다이얼로그가 표면 radius 를 갖게 합니다. 달력 툴바 버튼은 `--radius-md`, 달력 이벤트의 `10px` 는 `var(--radius-lg)`(같은 값)로 바꿉니다.

**설정 행 (S-9).** 설정 창에서 라벨 아래 `w-full` 로 쌓인 `Select` 7개(`settings-modal.vue:634,657,806,837,857,877,1035`)와 `config-schema-form.vue` 의 select · 입력을, 이미 있는 "가져오기" 행과 같은 `flex items-center justify-between gap-4` + 오른쪽 `w-60` 으로 바꿉니다. 동기화 패널의 로그인 폼과 확장 설치 URL 은 세로로 쌓인 폼이므로 `w-full` 그대로입니다.

**칩.** 요일 선택 토글은 지금도 32px 이라 `Button size="sm"` 으로만 바꿉니다. 태그 추천(지금 약 22px)은 누르는 배지이므로 32px 로 키우지 않고 `Badge` 모양으로 맞춥니다. 중요도 선택은 아이콘과 글자를 세로로 쌓은 3칸 선택기라 높이 규칙의 대상이 아닙니다.

**손대지 않는 것.** 중요도 선택기, 색 견본 버튼, 창 버튼 3개(48px 폭), `type="color"` 입력, 일정 모달의 스위치 상자(B-7, 레이아웃 구조), FullCalendar 내부 격자.

## 작업 단위
순서대로 합니다. W-3 ~ W-6 은 W-2 의 토큰에 기대므로 W-2 를 되돌리면 함께 되돌립니다. 그 밖에는 단위마다 따로 되돌릴 수 있습니다.

- [x] W-1 (11e6c2c) `apps/desktop/DESIGN.md` (영어) 작성 + AGENTS.md Conventions 에 한 줄 — S-1
- [x] W-2 (8b3cc09) 토큰 연결(`index.css`) + 툴팁 색 + 화면의 색 덮어쓰기 제거: `bg-croffle-primary … text-white`, `#A68A64` · `#8E7554`, 컨트롤의 `border-croffle-border` · `border-neutral-*`, `ring-croffle-primary/30` · `focus-within:ring-2` — S-2, S-5, S-10. 조사 A-1, A-2, B-3, B-4, F-2, F-5
- [x] W-3 (6624f69) 버튼: 높이 · 패딩 · 굵기 덮어쓰기 제거, `icon-xs` 추가와 아이콘 버튼 크기 정리, `<button>` → `Button`, 푸터 `min-w-20` · `gap-2`, 아이콘 `mr-*` 제거, 삭제 버튼 표현 통일, 태그 추천을 배지 모양으로 — S-3 ~ S-8. 조사 A-3 ~ A-9, C-1(버튼). 의존: W-2
- [x] W-4 (83b5cca) 입력 · 선택: `<input>` → `Input`, `h-10` 제거(`time-field.vue` 포함), 숫자 입력 `w-24`, 설정 행 `w-60` — S-3, S-8, S-9. 조사 B-1, B-2, B-5, B-6, C-1(입력). 의존: W-2
- [x] W-5 (3f845c8) 다이얼로그 · 카드 · radius: `DialogContent` `rounded-xl`, 업데이트 모달 `sm:max-w-md`, 푸터 배경 · 그림자 통일, 손으로 쓴 카드를 토큰 class 로, 배지 덮어쓰기 제거와 확장 버전 표시를 `Badge` 로, 달력 툴바 · 이벤트 radius — S-11, S-12. 조사 C-2 ~ C-5, D-1 ~ D-5. 의존: W-2
- [x] W-6 (289eb75) 글자 · 색 정리: 라벨 `font-medium`, 도움말 제목, `neutral-*` → 토큰, 죽은 class 제거, `config-schema-form.vue:47`, 아이콘 `h-4 w-4` → `size-4`(버튼 안에서는 제거), 달력의 rem 숫자 → 토큰 — S-13, S-14. 조사 E-1 ~ E-5, F-1, F-3, F-4. 의존: W-2

측정에서 나온 보완(W-3 · W-4 · W-6 범위)은 `da272f6` 입니다.

스크린샷(S-16)은 작업 단위가 아니라 검증입니다: W-2 전에 "전", W-6 뒤에 "후"를 찍습니다.

## 검증 기준
qa-designer(google gemini-4-argon-high)가 작성하고 PM이 검토 · 수정했습니다. 실행 기준은 저장소 루트입니다. PM 수정: V-4의 높이 검사를 컨트롤이 있는 파일로 좁힘(접힌 사이드바 버튼 · 아이콘 상자의 `h-10`은 컨트롤 높이 덮어쓰기가 아님), V-7에서 중요도 선택기와 태그 추천을 32px 대상에서 뺌(설계 참조), V-7에 Playwright를 저장소 밖에 설치한다고 적음(지금 이 기계에 없음), V-9에서 체크박스를 뺌(화면에 체크박스가 없음), V-10에 업데이트 모달을 열 수 없을 때의 판정을 적음.

- V-1 **문서화(`DESIGN.md` · `AGENTS.md`) 및 범위 밖 파일·토큰 불변성 검사**
  - 검증 대상: S-1, S-15, W-1, W-2
  - 실행 방법:
    ```bash
    mkdir -p .loctx/reports/1.3.0/rc2 && {
      echo "=== 1. DESIGN.md 존재 및 필수 규칙 섹션 확인 ==="
      test -s apps/desktop/DESIGN.md && wc -l apps/desktop/DESIGN.md
      grep -nE 'AGENTS\.md|DESIGN\.md' AGENTS.md
      echo "=== 2. 범위 밖 경로 및 i18n 로케일 JSON 무변경 확인 ==="
      OUT_DIFF=$(git diff --name-only befde1f..HEAD -- \
        packages \
        apps/desktop/src/main \
        apps/desktop/src/preload \
        apps/desktop/src/common \
        apps/desktop/src/common/i18n/locales)
      printf "%s" "$OUT_DIFF"
      echo "out_of_scope_changed_count=$(printf "%s" "$OUT_DIFF" | grep -c '.' || true)"
      echo "=== 3. index.css 내 --croffle-* 토큰 이름·값 불변 확인 ==="
      INDEX_CSS=$(find apps/desktop/src/renderer/src -name 'index.css' -print -quit)
      diff -u \
        <(git show "befde1f:${INDEX_CSS}" | grep -E '^\s*--croffle-') \
        <(grep -E '^\s*--croffle-' "$INDEX_CSS")
      echo "croffle_token_diff_exit=$?"
    } > .loctx/reports/1.3.0/rc2/v01-docs-and-scope.log 2>&1
    ```
  - 기대 결과: `apps/desktop/DESIGN.md`가 존재하고 비어 있지 않으며(1줄 이상), `AGENTS.md`의 `Conventions` 섹션에 `apps/desktop/DESIGN.md` 참조가 1건 이상 존재합니다. `out_of_scope_changed_count=0`으로 `packages/`, `apps/desktop/src/main/`, `apps/desktop/src/preload/`, `apps/desktop/src/common/`(i18n 로케일 JSON 포함) 하위 변경 파일이 0개이며, `index.css`의 `--croffle-*` 토큰 선언 줄이 `befde1f` 대비 `diff -u` 차이 0줄(`croffle_token_diff_exit=0`)입니다.
  - 실패 판정: `apps/desktop/DESIGN.md`가 없거나 비어 있는 경우, `AGENTS.md`에 참조가 없는 경우, 범위 밖 경로 또는 i18n 로케일 JSON에 변경 파일이 1개라도 있는 경우, `index.css`의 `--croffle-*` 토큰 이름·값이 1줄이라도 변경·추가·삭제된 경우, 또는 로그 출력이 비어 있거나 판정이 모호한 경우 실패로 판정합니다.
  - 증적 경로: `.loctx/reports/1.3.0/rc2/v01-docs-and-scope.log`

- V-2 **정적 게이트 (타입 검사 · 린트 · 포맷 · 단위 테스트 · 프로덕션 빌드)**
  - 검증 대상: S-15, W-1, W-2, W-3, W-4, W-5, W-6
  - 실행 방법:
    ```bash
    mkdir -p .loctx/reports/1.3.0/rc2 && {
      pnpm typecheck &&
      pnpm lint &&
      pnpm exec oxfmt --check &&
      pnpm -r test &&
      pnpm build
      echo "static_gates_exit=$?"
      test -s apps/desktop/out/main/index.js && echo "main_bundle=ok"
    } > .loctx/reports/1.3.0/rc2/v02-static-gates.log 2>&1
    ```
  - 기대 결과: 5개 명령(`pnpm typecheck`, `pnpm lint`, `pnpm exec oxfmt --check`, `pnpm -r test`, `pnpm build`)이 모두 종료 코드 `0`(`static_gates_exit=0`)으로 통과하고, 오류 0건이며, `apps/desktop/out/main/index.js` 번들이 생성(`main_bundle=ok`)됩니다.
  - 실패 판정: 어느 한 명령이라도 0이 아닌 종료 코드를 반환하거나, 테스트 실패·린트/포맷/타입 오류가 1건이라도 있거나, 빌드 산출물이 없거나, 출력이 비어 있거나 모호한 경우 실패로 판정합니다.
  - 증적 경로: `.loctx/reports/1.3.0/rc2/v02-static-gates.log`

- V-3 **토큰 연결 및 화면의 색·테두리·포커스·죽은 클래스 제거 정적 검사**
  - 검증 대상: S-2, S-5, S-10, S-14, W-2, W-6
  - 실행 방법:
    ```bash
    mkdir -p .loctx/reports/1.3.0/rc2 && {
      INDEX_CSS=$(find apps/desktop/src/renderer/src -name 'index.css' -print -quit)
      echo "=== 1. index.css 토큰 매핑 (:root 및 .dark) 확인 ==="
      grep -nE -- '--primary:\s*var\(--croffle-primary\)|--primary-foreground:\s*oklch\(1 0 0\)|--ring:\s*var\(--croffle-primary\)|--border:\s*var\(--croffle-border\)|--input:\s*var\(--croffle-border\)' "$INDEX_CSS"
      echo "=== 2. TooltipContent.vue 전경/배경 토큰 확인 ==="
      grep -nE 'bg-foreground.*text-background|text-background.*bg-foreground' apps/desktop/src/renderer/src/components/ui/tooltip/TooltipContent.vue
      echo "=== 3. 화면 파일 금지 패턴 검사 (0건이어야 함) ==="
      ! grep -rnE 'bg-croffle-primary|bg-croffle-hover|#A68A64|#8E7554|ring-croffle-primary|focus-within:ring-2|border-croffle-border|text-croffle-text-dark|hover:bg-opacity-90' \
        apps/desktop/src/renderer/src --include='*.vue' --exclude-dir=ui && echo "forbidden_overrides=0"
      ! grep -rnE '\b(bg|text|border|ring|divide|placeholder|from|via|to)-neutral-[0-9]+' \
        apps/desktop/src/renderer/src --include='*.vue' --exclude-dir=ui && echo "neutral_classes=0"
      ! grep -rnE '(bg|text|border|ring)-\[#[0-9A-Fa-f]{3,8}\]' \
        apps/desktop/src/renderer/src --include='*.vue' --exclude-dir=ui && echo "arbitrary_hex_classes=0"
    } > .loctx/reports/1.3.0/rc2/v03-tokens-and-colors.log 2>&1
    ```
  - 기대 결과: `index.css`의 `:root`와 `.dark` 두 블록 모두에 `--primary`, `--primary-foreground`, `--ring`, `--border`, `--input` 5개 토큰 매핑(총 10줄)이 존재하고, `TooltipContent.vue`에 `bg-foreground` 및 `text-background`가 적용되어 있으며, 화면 파일(`components/ui` 제외 `.vue`)에서 수동 강조색/고정 hex 덮어쓰기, 테두리·포커스 링 덮어쓰기, 죽은 클래스(`text-croffle-text-dark`, `hover:bg-opacity-90`), `neutral-*` 클래스, 임의 hex 색상 클래스가 각각 0건(`forbidden_overrides=0`, `neutral_classes=0`, `arbitrary_hex_classes=0`; `grep` 종료 코드 1 → `! grep` 통과)입니다(창 닫기 버튼 hover의 빨강은 예외로 허용).
  - 실패 판정: `index.css` 매핑 10줄 중 하나라도 누락되었거나, `TooltipContent.vue`가 변경되지 않았거나, 금지 패턴 중 어느 하나라도 1건 이상 매칭되거나, 결과가 모호한 경우 실패로 판정합니다.
  - 증적 경로: `.loctx/reports/1.3.0/rc2/v03-tokens-and-colors.log`

- V-4 **부품 치환(`Button` · `Input`) · 버튼 크기·간격·변형 및 예외 허용 정적 검사**
  - 검증 대상: S-3, S-4, S-5, S-6, S-7, S-8, W-3, W-4
  - 실행 방법:
    ```bash
    mkdir -p .loctx/reports/1.3.0/rc2 && {
      echo "=== 1. ui/button icon-xs 정의 확인 ==="
      grep -rnE 'icon-xs' apps/desktop/src/renderer/src/components/ui/button/
      echo "=== 2. 원시 <button> 사용처 확인 (예외: app.vue 창 버튼 3개 및 색 견본만 허용) ==="
      grep -rnE '<button\b' apps/desktop/src/renderer/src --include='*.vue' --exclude-dir=ui || true
      echo "=== 3. 원시 <input> 사용처 확인 (예외: type=\"color\"만 허용) ==="
      grep -rnE '<input\b' apps/desktop/src/renderer/src --include='*.vue' --exclude-dir=ui || true
      ! grep -rnE '<input\b' apps/desktop/src/renderer/src/components/settings/sync-panel.vue --include='*.vue' && echo "sync_panel_raw_input=0"
      echo "=== 4. 컨트롤 높이·아이콘 간격·삭제 버튼 덮어쓰기 0건 확인 ==="
      ! grep -nE '\b(h-10|h-11)\b' apps/desktop/src/renderer/src/components/schedule-modal.vue apps/desktop/src/renderer/src/components/time-field.vue apps/desktop/src/renderer/src/components/settings/config-schema-form.vue && echo "height_overrides_h10_h11=0"
      ! grep -nE '(^|[" ])h-11\b' apps/desktop/src/renderer/src/components/right-sidebar.vue && echo "sidebar_cta_h11=0"
      ! grep -rnE 'size="icon"[^>]*\bh-7\s+w-7\b|\bh-7\s+w-7\b[^>]*size="icon"' apps/desktop/src/renderer/src --include='*.vue' --exclude-dir=ui && echo "icon_h7_w7_override=0"
      ! grep -rnE '\bmr-1\.5\b' apps/desktop/src/renderer/src/components/schedule-modal.vue && echo "button_icon_mr_override=0"
      ! grep -rnE 'hover:text-red-500' apps/desktop/src/renderer/src/components/settings-modal.vue && echo "legacy_delete_btn_color=0"
    } > .loctx/reports/1.3.0/rc2/v04-component-usage.log 2>&1
    ```
  - 기대 결과:
    1. `components/ui/button/`에 `icon-xs`(28px) size가 정의되어 있습니다.
    2. 화면 파일의 `<button` 매칭은 `app.vue`의 창 제어 버튼 3개(최소화·최대화·닫기)와 색 견본(color swatch) 버튼뿐이며, `calendar.vue`, `help-modal.vue`, `schedule-modal.vue`(색 견본 제외), `settings-modal.vue`(색 견본 제외)의 일반·아이콘·칩 버튼에는 `<button`이 0건입니다.
    3. 화면 파일의 `<input` 매칭은 `type="color"` 예외뿐이며, `sync-panel.vue` 5곳과 `settings-modal.vue:1073` 확장 설치 URL 입력을 포함한 텍스트/비밀번호/숫자 `<input>`은 0건(`sync_panel_raw_input=0`)입니다.
    4. `schedule-modal.vue` · `time-field.vue` · `config-schema-form.vue`에서 `h-10` · `h-11`이 0건(`height_overrides_h10_h11=0`), `right-sidebar.vue` 일정 추가 버튼의 `h-11`이 0건(`sidebar_cta_h11=0`; 접힌 상태의 `group-data-[collapsible=icon]:h-10`은 대상 아님), `size="icon"` 위 `h-7 w-7` 덮어쓰기가 0건, `schedule-modal.vue` 버튼 내 아이콘 `mr-1.5`가 0건, `settings-modal.vue` 줄 안 삭제 버튼의 `hover:text-red-500` 덮어쓰기가 0건(`ghost` + `text-destructive`로 통일)입니다.
  - 실패 판정: 허용된 예외(창 버튼 3개, 색 견본 버튼, `<input type="color">`) 외에 원시 `<button>` 또는 `<input>`이 1건이라도 남아 있거나, `icon-xs`가 없거나, 높이·간격·삭제 버튼 덮어쓰기가 1건이라도 검출되거나, 판정이 모호한 경우 실패로 판정합니다.
  - 증적 경로: `.loctx/reports/1.3.0/rc2/v04-component-usage.log`

- V-5 **입력·선택 너비(`w-full` · `w-60` · `w-24`) 및 설정 행 레이아웃 정적 검사**
  - 검증 대상: S-9, W-4
  - 실행 방법:
    ```bash
    mkdir -p .loctx/reports/1.3.0/rc2 && {
      echo "=== 1. 숫자 입력 w-28 제거 및 w-24 통일 확인 ==="
      ! grep -rnE '\bw-28\b' apps/desktop/src/renderer/src/components/schedule-modal.vue && echo "schedule_modal_w28=0"
      grep -cE '\bw-24\b' apps/desktop/src/renderer/src/components/schedule-modal.vue
      echo "=== 2. settings-modal.vue 및 config-schema-form.vue의 w-60 설정 행 확인 ==="
      grep -nE '\bw-60\b' apps/desktop/src/renderer/src/components/settings-modal.vue apps/desktop/src/renderer/src/components/settings/config-schema-form.vue
      ! grep -nE 'SelectTrigger[^>]*\bw-full\b|\bw-full\b[^>]*SelectTrigger' \
        apps/desktop/src/renderer/src/components/settings-modal.vue \
        apps/desktop/src/renderer/src/components/settings/config-schema-form.vue && echo "settings_select_w_full=0"
      echo "=== 3. 세로 스택 폼(sync-panel.vue 및 확장 설치 URL) w-full 유지 확인 ==="
      grep -nE '<Input\b' apps/desktop/src/renderer/src/components/settings/sync-panel.vue apps/desktop/src/renderer/src/components/settings-modal.vue
    } > .loctx/reports/1.3.0/rc2/v05-input-widths.log 2>&1
    ```
  - 기대 결과: `schedule-modal.vue`에서 숫자 입력의 `w-28`이 0건(`schedule_modal_w28=0`)이고 3곳 모두 `w-24`를 사용합니다. `settings-modal.vue`(기존 `w-full` 7곳 + `w-40` 1곳 = 총 8개 `Select`)와 `config-schema-form.vue`의 설정 행 컨트롤이 오른쪽 고정 폭 `w-60`을 사용하며(`settings_select_w_full=0`), 세로로 쌓인 `sync-panel.vue` 로그인 폼과 `settings-modal.vue` 확장 설치 URL 입력은 `w-full`을 유지합니다.
  - 실패 판정: 숫자 입력에 `w-28`이 남아 있거나, `settings-modal.vue` 또는 `config-schema-form.vue`의 설정 행 선택/입력에 `w-full` 또는 `w-40`이 남아 있거나, `w-60` 적용 개수가 부족하거나, 결과가 모호한 경우 실패로 판정합니다.
  - 증적 경로: `.loctx/reports/1.3.0/rc2/v05-input-widths.log`

- V-6 **다이얼로그 · 카드 · 배지 · Radius · 타이포그래피 · 아이콘 크기 정적 검사**
  - 검증 대상: S-7, S-11, S-12, S-13, S-14, W-5, W-6
  - 실행 방법:
    ```bash
    mkdir -p .loctx/reports/1.3.0/rc2 && {
      echo "=== 1. DialogContent rounded-xl 및 확인형 다이얼로그 sm:max-w-md 확인 ==="
      grep -nE 'rounded-xl' apps/desktop/src/renderer/src/components/ui/dialog/DialogContent.vue
      ! grep -nE 'rounded-lg' apps/desktop/src/renderer/src/components/ui/dialog/DialogContent.vue && echo "dialog_rounded_lg=0"
      grep -nE 'sm:max-w-md' \
        apps/desktop/src/renderer/src/components/confirm-modal.vue \
        apps/desktop/src/renderer/src/components/choice-modal.vue \
        apps/desktop/src/renderer/src/components/close-behavior-modal.vue \
        apps/desktop/src/renderer/src/components/help-modal.vue \
        apps/desktop/src/renderer/src/components/update-modal.vue
      echo "=== 2. 다이얼로그 푸터 규격 (border-t bg-muted/20 px-6 py-4, gap-2, min-w-20) 및 비표준 제거 확인 ==="
      ! grep -rnE 'bg-muted/10|shadow-2xl' apps/desktop/src/renderer/src --include='*.vue' --exclude-dir=ui && echo "nonstandard_footer_shadow=0"
      grep -nE 'min-w-20' \
        apps/desktop/src/renderer/src/components/schedule-modal.vue \
        apps/desktop/src/renderer/src/components/settings-modal.vue \
        apps/desktop/src/renderer/src/components/confirm-modal.vue \
        apps/desktop/src/renderer/src/components/choice-modal.vue \
        apps/desktop/src/renderer/src/components/close-behavior-modal.vue \
        apps/desktop/src/renderer/src/components/help-modal.vue \
        apps/desktop/src/renderer/src/components/update-modal.vue
      echo "=== 3. 배지 중복 rounded-md 제거 및 달력 하드코딩(10px, rem) 제거 확인 ==="
      ! grep -rnE '<Badge[^>]*\brounded-md\b' apps/desktop/src/renderer/src --include='*.vue' --exclude-dir=ui && echo "badge_redundant_rounded_md=0"
      ! grep -nE 'border-radius:\s*10px|0\.8rem|0\.75rem|0\.65rem' apps/desktop/src/renderer/src/components/calendar.vue && echo "calendar_hardcoded_units=0"
      echo "=== 4. 타이포그래피(라벨 font-medium, 도움말 제목, 소제목) 및 아이콘 h-4 w-4 제거 확인 ==="
      ! grep -nE 'text-xl\s+font-bold' apps/desktop/src/renderer/src/components/help-modal.vue && echo "help_title_override=0"
      ! grep -rnE '<(Label|label)[^>]*font-semibold' apps/desktop/src/renderer/src --include='*.vue' --exclude-dir=ui && echo "label_font_semibold=0"
      grep -nE 'text-foreground' apps/desktop/src/renderer/src/components/settings/config-schema-form.vue
      ! grep -rnE '\bh-4\s+w-4\b|\bw-4\s+h-4\b' apps/desktop/src/renderer/src --include='*.vue' --exclude-dir=ui && echo "icon_h4_w4=0"
    } > .loctx/reports/1.3.0/rc2/v06-dialog-card-typo.log 2>&1
    ```
  - 기대 결과:
    1. `DialogContent.vue`가 `rounded-xl`을 쓰고(`dialog_rounded_lg=0`), 확인형 모달 5종(`confirm`, `choice`, `close-behavior`, `help`, `update`) 모두 `sm:max-w-md`가 존재합니다.
    2. 화면 파일에서 `bg-muted/10` 및 `shadow-2xl`이 0건(`nonstandard_footer_shadow=0`)이고, 모든 다이얼로그 푸터 버튼에 `min-w-20`이 적용되어 있습니다.
    3. `<Badge`의 중복 `rounded-md` 덮어쓰기가 0건(`badge_redundant_rounded_md=0`)이고, `calendar.vue`의 `border-radius: 10px` 및 `0.8rem` · `0.75rem` · `0.65rem` 하드코딩이 0건(`calendar_hardcoded_units=0`)입니다.
    4. `help-modal.vue` 제목의 `text-xl font-bold`가 0건(`help_title_override=0`), 라벨의 `font-semibold`가 0건(`label_font_semibold=0`), `config-schema-form.vue` 소제목이 `text-foreground`를 사용하며, 화면 파일 전체에서 `h-4 w-4` 표기가 0건(`icon_h4_w4=0`)입니다.
  - 실패 판정: 위 항목 중 하나라도 누락되거나 0건이어야 할 패턴이 1건 이상 검출되거나 출력이 모호한 경우 실패로 판정합니다.
  - 증적 경로: `.loctx/reports/1.3.0/rc2/v06-dialog-card-typo.log`

- V-7 **런타임 실측 검사 — 컨트롤 높이 · 아이콘 버튼 크기 · 입력 너비 · 푸터 버튼 규격 (Playwright + Electron)**
  - 검증 대상: S-3, S-4, S-6, S-7, S-8, S-9, W-3, W-4
  - 실행 방법:
    `pnpm build` 완료 후, `.loctx/reports/1.3.0/rc2/`에 `npm init -y && npm i playwright-core`로 Playwright를 저장소 밖 의존성으로 설치하고(저장소의 `package.json` · lockfile은 건드리지 않음, Electron은 앱의 것을 씀), `.loctx/reports/1.3.0/rc2/verify-runtime.mjs` 일회용 스크립트(커밋하지 않음)에서 Playwright `_electron.launch({ args: ['apps/desktop/out/main/index.js'] })`로 앱을 구동하여 라이트(`light`) 및 다크(`dark`) 두 테마 각각에서 다음 7개 화면을 순서대로 열고 DOM 실측치를 JSON/로그로 기록합니다: (1) 달력 메인 화면, (2) 일정 모달 — 보기 모드, (3) 일정 모달 — 편집 모드(반복·알림·태그·중요도 펼침), (4) 설정 모달 — 일반·달력·알림·동기화·확장(설치된 확장 설정 `config-schema-form` 포함) 모든 탭, (5) 확인 모달(`confirm-modal`), (6) 도움말 모달(`help-modal`), (7) 업데이트 모달(`update-modal`).
    ```bash
    xvfb-run -a node .loctx/reports/1.3.0/rc2/verify-runtime.mjs --check=dimensions \
      > .loctx/reports/1.3.0/rc2/v07-runtime-dimensions.log 2>&1
    ```
    스크립트는 각 화면에서 보이는(visible) 요소에 대해 `getBoundingClientRect()`와 `getComputedStyle()`로 다음을 단언(assert)합니다(색 견본 버튼, `app.vue` 창 버튼 3개, `<input type="color">`는 제외):
    1. 일반 `Button`(`default`/`outline`/`destructive`/`ghost`), `Input`(`time-field` 포함), `SelectTrigger`의 `height === 36` (±0.5px); 요일 선택 토글 및 `size="sm"` 버튼의 `height === 32` (±0.5px). 중요도 선택(아이콘과 글자를 세로로 쌓은 3칸 선택기)과 태그 추천(배지 모양)은 측정 대상에서 제외.
    2. 아이콘 버튼의 `width × height`가 제목 표시줄(`app.vue`, 창 제어 버튼 3개 제외) `28 × 28`px, 본문/모달 내 아이콘 버튼 `32 × 32`px, 좌측 사이드바 레일(`left-sidebar.vue`) `36 × 36`px이며, 버튼 안 `<svg>`의 `width === 16`, `height === 16`이고 텍스트와 함께 있는 버튼의 `columnGap === '8px'`.
    3. 모든 다이얼로그 푸터 버튼의 `width >= 80`px(`min-w-20`), 푸터 버튼 컨테이너의 `columnGap === '8px'`(`gap-2`), 주 버튼(`bg-primary`)이 푸터 내 버튼 중 가장 오른쪽(`rect.right` 최대)에 위치하며, 설정 모달 취소 버튼과 저장 버튼의 `paddingLeft === '16px'`, `paddingRight === '16px'`, `fontWeight === '500'`이 서로 정확히 일치함.
    4. 설정 모달의 행 오른쪽 `SelectTrigger` 8개 및 `config-schema-form` 행 컨트롤의 `width === 240` (±1px, `w-60`), 일정 모달 숫자 입력 3개의 `width === 96` (±1px, `w-24`), 동기화 로그인 폼 및 확장 설치 URL 입력의 폭이 부모 컨테이너 내부 폭(`w-full`, ±1px)과 일치함.
  - 기대 결과: 라이트·다크 양쪽 테마와 7개 대상 화면 전체에서 측정된 모든 대상 컨트롤이 위 수치 기준을 100% 만족하여 `violations: 0` 및 종료 코드 `0`을 출력합니다.
  - 실패 판정: 어느 한 화면·테마에서라도 높이·너비·간격·패딩·굵기 실측치가 허용 오차를 벗어나거나(`violations > 0`), 7개 화면 중 하나라도 열리지 않아 측정 대상 수가 0이거나, 스크립트가 비정상 종료되는 경우 실패로 판정합니다.
  - 증적 경로: `.loctx/reports/1.3.0/rc2/v07-runtime-dimensions.log`

- V-8 **런타임 실측 검사 — 표면·컨트롤 Radius · 다이얼로그 폭·푸터 · 타이포그래피 · 다크 모드 글자색 (Playwright + Electron)**
  - 검증 대상: S-11, S-12, S-13, S-14, W-5, W-6
  - 실행 방법:
    V-7과 동일한 7개 화면(달력, 일정 모달 보기·편집, 설정 모달 전 탭 및 확장 설정, 확인 모달, 도움말 모달, 업데이트 모달)을 라이트 및 다크 모드에서 열고 `getComputedStyle()`과 `getBoundingClientRect()`로 실측합니다:
    ```bash
    xvfb-run -a node .loctx/reports/1.3.0/rc2/verify-runtime.mjs --check=radius-typo \
      > .loctx/reports/1.3.0/rc2/v08-runtime-radius-typo.log 2>&1
    ```
    스크립트는 다음 항목을 단언합니다:
    1. 컨트롤(`Button`, `Input`, `SelectTrigger`, 달력 툴바 버튼, 설정 탭 항목)의 `borderRadius === '8px'`(원형 `rounded-full` 제외), 달력 이벤트(`.fc-event`)의 `borderRadius === '10px'`, 열린 모든 다이얼로그(`DialogContent`)와 카드(`Card`, 설정/일정 내 토큰 카드)의 `borderRadius === '14px'`, 예외인 달력 외곽 카드(`calendar.vue`)만 `borderRadius === '24px'`.
    2. 확인형 다이얼로그(확인·도움말·업데이트 모달) `DialogContent`의 `getComputedStyle(el).maxWidth === '448px'`(`sm:max-w-md`), 폼형 다이얼로그의 구분된 푸터(`schedule-modal`, `settings-modal`)의 `borderTopWidth === '1px'`, `paddingTop === '16px'`, `paddingBottom === '16px'`, `paddingLeft === '24px'`, `paddingRight === '24px'`.
    3. 모든 다이얼로그 제목(`DialogTitle`)의 `fontSize === '18px'` 및 `fontWeight === '600'`, 섹션 소제목의 `fontSize === '16px'` 및 `fontWeight === '700'`, 폼/설정 라벨의 `fontSize === '14px'` 및 `fontWeight === '500'`.
    4. 다크 모드(`.dark`)에서 확장 설정(`config-schema-form.vue`) 소제목의 `getComputedStyle(el).color`가 루트 `--foreground` 계산 색상과 일치하며 어두운 색(`rgb(23, 23, 23)` 등 `neutral-900`)이 아님.
  - 기대 결과: 라이트·다크 전 화면에서 모든 측정 항목이 일치하여 `violations: 0` 및 종료 코드 `0`을 기록합니다.
  - 실패 판정: Radius, 다이얼로그 폭/푸터 패딩, 폰트 크기/굵기, 다크 모드 소제목 글자색 중 1건이라도 기준치와 다르거나(`violations > 0`), 어느 한 모달이라도 측정에서 누락된 경우 실패로 판정합니다.
  - 증적 경로: `.loctx/reports/1.3.0/rc2/v08-runtime-radius-typo.log`

- V-9 **런타임 실측 검사 — 강조색(`--croffle-accent-hue`) 변경 시 주 버튼 · 스위치 · 포커스 링 동시 변경 (Playwright + Electron)**
  - 검증 대상: S-2, S-5, S-10, W-2
  - 실행 방법:
    Playwright Electron 스크립트로 라이트 및 다크 모드에서 각각 기본 강조색 상태(hue A, 기본값 `69.8`)와 변경된 강조색 상태(hue B, 예: 설정 창의 다른 강조색 견본 클릭 또는 `--croffle-accent-hue: 210` 적용)에서 달력(우측 사이드바 일정 추가 버튼), 일정 편집 모달(저장 버튼, 종일/반복 `Switch`, 입력 포커스 링), 설정 모달(저장 버튼, 동기화 패널 주 버튼, 스위치), 확인 모달(확인 버튼), 도움말 모달(닫기 버튼), 툴팁(`TooltipContent`)의 계산 스타일을 측정합니다:
    ```bash
    xvfb-run -a node .loctx/reports/1.3.0/rc2/verify-runtime.mjs --check=accent-hue \
      > .loctx/reports/1.3.0/rc2/v09-accent-hue-sync.log 2>&1
    ```
  - 기대 결과:
    1. hue A 상태에서 모든 화면의 주 버튼(`Button` default variant), 체크된 `Switch`(`data-state="checked"`)의 `getComputedStyle(el).backgroundColor`가 서로 100% 동일한 값(`color_A`)이고, 포커스된 컨트롤의 `--ring` 계산 값이 `--croffle-primary`와 일치합니다(고정 갈색 `rgb(166, 138, 100)`이나 검정 `rgb(10, 10, 10)` 등으로 따로 노는 요소 0개).
    2. hue B로 변경하면 주 버튼, 체크된 `Switch`의 `backgroundColor`가 모두 새로운 `color_B`(`color_B !== color_A`)로 함께 바뀌고, 포커스 링 색상(`--ring` 및 `boxShadow`/`outlineColor`)도 함께 hue B 계열로 바뀝니다.
    3. 반면 툴팁(`TooltipContent`)의 `backgroundColor`는 hue A와 hue B에서 변하지 않고 전경색(`--foreground`)을 유지합니다.
  - 실패 판정: hue 변경 시 주 버튼·스위치·포커스 링 중 하나라도 색상이 바뀌지 않거나, 화면 간 주 버튼 배경색이 서로 다르거나, 툴팁 배경이 강조색으로 바뀌거나, 출력이 모호한 경우 실패로 판정합니다.
  - 증적 경로: `.loctx/reports/1.3.0/rc2/v09-accent-hue-sync.log`

- V-10 **라이트 / 다크 주요 화면 전(`befde1f`) · 후(`HEAD`) 스크린샷 채증 및 무결성 검증**
  - 검증 대상: S-15, S-16, W-2, W-3, W-4, W-5, W-6
  - 실행 방법:
    동일한 Playwright 캡처 스크립트(`.loctx/reports/1.3.0/rc2/capture-screens.mjs`, 고정 뷰포트 `1280×800`)를 사용하여, 기준 커밋 `befde1f`(`git worktree` 또는 W-2 작업 전 체크아웃)에서 `pnpm build` 후 `.loctx/reports/1.3.0/rc2/v10-screenshots/before/`에 캡처하고, `HEAD`(W-6 완료 후)에서 `pnpm build` 후 `.loctx/reports/1.3.0/rc2/v10-screenshots/after/`에 캡처합니다:
    ```bash
    mkdir -p .loctx/reports/1.3.0/rc2/v10-screenshots/before .loctx/reports/1.3.0/rc2/v10-screenshots/after && {
      xvfb-run -a node .loctx/reports/1.3.0/rc2/capture-screens.mjs --out=.loctx/reports/1.3.0/rc2/v10-screenshots/after
      echo "=== before / after 파일 대조 ==="
      diff -u \
        <(cd .loctx/reports/1.3.0/rc2/v10-screenshots/before && ls -1 *.png) \
        <(cd .loctx/reports/1.3.0/rc2/v10-screenshots/after && ls -1 *.png)
      find .loctx/reports/1.3.0/rc2/v10-screenshots -name '*.png' -size 0
      echo "before_count=$(ls -1 .loctx/reports/1.3.0/rc2/v10-screenshots/before/*.png | wc -l)"
      echo "after_count=$(ls -1 .loctx/reports/1.3.0/rc2/v10-screenshots/after/*.png | wc -l)"
    } > .loctx/reports/1.3.0/rc2/v10-screenshots.log 2>&1
    ```
    캡처 대상 화면(라이트 `light` / 다크 `dark` 각 1장씩):
    - 달력 메인 화면 (`01-calendar-{light,dark}.png`)
    - 일정 모달 — 보기 (`02-schedule-view-{light,dark}.png`) 및 편집 (`03-schedule-edit-{light,dark}.png`)
    - 설정 모달 — 모든 탭: 일반, 달력/표시, 알림, 동기화, 확장 목록 및 설치된 확장 설정 `config-schema-form` (`04-settings-<tab>-{light,dark}.png`)
    - 확인 모달 (`05-confirm-modal-{light,dark}.png`)
    - 도움말 모달 (`06-help-modal-{light,dark}.png`)
    - 업데이트 모달 (`07-update-modal-{light,dark}.png`) — 업데이트가 있어야 뜨는 화면이라, 스크립트가 렌더러의 update 스토어 상태를 넣어 엽니다. 그렇게 열 수 없으면 이 화면은 미확인으로 보고합니다(통과로 치지 않음).
  - 기대 결과: `before/`와 `after/` 디렉터리에 라이트·다크의 모든 필수 화면 PNG 파일이 동일한 파일명 구성으로 생성되고(`diff -u` 종료 코드 `0`, 각 디렉터리 최소 20장 이상, 0바이트 파일 0개), 캡처 중 렌더러 `pageerror` 및 `console.error`가 0건이며, 레이아웃 구조·문구 깨짐이나 다크 모드 글자 묻힘(`config-schema-form.vue` 소제목 포함)이 없습니다.
  - 실패 판정: `before/` 또는 `after/` 중 어느 한쪽이라도 필수 화면(달력, 일정 보기·편집, 설정 전 탭 및 확장 설정, 확인·도움말·업데이트 모달 × 라이트·다크) 스크린샷이 1장이라도 누락되었거나, 0바이트 이미지가 있거나, 런타임 콘솔 에러가 발생했거나, 증적 경로에 파일이 없는 경우 실패로 판정합니다.
  - 증적 경로: `.loctx/reports/1.3.0/rc2/v10-screenshots/` (및 `.loctx/reports/1.3.0/rc2/v10-screenshots.log`)

## 우려
- **강조색 위 흰 글자 대비가 낮습니다(약 2:1).** 지금도 같은 상태이고 이번에 스위치 · 체크박스 · 확인 모달 버튼으로 넓어집니다. 색 체계 문제라 고치지 않고 로컬 로드맵에 적습니다.
- **확장 화면.** 확장이 호스트의 `--primary` 를 직접 읽고 있었다면 검정에서 강조색으로 바뀝니다. 공개 타입 · 문서는 `--croffle-*` 만 안내하므로 약속을 깨는 것은 아닙니다. 설치된 확장 설정 화면(`config-schema-form.vue`)은 스크린샷으로 확인합니다.
- **같은 파일을 여러 단위가 고칩니다.** `settings-modal.vue` · `schedule-modal.vue` 를 W-2 ~ W-6 이 모두 건드립니다. 줄 단위로 나뉘지만 뒤 단위를 남기고 앞 단위만 되돌리면 충돌할 수 있습니다.
- **일정 모달이 조금 낮아집니다.** 입력 13개가 40 → 36px 이라 모달 높이가 50px 가량 줄어듭니다(질의서 1 에서 승인).

## 범위 밖
- 색 팔레트 · 폰트 · 레이아웃 재설계, 대비 개선.
- 일정 모달의 스위치 상자와 설정의 스위치 행을 같은 틀로 맞추는 일(B-7).
- UI 규칙을 검사하는 lint 규칙 · 스크립트 추가.
- V-11 재현(전송 중 재편집) — rc.2 의 다른 작업 단위.
