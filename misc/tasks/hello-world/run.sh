#!/bin/sh

set -eo pipefail

echo "==== Environment ===="

env

echo "==== Inputs ===="

ls -lah /oplet/inputs

REMAINING=30

while [ $REMAINING -gt 0 ]; do
  echo "Waiting... (${REMAINING}s remaining)"
  REMAINING=$(($REMAINING - 1))
  sleep 1
done

echo "==== Outputs ===="

echo "hello world" | tee /oplet/outputs/hello-world.txt