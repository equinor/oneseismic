import argparse
import json
import requests
import sys
import urllib

from .login import login

def assumehttps(url):
    """
    Getting this right is surprisingly difficult.

    The urllib.parse library only recognises <url>/<path> as netloc = <url>
    whenever <url> has // (usually looks like it's a part of <scheme>), but
    <scheme>:// is often omitted in practice.

    oneseismic.domain.com =>
        scheme = ''
        netloc = ''
        path = oneseismic.domain.com

    This is consistent with how oneseismic.domain.com looks like a relative
    path - it's only from context we know to assume a http(s) and a network
    request [1] if the protocol is unspecified, in which case a relative path
    does not make sense.

    So if both the protocol is unset *and* the netloc is unset, assume this
    means a https request, which it will for any non-test use.

    [1] It doesn't even have to be - it could for all intents & purposes be a
    file:// get with a properly-formatted on-disk file.
    """
    parts = urllib.parse.urlsplit(url)

    if parts.scheme == '' and parts.netloc == '':
        parts = urllib.parse.urlsplit(f'//{url}', scheme = 'https')

    return urllib.parse.urlunsplit((
            parts.scheme,
            parts.netloc,
            parts.path + '/config',
            parts.query,
            parts.fragment,
    ))

def main(argv):
    parser = argparse.ArgumentParser(
        prog = 'login',
        description = 'Log in to oneseismic',
    )
    parser.add_argument('--client-id',
        type = str,
        help = 'The client/application ID that will '
               'perform requests on your behalf',
    )
    parser.add_argument('--auth-server',
            type = str,
            help = 'The authority or login server, '
                   'e.g. https://login.microsoftonline.com/<tenant-id>',
            )
    parser.add_argument('--scopes',
            type  = str,
            nargs = '+',
            help  = 'Scopes/actions to allow oneseismic to perform',
    )
    parser.add_argument( '--cache-dir',
            type = str,
            help = 'Directory to store oneseismic config and token cache. '
                   'Providing a fresh will effectively cause new login, '
                   'and managing multiple caches will enable multiple '
                   'oneseismic instances',
    )
    parser.add_argument('url',
            type  = str,
            nargs = '?',
            help  = 'Base URL to get oneseismic config from. '
                    'Any flag passed on the command line will override '
                    'the server provided configuration, but generally '
                    'only this option should be needed',
    )
    args = parser.parse_args(argv)

    config = {}
    base_url = None
    if args.url is not None:
        url = assumehttps(args.url)
        r = requests.get(url)
        if r.status_code != 200:
            msg = f'GET {url} status {r.status_code}: {r.content}'
            raise RuntimeError(msg)
        config.update(json.loads(r.content))

        # Store only the base URL, i.e. remove path and query components
        u = urllib.parse.urlparse(url)
        base_url = urllib.parse.urlunsplit((u.scheme, u.netloc, '', '', ''))

    if args.client_id is not None:
        config['client_id'] = args.client_id
    if args.auth_server is not None:
        config['authority'] = args.auth_server
    if args.scopes is not None:
        config['scopes'] = args.scopes

    login(
        url = base_url,
        client_id = config['client_id'],
        auth_server = config['authority'],
        scopes = config['scopes'],
        cache_dir = args.cache_dir,
    )

if __name__ == '__main__':
    main((sys.argv[1:]))
