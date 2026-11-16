---
intent: "004"
status: draft
approvals: []
summary_for_user: |
  보안 패치를 작업 단위 8개로 나눴습니다. 서버(W-1~W-4: Go 모듈 · Alpine 3.23 · 대시보드 npm · 노트)를 먼저 병합하고 감사 보완 뒤 `server-v0.1.1-rc.1`, 그다음 데스크톱 1.3.1(W-5~W-8: Electron 43.8.10 · 같은 메이저 안 전체 업데이트 · 남는 전이 의존성 override · 노트)입니다.
  메이저 업데이트는 하지 않습니다. 예외는 두 가지입니다: oxfmt(0.x)는 tinypool 을 고치는 버전까지 올리고 포맷 변화는 커밋 하나로 따로 두며, 패치가 상위 메이저에만 있는 전이 의존성(예: minimatch 11)은 override 로 넘깁니다.
  검증 기준 V-1~V-14 는 qa-designer 가 쓰고 PM 이 고쳤습니다. 목표는 govulncheck 와 pnpm audit 모두 0건, 기능 코드 diff 0줄, spec 003 의 V-1 · V-5 · V-12 재통과입니다. 수동 확인은 데스크톱 1.3.1 이 기존 1.3.0 프로필로 떠서 서버 rc.1 과 동기화되는지(V-12)입니다.
  우려: oxfmt 포맷 차이가 저장소 전체에 생길 수 있고, better-sqlite3 마이너 업데이트는 네이티브 재빌드를 거칩니다.
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
- **이미지:** `apps/server/Dockerfile` 의 `FROM alpine:3.21` → `alpine:3.23`. 빌드 단계가 최신 패치를 받는지는 `docker build --pull` 로 확인합니다(V-2). `apk add` 패키지(ca-certificates, tzdata)와 사용자 생성은 그대로. 빌드 단계 태그는 바꾸지 않습니다.
- **대시보드:** `apps/server/web` 에서 같은 메이저 안 업데이트(`pnpm up`)로 minimatch 경유 보고를 없앱니다. 안 되면 그 lockfile 에 한해 override.
- **문서:** `docs/server/release/0.1.1.md` 의 업그레이드 안내에 "의존성 · 런타임 이미지(Alpine 3.23) 갱신" 한 줄, 바이너리 크기 재측정.

### 데스크톱
- **Electron:** `apps/desktop/package.json` 의 `electron` 을 `^43.8.10` 으로. 메이저는 43 유지.
- **업데이트 범위:** 데스크톱 · 루트 · `packages/*` 의 `dependencies` 와 `devDependencies` 를 **모두** 같은 메이저 안 최신으로 올립니다(0.x 패키지는 같은 마이너 안, oxfmt 만 예외 — 아래). 런타임 쪽 예: `better-sqlite3` 13.2.0, `vue-i18n`, `vue-router`. 개발 쪽 예: `@iconify-json/lucide`(devDependencies). `better-sqlite3` 는 Electron ABI 에 맞춰 다시 빌드되는지(`electron-builder install-app-deps` / postinstall) 확인합니다.
- **개발 · 빌드 도구:** `electron-builder` 26.18.x, `vite` 8.3.8, `@vitejs/plugin-vue`, `postcss`, `eslint` 등 같은 메이저 안 최신. **oxfmt** 는 0.x 라 마이너가 포맷 결과를 바꿀 수 있습니다. tinypool 보고를 없애는 가장 낮은 oxfmt 버전으로 올리고, 포맷 결과가 바뀌면 그 차이를 **`pnpm format` 만 실행한 커밋 하나**(`chore: oxfmt …`)로 따로 둡니다. 이 커밋은 "기능 코드 diff 0줄" 검사에서 빼고 대신 "그 커밋의 부모에서 새 oxfmt 로 `pnpm format` 을 다시 돌리면 같은 트리가 나온다"로 확인합니다(V-11).
- **override:** 위로도 남는 전이 의존성(undici, fast-uri, minimatch 등)은 `pnpm.overrides` 에 **패치된 최소 버전**으로 둡니다. 루트 `package.json` 과 `apps/server/web/package.json`(대시보드는 자체 lockfile) 모두 해당합니다. 패치된 버전이 상위 메이저에만 있으면(예: minimatch 9/10 → 11.0.1) 메이저를 넘는 override 도 허용합니다. 그 경우 그 패키지를 쓰는 도구(electron-builder, eslint, vue-tsc 등)의 게이트 · 빌드 · 패키징 통과로 확인하고, 깨지면 그 항목은 남은 위험으로 표에 적고 사용자에게 묻습니다. 이유는 아래 표에 적습니다(JSON 은 주석 불가).
- **버전:** 데스크톱 버전은 릴리스 워크플로가 올립니다(`apps/desktop/package.json` 의 `version` 은 손대지 않음).
- **문서:** 데스크톱 1.3.1 릴리스 노트 본문은 `.loctx/release/1.3.1-rc.1_Pre-Release.md` 에 씁니다(`.github/release-notes.md` 규약).
- **기록:** 서버 부분의 history 는 `.loctx/history/server-0.1.1.md`, 데스크톱 부분은 `.loctx/history/1.3.1.md` 입니다. 병합 직전 SHA 대조도 각 파일 기준입니다.

