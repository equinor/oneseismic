#!/usr/bin/env python

import os
import pytest
import requests

import subprocess
from time import sleep

subprocess.call(["../api", "defaults", "--config", ".sc-api.yaml"])
os.environ["STITCH_CMD"] = "/bin/cat"
os.environ["MANIFEST_SRC"] = "path"
os.environ["MANIFEST_PATH"] = "./"
os.environ["NO_AUTH"] = "True"
os.environ["HTTP_ONLY"] = "True"
os.environ["HOST_ADDR"] = "localhost:7020"

f = open("sample.manifest", "w")
f.write("my_manifest\n")
f.close()


def test_get_main_page():
    p = subprocess.Popen(
        ["../api serve --config .sc-api.yaml"], shell=True)
    sleep(2)
    r = requests.get('http://localhost:7020/')
    p.kill()
    assert r.status_code == 200

def test_post():
    p = subprocess.Popen(
        ["../api serve --config .sc-api.yaml"], shell=True)
    r = requests.post('http://localhost:7020/stitch/sample',
                      data={'please': 'return'})
    sleep(2)
    p.kill()
    assert r.content == b'M:\x0c\x00\x00\x00my_manifest\nplease=return'
    subprocess.Popen.kill(p)


if __name__ == "__main__":
    test_get_main_page()
    test_post()
