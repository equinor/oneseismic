import os
import subprocess
from struct import pack, unpack
from urllib.parse import parse_qs, urlparse

import pytest
import requests

from seismic_cloud_sdk import (ApiClient, Configuration, ManifestApi,
                               StitchApi, SurfaceApi)

sc_uri = os.environ.get("SC_API_HOST_ADDR", "http://localhost:8080")
auth_uri = os.environ.get("AUTH_ADDR", "http://localhost:8089")
fixturesPath = os.environ.get("FIXTURES_PATH")


manifest = {"cubeid": "exists"}
surface_data = pack("<QQQQQQQQQ", 0, 0, 0, 0, 0, 1, 0, 1, 0)
stitch_response = b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x80?\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"

with open(fixturesPath + "/surfaces/test-surface", "wb") as f:
    f.write(surface_data)


r = requests.get(auth_uri + "/oauth2/authorize", allow_redirects=False)
assert r.status_code == 302

token = parse_qs(urlparse(r.headers["Location"]).fragment)["access_token"][0]

auth_header = {"Authorization": f"Bearer {token}"}

config = Configuration()
config.host = sc_uri
config.api_key = {"Authorization": token}
config.api_key_prefix = {"Authorization": "Bearer"}

client = ApiClient(configuration=config)
manifest_api = ManifestApi(api_client=client)
surface_api = SurfaceApi(api_client=client)
stitch_api = StitchApi(api_client=client)


def test_version_no_auth():
    r = requests.get(sc_uri)
    assert r.status_code == 401


def test_version():
    r = requests.get(sc_uri, headers=auth_header)
    assert r.status_code == 200
    assert r.content.startswith(b"Seismic Cloud API ")


def test_surface_list():
    surfaces = surface_api.list_surfaces()
    assert len(surfaces) == 1
    assert surfaces[0].surface_id == "test-surface"


def test_surface_get():
    surface = surface_api.download_surface("test-surface")
    assert surface.encode() == surface_data


def test_surface_get_fail():
    try:
        surface = surface_api.download_surface("not-exist")
        assert False
    except Exception as e:
        assert e.status == 404


def test_manifest_get():
    manifest_api.upload_manifest("exists", manifest)
    retrieved_manifest = manifest_api.download_manifest("exists")
    assert retrieved_manifest.cubeid == manifest["guid"]


def test_manifest_get_fail():
    try:
        retrieved_manifest = manifest_api.download_manifest("not-exists")
        assert False
    except Exception as e:
        assert e.status == 404


def test_stitch():
    values = stitch_api.stitch("exists", "test-surface")
    assert len(values) > 0


def test_stitch_dim():
    values = stitch_api.stitch_dim("exists", 1, 2)
    assert len(values) > 0
