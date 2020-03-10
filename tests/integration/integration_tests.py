import os
import pytest

from urllib.parse import parse_qs, urlparse
from azure.storage.blob import (
    BlobServiceClient,
    BlobClient,
    ContainerClient,
)

import requests

protocol = "DefaultEndpointsProtocol=http;"
account_name = "AccountName={};".format(os.getenv("AZURE_STORAGE_ACCOUNT"))
account_key = "AccountKey={};".format(os.getenv("AZURE_STORAGE_ACCESS_KEY"))
uri = os.getenv("AZURE_STORAGE_URL") % os.getenv("AZURE_STORAGE_ACCOUNT")
blob_endpoint = "BlobEndpoint={};".format(uri)


az_connection_str = protocol + account_name + account_key + blob_endpoint
uri = os.getenv("HOST_ADDR", "http://localhost:8080")
auth_uri = os.getenv("AUTH_ADDR", "http://localhost:8089")

r = requests.get(auth_uri + "/oauth2/authorize", allow_redirects=False)
assert r.status_code == 302

token = parse_qs(urlparse(r.headers["Location"]).fragment)["access_token"][0]

auth_header = {"Authorization": f"Bearer {token}"}


def _delete_all_containers(blob_service_client):
    containers = blob_service_client.list_containers()
    for c in containers:
        blob_service_client.delete_container(c)


container_names = ["a", "b", "c"]


@pytest.fixture
def create_containers():
    blob_service_client = BlobServiceClient.from_connection_string(az_connection_str)
    _delete_all_containers(blob_service_client)

    for name in container_names:
        blob_service_client.create_container(name)
    yield
    for name in container_names:
        blob_service_client.delete_container(name)


def test_no_auth(create_containers):
    r = requests.get(uri)
    assert r.status_code == 401


def test_auth(create_containers):
    r = requests.get(uri, headers=auth_header)
    assert r.status_code == 200
    assert set(r.json()).difference(set(container_names)) == set()
