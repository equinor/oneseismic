import os

import numpy as np
import segyio
from hypothesis import given
from hypothesis.strategies import integers, lists

from ..upload import load_segment, pad, segment_limit


def datadir(filename):
    root = os.path.dirname(__file__)
    data = os.path.join(root, "data")
    return os.path.join(data, filename)


def test_src_segment_limit():
    segment = 0
    end = 10
    max_width = 5
    expected = 5
    assert segment_limit(segment, end, max_width) == expected

    segment = 0
    end = 5
    max_width = 2
    expected = 2
    assert segment_limit(segment, end, max_width) == expected

    segment = 1
    end = 9
    max_width = 5
    expected = 4
    assert segment_limit(segment, end, max_width) == expected


def test_load_segment():
    f = open(datadir("small.sgy"), "rb")
    f.seek(3600)

    cube_size = (5, 5, 50)
    segment_width = 2
    format = 1

    sgy = segyio.open(datadir("small.sgy"))
    cube = segyio.tools.cube(sgy)

    segment = 0
    seg = load_segment(cube_size, segment_width, segment, format, f)
    expected = cube[0:2, :, :]
    assert np.isclose(seg, expected).all()

    f.seek(3600)

    segment = 2
    seg = load_segment(cube_size, segment_width, segment, format, f)
    expected = cube[4:5, :, :]
    assert np.isclose(seg, expected).all()


@given(
    lists(integers(min_value=1, max_value=32), min_size=3, max_size=3),
    lists(integers(min_value=1, max_value=32), min_size=3, max_size=3),
)
def test_pad(fragment_dims, srcdims):

    src = np.ones(srcdims)

    result = pad(fragment_dims, src)

    assert result.shape[0] % fragment_dims[0] == 0
    assert result.shape[1] % fragment_dims[1] == 0
    assert result.shape[2] % fragment_dims[2] == 0

    assert result.shape[0] >= src.shape[0]
    assert result.shape[1] >= src.shape[1]
    assert result.shape[2] >= src.shape[2]

    assert (result[: src.shape[0], : src.shape[1], : src.shape[2]] == 1).all()

    assert (result[src.shape[0] :, :, :] == 0).all()
    assert (result[:, src.shape[1] :, :] == 0).all()
    assert (result[:, :, src.shape[2] :] == 0).all()
