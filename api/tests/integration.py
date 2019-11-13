#!/usr/bin/env python

import base64
import hmac
import hashlib
import json
import os
import pytest
import requests
import subprocess
from time import sleep, time
from bs4 import BeautifulSoup
from struct import pack 

COVERAGE_LIMIT = int(os.environ["COVERAGE_LIMIT"])

os.environ["STITCH_GRPC_ADDR"] = "0.0.0.0:10000"
os.environ["MANIFEST_SRC"] = "path"
os.environ["MANIFEST_PATH"] = "./"
os.environ["LOCAL_SURFACE_PATH"] = "./"
os.environ["AUTHSERVER"] = "https://login.microsoftonline.com/common"
os.environ["API_SECRET"] = "SECRET_KEY"
os.environ["HTTP_ONLY"] = "True"
os.environ["HOST_ADDR"] = "localhost:7020"


def authzToken():
    header = json.dumps({'alg': "HS256", "typ": "JWT"})
    h64 = base64.urlsafe_b64encode(bytes(header, encoding='utf8'))
    h64 = h64.decode("utf-8") .rstrip('=')
    body = json.dumps({"sub": "test-services",
                       "exp": f"{time()+3600}",
                       "aud": "sc-api"
                       })

    bd64 = base64.urlsafe_b64encode(bytes(body, encoding='utf8'))
    bd64 = bd64.decode("utf-8") .rstrip('=')
    tok = h64 + "." + bd64
    sig = hmac.new(bytes(os.environ["API_SECRET"], encoding='utf8'),
                   bytes(tok, encoding='utf8'),
                   digestmod=hashlib.sha256).digest()
    s64 = base64.urlsafe_b64encode(sig)
    s64 = s64.decode("utf-8") .rstrip('=')
    return tok + "." + s64


def cmp_bytes(a: bytes, b: bytes):
    n = 0
    if len(a) != len(b):
        return -1
    for c, d in zip(a, b):
        n += 1
        if c != d:
            return n
    return 0


def test_code_coverage():
    with open("../coverage/index.html") as fp:
        soup = BeautifulSoup(fp, 'html.parser')
    coverageDiv = soup.find('div', attrs={'id': 'totalcov'})
    coverage = int(coverageDiv.contents[0][0:2])
    assert coverage > COVERAGE_LIMIT, "Please increase test coverage above {}% or lower the limit".format(
        COVERAGE_LIMIT)


def test_create_defaults():
    subprocess.call(["../api", "defaults", "--config",
                     ".sc-api.yaml"], shell=False)
    assert os.path.exists('.sc-api.yaml') == True
    with open('.sc-api.yaml', "r") as f:
        config = f.read()
        assert 'authserver: '+os.environ["AUTHSERVER"] in config
        assert 'tls: false' in config
        assert 'issuer: ""\n' in config


def test_get_post():
    test_manifest = '{"basename":"checker","cubexs":2,"cubeys":2,"cubezs":2,"fragmentxs":2,"fragmentys":2,"fragmentzs":2}'
    surface = b'\x01\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00'
    want = b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x80?\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
    api = subprocess.Popen(["../api", "serve", "--config",
                          ".sc-api.yaml"], shell=False)

    cs = subprocess.Popen(["../../corestub/corestub"], shell=False)
            
    sleep(2)
    try:
        r = requests.get('http://localhost:7020/',
                         headers={'Authorization': 'Bearer '+authzToken()})
        assert (r.status_code == 200), authzToken()

        with open("manifest", "w") as f:
            f.write(test_manifest)

        with open("surface", "wb") as f:
            f.write(surface)
        r = requests.get('http://localhost:7020/stitch/manifest/surface',
                          headers={'Authorization': 'Bearer '+authzToken()})

        if cmp_bytes(r.content, want) != 0:
            print(cmp_bytes(r.content, want))
        assert r.content == want
    finally:
        api.kill()
        cs.kill()


if __name__ == "__main__":
    test_code_coverage()
    test_create_defaults()
    test_get_post()
