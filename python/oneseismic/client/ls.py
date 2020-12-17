import argparse

from .client import http_session

def ls(session):
    """List available cubes

    List the cubes stored in oneseismic. The ids returned should all be valid
    arguments for the oneseismic.client.cube class.

    Parameters
    ----------
    session : oneseismic.http_session
        Session with authorization headers set

    Returns
    -------
    guids : iterable of str
        Cube GUIDs

    See also
    --------
    oneseismic.client.cube
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
    session = http_session.fromconfig()
    if args.url is not None:
        session.base_url = args.url
    for cube in ls(session):
        print(cube)
