import os

import pytest
from azure.storage.blob import BlobServiceClient

from blobio import BlobIO
from upload import upload

host = os.getenv("AZURITE", "localhost")

connect_str = """DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;\
AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;\
BlobEndpoint=http://{}:10000/devstoreaccount1;""".format(
    host
)


container = "dir"
filename = "file"
data = b"data"


@pytest.fixture(scope="session")
def delete_all_containers():
    blob = BlobServiceClient.from_connection_string(connect_str)
    for c in blob.list_containers():
        blob.delete_container(c)


@pytest.fixture
def blob_service_client(scope="session"):
    bsc = BlobServiceClient.from_connection_string(connect_str)
    bsc.create_container(container)
    blob_client = bsc.get_blob_client(container=container, blob=filename)
    blob_client.upload_blob(data)
    yield bsc
    bsc.delete_container(container)


def test_blobio(blob_service_client, delete_all_containers):

    stream = BlobIO(blob=blob_service_client, container=container, filename=filename,)

    assert stream.read() == data


def test_upload(delete_all_containers):
    params = {
        "subcube-dims": (120, 120, 120,),
    }

    blob = BlobServiceClient.from_connection_string(connect_str)

    upload.upload(params, "tests/data/small.json", "tests/data/small.sgy", blob)
