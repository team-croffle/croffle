---
id: 2026-11-17_check-wording-and-raw-evidence
kind: failure
status: active
supersedes: null
topics: [spec, test, evidence, audit]
decisions: [dry-run-checks-before-approval, raw-file-per-command, mismatch-stops-the-pass]
symptoms: ["a check's expected text never appears in the tool's real output", "go test -run prints 'no tests to run' and the check is marked passed", "a gate log keeps only exit=0", "a manual check is passed on one sentence from the user"]
applies_to: all
source: project
symptom: "the audit finds a check marked passed whose wording does not match the raw output, or whose evidence is a summary"
cause: "checks are approved without ever being run; at test time the PM is runner and judge, and reads a mismatch as 'means the same, so pass'"
fix: "dry-run every check's commands before showing the spec summary; keep one raw file per command; judge per sub-condition with file:line; a mismatch stops and goes to the user as a wording correction"
check: "before writing a V-n as passed: every expected string is found with grep -F in a raw file, and every test it names exists (go test -list)"
verified: 3
approved: { by: user, at: 2026-11-17, words: "P-3. 인용" }
---
## 결정
- **spec 단계:** 사용자에게 요약을 보이기 전에 V-n 의 명령을 지금 트리에서 한 번씩 돌려 봅니다. 명령 · 스크립트 · 경로 · 인용한 테스트 이름이 있는지, 기대 문자열이 실제 출력 형식과 맞는지 대조하고, 못 돌린 것(구현 뒤에만 의미 있는 것, 수동 확인)은 그렇다고 적습니다.
- **증적:** 명령 하나 = 원본 파일 하나. 첫 줄 `$ 명령`, 본문은 stdout · stderr 전체, 끝 줄 `exit=N`. 디렉터리에 `00-context.txt`(SHA · 시각 · `git status`). `| wc -l`, `grep`, `tail` 같은 발췌 · 집계는 원본을 남긴 뒤 따로 둡니다. 임시 파일을 덮어쓰지 않습니다.
- **판정:** V-n 마다 기대 조건을 하위 항목으로 쪼개 "기대 / 원본 파일:줄 / 일치 · 불일치 · 미확인"을 적습니다. 한 칸이라도 일치가 아니면 그 V-n 은 통과로 적지 않습니다.
- **불일치:** 제품은 맞는데 문구가 틀렸으면 history 에 "V-n 문구 불일치, 정정 대기"로 적고 사용자에게 정정안을 올립니다. `kind: change` 승인 뒤 같은 원본으로 다시 판정합니다. 검사에 맞추려고 테스트 이름을 바꾸거나 추가하는 것도 정정 승인 뒤에만 합니다. "연속 동작" 지시는 이 멈춤을 면제하지 않습니다.
- **수동 확인:** 사용자에게 확인한 기기 · 빌드와 하위 조건을 하나씩 받아 적습니다. 답에 없는 조건은 미확인으로 둡니다.
## 이유
서버 0.1.1 감사(A-3), 서버 0.1.1-rc.1 전 감사(A-1 · A-2), 데스크톱 1.3.1 감사(A-1 · A-2)에서 세 번 연속 지적됐습니다. 실제 사례:
- govulncheck v1.9.0 은 보고가 0건이면 결과 부분에 절별 제목(`=== Symbol Results ===` 등)이 없고 `No vulnerabilities found.` 로 끝납니다. `-show verbose` 면 그 앞에 진행 메시지와 스캔한 패키지 · 모듈 목록이 나옵니다(기대 문구는 보고가 있을 때의 형식이었음).
- `go test -run '<이름>'` 은 맞는 테스트가 없어도 `testing: warning: no tests to run` 과 함께 exit 0 입니다. spec 003 의 없는 테스트 이름이 spec 004 로 복사됐습니다.
- `-e '…'` 안의 SQL 작은따옴표, better-sqlite3 13 의 `prebuilds/<platform>.node` 처럼 명령 · 경로가 실제와 다르면 사전 실행으로만 드러납니다.
- 게이트 로그를 파일 하나에 덮어써 `exit=0` 만 남았고, `git diff | wc -l` 은 앞 명령의 종료 코드를 잃습니다.
위원회 의견: `.loctx` 의 committee-evidence 요약(이 노트에 옮김). 규칙화(spec · test skill, `evidence: raw` 의 정의, 실행자 · 판정자 분리)는 reins 0.3.2 로 사용자가 적용합니다.
## 적용
- 첫 적용: 데스크톱 1.3.1. 사전 실행은 V-9 · V-10(경로) · V-11(경로) · V-13 만 했고 V-7 · V-8 은 하지 않았습니다(감사 1.3.1 C-3). 원본 · 판정 표는 V-7~V-13 본 실행에 적용했습니다. 수동 V-12 는 감사 뒤에야 항목별로 받았습니다 — 다음부터는 사전 실행을 모든 V 에, 수동 확인 항목은 처음부터.
## 바뀌면
reins 의 spec · test skill 이 사전 실행과 원본 형식을 강제하게 되면 이 노트를 그 skill 을 가리키는 노트로 줄입니다.
