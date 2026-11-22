---
intent: "007"
status: accepted
approvals:
  - { role: user, by: bluenyang, at: "2026-11-22", words: "진행", rev: "5867f47", kind: summary }
  - { role: user, by: bluenyang, at: "2026-11-22", words: "승인한다", rev: "d20504f", kind: change, note: "check wording: V-1 ⑦, V-5 b · c (node --test takes a glob, A-1), V-7 b (any quote in the Changeset front matter); judged on the same raw output (test/judgement.md)" }
summary_for_user: |
  1.3.3 을 7개 작업 단위로 나눕니다. W-1 hook 판별기를 Node 로(파이썬과 60개 사례 출력 대조, 다른 저장소 커밋은 HEAD 가 120초보다 오래면 게이트 생략) → W-2 두 검사 스크립트 테스트 · 루트 `test:repo` · CI `repo-scripts` 작업 → W-3 로그아웃해도 동기화 스위치 유지(서버 주소를 바꾸면 끔) → W-4 할 일 데이터(공개 타입 `kind?` · `completedAt?` minor, 마이그레이션 0009, 저장할 때 시작 = 마감, 할 일 + 반복 · 일정 + 완료 시각 거절, 동기화 와이어 · 픽스처, `kind` 없는 옛 기기 행은 일정으로) → W-5 리마인더(완료한 할 일 제외, "마감" 문구) → W-6 화면(모달의 일정/할 일 전환 · 마감 칸 · 반복 안내, 캘린더 체크 상자 · 취소선, 오른쪽 사이드바 "할 일" 칸, 다가오는 일정 · 오늘 개수에서 할 일 제외) → W-7 문서.
  검사는 V-1~V-13 입니다. qa-designer(gemini)가 설계했고, 지금 트리에서 돌릴 수 있는 명령은 모두 미리 돌려 기대 문자열을 실제 출력과 맞췄습니다(`.loctx/reports/1.3.3/dryrun/`). 사전 실행 뒤 PM 이 고친 것: V-3 사례 60개(읽을 수 없는 입력 3개는 앞부분만 대조), V-6 · V-8 은 Electron · DB 없는 모듈로 빼서 시험(지금 코드는 테스트에서 직접 불러올 수 없음), V-13 찾는 문구.
  우려: 1.3.2 이하 기기가 할 일을 고치면 일반 일정이 됩니다(릴리스 노트로 알림). hook 교체 중 판별기 오류는 모든 Bash 를 막으므로 새 판별기를 시험한 뒤 호출 줄을 마지막에 바꿉니다.
---
## 요구 해석
기준: intent 007(승인 `6eb0eff`의 직전 본문 `4df2fca`), 계획 원본 `.loctx/context/plan/1.3.2_todo.md`(2026-10-20 사용자 승인 모델), 인계 `.loctx/history/1.3.3.md`(hook · 테스트 기대 결과 표).

### hook 판별기 · 테스트
- S-1 (계획 1 "hook 판별기를 Node 로", 바라는 결과 "판별 결과는 … 옮기기 전후로 같습니다") — `.claude/hooks/bash_command.py` 를 `.claude/hooks/bash-command.mjs` 로 옮깁니다. 판정 함수 `classify(command)` 를 export 하고, 파일을 직접 실행하면 stdin 의 hook JSON 을 읽어 한 줄(`none` / `commit` / `merge <pr> [<sha>]` / `push <remote> <branch>` / `refuse <이유>`)을 씁니다. 정규식 · 문구 · 판정 순서는 파이썬과 같습니다. 인계 문서의 기대 결과 표 전체가 옮기기 전(파이썬)과 뒤(Node)에서 같은 출력을 냅니다.
- S-2 (같은 계획, 인계 "같이 고칠 것") — `pre-bash.sh` · `post-bash.sh` 는 `python3 …/bash_command.py` 대신 `node …/bash-command.mjs` 를 부르고, `command -v python3` 검사를 `command -v node` 로 바꿉니다. `pre-bash.sh` 머리 주석의 shlex 문장("splits the command the way the shell does")을 지금 방식(정확한 꼴만 허용, 그 밖의 push · merge 단어 거부, 변수 조립 · 스크립트 파일은 못 봄)으로 고칩니다. `bash_command.py` 는 지웁니다. `AGENTS.md` · knowledge 에 python 판별기를 가리키는 문장이 있으면 같이 고칩니다.
- S-3 (intent "에이전트가 정한 것 — 다른 저장소에서 한 커밋") — `post-bash.sh` 는 판정이 `commit` 이어도 이 저장소의 `HEAD` 커밋 시각(`%ct`)이 지금보다 120초 넘게 오래됐으면 게이트를 돌리지 않고 exit 0 입니다. 다른 저장소에서 커밋하면 이 저장소의 `HEAD` 는 새로 생기지 않으므로 넘어갑니다. 한계(이 저장소에서도 120초 안에 커밋한 직후 다른 저장소에서 커밋하면 게이트가 돎)는 주석에 적습니다.
- S-4 (바라는 결과 "판별기와 두 검사 스크립트에 `node --test` 테스트가 생겨 CI 에서 돕니다", 에이전트가 정한 것 "테스트 위치와 실행") — 테스트는 `scripts/test/*.test.mjs` 입니다.
  - `bash-command.test.mjs`: 기대 결과 표의 모든 행을 `classify` 로, JSON 이 아닌 입력은 CLI 로 시험합니다.
  - `post-bash.test.mjs`: 임시 저장소에서 `post-bash.sh` 를 돌려 S-3 을 시험합니다(오래된 `HEAD` → exit 0, 방금 커밋한 `HEAD` → 게이트 단계로 감).
  - `check-spec-checks.test.mjs`, `check-design-commit-msg.test.mjs`: 인계 문서의 두 표 전체. 임시 저장소(`mkdtemp` + `git init`, 사용자 이름 · 메일은 저장소 로컬 설정)에서 스크립트를 실행합니다.
  - 루트 `package.json` 에 `"test:repo": "node --test 'scripts/test/*.test.mjs'"`.
  - CI(`.github/workflows/ci.yml`): 경로 필터에 `repo`(`scripts/**`, `.claude/hooks/**`, 그리고 root 필터가 잡는 파일)를 더하고, Node 만 설치해 `node --test 'scripts/test/*.test.mjs'` 를 돌리는 작업(A-1) `repo-scripts` 를 둡니다(pnpm 설치 없음). `CI result` 의 `needs` 에 넣습니다.

### 동기화 스위치
- S-5 (바라는 결과 "로그아웃했다가 다시 로그인하면 동기화 스위치가 로그아웃 전 상태로") — 로그아웃(`resetBinding`)은 `enabled` 를 건드리지 않습니다. 로그아웃 중에는 계정 · 토큰이 없어 동기화가 돌지 않고(`sync-service.ts:85`), 다시 로그인하면 남은 `enabled` 대로 돕니다. 서버 주소를 바꾸는 것(`setServer`)은 새 연결이므로 지금처럼 `enabled` 를 끕니다. 서버를 지우면(`removeServer`) 행이 없어집니다. 설정 화면의 스위치는 로그아웃 중에 남은 값을 보이되 지금처럼 비활성입니다.

