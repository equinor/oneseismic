import pytest
import subprocess
import requests


def test_happy():
    requests.get("http://api:8080")

def test_sad():
    with pytest.raises(requests.exceptions.ConnectionError):
        requests.get("http://api:8081")
