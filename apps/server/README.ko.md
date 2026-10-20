# croffle-server

English: **[README.md](./README.md)**

[Croffle](../../README.ko.md)의 선택적 셀프호스트 동기화 백엔드입니다.

**없어도 Croffle은 완전히 동작합니다.** 데스크톱 앱은 로컬 캘린더가 먼저이고, 이
서버는 한 계정의 여러 기기가 변경을 주고받고 일정을 팀에 공유하기 위해 존재합니다.
기기를 하나만 쓰신다면 필요하지 않습니다.

상태: **0.1 beta.** Go 바이너리 하나, SQLite 파일 하나, 읽기 전용 대시보드.

---

## 빠른 시작

```yaml
# compose.yaml
services:
  croffle-server:
    # 버전을 고정하세요. :latest 는 정식 릴리스마다 움직입니다.
    image: ghcr.io/team-croffle/croffle-server:0.1.0
    restart: unless-stopped
    environment:
      CROFFLE_DB_PATH: /var/lib/croffle/server.db
      # 실제로 리버스 프록시가 앞에 있을 때만 켭니다.
      CROFFLE_TRUST_PROXY: 'true'
    volumes:
      - croffle-data:/var/lib/croffle
    ports:
      - '8787:8787'

volumes:
  croffle-data:
```

```bash
docker compose up -d

# 계정을 만들고 기기 토큰을 발급합니다. 토큰은 한 번만 출력됩니다.
docker compose exec croffle-server croffle-server user create alice --password-stdin
docker compose exec croffle-server croffle-server token issue alice --device-name "laptop"
```

데스크톱 앱에서 **설정 → 동기화**로 들어가 서버 URL을 넣고, 아이디·비밀번호로
로그인하거나 토큰을 붙여넣으면 됩니다. 내부적으로는 둘 다 같은 것(토큰)으로
끝나므로, 두 화면은 하나의 방식입니다.

브라우저로 서버 URL을 열면 대시보드가 나옵니다. 읽기 전용이고, 상태를 바꾸는 일은
모두 위의 명령으로 합니다.

---

## 설정

환경변수로 설정합니다. 웹훅을 제외한 모든 항목에는 같은 이름의 플래그도 있고
(`CROFFLE_DB_PATH` → `--db-path`), 둘 다 주면 플래그가 이깁니다. 웹훅 설정은
의도적으로 환경변수 전용입니다. 이벤트가 어디로 가는지는 운영자가 정하는 것이라,
그걸 바꿀 수 있는 플래그도 엔드포인트도 두지 않았습니다.

| 변수                          | 기본값                       | 의미                                                                                                                                            |
| ----------------------------- | ---------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| `CROFFLE_LISTEN`              | `:8787`                      | 수신 주소. 컨테이너에서는 포트 매핑이 동작하도록 `:8787`을 유지하세요.                                                                          |
| `CROFFLE_DB_PATH`             | `/var/lib/croffle/server.db` | SQLite 파일. 그 디렉터리가 있고 쓰기 가능해야 합니다.                                                                                           |
| `CROFFLE_LOG_LEVEL`           | `info`                       | `debug`·`info`·`warn`·`error`. 로그는 stdout의 JSON입니다.                                                                                      |
| `CROFFLE_TRUST_PROXY`         | `false`                      | `X-Forwarded-For`·`X-Forwarded-Proto`를 신뢰합니다. **프록시 뒤에서만 켜세요.** 그렇지 않으면 어떤 클라이언트든 자기 주소를 위조할 수 있습니다. |
| `CROFFLE_BASE_PATH`           | `/`                          | 프록시가 하위 경로를 떼지 않고 넘길 때의 마운트 지점(예: `/croffle`).                                                                           |
| `CROFFLE_RETAIN_COMMITS_DAYS` | `90`                         | 변경 이력 보존 기간. 최소 7일.                                                                                                                  |
| `CROFFLE_RETAIN_OPS_DAYS`     | `30`                         | 재시도 키 보존 기간. 최소 7일.                                                                                                                  |
| `CROFFLE_WEBHOOK_URL`         | —                            | 선택: 이벤트를 POST할 주소. secret이 필요합니다.                                                                                                |
| `CROFFLE_WEBHOOK_SECRET`      | —                            | 수신기가 검증할 HMAC 키.                                                                                                                        |
| `CROFFLE_WEBHOOK_EVENTS`      | `commits_applied,row_shared` | 쉼표로 구분한 이벤트 목록.                                                                                                                      |

