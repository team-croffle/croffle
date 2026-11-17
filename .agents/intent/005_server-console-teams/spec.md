---
intent: "005"
status: accepted
approvals:
  - { role: user, by: bluenyang, at: "2026-11-18", words: "진행", rev: "859785a", kind: summary }
summary_for_user: |
  서버 0.2.0 을 작업 단위 8개로 나눴습니다: TOTP secret 암호화(AES-GCM, 새 의존성 없음) → 키 CLI(generate · rotate) → 운영자 역할(0006) → 다중 팀 CLI → 팀 범위 따라잡기(0007) → 운영 API `/dash/admin/*` → 대시보드 운영 화면 → 문서.
  intent 와 다른 점: 로드맵의 "scope별 커서"를 클라이언트 쪽 protocol 확장으로 열지 않고, 서버가 기기별로 "이 팀을 처음부터 받았는지"를 기록해 새 팀에 들어간 기기에 410 을 주는 방식으로 했습니다. 데스크톱 1.3.x 가 이미 410 이면 스냅샷을 다시 받으므로 데스크톱 수정이 없습니다. 대신 팀에 들어갈 때마다 그 계정의 기기가 전체 스냅샷을 받습니다(우려 3).
  PM 이 더한 규칙: 암호화된 secret 이 있는데 키가 없거나 다르면 서버가 뜨지 않습니다(키 없는 0.1.1 배포는 경고만). 마지막 운영자는 해제 · 비활성화할 수 없습니다. 팀 삭제는 0.2.1 로 미룹니다.
  검증 기준 V-1~V-13 은 qa-designer 가 쓰고 PM 이 고쳤습니다(qa 가 짚은 설계 문제 10개 반영). 수동 확인은 대시보드 운영 화면과 데스크톱 1.3.1 실동기화(업그레이드 직후 재스냅샷 없음 · 팀 합류 410 · 팀 제외)입니다.
---
## 요구 해석
- S-1 운영자가 키(`CROFFLE_SECRET_KEY`, 또는 키 파일 `CROFFLE_SECRET_KEY_FILE`)를 주면, 서버는 TOTP secret 을 그 키로 암호화해 저장합니다. 키가 설정된 서버는 시작할 때 평문 secret 을 모두 암호화본으로 바꾸며, 사용자는 OTP 를 다시 등록하지 않습니다. ← 바라는 결과 2, 계획 1
- S-2 키가 없는 서버도 0.1.1 처럼 뜹니다. secret 은 평문으로 저장되고, 시작 로그(warn)와 운영자 대시보드에 "평문 저장" 경고가 나옵니다. 단, DB 에 암호화된 secret 이 있는데 키가 없거나 키가 맞지 않으면 서버는 뜨지 않고 이유를 알립니다(OTP 를 검증할 수 없게 되는 것보다 낫다는 PM 해석). ← 바라는 결과 3, 질의서 A3
- S-3 운영자는 CLI 로 키를 바꿀 수 있습니다: 이전 키로 풀고 새 키로 다시 암호화합니다. 새 키를 만드는 CLI 도 둡니다. ← 계획 1, 질의서 A4
- S-4 계정에 운영자 여부가 생깁니다. 첫 운영자는 CLI 로 정하고, 운영자는 대시보드에서 다른 계정을 운영자로 만들거나 풀 수 있습니다. 마지막 운영자를 풀거나 비활성화할 수 없습니다. 대시보드의 운영 API 는 운영자 세션만 받고, 운영자가 아니면 `403` 입니다. ← 계획 2
- S-5 운영자는 팀을 만들고 이름을 바꾸며 멤버를 넣고 뺄 수 있습니다(CLI 와 대시보드). `team-default` 는 그대로 남습니다. ← 계획 3, 바라는 결과 4, 질의서 A2, 제약 4
- S-6 계정이 새 팀에 들어가면, 그 계정의 기기는 다음 pull 에서 그 팀의 기존 공유 행까지 받습니다. 배포된 데스크톱 1.3.0 · 1.3.1 은 코드를 바꾸지 않아도 됩니다: 서버가 그 기기의 pull 에 `410 cursor_too_old` 를 돌려주고, 데스크톱은 이미 그때 스냅샷(bootstrap)을 다시 받습니다(`apps/desktop/src/main/sync/pull.ts` `pull`). 0.2.0 으로 올린 직후에는 어떤 기기도 스냅샷을 다시 받지 않습니다. ← 계획 3, 바라는 결과 4 · 5, 제약 1
- S-7 계정이 팀에서 빠지면 서버는 그때부터 그 팀의 행을 그 계정에 내주지 않고, 그 팀으로 가는 push 는 지금처럼 `403 foreign_team` 입니다. 데스크톱 1.3.x 는 그 commit 만 실패로 두고 동기화를 계속합니다(`push.ts` `sendIsolating`). 이미 받아 둔 행의 처리는 데스크톱의 일입니다. ← 질의서 PM 해석
- S-8 운영자는 지금 CLI 가 하는 운영 작업을 모두 대시보드에서 할 수 있습니다: 계정(목록 · 생성 · 비활성/활성 · 비밀번호 재설정 · OTP 상태와 해제 · 운영자 지정), 기기 토큰(모든 계정의 목록 · 발급 · 회수), 팀(목록 · 생성 · 이름 변경 · 멤버 추가/제거), 서버 정보(`db info`) · vacuum · 커밋 로그. 발급한 토큰은 응답에 한 번만 나옵니다. 본인 화면(기기 · 보안 · 행 · 커밋)은 그대로입니다. ← 계획 4, 바라는 결과 1, 질의서 A1
- S-9 데스크톱 1.3.0 · 1.3.1 은 바꾸지 않아도 0.2.0 서버와 동기화됩니다. protocol 1 의 요청 · 응답에서 빠지거나 뜻이 바뀌는 필드가 없고, `/health` 의 `protocol` 은 `{min: 1, max: 1}` 그대로입니다. 새 기능은 `features` 로 알립니다. ← 바라는 결과 5, 제약 1
- S-10 서버는 바이너리 하나와 SQLite 하나로 뜹니다. 새 설정은 모두 선택이고, 키는 외부 서비스를 요구하지 않습니다. 새 Go 의존성은 없습니다(암호화는 표준 `crypto/aes` · `crypto/cipher`). ← 제약 2 · 3
- S-11 README(en · ko)에 운영자 · 키(생성 · 설정 · 교체 · 백업 주의) · 팀 안내와 호환 표가 생기고, 릴리스 노트 `docs/server/release/0.2.0.md` 가 생깁니다. ← 계획 5

## 설계

### 의존성
새 Go 의존성 없음. 대시보드(`apps/server/web`)에도 npm 의존성을 더하지 않습니다(라우터 없이 지금처럼 탭 상태로 화면을 바꿉니다).

### W-1 TOTP secret 암호화 (`internal/secretbox` 새 패키지, `internal/config`, `internal/store/sqlite`, `cmd`)
- 키: 32바이트, 설정값은 base64(표준, 패딩 허용). `CROFFLE_SECRET_KEY` 와 `CROFFLE_SECRET_KEY_FILE` 을 둘 다 주면 시작 오류입니다. 파일은 끝의 줄바꿈을 떼고 읽습니다. 길이가 32바이트가 아니면 시작 오류입니다.
- 형식: `v1:<keyId>:<base64(nonce ‖ ciphertext)>`. AES-256-GCM, nonce 12바이트(무작위), AAD 는 `accountId`(다른 계정 행으로 옮겨 붙인 secret 은 풀리지 않음). `keyId` 는 `hex(SHA-256(key))[:8]`. 평문 secret 은 base32 라 `:` 가 없으므로 접두로 구분합니다. 마이그레이션 SQL 은 없습니다(열 모양은 그대로 TEXT).
- store 경계: sqlite 의 `FindOTP` · `StartOTP` · `EnableOTP` 가 `SecretBox` 인터페이스(`Seal(accountID, plain)`, `Open(accountID, stored)`)를 거칩니다. 키가 없으면 통과(평문) 구현을 씁니다. auth 서비스는 바뀌지 않습니다(평문 secret 을 받음). 지금 `EnableOTP` 는 `WHERE secret = ?` 로 저장값을 직접 비교하는데, 암호화본은 nonce 때문에 매번 달라지므로 같은 트랜잭션에서 저장값을 `Open` 해 평문끼리 비교하도록 바꿉니다(`store/sqlite/otp.go:55`).
- 시작 순서(`db migrate` 뒤, listen 전): 저장된 secret 을 모두 훑어
  - 키 있음: 평문 행 → 한 트랜잭션에서 암호화본으로 바꿈(로그 info `secret key: encrypted N one-time code secrets`). `keyId` 가 다른 행이 있으면 시작 실패 `stored one-time code secrets were sealed with another key (id X); run secret rotate`.
  - 키 없음: 암호화된 행이 있으면 시작 실패 `stored one-time code secrets are encrypted but no secret key is set`. 평문 행이 있거나 없거나 warn `no secret key: one-time code secrets are stored in plain text` (행이 없어도 경고합니다 — 앞으로 등록되는 secret 도 평문이기 때문).
- `db info` 와 운영자용 `GET /dash/admin/info` 에 키 상태와 secret 개수를 더합니다. JSON 필드 `secretKey: "set" | "missing"`, `otpSecretsPlain`, `otpSecretsEncrypted`(정수), CLI 표 라벨 `secret key`, `otp secrets (plain)`, `otp secrets (encrypted)`. (qa-designer 지적 5)

### W-2 키 CLI (`internal/cli`)
- OTP 상태(`user otp status`, 운영 API 의 계정 목록)는 secret 을 풀지 않고 `enabledAt` 만으로 판정합니다(store 에 `OTPStatus` 를 더함). 키가 필요한 CLI 는 `secret rotate` 하나이고, 다른 CLI 명령은 W-1 의 시작 검사를 거치지 않습니다. (qa-designer 지적 6)
- `secret generate` — 새 키를 base64 한 줄로 stdout 에 씁니다(파일에 저장하는 것은 운영자 몫, README 에 `> key` 예시).
- `secret rotate --new-key-file <path>` | `--new-key-stdin`(둘 중 하나, 둘 다 주면 오류) — 지금 설정된 키(환경 변수 · 파일)로 모두 풀고 새 키로 한 트랜잭션에서 다시 암호화합니다. 지금 키가 없으면 평문 행을 새 키로 암호화합니다(= 처음 켜기). 하나라도 풀리지 않으면 아무것도 바꾸지 않고 실패합니다. 끝나면 "이제 설정을 새 키로 바꾸고 다시 시작하라"고 안내합니다.
- 서버가 떠 있는 동안 실행해도 DB 는 깨지지 않습니다(SQLite 단일 쓰기 트랜잭션). 다만 서버는 다시 시작하기 전까지 옛 키를 쓰므로, 그 사이의 새 OTP 등록은 옛 키로 저장되고 다음 시작이 `keyId` 불일치로 실패합니다. 서버가 떠 있는지 확실히 알 수단이 없어서, README 는 "서버를 멈춤 → rotate → 설정을 새 키로 → 시작" 순서를 안내하고, rotate 는 실행할 때마다 이 순서를 경고 문구로 출력합니다(PM 결정, 우려 2).

