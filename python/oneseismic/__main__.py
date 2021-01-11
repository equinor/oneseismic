import argparse
import importlib
import sys

if __name__ == "__main__":
    parser = argparse.ArgumentParser(prog = 'oneseismic')
    programs = {
        'login':   'login.__main__',
        'scan':    'scan.__main__',
        'upload':  'upload.__main__',
        'ls':      'client.ls',
    }
    parser.add_argument('cmd', choices = programs.keys())
    # only parse prog-name, i.e. python3 -m oneseismic scan args... parses only scan
    args = parser.parse_args(sys.argv[1:2])

    cmd = importlib.import_module('oneseismic.{}'.format(programs[args.cmd]))
    # forward everything after prog. by convention, there should be a main()
    # function in the file
    r = cmd.main(sys.argv[2:])
    if r is not None:
        print(r)