### override 표 (구현 때 채움)
| 패키지 | 버전 | 경유 | 이유 |
|---|---|---|---|
| (구현 때 채움) | | | |

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
qa-designer(google gemini-4-argon-high)가 작성하고 PM 이 검토 · 수정했습니다. qa-designer 가 짚은 모호한 점 5개(oxfmt 포맷 커밋과 무변경 검사의 충돌, 메이저를 넘는 override, 데스크톱 업데이트 범위, 빌드 이미지 확인 방법, history · 릴리스 노트 경로)는 설계에 정해 넣었습니다. PM 수정: V-8 의 메이저 목록을 손으로 적은 목록 대신 master 와 비교하는 스크립트로, V-10 기대 결과의 괄호 문장 삭제, V-11 에 포맷 커밋 처리.

- V-1 **[1단계 · 서버 0.1.1 병합 전] 서버 Go 모듈 최신화(`go list -m`) · 메이저 버전 불변 · `govulncheck` 심볼 및 모듈 결과 0건 검증**
  - 검증 대상: S-1, W-1
  - 실행 방법:
    1) 서버 병합 전 `apps/server` 디렉터리에서 `go.mod`의 툴체인 지시어(`grep -E '^go ' go.mod`)와 직접 의존성 목록 및 버전을 확인합니다: `go list -m -f '{{if not .Indirect}}{{.Path}} {{.Version}}{{end}}' all`
    2) 같은 메이저 안에서 남아 있는 상위 버전이 있는지 확인합니다: `go list -m -u all`
    3) 취약점 정적 분석을 상세 모드로 실행합니다: `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 -show verbose ./...`
  - 기대 결과:
    1) `apps/server/go.mod`의 Go 지시어는 `go 1.26.0` 그대로이고, 직접 의존성은 기존 6개 모듈(`github.com/pquerna/otp` `v1.x`, `github.com/prometheus/client_golang` `v1.x`, `golang.org/x/crypto` `v0.x` [`v0.59.0` 이상], `golang.org/x/term` `v0.x`, `golang.org/x/time` `v0.x`, `modernc.org/sqlite` `v1.x`)뿐이며 모든 모듈의 메이저 버전이 바뀌지 않았습니다.
    2) `go list -m -u all` 출력에서 직접 의존성 6개에 대해 같은 메이저 내 미적용 업데이트(`[v...]`)가 `0`건입니다.
    3) `govulncheck -show verbose ./...`가 종료 코드 `0`으로 끝나고, `=== Symbol Results ===` 아래 `No vulnerabilities found.`, `=== Package Results ===` 아래 `No other vulnerabilities found.`, `=== Module Results ===` 아래 `No other vulnerabilities found.`(기존 `GO-2026-5932`를 포함해 `Vulnerability #` 항목 `0`건)가 출력되며, 마지막 요약에 `Your code is affected by 0 vulnerabilities.`가 표시됩니다.
  - 실패 판정: `go 1.26.0` 지시어가 바뀌었거나, 직접 의존성이 6개에서 가감되었거나, 어느 한 모듈이라도 메이저 버전이 바뀌었거나, `govulncheck`의 Symbol · Package · Module Results 중 한 곳이라도 취약점이 `1`건 이상 보고되거나 종료 코드가 `0`이 아니면 실패입니다.
  - 증적 경로: `.loctx/reports/sec-004/v01-server-go-deps-govulncheck.log`

- V-2 **[1단계 · 서버 0.1.1 병합 전] 서버 Docker 이미지 빌드 및 런타임 컨테이너 Alpine 3.23.x · 빌드 스테이지 패치 확인**
  - 검증 대상: S-2, W-2
  - 실행 방법:
    1) 서버 병합 전 `apps/server/Dockerfile`의 베이스 이미지 태그와 런타임 설정을 확인합니다: `grep -n '^FROM ' apps/server/Dockerfile` 및 `git diff master...HEAD -- apps/server/Dockerfile`
    2) 최신 패치 베이스 이미지를 받아 서버 이미지를 빌드합니다: `docker build --pull -t croffle-server:0.1.1-sec-test apps/server`
    3) 빌드된 런타임 이미지에서 Alpine 배포판 버전과 실행 계정, 서버 버전을 확인합니다:
       - `docker run --rm --entrypoint cat croffle-server:0.1.1-sec-test /etc/alpine-release`
       - `docker run --rm --entrypoint id croffle-server:0.1.1-sec-test`
       - `docker run --rm croffle-server:0.1.1-sec-test --version`
       - `docker image inspect node:24-alpine golang:1.26-alpine --format '{{.RepoTags}} {{.Id}} {{.Created}}'`
  - 기대 결과:
    1) `apps/server/Dockerfile`의 빌드 단계는 `FROM node:24-alpine AS web`, `FROM golang:1.26-alpine AS build`를 유지하고, 런타임 단계만 `FROM alpine:3.21`에서 `FROM alpine:3.23`으로 변경되었으며 `apk add --no-cache ca-certificates tzdata`와 `croffle` 사용자 생성 설정은 그대로입니다.
    2) `docker build --pull`이 종료 코드 `0`으로 완료됩니다.
    3) 런타임 이미지의 `/etc/alpine-release` 출력이 `3.23.x`(`^3\.23\.[0-9]+$`)이고, 기본 실행 사용자는 `uid=...(croffle) gid=...(croffle)`이며, `--version`이 `sync protocol 1-1`을 출력합니다.
  - 실패 판정: `Dockerfile`에 `alpine:3.21`이 남아 있거나, 빌드 단계 태그(`node:24-alpine`, `golang:1.26-alpine`)의 메이저/마이너가 바뀌었거나, `docker build`가 실패하거나, `/etc/alpine-release`가 `3.23.x`가 아니거나, `ca-certificates`·`tzdata`·`croffle` 계정이 누락되면 실패입니다.
  - 증적 경로: `.loctx/reports/sec-004/v02-server-docker-alpine.log`

