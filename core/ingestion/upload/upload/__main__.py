import argparse
import json
import sys

from .upload import upload

def main(argv):
    parser = argparse.ArgumentParser('Ingest SEG-Y')
    parser.add_argument('meta', type = str, help = 'metadata json')
    parser.add_argument('input', type = str, help = 'input SEG-Y file')
    parser.add_argument('container', type = str)
    parser.add_argument('--size', type = int, default = 120)
    args = parser.parse_args(argv)

    with open(args.meta) as f:
        meta = json.load(f)

    with open(args.input, 'rb') as f:
        return upload(args, meta, f)

if __name__ == '__main__':
    print(main(sys.argv[1:]))
