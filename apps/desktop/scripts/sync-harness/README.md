# Sync verification harness

Tools for checking the desktop sync client by hand (spec 001, checks V-8 to V-15 and V-17).
Nothing here ships with the app.

| File              | What                                                                                                   |
| ----------------- | ------------------------------------------------------------------------------------------------------ |
| `stub-server.mjs` | In-memory stand-in for croffle-server 0.1 that can be told to misbehave. No dependencies.              |
| `make-cert.sh`    | Self-signed certificates for `localhost` and `127.0.0.1` in a temporary directory (never in the repo). |
| `rc-server.sh`    | Starts the real target, `ghcr.io/team-croffle/croffle-server:0.1.0-rc.1`, and creates an account.      |

Run the desktop with `pnpm dev`; its database is `apps/desktop/dev/croffle.db` and its log `apps/desktop/dev/logs/main.log`. Sync requests appear there as `Sync: <METHOD> <path> <status>` lines.

## Stub server

```sh
node scripts/sync-harness/stub-server.mjs        # http://localhost:8787, user alice / secret
```

The stub prints one JSON line per request: the route, the status, the `X-Croffle-Sync-Version` value, and whether `X-Croffle-Client` and `Authorization` were present. It never prints header values.

| Variable                                   | Effect                                                          | Check      |
| ------------------------------------------ | --------------------------------------------------------------- | ---------- |
| `PORT`                                     | listen port (8787)                                              |            |
| `PAGE`                                     | page size for pull and bootstrap (2), so paging is exercised    | V-12, V-13 |
| `PROTOCOL_MIN`, `PROTOCOL_MAX`             | protocol range in `/health` (1–1)                               | V-8        |
| `CLOCK_SKEW_MS`                            | added to the `/health` clock                                    | V-8        |
| `FAIL_503_TIMES`                           | the next N sync requests answer 503                             | V-8        |
| `FORCE_OTP=1`, `OTP_CODE`                  | login answers `otp_required` until the code (123456) is sent    | V-10       |
| `CHILD_FIRST=1`                            | child rows come before their parents in each page               | V-12       |
| `INVALID_ROW_ID`                           | a push containing this row id is rejected with `invalid_commit` | V-11       |
| `FAIL_BOOTSTRAP_AFTER`                     | serve N bootstrap pages, then answer 503 (interrupted snapshot) | V-13       |
| `STUB_USER`, `STUB_PASSWORD`, `STUB_TOKEN` | credentials; `STUB_TOKEN` is accepted as a pasted token         | V-10       |
| `TLS_CERT`, `TLS_KEY`                      | serve https with these files                                    | V-9        |

Test controls (not part of the protocol):

```sh
# a row written by another device (wire shape)
curl -XPOST localhost:8787/__stub/write -d '{"table":"tag","row":{"id":"t1","name":"x","color":"#f00","createdAt":1700000000000,"updatedAt":1700000000000}}'
curl -XPOST localhost:8787/__stub/purge -d '{"seq":5}'      # cursors before 5 now get 410
curl -XPOST localhost:8787/__stub/fail503 -d '{"times":2}'  # the next 2 sync requests answer 503
curl -XPOST localhost:8787/__stub/dropPush -d '{"times":4}' # the next 4 pushes apply, then drop the answer
curl -XPOST localhost:8787/__stub/delayPush -d '{"times":1,"ms":8000}' # the next push applies, answers 8 s later (V-11: edit the row meanwhile)
curl localhost:8787/__stub/state                            # everything the stub holds
```

## TLS (V-9)

```sh
dir=$(scripts/sync-harness/make-cert.sh)
TLS_CERT=$dir/localhost.crt TLS_KEY=$dir/localhost.key PORT=8443 node scripts/sync-harness/stub-server.mjs &
TLS_CERT=$dir/127.0.0.1.crt TLS_KEY=$dir/127.0.0.1.key PORT=8444 node scripts/sync-harness/stub-server.mjs &
```

Register `https://localhost:8443`. With the setting off it must be refused; with it on, `localhost:8443` connects while `https://127.0.0.1:8444` and an extension's `croffle.http.get('https://localhost:8443/health')` are still refused. Delete `$dir` afterwards.

## Real server (V-17)

```sh
scripts/sync-harness/rc-server.sh alice        # reads the password from stdin
docker exec croffle-rc croffle-server token issue alice --device-name device-b
docker rm -f croffle-rc                        # when done
```

A second device is a second profile: copy `apps/desktop/dev/` aside, or run a packaged build with its own user data directory.
