import math
import numpy as np
import pytest
from random import sample
import segyio
import struct
import sys

from hypothesis import given
from hypothesis.strategies import integers, floats, one_of

from ..scanners import LineScanner, GeoScanner

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

@given(
    integers(min_value = 2, max_value = 15),
    integers(min_value = 2, max_value = 15),
    one_of(
        floats(min_value = -10, max_value = -1),
        floats(min_value = 1, max_value = 10)
    ),
    one_of(
        floats(min_value = -10, max_value = -1),
        floats(min_value = 1, max_value = 10)
    ),
    floats(min_value = -math.pi, max_value = math.pi),
    floats(min_value = -10, max_value = 10),
    floats(min_value = -10, max_value = 10)
)
def test_geo_scanner(
        inlines,
        crosslines,
        ilinc,
        xlinc,
        rotation,
        offsetx,
        offsety
):
    """
    inlines, crosslines
        Number of inlines and crosslines
    ilinc, xlinc
        The distance between adjacent inlines and crosslines (these define the
        scaling and reflection of the transformation between the line number
        grid and the UTM grid)
    rotation
        A rotation that will be added so that cube is not perfectly aligned with
        the x and y axis. This rotation is used to compute the x and y
        components of the in- and crossline positional increment vectors
    offsetx, offsety
        The x and y component of the (iline_min, xline_min) trace (these define
        the translation of the transformation between the line number grid and
        the UTM grid)

    We align the inline with the x-axis and crossline with the y-axis and add
    a rotation. Before adding rotation we have:

       x = inline * ilinc, y = crossline * xlinc

    Applying the rotation matrix:

       cos(rot) -sin(rot)  *  inline * ilinc
       sin(rot) cos(rot)      crossline * xlinc

       =  inline * ilinc * cos(rot) - crossline * xlinc * sin(rot)
          inline * ilinc * sin(rot) + crossline * xlinc * cos(rot)

       =  inline * ilincx + crossline * xlincx
          inline * ilincy + crossline * xlincy

    Including the initial offset at (inline, crossline) = (0, 0) we get:

       x = inline * ilincx + crossline * xlincx + offsetx,
       y = inline * ilincy + crossline * xlincy + offsety

    """
    ilincx = ilinc * math.cos(rotation)
    ilincy = ilinc * math.sin(rotation)
    xlincx = - xlinc * math.sin(rotation)
    xlincy = xlinc * math.cos(rotation)

    headers = [
        {
            int(segyio.su.iline): big_endian(i),
            int(segyio.su.xline): big_endian(x),
            int(segyio.su.cdpx):
                i * ilincx + x * xlincx + offsetx,
            int(segyio.su.cdpy):
                i * ilincy + x * xlincy + offsety,
            int(segyio.su.ns): 0,
            int(segyio.su.dt): 0,
            int(segyio.su.scalco): 1,
        }
        for i in range(inlines + 1)
        for x in range(crosslines + 1)
    ]

    seg = GeoScanner(
        il_word=int(segyio.su.iline),
        xl_word=int(segyio.su.xline),
        endian=sys.byteorder,
    )

    for header in headers:
        seg.scan_trace_header(header)

    result = seg.report()

    utm_to_lineno = np.array(result['utm-to-lineno'])

    for header in sample(headers, 3):
        linenos = np.array([
            big_endian(header[int(segyio.su.iline)]),
            big_endian(header[int(segyio.su.xline)])
        ])
        cdps = np.array([
            header[int(segyio.su.cdpx)],
            header[int(segyio.su.cdpy)],
            1
        ])
        estimated_linenos = utm_to_lineno.dot(cdps)

        np.testing.assert_allclose(estimated_linenos, linenos, atol=1e-9)


