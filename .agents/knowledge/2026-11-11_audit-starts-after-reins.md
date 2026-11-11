---
id: 2026-11-11_audit-starts-after-reins
kind: decision
status: active
supersedes: null
topics: [audit, harness, migration]
decisions: [audit-scope-post-reins]
symptoms: ["no intent or spec for a version built before reins", "audit finds steps from the old .ai/ workflow"]
applies_to: all
source: user
approved: { by: user, at: 2026-11-11 }
---
## 결정
감사는 reins 하네스를 들인 뒤에 시작한 작업에만 적용합니다. 그 전의 옛 흐름(`.ai/` · `croffle-workflow`)으로 만든 작업은 감사에서 통과로 처리합니다.
경계는 하네스를 들인 PR(`chore/reins-harness`)의 병합입니다. 그 PR까지는 옛 흐름이고, 그 뒤에 쓴 intent부터 새 흐름으로 감사합니다.

사용자 원문 (2026-11-11): "다음 audit에서는 기존 흐름은 감사에서 통과시켜. 새 하네스 이후로만 하면 됨."
## 이유
옛 흐름에는 intent · spec · 승인 기록이 없어서, 새 기준으로 판정하면 이미 끝난 작업이 모두 "확인 불가"나 "위반"이 됩니다. 고칠 방법도 없는 판정이라 의미가 없습니다.
## 적용
- 서버 0.1.0 (PR #113)과 데스크톱 1.2.6 이전 버전: 감사 통과. `.agents/audit/server-0.1.0.md`에 이 결정을 근거로 기록합니다.
- 하네스 전환 PR 자체(생성 파일 추가, 사용자가 허락한 `gates.yaml` 손 수정)도 경계 쪽이라 통과입니다.
- 데스크톱 1.3.0부터는 흐름 전체를 감사합니다. 옛 작업 파일(`.loctx/context/migrated-work/`)을 재료로 쓰더라도, 새 intent · spec · 승인을 거쳐야 합니다.
## 바뀌면
사용자가 옛 작업을 다시 감사하라고 하면 이 기록을 `superseded`로 바꿉니다.
