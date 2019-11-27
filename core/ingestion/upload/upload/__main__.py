import argparse
import json
import sys
import os
from azure.storage.blob import BlockBlobService

from .upload import upload

def main(argv):
    parser = argparse.ArgumentParser('Ingest SEG-Y')
    parser.add_argument('meta', type = str, help = 'metadata json')
    parser.add_argument('input', type = str, help = 'input SEG-Y file')
    parser.add_argument('--container', type = str, default = 'fragments')
    parser.add_argument('--subcube-dim-0', type = int, default = 120)
    parser.add_argument('--subcube-dim-1', type = int, default = 120)
    parser.add_argument('--subcube-dim-2', type = int, default = 120)
    args = parser.parse_args(argv)

    params = {
        'container': args.container,
        'subcube-dims': (
            args.subcube_dim_0,
            args.subcube_dim_1,
            args.subcube_dim_2,
        ),
    }

    with open(args.meta) as f:
        meta = json.load(f)

    blob = BlockBlobService(
        account_name = os.environ['AZURE_ACCOUNT_NAME'],
        account_key  = os.environ['AZURE_ACCOUNT_KEY'],
    )

    import math

    # TODO: this mapping, while simple, should probably be done by the
    # geometric volume translation package
    dims = meta['dimensions']
    first = params['subcube-dims'][0]
    segments = int(math.ceil(len(dims[0]) / first))

    for seg in range(segments):
        with open(args.input, 'rb') as f:
            upload(params, meta, seg, blob, f)

if __name__ == '__main__':
    print(main(sys.argv[1:]))
