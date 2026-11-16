---
intent: "004"
status: draft
approvals: []
summary_for_user: |
  —
---
## 요구 해석
- S-1 서버 0.1.1 의 Go 의존성을 같은 메이저 안에서 최신으로 올리고, govulncheck 가 **모듈 단위 보고까지 0건**이 되게 합니다. ← 계획 1, 바라는 결과 1
- S-2 서버 이미지의 런타임 단계를 지원 중인 최신 Alpine(3.23)으로 올립니다. 빌드 단계(`golang:1.26-alpine`, `node:24-alpine`)는 마이너 태그라 빌드할 때마다 최신 패치를 받는 것을 확인만 합니다. ← 계획 2
- S-3 서버 대시보드(`apps/server/web`)의 `pnpm audit` 이 0건이 됩니다. ← 계획 3
- S-4 서버 패치는 0.1.1 의 안정화 작업으로 들어가고, 병합 · 감사 보완 뒤 `server-v0.1.1-rc.1` 을 냅니다. 프로토콜 · DB · 동작은 바뀌지 않습니다(spec 003 의 V-1 · V-5 · V-12 를 다시 통과). ← 계획 4, 질의서 2, 바라는 결과 3
- S-5 데스크톱 1.3.1 은 Electron 43.8.10 과, 같은 메이저 안에서 올린 런타임 의존성으로 나갑니다. 기능 변화는 없습니다. ← 계획 5, 바라는 결과 2
- S-6 저장소 루트 · 데스크톱의 `pnpm audit` 이 0건이 됩니다. 같은 메이저 안 업데이트로 안 되는 전이 의존성만 `pnpm.overrides` 로 올리고, override 마다 이유를 적습니다. ← 계획 6
- S-7 데스크톱 1.3.1 은 1.3.0 의 DB 와 동기화 설정을 그대로 쓰고, 서버 0.1.1 과 동기화됩니다. ← 바라는 결과 3, 계획 7
- S-8 `packages/types` · `packages/cli` 의 `package.json` 이 바뀌면 Changeset 을 더합니다. 공개 API 는 바뀌지 않습니다. ← 제약 3
- S-9 병합 전에 PR head 를 본 리뷰 기록을 history 에 남기고, 병합 직전 SHA 대조 원본을 남깁니다. ← 제약 5

## 설계

### 서버
- **Go 모듈:** `go get -u ./...` 뒤 `go mod tidy`. `go 1.26.0` 지시어는 그대로 둡니다(툴체인 메이저를 바꾸지 않음). 직접 의존성 6개는 같은 메이저 안 최신, 간접 의존성은 `go get -u` 가 고른 대로. 결과를 `go list -m -u all` 로 남깁니다.
- **govulncheck:** `golang.org/x/vuln/cmd/govulncheck@v1.9.0 -show verbose ./...` 의 "Symbol Results"와 "Module Results"가 모두 0 이어야 합니다.
- **이미지:** `apps/server/Dockerfile` 의 `FROM alpine:3.21` → `alpine:3.23`. `apk add` 패키지(ca-certificates, tzdata)와 사용자 생성은 그대로. 빌드 단계 태그는 바꾸지 않습니다.
- **대시보드:** `apps/server/web` 에서 같은 메이저 안 업데이트(`pnpm up`)로 minimatch 경유 보고를 없앱니다. 안 되면 그 lockfile 에 한해 override.
- **문서:** `docs/server/release/0.1.1.md` 의 업그레이드 안내에 "의존성 · 런타임 이미지(Alpine 3.23) 갱신" 한 줄, 바이너리 크기 재측정.

