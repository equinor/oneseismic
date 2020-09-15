import argparse
import json
import sys

from .scan import scan

def main(argv):
    parser = argparse.ArgumentParser(
        prog = 'scan',
        description = 'Understand seismic cube identity and layout',
    )
    parser.add_argument('input', type = str, help = 'input SEG-Y file')
    parser.add_argument('--primary-word',   '-P', type = int, default = 189,
        help = 'primary word byte-offset, defaults to 189 (inline)')
    parser.add_argument('--secondary-word', '-S', type = int, default = 193,
        help = 'primary word byte-offset, defaults to 193 (crossline)')
    parser.add_argument('--little-endian', action = 'store_true', default = None)
    parser.add_argument('--big-endian',    action = 'store_true', default = None)
    parser.add_argument('--pretty', action = 'store_true',
        help = 'pretty-print output')

    args = parser.parse_args(argv)

    with open(args.input, 'rb') as f:
        d = scan(f, args.primary_word, args.secondary_word, args.little_endian, args.big_endian)

    if args.pretty:
        return json.dumps(d, sort_keys = True, indent = 4)

    return json.dumps(d)

if __name__ == '__main__':
    print(main(sys.argv[1:]))
