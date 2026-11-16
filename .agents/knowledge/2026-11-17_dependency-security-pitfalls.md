---
id: 2026-11-17_dependency-security-pitfalls
kind: lesson
status: active
supersedes: null
topics: [dependencies, security, pnpm, go, gitleaks]
decisions: [overrides-live-in-pnpm-workspace-yaml, review-old-pins-on-each-security-pass, sqlite-libc-pair, no-credential-literals-in-tests]
symptoms: ["pnpm audit still reports a package that has an override", "jq .pnpm.overrides package.json prints nothing", "go get -u moves modernc.org/libc", "Gitleaks fails on a test file"]
applies_to: all
source: project
approved: { by: user, at: 2026-11-17, words: "P-4. 인용" }
---
## 결정
- **pnpm 12 의 override 는 각 루트의 `pnpm-workspace.yaml` `overrides:` 에 둡니다**(`package.json` 의 `pnpm` 필드가 아님). 저장소 루트와 `apps/server/web`(자체 루트)이 따로입니다.
- **보안 패치 때 기존 override 고정값도 다시 봅니다.** 한때 패치 버전이던 고정값이 시간이 지나면 그 자체로 취약 버전이 됩니다(1.3.1: fast-uri 3.1.8, undici 6.29.0 · 7.30.0).
- **`modernc.org/libc` 는 `modernc.org/sqlite` 의 go.mod 에 적힌 버전과 정확히 같아야 합니다**(sqlite 의 doc.go). `go get -u ./...` 는 이를 깨므로 업데이트 뒤 맞춥니다. `apps/server/go.mod` 의 해당 줄에 주석이 있습니다.
- **테스트에 credential 처럼 생긴 고정값(토큰, 해시, TOTP secret)을 넣지 않습니다.** 실행 중에 만듭니다. Gitleaks 는 전체 이력을 보므로, 한 번 커밋되면 이력을 다시 써야 지워집니다(서버 0.1.1 에서 실제로 force-push 함). 예전 형식이 필요하면 그 형식을 테스트 안에서 직접 만듭니다(예: argon2id 인코딩을 `x/crypto/argon2` 로).
- 0.x 도구(oxfmt)는 마이너가 포맷을 바꿀 수 있어, 취약점을 고치는 가장 낮은 버전으로 올리고 포맷 차이가 생기면 `pnpm format` 만 한 커밋으로 따로 둡니다.
## 이유
intent 004(서버 0.1.1 보안 패치 · 데스크톱 1.3.1)에서 하나씩 겪었습니다. libc 짝은 리뷰가 잡았고(P1), override 위치와 오래된 고정값은 구현 중에, Gitleaks 는 PR CI 에서 드러났습니다.
## 적용
- 다음 보안 패치: `pnpm-workspace.yaml` 두 곳의 overrides 전부를 audit 결과와 대조, `go mod tidy` 뒤 libc 짝 확인.
## 바뀌면
pnpm 이 override 위치를 다시 바꾸거나, modernc sqlite 가 libc 를 내장하면 해당 항목을 지웁니다.
