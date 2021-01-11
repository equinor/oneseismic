import io
import segyio
import struct

import pytest

from ..scan import scanner

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
    s = scanner(endian = endian)
    s.scan_first_header(head)
    d = s.report()
    assert d['samples'] == 25
    assert d['sampleinterval'] == 4000

@pytest.mark.parametrize('endian', ['little', 'big'])
def test_with_count_sans_interval(header, endian):
    head = header(endian)
    s = scanner(endian = endian)
    s.observed['samples'] = 1
    s.scan_first_header(head)
    d = s.report()
    assert d['samples'] == 1
    assert d['sampleinterval'] == 4000

@pytest.mark.parametrize('endian', ['little', 'big'])
def test_sans_count_with_interval(header, endian):
    head = header(endian)
    s = scanner(endian = endian)
    s.observed['sampleinterval'] = 1
    s.scan_first_header(head)
    d = s.report()
    assert d['samples'] == 25
    assert d['sampleinterval'] == 1

@pytest.mark.parametrize('endian', ['little', 'big'])
def test_with_count_with_interval(header, endian):
    head = header(endian)
    s = scanner(endian = endian)
    s.observed['samples'] = 1
    s.observed['sampleinterval'] = 1
    s.scan_first_header(head)
    d = s.report()

    assert d['samples'] == 1
    assert d['sampleinterval'] == 1
