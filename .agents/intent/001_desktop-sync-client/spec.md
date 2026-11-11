---
intent: "001"
status: accepted
approvals:
  - { role: user, by: bluenyang, at: "2026-11-11", words: "그래. 이번만이니까 게이트 추가까지 해서 승인할게", rev: "bf53558 (+ W-5 gate line added on these words)", kind: summary }
summary_for_user: |
  데스크톱 1.3.0 동기화 클라이언트를 작업 단위 16개(W-1~W-16)로 나눴습니다: DB · 트리거 → 프로토콜 계약과 첫 테스트 러너 → 전송 · 인증 → push · pull(bootstrap · 410 복구) → 오케스트레이터(첫 동기화 backfill) → 내부 브리지 → 검증 하네스 → 설정 탭 · README → desktop-v 태그 → npm staging 발행.
  intent 계획과 다른 점은 하나입니다: 내부 브리지를 기반 단계에서 엔진 뒤(W-11)로 옮겼습니다. 감싸는 함수가 먼저 있어야 브리지를 한 번에 완성하고 따로 되돌릴 수 있습니다.
  검증 기준 V-1~V-20은 qa-designer가 쓰고 제가 고쳤습니다. 실서버(0.1.0-rc.1) 검증은 Linux · Windows, 1.2.6 업데이트 확인은 stable 게시 직후, npm staging 실발행 확인은 그다음 Changeset 발행 때입니다.
  W-5가 들어가면 `pnpm -r test`를 게이트에 추가해야 하는데 `gates.yaml`은 손으로 고치려면 허락이 필요합니다. 또 npmjs.com의 trusted publishing을 "stage만 허용"으로 바꾸는 일은 2FA가 필요해 직접 하셔야 합니다.
---
## 요구 해석
- S-1 서버를 등록 · 로그인하면 5개 테이블(`schedule` · `tag` · `schedule_tags` · `schedule_reminder` · `schedule_exception`)이 기기 사이에 동기화된다. ← 바라는 결과 1
- S-2 서버를 등록하지 않으면 네트워크 호출이 0건이고 동작은 1.2.6과 같다. outbox는 쌓이지만 아무 데도 보내지 않는다. ← 바라는 결과 2
- S-3 오프라인에서도 UI는 정상이고, 쌓인 변경은 연결되면 자동으로 나간다. ← 바라는 결과 3
- S-4 `sync.*`는 내부 브리지(`croffleInternal`)에만 있고 `packages/types`는 바뀌지 않는다. ← 바라는 결과 4, 하지 않을 것
- S-5 프로토콜의 기준은 서버 0.1(`apps/server/internal/protocol`)과 `packages/sync-fixtures`다. 어긋나면 데스크톱을 맞추고, 서버를 고쳐야 하면 먼저 사용자에게 묻는다. ← 제약 1, 영향 범위
- S-6 outbox는 SQLite 트리거로 적재하고, 서버에서 받은 행(`lastWriterId = 'server'`)은 다시 outbox로 가지 않는다. ← 계획 1, 제약 2
- S-7 토큰은 `safeStorage`로만 디스크에 둔다. 키링이 없으면 메모리에만 두고 경고한다. 로그에 토큰 · 비밀번호 · OTP가 남지 않는다. ← 제약 3, 5
- S-8 TLS는 기본 strict이고, 검증되지 않은 인증서는 설정에서 켠 경우에만, 동기화 서버 호스트에 한해 허용한다. 확장의 `http.*`에는 영향이 없다. ← 문제(2026-10-20 답변 2)
- S-9 인증은 내부적으로 토큰 하나이고, UI는 ID/PW(+OTP)와 토큰 붙여넣기 두 가지다. ← 문제(답변 3)
- S-10 새 기기와 보존 기간 밖 커서(410)는 `/sync/bootstrap`으로 복구한다. ← 질의서 1
- S-11 첫 동기화 때 트리거 도입 전부터 있던 행을 outbox에 한 번 모두 적재해 서버로 올린다. ← 질의서 3
- S-12 연결할 때 `/health`로 프로토콜 범위를 확인하고(`X-Croffle-Sync-Version`), 서버 시계와 크게 어긋나면 경고한다. ← 계획 2
- S-13 데스크톱 쪽 컨포먼스 테스트가 서버와 같은 픽스처를 읽고, 이것이 저장소의 첫 Node 테스트 러너다. ← 계획 3
- S-14 설정에 동기화 탭이 생기고 en/ko 문구가 함께 들어간다. README에 "동기화 (beta, 선택)" 절이 생긴다. ← 계획 4
- S-15 데스크톱 태그가 첫 rc부터 `desktop-vX.Y.Z`가 되고, 기존 1.2.6 설치가 그 릴리스로 계속 업데이트된다. ← 계획 5
- S-16 Changeset 버전 PR이 병합되면 패키지가 npm에 바로 공개되지 않고 staging에 올라가며, 사용자가 승인해야 공개된다. AGENTS.md의 해당 문장이 실제 동작과 맞게 된다. ← 계획 6
- S-17 stable 1.3.0은 서버 `0.1.0-rc.1` 대상 테스트가 Linux와 Windows에서 통과한 뒤에만 낸다. macOS는 자동 게이트와 CI 빌드까지. ← 제약 6, 질의서 2

## 설계

### 데이터 (main)
- **0007** `sync_server`(기기 로컬, sync 컬럼 없음, 한 행): `id`, `url`, `accountId`, `teamId`, `allowInsecureTls`(0/1), `enabled`(0/1), `lastPulledSeq`, `bootstrapCursor`(진행 중 스냅샷의 커서, 없으면 null), `bootstrappedAt`(첫 스냅샷 완료 시각 — 빈 서버면 `lastPulledSeq`가 0으로 남아 그것만으로는 완료를 알 수 없음), `backfilledAt`, `lastSyncedAt`, `lastError`, `lastErrorKind`. 토큰은 이 테이블에 두지 않는다.
- **0007** `sync_outbox`: `seq`(PK autoinc), `opId`, `tableName`, `rowId`, `op`(upsert|delete), `baseVersion`, `rowVersion`, `updatedAtMs`, `enqueuedAt`, `attempts`, `lastError`, UNIQUE(`tableName`,`rowId`).
- **0008** 트리거: 5개 테이블 × AFTER INSERT / AFTER UPDATE, `WHEN NEW.lastWriterId IS NOT 'server'`. 같은 행을 여러 번 고치면 `ON CONFLICT(tableName,rowId) DO UPDATE`로 합치되 `opId`와 `baseVersion`은 유지한다(서버 충돌 판정은 첫 미동기 변경 직전 버전 기준). drizzle-kit이 트리거를 만들지 않으므로 SQL은 손으로 쓰고 journal에 올린다. 부팅 때 `ensureOutboxTriggers()`가 `CREATE TRIGGER IF NOT EXISTS`로 보정한다.
- 이미 커밋된 0000~0006은 건드리지 않는다.
- drizzle snapshot이 0001까지만 있어서 `db:generate`가 0002~0006을 다시 만들어 내는 상태였다(구현 중 확인). 0007에서 현재 스키마 전체의 `0007_snapshot.json`을 두고, SQL은 새 테이블만 손으로 쓴다. 이후 `db:generate`는 변경이 없으면 아무것도 만들지 않는다.

