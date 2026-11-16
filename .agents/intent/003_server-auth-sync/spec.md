---
intent: "003"
status: accepted
approvals:
  - { role: user, by: bluenyang, at: "2026-11-16", words: "F1. ㅇㅇ / F2. ㅇㅇ 고쳐.", rev: "감사 A-3-1 · A-3-2: V-1 · V-10 의 12번 SKIP 허용, V-4 · V-6~V-9 오류 응답 예시를 실제 모양으로", kind: change }
  - { role: user, by: bluenyang, at: "2026-11-16", words: "V-3는 전에 확인 했고, V-14는 (가)", rev: "V-14 기대 결과 · 실패 판정 변경(의도된 401 두 가지 허용, favicon 404 제거)", kind: change }
  - { role: user, by: bluenyang, at: "2026-11-16", words: "- TOTP 암호화는 바로 다음버전으로 넣자. - 만료는 어쩔 수 없고, 바이너리는 수용 진행", rev: "0d4a243", kind: summary }
summary_for_user: |
  서버 0.1.1 을 작업 단위 10개로 나눴습니다: `--password-stdin` → rate limit 교체 · 의존성 원칙 → 대시보드 CSRF · 기기 회수 → 토큰 저장소(마이그레이션 0004) → 기기 토큰 만료 설정 → `/auth/token`(access · refresh) → TOTP(0005) → 충돌 `differentFields` · 픽스처 13 → metrics → 문서.
  intent 와 다른 점: 계획 3(토큰)을 셋(W-4~W-6)으로 나눴고, TOTP 는 CSRF 와 `/auth/token` 뒤(W-7)로 옮겼습니다. 대시보드는 본인 계정에 한해 쓰기(기기 회수 · TOTP)가 생깁니다.
  데스크톱 1.3.0 호환: protocol 1 과 `/health` 의 범위는 그대로이고, 새 기능은 새 엔드포인트 · 선택 필드 · `features` 로만 엽니다. 만료 없는 토큰은 기본이고, 만료를 켜도 `--no-expiry` 로 낼 수 있습니다.
  검증 기준 V-1~V-15 는 qa-designer 가 쓰고 PM 이 고쳤습니다. 수동 확인은 터미널 비밀번호 입력(Linux · Windows Git Bash), 대시보드 화면, 데스크톱 1.3.0 실동기화(TOTP 로그인 포함)입니다.
  우려: TOTP secret 은 DB 에 평문으로 둡니다(암호화는 키 관리가 필요해 이후로). metrics 라이브러리로 바이너리가 수 MB 커집니다.
---
## 요구 해석
- S-1 `--password-stdin` 은 stdin 이 터미널이면 echo 없이 한 줄을 읽고 Enter 로 끝납니다. 파이프면 지금처럼 끝까지 읽고 끝의 줄바꿈만 뗍니다. README 의 계정 생성 안내는 터미널용(`docker exec -it`)과 파이프용 두 가지를 보여 줍니다. ← 계획 1, 바라는 결과 5
- S-2 직접 짠 보안 코드를 검증된 라이브러리로 바꿉니다. 로그인 rate limit 은 `golang.org/x/time/rate`, 대시보드 CSRF 는 Go 표준 `http.CrossOriginProtection`, TOTP 는 `pquerna/otp` 입니다. 비밀번호 해시(argon2id, `x/crypto`)와 opaque 토큰 구조는 그대로입니다. ← 계획 2, 질의서 A1
- S-3 0.1.0 에서 만든 계정의 비밀번호와 기기 토큰은 마이그레이션 뒤에도 그대로 로그인 · 동기화됩니다. ← 계획 2 "기존 해시 · 토큰은 그대로 검증"
- S-4 TOTP 를 켠 계정은 로그인(동기화 · 대시보드 모두)에 6자리 코드나 복구 코드가 필요합니다. 코드가 없으면 0.1 의 `401 otp_required` 를 돌려주므로 데스크톱 1.3.0 의 OTP 입력 흐름이 그대로 동작합니다. 등록은 대시보드에서 본인이, 해제는 본인(대시보드) 또는 운영자(CLI)가 합니다. ← 계획 2, 질의서 6
- S-5 만료 없는 기기 토큰(로그인 · `token issue` 로 받는 것)은 계속 받고, 계속 발급할 수 있습니다. 운영자가 `CROFFLE_DEVICE_TOKEN_TTL` 을 켜면 그 뒤 로그인으로 발급하는 기기 토큰에 만료가 붙습니다. 그 상태에서도 `token issue --no-expiry` 로 만료 없는 토큰을 따로 낼 수 있습니다. ← 계획 3, 질의서 A3 과 PM 해석
- S-6 새 엔드포인트 `POST /auth/token` 이 짧은 access token 과 refresh token 쌍을 발급합니다. refresh 는 쓸 때마다 새것으로 바뀌고, 이미 쓴 refresh 가 다시 오면 그 기기의 access · refresh 토큰을 모두 회수합니다. 동기화 엔드포인트는 기기 토큰과 access token 을 모두 받습니다. ← 계획 3, 바라는 결과 2
- S-7 기기 목록과 회수를 CLI(지금 있음)와 대시보드(본인 기기만) 양쪽에서 할 수 있습니다. 기기를 회수하면 그 기기의 기기 토큰 · access · refresh 가 모두 멈춥니다. ← 계획 3
- S-8 push 충돌 응답의 각 `Conflict` 에 `differentFields`(보낸 행과 서버 행의 값이 다른 필드 이름, 정렬된 배열)가 붙습니다. 승자는 지금처럼 데스크톱이 정하고, 서버의 적용 규칙(`baseVersion` 이 다르면 적용하지 않음)은 바뀌지 않습니다. 이 동작을 공유 픽스처 새 시나리오가 고정합니다. ← 계획 4, 질의서 A2
- S-9 `CROFFLE_METRICS_LISTEN` 이 비어 있으면 metrics 가 없습니다(포트도 열지 않음). 값을 주면 그 주소에서만 Prometheus 형식 `/metrics` 를 엽니다. 요청 수 · 지연 · push/pull · 충돌 · 로그인 실패 · rate limit 을 셉니다. 본 API 주소에는 `/metrics` 가 없습니다. ← 계획 5, 바라는 결과 4, 질의서 A4
- S-10 데스크톱 1.3.0 은 바꾸지 않아도 0.1.1 서버와 동기화됩니다. protocol 1 의 요청 · 응답에서 빠지거나 뜻이 바뀌는 필드가 없고, `/health` 의 `protocol` 범위는 `{min: 1, max: 1}` 그대로입니다. 새 기능은 `/health` 의 `features` 로 알립니다. ← 바라는 결과 6, 제약 1
- S-11 서버는 바이너리 하나와 SQLite 하나로 뜹니다. 새 설정은 모두 선택이고 기본값으로 0.1.0 과 같게 동작합니다. ← 제약 2
- S-12 서버 의존성 원칙을 "검증된 소수 라이브러리 허용"으로 바꾸고 AGENTS.md(프로젝트 영역)와 README 에 적습니다. 새 의존성마다 이 spec 의 "의존성" 표에 이유를 적습니다. ← 제약 3, 계획 6, 질의서 A5
- S-13 README(en · ko)에 인증(TOTP · 토큰 종류 · 만료) · metrics · 설정 안내와 서버 버전 ↔ protocol ↔ 데스크톱 호환 표가 생기고, 릴리스 노트 `docs/server/release/0.1.1.md` 가 생깁니다. ← 계획 6

## 설계

### 의존성
| 모듈 | 쓰는 곳 | 이유 |
|---|---|---|
| `golang.org/x/term` | CLI 비밀번호 입력 | 터미널 판별 · echo 끄기. Go 팀 모듈 |
| `golang.org/x/time/rate` | 로그인 rate limit | 손으로 짠 token bucket 대체. Go 팀 모듈 |
| `github.com/pquerna/otp` (+ `boombuler/barcode`) | TOTP | RFC 6238 구현 · otpauth URI · QR PNG. Go 생태계 표준 |
| `github.com/prometheus/client_golang` (+ 전이 의존) | metrics | Prometheus 공식 클라이언트 |

CSRF 는 Go 1.25 부터 표준인 `net/http.CrossOriginProtection` 을 쓰므로 의존성이 늘지 않습니다. 대시보드(`apps/server/web`)에는 npm 의존성을 더하지 않습니다(QR 은 서버가 PNG 로 줍니다).

### W-1 비밀번호 입력 (`internal/cli`)
- `readPassword` 가 stdin 이 `*os.File` 이고 `term.IsTerminal(fd)` 이면 `Password: ` 를 stderr 에 쓰고 `term.ReadPassword` 로 한 줄을 읽습니다. 아니면 지금처럼 `io.ReadAll(LimitReader)` 후 끝의 `\r\n` 을 뗍니다.
- 테스트는 파이프 경로만 자동으로 봅니다(터미널 경로는 수동 확인).
- README: `docker exec -it <c> croffle-server user create alice --password-stdin`(터미널), `printf '%s' "$PW" | docker exec -i …`(파이프). Windows Git Bash 는 `winpty docker exec -it …` 를 함께 적습니다.

### W-2 rate limit 교체 · 의존성 원칙 (`internal/auth`)
- `Limiter` 의 공개 모양(`Allow(key)`, `Reset(key)`)은 그대로 두고 안을 키별 `*rate.Limiter`(`rate.Every(30s)`, burst 5)로 바꿉니다. 오래 안 쓴 키 정리(10분, 최대 1만 키)는 지금 규칙을 유지합니다. 테스트용 `now` 주입은 `AllowN(now, 1)` 로 이어 갑니다.
- AGENTS.md 프로젝트 영역의 "stdlib only + `modernc.org/sqlite` + `x/crypto`" 문장(Stack · Layout 두 곳)을 "stdlib 위주, 검증된 소수 라이브러리 허용(목록은 `apps/server/README.md`)"으로 바꿉니다. README 에 의존성 절을 둡니다.

