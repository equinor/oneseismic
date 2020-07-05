import os
import pytest
import io
import json

from urllib.parse import parse_qs, urlparse
from azure.storage.blob import BlobServiceClient
from azure.core.exceptions import ResourceExistsError
import requests


HOST_ADDR = os.getenv("HOST_ADDR", "http://localhost:8080")
AUTH_ADDR = os.getenv("AUTH_ADDR", "http://localhost:8089")

CUBE_NAMES = ["a"]
DIM1 = [10, 20, 30]
MANIFEST = {
    "byteoffset-first-trace": 0,
    "byteorder": "",
    "dimensions": [[0, 1], DIM1, [100, 200, 300, 400]],
    "format": "",
    "guid": CUBE_NAMES[0],
    "sampleinterval": 0,
    "samples": 0,
}


def az_storage():
    protocol = "DefaultEndpointsProtocol=http;"
    account_name = "AccountName={};".format(os.getenv("AZURE_STORAGE_ACCOUNT"))
    account_key = "AccountKey={};".format(os.getenv("AZURE_STORAGE_ACCESS_KEY"))
    uri = os.getenv("AZURE_STORAGE_URL").format(os.getenv("AZURE_STORAGE_ACCOUNT"))
    blob_endpoint = "BlobEndpoint={};".format(uri)

    return protocol + account_name + account_key + blob_endpoint


def auth_header():
    r = requests.get(AUTH_ADDR + "/oauth2/authorize?client_id="+os.getenv("AUDIENCE"), allow_redirects=False)
    token = parse_qs(urlparse(r.headers["Location"]).fragment)["access_token"][0]

    return {"Authorization": f"Bearer {token}"}


AUTH_HEADER = auth_header()


@pytest.fixture(scope="session")
def create_cubes():
    blob_service_client = BlobServiceClient.from_connection_string(az_storage())

    for name in CUBE_NAMES:
        try:
            blob_service_client.create_container(name)
        except ResourceExistsError:
            None
        blob_client = blob_service_client.get_blob_client(
            container=name, blob="manifest.json"
        )

        blob_client.upload_blob(json.dumps(MANIFEST).encode())


def test_no_auth():
    r = requests.get(HOST_ADDR)
    assert r.status_code == 401


def test_auth():
    r = requests.get(HOST_ADDR, headers=AUTH_HEADER)
    assert r.status_code == 200


def test_list_cubes(create_cubes):
    r = requests.get(HOST_ADDR, headers=AUTH_HEADER)
    assert r.status_code == 200
    assert r.json() == ["a"]


def test_services(create_cubes):
    r = requests.get(HOST_ADDR + "/a", headers=AUTH_HEADER)
    assert r.status_code == 200
    assert r.json() == ["slice"]


def test_cube_404(create_cubes):
    r = requests.get(HOST_ADDR + "/b", headers=AUTH_HEADER)
    assert r.status_code == 404


def test_dimensions(create_cubes):
    r = requests.get(HOST_ADDR + "/a/slice", headers=AUTH_HEADER)
    assert r.status_code == 200
    assert r.json() == [0, 1, 2]


def test_lines(create_cubes):
    r = requests.get(HOST_ADDR + "/a/slice/1", headers=AUTH_HEADER)
    assert r.status_code == 200
    assert r.json() == DIM1


def test_lines_404(create_cubes):
    r = requests.get(HOST_ADDR + "/a/slice/3", headers=AUTH_HEADER)
    assert r.status_code == 404


@pytest.mark.skip(reason="TODO")
def test_slice(create_cubes):
    r = requests.get(HOST_ADDR + "/a/slice/1/10", headers=AUTH_HEADER)
    assert r.status_code == 200
    assert r.json()["tiles"] != None
