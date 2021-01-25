import argparse
import json
import os
import sys

from pathlib import Path

from .upload import upload
from ..internal import blobfs
from ..internal import localfs
from ..internal.argparse import blobfs_from_args
from ..internal.argparse import localfs_from_args
from ..internal.argparse import add_auth_args
from ..internal.argparse import get_blob_path

def main(argv):
    parser = argparse.ArgumentParser(
        prog = 'upload',
        description = 'Upload cubes to oneseismic storage',
    )

    parser.add_argument(
        'meta',
        type = str,
        help = 'Metadata (output from scan). Pass - for stdin',
    )
    parser.add_argument(
        'src',
        type = str,
        help = 'Input SEG-Y file. Can be a local file or a blob URL.',
    )
    parser.add_argument(
        'dst',
        type = str,
        help = '''
            Destination storage account. If connection strings are used, this
            is ignored. If omitted, the destination account is inferred from
            the output-credentials.
        ''',
        nargs = '?',
    )
    parser.add_argument(
        '--subcube-dim-0', '-i',
        type = int,
        default = 64,
        metavar = 'i',
        dest = 'i',
    )
    parser.add_argument(
        '--subcube-dim-1', '-j',
        type = int,
        default = 64,
        metavar = 'j',
        dest = 'j',
    )
    parser.add_argument(
        '--subcube-dim-2', '-k',
        type = int,
        default = 64,
        metavar = 'k',
        dest = 'k',
    )
    add_auth_args(parser, direction = 'input')
    add_auth_args(parser, direction = 'output')

    args = parser.parse_args(argv)

    if args.meta == '-':
        meta = json.load(sys.stdin)
    else:
        with open(args.meta) as f:
            meta = json.load(f)

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

    try:
        outputfs = blobfs_from_args(
            url     = args.dst,
            method  = args.output_auth_method,
            connstr = args.output_connection_string,
            creds   = args.output_credentials,
        )
    except ValueError:
        outputfs = localfs_from_args(args.dst)

    fragment_shape = (args.i, args.j, args.k)
    with inputfs.open(src, 'rb') as src:
        upload(meta, fragment_shape, src, outputfs)

if __name__ == '__main__':
    main(sys.argv[1:])
