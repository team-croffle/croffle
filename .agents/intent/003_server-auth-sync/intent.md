---
id: "003"
slug: server-auth-sync
version: "server-0.1.1"
status: draft
origin: user
approvals: []
waivers: []
supersedes: null
---
## 문제
서버 0.1.0 은 데스크톱 1.3.0 과 동기화가 붙는 최소 서버입니다. 계속 쓰려면 아래 네 가지가 부족합니다.

- **인증을 직접 구현했습니다.** argon2id 해시 · opaque 기기 토큰 · 로그인 rate limit 을 표준 라이브러리 위에 손으로 짰습니다(`apps/server/internal/auth/`). OTP 는 요청 필드만 있고 등록 수단이 없습니다.
- **토큰이 만료되지 않고 갱신할 수도 없습니다.** 기기 토큰은 로그인할 때나 CLI `token issue` 로 한 번 받고 끝입니다. 데스크톱은 비밀번호를 저장하지 않으므로, 만료를 넣으면 지금은 다시 로그인하는 수밖에 없습니다(`protocol/auth.go` 의 `ExpiresAt` 주석, 0.1 설계 기록의 "0.2의 리프레시 토큰").
- **충돌은 서버가 판정하지 않습니다.** push 에서 `baseVersion` 이 서버 행의 `version` 과 다르면 서버 행을 돌려줄 뿐이고, 승자는 데스크톱이 LWW 로 정합니다(`store/sqlite/sync.go:60`, `apps/desktop/src/main/sync/push.ts` 의 `settle`).
- **운영 지표가 없습니다.** `/health` 와 JSON 로그만 있습니다. metrics 는 0.1 설계에서 "0.2 이후"로 미뤘습니다.
- **`--password-stdin` 이 터미널에서 끝나지 않습니다** (2026-11-11 사용자 보고). `io.ReadAll` 로 EOF 까지 읽어서 Enter 로는 끝나지 않고, Windows Git Bash 처럼 Ctrl+D 가 EOF 로 가지 않는 터미널에서는 진행되지 않습니다.

사용자 요청 (2026-11-15 · 16):
- "v0.2에서는 성숙한 라이브러리를 쓴 인증, 토큰 발급형 인증 등이 있음. 근데, 이거는 v0.1.1로 두고 버전을 조정해야 할거 같음"
- "수정 후보는 v0.1.1로."
- "0.1.1 부터 진행해보자"

로컬 로드맵의 0.1.1 항목: "성숙한 라이브러리를 쓴 인증, 토큰 발급형 인증, changeset 비교와 conflict 해결, metrics".

## 바라는 결과
- 인증의 암호 · 토큰 처리를 검증된 라이브러리가 맡고, 직접 짠 보안 코드가 줄어듭니다.
- 데스크톱이 비밀번호를 저장하지 않고도 오래 연결을 유지할 수 있는 토큰 발급 체계가 서버에 있습니다(짧은 access token + 갱신용 토큰, 기기 단위 회수).
- 충돌 판정과 해결 규칙이 서버에 있고, 공유 픽스처로 데스크톱과 같은 결과가 보증됩니다.
- 운영자가 요청 수 · 지연 · push/pull · 충돌 · 로그인 실패를 지표로 볼 수 있습니다.
- `--password-stdin` 을 터미널에서 쓰면 한 줄을 화면에 보이지 않게 읽고 Enter 로 끝납니다. 파이프 입력은 지금처럼 동작합니다.
- **데스크톱 1.3.0 은 아무것도 바꾸지 않아도 0.1.1 서버와 계속 동기화됩니다.**

## 계획
1. **`--password-stdin` 수정** — 입력이 터미널이면 한 줄을 echo 없이 읽고, 파이프면 지금처럼 읽습니다. README(en · ko)의 계정 생성 안내를 고칩니다. 독립된 작은 단위라 먼저 합니다.
2. **인증 라이브러리 도입** — 열린 질문 1의 답에 따라 손으로 짠 부분을 라이브러리로 바꿉니다. 기존 해시 · 토큰은 그대로 검증되어야 합니다(마이그레이션 없이 로그인 유지).
3. **토큰 발급** — access token + refresh token(회전, 재사용 감지 시 그 기기의 토큰 전체 회수), 갱신 엔드포인트, 대시보드 · CLI 의 기기 목록과 회수. 기존 기기 토큰은 열린 질문 3의 답에 따라 계속 받습니다.
4. **changeset 비교 · 충돌 해결** — 열린 질문 2의 답에 따라 서버의 충돌 판정을 넓힙니다. 공유 픽스처(`packages/sync-fixtures`)에 시나리오를 더해 서버 `go test` 와 데스크톱 `node --test` 가 같이 읽게 합니다.
5. **metrics** — 열린 질문 4의 답에 따라 지표 엔드포인트를 둡니다.
6. **문서 · 호환 표** — README(en · ko)의 설정 · 인증 · 지표 안내, 서버 버전 ↔ protocol 호환 표, 릴리스 노트 `docs/server/release/0.1.1.md`.

머지 뒤 audit → `server-v0.1.1-rc.1` 입니다.

