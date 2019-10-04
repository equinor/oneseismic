import os
import json

from ..__main__ import main

def datadir(filename):
    root = os.path.dirname(__file__)
    data = os.path.join(root, 'data')
    return os.path.join(data, filename)

def test_small_little_endian():
    s = main(['--little-endian', datadir('small-lsb.sgy')])
    run = json.loads(s)
    with open(datadir('small-lsb.json')) as f:
        ref = json.load(f)

    assert run == ref

def test_small_big_endian():
    s = main(['--big-endian', datadir('small.sgy')])
    run = json.loads(s)
    with open(datadir('small.json')) as f:
        ref = json.load(f)

    assert run == ref

def test_small_big_endian_default():
    s = main([datadir('small.sgy')])
    run = json.loads(s)
    with open(datadir('small.json')) as f:
        ref = json.load(f)

    assert run == ref

def test_small_pretty_does_not_break_output():
    ugly = main([datadir('small.sgy')])
    pretty = main(['--pretty', datadir('small.sgy')])

    ugly = json.loads(ugly)
    pretty = json.loads(pretty)

    assert ugly == pretty

def test_small_big_endian_custom_inline_crossline_offset():
    s = main([
        '--primary-word', '5',
        '--secondary-word', '21',
        datadir('small-iline-5-xline-21.sgy'),
    ])
    run = json.loads(s)
    with open(datadir('small-iline-5-xline-21.json')) as f:
        ref = json.load(f)

    assert run == ref

def test_small_little_endian_custom_inline_crossline_offset():
    s = main([
        '--primary-word', '5',
        '--secondary-word', '21',
        '--little-endian',
        datadir('small-iline-5-xline-21-lsb.sgy'),
    ])
    run = json.loads(s)
    with open(datadir('small-iline-5-xline-21-lsb.json')) as f:
        ref = json.load(f)

    assert run == ref

def test_2byte_primary_2byte_secondary():
    s = main([
        '--primary-word', '29',
        '--secondary-word', '71',
        datadir('small-2byte-keys.sgy'),
    ])
    result = json.loads(s)

    assert result["primaryKey"] == [29, "TwoByte"]
    assert result["secondaryKey"] == [71, "TwoByte"]

    primaries = [k['primaryKey'] for k in result['segmentInfo']]
    assert primaries == [11, 12, 13, 14, 15]

    for s in result['segmentInfo']:
        assert s['binInfoStart']['crosslineNumber'] == 30
        assert s['binInfoStop']['crosslineNumber'] == 34

def test_2byte_primary_4byte_secondary():
    s = main([
        '--primary-word', '29',
        '--secondary-word', '193',
        datadir('small-2byte-keys.sgy'),
    ])
    result = json.loads(s)

    assert result["primaryKey"] == [29, "TwoByte"]
    assert result["secondaryKey"] == [193, "FourByte"]

    primaries = [k['primaryKey'] for k in result['segmentInfo']]
    assert primaries == [11, 12, 13, 14, 15]

    for s in result['segmentInfo']:
        assert s['binInfoStart']['crosslineNumber'] == 20
        assert s['binInfoStop']['crosslineNumber'] == 24

def test_4byte_primary_2byte_secondary():
    s = main([
        '--primary-word', '189',
        '--secondary-word', '71',
        datadir('small-2byte-keys.sgy'),
    ])
    result = json.loads(s)

    assert result["primaryKey"] == [189, "FourByte"]
    assert result["secondaryKey"] == [71, "TwoByte"]

    primaries = [k['primaryKey'] for k in result['segmentInfo']]
    assert primaries == [1, 2, 3, 4, 5]

    for s in result['segmentInfo']:
        assert s['binInfoStart']['crosslineNumber'] == 30
        assert s['binInfoStop']['crosslineNumber'] == 34
