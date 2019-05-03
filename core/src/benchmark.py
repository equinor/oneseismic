#! /usr/bin/env python3

import argparse
import tempfile
import os
from subprocess import Popen

def run_benchmark(dims, base_dir, generator_path,
                  stitch_path, surface_dir, verifyer_path):
    with tempfile.TemporaryDirectory(dir=base_dir) as tmpdirname:
        print('Subcube dimenions: x = {}, y = {}, z = {}'.format(*dims))

        cubes_dir = os.path.join(tmpdirname, 'cubes')
        os.mkdir(cubes_dir)

        generate_cmd = '{} --output-dir {} 4000 3000 1251 {} {} {}' \
                       .format(generator_path, cubes_dir, *dims)
        os.system(generate_cmd)

        for srfc in os.listdir(surface_dir):
            surface_path = os.path.join(surface_dir, srfc)
            verify_cmd = '{0} shatter.manifest -i {1} < {2} | {3} shatter.manifest {2} -i {1} ' \
                         .format(stitch_path, cubes_dir, surface_path, verifyer_path)

            if os.system(verify_cmd) != 0:
                print("Verify failed!")
                return

        for i in range(10):
            for srfc in os.listdir(surface_dir):
                surface_path = os.path.join(surface_dir, srfc)

                stitch_cmd = '{} shatter.manifest -t -i {} < {} > /dev/null' \
                             .format(stitch_path, cubes_dir, surface_path)

                os.system(stitch_cmd)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--base-dir', type = str, default = './')
    parser.add_argument('--generator-path', type = str)
    parser.add_argument('--stitch-path', type = str)
    parser.add_argument('--surface-dir', type = str)
    parser.add_argument('--verifyer-path', type = str)

    args = parser.parse_args()

    shatter_sizes = \
        [
            [ 280, 280, 280 ],
            [ 280, 280,  12 ],
            [ 280, 280,  24 ],
            [ 280, 280,  48 ],
            [  12, 280,  12 ],
            [  12, 280, 280 ],
            [ 280,  12,  12 ],
            [ 280,  12, 280 ],
            [  12,  12,  12 ]
        ]

    for sz in shatter_sizes:
        run_benchmark(sz, args.base_dir, args.generator_path,
                      args.stitch_path, args.surface_dir, args.verifyer_path)
