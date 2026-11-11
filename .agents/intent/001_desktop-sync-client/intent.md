---
id: "001"
slug: desktop-sync-client
version: "1.3.0"
status: approved
origin: user
approvals:
  - { role: user, by: bluenyang, at: "2026-11-11", words: "그래. 승인", rev: "057da73 (rebased onto master as 75860bc, intent.md unchanged)" }
waivers: []
supersedes: null
---
## 문제
Croffle은 로컬 전용이라 기기를 여러 대 쓰면 일정이 기기마다 따로 놉니다. 선택적 셀프호스트 서버(`croffle-server` 0.1)가 준비되었으므로, 데스크톱이 그 서버와 일정을 주고받는 **내장 동기화 클라이언트(beta)** 가 필요합니다.

사용자 결정 (2026-10-20, 옛 계획 `.loctx/context/plan/1.3.0_sync-client.md`의 열린 질문 답변):
- "서버를 개발해야 가능하다" → 정상 경로의 검증 기준은 서버 0.1입니다.
- TLS는 "설정으로 허용, 기본값은 strict". 범위는 동기화 서버 호스트로 한정합니다.
- 인증은 "내부는 토큰 하나". UI는 ID/PW · 토큰 두 가지, ID/PW는 로그인(+OTP)으로 토큰을 받는 경로입니다.
- `schedule_external_link`은 "아직 대상 아님".
- v1.3은 "patch로 쪼갠다" — 1.3.0 동기화 / 1.3.1 공유 / 1.3.2 To-Do.

## 바라는 결과
- 사용자가 설정에서 서버를 등록하고 로그인하면, 일정 · 태그 · 일정-태그 연결 · 리마인더 · 반복 예외가 기기 사이에 동기화됩니다.
- 서버를 등록하지 않으면 지금과 똑같이 동작하고 네트워크 호출이 없습니다.
- 오프라인이어도 앱은 정상이고, 변경은 쌓였다가 연결되면 나갑니다.
- 동기화는 내부 기능입니다. `packages/types`(확장 공개 API)는 바뀌지 않습니다.

## 계획
옛 작업 000-1 ~ 000-15(`.loctx/context/migrated-work/`)를 재료로, spec에서 W-n으로 다시 나눕니다. 순서는 위험이 큰 기반부터입니다.

1. **기반 (네트워크 없음)** — `sync_server` · `sync_outbox` 테이블(마이그레이션 0007), outbox 트리거(0008, 서버 적용 행은 `lastWriterId = 'server'`로 억제), 서버 설정 서비스(DB + `safeStorage` 토큰 + TLS 모드), 내부 `sync.*` 브리지. ← 000-1 ~ 000-4
2. **프로토콜** — `common`의 프로토콜 계약, 전송 계층(토큰 헤더 · 재시도 · TLS 전용 session), 인증(토큰 / ID·PW+OTP → 토큰), push 엔진, pull 엔진(`/sync/bootstrap` 스냅샷과 410 복구 포함), 오케스트레이터(첫 동기화 때 기존 행 전체를 outbox에 적재)(주기 · 디바운스 · 백오프 · 상태). 연결할 때 `/health`로 프로토콜 범위를 확인하고(`X-Croffle-Sync-Version`), 서버 시계와 크게 어긋나면 경고합니다 — 서버 README가 데스크톱 동작으로 약속한 내용입니다. ← 000-5 ~ 000-10
3. **컨포먼스** — `packages/sync-fixtures`를 읽는 데스크톱 쪽 테스트(저장소의 첫 Node 테스트 러너). ← 000-15
4. **사용자 표면** — 설정 → 동기화 탭 + i18n(en/ko), 검증 하네스(서버 0.1 연동 + 장애 주입 스텁), README "동기화 (beta, 선택)" 절. ← 000-11 ~ 000-13
5. **릴리스 준비** — 데스크톱 태그 스킴을 `desktop-vX.Y.Z`로 전환(release.yml, `tagNamePrefix`). 첫 rc 전에 병합합니다. ← 000-14

6. **패키지 발행을 staging으로** — 지금 `publish-packages`는 Changeset 버전 PR이 병합되면 npm에 바로 공개합니다. 이를 npm staging에 올리고 사용자가 승인하면 공개되도록 바꿉니다. AGENTS.md의 "Publishing goes to npm staging" 문장은 지금 사실이 아니므로, 바뀐 동작에 맞춰 함께 고칩니다. 사용자 요청 (2026-11-11): "changeset으로 올라오는거 바로 publish네. 다음 버전에서 stage로 변경해줄래?"