### W-3 운영자 역할 (마이그레이션 `0006_operator.sql`, `internal/store`, `internal/auth`, `internal/cli`, `internal/httpapi`)
- `ALTER TABLE account ADD COLUMN operatorAt INTEGER` — NULL 이면 운영자 아님. 기존 계정은 모두 NULL.
- CLI: `user create … --operator`, `user operator grant <user>` / `revoke <user> [--force]`, `user disable <user> [--force]`, `user list` 의 `STATE` 다음에 `OPERATOR` 열(`yes` / `no`) · `--json` 에 `operator` (bool). (qa-designer 지적 7)
- 규칙(store 한 트랜잭션에서 확인): 활성 운영자가 하나뿐이면 그 계정의 `revoke` · `disable` 은 `ErrLastOperator`. CLI 는 `--force` 없이 거부합니다(운영자가 0명이 되면 대시보드 운영 화면을 아무도 못 씀; CLI 는 DB 직접 접근이라 복구 가능).
- 대시보드: `GET /dash/me` 에 `operator: bool`. `requireOperator` 미들웨어는 세션 계정이 활성 운영자인지 매 요청 확인하고, 아니면 `403 not_operator`. 운영 API 는 모두 `/dash/admin/` 아래에 두고 기존 `dashRoute`(CSRF) 를 탑니다.
- 운영자가 되어도 동기화 권한은 바뀌지 않습니다(남의 행을 pull 할 수 없음). 대시보드의 행 · 커밋 내용 열람은 W-6 의 커밋 로그 한 곳뿐입니다.

### W-4 다중 팀 (`internal/store`, `internal/cli`)
- 마이그레이션 없음: 팀 생성 · 이름 변경 · 멤버 추가 · 제거는 지금 스키마로 됩니다.
- store: `CreateTeam(id?, name)`(id 를 주지 않으면 `team-<무작위 10자>`), `RenameTeam`, `RemoveTeamMember`, `ListTeamMembers`. 팀 삭제는 하지 않습니다(공유 행이 그 팀을 가리킴; intent 범위 밖, 우려 4).
- CLI: `team create <name> [--id ID]`, `team rename <team-id> <name>`, `team remove-member <team-id> <username>`, `team members <team-id>`. `team add-member` 는 그대로.
- 이름은 1–64자, 앞뒤 공백 제거, 중복 허용(id 가 식별자).

### W-5 팀 범위 따라잡기 (마이그레이션 `0007_device_scope.sql`, `internal/store/sqlite`, `internal/httpapi`, `internal/protocol`, `packages/sync-fixtures`)
- 문제: pull 커서는 기기당 serverSeq 하나입니다. 새 팀에 들어간 계정의 기기는 그 팀의 예전 commit(지금 커서 이하)을 받지 못합니다.
- 테이블 `device_scope(deviceId TEXT REFERENCES device ON DELETE CASCADE, teamId TEXT REFERENCES team ON DELETE CASCADE, coveredAt INTEGER NOT NULL, PRIMARY KEY (deviceId, teamId)) STRICT, WITHOUT ROWID` — "이 기기는 이 팀의 행을 처음부터 받았다". 마이그레이션이 **지금 있는 모든 기기 × 그 계정의 팀**으로 채웁니다(올린 직후 재스냅샷 없음, S-6).
- pull: 계정의 팀 중 그 기기의 `device_scope` 에 없는 팀이 있으면 `410 cursor_too_old`(기존 코드 · 문구 재사용, `details.reason = "team_scope"` 를 선택 필드로). 데스크톱 1.3.x 는 bootstrap 으로 넘어갑니다.
- bootstrap: 첫 페이지에서 그때의 계정 팀 목록을 커서(`BootstrapCursor`)에 함께 담고, 마지막 페이지(`hasMore=false`)를 줄 때 **커서의 팀 목록과 그 순간 계정 팀의 교집합**으로 `device_scope` 를 채웁니다(도중에 빠진 팀을 다시 심지 않음). 한 페이지로 끝나는 스냅샷은 그 요청의 계정 팀으로 채웁니다. 스냅샷 도중에 들어간 팀은 기록되지 않아 다음 pull 에서 다시 410 → 한 번 더 스냅샷(드묾, 정확성 우선). (qa-designer 지적 4)
- 멤버 제거: 그 계정의 모든 기기에서 그 팀의 `device_scope` 행을 지웁니다(다시 들어오면 다시 따라잡기).
- 새 기기는 **만들어질 때 계정이 속한 팀**으로 `device_scope` 를 채웁니다(로그인 · `token issue` · `/auth/token` 의 기기 생성, A-2). 아직 아무것도 받지 않은 기기는 그 팀들을 처음부터 받을 수 있으므로 따라잡을 것이 없고, 그 뒤에 들어간 팀만 410 의 대상입니다. 이 규칙이 없으면 bootstrap 없이 push 후 그 `serverSeq` 로 pull 하는 기존 픽스처 04 가 410 을 받아 protocol 1 의 답이 바뀝니다.
- 커서 0 의 pull(`lastServerSeq == 0`)은 검사하지 않고 통과시키며, 그 요청 때의 계정 팀을 모두 `device_scope` 에 기록합니다. 처음부터 받는 pull 은 남아 있는 모든 commit 을 받으므로 누락이 없고, 이미 지워진 구간이 있으면 지금처럼 retention 의 410 이 먼저 납니다. 페이지를 넘기는 도중(커서 > 0)에 들어간 팀은 기록이 없어 410 이 납니다. bootstrap 없이 pull 부터 하는 기존 픽스처 07 · 08 · 09 와 기존 테스트가 이 규칙으로 그대로 통과합니다(qa-designer 지적 2).
- 기기가 없는 조회(`PullQuery.DeviceID == ""`, 대시보드 `GET /dash/commits`)는 검사하지 않습니다.
- `/health` `features` 에 `team-scopes` 를 더합니다. 0.2.0 의 값은 정확히 `["conflict-fields", "device-token-ttl", "team-scopes", "token-pair", "totp"]`(이름순)입니다.
- protocol 1 의 요청 모양은 그대로입니다. 클라이언트가 보내는 팀별 커서(로드맵의 "scope별 커서")는 넣지 않습니다: 그것을 쓸 데스크톱이 아직 없고, 410 → 스냅샷이 같은 결과를 냅니다(intent 계획 3 의 "방법은 spec 에서" 에 따른 PM 결정, 우려 3).
- 픽스처(protocol 1, 서버 conformance 가 읽음): `14-team-join-catches-up.json`(B 가 팀 행을 만든 뒤 A 가 팀에 들어감 → A 의 pull 410 → bootstrap 에 그 행 → 다음 pull 200), `15-team-leave-hides.json`(A 가 빠짐 → 그 팀의 새 commit 이 A 의 pull 에 없음, A 의 그 팀 push 403 `foreign_team`). 픽스처에 멤버십 변경이 필요하므로 effect 를 더합니다: `{"effect": "addTeamMember", "teamId": "…", "accountId": "…"}` · `{"effect": "removeTeamMember", "teamId": "…", "accountId": "…"}`. 서버 러너는 `DisallowUnknownFields` 이므로 step 구조체에 `teamId` · `accountId` 를 더하고, `addTeamMember` 는 팀이 없으면 먼저 만듭니다(`team_member.teamId` 외래 키). 데스크톱 러너는 steps 를 읽지 않으므로 그대로 두되, 새 키 때문에 깨지지 않는지 V-1 에서 확인합니다. `packages/sync-fixtures/README.md` 의 Effects · 시나리오 표를 갱신합니다. (qa-designer 지적 3)

### W-6 운영 API (`internal/httpapi`)
모두 `/dash/admin/` 아래, 세션 + CSRF + `requireOperator`. JSON 은 기존 `/dash/*` 관례를 따릅니다: 목록은 객체로 감싸고(`{"accounts": [...]}`, `{"devices": [...]}`, `{"teams": [...]}`, `{"members": [...]}`, `{"commits": [...]}`), 시각은 unix ms 정수, 필드명은 기존 `/dash/*` 응답과 같은 이름. 계정 원소는 `id, username, createdAt, disabledAt?, operator, otp ("off"|"pending"|"enabled"), deviceCount`, 멤버 원소는 `accountId, username, role, createdAt`. 커밋 로그 쿼리는 기존 `/dash/commits` 와 같은 `sinceSeq` · `limit`. `POST accounts` 는 CLI `user create` 처럼 새 계정을 `team-default` 에 넣습니다. (qa-designer 지적 8)
| 메서드 · 경로 | 하는 일 | CLI 대응 |
|---|---|---|
| `GET accounts` | 계정 목록(운영자 · 비활성 · OTP 상태 · 기기 수) | `user list` · `user otp status` |
| `POST accounts` `{username, password, operator?}` | 생성 | `user create` |
| `POST accounts/{id}/disable` · `enable` | | `user disable` · `enable` |
| `POST accounts/{id}/password` `{password}` | 재설정(그 계정의 대시보드 세션 모두 끊음) | `user set-password` |
| `POST accounts/{id}/otp/disable` | OTP 해제 | `user otp disable` |
| `POST accounts/{id}/operator` `{operator: bool}` | 운영자 지정 · 해제 | `user operator` |
| `GET devices[?account=]` | 모든 기기 | `token list` |
| `POST devices` `{username, name?, expiresIn?, noExpiry?}` | 토큰 발급 → `{deviceId, token}` (한 번만) | `token issue` |
| `POST devices/{id}/revoke` | 회수 | `token revoke` |
| `GET teams` · `POST teams` `{name}` · `POST teams/{id}` `{name}` | 목록 · 생성 · 이름 변경 | `team list/create/rename` |
| `GET teams/{id}/members` · `POST teams/{id}/members` `{username}` · `DELETE teams/{id}/members/{accountId}` | 멤버 | `team members/add-member/remove-member` |
| `GET info` | `db info` + 키 상태(W-1) | `db info` |
| `POST vacuum` | VACUUM(끝날 때까지 쓰기가 막힌다는 확인 문구는 화면 몫) | `db vacuum` |
| `GET commits?sinceSeq=&limit=` | 전체 커밋 로그(페이지) | `log dump` |
- 비밀번호는 0.1.1 과 같은 규칙(비어 있지 않고 4096바이트 이하)을 서버가 확인합니다. 최소 길이는 새로 두지 않습니다. (qa-designer 지적 9) 운영 작업은 JSON 로그에 `operator action` 한 줄(누가 · 무엇을 · 대상 id, 비밀 값 없음)을 남깁니다.
- 마지막 운영자 보호(W-3)는 API 에서 `409 last_operator`.

