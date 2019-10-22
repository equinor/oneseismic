import pytest
import subprocess
import requests
import os

uri = os.environ.get("SC_API_HOST_ADDR", "http://localhost:8080")

def test_happy():
    requests.get(uri)