## 영향 범위
- `apps/server/internal/{auth,httpapi,protocol,store/sqlite,cli,config}`, 새 SQL 마이그레이션(기존 것은 고치지 않음), `go.mod` · `go.sum`.
- `apps/server/web` (대시보드의 기기 · 토큰 화면, 필요할 때).
- `packages/sync-fixtures` (충돌 시나리오 추가). 데스크톱 쪽 픽스처 러너가 새 시나리오를 읽어야 하면 `apps/desktop/test/**`.
- `apps/server/README.md` · `README.ko.md`, `docs/server/release/0.1.1.md`, AGENTS.md 의 서버 의존성 설명(프로젝트 영역).
- 영향 없음(PM 해석): 데스크톱 앱 코드(`apps/desktop/src/**`), `packages/types` · `packages/cli`, 데스크톱 릴리스.

## 제약
- **데스크톱 1.3.0 호환** — protocol 1 의 요청 · 응답 모양을 깨지 않습니다. 새 동작은 새 엔드포인트나 protocol 2 로만 열고, `/health` 의 `protocol.min` 은 1 로 둡니다(PM 해석, 열린 질문 3).
- 서버는 여전히 **바이너리 하나 · SQLite 하나**로 뜹니다. 외부 서비스(IdP, Redis, Postgres)를 요구하지 않습니다.
- 커밋된 SQL 마이그레이션은 고치지 않고 새 마이그레이션만 더합니다.
- 훅은 관찰 전용 · 비동기라는 0.1 결정을 지킵니다. 충돌 해결이 훅을 통해 바뀌지 않습니다.
- `server-release` 는 GitHub Release 를 만들지 않습니다.
- 서버만 앞서는 버전입니다. 데스크톱이 새 토큰 체계를 쓰는 작업은 이 intent 에 넣지 않습니다.

## 열린 질문
1. **"성숙한 라이브러리"의 범위** — 어느 쪽입니까?
   - (가) 지금 구조(argon2id + 기기당 opaque 토큰)는 유지하고, 직접 짠 부분만 검증된 라이브러리로 바꿉니다: TOTP `pquerna/otp`, rate limit `golang.org/x/time/rate`, 대시보드 세션 · CSRF 등. **PM 추천.**
   - (나) 서버를 OAuth2 인가 서버로 만듭니다(`zitadel/oidc` 또는 `ory/fosite`). 표준 토큰 엔드포인트와 refresh 가 생기지만 의존성과 코드가 크게 늘어납니다.
   - (다) JWT access token(`golang-jwt/jwt/v5`) + opaque refresh token. 0.1 설계에서 "JWT는 돌릴 수 없음"을 이유로 기기당 opaque 토큰을 골랐으므로 그 결정을 바꾸게 됩니다.
   - TOTP 등록(0.1 설계에서 0.2로 미룬 것)을 이번에 넣을지도 함께 정해 주십시오.
2. **"changeset 비교와 conflict 해결"의 뜻** — 어느 쪽입니까?
   - (가) 서버가 행 단위 LWW 로 승자를 정해 적용하고, 진 쪽에는 이긴 행을 돌려줍니다. 데스크톱의 `settle` 과 같은 규칙을 서버에도 둡니다.
   - (나) (가) + 필드 단위 비교: 서로 다른 필드만 바뀌었으면 병합합니다. payload 가 JSON 덩어리라 서버가 필드를 알아야 합니다.
   - (다) 판정은 지금처럼 데스크톱에 두고, 서버는 충돌 응답에 어떤 필드가 다른지 비교 결과를 더 줍니다.
3. **기존 기기 토큰** — 만료 없는 기기 토큰(데스크톱 1.3.0 이 쓰는 것)을 0.1.1 에서도 계속 받아도 됩니까? PM 추천은 "계속 받고, 운영자가 설정으로 만료를 켤 수 있게"입니다.
4. **metrics** — Prometheus 형식 `/metrics` 를 `prometheus/client_golang` 으로 두고, 기본은 꺼짐 · 켜면 별도 주소(`CROFFLE_METRICS_LISTEN`)에서만 열리게 해도 됩니까?
5. **의존성 원칙** — AGENTS.md 와 0.1 설계는 서버를 "stdlib + `modernc.org/sqlite` + `x/crypto`"로 적고 있습니다. 위 답에 따라 라이브러리가 늘어나면 이 원칙을 "검증된 소수 라이브러리 허용"으로 바꿔도 됩니까?

## 질의서 기록
—

### 에이전트가 정한 것
- 작업 순서: `--password-stdin` 을 먼저 합니다. 다른 단위와 겹치지 않고 사용자가 이미 겪은 문제라서입니다. 대안은 인증 단위에 묶는 것이었습니다.
- 버전 문자열은 `server-0.1.1` 로 적습니다. 데스크톱 버전과 `.loctx/reports/<version>/` 경로가 섞이지 않게 하려는 것입니다.
- 이 intent 의 브랜치 `feat/server-0.1.1` 에 2026-11-15 로드맵 번호 재조정 커밋(`5fcb36e`)을 함께 둡니다. 따로 PR 을 내는 대안보다 PR 수가 적습니다.

## 하지 않을 것
- 데스크톱의 새 토큰 체계 적용(refresh 사용) — 이후 데스크톱 패치에서 합니다.
- Vue 대시보드 고도화 · 다중 팀 · scope 별 커서 — 서버 0.2.0.
- 공유 일정 · 공유 취소 · 멤버 관리 — 서버 0.2.1.
- 계정 · 데이터 실제 삭제(purge), GitHub OIDC 같은 외부 로그인, Postgres 구현체.
