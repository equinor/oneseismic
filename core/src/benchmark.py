#! /usr/bin/env python3

import argparse
import tempfile
import os
from subprocess import Popen, PIPE
import numpy as np

DEVNULL = open(os.devnull, 'wb')

def run_benchmark(dims, base_dir, generator_path,
                  stitch_path, surface_dir, verifyer_path):

    with tempfile.TemporaryDirectory(dir=base_dir) as tmpdirname:
        print('Subcube dimenions: x = {}, y = {}, z = {}'.format(*dims))

        cubes_dir = os.path.join(tmpdirname, 'cubes')
        os.mkdir(cubes_dir)

        generate_cmd = '{} --output-dir {} 4000 3000 1251 {} {} {}' \
                       .format(generator_path, cubes_dir, *dims)
        os.system(generate_cmd)

        manifest = open(os.path.join(cubes_dir, 'shatter.manifest'))
        manifest = manifest.read().encode()
        manifest_size = np.int32(len(manifest))

        stitch_cmd = [str(stitch_path), '-i', cubes_dir]

        for srfc in os.listdir(surface_dir):
            surface_path = os.path.join(surface_dir, srfc)

            surface = open(surface_path, "rb").read()

            verify_cmd = [str(verifyer_path),
                          'shatter.manifest',
                          surface_path,
                          '-i',
                          cubes_dir]

            p1 = Popen(stitch_cmd, stdout=PIPE, stdin=PIPE)
            p2 = Popen(verify_cmd, stdin=p1.stdout)
            p1.stdin.write(b'M:')
            p1.stdin.write(manifest_size.tobytes())
            p1.stdin.write(manifest)
            p1.stdin.write(surface)
            p1.stdin.close()

            if p2.wait() != 0:
                print("Verify failed!")
                return

        stitch_cmd.append('-t')

        for i in range(10):
            for srfc in os.listdir(surface_dir):
                surface_path = os.path.join(surface_dir, srfc)
                surface = open(surface_path, "rb").read()

                p = Popen(stitch_cmd, stdout=DEVNULL, stdin=PIPE)

                p.stdin.write(b'M:')
                p.stdin.write(manifest_size.tobytes())
                p.stdin.write(manifest)
                p.stdin.write(surface)
                p.stdin.close()
                p.communicate()


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
