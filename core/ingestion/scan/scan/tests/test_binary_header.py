import io
import math
import struct

import pytest
import segyio
from hypothesis import given
from hypothesis.strategies import integers

from ..scan import scan_binary


@pytest.fixture
def textbin():
    return io.BytesIO(bytearray(3600))

unsupported_formats = [
    ('i4', 2),
    ('i2', 3),
    ('f8', 6),
    ('i3', 7),
    ('i1', 8),
    ('i8', 9),
    ('u4', 10),
    ('u2', 11),
    ('u8', 12),
    ('u3', 15),
    ('u1', 16),
    ('fixedpoint', 4),
]
@pytest.mark.parametrize('endian', ['big', 'little'])
@pytest.mark.parametrize('fmt', unsupported_formats)
def test_unsupported_format_raises(textbin, endian, fmt):
    if endian == 'big':
        packed = struct.pack('>h', fmt[1])
    else:
        packed = struct.pack('<h', fmt[1])

    fst = int(segyio.su.format) - 1
    lst = fst + 2
    textbin.getbuffer()[fst:lst] = packed

    textbin.seek(3200)
    with pytest.raises(NotImplementedError):
        _ = scan_binary(textbin, endian = endian)

supported_formats = [
    ('ibm', 1),
    ('f4',  5),
]
@pytest.mark.parametrize('endian', ['big', 'little'])
@pytest.mark.parametrize('fmt', supported_formats)
def test_supported_formats(textbin, endian, fmt):
    if endian == 'big':
        packed = struct.pack('>h', fmt[1])
    else:
        packed = struct.pack('<h', fmt[1])

    fst = int(segyio.su.format) - 1
    lst = fst + 2
    textbin.getbuffer()[fst:lst] = packed
    textbin.seek(3200)

    out = scan_binary(textbin, endian = endian)
    assert out['format'] == fmt[1]

@pytest.mark.parametrize('endian', ['big', 'little'])
@given(integers(min_value = 0, max_value = math.pow(2, 15) - 1))
def test_get_sample_count(textbin, endian, val):
    if endian == 'big':
        packfmt = '>h'
    else:
        packfmt = '<h'

    fst = int(segyio.su.hns) - 1
    lst = fst + 2
    textbin.getbuffer()[fst:lst] = struct.pack(packfmt, val)

    # set format to ibm float - it doesn't affect the outcome of this test, and
    # if it is unset then the scan_binary function will fail
    fst = int(segyio.su.format) - 1
    lst = fst + 2
    textbin.getbuffer()[fst:lst] = struct.pack(packfmt, 1)
    textbin.seek(3200)

    out = scan_binary(textbin, endian = endian)
    assert out['samples'] == val

@pytest.mark.parametrize('endian', ['big', 'little'])
@given(integers(min_value = 0, max_value = math.pow(2, 15) - 1))
def test_get_sample_interval(textbin, endian, val):
    if endian == 'big':
        packfmt = '>h'
    else:
        packfmt = '<h'

    fst = int(segyio.su.hdt) - 1
    lst = fst + 2
    textbin.getbuffer()[fst:lst] = struct.pack(packfmt, val)

    # set format to ibm float - it doesn't affect the outcome of this test, and
    # if it is unset then the scan_binary function will fail
    fst = int(segyio.su.format) - 1
    lst = fst + 2
    textbin.getbuffer()[fst:lst] = struct.pack(packfmt, 1)
    textbin.seek(3200)

    out = scan_binary(textbin, endian = endian)
    assert out['sampleinterval'] == val
