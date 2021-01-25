import argparse
import json
import sys

from .scan import scan
from .scan import resolve_endianness
from .scan import hashio
from .scan import lineset
from ..internal.argparse import add_auth_args
from ..internal.argparse import blobfs_from_args
from ..internal.argparse import localfs_from_args
from ..internal.argparse import get_blob_path

def main(argv):
    parser = argparse.ArgumentParser(
        prog = 'scan',
        description = 'Understand seismic cube identity and layout',
    )
    parser.add_argument('src', type = str, help = 'input SEG-Y file')
    parser.add_argument('--primary-word',   '-P', type = int, default = 189,
        help = 'primary word byte-offset, defaults to 189 (inline)')
    parser.add_argument('--secondary-word', '-S', type = int, default = 193,
        help = 'primary word byte-offset, defaults to 193 (crossline)')
    parser.add_argument('--little-endian', action = 'store_true', default = None)
    parser.add_argument('--big-endian',    action = 'store_true', default = None)
    parser.add_argument('--pretty', action = 'store_true',
        help = 'pretty-print output')
    add_auth_args(parser, direction = 'input')

    args = parser.parse_args(argv)
    endian = resolve_endianness(args.big_endian, args.little_endian)

    try:
        inputfs = blobfs_from_args(
            url     = args.src,
            method  = args.input_auth_method,
            connstr = args.input_connection_string,
            creds   = args.input_credentials,
        )
        container, blob = get_blob_path(args.src)
        inputfs.cd(container)
        src = blob
    except ValueError:
        inputfs = localfs_from_args(args.src)
        src = args.src

    with inputfs.open(src, 'rb') as f:
        stream = hashio(f)
        action = lineset(
            primary   = args.primary_word,
            secondary = args.secondary_word,
            endian    = endian,
        )

        d = scan(stream, action)
        d['guid'] = stream.hexdigest()

    if args.pretty:
        return json.dumps(d, sort_keys = True, indent = 4)

    return json.dumps(d)

if __name__ == '__main__':
    print(main(sys.argv[1:]))