### W-3 대시보드 CSRF · 기기 회수 (`internal/httpapi`, `web`)
- 0.1 의 대시보드는 읽기 전용이었습니다. 0.1.1 부터 **본인 계정에 한한 쓰기**(기기 회수, W-7 의 TOTP)가 생깁니다. 세션이 동기화 자격이 아니라는 0.1 의 원칙(push/pull 불가)은 그대로입니다.
- `/dash/*` 의 상태를 바꾸는 요청(POST · DELETE)을 `http.CrossOriginProtection` 으로 감쌉니다. `Sec-Fetch-Site`/`Origin` 이 다른 출처면 403 입니다. `SameSite=Strict` 쿠키는 유지합니다. 로그인 · 로그아웃도 같은 보호를 받습니다.
- `POST /dash/devices/{id}/revoke`: 세션 계정의 기기만 회수하고 `204` 를 줍니다. 남의 기기 id 와 없는 id 는 똑같이 `404 not_found` 입니다. store 에 `RevokeAccountDevice(accountID, deviceID)` 를 더하고, 맞는 행이 없으면 `ErrNotFound` 를 돌려줍니다(CLI 의 `RevokeDevice` 는 그대로). 대시보드 기기 표에 "회수" 버튼과 확인 창을 둡니다.
- `CROFFLE_DASH_TRUSTED_ORIGINS`(쉼표 구분, 기본 빈 값): `CrossOriginProtection.AddTrustedOrigin` 에 넘깁니다(우려 6).

### W-4 토큰 저장소 (마이그레이션 `0004_tokens.sql`, `internal/store`)
- `ALTER TABLE device ADD COLUMN tokenExpiresAt INTEGER` — NULL 은 만료 없음. 0.1.0 의 모든 기존 토큰은 NULL 이 되어 그대로 동작합니다(S-3).
- 새 테이블 `auth_token`:
  - `tokenHash BLOB PK`, `deviceId TEXT NOT NULL REFERENCES device ON DELETE CASCADE`, `kind TEXT NOT NULL CHECK (kind IN ('access','refresh'))`, `familyId TEXT NOT NULL`, `createdAt`, `expiresAt INTEGER NOT NULL`, `usedAt INTEGER`(refresh 를 바꾼 시각), `revokedAt INTEGER`. 인덱스 `(deviceId)`, `(expiresAt)`.
- store 메서드: `CreateTokenPair`, `FindAuthToken(hash)`, `RotateRefresh(oldHash, newPair)`(한 트랜잭션: 옛 refresh 의 `usedAt` 기록 + 새 쌍 삽입), `RevokeTokenFamily(familyId)`, `RevokeDevice` 가 `auth_token` 도 함께 회수. retention sweep 이 만료된 `auth_token` 행을 지웁니다.
- 기존 마이그레이션 0001~0003 은 고치지 않습니다.

### W-5 기기 토큰 만료 설정 (`internal/config`, `internal/auth`, `internal/cli`)
- `CROFFLE_DEVICE_TOKEN_TTL`(Go duration, 기본 `0` = 만료 없음). 0 이 아니면 **그 뒤 `/auth/login` 으로 발급하는** 기기 토큰에 `tokenExpiresAt = now + TTL` 이 붙고, 로그인 응답의 `expiresAt`(0.1 에 자리만 있던 필드)에 그 값이 실립니다. 이미 발급된 토큰은 바뀌지 않습니다.
- `Authenticate` 는 `tokenExpiresAt` 이 지났으면 `401 unauthorized` 입니다(코드 추가 없음. 1.3.0 은 다시 로그인하라고 안내합니다).
- CLI `token issue <user> [--expires-in DURATION | --no-expiry]`: 기본은 설정을 따르고, `--no-expiry` 는 설정과 상관없이 만료 없음입니다. `token list` 표에 `EXPIRES` 열(만료 없음은 `never`), `--json` 에 `tokenExpiresAt`(RFC 3339, 만료 없음은 생략)을 더합니다. `GET /dash/devices` 의 각 기기에 `tokenExpiresAt`(unix ms, 만료 없음은 생략)을 더하고 대시보드 표에 보여 줍니다. 만료가 지난 기기의 `state` 는 `expired` 입니다.

### W-6 access · refresh 발급 (`internal/auth`, `internal/httpapi`, `internal/protocol`)
- `POST /auth/token`, 본문 `{ grantType, … }`:
  - `grantType: "password"` — `username`, `password`, `otp?`, `clientId`, `deviceName?`. `/auth/login` 과 같은 검증 · rate limit · OTP 규칙. 기기 행을 만들거나 갱신하되 **기기 토큰은 건드리지 않습니다**(같은 기기가 1.3.0 방식 토큰을 쓰고 있어도 끊기지 않게).
  - `grantType: "refresh_token"` — `refreshToken`. 유효하면 옛 것은 `usedAt` 을 찍고 새 쌍을 줍니다. `usedAt` 이 이미 있는 refresh 면 그 기기의 access · refresh 토큰을 family 와 상관없이 모두 회수하고(S-6. 처음 설계는 `familyId` 만 회수했으나 리뷰 1에서 S-6 과 어긋난다는 지적을 받아 고침, 기기 토큰은 그대로) `401 unauthorized` 를 줍니다(로그에 `refresh reuse` 경고).
  - 응답 `{ accessToken, accessExpiresAt, refreshToken, refreshExpiresAt, accountId, teamId? }`, 시각은 unix ms.
- 수명: `CROFFLE_ACCESS_TOKEN_TTL`(기본 15분), `CROFFLE_REFRESH_TOKEN_TTL`(기본 30일, 회전할 때마다 새로 계산 — 30일 안에 한 번이라도 쓰면 계속 이어짐).
- `requireAuth` → `Authenticate`: 먼저 `device.tokenHash`, 없으면 `auth_token` 의 `kind='access'` 를 찾습니다. refresh token 은 동기화 요청에 쓸 수 없습니다(401). 기기 회수 · 계정 비활성은 두 경로 모두에 적용됩니다.
- `/health` 에 `features` 를 더합니다. 0.1.1 의 값은 정확히 `["conflict-fields", "device-token-ttl", "token-pair", "totp"]`(이름순)입니다. metrics 는 운영자 쪽 기능이라 넣지 않습니다. protocol 범위는 그대로입니다(S-10). `features` 는 W-6 에서 `token-pair` 로 시작해 W-5 · W-7 · W-8 이 각자 이름을 더합니다.
- 데스크톱은 이 엔드포인트를 아직 쓰지 않습니다(intent 하지 않을 것).

### W-7 TOTP (마이그레이션 `0005_otp.sql`, `internal/auth`, `internal/httpapi`, `internal/cli`, `web`)
- 테이블 `account_otp(accountId PK REFERENCES account ON DELETE CASCADE, secret TEXT NOT NULL, enabledAt INTEGER, lastStep INTEGER NOT NULL DEFAULT 0, createdAt)` — `enabledAt` 이 NULL 이면 등록 중(아직 확인 전). `otp_recovery_code(codeHash BLOB PK, accountId, usedAt)`.
- 검증: `totp.ValidateCustom`(SHA1, 6자리, 30초, 앞뒤 1스텝 허용). 이미 쓴 스텝(`lastStep` 이하)은 거부해 재사용을 막습니다. 복구 코드는 16자 base32(80비트), SHA-256 해시로 저장하고 한 번 쓰면 끝입니다. OTP 실패도 rate limit 시도로 셉니다.
- 로그인(`/auth/login`, `/auth/token` password, `/dash/login`): 비밀번호가 맞고 TOTP 가 켜져 있는데 `otp` 가 비었으면 `401 otp_required`, 틀리면 `401 unauthorized`(비밀번호 오류와 같은 답).
- 대시보드(본인, CSRF 보호):
  - `POST /dash/otp/enroll` → `200 { secret, otpauthUri, qrPng }`(`qrPng` 는 `data:image/png;base64,…`). 이미 켜져 있으면 `409 otp_enabled`. 확인 전에 다시 부르면 secret 을 새로 만듭니다.
  - `POST /dash/otp/confirm { code }` → `200 { recoveryCodes: [10개] }`. 코드가 틀리면 `400 invalid_code`. 복구 코드는 이 응답에만 나옵니다.
  - `POST /dash/otp/disable { password, code }` → `204`. `code` 는 TOTP 코드나 복구 코드 둘 다 받습니다. 틀리면 `401 unauthorized`.
  - `GET /dash/me` 에 `otpEnabled`. `POST /dash/login` 본문에 `otp` 를 더하고, 대시보드 로그인 폼에 `otp_required` 일 때 코드 칸이 나옵니다.
- 로그인의 `otp` 칸도 6자리 TOTP 코드와 복구 코드를 둘 다 받습니다(길이로 구분).
- CLI: `user otp status <user>`(`enabled` / `pending` / `off`), `user otp disable <user>`(운영자가 기기를 잃은 사용자를 풀어 줄 때). CLI `token issue` 는 운영자 경로라 OTP 를 묻지 않습니다(0.1 과 같음).
- secret 은 DB 에 평문으로 둡니다(아래 우려 1).

