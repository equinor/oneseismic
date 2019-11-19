#!/usr/bin/env bash
go test ./... -coverprofile cover.out > /dev/null
COVERAGE=`go tool cover -func cover.out | grep total | awk '{print substr($3, 1, length($3)-1)}'`
COVERAGE=${COVERAGE%.*}

if (( COVERAGE < COVERAGE_LIMIT )); then
    echo "Test coverage too low:" $COVERAGE% "<" $COVERAGE_LIMIT%
    exit 1
fi
