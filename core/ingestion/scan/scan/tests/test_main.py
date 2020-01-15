import json
import os

from ..__main__ import main

expected = {
    "byteorder": "big",
    "format": 1,
    "samples": 50,
    "sampleinterval": 4.0,
    "byteoffset-first-trace": 3600,
    "guid": "86f5f8f783fabe2773531d5529226d37b6c9bdcf",
    "dimensions": [
        [1, 2, 3, 4, 5],
        [20, 21, 22, 23, 24],
        [
            0.0,
            4.0,
            8.0,
            12.0,
            16.0,
            20.0,
            24.0,
            28.0,
            32.0,
            36.0,
            40.0,
            44.0,
            48.0,
            52.0,
            56.0,
            60.0,
            64.0,
            68.0,
            72.0,
            76.0,
            80.0,
            84.0,
            88.0,
            92.0,
            96.0,
            100.0,
            104.0,
            108.0,
            112.0,
            116.0,
            120.0,
            124.0,
            128.0,
            132.0,
            136.0,
            140.0,
            144.0,
            148.0,
            152.0,
            156.0,
            160.0,
            164.0,
            168.0,
            172.0,
            176.0,
            180.0,
            184.0,
            188.0,
            192.0,
            196.0,
        ],
    ],
}

expected_lsb = expected.copy()
expected_lsb.update(
    {"byteorder": "little", "guid": "2efb1a0bebb577b27a9e0503b06cf0935eebca78"}
)

expected_il5_xl21 = expected.copy()
expected_il5_xl21.update({"guid": "35368c1a2aec523c324ae1fd1fb42f1994f46fbe"})

expected_il5_xl21_lsb = expected.copy()
expected_il5_xl21_lsb.update(
    {"byteorder": "little", "guid": "d3f1e9fa8b1ebaae26c55fa3e9beba0b4fe57287"}
)

expected_2byte_keys = expected.copy()
expected_2byte_keys.update(
    {
        "guid": "526026c67842afbb37ac99e81371550258554665",
        "dimensions": expected["dimensions"].copy(),
    }
)

expected_missing_line_numbers = expected.copy()
expected_missing_line_numbers.update(
    {
        "guid": "b9bff5837e3f654585cc8a730fedb9730309e4ee",
        "dimensions": expected["dimensions"].copy(),
    }
)
expected_missing_line_numbers["dimensions"][0] = [1, 2, 3, 5, 6]
expected_missing_line_numbers["dimensions"][1] = [20, 21, 23, 24, 25]


def datadir(filename):
    root = os.path.dirname(__file__)
    data = os.path.join(root, "data")
    return os.path.join(data, filename)


def test_small_little_endian():
    s = main(["--little-endian", datadir("small-lsb.sgy")])
    run = json.loads(s)

    assert run == expected_lsb


def test_small_big_endian():
    s = main(["--big-endian", datadir("small.sgy")])
    run = json.loads(s)

    assert run == expected


def test_small_big_endian_default():
    s = main([datadir("small.sgy")])
    run = json.loads(s)

    assert run == expected


def test_small_pretty_does_not_break_output():
    ugly = main([datadir("small.sgy")])
    pretty = main(["--pretty", datadir("small.sgy")])

    ugly = json.loads(ugly)
    pretty = json.loads(pretty)

    assert ugly == pretty


def test_small_big_endian_custom_inline_crossline_offset():
    s = main(
        [
            "--primary-word",
            "5",
            "--secondary-word",
            "21",
            datadir("small-iline-5-xline-21.sgy"),
        ]
    )
    run = json.loads(s)

    assert run == expected_il5_xl21


def test_small_little_endian_custom_inline_crossline_offset():
    s = main(
        [
            "--primary-word",
            "5",
            "--secondary-word",
            "21",
            "--little-endian",
            datadir("small-iline-5-xline-21-lsb.sgy"),
        ]
    )
    run = json.loads(s)

    assert run == expected_il5_xl21_lsb


def test_2byte_primary_2byte_secondary():
    s = main(
        [
            "--primary-word",
            "29",
            "--secondary-word",
            "71",
            datadir("small-2byte-keys.sgy"),
        ]
    )
    result = json.loads(s)

    expected_2byte_keys["dimensions"][0] = [11, 12, 13, 14, 15]
    expected_2byte_keys["dimensions"][1] = [30, 31, 32, 33, 34]

    assert result == expected_2byte_keys


def test_2byte_primary_4byte_secondary():
    s = main(
        [
            "--primary-word",
            "29",
            "--secondary-word",
            "193",
            datadir("small-2byte-keys.sgy"),
        ]
    )
    result = json.loads(s)

    expected_2byte_keys["dimensions"][0] = [11, 12, 13, 14, 15]
    expected_2byte_keys["dimensions"][1] = [20, 21, 22, 23, 24]

    assert result == expected_2byte_keys


def test_4byte_primary_2byte_secondary():
    s = main(
        [
            "--primary-word",
            "189",
            "--secondary-word",
            "71",
            datadir("small-2byte-keys.sgy"),
        ]
    )
    result = json.loads(s)

    expected_2byte_keys["dimensions"][0] = [1, 2, 3, 4, 5]
    expected_2byte_keys["dimensions"][1] = [30, 31, 32, 33, 34]

    assert result == expected_2byte_keys


def test_missing_line_numbers():
    s = main([datadir("missing-line-numbers.sgy")])
    result = json.loads(s)

    assert result == expected_missing_line_numbers
