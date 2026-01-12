#!/bin/sh

apk add ca-certificates

set -eo pipefail

openssl s_client -connect "$server_address" -showcerts </dev/null | openssl x509 -outform pem > /tmp/out.pem

/usr/local/bin/cert-chain-resolver -s -o /oplet/outputs/bundled.pem /tmp/out.pem