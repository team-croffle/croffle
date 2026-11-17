# croffle-server

English: **[README.md](./README.md)**

[Croffle](../../README.ko.md)의 선택적 셀프호스트 동기화 백엔드입니다.

**없어도 Croffle은 완전히 동작합니다.** 데스크톱 앱은 로컬 캘린더가 먼저이고, 이
서버는 한 계정의 여러 기기가 변경을 주고받고 일정을 팀에 공유하기 위해 존재합니다.
기기를 하나만 쓰신다면 필요하지 않습니다.

상태: **0.2 beta.** Go 바이너리 하나, SQLite 파일 하나, 자기 계정용 대시보드와 운영자용 서버 전체 화면.

---

## 빠른 시작

```yaml
# compose.yaml
services:
  croffle-server:
    # 버전을 고정하세요. :latest 는 정식 릴리스마다 움직입니다.
    image: ghcr.io/team-croffle/croffle-server:0.2.0
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

# 운영자를 겸하는 첫 계정을 만들고 기기 토큰을 발급합니다. 토큰은 한 번만 출력됩니다.
docker compose exec croffle-server croffle-server user create alice --password-stdin --operator
docker compose exec croffle-server croffle-server token issue alice --device-name "laptop"
```

누군가 일회용 코드를 켜기 전에 [비밀 키](#비밀-키)도 설정해 두십시오.

데스크톱 앱에서 **설정 → 동기화**로 들어가 서버 URL을 넣고, 아이디·비밀번호로
로그인하거나 토큰을 붙여넣으면 됩니다. 내부적으로는 둘 다 같은 것(토큰)으로
끝나므로, 두 화면은 하나의 방식입니다.

브라우저로 서버 URL을 열면 대시보드가 나옵니다. 로그인한 계정 자신의 데이터를
보여주고, 그 계정이 자기 기기를 회수하거나 일회용 코드를 켤 수 있습니다([인증](#인증)).
[운영자](#운영자)에게는 운영 화면이 더 보입니다: 계정, 모든 기기와 토큰, 팀, 서버 자체 —
아래 명령이 하는 일 전부입니다.

---

## 설정

환경변수로 설정합니다. 앞의 일곱 항목에는 같은 이름의 플래그도 있고
(`CROFFLE_DB_PATH` → `--db-path`), 둘 다 주면 플래그가 이깁니다. 나머지는
환경변수 전용입니다. 웹훅은 의도적으로 그렇습니다. 이벤트가 어디로 가는지는
운영자가 정하는 것이라, 그걸 바꿀 수 있는 플래그도 엔드포인트도 두지 않았습니다.

| 변수                           | 기본값                       | 의미                                                                                                                                                         |
| ------------------------------ | ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `CROFFLE_LISTEN`               | `:8787`                      | 수신 주소. 컨테이너에서는 포트 매핑이 동작하도록 `:8787`을 유지하세요.                                                                                       |
| `CROFFLE_DB_PATH`              | `/var/lib/croffle/server.db` | SQLite 파일. 그 디렉터리가 있고 쓰기 가능해야 합니다.                                                                                                        |
| `CROFFLE_LOG_LEVEL`            | `info`                       | `debug`·`info`·`warn`·`error`. 로그는 stdout의 JSON입니다.                                                                                                   |
| `CROFFLE_TRUST_PROXY`          | `false`                      | `X-Forwarded-For`·`X-Forwarded-Proto`를 신뢰합니다. **프록시 뒤에서만 켜세요.** 그렇지 않으면 어떤 클라이언트든 자기 주소를 위조할 수 있습니다.              |
| `CROFFLE_BASE_PATH`            | `/`                          | 프록시가 하위 경로를 떼지 않고 넘길 때의 마운트 지점(예: `/croffle`).                                                                                        |
| `CROFFLE_RETAIN_COMMITS_DAYS`  | `90`                         | 변경 이력 보존 기간. 최소 7일.                                                                                                                               |
| `CROFFLE_RETAIN_OPS_DAYS`      | `30`                         | 재시도 키 보존 기간. 최소 7일.                                                                                                                               |
| `CROFFLE_WEBHOOK_URL`          | —                            | 선택: 이벤트를 POST할 주소. secret이 필요합니다.                                                                                                             |
| `CROFFLE_WEBHOOK_SECRET`       | —                            | 수신기가 검증할 HMAC 키.                                                                                                                                     |
| `CROFFLE_WEBHOOK_EVENTS`       | `commits_applied,row_shared` | 쉼표로 구분한 이벤트 목록.                                                                                                                                   |
| `CROFFLE_DEVICE_TOKEN_TTL`     | `0` (만료 없음)              | 앞으로 로그인으로 발급하는 기기 토큰의 수명. Go duration(`720h`)으로 씁니다. **데스크톱 1.3.0 은 만료될 때마다 다시 로그인해야 합니다.** [인증](#인증) 참고. |
| `CROFFLE_ACCESS_TOKEN_TTL`     | `15m`                        | `POST /auth/token` 이 주는 access token 의 수명.                                                                                                             |
| `CROFFLE_REFRESH_TOKEN_TTL`    | `720h`                       | refresh token 의 수명. 갱신할 때마다 다시 계산됩니다. access 수명보다 길어야 합니다.                                                                         |
| `CROFFLE_METRICS_LISTEN`       | — (꺼짐)                     | Prometheus metrics 를 여는 별도 주소 `host:port`(예: `127.0.0.1:9090`). 비워 두면 포트를 열지 않습니다. [metrics](#metrics) 참고.                            |
| `CROFFLE_DASH_TRUSTED_ORIGINS` | —                            | 대시보드의 상태를 바꾸는 요청을 더 허용할 출처(`https://croffle.example.com`), 쉼표 구분. `Host` 를 바꿔 넘기는 프록시용.                                    |
| `CROFFLE_SECRET_KEY`           | — (없음)                     | 일회용 코드 secret 을 암호화하는 32바이트 키의 base64. `secret generate` 로 만듭니다. [비밀 키](#비밀-키) 참고.                                              |
| `CROFFLE_SECRET_KEY_FILE`      | —                            | 그 키를 담은 파일(예: Docker secret). 둘 중 하나만 설정합니다.                                                                                               |

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
    image: ghcr.io/team-croffle/croffle-server:0.2.0
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

| 명령                                                                                        | 하는 일                                                                                                 |
| ------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| `user create <username> --password-stdin [--operator]`                                      | 계정을 만들고 `team-default` 에 넣습니다. `--operator` 는 [운영자](#운영자)로 만듭니다.                 |
| `user list`                                                                                 | 계정, 비활성 여부, 운영자 여부를 보여줍니다.                                                            |
| `user disable <username> [--force]` / `user enable <username>`                              | 인증을 막거나 되돌립니다. 마지막 활성 운영자는 `--force` 가 있어야 합니다.                              |
| `user operator grant <username>` · `user operator revoke <username> [--force]`              | 계정을 운영자로 만들거나 일반 계정으로 되돌립니다.                                                      |
| `user set-password <username> --password-stdin`                                             | 비밀번호를 바꾸고 그 계정의 브라우저 로그인을 모두 끊습니다. 기존 기기 토큰은 그대로 동작합니다.        |
| `token issue <username> [--device-name N] [--client-id ID] [--expires-in D \| --no-expiry]` | 기기 토큰을 발급합니다. **한 번만 출력됩니다.** 플래그가 없으면 `CROFFLE_DEVICE_TOKEN_TTL` 을 따릅니다. |
| `token list [username]`                                                                     | 기기 목록: 생성·마지막 접속·토큰 만료(`never`)와 `pending`/`active`/`expired`/`revoked`.                |
| `user otp status <username>` · `user otp disable <username>`                                | 계정의 일회용 코드 상태(`off`/`pending`/`enabled`)를 보거나 끕니다.                                     |
| `token revoke <device-id>`                                                                  | 그 기기의 토큰을 회수합니다.                                                                            |
| `team list` · `team create <name> [--id ID]` · `team rename <team-id> <name>`               | 팀. 멤버십은 [팀](#팀) 참고.                                                                            |
| `team members <team-id>` · `team add-member <team-id> <username>` · `team remove-member …`  | 팀의 멤버.                                                                                              |
| `secret generate` · `secret rotate --new-key-file <path>`                                   | [비밀 키](#비밀-키)를 만들거나, 일회용 코드 secret 을 새 키로 다시 암호화합니다.                        |
| `db migrate` · `db info` · `db vacuum`                                                      | 마이그레이션 적용, 통계 확인, 공간 회수.                                                                |
| `log dump [--since-seq N] [--limit N]`                                                      | 변경 로그를 JSON 라인으로 출력합니다. **클라이언트와 서버가 어긋났을 때 가장 먼저 볼 것.**              |

`--password`보다 `--password-stdin`을 쓰세요. 플래그로 넘긴 비밀번호는 쉘 히스토리와
프로세스 목록에 남습니다.

`--password-stdin` 은 stdin 이 무엇이냐에 따라 다르게 읽습니다.

- **터미널** — `Password: ` 를 띄우고, 입력을 화면에 보이지 않게 한 줄 읽고, Enter 에서
  끝납니다. `docker compose exec` 는 기본으로 터미널을 붙여 줍니다. `docker exec` 를
  쓸 때는 `-it` 를 붙입니다.

  ```bash
  docker exec -it croffle-server croffle-server user create alice --password-stdin
  ```

  Windows Git Bash(MinTTY)에서는 `winpty` 로 감싸야 컨테이너에 터미널이 붙습니다.

  ```bash
  winpty docker exec -it croffle-server croffle-server user create alice --password-stdin
  ```

- **파이프나 파일** — 끝까지 읽고 마지막 줄바꿈만 뗍니다. 스크립트용입니다. `-t` 없이
  `-i` 만 붙입니다.

  ```bash
  printf '%s' "$PW" | docker exec -i croffle-server croffle-server user create alice --password-stdin
  ```

### revoke 와 disable 의 차이

같지 않고, 그 차이가 중요합니다.

- **`token revoke`** 는 기기 하나의 토큰을 회수합니다. 계정은 다시 로그인해 새 토큰을
  받을 수 있습니다. 노트북을 잃어버렸을 때 쓰세요.
- **`user disable`** 은 계정을 멈춥니다. 기존 토큰이 즉시 동작을 멈추고 새로 받을 수도
  없습니다.

기기를 회수하면 그 기기가 가진 토큰이 모두 멈춥니다. 기기 토큰과 access · refresh
토큰 전부입니다. 계정은 대시보드에서 자기 기기를 직접 회수할 수도 있습니다.

---

## 인증

동기화 요청은 모두 bearer 토큰을 실어 보냅니다. 토큰은 세 가지입니다.

| 토큰          | 받는 곳                           | 수명                                                | 쓰는 곳                 |
| ------------- | --------------------------------- | --------------------------------------------------- | ----------------------- |
| 기기 토큰     | `POST /auth/login`, `token issue` | 만료 없음(`CROFFLE_DEVICE_TOKEN_TTL` 을 켜면 그 값) | 데스크톱 1.3.0          |
| access token  | `POST /auth/token`                | `CROFFLE_ACCESS_TOKEN_TTL`(15분)                    | 동기화 요청             |
| refresh token | `POST /auth/token`                | `CROFFLE_REFRESH_TOKEN_TTL`(30일)                   | `POST /auth/token` 에만 |

**기기 토큰.** 만료 없음이 기본이고, 앞으로도 계속 쓸 수 있습니다. 데스크톱은
비밀번호를 저장하지 않으므로 토큰을 조용히 갱신할 수 없기 때문입니다.
`CROFFLE_DEVICE_TOKEN_TTL` 을 켜면 그 뒤 로그인으로 발급하는 토큰에 만료가 붙습니다
(이미 발급한 토큰은 그대로입니다). 데스크톱 1.3.0 은 아직 refresh token 을 쓰지 않아서,
토큰이 만료될 때마다 다시 로그인하라고 안내합니다. 설정과 상관없이
`token issue --no-expiry` 로 만료 없는 토큰을 낼 수 있습니다.

**access · refresh token.** `POST /auth/token` 에
`{"grantType": "password", "username", "password", "otp"?, "clientId", "deviceName"?}`
를 보내면 `accessToken`, `refreshToken` 과 각각의 만료 시각(unix ms)을 받습니다. access
token 이 끝나기 전에 `{"grantType": "refresh_token", "refreshToken"}` 으로 다음 쌍을
받습니다. refresh token 은 **한 번만** 쓸 수 있습니다. 이미 쓴 것을 다시 내밀면 그
기기의 access · refresh 토큰을 모두 회수합니다(기기 토큰은 그대로). 다시 내미는 것은 훔친 사본뿐이기 때문입니다.
refresh token 은 동기화 엔드포인트에서 받지 않습니다. 이 방식으로 로그인해도 그
기기의 기기 토큰은 건드리지 않으므로, 같은 기기의 1.3.0 로그인은 계속 동작합니다.
데스크톱은 아직 이 엔드포인트를 쓰지 않습니다.

**일회용 코드(TOTP).** 계정이 대시보드의 **Security** 에서 직접 켭니다. 인증 앱으로
QR 코드를 읽고, 코드로 확인하고, 한 번만 보여주는 복구 코드 10개를 보관합니다. 그
뒤로는 데스크톱 · `POST /auth/token` · 대시보드의 모든 로그인에 6자리 코드나 복구 코드가
필요합니다. 코드가 없으면 `401 otp_required` 를 돌려주고, 데스크톱 1.3.0 은 이미 이를
받아 코드를 묻습니다. 코드는 한 번만 받고(이미 쓴 시간 단계의 코드는 거부), 복구
코드도 각각 한 번만 쓸 수 있으며, 틀린 코드는 틀린 비밀번호처럼 로그인 rate limit 에
셉니다. 끄려면 계정이 비밀번호와 코드로 확인합니다. 인증 앱과 복구 코드를 모두 잃은
사람은 운영자가 `user otp disable`(또는 **Accounts** 화면)로 풀어 줄 수 있습니다.
`token issue` 는 운영자 명령이라 코드를 묻지 않습니다. 코드의 바탕이 되는 secret 은
[비밀 키](#비밀-키)를 설정하면 암호화해 저장합니다.

로그인 rate limit 은 주소 · 사용자명마다 5번, 그 뒤 30초에 1번입니다.
`otp_required` 에서 멈춘 로그인도 한 번으로 셉니다.

---

## 운영자

운영자는 대시보드의 운영 화면을 쓸 수 있는 계정입니다.

- **Accounts** — 계정 만들기, 비활성화 · 활성화, 비밀번호 재설정(그 계정의 모든
  브라우저 로그인이 끊기고, 기기 토큰은 그대로 동작), 일회용 코드 끄기, 운영자
  지정 · 해제.
- **All devices** — 모든 계정의 기기. 토큰 발급(한 번만 보여 주고 복사 버튼 제공)과 회수.
- **Manage teams** — 팀 만들기 · 이름 바꾸기, 멤버 추가 · 제거.
- **Server** — 데이터베이스 정보와 비밀 키 상태, vacuum, 모든 계정의 커밋 로그(메타데이터만.
  행 내용은 `log dump` 로 봅니다).

첫 운영자는 명령으로 정하고(`user create <name> --operator` 또는
`user operator grant <name>`), 그 뒤로는 대시보드에서 정하면 됩니다. 운영자라고 해서
동기화가 달라지지는 않습니다. 운영자의 기기도 다른 계정과 같은 행만 받습니다. 운영
작업은 하나마다 로그에 `operator action` 한 줄(누가 · 무엇을 · 어디에, 비밀번호나
토큰은 없음)을 남깁니다.

마지막 활성 운영자는 대시보드에서 해제하거나 비활성화할 수 없습니다
(`409 last_operator`). 그래서 운영 화면을 쓸 사람이 항상 남습니다. 명령으로는
`--force` 로 할 수 있습니다. 명령을 실행할 수 있는 사람은 되돌릴 수도 있기 때문입니다.

---

## 비밀 키

일회용 코드 secret 은 서버가 코드를 확인하려면 읽을 수 있어야 하므로 해시로 저장할 수
없습니다. 비밀 키가 있으면 암호화해 저장하고(AES-256-GCM, 계정에 묶임), 없으면 그대로
저장하되 서버가 시작할 때와 운영 화면에서 그렇다고 알립니다.

```bash
# 키를 만들어 데이터베이스 볼륨 밖에 둡니다.
docker run --rm ghcr.io/team-croffle/croffle-server:0.2.0 secret generate > croffle-secret.key
```

서버에는 `CROFFLE_SECRET_KEY_FILE`(Docker secret 이나 읽기 전용 파일로 마운트) 또는
`CROFFLE_SECRET_KEY` 로 넘깁니다. 다음 시작 때 서버가 아직 평문인 secret 을 모두
암호화합니다. 아무도 코드를 다시 등록하지 않아도 됩니다.

- **키는 데이터베이스와 따로 백업하십시오.** 그러면 데이터베이스 사본만으로는 코드를
  만들 수 없고, 키를 잃어도 데이터베이스는 잃지 않습니다.
- **키를 잃으면** 암호화된 secret 이 남아 있는 동안 서버가 시작을 거부합니다. 그 계정들의
  일회용 코드를 `user otp disable`(키가 필요 없음)로 끄고, 키 없이 또는 새 키로 시작한
  뒤 다시 등록하게 하십시오.
- **키를 바꾸려면** 서버를 멈추고, 옛 키가 설정된 채로 `secret rotate` 를 실행하고,
  설정을 새 키로 바꾼 뒤 서버를 시작합니다. compose 라면(서비스는 아직 옛 키를 가리킴):

  ```bash
  docker run --rm ghcr.io/team-croffle/croffle-server:0.2.0 secret generate > croffle-secret.new
  docker compose stop croffle-server
  docker compose run --rm -v "$PWD/croffle-secret.new:/new.key:ro" croffle-server \
    secret rotate --new-key-file /new.key
  mv croffle-secret.new croffle-secret.key   # 서비스가 마운트하는 파일
  docker compose up -d
  ```

  서버가 떠 있는 동안의 rotate 는 안전하지 않습니다. 떠 있는 서버는 다시 시작할 때까지
  옛 키를 쓰므로, 그 사이에 등록한 코드는 옛 키로 잠기고 다음 시작이 실패합니다. 되돌리려면
  새 키를 설정한 채로 `secret rotate --new-key-file <새 키> --old-key-file <옛 키>` 를
  실행하십시오. 두 키 중 어느 것으로 잠긴 secret 이든 열어 모두 새 키로 잠급니다.

- 저장된 secret 과 맞지 않는 키로 시작하면 서버는 시작을 거부하고, secret 을 잠근 키의
  id 를 알려 줍니다.

---

## 팀

모든 계정은 `team-default` 에 들어 있습니다. 운영자는 팀을 더 만들고 계정을 넣고 뺄 수
있습니다(`team create`, `team add-member`, `team remove-member`, 또는 **Manage teams**).
팀에 공유한 일정은 그 팀의 멤버에게 보입니다.

- **팀에 들어갈 때.** 그 계정의 기기들은 팀의 예전 행도 받아야 합니다. 기기마다 다음
  동기화에서 스냅샷을 한 번 새로 받습니다. 서버가 pull 에 `410 cursor_too_old`
  (`details.reason: "team_scope"`)로 답하고, 데스크톱 1.3.x 는 이미 이를 스냅샷으로
  처리합니다. 0.2.0 으로 올린 것만으로는 어떤 기기도 스냅샷을 다시 받지 않습니다.
- **팀에서 빠지면 연결이 끊어질 뿐, 지워지는 것은 없습니다.** 팀은 빠진 계정이 만든
  행까지 모두 그대로 갖고, 빠진 계정의 기기도 받아 둔 사본을 그대로 갖습니다. 그때부터
  서버는 그 팀의 행을 — 그 계정이 공유한 행도 — 그 계정에 보내지 않고, 그 계정의 변경은
  팀에 가지 않습니다(`403 foreign_team`. 데스크톱은 그 변경만 실패로 두고 동기화를
  계속합니다). 데스크톱이 그런 사본을 별도 일정으로 바꿀 수 있게 되기 전까지는, 그
  사본을 고친 내용이 그 기기에만 남습니다.
- 팀은 아직 지울 수 없습니다. 이름을 바꾸십시오.
- 0.1.x 에서 팀에 들어간 뒤 스냅샷을 다시 받지 않은 기기는 그 팀의 예전 행을 놓쳤을 수
  있습니다. 그 데스크톱에서 **설정 → 동기화 → 로그아웃** 후 다시 로그인하십시오.
  로그아웃하면 기기의 변경 로그 위치가 지워져 다음 동기화에서 전체 스냅샷을 받습니다
  (기기 회수만으로는 안 됩니다. 같은 계정으로 다시 로그인하면 예전 위치가 남습니다).

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

## metrics

기본은 꺼짐입니다. `CROFFLE_METRICS_LISTEN`(예: `127.0.0.1:9090`)을 주면 서버가 **두
번째 수신 주소**를 열고, 거기서 Prometheus 텍스트 형식의 `GET /metrics` 하나만
제공합니다. API 주소에는 metrics 가 없습니다. **이 주소는 내부망에만 두세요.** 자체
인증이 없습니다.

| 지표                                    | 라벨                                                                                                                    |
| --------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| `croffle_http_requests_total`           | `route`, `method`, `code`                                                                                               |
| `croffle_http_request_duration_seconds` | `route`, `method`                                                                                                       |
| `croffle_sync_push_commits_total`       | `result`: `accepted`, `conflict`, `retry`                                                                               |
| `croffle_sync_pull_changes_total`       | —                                                                                                                       |
| `croffle_sync_bootstrap_rows_total`     | —                                                                                                                       |
| `croffle_auth_attempts_total`           | `surface`: `sync`, `token`, `dashboard`; `result`: `ok`, `invalid`, `otp_required`, `disabled`, `rate_limited`, `error` |
| `croffle_auth_refresh_reuse_total`      | —                                                                                                                       |

여기에 Go 런타임과 프로세스 기본 지표가 붙습니다. `route` 는 라우트 패턴
(`POST /sync/push`, `POST /dash/devices/{id}/revoke`)이고, 대시보드 파일과 모르는
경로는 `other` 입니다. 계정 · 사용자명 · 기기 id · 주소는 어떤 라벨에도 들어가지
않습니다.

```yaml
# docker compose: 호스트에만 공개
environment:
  CROFFLE_METRICS_LISTEN: ':9090'
ports:
  - '127.0.0.1:9090:9090'
```

---

## 호환성

| 서버  | 동기화 프로토콜 | 데스크톱 |
| ----- | --------------- | -------- |
| 0.1.0 | 1               | 1.3.x    |
| 0.1.1 | 1               | 1.3.x    |
| 0.2.0 | 1               | 1.3.x    |

0.1.1 과 0.2.0 은 프로토콜 1 을 바꾸지 않고 그 위에 기능을 더하므로, 데스크톱 1.3.0 은
그대로 동작합니다. 일부 기능(토큰 쌍, `differentFields`)은 데스크톱이 아직 쓰지
않습니다. 일회용 코드는 1.3.0 이 이미 `otp_required` 에 답하므로, 팀 합류는 이미
`cursor_too_old` 를 스냅샷으로 처리하므로 그대로 동작합니다.

`GET /health`가 이 서버가 말하는 범위와 선택 기능을 알려줍니다.

```json
{
  "server": "0.2.0",
  "protocol": { "min": 1, "max": 1 },
  "features": ["conflict-fields", "device-token-ttl", "team-scopes", "token-pair", "totp"],
  "db": "ok",
  "retention": { "oldestCommitSeq": 1041, "purgedThroughSeq": 1040 },
  "time": 1792000000000
}
```

데스크톱이 연결할 때 이를 확인하고 어느 쪽을 올려야 하는지 알려줍니다.
`retention`은 재부트스트랩이 왜 일어났는지를 설명해 주고(위의 [보존 정책](#보존-정책) 참고), `time`은
밀리초 단위 서버 시계입니다. 데스크톱이 자기 시계와 비교해 시간이 많이 어긋난
기기를 경고합니다. 충돌은 벽시계로 판정하므로, 틀린 시계는 항상 이기거나 항상
집니다.

`/health`는 무인증 공개이고 데이터베이스에 닿지 못하면 503을 반환합니다. 컨테이너
헬스체크와 프록시가 둘 다 이걸 봅니다.

push 충돌(`conflicts[]`)에는 보낸 행과 서버 행에서 값이 다른 필드 이름이
`differentFields` 로 붙습니다(이름순, `version` · `updatedAtMs` · `lastWriterId` ·
`ownerId` 제외, 데이터가 같으면 `[]`). 정보일 뿐입니다. 충돌 때 서버는 여전히 아무것도
적용하지 않고, 승자는 여전히 클라이언트가 last-writer-wins 로 정합니다.

---

## 보안 메모

- **토큰은 기기당**입니다. 노트북 하나를 잃으면 그 하나만 회수하면 됩니다.
- 토큰은 해시만 저장하므로 데이터베이스 사본을 동작하는 자격증명으로 바꿀 수
  없습니다. 비밀번호도 같습니다(argon2id).
- **대시보드 세션과 동기화 토큰은 서로 다른 자격증명**이고, 상대의 자리에서는 둘 다
  거부됩니다. 대시보드 세션으로는 push할 수 없습니다.
- 대시보드는 일반 계정에게 자기 데이터만 보여주고, 그 계정 자신의 상태(기기, 일회용
  코드)만 바꿉니다. 운영 화면은 모든 계정에 작용하며 요청마다 확인합니다. 운영자에서
  해제된 계정은 다음 클릭부터 운영 화면을 쓰지 못합니다. 상태를 바꾸는 요청은 다른
  출처에서 오면 거부합니다.
- **운영자 세션은 권한이 큽니다.** 서버를 운영하는 사람만 운영자로 두고, 그들은 일회용
  코드를 켜 두십시오.
- 일회용 코드 secret 은 [비밀 키](#비밀-키)를 설정하면 암호화해 저장합니다(복구 코드는
  언제나 해시만). 키가 없으면 데이터베이스 파일을 가진 사람이 코드를 만들 수 있지만,
  비밀번호는 여전히 필요합니다.
- 명령은 설계상 인증을 우회합니다. 데이터베이스 파일에 바이너리를 실행할 수 있는
  사람은 이미 운영자입니다.

---

## 0.2 범위의 한계

- **공유 취소가 아직 없고**, 공유한 일정을 원격에서 관리할 수도 없습니다. 멤버가 직접
  다른 사람을 초대하거나 팀을 나갈 수 없습니다. 팀은 운영자가 관리합니다.
- 팀은 지울 수 없습니다.
- 태그와 리마인더는 일정과 함께 공유되지 않습니다.
- 데스크톱(1.3.x)은 기기 토큰만 씁니다. 아직 `POST /auth/token` 으로 토큰을 갱신하지
  않으므로, 만료되는 기기 토큰을 쓰면 다시 로그인해야 합니다.
- 팀에 들어가면 그 계정의 기기마다 전체 스냅샷을 한 번 받습니다.
- 팀에서 빠진 뒤 그 기기에 남은 사본은 아직 별도 일정이 아닙니다([팀](#팀) 참고).
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

### 의존성

서버는 대부분 Go 표준 라이브러리입니다. 의존성은 손으로 짜야 할 보안 · 프로토콜 코드를
대신할 때만, 검증된 출처의 것만 더합니다.

| 모듈                                  | 쓰는 곳                                  |
| ------------------------------------- | ---------------------------------------- |
| `modernc.org/sqlite`                  | cgo 없는 SQLite                          |
| `golang.org/x/crypto`                 | argon2id 비밀번호 해시                   |
| `golang.org/x/term`                   | 명령의 비밀번호 입력(화면 숨김)          |
| `golang.org/x/time/rate`              | 로그인 rate limit                        |
| `github.com/pquerna/otp`              | 일회용 코드(TOTP, RFC 6238)와 그 QR 코드 |
| `github.com/prometheus/client_golang` | 선택 기능인 metrics 엔드포인트           |

일회용 코드 secret 의 암호화는 표준 라이브러리(`crypto/aes`, `crypto/cipher`)로 합니다.
0.2.0 에서 더한 의존성은 없습니다.

여전히 바이너리 하나와 SQLite 파일 하나입니다. 다른 서비스를 요구하지 않습니다.

라이선스: 저장소와 동일한 MIT.
