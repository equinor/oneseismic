import os
import json

from ..__main__ import main

expected = {
    'byteorder': 'big',
    'format': 1,
    'samples': 50,
    'sampleinterval': 4.0,
    'byteoffset-first-trace': 3600,
    'guid': 'e939ed45a8258f1fbaf3c21ca83f54369c4c4def79384b4cb0fc069a4eb1a191',
    'dimensions': [[1, 2, 3, 4, 5],
                   [20, 21, 22, 23, 24],
                   [0.0, 4.0, 8.0, 12.0, 16.0, 20.0, 24.0, 28.0, 32.0, 36.0,
                    40.0, 44.0, 48.0, 52.0, 56.0, 60.0, 64.0, 68.0, 72.0, 76.0,
                    80.0, 84.0, 88.0, 92.0, 96.0, 100.0, 104.0, 108.0, 112.0,
                    116.0, 120.0, 124.0, 128.0, 132.0, 136.0, 140.0, 144.0,
                    148.0, 152.0, 156.0, 160.0, 164.0, 168.0, 172.0, 176.0,
                    180.0, 184.0, 188.0, 192.0, 196.0]
                   ]
}

expected_lsb = expected.copy()
expected_lsb.update({
    'byteorder' : 'little',
    'guid': 'f93a0b5102b1891039badf432e92f53df2f30218340617453c5dc23024521973'
})

expected_il5_xl21 = expected.copy()
expected_il5_xl21.update({
    'guid': 'f53a4af4ffa57e1e00b47f1a07ad31263f345cfae44a43a356b9ca0650eae41a'
})

expected_il5_xl21_lsb = expected.copy()
expected_il5_xl21_lsb.update({
    'byteorder' : 'little',
    'guid': '1fedca281874fb195031669019ba7c430c6a3695cf3e48dd92dd28b833150cc9'
})

expected_2byte_keys = expected.copy()
expected_2byte_keys.update({
    'guid': 'ce39724dee2eac8038148f8823c008382815f0c2cf27be055861c226039568ad',
    'dimensions': expected['dimensions'].copy()
})

expected_missing_line_numbers = expected.copy()
expected_missing_line_numbers.update({
    'guid': 'b8aa885d4c3524f3651083a0a27433135f41bf4e1db53a7acdb2548e0d8d6c2c',
    'dimensions': expected['dimensions'].copy()
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

    expected_2byte_keys['dimensions'][0] = [11, 12, 13, 14, 15]
    expected_2byte_keys['dimensions'][1] = [30, 31, 32, 33, 34]

    assert result == expected_2byte_keys

def test_2byte_primary_4byte_secondary():
    s = main([
        '--primary-word', '29',
        '--secondary-word', '193',
        datadir('small-2byte-keys.sgy'),
    ])
    result = json.loads(s)

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

    expected_2byte_keys['dimensions'][0] = [1, 2, 3, 4, 5]
    expected_2byte_keys['dimensions'][1] = [30, 31, 32, 33, 34]

    assert result == expected_2byte_keys

def test_missing_line_numbers():
    s = main([datadir('missing-line-numbers.sgy')])
    result = json.loads(s)

    assert result == expected_missing_line_numbers