### W-7 대시보드 운영 화면 (`apps/server/web`)
- 운영자에게만 탭이 더 보입니다: Accounts, Devices(all), Teams(관리), Server(info · vacuum · 커밋 로그). 운영자가 아니면 지금 화면 그대로입니다.
- 키가 `missing` 이면 운영자 화면 위에 경고 띠(문구: 평문 저장 · README 의 키 절 링크).
- 토큰 발급 결과는 모달에 한 번 보여 주고 복사 버튼을 둡니다. 비활성 · 회수 · vacuum · 멤버 제거 · 운영자 해제는 확인 창을 거칩니다.
- 기존 컴포넌트(`data-table.vue`)를 씁니다. 스타일은 `style.css` 의 지금 토큰을 따릅니다.

### W-8 문서 · 호환 표
- `apps/server/README.md` · `README.ko.md`: 설정 표에 `CROFFLE_SECRET_KEY` · `CROFFLE_SECRET_KEY_FILE`, "운영자"(첫 운영자 지정 · 마지막 운영자 규칙), "비밀 키"(생성 · 설정 · 교체 순서 · **키를 잃으면 OTP 를 쓰는 계정은 운영자가 OTP 를 해제해야 함** · DB 백업과 키를 따로 보관), "팀"(만들기 · 멤버 · 들어가면 다음 동기화에서 스냅샷을 다시 받음), 명령 목록, 호환 표(0.2.0 ↔ protocol 1 ↔ 데스크톱 1.3.x).
- `docs/server/release/0.2.0.md` 를 `.github/release-notes.md` 규약대로. en · ko 내용이 같아야 합니다.
- 번호 재조정 전 표기로 다중 팀을 "0.3" 이라 적은 Go 주석 4곳(`internal/store/sqlite/pull.go:21`, `internal/store/store.go:59`, `internal/auth/service.go:51`, `internal/httpapi/auth.go:274`)을 바로잡습니다. 커밋된 마이그레이션(`0001_init.sql:116`)의 주석은 그대로 둡니다. (qa-designer 지적 10)

## 작업 단위
각 단위는 따로 되돌릴 수 있고, 그 커밋은 게이트(`make -C apps/server check`, 대시보드를 고치면 `pnpm --dir apps/server/web build`, 공통 게이트)를 통과합니다.

- [x] W-1 (678a6b8) TOTP secret 암호화 · 시작 시 이전 · 키 없음 경고 — S-1, S-2, S-10
- [x] W-2 (1b690f0) `secret generate` · `secret rotate` — S-3
- [x] W-3 (97b82b1) 운영자 역할(0006) · CLI · `requireOperator` — S-4
- [x] W-4 (4abd54b) 다중 팀: store · CLI(create · rename · members · remove-member) — S-5
- [x] W-5 (420f68a) 팀 범위 따라잡기(0007 `device_scope`) · 410 · bootstrap 기록 · 픽스처 14 · 15 · `team-scopes` — S-6, S-7, S-9
- [x] W-6 (1b86743) 운영 API `/dash/admin/*` — S-8, S-4
- [x] W-7 (c0f851b) 대시보드 운영 화면 · 키 경고 띠 — S-8, S-2
- [x] W-8 (333ba62) README(en · ko) · 호환 표 · 릴리스 노트 · 주석 정정 — S-11

## 검증 기준
qa-designer(google gemini-4-argon-high)가 작성하고 PM 이 검토 · 수정했습니다(원본 `.loctx/reports/server-0.2.0/qa-checks.md`). qa-designer 가 짚은 spec 의 문제 10개는 설계에 반영했습니다(각 자리에 "qa-designer 지적 n"). PM 수정: V-13 의 "0.3" 판정을 지정한 4곳 기준으로 좁힘(0.3 계획을 가리키는 정당한 표기를 허용).

모든 검사는 `.agents/knowledge/2026-11-17_check-wording-and-raw-evidence.md`에 따라 실행합니다:
- 명령 하나마다 별도의 원본 로그 파일(첫 줄 `$ 명령`, 본문 stdout·stderr 전체, 마지막 줄 `exit=N`)과 디렉터리별 `00-context.txt`(커밋 SHA · 시각 · `git status`)를 남깁니다.
- `go test -run '<패턴>'`을 실행할 때는 반드시 `go test -list '<패턴>'`으로 대상 테스트 함수가 실제로 존재하는지 함께 기록하며, 출력에 `testing: warning: no tests to run`이나 `[no test files]`가 있으면 종료 코드가 `0`이라도 실패입니다.
- 수동 확인은 확인한 환경(OS · 브라우저 · 데스크톱 버전)과 하위 조건별 결과를 하나씩 기록하며, 미확인 항목이 하나라도 있으면 통과로 적지 않습니다.

- V-1 **프로젝트 정적 게이트(8개) · 서버 및 대시보드 통합 빌드(`build-all`) · 공유 픽스처 15개 컨포먼스**
  - 검증 대상: S-6, S-7, S-9, S-10, W-1~W-8
  - 실행 방법:
    1) 저장소 루트에서 `.agents/harness/gates.yaml`의 8개 게이트 명령을 순서대로 각각 실행합니다:
       - `pnpm typecheck`
       - `pnpm lint`
       - `pnpm format:check`
       - `pnpm -r test`
       - `pnpm build`
       - `pnpm --filter @croffledev/croffle-cli build`
       - `pnpm --dir apps/server/web build`
       - `make -C apps/server check`
    2) `make -C apps/server build-all`을 실행한 뒤 생성된 단일 바이너리 `apps/server/bin/croffle-server`의 존재를 확인하고 `./apps/server/bin/croffle-server --version`을 실행합니다.
    3) `go -C apps/server test -v ./internal/conformance -run TestScenarios`를 실행하여 픽스처별 실행 결과를 확인합니다.
  - 기대 결과:
    1) 8개 게이트 명령과 `make -C apps/server build-all`이 모두 종료 코드 `0`으로 통과합니다.
    2) `pnpm -r test`(`apps/desktop/test/sync-protocol.test.ts`)와 `go -C apps/server test -v ./internal/conformance -run TestScenarios` 양쪽 모두 새 시나리오 `14-team-join-catches-up.json`과 `15-team-leave-hides.json`을 포함한 `packages/sync-fixtures/scenarios/*.json` 15개 시나리오 전체를 읽고(`protocol: 1` 확인), 서버 러너는 데스크톱 전용 `12-row-codec` 1개만 `SKIP`하고 나머지 14개 시나리오(`01`~`11`, `13`~`15`)를 모두 `PASS`하며, 데스크톱 러너는 `fail 0`으로 통과합니다.
    3) `./apps/server/bin/croffle-server --version`은 `sync protocol 1-1`을 출력합니다.
  - 실패 판정: 어느 한 단계라도 종료 코드가 `0`이 아니거나, `gofmt`·`go vet`·`go test`·`oxlint`·`oxfmt`·`typecheck` 오류가 1건이라도 출력되거나, 데스크톱 러너가 15개 픽스처를 모두 읽지 않거나, 서버 러너가 `12-row-codec` 밖의 시나리오를 하나라도 `SKIP`하거나 실행하지 않거나, `apps/server/bin/croffle-server`가 생성되지 않으면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.2.0/v01-gates-and-build/`

- V-2 **커밋된 마이그레이션(0001~0005) · 데스크톱/패키지 소스 · Go/npm 의존성 불변 및 `team-default` 보존 검증 (음성 확인)**
  - 검증 대상: S-5, S-9, S-10, W-1~W-7
  - 실행 방법:
    1) 기존 서버 마이그레이션(0001~0005) 불변 확인:
       `git diff master...HEAD -- apps/server/internal/store/sqlite/migrations/0001_init.sql apps/server/internal/store/sqlite/migrations/0002_sync_meta.sql apps/server/internal/store/sqlite/migrations/0003_dash_session.sql apps/server/internal/store/sqlite/migrations/0004_tokens.sql apps/server/internal/store/sqlite/migrations/0005_otp.sql` 및 `ls -1 apps/server/internal/store/sqlite/migrations/`
    2) 데스크톱 앱 코드 · 드리즐 · 공개 타입 · CLI 불변 확인:
       `git diff master...HEAD -- apps/desktop/src apps/desktop/drizzle packages/types packages/cli`
    3) 서버 Go 의존성 및 대시보드 npm 의존성 불변 확인:
       `git diff master...HEAD -- apps/server/go.mod apps/server/go.sum apps/server/web/pnpm-lock.yaml` 및 `git diff master...HEAD -- apps/server/web/package.json`
  - 기대 결과:
    1) `0001_init.sql`~`0005_otp.sql` 5개 파일의 diff 출력이 정확히 `0`줄(`0001_init.sql`의 `team-default` 시드 행 보존)이고, `migrations/` 디렉터리에는 `0006_operator.sql`, `0007_device_scope.sql` 2개만 새로 추가되어 총 7개 파일입니다.
    2) `apps/desktop/src`, `apps/desktop/drizzle`, `packages/types`, `packages/cli`의 diff 출력이 `0`줄입니다.
    3) `apps/server/go.mod`, `apps/server/go.sum`, `apps/server/web/pnpm-lock.yaml`의 diff 출력이 `0`줄이고, `apps/server/web/package.json`에도 `dependencies` / `devDependencies` 변경이 없습니다(새 Go 모듈 `0`개, 새 npm 패키지 `0`개).
  - 실패 판정: `0001`~`0005` 마이그레이션이나 `apps/desktop/src` · `apps/desktop/drizzle` · `packages/types` · `packages/cli` · `apps/server/go.mod` · `apps/server/web/pnpm-lock.yaml`에 1줄이라도 변경이 있거나, 새 의존성이 추가되었으면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.2.0/v02-immutability-and-deps/`

