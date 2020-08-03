import json
import numpy as np
import oneseismic
import requests
import requests_mock

session = requests.Session()
adapter = requests_mock.Adapter()
session.mount('mock://', adapter)

dim0 = '[10, 11, 12, 13]'
dim1 = '[20, 21, 22]'
dim2 = '[30, 31]'

slice_0_12 = json.dumps({
    "slice_shape": {
        "dim0": 3,
        "dim1": 2
    },
    "tiles": [
        {
            "layout": {
                "initial_skip": 0,
                "chunk_size": 2,
                "iterations": 3,
                "substride": 2,
                "superstride": 2,
            },
            "v": [2.00, 2.01, 2.10, 2.11, 2.20, 2.21]
        },
    ]
})

slice_1_22 = json.dumps({
    "slice_shape": {
        "dim0": 4,
        "dim1": 2
    },
    "tiles": [
        {
            "layout": {
                "initial_skip": 0,
                "chunk_size": 2,
                "iterations": 3,
                "substride": 2,
                "superstride": 2,
            },
            "v": [0.20, 0.21, 1.20, 1.21, 2.20, 2.21]
        },
        {
            "layout": {
                "initial_skip": 6,
                "chunk_size": 2,
                "iterations": 1,
                "substride": 2,
                "superstride": 2,
            },
            "v": [3.20, 3.21]
        },
    ]
})

slice_2_30 = json.dumps({
    "slice_shape": {
        "dim0": 4,
        "dim1": 3
    },
    "tiles": [
        {
            "layout": {
                "initial_skip": 0,
                "chunk_size": 3,
                "iterations": 3,
                "substride": 3,
                "superstride": 3,
            },
            "v": [0.00, 0.10, 0.20, 1.00, 1.10, 1.20, 2.00, 2.10, 2.20]
        },
        {
            "layout": {
                "initial_skip": 9,
                "chunk_size": 3,
                "iterations": 1,
                "substride": 3,
                "superstride": 3,
            },
            "v": [3.00, 3.10, 3.20]
        },
    ]
})


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
    kwargs['m'].get('http://api/test_id/slice/0/12', text=slice_0_12)
    kwargs['m'].get('http://api/test_id/slice/1/22', text=slice_1_22)
    kwargs['m'].get('http://api/test_id/slice/2/30', text=slice_2_30)

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

    assert (cube.slice(0, 12) == expected_0_12).all()
    assert (cube.slice(1, 22) == expected_1_22).all()
    assert (cube.slice(2, 30) == expected_2_30).all()