### W-8 충돌 필드 비교 (`internal/protocol`, `internal/httpapi`, `internal/store`, `packages/sync-fixtures`)
- `protocol.Conflict` 에 `DifferentFields []string json:"differentFields"` 를 더합니다.
- 계산: 보낸 commit 의 wire `row` 와 서버 행을 `MergeRow` 로 만든 wire 행을 최상위 키 단위로 비교합니다. 값은 JSON 으로 풀어 의미로 비교합니다(키 순서 · 숫자 표기 무시, 없는 키와 `null` 은 같음). 장부 필드 `version`, `updatedAtMs`, `lastWriterId`, `ownerId`(서버가 덮어씀)는 제외합니다. 삭제 commit 이 `row` 없이 오면(protocol 1 이 허용) `deletedAt` 하나만 비교합니다. 데스크톱 1.3.0 은 삭제 때도 전체 행 + `deletedAt` 을 보내므로 픽스처는 그 모양을 씁니다. `teamId`, `deletedAt` 과 payload 필드는 포함합니다. 이름순 정렬, 다른 것이 없으면 `[]`.
- 비교에 보낸 행이 필요하므로 `store.Conflict` 에 commit 의 payload 를 싣거나 handler 가 opId 로 commit 을 찾아 계산합니다(구현 때 handler 쪽으로 정함. store 는 바뀌지 않는 편이 낫습니다).
- 서버의 적용 · 거부 규칙과 `applied_op` 기록은 바꾸지 않습니다.
- 픽스처 `13-push-conflict-different-fields.json`(protocol 1): 두 기기가 같은 행의 다른 필드(제목 / 시작 시각)를 바꾸고, 같은 필드를 같은 값으로 바꾸고(`[]`), 한쪽이 삭제하는(`deletedAt`) 경우. 서버 `go test ./internal/conformance` 가 읽습니다. 데스크톱 러너는 `rowCodec` 만 읽으므로 이 시나리오는 개수 · protocol 확인만 거칩니다(데스크톱 코드 변경 없음).
- `packages/sync-fixtures/README.md` 의 "desktop: work item 1.3.0#15" 문장은 그대로 두고 새 시나리오만 설명에 더합니다.

### W-9 metrics (`internal/metrics` 새 패키지, `internal/httpapi`, `internal/config`, `cmd`)
- `CROFFLE_METRICS_LISTEN`(기본 빈 값 = 꺼짐). 값이 있으면 별도 `http.Server` 가 그 주소에서 `GET /metrics` 하나만 엽니다. 본 API 의 mux 에는 등록하지 않습니다.
- 전용 `prometheus.Registry`(전역 기본 registry 를 쓰지 않음) + Go · process collector.
- 지표(이름은 `croffle_` 접두):
  - `http_requests_total{route,method,code}` · `http_request_duration_seconds{route,method}` — `route` 는 `r.Pattern`(mux 패턴)이라 경로 값이 라벨에 들어가지 않습니다. 패턴이 없거나 catch-all(`/`, 대시보드 SPA · 404)이면 `other`.
  - `sync_push_commits_total{result="accepted|conflict|retry"}`, `sync_pull_changes_total`, `sync_bootstrap_rows_total`
  - `auth_attempts_total{surface="sync|token|dashboard",result="ok|invalid|otp_required|disabled|rate_limited"}`, `auth_refresh_reuse_total`
- 라벨에 계정 id · 사용자명 · IP 를 넣지 않습니다.
- httpapi 에는 `Metrics` 인터페이스(nil 이면 아무것도 안 함)로 주입해 metrics 꺼짐일 때 비용이 없게 합니다.

### W-10 문서 · 호환 표
- `apps/server/README.md` · `README.ko.md`: 설정 표에 새 변수 5개(`CROFFLE_DEVICE_TOKEN_TTL`, `CROFFLE_ACCESS_TOKEN_TTL`, `CROFFLE_REFRESH_TOKEN_TTL`, `CROFFLE_METRICS_LISTEN`, `CROFFLE_DASH_TRUSTED_ORIGINS`), "인증"(토큰 3종 · 만료 · TOTP 등록/해제 · 복구 코드), "metrics"(켜는 법 · 지표 목록 · 외부에 열지 말 것), "의존성", "호환 표"(서버 0.1.0 / 0.1.1 ↔ protocol 1 ↔ 데스크톱 1.3.0, 0.1.1 의 새 기능은 데스크톱이 아직 쓰지 않음).
- `docs/server/release/0.1.1.md` 를 `.github/release-notes.md` 규약대로 씁니다.
- en · ko 내용이 같아야 합니다(0.1.0 노트의 Korean parity 와 같은 규칙).

## 작업 단위
각 단위는 따로 되돌릴 수 있고, 그 커밋은 게이트(`make -C apps/server check`, 대시보드를 고치면 `pnpm --dir apps/server/web build`, 공통 게이트)를 통과합니다.

- [x] W-1 (f3a67f4) `--password-stdin` 터미널 입력 + README 안내 — S-1
- [x] W-2 (305f488) rate limit 을 `x/time/rate` 로 교체 + 의존성 원칙(AGENTS.md · README) — S-2, S-12
- [x] W-3 (17bff1e) 대시보드 CSRF(`http.CrossOriginProtection`) + 대시보드 기기 회수 — S-2, S-7
- [x] W-4 (601ac72) 마이그레이션 0004 + 토큰 store 메서드 + sweep — S-3, S-6
- [x] W-5 (1e0c685) 기기 토큰 만료 설정 + CLI `--expires-in` / `--no-expiry` + 만료 표시 — S-5. 의존: W-4
- [x] W-6 (158c9a9) `POST /auth/token`(password · refresh, 회전 · 재사용 감지) + access token 인증 + `/health` `features` — S-6, S-7, S-10. 의존: W-4
- [x] W-7 (7fb8d9a) TOTP: 마이그레이션 0005 + 로그인 검증 + 대시보드 등록/해제 · 복구 코드 + CLI — S-4. 의존: W-3(CSRF), W-6(`/auth/token` 의 OTP 규칙)
- [x] W-8 (63b8477) 충돌 `differentFields` + 픽스처 13 — S-8, S-10
- [x] W-9 (ee4775b) metrics — S-9. W-6 · W-7 뒤에 두어 인증 지표를 한 번에 붙입니다
- [x] W-10 (93c3bd2) README(en · ko) · 호환 표 · 릴리스 노트 0.1.1 — S-13

intent 계획과 다른 점: 계획 3(토큰 발급)을 저장소(W-4) · 기기 토큰 만료(W-5) · access/refresh(W-6) 셋으로 나눴습니다. 만료 설정과 새 엔드포인트를 따로 되돌릴 수 있게 하려는 것입니다. 계획 2의 TOTP(W-7)는 대시보드 CSRF(W-3)와 `/auth/token`(W-6) 뒤로 옮겼습니다. 등록 화면은 CSRF 보호가 먼저 있어야 하고, OTP 규칙을 세 로그인 입구에 한 번에 붙이기 위해서입니다.

## 검증 기준
> 2026-11-16 변경(감사 A-3-2 · 사용자 "F2. ㅇㅇ 고쳐."): V-4 · V-6~V-9 의 오류 응답 예시를 실제 모양(0.1.0 부터 최상위 `code` · `message`)으로 고쳤습니다. 처음 문구의 `{"error":{…}}` 는 qa-designer 가 쓴 예시로, 프로토콜과 맞지 않았습니다.

qa-designer(google gemini-4-argon-high)가 작성하고 PM 이 검토 · 수정했습니다. qa-designer 가 짚은 모호한 점 5개(features 목록, 대시보드 새 엔드포인트의 응답, 삭제 충돌의 비교 기준, 만료 필드 이름, 새 변수 목록)는 설계에 정해 넣었습니다. PM 수정: V-2 대시보드 `package.json` 의 `version` 줄 허용, V-4 의 30초 리필은 단위 테스트로 판정, V-8 회수 응답을 `204` 로 고정, V-9 응답 필드 이름, V-11 catch-all 라벨, V-12 `features` 정확한 값.