### 프로토콜 계약 (common)
- `src/common/sync-protocol.ts`: 서버 `protocol` 패키지와 같은 타입(Commit, PushRequest/Response, Conflict, Change, PullRequest/Response, BootstrapRequest/Response, LoginRequest/Response, WhoAmIResponse), `PROTOCOL_VERSION = 1`, 헤더 이름, `encodeRow(table, row)` / `decodeRow(table, wire)`, `wins(a, b)`.
  - 와이어 규칙(픽스처 12번): 초 단위 날짜 컬럼 → ms, `deletedAt`은 원래 ms, boolean → 0/1, DTO `recurrenceRule` ↔ 컬럼 `recurringRule`.
  - `wins`: `updatedAtMs`가 크면 이김, 같으면 `lastWriterId` 사전식으로 큰 쪽(`'server'` > `'device:…'`).
  - 다른 파일을 import하지 않는다(테스트 러너가 번들 없이 바로 읽게).
- 픽스처의 row codec은 `schedule`만 있다. 나머지 4개 테이블은 데스크톱 단위 테스트로 덮는다. 픽스처는 읽기만 한다(intent 영향 범위).

### 전송 · 인증 (main)
- `sync/transport.ts`: Electron `net` + 동기화 전용 `session`(`session.fromPartition('croffle-sync')`). 모든 요청에 `Authorization: Bearer`, `X-Croffle-Sync-Version: 1`, `X-Croffle-Client: <clientId>`. 타임아웃, 5xx · 네트워크 오류는 지수 백오프로 재시도, 4xx는 재시도하지 않음. 오류는 `kind`(auth | offline | tls | server | protocol | clock)로 분류하고 메시지에 토큰을 넣지 않는다. 요청마다 `[Sync] <METHOD> <path> <status>` 한 줄만 로그에 남긴다(헤더 · 본문 없음).
- TLS: 기본 strict. `allowInsecureTls`가 켜졌을 때만 전용 session의 `setCertificateVerifyProc`에서 **등록된 서버 호스트**에 한해 통과. 기본 session(확장 `http.*`)은 건드리지 않는다.
- 연결 확인 `checkServer()`: `GET /health` → `protocol.min ≤ 1 ≤ max`가 아니면 `protocol` 오류(어느 쪽을 올려야 하는지 포함). `time`과 로컬 시계 차가 임계값(기본 5분)을 넘으면 `clock` 경고(동기화는 계속).
- `sync/auth.ts`: ID/PW → `POST /auth/login`(`clientId`, `deviceName`), 401 `otp_required`면 OTP를 받아 재시도. 토큰 붙여넣기 → `POST /auth/whoami`로 확인한 뒤에만 저장. 로그아웃 · 서버 변경 → 토큰 삭제, `lastPulledSeq`/`bootstrapCursor`/`backfilledAt` 초기화. outbox는 지우지 않는다(로컬 변경을 잃지 않게).
- `sync-config-service`: `sync_server` CRUD + 토큰 `safeStorage`(`userData/sync-token.bin`). `isEncryptionAvailable()`이 false면 메모리에만 두고 상태에 `tokenVolatile: true`.

### 엔진 (main)
- **push** `sync/push.ts`: outbox를 seq 순으로 100개씩 → Commit(`version = rowVersion`, `baseVersion`, `row = encodeRow(현재 행)`; 삭제는 tombstone 행). `accepted` → `DELETE WHERE opId = ? AND rowVersion = ?`(그 사이 다시 고친 행은 남음). `conflicts` → `wins(로컬, 서버)`: 서버가 이기면 서버 행을 `lastWriterId='server'`로 적용하고 outbox 행 삭제, 로컬이 이기면 `baseVersion = serverVersion`, `version = max(local, serverVersion) + 1`로 올려 다음 push에 새 opId로 보낸다. 400 `invalid_commit`은 그 항목만 `lastError`에 기록하고 건너뛴다.
- **pull** `sync/pull.ts`: `lastPulledSeq`부터 `hasMore`가 false일 때까지. 각 change를 `decodeRow` 후 FK 순서(tag, schedule → schedule_tags · schedule_reminder · schedule_exception)로 upsert, `lastWriterId='server'`. 로컬에 미동기 변경이 있는 행은 `wins`로 판정해 로컬이 이기면 건너뛴다. 동명 태그는 애플리케이션 검증을 거치지 않고 행 그대로 upsert한다. FK가 아직 없는 자식 행은 다음 페이지 뒤 재시도.
- **bootstrap**: `lastPulledSeq = 0`이고 bootstrap을 마친 적이 없을 때, 또는 pull이 410 `cursor_too_old`일 때. 페이지마다 커서를 `bootstrapCursor`에 저장(중단 후 재개), 마지막 페이지 뒤 `lastPulledSeq = serverSeq`. 적용 규칙은 pull과 같다.
- **backfill**: bootstrap이 끝나고 `backfilledAt`이 없으면, 5개 테이블에서 `lastWriterId IS NOT 'server'`인 행 전체를 `INSERT … SELECT`로 outbox에 적재(`baseVersion = 0`, 이미 있으면 그대로), `backfilledAt` 기록. 서버 행과 id가 겹치면 push의 충돌 규칙으로 정리된다.
- **오케스트레이터** `sync/sync-service.ts`: 순서 `checkServer → (bootstrap) → (backfill) → push → pull`. 앱 시작 · 로그인 · 주기(5분) · 로컬 변경 디바운스(5초) · 수동 "지금 동기화". 동시 실행 1개. 실패 시 백오프(최대 30분). 상태(`enabled`, `running`, `pending`, `lastSyncedAt`, `lastErrorKind`, `tokenVolatile`, `clockSkewMs`)를 renderer로 브로드캐스트. pull로 바뀐 행이 있으면 기존 일정 변경 이벤트로 캘린더를 갱신.

### 브리지 · UI
- 내부 API `croffleInternal.sync`: `getStatus`, `onStatus`, `setServer`, `setAllowInsecureTls`, `signInWithPassword`, `signInWithToken`, `signOut`, `setEnabled`, `syncNow`. IPC 핸들러는 기존 `validateSender`를 쓴다. `packages/types`에는 없다.
- `stores/sync-store.ts` + `components/settings/sync-panel.vue` + `settings-modal.vue` 탭 추가. 내용: beta 안내, 서버 URL, 인증 2종(OTP 필드는 `otp_required` 때만), 사용 스위치, 고급(TLS 허용 + 경고), 상태(마지막 동기화 · 대기 N건 · 오류 종류별 안내 · 시계 경고 · 지금 동기화), 토큰 휘발 경고. i18n은 `settings.tabs.sync`, `settings.sync.*`를 en/ko 동시.