### 데스크톱
- **Electron:** `apps/desktop/package.json` 의 `electron` 을 `^43.8.10` 으로. 메이저는 43 유지.
- **런타임 의존성:** `dependencies` 중 같은 메이저 안에 새 버전이 있는 것(예: `better-sqlite3` 13.2.0, `vue-i18n`, `vue-router`, `@iconify-json/lucide`)을 올립니다. `better-sqlite3` 는 Electron ABI 에 맞춰 다시 빌드되는지(`electron-builder install-app-deps` / postinstall) 확인합니다.
- **개발 · 빌드 도구:** `electron-builder` 26.18.x, `vite` 8.3.8, `@vitejs/plugin-vue`, `postcss`, `eslint` 등 같은 메이저 안 최신. **oxfmt** 는 0.x 라 마이너가 포맷 결과를 바꿀 수 있습니다. tinypool 보고를 없애는 가장 낮은 oxfmt 버전으로 올리고, 포맷 결과가 바뀌면 그 차이를 한 커밋(`chore: oxfmt …`)으로 따로 둡니다.
- **override:** 위로도 남는 전이 의존성(undici, fast-uri, minimatch 등)은 루트 `package.json` 의 `pnpm.overrides` 에 패치된 최소 버전으로 둡니다. 각 항목 옆이 아니라 spec 의 표에 이유를 적습니다(JSON 은 주석 불가).
- **버전:** 데스크톱 버전은 릴리스 워크플로가 올립니다(`apps/desktop/package.json` 의 `version` 은 손대지 않음).
- **문서:** 데스크톱 1.3.1 릴리스 노트 본문은 `.loctx/release/` 에 씁니다(`.github/release-notes.md` 규약).

### override 표 (구현 때 채움)
| 패키지 | 버전 | 경유 | 이유 |
|---|---|---|---|
| — | — | — | — |

## 작업 단위
각 단위는 따로 되돌릴 수 있고, 그 커밋은 관련 게이트를 통과합니다. 서버(W-1~W-4)를 먼저 병합하고 rc.1 을 낸 뒤 데스크톱(W-5~W-8)을 합니다.

- [ ] W-1 서버 Go 모듈 업데이트 + govulncheck 0 — S-1
- [ ] W-2 서버 런타임 이미지 Alpine 3.23 — S-2
- [ ] W-3 서버 대시보드 npm 패치 — S-3
- [ ] W-4 서버 릴리스 노트 · 바이너리 크기 — S-4
- [ ] W-5 데스크톱 Electron 43.8.10 + 런타임 의존성(같은 메이저) — S-5, S-7
- [ ] W-6 개발 · 빌드 도구(같은 메이저) + oxfmt — S-6
- [ ] W-7 남은 전이 의존성 `pnpm.overrides` + override 표 — S-6
- [ ] W-8 Changeset(필요할 때) · 데스크톱 릴리스 노트 초안 — S-8

## 검증 기준
—

## 우려
1. **oxfmt 마이너 업데이트가 포맷을 바꿀 수 있습니다.** 그러면 저장소 전체에 포맷 차이가 생깁니다. 별도 커밋으로 분리해 되돌리기 쉽게 하고, 리뷰는 "포맷만 바뀌었는지"를 봅니다.
2. **`better-sqlite3` 마이너 업데이트는 네이티브 모듈 재빌드를 거칩니다.** Electron ABI 가 맞지 않으면 앱이 뜨지 않습니다. 실행 확인(V-n)에 앱 기동과 DB 열기를 넣습니다.
3. **override 는 상위 패키지가 기대하지 않은 버전을 강제합니다.** 패치 버전 안에서만 올리고, 빌드 · 패키징(`build:unpack`)으로 확인합니다.
4. **서버 0.1.1 감사는 이미 끝났습니다.** 서버 패치를 rc.1 에 넣으므로 병합 뒤 그 범위의 감사를 한 번 더 받습니다(intent 004 서버 부분).

## 범위 밖
- 메이저 업데이트(Electron 44/45, FullCalendar 7, TypeScript 7, electron-vite 6 등).
- 기능 변경, 데스크톱 1.3.2 이후 작업, 서버 TOTP secret 암호화.
