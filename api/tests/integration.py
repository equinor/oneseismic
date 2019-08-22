#!/usr/bin/env python

import os
import pytest
import requests

import subprocess

subprocess.call(["../api", "defaults", "--config", ".sc-api.yaml"])
os.environ["STITCH_CMD"] = "/usr/bin/cat"
os.environ["MANIFEST_SRC"] = "path"
# os.environ["MANIFEST_PATH"] = ""
os.environ["NO_AUTH"] = "True"
os.environ["HTTP_ONLY"] = "True"
os.environ["HOST_ADDR"] = "localhost:8070"

f = open("sample.manifest", "w")
f.write("my_manifest\n")
f.close()


def test_get_main_page():
    p = subprocess.Popen(
        ["../api", "serve", "--config", ".sc-api.yaml"])
    r = requests.get('http://localhost:8070/')
    subprocess.Popen.kill(p)
    assert r.status_code == 200


def test_post():
    p = subprocess.Popen(
        ["../api", "serve", "--config", ".sc-api.yaml"])
    r = requests.post('http://localhost:8070/stitch/sample',
                      data={'please': 'return'})
    subprocess.Popen.kill(p)
    assert r.content == b'M:\x0c\x00\x00\x00my_manifest\nplease=return'
    subprocess.Popen.kill(p)

if __name__ == "__main__":
    test_get_main_page()
    test_post()