### 할 일 — 데이터
- S-6 (문제 "모델은 2026-10-20 에 사용자가 승인한 것", 바라는 결과 "공개 타입의 `kind` · `completedAt`(선택 필드)", 제약 "선택(optional)") — 공개 `Schedule` 에 `kind?: 'event' | 'todo'`(없으면 일정) 와 `completedAt?: Date | null`(할 일을 끝낸 시각)을 더합니다. 데스크톱 사본(`common/models/schedule.ts`)도 같게 둡니다(`assert-public-types.ts:41`). `@croffledev/croffle-types` minor Changeset 하나.
- S-7 (계획 3 "새 마이그레이션(0009: ADD COLUMN + 상수 기본값만)") — `schedule` 표에 `kind text NOT NULL DEFAULT 'event'`(값 `event` · `todo`) 와 `completedAt integer`(초, NULL 허용)를 더하는 마이그레이션 0009 를 `db:generate` 로 만듭니다. 백필 UPDATE 는 없습니다. outbox 트리거는 열을 나열하지 않으므로 바뀌지 않습니다.
- S-8 (바라는 결과 "할 일은 제목 · 마감 · … 반복은 없습니다", 에이전트가 정한 것 "저장할 때 시작과 끝을 둘 다 마감으로") — 서비스 검증(`validateScheduleData`)과 저장:
  - `kind` 가 `event` · `todo` 가 아니면 거절합니다.
  - 할 일은 `startDate` 를 `endDate`(마감)로 맞춰 저장하고, 반복 규칙이 있으면 거절합니다.
  - 일정(`event`)에 `completedAt` 이 있으면 거절합니다.
  - 고치기(`updateSchedule`)는 넘어오지 않은 `kind` · `completedAt` 을 그대로 둡니다(만들기 · 고치기 · 합친 값 세 곳 모두). 할 일 → 일정으로 바꾸면 `completedAt` 을 지웁니다.
  - 매퍼(`toEntity` · `toInterface`)가 두 필드를 옮기므로 확장 API · 가져오기 · 내보내기가 함께 다룹니다. 가져온 파일에 `kind` 가 없으면 일정입니다.
- S-9 (바라는 결과 "할 일은 … 동기화에서 함께 다닙니다") — 동기화 와이어의 `schedule` 열 목록(`common/sync-protocol.ts:143`)에 `kind`(text) · `completedAt`(초 → 와이어 ms)을 더하고, 공유 픽스처 `packages/sync-fixtures/scenarios/12-row-codec.json`(데스크톱 러너만 읽음)의 schedule 사례에 두 열을 더합니다. 와이어 행에 `kind` 가 없거나 null 이면(1.3.2 이하 기기가 쓴 행) 로컬에서는 `event` 로 읽습니다. NOT NULL 열에 null 이 들어가 적용이 실패하지 않게 하려는 것입니다.

### 할 일 — 리마인더
- S-10 (바라는 결과 "리마인더는 마감 기준 … 완료한 할 일은 울리지 않습니다", 에이전트가 정한 것 "종일 할 일은 마감일 0시") — 할 일은 시작 = 마감이므로 지금의 계산(`toRemindAt`, 종일은 그날 0시)이 그대로 마감 기준입니다. `buildReminderCandidates` 는 `completedAt` 이 있는 할 일을 후보에서 뺍니다. 알림 본문은 할 일이면 "마감" 문구(`reminder.bodyTodoAllDay` · `reminder.bodyTodoTimed`, ko · en)를 씁니다. 중복 방지 키 · 되감기 창은 그대로입니다(완료를 되돌려도 이미 울린 알림은 다시 울리지 않음).

### 할 일 — 화면
- S-11 (바라는 결과 "일정 모달에서 일정 / 할 일을 고를 수 있습니다", 계획 5) — 일정 모달 편집 화면 위쪽에 일정 · 할 일 전환(segmented)을 둡니다. 할 일이면:
  - 날짜 · 시각 칸은 "마감" 하나(종일이면 날짜만)이고 시작 칸은 보이지 않습니다.
  - 반복 칸은 꺼지고 "할 일은 반복할 수 없습니다" 안내가 보입니다. 반복이 있는 일정을 할 일로 바꾸면 저장할 때 반복을 지웁니다.
  - 보기 화면은 "할 일 · 마감 <날짜/시각>"과 완료 상태, 완료 / 완료 취소 버튼을 보입니다.
- S-12 (바라는 결과 "할 일은 마감일에 캘린더에 보이고, 완료 체크로 끝냅니다. 완료한 할 일은 취소선으로 남고") — 캘린더의 할 일 이벤트는 제목 앞에 체크 상자를 그립니다(`eventContent`). 체크 상자를 누르면 보기 창을 열지 않고 `completedAt` 을 지금 시각 / null 로 바꿉니다. 완료한 할 일은 취소선 · 흐린 글자입니다. 드래그로 날짜를 옮기면 마감이 옮겨지고(반복이 없으므로 확인 창 없음, 지금 `use-schedule-drag.ts:68` 의 경로), 시작 = 마감이 유지됩니다.
- S-13 (질의서 1 "가 방식.") — 오른쪽 사이드바에 "할 일" 칸을 더합니다. 미완료 할 일 전부를 마감 순으로 보이고, 마감이 지난 것은 `text-destructive` 로 "지남"을 표시합니다. 체크 상자로 완료하면 목록에서 빠집니다. 항목을 더블클릭하면 보기 창이 열립니다(다가오는 일정과 같음). 비면 "할 일이 없습니다". 지난 마감까지 모두 읽어야 하므로 날짜 범위 없이 미완료 할 일을 주는 내부 IPC 를 둡니다(호스트 전용 `croffleInternal`, 공개 타입 아님).
- S-14 (PM 해석: 같은 할 일이 두 곳에 겹치지 않게) — "다가오는 일정"(`selectUpcoming`)과 왼쪽 사이드바의 "오늘 일정 N개"는 할 일을 세지 않습니다. 오른쪽 사이드바의 고른 날 목록은 그날 마감인 할 일을 체크 상자와 함께 보입니다.
- S-15 (문구 · 디자인) — 새 문구는 ko · en 카탈로그에 함께 넣습니다. 체크 상자 · segmented 전환 · 취소선 규칙이 `DESIGN.md` 에 없으면 같은 변경에서 더합니다(DESIGN.md 를 바꾸는 커밋은 `Spec-check:` 줄).

### 문서
- S-16 (계획 6) — README · README.ko 기능 목록에 할 일, `packages/types/README.md` 의 Schedule 표에 두 필드. 릴리스 노트(`.loctx/release/`)에는 "동기화하는 모든 기기를 1.3.3 으로 올리십시오"(에이전트가 정한 것 "다른 기기가 1.3.2 이하일 때")를 적습니다. 릴리스 노트는 release 단계에서 씁니다.

## 설계
- **hook(S-1~S-4)** — 새 판별기를 먼저 만들고 테스트로 표를 통과시킨 뒤, 호출 줄을 마지막 커밋에서 바꿉니다. 바꾸는 순간부터 이 세션의 모든 Bash 가 새 판별기를 지나므로, 호출 줄을 바꾸기 전에 `printf '{"tool_input":{"command":"echo hi"}}' | node .claude/hooks/bash-command.mjs` 가 `none` 인지 확인합니다. 파이썬 원본 출력은 표의 명령을 그대로 넣어 한 번 돌려 `.loctx/reports/1.3.3/hook-py-baseline.txt` 에 남깁니다(테스트의 기대값과 대조).
- **post hook 의 신선도(S-3)** — `now=$(date +%s)`, `ct=$(git log -1 --format=%ct)`; `[ $((now - ct)) -gt 120 ] && exit 0`. 판정이 `commit` 일 때만 봅니다.
- **동기화 스위치(S-5)** — `UNBOUND_STATE` 를 Electron 없는 `src/main/sync/binding-state.ts` 로 옮겨 `SIGNED_OUT_STATE`(`enabled` 없음)와 `NEW_SERVER_STATE`(= 그것 + `enabled: false`)로 나누고, `resetBinding` 은 앞의 것, `setServer` 는 뒤의 것을 씁니다(V-6 이 테스트로 봄).
- **할 일 저장(S-6~S-9)** — DB `kind` 는 drizzle `text('kind', { enum: ['event', 'todo'] }).notNull().default('event')`, `completedAt` 은 `integer('completedAt', { mode: 'timestamp' })`. `ScheduleEntity` · `ScheduleEntityInput` · `toInterface`(`kind` 는 늘 채움) · `toEntity`. 와이어는 `col('kind', 'text')`, `col('completedAt', 'sec')`; `decodeRow` 뒤 `kind == null` 이면 `'event'`.
- **할 일 목록(S-13)** — main `schedule-service` 에 `getOpenTodos()`(`kind = 'todo' AND completedAt IS NULL AND deletedAt IS NULL`, `endDate` 오름차순) → `schedule:getOpenTodos` IPC → `croffleInternal.todos.listOpen()` → 렌더러 `todo-store`(일정이 바뀌는 이벤트에 다시 읽음).
- **캘린더(S-12)** — `toSeriesEvent` 가 `extendedProps.kind` · `completed` 를 싣고, `calendar.vue` 의 `eventContent` 가 할 일이면 체크 상자 + 제목을 그립니다. 체크 상자의 클릭은 `stopPropagation` 해 `eventClick` 이 열리지 않게 합니다.
- **테스트(데스크톱)** — `test/todo.test.ts`(할 일 규칙을 DB 없는 `src/main/calendar/schedule-kind.ts` 로 빼서 서비스가 쓰게 하고: 잘못된 `kind` 거절, 할 일 + 반복 거절, 할 일의 시작 = 마감, 일정 + `completedAt` 거절, 고치기에서 빠진 필드 유지, 매퍼 왕복 · `kind` 없는 가져오기 = 일정, 0009 의 열 · 기본값. 미완료 할 일 조회는 V-11 화면 검사로 봄), `test/reminder.test.ts`(완료한 할 일 제외, 미완료 할 일의 알림 시각 = 마감 − 분, 종일 할 일 = 0시 − 분, 일정은 그대로), `test/sync-protocol.test.ts` 는 픽스처로 두 열 + `kind` 없는 와이어 행 사례, `test/upcoming.test.ts` 에 할 일 제외 사례, `test/outbox-triggers.test.ts` 는 0009 적용 뒤 트리거 유지를 그대로 봄.

