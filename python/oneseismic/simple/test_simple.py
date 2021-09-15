import pytest

from .simple_client import simple_client

class noschema_simple_client(simple_client):
    """

    simple_client tries to fetch the schema, which means it needs a running
    server and a valid URL. Hijack the init() because we really only need to
    test curtainBy*
    """
    def __init__(self, url):
        self.client = None

def test_curtain_accepts_pairs():
    sc = noschema_simple_client('<url>')

    _ = sc.curtainByIndex('<id>', [(1,2)])
    _ = sc.curtainByIndex('<id>', [[1,2]])
    _ = sc.curtainByLineno('<id>', [(1,2)])
    _ = sc.curtainByLineno('<id>', [[1,2]])

def test_curtain_fails_if_not_all_pairs():
    sc = noschema_simple_client('<url>')

    with pytest.raises(ValueError):
        _ = sc.curtainByIndex('<id>', [[]])
    with pytest.raises(ValueError):
        _ = sc.curtainByLineno('<id>', [[]])

    with pytest.raises(ValueError):
        _ = sc.curtainByIndex('<id>', [[1]])
    with pytest.raises(ValueError):
        _ = sc.curtainByLineno('<id>', [[1]])

    with pytest.raises(ValueError):
        _ = sc.curtainByIndex('<id>', [[1, 2, 3]])
    with pytest.raises(ValueError):
        _ = sc.curtainByLineno('<id>', [[1, 2, 3]])
