import io
import math
import logging
import os

import numpy as np
import segyio
import segyio._segyio
import azure.storage.blob
import tqdm

def segment_limit(segment, end, max_width):
    """ Unpadded segment width

    The segment width of the source array. This is equal to the destination
    arrays segment width for all but the last segment.
    """
    return min((segment + 1) * max_width, end) - (segment * max_width)


def load_segment(cube_dims, segment_width, segment, format, f):
    """ Load a segment from stream

    A segment consists of a part of the data, split along the first axis.

    Parameters
    ----------
    cube_dims : tuple of int
        dimensions of the entire unpadded cube

    segment_width : int
        dimensions of the fragments the cube is to be split into

    segment : int
        the segment to be loaded

    format : int
        formating code used by the segyio library to interpret the data

    f : stream_like
        an open io.IOBase like stream

    Returns
    -------
    segment : numpy.ndarray

    """
    segment_dims = (
        segment_limit(segment, cube_dims[0], segment_width),
        cube_dims[1],
        cube_dims[2],
    )
    shape = segment_dims[:-1]

    # Datatype corresponding to the layout of a trace in the SEGY file
    # [<header>|<samples>]. This allows for a segment of the file to be
    # memcopied into an array and the traces to be extracted using numpy array
    # slicing. Since we are not using the header it is treated as a blob.
    srcdtype = np.dtype([
        ('header', 'b', 240),
        ('samples', 'f4', segment_dims[-1]),
    ])

    src = np.empty(shape = shape, dtype = srcdtype)

    f.seek(
        segment * (segment_width * cube_dims[1] * srcdtype.itemsize),
        io.SEEK_CUR,
    )
    f.readinto(src)

    return segyio.tools.native(
        data = src['samples'],
        format = format
    )


def pad(fragment_dims, src):
    r""" Pad array so that dimensions are a multiple of fragment_dims

    The dimensions of the destinatin ndarray (dst1, dst2, dst3) is set to be a
    multiple of the fragment_dims along the corrresponding axis such that the
    source dimensions (src1, src2, src3) are src1 <= dst1, src2 <= dst2,
    src3 <= dst3.

    The source data is split along the first axis such that a segment of maximum
    width sz1 is extracted.

                            dst1       ...        dst1
                             ^                     ^
                      /¨¨¨¨¨¨ ¨¨¨¨¨¨\       /¨¨¨¨¨¨ ¨¨¨¨¨¨\
                            src1       ...      src1
                             ^                   ^
                      /¨¨¨¨¨¨ ¨¨¨¨¨¨\       /¨¨¨¨ ¨¨¨¨\
              /   /   . – . – . – . +  ...  . – . – . – # +
              |   |   |             |  ...  |         ¦   |
              |   |   .             .  ...  .         ¦   #
              |   |   |             |  ...  |         ¦   |
              |   |   .             .  ...  .         ¦   #
              |   |   |             |  ...  |         ¦   |
              |   |   . – . – . – . +  ...  . – . – . – # +
              |   |   |             |  ...  |         ¦   |
    src2      |  <    .             .  ...  .         ¦   #
              |   |   |             |  ...  |         ¦   |
    dst2     <    |   .             .  ...  .         ¦   #
              |   |   |             |  ...  |         ¦   |
              |   |   . – . – . – . +  ...  . – . – . – # +
              |   |   |             |  ...  |         ¦   |
              |   |   .             .  ...  .         ¦   #
              |   |   |             |  ...  |         ¦   |
              |   \   .-------------.  ...  .---------+   #
              |       #             #  ...  #             #
              \       # – # – # – # +  ...  # – # – # – # +

    Parameters
    ----------
    fragment_dims : tuple of int
        after the padding has been added, the resulting cube can be split up in
        fragments of this dimensionality

    src : numpy.ndarray

    Returns
    -------
    dst : numpy.ndarray

    """

    srcdims = src.shape

    dstshape = (
        int(math.ceil(srcdims[0] / fragment_dims[0])) * fragment_dims[0],
        int(math.ceil(srcdims[1] / fragment_dims[1])) * fragment_dims[1],
        int(math.ceil(srcdims[2] / fragment_dims[2])) * fragment_dims[2],
    )
    dstdtype = np.dtype('f4')

    dst = np.zeros(shape = dstshape, dtype = dstdtype)

    dst[:srcdims[0], :srcdims[1], :srcdims[2]] = src[:, :, :]

    return dst


def upload(params, meta, segment, blob, f):
    samples = meta['samples']
    dims = meta['dimensions']
    format = meta['format']
    fragment_dims = params['subcube-dims']
    f.seek(meta['byteoffset-first-trace'], io.SEEK_CUR)

    cube_dims = (len(dims[0]), len(dims[1]), len(dims[2]))

    src = load_segment(cube_dims, fragment_dims[0], segment, format, f)
    dst = pad(fragment_dims, src)

    for i, d in enumerate(dst.shape):
        if d % fragment_dims[i] != 0:
            msg = 'inconsistency in dimension {} (shape = {}) and fragment_dims {}'
            raise RuntimeError(msg.format(i, d, fragment_dims[i]))

    xyz = [
        (x, y, z)
        for x in [segment]
        for y in range(dst.shape[1] // fragment_dims[1])
        for z in range(dst.shape[2] // fragment_dims[2])
    ]

    container = meta['guid']
    basename = '{}/{}-{}-{}'.format(
        'src',
        fragment_dims[0], fragment_dims[1], fragment_dims[2],
    )

    exists = blob.create_container(
        container_name = container,
        # public_access = 'off',
    )

    if not exists:
        # TODO: this should not be a hard fail, but rather check if the upload
        # is incomplete, or this is a new fragmentation if the upload actually
        # was complete, exit (maybe as success (fast-forward), returning the
        # cube ID)
        raise RuntimeError('container {} already exists'.format(container))

    tqdm_opts = {
        'desc': 'uploading segment {}'.format(segment),
        'unit': ' fragment',
        'total': len(xyz),
    }
    for x, y, z in tqdm.tqdm(xyz, **tqdm_opts):
        fname = '{}-{}-{}.f32'.format(x, y, z)
        y = slice(y * fragment_dims[1], (y + 1) * fragment_dims[1])
        z = slice(z * fragment_dims[2], (z + 1) * fragment_dims[2])
        blob_name = '{}/{}'.format(basename, fname)
        logging.info('uploading %s to %s', blob_name, container)
        # TODO: consider implications and consequences and how to handle an
        # already-existing fragment with this ID
        blob.create_blob_from_bytes(
            container_name = container,
            blob_name = blob_name,
            blob = bytes(dst[:, y, z]),
        )