### 테스트 러너
- Node 24의 `node --test`가 `.ts`를 바로 실행한다(2026-11-11 실측). 새 의존성 없음. `apps/desktop`에 `test` 스크립트, 테스트는 `src/common/*.test.ts`. 픽스처 경로는 저장소 상대. 루트 `pnpm -r test`와 CI desktop 스코프에 테스트 단계 추가. `.ts` 확장자 import를 위해 tsconfig에 `allowImportingTsExtensions`(noEmit이라 가능)를 넣는다.

### 릴리스 · 발행
- 데스크톱 태그: `release.yml`의 태그를 `desktop-v${version}`으로, `electron-builder.yml`의 `publish.tagNamePrefix: desktop-v`. 접두어 태그에서는 electron-updater가 릴리스 노트를 계산하지 못하므로(`semver.valid` 실패), 빌드 때 `releaseInfo.releaseNotes`를 `latest.yml`에 넣고 `update-modal.vue`는 노트가 없으면 릴리스 페이지 링크를 보여 준다. 릴리스 제목 관례는 유지한다.
- npm staging: `publish-packages`의 `changeset publish`를 스크립트로 바꾼다. 패키지마다 npm에 없는 버전이면 `pnpm pack`(workspace 의존성 치환) → 풀어서 `npm stage publish --provenance`(npm CLI ≥ 11.15.0, OIDC). 출력된 stage id를 job summary에 남긴다. git 태그(`@croffledev/<pkg>@<version>`)는 stage 때 만들지 않고, 다음 실행에서 "npm에 공개되었는데 태그가 없는 버전"을 찾아 만든다. npmjs.com의 trusted publishing 설정을 "stage만 허용"으로 바꾸는 일은 사용자가 한다(2FA).
- AGENTS.md: Versioning · CI/CD · Changeset 규칙 문장을 바뀐 동작에 맞춘다. 템플릿 `.github/release-notes.md`도.

## 작업 단위
각 단위는 따로 되돌릴 수 있고, 그 커밋은 게이트를 통과한다. 재료가 된 옛 작업 번호를 괄호에 적었다.

- [x] W-1 (96b2d9f) `sync_server` · `sync_outbox` 스키마 + 마이그레이션 0007 (000-1) — S-1, S-2
- [x] W-2 (d71adab) outbox 트리거 0008 + `ensureOutboxTriggers()` + `sync/outbox.ts`(peek · count · ack · fail) + `SERVER_WRITER_ID` (000-2) — S-6
- [x] W-3 (965a6d3) `sync-config-service`: 서버 설정 CRUD + `safeStorage` 토큰 + TLS 플래그 (000-3) — S-7, S-8
- [ ] W-4 프로토콜 계약 `common/sync-protocol.ts` (타입 · encode/decode · wins) (000-5) — S-5
- [ ] W-5 테스트 러너 + 픽스처 리더 + 나머지 4개 테이블 codec 단위 테스트 + CI 테스트 단계 + `.agents/harness/gates.yaml`에 `test` 게이트(`pnpm -r test`) 추가 (000-15) — S-13. 의존: W-4. 게이트 추가는 하네스 손 수정이며 사용자가 이번 한 번만 허락함(intent `waivers:`)
- [ ] W-6 전송 계층: 전용 session · 헤더 · 재시도/백오프 · TLS 호스트 한정 · `checkServer()`(프로토콜 범위 · 시계) (000-6) — S-8, S-12. 의존: W-3, W-4
- [ ] W-7 인증: 로그인(+OTP) · 토큰 붙여넣기(whoami) · 로그아웃/서버 변경 초기화 (000-7) — S-9, S-7. 의존: W-3, W-6
- [ ] W-8 push 엔진 (000-8) — S-1, S-3. 의존: W-2, W-6
- [ ] W-9 pull 엔진 + bootstrap + 410 복구 (000-9 + 질의서 1) — S-1, S-10. 의존: W-2, W-6
- [ ] W-10 오케스트레이터 + backfill + 상태 브로드캐스트 (000-10 + 질의서 3) — S-3, S-11, S-2. 의존: W-7, W-8, W-9
- [ ] W-11 내부 `sync.*` 브리지: common API 타입 + `ipc/sync.handler.ts` + `preload/api/sync.ts` (000-4) — S-4. 의존: W-10
- [ ] W-12 검증 하네스: 장애 주입 스텁 서버(TLS · OTP · 충돌 · FK 역순 · 410) + 서버 `0.1.0-rc.1` 컨테이너 대상 실행 스크립트 (000-11) — S-17. 의존: W-10
- [ ] W-13 설정 → 동기화 탭 + `sync-store` + i18n en/ko (000-12) — S-14. 의존: W-11
- [ ] W-14 README / README.ko "동기화 (beta, 선택)" 절 (000-13) — S-14
- [ ] W-15 데스크톱 태그 `desktop-v*` + 업데이트 창 릴리스 노트 보완 + AGENTS.md · 노트 규약 갱신 (000-14) — S-15. 첫 rc 전에 병합
- [ ] W-16 npm staging 발행: 발행 스크립트 + `publish-packages.yml` + AGENTS.md Changeset 규칙 (계획 6) — S-16

옛 계획의 W-4(브리지)를 엔진 뒤(W-11)로 옮겼다. 브리지가 감싸는 함수(로그인, 동기화)가 먼저 있어야 브리지 단위가 한 번에 완성되고 따로 되돌릴 수 있기 때문이다.

## 검증 기준
qa-designer(google gemini-4-argon-high)가 작성하고 PM이 검토 · 수정했습니다. PM 수정: V-1에 `pnpm -r test` 추가, V-2를 데스크톱 러너가 실제로 읽는 범위(`rowCodec`)로 좁힘, V-6의 "네트워크 요청 0건"을 전송 계층 요청 로그로 관찰하게 함, V-15의 대기 3건이 합쳐지지 않게 서로 다른 일정으로 바꿈, V-19 · V-20에 판정 시점을 적음.

- V-1 **프로젝트 정적 게이트 · 빌드 · 서버 호환성**
  - 검증 대상: S-17, W-1~W-16
  - 실행 방법: 저장소 루트에서 `pnpm typecheck && pnpm lint && pnpm exec oxfmt --check && pnpm -r test && pnpm build && pnpm --filter @croffledev/croffle-cli build && pnpm --dir apps/server/web build && make -C apps/server check` 실행
  - 기대 결과: 8개 명령 모두 종료 코드 0으로 통과하고, `apps/desktop/out/main/index.js`, `apps/desktop/out/preload/index.js`, `apps/desktop/out/renderer/index.html` 산출물이 생성됨
  - 실패 판정: 어느 한 단계라도 종료 코드 ≠ 0이거나, 타입·린트·포맷 오류가 출력되거나, 빌드 산출물이 누락됨
  - 증적 경로: `.loctx/reports/1.3.0/v01-gates.log`