- V-3 **TOTP secret AES-256-GCM(`SecretBox`) 암호화 · AAD(`accountId`) 바인딩 · 시작 시 평문 자동 이전 · 무중단 OTP 로그인**
  - 검증 대상: S-1, S-10, W-1
  - 실행 방법:
    1) 자동 테스트:
       `go -C apps/server test -v ./internal/secretbox ./internal/config ./internal/store/sqlite ./internal/auth ./internal/httpapi -run 'TestSecretBox|TestSeal|TestOpen|TestSecretKey|TestOTPEncrypt|TestMigratePlainOTP'`
    2) 평문 DB 준비(0.1.1 방식): 임시 DB(`$TMP_DB`)에서 암호화 키 없이 서버를 띄워 계정 `alice`(`acct-1`)와 `bob`(`acct-2`)을 만들고, `alice`의 대시보드 세션에서 `POST /dash/otp/enroll` → `POST /dash/otp/confirm`으로 TOTP를 활성화하여 `account_otp.secret`에 base32 평문 시크릿 `$PLAIN_SECRET`이 저장된 것을 확인합니다.
    3) 키 설정 후 기동 및 자동 이전: 32바이트 키 `$KEY1`(base64)을 생성하여 (a) `CROFFLE_SECRET_KEY="$KEY1"`, (b) 끝에 줄바꿈(`\n`)이 포함된 키 파일 `CROFFLE_SECRET_KEY_FILE="$KEY1_FILE"` 두 방식으로 각각 서버를 기동합니다. 시작 JSON 로그와 `./apps/server/bin/croffle-server db info --json --db "$TMP_DB"`, SQLite `SELECT accountId, secret, enabledAt, lastStep FROM account_otp`를 확인합니다.
    4) 무중단 로그인 및 신규 등록 확인:
       - `alice`가 OTP를 다시 등록하지 않고 기존 `$PLAIN_SECRET`으로 만든 현재 스텝의 6자리 코드로 `POST /auth/login`(또는 다음 스텝 코드로 `POST /auth/token`, `POST /dash/login`)을 호출합니다.
       - 암호화 키가 켜진 상태에서 `bob`으로 `POST /dash/otp/enroll` → `POST /dash/otp/confirm`을 호출해 신규 TOTP 등록·확인이 성공하는지 확인합니다.
    5) **음성 확인 (AAD `accountId` 바인딩)**: SQLite에서 `alice`(`acct-1`)의 암호화된 `account_otp.secret` 문자열(`v1:<keyId>:...`)을 그대로 `bob`(`acct-2`)의 `account_otp.secret`에 복사해 덮어쓴 뒤, `bob`의 비밀번호와 `$PLAIN_SECRET`의 6자리 코드로 `POST /auth/login`을 호출합니다.
  - 기대 결과:
    1) 단위 및 통합 테스트가 모두 PASS합니다.
    2) 3)에서 서버 시작 시 평문 `account_otp.secret`이 한 트랜잭션에서 `v1:<keyId>:<base64(nonce||ciphertext)>` 형식(`keyId`는 `hex(SHA-256(key))[:8]` 8자)으로 바뀌고, DB 파일 안에 `$PLAIN_SECRET` 평문 문자열이 `0`건이며, `enabledAt`·`lastStep`은 이전 전과 동일하게 보존됩니다. 시작 JSON 로그에 `"secret key: encrypted 1 one-time code secrets"`(level `INFO`)가 출력되고, `db info --json`에 `"secretKey": "set"`, 평문 `0`건, 암호화 `1`건(4단계 `bob` 등록 후 `2`건)이 보고됩니다.
    3) 4)에서 `alice`는 재등록 없이 기존 시크릿 코드로 HTTP `200` 로그인되며, 키가 설정된 상태에서 `bob`의 신규 `enroll` → `confirm`(`EnableOTP`)도 무작위 nonce와 무관하게 HTTP `200`으로 활성화되고 DB에 `v1:<keyId>:...`로 저장됩니다.
    4) 5)에서 `acct-1`의 암호문을 `acct-2` 행에 옮겨 붙이면 AES-GCM AAD(`accountId`) 검증이 실패하여 `bob`의 로그인이 HTTP `200`으로 통과하지 않고 거부됩니다.
  - 실패 판정: 키가 설정된 서버 기동 후에도 `account_otp.secret`에 평문이 남거나, `v1:<8자 hex keyId>:<base64>` 형식이 아니거나, 기존 OTP 사용자가 재등록 없이 로그인하지 못하거나, 키가 켜진 상태에서 신규 `POST /dash/otp/confirm`이 실패하거나, 다른 계정 행에 복사한 암호문이 복호화되어 로그인이 성공하면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.2.0/v03-totp-secret-encryption/`

- V-4 **TOTP 암호화 키 미설정 경고 · 잘못된 키 거부 · 암호화 DB에 키 누락/불일치 시 시작 거부 및 CLI 복구 (음성 확인)**
  - 검증 대상: S-1, S-2, W-1, 우려 1
  - 실행 방법:
    1) 자동 테스트:
       `go -C apps/server test -v ./internal/config ./internal/secretbox ./internal/store/sqlite -run 'TestSecretKey.*Error|TestSecretKey.*Warn|TestSecretBox.*Mismatch|TestSecretBox.*Missing'`
    2) **키 미설정 평문 기동 + 경고 확인**:
       - OTP 행이 `0`건인 새 DB와 평문 OTP 행이 `1`건인 DB 각각에 대해 `CROFFLE_SECRET_KEY`와 `CROFFLE_SECRET_KEY_FILE`을 모두 비운 채 서버를 기동하고 시작 JSON 로그와 `GET /dash/admin/info`(운영자 세션) 응답을 확인합니다.
    3) **음성 확인 (잘못된 키 설정 시 시작 실패)**:
       - (a) `CROFFLE_SECRET_KEY`와 `CROFFLE_SECRET_KEY_FILE`을 둘 다 설정해 서버 기동
       - (b) 32바이트가 아닌 base64 키(예: 16바이트 `dG9vLXNob3J0LWtleQ==`) 및 유효하지 않은 base64 문자열로 서버 기동
    4) **음성 확인 (암호화된 secret이 있는 DB에 키 누락 / 다른 키 설정 시 시작 실패)**:
       - (a) V-3에서 `$KEY1`으로 암호화된 `account_otp` 행이 들어 있는 DB에 대해 키 환경변수 없이 서버 기동
       - (b) 같은 DB에 `$KEY1`과 다른 32바이트 키 `$KEY2`를 설정해 서버 기동
    5) **키 분실 시 CLI 복구 경로 확인 (우려 1)**:
       - 4a)의 암호화된 DB에서 키 환경변수 없이 `./apps/server/bin/croffle-server user otp disable alice --db "$TMP_DB"`(및 `bob`)를 실행한 뒤, 다시 키 없이 서버를 기동합니다.
  - 기대 결과:
    1) 2)에서 OTP 행 유무(0건·1건)와 상관없이 서버가 정상 기동되고, 시작 JSON 로그에 level `WARN`으로 `"no secret key: one-time code secrets are stored in plain text"`가 출력되며, `GET /dash/admin/info`에 `"secretKey": "missing"`이 반환됩니다.
    2) 3a), 3b)에서 서버는 리스닝을 시작하지 않고 비-0 종료 코드(`exit!=0`)로 즉시 종료됩니다.
    3) 4a)에서 서버는 비-0 종료 코드로 종료되고 `"stored one-time code secrets are encrypted but no secret key is set"` 오류를 출력합니다.
    4) 4b)에서 서버는 비-0 종료 코드로 종료되고 `"stored one-time code secrets were sealed with another key (id <key1Id>); run secret rotate"` 오류를 출력합니다.
    5) 5)에서 CLI `user otp disable`은 시작 검사를 거치지 않아 키 없이도 종료 코드 `0`으로 암호화된 OTP 행을 삭제하며, 삭제 후에는 키 없이도 서버가 정상 기동됩니다.
  - 실패 판정: 키가 없을 때 시작 `WARN` 로그가 누락되거나(특히 OTP 행이 0건일 때 누락), 두 키 변수를 동시에 주거나 32바이트가 아닌 키를 주었는데 서버가 기동되거나, 암호화 행이 있는 DB에서 키가 없거나 다른 키인데 서버가 기동되거나, 키 없이 CLI `user otp disable`을 실행할 수 없으면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.2.0/v04-totp-key-warnings-and-guards/`