버전을 다 만든 뒤 merge → audit → rc(`desktop-v1.3.0-rc.1`) → 최종 조정 → stable(사용자 지시 시)입니다. 작업 묶음마다 rc를 내지 않습니다.

## 영향 범위
- `apps/desktop`: main(DB · sync · IPC), preload(내부 브리지), renderer(설정 탭, 스토어), common(프로토콜 계약, i18n)
- `apps/desktop/drizzle/`: 새 마이그레이션 0007, 0008
- `packages/sync-fixtures`: 읽기만 (서버와 같은 시나리오)
- `.github/workflows/release.yml`, `apps/desktop/electron-builder.yml`: 태그 스킴
- `README.md` / `README.ko.md`, `docs/ROADMAP.md`(릴리스 단계)
- `.github/workflows/publish-packages.yml`, 루트 `package.json`의 `release` 스크립트, `AGENTS.md`(Changeset 규칙): 패키지 발행을 staging으로
- 영향 없음: `packages/types`, `packages/cli`의 코드(발행 방식만 바뀜), `apps/server`(프로토콜이 어긋나면 서버를 고치지 않고 먼저 사용자에게 묻습니다)

## 제약
- 프로토콜의 단일 기준은 이미 병합된 서버 0.1과 `packages/sync-fixtures`입니다. 옛 계획의 엔드포인트 모양보다 이쪽이 우선합니다.
- 쓰기 경로에 트랜잭션이 없고 서비스가 async라, outbox는 트리거로 적재합니다(2026-10-20 결정).
- 토큰은 `safeStorage`로만 디스크에 둡니다. 키링이 없으면 메모리 보관 + 경고, 평문 저장 금지.
- 이미 커밋된 마이그레이션(0000~0006)은 건드리지 않습니다. 이후 patch는 ADD COLUMN만 쓰고 부팅 시 트리거를 보정합니다.
- 로그에 토큰 · 비밀번호 · OTP가 남지 않아야 합니다.
- stable 1.3.0은 서버 0.1(`0.1.0-rc.1` 이미지)을 대상으로 한 테스트가 Linux와 Windows에서 통과한 뒤에만 냅니다.

## 열린 질문
—

## 질의서 기록
2026-11-11, 사용자 답변:
1. **bootstrap** — 질문: 새 기기의 첫 동기화와 보존 기간 밖 커서(410) 복구를 1.3.0 pull 엔진에 넣을까요? 답: "넣는게 좋을 것 같은데." → 넣습니다. 계획 2(프로토콜)의 pull 엔진에 `/sync/bootstrap` 스냅샷과 410 복구를 포함합니다.
2. **서버 rc 기준** — 질문: 데스크톱 검증 대상을 서버 `0.1.0-rc.1` 이미지로 삼을까요? 답: "예. Linux와 Windows까지는 검증가능해" → 서버 `0.1.0-rc.1` 대상 검증은 Linux와 Windows에서 합니다. macOS는 사용자가 검증할 수 없으므로 자동 게이트와 CI 빌드까지만 확인하고, 릴리스 노트에 macOS는 실서버 검증 전이라고 적습니다(에이전트 해석).
3. **기존 데이터의 첫 push** — 질문: 첫 연결 때 트리거 도입 전부터 있던 행을 outbox에 한 번 모두 적재할까요? 답: "예. 그렇게 해서 첫 동기화 해야지" → 첫 동기화 때 기존 행 전체를 outbox에 적재합니다. 계획 1(기반) 또는 2의 오케스트레이터에 둡니다(배치는 spec에서 정함).

### 에이전트가 정한 것
- 옛 계획의 "rc.1 / rc.2 / rc.3 묶음과 브랜치 세 개"는 쓰지 않습니다. 대안: 그대로 유지. 이유: 새 흐름에서 rc는 버전 완성 뒤의 preview입니다(2026-11-11 사용자 결정). 브랜치는 `feat/desktop-1.3.0-sync-client` 하나로 하고, 커밋은 W-n 단위로 나눕니다.
- 검증 하네스(000-11)는 사용자 표면 앞으로 당길 수 있게 spec에서 의존만 지킵니다. 이유: 옛 계획도 당기는 것을 허용했습니다.

## 하지 않을 것
- 일정 공유와 `schedule.teamId` (1.3.1), To-Do (1.3.2), 공유 권한 · 공유 취소 (1.3.5)
- 팀 · 원격 일정 관리 UI
- `schedule_external_link` 동기화
- 확장 공개 API(`packages/types`)에 `sync.*` 노출
- 리프레시 토큰, 조용한 토큰 갱신 (서버 0.2 이후)
- 시계 오차 보정 (0.1 beta는 클라이언트 벽시계 LWW를 감수)
