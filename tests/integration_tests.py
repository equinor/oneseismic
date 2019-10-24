import os
import subprocess

import pytest
import requests
from struct import pack, unpack
import math
import json

uri = os.environ.get("SC_API_HOST_ADDR", "http://localhost:8080")
fixtures = os.environ.get("MANIFEST_PATH")


def test_happy():
    requests.get(uri)


def test_sad():
    with pytest.raises(requests.exceptions.ConnectionError):
        requests.get("http://some.missing.url:666")


def test_get_post():
    manifest = {
        "basename":"testmanifest",
    }
    r = requests.get(uri)
    assert r.status_code == 200
    with open(fixtures+"/sample", "w") as f:
        json.dump(manifest, f)
    data = (1,2,3)
    r = requests.post(uri + "/stitch/sample", data=pack("<fff", *data))
    got = unpack("<qf", r.content)
    assert got[0] == 0
    assert got[1] - math.sin(data[2]) < 1e-7
