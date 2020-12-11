import argparse
from . import client

def main(argv):
    parser = argparse.ArgumentParser(
        prog = 'ls',
        description = 'list cubes',
    )
    parser.add_argument('url',
        type  = str,
        nargs = '?',
        help  = 'URL to the oneseismic installation to list. '
                'If unspecified, ls will use the url from the cached config',
    )

    args = parser.parse_args(argv)
    c = client(args.url)
    for cube in c.ls():
        print(cube)
