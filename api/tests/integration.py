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

os.environ["STITCH_CMD"] = "/bin/cat"
os.environ["MANIFEST_SRC"] = "path"
os.environ["MANIFEST_PATH"] = "./"
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
    test_manifest = '{"basename":"testmanifest","cubexs":1,"cubeys":1,"cubezs":1,"fragmentxs":1,"fragmentys":1,"fragmentzs":1}'
    p = subprocess.Popen(["../api", "serve", "--config",
                          ".sc-api.yaml"], shell=False)
    sleep(2)
    try:
        r = requests.get('http://localhost:7020/',
                         headers={'Authorization': 'Bearer '+authzToken()})
        assert (r.status_code == 200), authzToken()

        with open("sample", "w") as f:
            f.write(test_manifest)
        r = requests.post('http://localhost:7020/stitch/sample',
                          data={'point': 'reply'},
                          headers={'Authorization': 'Bearer '+authzToken()})
        want = b'M:\x69\x00\x00\x00' + \
            bytes(test_manifest, encoding='utf-8')+b'point=reply'
        if cmp_bytes(r.content, want) != 0:
            print(cmp_bytes(r.content, want))
        assert r.content == want
    finally:
        p.kill()


if __name__ == "__main__":
    test_create_defaults()
    test_get_post()
