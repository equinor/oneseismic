#!/bin/sh
set -e

echo "Running oneseismic-core"

echo " on -A $AZ_BLOB_ACCOUNT"

./one-server -b "0.0.0.0:50051" -A "$AZ_BLOB_ACCOUNT" -k "$AZ_BLOB_KEY"
