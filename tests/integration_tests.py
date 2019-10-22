import pytest
import subprocess
import requests
import os

uri = os.environ.get("SC_API_HOST_ADDRESS", "http://localhost:8080")

def test_happy():
    requests.get(uri)

def test_sad():
    with pytest.raises(requests.exceptions.ConnectionError):
        requests.get(uri+"0")