- V-2 **데스크톱 프로토콜 컨포먼스 · 5개 테이블 코덱 · LWW 단위 테스트**
  - 검증 대상: S-5, S-13, W-4, W-5
  - 실행 방법: `pnpm -r test` 및 `node --test apps/desktop/src/common/*.test.ts` 실행, `.github/workflows/ci.yml`에 데스크톱 테스트 스텝 존재 확인, `git status --porcelain packages/sync-fixtures apps/server` 실행
  - 기대 결과: 러너가 `packages/sync-fixtures/scenarios/*.json` 12개를 모두 읽고(파일 수 · `protocol: 1` 확인) `rowCodec` 블록(`12-row-codec.json`의 codec 2건 · lww 3건)을 통과하며, 나머지 테이블을 덮는 단위 테스트까지 5개 테이블(`schedule`, `tag`, `schedule_tags`, `schedule_reminder`, `schedule_exception`)의 `encodeRow`/`decodeRow`(초↔ms 변환, `deletedAt` ms 유지, boolean↔0/1, `recurringRule`↔`recurrenceRule`) 및 `wins`(`updatedAtMs` 비교, 동률 시 `'server'` > `'device:…'` 사전식 비교) 테스트가 fail 0·skip 0으로 통과함. `packages/sync-fixtures`와 `apps/server`의 수정·추가 파일 0개
  - 실패 판정: 테스트 1건이라도 실패·누락·스킵되거나, CI 워크플로에 테스트 단계가 없거나, 픽스처·서버 파일이 변경됨
  - 증적 경로: `.loctx/reports/1.3.0/v02-conformance-test.log`

- V-3 **확장 공개 API(`packages/types`) 불변 및 내부 브리지 격리 (음성 확인)**
  - 검증 대상: S-4, W-11
  - 실행 방법:
    1) `git diff master...HEAD -- packages/types/`
    2) `grep -rnE "sync|outbox|serverSeq|opId|bootstrapCursor" packages/types/`
    3) `pnpm build` 후 `apps/desktop/out/preload/index.js` 정적 검사 및 `[수동 · Linux]` `pnpm dev` DevTools 콘솔에서 `window.croffle.sync`와 내부 브리지 노출 범위 확인
  - 기대 결과: 1) diff 출력 0줄, 2) grep 매치 0건(종료 코드 1), 3) `window.croffle.sync`는 `undefined`이고 `croffleInternal.sync`(`getStatus`, `onStatus`, `setServer`, `setAllowInsecureTls`, `signInWithPassword`, `signInWithToken`, `signOut`, `setEnabled`, `syncNow`)만 앱 렌더러가 일회성으로 가져가며 비허가 발신자 IPC 호출은 `validateSender`가 거부함
  - 실패 판정: `packages/types`에 변경이나 동기화 관련 식별자가 1건이라도 있거나, `window.croffle.sync`가 접근 가능함
  - 증적 경로: `.loctx/reports/1.3.0/v03-public-api-isolation.log`

- V-4 **DB 마이그레이션 0007·0008 무결성 · 기존 1.2.6 DB 업그레이드 · 트리거 부팅 보정**
  - 검증 대상: S-1, S-2, S-6, W-1, W-2
  - 실행 방법:
    1) `git diff master...HEAD -- apps/desktop/drizzle/0000* apps/desktop/drizzle/0001* apps/desktop/drizzle/0002* apps/desktop/drizzle/0003* apps/desktop/drizzle/0004* apps/desktop/drizzle/0005* apps/desktop/drizzle/0006*`
    2) `pnpm --filter @croffledev/desktop db:generate && git status --porcelain apps/desktop/drizzle/`
    3) `[수동 · Linux/Windows]` 1.2.6 DB 백업본을 `apps/desktop/dev/croffle.db`로 복사 후 `pnpm dev` 기동 → `sqlite3 apps/desktop/dev/croffle.db "PRAGMA table_info(sync_server); PRAGMA table_info(sync_outbox); SELECT count(*), group_concat(name) FROM sqlite_master WHERE type='trigger';"` 실행 → 트리거 1개를 `DROP TRIGGER`로 삭제한 뒤 앱 재기동 후 트리거 개수 재조회
  - 기대 결과: 1) 0000~0006 diff 0줄, 2) `db:generate`가 추가 파일·변경을 만들지 않음(0007에서 snapshot을 보강한 뒤 기준. 그 전 master는 snapshot이 0001까지라 깨끗하지 않았음), 3) 1.2.6의 기존 일정·태그 행 수가 100% 보존되고 `sync_server`(토큰 컬럼 없음)와 `sync_outbox`(`UNIQUE(tableName,rowId)`)가 생성되며, 5개 테이블×INSERT/UPDATE 트리거 총 10개가 존재하고 강제 삭제한 트리거가 재기동 시 `ensureOutboxTriggers()`로 복구되어 다시 10개가 됨
  - 실패 판정: 0000~0006이 수정되었거나, 스키마와 journal이 어긋나거나, `sync_server`에 토큰 저장 컬럼이 있거나, 트리거가 10개가 아니거나, 재기동 후 삭제된 트리거가 복구되지 않음
  - 증적 경로: `.loctx/reports/1.3.0/v04-db-migration-triggers.log`

- V-5 **outbox 트리거 적재 · 다중 수정 병합(coalesce) · 서버 행 억제 (음성 확인)**
  - 검증 대상: S-6, W-2
  - 실행 방법: `[수동 · Linux]` 초기화된 `apps/desktop/dev/croffle.db`로 `pnpm dev` 기동 후:
    1) 일정 1건 생성(`version=1`) 후 같은 일정을 연속 3회 수정(`version=4`), 태그 1건 생성·연결, 리마인더 1건 추가, 반복 일정 "이 일정만" 드래그(예외 1건 생성), 일정 삭제(tombstone) 수행 후 `sqlite3 apps/desktop/dev/croffle.db "SELECT tableName, rowId, op, baseVersion, rowVersion FROM sync_outbox ORDER BY seq;"` 실행
    2) `sqlite3 apps/desktop/dev/croffle.db`로 `lastWriterId = 'server'`인 `schedule` 행 INSERT/UPDATE 및 `schedule_external_link` 행 INSERT 후 `sync_outbox` 재조회
  - 기대 결과: 1) 5개 테이블(`schedule`, `tag`, `schedule_tags`, `schedule_reminder`, `schedule_exception`) 변경이 모두 적재되고, 3회 연속 수정한 일정은 `sync_outbox`에 단일 행(1건)으로 합쳐지며 `opId`와 첫 변경 직전 `baseVersion`(`0`)은 그대로 유지된 채 `rowVersion`만 `4`로 갱신됨. 2) `lastWriterId = 'server'`로 기록된 행과 `schedule_external_link` 변경은 `sync_outbox`에 0건 적재됨
  - 실패 판정: 같은 `(tableName, rowId)`가 중복 행으로 쌓이거나, 재수정 시 `baseVersion`이 덮어써지거나, `lastWriterId = 'server'` 또는 `schedule_external_link` 행이 outbox에 적재됨
  - 증적 경로: `.loctx/reports/1.3.0/v05-outbox-triggers.log`

