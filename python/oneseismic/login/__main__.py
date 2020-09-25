import argparse
import sys

from .login import login

def main(argv):
    parser = argparse.ArgumentParser(
        prog = 'login',
        description = 'Log in to oneseismic',
    )
    parser.add_argument('--client-id', type=str, required=True)
    parser.add_argument('--auth-server', type=str, required=True)
    parser.add_argument('--scopes', type=str, nargs='+', required=True)
    parser.add_argument('--cache-dir', type=str, required=False)

    args = parser.parse_args(argv)
    login(args.client_id, args.auth_server, args.scopes, args.cache_dir)

if __name__ == '__main__':
    main((sys.argv[1:]))
