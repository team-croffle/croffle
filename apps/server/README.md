# croffle-server

한국어: **[README.ko.md](./README.ko.md)**

Optional self-hosted sync backend for [Croffle](../../README.md).

**Croffle works fully offline without it.** The desktop app is a local calendar
first; this server exists so several devices of one account can exchange
changes, and so a schedule can be shared with a team. If you only use one
machine, you do not need this.

Status: **0.1 beta.** One Go binary, one SQLite file, a read-only dashboard.

---

## Quickstart

```yaml
# compose.yaml
services:
  croffle-server:
    # Pin a version; :latest moves with every stable release.
    image: ghcr.io/team-croffle/croffle-server:0.1.0
    restart: unless-stopped
    environment:
      CROFFLE_DB_PATH: /var/lib/croffle/server.db
      # Only when a reverse proxy is actually in front of it.
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

# An account, then a token for one device. The token is printed once.
docker compose exec croffle-server croffle-server user create alice --password-stdin
docker compose exec croffle-server croffle-server token issue alice --device-name "laptop"
```

In the desktop app: **Settings → Sync**, enter the server URL, then either sign
in with the username and password or paste the token. Internally both end up as
the same thing — a token — so the two screens are one mechanism.

Open the server URL in a browser for the dashboard. It is read-only: everything
that changes state is a command above.

---

## Configuration

Environment variables. Every setting except the webhook ones also exists as a
flag with the same name (`CROFFLE_DB_PATH` → `--db-path`), and the flag wins
when both are given. The webhook settings are environment-only on purpose: an
operator configures where events go, so there is no flag — and no endpoint —
that could redirect them.

| Variable                      | Default                      | Meaning                                                                                                                                  |
| ----------------------------- | ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| `CROFFLE_LISTEN`              | `:8787`                      | Address to listen on. Keep `:8787` in a container so the port mapping works.                                                             |
| `CROFFLE_DB_PATH`             | `/var/lib/croffle/server.db` | The SQLite file. Its directory must exist and be writable.                                                                               |
| `CROFFLE_LOG_LEVEL`           | `info`                       | `debug`, `info`, `warn`, `error`. Logs are JSON on stdout.                                                                               |
| `CROFFLE_TRUST_PROXY`         | `false`                      | Honour `X-Forwarded-For` and `X-Forwarded-Proto`. **Only turn this on behind a proxy** — otherwise any client can forge its own address. |
| `CROFFLE_BASE_PATH`           | `/`                          | Mount point, e.g. `/croffle`, when a proxy forwards a subpath without stripping it.                                                      |
| `CROFFLE_RETAIN_COMMITS_DAYS` | `90`                         | How long the change history is kept. Minimum 7.                                                                                          |
| `CROFFLE_RETAIN_OPS_DAYS`     | `30`                         | How long retry keys are kept. Minimum 7.                                                                                                 |
| `CROFFLE_WEBHOOK_URL`         | —                            | Optional: where to POST events. Requires a secret.                                                                                       |
| `CROFFLE_WEBHOOK_SECRET`      | —                            | HMAC key the receiver verifies with.                                                                                                     |
| `CROFFLE_WEBHOOK_EVENTS`      | `commits_applied,row_shared` | Comma-separated event list.                                                                                                              |

Invalid values fail at startup rather than being defaulted: a server listening
on the wrong port or writing to the wrong file is worse than one that refuses to
boot.

---

## Reverse proxy

**The server speaks plain HTTP and expects TLS to be terminated in front of
it.** Any proxy will do — Traefik, Caddy, nginx, an ingress, a tunnel. There is
nothing proxy-specific in the server beyond honouring `X-Forwarded-*` when you
say a proxy is there.

<details>
<summary>Traefik (labels)</summary>

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

Serving it under a path instead of a host? Either strip the prefix in Traefik
(`StripPrefix` middleware) or tell the server about it with
`CROFFLE_BASE_PATH=/croffle` — one of the two, not both.

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
  # TLS directives omitted

  location / {
    proxy_pass http://croffle-server:8787;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }
}
```

</details>

### Without TLS, or with your own certificate

The desktop refuses an untrusted certificate by default. On a LAN you can use a
plain `http://` URL, and for a private certificate authority there is a switch
in **Settings → Sync → Advanced**. It applies to that one server, and it is off
until you turn it on.