## 작업 단위
- [x] W-1 hook 판별기 Node 이전 — `bash-command.mjs` · `scripts/test/bash-command.test.mjs` · 파이썬 기준 출력, 그다음 `pre-bash.sh` · `post-bash.sh` 호출 줄 · 주석, `bash_command.py` 삭제, S-3 신선도와 `post-bash.test.mjs`. (S-1 · S-2 · S-3)
- [x] W-2 검사 스크립트 테스트와 실행 경로 — `check-spec-checks.test.mjs` · `check-design-commit-msg.test.mjs`, 루트 `test:repo`, CI `repo` 필터 · `repo-scripts` 작업. (S-4)
- [x] W-3 동기화 스위치 유지 — `sync-config.ts` 와 그 테스트. (S-5)
- [x] W-4 할 일 데이터 — 공개 타입 · 데스크톱 사본 · Changeset, 마이그레이션 0009, 매퍼 · 서비스 검증, 와이어 · 픽스처, `getOpenTodos` + 내부 IPC · preload, 테스트. (S-6 ~ S-9, S-13 의 IPC)
- [x] W-5 할 일 리마인더 — 후보 제외 · 본문 문구 · 테스트. (S-10)
- [x] W-6 할 일 화면 — 일정 모달 전환 · 마감 칸 · 반복 안내 · 보기 화면 완료 버튼, 캘린더 체크 상자 · 취소선 · 드래그 확인, 오른쪽 사이드바 "할 일" 칸, 다가오는 일정 · 오늘 개수 제외, ko · en, DESIGN.md. (S-11 ~ S-15)
- [x] W-7 문서 — README · README.ko, `packages/types/README.md`. (S-16, 릴리스 노트는 release 단계)

## 검증 기준
- **V-1 게이트 (S-1 ~ S-16, W-1 ~ W-7)** — 저장소 루트에서 아래 일곱 명령을 각각 따로 실행해 `.loctx/reports/1.3.3/test/` 아래 지정한 원본 파일(첫 줄 `$ 명령`, 본문 stdout · stderr 전체, 끝 줄 `exit=N`)에 남깁니다: ① `pnpm typecheck` → `v1-typecheck.txt`, ② `pnpm lint` → `v1-lint.txt`, ③ `pnpm format:check` → `v1-format.txt`, ④ `pnpm -r test` → `v1-test.txt`, ⑤ `pnpm build` → `v1-build.txt`, ⑥ `pnpm --filter @croffledev/croffle-cli build` → `v1-build-cli.txt`, ⑦ `node --test 'scripts/test/*.test.mjs'` → `v1-test-repo.txt` (2026-11-22 정정, A-1 · 사용자 "승인한다").
  - 기대:
    1. 일곱 개 원본 파일 모두 끝 줄이 `exit=0` 입니다.
    2. `v1-typecheck.txt` 에 `packages/types typecheck: Done`, `packages/cli typecheck: Done`, `apps/desktop typecheck: Done` 줄이 모두 있습니다.
    3. `v1-format.txt` 에 `All matched files use the correct format.` 줄이 있습니다.
    4. `v1-test.txt` 에 `$ node --test --disable-warning=ExperimentalWarning --disable-warning=MODULE_TYPELESS_PACKAGE_JSON test/**/*.test.ts` 줄과 `ℹ fail 0` 줄이 있고, `ℹ pass` 가 89 이상입니다(사전 실행 74 + V-6 3 + V-8 `todo.test.ts` 6 + V-8 `sync-protocol.test.ts` 1 + V-9 4 + V-10 `upcoming.test.ts` 1).
    5. `v1-build.txt` 에 `out/main/index.js`, `out/preload/index.js`, `out/renderer/index.html` 산출물 줄이 있고, `v1-build-cli.txt` 에 `ESM dist/cli.js` 와 `Build success` 줄이 있습니다.
    6. `v1-test-repo.txt` 에 `ℹ fail 0` 줄이 있고, `ℹ pass` 가 14 이상입니다(V-5 의 14개 테스트).
  - 실패: 어느 하나라도 `exit≠0` 이거나, `ℹ fail` 이 0 이 아니거나, `ℹ pass` 가 기준 미만(데스크톱 89 미만, 루트 14 미만)이거나, 위 기대 문자열이 원본에 없습니다.
  - 설계: qa-designer(gemini-4-argon-high) — 과제 `.loctx/reports/1.3.3/qa/task-1.md`, 답 `.loctx/reports/1.3.3/qa/answer-1.md`
  - 사전 실행: `.loctx/reports/1.3.3/dryrun/v1-{typecheck,lint,format,test,build,build-cli,test-repo}.txt` — ①~⑥ `exit=0`, 기대 문자열 모두 있음(`v1-typecheck.txt:7–9`, `v1-format.txt:5`, `v1-test.txt:3` · `:104 ℹ pass 74` · `:105 ℹ fail 0`, `v1-build.txt:8,15,22`, `v1-build-cli.txt:8–9`). ⑦ 은 `Could not find 'scripts/test/'` · `exit=1`(구현 전이라 정상)

- **V-2 범위와 불변 파일 (S-6 · S-7, W-1 ~ W-7)** — 저장소 루트에서 `git diff --name-only origin/master...HEAD -- apps/server apps/desktop/drizzle/0000* apps/desktop/drizzle/0001* apps/desktop/drizzle/0002* apps/desktop/drizzle/0003* apps/desktop/drizzle/0004* apps/desktop/drizzle/0005* apps/desktop/drizzle/0006* apps/desktop/drizzle/0007* apps/desktop/drizzle/0008* .agents/harness .claude/settings.json .claude/skills` 를 실행해 `.loctx/reports/1.3.3/test/v2-scope.txt` 에 남깁니다.
  - 기대:
    1. 표준 출력이 0줄(본문 비어 있음)이고 `exit=0` 입니다(`apps/server`, 기존 커밋된 마이그레이션 `0000`~`0008`, reins 관리 파일 무변경).
  - 실패: 파일 경로가 한 줄이라도 출력되거나 `exit≠0` 입니다.
  - 설계: qa-designer(gemini-4-argon-high) — 과제 `.loctx/reports/1.3.3/qa/task-1.md`, 답 `.loctx/reports/1.3.3/qa/answer-1.md`
  - 사전 실행: `dryrun/v2-scope.txt` — 출력 0줄, `exit=0`

