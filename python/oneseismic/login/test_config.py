import xdg
import os
import pytest

from .login import config
from .login import tokens

def test_config_paths_and_defaults():
    assert config().root == os.path.join(xdg.XDG_CACHE_HOME, 'oneseismic')
    assert config(cache_dir = 'root').root == os.path.join('root', 'oneseismic')

def test_store_load_roundtrip(tmp_path):
    expected = {
        'key1': 'v1',
        'key2': 'v2',
    }
    store = config(cache_dir = tmp_path)
    store.store(expected)

    load = config(cache_dir = tmp_path)
    result = load.load()
    assert expected == result

def test_load_non_existing_config_fails(tmp_path):
    c = config(cache_dir = tmp_path)
    with pytest.raises(FileNotFoundError):
        c.load()

def test_tokens_paths_and_defaults():
    assert tokens().root == os.path.join(xdg.XDG_CACHE_HOME, 'oneseismic')
    assert tokens(cache_dir = 'root').root == os.path.join('root', 'oneseismic')

def test_shared_cache_between_instances(tmp_path):
    t1 = tokens(cache_dir = tmp_path)
    t2 = tokens(cache_dir = tmp_path)
    # in the current implementation, cache synchronization is implemented as
    # object-identity, and it is assumed that concurrent objects all call
    # flush(). This makes the test super easy to write, but less future proof.
    # Should the change, the test must be updated to capture
    # differently-sourced caches synchronizing
    assert t1.cache() is t2.cache()

def test_tokens_are_persisted_explicit_flush(tmp_path):
    t = tokens(cache_dir = tmp_path)
    assert not os.path.exists(t.path())
    cache = t.cache()
    # Pretend something happened by setting has_state_changed
    cache.has_state_changed = True
    t.flush(cache)
    assert os.path.exists(t.path())

def test_changes_are_persisted_on_token_refresh(tmp_path):
    t = tokens(cache_dir = tmp_path)
    assert not os.path.exists(t.path())

    # fake the msal application class
    class fakeapp:
        def __init__(self, cache):
            self.cache = cache

        def get_accounts(self):
            return ['root']

        def acquire_token_silent(self, *args, **kwargs):
            # Pretend that the token was refreshed instead of fetched from
            # cache
            self.cache.has_state_changed = True
            return { 'access_token': 'fake-token' }

    t.app = fakeapp(t.cache())
    t.scopes = ['360']
    headers = t.headers()
    assert headers == { 'Authorization': 'Bearer fake-token' }
    assert os.path.exists(t.path())

# TODO: tests for application-init from cache, which needs a good, but useless
# token cache bundled with the tests
