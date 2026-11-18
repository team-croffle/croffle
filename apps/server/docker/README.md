# croffle-server with Docker Compose

한국어: [README.ko.md](./README.ko.md)

A first deployment, with the secret key from the start. Run these in this folder.

```bash
# 1. The secret key, readable by you alone. Keep a copy apart from the data
#    volume: without it, accounts using one-time codes cannot sign in.
umask 077; set -C   # never overwrites an existing file (file exists)
key=$(docker run --rm ghcr.io/team-croffle/croffle-server:0.2.0 secret generate) && [ -n "$key" ] &&
  echo "CROFFLE_SECRET_KEY=$key" > croffle-secret.env

# 2. Start. The log says "secret key: encrypted 0 one-time code secrets".
docker compose up -d
docker compose logs croffle-server | grep 'secret key'

# 3. The first account, as the operator (the password is typed, hidden).
docker compose exec croffle-server croffle-server user create <you> --password-stdin --operator
```

If `croffle-secret.env` is already there, step 1 stops with `file exists`: keep
that file, it holds the key your data is sealed with. A `.env` file in this
folder is not read for the key and does not change this example.

Then open `http://<host>:8787` and sign in: the operator tabs (Accounts, All
devices, Manage teams, Server) do the rest. In the desktop app, **Settings →
Sync**, enter the same URL and sign in.

Why an env file and not a key file: the image runs as an unprivileged user, and
a key file mounted from the host is readable inside only if it is readable by
everyone on the host. Compose reads the env file itself, so the file stays
private. Configuration, upgrades and backups: [the server README](../README.md).
