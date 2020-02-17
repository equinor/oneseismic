import os
import subprocess
from struct import pack, unpack
from urllib.parse import parse_qs, urlparse

import pytest
import requests


sc_uri = os.environ.get("SC_API_HOST_ADDR", "http://localhost:8080")
auth_uri = os.environ.get("AUTH_ADDR", "http://localhost:8089")
fixturesPath = os.environ.get("FIXTURES_PATH")


manifest = {"guid": "exists"}
surface_data = pack("<QQQQQQQQQ", 0, 0, 0, 0, 0, 1, 0, 1, 0)
stitch_response = b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x80?\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"

with open(fixturesPath + "/surfaces/test-surface", "wb") as f:
    f.write(surface_data)


r = requests.get(auth_uri + "/oauth2/authorize", allow_redirects=False)
assert r.status_code == 302

token = parse_qs(urlparse(r.headers["Location"]).fragment)["access_token"][0]

auth_header = {"Authorization": f"Bearer {token}"}


def test_version_no_auth():
    r = requests.get(sc_uri)
    assert r.status_code == 401


def test_surface_list():
    r = requests.get(sc_uri+"/surface", headers=auth_header)
    assert r.status_code == 200
    assert r.json()[0]["surfaceID"] == "test-surface"


def test_surface_get():
    r = requests.get(sc_uri+"/surface/test-surface", headers=auth_header)
    assert r.status_code == 200
    assert r.content == surface_data


def test_surface_get_fail():
    r = requests.get(sc_uri+"/surface/not-exist", headers=auth_header)
    assert r.status_code == 404


def test_manifest_get():
    r = requests.post(sc_uri+"/manifest/exists",
                      headers=auth_header, json=manifest)
    assert r.status_code == 200
    r = requests.get(sc_uri+"/manifest/exists", headers=auth_header)
    assert r.status_code == 200
    assert r.json() == manifest


def test_manifest_get_fail():
    r = requests.get(sc_uri+"/manifest/not-exists", headers=auth_header)
    assert r.status_code == 404


def test_stitch():
    r = requests.get(sc_uri+"/stitch/exists/test-surface", headers=auth_header)
    assert r.status_code == 200
    assert len(r.content) > 0


def dimPath(manifest_id, dim, lineno):
    return f"/stitch/{manifest_id}/dim/{dim}/{lineno}"


def test_stitch_dim():
    r = requests.get(sc_uri+dimPath("exists", 1, 2), headers=auth_header)
    assert r.status_code == 200
    assert len(r.content) > 0
