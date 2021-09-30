import os
import json

from ..__main__ import main

expected = {
    'byteorder': 'big',
    'format': 1,
    'samples': 50,
    'sampleinterval': 4000,
    'sample-value-min' : 1.2100000381469727,
    'sample-value-max' : 5.240489959716797,
    'byteoffset-first-trace': 3600,
    'guid': '86f5f8f783fabe2773531d5529226d37b6c9bdcf',
    'dimensions': [[1, 2, 3, 4, 5],
                   [20, 21, 22, 23, 24],
                   [     0,    4000,    8000,  12000,  16000,  20000,  24000,
                     28000,   32000,   36000,  40000,  44000,  48000,  52000,
                     56000,   60000,   64000,  68000,  72000,  76000,  80000,
                     84000,   88000,   92000,  96000, 100000, 104000, 108000,
                    112000,  116000,  120000, 124000, 128000, 132000, 136000,
                    140000,  144000,  148000, 152000, 156000, 160000, 164000,
                    168000,  172000,  176000, 180000, 184000, 188000, 192000,
                    196000]
                   ],
    'key1-last-trace': {
        '1': 4,
        '2': 9,
        '3': 14,
        '4': 19,
        '5': 24,
    },
    'key-words': [189, 193],
}

expected_lsb = expected.copy()
expected_lsb.update({
    'byteorder' : 'little',
    'guid': '2efb1a0bebb577b27a9e0503b06cf0935eebca78'
})

expected_il5_xl21 = expected.copy()
expected_il5_xl21.update({
    'guid': '35368c1a2aec523c324ae1fd1fb42f1994f46fbe',
    'key-words': [5, 21],
    'sample-value-min' : 0.037812501192092896,
    'sample-value-max' : 0.6550612449645996
})

expected_il5_xl21_lsb = expected_il5_xl21.copy()
expected_il5_xl21_lsb.update({
    'byteorder' : 'little',
    'guid': 'd3f1e9fa8b1ebaae26c55fa3e9beba0b4fe57287',
})

expected_2byte_keys = expected.copy()
expected_2byte_keys.update({
    'guid': '526026c67842afbb37ac99e81371550258554665',
    'dimensions': expected['dimensions'].copy(),
})

expected_missing_line_numbers = expected.copy()
expected_missing_line_numbers.update({
    'guid': 'b9bff5837e3f654585cc8a730fedb9730309e4ee',
    'dimensions': expected['dimensions'].copy(),
    'key1-last-trace': {
        '1': 4,
        '2': 9,
        '3': 14,
        '5': 19,
        '6': 24,
    },
})
expected_missing_line_numbers['dimensions'][0] = [1, 2, 3, 5, 6]
expected_missing_line_numbers['dimensions'][1] = [20, 21, 23, 24, 25]

def datadir(filename):
    root = os.path.dirname(__file__)
    data = os.path.join(root, 'data')
    return os.path.join(data, filename)

def test_small_little_endian():
    s = main(['--little-endian', datadir('small-lsb.sgy')])
    run = json.loads(s)

    assert run == expected_lsb

def test_small_big_endian():
    s = main(['--big-endian', datadir('small.sgy')])
    run = json.loads(s)

    assert run == expected

def test_small_big_endian_default():
    s = main([datadir('small.sgy')])
    run = json.loads(s)

    assert run == expected

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

    assert run == expected_il5_xl21

def test_small_little_endian_custom_inline_crossline_offset():
    s = main([
        '--primary-word', '5',
        '--secondary-word', '21',
        '--little-endian',
        datadir('small-iline-5-xline-21-lsb.sgy'),
    ])
    run = json.loads(s)

    assert run == expected_il5_xl21_lsb

def test_2byte_primary_2byte_secondary():
    s = main([
        '--primary-word', '29',
        '--secondary-word', '71',
        datadir('small-2byte-keys.sgy'),
    ])
    result = json.loads(s)

    expected_2byte_keys['key-words'] = [29, 71]
    expected_2byte_keys['dimensions'][0] = [11, 12, 13, 14, 15]
    expected_2byte_keys['dimensions'][1] = [30, 31, 32, 33, 34]
    expected_2byte_keys['key1-last-trace'] = {
        '11': 4,
        '12': 9,
        '13': 14,
        '14': 19,
        '15': 24,
    }
    assert result == expected_2byte_keys

def test_2byte_primary_4byte_secondary():
    s = main([
        '--primary-word', '29',
        '--secondary-word', '193',
        datadir('small-2byte-keys.sgy'),
    ])
    result = json.loads(s)

    expected_2byte_keys['key-words'] = [29, 193]
    expected_2byte_keys['dimensions'][0] = [11, 12, 13, 14, 15]
    expected_2byte_keys['dimensions'][1] = [20, 21, 22, 23, 24]

    assert result == expected_2byte_keys

def test_4byte_primary_2byte_secondary():
    s = main([
        '--primary-word', '189',
        '--secondary-word', '71',
        datadir('small-2byte-keys.sgy'),
    ])
    result = json.loads(s)

    expected_2byte_keys['key-words'] = [189, 71]
    expected_2byte_keys['dimensions'][0] = [1, 2, 3, 4, 5]
    expected_2byte_keys['dimensions'][1] = [30, 31, 32, 33, 34]
    expected_2byte_keys['key1-last-trace'] = {
        '1': 4,
        '2': 9,
        '3': 14,
        '4': 19,
        '5': 24,
    }
    assert result == expected_2byte_keys

def test_missing_line_numbers():
    s = main([datadir('missing-line-numbers.sgy')])
    result = json.loads(s)
    assert result == expected_missing_line_numbers

def test_outline_missing_line_numbers():
    s = main([
        datadir('missing-line-numbers.sgy'),
    ])
    result = json.loads(s)

    expected_line_numbers = [
        [1, 2, 3, 5, 6],
        [20, 21, 23, 24, 25],
    ]
    assert result['dimensions'][:2] == expected_line_numbers