잘못된 값은 기본값으로 때우지 않고 **기동을 거절**합니다. 엉뚱한 포트에서 듣거나
엉뚱한 파일에 쓰는 서버가, 뜨지 않는 서버보다 나쁩니다.

---

## 리버스 프록시

**서버는 평문 HTTP로 뜨고 TLS는 앞에서 끝내는 것을 전제합니다.** 프록시는 무엇이든
됩니다 — Traefik, Caddy, nginx, ingress, 터널. 서버에는 프록시에 종속된 코드가
없고, 프록시가 있다고 알려줬을 때 `X-Forwarded-*`를 존중하는 것뿐입니다.

<details>
<summary>Traefik (라벨)</summary>

```yaml
services:
  croffle-server:
    image: ghcr.io/team-croffle/croffle-server:0.1.0
    environment:
      CROFFLE_TRUST_PROXY: 'true'
    labels:
      traefik.enable: 'true'
      traefik.http.routers.croffle.rule: Host(`croffle.example.com`)
      traefik.http.routers.croffle.entrypoints: websecure
      traefik.http.routers.croffle.tls.certresolver: letsencrypt
      traefik.http.services.croffle.loadbalancer.server.port: '8787'
```

호스트가 아니라 경로 아래에 두려면, Traefik에서 접두어를 떼거나(`StripPrefix`
미들웨어) `CROFFLE_BASE_PATH=/croffle`로 서버에 알려주세요. 둘 중 하나만 하면
됩니다.

</details>

<details>
<summary>Caddy</summary>

```caddyfile
croffle.example.com {
  reverse_proxy croffle-server:8787
}
```

</details>

<details>
<summary>nginx</summary>

```nginx
server {
  server_name croffle.example.com;
  # TLS 설정 생략

  location / {
    proxy_pass http://croffle-server:8787;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }
}
```

</details>

### TLS 없이, 또는 자체 인증서로

데스크톱은 검증되지 않은 인증서를 기본적으로 거부합니다. LAN에서는 `http://` URL을
쓰면 되고, 사설 인증기관을 쓴다면 **설정 → 동기화 → 고급**에 스위치가 있습니다.
그 서버 하나에만 적용되고, 켜기 전까지는 꺼져 있습니다.

---

## 명령

모든 명령은 `--db`로 데이터베이스 파일을 지정할 수 있고, 목록 명령은 `--json`을
받습니다.

| 명령                                                        | 하는 일                                                                                    |
| ----------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| `user create <username> --password-stdin`                   | 계정을 만들고 팀에 넣습니다.                                                               |
| `user list`                                                 | 계정과 비활성 여부를 보여줍니다.                                                           |
| `user disable <username>` / `user enable <username>`        | 인증을 막거나 되돌립니다.                                                                  |
| `user set-password <username> --password-stdin`             | 비밀번호를 바꿉니다. 기존 기기 토큰은 그대로 동작합니다.                                   |
| `token issue <username> [--device-name N] [--client-id ID]` | 기기 토큰을 발급합니다. **한 번만 출력됩니다.**                                            |
| `token list [username]`                                     | 기기 목록: 생성·마지막 접속과 `pending`/`active`/`revoked`.                                |
| `token revoke <device-id>`                                  | 그 기기의 토큰을 회수합니다.                                                               |
| `team list` · `team add-member <team-id> <username>`        | 팀과 멤버십.                                                                               |
| `db migrate` · `db info` · `db vacuum`                      | 마이그레이션 적용, 통계 확인, 공간 회수.                                                   |
| `log dump [--since-seq N] [--limit N]`                      | 변경 로그를 JSON 라인으로 출력합니다. **클라이언트와 서버가 어긋났을 때 가장 먼저 볼 것.** |

`--password`보다 `--password-stdin`을 쓰세요. 플래그로 넘긴 비밀번호는 쉘 히스토리와
프로세스 목록에 남습니다.

### revoke 와 disable 의 차이

같지 않고, 그 차이가 중요합니다.

- **`token revoke`** 는 기기 하나의 토큰을 회수합니다. 계정은 다시 로그인해 새 토큰을
  받을 수 있습니다. 노트북을 잃어버렸을 때 쓰세요.
- **`user disable`** 은 계정을 멈춥니다. 기존 토큰이 즉시 동작을 멈추고 새로 받을 수도
  없습니다.

---

## 백업과 복구

서버 전체가 SQLite 파일 하나이고, `sqlite3 .backup`은 서버가 돌아가는 중에도
일관된 사본을 만들어 줍니다. 이미지에는 `sqlite3`이 들어 있지 않으니, 같은 볼륨을
공유하는 일회용 컨테이너로 실행해 결과를 `compose.yaml` 옆에 받으세요.

