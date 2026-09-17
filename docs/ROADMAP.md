# Croffle 공개 로드맵

커뮤니티·SNS·릴리즈 노트에 쓸 **방향성만**. 구현 세부는 내부 로드맵.

- 내부: [ROADMAP.md](./ROADMAP.md) (데스크톱) · [ROADMAP.server.md](./ROADMAP.server.md) (서버)
- 계획은 바뀔 수 있음. 일정 약속이 아님.
- 현재 데스크톱 **1.2.2**. 서버는 아직 없음.

최종 갱신: 2026-09-18

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

### 1.3 — 선택적 동기화 (beta) + 할 일

[Croffle Server 0.1 beta](#서버)와 맞춤.

- 직접 돌리는 서버에 기기를 연결해 일정 동기화
- 팀에 일정 공유 (앱 안에서 팀·원격 일정은 관리하지 않음)
- 마감 기준 **할 일(To-Do)**

### 1.3.5 — 공유를 세밀하게

[Croffle Server 0.5 beta](#서버)와 맞춤.

- 공유 일정의 수정·삭제 권한
- 태그와 리마인더를 선택적으로 함께 공유

---

## 서버

없어도 데스크톱은 그대로 동작합니다. **셀프호스트**만. 관리 화면은 서버 대시보드, 데스크톱은 캘린더를 유지합니다.
Go로 만들어 **바이너리 하나로 띄우는 것**을 목표로 합니다. Postgres 같은 별도 인프라를 요구하지 않습니다.

### 0.1 beta — 동기화가 붙기

데스크톱 1.3과 맞춤.

- 기기 사이 개인 일정 동기화
- 최소 대시보드
- 팀에 일정 공유 (beta)

### 0.2–0.4 — 쓸 수 있는 서버로

- 인증·충돌 해결
- 여러 팀, 멤버·팀 관리, 공유 취소

### 0.5 beta — 공유 데이터

데스크톱 1.3.5와 맞춤.

- 공유 권한, 태그 공유, 리마인더 선택 공유

---

## 바뀌지 않는 것

- **캘린더 우선.** 동기화·팀은 부가.
- **로컬이 기본.** 클라우드 강제는 없음.
- **확장으로 기능을 더함.** 모든 기능을 앱에 넣지 않음.
- **서버는 선택.** 셀프호스트. SaaS가 전제가 아님.
- **띄우기 쉬운 서버.** 바이너리 하나, 무거운 의존성 없이.

---

## English (short)

Croffle stays a **calendar-first local desktop**. Extensions are here now. **Self-hosted sync is optional** and comes later (desktop 1.3 / server 0.1 beta). The app works without a server.

- **1.2** — richer local schedules (reminders, tags, import/export, drag to another day) and deeper extension hooks.
- **1.3** — optional device sync, share a schedule with a team, To-Do with deadlines. No team admin inside the desktop app.
- **1.3.5** — share permissions, tags, and optional reminder sharing.
- **Server** — self-host only, a single Go binary with no heavy dependencies; dashboard for account/team admin; desktop stays the calendar.

Plans can change.

---

## 이 파일 쓰는 법

- SNS·Discussions·릴리즈 “What’s next”에 복사. 버그픽스·내부 용어·버전 쪼개기는 넣지 말 것.
- 범위가 바뀌면 내부 로드맵을 먼저 고치고, 여기에는 방향만 반영.