- **V-3 hook 판별기 파이썬 · Node 출력 대조 (S-1, W-1)** — 사례 파일 `.loctx/reports/1.3.3/hook-cases.jsonl`(60줄, 한 줄에 hook 입력 JSON 하나: 인계 표의 모든 명령 51개 + 경계 사례 6개 + 읽을 수 없는 입력 3개 — 숫자 command, JSON 아님, command 없음)과 실행기 `.loctx/reports/1.3.3/run-hook-cases.sh`(줄마다 판별기에 넣고 `<줄 번호><TAB><판정>` 출력)를 씁니다. 구현 뒤 저장소 루트에서 아래 네 명령을 각각 실행해 `.loctx/reports/1.3.3/test/` 에 남깁니다.
  - a) `git show origin/master:.claude/hooks/bash_command.py > .loctx/reports/1.3.3/test/bash_command_origin.py` → `v3-extract.txt`
  - b) `sh .loctx/reports/1.3.3/run-hook-cases.sh .loctx/reports/1.3.3/hook-cases.jsonl python3 .loctx/reports/1.3.3/test/bash_command_origin.py` → `v3-py.txt`
  - c) `sh .loctx/reports/1.3.3/run-hook-cases.sh .loctx/reports/1.3.3/hook-cases.jsonl node .claude/hooks/bash-command.mjs` → `v3-node.txt`
  - d) `diff <(sed -n '2,58p' .loctx/reports/1.3.3/test/v3-py.txt) <(sed -n '2,58p' .loctx/reports/1.3.3/test/v3-node.txt)` 를 `bash -c` 로 → `v3-diff.txt`
  - 기대:
    1. 네 파일 모두 끝 줄이 `exit=0` 입니다.
    2. `v3-py.txt` · `v3-node.txt` 의 본문이 각각 60줄(`1<TAB>` ~ `60<TAB>`)입니다.
    3. `v3-diff.txt` 의 본문이 0줄입니다(사례 1~57 의 판정이 글자까지 같음).
    4. 사례 58 · 59 · 60 은 두 파일 모두 `refuse cannot read the Bash command from the hook input (` 로 시작합니다(괄호 안의 예외 문구는 언어마다 달라 대조하지 않음).
    5. `v3-py.txt` 가 사전 실행의 `.loctx/reports/1.3.3/hook-py-baseline.txt` 와 본문이 같습니다(같은 판별기).
  - 실패: 줄 수가 60 이 아님, `v3-diff.txt` 에 한 줄이라도 나옴, 사례 58~60 의 앞부분이 다름, 또는 `exit≠0`.
  - 설계: qa-designer(gemini-4-argon-high) — 과제 `.loctx/reports/1.3.3/qa/task-1.md`, 답 `.loctx/reports/1.3.3/qa/answer-1.md` (PM 이 사전 실행 뒤 고침: 조정 내용은 이 검사 본문)
  - 사전 실행: `.loctx/reports/1.3.3/hook-py-baseline.txt` — 지금 판별기로 60줄, 인계 표와 모두 일치, `exit=0`. b ~ d 는 구현 뒤 — Node 판별기가 아직 없음

- **V-4 hook 스크립트 전환과 파이썬 판별기 제거 (S-2 · S-3, W-1)** — 구현 뒤 저장소 루트에서 아래 세 명령을 각각 실행해 `.loctx/reports/1.3.3/test/` 에 남깁니다:
  - a) `test ! -e .claude/hooks/bash_command.py && test -f .claude/hooks/bash-command.mjs` → `v4-files.txt`
  - b) `grep -rnE "bash_command\.py|python3|splits the command the way the shell does" .claude/hooks AGENTS.md .agents/knowledge` → `v4-no-python.txt`
  - c) `grep -nE "command -v node|bash-command\.mjs|120" .claude/hooks/pre-bash.sh .claude/hooks/post-bash.sh` → `v4-hooks-node.txt`
  - 기대:
    1. `v4-files.txt` 가 본문 없이 `exit=0` 입니다(`bash_command.py` 삭제, `bash-command.mjs` 존재).
    2. `v4-no-python.txt` 는 출력 0줄이고 `exit=1` 입니다(`.claude/hooks`, `AGENTS.md`, `.agents/knowledge` 에 `bash_command.py`, `python3`, 옛 shlex 주석 문구가 남지 않음).
    3. `v4-hooks-node.txt` 가 `exit=0` 이고, `pre-bash.sh` 와 `post-bash.sh` 두 파일 모두에 `command -v node` 와 `bash-command.mjs` 호출 줄이 있으며, `post-bash.sh` 에 `120` 초 신선도 검사 줄과 120초 한계를 설명하는 주석 줄이 함께 있습니다.
  - 실패: `v4-files.txt` 가 `exit≠0` 이거나, `v4-no-python.txt` 에 한 줄이라도 출력(`exit=0`)되거나, `v4-hooks-node.txt` 에 두 스크립트의 `node` 호출 · `120` 초 검사와 주석이 빠져 있습니다.
  - 설계: qa-designer(gemini-4-argon-high) — 과제 `.loctx/reports/1.3.3/qa/task-1.md`, 답 `.loctx/reports/1.3.3/qa/answer-1.md`
  - 사전 실행: `dryrun/v4-no-python.txt` — 지금은 `pre-bash.sh` · `post-bash.sh` · `bash_command.py` 에서 6줄 나옴(구현 전 상태, `AGENTS.md` · knowledge 에는 없음). a · c 는 구현 뒤

- **V-5 루트 스크립트 테스트와 CI 작업 (S-1 · S-3 · S-4, W-1 · W-2)** — 구현 뒤 저장소 루트에서 아래 명령을 각각 실행해 `.loctx/reports/1.3.3/test/` 에 남깁니다:
  - a) `node --test scripts/test/bash-command.test.mjs scripts/test/post-bash.test.mjs scripts/test/check-spec-checks.test.mjs scripts/test/check-design-commit-msg.test.mjs` → `v5-repo-tests.txt`
  - b) `grep -nF '"test:repo": "node --test '"'"'scripts/test/*.test.mjs'"'"'"' package.json` → `v5-package-json.txt` (2026-11-22 정정, A-1 · 사용자 "승인한다")
  - c) `grep -nE 'repo:|repo-scripts|scripts/\*\*|\.claude/hooks/\*\*|node --test '"'"'scripts/test/\*\.test\.mjs' .github/workflows/ci.yml` → `v5-ci-yml.txt` (2026-11-22 정정, A-1 · 사용자 "승인한다")
  - d) (merge 단계, PR 을 연 뒤 1차 감사 전) `gh pr checks <PR번호>` → `v5-pr-checks.txt`
  - 기대:
    1. `v5-repo-tests.txt` 가 `exit=0`, `ℹ fail 0` 이고 아래 14개 테스트 이름이 모두 `✔ <이름>` 줄로 나타납니다:
       - `bash-command.test.mjs` (3개):
         - `classifies every allowed push merge commit and neutral command in the history table`
         - `refuses every non-exact push and merge command in the history table`
         - `cli refuses non-json and missing command input on stdin`
       - `post-bash.test.mjs` (2개):
         - `skips gates and exits 0 when HEAD commit is older than 120 seconds`
         - `runs the gate step when HEAD commit is fresh`
       - `check-spec-checks.test.mjs` (5개, 인계 표 5행 전부):
         - `rejects a newly added V-n heading when design or dry-run line is missing`
         - `accepts a newly added V-n heading when both design and dry-run lines are present`
         - `ignores edits to the body of an existing V-n check`
         - `requires both lines on every V-n in a newly added spec file`
         - `requires both lines when a V-n heading is moved or renumbered in the diff`
       - `check-design-commit-msg.test.mjs` (4개, 인계 표 11행 전부):
         - `exits 0 in commit-msg mode when DESIGN.md is not staged`
         - `rejects malformed or commented Spec-check lines in commit-msg mode`
         - `accepts a valid Spec-check line and detects staged rename of DESIGN.md`
         - `checks commit ranges including amended commits merge commits and root commits`
    2. `v5-package-json.txt` 에 `"test:repo": "node --test 'scripts/test/*.test.mjs'"` 줄이 1줄 있고 `exit=0` 입니다.
    3. `v5-ci-yml.txt` 가 `exit=0` 이고, `changes` 작업의 `repo` 출력과 필터(`scripts/**`, `.claude/hooks/**`), `pnpm` 설치 없이 `node --test 'scripts/test/*.test.mjs'` 를 돌리는 `repo-scripts` 작업, `ci-ok`(`CI result`)의 `needs` 목록에 포함된 `repo-scripts` 가 모두 나타납니다.
    4. `v5-pr-checks.txt` 가 `exit=0` 이고, PR 의 실제 GitHub Actions 실행에서 `repo-scripts`(또는 그 작업의 `name`)와 `CI result` 줄이 모두 `pass` 입니다.
  - 실패: `v5-repo-tests.txt` 에 `ℹ fail 0` 또는 14개 `✔` 줄 중 하나라도 없거나, `package.json` · `ci.yml` 에 해당 줄이 없거나, PR 의 CI 실행에서 `repo-scripts` 작업이 빠졌거나 실패합니다.
  - 설계: qa-designer(gemini-4-argon-high) — 과제 `.loctx/reports/1.3.3/qa/task-1.md`, 답 `.loctx/reports/1.3.3/qa/answer-1.md`
  - 사전 실행: 구현 뒤 — 테스트 파일 · `test:repo` · CI 작업이 아직 없음(`dryrun/v1-test-repo.txt` 의 `Could not find`)

