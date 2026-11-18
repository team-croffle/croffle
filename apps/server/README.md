# croffle-server

한국어: **[README.ko.md](./README.ko.md)**

Optional self-hosted sync backend for [Croffle](../../README.md).

**Croffle works fully offline without it.** The desktop app is a local calendar
first; this server exists so several devices of one account can exchange
changes, and so a schedule can be shared with a team. If you only use one
machine, you do not need this.

Status: **0.2 beta.** One Go binary, one SQLite file, a dashboard for your own account and, for operators, for the whole server.

---

## Quickstart

```yaml
# compose.yaml
services:
  croffle-server:
    # Pin a version; :latest moves with every stable release.
    image: ghcr.io/team-croffle/croffle-server:0.2.0
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

# A first account that is also the operator, then a token for one device.
# The token is printed once.
docker compose exec croffle-server croffle-server user create alice --password-stdin --operator
docker compose exec croffle-server croffle-server token issue alice --device-name "laptop"
```

Also set a [secret key](#secret-key) before anyone turns on one-time codes; the
[Compose example](./docker) starts with one.

In the desktop app: **Settings → Sync**, enter the server URL, then either sign
in with the username and password or paste the token. Internally both end up as
the same thing — a token — so the two screens are one mechanism.

Open the server URL in a browser for the dashboard. It shows the signed-in
account's own data, and lets that account revoke its own devices, turn on
one-time codes ([Authentication](#authentication)) and change its password
under **Security** (every other browser is signed out; devices keep syncing). An [operator](#operators)
also gets the operator screens: accounts, every device and token, teams, and the
server itself — everything the commands below do.

---

## Configuration

Environment variables. The first seven settings also exist as a flag with the
same name (`CROFFLE_DB_PATH` → `--db-path`), and the flag wins when both are
given. The rest are environment-only. For the webhook that is on purpose: an
operator configures where events go, so there is no flag — and no endpoint —
that could redirect them.

| Variable                       | Default                      | Meaning                                                                                                                                                                                     |
| ------------------------------ | ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `CROFFLE_LISTEN`               | `:8787`                      | Address to listen on. Keep `:8787` in a container so the port mapping works.                                                                                                                |
| `CROFFLE_DB_PATH`              | `/var/lib/croffle/server.db` | The SQLite file. Its directory must exist and be writable.                                                                                                                                  |
| `CROFFLE_LOG_LEVEL`            | `info`                       | `debug`, `info`, `warn`, `error`. Logs are JSON on stdout.                                                                                                                                  |
| `CROFFLE_TRUST_PROXY`          | `false`                      | Honour `X-Forwarded-For` and `X-Forwarded-Proto`. **Only turn this on behind a proxy** — otherwise any client can forge its own address.                                                    |
| `CROFFLE_BASE_PATH`            | `/`                          | Mount point, e.g. `/croffle`, when a proxy forwards a subpath without stripping it.                                                                                                         |
| `CROFFLE_RETAIN_COMMITS_DAYS`  | `90`                         | How long the change history is kept. Minimum 7.                                                                                                                                             |
| `CROFFLE_RETAIN_OPS_DAYS`      | `30`                         | How long retry keys are kept. Minimum 7.                                                                                                                                                    |
| `CROFFLE_WEBHOOK_URL`          | —                            | Optional: where to POST events. Requires a secret.                                                                                                                                          |
| `CROFFLE_WEBHOOK_SECRET`       | —                            | HMAC key the receiver verifies with.                                                                                                                                                        |
| `CROFFLE_WEBHOOK_EVENTS`       | `commits_applied,row_shared` | Comma-separated event list.                                                                                                                                                                 |
| `CROFFLE_DEVICE_TOKEN_TTL`     | `0` (never)                  | Lifetime of device tokens issued by a sign-in from now on, as a Go duration (`720h`). **Desktop 1.3.0 has to sign in again each time one expires** — see [Authentication](#authentication). |
| `CROFFLE_ACCESS_TOKEN_TTL`     | `15m`                        | Lifetime of access tokens from `POST /auth/token`.                                                                                                                                          |
| `CROFFLE_REFRESH_TOKEN_TTL`    | `720h`                       | Lifetime of refresh tokens; it restarts at every refresh. Must be longer than the access lifetime.                                                                                          |
| `CROFFLE_METRICS_LISTEN`       | — (off)                      | `host:port` for Prometheus metrics on a separate listener, e.g. `127.0.0.1:9090`. Empty opens no port. See [Metrics](#metrics).                                                             |
| `CROFFLE_DASH_TRUSTED_ORIGINS` | —                            | Comma-separated extra origins (`https://croffle.example.com`) allowed to send dashboard requests that change state, for a proxy that rewrites `Host`.                                       |
| `CROFFLE_SECRET_KEY`           | — (none)                     | Base64 of 32 bytes that encrypts one-time code secrets. Make one with `secret generate`. See [Secret key](#secret-key).                                                                     |
| `CROFFLE_SECRET_KEY_FILE`      | —                            | A file holding that key instead (a Docker secret, for instance). Set one of the two, not both.                                                                                              |

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

| Command                                                                                     | What it does                                                                                                 |
| ------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| `user create <username> --password-stdin [--operator]`                                      | Creates a login and adds it to `team-default`. `--operator` makes it an [operator](#operators).              |
| `user list`                                                                                 | Lists logins, whether they are disabled, and whether they are operators.                                     |
| `user disable <username> [--force]` / `user enable <username>`                              | Blocks or restores authentication. The last active operator needs `--force`.                                 |
| `user operator grant <username>` · `user operator revoke <username> [--force]`              | Makes an account an operator, or a regular account again.                                                    |
| `user set-password <username> --password-stdin`                                             | Changes a password and signs the account's browsers out. Existing device tokens keep working.                |
| `token issue <username> [--device-name N] [--client-id ID] [--expires-in D \| --no-expiry]` | Issues a device token. **Printed once.** Follows `CROFFLE_DEVICE_TOKEN_TTL` unless a flag says otherwise.    |
| `token list [username]`                                                                     | Lists devices: created, last seen, token expiry (`never`), and `pending` / `active` / `expired` / `revoked`. |
| `user otp status <username>` · `user otp disable <username>`                                | Shows (`off` / `pending` / `enabled`) or removes an account's one-time codes.                                |
| `token revoke <device-id>`                                                                  | Withdraws that device's token.                                                                               |
| `team list` · `team create <name> [--id ID]` · `team rename <team-id> <name>`               | Teams. [Teams](#teams) explains membership.                                                                  |
| `team members <team-id>` · `team add-member <team-id> <username>` · `team remove-member …`  | Who is in a team.                                                                                            |
| `secret generate` · `secret rotate --new-key-file <path>`                                   | Make a [secret key](#secret-key), or re-encrypt the one-time code secrets with a new one.                    |
| `db migrate` · `db info` · `db vacuum`                                                      | Apply migrations, show counts, reclaim space.                                                                |
| `log dump [--since-seq N] [--limit N]`                                                      | Prints the change log as JSON lines. **The first thing to reach for when a client and the server disagree.** |

Prefer `--password-stdin` over `--password`: a password in a flag ends up in the
shell history and in the process list.

`--password-stdin` reads differently depending on what stdin is:

- **A terminal** — it prints `Password: `, reads one line without echoing it, and
  stops at Enter. `docker compose exec` gives the command a terminal by default;
  with plain `docker exec`, pass `-it`:

  ```bash
  docker exec -it croffle-server croffle-server user create alice --password-stdin
  ```

  In Windows Git Bash (MinTTY), wrap it in `winpty`, or the container does not
  get a terminal:

  ```bash
  winpty docker exec -it croffle-server croffle-server user create alice --password-stdin
  ```

- **A pipe or a file** — it reads to the end and drops the trailing newline, for
  scripts. Use `-i` without `-t`:

  ```bash
  printf '%s' "$PW" | docker exec -i croffle-server croffle-server user create alice --password-stdin
  ```

### Revoking versus disabling

They are not the same, and the difference matters:

- **`token revoke`** withdraws one device's token. The account can sign in again
  and get a new one. Use it for a lost laptop.
- **`user disable`** stops the account. Every existing token stops working
  immediately and no new one can be obtained.

Revoking a device withdraws every token it holds: its device token and any
access and refresh tokens. An account can also revoke its own devices from the
dashboard.

---

## Authentication

Every sync request carries a bearer token. There are three kinds:

| Token         | From                              | Lifetime                                 | Used by                 |
| ------------- | --------------------------------- | ---------------------------------------- | ----------------------- |
| Device token  | `POST /auth/login`, `token issue` | never, unless `CROFFLE_DEVICE_TOKEN_TTL` | desktop 1.3.0           |
| Access token  | `POST /auth/token`                | `CROFFLE_ACCESS_TOKEN_TTL` (15 minutes)  | sync requests           |
| Refresh token | `POST /auth/token`                | `CROFFLE_REFRESH_TOKEN_TTL` (30 days)    | `POST /auth/token` only |

**Device tokens.** Never expiring is the default, and stays possible: the
desktop does not keep your password, so it cannot renew a token silently.
Setting `CROFFLE_DEVICE_TOKEN_TTL` gives tokens issued by a sign-in from then on
an expiry (tokens issued earlier are not touched). Desktop 1.3.0 then asks you to
sign in again whenever one expires, because it does not use refresh tokens yet.
`token issue --no-expiry` still issues a token that never expires, whatever the
setting says.

**Access and refresh tokens.** `POST /auth/token` with
`{"grantType": "password", "username", "password", "otp"?, "clientId", "deviceName"?}`
returns `accessToken`, `refreshToken` and their expiries (unix ms). Exchange the
refresh token for the next pair with `{"grantType": "refresh_token", "refreshToken"}`
before the access token runs out. Each refresh token works **once**: presenting a
spent one again revokes every access and refresh token of that device, because
only a stolen copy is ever replayed. The device token is left alone. A refresh token is never accepted by a sync
endpoint. Signing in this way leaves the device's device token alone, so a 1.3.0
sign-in on the same machine keeps working. The desktop does not use this
endpoint yet.

**One-time codes (TOTP).** An account turns them on itself, in the dashboard
under **Security**: scan the QR code with an authenticator app, confirm with a
code, and keep the ten recovery codes it shows once. From then on every sign-in —
desktop, `POST /auth/token` and the dashboard — needs a six-digit code or a
recovery code. Without one the answer is `401 otp_required`, which desktop 1.3.0
already handles by asking for the code. A code is accepted once (a code for a
time step already used is refused), recovery codes work once each, and wrong
codes count towards the login rate limit like wrong passwords. To turn codes off,
the account confirms with its password and a code; an operator can remove them
with `user otp disable` (or the **Accounts** screen) for someone who has lost
both. `token issue` is an operator command and does not ask for a code. The
secrets behind the codes are encrypted when a [secret key](#secret-key) is set.

The login rate limit is five attempts per address and username, then one per 30
seconds. A sign-in that stops at `otp_required` counts as one attempt.

---

## Operators

An operator is an account that may use the dashboard's operator screens:

- **Accounts** — create, disable and enable, reset a password (this signs the
  account out of every browser; its device tokens keep working), turn off
  one-time codes, make or remove operators.
- **All devices** — every account's devices; issue a token (shown once, with a
  copy button) and revoke one.
- **Manage teams** — create and rename teams, add and remove members.
- **Server** — database facts and the secret key state, vacuum, and the commit
  log of every account (metadata only; `log dump` has the row contents).

Name the first operator with the command line — `user create <name> --operator`
or `user operator grant <name>` — and the rest from the dashboard. Being an
operator changes nothing about sync: an operator's devices see the same rows as
anyone's. Every operator action leaves one `operator action` line in the log
(who, what, on what; never a password or a token).

The last active operator cannot be removed or disabled from the dashboard
(`409 last_operator`), so the server always keeps someone who can use the
operator screens. The command line can do it with `--force`, because whoever
runs it can also undo it.

---

## Secret key

One-time code secrets have to be readable by the server to check a code, so
they cannot be stored as hashes. With a secret key they are stored encrypted
(AES-256-GCM, bound to the account); without one they are stored as they are,
and the server says so at start-up and on the operator screens.

```bash
# Make a key in a file only you can read, outside the database volume.
# set -C refuses to overwrite an existing file: a new key in place of the old
# one locks out every secret sealed with it.
umask 077; set -C
key=$(docker run --rm ghcr.io/team-croffle/croffle-server:0.2.0 secret generate) && [ -n "$key" ] &&
  echo "CROFFLE_SECRET_KEY=$key" > croffle-secret.env
```

Give it to the service with `env_file: [croffle-secret.env]`, as the
[Compose example](./docker) does. Compose reads that file on the host, so it can
stay readable by you alone. At the next start the server encrypts every secret
that is still plain; nobody has to set up codes again.

`CROFFLE_SECRET_KEY_FILE` works too, but the image runs as an unprivileged user
(uid 100), so the mounted file has to be readable by that user: a key file only
you can read (`600`) stops the server with `permission denied`.

- **Back the key up separately from the database.** A copy of the database
  alone is then not enough to generate codes, and a lost key does not lose the
  database.
- **If the key is lost**, the server refuses to start while encrypted secrets
  exist. Turn off one-time codes for those accounts with `user otp disable`
  (it needs no key), start without the key or with a new one, and let them set
  codes up again.
- **To change the key**: stop the server, run `secret rotate` with the old key
  still configured, switch the configuration to the new key, and start the
  server. With compose (`croffle-secret.env` still holds the old key; the new
  one goes in on standard input). Each step runs only if the one before it
  succeeded, so a failed rotate leaves the old key configured:

  ```bash
  umask 077; set -C
  cp croffle-secret.env croffle-secret.env.old &&   # the old key, until the server is up
  new=$(docker run --rm ghcr.io/team-croffle/croffle-server:0.2.0 secret generate) && [ -n "$new" ] &&
    docker compose stop croffle-server &&
    printf '%s\n' "$new" | docker compose run --rm -T croffle-server secret rotate --new-key-stdin &&
    printf 'CROFFLE_SECRET_KEY=%s\n' "$new" >| croffle-secret.env &&   # >| replaces it on purpose
    docker compose up -d
  ```

  A rotate while the server runs is not safe: the running server keeps the old
  key until it restarts, so a code set up in between is sealed with the old key
  and the next start fails. To recover, keep the new key configured and run
  `secret rotate --new-key-file <new> --old-key-file <old>`: it opens secrets
  sealed with either key and seals them all with the new one. With compose, the
  old key in `croffle-secret.env.old` and the new one configured, pass both on
  standard input:

  ```bash
  { sed -n 's/^CROFFLE_SECRET_KEY=//p' croffle-secret.env.old; sed -n 's/^CROFFLE_SECRET_KEY=//p' croffle-secret.env; } |
    docker compose run --rm -T --entrypoint sh croffle-server -c 'umask 077; read -r old; read -r new
      printf "%s\n" "$old" > /tmp/old.key
      printf "%s\n" "$new" | croffle-server secret rotate --new-key-stdin --old-key-file /tmp/old.key'
  ```

- A server started with a key that does not match the stored secrets refuses
  to start and names the key the secrets were sealed with.

---

## Teams

Every account is in `team-default`. An operator can make more teams and move
accounts in and out (`team create`, `team add-member`, `team remove-member`, or
**Manage teams**). A schedule shared with a team is visible to its members.

- **Joining a team.** The account's devices have to receive the team's earlier
  rows too. Each device takes one fresh snapshot at its next sync: the server
  answers its pull with `410 cursor_too_old` (`details.reason: "team_scope"`),
  which desktop 1.3.x already turns into a snapshot. Upgrading to 0.2.0 sends no
  device to a snapshot.
- **Leaving a team cuts the link; nothing is deleted.** The team keeps every
  row, including the ones the leaver created, and the leaver's devices keep the
  copies they have. From then on the server sends none of the team's rows to
  the account — its own shared rows included — and the account's changes to
  them do not reach the team (`403 foreign_team`; the desktop marks that change
  as failed and keeps syncing). Until the desktop can turn such a copy into a
  separate schedule, an edit to it stays on that device.
- Teams cannot be deleted yet; rename one instead.
- A device that was in a team under 0.1.x without ever taking a snapshot after
  joining may miss that team's older rows. On that desktop, choose **Settings →
  Sync → Sign out** and sign in again: signing out clears the device's position
  in the change log, so the next sync takes a full snapshot. (Revoking the
  device is not enough: signing in again as the same account keeps the old
  position.)

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

## Metrics

Off by default. Set `CROFFLE_METRICS_LISTEN` (for example `127.0.0.1:9090`) and
the server opens a **second listener** that serves `GET /metrics` in the
Prometheus text format, and nothing else. The API listener never serves metrics.
**Keep that address on a private network**: it has no authentication of its own.

| Metric                                  | Labels                                                                                                                  |
| --------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| `croffle_http_requests_total`           | `route`, `method`, `code`                                                                                               |
| `croffle_http_request_duration_seconds` | `route`, `method`                                                                                                       |
| `croffle_sync_push_commits_total`       | `result`: `accepted`, `conflict`, `retry`                                                                               |
| `croffle_sync_pull_changes_total`       | —                                                                                                                       |
| `croffle_sync_bootstrap_rows_total`     | —                                                                                                                       |
| `croffle_auth_attempts_total`           | `surface`: `sync`, `token`, `dashboard`; `result`: `ok`, `invalid`, `otp_required`, `disabled`, `rate_limited`, `error` |
| `croffle_auth_refresh_reuse_total`      | —                                                                                                                       |

plus the standard Go runtime and process metrics. `route` is the route pattern
(`POST /sync/push`, `POST /dash/devices/{id}/revoke`); the dashboard's files and
unknown paths are `other`. No label carries an account, a username, a device id
or an address.

```yaml
# docker compose: publish it to the host only
environment:
  CROFFLE_METRICS_LISTEN: ':9090'
ports:
  - '127.0.0.1:9090:9090'
```

---

## Compatibility

| Server | Sync protocol | Desktop |
| ------ | ------------- | ------- |
| 0.1.0  | 1             | 1.3.x   |
| 0.1.1  | 1             | 1.3.x   |
| 0.2.0  | 1             | 1.3.x   |

0.1.1 and 0.2.0 add features on top of protocol 1 without changing it, so
desktop 1.3.0 works with them unchanged. The desktop does not use some of them
yet (token pairs, `differentFields`); one-time codes work with 1.3.0 because it
already answers `otp_required`, and joining a team works because it already
answers `cursor_too_old` with a snapshot.

`GET /health` reports the range this server speaks, and the optional features:

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

The desktop checks it when you connect and tells you which side needs updating.
`retention` is what explains a re-bootstrap (see [Retention](#retention) above), and `time` is the
server clock in milliseconds, which the desktop compares with its own to warn
about a badly skewed machine — conflicts are resolved by wall clock, so a wrong
one always wins or always loses.

`/health` is public and answers 503 when the database is unreachable, which is
what a container health check and a proxy both look at.

A push conflict (`conflicts[]`) also names the fields that differ between what
was sent and the server's row, in `differentFields` (sorted; without `version`,
`updatedAtMs`, `lastWriterId` and `ownerId`; `[]` when the data is the same). It
is information only: the server still applies nothing on a conflict, and the
client still decides the winner by last-writer-wins.

---

## Security notes

- **Tokens are per device.** Losing one laptop means revoking one laptop.
- Only a hash of a token is stored, so a copy of the database cannot be turned
  into a working credential. The same is true of passwords (argon2id).
- **The dashboard session is a different credential from a sync token**, and
  neither is accepted where the other belongs. A dashboard session cannot push.
- The dashboard shows a regular account its own data and changes only that
  account's own state (its devices, its one-time codes). The operator screens
  act on every account and are checked on every request: an account that stops
  being an operator loses them at its next click. Requests that change state are
  refused from another origin.
- **An operator session is powerful.** Make only the people who run the server
  operators, and turn on one-time codes for them.
- One-time code secrets are encrypted with the [secret key](#secret-key) when
  one is set (recovery codes are only ever stored as hashes). Without a key,
  whoever has the database file could generate codes, though they would still
  need the password.
- The commands bypass authentication by design: whoever can run the binary
  against the database file is the operator.

---

## 0.2 limits

- **Sharing cannot be undone** yet, and shared schedules cannot be managed
  remotely. Members cannot invite others or leave a team themselves: an operator
  manages teams.
- Teams cannot be deleted.
- Tags and reminders are not shared along with a schedule.
- The desktop (1.3.x) uses device tokens only: it does not renew tokens with
  `POST /auth/token` yet, so an expiring device token means signing in again.
- Joining a team sends each of the account's devices to one full snapshot.
- After leaving a team, the copies on the leaver's devices are not yet separate
  schedules (see [Teams](#teams)).
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

### Dependencies

The server is mostly the Go standard library. A dependency is added only when it
replaces security-sensitive or protocol code we would otherwise write by hand,
and only from a well-established source:

| Module                                | Used for                                          |
| ------------------------------------- | ------------------------------------------------- |
| `modernc.org/sqlite`                  | SQLite without cgo                                |
| `golang.org/x/crypto`                 | argon2id password hashing                         |
| `golang.org/x/term`                   | the hidden password prompt of the commands        |
| `golang.org/x/time/rate`              | the login rate limit                              |
| `github.com/pquerna/otp`              | one-time codes (TOTP, RFC 6238) and their QR code |
| `github.com/prometheus/client_golang` | the optional metrics endpoint                     |

Encrypting the one-time code secrets uses the standard library
(`crypto/aes`, `crypto/cipher`); 0.2.0 adds no dependency.

Still one binary and one SQLite file: nothing here needs another service.

Licence: MIT, the same as the rest of the repository.
