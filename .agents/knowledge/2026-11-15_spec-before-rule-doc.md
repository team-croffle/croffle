---
id: 2026-11-15_spec-before-rule-doc
kind: failure
status: active
supersedes: null
topics: [spec, approval, design-system, audit]
decisions: [spec-change-with-rule-doc]
symptoms: ["DESIGN.md says something the approved spec does not", "a check fails as written but the code follows the rule document", "spec corrected after the tests ran"]
applies_to: all
source: project
symptom: "a rule document (apps/desktop/DESIGN.md) or the code follows a rule that differs from the approved S-n"
cause: "the rule document is both a deliverable and a second copy of the spec's rules; editing it was treated as implementation, not as changing an approved requirement"
fix: "change spec.md in the same commit, with a kind: change approval already quoted from the user; if there is no such approval, stop and present the correction first"
check: "before a commit that touches apps/desktop/DESIGN.md: compare its rules with the S-n of the spec in force and say the result in the commit body"
verified: 3
approved: { by: user, at: 2026-11-14 }
updated: { by: user, at: 2026-11-21, words: "모두 수용할게." }
---
## 결정
승인된 spec 의 S-n 과 다른 규칙을 규칙 문서(`apps/desktop/DESIGN.md` 등 S-n 이 정하는 문서)나 코드에 넣는 커밋은, 같은 커밋에서 `spec.md` 를 고치고 사용자의 `kind: change` 승인 원문을 함께 싣습니다. 승인이 아직 없으면 커밋하지 않고 멈춰서 정정안을 먼저 제시합니다.

"spec 승인 뒤 PR · CI 까지 이어서 진행" 지시는 이 멈춤을 면제하지 않습니다.
## 이유
1.3.0-rc.2 감사 A-1: 아이콘 버튼 36px 등 네 가지 기준을 `DESIGN.md` 와 코드에 먼저 넣고(`032580d`, `eae2a3b`), 테스트가 "적힌 대로는 실패"한 뒤에야 정정 승인을 받았습니다(`9bfd979`). 실패를 숨기지는 않았지만 순서가 뒤집혔고, 사용자는 사후 수용으로 기록하게 했습니다(2026-11-14 "A-1. 사후 수용으로 기록해."). advisor 의견: `.loctx/reports/1.3.0/rc2/post-audit/advisor-a1.md`.
## 적용
- `DESIGN.md` 를 건드리는 커밋 전에 그 규칙을 현재 spec 의 S-n 과 대조하고, 결과를 커밋 본문에 `Spec-check: <S-n / A-n> — <결과>` 한 줄로 적습니다. 이 줄이 없으면 husky commit-msg(`scripts/check-design-commit-msg.mjs`)가 커밋을 거부합니다(2026-11-21, 같은 누락이 두 번째라 기계적 장치로).
- 검사(V-n)가 설계와 어긋난 것을 발견해도 같습니다: 코드를 맞추거나 정정안을 먼저 냅니다. 검사 스크립트의 기대값을 바꾸는 것도 기준 변경입니다.
- 첫 적용: `de85350` (정정 7~9 와 코드가 한 커밋, 승인 원문 포함).
- 데스크톱 1.3.2: 대조 기록 누락이 두 번 나왔습니다 — PR #142 감사 A-5(`5eb9a5d` · `5554664`, 사용자 "감사 기록에 대조 결과를 적고 수용함"), PR #144 감사 B-1(`d07a872` · `845d998` · `94305e6`, 사용자 "수용"). 대조 결과는 감사 기록에 있습니다.
## 바뀌면
규칙 문서가 spec 에서 생성되도록 바뀌면(사본이 하나가 되면) 이 노트를 닫습니다.
