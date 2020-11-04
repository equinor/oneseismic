import argparse
import json
import requests
import sys

from .login import login

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
    if args.url is not None:
        authurl = f'{args.url}/config'
        r = requests.get(authurl)
        if r.status_code != 200:
            msg = f'GET {authurl} status {r.status_code}: {r.content}'
            raise RuntimeError(msg)
        config.update(json.loads(r.content))

    if args.client_id is not None:
        config['client_id'] = args.client_id
    if args.auth_server is not None:
        config['authority'] = args.auth_server
    if args.scopes is not None:
        config['scopes'] = args.scopes

    login(
        client_id = config['client_id'],
        auth_server = config['authority'],
        scopes = config['scopes'],
        cache_dir = args.cache_dir,
    )

if __name__ == '__main__':
    main((sys.argv[1:]))