@given(
    integers(min_value = 2, max_value = 15),
    integers(min_value = 2, max_value = 15),
    one_of(
        floats(min_value = -10, max_value = -1),
        floats(min_value = 1, max_value = 10)
    ),
    one_of(
        floats(min_value = -10, max_value = -1),
        floats(min_value = 1, max_value = 10)
    ),
    floats(min_value = -math.pi, max_value = math.pi),
    one_of(
        floats(min_value = -math.pi / 2, max_value = -1e-3),
        floats(min_value = 1e-3, max_value = math.pi / 2)
    ),
    floats(min_value = -10, max_value = 10),
    floats(min_value = -10, max_value = 10)
)
def test_geo_scanner_lines_not_perpendicular(
        inlines,
        crosslines,
        ilinc,
        xlinc,
        rotation,
        deviation,
        offsetx,
        offsety
):
    """
    deviation
        A rotation added to the crossline direction to make it non-pependicular
        to the inline direction

    See test_geo_scanner for a description of the other parameters
    """
    ilincx = ilinc * math.cos(rotation)
    ilincy = ilinc * math.sin(rotation)
    xlincx = - xlinc * math.sin(rotation + deviation)
    xlincy = xlinc * math.cos(rotation + deviation)

    headers = [
        {
            int(segyio.su.iline): big_endian(i),
            int(segyio.su.xline): big_endian(x),
            int(segyio.su.cdpx):
                i * ilincx + x * xlincx + offsetx,
            int(segyio.su.cdpy):
                i * ilincy + x * xlincy + offsety,
            int(segyio.su.ns): 0,
            int(segyio.su.dt): 0,
            int(segyio.su.scalco): 1,
        }
        for i in range(inlines + 1)
        for x in range(crosslines + 1)
    ]

    seg = GeoScanner(
        il_word=int(segyio.su.iline),
        xl_word=int(segyio.su.xline),
        endian=sys.byteorder,
    )

    for header in headers:
        seg.scan_trace_header(header)

    result = seg.report()

    # Scanner should return an empty dict when assumption of perpendicular lines
    # doesn't hold
    assert result == {}


@given(
    integers(min_value = 4, max_value = 15),
    integers(min_value = 4, max_value = 15),
    one_of(
        floats(min_value = -10, max_value = -1),
        floats(min_value = 1, max_value = 10)
    ),
    one_of(
        floats(min_value = -10, max_value = -1),
        floats(min_value = 1, max_value = 10)
    ),
    floats(min_value = -math.pi, max_value = math.pi),
    floats(min_value = -10, max_value = 10),
    floats(min_value = -10, max_value = 10)
)
def test_geo_scanner_out_of_order_and_missing_lines(
        inlines,
        crosslines,
        ilinc,
        xlinc,
        rotation,
        offsetx,
        offsety
):
    """
    See test_geo_scanner for a description of the other parameters
    """
    ilincx = ilinc * math.cos(rotation)
    ilincy = ilinc * math.sin(rotation)
    xlincx = - xlinc * math.sin(rotation)
    xlincy = xlinc * math.cos(rotation)

    headers = [
        {
            int(segyio.su.iline): big_endian(i),
            int(segyio.su.xline): big_endian(x),
            int(segyio.su.cdpx):
                i * ilincx + x * xlincx + offsetx,
            int(segyio.su.cdpy):
                i * ilincy + x * xlincy + offsety,
            int(segyio.su.ns): 0,
            int(segyio.su.dt): 0,
            int(segyio.su.scalco): 1,
        }
        for i in range(inlines + 1)
        for x in range(crosslines + 1)
    ]

    seg = GeoScanner(
        il_word=int(segyio.su.iline),
        xl_word=int(segyio.su.xline),
        endian=sys.byteorder,
    )

    samples = int(len(headers) / 2) + 1
    for header in sample(headers, samples):
        seg.scan_trace_header(header)

    result = seg.report()

    utm_to_lineno = np.array(result['utm-to-lineno'])

    for header in sample(headers, 3):
        linenos = np.array([
            big_endian(header[int(segyio.su.iline)]),
            big_endian(header[int(segyio.su.xline)])
        ])
        cdps = np.array([
            header[int(segyio.su.cdpx)],
            header[int(segyio.su.cdpy)],
            1
        ])
        estimated_linenos = utm_to_lineno.dot(cdps)

        np.testing.assert_allclose(estimated_linenos, linenos, atol=1e-9)


