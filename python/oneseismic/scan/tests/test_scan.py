import segyio
import struct
import sys

from hypothesis import given
from hypothesis.strategies import integers

from ..scanners import LineScanner

def big_endian(i):
    """Convert int to a big-endian integer
    """
    return int.from_bytes(struct.pack('>i', i), byteorder = sys.byteorder)

@given(
    integers(min_value = 1, max_value = 15),
    integers(min_value = 1, max_value = 15),
)
def test_regular_intervals(inlines, crosslines):
    headers = [
        {
            int(segyio.su.iline): big_endian(i),
            int(segyio.su.xline): big_endian(x),
            int(segyio.su.cdpx): 0,
            int(segyio.su.cdpy): 0,
            int(segyio.su.ns): 0,
            int(segyio.su.dt): 0,
        }
        for i in range(1, inlines + 1)
        for x in range(1, crosslines + 1)
    ]

    seg = LineScanner(
            primary = int(segyio.su.iline),
            secondary = int(segyio.su.xline),
            endian = sys.byteorder,
        )

    for header in headers:
        seg.scan_trace_header(header)

    assert len(seg.key2s) == crosslines
