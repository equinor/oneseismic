#!/usr/bin/env python

import os
import pytest
import requests
import subprocess
from time import sleep

os.environ["STITCH_CMD"] = "/bin/cat"
os.environ["MANIFEST_SRC"] = "path"
os.environ["MANIFEST_PATH"] = "./"
os.environ["NO_AUTH"] = "True"
os.environ["HTTP_ONLY"] = "True"
os.environ["HOST_ADDR"] = "localhost:7020"


def test_create_defaults():
    subprocess.call(["../api", "defaults", "--config", ".sc-api.yaml"])
    assert os.path.exists('.sc-api.yaml') == True
    with open('.sc-api.yaml') as f:
        config = f.read()
        assert 'authserver: http://oauth2.example.com' in config
        assert 'tls: false' in config
        assert 'issuer: ""\n' in config


def test_get_main_page():
    p = subprocess.Popen(
        ["../api serve --config .sc-api.yaml"], shell=True)
    sleep(1)
    r = requests.get('http://localhost:7020/')
    p.kill()
    assert r.status_code == 200


def test_post():
    f = open("sample.manifest", "w")
    f.write("my_manifest\n")
    f.close()

    p = subprocess.Popen(
        ["../api serve --config .sc-api.yaml"], shell=True)
    r = requests.post('http://localhost:7020/stitch/sample',
                      data={'please': 'return'})
    sleep(1)
    p.kill()
    assert r.content == b'M:\x0c\x00\x00\x00my_manifest\nplease=return'


if __name__ == "__main__":
    test_create_defaults()
    test_get_main_page()
    test_post()
