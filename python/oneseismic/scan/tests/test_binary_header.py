import io
import math
import struct
import segyio
import segyio._segyio

import pytest
from hypothesis import given
from hypothesis.strategies import integers

from ..scanners import scanner

def emptybinary():
    return bytearray(400)

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
def test_unsupported_format_raises(endian, fmt):
    chunk = emptybinary()
    if endian == 'big':
        packed = struct.pack('>h', fmt[1])
    else:
        packed = struct.pack('<h', fmt[1])

    fst = int(segyio.su.format) - 3201
    lst = fst + 2
    chunk[fst:lst] = packed
    binary = segyio.field.Field(buf = chunk, kind = 'binary')
    with pytest.raises(NotImplementedError):
        scan = scanner(endian = endian)
        _ = scan.scan_binary_header(binary)

supported_formats = [
    ('ibm', 1),
    ('f4',  5),
]
@pytest.mark.parametrize('endian', ['big', 'little'])
@pytest.mark.parametrize('fmt', supported_formats)
def test_supported_formats(endian, fmt):
    chunk = emptybinary()
    if endian == 'big':
        packed = struct.pack('>h', fmt[1])
    else:
        packed = struct.pack('<h', fmt[1])

    fst = int(segyio.su.format) - 3201
    lst = fst + 2
    chunk[fst:lst] = packed
    binary = segyio.field.Field(buf = chunk, kind = 'binary')

    scan = scanner(endian = endian)
    scan.scan_binary_header(binary)
    out = scan.report()
    assert out['format'] == fmt[1]

@pytest.mark.parametrize('endian', ['big', 'little'])
@given(integers(min_value = 0, max_value = math.pow(2, 15) - 1))
def test_get_sample_count(endian, val):
    chunk = emptybinary()
    if endian == 'big':
        packfmt = '>h'
    else:
        packfmt = '<h'

    fst = int(segyio.su.hns) - 3201
    lst = fst + 2
    chunk[fst:lst] = struct.pack(packfmt, val)

    # set format to ibm float - it doesn't affect the outcome of this test, and
    # if it is unset then the scan_binary function will fail
    fst = int(segyio.su.format) - 3201
    lst = fst + 2
    chunk[fst:lst] = struct.pack(packfmt, 1)

    binary = segyio.field.Field(buf = chunk, kind = 'binary')
    scan = scanner(endian = endian)
    scan.scan_binary_header(binary)
    out = scan.report()
    assert out['samples'] == val

@pytest.mark.parametrize('endian', ['big', 'little'])
@given(integers(min_value = 0, max_value = math.pow(2, 15) - 1))
def test_get_sample_interval(endian, val):
    chunk = emptybinary()
    if endian == 'big':
        packfmt = '>h'
    else:
        packfmt = '<h'

    fst = int(segyio.su.hdt) - 3201
    lst = fst + 2
    chunk[fst:lst] = struct.pack(packfmt, val)

    # set format to ibm float - it doesn't affect the outcome of this test, and
    # if it is unset then the scan_binary function will fail
    fst = int(segyio.su.format) - 3201
    lst = fst + 2
    chunk[fst:lst] = struct.pack(packfmt, 1)

    binary = segyio.field.Field(buf = chunk, kind = 'binary')
    scan = scanner(endian = endian)
    scan.scan_binary_header(binary)
    out = scan.report()
    assert out['sampleinterval'] == val
