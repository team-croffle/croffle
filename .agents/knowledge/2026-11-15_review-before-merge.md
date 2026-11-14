---
id: 2026-11-15_review-before-merge
kind: failure
status: active
supersedes: null
topics: [review, merge, audit]
decisions: [review-path-before-merge]
symptoms: ["a commit was merged before any reviewer saw it", "review recorded after the merge", "merge approval read as review waiver"]
applies_to: all
source: project
symptom: "a code commit reaches the default branch without a review of that exact commit"
cause: "a fix made late (after the first review round, or after an audit) went straight to test and merge; the user's merge approval was read as covering the review step"
fix: "review the commit first; write the review file's path and the reviewed SHA in history; only then run the merge command"
check: "before `gh pr merge`: .loctx/history/<version>.md names a review file whose reviewed SHA is the PR head"
verified: 1
approved: { by: user, at: 2026-11-15 }
---
## 결정
병합 명령을 실행하기 전에, 그 PR head 커밋을 본 리뷰 결과 파일의 경로와 대상 SHA 를 history 에 먼저 적습니다. 그 기록이 없으면 병합하지 않습니다. 리뷰 뒤에 코드 커밋이 더 생기면 그 커밋도 리뷰를 받습니다.

사용자의 병합 승인은 리뷰 면제가 아닙니다. 리뷰를 건너뛰려면 intent 의 `waivers:` 에 그 지시가 따로 있어야 합니다. 조건이 붙은 승인("…을 설명하고 병합")은 조건을 먼저 이행합니다.
## 이유
1.3.0-rc.2 에서 두 번 일어났습니다. PR #122 의 마지막 코드 커밋(`70be3cc`)을 리뷰어가 보지 않은 채 병합했고(감사 A-2), 그 보완 중에 PR #124(`6557133`)를 다시 리뷰 전에 병합했습니다(감사 B-1). #124 는 "사유를 설명하고 병합" 조건의 설명도 병합 뒤에 했습니다(감사 D-2). 두 번 모두 사후 리뷰에서 회귀는 없었지만, #124 의 사후 리뷰가 짚은 문제 때문에 PR 이 하나 더 필요했습니다. 사용자는 사후 수용했습니다(2026-11-14 "A-2, A-4. 수용", 2026-11-15 "1번은 수용은 할게 그럼.").
## 적용
- 한 줄짜리 수정도 같습니다.
- 첫 적용: PR #125 (`.loctx/reports/1.3.0/rc2/post-audit/review7-codex.md`, 대상 `d0347d5`, 병합 전).
## 바뀌면
병합 전 리뷰를 저장소 설정(필수 리뷰)이 강제하게 되면 이 노트를 닫습니다.