```bash
docker run --rm --volumes-from "$(docker compose ps -q croffle-server)" \
  -v "$PWD:/backup" alpine:3.21 \
  sh -c 'apk add -q --no-cache sqlite &&
         sqlite3 /var/lib/croffle/server.db ".backup /backup/croffle-$(date +%F).db"'
```

**`server.db`만 복사하면 안 됩니다.** 데이터베이스는 WAL 모드로 동작하므로 최근
커밋은 옆에 있는 `server.db-wal`에 들어 있고, 그 사본은 거의 비어 있을 수 있습니다.
`.backup` 대신 파일 복사를 쓰고 싶다면 서버를 먼저 멈추고 `server.db`·
`server.db-wal`·`server.db-shm`을 함께 가져가세요.

복구는 서버를 멈추고, 남은 WAL 없이 파일을 `CROFFLE_DB_PATH`에 되돌린 뒤 다시
띄우면 됩니다. 이미지의 서버는 uid 100으로 동작하기 때문에 사본의 소유자를
넘겨줍니다.

```bash
docker compose stop
docker run --rm --volumes-from "$(docker compose ps -aq croffle-server)" \
  -v "$PWD:/backup" alpine:3.21 \
  sh -c 'rm -f /var/lib/croffle/server.db /var/lib/croffle/server.db-wal /var/lib/croffle/server.db-shm &&
         cp /backup/croffle-2026-10-20.db /var/lib/croffle/server.db &&
         chown 100:101 /var/lib/croffle/server.db'
docker compose up -d
```

기기 토큰은 복구 후에도 그대로 동작하므로 다시 발급할 필요가 없습니다.

> **예전** 백업으로 복구하면 변경 시퀀스가 뒤로 갑니다. 커서가 그보다 앞선 기기는
> `410 cursor_too_old`를 받고 다시 부트스트랩하며, 데스크톱이 알아서 처리합니다.
> 클라이언트의 데이터가 사라지지는 않지만, 각 기기가 상태를 한 번 다시 받습니다.

---

## 보존 정책

이력은 영구 보관하지 않습니다.

- **변경 로그**: `CROFFLE_RETAIN_COMMITS_DAYS` (기본 90일)
- **재시도 키**: `CROFFLE_RETAIN_OPS_DAYS` (기본 30일)

모든 행의 **현재 상태는 절대 지우지 않습니다.** 이력만 지웁니다. 보존 기간보다 오래
떨어져 있던 기기는 "커서가 너무 오래됐다"는 응답을 받고 새 스냅샷을 받아 갑니다.
잃는 것은 없습니다. `GET /health`와 `db info`가 보존 범위를 알려주고, 그게 이런 일이
생겼을 때의 설명이 됩니다.

두 기간 모두 **7일 미만으로 둘 수 없습니다.** 더 짧으면 보존이 지탱하는 보장이
깨지고(휴가 간 노트북의 커서가 만료되거나, 늦은 재시도가 두 번 적용되거나) 증상이
설정에서 멀리 떨어진 곳에 나타납니다.

정리로 비워진 공간은 `db vacuum`으로 회수합니다. 전체 복사 동안 데이터베이스를
잠그기 때문에 의도적으로 수동이고, 시점은 운영자가 고릅니다.

---

## 웹훅

서버에는 **플러그인 시스템이 없습니다.** 여기서 확장성은 "무슨 일이 있었는지 다른
쪽에 알릴 수 있다"는 뜻입니다.

```yaml
environment:
  CROFFLE_WEBHOOK_URL: https://hooks.example.com/croffle
  CROFFLE_WEBHOOK_SECRET: 길고-무작위한-문자열
  CROFFLE_WEBHOOK_EVENTS: commits_applied,row_shared
```

| 이벤트              | 시점                                            |
| ------------------- | ----------------------------------------------- |
| `commits_applied`   | push가 적용됨. 어떤 행이 움직였는지 알려줍니다. |
| `row_shared`        | 개인 일정에 팀이 붙음.                          |
| `device_registered` | 로그인이 기기를 만들거나 되살림.                |
| `auth_succeeded`    | 자격증명이 통과함(동기화 또는 대시보드).        |
| `retention_swept`   | 정리가 이력을 지움.                             |

이벤트는 **어떤** 행이 움직였는지만 알리고, 그 안에 무엇이 적혀 있는지는 알리지
않습니다. payload도, 토큰도, 비밀번호도 들어가지 않습니다. 전송은 요청 경로 밖에서
최선 노력으로 이뤄집니다. 느린 수신기가 동기화를 늦출 수 없고, 무한정 쌓이는 대신
버려집니다.

