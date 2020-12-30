import argparse

from .client import http_session
from .client import ls

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
    session = http_session.fromconfig()
    if args.url is not None:
        session.base_url = args.url
    for cube in ls(session):
        print(cube)
