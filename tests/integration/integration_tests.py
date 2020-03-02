import os
import subprocess
from struct import pack, unpack
from urllib.parse import parse_qs, urlparse

import requests


sc_uri = os.environ.get("SC_API_HOST_ADDR", "http://localhost:8080")
auth_uri = os.environ.get("AUTH_ADDR", "http://localhost:8089")

r = requests.get(auth_uri + "/oauth2/authorize", allow_redirects=False)
assert r.status_code == 302

token = parse_qs(urlparse(r.headers["Location"]).fragment)["access_token"][0]

auth_header = {"Authorization": f"Bearer {token}"}


def test_no_auth():
    r = requests.get(sc_uri)
    assert r.status_code == 401

def test_auth():
    r = requests.get(sc_uri, headers=auth_header)
    assert r.status_code == 200