- V-6 **서버 미등록 상태에서 네트워크 호출 0건 (음성 확인)**
  - 검증 대상: S-2, W-1, W-10
  - 실행 방법: `[수동 · Linux/Windows]` 서버를 등록하지 않은 상태에서 `pnpm dev`로 앱을 띄우고 일정·태그·리마인더를 생성·수정·삭제한 뒤 주기 타이머(5분) 이상인 6분 동안 대기 → `sqlite3 apps/desktop/dev/croffle.db "SELECT count(*) FROM sync_server; SELECT count(*) FROM sync_outbox;"` 및 `apps/desktop/dev/logs/main.log`, 전용 세션(`croffle-sync`) 요청 로그 확인
  - 기대 결과: `sync_server`는 0건, `sync_outbox`는 로컬 변경 수만큼 적재되어 있으면서 전송 계층의 요청 로그(`[Sync] <METHOD> <path> <status>`)가 `main.log`에 0줄이고, `main.log`에 동기화 실패·재시도 로그가 0건이며 앱 동작은 1.2.6과 동일함
  - 실패 판정: 서버 미설정 상태에서 `/health`, `/auth/*`, `/sync/*` 등 네트워크 요청이 1회라도 시도되거나 오류 로그/알림이 발생함
  - 증적 경로: `.loctx/reports/1.3.0/v06-no-server-no-network.log`

- V-7 **토큰 `safeStorage` 보관 · 키링 부재 시 휘발 폴백 · 로그/디스크 평문 노출 0건 (음성 확인)**
  - 검증 대상: S-7, W-3, W-7
  - 실행 방법:
    1) `[수동 · Linux/Windows]` 정상 키링 환경에서 ID/PW 로그인(성공·실패), OTP 로그인, 토큰 붙여넣기 수행 후 `grep -rnaiE "<발급토큰>|<입력비밀번호>|<입력OTP>|bearer |password|otp" apps/desktop/dev/croffle.db apps/desktop/dev/settings.json apps/desktop/dev/logs/main.log` 실행 및 `apps/desktop/dev/sync-token.bin` 확인
    2) `[수동 · Linux]` 키링 비활성 환경(`--password-store=basic` 등으로 `safeStorage.isEncryptionAvailable() === false`)에서 `pnpm dev` 기동 후 로그인 → `sync-token.bin` 파일 유무, `getStatus().tokenVolatile`, UI 경고 문구 확인 → 앱 재시작 후 로그인 상태 확인
  - 기대 결과: 1) 정상 환경에서는 `sync-token.bin`에만 암호문 바이트로 저장되어 재시작 후에도 인증이 유지되고, `croffle.db`·`settings.json`·`main.log`에서 토큰·비밀번호·OTP 평문 및 `Authorization: Bearer` 헤더 값 매치가 0건임. 2) 키링 비활성 시 디스크에 토큰 파일이 생성되지 않고 `tokenVolatile: true` 및 UI 경고가 표시되며 앱 재시작 시 토큰이 소멸됨
  - 실패 판정: 로그·DB·설정 파일 어디에든 토큰·비밀번호·OTP 평문이 1건이라도 발견되거나, 키링이 없을 때 평문 파일로 디스크에 저장됨
  - 증적 경로: `.loctx/reports/1.3.0/v07-token-safestorage-and-logs.log`

- V-8 **전송 계층: `/health` 프로토콜 범위 검사 · 시계 오차 경고 · 필수 헤더 · 재시도/백오프**
  - 검증 대상: S-12, W-6, W-12
  - 실행 방법: 검증 하네스 스텁(`W-12`)을 이용해:
    1) `/health`가 `protocol: { min: 2, max: 2 }`를 반환할 때 연결/동기화 실행
    2) `/health`가 로컬 시계와 10분(`600000ms`) 이상 차이 나는 `time`을 반환할 때 동기화 실행
    3) 503 2회 후 200을 반환하는 경우와 400/401을 반환하는 경우의 요청 횟수·간격 및 수신 헤더(`Authorization`, `X-Croffle-Sync-Version`, `X-Croffle-Client`) 확인
  - 기대 결과: 1) `lastErrorKind = 'protocol'`과 업그레이드 안내가 설정 상태에 기록되고 `/sync/*` 요청은 보내지 않음. 2) `clockSkewMs` 경고가 상태와 UI에 표시되면서 동기화는 정상 완료됨. 3) 모든 요청에 `X-Croffle-Sync-Version: 1`, `X-Croffle-Client: <clientId>`, `Authorization: Bearer <token>`이 실리고, 503은 지수 백오프로 재시도해 성공하며 400/401은 재시도 없이 즉시 1회로 끝남
  - 실패 판정: 비호환 프로토콜에서 push/pull이 실행되거나, 시계 오차 경고가 누락/동기화가 중단되거나, 4xx 응답을 재시도하거나, 필수 헤더가 빠짐
  - 증적 경로: `.loctx/reports/1.3.0/v08-transport-health-clock.log`

- V-9 **TLS strict 기본값 · 동기화 호스트 한정 허용 · 타 호스트 및 확장 `http.*` 무영향 (음성 확인)**
  - 검증 대상: S-8, W-3, W-6, W-12
  - 실행 방법: 검증 하네스로 서로 다른 두 호스트의 자기서명 HTTPS 서버(`https://localhost:<portA>`, `https://127.0.0.1:<portB>`)를 띄우고 동기화 서버를 `https://localhost:<portA>`로 등록한 뒤:
    1) 기본값(`allowInsecureTls = false`)에서 연결 시도
    2) `allowInsecureTls = true`로 켠 뒤 `https://localhost:<portA>` 연결 시도
    3) `allowInsecureTls = true` 상태에서 동기화 세션으로 `https://127.0.0.1:<portB>`에 요청하고, 동시에 확장용 `main/http.ts`(`window.croffle.http.get`)로 `https://localhost:<portA>/health` 호출
  - 기대 결과: 1) 기본값에서는 `lastErrorKind = 'tls'`로 거부됨. 2) `allowInsecureTls = true`일 때 등록된 `localhost:<portA>`는 정상 연결됨. 3) 등록되지 않은 다른 자기서명 호스트(`127.0.0.1:<portB>`)는 여전히 인증서 오류로 거부되고, 확장 API `croffle.http.get('https://localhost:<portA>/health')`도 `{ ok: false, status: 0 }`으로 여전히 거부됨
  - 실패 판정: 기본값에서 자기서명 인증서가 통과하거나, `allowInsecureTls = true`일 때 타 호스트 또는 확장 `croffle.http.*`까지 자기서명 인증서를 허용함
  - 증적 경로: `.loctx/reports/1.3.0/v09-tls-isolation.log`