- V-1 **프로젝트 정적 게이트 · 서버 및 대시보드 통합 빌드(`build-all`) · 공유 픽스처 러너**
  - 검증 대상: S-8, S-10, S-11, W-1~W-10
  - 실행 방법:
    1) 저장소 루트에서 `.agents/harness/gates.yaml`의 8개 게이트 명령을 순서대로 실행합니다: `pnpm typecheck && pnpm lint && pnpm exec oxfmt --check && pnpm -r test && pnpm build && pnpm --filter @croffledev/croffle-cli build && pnpm --dir apps/server/web build && make -C apps/server check`
    2) `make -C apps/server build-all`을 실행한 뒤 생성된 단일 바이너리 `apps/server/bin/croffle-server`의 존재 확인 및 `./apps/server/bin/croffle-server --version`을 실행합니다.
  - 기대 결과: 8개 게이트 명령과 `make -C apps/server build-all`이 모두 종료 코드 `0`으로 통과합니다. `pnpm -r test`(`apps/desktop/test/sync-protocol.test.ts`)와 `make -C apps/server check`(`go test ./internal/conformance`) 양쪽 모두 새 시나리오 `13-push-conflict-different-fields.json`을 포함한 `packages/sync-fixtures/scenarios/*.json` 13개 시나리오 전체를 읽고(`protocol: 1` 확인) fail `0`으로 통과하며(서버 러너는 `12-row-codec` 을 SKIP),  `./apps/server/bin/croffle-server --version`은 `sync protocol 1-1`을 출력합니다. (2026-11-16 변경, 감사 A-3-1 · 사용자 "F1. ㅇㅇ": `12-row-codec.json` 은 `rowCodec` 만 있는 데스크톱 전용 픽스처라 서버 러너는 0.1.0 부터 SKIP 합니다. 서버 러너는 12번을 SKIP, 나머지 12개를 PASS 하면 통과입니다.)
  - 실패 판정: 어느 한 단계라도 종료 코드가 `0`이 아니거나, `gofmt`·`go vet`·`go test`·`oxlint`·`oxfmt`·`typecheck` 오류가 1건이라도 출력되거나, 데스크톱 러너가 13개 픽스처를 모두 읽지 않거나, 서버 러너가 `12-row-codec` 밖의 시나리오를 하나라도 SKIP 하거나 실행하지 않거나, `apps/server/bin/croffle-server`가 생성되지 않으면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.1.1/v01-gates-and-build.log`

- V-2 **커밋된 마이그레이션(0001~0003) · 데스크톱/패키지 소스 불변 및 서버 의존성 원칙 검증 (음성 확인)**
  - 검증 대상: S-2, S-10, S-12, W-2, W-4, W-7
  - 실행 방법:
    1) 기존 서버 마이그레이션 불변 확인: `git diff master...HEAD -- apps/server/internal/store/sqlite/migrations/0001_init.sql apps/server/internal/store/sqlite/migrations/0002_sync_meta.sql apps/server/internal/store/sqlite/migrations/0003_dash_session.sql`
    2) 데스크톱 앱 · 드리즐 · 공개 타입 · CLI 불변 확인: `git diff master...HEAD -- apps/desktop/src apps/desktop/drizzle packages/types packages/cli`
    3) 대시보드 npm 의존성 불변 확인: `git diff master...HEAD -- apps/server/web/pnpm-lock.yaml` 및 `git diff master...HEAD -- apps/server/web/package.json | grep -E '^[+-] ' | grep -v '"version"'`
    4) 서버 Go 직접 의존성 및 문서 확인: `awk '/^require \(/,/\)/' apps/server/go.mod | grep -v '// indirect'`로 직접(direct) 의존성을 추출하고, `AGENTS.md`(Stack · Layout 두 곳)에서 `"stdlib only"` 문자열 검색(`! grep -n "stdlib only" AGENTS.md`) 및 `apps/server/README.md` · `apps/server/README.ko.md`의 의존성 절을 확인합니다.
  - 기대 결과:
    1) `0001_init.sql`~`0003_dash_session.sql`의 diff 출력이 `0`줄이고, 새 마이그레이션은 `0004_tokens.sql`, `0005_otp.sql`만 추가되어 있습니다.
    2) `apps/desktop/src`, `apps/desktop/drizzle`, `packages/types`, `packages/cli`의 diff 출력이 `0`줄입니다.
    3) `apps/server/web/pnpm-lock.yaml`의 diff가 `0`줄이고, `package.json`은 `"version"` 줄 외의 변경이 `0`줄입니다(새 npm 의존성 `0`개).
    4) `apps/server/go.mod`의 직접 의존성은 기존(`modernc.org/sqlite`, `golang.org/x/crypto`)과 spec 의존성 표에 명시된 모듈(`golang.org/x/term`, `golang.org/x/time`, `github.com/pquerna/otp`, `github.com/prometheus/client_golang`)뿐이며, `AGENTS.md`의 Stack · Layout 두 곳 모두 `"stdlib only"`가 제거되고 `"검증된 소수 라이브러리 허용"` 원칙과 `apps/server/README.md` 참조로 갱신되어 있습니다.
  - 실패 판정: `0001`~`0003` 마이그레이션이나 `apps/desktop/src` · `packages/types` · `packages/cli` · `apps/server/web/package.json`에 1줄이라도 변경이 있거나, spec 표에 없는 직접 의존성이 `go.mod`에 추가되었거나, `AGENTS.md`에 `"stdlib only"`가 남아 있으면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.1.1/v02-immutability-and-deps.log`

- V-3 **`--password-stdin` 파이프 자동 검증 및 터미널(TTY · `docker exec -it` · Windows `winpty`) 에코 억제 확인**
  - 검증 대상: S-1, W-1
  - 실행 방법:
    1) 자동 테스트: `go test -v ./internal/cli -run 'TestUserCreate|TestUserSetPassword|TestReadPassword'`를 실행하여 비-TTY 파이프 입력에서 끝의 `\n` · `\r\n` 제거, 빈 비밀번호 거부, 4096바이트 상한 처리가 통과하는지 확인합니다. 이어서 임시 DB에 `printf 'secret-pass-1\r\n' | ./apps/server/bin/croffle-server user create alice --password-stdin --db "$TMP_DB"`를 실행합니다.
    2) `[수동 · Linux]` 호스트 대화형 터미널(TTY)에서 `./apps/server/bin/croffle-server user create bob --password-stdin --db "$TMP_DB"`를 실행하고, 컨테이너 환경에서 `docker exec -it <container> croffle-server user create carol --password-stdin` 및 `docker exec -it <container> croffle-server user set-password carol --password-stdin`을 실행합니다. 비밀번호를 입력한 뒤 `Ctrl+D` 없이 `Enter`만 누릅니다.
    3) `[수동 · Windows]` Windows Git Bash(MINGW64)에서 `winpty docker exec -it <container> croffle-server user create dave --password-stdin`을 실행하고 비밀번호 입력 후 `Enter`만 누릅니다.
  - 기대 결과:
    1) `go test ./internal/cli`가 PASS하고, 파이프 실행이 종료 코드 `0`으로 `created account alice`를 출력하며 끝의 `\r\n`을 제외한 `"secret-pass-1"`로 해시가 저장됩니다.
    2~3) Linux TTY, `docker exec -it`, Windows Git Bash `winpty docker exec -it` 세 환경 모두에서 stderr에 `Password: `가 출력되고, 타이핑한 비밀번호 문자가 터미널 화면에 `0`글자 노출(echo 없음)되며, `Enter` 한 번으로 입력이 즉시 끝나 계정이 생성·비밀번호가 변경됩니다.
  - 실패 판정: 터미널에서 타이핑한 비밀번호가 화면에 1글자라도 에코되거나, `Enter`를 눌러도 종료되지 않고 EOF(`Ctrl+D`)를 기다리거나, 파이프 입력에서 줄바꿈 문자가 비밀번호에 포함되면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.1.1/v03-password-stdin.log`

- V-4 **로그인 Rate Limit(`golang.org/x/time/rate`) 교체 · 버스트 5회 · 30초 리필 · 성공 시 초기화 · 유휴 키 정리**
  - 검증 대상: S-2, W-2
  - 실행 방법:
    1) `grep -n "golang.org/x/time/rate" apps/server/internal/auth/ratelimit.go`로 표준 `rate.Limiter` 사용을 확인합니다.
    2) `go test -v ./internal/auth -run 'TestLimiter|TestLoginIsRateLimited'` 및 `go test -v ./internal/httpapi -run 'TestLogin.*RateLimit'`을 실행합니다(주입된 시계 `AllowN(now, 1)`, 버스트 5, 30초 리필, `Reset`, 10분 유휴 키 및 10,000키 상한 정리 포함).
    3) 로컬 서버를 띄우고 같은 IP · 같은 `username`(`"alice"`)으로 틀린 비밀번호를 연속 6회 `POST /auth/login`에 전송한 뒤, 같은 IP에서 다른 `username`(`"bob"`)으로 1회 전송합니다.
  - 기대 결과:
    1) `apps/server/internal/auth/ratelimit.go`가 직접 짠 토큰 버킷 대신 `golang.org/x/time/rate`(`rate.Every(30 * time.Second)`, burst `5`)를 사용합니다.
    2~3) `"alice"`에 대한 처음 5회 실패는 HTTP `401` (`{"code":"unauthorized","message":"invalid credentials"}`)을 반환하고, 6번째 요청은 HTTP `429` (`{"code":"too_many_attempts",...}`)와 `Retry-After: 30` 헤더를 반환합니다. 직후 `"bob"`에 대한 요청은 차단되지 않고 HTTP `401`을 반환합니다. 30초 리필(주입한 시계로 30초를 넘기면 1회 다시 허용)과 성공 시 `Reset` 초기화는 1)의 단위 테스트가 PASS 하는 것으로 판정합니다.
  - 실패 판정: `golang.org/x/time/rate`를 사용하지 않거나, 5회 이내에 429가 반환되거나, 6번째 시도에 429 및 `Retry-After: 30`이 반환되지 않거나, 다른 사용자명까지 함께 차단되면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.1.1/v04-rate-limiter.log`

