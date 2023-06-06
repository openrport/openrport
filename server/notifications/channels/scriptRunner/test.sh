#!/usr/bin/env sh

cat /dev/stdin | go run test_app/test_app.go "$@"