- V-5 **`secret generate` · `secret rotate` 키 생성 및 재암호화 · 트랜잭션 원자성 (음성 확인 포함)**
  - 검증 대상: S-3, W-2, 우려 2
  - 실행 방법:
    1) 자동 테스트:
       `go -C apps/server test -v ./internal/cli ./internal/secretbox ./internal/store/sqlite -run 'TestSecretGenerate|TestSecretRotate'`
    2) 키 생성: `./apps/server/bin/croffle-server secret generate`를 두 번 실행해 각각 `$KEY1_FILE`, `$KEY2_FILE`에 저장하고, 출력이 base64 한 줄이며 디코딩 시 정확히 32바이트이고 두 키가 서로 다른지 확인합니다.
    3) 평문 → 첫 키 암호화(`secret rotate`): 평문 OTP secret 2건(`alice`, `bob`)이 들어 있는 DB에서 현재 키 환경변수 없이 `./apps/server/bin/croffle-server secret rotate --new-key-file "$KEY1_FILE" --db "$TMP_DB"`를 실행합니다.
    4) 이전 키(`$KEY1`) → 새 키(`$KEY2`) 회전: `CROFFLE_SECRET_KEY_FILE="$KEY1_FILE"`을 설정한 상태에서 `./apps/server/bin/croffle-server secret rotate --new-key-file "$KEY2_FILE" --db "$TMP_DB"`를 실행하고 출력 문구와 DB의 `account_otp.secret` 접두(`v1:<key2Id>:`)를 확인합니다. 이어서 `CROFFLE_SECRET_KEY_FILE="$KEY2_FILE"`로 서버를 기동해 `alice`의 TOTP 코드로 `POST /auth/login`을 호출합니다.
    5) **음성 확인 (복호화 실패 시 단일 트랜잭션 롤백)**: 2개 행 중 `bob`의 `account_otp.secret`을 잘못된 암호문(`v1:deadbeef:bm90LWEtdmFsaWQtY2lwaGVydGV4dA==`)으로 변조한 뒤 `CROFFLE_SECRET_KEY_FILE="$KEY2_FILE"`로 `secret rotate`를 실행하고, 실행 전후 `alice` 행의 암호문 문자열이 그대로 유지되었는지 확인합니다.
  - 기대 결과:
    1) `secret generate`는 매번 무작위 32바이트 키의 base64 한 줄을 stdout에 출력합니다.
    2) 3)에서 현재 키 없이 실행한 `secret rotate`가 평문 2건을 모두 `v1:<key1Id>:...`로 암호화합니다.
    3) 4)에서 `$KEY1` → `$KEY2` 회전이 종료 코드 `0`으로 완료되어 두 행 모두 `v1:<key2Id>:...`로 바뀌고, 출력에 서버 중지 후 실행하라는 경고 문구와 설정을 새 키로 바꾸고 재시작하라는 안내가 포함되며, `$KEY2`로 기동한 서버에서 `alice`가 정상 로그인됩니다.
    4) 5)에서 1개 행이라도 복호화에 실패하면 `secret rotate`는 비-0 종료 코드(`exit!=0`)로 실패하고 트랜잭션이 롤백되어 정상 행(`alice`)도 회전 전 값 그대로 남습니다.
  - 실패 판정: `secret generate` 출력이 32바이트 base64가 아니거나, `secret rotate` 후 새 키로 서버가 뜨지 않거나 OTP 로그인이 실패하거나, 일부 행 복호화 실패 시 다른 행이 반쯤 재암호화된 채 커밋되면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.2.0/v05-secret-generate-and-rotate/`

- V-6 **운영자 역할(마이그레이션 0006) · CLI(`user create --operator` · `user operator grant/revoke` · `user list`) · 마지막 운영자 보호(`ErrLastOperator`, `409 last_operator`) · `requireOperator`(`403 not_operator`) 및 동기화 격리 (음성 확인)**
  - 검증 대상: S-4, W-3, W-6
  - 실행 방법:
    1) 자동 테스트:
       `go -C apps/server test -v ./internal/store/sqlite ./internal/cli ./internal/httpapi -run 'TestOperator|TestLastOperator|TestRequireOperator'`
    2) CLI 운영자 생성 · 목록 · 승격/해제 및 마지막 운영자 보호:
       - `./apps/server/bin/croffle-server user create admin1 --password-stdin --operator --db "$TMP_DB"`로 첫 운영자를, `user create regular1 --password-stdin --db "$TMP_DB"`로 일반 계정을 생성합니다.
       - `./apps/server/bin/croffle-server user list --db "$TMP_DB"` 및 `user list --json --db "$TMP_DB"`를 실행합니다.
       - **음성 확인 (CLI 마지막 운영자 보호)**: 활성 운영자가 `admin1` 1명뿐일 때 `--force` 없이 `user operator revoke admin1 --db "$TMP_DB"` 및 `user disable admin1 --db "$TMP_DB"`를 실행합니다.
       - `user operator grant regular1 --db "$TMP_DB"` 실행 후 `user operator revoke admin1 --db "$TMP_DB"` 실행(활성 운영자 2명 → 1명 성공 확인), 다시 `user operator grant admin1` → `user operator revoke regular1`로 `admin1`만 운영자로 둡니다.
    3) HTTP `GET /dash/me` · `requireOperator` (`403 not_operator`) · API 마지막 운영자 보호 (`409 last_operator`):
       - 일반 계정 `regular1`로 `POST /dash/login` 및 `GET /dash/me`를 호출해 `operator` 필드를 확인하고, `regular1` 세션 쿠키로 `GET /dash/admin/accounts`, `POST /dash/admin/accounts`, `GET /dash/admin/devices`, `GET /dash/admin/teams`, `GET /dash/admin/info`, `POST /dash/admin/vacuum`, `GET /dash/admin/commits`를 각각 호출합니다 (음성 확인).
       - 운영자 `admin1`으로 `POST /dash/login` 및 `GET /dash/me`를 호출해 `operator: true`를 확인한 뒤, 유일한 활성 운영자인 자기 자신에 대해 `POST /dash/admin/accounts/{admin1_id}/operator` (`{"operator":false}`)와 `POST /dash/admin/accounts/{admin1_id}/disable`을 호출합니다 (음성 확인).
    4) **음성 확인 (운영자의 동기화 및 본인 뷰 격리)**:
       - `regular1`의 기기 토큰으로 개인 일정(`teamId: null`) 1건을 `POST /sync/push`합니다.
       - 운영자 `admin1`의 기기 토큰으로 `POST /sync/pull` · `POST /sync/bootstrap`을 호출하고, `admin1`의 대시보드 세션으로 본인 탭용 `GET /dash/rows`를 호출합니다.
  - 기대 결과:
    1) `user list` 표에 `OPERATOR` 열이 표시되고 `user list --json`에서 `admin1`은 `"operator": true`, `regular1`은 `"operator": false`입니다.
    2) 활성 운영자가 1명뿐일 때 `--force` 없는 CLI `user operator revoke`와 `user disable`은 모두 비-0 종료 코드(`exit!=0`)로 거부되며 `admin1`은 활성 운영자로 남습니다. 운영자가 2명 이상일 때는 `revoke`가 정상 통과합니다.
    3) `regular1`의 `POST /dash/login` · `GET /dash/me` 응답은 `"operator": false`이고, `regular1` 세션으로 보낸 모든 `/dash/admin/*` 요청은 HTTP `403` (`{"code":"not_operator",...}`)을 반환합니다.
    4) `admin1`의 `GET /dash/me` 응답은 `"operator": true`이고, 유일한 활성 운영자인 `admin1`에 대한 API 운영자 해제 및 계정 비활성화 요청은 모두 HTTP `409` (`{"code":"last_operator",...}`)를 반환합니다.
    5) 4)에서 운영자 `admin1`의 `POST /sync/pull`, `POST /sync/bootstrap`, `GET /dash/rows`에는 `regular1`의 개인 일정이 `0`건 반환됩니다(커밋 로그는 `GET /dash/admin/commits`에서만 조회됨).
  - 실패 판정: 일반 계정 세션이 `/dash/admin/*` 중 하나라도 `403 not_operator`가 아닌 응답을 받거나, 마지막 활성 운영자가 API에서 해제/비활성화되거나(`409 last_operator` 미반환), CLI에서 `--force` 없이 해제/비활성화되거나, 운영자가 동기화 엔드포인트로 타인의 비공유 행을 받을 수 있으면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.2.0/v06-operator-role-and-protection/`

- V-7 **다중 팀 관리(CLI · store: 생성 · 이름 변경 · 멤버 목록/추가/제거) 및 `team-default` 보존**
  - 검증 대상: S-5, W-4
  - 실행 방법:
    1) 자동 테스트:
       `go -C apps/server test -v ./internal/store/sqlite ./internal/cli -run 'TestTeamCreate|TestTeamRename|TestTeamMembers|TestTeamRemoveMember'`
    2) CLI 팀 생명주기 실행:
       - `./apps/server/bin/croffle-server team list --json --db "$TMP_DB"`로 기본 `team-default` 확인
       - `./apps/server/bin/croffle-server team create "  Design Team  " --db "$TMP_DB"`(ID 미지정) 및 `./apps/server/bin/croffle-server team create "Engineering" --id team-eng --db "$TMP_DB"`(ID 지정) 실행
       - `./apps/server/bin/croffle-server team rename team-eng "Core Engineering" --db "$TMP_DB"` 실행 후 `team list --json --db "$TMP_DB"` 확인
       - `./apps/server/bin/croffle-server team add-member team-eng alice --db "$TMP_DB"` 및 `team add-member team-eng bob --db "$TMP_DB"` 실행 후 `team members team-eng --db "$TMP_DB"` 및 `team members team-eng --json --db "$TMP_DB"` 확인
       - `./apps/server/bin/croffle-server team remove-member team-eng bob --db "$TMP_DB"` 실행 후 `team members team-eng --json --db "$TMP_DB"` 확인
    3) **음성 확인 (이름 길이 및 공백 검증)**:
       - 빈 이름(`"   "`)과 65자 문자열로 각각 `team create` 및 `team rename`을 실행합니다.
  - 기대 결과:
    1) `team-default`가 그대로 유지된 채, ID 미지정 생성 시 `team-<10자>` 형식 ID가 생성되고 이름 앞뒤 공백이 제거된 `"Design Team"`으로 저장되며, `--id team-eng` 지정 시 해당 ID로 생성된 뒤 `"Core Engineering"`으로 이름이 변경됩니다.
    2) `team members team-eng`(--json 포함)에 `alice`, `bob` 2명이 표시되었다가 `team remove-member team-eng bob` 실행 후 `alice` 1명만 남습니다.
    3) 공백뿐인 이름(0자)이나 64자를 초과하는 이름은 비-0 종료 코드(`exit!=0`)로 거부됩니다.
  - 실패 판정: `team-default`가 사라지거나, 자동 생성된 팀 ID가 `team-` 접두 10자가 아니거나, 앞뒤 공백 제거 및 1~64자 제한이 지켜지지 않거나, `remove-member` 후에도 멤버가 남아 있으면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.2.0/v07-multi-team-cli/`

- V-8 **팀 범위 따라잡기(`0007_device_scope` · 업그레이드 직후 무재스냅샷 · 팀 합류 시 `410 cursor_too_old` + `bootstrap` 기록 · 팀 제외 시 가시성/push 차단) 및 공유 픽스처 14 · 15**
  - 검증 대상: S-6, S-7, S-9, W-5
  - 실행 방법:
    1) 단위 및 통합 테스트:
       `go -C apps/server test -v ./internal/store/sqlite ./internal/httpapi ./internal/protocol -run 'TestDeviceScope|TestPullTeamScope|TestBootstrap.*Scope|TestRemoveTeamMember.*Scope|TestMigrate0007'`
    2) 공유 픽스처 14 · 15 컨포먼스 실행:
       - `go -C apps/server test -v ./internal/conformance -run 'TestScenarios/14-team-join-catches-up|TestScenarios/15-team-leave-hides'`
       - `node --test apps/desktop/test/sync-protocol.test.ts`
    3) 세부 시나리오 확인:
       - (a) **0.1.1 → 0.2.0 업그레이드 직후 재스냅샷 없음(`0007_device_scope.sql`)**: 0.1.1 스키마(버전 5)에서 계정 `alice`(`team-default` 소속)와 기기 `client-1`(`lastServerSeq = 3`)이 존재하는 DB를 0.2.0으로 마이그레이션한 뒤, `SELECT deviceId, teamId FROM device_scope`에 `(client-1, team-default)`가 채워져 있고 `client-1`의 첫 `POST /sync/pull` (`{"lastServerSeq": 3}`)이 `410` 없이 HTTP `200`을 반환하는지 확인합니다.
       - (b) **팀 합류 시 과거 공유 행 따라잡기(`14-team-join-catches-up.json`)**: `bob`(`client-2`, `team-2` 소속)이 `team-2` 공유 행 `row-shared-old`(`serverSeq = 1`)를 먼저 push하고, `alice`(`client-1`, `team-default`만 소속)가 자기 행을 push·pull하여 커서가 전진한 뒤, `addTeamMember`로 `alice`를 `team-2`에 추가합니다. 이어서 `client-1`이 `POST /sync/pull`을 호출하면 HTTP `410` (`{"code":"cursor_too_old","details":{"reason":"team_scope"},...}`)을 받고, `POST /sync/bootstrap` 마지막 페이지(`hasMore: false`)를 받으면 `row-shared-old`가 포함되어 내려오며 `device_scope`에 `(client-1, team-2)`가 기록되어 직후 `POST /sync/pull`이 HTTP `200` (`changes: []`)을 반환하는지 확인합니다.
       - (c) **다중 페이지 `bootstrap` 도중 새 팀 합류 시 정확성**: `limit: 1`로 `POST /sync/bootstrap` 1페이지(`hasMore: true`, `cursor` 반환)를 받은 직후 `alice`가 제3의 팀 `team-3`에 추가되고, 1페이지의 `cursor`로 마지막 페이지(`hasMore: false`)를 마쳤을 때 `device_scope`에는 1페이지 시점의 팀만 기록되어(`team-3` 미기록), 직후 `POST /sync/pull`이 다시 HTTP `410`을 돌려주는지 확인합니다.
       - (d) **팀 제외 시 가시성 차단 · `403 foreign_team` · 재합류(`15-team-leave-hides.json`)**: `removeTeamMember`로 `alice`를 `team-2`에서 제거한 뒤:
         1. `device_scope`에서 `alice` 기기의 `team-2` 행이 삭제되었는지 확인합니다.
         2. `bob`이 `team-2`에 새 커밋 `row-shared-new`를 push해도 `alice`의 `POST /sync/pull`과 `POST /sync/bootstrap`에 `team-2` 행이 `0`건인지 확인합니다.
         3. `alice`가 `teamId: "team-2"`인 커밋을 `POST /sync/push`하면 HTTP `403` (`{"code":"foreign_team",...}`)으로 거부되는지 확인합니다.
         4. 다시 `alice`를 `team-2`에 추가하면 다음 `POST /sync/pull`이 다시 HTTP `410` (`cursor_too_old`)을 반환하는지 확인합니다.
  - 기대 결과:
    1) 위 단위/통합 테스트와 컨포먼스 `14-team-join-catches-up`, `15-team-leave-hides`가 모두 PASS합니다.
    2) (a)~(d) 모든 단계가 기대 상태 코드(`200`, `410 cursor_too_old` + `details.reason: "team_scope"`, `403 foreign_team`) 및 `device_scope` 상태와 일치합니다.
    3) `packages/sync-fixtures/README.md`의 Effects 표에 `addTeamMember` · `removeTeamMember`가, 시나리오 표에 `14-team-join-catches-up.json` · `15-team-leave-hides.json`이 추가되어 있습니다.
  - 실패 판정: 0.2.0 업그레이드 직후 기존 기기가 `410`을 받거나, 새 팀에 들어간 기기가 `pull`에서 `410 cursor_too_old`를 받지 않아 기존 공유 행을 놓치거나, `bootstrap` 완료 후에도 `pull`이 계속 `410`을 내거나, 팀에서 빠진 계정에 그 팀의 행이 내려가거나 `push`가 `403 foreign_team`으로 거부되지 않으면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.2.0/v08-team-scope-catchup-and-fixtures/`

- V-9 **대시보드 운영 API(`/dash/admin/*`) 전 엔드포인트 · 비밀번호 재설정 시 세션 강제 종료 · 토큰 1회 노출 · CSRF 보호 · 감사 로그(`operator action`) 검증**
  - 검증 대상: S-4, S-8, W-6
  - 실행 방법:
    1) 자동 테스트:
       `go -C apps/server test -v ./internal/httpapi -run 'TestAdmin|TestDashAdmin'`
    2) 기동된 로컬 서버에서 운영자 `admin1` 세션 쿠키로 `/dash/admin/*`의 모든 엔드포인트를 호출해 검증합니다:
       - **계정**:
         - `GET /dash/admin/accounts` (계정 목록: 운영자 여부 · 비활성 · OTP 상태 · 기기 수)
         - `POST /dash/admin/accounts` (`{"username":"carol","password":"carol-pass-1","operator":false}`)로 생성 → 빈 비밀번호(`400`) · 중복 사용자명(`409`) 음성 확인
         - `carol`로 대시보드(`POST /dash/login` → 쿠키 `$CAROL_COOKIE`)와 기기(`POST /auth/login` → 토큰 `$CAROL_TOKEN`)를 각각 로그인해 둡니다.
         - `POST /dash/admin/accounts/{carol_id}/password` (`{"password":"carol-pass-2"}`) 호출 직후 `$CAROL_COOKIE`로 `GET /dash/me`를 호출해 세션이 끊겼는지(`401`) 확인하고, `$CAROL_TOKEN`으로 `POST /auth/whoami`를 호출해 기기 토큰은 유지(`200`)되는지 확인합니다.
         - `POST /dash/admin/accounts/{carol_id}/disable` 호출 직후 `$CAROL_TOKEN`의 `POST /auth/whoami`가 `403 account_disabled`로 막히는지 확인한 뒤 `POST /dash/admin/accounts/{carol_id}/enable`로 다시 켭니다.
         - `carol` 계정에 TOTP를 켠 뒤 운영자 API `POST /dash/admin/accounts/{carol_id}/otp/disable`을 호출해 OTP 없이 `carol`이 로그인되는지 확인합니다.
         - `POST /dash/admin/accounts/{carol_id}/operator` (`{"operator":true}` 후 `{"operator":false}`) 동작을 확인합니다.
       - **기기 토큰**:
         - `POST /dash/admin/devices` (`{"username":"carol","name":"issued-by-web","noExpiry":true}`) 호출 → 응답에 `deviceId`와 평문 `token`이 1회 반환되는지 확인하고 그 `token`으로 `POST /auth/whoami` 호출(`200`)
         - `GET /dash/admin/devices` 및 `GET /dash/admin/devices?account=carol` 호출 → 기기 목록에 방금 발급한 기기가 보이며 응답 본문에 평문 `token`이나 `tokenHash`가 없는지 확인
         - `POST /dash/admin/devices/{deviceId}/revoke` 호출 → 직후 해당 토큰으로 `POST /auth/whoami` 호출 시 `401` 거부 확인
       - **팀**:
         - `POST /dash/admin/teams` (`{"name":"QA Team"}`) → `POST /dash/admin/teams/{teamId}` (`{"name":"QA & Release"}`) → `GET /dash/admin/teams` 확인
         - `POST /dash/admin/teams/{teamId}/members` (`{"username":"carol"}`) → `GET /dash/admin/teams/{teamId}/members` (`carol` 포함 확인) → `DELETE /dash/admin/teams/{teamId}/members/{carol_id}` → `GET /dash/admin/teams/{teamId}/members` (`carol` 제거 확인)
       - **서버 정보 · vacuum · 커밋 로그**:
         - `GET /dash/admin/info` (`schemaVersion: 7`, `secretKey`, 평문·암호화 secret 개수 등 확인)
         - `POST /dash/admin/vacuum` (정상 완료 확인)
         - `GET /dash/admin/commits?since=0&limit=10` (전체 계정의 커밋 로그 페이지 반환 확인)
       - **음성 확인 (CSRF 및 감사 로그 비밀값 부재)**:
         - 운영자 쿠키와 함께 `Origin: https://evil.example` 헤더를 붙여 `POST /dash/admin/accounts`, `POST /dash/admin/devices`, `POST /dash/admin/vacuum`, `DELETE /dash/admin/teams/{teamId}/members/{carol_id}`를 호출합니다.
         - 서버 stdout JSON 로그에서 `"operator action"` 로그 줄들을 확인합니다.
  - 기대 결과:
    1) 모든 `/dash/admin/*` 엔드포인트가 명세된 상태 코드와 JSON 필드를 반환합니다.
    2) `POST /dash/admin/accounts/{id}/password` 실행 즉시 대상 계정의 기존 대시보드 세션(`$CAROL_COOKIE`)이 모두 삭제되어 `GET /dash/me`가 HTTP `401`을 반환합니다.
    3) `POST /dash/admin/devices`가 반환한 `token`은 즉시 동기화 인증에 동작하며, 이후 목록 조회 응답이나 서버 로그에는 평문 토큰·비밀번호가 절대 나타나지 않습니다.
    4) 교차 출처(`Origin: https://evil.example`)로 보낸 `/dash/admin/*` 상태 변경 요청은 모두 HTTP `403` (`{"code":"cross_origin",...}`)으로 거부됩니다.
    5) 모든 운영 작업마다 서버 JSON 로그에 `"operator action"`(수행 운영자 ID · 작업 종류 · 대상 ID)이 기록되며, 비밀번호(`carol-pass-1`, `carol-pass-2`)나 발급된 토큰 평문 문자열은 로그 파일에 `0`건입니다.
  - 실패 판정: 엔드포인트 중 하나라도 누락·오동작하거나, 비밀번호 재설정 후 대상 계정의 기존 대시보드 세션이 살아 있거나, 교차 출처 요청이 `403`으로 거부되지 않거나, 감사 로그가 누락되거나 로그에 비밀번호·토큰 평문이 노출되면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.2.0/v09-admin-api-and-audit-log/`

- V-10 **`GET /health` 프로토콜 범위(`{min: 1, max: 1}`) 불변 및 `features` 목록(`team-scopes` 추가) 확인**
  - 검증 대상: S-9, S-10, W-5
  - 실행 방법:
    1) `go -C apps/server test -v ./internal/httpapi ./internal/protocol -run 'TestHealthReportsFeatures|TestFeaturesAreSortedAndUnique'`를 실행합니다.
    2) 기동된 0.2.0 서버에 `curl -sf http://127.0.0.1:18787/health`를 호출하여 응답 JSON 구조를 확인합니다.
  - 기대 결과:
    HTTP `200` 응답 JSON에 `protocol: {"min": 1, "max": 1}`, `db: "ok"`, `server`, `time`(unix ms 정수), `retention: {"oldestCommitSeq": ..., "purgedThroughSeq": ...}`이 0.1.1과 동일하게 존재하고, `features`가 정확히 `["conflict-fields", "device-token-ttl", "team-scopes", "token-pair", "totp"]`(사전순 정렬 5개)입니다.
  - 실패 판정: `protocol.min` 또는 `protocol.max`가 `1`이 아니거나, 기존 `/health` 필드가 누락·변경되었거나, `features` 배열이 위 5개(순서 포함)와 하나라도 다르면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.2.0/v10-health-and-features/`

- V-11 **`[수동 · Linux]` 대시보드 웹 UI 통합 검증 (일반 계정 화면 불변 · 운영자 4개 탭 · 키 미설정 경고 띠 · 토큰 1회 모달 · 확인 창)**
  - 검증 대상: S-2, S-4, S-5, S-8, W-7
  - 실행 방법:
    `[수동 · Linux]` `make -C apps/server build-all`로 빌드한 바이너리를 기동하고 브라우저에서 대시보드(`http://127.0.0.1:8787`)를 열어 아래 하위 조건을 차례로 수행합니다:
    1) **일반 계정(`regular1`, 비운영자) 로그인**:
       - 상단 네비게이션에 기존 5개 탭(`Devices`, `Teams`, `Rows`, `Commits`, `Security`)만 보이고 운영자 탭(`Accounts`, `Devices (all)` 등)과 키 경고 띠가 보이지 않는지 확인합니다.
       - 본인 기기 목록·회수, 본인 소속 팀 읽기, 행·커밋 조회, Security(TOTP) 탭이 0.1.1과 동일하게 동작하는지 확인합니다.
    2) **키 미설정(`CROFFLE_SECRET_KEY` 없음) 상태에서 운영자 계정(`admin1`) 로그인**:
       - 화면 상단에 "평문 저장" 경고 띠와 README의 키 안내 링크/문구가 표시되는지 확인합니다.
       - 운영자 전용 탭 4곳(`Accounts`, `Devices`(전체), `Teams`(관리), `Server`)이 표시되는지 확인합니다.
    3) **운영자 4개 탭 기능 및 확인 창(`confirm`) 검증**:
       - `Accounts` 탭: 계정 목록 확인 → 새 계정 생성 → 계정 비활성(확인 창 거침) 및 활성 → 비밀번호 재설정 → OTP 상태 표시 및 OTP 해제 → 운영자 지정 및 해제(확인 창 거침).
       - `Devices`(전체) 탭: 전체 계정의 기기 목록 조회 → 특정 계정에 기기 토큰 발급 → **모달에 발급된 토큰이 한 번 표시되고 복사 버튼이 동작**하는지 확인 → 모달을 닫은 뒤 토큰이 다시 노출되지 않는지 확인 → 기기 회수(확인 창 거침).
       - `Teams`(관리) 탭: 팀 목록 조회 → 새 팀 생성 → 팀 이름 변경 → 팀 멤버 목록 조회 및 멤버 추가 → 멤버 제거(확인 창 거침).
       - `Server` 탭: 서버 정보(`db info` 수치 + `secretKey: "missing"` 및 평문/암호화 secret 개수) 확인 → `Vacuum` 버튼 클릭 시 쓰기가 막힌다는 안내가 담긴 확인 창을 거쳐 실행되는지 확인 → 전체 커밋 로그 조회 및 페이지네이션(`Load more`) 확인.
    4) **키 설정 후 경고 띠 소거 확인**:
       - 서버를 `CROFFLE_SECRET_KEY`를 설정해 재기동한 뒤 운영자 화면을 새로고침하여 상단 경고 띠가 사라지고 `Server` 탭의 키 상태가 `set`으로 바뀌는지 확인합니다.
  - 기대 결과:
    위 1)~4)의 모든 하위 항목이 외부 네트워크 요청 없이 정상 동작하고, 비활성 · 회수 · vacuum · 멤버 제거 · 운영자 해제 5가지 작업 모두 확인 창을 거치며, 페이지의 JS 예외(`pageerror`)가 `0`건이고 브라우저 콘솔에 의도된 인증 응답(`GET /dash/me` 미로그인 401, OTP 계정의 1차 `POST /dash/login` 401 `otp_required`) 외의 리소스/API 오류가 `0`건입니다.
  - 실패 판정: 일반 계정에 운영자 탭이 노출되거나, 키 미설정 시 운영자 화면에 경고 띠가 없거나, 키 설정 후에도 경고 띠가 남거나, 토큰 발급 모달·복사 버튼이 없거나, 5가지 위험 작업 중 하나라도 확인 창 없이 즉시 실행되거나, 콘솔 JS 예외가 1건이라도 발생하면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.2.0/v11-dashboard-operator-ui.md`

- V-12 **`[수동 · Linux]` 데스크톱 1.3.1 클라이언트 ↔ 서버 0.2.0 실동기화 호환성 (업그레이드 직후 무재스냅샷 · 새 팀 합류 시 `410` 자동 스냅샷 · 팀 제외 시 격리)**
  - 검증 대상: S-6, S-7, S-9, W-5
  - 실행 방법:
    `[수동 · Linux]` 코드를 전혀 수정하지 않은 데스크톱 1.3.1과 서버 0.2.0 바이너리로 다음 3단계를 차례로 수행하고 데스크톱 로그(`apps/desktop/dev/logs/main.log`)와 로컬 SQLite DB를 확인합니다:
    1) **0.1.1 → 0.2.0 업그레이드 직후 재스냅샷 없음 확인**:
       - 0.1.1 서버(또는 0.1.1 스키마 DB)에 데스크톱 1.3.1(`alice`, 기기 A)이 연결되어 초기 `bootstrap`과 일정 1건 `push`/`pull`을 마친 상태에서 서버만 0.2.0(`CROFFLE_SECRET_KEY` 설정)으로 올려 재기동합니다.
       - 기기 A에서 동기화(`Sync now`)를 실행하고 `main.log`에 찍힌 HTTP 요청을 확인합니다.
    2) **새 팀 합류 시 팀의 기존 공유 행 자동 수신 (`410 cursor_too_old` → `bootstrap`)**:
       - 대시보드 운영 화면에서 새 팀 `team-alpha`를 만들고 계정 `bob`만 멤버로 넣은 뒤, `bob`이 `teamId: "team-alpha"`인 공유 일정 `"Alpha Retro"`(1건 이상)를 서버에 push합니다.
       - 기기 A(`alice`)에서 동기화를 실행해 `"Alpha Retro"`가 내려오지 않고 `POST /sync/pull 200`으로 끝남을 확인합니다.
       - 대시보드 운영 화면에서 `alice`를 `team-alpha` 멤버로 추가합니다.
       - 기기 A(`alice`)에서 동기화를 실행하고 `main.log`의 요청 순서와 데스크톱 캘린더 화면(및 `schedule` 테이블)을 확인합니다.
    3) **팀 제외 시 이후 변경 비가시성 및 `403 foreign_team` 격리**:
       - 대시보드 운영 화면에서 `alice`를 `team-alpha` 멤버에서 제거합니다.
       - `bob`이 `team-alpha`에 새 공유 일정 `"After Leave"`를 push한 뒤 기기 A(`alice`)에서 동기화를 실행해 `"After Leave"`가 내려오지 않음을 확인합니다.
       - 기기 A에서 로컬에 남아 있던 `"Alpha Retro"`(`teamId = "team-alpha"`)를 수정하고 동시에 새 개인 일정 `"My Private Task"`(`teamId = null`)를 만든 뒤 동기화를 실행합니다.
  - 기대 결과:
    1) 1)에서 0.2.0 업그레이드 직후 기기 A의 동기화는 `GET /health 200` → `POST /sync/pull 200`으로 끝나며 `POST /sync/bootstrap`이 호출되지 않습니다(무재스냅샷).
    2) 2)에서 `alice`가 `team-alpha`에 합류한 직후 기기 A의 동기화는 `POST /sync/pull 410` → `"Cursor is older than the server keeps; taking a snapshot"` 로그 → `POST /sync/bootstrap 200` → `POST /sync/pull 200`으로 자동 진행되며, 합류 전에 만들어진 공유 일정 `"Alpha Retro"`가 데스크톱 1.3.1에 정상 수신·표시됩니다.
    3) 3)에서 팀 제외 후에는 `"After Leave"`가 기기 A에 내려오지 않고, 빠진 팀 행 수정 push는 `403 foreign_team`으로 `sendIsolating`에 의해 해당 outbox 항목 1건만 `failed`로 격리되며, 함께 만든 개인 일정 `"My Private Task"`는 정상 push(`accepted`)되어 동기화가 멈추지 않습니다.
  - 실패 판정: 업그레이드 직후 기기 A가 불필요하게 `bootstrap`을 다시 받거나, 새 팀에 들어간 뒤 `410` → `bootstrap`이 일어나지 않아 기존 공유 일정이 누락되거나, `bootstrap` 후에도 매번 `410`이 반복되거나, 팀에서 빠진 뒤에도 그 팀의 새 일정이 내려오거나 개인 일정 동기화까지 함께 막히면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.2.0/v12-desktop-131-real-sync.md`

- V-13 **영/한 `README` 대칭 · 호환 표 · 릴리스 노트(`docs/server/release/0.2.0.md`) · 구버전("0.3") 코드 주석 정정 확인**
  - 검증 대상: S-11, W-8
  - 실행 방법:
    1) `apps/server/README.md`와 `apps/server/README.ko.md`에서 다음 6개 항목의 존재 및 영/한 내용 대칭을 확인합니다:
       - 설정 환경변수 표의 `CROFFLE_SECRET_KEY`, `CROFFLE_SECRET_KEY_FILE`
       - 운영자(Operators) 절: 첫 운영자 지정(`user create --operator`, `user operator grant/revoke`), 마지막 활성 운영자 보호 규칙(`--force`), 대시보드 운영 화면 안내
       - 비밀 키(Secret key) 절: `secret generate` 키 생성, 키 설정, `secret rotate` 교체 순서(서버 중지 → rotate → 새 키로 설정 변경 → 시작), **키 분실 시 OTP 사용 계정은 운영자가 CLI `user otp disable`로 해제해야 한다는 주의사항**, DB 백업과 키를 따로 보관할 것
       - 팀(Teams) 절: 팀 생성 · 이름 변경 · 멤버 추가/제거, 계정이 새 팀에 들어가면 다음 동기화에서 스냅샷을 다시 받아 기존 공유 행을 따라잡는 동작
       - CLI 명령 표의 새 명령(`secret generate`, `secret rotate`, `user create --operator`, `user operator grant/revoke`, `team create/rename/members/remove-member`)
       - 호환 표: 서버 `0.1.0` / `0.1.1` / `0.2.0` ↔ sync protocol `1` ↔ 데스크톱 `1.3.x`, `/health`의 `features`(`team-scopes` 포함 5개)
    2) `docs/server/release/0.2.0.md` 파일의 존재 및 `.github/release-notes.md`의 Server 템플릿 섹션(`# croffle-server v0.2.0`, `Image: ghcr.io/team-croffle/croffle-server:0.2.0 · tag server-v0.2.0 · protocol 1–1`, `## Overview`, `## Upgrade notes`, `## Compatibility`, `## Known limitations`, `<details><summary>한국어</summary>...</details>`) 준수 및 영/한 대칭 여부를 확인합니다.
    3) 코드 주석의 번호 재조정 전 `"0.3"` 표기 정정 확인:
       `grep -rn "0\.3" apps/server/internal/`
  - 기대 결과:
    1) `apps/server/README.md`와 `apps/server/README.ko.md` 양쪽에 위 6개 항목이 빠짐없이 대칭으로 기술되어 있고, 0.1.1의 낡은 문구("One team. Multiple teams come later.", "Encrypting the secrets is planned for 0.2.0.", "accounts and teams are commands.")가 0.2.0 동작에 맞게 갱신되어 있습니다.
    2) `docs/server/release/0.2.0.md`가 템플릿 형식과 영/한 대칭을 모두 충족합니다.
    3) 설계 W-8 이 짚은 4곳(`pull.go:21`, `store.go:59`, `service.go:51`, `auth.go:274` — 줄 번호는 구현 뒤 바뀔 수 있으므로 그 주석 문장으로 찾음)에 다중 팀 · 팀별 커서를 "0.3" 으로 가리키는 표기가 없고, `grep -rn "0\.3" apps/server/internal/` 의 나머지 결과는 `0001_init.sql:116` 과, 0.3 을 공유 권한 등 실제 0.3 계획으로 가리키는 줄뿐입니다(결과 줄마다 판정을 기록).
  - 실패 판정: 영/한 README 중 한쪽이라도 위 항목이 누락되거나 내용이 어긋나거나, 0.1.1 한계 문구가 모순되게 남아 있거나, `docs/server/release/0.2.0.md`가 없거나 템플릿을 따르지 않거나, `0001_init.sql` 밖의 `apps/server/internal/` 코드에 `"0.3"` 주석이 남아 있으면 실패입니다.
  - 증적 경로: `.loctx/reports/server-0.2.0/v13-docs-release-note-and-comments/`

## 조정 기록
- A-1 (2026-11-18, W-4) 운영자가 정하는 팀 id(`team create --id`)를 1–64자 `[a-z0-9-]` 로 제한 · 운영 API 경로(`/dash/admin/teams/{id}`)와 로그에 그대로 들어가므로. 없는 팀에 `add-member` 하면 외래 키 오류 대신 `ErrNotFound` · W-4 커밋
- A-2 (2026-11-18, W-5) 새 기기를 만들 때 계정의 팀으로 `device_scope` 를 채움 · 처음 설계(커서 0 pull · bootstrap 완료 때만 기록)로는 기존 픽스처 04(bootstrap 없이 push → 그 `serverSeq` 로 pull)가 410 을 받아 protocol 1 의 답이 바뀌었음(S-9). 마이그레이션이 기존 기기에 쓰는 규칙과 같음 · W-5 커밋
- A-3 (2026-11-18, W-6) ① `route` 가 같은 경로의 405 대체 패턴을 경로마다 한 번만 등록하고 `Allow` 에 메서드를 모두 적음 · 운영 API 는 한 경로에 GET · POST 가 있어 기존 방식(메서드마다 대체 패턴 등록)이 패턴 충돌로 시작 시 panic. ② `GET /dash/admin/commits` 는 메타데이터(seq · 표 · 행 id · op · 소유자 · 팀 · 버전 · 시각 · 작성자)만 주고 행 본문은 주지 않음 · 운영자에게도 남의 일정 내용을 보이지 않으려는 PM 판단, 본문은 CLI `log dump`. ③ 응답에 `tokenExpiresAt`(발급) · `memberCount`(팀 목록) · `sizeBefore`/`sizeAfter`(vacuum) 를 더함 · W-6 커밋
- A-4 (2026-11-18, W-5) 로그인 응답의 `teamId`(protocol 1 의 단일 필드)는 계정이 `team-default` 에 있으면 그것을 줌 · `TeamsOf` 가 id 순이라 생성된 팀 id(`team-0f…`)가 앞서면 두 번째 팀에 들어가는 것만으로 이 값이 바뀌고, 데스크톱 1.3.x 는 이 값을 공유 대상 팀으로 저장함(S-9). W-8 의 주석 정정 중 발견 · 별도 커밋
- A-5 (2026-11-18, W-5, 리뷰 1 F-1) 커서 0 pull 도 팀 범위를 검사하고, pull 은 덮음을 기록하지 않음 · 커서 0 은 retention 검사를 건너뛰므로 기록이 정리된 서버에서 커서 0 pull 이 팀을 덮었다고 기록하면 남은 행을 영영 놓침. A-2 로 새 기기는 생성 때 덮이므로 예외가 필요 없음
- A-6 (2026-11-18, W-5, 리뷰 1 F-3) 스냅샷 커서가 팀 id 대신 멤버십(팀 id + 가입 시각 `team_member.createdAt`)을 담고, 마지막 페이지는 가입 시각이 그대로인 멤버십만 덮음 · 스냅샷 도중 나갔다 다시 들어온 팀은 그 사이의 행을 건너뛰었을 수 있음
- A-7 (2026-11-18, W-6, 리뷰 1 F-4) 비밀번호 재설정과 세션 종료를 한 트랜잭션(`ResetPassword`)으로, 대시보드 로그인은 확인한 비밀번호 해시가 그대로이고 계정이 활성일 때만 세션을 만듦 · 재설정 직전에 비밀번호를 확인한 로그인이 재설정 뒤에 세션을 만드는 경합. CLI `user set-password` 도 같은 동작(브라우저 로그인 종료)으로 맞춤
- A-8 (2026-11-18, W-7, 리뷰 1 F-5) 팀 화면은 멤버를 불러온 뒤에 팀과 목록을 함께 표시하고, 오류 때 목록을 닫음 · 실패하면 다른 팀의 멤버가 이 팀 이름 아래 남아 제거가 엉뚱한 팀에 갈 수 있었음
- A-9 (2026-11-18, W-2, 리뷰 1 F-7) `secret rotate --old-key-file` 로 두 키를 모두 열어 새 키로 다시 잠금 · 경고 문구가 말한 복구("옛 키로 다시 rotate")는 키가 섞인 DB 에서 동작하지 않았음
- A-10 (2026-11-18, W-8, 리뷰 1 F-6) 0.1.x 누락 복구 안내를 "기기 회수 후 재로그인"에서 "데스크톱 설정 → 동기화 → 로그아웃 후 재로그인"으로 · 데스크톱은 같은 계정 재로그인 때 커서를 유지함(`auth.ts` bind)

## 우려
1. **키를 잃으면 암호화된 OTP secret 을 되살릴 수 없습니다.** 서버는 뜨지 않고(S-2), 운영자는 키 없이 시작하려면 OTP 를 쓰는 계정의 OTP 를 CLI 로 해제해야 합니다(`user otp disable` 은 `DELETE FROM account_otp` 라 secret 을 읽지 않으므로 키 없이 됩니다 — `store/sqlite/otp.go:132`. 단 W-1 의 시작 검사보다 먼저 쓸 수 있어야 하므로, CLI 명령은 시작 검사를 거치지 않습니다). README 에 백업 주의를 둡니다.
2. **서버가 떠 있는 동안 `secret rotate`** — 서버가 옛 키를 들고 있어 그 사이 새 등록이 옛 키로 저장되면 다음 시작이 실패합니다. 막을 확실한 수단(pid · 잠금 파일)이 없어 순서 안내와 경고 문구로 둡니다. 실패해도 "옛 키로 rotate 한 번 더"로 복구됩니다.
3. **팀별 커서를 클라이언트에 열지 않습니다.** 팀에 들어갈 때마다 그 계정의 모든 기기가 전체 스냅샷을 다시 받습니다. 개인 · 소규모 셀프호스트에서는 행 수가 작아 문제가 없다고 보지만, 큰 팀에서는 비용이 큽니다. 데스크톱이 팀을 다루는 1.3.2 이후에 protocol 확장으로 다시 봅니다.
4. **팀 삭제가 없습니다.** 잘못 만든 팀은 이름만 바꿀 수 있습니다. 삭제는 공유 행 처리(공유 취소, 0.2.1)와 함께 정합니다.
5. **0.1.x 에서 이미 생긴 누락은 따라잡지 않습니다.** 0.1.x 에서도 `team add-member` 로 팀에 넣을 수 있었으므로, 그 뒤 스냅샷을 다시 받지 않은 기기는 그 팀의 예전 행을 이미 놓쳤을 수 있습니다. 0007 이 지금 기기를 모두 "따라잡음"으로 채우면(S-6 의 "올린 직후 재스냅샷 없음") 이 누락은 그대로 남습니다. 0.1.x 의 공유는 beta 이고 데스크톱 1.3.0 · 1.3.1 에는 공유 UI 가 없어 실제 공유 행은 거의 없다고 보지만, 필요하면 운영자가 그 기기를 회수 · 재로그인시켜 스냅샷을 받게 할 수 있습니다(README 에 한 줄).
6. **운영자 세션의 권한이 큽니다.** 운영자 세션을 빼앗기면 모든 계정을 바꿀 수 있습니다. 0.2.0 은 지금 세션 규칙(SameSite=Strict, CSRF, 만료)과 감사 로그 한 줄로 두고, 운영자에게 OTP 를 강제하는 것은 하지 않습니다(필요하면 질문 바랍니다).

## 범위 밖
- 데스크톱의 팀 선택 · 팀별 커서 · 공유 UI · 빠진 팀의 행 처리 — 데스크톱 1.3.2 이후.
- 클라이언트가 보내는 팀별 커서(protocol 확장) — 우려 3.
- 팀 삭제, 공유 취소, 팀원 스스로의 관리(초대 · 탈퇴 · 사용자의 팀 개설) — 0.2.1.
- 운영자 OTP 강제, 세분화된 운영 권한(읽기 전용 운영자 등).
- KMS · Vault, OTP 외 저장 데이터 암호화.
