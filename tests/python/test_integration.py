import os
import subprocess
import json
from urllib.request import urlopen
from urllib.parse import urljoin
import pytest
import tempfile
import numpy as np
from oneseismic import simple
from urllib.parse import urlsplit
import segyio
import azure
import datetime
from data.create import *

# required
SERVER_URL = os.getenv("SERVER_URL")
STORAGE_LOCATION = os.getenv("STORAGE_LOCATION")
BLOB_URL = os.getenv("BLOB_URL")
UPLOAD_WITH_PYTHON = os.getenv("UPLOAD_WITH_PYTHON")

# optional
UPLOAD_WITH_CLIENT_VERSION = os.getenv("UPLOAD_WITH_CLIENT_VERSION")
FETCH_WITH_CLIENT_VERSION = os.getenv("FETCH_WITH_CLIENT_VERSION")

# azure
AZURE_STORAGE_ACCOUNT_KEY = os.getenv("AZURE_STORAGE_ACCOUNT_KEY")


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


def upload(path, storage_location=STORAGE_LOCATION, scan_meta=None):
    if not scan_meta:
        scan_meta = scan(path)
    scan_insights = tempfile.mktemp('scan_insights.json')
    with open(scan_insights, "w") as f:
        f.write(json.dumps(scan_meta))

    subprocess.run([UPLOAD_WITH_PYTHON, "-m", "oneseismic", "upload",
                   scan_insights, path, storage_location], encoding="utf-8",
                   capture_output=True, check=True)

    return scan_meta["guid"]

@pytest.fixture
def cube_guid(tmpdir_factory):
    custom = str(tmpdir_factory.mktemp('files').join('custom.sgy'))
    create_custom(custom)
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
    create_random(path)

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


def sign_azure_request(
    account_name=urlsplit(STORAGE_LOCATION).netloc.split('.')[0],
    container_name=None,
    account_key=AZURE_STORAGE_ACCOUNT_KEY,
    resource_types=azure.storage.blob.ResourceTypes(
        container=True, object=True),
    permission=None,
    expiry=None,
):
    if not expiry:
        expiry = datetime.datetime.utcnow() + datetime.timedelta(seconds=30)
    if not permission:
        if container_name:
            permission = azure.storage.blob.ContainerSasPermissions(
                read=True, list=True)
        else:
            permission = azure.storage.blob.AccountSasPermissions(
                read=True, write=True, list=True)

    if container_name:
        return azure.storage.blob.generate_account_sas(
            account_name=account_name,
            account_key=account_key,
            resource_types=resource_types,
            permission=permission,
            expiry=expiry)
    else:
        return azure.storage.blob.generate_account_sas(
            account_name=account_name,
            account_key=account_key,
            resource_types=resource_types,
            permission=permission,
            expiry=expiry)


@pytest.mark.parametrize('token', [
    (""),
    ("bad_token"),
])
@pytest.mark.cloud
def test_upload_fails_with_bad_credentials(token, tmpdir):
    """ Tests that cloud resources can't be accessed without good credentials
    Details are considered to be covered by azure itself, as well as any incorrect
    parameters users might provide (e.g. insufficient permissions)
    """
    path = str(tmpdir.join('noupload.segy'))
    data = np.array(
        [
            [1.25, 1.5],
            [random.uniform(2.5, 2.75), random.uniform(2.75, 3)]
        ], dtype=np.float32)
    segyio.tools.from_array(path, data)

    with pytest.raises(Exception) as exc:
        upload(path, storage_location=STORAGE_LOCATION+"?"+token)
    assert "Server failed to authenticate the request" in str(exc.value.stderr)

    client = simple.simple_client(SERVER_URL)
    with pytest.raises(Exception) as exc:
        client.sliceByIndex("whicheverguid", dim=0, index=0)(sas=token).numpy()
    assert "Unauthorized" in str(exc)


@pytest.fixture
def azure_upload():
    guid = ""

    def create_file(path, data):
        nonlocal guid
        segyio.tools.from_array(path, data)
        scan_meta = scan(path)
        guid = scan_meta["guid"]

        return scan_meta

    yield create_file

    # random data, delete
    token = sign_azure_request(
        permission=azure.storage.blob.AccountSasPermissions(delete=True))
    container_client = azure.storage.blob.ContainerClient(
        STORAGE_LOCATION, guid, token)
    container_client.delete_container()


@pytest.mark.cloud
def test_azure_flow(tmpdir, azure_upload):
    path = str(tmpdir.join('upload.segy'))
    data = np.array(
        [
            [1.75, 1.5],
            [random.uniform(2.5, 2.75), random.uniform(2.75, 3)]
        ], dtype=np.float32)

    scan_meta = azure_upload(path, data)
    upload_token = sign_azure_request()
    upload(path, storage_location=STORAGE_LOCATION +
           "?"+upload_token, scan_meta=scan_meta)

    guid = scan_meta["guid"]
    client = simple.simple_client(SERVER_URL)
    token = sign_azure_request(container_name=guid)
    res = client.sliceByIndex(guid, dim=0, index=0)(sas=token).numpy()
    np.testing.assert_array_equal(res, data)


@pytest.mark.cloud
def test_azure_reupload_forbidden(tmpdir, azure_upload):
    """
    Current implementation assumes that once cube has been uploaded to storage,
    it won't be modified any more.

    Test assures that accidental reupload doesn't cause issues (delete followed
    by reupload will, but this is assumed to never happen)
    """
    path = str(tmpdir.join('upload.segy'))
    data = np.array(
        [
            [random.uniform(2.5, 2.75), random.uniform(2.75, 3)]
        ], dtype=np.float32)

    scan_meta = azure_upload(path, data)
    upload_token = sign_azure_request()
    upload(path, storage_location=STORAGE_LOCATION +
           "?"+upload_token, scan_meta=scan_meta)
    with pytest.raises(Exception) as exc:
        upload(path, storage_location=STORAGE_LOCATION +
               "?"+upload_token, scan_meta=scan_meta)
    assert "The specified blob already exists" in str(exc.value.stderr)


# TODO: test error cases
