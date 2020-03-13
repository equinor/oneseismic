import pytest

import oneseismic.geometry as geo

def test_valid_cube_shape():
    cs = geo.CS3(64,64,64)
    cs = geo.CS3([64,64,64])
    cs = geo.CS3((64,64,64))
    with pytest.raises(TypeError):
        cs = geo.CS3(-64,64,64)
        cs = geo.CS3([64,-64,64])
        cs = geo.CS3((64,64,-64))

def test_valid_fragment_shape():
    fs = geo.FS3(8,8,8)
    fs = geo.FS3([8,8,8])
    fs = geo.FS3((8,8,8))
    with pytest.raises(TypeError):
        fs = geo.FS3(-8,8,8)
        fs = geo.FS3([8,-8,8])
        fs = geo.FS3((8,8,-8))

def test_valid_gvt():
    gvt = geo.GVT3(geo.CS3(64,64,64), geo.FS3(8,8,8))

def test_valid_fragment_shape():
    gvt = geo.GVT3(geo.CS3(9,15,23), geo.FS3(3,9,5))
    assert gvt.fragment_shape()[0] == 3
    assert gvt.fragment_shape()[1] == 9
    assert gvt.fragment_shape()[2] == 5
    with pytest.raises(IndexError):
        assert gvt.fragment_shape()[3] == 6

def test_gvt_fragment_count():
    gvt = geo.GVT3(geo.CS3(9,15,23), geo.FS3(3,9,5))
    assert gvt.fragment_count(0) == 3
    assert gvt.fragment_count(1) == 2
    assert gvt.fragment_count(2) == 5
    with pytest.raises(ValueError):
        assert gvt.fragment_count(3) == 0

def dstshape(gvt):
    dstshape = (
        gvt.fragment_count(0) * gvt.fragment_shape()[0],
        gvt.fragment_count(1) * gvt.fragment_shape()[1],
        gvt.fragment_count(2) * gvt.fragment_shape()[2],
    )
    return dstshape

def test_dstshape():

    expected = (5,5,8)
    gvt = geo.GVT3(geo.CS3(3,5,6), geo.FS3(5,5,4))
    assert dstshape(gvt) == expected

    expected = (16,8,16)
    gvt = geo.GVT3(geo.CS3(14,5,13), geo.FS3(8,8,8))
    assert dstshape(gvt) == expected

    expected = (64,64,64)
    gvt = geo.GVT3(geo.CS3(64,64,64), geo.FS3(8,8,8))
    assert dstshape(gvt) == expected
