import argparse
from . import client

def main(argv):
    parser = argparse.ArgumentParser(
        prog = 'ls',
        description = 'list cubes',
    )
    parser.add_argument('url',
        type = str,
    )

    args = parser.parse_args(argv)
    c = client(args.url)
    for cube in c.ls():
        print(cube)
