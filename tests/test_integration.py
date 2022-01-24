import os
import subprocess
import json
from urllib.request import urlopen
from urllib.parse import urljoin
import pytest
import random
import tempfile
import numpy as np
from oneseismic import simple
import segyio

# required
SERVER_URL = os.getenv("SERVER_URL")
STORAGE_LOCATION = os.getenv("STORAGE_LOCATION")
BLOB_URL = os.getenv("BLOB_URL")
UPLOAD_WITH_PYTHON = os.getenv("UPLOAD_WITH_PYTHON")

# optional
UPLOAD_WITH_CLIENT_VERSION = os.getenv("UPLOAD_WITH_CLIENT_VERSION")
FETCH_WITH_CLIENT_VERSION = os.getenv("FETCH_WITH_CLIENT_VERSION")


@pytest.fixture(scope="module", autouse=True)
def assert_environment():
    from .get_version import get_version
    if FETCH_WITH_CLIENT_VERSION:
        print("Fetch with client version "+FETCH_WITH_CLIENT_VERSION)
        assert FETCH_WITH_CLIENT_VERSION in get_version()

    if UPLOAD_WITH_CLIENT_VERSION:
        print("Uploaded with client version "+UPLOAD_WITH_CLIENT_VERSION)
        oneseismic_version = subprocess.run(
            [UPLOAD_WITH_PYTHON, "get_version.py"], encoding="utf-8",
            capture_output=True, check=True)
        assert UPLOAD_WITH_CLIENT_VERSION in oneseismic_version.stdout


def scan(path):
    scan = subprocess.run([UPLOAD_WITH_PYTHON, "-m", "oneseismic",
                          "scan", path], encoding="utf-8", capture_output=True,
                          check=True)
    return json.loads(scan.stdout)


def upload(path, scan_meta=None):
    if not scan_meta:
        scan_meta = scan(path)
    scan_insights = tempfile.mktemp('scan_insights.json')
    with open(scan_insights, "w") as f:
        f.write(json.dumps(scan_meta))

    subprocess.run([UPLOAD_WITH_PYTHON, "-m", "oneseismic", "upload",
                   scan_insights, path,  STORAGE_LOCATION], encoding="utf-8",
                   check=True)
    return scan_meta["guid"]


def create_segy(path):
    """ Create file with data suitable for relevant tests.

    | xlines-ilines | 1             | 2             | 3             |
    |---------------|---------------|---------------|---------------|
    | 10            | 100, 101, 102 | 106, 107, 108 | 112, 113, 114 |
    | 11            | 103, 104, 105 | 109, 110, 111 | 115, 116, 177 |

    UTM coordinates for headers:

    | xlines-ilines | 1           | 2             | 3             |
    |---------------|-------------|---------------|---------------|
    | 10            | x=1, y=3    | x=2.1, y=3    | x=3.2, y=3    |
    | 11            | x=1, y=6.3  | x=2.1, y=6.3  | x=3.2, y=6.3  |
    """
    spec = segyio.spec()

    spec.sorting = 2
    spec.format = 1
    spec.samples = [0, 1, 2]
    spec.ilines = [1, 2, 3]
    spec.xlines = [10, 11]

    # We use scaling constant of -10, meaning that values will be divided by 10
    il_step_x = int(1.1 * 10)
    il_step_y = int(0 * 10)
    xl_step_x = int(0 * 10)
    xl_step_y = int(3.3 * 10)
    ori_x = int(1 * 10)
    ori_y = int(3 * 10)

    with segyio.create(path, spec) as f:
        data = 100
        tr = 0
        for il in spec.ilines:
            for xl in spec.xlines:
                f.header[tr] = {
                    segyio.su.iline: il,
                    segyio.su.xline: xl,
                    segyio.su.cdpx:
                        (il - spec.ilines[0]) * il_step_x +
                        (xl - spec.xlines[0]) * xl_step_x +
                        ori_x,
                    segyio.su.cdpy:
                        (il - spec.ilines[0]) * il_step_y +
                        (xl - spec.xlines[0]) * xl_step_y +
                        ori_y,
                    segyio.su.scalco: -10,
                }
                data = data + len(spec.samples)
                f.trace[tr] = np.arange(start=data - len(spec.samples),
                                        stop=data, step=1, dtype=np.single)
                tr += 1

        f.bin.update(tsort=segyio.TraceSortingFormat.INLINE_SORTING)


@pytest.fixture(scope="module", autouse=True)
def cube_guid(tmpdir_factory):
    custom = str(tmpdir_factory.mktemp('files').join('custom.sgy'))
    create_segy(custom)
    guid = scan(custom)["guid"]
    # no file reupload must happen, so safeguard
    try:
        path = urljoin(BLOB_URL, guid)
        urlopen(path)
    except IOError as e:
        print('File '+path+' is not in the blob yet: ' + str(e) + '. Uploading')
        uploaded_guid = upload(custom)
        assert guid == uploaded_guid
    yield guid


@pytest.mark.local
def test_upload(tmpdir):
    path = str(tmpdir.join('simple.segy'))
    # random number assures random guid
    data = np.array(
        [
            [1.25, 1.5],
            [random.uniform(2.5, 2.75), random.uniform(2.75, 3)]
        ], dtype=np.float32)
    segyio.tools.from_array(path, data)

    guid = upload(path)
    client = simple.simple_client(SERVER_URL)
    res = client.sliceByIndex(guid, dim=0, index=0)().numpy()
    np.testing.assert_array_equal(res[0], np.array([1.25, 1.5]))


@pytest.mark.local
@pytest.mark.version
def test_slice(cube_guid):
    client = simple.simple_client(SERVER_URL)
    res_lineno = client.sliceByLineno(cube_guid, dim=0, lineno=2)().numpy()
    res_index = client.sliceByIndex(cube_guid, dim=0, index=1)().numpy()

    assert len(res_lineno) == 2
    np.testing.assert_array_equal(res_lineno[0], np.array([106, 107, 108]))
    np.testing.assert_array_equal(res_lineno[1], np.array([109, 110, 111]))
    np.testing.assert_array_equal(res_lineno, res_index)


@pytest.mark.local
@pytest.mark.version
def test_curtain(cube_guid):
    client = simple.simple_client(SERVER_URL)
    res_lineno = client.curtainByLineno(
        cube_guid, [[3, 11], [1, 10], [2, 11]])().numpy()
    res_index = client.curtainByIndex(
        cube_guid, [[2, 1], [0, 0], [1, 1]])().numpy()
    res_utm = client.curtainByUTM(
        cube_guid, [[3.2, 6.3], [1, 3], [2.1, 6.3]])().numpy()

    assert len(res_lineno) == 3
    np.testing.assert_array_equal(res_lineno[0], np.array([100, 101, 102]))
    np.testing.assert_array_equal(res_lineno[1], np.array([109, 110, 111]))
    np.testing.assert_array_equal(res_lineno[2], np.array([115, 116, 117]))
    np.testing.assert_array_equal(res_lineno, res_index)
    np.testing.assert_array_equal(res_lineno, res_utm)

# TODO: test error cases
# TODO: test azure: re-upload with same guid
