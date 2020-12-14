import argparse
from . import client

def ls(session):
    """List available cubes

    List the cubes stored in oneseismic. The ids returned should all be
    valid parameters for the cube() method.

    Parameters
    ----------
    session : oneseismic.http_session
        Session with authorization headers set

    Returns
    -------
    ids : iterable of str
        Cube IDs
    """
    return session.get('query').json()['links'].keys()

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
    for cube in ls(c.session):
        print(cube)
