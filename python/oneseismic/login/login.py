import argparse
import contextlib
import json
import msal
import os
import pathlib
import sys
from xdg import XDG_CACHE_HOME

class config:
    """Read & write config to disk

    Automate the reading and writing of configuration to disk. The main goal of
    the config class is to simplify path handling and file reading.

    Parameters
    ----------
    cache_dir : path or str, optional
        Configuration cache directory, defaults to XDG_CACHE_HOME

    Notes
    -----
    Using different cache directories would effectively result in different
    profiles, which would enable working with multiple oneseismic
    installations concurrently.

    See also
    --------
    tokens
    """
    def __init__(self, cache_dir = None):
        if cache_dir is None:
            cache_dir = XDG_CACHE_HOME
        self.root = os.path.join(str(cache_dir), 'oneseismic')
        self.fname = 'config.json'

    def path(self):
        """Path to config file

        Notes
        -----
        This function is meant for internal use.
        """
        return os.path.join(self.root, self.fname)

    def store(self, cfg):
        """Store configuration

        Store the configuration cfg. This is a pure IO function, and  the
        configuration *is not checked* for consistency or correctness.

        Parameters
        ----------
        cfg : dict
        """
        pathlib.Path(self.root).mkdir(exist_ok = True)
        with open(self.path(), 'w') as f:
            json.dump(cfg, f)

    def load(self):
        """Load configuration

        Load a configuration from disk. The contents of the must be valid JSON,
        but it is only checked syntactically. There are no checks for missing
        or mismatching key/values.

        Returns
        -------
        cfg : dict
        """
        with open(self.path()) as cfg:
            return json.load(cfg)

class tokens:
    """Acquire, read & write access tokens to disk

    The tokens class is useful for managing state and disk caches, and most
    methods are for internal use.

    Some effort is put into making tokens behave as predictable as possible,
    while at the same time managing persistency and token refreshing. Users of
    oneseismic should generally not be aware of the existence of this class.

    Parameters
    ----------
    cache_dir : path or str, optional
        Configuration cache directory, defaults to XDG_CACHE_HOME

    Notes
    -----
    The 'tokens' name is not great, and it is quite likely to change in the
    future. This should not cause too many breakages since its serves as an
    implementation detail, but it is something to keep in mind.
    """
    def __init__(self, cache_dir = None):
        if cache_dir is None:
            cache_dir = XDG_CACHE_HOME
        self.root = os.path.join(str(cache_dir), 'oneseismic')
        self.fname = 'accessToken.json'
        self.app = None

    def path(self):
        """Path to config file

        Returns
        -------
        path : str

        Notes
        -----
        This function is meant for internal use.
        """
        return os.path.join(self.root, self.fname)

    cachecache = {}
    def cache(self):
        """Get a token cache

        Load a token cache from disk, or create a new cache instance. Cache
        instances map onto the backing storage, and will attempt to
        synchronize.

        To ensure consistency, token caches always be obtained with this
        method, and never constructed explicitly.

        Returns
        -------
        cache : msal.SerializeableTokenCache

        Notes
        -----
        This function is meant for internal use.
        """
        path = self.path()
        if path in self.cachecache:
            return self.cachecache[path]

        pathlib.Path(self.root).mkdir(exist_ok = True)
        cache = msal.SerializableTokenCache()

        with contextlib.suppress(FileNotFoundError):
            with open(path) as f:
                cache.deserialize(f.read())

        self.cachecache[path] = cache
        return cache

    def flush(self, cache):
        """Write a token cache to disk

        Notes
        -----
        This function is meant for internal use.
        """
        if not cache.has_state_changed:
            return

        with open(self.path(), 'w') as f:
            f.write(cache.serialize())

    def acquire(self, config):
        """Acquire a fresh token

        Conceptually "log in" to oneseismic. This must be called before load()
        or any other function that tries to call oneseismic.

        This function must be called on "cold" systems, and should really only
        be useful for the login() program or similar features.

        Returns
        -------
        self : tokens
            Returns itself to support chaining

        See also
        --------
        load

        Examples
        --------
        >>> cfg = getconfig()
        >>> tokens().acquire(cfg)
        >>> session = oneseismic.http_session.fromconfig()
        >>> oneseismic.ls(session)
        ['038855', '0d235a']
        """
        os.makedirs(pathlib.Path(self.root), exist_ok = True)

        cache = self.cache()
        app = msal.PublicClientApplication(
            config['client_id'],
            authority = config['auth_server'],
            token_cache = cache,
        )

        flow = app.initiate_device_flow(config['scopes'])
        print(flow['message'])
        sys.stdout.flush()
        app.acquire_token_by_device_flow(flow)
        # only self-assign when no exceptions occured
        self.app = app
        self.flush(cache)
        return self

    def load(self, config):
        """Load a cache from disk

        This is the "warm" twin of acquire, it loads tokens from persistent
        storage. It assumes acquire() has been called on the system, i.e. there
        is a valid token or a refreshable token in storage.

        Generally, this function should be called immediately after the object
        is initialized [1], see Examples. It should be called only once for
        each instance.

        [1] except in tests, where you might want to inject custom dependencies

        Returns
        -------
        self : tokens
            Returns itself to support chaining

        Examples
        --------
        Cookie-cutter init:
        >>> cfg = getconfig()
        >>> tok = tokens().load(cfg)
        >>> session = oneseismic.http_session(url = 'url', tokens = tok)
        """
        self.scopes = config['scopes']
        self.app = msal.PublicClientApplication(
            config['client_id'],
            authority = config['auth_server'],
            token_cache = self.cache(),
        )
        return self

    def headers(self):
        """Get authorization headers

        Obtain headers to authorize a request. This should be called every time
        right before a request is made, as it might refresh the token.

        Tokens might be refreshed and persisted.

        Returns
        -------
        headers : dict
            Dictionary of headers to set on a HTTP request
        """
        result = self.app.acquire_token_silent(
            scopes = self.scopes,
            account = self.app.get_accounts()[0],
        )

        if 'access_token' not in result:
            raise RuntimeError("Invalid token found in cache")

        self.flush(self.cache())
        return { 'Authorization': 'Bearer {}'.format(result['access_token']) }

def login(url, client_id, auth_server, scopes, cache_dir=None):
    """ Log in to one seismic

    Fetches token and caches it on disk. This function will prompt user to open
    url to provide credentials. Once this is done, the token can be loaded from
    the cache and refreshed non-interactively.

    For non-interactive workflows run the oneseismic-login executable before
    running your script.

    The url parameter can be None, in which case all programs that depend on it
    must be explicitly passed the URL to the oneseismic service. This can
    happen when login is run with explicit client-id, authority, and scopes
    parameters, but is not recommended.

    Parameters
    ----------
    url: str
        URL to oneseismic

    client_id : str
        The Application (client) ID of the Azure AD app registration.

    auth_server : str
        Use <authentication-endpoint>/<tenant-id>/v2.0, and replace
        <authentication-endpoint> with the authentication endpoint for your
        cloud environment (e.g., "https://login.microsoft.com"), also replacing
        <tenant-id> with the Directory (tenant) ID in which the app registration
        was created.
    """

    cfg = {
        'url': url,
        'client_id': client_id,
        'auth_server': auth_server,
        'scopes': scopes,
    }
    tokens(cache_dir = cache_dir).acquire(cfg)
    config(cache_dir = cache_dir).store(cfg)