- V-3 **[1단계 · 서버 0.1.1 병합 전] 서버 대시보드(`apps/server/web`) `pnpm audit --json` 취약점 0건 · 메이저 버전 불변 · 빌드 검증**
  - 검증 대상: S-3, W-3
  - 실행 방법:
    1) 서버 병합 전 대시보드 의존성 잠금 파일 일관성과 빌드를 확인합니다: `pnpm --dir apps/server/web install --frozen-lockfile && pnpm --dir apps/server/web build`
    2) 대시보드 취약점 감사를 JSON 모드로 실행합니다: `pnpm --dir apps/server/web audit --json`
    3) `apps/server/web/package.json`의 의존성 메이저 버전을 확인하고(`jq '{dependencies, devDependencies, pnpm}' apps/server/web/package.json`), `apps/server/web/package.json`을 수정한 커밋이 `apps/server/web/pnpm-lock.yaml`을 같은 커밋에서 함께 수정했는지 확인합니다.
  - 기대 결과:
    1) `pnpm --dir apps/server/web install --frozen-lockfile`과 `pnpm --dir apps/server/web build`가 모두 종료 코드 `0`으로 통과합니다.
    2) `pnpm --dir apps/server/web audit --json`이 종료 코드 `0`으로 끝나고, `.metadata.vulnerabilities`의 `info`, `low`, `moderate`, `high`, `critical` 값이 모두 `0`(`{"info":0,"low":0,"moderate":0,"high":0,"critical":0}`)이며 `.advisories`가 빈 객체(`{}`)입니다(기존 `minimatch` moderate 1건 해소).
    3) `apps/server/web/package.json`의 메이저 버전(`vue` `3.x`, `@types/node` `24.x`, `@vitejs/plugin-vue` `5.x`, `typescript` `5.x`, `vite` `6.x`, `vue-tsc` `2.x`)이 그대로이며, 만약 `pnpm.overrides`를 사용했다면 해당 항목이 spec의 override 표에 이유와 함께 기재되어 있습니다.
  - 실패 판정: `pnpm --dir apps/server/web audit --json`의 `metadata.vulnerabilities` 중 하나라도 `0`이 아니거나, 대시보드 빌드가 실패하거나, 의존성 메이저 버전이 바뀌었거나, `package.json`과 `pnpm-lock.yaml`이 다른 커밋으로 분리되었으면 실패입니다.
  - 증적 경로: `.loctx/reports/sec-004/v03-server-web-audit.log`

- V-4 **[1단계 · 서버 0.1.1 병합 전] 서버 0.1.1 회귀 검증 (`.agents/intent/003_server-auth-sync/spec.md`의 V-1 · V-5 · V-12 재실행)**
  - 검증 대상: S-4, W-1, W-3
  - 실행 방법:
    서버 병합 전 `.agents/intent/003_server-auth-sync/spec.md`에 정의된 `V-1`, `V-5`, `V-12` 세 검사를 원문 그대로 재실행합니다:
    1) **spec 003 V-1 재실행 (정적 게이트 8종 · `build-all` · 공유 픽스처 13개)**:
       - `pnpm typecheck && pnpm lint && pnpm exec oxfmt --check && pnpm -r test && pnpm build && pnpm --filter @croffledev/croffle-cli build && pnpm --dir apps/server/web build && make -C apps/server check`
       - `make -C apps/server build-all && ./apps/server/bin/croffle-server --version`
    2) **spec 003 V-5 재실행 (0.1.0 DB 마이그레이션 0004·0005 후 기존 비밀번호 해시·기기 토큰 무중단 동작)**:
       - `go test -v ./internal/store/sqlite ./internal/httpapi -run 'TestMigrateFrom010|TestLegacyDeviceTokenAndPasswordAfterMigration'` (디렉터리: `apps/server`)
       - 0.1.0 스키마(버전 3, 계정 `alice` · 기기 토큰 `T_010` · 일정 1건) DB에 `./apps/server/bin/croffle-server db migrate --db "$DB010"` 및 `db info --json --db "$DB010"` 실행 후 0.1.1 서버를 기동하여 기존 토큰 `T_010`으로 `POST /auth/whoami`, `POST /sync/pull`, `POST /sync/push`와 기존 비밀번호로 `POST /auth/login`을 호출합니다.
    3) **spec 003 V-12 재실행 (`GET /health` 프로토콜 범위 및 `features` 목록 확인)**:
       - `go test -v ./internal/httpapi -run TestHealth` (디렉터리: `apps/server`)
       - 기동된 서버에 `curl -sf http://127.0.0.1:18787/health`를 호출해 JSON 응답을 검증합니다.
  - 기대 결과:
    1) spec 003 V-1: 8개 게이트와 `make -C apps/server build-all`이 모두 종료 코드 `0`으로 통과하고, 픽스처 13개에 대해 데스크톱 러너 13개 PASS · 서버 컨포먼스 러너 `12-row-codec` SKIP 외 12개 PASS하며, `--version`이 `sync protocol 1-1`을 출력합니다.
    2) spec 003 V-5: `schemaVersion`이 `5`, 기존 `device.tokenExpiresAt`이 `NULL`, 기존 데이터(`accounts: 1`, `devices: 1`, `rows: 1`, `commits: 1`)가 보존되며, 기존 토큰 `T_010`으로 호출한 `/auth/whoami`·`/sync/pull`·`/sync/push`와 기존 비밀번호로 호출한 `/auth/login`이 모두 HTTP `200`을 반환합니다.
    3) spec 003 V-12: `GET /health`가 HTTP `200`으로 `protocol: {"min": 1, "max": 1}`, `db: "ok"`, `server`, `time`, `retention`, 그리고 `features: ["conflict-fields", "device-token-ttl", "token-pair", "totp"]`(정확히 이 순서 4개)를 반환합니다.
  - 실패 판정: spec 003의 `V-1`, `V-5`, `V-12` 중 어느 하나라도 해당 검사의 실패 판정 조건에 걸리면 실패입니다.
  - 증적 경로: `.loctx/reports/sec-004/v04-server-regression-003.log`

