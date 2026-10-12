#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
rm -f go.work go.work.sum
go work init . ./app/openapi ./bus/nats ./enjintest ./examples ./filestore/minio ./google ./mail/ses
