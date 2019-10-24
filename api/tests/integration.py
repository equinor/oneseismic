#!/usr/bin/env python

import os
import subprocess
from time import sleep

import pytest
import requests
from bs4 import BeautifulSoup

COVERAGE_LIMIT = int(os.environ["COVERAGE_LIMIT"])

os.environ["STITCH_CMD"] = "/bin/cat"
os.environ["MANIFEST_SRC"] = "path"
os.environ["MANIFEST_PATH"] = "./"
os.environ["NO_AUTH"] = "True"
os.environ["HTTP_ONLY"] = "True"
os.environ["HOST_ADDR"] = "localhost:7020"


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
        soup = BeautifulSoup(fp, "html.parser")
    coverageDiv = soup.find("div", attrs={"id": "totalcov"})
    coverage = int(coverageDiv.contents[0][0:2])
    assert (
        coverage > COVERAGE_LIMIT
    ), "Please increase test coverage above {}% or lower the limit".format(
        COVERAGE_LIMIT
    )


def test_create_defaults():
    subprocess.call(["../api defaults --config .sc-api.yaml"], shell=True)
    assert os.path.exists(".sc-api.yaml") == True
    with open(".sc-api.yaml", "r") as f:
        config = f.read()
        assert "authserver: http://oauth2.example.com" in config
        assert "tls: false" in config
        assert 'issuer: ""\n' in config


def test_get_post():
    test_manifest = '{"basename":"testmanifest","cubexs":1,"cubeys":1,"cubezs":1,"fragmentxs":1,"fragmentys":1,"fragmentzs":1}'
    p = subprocess.Popen(["../api serve --config .sc-api.yaml"], shell=True)
    sleep(0.5)
    try:
        r = requests.get("http://localhost:7020/")
        assert r.status_code == 200

        with open("sample", "w") as f:
            f.write(test_manifest)
        r = requests.post(
            "http://localhost:7020/stitch/sample", data={"point": "reply"}
        )
        want = (
            b"M:\x69\x00\x00\x00"
            + bytes(test_manifest, encoding="utf-8")
            + b"point=reply"
        )
        if cmp_bytes(r.content, want) != 0:
            print(cmp_bytes(r.content, want))
        assert r.content == want
    finally:
        p.kill()


if __name__ == "__main__":
    test_code_coverage()
    test_create_defaults()
    test_get_post()
