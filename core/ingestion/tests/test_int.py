import json
import os
from pathlib import Path

import pytest
import segyio
from azure.storage.blob import BlobServiceClient

from blobio import BlobIO
from scan import scan
from upload import upload

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
    "sampleinterval": 4000,
    "byteoffset-first-trace": 3600,
    "guid": "86f5f8f783fabe2773531d5529226d37b6c9bdcf",
    "dimensions": [
        [x for x in range(1, 6)],
        [x for x in range(20, 25)],
        [x for x in range(0, 200000, 4000)],
    ],
}


container = "dir"
small_segy = Path("tests/data/small.sgy")
small_json = Path("tests/data/small.json")


@pytest.fixture
def blob_service():
    blob = BlobServiceClient.from_connection_string(connect_str)
    for c in blob.list_containers():
        blob.delete_container(c)

    bsc = BlobServiceClient.from_connection_string(connect_str)
    bsc.create_container(container)

    bc_segy = bsc.get_blob_client(container=container, blob=small_segy.name)
    with open(small_segy, "rb") as f:
        bc_segy.upload_blob(f.read())

    bc_json = bsc.get_blob_client(container=container, blob=small_json.name)
    with open(small_json, "rb") as f:
        bc_json.upload_blob(f.read())

    yield bsc

    bsc.delete_container(container)


@pytest.mark.skipif(os.getenv("AZURITE") is None, reason="Need Azurite")
def test_read_blobio(blob_service):

    blobio = BlobIO(blob_service, container)

    with open(small_segy, "rb") as f:
        assert blobio.open(small_segy.name).read() == f.read()


@pytest.mark.skipif(os.getenv("AZURITE") is None, reason="Need Azurite")
def test_upload_from_blobio(blob_service):
    blobio = BlobIO(blob_service, container)
    meta = json.loads(blobio.open(small_json.name).read())
    shape = [len(x) for x in meta["dimensions"]]

    params = {"subcube-dims": list(shape)}

    segy_stream = blobio.open(small_segy.name)

    names = upload.upload(params, meta, segy_stream, blob_service)

    data = segyio.tools.cube(small_segy)
    cc = blob_service.get_container_client(meta["guid"])
    assert len(names) == 1
    assert len(names[0]) == 1

    download_stream = cc.get_blob_client(names[0][0]).download_blob()
    assert data.tobytes() == download_stream.readall()


@pytest.mark.skipif(os.getenv("AZURITE") is None, reason="Need Azurite")
def test_blobio_scan(blob_service):
    blobio = BlobIO(blob_service, container)
    b = blobio.open(blobname=small_segy.name)
    assert scan.scan(b) == expected