- V-5 **[1단계 · 서버 0.1.1 병합 전] 서버 기능 코드 무변경(`git diff`) · 서버 릴리스 노트(`docs/server/release/0.1.1.md`) 및 바이너리 크기 갱신 확인**
  - 검증 대상: S-4, W-4
  - 실행 방법:
    1) 서버 병합 전 기능 소스 코드와 공개 타입에 변경이 없는지 확인합니다:
       - `git diff master...HEAD -- apps/desktop/src apps/server/internal packages/types/api packages/types/models`
       - `git diff master...HEAD -- apps/server/cmd apps/server/web/src`
    2) `make -C apps/server build-all` 또는 `CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /tmp/croffle-server-stripped ./cmd/croffle-server`로 빌드한 linux/amd64 strip 바이너리의 크기를 측정합니다(`stat -c%s` 및 `ls -lh`).
    3) `docs/server/release/0.1.1.md`의 영문 `## Upgrade notes`와 국문 `<details><summary>한국어</summary>` 아래 `## 업그레이드 안내` 절을 확인합니다.
  - 기대 결과:
    1) `git diff master...HEAD -- apps/desktop/src apps/server/internal packages/types/api packages/types/models` 및 `apps/server/cmd`, `apps/server/web/src`의 diff 출력이 모두 `0`줄입니다(바뀐 파일은 의존성 매니페스트·lockfile·`Dockerfile`·문서뿐).
    2) `docs/server/release/0.1.1.md`의 영문과 국문 업그레이드 안내 양쪽에 의존성 및 런타임 이미지(`Alpine 3.23`) 갱신 안내가 반영되어 있고, 기재된 strip 기준 `linux/amd64` 바이너리 크기가 실측값과 일치합니다.
  - 실패 판정: 서버·데스크톱·타입 기능 소스에 1줄이라도 변경이 있거나, `docs/server/release/0.1.1.md`에 의존성·Alpine 3.23 갱신 안내가 누락되었거나 영/한 내용이 어긋나거나, 기재된 바이너리 크기가 실측값과 다르면 실패입니다.
  - 증적 경로: `.loctx/reports/sec-004/v05-server-no-diff-and-release-note.log`

- V-6 **[1단계 · 서버 0.1.1 병합 전] 서버 PR 병합 직전 Head SHA 리뷰 기록 대조**
  - 검증 대상: S-4, S-9
  - 실행 방법:
    서버 PR을 병합(`gh pr merge`)하기 직전에 다음 명령을 실행하고 표준출력을 그대로 증적 파일에 저장합니다:
    1) `head=$(gh pr view <서버_PR_번호> --json headRefOid -q .headRefOid)`로 PR의 현재 head 40자 SHA를 가져옵니다.
    2) `grep -n "$head" .loctx/history/server-0.1.1.md`를 실행해 history 파일에 해당 `$head` SHA와 리뷰 파일 경로가 함께 기록되어 있는지 확인합니다.
    3) history 줄에 적힌 리뷰 파일의 첫 줄을 `head -n 1 <리뷰파일경로>`로 출력하여 `$head`와 대조합니다.
  - 기대 결과:
    1) `gh pr view`가 반환한 40자 `$head` SHA가 `.loctx/history/server-0.1.1.md`의 리뷰 완료 항목에 리뷰 파일 경로와 함께 기록되어 있습니다.
    2) 해당 리뷰 파일의 1행이 `Reviewed SHA: <head>`로서 `$head`와 정확히 일치하고, 리뷰 판정이 승인(결함 0건)입니다.
  - 실패 판정: `.loctx/history/server-0.1.1.md`에 현재 PR `$head` SHA가 없거나, 리뷰 파일의 첫 줄이 `$head`를 가리키지 않거나(리뷰 뒤 추가 커밋·재작성 발생), 증적 원본 로그가 병합 전에 저장되지 않으면 실패입니다(병합 중단).
  - 증적 경로: `.loctx/reports/sec-004/v06-server-premerge-sha.log`

