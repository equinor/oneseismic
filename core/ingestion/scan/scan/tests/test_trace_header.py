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
    assert d['samples'] == 25
    assert d['sampleinterval'] == 4000

@pytest.mark.parametrize('endian', ['little', 'big'])
def test_with_count_sans_interval(header, endian):
    head = header(endian)
    prev = {
        'samples': 1,
    }
    d = updated_count_interval(head, prev, endian)
    assert 'samples' not in d
    assert d['sampleinterval'] == 4000

@pytest.mark.parametrize('endian', ['little', 'big'])
def test_sans_count_with_interval(header, endian):
    head = header(endian)
    prev = {
        'sampleinterval': 1,
    }
    d = updated_count_interval(head, prev, endian)
    assert d['samples'] == 25
    assert 'sampleinterval' not in d

@pytest.mark.parametrize('endian', ['little', 'big'])
def test_with_count_with_interval(header, endian):
    head = header(endian)
    prev = {
        'samples': 1,
        'sampleinterval': 1,
    }
    d = updated_count_interval(head, prev, endian)
    assert 'samples' not in d
    assert 'sampleinterval' not in d
