#!/bin/sh
set -e

echo "yoyou"
echo "Running nc on  $@"

nc -lk -p 5000 -e "$@"
