import argparse
import importlib
import sys

if __name__ == "__main__":
    parser = argparse.ArgumentParser(prog = 'oneseismic')
    programs = [
        'login',
        'scan',
        'upload',
    ]
    parser.add_argument('cmd', choices = programs)
    # only parse prog-name, i.e. python3 -m oneseismic scan args... parses only scan
    args = parser.parse_args(sys.argv[1:2])

    cmd = importlib.import_module('oneseismic.{}.__main__'.format(args.cmd))
    # forward everything after prog
    cmd.main(sys.argv[2:])
