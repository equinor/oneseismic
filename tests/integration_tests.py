import os
from urllib.parse import parse_qs, urlparse

from hypothesis import given, settings, strategies as st
import json
import numpy as np
import pytest
import requests
import segyio
import tempfile
from azure.core.credentials import AccessToken
from azure.core.exceptions import ResourceNotFoundError
from azure.storage.blob import BlobServiceClient
from oneseismic import scan
from oneseismic import upload
from oneseismic import client

API_ADDR = os.getenv("API_ADDR", "http://localhost:8080")
AUTHSERVER = os.getenv("AUTHSERVER", "http://localhost:8089")
AUDIENCE = os.getenv("AUDIENCE")
STORAGE_URL = os.getenv("STORAGE_URL")


class CustomTokenCredential(object):
    def get_token(self, *scopes, **kwargs):
        r = requests.post(AUTHSERVER + "/oauth2/v2.0/token")
        access_token = r.json()["access_token"]
        return AccessToken(access_token, 1)


def auth_header():
    r = requests.get(
        AUTHSERVER + "/oauth2/v2.0/authorize" + "?client_id=" + AUDIENCE,
        headers={"content-type": "application/json"},
        allow_redirects=False,
    )
    token = parse_qs(urlparse(r.headers["location"]).fragment)["access_token"]

    return {"Authorization": f"Bearer {token[0]}"}


class client_auth:
    def __init__(self, auth):
        self.auth = auth

    def token(self):
        return dict(self.auth)


AUTH_HEADER = auth_header()
AUTH_CLIENT = client_auth(auth_header())


def upload_cube(data):
    """ create segy of data and upload to azure blob

    return: guid of cube
    """
    fname = tempfile.mktemp("segy")
    segyio.tools.from_array(fname, data)

    with open(fname, "rb") as f:
        meta = scan.scan(f)

    credential = CustomTokenCredential()
    blob_service_client = BlobServiceClient(STORAGE_URL, credential)

    try:
        blob_service_client.delete_container("results")
    except ResourceNotFoundError as error:
        pass
    try:
        blob_service_client.delete_container(meta["guid"])
    except ResourceNotFoundError as error:
        pass

    blob_service_client.create_container("results")

    shape = [64, 64, 64]
    params = {"subcube-dims": shape}
    with open(fname, "rb") as f:
        upload.upload(params, meta, f, blob_service_client)

    return meta["guid"]


@pytest.fixture(scope="session")
def cube():
    """ Generate and upload simplest cube, no specific data needed

    return: guid of cube
    """
    data = np.ndarray(shape=(2, 2, 2), dtype=np.float32)

    return upload_cube(data)


def test_cube_404(cube):
    c = client.client(API_ADDR, AUTH_CLIENT)
    with pytest.raises(RuntimeError) as e:
        c.cube("not_found").slice(0, 1)
    assert "404" in str(e.value)


def test_invalid_token_should_fail(cube):
    with open('invalid_tokens.json') as f:
        tokens = json.load(f)

    for t in tokens:
        token = {"Authorization": f"Bearer {t['jwt']}"}
        auth = client_auth(token)
        c = client.client(API_ADDR, auth)

        with pytest.raises(RuntimeError) as e:
            c.cube(cube).slice(0, 1)

        assert "401" in str(e.value) or "500" in str(e.value)


@settings(deadline=None, max_examples=6)
@given(
    w=st.integers(min_value=2, max_value=200),
    h=st.integers(min_value=2, max_value=200),
    d=st.integers(min_value=2, max_value=200),
)
def test_slices(w, h, d):
    data = np.ndarray(shape=(w, h, d), dtype=np.float32)
    for i in range(w):
        for j in range(h):
            for k in range(d):
                data[i, j, k] = (i * 1) + (j * 1000) + (k * 1000000)

    guid = upload_cube(data)

    c = client.client(API_ADDR, AUTH_CLIENT)
    cube = c.cube(guid)

    # test end slices
    np.testing.assert_almost_equal(cube.slice(0, 1), data[0, :, :])
    np.testing.assert_almost_equal(cube.slice(1, 1), data[:, 0, :])
    np.testing.assert_almost_equal(cube.slice(2, 0), data[:, :, 0])
    np.testing.assert_almost_equal(cube.slice(0, w), data[w - 1, :, :])
    np.testing.assert_almost_equal(cube.slice(1, h), data[:, h - 1, :])
    np.testing.assert_almost_equal(
        cube.slice(2, (d - 1) * 4000),
        data[:, :, d - 1]
    )

    # test end slices between the two first segments
    if w > 64:
        np.testing.assert_almost_equal(cube.slice(0, 64), data[63, :, :])
        np.testing.assert_almost_equal(cube.slice(0, 65), data[64, :, :])
    if h > 64:
        np.testing.assert_almost_equal(cube.slice(1, 64), data[:, 63, :])
        np.testing.assert_almost_equal(cube.slice(1, 65), data[:, 64, :])
    if d > 64:
        np.testing.assert_almost_equal(cube.slice(2, 63 * 4000), data[:, :, 63])
        np.testing.assert_almost_equal(cube.slice(2, 64 * 4000), data[:, :, 64])
