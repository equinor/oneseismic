#!/bin/bash
set -e
echo "Running nc $@"

exec "nc -lk -p 5000 -e $@"
