import io
import math

import numpy as np
import segyio
import segyio._segyio

def limit(lane, end, lane_width):
    """TODO: docstr
    """
    return min((lane + 1) * lane_width, end) - (lane * lane_width)

def partition(args, meta, lane, f):
    samples = meta['samples']
    dims = meta['dimensions']
    size = [args.size, args.size, args.size]

    srcdims = (
        limit(lane, len(dims[0]), size[0]),
        len(dims[1]),
        len(dims[2]),
    )
    srcshape = srcdims[:-1]
    srcdtype = np.dtype([
        ('header', 'b', 240),
        ('samples', 'f4', srcdims[-1]),
    ])

    dstshape = (
        size[0],
        int(math.ceil(len(dims[1]) / size[1])) * size[1],
        int(math.ceil(len(dims[2]) / size[2])) * size[2],
    )
    dstdtype = np.dtype('f4')

    src = np.empty(shape = srcshape, dtype = srcdtype)
    dst = np.zeros(shape = dstshape, dtype = dstdtype)

    f.seek(
        lane * (size[0] * len(dims[1])) * (srcdtype.itemsize),
        io.SEEK_CUR,
    )
    f.readinto(src)
    samples = src['samples']
    dst[:srcdims[0], :srcdims[1], :srcdims[2]] = samples[:, :, :]

    return segyio.tools.native(
        data = dst,
        format = meta['format'],
        copy = False,
    )

def upload(args, meta, f):
    samples = meta['samples']
    dims = meta['dimensions']
    f.seek(meta['byteoffset-first-trace'], io.SEEK_CUR)

    lane = 0

    dst = partition(args, meta, lane, f)
    size = (args.size, args.size, args.size)

    for i, d in enumerate(dst.shape):
        if d % size[i] != 0:
            msg = 'inconsistency in dimension {} (shape = {}) and size {}'
            raise RuntimeError(msg.format(i, d, size[i]))

    xyz = [
        (x, y, z)
        for x in [lane]
        for y in range(dst.shape[1] // size[1])
        for z in range(dst.shape[2] // size[2])
    ]

    basename = '{}/{}/{}-{}-{}'.format(
        meta['guid'],
        'src',
        size[0], size[1], size[2],
    )

    for i, (x, y, z) in enumerate(xyz):
        fname = '{}-{}-{}.f32'.format(x, y, z)
        a = np.memmap(fname, dtype = dst.dtype, mode = 'w+', shape = size)
        x = slice(x * size[0], (x + 1) * size[0])
        y = slice(y * size[1], (y + 1) * size[1])
        z = slice(z * size[2], (z + 1) * size[2])
        a[:] = dst[x, y, z]
