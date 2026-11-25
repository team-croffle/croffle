# Croffle 공개 로드맵

커뮤니티·SNS·릴리즈 노트에 쓸 **방향성만**. 구현 세부는 내부 로드맵.

- 정해진 방향만 여기에 씁니다. 구현 세부와 아이디어는 내부 계획에 둡니다.
- 계획은 바뀔 수 있음. 일정 약속이 아님.
- 현재 데스크톱 **1.3.1** (stable — 선택적 동기화 beta), **1.3.2-rc.2** (preview — 디자인 시스템 적용), **1.3.3-rc.1** (preview — 할 일). 서버 **0.2.0** (beta).

최종 갱신: 2026-11-26 — 데스크톱 1.3.3-rc.1(할 일). 2026-11-22 — 공유를 **캘린더** 모델로 다시 설계하면서 데스크톱 1.3.5 를 1.4 로, 서버 0.2.1 · 0.3 을 0.3 으로 합쳤습니다.

---

## 한 줄

**캘린더가 중심인 로컬 데스크톱.** 확장은 지금, **셀프호스트 동기화는 선택**이며 나중에 붙습니다. 서버 없이도 Croffle을 씁니다.

---

## 데스크톱

### 1.2 — 로컬 일정을 깊게

오프라인 캘린더를 먼저 씁니다. 동기화는 아직 없습니다.

- 일정 리마인더, 태그, 가져오기/보내기
- 캘린더에서 일정을 다른 날로 옮기기
- 확장 연결을 더 넓힘 (컨텍스트 메뉴 등)
- 확장이 쓸 수 있는 API와 앱만 쓰는 API를 나눔 (동기화가 붙기 전에)

### 1.3 — 선택적 동기화 (beta) + 할 일

[Croffle Server 0.1–0.2 beta](#서버)와 맞춤.

- 직접 돌리는 서버에 기기를 연결해 일정 동기화
- 화면을 디자인 시스템에 맞춤
- 마감 기준 **할 일(To-Do)**

### 1.4 — 캘린더와 공유 (beta)

[Croffle Server 0.3 beta](#서버)와 맞춤. 서버 0.3 은 데스크톱 1.4 이상이 필요합니다(beta 라 이전 데스크톱과 호환하지 않음).

- 내 개인 캘린더와 내가 속한 팀 캘린더를 메인 캘린더에 함께 표시, 캘린더별로 켜고 끄기
- 일정을 팀 캘린더에 **추가**하거나, 내 일정을 여러 팀 캘린더에 **공유**
- 공유 취소 (내 일정은 남고 팀 캘린더에서만 빠짐)
- 팀 캘린더에서 나갈 때 팀 일정을 남길지 지울지 (캘린더 정책에 따름)
- 캘린더와 멤버 관리는 서버 대시보드에서. 데스크톱은 쓰는 쪽

---

## 서버

없어도 데스크톱은 그대로 동작합니다. **셀프호스트**만. 관리 화면은 서버 대시보드, 데스크톱은 캘린더를 유지합니다.
Go로 만들어 **바이너리 하나로 띄우는 것**을 목표로 합니다. Postgres 같은 별도 인프라를 요구하지 않습니다.

### 0.1 beta — 동기화가 붙기

데스크톱 1.3과 맞춤.

- 기기 사이 개인 일정 동기화
- 최소 대시보드
- 팀에 일정 공유 (beta)

### 0.1.1–0.2 — 쓸 수 있는 서버로

- 인증·충돌 해결
- 운영자 화면, 여러 팀

### 0.3 beta — 캘린더

데스크톱 1.4와 맞춤. 데스크톱 1.4 이상이 필요합니다.

- 계정마다 **개인 캘린더**, 누구나 만드는 **팀 캘린더**
- 역할: 읽기 전용 · 편집자 · 공동 소유자 · 소유자. 초대는 받는 사람이 수락
- 팀 캘린더에 일정 **추가**(팀의 일정) · 내 일정 **공유**(내 일정을 여러 팀 캘린더에), 공유 취소
- 공유 일정은 태그와 함께, 리마인더는 각자
- 나가기 · 소유자 넘기기 · 캘린더 삭제(이름을 입력해 확인)
- 운영자는 캘린더 목록만 보고 일정 내용은 보지 않음

### 0.3.x — 캘린더 다듬기

- 일정 변경 로그와 지운 일정 복구(1주일)
- 가입 방식: 자유 가입 · OIDC · 운영자만
- 대시보드 화면 고도화

---

## 바뀌지 않는 것

- **캘린더 우선.** 동기화·팀은 부가.
- **로컬이 기본.** 클라우드 강제는 없음.
- **확장으로 기능을 더함.** 모든 기능을 앱에 넣지 않음.
- **서버는 선택.** 셀프호스트. SaaS가 전제가 아님.
- **띄우기 쉬운 서버.** 바이너리 하나, 무거운 의존성 없이.

---

## English (short)

Croffle stays a **calendar-first local desktop**. Extensions are here now. **Self-hosted sync is optional** (desktop 1.3 / server 0.1–0.2 beta). The app works without a server.

- **1.2** — richer local schedules (reminders, tags, import/export, drag to another day) and deeper extension hooks, with a clear line between the API extensions use and the app's own.
- **1.3** — optional device sync, a design-system pass, To-Do with deadlines.
- **1.4** — calendars (beta): your personal calendar and your team calendars side by side, each can be shown or hidden; add a schedule to a team calendar or share one of yours with several; unshare. Needs server 0.3.
- **Server 0.3** — calendars: a personal calendar per account, team calendars anyone can create, roles (viewer, editor, co-owner, owner), invitations the invitee accepts. Operators see which calendars exist, not what is in them. Needs desktop 1.4 (beta, no compatibility with earlier desktops).
- **Server 0.3.x** — change history and restoring deleted schedules (one week), sign-up modes (open, OIDC, operators only), a reworked dashboard.
- **Server** — self-host only, a single Go binary with no heavy dependencies; calendar and member management in its dashboard; the desktop stays the calendar.

Plans can change.

---

## 이 파일 쓰는 법

- SNS·Discussions·릴리즈 “What’s next”에 복사. 버그픽스·내부 용어·버전 쪼개기는 넣지 말 것.
- 범위가 바뀌면 내부 로드맵을 먼저 고치고, 여기에는 방향만 반영.