---

## Commands

Every command takes `--db` to point at the database file, and the listings take
`--json`.

| Command                                                     | What it does                                                                                                 |
| ----------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| `user create <username> --password-stdin`                   | Creates a login and adds it to the team.                                                                     |
| `user list`                                                 | Lists logins and whether they are disabled.                                                                  |
| `user disable <username>` / `user enable <username>`        | Blocks or restores authentication.                                                                           |
| `user set-password <username> --password-stdin`             | Changes a password. Existing device tokens keep working.                                                     |
| `token issue <username> [--device-name N] [--client-id ID]` | Issues a device token. **Printed once.**                                                                     |
| `token list [username]`                                     | Lists devices: created, last seen, and `pending` / `active` / `revoked`.                                     |
| `token revoke <device-id>`                                  | Withdraws that device's token.                                                                               |
| `team list` · `team add-member <team-id> <username>`        | Teams and membership.                                                                                        |
| `db migrate` · `db info` · `db vacuum`                      | Apply migrations, show counts, reclaim space.                                                                |
| `log dump [--since-seq N] [--limit N]`                      | Prints the change log as JSON lines. **The first thing to reach for when a client and the server disagree.** |

Prefer `--password-stdin` over `--password`: a password in a flag ends up in the
shell history and in the process list.

### Revoking versus disabling

They are not the same, and the difference matters:

- **`token revoke`** withdraws one device's token. The account can sign in again
  and get a new one. Use it for a lost laptop.
- **`user disable`** stops the account. Every existing token stops working
  immediately and no new one can be obtained.

---

## Backup and restore

The whole server is one SQLite file, and `sqlite3 .backup` copies it
consistently while the server keeps running. The image itself ships no
`sqlite3`, so run it in a throwaway container sharing the same volume and write
the result next to your `compose.yaml`:

```bash
docker run --rm --volumes-from "$(docker compose ps -q croffle-server)" \
  -v "$PWD:/backup" alpine:3.21 \
  sh -c 'apk add -q --no-cache sqlite &&
         sqlite3 /var/lib/croffle/server.db ".backup /backup/croffle-$(date +%F).db"'
```

**Do not just copy `server.db`.** The database runs in WAL mode, so recent
commits live in the `server.db-wal` file beside it: that copy can come out
nearly empty. If you would rather copy files than use `.backup`, stop the server
first, and take `server.db`, `server.db-wal` and `server.db-shm` together.

To restore, stop the server, put the file back as `CROFFLE_DB_PATH` with no
stale WAL next to it, and start again. The server in the image runs as uid 100,
which is why the copy is handed over:

```bash
docker compose stop
docker run --rm --volumes-from "$(docker compose ps -aq croffle-server)" \
  -v "$PWD:/backup" alpine:3.21 \
  sh -c 'rm -f /var/lib/croffle/server.db /var/lib/croffle/server.db-wal /var/lib/croffle/server.db-shm &&
         cp /backup/croffle-2026-10-20.db /var/lib/croffle/server.db &&
         chown 100:101 /var/lib/croffle/server.db'
docker compose up -d
```

Device tokens survive a restore, so nothing has to be re-issued.

> Restoring an **older** backup moves the change sequence backwards. A device
> whose cursor is ahead of it is answered `410 cursor_too_old` and bootstraps
> again, which the desktop handles on its own — no data is lost on the client,
> but each device re-downloads its state once.

---

## Retention

History is not kept forever:

- the **change log** is kept for `CROFFLE_RETAIN_COMMITS_DAYS` (90 by default)
- **retry keys** are kept for `CROFFLE_RETAIN_OPS_DAYS` (30 by default)

The current state of every row is never deleted — only history. A device that
was away longer than the window is told its cursor is too old and fetches a
fresh snapshot instead; nothing is lost. `GET /health` and `db info` report the
retained range, which is what explains that to you when it happens.

Neither window may be set below 7 days. Shorter windows break the guarantees
they support — a cursor expiring while a laptop is on holiday, a late retry
being applied twice — and the symptom would show up far from the setting.

