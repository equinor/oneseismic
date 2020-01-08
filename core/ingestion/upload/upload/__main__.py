import argparse
import json
import sys
import os
from azure.storage.blob import BlobServiceClient

from .upload import upload

def main(argv):
    parser = argparse.ArgumentParser('Ingest SEG-Y')
    parser.add_argument('meta', type = str, help = 'metadata json')
    parser.add_argument('input', type = str, help = 'input SEG-Y file')
    parser.add_argument('--subcube-dim-0', type = int, default = 120)
    parser.add_argument('--subcube-dim-1', type = int, default = 120)
    parser.add_argument('--subcube-dim-2', type = int, default = 120)
    args = parser.parse_args(argv)

    params = {
        'subcube-dims': (
            args.subcube_dim_0,
            args.subcube_dim_1,
            args.subcube_dim_2,
        ),
    }

    with open(args.meta) as f:
        meta = json.load(f)

    blob = BlobServiceClient.from_connection_string(os.environ['AZURE_CONNECTION_STRING'])

    # TODO: this mapping, while simple, should probably be done by the
    # geometric volume translation package
    import math
    dims = meta['dimensions']
    first = params['subcube-dims'][0]
    segments = int(math.ceil(len(dims[0]) / first))

    for seg in range(segments):
        with open(args.input, 'rb') as f:
            upload_segment(params, meta, seg, blob, f)

if __name__ == '__main__':
    print(main(sys.argv[1:]))