- **V-6 동기화 스위치 유지 단위 테스트 (S-5, W-3)** — `sync-config.ts` 는 Electron(`app`, `safeStorage`)과 DB 관리자를 불러 테스트에서 직접 부를 수 없으므로, 로그아웃 · 새 서버 때 쓰는 갱신 값을 Electron 없는 모듈 `src/main/sync/binding-state.ts`(`SIGNED_OUT_STATE`, `NEW_SERVER_STATE`)로 빼고 `sync-config.ts` 가 그것을 씁니다. 구현 뒤 `apps/desktop` 에서 `node --test --disable-warning=ExperimentalWarning --disable-warning=MODULE_TYPELESS_PACKAGE_JSON test/sync-binding-state.test.ts` → `.loctx/reports/1.3.3/test/v6-sync-binding.txt`. 테스트는 커밋된 마이그레이션을 적용한 `node:sqlite` 메모리 DB(`outbox-triggers.test.ts` 와 같은 방식)의 `sync_server` 행에 두 상태를 UPDATE 로 적용해 봅니다.
  - 기대:
    1. `exit=0`, `ℹ fail 0` 이고 아래 3개가 모두 `✔` 입니다.
       - `sign-out state clears the account and cursors and leaves enabled alone` — `SIGNED_OUT_STATE` 에 `enabled` 키가 없고, `enabled = 1` 인 행에 적용하면 `accountId` · `teamId` · `lastPulledSeq`(0) · `bootstrapCursor` · `bootstrappedAt` · `backfilledAt` · `lastSyncedAt` · `lastErrorKind` · `lastError` 가 초기화되고 `url` · `allowInsecureTls` · `enabled`(1)는 그대로입니다. `enabled = 0` 인 행은 0 그대로입니다.
       - `sign-in after sign-out keeps the enabled value` — 위 행에 로그인 때 쓰는 갱신(`accountId` 설정)을 적용해도 `enabled` 가 로그아웃 전 값입니다.
       - `a new server URL turns enabled off` — `NEW_SERVER_STATE` 는 `enabled: false` 를 갖고, 적용하면 `enabled = 0` 입니다.
  - 실패: `exit≠0`, `ℹ fail` 이 0 이 아님, 또는 위 3개 `✔` 줄 중 하나라도 없음.
  - 설계: qa-designer(gemini-4-argon-high) — 과제 `.loctx/reports/1.3.3/qa/task-1.md`, 답 `.loctx/reports/1.3.3/qa/answer-1.md` (PM 이 사전 실행 뒤 고침: 조정 내용은 이 검사 본문)
  - 사전 실행: 구현 뒤 — `binding-state.ts` 와 테스트가 아직 없음

- **V-7 공개 타입 · Changeset · 마이그레이션 0009 · outbox 트리거 (S-6 · S-7, W-4)** — 구현 뒤 아래 네 명령을 각각 실행해 `.loctx/reports/1.3.3/test/` 에 남깁니다:
  - a) 저장소 루트에서 `grep -nE "kind\?: 'event' \| 'todo';|completedAt\?: Date \| null;" packages/types/models/schedule.d.ts apps/desktop/src/common/models/schedule.ts` → `v7-types.txt`
  - b) 저장소 루트에서 `git diff --name-only origin/master...HEAD -- '.changeset/*.md'` → `v7-changeset-files.txt`, 그리고 `grep -nHE '^["'"'"']@croffledev/croffle-types["'"'"']: minor$' .changeset/*.md` → `v7-changeset-body.txt`
  - c) 저장소 루트에서 `pnpm --filter @croffledev/desktop db:generate` → `v7-db-generate.txt`
  - d) `apps/desktop` 에서 `node --test --disable-warning=ExperimentalWarning --disable-warning=MODULE_TYPELESS_PACKAGE_JSON test/outbox-triggers.test.ts` → `v7-outbox.txt`
  - 기대:
    1. `v7-types.txt` 가 `exit=0` 이고, `packages/types/models/schedule.d.ts` 와 `apps/desktop/src/common/models/schedule.ts` 두 파일 모두에서 선택 필드 `kind?: 'event' | 'todo';` 와 `completedAt?: Date | null;` 가 각 1줄씩 총 4줄 출력됩니다.
    2. `v7-changeset-files.txt` 가 `exit=0` 이고 `.changeset/README.md` 가 아닌 새 `.changeset/<이름>.md` 파일 경로가 정확히 1줄 출력되며, `v7-changeset-body.txt` 가 `exit=0` 이고 그 파일의 머리말 `@croffledev/croffle-types: minor` 줄(따옴표 종류는 묻지 않음 — 2026-11-22 정정: oxfmt 가 YAML 을 작은따옴표로 바꿈, 사용자 "승인한다") 1줄만 출력됩니다.
    3. `v7-db-generate.txt` 가 `exit=0` 이고 `schedule 20 columns 0 indexes 0 fks` 줄과 `No schema changes, nothing to migrate` 줄이 있습니다(스키마와 마이그레이션 0009 가 일치해 새 파일이 더 생기지 않음).
    4. `v7-outbox.txt` 가 `exit=0`, `ℹ fail 0`, `ℹ pass 9` 이고 기존 9개 트리거 테스트가 모두 `✔` 입니다.
  - 실패: 두 모델 파일 중 한쪽에라도 선택 필드가 없거나 필수 필드로 들어갔거나, 새 Changeset 파일이 1개가 아니거나 `minor` 가 아니거나, `db:generate` 가 새 마이그레이션을 만들거나 `schedule 20 columns` 가 아니거나, `v7-outbox.txt` 에서 실패가 납니다.
  - 설계: qa-designer(gemini-4-argon-high) — 과제 `.loctx/reports/1.3.3/qa/task-1.md`, 답 `.loctx/reports/1.3.3/qa/answer-1.md`
  - 사전 실행: `dryrun/v7-db-generate.txt:12` `schedule 18 columns 0 indexes 0 fks` · `:17` `No schema changes, nothing to migrate` (구현 뒤 20), `dryrun/v7-outbox.txt:17–18` `ℹ pass 9` · `ℹ fail 0`, `dryrun/v7-changeset-files.txt` 0줄. a 는 구현 뒤