- V-7 **[2단계 · 데스크톱 1.3.1 병합 전] 저장소 루트 · 데스크톱 및 서버 대시보드 `pnpm audit --json` 취약점 0건 · `pnpm.overrides` 사유 표 대조**
  - 검증 대상: S-3, S-6, W-6, W-7
  - 실행 방법:
    1) 데스크톱 병합 전 저장소 루트와 `apps/server/web` 양쪽에서 `pnpm audit --json`을 실행합니다:
       - `pnpm audit --json`
       - `pnpm --dir apps/server/web audit --json`
    2) 루트 `package.json`(및 `apps/server/web/package.json`)의 `pnpm.overrides` 항목을 추출합니다:
       - `jq -r '.pnpm.overrides // {} | to_entries[] | "\(.key)\t\(.value)"' package.json`
    3) 추출한 모든 `pnpm.overrides` 항목을 `.agents/intent/004_dependency-security/spec.md`의 `### override 표`와 1:1로 대조합니다.
  - 기대 결과:
    1) 루트 `pnpm audit --json`과 `pnpm --dir apps/server/web audit --json` 모두 종료 코드 `0`으로 끝나고, 각 출력의 `.metadata.vulnerabilities`가 `{"info":0,"low":0,"moderate":0,"high":0,"critical":0}`이며 `.advisories`가 `{}`입니다(기존 루트의 critical 2 · high 7 · moderate 9 · low 2건 전량 해소).
    2) `package.json`의 `pnpm.overrides`에 들어간 모든 패키지 키와 버전이 `.agents/intent/004_dependency-security/spec.md`의 `### override 표`에 빠짐없이 등재되어 있고, 각 행마다 `패키지`, `버전`, `경유`, `이유`(같은 메이저 내 직접 의존성 업데이트만으로 해결되지 않는 사유와 최소 패치 버전 근거)가 빈칸이나 `—` 없이 채워져 있습니다.
    3) 직접 의존성의 같은 메이저 업데이트만으로 해소되는 패키지(예: `oxfmt` 업데이트로 해결되는 경우의 `tinypool`)는 불필요하게 `pnpm.overrides`에 들어가지 않습니다.
  - 실패 판정: 루트 또는 `apps/server/web`의 `pnpm audit --json`에서 어느 등급이든 취약점 수가 `1` 이상이거나, `pnpm.overrides`에 있는 항목 중 하나라도 `spec.md`의 override 표에 없거나 사유가 비어 있으면 실패입니다.
  - 증적 경로: `.loctx/reports/sec-004/v07-root-pnpm-audit-and-overrides.log`

- V-8 **[2단계 · 데스크톱 1.3.1 병합 전] Electron 43.8.10 버전 · 전체 패키지 메이저 버전 불변 · lockfile 동기화 확인**
  - 검증 대상: S-5, S-6, W-5, W-6, W-7
  - 실행 방법:
    1) `apps/desktop/package.json`의 `version` 필드, `electron` 선언 버전, 그리고 실제 설치된 Electron 버전을 확인합니다:
       - `jq -r '{version, electron: .devDependencies.electron}' apps/desktop/package.json`
       - `node -p 'require("./apps/desktop/node_modules/electron/package.json").version'`
    2) `master` 대비 워크스페이스의 모든 매니페스트(`package.json`, `apps/desktop/package.json`, `packages/types/package.json`, `packages/cli/package.json`, `apps/server/web/package.json`)에서 `dependencies`와 `devDependencies`의 메이저 버전이 바뀌지 않았는지 비교합니다(기대 결과 2의 스크립트, 원본 diff 도 함께 남김):
       - `git diff master...HEAD -- package.json apps/desktop/package.json packages/types/package.json packages/cli/package.json apps/server/web/package.json`
    3) `pnpm install --frozen-lockfile`을 실행하고, `git log --name-only master..HEAD`로 `package.json`을 수정한 커밋마다 해당 `pnpm-lock.yaml`이 같은 커밋에 포함되었는지, 또한 `oxfmt` 포맷 결과가 바뀌었다면 별도 커밋(`chore: oxfmt …`)으로 분리되었는지 확인합니다.
  - 기대 결과:
    1) `apps/desktop/package.json`의 `"version"`은 `"1.3.0"` 그대로이고(릴리스 워크플로가 범프), `devDependencies.electron`은 `"^43.8.10"`, 실제 설치된 `apps/desktop/node_modules/electron/package.json`의 `version`은 정확히 `"43.8.10"`입니다.
    2) 다섯 매니페스트의 `dependencies` · `devDependencies` 모든 항목에 대해, `master` 의 선언 범위와 `HEAD` 의 선언 범위에서 메이저(0.x 는 마이너까지)가 같습니다. 비교는 `git show master:<파일>` 과 작업 트리의 JSON 을 읽어 항목마다 메이저(0.x 는 `0.<minor>`)를 비교하는 스크립트로 하고, 다른 항목 목록(oxfmt 를 뺀 `0`건)을 출력합니다.
    3) `pnpm install --frozen-lockfile`이 종료 코드 `0`으로 통과하고, `package.json` 변경과 `pnpm-lock.yaml` 변경이 모두 같은 커밋에 묶여 있습니다.
  - 실패 판정: 설치된 Electron 버전이 `43.8.10`이 아니거나, 위 패키지 중 하나라도 메이저 버전(또는 `0.x` 패키지의 브레이킹 마이너, 단 `oxfmt`는 `0.x` 내 허용)이 바뀌었거나, `apps/desktop/package.json`의 `"version"`이 수동 변경되었거나, `pnpm install --frozen-lockfile`이 실패하면 실패입니다.
  - 증적 경로: `.loctx/reports/sec-004/v08-desktop-versions-and-majors.log`

