#! /usr/bin/env python3

import argparse
import numpy as np
import itertools as itr
import math

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('xs', type = int)
    parser.add_argument('ys', type = int)
    parser.add_argument('zs', type = int)

    parser.add_argument('cubeletsize',
                        type = int,
                        default = 4,
                        nargs = '?',
                        help = 'Cuboid size in megabytes',
    )
    parser.add_argument('--output', type = str, default = 'cuboid')

    args = parser.parse_args()

    size = args.cubeletsize * (2**20 / 4)

    dims = (args.xs, args.ys, args.zs)
    cube = np.arange(0, np.prod(dims), dtype = 'single').reshape(dims)

    smallcube = np.cbrt(size)

    x = math.ceil(args.xs / smallcube)
    y = math.ceil(args.ys / smallcube)
    z = math.ceil(args.zs / smallcube)

    for u, v, w in itr.product(range(x), range(y), range(z)):
        u0 = int(u * smallcube)
        v0 = int(v * smallcube)
        w0 = int(w * smallcube)
        u1 = int(u0 + smallcube)
        v1 = int(v0 + smallcube)
        w1 = int(w0 + smallcube)

        cubelet = cube[u0:u1, v0:v1, w0:w1]
        fname = '{}-{}-{}-{}.f32'.format(args.output, u, v, w)
        print('writing {}'.format(fname))
        cubelet.tofile(fname)