def test_geo_scanner_corners_are_selected():
    headers = [
        {
            int(segyio.su.iline): big_endian(2),
            int(segyio.su.xline): big_endian(20),
            int(segyio.su.cdpx): 43.464101615137764,
            int(segyio.su.cdpy): -29.96152422706632,
            int(segyio.su.scalco): 1,
        },
        {
            int(segyio.su.iline): big_endian(7),
            int(segyio.su.xline): big_endian(25),
            int(segyio.su.cdpx): 59.62435565298215,
            int(segyio.su.cdpy): -37.9519052838329,
            int(segyio.su.scalco): 1,
        },
        # Point is slightly "off". Should not be selected since it is not
        # closest to any of the corners
        {
            int(segyio.su.iline): big_endian(3),
            int(segyio.su.xline): big_endian(22),
            int(segyio.su.cdpx): 48.5,
            int(segyio.su.cdpy): -34.3,
            int(segyio.su.scalco): 1,
        },
        {
            int(segyio.su.iline): big_endian(3),
            int(segyio.su.xline): big_endian(23),
            int(segyio.su.cdpx): 49.69615242270664,
            int(segyio.su.cdpy): -36.75575286112627,
            int(segyio.su.scalco): 1,
        },
        {
            int(segyio.su.iline): big_endian(6),
            int(segyio.su.xline): big_endian(21),
            int(segyio.su.cdpx): 51.89230484541328,
            int(segyio.su.cdpy): -28.559600438419636,
            int(segyio.su.scalco): 1,
        },
    ]

    seg = GeoScanner(
        il_word=int(segyio.su.iline),
        xl_word=int(segyio.su.xline),
        endian=sys.byteorder,
    )

    for header in headers:
        seg.scan_trace_header(header)

    result = seg.report()

    utm_to_lineno = np.array(result['utm-to-lineno'])

    np.testing.assert_allclose(
        utm_to_lineno.dot(np.array([59.62435565298215, -37.9519052838329, 1])),
        np.array([7, 25])
    )


def test_geo_scanner_linearly_dependent_points():
    # The traces are all along a line. This should trigger an exception when
    # inverting the matrix M in GeoScanner.report() so an empty dict should be
    # returned.
    headers = [
        {
            int(segyio.su.iline): big_endian(1),
            int(segyio.su.xline): big_endian(10),
            int(segyio.su.cdpx): 2,
            int(segyio.su.cdpy): 20,
            int(segyio.su.scalco): 1,
        },
        {
            int(segyio.su.iline): big_endian(2),
            int(segyio.su.xline): big_endian(11),
            int(segyio.su.cdpx): 3,
            int(segyio.su.cdpy): 21,
            int(segyio.su.scalco): 1,
        },
        {
            int(segyio.su.iline): big_endian(2),
            int(segyio.su.xline): big_endian(12),
            int(segyio.su.cdpx): 4,
            int(segyio.su.cdpy): 22,
            int(segyio.su.scalco): 1,
        },
    ]

    seg = GeoScanner(
        il_word=int(segyio.su.iline),
        xl_word=int(segyio.su.xline),
        endian=sys.byteorder,
    )

    for header in headers:
        seg.scan_trace_header(header)

    result = seg.report()

    assert result == {}


def test_geo_scanner_all_zero_cdp():
    # All zero cdp headers should result in an empty dict being returned
    headers = [
        {
            int(segyio.su.iline): big_endian(1),
            int(segyio.su.xline): big_endian(10),
            int(segyio.su.cdpx): 0,
            int(segyio.su.cdpy): 0,
            int(segyio.su.scalco): 1,
        },
        {
            int(segyio.su.iline): big_endian(4),
            int(segyio.su.xline): big_endian(10),
            int(segyio.su.cdpx): 0,
            int(segyio.su.cdpy): 0,
            int(segyio.su.scalco): 1,
        },
        {
            int(segyio.su.iline): big_endian(1),
            int(segyio.su.xline): big_endian(13),
            int(segyio.su.cdpx): 0,
            int(segyio.su.cdpy): 0,
            int(segyio.su.scalco): 1,
        },
    ]

    seg = GeoScanner(
        il_word=int(segyio.su.iline),
        xl_word=int(segyio.su.xline),
        endian=sys.byteorder,
    )

    for header in headers:
        seg.scan_trace_header(header)

    result = seg.report()

    assert result == {}


@pytest.mark.parametrize('scale, expected_cdpx, expected_cdpy',[
    (10, 10, 20),
    (-10, 0.1, 0.2),
    (0, 1, 2),
])
def test_geo_scanner_scaled_cdp(scale, expected_cdpx, expected_cdpy):
    header = {
            int(segyio.su.iline): big_endian(10),
            int(segyio.su.xline): big_endian(20),
            int(segyio.su.cdpx): 1,
            int(segyio.su.cdpy): 2,
            int(segyio.su.ns): 0,
            int(segyio.su.dt): 0,
            int(segyio.su.scalco): scale,
    }

    seg = GeoScanner(
        il_word=int(segyio.su.iline),
        xl_word=int(segyio.su.xline),
        endian=sys.byteorder,
    )

    seg.scan_trace_header(header)

    np.testing.assert_allclose(seg.p0, [expected_cdpx, expected_cdpy])