- V-9 **[2단계 · 데스크톱 1.3.1 병합 전] Electron 43.8.10 런타임 아래 `better-sqlite3` 네이티브 모듈(ABI) 로드 검증**
  - 검증 대상: S-5, S-7, W-5
  - 실행 방법:
    1) `pnpm --filter @croffledev/desktop postinstall`(`electron-builder install-app-deps`)을 실행해 네이티브 모듈이 Electron 43.8.10 헤더/ABI에 맞춰 빌드되는지 확인합니다.
    2) Electron 43.8.10 바이너리 자체의 Node 런타임(`ELECTRON_RUN_AS_NODE=1`)으로 `better-sqlite3`를 직접 로드해 인메모리 DB 쿼리를 수행합니다:
       `ELECTRON_RUN_AS_NODE=1 ./apps/desktop/node_modules/.bin/electron -e 'const Database = require("./apps/desktop/node_modules/better-sqlite3"); const pkg = require("./apps/desktop/node_modules/better-sqlite3/package.json"); const db = new Database(":memory:"); db.exec("CREATE TABLE t(id INTEGER PRIMARY KEY, v TEXT); INSERT INTO t(v) VALUES (\"ok\");"); const row = db.prepare("SELECT v, sqlite_version() AS sqlite FROM t").get(); console.log(JSON.stringify({ electron: process.versions.electron, abi: process.versions.modules, betterSqlite3: pkg.version, row })); db.close();'`
  - 기대 결과:
    1) `postinstall`과 Electron 런타임 실행이 모두 종료 코드 `0`으로 통과합니다.
    2) 표준출력 JSON에 `"electron":"43.8.10"`, `"betterSqlite3":"13.2.0"`(또는 `13.x` 최신 패치/마이너), `"row":{"v":"ok","sqlite":"..."}`가 출력되며 `NODE_MODULE_VERSION` 불일치나 `ERR_DLOPEN_FAILED` 예외가 발생하지 않습니다.
  - 실패 판정: 종료 코드가 `0`이 아니거나, `ERR_DLOPEN_FAILED` · `NODE_MODULE_VERSION` 불일치 오류가 출력되거나, `process.versions.electron`이 `43.8.10`이 아니면 실패입니다.
  - 증적 경로: `.loctx/reports/sec-004/v09-better-sqlite3-electron-abi.log`

- V-10 **[2단계 · 데스크톱 1.3.1 병합 전] 전체 정적 게이트 8종 및 데스크톱 패키징(`pnpm build:unpack` 또는 `pnpm build:linux`) 검증**
  - 검증 대상: S-5, S-6, S-8, W-5, W-6, W-7
  - 실행 방법:
    1) 저장소 루트에서 `.agents/harness/gates.yaml`의 8개 게이트 명령을 순서대로 실행합니다:
       `pnpm typecheck && pnpm lint && pnpm exec oxfmt --check && pnpm -r test && pnpm build && pnpm --filter @croffledev/croffle-cli build && pnpm --dir apps/server/web build && make -C apps/server check`
    2) `pnpm build:unpack`(또는 `pnpm build:linux`)을 실행하여 `electron-builder` 26.x와 `pnpm.overrides` 적용 상태에서 데스크톱 앱이 실제로 패키징되는지 확인하고, 산출물 `apps/desktop/dist/linux-unpacked/croffle` 및 `apps/desktop/dist/linux-unpacked/resources/app.asar.unpacked` 내 `better_sqlite3.node` 파일 존재를 확인합니다.
  - 기대 결과:
    1) 8개 게이트 명령이 모두 종료 코드 `0`으로 통과합니다.
    2) `pnpm build:unpack`(또는 `pnpm build:linux`)이 종료 코드 `0`으로 완료되고, `apps/desktop/dist/linux-unpacked/croffle` 실행 파일과 언팩된 `better_sqlite3.node` 바이너리가 정상 생성됩니다.
  - 실패 판정: 8개 게이트 또는 `pnpm build:unpack`(`build:linux`) 중 어느 하나라도 종료 코드가 `0`이 아니거나, `apps/desktop/dist/linux-unpacked/croffle`이 생성되지 않으면 실패입니다.
  - 증적 경로: `.loctx/reports/sec-004/v10-desktop-gates-and-unpack.log`

