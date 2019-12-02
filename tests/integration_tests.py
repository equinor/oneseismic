import os
import subprocess
from struct import pack, unpack
from urllib.parse import parse_qs, urlparse

import pytest
import requests

sc_uri = os.environ.get("SC_API_HOST_ADDR", "http://localhost:8080")
auth_uri = os.environ.get("AUTH_ADDR", "http://localhost:8089")
fixturesPath = os.environ.get("FIXTURES_PATH")

manifest_data = b'{"basename":"checker","cubexs":0,"cubeys":0,"cubezs":0,"fragmentxs":0,"fragmentys":0,"fragmentzs":0}'
surface_data = pack("<QQQQQQQQQ", 0, 0, 0, 0, 0, 1, 0, 1, 0)
stitch_response = b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x80?\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"

with open(fixturesPath + "/manifests/test-manifest", "wb") as f:
    f.write(manifest_data)


with open(fixturesPath + "/surfaces/test-surface", "wb") as f:
    f.write(surface_data)


r = requests.get(auth_uri + "/oauth2/authorize", allow_redirects=False)
assert r.status_code == 302

token = parse_qs(urlparse(r.headers["Location"]).fragment)["access_token"][0]

auth_header = {"Authorization": f"bearer {token}"}


def test_version_no_auth():
    r = requests.get(sc_uri)
    assert r.status_code == 401


def test_version():
    r = requests.get(sc_uri, headers=auth_header)
    assert r.status_code == 200
    assert r.content.startswith(b"Seismic Cloud API ")


def test_surface_list():
    r = requests.get(sc_uri + "/surface", headers=auth_header)
    assert r.status_code == 200


def test_manifest_list():
    r = requests.get(sc_uri + "/manifest", headers=auth_header)
    assert r.status_code == 200


def test_surface_get():
    r = requests.get(sc_uri + "/surface/test-surface", headers=auth_header)
    assert r.status_code == 200
    assert r.content == surface_data


def test_surface_get_fail():
    r = requests.get(sc_uri + "/surface/not-exist", headers=auth_header)
    assert r.status_code == 404


def test_manifest_get():
    r = requests.get(sc_uri + "/manifest/test-manifest", headers=auth_header)
    assert r.status_code == 200
    assert r.content == manifest_data


def test_stitch():
    r = requests.get(sc_uri + "/stitch/test-manifest/test-surface", headers=auth_header)
    assert r.status_code == 200
    assert r.content == stitch_response


def test_stitch_fail_manifest():
    r = requests.get(sc_uri + "/stitch/no-exist/test-surface", headers=auth_header)
    assert r.status_code == 404
    assert r.content == b"Not Found"


def test_stitch_fail_surface():
    r = requests.get(sc_uri + "/stitch/test-manifest/no-surface", headers=auth_header)
    assert r.status_code == 500
    assert r.content == b"Internal Server Error"
