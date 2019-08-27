#!/bin/sh
set -e

echo "yoyou"
echo "Running nc on  $@"

# mount -t tmpfs -o size=16g tmpfs /mnt/ramdisk
# mkdir /mnt/ramdisk/blobfusetmp
# chown appuser /mnt/ramdisk/blobfusetmp


touch ~/fuse_connection.cfg
chmod 600 ~/fuse_connection.cfg
~/fuse_connection.cfg < envsubst < tmp_fuse_connection.cfg
# nc -lk -p 5000 -e "$@"
exec "$@"
