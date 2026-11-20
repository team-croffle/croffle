---
id: 2026-11-21_two-fails-find-the-cause
kind: failure
status: active
supersedes: null
topics: [implement, review, stop, delegation]
decisions: [two-fails-stop-and-decide]
symptoms: ["the same work unit is reviewed FAIL again and again while the PM keeps patching", "the PM hands the user a list of options for a decision that is the PM's", "a stop rule is read as 'ask the user'"]
applies_to: all
source: project
symptom: "a work unit fails review twice or more; the PM either keeps patching the same approach or stops and pushes the choice of method onto the user"
cause: "the PM treats a failing review as 'fix the finding' instead of asking why the approach keeps failing; when it stops, it reads 'bring options to the user' as the duty instead of finding the cause and deciding"
fix: "after the second FAIL of the same unit: stop patching, write the cause (the approach, not the finding), choose the method that removes the cause within what the user decided, record it as A-n, get it reviewed; ask the user only when the remedy narrows or reverses what the user decided"
check: "the second FAIL of a unit in the review ledger is followed by a cause line in history and an A-n, not by a third patch of the same kind or by a question about method"
verified: 2
approved: { by: user, at: 2026-11-21, words: "모두 수용할게." }
updated: { by: user, at: 2026-11-21, words: "그럼 네가 하는건데, 왜 나한테 짬을 때리지? 일 하기 싫어?" }
---
## 결정
같은 작업 단위가 리뷰에서 두 번 FAIL 하면 같은 방식의 땜질을 멈춥니다. 지적 하나하나가 아니라 **방식이 왜 계속 실패하는지**를 원인으로 적고(필요하면 pl · advisor 의견), 사용자가 정한 원칙 안에서 원인을 없애는 방식을 PM 이 고릅니다. 바꾼 방식은 조정(`A-n`)으로 기록하고 다시 리뷰를 받습니다.

사용자에게 멈춰 묻는 것은 해결책이 사용자가 정한 것의 범위나 보장을 줄이거나 뒤집을 때뿐입니다(예: 수용한 장치를 빼기, 알려진 허점을 안고 머지). 구현 방법을 고르는 선택지를 사용자에게 올리지 않습니다.
## 이유
- 데스크톱 1.3.2 W-6(설정 바로 적용): 리뷰 3 ~ 7 이 연달아 FAIL 했고, 화면 전체를 기준값과 비교하는 설계 자체가 원인이었는데 늦게 멈췄습니다(PR #142 감사 A-1, 사용자 "회고로 넘김").
- 데스크톱 1.3.2 회고, 서버 0.2.0 회고 브랜치의 hook: 정규식으로 셸 명령을 해석하는 방식이 리뷰마다 새 우회와 오탐을 냈습니다(review-srvretro 1 ~ 3). PM 은 두 번 멈췄지만 원인을 정하고 결정하는 대신 가 · 나 · 다 선택지를 사용자에게 올렸고, 사용자가 "뭐 어디까지 물어보는거야?", "그럼 네가 하는건데, 왜 나한테 짬을 때리지? 일 하기 싫어?"라고 지적했습니다. 원인(셸 문법을 흉내 내는 해석은 끝이 없고, hook 은 공격이 아니라 PM 의 실수를 막는 장치)을 정한 뒤 정확한 꼴만 허용하고 그 밖의 단어는 거부하는 방식으로 바꾸자 다음 리뷰에서 지적 1건만 남고 그 다음 리뷰가 통과했습니다.
## 적용
- 리뷰 원장(`.loctx/reports/review-ledger.tsv`)에서 같은 단위의 두 번째 FAIL 이 보이면 history 에 "원인: …"을 한 줄 적은 뒤에 다음 수정을 합니다.
- 원인이 사용자가 정한 원칙과 충돌하면(원칙을 지키는 방법이 없으면) 그때만 원인과 함께 사용자에게 올립니다.
## 바뀌면
reins 의 implement · review skill 이 두 번째 FAIL 뒤의 원인 기록을 요구하게 되면 이 노트를 그 skill 을 가리키는 노트로 줄입니다.
