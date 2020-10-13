import argparse
import sys

from .login import login

def main(argv):
    parser = argparse.ArgumentParser(
        prog = 'login',
        description = 'Log in to oneseismic',
    )
    parser.add_argument(
        '--client-id',
        type=str,
        required=True,
        help='The Application (client) ID of the Azure AD app registration.',
    )
    parser.add_argument(
        '--auth-server',
        type=str,
        required=True,
        help='Authentication server on the form <authentication-endpoint>/<tenant-id>/v2.0.',
    )
    parser.add_argument(
        '--scopes',
        type=str,
        nargs='+',
        required=True,
        help='Scopes requested to access a protected API (resource).',
    )
    parser.add_argument(
        '--cache-dir',
        type=str,
        required=False,
        help='Cache directory for storing token.',
    )

    args = parser.parse_args(argv)
    login(args.client_id, args.auth_server, args.scopes, args.cache_dir)

if __name__ == '__main__':
    main((sys.argv[1:]))