- V-10 **인증 흐름: ID/PW 로그인 · OTP 재시도 · 토큰 붙여넣기(`whoami`) · 로그아웃/서버 변경 초기화**
  - 검증 대상: S-9, S-7, W-7, W-12
  - 실행 방법: 검증 하네스(`FORCE_OTP=1` 포함)에서:
    1) 오타 비밀번호로 로그인 및 유효하지 않은 토큰 붙여넣기 시도
    2) 일반 ID/PW 로그인 및 `FORCE_OTP=1`에서 1차 `401 otp_required` 수신 → OTP 입력란 활성화 → OTP 포함 2차 로그인 수행
    3) 유효한 토큰 붙여넣기(`POST /auth/whoami` 검증) 수행
    4) `sync_outbox`에 미전송 변경 2건을 남긴 상태에서 `signOut()` 및 `setServer(다른 URL)` 실행 후 `sync_server`와 `sync_outbox` 조회
  - 기대 결과: 1) 실패 시 기존 토큰과 `sync_server` 행이 변경되지 않고 `lastErrorKind = 'auth'`가 됨. 2~3) 두 방식 모두 내부적으로 단일 토큰으로 저장되고 `accountId`·`teamId`가 기록됨. 4) 로그아웃 및 서버 변경 시 저장된 토큰이 삭제되고 `lastPulledSeq = 0`, `bootstrapCursor = NULL`, `backfilledAt = NULL`로 초기화되나 `sync_outbox`의 미전송 2건은 그대로 유지됨
  - 실패 판정: `whoami` 확인 없이 잘못된 토큰이 저장되거나, `otp_required` 흐름이 동작하지 않거나, 로그아웃/서버 변경 시 `sync_outbox`가 삭제되어 로컬 변경을 잃음
  - 증적 경로: `.loctx/reports/1.3.0/v10-auth-flows.log`

- V-11 **Push 엔진: 100건 배치 · in-flight 재편집 보존 · LWW 충돌 해결 · `invalid_commit` 격리**
  - 검증 대상: S-1, S-3, W-8, W-12
  - 실행 방법: 검증 하네스를 대상으로:
    1) `sync_outbox`에 150건 적재 후 push 실행(배치 분할 확인)
    2) push 요청 전송 중 같은 행을 로컬에서 한 번 더 수정(`rowVersion` 증가)한 뒤 응답 ack 처리 후 `sync_outbox` 확인
    3) 충돌 시나리오 2종 실행: (a) 서버 `updatedAtMs`가 더 큰 충돌, (b) 로컬 `updatedAtMs`가 더 큰 충돌
    4) 400 `invalid_commit` 응답을 유발하는 항목 1건과 정상 항목 1건을 함께 보유한 상태에서 push 실행
  - 기대 결과: 1) 100건 + 50건 두 번의 요청으로 모두 accepted되고 outbox가 0건이 됨. 2) `DELETE WHERE opId = ? AND rowVersion = ?` 조건에 의해 in-flight 중 재편집된 행은 지워지지 않고 남아 다음 push에서 새 `opId`로 전송됨. 3a) 서버 승리 시 로컬 행이 서버 값(`lastWriterId='server'`)으로 덮어써지고 outbox에서 삭제됨. 3b) 로컬 승리 시 `baseVersion = serverVersion`, `rowVersion = max(local, serverVersion) + 1`로 올려 다음 push에 새 `opId`로 accepted됨. 4) `invalid_commit` 항목만 `lastError`가 기록되고 정상 항목은 push됨
  - 실패 판정: in-flight 재편집 변경이 사라지거나, 로컬 승리 재전송 시 `version <= baseVersion`으로 실패하거나, 서버 승리 적용이 다시 outbox에 적재됨
  - 증적 경로: `.loctx/reports/1.3.0/v11-push-engine.log`

- V-12 **Pull 엔진: 페이지네이션 · 핑퐁 없음(음성 확인) · 로컬 미동기 보호 · FK 역순 · 동명 태그**
  - 검증 대상: S-1, S-6, W-9, W-12
  - 실행 방법: 검증 하네스(`CHILD_FIRST=1` 포함)를 대상으로:
    1) `sync_outbox`가 0건인 상태에서 5개 테이블 다중 페이지(`hasMore: true` → `false`) 변경분을 pull → 직후 `SELECT count(*) FROM sync_outbox` 조회 및 동일 커서로 즉시 2차 pull 실행
    2) 로컬 `sync_outbox`에 미동기 수정(`updatedAtMs`가 서버보다 큼)이 있는 행에 대해 서버 변경분 pull 실행
    3) `CHILD_FIRST=1`(자식 테이블 `schedule_tags`·`schedule_reminder`·`schedule_exception`이 부모 `schedule`·`tag`보다 앞 페이지에 오는 응답) pull 실행
    4) 로컬에 이미 존재하는 태그 이름과 동일한 이름(다른 `id`)의 서버 태그를 pull 실행
  - 기대 결과: 1) 모든 페이지 적용 후 `lastPulledSeq`가 최종 `serverSeq`로 갱신되고 **pull 직후 `sync_outbox` 행 수는 0건**(핑퐁 없음)이며 2차 pull은 변경 0건을 반환함. 2) 로컬 미동기 변경이 `wins`에서 이기면 서버 변경으로 덮어쓰지 않고 로컬 값과 outbox 행을 유지함. 3) FK 역순에서도 외래키 오류 없이 부모 페이지 적용 뒤 보류된 자식 행들이 모두 반영됨. 4) 동명 태그도 애플리케이션 중복 검사를 우회하여 정상 upsert됨
  - 실패 판정: pull 적용 후 `sync_outbox`에 1건이라도 행이 생기거나, 더 최신인 로컬 미동기 변경이 덮어써지거나, FK 역순·동명 태그에서 오류가 발생함
  - 증적 경로: `.loctx/reports/1.3.0/v12-pull-engine-no-pingpong.log`

- V-13 **Bootstrap 스냅샷 · 중단 후 커서 재개 · 410 `cursor_too_old` 자동 복구**
  - 검증 대상: S-10, W-9, W-12
  - 실행 방법: 검증 하네스에서:
    1) `lastPulledSeq = 0`인 초기 상태에서 2페이지 이상의 스냅샷을 가진 서버로 동기화 시작 → 1페이지 적용 직후 강제 중단하여 `sync_server.bootstrapCursor` 확인 → 다시 동기화를 실행해 완료
    2) `lastPulledSeq > 0`인 기기에 대해 서버 보존 기간 만료(`410 cursor_too_old` / `setPurgedThroughSeq`)를 주입한 뒤 `syncNow` 실행
  - 기대 결과: 1) 1페이지 후 `bootstrapCursor`가 DB에 저장되어 재실행 시 저장된 커서부터 이어서 2페이지를 받고, 마지막 페이지 완료 후 `bootstrapCursor = NULL`, `lastPulledSeq = serverSeq`가 됨. 2) `/sync/pull`의 410 응답을 받으면 자동으로 `/sync/bootstrap`으로 전환해 최신 스냅샷을 모두 적용하고 `lastPulledSeq`를 복구함
  - 실패 판정: 중단 후 처음부터 다시 받거나 완료 후 `bootstrapCursor`가 비워지지 않거나, 410 수신 시 bootstrap으로 복구하지 못하고 오류 상태에 머묾
  - 증적 경로: `.loctx/reports/1.3.0/v13-bootstrap-and-410.log`

