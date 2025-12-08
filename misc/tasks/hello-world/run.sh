#!/bin/sh

set -eo pipefail

echo "==== Environment ===="

env

echo "==== Inputs ===="

ls -lah /oplet/inputs

echo "==== Volumes ===="

ls -lah /cache

echo "$(date)" > "/cache/${OPLET_RUN_ID}.txt"

REMAINING=5

while [ $REMAINING -gt 0 ]; do
  echo "Waiting... (${REMAINING}s remaining)"
  REMAINING=$(($REMAINING - 1))
  sleep 1
done

echo "==== Outputs ===="

echo "hello world" | tee /oplet/outputs/hello-world.txt

if [ "${must_fail}" == "true" ]; then
  exit 1
fi