- V-5 **0.1.0 DB 마이그레이션(0004 · 0005) 적용 후 기존 비밀번호 해시 · 기기 토큰 무중단 동작 (음성 확인)**
  - 검증 대상: S-3, S-10, W-4, W-7
  - 실행 방법:
    1) 자동 테스트: `go test -v ./internal/store/sqlite ./internal/httpapi -run 'TestMigrateFrom010|TestLegacyDeviceTokenAndPasswordAfterMigration'`을 실행합니다(0001~0003만 적용된 0.1.0 스키마 DB에 계정·기기 토큰·동기화 행을 생성한 뒤 0004·0005 마이그레이션을 적용하고 기존 자격증명을 검증하는 테스트).
    2) 0.1.0 스키마(버전 3)로 생성되어 계정 `alice`와 기기 토큰 `T_010`, 일정 1건이 들어 있는 SQLite DB 파일에 `./apps/server/bin/croffle-server db migrate --db "$DB010"` 및 `./apps/server/bin/croffle-server db info --json --db "$DB010"`을 실행한 뒤 0.1.1 서버를 기동합니다.
    3) 마이그레이션 전에 발급된 기존 토큰 `T_010`으로 `POST /auth/whoami`, `POST /sync/pull`, `POST /sync/push`를 호출하고, 0.1.0에서 만든 `alice`의 비밀번호로 `POST /auth/login`을 호출합니다.
  - 기대 결과:
    1) `db migrate`가 2개 마이그레이션(0004, 0005)을 적용하여 `schemaVersion`이 `5`가 되고, 기존 `device` 행의 `tokenExpiresAt`은 `NULL`이며 기존 `accounts: 1`, `devices: 1`, `rows: 1`, `commits: 1`이 그대로 보존됩니다.
    2) 재발급 없이 기존 0.1.0 기기 토큰 `T_010`으로 호출한 `/auth/whoami`, `/sync/pull`, `/sync/push`가 모두 HTTP `200`을 반환하고, 기존 0.1.0 비밀번호 해시로 호출한 `/auth/login`도 HTTP `200`으로 새 기기 토큰을 발급합니다.
  - 실패 판정: 마이그레이션 중 오류가 나거나, 기존 `device.tokenExpiresAt`이 `NULL`이 아니거나, 0.1.0에서 만든 기기 토큰 또는 비밀번호 해시가 마이그레이션 뒤 `401` 등으로 거부되면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.1.1/v05-upgrade-from-010.log`

- V-6 **기기 토큰 만료 설정(`CROFFLE_DEVICE_TOKEN_TTL`) · CLI `--expires-in` / `--no-expiry` · 만료 표시**
  - 검증 대상: S-5, W-5
  - 실행 방법:
    1) 자동 테스트: `go test -v ./internal/config ./internal/auth ./internal/cli ./internal/httpapi -run 'TestDeviceTokenTTL|TestTokenIssueExpiry'`를 실행합니다.
    2) 기본값(`CROFFLE_DEVICE_TOKEN_TTL` 미설정 = `0`)으로 기동한 서버에서 `POST /auth/login`을 호출해 `T_noexp_login`을 받고 응답의 `expiresAt` 유무를 확인합니다.
    3) 같은 DB로 `CROFFLE_DEVICE_TOKEN_TTL=2s`를 설정해 서버를 재기동한 뒤:
       - (a) `POST /auth/login`으로 새 기기 토큰 `T_ttl`을 발급받고 응답 JSON의 `expiresAt` 값과 발급 직후 `POST /auth/whoami` 응답을 확인합니다.
       - (b) `CROFFLE_DEVICE_TOKEN_TTL=2s` 환경에서 `./apps/server/bin/croffle-server token issue alice --no-expiry --device-name "perm-cli" --db "$TMP_DB"`로 `T_perm_cli`를, `--expires-in 1h --device-name "1h-cli"`로 `T_1h_cli`를 각각 발급합니다.
       - (c) 3초 대기 후 `T_noexp_login`(설정 전 발급), `T_perm_cli`(`--no-expiry` 발급), `T_1h_cli`(`--expires-in 1h` 발급), `T_ttl`(2초 TTL 발급) 4개 토큰으로 각각 `POST /auth/whoami`와 `POST /sync/pull`을 호출합니다.
       - (d) `./apps/server/bin/croffle-server token list alice --db "$TMP_DB"`, `token list alice --json --db "$TMP_DB"`, 대시보드 `GET /dash/devices` 출력을 확인합니다.
  - 기대 결과:
    1) 기본값(`0`)에서 `/auth/login` 응답에는 `expiresAt` 필드가 생략(또는 null)되고 만료되지 않습니다.
    2) `CROFFLE_DEVICE_TOKEN_TTL=2s`에서 새로 로그인한 응답에는 `expiresAt`(약 `now + 2000` unix ms)이 실리며, 만료 전에는 HTTP `200`, 3초 경과 후에는 `/auth/whoami`와 `/sync/pull` 모두 HTTP `401` (`{"code":"unauthorized",...}`)을 반환합니다.
    3) 반면 TTL을 켜기 전에 발급된 `T_noexp_login`, TTL이 켜진 상태에서도 `--no-expiry`로 발급한 `T_perm_cli`, 아직 만료 전인 `T_1h_cli`는 3초 후에도 모두 HTTP `200`으로 통과합니다.
    4) CLI `token list`(표 및 `--json`)와 `GET /dash/devices`에 각 기기의 만료 시각(또는 만료 없음)이 표시됩니다.
  - 실패 판정: `CROFFLE_DEVICE_TOKEN_TTL` 설정이 기존 토큰까지 만료시키거나, 만료 시각이 지난 기기 토큰이 `200`을 반환하거나, `--no-expiry`로 발급한 토큰에 만료가 붙거나, `token list`·대시보드에 만료 정보가 표시되지 않으면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.1.1/v06-device-token-ttl.log`