- V-14 **첫 동기화 기존 로컬 데이터 일괄 적재(Backfill)**
  - 검증 대상: S-11, W-10
  - 실행 방법: `[수동 · Linux]` 서버 연결 전에 생성되어 `sync_outbox`에는 없는 기존 5개 테이블 데이터(`schedule`, `tag`, `schedule_tags`, `schedule_reminder`, `schedule_exception` 각 1건 이상, `lastWriterId != 'server'`)가 들어 있는 DB로 빈 서버에 최초 로그인 후 동기화 실행 → `sqlite3 apps/desktop/dev/croffle.db "SELECT backfilledAt FROM sync_server; SELECT count(*) FROM sync_outbox;"` 및 서버 수신 건수 확인 → 한 번 더 `syncNow` 실행
  - 기대 결과: 첫 동기화의 `bootstrap` 직후 `backfilledAt`이 null일 때 기존 로컬 행 전체가 `baseVersion = 0`으로 `sync_outbox`에 적재되어 첫 `push`로 서버에 모두 업로드되고, `sync_server.backfilledAt`에 타임스탬프가 기록되며, 두 번째 `syncNow`에서는 backfill이 재실행되지 않고 전송 건수가 0건임
  - 실패 판정: 트리거 도입 전/연결 전 기존 데이터가 서버로 올라가지 않거나, `lastWriterId = 'server'` 행까지 적재되거나, 매 동기화마다 중복 backfill이 일어남
  - 증적 경로: `.loctx/reports/1.3.0/v14-first-sync-backfill.log`

- V-15 **오프라인 동작 · 로컬 변경 디바운스 · 백오프 · 온라인 복귀 시 자동 동기화**
  - 검증 대상: S-3, W-10
  - 실행 방법: `[수동 · Linux/Windows]` 서버와 연결된 상태에서:
    1) 온라인 상태로 일정을 연속 수정하여 5초 디바운스 후 1회 자동 동기화되는지 확인
    2) 서버를 중지한 뒤 서로 다른 일정 3건에 각각 생성 · 수정 · 삭제를 1번씩 수행하고 캘린더 UI 조작 및 설정 동기화 탭 상태 확인
    3) 서버를 다시 기동한 뒤 자동(또는 수동) 동기화 결과 확인
  - 기대 결과: 1) 연속 변경 후 약 5초 뒤 1회 동기화 실행됨. 2) 오프라인 중에도 UI 멈춤·저장 실패가 없고 상태는 `pending: 3`, `lastErrorKind = 'offline'`으로 표시되며 실패 시 백오프 간격이 늘어남. 3) 서버 복구 후 쌓인 3건이 모두 서버로 반영되고 `pending: 0`, `lastErrorKind: null`, `lastSyncedAt` 갱신 상태가 됨
  - 실패 판정: 오프라인일 때 일정 CRUD가 막히거나, 동시 동기화가 2개 이상 겹쳐 돌거나, 서버 복구 후에도 대기 항목이 전송되지 않음
  - 증적 경로: `.loctx/reports/1.3.0/v15-offline-recovery.log`

- V-16 **설정 → 동기화 탭 UI · 한/영 i18n 카탈로그 대칭 · README 영/한 문서화**
  - 검증 대상: S-14, W-13, W-14
  - 실행 방법:
    1) `node` 스크립트로 `apps/desktop/src/common/i18n/locales/en.json`과 `ko.json`의 `settings.tabs.sync` 및 `settings.sync.**` 키 경로 목록을 추출·비교
    2) `README.md`와 `README.ko.md`에서 "Sync (beta, optional)" / "동기화 (beta, 선택)" 절과 로컬 우선·셀프호스트 서버 안내 문구 확인
    3) `[수동 · Linux/Windows]` `pnpm dev`에서 설정 모달 → 5번째 동기화 탭을 열고 언어를 English / 한국어로 전환하며 모든 상태(미연결, ID/PW 로그인, OTP 입력, 토큰 붙여넣기, 대기 N건, 오류 종류별 안내, 시계 오차 경고, 토큰 휘발 경고, 고급 TLS 토글 경고, 지금 동기화 버튼) 확인
  - 기대 결과: 1) `en.json`과 `ko.json`의 동기화 키 집합이 100% 일치하고 빈 문자열이나 원시 키 노출이 0건임. 2) 영/한 README 모두에 동기화 절이 대칭으로 작성됨. 3) UI의 모든 컨트롤이 `sync-store` 및 `croffleInternal.sync`와 연결되어 정상 작동함
  - 실패 판정: en/ko 키 불일치·미번역 키 노출이 있거나, UI에서 OTP/TLS/오류 상태 표시가 누락되거나, 영/한 README 중 한쪽이 누락됨
  - 증적 경로: `.loctx/reports/1.3.0/v16-ui-i18n-readme.log`

- V-17 **실서버 `croffle-server:0.1.0-rc.1` 대상 두 기기 종단간 동기화 (Linux · Windows)**
  - 검증 대상: S-1, S-5, S-17, W-12
  - 실행 방법: `[수동 · Linux 및 Windows 각각 수행]`
    1) `docker run -d --name croffle-srv -p 8787:8787 -e CROFFLE_DB_PATH=/tmp/server.db ghcr.io/team-croffle/croffle-server:0.1.0-rc.1` 실행 후 `docker exec -i croffle-srv croffle-server user create alice --password-stdin` 및 `token issue alice --device-name "device-b"`로 계정·토큰 생성
    2) 기기 A(기본 프로필, `http://localhost:8787`, ID/PW 로그인)에서 기존 일정 backfill + 신규 일정·태그·일정-태그·리마인더·반복 예외 생성 후 동기화
    3) 기기 B(별도 `--user-data-dir` 프로필, 토큰 붙여넣기 로그인)에서 첫 동기화(`bootstrap`)로 5개 테이블 수신 확인
    4) 기기 A와 B에서 동일 일정을 서로 다른 제목으로 수정(`updatedAtMs` 큰 쪽 승리 확인)하고, 기기 B에서 일정 1건 삭제(tombstone) 후 양쪽 동기화 실행 → 두 DB의 5개 테이블 내용과 `sync_outbox` 건수 조회
  - 기대 결과: Linux와 Windows 모두에서 기기 A·B의 5개 테이블(`schedule`, `tag`, `schedule_tags`, `schedule_reminder`, `schedule_exception`) 행·버전·삭제 상태가 완전히 일치하고, 동기화 직후 양쪽 `sync_outbox`는 0건(핑퐁 없음)이며, `main.log`에 토큰·비밀번호 노출이 0건임(macOS는 CI 빌드 통과로 갈음하고 릴리스 노트에 실서버 검증 전임을 명시)
  - 실패 판정: Linux 또는 Windows 중 한 곳이라도 두 기기 데이터가 수렴하지 않거나, 5개 테이블 중 일부가 누락되거나, 충돌 후 버전 역전/오류가 발생함
  - 증적 경로: `.loctx/reports/1.3.0/v17-real-server-linux.log`, `.loctx/reports/1.3.0/v17-real-server-windows.log`

