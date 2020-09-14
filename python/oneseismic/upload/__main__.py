import argparse
import sys
import os
import json
from azure.storage.blob import BlobServiceClient

from .upload import upload

def main(argv):
    dimhelp = 'fragment size (samples) in {} direction'
    parser = argparse.ArgumentParser(
        prog = 'upload',
        description = 'Upload cubes to oneseismic storage',
        epilog = '%(prog)s relies on azure connection strings, see {}'.format(
            'https://docs.microsoft.com/azure/storage/common/storage-configure-connection-string'
        ),
    )
    parser.add_argument('meta', type = str, help = 'metadata json')
    parser.add_argument('input', type = str, help = 'input SEG-Y file')
    parser.add_argument(
        '--subcube-dim-0', '-i',
        type = int,
        default = 120,
        metavar = 'I',
        help = dimhelp.format('X'),
    )
    parser.add_argument(
        '--subcube-dim-1', '-j',
        type = int,
        default = 120,
        metavar = 'J',
        help = dimhelp.format('Y'),
    )
    parser.add_argument(
        '--subcube-dim-2', '-k',
        type = int,
        default = 120,
        metavar = 'K',
        help = dimhelp.format('Z'),
    )
    parser.add_argument(
        '--connection-string', '-s',
        metavar = '',
        type = str,
        help = '''
            Azure connection string for blob store auth. Can also be set
            with the env-var AZURE_CONNECTION_STRING
        ''',
    )
    args = parser.parse_args(argv)

    params = {
        'subcube-dims': (
            args.subcube_dim_0,
            args.subcube_dim_1,
            args.subcube_dim_2,
        ),
    }

    if args.meta == '-':
        meta = json.load(sys.stdin)
    else:
        with open(args.meta) as f:
            meta = json.load(f)

    connection_string = os.environ.get('AZURE_CONNECTION_STRING', None)
    if args.connection_string:
        connection_string = args.connection_string

    if connection_string is None:
        problem = 'No azure connection string'
        solution = 'use --connection-string or env-var AZURE_CONNECTION_STRING'
        sys.exit('{} - {}'.format(problem, solution))


    blob = BlobServiceClient.from_connection_string(connection_string)
    with open(args.input, 'rb') as input:
        upload(params, meta, input, blob)

if __name__ == '__main__':
    print(main(sys.argv[1:]))