- **V-8 할 일 데이터 · 서비스 검증 · 매퍼 · 동기화 와이어 테스트 (S-7 · S-8 · S-9, W-4)** — 구현 뒤 `apps/desktop` 에서 아래 두 명령을 각각 실행해 `.loctx/reports/1.3.3/test/` 에 남깁니다:
  - a) `node --test --disable-warning=ExperimentalWarning --disable-warning=MODULE_TYPELESS_PACKAGE_JSON test/todo.test.ts` → `v8-todo.txt`
  - b) `node --test --disable-warning=ExperimentalWarning --disable-warning=MODULE_TYPELESS_PACKAGE_JSON test/sync-protocol.test.ts` → `v8-sync-protocol.txt`
  - 기대:
    1. `v8-todo.txt` 가 `exit=0`, `ℹ fail 0` 이고 아래 6개 테스트가 모두 `✔` 입니다. 서비스는 DB 관리자를 불러 테스트에서 부를 수 없으므로, 할 일 규칙(검증 · 정규화 · 합치기)은 DB 없는 함수 `src/main/calendar/schedule-kind.ts` 로 빼고 서비스가 그것을 씁니다. 테스트는 그 함수와 매퍼를 부릅니다:
       - `migration 0009 adds kind default event and nullable completedAt without backfill UPDATE` — `apps/desktop/drizzle/0009_*.sql` 에 `UPDATE` 문이 없고, 모든 마이그레이션을 적용한 새 인메모리 DB 의 `PRAGMA table_info(schedule)` 에서 `kind`(`notnull: 1`, 기본값 `'event'`)와 `completedAt`(`notnull: 0`, 기본값 `null`) 열이 확인되며, 두 열을 생략하고 넣은 행이 `kind = 'event'`, `completedAt = null` 로 읽힙니다.
       - `rejects an invalid kind value` — `kind` 가 `'event'` · `'todo'` 가 아니면 할 일 규칙 함수가 예외를 던집니다.
       - `aligns todo startDate to endDate and rejects recurrence on a todo` — 할 일(`kind: 'todo'`)은 저장 · 정규화 시 `startDate` 가 `endDate`(마감)로 맞춰지고, `recurrenceRule` 이 있으면 거절됩니다.
       - `rejects completedAt on an event and clears completedAt when switching todo to event` — 일정(`kind: 'event'` 또는 생략)에 `completedAt` 이 있으면 거절되고, 완료된 할 일을 일정(`kind: 'event'`)으로 고치면 `completedAt` 이 `null` 로 지워집니다.
       - `update merge preserves omitted kind and completedAt` — 고치기의 합친 값 계산에서 `kind` 와 `completedAt` 을 생략하면 기존 할 일의 `kind: 'todo'` 와 `completedAt` 시각이 유지되고, `completedAt: null` 을 넘기면 완료가 해제됩니다.
       - `scheduleMapper maps kind and completedAt and defaults missing kind on import to event` — `toInterface` 와 `toEntity` 가 두 필드를 왕복 변환하고, `kind` 가 없는 가져오기 입력은 `'event'` 로 다룹니다.
    2. `v8-sync-protocol.txt` 가 `exit=0`, `ℹ fail 0` 이고, 공유 픽스처(`packages/sync-fixtures/scenarios/12-row-codec.json`)의 `schedule` 사례들이 `kind` · `completedAt` 을 포함해 `row-codec: case 0 (schedule) encodes to the wire row` · `row-codec: case 0 (schedule) decodes back to the local row` · `row-codec: case 1 (schedule) encodes to the wire row` · `row-codec: case 1 (schedule) decodes back to the local row` 를 통과하며, 새 테스트 `schedule decode defaults missing or null kind from an older client to event`(1.3.2 이하 기기가 보낸 `kind` 없는 와이어 행을 `kind: 'event'` 로 복원하고 할 일의 `completedAt` 초 ↔ 밀리초 변환을 검증)가 `✔` 입니다.
  - 실패: 두 파일 중 하나라도 `exit≠0` 이거나 `ℹ fail` 이 0 이 아니거나 위 `✔` 테스트 줄이 빠져 있습니다.
  - 설계: qa-designer(gemini-4-argon-high) — 과제 `.loctx/reports/1.3.3/qa/task-1.md`, 답 `.loctx/reports/1.3.3/qa/answer-1.md` (PM 이 사전 실행 뒤 고침: 조정 내용은 이 검사 본문)
  - 사전 실행: `dryrun/v8-sync-protocol.txt:5–8` — `row-codec: case 0/1 (schedule) encodes … / decodes …` 네 줄이 지금 이름 그대로 있음, `ℹ pass 25`. `todo.test.ts` 는 구현 뒤

- **V-9 할 일 리마인더 단위 테스트 (S-10, W-5)** — 구현 뒤 `apps/desktop` 에서 `node --test --disable-warning=ExperimentalWarning --disable-warning=MODULE_TYPELESS_PACKAGE_JSON test/reminder.test.ts` 를 실행해 `.loctx/reports/1.3.3/test/v9-reminder.txt` 에 남깁니다.
  - 기대:
    1. `exit=0`, `ℹ fail 0` 이고 아래 4개 테스트가 모두 `✔` 입니다:
       - `excludes completed todos from reminder candidates` — `completedAt` 이 있는 할 일은 `buildReminderCandidates` 결과에서 빠지고, 미완료 할 일은 남습니다.
       - `schedules timed todo reminder relative to due time` — 시각이 있는 미완료 할 일(`startDate = endDate = 마감`)의 `remindAt` 은 `마감 − minutes * 60_000` 입니다.
       - `schedules all-day todo reminder relative to local midnight of the due date` — 종일 미완료 할 일의 `remindAt` 은 마감일 로컬 0시(`startOfLocalDay`)에서 `minutes` 분 전입니다.
       - `formats todo reminder body with due copy in ko and en and keeps event reminder behaviour` — 할 일 후보의 알림 본문은 `ko` · `en` 모두 마감 문구(`reminder.bodyTodoAllDay` · `reminder.bodyTodoTimed`)를 쓰고, 일반 일정(`event`)은 기존 시작 기준 계산과 `reminder.bodyAllDay` · `reminder.bodyTimed` 문구를 그대로 씁니다.
  - 실패: `exit≠0`, `ℹ fail` 이 0 이 아님, 또는 위 4개 `✔` 줄 중 하나라도 없습니다.
  - 설계: qa-designer(gemini-4-argon-high) — 과제 `.loctx/reports/1.3.3/qa/task-1.md`, 답 `.loctx/reports/1.3.3/qa/answer-1.md`
  - 사전 실행: 구현 뒤 — `reminder.test.ts` 가 아직 없음

- **V-10 다가오는 일정 할 일 제외 · 카탈로그 · 화면 토큰 규칙 (S-14 · S-15, W-6)** — 구현 뒤 아래 세 명령을 각각 실행해 `.loctx/reports/1.3.3/test/` 에 남깁니다:
  - a) `apps/desktop` 에서 `node --test --disable-warning=ExperimentalWarning --disable-warning=MODULE_TYPELESS_PACKAGE_JSON test/upcoming.test.ts test/i18n-catalogs.test.ts` → `v10-upcoming-i18n.txt`
  - b) 저장소 루트에서 `bash apps/desktop/scripts/check-ui-tokens.sh` → `v10-ui-tokens.txt`
  - c) 저장소 루트에서 `node scripts/check-design-commit-msg.mjs --range origin/master..HEAD` → `v10-design-commits.txt`
  - 기대:
    1. `v10-upcoming-i18n.txt` 가 `exit=0`, `ℹ fail 0`, `ℹ pass 10` 이고, `upcoming.test.ts` 의 기존 6개 테스트와 새 테스트 `excludes todos even when due within 7 days` 가 모두 `✔` 이며, `i18n-catalogs.test.ts` 의 3개 테스트(`ko and en have the same keys`, `ko has no untranslated English values`, `help copy in ko uses polite haeyo style`)가 모두 `✔` 입니다.
    2. `v10-ui-tokens.txt` 가 `exit=0` 이고, `# palette colours, hex values, rgba()` 아래에는 `./app.vue` 의 닫기 버튼 호버 1줄만, `# text-destructive-foreground (only over bg-destructive)` 아래에는 같은 줄에 `bg-destructive` 가 있는 `components/ui/button/index.ts` 와 `components/ui/badge/index.ts` 2줄만, `# browser confirm()` 아래에는 메서드 호출 `patches.confirm(` 1줄만 있으며 브라우저 `confirm(` 호출이나 허용되지 않은 팔레트 · hex · `rgba(` 가 0줄입니다.
    3. `v10-design-commits.txt` 가 본문 0줄이고 `exit=0` 입니다(`apps/desktop/DESIGN.md` 를 바꾼 커밋이 있으면 모두 유효한 `Spec-check:` 줄을 가짐).
  - 실패: 테스트가 하나라도 실패하거나, `v10-ui-tokens.txt` 에 허용된 줄 밖의 출력이 나오거나, `v10-design-commits.txt` 가 `exit≠0` 입니다.
  - 설계: qa-designer(gemini-4-argon-high) — 과제 `.loctx/reports/1.3.3/qa/task-1.md`, 답 `.loctx/reports/1.3.3/qa/answer-1.md`
  - 사전 실행: `dryrun/v10-upcoming-i18n.txt` — 기존 9개 `✔`(upcoming 6 · 카탈로그 3), `ℹ pass 9`; `dryrun/v10-ui-tokens.txt` — 허용 줄 4개만, `exit=0`; `dryrun/v10-design-commits.txt` — 0줄, `exit=0`

