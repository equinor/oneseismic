import argparse
import json
import sys

from .scan import scan
from .scan import resolve_endianness
from .scan import hashio
from .segmenter import segmenter

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
    parser.add_argument(
        '--method',
        choices = ['manifest', 'outline'],
        default = 'manifest',
        type = str,
        help = '''Scan methods
            manifest : scan geometry and volume metadata
            outline  : scan the in/crossline set
        ''',
    )

    args = parser.parse_args(argv)
    endian = resolve_endianness(args.big_endian, args.little_endian)

    with open(args.input, 'rb') as f:
        stream = f
        if args.method == 'manifest':
            from .segmenter import segmenter
            action = segmenter(
                primary   = args.primary_word,
                secondary = args.secondary_word,
                endian    = endian,
            )
            stream = hashio(f)

        elif args.method == 'outline':
            from .segmenter import outline
            action = outline(
                primary   = args.primary_word,
                secondary = args.secondary_word,
                endian    = endian,
            )

        d = scan(stream, action)

        if args.method == 'manifest':
            d['guid'] = stream.hexdigest()

    if args.pretty:
        return json.dumps(d, sort_keys = True, indent = 4)

    return json.dumps(d)

if __name__ == '__main__':
    print(main(sys.argv[1:]))
