---
id: "004"
slug: dependency-security
version: "server-0.1.1 · 1.3.1"
status: draft
origin: user
approvals: []
waivers: []
supersedes: null
---
## 문제
서버 0.1.1 과 데스크톱의 의존성에 알려진 취약점 보고가 있습니다(2026-11-16 조사, `.loctx/reports/server-0.1.1/vuln/survey.md`).

- **서버(Go):** govulncheck 기준 코드가 닿는 취약점은 0건입니다. 모듈 단위로 `golang.org/x/crypto` 1건(GO-2026-5932, 쓰지 않는 `openpgp`)이 있고, `x/crypto` v0.59.0 을 비롯해 여러 모듈에 새 버전이 있습니다. 런타임 이미지는 `alpine:3.21` 입니다.
- **서버 대시보드(npm):** moderate 1건(minimatch, vue-tsc 경유 · 빌드 도구).
- **데스크톱 · 저장소 루트(npm):** critical 2 · high 7 · moderate 9 · low 2. 모두 개발 · 빌드 도구 경유입니다(oxfmt → tinypool, `@electron/get` · node-gyp → undici, electron-builder → fast-uri · minimatch). 앱에 들어가는 런타임 의존성에는 보고가 없습니다.
- **Electron 런타임:** 43.8.7 을 씁니다. 43.x 최신 패치는 43.8.10 입니다(43 은 지원 중).

사용자 요청 (2026-11-16):
- "서버랑 desktop 둘 다 vulnerabilities 패치 업데이트도 해야 함 ㅇㅇ"
- "취약점은 일단 서버를 다 패치한 다음, v1.3.1 desktop으로 낼거야. v1.3.1 기능은 v1.3.2로 미루자."

## 바라는 결과
- 서버 0.1.1 은 의존성 · 빌드 이미지의 알려진 취약점 보고가 없는 상태로 `server-v0.1.1-rc.1` 을 냅니다.
- 데스크톱 1.3.1 은 Electron 43.x 최신 패치와 패치된 의존성으로 나가고, 기능 변화는 없습니다.
- 두 제품 모두 동작은 그대로입니다. 프로토콜 1, 데스크톱 1.3.0 ↔ 서버 0.1.1 호환, 기존 DB 와 토큰이 유지됩니다.

## 계획
1. **서버 Go 의존성** — 직접 · 간접 모듈을 최신 패치 · 마이너로 올립니다(`x/crypto` v0.59.0 포함). govulncheck 로 모듈 단위 보고까지 0 인지 확인합니다.
2. **서버 빌드 · 런타임 이미지** — `alpine:3.21` 의 지원 상태를 확인하고 지원 중인 최신 Alpine 으로 올립니다. 빌드 단계 `golang:1.26-alpine` · `node:24-alpine` 은 최신 패치를 받는지 확인합니다.
3. **서버 대시보드 npm** — `apps/server/web` 의 의존성을 패치해 audit 0 을 만듭니다.
4. 서버 쪽 검사 · 리뷰 · 병합 → 감사 보완 → `server-v0.1.1-rc.1`.
5. **데스크톱 Electron** — 43.8.10 (43.x 최신 패치).
6. **데스크톱 · 루트 npm** — 개발 · 빌드 도구의 취약점을 패치합니다. 범위 안 업데이트로 안 되는 전이 의존성은 `pnpm.overrides` 로 올립니다.
7. 데스크톱 검사(게이트 + 실행 확인 + 동기화) · 리뷰 · 병합 → 감사 → `desktop-v1.3.1-rc.1`.

## 영향 범위
- 서버: `apps/server/go.mod` · `go.sum`, `apps/server/Dockerfile`, `apps/server/web/package.json` · `pnpm-lock.yaml`, 릴리스 노트 `docs/server/release/0.1.1.md`.
- 데스크톱: `apps/desktop/package.json`, 루트 `package.json` · `pnpm-lock.yaml`, 릴리스 노트(`.loctx/release/`).
- 로드맵: 데스크톱 1.3.1 을 보안 패치로, 일정 공유를 1.3.2 로(로컬 로드맵. 공개 로드맵은 패치 번호를 쓰지 않아 바꾸지 않음).
- 영향 없음(PM 해석): 앱 · 서버 코드 동작, 프로토콜, DB 스키마, `packages/types` · `packages/cli` 의 공개 API.

## 제약
- 기능 변화 없음. 메이저 업데이트는 하지 않습니다(Electron 44/45, Vite 메이저 등).
- `package.json` 과 lockfile 은 같은 커밋에서 바꿉니다.
- `packages/types` · `packages/cli` 의 `package.json` 이 바뀌면 Changeset 을 더합니다.
- 서버 릴리스는 GitHub Release 를 만들지 않습니다.
- 병합 전에 PR head 를 본 리뷰 기록이 history 에 있어야 합니다(`.agents/knowledge/2026-11-15_review-before-merge.md`, 감사 server-0.1.1 A-2).

## 열린 질문
1. **할 일(To-Do) 의 버전** — 일정 공유를 1.3.2 로 미루면, 원래 1.3.2 였던 할 일은 1.3.3 으로 미뤄도 됩니까? PM 해석은 "순서 유지: 1.3.2 공유, 1.3.3 할 일"입니다.
2. **서버 rc 의 범위** — 서버 보안 패치를 0.1.1 의 안정화 작업으로 넣고 `server-v0.1.1-rc.1` 을 그 뒤에 냅니다(PM 추천). 대안은 지금 상태로 rc.1 을 내고 패치를 rc.2 로 내는 것입니다.

## 질의서 기록
—

### 에이전트가 정한 것
- intent 하나에 두 제품을 둡니다. 사용자가 한 문장으로 요청했고 순서(서버 먼저)가 정해져 있기 때문입니다. spec 은 서버 · 데스크톱 단위를 나눠 따로 되돌릴 수 있게 합니다.
- 개발 · 빌드 도구만 걸린 보고도 패치합니다. 앱에 들어가지 않더라도 빌드 기기와 CI 에서 돕니다.

## 하지 않을 것
- 메이저 업데이트, 기능 변경, 데스크톱 1.3.2 이후 기능.
- 서버 TOTP secret 암호화(서버 0.2.0).
