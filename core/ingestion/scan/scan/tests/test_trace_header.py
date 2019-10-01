import io
import segyio
import struct

import pytest

from ..scan import updated_count_interval

def pytest_generate_tests(metafunc):
    if 'header' in metafunc.fixturenames:
        metafunc.parametrize('header', ['little', 'big'], indirect=True)

@pytest.fixture
def header():
    def f(endian):
        stream = io.BytesIO(bytearray(240))
        b = stream.getbuffer()

        if endian == 'little':
            packfmt = '<h'
        else:
            packfmt = '>h'

        fst = int(segyio.su.ns) - 1
        lst = fst + 2
        b[fst:lst] = struct.pack(packfmt, 25)

        fst = int(segyio.su.dt) - 1
        lst = fst + 2
        b[fst:lst] = struct.pack(packfmt, 4000)

        chunk = stream.read(240)
        return segyio.field.Field(buf = chunk, kind = 'trace')
    return f

@pytest.mark.parametrize('endian', ['little', 'big'])
def test_sans_count_sans_interval(header, endian):
    head = header(endian)
    d = updated_count_interval(head, {}, endian)
    assert d['sampleCount'] == 25
    assert d['sampleInterval'] == 4000

@pytest.mark.parametrize('endian', ['little', 'big'])
def test_with_count_sans_interval(header, endian):
    head = header(endian)
    prev = {
        'sampleCount': 1,
    }
    d = updated_count_interval(head, prev, endian)
    assert 'sampleCount' not in d
    assert d['sampleInterval'] == 4000

@pytest.mark.parametrize('endian', ['little', 'big'])
def test_sans_count_with_interval(header, endian):
    head = header(endian)
    prev = {
        'sampleInterval': 1,
    }
    d = updated_count_interval(head, prev, endian)
    assert d['sampleCount'] == 25
    assert 'sampleInterval' not in d

@pytest.mark.parametrize('endian', ['little', 'big'])
def test_with_count_with_interval(header, endian):
    head = header(endian)
    prev = {
        'sampleCount': 1,
        'sampleInterval': 1,
    }
    d = updated_count_interval(head, prev, endian)
    assert 'sampleCount' not in d
    assert 'sampleInterval' not in d
