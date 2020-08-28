import numpy as np
import oneseismic
import requests
import requests_mock
from oneseismic.core_pb2 import slice_response

session = requests.Session()
adapter = requests_mock.Adapter()
session.mount('mock://', adapter)

dim0 = '[10, 11, 12, 13]'
dim1 = '[20, 21, 22]'
dim2 = '[30, 31]'


def slice_0_12():
    sr = slice_response()
    sr.slice_shape.dim0 = 3
    sr.slice_shape.dim1 = 2
    tile = sr.tiles.add()
    tile.layout.initial_skip = 0
    tile.layout.chunk_size = 2
    tile.layout.iterations = 3
    tile.layout.substride = 2
    tile.layout.superstride = 2
    tile.v.extend([2.00, 2.01, 2.10, 2.11, 2.20, 2.21])
    return sr


def slice_1_22():
    sr = slice_response()
    sr.slice_shape.dim0 = 4
    sr.slice_shape.dim1 = 2
    tile = sr.tiles.add()
    tile.layout.initial_skip = 0
    tile.layout.chunk_size = 2
    tile.layout.iterations = 3
    tile.layout.substride = 2
    tile.layout.superstride = 2
    tile.v.extend([0.20, 0.21, 1.20, 1.21, 2.20, 2.21])
    tile = sr.tiles.add()
    tile.layout.initial_skip = 6
    tile.layout.chunk_size = 2
    tile.layout.iterations = 1
    tile.layout.substride = 2
    tile.layout.superstride = 2
    tile.v.extend([3.20, 3.21])
    return sr


def slice_2_30():
    sr = slice_response()
    sr.slice_shape.dim0 = 4
    sr.slice_shape.dim1 = 3
    tile = sr.tiles.add()
    tile.layout.initial_skip = 0
    tile.layout.chunk_size = 3
    tile.layout.iterations = 3
    tile.layout.substride = 3
    tile.layout.superstride = 3
    tile.v.extend([0.00, 0.10, 0.20, 1.00, 1.10, 1.20, 2.00, 2.10, 2.20])
    tile = sr.tiles.add()
    tile.layout.initial_skip = 9
    tile.layout.chunk_size = 3
    tile.layout.iterations = 1
    tile.layout.substride = 3
    tile.layout.superstride = 3
    tile.v.extend([3.00, 3.10, 3.20])
    return sr


class no_auth:
    def token(self):
        return ''


client = oneseismic.client('http://api', auth=no_auth())
cube = client.cube('test_id')


@requests_mock.Mocker(kw='m')
def test_dims(**kwargs):
    kwargs['m'].get('http://api/test_id/slice/0', text=dim0)
    kwargs['m'].get('http://api/test_id/slice/1', text=dim1)
    kwargs['m'].get('http://api/test_id/slice/2', text=dim2)
    assert cube.dim0 == [10, 11, 12, 13]
    assert cube.dim1 == [20, 21, 22]
    assert cube.dim2 == [30, 31]


@requests_mock.Mocker(kw='m')
def test_slice(**kwargs):
    b12 = slice_0_12().SerializeToString()
    print(b12)
    print(len(b12))
    b12 = len(b12).to_bytes(4, byteorder="little") + b12
    print(b12)
    kwargs['m'].get(
        'http://api/test_id/slice/0/12', content=b12
    )
    # kwargs['m'].get(
    #     'http://api/test_id/slice/1/22', content=slice_1_22().SerializeToString()
    # )
    # kwargs['m'].get(
    #     'http://api/test_id/slice/2/30', content=slice_2_30().SerializeToString()
    # )

    expected_0_12 = np.asarray(
        [
            [2.00, 2.01],
            [2.10, 2.11],
            [2.20, 2.21]
        ]
    )

    expected_1_22 = np.asarray(
        [
            [0.20, 0.21],
            [1.20, 1.21],
            [2.20, 2.21],
            [3.20, 3.21]
        ]
    )

    expected_2_30 = np.asarray(
        [
            [0.00, 0.10, 0.20],
            [1.00, 1.10, 1.20],
            [2.00, 2.10, 2.20],
            [3.00, 3.10, 3.20],
        ]
    )

    assert np.isclose(cube.slice(0, 12), expected_0_12).all()
    # assert np.isclose(cube.slice(1, 22), expected_1_22).all()
    # assert np.isclose(cube.slice(2, 30), expected_2_30).all()