- V-7 **`POST /auth/token` 발급 · 회전 · 기존 기기 토큰 공존 · refresh 재사용 시 그 기기의 access · refresh 전체 회수 · 동기화 엔드포인트의 refresh 거부 (음성 확인)**
  - 검증 대상: S-6, W-4, W-6
  - 실행 방법:
    1) 자동 테스트: `go test -v ./internal/store/sqlite ./internal/auth ./internal/httpapi -run 'TestAuthToken|TestRefreshRotation|TestRefreshReuseRevokesFamily|TestRejectRefreshTokenOnSync|TestSweepExpiredAuthTokens'`를 실행합니다.
    2) 기기 `client-1`로 `POST /auth/login`을 호출해 기기 토큰 `D1`을 받은 뒤, 같은 `clientId: "client-1"`로 `POST /auth/token` (`{"grantType":"password","username":"alice","password":"...","clientId":"client-1","deviceName":"laptop"}`)을 호출해 `(A1, R1)` 쌍을 받습니다.
    3) 기존 기기 토큰 `D1`과 새 access token `A1`으로 각각 `POST /auth/whoami`, `POST /sync/pull`, `POST /sync/push`를 호출합니다.
    4) **음성 확인 (refresh 토큰으로 동기화/인증 엔드포인트 호출)**: `Authorization: Bearer <R1>` 헤더로 `POST /auth/whoami`, `POST /sync/push`, `POST /sync/pull`, `POST /sync/bootstrap` 4개 엔드포인트를 각각 호출합니다.
    5) **정상 회전**: `POST /auth/token` (`{"grantType":"refresh_token","refreshToken":"<R1>"}`)을 호출해 새 쌍 `(A2, R2)`를 받고, `A2`로 `POST /auth/whoami`를 호출합니다.
    6) **음성 확인 (소비된 refresh 토큰 재사용 감지 및 기기 토큰 쌍 전체 회수)**: 이미 사용한 `R1`으로 다시 `POST /auth/token` (`{"grantType":"refresh_token","refreshToken":"<R1>"}`)을 호출한 뒤, 직전에 발급받은 유효 토큰 `A2`로 `POST /auth/whoami` · `POST /sync/pull`을 호출하고, `R2`로 `POST /auth/token`을 호출합니다. 서버 표준출력 JSON 로그에서 `refresh reuse` 경고 로그를 확인합니다.
  - 기대 결과:
    1) `POST /auth/token` password 응답은 HTTP `200`이며 `accessToken`, `accessExpiresAt`(기본 약 `now + 15m` unix ms), `refreshToken`, `refreshExpiresAt`(기본 약 `now + 30d` unix ms), `accountId`, `teamId` 필드를 모두 반환합니다.
    2) 같은 `clientId`로 `/auth/token`을 호출해도 `device.tokenHash`는 바뀌지 않아 기존 기기 토큰 `D1`과 새 `A1` 모두 HTTP `200`으로 동작합니다.
    3) 4)에서 refresh token `R1`을 Bearer로 보낸 `/auth/whoami`, `/sync/push`, `/sync/pull`, `/sync/bootstrap` 요청은 4건 모두 HTTP `401` (`{"code":"unauthorized",...}`)로 거부됩니다.
    4) 5)에서 `R1` 회전은 HTTP `200`으로 새로운 `A2` · `R2`(`A2 != A1`, `R2 != R1`)를 발급하고 `A2`는 정상 인증됩니다.
    5) 6)에서 이미 쓴 `R1`을 다시 제출하면 HTTP `401` (`{"code":"unauthorized",...}`)을 반환하고 그 기기의 access · refresh 토큰 전체(다른 family 포함)가 회수되어, `A2`를 쓴 `/auth/whoami`·`/sync/pull`과 `R2`를 쓴 `/auth/token`이 모두 HTTP `401`로 즉시 거부되며, 서버 로그에 `refresh reuse` 경고가 기록됩니다.
    6) retention `Sweep` 실행 시 만료된 `auth_token` 행이 삭제됩니다.
  - 실패 판정: `/auth/token`이 같은 기기의 기존 기기 토큰 `D1`을 끊거나, refresh token으로 동기화 엔드포인트가 `200`을 주거나, 이미 사용한 refresh token 재전송 시 `200`이 나거나 그 기기의 `A2`·`R2`(또는 다른 family 의 토큰)가 계속 유효하면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.1.1/v07-token-pair-rotation-reuse.log`

- V-8 **대시보드 CSRF(`http.CrossOriginProtection`) · 본인 기기 회수 · 타 계정 기기 회수 404 · 회수 시 토큰 3종 즉시 차단 (음성 확인)**
  - 검증 대상: S-2, S-7, W-3, W-6
  - 실행 방법:
    1) 자동 테스트: `go test -v ./internal/httpapi -run 'TestDashCSRF|TestDashRevokeDevice|TestRevokeDeviceStopsAllTokens'`를 실행합니다.
    2) 계정 `alice`(기기 `dev-alice`: 기기 토큰 `D_a` 및 `/auth/token`으로 발급한 `A_a`, `R_a`)와 계정 `bob`(기기 `dev-bob`: 기기 토큰 `D_b`)을 준비하고, `alice`로 `POST /dash/login`을 호출해 `croffle_dash` 세션 쿠키를 획득합니다.
    3) **음성 확인 (대시보드 세션의 동기화 불가)**: `croffle_dash` 쿠키만 실어 `POST /sync/pull` 및 `POST /sync/push`를 호출합니다.
    4) **음성 확인 (교차 출처 요청 403 거부)**: 유효한 `croffle_dash` 쿠키와 함께 `Origin: https://evil.example`(또는 `Sec-Fetch-Site: cross-site`) 헤더를 붙여 `POST /dash/login`, `POST /dash/logout`, `POST /dash/devices/dev-alice/revoke`, `POST /dash/otp/enroll`을 각각 호출합니다.
    5) **음성 확인 (타 계정 기기 회수 404 거부)**: `alice`의 유효한 세션(동일 출처)으로 `bob`의 기기인 `POST /dash/devices/dev-bob/revoke` 및 없는 기기 `POST /dash/devices/unknown-id/revoke`를 호출한 뒤, `D_b`로 `POST /auth/whoami`를 호출합니다.
    6) **본인 기기 회수 및 토큰 3종 일괄 차단**: `alice` 세션(동일 출처)으로 `POST /dash/devices/dev-alice/revoke`를 호출한 직후, `dev-alice`의 기기 토큰 `D_a`(`POST /sync/pull`), access token `A_a`(`POST /sync/pull`), refresh token `R_a`(`POST /auth/token` refresh)를 각각 호출합니다(CLI `croffle-server token revoke`에 대해서도 동일하게 3종 차단 확인).
  - 기대 결과:
    1) 3)에서 대시보드 세션 쿠키로 보낸 `/sync/pull`, `/sync/push`는 HTTP `401`로 거부됩니다.
    2) 4)에서 교차 출처 헤더가 붙은 모든 `/dash/*` POST 요청은 HTTP `403`으로 거부되며 `dev-alice`는 회수되지 않습니다.
    3) 5)에서 타 계정 기기 `dev-bob`과 없는 기기 ID에 대한 회수 요청은 둘 다 HTTP `404`를 반환하고, `bob`의 기기 토큰 `D_b`는 여전히 HTTP `200`으로 정상 동작합니다.
    4) 6)에서 본인 기기 `dev-alice` 회수는 HTTP `204`를 반환하고 `GET /dash/devices`에서 `state: "revoked"`가 되며, 그 즉시 `D_a`, `A_a`, `R_a` 세 토큰 모두 HTTP `401` (`{"code":"unauthorized",...}`)로 거부됩니다.
  - 실패 판정: 교차 출처 `/dash/*` POST가 `403`이 아닌 상태 코드로 통과하거나, 타 계정 기기 회수가 `404`가 아니거나 실제로 남의 기기를 회수하거나, 기기 회수 후에도 기기 토큰·access·refresh 중 하나라도 계속 동작하면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.1.1/v08-dash-csrf-and-device-revoke.log`

- V-9 **TOTP(`pquerna/otp`) 등록 · 확인 · 3개 로그인 입구 검증 · 스텝 재사용 거부 · 복구 코드 1회성 · CLI/대시보드 해제 (음성 확인)**
  - 검증 대상: S-2, S-4, W-7
  - 실행 방법:
    1) 자동 테스트: `go test -v ./internal/auth ./internal/httpapi ./internal/cli -run 'TestTOTP|TestOTP'`를 실행합니다.
    2) `alice`로 대시보드 로그인 후 `GET /dash/me`(`otpEnabled: false` 확인) → `POST /dash/otp/enroll`을 호출해 `secret`, `otpauthUri`(`otpauth://` URI), `qrPng`(`data:image/png;base64,...`)를 받습니다. 아직 `confirm` 전에 비밀번호만으로 `POST /auth/login`을 호출해 봅니다.
    3) `secret`으로 생성한 6자리 코드 $C_1$(스텝 $S_1$)으로 `POST /dash/otp/confirm` (`{"code":"<C1>"}`)을 호출해 16자 base32 복구 코드 10개(`R_1..R_{10}`)를 받고, `GET /dash/me`와 CLI `./apps/server/bin/croffle-server user otp status alice --db "$TMP_DB"`를 실행합니다. SQLite `otp_recovery_code` 테이블에 평문 코드가 없고 32바이트 해시만 10건 있는지 확인합니다.
    4) **세 로그인 입구(`POST /auth/login`, `POST /auth/token` password, `POST /dash/login`) 검증**:
       - (a) 올바른 비밀번호 + 빈 `otp`로 3개 엔드포인트 각각 호출
       - (b) 올바른 비밀번호 + 틀린 `otp`(`"000000"`)로 호출
    5) **음성 확인 (동일/이전 스텝 TOTP 코드 재사용 거부)**: 다음 30초 스텝 $S_2$의 유효한 6자리 코드 $C_2$로 `POST /auth/login`을 호출해 성공(HTTP `200`)한 직후, 같은 스텝 $S_2$ 안에서 동일한 코드 $C_2$(및 이전 스텝 코드 $C_1$)로 `POST /auth/login`과 `POST /auth/token`을 다시 호출합니다.
    6) **음성 확인 (복구 코드 1회성 소비)**: 복구 코드 $R_1$을 `otp`에 넣어 `POST /auth/token` (`grantType: "password"`)을 호출해 성공(HTTP `200`)한 직후, 같은 복구 코드 $R_1$으로 다시 `POST /auth/login`을 호출합니다. 이어서 다른 복구 코드 $R_2$로 `POST /dash/login`을 호출합니다.
    7) **해제 및 운영자 예외**:
       - CLI `./apps/server/bin/croffle-server token issue alice --db "$TMP_DB"`가 OTP 없이 토큰을 발급하는지 확인합니다.
       - 대시보드 `POST /dash/otp/disable` (`{"password":"...","code":"<유효코드>"}`)로 해제 후 `otpEnabled: false`를 확인하고, 다시 등록·확인한 뒤 운영자 CLI `./apps/server/bin/croffle-server user otp disable alice --db "$TMP_DB"`로 해제하여 비밀번호만으로 `POST /auth/login`이 HTTP `200`을 반환하는지 확인합니다.
  - 기대 결과:
    1) `enroll`은 HTTP `200`으로 base32 `secret`, `otpauthUri`(`otpauth://totp/...`), `qrPng`(PNG data URL)를 반환하며, `confirm` 전(`enabledAt IS NULL`)에는 비밀번호만으로 로그인됩니다.
    2) `confirm`은 HTTP `200`과 `recoveryCodes`(16자 base32 10개)를 반환하고 `GET /dash/me`의 `otpEnabled`가 `true`, CLI `user otp status alice`가 활성 상태를 출력하며, DB에는 SHA-256 해시만 저장됩니다.
    3) 4a)에서 `/auth/login`, `/auth/token`, `/dash/login` 세 곳 모두 빈 `otp`에 대해 HTTP `401`과 `{"code":"otp_required",...}`를 반환합니다. 4b)에서 틀린 코드는 HTTP `401` (`{"code":"unauthorized",...}`)을 반환하며 rate limit 시도로 집계됩니다.
    4) 5)에서 코드 $C_2$의 첫 번째 로그인은 HTTP `200`이고 `lastStep`이 갱신되며, 같은 스텝의 $C_2$ 또는 이전 스텝 $C_1$을 재사용한 두 번째 요청은 HTTP `401` (`unauthorized`)로 거부됩니다.
    5) 6)에서 복구 코드 $R_1$의 첫 사용은 HTTP `200`이고 `usedAt`이 기록되며, $R_1$을 다시 쓰면 HTTP `401` (`unauthorized`)로 거부되고 미사용 $R_2$는 HTTP `200`으로 통과합니다.
    6) 7)에서 대시보드 `disable`과 CLI `user otp disable` 모두 TOTP를 해제하여 이후 OTP 없이 로그인이 성공합니다.
  - 실패 판정: 세 로그인 입구 중 한 곳이라도 빈 OTP에 `401 otp_required`를 반환하지 않거나, 같은 스텝의 TOTP 코드가 2번 통과하거나, 복구 코드가 2번 통과하거나, DB에 복구 코드 평문이 저장되거나, CLI `user otp disable`이 동작하지 않으면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.1.1/v09-totp-and-recovery.log`

- V-10 **충돌 응답 `differentFields` 계산 규칙 · 적용/거부 불변성 · 공유 픽스처 13번 컨포먼스**
  - 검증 대상: S-8, S-10, W-8
  - 실행 방법:
    1) 단위 테스트: `go test -v ./internal/protocol ./internal/httpapi -run 'TestDifferentFields|TestConflictDifferentFields'`를 실행합니다.
    2) 컨포먼스 테스트: `packages/sync-fixtures/scenarios/13-push-conflict-different-fields.json`을 확인하고 `go test -v ./internal/conformance -run TestScenarios` 및 `node --test apps/desktop/test/sync-protocol.test.ts`를 실행합니다.
    3) 단위 테스트와 픽스처 13번에서 다음 4가지 케이스를 검증합니다:
       - (a) 같은 행(`baseVersion` 불일치)에 대해 서버와 클라이언트의 payload 필드(예: `title`, `startDate`)가 다른 충돌 → `differentFields`에 변경된 필드 이름만 오름차순 정렬된 배열(예: `["startDate", "title"]`)로 반환됨
       - (b) 장부 필드(`version`, `updatedAtMs`, `lastWriterId`, `ownerId`)만 다르고 모든 payload · `teamId` · `deletedAt` 값이 의미상 같은 충돌(JSON 키 순서 차이, `1` vs `1.0` 숫자 표기, 없는 키 vs `null` 포함) → `differentFields: []`(빈 배열)로 반환됨
       - (c) 한쪽이 삭제(`deletedAt` 변경)하거나 `teamId`를 바꾼 충돌 → `"deletedAt"` 또는 `"teamId"`가 `differentFields`에 포함됨
       - (d) 충돌 시 서버 행(`sync_row`)과 `applied_op`는 바뀌지 않으며, 클라이언트가 `baseVersion = serverVersion`, `version = serverVersion + 1`로 재전송하면 `accepted`됨
  - 기대 결과:
    1) `go test ./internal/conformance`에서 `12-row-codec`(데스크톱 전용, SKIP)을 뺀 `01`~`13`번 시나리오가 모두 PASS하고(V-1 과 같은 2026-11-16 변경), 데스크톱 `sync-protocol.test.ts`도 13개 시나리오를 모두 읽고 PASS합니다.
    2) 모든 충돌 응답 원소에 `differentFields` 배열(차이가 없을 때는 `[]`)이 포함되며, 장부 필드 4종(`version`, `updatedAtMs`, `lastWriterId`, `ownerId`)은 어떤 경우에도 `differentFields`에 나타나지 않습니다.
    3) `packages/sync-fixtures/README.md`에 `"desktop: work item 1.3.0#15"` 문장이 유지된 채 13번 시나리오 설명이 추가되어 있습니다.
  - 실패 판정: `differentFields`가 사전순 정렬되지 않았거나, 차이가 없을 때 `[]` 대신 `null`/생략되거나, `version`·`updatedAtMs`·`lastWriterId`·`ownerId`가 포함되거나, 없는 키와 `null`을 다르게 판정하거나, 충돌한 커밋이 서버 행을 덮어쓰면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.1.1/v10-conflict-different-fields.log`