- V-11 **[2단계 · 데스크톱 1.3.1 병합 전] 기능 코드 · 공개 API 무변경(`git diff`) 및 `packages/*` 변경 여부에 따른 Changeset 조건부 검증 (음성 확인)**
  - 검증 대상: S-5, S-8, W-5, W-6, W-7, W-8
  - 실행 방법:
    1) 데스크톱 · 서버 · 공개 타입 기능 소스의 무변경을 확인합니다. `chore: oxfmt …` 포맷 커밋(`$F`)이 있으면 그 커밋을 빼고 비교합니다(`git diff master $F^ -- <경로>` 와 `git diff $F HEAD -- <경로>`). `$F` 자체는 `git checkout $F^` 상태에서 새 oxfmt 로 `pnpm format` 을 돌린 트리와 `$F` 의 트리가 같은지(`git diff --stat` 0줄)로 확인합니다:
       - `git diff master...HEAD -- apps/desktop/src apps/server/internal packages/types/api packages/types/models`
       - `git diff master...HEAD -- apps/desktop/drizzle packages/types/index.d.ts packages/types/enums.d.ts packages/types/croffle-api.d.ts packages/cli/src`
    2) 공개 패키지 매니페스트(`packages/types/package.json`, `packages/cli/package.json`)의 변경 여부와 `.changeset/` 아래 마크다운 파일(`README.md` 제외)의 존재 여부를 대조합니다:
       - `git diff --name-only master...HEAD -- packages/types/package.json packages/cli/package.json`
       - `find .changeset -maxdepth 1 -name '*.md' ! -name 'README.md'`
  - 기대 결과:
    1) (포맷 커밋을 뺀) `git diff` 의 출력이 정확히 `0`줄이고, 포맷 커밋이 있다면 다시 만든 트리와 같습니다(드리즐 마이그레이션, `packages/types` 선언 파일, `packages/cli/src`도 모두 `0`줄).
    2) `packages/types/package.json` 또는 `packages/cli/package.json`이 변경된 경우에는 `.changeset/*.md`(`README.md` 제외) 파일이 존재하여 변경된 패키지의 `patch` 범프를 선언하고 있으며, 반대로 두 `package.json`이 모두 변경되지 않은 경우에는 새 `.changeset/*.md` 파일이 `0`개입니다(동치 조건: Changeset 존재 ⇔ `packages/*/package.json` 변경).
  - 실패 판정: `apps/desktop/src`, `apps/server/internal`, `packages/types/api`, `packages/types/models`, `apps/desktop/drizzle`, `packages/cli/src` 중 한 곳이라도 diff가 `1`줄 이상 나오거나, `packages/*/package.json`이 바뀌었는데 Changeset이 없거나, 바뀌지 않았는데 Changeset이 추가되었으면 실패입니다.
  - 증적 경로: `.loctx/reports/sec-004/v11-no-feature-diff-and-changeset.log`

- V-12 **[2단계 · 데스크톱 1.3.1 병합 전] 데스크톱 1.3.1 패키지 기동 · 기존 1.3.0 DB 열기 · 서버 0.1.1(`server-v0.1.1-rc.1`) 실동기화 확인**
  - 검증 대상: S-5, S-7, W-5
  - 실행 방법:
    `[수동 · Linux]` V-10에서 생성한 언팩 빌드(`./apps/desktop/dist/linux-unpacked/croffle`, 또는 `build:linux` 산출물)와 패치된 서버 0.1.1(`server-v0.1.1-rc.1`)을 사용해 다음을 수행합니다:
    1) 데스크톱 1.3.0에서 사용하던 기존 프로필 디렉터리(기존 SQLite DB 파일과 일정·태그·리마인더 데이터 및 서버 0.1.1 동기화 설정·토큰이 저장된 상태)를 대상으로 데스크톱 1.3.1 바이너리를 기동합니다.
    2) 앱 화면이 정상 표시되고 기존 1.3.0 DB의 일정·태그·설정이 유실 없이 열리는지 확인합니다.
    3) 재로그인 없이 기존 동기화 설정과 기기 토큰으로 서버 0.1.1에 대해 동기화(`pull`)를 실행한 뒤, 데스크톱 1.3.1에서 새 일정을 1건 추가·수정하여 서버 0.1.1로 `push`되는지 확인합니다.
    4) 로컬 SQLite DB의 `SELECT count(*) FROM sync_outbox;` 결과와 `main.log`의 오류 유무를 확인합니다.
  - 기대 결과:
    1) Electron 43.8.10 및 업그레이드된 `better-sqlite3` 환경에서 데스크톱 1.3.1이 오류 없이 기동되고, 기존 1.3.0 DB의 모든 테이블 데이터가 그대로 조회됩니다.
    2) 기존 1.3.0의 동기화 설정과 토큰이 그대로 유지되어 서버 0.1.1(`protocol 1-1`)과 `/health` 확인 · `pull` · `push`가 모두 HTTP `200`으로 성공합니다.
    3) 동기화 직후 `sync_outbox`의 남은 행이 `0`건이고, `main.log`에 DB 로드 오류나 동기화 예외가 `0`건입니다.
  - 실패 판정: 앱이 기동되지 않거나, `better-sqlite3` ABI 오류·DB 스키마 오류가 발생하거나, 기존 1.3.0 일정/동기화 설정이 풀리거나, 서버 0.1.1과의 `pull`/`push`가 실패하거나 `sync_outbox`에 미전송 행이 남으면 실패입니다.
  - 증적 경로: `.loctx/reports/sec-004/v12-desktop-launch-and-sync.log`