- **V-11 할 일 화면 · 동기화 스위치 수동 확인 (S-5 · S-11 · S-12 · S-13 · S-14 · S-15, W-3 · W-6)** — 격리된 작업 디렉터리와 `--user-data-dir`(실제 `~/.config/Croffle` 을 건드리지 않음)로 띄운 개발 실행(`pnpm dev` 와 같은 Vite 렌더러 + 미패키지 메인 실행)에서 라이트와 다크 각각 아래 8개 항목을 확인합니다. 실행 주체(PM 의 Playwright `_electron` 또는 사용자), 실행 명령 · 환경, 항목별 관찰 결과를 `.loctx/reports/1.3.3/test/v11-light.txt` 와 `.loctx/reports/1.3.3/test/v11-dark.txt` 에 적고, 스크린샷 16장(`1-modal-light.png` … `8-en-dark.png`)을 `.loctx/reports/1.3.3/test/v11/screens/` 에 저장한 뒤 `ls -l .loctx/reports/1.3.3/test/v11/screens/` 출력을 `.loctx/reports/1.3.3/test/v11-screens-ls.txt` 에 남깁니다.
  - 기대:
    1. **모달 종류 전환 · 마감 칸 · 반복 비활성 (S-11)**: 일정 모달 편집 화면 위쪽에 `일정` / `할 일` 전환(segmented)이 있습니다. `할 일`을 고르면 시작 날짜 · 시각 칸이 사라지고 `마감` 칸 하나만 보이며(종일이면 날짜만, 종일을 끄면 날짜와 시각), 반복 칸은 비활성(disabled)이고 "할 일은 반복할 수 없습니다" 안내가 보입니다. 반복이 있는 일정을 편집으로 열어 `할 일`로 바꾸어 저장하면 반복 규칙이 지워진 할 일로 저장됩니다(`1-modal-{light,dark}.png`).
    2. **보기 화면과 완료 버튼 (S-11)**: 할 일을 열어 보기 화면을 보면 "할 일 · 마감 <날짜/시각>"과 완료 상태, 완료 / 완료 취소 버튼이 보입니다. 완료 버튼을 누르면 완료 상태로 바뀌고, 완료 취소를 누르면 미완료로 돌아옵니다(`2-viewer-{light,dark}.png`).
    3. **캘린더 체크 상자와 취소선 (S-12)**: 캘린더의 마감일 칸에 할 일이 체크 상자 + 제목으로 보입니다. 체크 상자를 누르면 보기 창이 열리지 않고 즉시 완료로 바뀌며, 완료된 할 일은 캘린더에 취소선(`line-through`)과 흐린 글자로 남습니다. 체크 상자를 다시 누르면 취소선이 풀리고 미완료로 돌아옵니다(`3-calendar-{light,dark}.png`).
    4. **드래그로 마감 이동 (S-12)**: 월간 보기에서 할 일 이벤트를 다른 날짜 칸으로 끌어 놓으면 확인 창 없이 마감이 그 날짜로 옮겨지고, 보기 화면에서 마감 날짜가 옮긴 날(시각 유지)로 바뀐 것이 확인됩니다(`4-drag-{light,dark}.png`).
    5. **오른쪽 사이드바 "할 일" 칸 (S-13)**: 미완료 할 일 전부가 마감 오름차순으로 나열되고, 마감이 지금보다 지난 항목에는 `text-destructive` 로 "지남" 표시가 붙습니다. 항목을 더블클릭하면 보기 창이 열립니다. 체크 상자를 눌러 완료하면 "할 일" 목록에서 빠집니다. 미완료 할 일이 모두 완료되면 "할 일이 없습니다" 문구가 보입니다(`5-sidebar-todo-{light,dark}.png`).
    6. **다가오는 일정 · 오늘 일정 개수 제외와 고른 날 목록 (S-14)**: 오늘 마감인 할 일과 7일 안 마감인 할 일이 있어도 왼쪽 사이드바의 "오늘 일정 N개" 부제와 오른쪽 사이드바의 "다가오는 일정"에는 할 일이 포함되지 않고 일반 일정만 세어집니다. 오른쪽 사이드바 위쪽의 고른 날 목록에는 그날 마감인 할 일이 체크 상자와 함께 보입니다(`6-counts-upcoming-{light,dark}.png`).
    7. **동기화 스위치 로그아웃 유지 표시 (S-5)**: 동기화를 켠(`enabled = true`) 뒤 로그아웃된 상태에서 설정 › 동기화 탭을 열면 동기화 스위치가 꺼지지 않고 켜진 상태 그대로 비활성(disabled)으로 보입니다(`7-sync-switch-{light,dark}.png`).
    8. **한국어 · 영어 및 라이트 · 다크 (S-15)**: 설정 › 일반에서 언어를 한국어(`ko`)와 영어(`en`)로 전환했을 때 위 할 일 관련 모든 새 문구가 두 언어와 라이트 · 다크 두 테마에서 누락이나 레이아웃 깨짐 없이 표시됩니다(`8-en-{light,dark}.png`).
  - 실패: 1~8 중 하나라도 기대와 다르거나 미확인으로 남거나, `v11-screens-ls.txt` 에 스크린샷 16장이 없거나 0바이트 파일이 있습니다.
  - 설계: qa-designer(gemini-4-argon-high) — 과제 `.loctx/reports/1.3.3/qa/task-1.md`, 답 `.loctx/reports/1.3.3/qa/answer-1.md`
  - 사전 실행: 구현 뒤 — 화면이 아직 없음. 실행 조건(격리 user-data-dir, 렌더러 개발 서버 시작 메시지, 남은 프로세스 · 포트 없음)은 실행 전에 확인

- **V-12 문서 확인 (S-16, W-7)** — 구현 뒤 저장소 루트에서 아래 세 명령을 각각 실행해 `.loctx/reports/1.3.3/test/` 에 남깁니다:
  - a) `grep -nE -i "to-do|todo" README.md` → `v12-readme-en.txt`
  - b) `grep -n "할 일" README.ko.md` → `v12-readme-ko.txt`
  - c) `grep -nE "kind|completedAt" packages/types/README.md` → `v12-types-readme.txt`
  - 기대:
    1. `v12-readme-en.txt` 가 `exit=0` 이고 `README.md` 의 기능 목록(`Key Features`)에 할 일(To-Do)을 설명하는 줄이 1줄 이상 있습니다.
    2. `v12-readme-ko.txt` 가 `exit=0` 이고 `README.ko.md` 의 기능 목록(`주요 기능`)에 할 일을 설명하는 줄이 1줄 이상 있습니다.
    3. `v12-types-readme.txt` 가 `exit=0` 이고 `packages/types/README.md` 에 `Schedule` 의 `kind` 와 `completedAt` 두 필드가 모두 안내되어 있습니다.
  - 실패: 세 명령 중 하나라도 `exit≠0` 이거나 해당 파일에 할 일 기능 또는 두 필드의 설명이 없습니다.
  - 설계: qa-designer(gemini-4-argon-high) — 과제 `.loctx/reports/1.3.3/qa/task-1.md`, 답 `.loctx/reports/1.3.3/qa/answer-1.md`
  - 사전 실행: `dryrun/v12-readme-en.txt` · `v12-readme-ko.txt` — 지금은 둘 다 0줄 · `exit=1`(아직 없음). c 는 구현 뒤

