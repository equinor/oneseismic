import argparse
import json
import msal
import os
from pathlib import Path
import sys
from xdg import XDG_CACHE_HOME

def store_config(client_id, auth_server, scopes, cache_dir):
    config_file = os.path.join(
        cache_dir,
        "config.json"
    )

    config = {
        "client_id": client_id,
        "auth_server": auth_server,
        "scopes": scopes
    }

    json.dump(config, open(config_file, 'w'))


def fetch_token(client_id, auth_server, scopes, cache_dir):
    cache_file = os.path.join(
        cache_dir,
        "accessToken.json"
    )
    cache = msal.SerializableTokenCache()

    if os.path.exists(cache_file):
        cache.deserialize(open(cache_file, "r").read())

    app = msal.PublicClientApplication(
        client_id,
        authority=auth_server,
        token_cache=cache,
    )

    flow = app.initiate_device_flow(scopes)

    print(flow["message"])
    sys.stdout.flush()

    app.acquire_token_by_device_flow(flow)
    open(cache_file, "w").write(cache.serialize())


def login(client_id, auth_server, scopes, cache_dir=None):
    """ Log in to one seismic

    Fetches token and caches it on disk. This function will prompt user to open
    url to provide credentials. Once this is done, the token can be loaded from
    the cache and refreshed non-interactively.

    For non-interactive workflows run the oneseismic-login executable before
    running your script.

    Parameters
    ----------

    client_id : str
        The Application (client) ID of the Azure AD app registration.

    auth_server : str
        Use <authentication-endpoint>/<tenant-id>/v2.0, and replace
        <authentication-endpoint> with the authentication endpoint for your
        cloud environment (e.g., "https://login.microsoft.com"), also replacing
        <tenant-id> with the Directory (tenant) ID in which the app registration
        was created.
    """
    cache_dir = cache_dir or os.path.join(XDG_CACHE_HOME, "oneseismic")
    Path(cache_dir).mkdir(exist_ok=True)

    store_config(client_id, auth_server, scopes, cache_dir)
    fetch_token(client_id, auth_server, scopes, cache_dir)
