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
    parser.add_argument('--client-id', type=str,)
    parser.add_argument('--auth-server', type=str)
    parser.add_argument('--scopes', type=str, nargs='+')
    parser.add_argument('--cache-dir', type=str, required=False)
    parser.add_argument('url', type = str, nargs = '?')
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
