#!/usr/bin/env python

import os
import pytest
import requests
import subprocess
from time import sleep
from bs4 import BeautifulSoup

COVERAGE_LIMIT = int(os.environ["COVERAGE_LIMIT"])

os.environ["STITCH_CMD"] = "/bin/cat"
os.environ["MANIFEST_SRC"] = "path"
os.environ["MANIFEST_PATH"] = "./"
os.environ["NO_AUTH"] = "True"
os.environ["HTTP_ONLY"] = "True"
os.environ["HOST_ADDR"] = "localhost:7020"


def test_code_coverage():
    with open("../coverage/index.html") as fp:
        soup = BeautifulSoup(fp, 'html.parser')
    coverageDiv = soup.find('div', attrs={'id': 'totalcov'})
    coverage = int(coverageDiv.contents[0][0:2])
    assert coverage > COVERAGE_LIMIT, "Please increase test coverage above {}% or lower the limit".format(
        COVERAGE_LIMIT)


def test_create_defaults():
    subprocess.call(["../api defaults --config .sc-api.yaml"], shell=True)
    assert os.path.exists('.sc-api.yaml') == True
    with open('.sc-api.yaml', "r") as f:
        config = f.read()
        assert 'authserver: http://oauth2.example.com' in config
        assert 'tls: false' in config
        assert 'issuer: ""\n' in config


def test_get_post():
    p = subprocess.Popen(
        ["../api serve --config .sc-api.yaml"], shell=True)
    sleep(0.5)
    r = requests.get('http://localhost:7020/')
    assert r.status_code == 200

    with open("sample.manifest", "w") as f:
        f.write("my_manifest\n")
    r = requests.post('http://localhost:7020/stitch/sample',
                      data={'please': 'return'})
    p.kill()
    assert r.content == b'M:\x0c\x00\x00\x00my_manifest\nplease=return'


if __name__ == "__main__":
    test_code_coverage()
    test_create_defaults()
    test_get_post()