각 요청에는 `X-Croffle-Event`·`X-Croffle-Delivery`와, 본문을 secret으로 HMAC한
`X-Croffle-Signature: sha256=<hex>`가 붙습니다. GitHub과 같은 모양이라 이미 가진
검증 코드가 그대로 동작합니다.

```js
import { createHmac, timingSafeEqual } from 'node:crypto';

function verify(secret, body, signature) {
  const expected = 'sha256=' + createHmac('sha256', secret).update(body).digest('hex');
  return timingSafeEqual(Buffer.from(expected), Buffer.from(signature));
}
```

secret 없이 URL만 주면 **기동을 거절합니다.** 발신자를 검증할 수 없는 수신기는 URL을
알게 된 누구든 믿어야 하기 때문입니다.

---

## 호환성

| 서버  | 동기화 프로토콜 | 데스크톱 |
| ----- | --------------- | -------- |
| 0.1.x | 1               | 1.3.x    |

`GET /health`가 이 서버가 말하는 범위를 알려줍니다.

```json
{
  "server": "0.1.0",
  "protocol": { "min": 1, "max": 1 },
  "db": "ok",
  "retention": { "oldestCommitSeq": 1041, "purgedThroughSeq": 1040 },
  "time": 1792000000000
}
```

데스크톱이 연결할 때 이를 확인하고 어느 쪽을 올려야 하는지 알려줍니다.
`retention`은 재부트스트랩이 왜 일어났는지를 설명해 주고(아래 참고), `time`은
밀리초 단위 서버 시계입니다. 데스크톱이 자기 시계와 비교해 시간이 많이 어긋난
기기를 경고합니다. 충돌은 벽시계로 판정하므로, 틀린 시계는 항상 이기거나 항상
집니다.

`/health`는 무인증 공개이고 데이터베이스에 닿지 못하면 503을 반환합니다. 컨테이너
헬스체크와 프록시가 둘 다 이걸 봅니다.

---

## 보안 메모

- **토큰은 기기당**입니다. 노트북 하나를 잃으면 그 하나만 회수하면 됩니다.
- 토큰은 해시만 저장하므로 데이터베이스 사본을 동작하는 자격증명으로 바꿀 수
  없습니다. 비밀번호도 같습니다(argon2id).
- **대시보드 세션과 동기화 토큰은 서로 다른 자격증명**이고, 상대의 자리에서는 둘 다
  거부됩니다. 대시보드 세션으로는 push할 수 없습니다.
- 대시보드는 로그인한 계정의 데이터만 보여줍니다. 다른 계정을 보려면 위의 명령을
  쓰는데, 그건 이미 파일 접근을 요구합니다.
- 명령은 설계상 인증을 우회합니다. 데이터베이스 파일에 바이너리를 실행할 수 있는
  사람은 이미 운영자입니다.

---

## 0.1 범위의 한계

- **팀은 하나**입니다. 다중 팀은 이후 버전입니다.
- **공유 취소가 아직 없고**, 공유한 일정을 원격에서 관리할 수도 없습니다.
- 태그와 리마인더는 일정과 함께 공유되지 않습니다.
- 토큰은 만료되지 않습니다. 데스크톱이 비밀번호를 저장하지 않아 조용히 갱신할 수
  없기 때문입니다. 리프레시 토큰은 이후 버전입니다.
- 대시보드는 읽기 전용이고 metrics는 아직 없습니다.
- 계정 비활성화는 차단일 뿐이고, 데이터를 지우지는 않습니다.

---

## 개발

```bash
cd apps/server
make check      # gofmt, vet, go test (공유 프로토콜 시나리오 포함)
make run        # ./server.db 로 :8787 에서 실행
make build      # 대시보드 없는 바이너리
make build-all  # 대시보드 + 바이너리
```

대시보드는 [`web/`](./web)의 Vue 앱이고 자체 lockfile을 가지며 `go:embed`로
바이너리에 들어갑니다. `make build`는 이를 건너뛰므로 Go 변경에 Node 툴체인이 필요하지
않고, 그때 바이너리는 `/`에서 그 사실을 알려줍니다.

[`packages/sync-fixtures`](../../packages/sync-fixtures)의 프로토콜 시나리오는 이
서버의 테스트**와** 데스크톱이 함께 읽습니다. 한쪽만 바뀐 와이어 변경은 CI에서
걸립니다.

라이선스: 저장소와 동일한 MIT.