Space freed by the sweep is reclaimed by `db vacuum`, which is deliberately
manual: it locks the database for the length of a full copy, so you pick the
moment.

---

## Webhooks

The server has **no plugin system**. Extensibility here means it can tell
something else what happened:

```yaml
environment:
  CROFFLE_WEBHOOK_URL: https://hooks.example.com/croffle
  CROFFLE_WEBHOOK_SECRET: a-long-random-string
  CROFFLE_WEBHOOK_EVENTS: commits_applied,row_shared
```

| Event               | When                                           |
| ------------------- | ---------------------------------------------- |
| `commits_applied`   | A push was applied. Says which rows moved.     |
| `row_shared`        | A private schedule gained a team.              |
| `device_registered` | A login created or re-armed a device.          |
| `auth_succeeded`    | A credential was accepted (sync or dashboard). |
| `retention_swept`   | A sweep removed history.                       |

An event says **which** row moved, never what it contains: no payload, no
token, no password. Delivery is best-effort and off the request path — a slow
receiver cannot slow down a sync, and events are dropped rather than queued
without bound.

Each request carries `X-Croffle-Event`, `X-Croffle-Delivery` and
`X-Croffle-Signature: sha256=<hex>`, an HMAC of the body with your secret —
the same shape GitHub uses, so an existing verification snippet works:

```js
import { createHmac, timingSafeEqual } from 'node:crypto';

function verify(secret, body, signature) {
  const expected = 'sha256=' + createHmac('sha256', secret).update(body).digest('hex');
  return timingSafeEqual(Buffer.from(expected), Buffer.from(signature));
}
```

A URL without a secret refuses to start: a receiver that cannot verify the
sender has to trust anyone who learns the URL.

---

## Compatibility

| Server | Sync protocol | Desktop |
| ------ | ------------- | ------- |
| 0.1.x  | 1             | 1.3.x   |

`GET /health` reports the range this server speaks:

```json
{
  "server": "0.1.0",
  "protocol": { "min": 1, "max": 1 },
  "db": "ok",
  "retention": { "oldestCommitSeq": 1041, "purgedThroughSeq": 1040 },
  "time": 1792000000000
}
```

The desktop checks it when you connect and tells you which side needs updating.
`retention` is what explains a re-bootstrap (see below), and `time` is the
server clock in milliseconds, which the desktop compares with its own to warn
about a badly skewed machine — conflicts are resolved by wall clock, so a wrong
one always wins or always loses.

`/health` is public and answers 503 when the database is unreachable, which is
what a container health check and a proxy both look at.

---

## Security notes

- **Tokens are per device.** Losing one laptop means revoking one laptop.
- Only a hash of a token is stored, so a copy of the database cannot be turned
  into a working credential. The same is true of passwords (argon2id).
- **The dashboard session is a different credential from a sync token**, and
  neither is accepted where the other belongs. A dashboard session cannot push.
- The dashboard shows the signed-in account's own data. Looking at another
  account means using the commands above, which already require access to the
  file.
- The commands bypass authentication by design: whoever can run the binary
  against the database file is the operator.

---

## 0.1 limits

- **One team.** Multiple teams come later.
- **Sharing cannot be undone** yet, and shared schedules cannot be managed
  remotely.
- Tags and reminders are not shared along with a schedule.
- Tokens do not expire, because the desktop does not store your password and so
  could not renew one silently. Refresh tokens come with a later version.
- The dashboard is read-only, and there are no metrics yet.
- Disabling an account blocks it; it does not erase its data.

---

## Development

```bash
cd apps/server
make check      # gofmt, vet, go test (includes the shared protocol scenarios)
make run        # serves on :8787 with ./server.db
make build      # binary without the dashboard
make build-all  # dashboard + binary
```

The dashboard is a Vue app in [`web/`](./web) with its own lockfile, built into
the binary with `go:embed`. `make build` skips it, so a Go change never needs a
Node toolchain; the binary then says so on `/`.

The protocol scenarios in [`packages/sync-fixtures`](../../packages/sync-fixtures)
are read by this server's tests **and** by the desktop, so a wire change that
lands on only one side fails in CI.

Licence: MIT, the same as the rest of the repository.