- V-13 **[2단계 · 데스크톱 1.3.1 병합 전] 데스크톱 1.3.1 릴리스 노트(`.loctx/release/`) 존재 및 규약 확인**
  - 검증 대상: S-5, W-8
  - 실행 방법:
    1) `.loctx/release/` 디렉터리에서 데스크톱 1.3.1 프리릴리스 노트 초안 파일(`.loctx/release/1.3.1-rc.1_Pre-Release.md`)의 존재와 내용을 확인합니다.
    2) `.github/release-notes.md`의 데스크톱 Pre-release 템플릿 규약(`<details><summary>한국어 (Korean)</summary>`, `# Pre-Release v1.3.1-rc.1`, `## Overview`, `## Changelogs`, 영문 `# Pre-Release v1.3.1-rc.1`, `## Overview`, `## Changelogs`) 준수 여부와 Electron 43.8.10 및 의존성 보안 패치 안내 포함 여부를 확인합니다.
    3) `git status --ignored .loctx/`로 `.loctx/release/` 파일이 git 추적 대상에 올라가지 않았는지(gitignore 유지) 확인합니다.
  - 기대 결과:
    1) `.loctx/release/1.3.1-rc.1_Pre-Release.md` 파일이 존재하고 `.github/release-notes.md`의 한/영 프리릴리스 템플릿 구조를 갖추고 있으며, 기능 변화 없이 Electron 43.8.10 및 의존성 보안 패치를 포함한 버전임이 한/영 양쪽에 기술되어 있습니다.
    2) `.loctx/` 하위 파일은 커밋 대상에 포함되지 않습니다(`git status --short`에 `.loctx/` 0건).
  - 실패 판정: `.loctx/release/`에 1.3.1 릴리스 노트 파일이 없거나, 한/영 섹션 중 한쪽이 누락되었거나, `.loctx/` 파일이 git에 스테이징/커밋되었으면 실패입니다.
  - 증적 경로: `.loctx/reports/sec-004/v13-desktop-release-note.log`

- V-14 **[2단계 · 데스크톱 1.3.1 병합 전] 데스크톱 PR 병합 직전 Head SHA 리뷰 기록 대조**
  - 검증 대상: S-9
  - 실행 방법:
    데스크톱 1.3.1 PR을 병합(`gh pr merge`)하기 직전에 다음 명령을 실행하고 표준출력을 그대로 증적 파일에 저장합니다:
    1) `head=$(gh pr view <데스크톱_PR_번호> --json headRefOid -q .headRefOid)`로 데스크톱 PR의 현재 head 40자 SHA를 가져옵니다.
    2) `grep -n "$head" .loctx/history/1.3.1.md`를 실행해 history 파일에 해당 `$head` SHA와 리뷰 파일 경로가 함께 기록되어 있는지 확인합니다.
    3) history 줄에 적힌 리뷰 파일의 첫 줄을 `head -n 1 <리뷰파일경로>`로 출력하여 `$head`와 대조합니다.
  - 기대 결과:
    1) `gh pr view`가 반환한 40자 `$head` SHA가 `.loctx/history/1.3.1.md`의 리뷰 완료 항목에 리뷰 파일 경로와 함께 기록되어 있습니다.
    2) 해당 리뷰 파일의 1행이 `Reviewed SHA: <head>`로서 `$head`와 정확히 일치하고, 리뷰 판정이 승인(결함 0건)입니다.
  - 실패 판정: `.loctx/history/1.3.1.md`에 현재 PR `$head` SHA가 없거나, 리뷰 파일의 첫 줄이 `$head`를 가리키지 않거나(리뷰 뒤 추가 커밋·재작성 발생), 증적 원본 로그가 병합 전에 저장되지 않으면 실패입니다(병합 중단).
  - 증적 경로: `.loctx/reports/sec-004/v14-desktop-premerge-sha.log`

## 우려
1. **oxfmt 마이너 업데이트가 포맷을 바꿀 수 있습니다.** 그러면 저장소 전체에 포맷 차이가 생깁니다. 별도 커밋으로 분리해 되돌리기 쉽게 하고, 리뷰는 "포맷만 바뀌었는지"를 봅니다.
2. **`better-sqlite3` 마이너 업데이트는 네이티브 모듈 재빌드를 거칩니다.** Electron ABI 가 맞지 않으면 앱이 뜨지 않습니다. 실행 확인(V-n)에 앱 기동과 DB 열기를 넣습니다.
3. **override 는 상위 패키지가 기대하지 않은 버전을 강제합니다.** 패치 버전 안에서만 올리고, 빌드 · 패키징(`build:unpack`)으로 확인합니다.
4. **서버 0.1.1 감사는 이미 끝났습니다.** 서버 패치를 rc.1 에 넣으므로 병합 뒤 그 범위의 감사를 한 번 더 받습니다(intent 004 서버 부분).

## 범위 밖
- 메이저 업데이트(Electron 44/45, FullCalendar 7, TypeScript 7, electron-vite 6 등).
- 기능 변경, 데스크톱 1.3.2 이후 작업, 서버 TOTP secret 암호화.