- **V-13 릴리스 노트 확인 (S-16, W-7)** — 이 검사는 test 단계가 아니라 release 단계(`1.3.3-rc.1` 릴리스 노트를 작성한 뒤)에 저장소 루트에서 아래 세 명령을 각각 실행해 `.loctx/reports/1.3.3/test/` 에 남깁니다:
  - a) `nl -ba .loctx/release/1.3.3-rc.1_Pre-Release.md` → `v13-note.txt`
  - b) `grep -nE '모든 기기|every device' .loctx/release/1.3.3-rc.1_Pre-Release.md` → `v13-sync-notice.txt`
  - c) `git check-ignore -v .loctx/release/1.3.3-rc.1_Pre-Release.md` → `v13-ignored.txt`
  - 기대:
    1. `v13-note.txt` 가 `exit=0` 이고 `.github/release-notes.md` 의 데스크톱 Pre-release 양식(한국어 · 영어 두 절)을 따릅니다.
    2. 한국어와 영어 두 절 모두에 "동기화하는 모든 기기를 1.3.3 으로 올리십시오"(1.3.2 이하 기기가 할 일을 수정하면 `kind` · `completedAt` 열이 빠져 일반 일정으로 바뀜) 안내가 들어 있습니다(`v13-sync-notice.txt` 가 `exit=0` 이고 `모든 기기` 줄 1개 이상 · `every device` 줄 1개 이상).
    3. `v13-ignored.txt` 가 `exit=0` 으로 `.loctx/` 무시 규칙을 출력합니다.
  - 실패: 파일이 없거나, 한국어 · 영어 두 절 중 한쪽에라도 동기화 기기 업그레이드 안내가 없거나, `git check-ignore` 가 `exit≠0` 입니다.
  - 설계: qa-designer(gemini-4-argon-high) — 과제 `.loctx/reports/1.3.3/qa/task-1.md`, 답 `.loctx/reports/1.3.3/qa/answer-1.md` (PM 이 사전 실행 뒤 고침: 조정 내용은 이 검사 본문)
  - 사전 실행: release 단계 — 릴리스 노트가 아직 없음. 파일 이름은 기존 `.loctx/release/1.3.2-rc.2_Pre-Release.md` 의 형식

## 조정 기록
- A-1 (2026-11-22, W-1 · W-2) 루트 테스트 실행 명령을 `node --test scripts/test/` 에서 `node --test 'scripts/test/*.test.mjs'` 로 · Node 24 의 `node --test` 는 디렉터리 인자를 파일로 읽어 `Cannot find module …/scripts/test` 로 실패함(사전 실행 `dryrun/v1-test-repo.txt` 는 디렉터리가 없어서 난 다른 오류라 놓침). S-4 · 설계는 바꿈. V-1 ⑦ · V-5 b · c 의 명령 · 기대 문자열 정정은 test 전에 사용자 승인(`kind: change`)을 받음 · W-1 커밋
- A-2 (2026-11-22, W-4) 미완료 할 일 내부 API 를 `croffleInternal.todos.listOpen()` 대신 기존 내부 일정 API 옆 `croffleInternal.calendar.schedules.listOpenTodos()`(IPC `schedule:listOpenTodos`)로 · 내부 일정 API(`InternalSchedulesApi`)가 이미 있어 새 묶음을 만들 이유가 없음. 공유 픽스처 12 에 완료한 할 일 사례(case 2, `row-3`)를 더함 — 뒤 사례들의 번호가 하나씩 밀림(V-8 이 이름으로 부르는 case 0 · 1 은 그대로) · W-4 커밋
- A-3 (2026-11-22, W-5) `formatReminderBody` 를 `src/common/reminder-format.ts` 로 옮기고, `reminder.ts` 에는 본문의 카탈로그 키 · 값을 고르는 순수 함수 `reminderBody` 를 둠 · `reminder.ts` 가 i18n 을 부르면 i18n 이 TypeScript enum(`AppSettingLanguage`)을 불러 `node --test`(strip-only)에서 읽히지 않음. V-9 의 네 번째 테스트는 `reminderBody` 와 ko · en 카탈로그 파일로 앱과 같은 본문을 만들어 대조함(이름 · 기대는 그대로) · W-5 커밋
- A-4 (2026-11-22, W-6) 화면의 작은 조정 세 가지 · ① 오른쪽 사이드바 "할 일" 칸에 불러오는 중 · 오류 상태를 둠(DESIGN.md "Every panel that loads data has a loading, an error and an empty state"), ② 고른 날 목록에서 일정(할 일이 아닌 것)의 아이콘을 체크 표시 대신 달력으로(체크 상자와 헷갈리지 않게), ③ 할 일일 때 "하루 종일" 안내를 "끄면 마감 시각을 정합니다"로(시작 · 종료 시각이 없으므로) · W-6 커밋
- A-5 (2026-11-22, W-1 · W-6) 리뷰 1(FAIL) 지적 R-1 ~ R-4 수정 · R-1 캘린더의 일반 일정 내용을 FullCalendar 기본 렌더러(시간 글자 `fc-event-time`, 빈 제목은 공백)와 같게, R-2 판별기의 앞뒤 공백 제거를 Python `str.strip()` 의 문자 집합으로(U+FEFF · U+001C · U+0085 회귀 사례 추가, Python · Node 를 공백 문자 96개 사례로 대조해 차이 0), R-3 할 일 목록은 마지막 요청의 답만 반영, R-4 "지남" 표시가 1분마다 갱신되는 현재 시각을 씀 · 리뷰 1 수정 커밋

## 우려
- **1.3.2 이하 기기와 섞인 동기화** — 그 기기가 할 일을 고치면 `kind` · `completedAt` 없이 올라가 모든 기기에서 일반 일정이 됩니다(intent 에이전트가 정한 것). S-9 는 적용 실패만 막고 이 손실은 막지 못합니다. 릴리스 노트로 알립니다.
- **hook 교체 중 막힘** — 새 판별기에 오류가 있으면 모든 Bash 가 거부됩니다. 설계의 확인 순서를 지키고, 막히면 사용자에게 `.claude/hooks/pre-bash.sh` 의 호출 줄을 되돌려 달라고 합니다(Bash 없이 Edit 로 되돌릴 수 있음).
- **post hook 신선도 120초** — 게이트가 오래 걸리는 커밋 직후 연달아 다른 저장소에서 커밋하는 드문 경우는 여전히 게이트를 돕니다(해롭지 않음).
- **캘린더 체크 상자의 클릭 판정** — FullCalendar 이벤트 안의 버튼이 드래그 시작 · `eventClick` 과 겹칠 수 있습니다. 화면 검사에서 봅니다.

## 범위 밖
- 반복 할 일, 할 일의 하위 항목, 할 일만 보는 목록 화면.
- 캘린더 고르기 · 팀 캘린더 · 공유(서버 0.3.0 · 데스크톱 1.4.0).
- 서버 코드 변경(`apps/server/**`). 공유 픽스처 12 는 데스크톱 러너만 읽습니다.
- 기존 마이그레이션 수정, 공개 타입의 필수 필드.
- reins 관리 파일(`gates.yaml`, `.claude/settings.json`, `.claude/skills/**`). 그래서 `test:repo` 는 게이트(`pnpm -r test`)에 들어가지 않고 CI 작업으로만 돕니다.
