import os
from pathlib import Path

import pytest
from azure.storage.blob import BlobServiceClient
from upload import upload

from blobio import BlobIO
from scan import scan

host = os.getenv("AZURITE", "localhost")

connect_str = """\
DefaultEndpointsProtocol=http;\
AccountName=devstoreaccount1;\
AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;\
BlobEndpoint=http://{}:10000/devstoreaccount1;""".format(
    host
)

expected = {
    "byteorder": "big",
    "format": 1,
    "samples": 50,
    "sampleinterval": 4.0,
    "byteoffset-first-trace": 3600,
    "guid": "86f5f8f783fabe2773531d5529226d37b6c9bdcf",
    "dimensions": [
        [x for x in range(1, 6)],
        [x for x in range(20, 25)],
        [float(x) for x in range(0, 200, 4)],
    ],
}


container = "dir"
small_segy = Path("tests/data/small.sgy")
small_json = Path("tests/data/small.json")


@pytest.fixture(scope="session")
def delete_all_containers():
    blob = BlobServiceClient.from_connection_string(connect_str)
    for c in blob.list_containers():
        blob.delete_container(c)


@pytest.fixture
def blob_service(scope="session"):
    bsc = BlobServiceClient.from_connection_string(connect_str)
    bsc.create_container(container)
    blob_client = bsc.get_blob_client(container=container, blob=small_segy.name)
    with open(small_segy, "rb") as f:
        blob_client.upload_blob(f.read())

    yield bsc

    bsc.delete_container(container)


@pytest.mark.skipif(os.getenv("AZURITE") is None, reason="Need Azurite")
def test_blobio(blob_service, delete_all_containers):

    blobio = BlobIO(blob_service, container)

    b = blobio.open(small_segy.name)
    with open(small_segy, "rb") as f:
        assert b.read() == f.read()


@pytest.mark.skipif(os.getenv("AZURITE") is None, reason="Need Azurite")
def test_upload(delete_all_containers):
    params = {
        "subcube-dims": (120, 120, 120,),
    }

    blob_service = BlobServiceClient.from_connection_string(connect_str)
    upload.upload(params, small_json, small_segy, blob_service)


@pytest.mark.skipif(os.getenv("AZURITE") is None, reason="Need Azurite")
def test_blobio_scan(blob_service):
    blobio = BlobIO(blob_service, container)
    b = blobio.open(blobname=small_segy.name)
    assert scan.scan(b) == expected