- V-18 **데스크톱 기존 기능 회귀 스모크**
  - 검증 대상: S-2, S-17, W-1~W-13
  - 실행 방법: `[수동 · Linux/Windows]` `pnpm dev`에서 월/주/일 뷰 전환, 일반·반복 일정 생성/수정/드래그/삭제, 태그 필터, 리마인더 알림, 일정 가져오기/내보내기, 설정(일반·캘린더·알림·확장 탭) 변경, 로컬 확장(`.zip`) 설치·실행·삭제를 차례로 수행
  - 기대 결과: 전 항목이 오류나 지연 없이 동작하고 DevTools 콘솔 및 `apps/desktop/dev/logs/main.log`에 미처리 예외가 0건임
  - 실패 판정: 기존 기능 중 하나라도 깨지거나 DB 트리거·전용 세션 도입으로 인한 부작용 오류가 발생함
  - 증적 경로: `.loctx/reports/1.3.0/v18-regression-smoke.log`

- V-19 **데스크톱 `desktop-v*` 태그 스킴 · 1.2.6 업데이트 호환 · 업데이트 모달 릴리스 노트**
  - 검증 대상: S-15, W-15
  - 실행 방법:
    1) `.github/workflows/release.yml`의 태그(`desktop-v${version}`), `apps/desktop/electron-builder.yml`의 `publish.tagNamePrefix: desktop-v`, `AGENTS.md` 및 `.github/release-notes.md` 문구 확인, `git tag -l "v1.2.*"`로 기존 태그 무변경 확인
    2) 패키징 산출물 `latest.yml` / `latest-linux.yml`에 `version`과 `releaseNotes`가 포함되는지 확인하고, `update-modal.vue`에서 `releaseNotes`가 있을 때 본문 렌더·없을 때 GitHub 릴리스 페이지 링크 폴백 동작 확인
    3) `[수동 · Linux/Windows, stable 게시 직후]` `desktop-v1.3.0` 정식 릴리스 게시 후 기존 `1.2.6` 설치본을 실행해 업데이트 감지와 모달 표시 확인
  - 판정 시점: 1)~2)는 rc 전 테스트에서, 3)은 stable 게시 직후에 판정합니다(rc는 `/releases/latest`에 잡히지 않아 1.2.6이 보지 못함).
  - 기대 결과: 1) 새 태그는 `desktop-v1.3.0-rc.1` / `desktop-v1.3.0` 형식으로만 생성되고 `v1.2.6` 이하 태그는 그대로 유지됨. 2~3) `1.2.6` 클라이언트가 `desktop-v1.3.0` 정식 릴리스를 감지하여 업데이트 모달을 띄우고, 모달 안에 릴리스 노트(또는 릴리스 페이지 링크)가 정상 표시됨
  - 실패 판정: 접두어 누락으로 `v1.3.0` 태그가 생성되거나, electron-builder가 별도 릴리스를 만들거나, 1.2.6에서 업데이트 감지가 안 되거나, 업데이트 모달 노트가 빈 화면/오류로 표시됨
  - 증적 경로: `.loctx/reports/1.3.0/v19-desktop-tag-and-updater.log`

- V-20 **npm Staged Publishing 전환 및 승인 전 `npm view` 비노출 확인**
  - 검증 대상: S-16, W-16
  - 실행 방법:
    1) `.github/workflows/publish-packages.yml`, 루트 `package.json` 발행 스크립트, `AGENTS.md`를 확인하여 `changeset publish` 대신 `pnpm pack` → 압축 해제 폴더에서 `npm stage publish --provenance` 실행 및 stage ID의 `$GITHUB_STEP_SUMMARY` 출력, 이후 실행에서의 지연 git 태그(`@croffledev/<pkg>@<version>`) 생성 로직 확인
    2) 발행 스크립트를 `--dry-run`으로 실행해 `workspace:*` 의존성 치환과 명령 조립 검증
    3) 실제 패키지 변경 시 `chore: version packages` PR 병합으로 `publish-packages` 워크플로가 돈 직후(npmjs.com 승인 전) `npm view @croffledev/<pkg>@<새버전> version` 및 `git ls-remote --tags origin "@croffledev/<pkg>@<새버전>"` 실행 → npmjs.com에서 stage 승인 후 다시 `npm view` 확인 및 다음 워크플로 실행 후 git 태그 확인
  - 판정 시점: 1)~2)는 이 버전의 테스트에서, 3)은 staging으로 바뀐 뒤 처음 나가는 Changeset 발행 때 판정합니다.
  - 기대 결과: 1~2) `AGENTS.md` 설명과 워크플로 구현이 일치하고 `--dry-run`이 종료 코드 0으로 통과함. 3) **npmjs.com에서 승인하기 전에는 `npm view`가 해당 버전을 반환하지 않고(E404 / 미노출) git 태그도 없으며**, 사용자가 npmjs.com에서 승인한 뒤에야 `npm view`에 새 버전이 나타나고 후속 실행에서 git 태그가 생성됨
  - 실패 판정: PR 병합만으로 승인 없이 npm에 버전이 즉시 공개되거나, 승인 전에 git 태그가 생성되거나, `AGENTS.md`와 실제 워크플로 동작이 불일치함
  - 증적 경로: `.loctx/reports/1.3.0/v20-npm-stage-publish.log`

## 우려
- **트리거와 마이그레이션**: SQLite는 테이블을 다시 만드는 마이그레이션에서 트리거를 떨어뜨린다. 부팅 보정(`ensureOutboxTriggers`)으로 막지만, 이후 버전의 마이그레이션은 ADD COLUMN만 써야 한다.
- **억제 규약**: 서버 행 적용 경로가 `lastWriterId='server'`를 빠뜨리면 받은 변경이 다시 push된다(핑퐁). 적용 경로를 한 함수로 모은다.
- **시계 LWW**: 클라이언트 벽시계가 틀리면 그 기기가 항상 이기거나 진다. 경고만 하고 보정은 하지 않는다(intent 하지 않을 것).
- **토큰 만료 없음**: 서버 0.1 토큰은 만료되지 않는다. 분실 기기는 서버 CLI로 revoke해야 한다(서버 README).
- **electron-updater**: 접두어 태그로 업데이트 경로 자체는 유지된다(2026-10-20 분석: 버전은 `latest.yml`에서 읽음). 다만 실제 1.2.6 설치가 `desktop-v1.3.0-rc.1`을 보는지는 rc를 내야 확인된다. rc는 `/releases/latest`에 잡히지 않으므로 1.2.6 클라이언트는 stable `desktop-v1.3.0`부터 업데이트를 받는다.
- **npm staging 첫 실행**: `npm stage publish`가 tarball 경로를 받는지 문서가 분명하지 않다. 풀어 둔 폴더에서 실행하는 방식으로 설계했고, CI에서 `--dry-run`으로 먼저 확인한다. npmjs.com 설정 변경 전에는 OIDC 토큰이 stage를 거부할 수 있다.
- **macOS**: 실서버 검증을 하지 못한다. 릴리스 노트에 밝힌다.

## 범위 밖
- 일정 공유와 `schedule.teamId`(1.3.1), To-Do(1.3.2), 공유 권한 · 공유 취소(1.3.5)
- 팀 · 원격 일정 관리 UI, `schedule_external_link` 동기화
- `packages/types`에 `sync.*` 노출
- 리프레시 토큰 · 조용한 갱신, 시계 오차 보정
- `packages/sync-fixtures`에 시나리오 추가, 서버 코드 수정
- `croffle-plugin --version` 하드코딩 버그(백로그)
