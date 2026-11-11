#!/usr/bin/env bash
# Self-signed certificates for the TLS checks (V-9), written to a temporary directory and never
# into the repository. Prints the directory. Two hosts: localhost (the registered sync server)
# and 127.0.0.1 (another self-signed host that must stay refused).
set -euo pipefail
dir="$(mktemp -d -t croffle-sync-certs.XXXXXX)"
for name in localhost 127.0.0.1; do
  openssl req -x509 -newkey rsa:2048 -nodes -days 2 -subj "/CN=${name}" \
    -addext "subjectAltName=$([[ $name == localhost ]] && echo DNS || echo IP):${name}" \
    -keyout "${dir}/${name}.key" -out "${dir}/${name}.crt" 2>/dev/null
done
echo "${dir}"
