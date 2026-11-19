---
id: 2026-11-15_review-before-merge
kind: failure
status: active
supersedes: null
topics: [review, merge, audit]
decisions: [review-path-before-merge]
symptoms: ["a commit was merged before any reviewer saw it", "review recorded after the merge", "merge approval read as review waiver", "a reviewer-suggested fix merged without a review of the fix", "a waiver for a direct push read as a review waiver"]
applies_to: all
source: project
symptom: "a change of content (code, tests, spec text, release documents) reaches an audit or the default branch without a review of it"
cause: "a fix made late (after the first review round, or after an audit) went straight to test and merge; the user's merge approval was read as covering the review step"
fix: "every change of content comes before the last review; each review adds a line to .loctx/reports/review-ledger.tsv; the Claude Code PreToolUse hook (.claude/hooks/premerge-review-check.sh) refuses gh pr merge and a direct push to master when anything but .agents/audit/*.md changed since the last reviewed SHA"
check: "sh .claude/hooks/premerge-review-check.sh pr <n> exits 0 before the first audit and before the merge (the hook runs it on every gh pr merge)"
verified: 2
approved: { by: user, at: 2026-11-15 }
updated: { by: user, at: 2026-11-19, words: "K-1/2는 승인" }
---
## 결정
병합 명령을 실행하기 전에, 그 PR head 커밋을 본 리뷰 결과 파일의 경로와 대상 SHA 를 history 에 먼저 적습니다. 그 기록이 없으면 병합하지 않습니다. 리뷰 뒤에 코드 커밋이 더 생기면 그 커밋도 리뷰를 받습니다.

사용자의 병합 승인은 리뷰 면제가 아닙니다. 리뷰를 건너뛰려면 intent 의 `waivers:` 에 그 지시가 따로 있어야 합니다. 조건이 붙은 승인("…을 설명하고 병합")은 조건을 먼저 이행합니다.
## 결정 (2026-11-19 갱신)
- 범위는 "내용을 바꾸는 모든 변경"입니다: 코드, 테스트, spec 본문, 릴리스 문서. 마지막 리뷰 뒤에는 감사 기록(`.agents/audit/*.md`)만 커밋합니다. 0.3.3 흐름이 감사 기록을 작업 브랜치에 커밋하게 하므로 그것만 예외 경로입니다.
- 리뷰어가 제안한 수정도, 사용자가 승인한 정정도, 그 수정분을 다시 리뷰받습니다. 리뷰 한 번은 1~5분입니다.
- "master 에 직접" waiver 는 브랜치 · PR 생략이지 리뷰 생략이 아닙니다.
- 기억이 아니라 hook 이 막습니다: 리뷰마다 `.loctx/reports/review-ledger.tsv` 에 한 줄(`<PR|master>\t<리뷰한 전체 SHA>\t<리뷰 파일>\t<판정>`), `gh pr merge` 와 master 직접 push 는 `.claude/hooks/premerge-review-check.sh` 가 막습니다. 1차 감사에 넘기기 전에도 같은 확인을 돌립니다.
1.3.0-rc.2 에서 두 번 일어났습니다. PR #122 의 마지막 코드 커밋(`70be3cc`)을 리뷰어가 보지 않은 채 병합했고(감사 A-2), 그 보완 중에 PR #124(`6557133`)를 다시 리뷰 전에 병합했습니다(감사 B-1). #124 는 "사유를 설명하고 병합" 조건의 설명도 병합 뒤에 했습니다(감사 D-2). 두 번 모두 사후 리뷰에서 회귀는 없었지만, #124 의 사후 리뷰가 짚은 문제 때문에 PR 이 하나 더 필요했습니다. 사용자는 사후 수용했습니다(2026-11-14 "A-2, A-4. 수용", 2026-11-15 "1번은 수용은 할게 그럼.").
## 적용
- 서버 0.1.1 PR #129(A-2): 이 노트가 있었지만 바로 병합했습니다. 처방(skill 문구 · hook)은 "reins 0.3.2 에서 적용"으로 미뤘고, 그때까지 병합마다 SHA 대조를 했습니다(#130~#136 재발 없음). 그 처방은 reins 0.3.3 에 들어오지 않았고, 들어왔는지 대조하지도 않았습니다.
- 서버 0.2.0 (2026-11-18~19): 리뷰 뒤 변경이 여섯 번 리뷰 없이 감사나 병합에 갔습니다 — #138 A-15 · 노트 · 재작성(1차 감사 F-1), A-17 `0055236`(리뷰 4 지적의 수정), #140 `c402710` · `5ba519f`(1차 감사 F-2), #140 head `e801612`(승인된 V-14 정정), #141 `02a5f23`(리뷰 지적의 수정), 직접 커밋 `9c16d09`(waiver). 미리뷰 변경 중 `9c16d09` 에는 실제 결함(키 덮어쓰기, A-20)이 있었습니다. 원인 조사(Fable · Astra): 규칙이 병합 때 읽는 자리와 장치에 없었고, 미뤄진 처방이 전달되지 않았으며, PM 이 리뷰 결과를 "한 번 지난 단계"로 다뤘습니다(`.loctx` 의 r1-cause 요약을 이 노트에 옮김). 이 갱신과 hook 이 그 처방입니다.
- 한 줄짜리 수정도 같습니다.
- 첫 적용: PR #125 (`.loctx/reports/1.3.0/rc2/post-audit/review7-codex.md`, 대상 `d0347d5`, 병합 전).
## 바뀌면
병합 전 리뷰를 저장소 설정(필수 리뷰 · 리뷰 원장을 보는 CI job)이 강제하거나, reins 의 merge · review skill 이 같은 확인을 갖게 되면 이 노트를 그 장치를 가리키는 노트로 줄입니다. reins 쪽 개선은 `.loctx/ROADMAP.local.md` 의 열린 항목으로 추적합니다.
