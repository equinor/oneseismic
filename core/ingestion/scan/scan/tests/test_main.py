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