- V-11 **Prometheus `/metrics` 포트 격리 · 기본 꺼짐(음성 확인) · 지표 집계 · 개인정보/고카디널리티 라벨 부재 (음성 확인)**
  - 검증 대상: S-9, S-11, W-9
  - 실행 방법:
    1) 자동 테스트: `go test -v ./internal/metrics ./internal/httpapi ./internal/config -run 'TestMetrics'`를 실행합니다.
    2) **음성 확인 (`CROFFLE_METRICS_LISTEN` 미설정 / 빈 값)**: 기본 설정(`CROFFLE_LISTEN=127.0.0.1:18787`, `CROFFLE_METRICS_LISTEN=""`)으로 서버를 기동한 뒤, 프로세스가 연 TCP 리스닝 소켓 수(`ss -ltnp | grep croffle-server` 또는 `/proc/$PID/net/tcp`)를 확인하고 `curl -i http://127.0.0.1:18787/metrics`를 호출합니다.
    3) **별도 주소 활성화 (`CROFFLE_LISTEN=127.0.0.1:18787`, `CROFFLE_METRICS_LISTEN=127.0.0.1:19090`)**:
       - (a) 본 API 주소 `curl -i http://127.0.0.1:18787/metrics` 및 metrics 주소의 다른 경로 `curl -i http://127.0.0.1:19090/health`를 호출합니다 (음성 확인).
       - (b) 사용자 `alice`(계정 ID `acct-priv-999`)로 로그인 성공(`ok`), 틀린 비밀번호(`invalid`), OTP 누락(`otp_required`), 비활성 계정(`disabled`), rate limit(`rate_limited`), refresh 재사용 1회, `POST /sync/push`(accepted 1건, conflict 1건, retry 1건), `POST /sync/pull`(변경 1건 이상), `POST /sync/bootstrap`(행 1건 이상), 동적 라우트 `POST /dash/devices/dev-secret-777/revoke`, 미등록 경로 `GET /no-such-route/alice/acct-priv-999` 요청을 차례로 발생시킵니다.
       - (c) `curl -sf http://127.0.0.1:19090/metrics` 응답 본문을 저장하여 지표 이름과 라벨 값을 검사합니다.
  - 기대 결과:
    1) 2)에서 `CROFFLE_METRICS_LISTEN`이 비어 있으면 프로세스는 `127.0.0.1:18787` 단 1개의 포트만 리스닝하며, `http://127.0.0.1:18787/metrics`는 Prometheus 지표를 반환하지 않습니다(`# HELP croffle_` 0건).
    2) 3a)에서 `CROFFLE_METRICS_LISTEN=127.0.0.1:19090`을 켜도 본 API `127.0.0.1:18787/metrics`에는 `/metrics`가 없고, metrics 서버의 `127.0.0.1:19090/health`는 HTTP `404`를 반환하며 오직 `GET http://127.0.0.1:19090/metrics`만 HTTP `200` Prometheus 텍스트를 반환합니다.
    3) 3c)에서 다음 `croffle_` 지표와 Go/process 기본 컬렉터 지표가 모두 출력되고 발생시킨 횟수만큼 카운터가 증가합니다:
       - `croffle_http_requests_total{code=...,method=...,route=...}` 및 `croffle_http_request_duration_seconds_{bucket,sum,count}`
       - `croffle_sync_push_commits_total{result="accepted"}`, `{result="conflict"}`, `{result="retry"}`
       - `croffle_sync_pull_changes_total`, `croffle_sync_bootstrap_rows_total`
       - `croffle_auth_attempts_total{result="ok|invalid|otp_required|disabled|rate_limited",surface="sync|token|dashboard"}`
       - `croffle_auth_refresh_reuse_total`
    4) **라벨 음성 확인**: `/metrics` 응답 전체에서 계정 ID(`acct-priv-999`), 사용자명(`alice`), 기기 ID(`dev-secret-777`), IP(`127.0.0.1`) 문자열이 `0`건(`! grep -E "acct-priv-999|alice|dev-secret-777|127\.0\.0\.1"`)이며, 동적 경로의 `route` 라벨은 패턴(`POST /dash/devices/{id}/revoke`)으로, 미등록 경로(catch-all `/` 포함)는 `route="other"`로만 기록됩니다.
  - 실패 판정: `CROFFLE_METRICS_LISTEN`이 비어 있을 때 추가 포트가 열리거나, 본 API 포트에서 `/metrics`가 열리거나, 필수 `croffle_*` 지표 중 하나라도 누락되거나, 라벨에 계정 ID·사용자명·기기 ID·IP·원시 URL 경로가 1건이라도 노출되면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.1.1/v11-prometheus-metrics.log`

- V-12 **`GET /health` 프로토콜 범위(`{min: 1, max: 1}`) 불변 및 `features` 목록 확인**
  - 검증 대상: S-10, S-11, W-6, W-8
  - 실행 방법:
    1) `go test -v ./internal/httpapi -run TestHealth`를 실행합니다.
    2) 기동된 0.1.1 서버에 `curl -sf http://127.0.0.1:18787/health`를 호출하여 응답 JSON 구조를 검증합니다.
  - 기대 결과: HTTP `200` 응답 JSON에 `protocol: {"min": 1, "max": 1}`, `db: "ok"`, `server`, `time`(unix ms 정수), `retention: {"oldestCommitSeq": ..., "purgedThroughSeq": ...}`이 0.1.0과 동일하게 존재하고, `features`가 정확히 `["conflict-fields", "device-token-ttl", "token-pair", "totp"]`(이 순서, 4개)입니다.
  - 실패 판정: `protocol.min` 또는 `protocol.max`가 `1`이 아니거나, 기존 0.1.0 `/health` 필드가 누락·변경되었거나, `features`가 위 4개와 하나라도 다르면(누락 · 추가 · 순서) 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.1.1/v12-health-and-features.log`

- V-13 **데스크톱 1.3.0 클라이언트 ↔ 서버 0.1.1 무수정 종단간 동기화 호환성 (`apps/desktop/scripts/sync-harness`)**
  - 검증 대상: S-4, S-8, S-10, S-11, W-6, W-7, W-8
  - 실행 방법:
    `[수동 · Linux]` 데스크톱 1.3.0의 실서버 검증 하네스(`apps/desktop/scripts/sync-harness/rc-server.sh` 및 `apps/desktop/scripts/sync-harness/README.md`) 절차를 0.1.1 서버에 대해 수행합니다:
    1) `make -C apps/server docker VERSION=0.1.1-test`(또는 `./apps/server/bin/croffle-server`)로 0.1.1 서버를 `127.0.0.1:8787`에 띄우고, `apps/desktop/scripts/sync-harness/rc-server.sh`와 동일하게 `croffle-server user create alice --password-stdin` 및 `croffle-server token issue alice --device-name device-b`로 계정과 기기 B 토큰을 준비합니다.
    2) 전혀 수정하지 않은 데스크톱 1.3.0(`pnpm dev`, 기기 A)에서 서버 `http://127.0.0.1:8787`을 등록하고 ID/PW 로그인 후 5개 테이블(`schedule`, `tag`, `schedule_tags`, `schedule_reminder`, `schedule_exception`) 데이터를 생성·동기화합니다.
    3) 별도 `--user-data-dir` 프로필로 띄운 데스크톱 1.3.0(기기 B)에서 토큰 붙여넣기(`POST /auth/whoami`)로 로그인해 `bootstrap`으로 5개 테이블이 모두 내려오는지 확인합니다.
    4) 기기 A와 기기 B에서 동일 일정을 서로 다른 제목으로 수정한 뒤 양쪽을 동기화하여, 0.1.1 서버가 반환하는 `Conflict.differentFields` 선택 필드가 있어도 데스크톱 1.3.0의 `push.ts` LWW 판정이 오류 없이 수렴하는지 확인합니다.
    5) 대시보드에서 `alice` 계정에 TOTP를 활성화한 뒤, 기기 A에서 로그아웃 후 다시 ID/PW 로그인을 시도하여 1차 `401 otp_required` 수신 → 데스크톱 동기화 탭의 OTP 입력 칸 표시 → 6자리 TOTP 코드 입력 후 2차 로그인 및 동기화가 완료되는지 확인합니다.
  - 기대 결과: 데스크톱 1.3.0 코드 변경 `0`줄 상태에서 `/health` 프로토콜 확인, ID/PW 로그인, 토큰 붙여넣기(`whoami`), TOTP 2단계 로그인(`401 otp_required` → OTP 재시도), 5개 테이블 `bootstrap`/`push`/`pull`, `differentFields`가 포함된 충돌 응답에서의 LWW 수렴까지 모두 성공하며, 동기화 직후 기기 A·B의 5개 테이블 내용이 일치하고 양쪽 `sync_outbox`는 `0`건(핑퐁 없음), `apps/desktop/dev/logs/main.log`의 동기화 오류는 `0`건입니다.
  - 실패 판정: 데스크톱 1.3.0이 `/health`의 `features`나 `Conflict.differentFields` 때문에 예외를 내거나, TOTP 활성화 계정으로 데스크톱 로그인이 안 되거나, 두 기기 데이터가 수렴하지 않거나 `sync_outbox`에 행이 남으면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.1.1/v13-desktop-130-compat-sync.log`

- V-14 **대시보드 웹 UI 통합 검증 (기기 만료 표시 · 기기 회수 · TOTP 등록/QR/복구 코드/로그인/해제)**
  - 검증 대상: S-4, S-5, S-7, W-3, W-5, W-7
  - 실행 방법:
    `[수동 · Linux]` `make -C apps/server build-all`로 빌드한 바이너리를 기동하고 브라우저에서 대시보드(`http://127.0.0.1:8787`)를 열어 차례로 수행합니다:
    1) `alice`로 로그인 → 기기 표에서 기기 목록과 각 기기의 만료 시각(또는 만료 없음) 열 확인 → 기기 1개의 "회수" 버튼 클릭 → 확인 창 승인 후 상태가 `revoked`로 바뀌는지 확인합니다.
    2) TOTP 등록 실행 → 서버가 반환한 PNG data URL QR 이미지와 secret 표시 확인 → 6자리 코드 입력 및 확인(`confirm`) → 복구 코드 10개가 화면에 1회 표시되는지 확인합니다.
    3) 로그아웃 후 다시 로그인 시도 → `otp_required`에 의해 OTP 코드 입력 칸이 나타나고 6자리 TOTP 코드(또는 복구 코드) 입력 시 로그인되는지 확인합니다.
    4) 대시보드에서 비밀번호와 유효한 코드를 입력해 TOTP를 해제(`otpEnabled: false`)합니다.
  - 기대 결과: 외부 스크립트/폰트/QR 생성 네트워크 요청 없이 대시보드 안에서 기기 만료 확인, 확인 창을 거친 본인 기기 회수, TOTP QR 표시·확인·복구 코드 10개 안내·OTP 로그인·해제가 모두 정상 동작하며, 페이지의 JS 예외(pageerror)가 `0`건이고, 브라우저 콘솔의 리소스 실패는 의도된 응답 두 가지(시작할 때 로그인 확인 `GET /dash/me` 401, 코드가 필요한 계정의 `POST /dash/login` 401 `otp_required`)뿐입니다(`favicon.ico` 404 없음). (2026-11-16 변경: 처음 문구 "콘솔 오류 0건"은 S-4 가 요구하는 `otp_required` 401 과 충돌해 사용자 결정 "V-14는 (가)"로 고침)
  - 실패 판정: 대시보드에서 기기 만료 열이나 회수 확인 창이 누락되었거나, QR 이미지가 깨지거나, 복구 코드 10개가 표시되지 않거나, `otp_required` 시 로그인 폼에 코드 입력 칸이 나타나지 않거나, JS 예외가 1건이라도 있거나, 위 두 가지 밖의 리소스 실패가 콘솔에 남으면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.1.1/v14-dashboard-ui.log`

- V-15 **영/한 `README` 대칭 · 호환 표 · 릴리스 노트(`docs/server/release/0.1.1.md`) 검증**
  - 검증 대상: S-1, S-12, S-13, W-1, W-2, W-10
  - 실행 방법:
    1) `apps/server/README.md`와 `apps/server/README.ko.md`에서 다음 6개 항목의 존재 및 영/한 내용 대칭을 확인합니다:
       - 계정 생성 `--password-stdin`의 터미널용(`docker exec -it`), 파이프용(`printf '%s' "$PW" | docker exec -i …`), Windows Git Bash용(`winpty docker exec -it …`) 예시
       - 설정 환경변수 표의 새 변수(`CROFFLE_DEVICE_TOKEN_TTL`, `CROFFLE_ACCESS_TOKEN_TTL`, `CROFFLE_REFRESH_TOKEN_TTL`, `CROFFLE_METRICS_LISTEN`, `CROFFLE_DASH_TRUSTED_ORIGINS`)
       - 인증 절(토큰 3종: 기기 토큰 · access · refresh, 만료 설정 시 데스크톱 1.3.0은 만료마다 재로그인해야 한다는 주의사항, TOTP 등록/해제 및 복구 코드, CLI `token issue --expires-in/--no-expiry` 및 `user otp status/disable`)
       - metrics 절(`CROFFLE_METRICS_LISTEN` 설정법, `croffle_*` 지표 목록, 외부에 포트를 열지 말 것)
       - 의존성 절("검증된 소수 라이브러리 허용" 원칙과 모듈 목록)
       - 호환 표(서버 `0.1.0` / `0.1.1` ↔ sync protocol `1` ↔ 데스크톱 `1.3.0`, 0.1.1의 새 기능은 데스크톱 1.3.0이 아직 사용하지 않음 명시)
    2) `docs/server/release/0.1.1.md` 파일의 존재 및 `.github/release-notes.md`의 Server 템플릿 섹션(`# croffle-server v0.1.1`, `Image: ghcr.io/team-croffle/croffle-server:0.1.1 · tag server-v0.1.1 · protocol 1–1`, `## Overview`, `## Upgrade notes`, `## Compatibility`, `## Known limitations`, `<details><summary>한국어</summary>...</details>`) 준수 여부를 확인합니다.
  - 기대 결과: `apps/server/README.md`와 `apps/server/README.ko.md` 양쪽에 위 6개 항목이 대칭으로 기술되어 있고, 0.1.0의 낡은 문구("Tokens do not expire", "The dashboard is read-only, and there are no metrics yet")가 0.1.1 동작에 맞게 고쳐져 있으며, `docs/server/release/0.1.1.md`가 영/한 섹션을 모두 갖추고 있습니다.
  - 실패 판정: 영/한 README 중 한쪽이라도 위 6개 항목 중 누락이 있거나, 영/한 내용이 서로 어긋나거나, 0.1.0 한계 문구가 수정되지 않고 모순되게 남아 있거나, `docs/server/release/0.1.1.md`가 없거나 템플릿 형식을 따르지 않으면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.1.1/v15-docs-and-release-note.log`


## 우려
1. **TOTP secret 평문 저장.** DB 파일을 가져간 사람은 OTP 를 만들 수 있습니다. 다만 비밀번호 해시도 함께 있어야 로그인할 수 있고, 0.1 은 이미 "DB 파일에 접근할 수 있으면 운영자"를 전제로 합니다(CLI). 암호화하려면 운영자가 관리할 키(`CROFFLE_SECRET_KEY`)가 새로 필요해 이번에는 넣지 않았습니다. 사용자 결정(2026-11-16): "TOTP 암호화는 바로 다음버전으로 넣자." — 서버 0.2.0 으로 로컬 로드맵에 적었습니다.
2. **`CROFFLE_DEVICE_TOKEN_TTL` 을 켜면 데스크톱 1.3.0 은 만료 때마다 다시 로그인해야 합니다.** 1.3.0 은 refresh 를 쓰지 않기 때문입니다. 기본은 꺼짐이고 README 에 이 점을 적습니다. 사용자: "만료는 어쩔 수 없고" — 수용.
3. **`client_golang` 의 전이 의존성**(protobuf, procfs 등)으로 바이너리가 커집니다(수 MB 예상). metrics 를 끈 서버에도 코드는 들어갑니다. 빌드 태그로 빼는 방법은 복잡도에 비해 이득이 작아 쓰지 않습니다. 구현 때 크기 차이를 재서 노트에 적겠습니다. 사용자: "바이너리는 수용".
4. **대시보드가 읽기 전용이 아니게 됩니다.** 0.1 설계 기록의 "anything that changes state is a CLI command" 를 "본인 계정에 한한 쓰기는 대시보드에서"로 넓힙니다. intent 계획 3("대시보드 · CLI 의 기기 목록과 회수")과 질의서 6(TOTP)에서 나온 것입니다. 관련 코드 주석을 함께 고칩니다.
5. **`differentFields` 는 데스크톱의 행 인코딩에 기댑니다.** 데스크톱이 보낸 행에 없는 키는 `null` 과 같다고 보므로, 클라이언트가 필드를 빼고 보내면 "다르다"고 나오지 않을 수 있습니다. 판정은 데스크톱이 하므로 정보용 필드로만 문서화합니다.
6. **`CrossOriginProtection` 은 `Origin` 을 요청의 `Host` 와 비교합니다.** 앞단 프록시가 `Host` 를 바꿔 넘기면 대시보드의 쓰기 요청이 403 이 됩니다. README 의 프록시 예시(이미 `Host` 를 넘김)를 확인하고, 필요하면 신뢰할 출처를 `CROFFLE_DASH_TRUSTED_ORIGINS`(쉼표 구분, 기본 빈 값)로 더할 수 있게 합니다. 이 변수는 W-3 에 포함합니다.

## 범위 밖
- 데스크톱이 `/auth/token` · refresh · `differentFields` 를 쓰는 작업(이후 데스크톱 패치).
- 서버 쪽 승자 판정 · 필드 단위 병합(A2), JWT · OAuth2 인가 서버(A1).
- TOTP secret 암호화(서버 0.2.0), WebAuthn, 외부 로그인(GitHub OIDC).
- 다중 팀 · 팀 관리 · 공유 취소(서버 0.2.0 · 0.2.1).
- metrics 의 인증 · TLS(별도 주소를 내부망에만 두는 것으로 대신함).
